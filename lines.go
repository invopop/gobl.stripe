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
	"github.com/stripe/stripe-go/v84"
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

	price := CurrencyAmount(line.Amount, FromCurrency(line.Currency)).Divide(invLine.Quantity)
	invLine.Item.Price = &price

	if len(line.DiscountAmounts) > 0 && line.Discountable {
		invLine.Discounts = FromInvoiceLineDiscounts(line.DiscountAmounts, line.Currency)
	}

	invLine.Taxes = FromInvoiceLineItemTaxesToTaxSet(line.Taxes, regimeDef)

	if line.Period != nil {
		invLine.Period = &cal.Period{
			Start: *newDateFromTS(line.Period.Start),
			End:   *newDateFromTS(line.Period.End),
		}
	}

	return invLine
}

// newQuantityFromInvoiceLine resolves the quantity for a GOBL invoice line item.
// In v84, we don't have direct access to BillingScheme, so we use the quantity directly.
// If quantity is 0, we default to 1 (typically for tiered pricing).
func newQuantityFromInvoiceLine(line *stripe.InvoiceLineItem) num.Amount {
	if line.Quantity == 0 {
		return num.MakeAmount(1, 0)
	}
	return num.MakeAmount(line.Quantity, 0)
}

// fromInvoiceLineToItem creates a new GOBL item from a Stripe invoice line item.
func fromInvoiceLineToItem(line *stripe.InvoiceLineItem) *org.Item {
	item := &org.Item{
		Name:     setItemName(line),
		Currency: currency.Code(strings.ToUpper(string(line.Currency))),
	}

	// In v84, metadata would need to be expanded from the product
	// For now, we use the line metadata if available
	if len(line.Metadata) > 0 {
		item.Ext = newExtensionsWithPrefix(line.Metadata, customDataItemExt)
	}

	return item
}

// setItemName sets the name of the item for a GOBL invoice line item.
// In v84, Price and Plan objects are not directly available, so we use the Description field.
func setItemName(line *stripe.InvoiceLineItem) string {
	if line.Description != "" {
		return line.Description
	}

	// In v84, to get product name, you would need to expand the price/product
	// or make separate API calls. For now, we'll use just the description.
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
	// We can set the amount directly from the one received in Stripe
	if discountAmount.Discount == nil {
		return &bill.LineDiscount{
			Amount: CurrencyAmount(discountAmount.Amount, FromCurrency(curr)),
		}
	}

	// In v84, coupon is in Discount.Source.Coupon
	if discountAmount.Discount.Source == nil || discountAmount.Discount.Source.Coupon == nil {
		return &bill.LineDiscount{
			Amount: CurrencyAmount(discountAmount.Amount, FromCurrency(curr)),
		}
	}

	return &bill.LineDiscount{
		Amount: CurrencyAmount(discountAmount.Amount, FromCurrency(curr)),
		Reason: discountAmount.Discount.Source.Coupon.Name,
	}
}

// FromInvoiceLineItemTaxesToTaxSet converts Stripe invoice line item taxes into a GOBL tax set.
func FromInvoiceLineItemTaxesToTaxSet(taxes []*stripe.InvoiceLineItemTax, regimeDef *tax.RegimeDef) tax.Set {
	var ts tax.Set
	for _, taxItem := range taxes {
		taxCombo := FromInvoiceLineItemTaxToTaxCombo(taxItem, regimeDef)
		if taxCombo != nil {
			ts = append(ts, taxCombo)
		}
	}
	return ts
}

