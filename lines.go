package goblstripe

import (
	"strconv"
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stripe/stripe-go/v81"
)

// Invoice Lines

// FromInvoiceLines converts Stripe invoice line items into GOBL bill lines.
func FromInvoiceLines(lines []*stripe.InvoiceLineItem, regimeDef *tax.RegimeDef) []*bill.Line {
	invLines := make([]*bill.Line, 0, len(lines))
	for _, line := range lines {
		line := FromInvoiceLine(line, regimeDef)
		if line != nil {
			invLines = append(invLines, line)
		}
	}
	return invLines
}

// FromInvoiceLine converts a single Stripe invoice line item into a GOBL bill line.
func FromInvoiceLine(line *stripe.InvoiceLineItem, regimeDef *tax.RegimeDef) *bill.Line {
	qty, price := resolveInvoiceLineQuantityAndPrice(line)
	invLine := &bill.Line{
		Quantity: qty,
		Item:     fromInvoiceLineToItem(line),
	}
	invLine.Item.Price = &price

	if len(line.DiscountAmounts) > 0 && line.Discountable {
		invLine.Discounts = FromInvoiceLineDiscounts(line.DiscountAmounts, line.Currency)
	}

	invLine.Taxes = FromInvoiceTaxAmountsToTaxSet(line.TaxAmounts, regimeDef)

	if line.Period != nil {
		invLine.Period = &cal.Period{
			Start: *newDateFromTS(line.Period.Start, regimeDef.TimeLocation()),
			End:   *newDateFromTS(line.Period.End, regimeDef.TimeLocation()),
		}
	}

	return invLine
}

// resolveInvoiceLineQuantityAndPrice picks the (quantity, price) pair to use for
// a GOBL invoice line. The default per-unit form is `quantity = line.Quantity`
// and `price = line.Amount / quantity` (rounded to currency subunits). When that
// rounded price doesn't multiply back to line.Amount — e.g. Stripe's
// transform_quantity package pricing where amount is not a whole multiple of
// quantity at 2 decimals — we collapse to a lump-sum representation
// (`quantity = 1`, `price = line.Amount`) instead of fabricating a per-unit
// price the data won't support. Tiered billing always takes the lump-sum path
// because no honest single per-unit price exists across tier breakpoints.
// Zero-quantity zero-amount lines (typically inactive add-ons) preserve the
// per-unit display (qty=0, price=Price.UnitAmount) — the sum is 0 either way,
// but the unit price keeps the line readable. The Stripe-supplied item
// description carries the per-unit narrative in all paths.
func resolveInvoiceLineQuantityAndPrice(line *stripe.InvoiceLineItem) (num.Amount, num.Amount) {
	curr := FromCurrency(line.Currency)
	amount := CurrencyAmount(line.Amount, curr)

	if line.Price == nil {
		return num.MakeAmount(1, 0), amount
	}

	if line.Price.BillingScheme == stripe.PriceBillingSchemeTiered {
		return num.MakeAmount(1, 0), amount
	}

	if line.Quantity == 0 {
		if line.Amount != 0 {
			// Quantity 0 with non-zero amount is anomalous Stripe data; lump
			// sum is the only representation that reconciles.
			return num.MakeAmount(1, 0), amount
		}
		price := CurrencyAmount(line.Price.UnitAmount, curr)
		return num.AmountZero, price
	}

	qty := num.MakeAmount(line.Quantity, 0)
	price := amount.Divide(qty)
	if !price.Multiply(qty).Equals(amount) {
		// Rounded per-unit price doesn't round-trip to line.Amount; fall back
		// to a lump-sum line so the tax base reconciles with Stripe.
		return num.MakeAmount(1, 0), amount
	}
	return qty, price
}

// fromInvoiceLineToItem creates a new GOBL item from a Stripe invoice line item.
func fromInvoiceLineToItem(line *stripe.InvoiceLineItem) *org.Item {

	item := &org.Item{
		Name:     setItemName(line),
		Currency: currency.Code(strings.ToUpper(string(line.Currency))),
	}

	if line.Price != nil && line.Price.Product != nil && line.Price.Product.Metadata != nil {
		item.Ext = newExtensionsWithPrefix(line.Price.Product.Metadata, customDataItemExt)
	}

	return item
}

