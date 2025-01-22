package goblstripe

import (
	"strconv"
	"strings"
	"time"

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

const (
	MetaKeyDateFrom = "stripe-date-from"
	MetaKeyDateTo   = "stripe-date-to"
)

// FromLines converts Stripe invoice line items into GOBL bill lines.
func FromLines(lines []*stripe.InvoiceLineItem) []*bill.Line {
	invLines := make([]*bill.Line, 0, len(lines))
	for _, line := range lines {
		invLines = append(invLines, FromLine(line))
	}
	return invLines
}

// FromLine converts a single Stripe invoice line item into a GOBL bill line.
func FromLine(line *stripe.InvoiceLineItem) *bill.Line {
	invLine := &bill.Line{
		Quantity: newQuantity(line),
		Item:     FromLineToItem(line),
	}

	if len(line.Discounts) > 0 && line.Discountable {
		invLine.Discounts = FromDiscounts(line.Discounts)
	}

	invLine.Taxes = FromTaxAmountsToTaxSet(line.TaxAmounts)

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

// FromLineToItem creates a new GOBL item from a Stripe invoice line item.
func FromLineToItem(line *stripe.InvoiceLineItem) *org.Item {
	return &org.Item{
		Name:     line.Description,
		Currency: currency.Code(strings.ToUpper(string(line.Currency))),
		Price:    resolvePrice(line),
		Meta: cbc.Meta{
			MetaKeyDateFrom: cal.DateOf(time.Unix(line.Period.Start, 0).UTC()).String(),
			MetaKeyDateTo:   cal.DateOf(time.Unix(line.Period.End, 0).UTC()).String(),
		},
	}
}

// resolvePrice resolves the price for a GOBL invoice line item.
func resolvePrice(line *stripe.InvoiceLineItem) num.Amount {
	if line.Price == nil {
		return num.AmountZero
	}

	switch line.Price.BillingScheme {
	case stripe.PriceBillingSchemePerUnit:
		return currencyAmount(int64(line.Price.UnitAmount), FromCurrency(line.Currency))
	case stripe.PriceBillingSchemeTiered:
		return currencyAmount(int64(line.Amount), FromCurrency(line.Currency))
	}

	return num.AmountZero
}

// FromDiscounts creates a list of discounts for a GOBL invoice line item.
func FromDiscounts(discounts []*stripe.Discount) []*bill.LineDiscount {
	invDiscounts := make([]*bill.LineDiscount, 0, len(discounts))
	for _, discount := range discounts {
		invDiscounts = append(invDiscounts, FromDiscount(discount))
	}
	return invDiscounts
}

func FromDiscount(discount *stripe.Discount) *bill.LineDiscount {

	if !discount.Coupon.Valid {
		return nil
	}

	if discount.Coupon.PercentOff != 0 {
		return &bill.LineDiscount{
			Percent: percentFromFloat(discount.Coupon.PercentOff),
			Reason:  discount.Coupon.Name,
		}
	}

	if discount.Coupon.AmountOff != 0 {
		return &bill.LineDiscount{
			Amount: currencyAmount(discount.Coupon.AmountOff, FromCurrency(discount.Coupon.Currency)),
			Reason: discount.Coupon.Name,
		}
	}

	return nil
}

// FromTaxAmountsToTaxSet converts Stripe invoice tax amounts into a GOBL tax set.
func FromTaxAmountsToTaxSet(taxAmounts []*stripe.InvoiceTotalTaxAmount) tax.Set {
	var ts tax.Set
	for _, taxAmount := range taxAmounts {
		ts = append(ts, FromTaxAmountToTaxCombo(taxAmount))
	}
	return ts
}

// FromTaxAmountToTaxCombo creates a new GOBL tax combo from a Stripe invoice tax amount.
func FromTaxAmountToTaxCombo(taxAmount *stripe.InvoiceTotalTaxAmount) *tax.Combo {
	tc := &tax.Combo{
		Category: extractTaxCat(taxAmount.TaxRate.TaxType),
		Country:  l10n.TaxCountryCode(taxAmount.TaxRate.Country),
	}

	taxDate := newDateFromTS(taxAmount.TaxRate.Created)
	// Based on the country and the percentage, we can determine the tax rate and value.
	rate, val := lookupRateValue(taxAmount.TaxRate.Percentage, tc.Country.Code(), tc.Category, taxDate)
	if val == nil {
		// No matching rate found in the regime. Set the tax percent directly.
		tc.Percent = percentFromFloat(taxAmount.TaxRate.Percentage)
		return tc
	}

	tc.Rate = rate.Key
	tc.Ext = val.Ext

	return tc
}

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
