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
		invLines = append(invLines, FromInvoiceLine(line, regimeDef))
	}
	return invLines
}

// FromInvoiceLine converts a single Stripe invoice line item into a GOBL bill line.
func FromInvoiceLine(line *stripe.InvoiceLineItem, regimeDef *tax.RegimeDef) *bill.Line {
	invLine := &bill.Line{
		Quantity: newQuantityFromInvoiceLine(line),
		Item:     fromInvoiceLineToItem(line),
	}

	price := currencyAmount(line.Amount, FromCurrency(line.Currency)).Divide(invLine.Quantity)
	invLine.Item.Price = &price

	if len(line.Discounts) > 0 && line.Discountable {
		invLine.Discounts = FromInvoiceLineDiscounts(line.Discounts)
	}

	invLine.Taxes = FromInvoiceTaxAmountsToTaxSet(line.TaxAmounts, regimeDef)

	return invLine
}

// newQuantityFromInvoiceLine resolves the quantity for a GOBL invoice line item.
// If it is a per unit scheme, the quantity is the line quantity.
// If it is a tiered scheme, the quantity is 1.
func newQuantityFromInvoiceLine(line *stripe.InvoiceLineItem) num.Amount {
	if line.Price == nil {
		return num.AmountZero
	}

	switch line.Price.BillingScheme {
	case stripe.PriceBillingSchemePerUnit:
		return num.MakeAmount(line.Quantity, 0)
	case stripe.PriceBillingSchemeTiered:
		return num.MakeAmount(1, 0)
	default:
		return num.AmountZero
	}
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
func FromInvoiceLineDiscounts(discounts []*stripe.Discount) []*bill.LineDiscount {
	invDiscounts := make([]*bill.LineDiscount, 0)
	for _, discount := range discounts {
		lineDiscount := FromInvoiceLineDiscount(discount)
		if lineDiscount != nil {
			invDiscounts = append(invDiscounts, FromInvoiceLineDiscount(discount))
		}
	}
	return invDiscounts
}

// FromInvoiceLineDiscount creates a discount for a GOBL invoice line item.
func FromInvoiceLineDiscount(discount *stripe.Discount) *bill.LineDiscount {
	if discount.Coupon == nil {
		return nil
	}

	if !discount.Coupon.Valid {
		return nil
	}

	// If the discount is a percentage, we can directly set the percent.
	if discount.Coupon.PercentOff != 0 {
		return &bill.LineDiscount{
			Percent: percentFromFloat(discount.Coupon.PercentOff),
			Reason:  discount.Coupon.Name,
		}
	}

	// If the discount is an amount, we can directly set the amount.
	if discount.Coupon.AmountOff != 0 {
		return &bill.LineDiscount{
			Amount: currencyAmount(discount.Coupon.AmountOff, FromCurrency(discount.Coupon.Currency)),
			Reason: discount.Coupon.Name,
		}
	}

	return nil
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
	if taxAmount.TaxRate.Country == "" && taxAmount.TaxRate.TaxType == "" {
		return nil
	}
	tc := new(tax.Combo)
	tc.Category = extractTaxCat(taxAmount.TaxRate)

	// Instead of the percentage we can also look at the taxability_reason field.
	// There are different types defined and we could map them to the tax categories in GOBL.

	if taxAmount.TaxabilityReason == stripe.InvoiceTotalTaxAmountTaxabilityReasonReverseCharge {
		if hasRate(regimeDef.Country.Code(), tc.Category, tax.RateExempt) {
			tc.Country = regimeDef.Country
			tc.Rate = tax.RateExempt
		}
		return tc
	}

	taxDate := newDateFromTS(taxAmount.TaxRate.Created)
	tc.Country = l10n.TaxCountryCode(taxAmount.TaxRate.Country)
	// Based on the country and the percentage, we can determine the tax rate and value.
	rate, val := lookupRateValue(taxAmount.TaxRate.EffectivePercentage, tc.Country.Code(), tc.Category, taxDate)
	if val == nil {
		// No matching rate found in the regime. Set the tax percent directly.
		tc.Percent = percentFromFloat(taxAmount.TaxRate.EffectivePercentage)
		return tc
	}

	tc.Rate = rate.Key
	tc.Ext = val.Ext

	return tc
}

// Credit Notes Lines

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
	invLine := &bill.Line{
		Quantity: newQuantityFromCreditNote(line),
		Item:     FromCreditNoteLineToItem(line, curr),
	}

	if len(line.DiscountAmounts) > 0 {
		invLine.Discounts = FromCreditNoteLineDiscounts(line.DiscountAmounts, curr)
	}

	invLine.Taxes = FromCreditNoteTaxAmountsToTaxSet(line.TaxAmounts, regimeDef)

	return invLine
}