// setItemName sets the name of the item for a GOBL invoice line item.
func setItemName(line *stripe.InvoiceLineItem) string {
	if line.Description != "" {
		return line.Description
	}

	if line.Price != nil {
		if line.Price.Product != nil {
			return line.Price.Product.Name
		}
	}

	if line.Plan != nil {
		if line.Plan.Product != nil {
			return line.Plan.Product.Name
		}
	}

	return ""
}

// FromInvoiceLineDiscounts creates a list of discounts for a GOBL invoice line item.
func FromInvoiceLineDiscounts(discounts []*stripe.InvoiceLineItemDiscountAmount, curr stripe.Currency) []*bill.LineDiscount {
	invDiscounts := make([]*bill.LineDiscount, 0)
	for _, discount := range discounts {
		lineDiscount := FromInvoiceLineDiscount(discount, curr)
		if lineDiscount != nil {
			invDiscounts = append(invDiscounts, lineDiscount)
		}
	}
	return invDiscounts
}

// FromInvoiceLineDiscount creates a discount for a GOBL invoice line item.
func FromInvoiceLineDiscount(discountAmount *stripe.InvoiceLineItemDiscountAmount, curr stripe.Currency) *bill.LineDiscount {
	if discountAmount.Amount == 0 {
		return nil
	}

	// We can set the amount directly from the one received in Stripe
	if discountAmount.Discount == nil {
		return &bill.LineDiscount{
			Amount: CurrencyAmount(discountAmount.Amount, FromCurrency(curr)),
		}
	}

	if discountAmount.Discount.Coupon == nil {
		return &bill.LineDiscount{
			Amount: CurrencyAmount(discountAmount.Amount, FromCurrency(curr)),
		}
	}

	return &bill.LineDiscount{
		Amount: CurrencyAmount(discountAmount.Amount, FromCurrency(curr)),
		Reason: discountAmount.Discount.Coupon.Name,
	}
}

// FromInvoiceTaxAmountsToTaxSet converts Stripe invoice tax amounts into a GOBL tax set.
func FromInvoiceTaxAmountsToTaxSet(taxAmounts []*stripe.InvoiceTotalTaxAmount, regimeDef *tax.RegimeDef) tax.Set {
	var ts tax.Set
	for _, taxAmount := range taxAmounts {
		taxCombo := FromInvoiceTaxAmountToTaxCombo(taxAmount, regimeDef)
		if taxCombo != nil {
			ts = append(ts, taxCombo)
		}
	}
	return ts
}

// FromInvoiceTaxAmountToTaxCombo creates a new GOBL tax combo from a Stripe invoice tax amount.
func FromInvoiceTaxAmountToTaxCombo(taxAmount *stripe.InvoiceTotalTaxAmount, regimeDef *tax.RegimeDef) *tax.Combo {
	tc := new(tax.Combo)
	tc.Category = extractTaxCat(taxAmount.TaxRate)

	if tc.Category == "" {
		return nil
	}

	// Instead of the percentage we can also look at the taxability_reason field.
	// There are different types defined and we could map them to the tax categories in GOBL.

	if taxAmount.TaxabilityReason == stripe.InvoiceTotalTaxAmountTaxabilityReasonReverseCharge {
		tc.Country = regimeDef.Country
		tc.Key = tax.KeyReverseCharge
		return tc
	}

	taxDate := newDateFromTS(taxAmount.TaxRate.Created, regimeDef.TimeLocation())
	if taxAmount.TaxRate.Country == "" {
		tc.Country = regimeDef.Country
	} else {
		tc.Country = l10n.TaxCountryCode(taxAmount.TaxRate.Country)
	}
	// When Stripe tax is not used, the the effective percentage is 0. We should use percentage
	percent := taxAmount.TaxRate.EffectivePercentage
	if percent == 0 && taxAmount.Amount != 0 {
		percent = taxAmount.TaxRate.Percentage
	}

	// Based on the country and the percentage, we can determine the tax rate and value.
	rate, val := lookupRateValue(percent, tc.Country.Code(), tc.Category, taxDate)
	if val == nil {
		// No matching rate found in the regime. Set the tax percent directly.
		tc.Percent = percentFromFloat(percent)
		return tc
	}

	tc.Rate = rate.Rate
	tc.Ext = val.Ext

	return tc
}