// FromInvoiceLineItemTaxToTaxCombo creates a new GOBL tax combo from a Stripe invoice line item tax.
func FromInvoiceLineItemTaxToTaxCombo(taxItem *stripe.InvoiceLineItemTax, regimeDef *tax.RegimeDef) *tax.Combo {
	tc := new(tax.Combo)

	// In v84, we need to get tax category from TaxRateDetails if available
	if taxItem.TaxRateDetails != nil && taxItem.Type == stripe.InvoiceLineItemTaxTypeTaxRateDetails {
		// We have tax rate details, but we'd need to fetch the full TaxRate to get category
		// For now, we'll use a default category based on the regime
		tc.Category = tax.CategoryVAT // Default, should be determined from expanded tax rate
	}

	if tc.Category == "" {
		// If we can't determine category, skip this tax
		return nil
	}

	// Check for reverse charge
	if taxItem.TaxabilityReason == stripe.InvoiceLineItemTaxTaxabilityReasonReverseCharge {
		tc.Country = regimeDef.Country
		tc.Key = tax.KeyReverseCharge
		return tc
	}

	// Set country from regime
	tc.Country = regimeDef.Country

	// Calculate percentage from amount and taxable amount
	if taxItem.TaxableAmount > 0 {
		percentage := float64(taxItem.Amount) / float64(taxItem.TaxableAmount) * 100
		tc.Percent = percentFromFloat(percentage)
	}

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

	invLine.Taxes = FromCreditNoteLineItemTaxesToTaxSet(line.Taxes, regimeDef)

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
		return CurrencyAmount(line.Amount, curr)
	}

	// The unit amount can be 0 when discounts applied the line amount.
	if line.UnitAmount == 0 {
		// We could use unit amount excluding tax, but if the tax is included it will not match.
		unitAmount := line.Amount / line.Quantity
		return CurrencyAmount(unitAmount, curr)
	}

	return CurrencyAmount(line.UnitAmount, curr)
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
func FromCreditNoteLineDiscount(discountAmount *stripe.CreditNoteLineItemDiscountAmount, curr currency.Code) *bill.LineDiscount {
	if discountAmount.Discount == nil {
		return &bill.LineDiscount{
			Amount: CurrencyAmount(discountAmount.Amount, curr),
		}
	}

	// In v84, coupon is in Discount.Source.Coupon
	if discountAmount.Discount.Source == nil || discountAmount.Discount.Source.Coupon == nil {
		return &bill.LineDiscount{
			Amount: CurrencyAmount(discountAmount.Amount, curr),
		}
	}

	return &bill.LineDiscount{
		Amount: CurrencyAmount(discountAmount.Amount, curr),
		Reason: discountAmount.Discount.Source.Coupon.Name,
	}
}

// FromCreditNoteLineItemTaxesToTaxSet converts Stripe credit note line item taxes into a GOBL tax set.
func FromCreditNoteLineItemTaxesToTaxSet(taxes []*stripe.CreditNoteLineItemTax, regimeDef *tax.RegimeDef) tax.Set {
	var ts tax.Set
	for _, taxItem := range taxes {
		tc := FromCreditNoteLineItemTaxToTaxCombo(taxItem, regimeDef)
		if tc != nil {
			ts = append(ts, tc)
		}
	}
	return ts
}

// FromCreditNoteLineItemTaxToTaxCombo creates a new GOBL tax combo from a Stripe credit note line item tax.
func FromCreditNoteLineItemTaxToTaxCombo(taxItem *stripe.CreditNoteLineItemTax, regimeDef *tax.RegimeDef) *tax.Combo {
	tc := new(tax.Combo)

	// In v84, we need to get tax category from TaxRateDetails if available
	if taxItem.TaxRateDetails != nil && taxItem.Type == stripe.CreditNoteLineItemTaxTypeTaxRateDetails {
		// We have tax rate details, but we'd need to fetch the full TaxRate to get category
		// For now, we'll use a default category based on the regime
		tc.Category = tax.CategoryVAT // Default, should be determined from expanded tax rate
	}

	if tc.Category == "" {
		// If we can't determine category, skip this tax
		return nil
	}

	// Check for reverse charge
	if taxItem.TaxabilityReason == stripe.CreditNoteLineItemTaxTaxabilityReasonReverseCharge {
		tc.Country = regimeDef.Country
		tc.Key = tax.KeyReverseCharge
		return tc
	}

	// Set country from regime
	tc.Country = regimeDef.Country

	// Calculate percentage from amount and taxable amount
	if taxItem.TaxableAmount > 0 {
		percentage := float64(taxItem.Amount) / float64(taxItem.TaxableAmount) * 100
		tc.Percent = percentFromFloat(percentage)
	}

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