// newQuantityFromCreditNote resolves the quantity for a GOBL credit note line item.
// If it is a per unit scheme, the quantity is the line quantity.
// If it is a tiered scheme (line_quantity = 0), the quantity is 1.
func newQuantityFromCreditNote(line *stripe.CreditNoteLineItem) num.Amount {
	if line.Quantity == 0 {
		return num.MakeAmount(1, 0)
	}

	return num.MakeAmount(line.Quantity, 0)
}

// FromCreditNoteLineToItem creates a new GOBL item from a Stripe credit note line item.
func FromCreditNoteLineToItem(line *stripe.CreditNoteLineItem, curr currency.Code) *org.Item {
	price := resolveCreditNoteLinePrice(line, curr)
	return &org.Item{
		Name:     line.Description,
		Currency: curr,
		Price:    &price,
	}
}

// resolveCreditNoteLinePrice resolves the price for a GOBL credit note line item.
// If it is a per unit scheme, the price is the unit amount.
// If it is a tiered scheme (line_quantity = 0), the price is the complete amount.
func resolveCreditNoteLinePrice(line *stripe.CreditNoteLineItem, curr currency.Code) num.Amount {
	if line.Quantity == 0 {
		return currencyAmount(line.Amount, curr)
	}

	// The unit amount can be 0 when discounts applied the line amount.
	if line.UnitAmount == 0 {
		// We could use unit amount excluding tax, but if the tax is included it will not match.
		unitAmount := line.Amount / line.Quantity
		return currencyAmount(unitAmount, curr)
	}
	return currencyAmount(line.UnitAmount, curr)
}

// FromCreditNoteLineDiscounts creates a list of discounts for a GOBL credit note line item.
func FromCreditNoteLineDiscounts(discounts []*stripe.CreditNoteLineItemDiscountAmount, curr currency.Code) []*bill.LineDiscount {
	invDiscounts := make([]*bill.LineDiscount, 0, len(discounts))
	for _, discount := range discounts {
		invDiscounts = append(invDiscounts, FromCreditNoteLineDiscount(discount, curr))
	}
	return invDiscounts
}

// FromCreditNoteLineDiscount creates a discount for a GOBL credit note line item.
func FromCreditNoteLineDiscount(discount *stripe.CreditNoteLineItemDiscountAmount, curr currency.Code) *bill.LineDiscount {
	return &bill.LineDiscount{
		Amount: currencyAmount(discount.Amount, curr),
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
	if taxAmount.TaxRate.Country == "" && taxAmount.TaxRate.TaxType == "" {
		return nil
	}

	tc := new(tax.Combo)
	tc.Category = extractTaxCat(taxAmount.TaxRate)

	// Instead of the percentage we can also look at the taxability_reason field.
	// There are different types defined and we could map them to the tax categories in GOBL.

	if taxAmount.TaxabilityReason == stripe.CreditNoteTaxAmountTaxabilityReasonReverseCharge {
		if hasRate(regimeDef.Country.Code(), tc.Category, tax.RateExempt) {
			tc.Country = regimeDef.Country
			tc.Rate = tax.RateExempt
		}
		return tc
	}

	taxDate := newDateFromTS(taxAmount.TaxRate.Created)
	tc.Country = l10n.TaxCountryCode(taxAmount.TaxRate.Country)
	// Based on the country and the percentage, we can determine the tax rate and value.
	rate, val := lookupRateValue(taxAmount.TaxRate.EffectivePercentage, tc.Country.Code(), tc.Category, taxDate)
	if val == nil {
		// No matching rate found in the regime. Set the tax percent directly.
		tc.Percent = percentFromFloat(taxAmount.TaxRate.EffectivePercentage)
		return tc
	}

	tc.Rate = rate.Key
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

			if v != r.Value(*date, v.Tags, v.Ext) {
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

// hasRate checks if a tax rate is defined in the regime.
func hasRate(country l10n.Code, cat cbc.Code, key cbc.Key) bool {
	regimeDef := tax.RegimeDefFor(country)
	catDef := regimeDef.CategoryDef(cat)
	if catDef == nil {
		return false
	}

	for _, r := range catDef.Rates {
		if r.Key == key {
			return true
		}
	}

	return false
}