// Credit Notes Lines

// creditNoteLineFromTotals creates a single GOBL bill line from the credit note's
// top-level totals. This is used when the credit note has no individual line items.
// When taxes are inclusive, GOBL interprets line prices as tax-inclusive, so we use
// the subtotal (which includes tax). When taxes are exclusive, we use the
// subtotal excluding tax.
func creditNoteLineFromTotals(doc *stripe.CreditNote, curr currency.Code, regimeDef *tax.RegimeDef) *bill.Line {
	subtotal := doc.SubtotalExcludingTax
	if len(doc.TaxAmounts) > 0 && doc.TaxAmounts[0].Inclusive {
		subtotal = doc.Subtotal
	}
	price := CurrencyAmount(subtotal, curr)
	line := &bill.Line{
		Quantity: num.MakeAmount(1, 0),
		Item: &org.Item{
			Name:     "Credit",
			Currency: curr,
			Price:    &price,
		},
	}
	line.Taxes = FromCreditNoteTaxAmountsToTaxSet(doc.TaxAmounts, regimeDef)
	return line
}

// FromCreditNoteLines converts Stripe credit note line items into GOBL bill lines.
func FromCreditNoteLines(lines []*stripe.CreditNoteLineItem, curr currency.Code, regimeDef *tax.RegimeDef) []*bill.Line {
	invLines := make([]*bill.Line, 0, len(lines))
	for _, line := range lines {
		invLines = append(invLines, FromCreditNoteLine(line, curr, regimeDef))
	}
	return invLines
}

// FromCreditNoteLine converts a single Stripe credit note line item into a GOBL bill line.
func FromCreditNoteLine(line *stripe.CreditNoteLineItem, curr currency.Code, regimeDef *tax.RegimeDef) *bill.Line {
	qty, price := resolveCreditNoteLineQuantityAndPrice(line, curr)
	invLine := &bill.Line{
		Quantity: qty,
		Item: &org.Item{
			Name:     line.Description,
			Currency: curr,
			Price:    &price,
		},
	}

	if len(line.DiscountAmounts) > 0 {
		invLine.Discounts = FromCreditNoteLineDiscounts(line.DiscountAmounts, curr)
	}

	invLine.Taxes = FromCreditNoteTaxAmountsToTaxSet(line.TaxAmounts, regimeDef)

	return invLine
}

// resolveCreditNoteLineQuantityAndPrice mirrors resolveInvoiceLineQuantityAndPrice
// for credit notes. CreditNoteLineItem carries no Price object, so there is no
// tiered branch here. Default per-unit form is `quantity = line.Quantity`,
// `price = line.Amount / quantity`; if that doesn't round-trip back to
// line.Amount at currency-subunit precision we fall back to lump-sum.
func resolveCreditNoteLineQuantityAndPrice(line *stripe.CreditNoteLineItem, curr currency.Code) (num.Amount, num.Amount) {
	amount := CurrencyAmount(line.Amount, curr)
	if line.Quantity == 0 {
		return num.MakeAmount(1, 0), amount
	}
	qty := num.MakeAmount(line.Quantity, 0)
	price := amount.Divide(qty)
	if !price.Multiply(qty).Equals(amount) {
		return num.MakeAmount(1, 0), amount
	}
	return qty, price
}

// FromCreditNoteLineDiscounts creates a list of discounts for a GOBL credit note line item.
func FromCreditNoteLineDiscounts(discounts []*stripe.CreditNoteLineItemDiscountAmount, curr currency.Code) []*bill.LineDiscount {
	invDiscounts := make([]*bill.LineDiscount, 0, len(discounts))
	for _, discount := range discounts {
		lineDiscount := FromCreditNoteLineDiscount(discount, curr)
		if lineDiscount != nil {
			invDiscounts = append(invDiscounts, lineDiscount)
		}
	}
	return invDiscounts
}

