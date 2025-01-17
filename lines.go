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

// I was thinking of adding an object/struct only for this

// FromLines converts Stripe invoice line items into GOBL bill lines.
func FromLines(lines []*stripe.InvoiceLineItem, customerExempt bool, issueDate cal.Date) []*bill.Line {
	invLines := make([]*bill.Line, 0, len(lines))
	for _, line := range lines {
		invLines = append(invLines, FromLine(line, customerExempt, issueDate))
	}
	return invLines
}

// FromLine converts a single Stripe invoice line item into a GOBL bill line.
func FromLine(line *stripe.InvoiceLineItem, customerExempt bool, issueDate cal.Date) *bill.Line {
	invLine := &bill.Line{
		Quantity: newQuantity(line),
		Item:     newItem(line),
	}

	if len(line.DiscountAmounts) > 0 && line.Discountable {
		invLine.Discounts = newDiscounts(line.DiscountAmounts)
	}
	if !customerExempt {
		invLine.Taxes = newTaxes(line.TaxAmounts, issueDate)
	}

	return invLine
}

// newQuantity resolves the quantity for a GOBL invoice line item.
func newQuantity(line *stripe.InvoiceLineItem) num.Amount {
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

// newItem creates a new GOBL item from a Stripe invoice line item.
func newItem(line *stripe.InvoiceLineItem) *org.Item {
	return &org.Item{
		Name:     line.Description,
		Currency: currency.Code(strings.ToUpper(string(line.Currency))),
		Price:    resolvePrice(line),
	}
}

// resolvePrice resolves the price for a GOBL invoice line item.
func resolvePrice(line *stripe.InvoiceLineItem) num.Amount {
	if line.Price == nil {
		return num.AmountZero
	}

	switch line.Price.BillingScheme {
	case stripe.PriceBillingSchemePerUnit:
		return num.MakeAmount(int64(line.Price.UnitAmount), 2) // The unit price comes in cents
	case stripe.PriceBillingSchemeTiered:
		return num.MakeAmount(int64(line.Amount), 2) // The unit price comes in cents
	}

	return num.AmountZero
}

// newDiscounts creates a list of discounts for a GOBL invoice line item.
func newDiscounts(discounts []*stripe.InvoiceLineItemDiscountAmount) []*bill.LineDiscount {
	invDiscounts := make([]*bill.LineDiscount, 0, len(discounts))
	for _, discount := range discounts {
		invDiscounts = append(invDiscounts, &bill.LineDiscount{
			Amount: num.MakeAmount(discount.Amount, 2),
		})
	}
	return invDiscounts
}

// newTaxes converts Stripe invoice tax amounts into a GOBL tax set.
func newTaxes(taxAmounts []*stripe.InvoiceTotalTaxAmount, issueDate cal.Date) tax.Set {
	var ts tax.Set
	for _, taxAmount := range taxAmounts {
		ts = append(ts, newTaxCombo(taxAmount, issueDate))
	}
	return ts
}

// newTaxCombo creates a new GOBL tax combo from a Stripe invoice tax amount.
func newTaxCombo(taxAmount *stripe.InvoiceTotalTaxAmount, issueDate cal.Date) *tax.Combo {
	// We consider rate type percentage but it could be also flat amount, such as a retail delivery fee
	tc := new(tax.Combo)
	cat := extractTaxCat(taxAmount.TaxRate.TaxType)
	tc.Category = cat
	tc.Country = l10n.TaxCountryCode(taxAmount.TaxRate.Country)
	rate, val := lookupRateValue(taxAmount.TaxRate.Percentage, tc.Country.Code(), cat, issueDate)
	if val == nil {
		// No matching rate found in the regime. Set the tax percent directly.
		tc.Percent = taxPercent(taxAmount.TaxRate.Percentage)
		return tc
	}

	tc.Rate = rate.Key
	tc.Ext = val.Ext

	return tc
}

// lookupRateValue looks up a tax rate and value from a regime definition.
func lookupRateValue(sRate float64, country l10n.Code, cat cbc.Code, issueDate cal.Date) (rate *tax.RateDef, val *tax.RateValueDef) {
	regimeDef := tax.RegimeDefFor(country)
	catDef := regimeDef.CategoryDef(cat)
	if catDef == nil {
		return nil, nil
	}

	for _, r := range catDef.Rates {
		for _, v := range r.Values {
			if v.Percent != *taxPercent(sRate) {
				// Rate value percent doesn't match.
				continue
			}

			if v.Surcharge != nil {
				// Rates values with surcharges not supported.
				continue
			}

			if v != r.Value(issueDate, v.Tags, v.Ext) {
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

// taxPercent creates a new tax percent from a float64.
func taxPercent(f float64) *num.Percentage {
	str := strconv.FormatFloat(f, 'f', -1, 64) + "%"
	p, _ := num.PercentageFromString(str)

	// Ensure the tax percent has at least 1 decimal
	if p.Exp() < 3 {
		p = p.Rescale(3)
	}

	return &p
}