// FromCreditNoteLineDiscount creates a discount for a GOBL credit note line item.
func FromCreditNoteLineDiscount(discountAmount *stripe.CreditNoteLineItemDiscountAmount, curr currency.Code) *bill.LineDiscount {
	if discountAmount.Amount == 0 {
		return nil
	}

	if discountAmount.Discount == nil {
		return &bill.LineDiscount{
			Amount: CurrencyAmount(discountAmount.Amount, curr),
		}
	}

	if discountAmount.Discount.Coupon == nil {
		return &bill.LineDiscount{
			Amount: CurrencyAmount(discountAmount.Amount, curr),
		}
	}

	return &bill.LineDiscount{
		Amount: CurrencyAmount(discountAmount.Amount, curr),
		Reason: discountAmount.Discount.Coupon.Name,
	}
}

// FromCreditNoteTaxAmountsToTaxSet converts Stripe credit note tax amounts into a GOBL tax set.
func FromCreditNoteTaxAmountsToTaxSet(taxAmounts []*stripe.CreditNoteTaxAmount, regimeDef *tax.RegimeDef) tax.Set {
	var ts tax.Set
	for _, taxAmount := range taxAmounts {
		tc := FromCreditNoteTaxAmountToTaxCombo(taxAmount, regimeDef)
		if tc != nil {
			ts = append(ts, tc)
		}
	}
	return ts
}

// FromCreditNoteTaxAmountToTaxCombo creates a new GOBL tax combo from a Stripe credit note tax amount.
func FromCreditNoteTaxAmountToTaxCombo(taxAmount *stripe.CreditNoteTaxAmount, regimeDef *tax.RegimeDef) *tax.Combo {
	tc := new(tax.Combo)
	tc.Category = extractTaxCat(taxAmount.TaxRate)

	if tc.Category == "" {
		return nil
	}

	// Instead of the percentage we can also look at the taxability_reason field.
	// There are different types defined and we could map them to the tax categories in GOBL.

	if taxAmount.TaxabilityReason == stripe.CreditNoteTaxAmountTaxabilityReasonReverseCharge {
		tc.Country = regimeDef.Country
		tc.Key = tax.KeyReverseCharge
		return tc
	}

	taxDate := newDateFromTS(taxAmount.TaxRate.Created, regimeDef.TimeLocation())
	if taxAmount.TaxRate.Country == "" {
		tc.Country = regimeDef.Country
	} else {
		tc.Country = l10n.TaxCountryCode(taxAmount.TaxRate.Country)
	}

	// When Stripe tax is not used, the the effective percentage is 0. We should use percentage
	percent := taxAmount.TaxRate.EffectivePercentage
	if percent == 0 && taxAmount.Amount != 0 {
		percent = taxAmount.TaxRate.Percentage
	}
	// Based on the country and the percentage, we can determine the tax rate and value.
	rate, val := lookupRateValue(percent, tc.Country.Code(), tc.Category, taxDate)
	if val == nil {
		// No matching rate found in the regime. Set the tax percent directly.
		tc.Percent = percentFromFloat(percent)
		return tc
	}

	tc.Rate = rate.Rate
	tc.Ext = val.Ext

	return tc
}

//Useful functions

// lookupRateValue looks up a tax rate and value from a regime definition.
func lookupRateValue(sRate float64, country l10n.Code, cat cbc.Code, date *cal.Date) (rate *tax.RateDef, val *tax.RateValueDef) {
	regimeDef := tax.RegimeDefFor(country)
	catDef := regimeDef.CategoryDef(cat)
	if catDef == nil {
		return nil, nil
	}
	for _, r := range catDef.Rates {
		for _, v := range r.Values {
			if v.Percent.Rescale(3) != *percentFromFloat(sRate) {
				// Rate value percent doesn't match.
				continue
			}

			if v.Surcharge != nil {
				// Rates values with surcharges not supported.
				continue
			}

			if v != r.Value(*date, v.Ext) {
				// Value rate is not applicable on the date of invoicing.
				continue
			}

			if val != nil {
				// There's a previous matching value, we can't determine which one to use.
				return nil, nil
			}

			rate = r
			val = v

		}
	}

	return rate, val
}

// percentFromFloat creates a new tax percent from a float64.
func percentFromFloat(f float64) *num.Percentage {
	str := strconv.FormatFloat(f, 'f', -1, 64) + "%"
	p, _ := num.PercentageFromString(str)

	// Ensure the tax percent has at least 1 decimal
	if p.Exp() < 3 {
		p = p.Rescale(3)
	}

	return &p
}
