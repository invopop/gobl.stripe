package goblstripe_test

import (
	"context"
	"testing"
	"time"

	goblstripe "github.com/invopop/gobl.stripe"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-go/v81"
)

func TestFromLines(t *testing.T) {
	lines := []*stripe.InvoiceLineItem{
		{
			ID:                 "il_1Qf1WLQhcl5B85YleQz6ZuEfd",
			Amount:             -11000,
			AmountExcludingTax: -11000,
			Currency:           stripe.CurrencyEUR,
			Quantity:           2000,
			Price: &stripe.Price{
				BillingScheme: stripe.PriceBillingSchemeTiered,
				Currency:      stripe.CurrencyEUR,
				TaxBehavior:   stripe.PriceTaxBehaviorExclusive,
				UnitAmount:    0,
			},
			Period: &stripe.Period{
				Start: 1736351413,
				End:   1739029692,
			},
			Description: "Unused time on 2000 × Pro Plan after 08 Jan 2025",
			TaxAmounts: []*stripe.InvoiceTotalTaxAmount{
				{
					Amount:    -2310,
					Inclusive: false,
					TaxRate: &stripe.TaxRate{
						TaxType:    stripe.TaxRateTaxTypeVAT,
						Country:    "ES",
						Percentage: 21.0,
					},
					TaxabilityReason: stripe.InvoiceTotalTaxAmountTaxabilityReasonStandardRated,
					TaxableAmount:    -11000,
				},
			},
			UnitAmountExcludingTax: -6,
		},
		{
			ID:                 "il_1Qf1WLQhcl5B85YleQz6ZuEw",
			Amount:             19999,
			AmountExcludingTax: 19999,
			Currency:           stripe.CurrencyEUR,
			Quantity:           10000,
			Price: &stripe.Price{
				BillingScheme: stripe.PriceBillingSchemeTiered,
				Currency:      stripe.CurrencyEUR,
				TaxBehavior:   stripe.PriceTaxBehaviorExclusive,
				UnitAmount:    0,
			},
			Period: &stripe.Period{
				Start: 1736351413,
				End:   1739029692,
			},
			Description: "Remaining time on 10000 × Pro Plan after 08 Jan 2025",
			TaxAmounts: []*stripe.InvoiceTotalTaxAmount{
				{
					Amount:    4200,
					Inclusive: false,
					TaxRate: &stripe.TaxRate{
						TaxType:    stripe.TaxRateTaxTypeVAT,
						Country:    "ES",
						Percentage: 21.0,
					},
					TaxabilityReason: stripe.InvoiceTotalTaxAmountTaxabilityReasonStandardRated,
					TaxableAmount:    19999,
				},
			},
			UnitAmountExcludingTax: 2,
		},
		{
			ID:                 "il_1Qf1WLQhcl5B85YleQz6Zusc",
			Amount:             10000,
			AmountExcludingTax: 10000,
			Currency:           stripe.CurrencyEUR,
			Quantity:           1,
			Price: &stripe.Price{
				BillingScheme: stripe.PriceBillingSchemePerUnit,
				Currency:      stripe.CurrencyEUR,
				TaxBehavior:   stripe.PriceTaxBehaviorExclusive,
				UnitAmount:    10000,
			},
			Period: &stripe.Period{
				Start: 1736351413,
				End:   1739029692,
			},
			Description: "Remaining time on Chargebee Addon after 08 Jan 2025",
			TaxAmounts: []*stripe.InvoiceTotalTaxAmount{
				{
					Amount:    2100,
					Inclusive: false,
					TaxRate: &stripe.TaxRate{
						TaxType:    stripe.TaxRateTaxTypeVAT,
						Country:    "ES",
						Percentage: 21.0,
					},
					TaxabilityReason: stripe.InvoiceTotalTaxAmountTaxabilityReasonStandardRated,
					TaxableAmount:    10000,
				},
			},
			UnitAmountExcludingTax: 10000,
		},
		{
			ID:                 "il_1Qf1WLQhcl5B85YleQz6Zutd",
			Amount:             30882,
			AmountExcludingTax: 25522,
			Currency:           stripe.CurrencyUSD,
			Quantity:           3,
			Price: &stripe.Price{
				BillingScheme: stripe.PriceBillingSchemePerUnit,
				Currency:      stripe.CurrencyUSD,
				TaxBehavior:   stripe.PriceTaxBehaviorInclusive,
				UnitAmount:    10294,
			},
			Period: &stripe.Period{
				Start: 1736351413,
				End:   1739029692,
			},
			Description: "Chargebee Addon",
			TaxAmounts: []*stripe.InvoiceTotalTaxAmount{
				{
					Amount:    5360,
					Inclusive: false,
					TaxRate: &stripe.TaxRate{
						TaxType:    stripe.TaxRateTaxTypeVAT,
						Country:    "ES",
						Percentage: 21.0,
					},
					TaxabilityReason: stripe.InvoiceTotalTaxAmountTaxabilityReasonStandardRated,
					TaxableAmount:    25522,
				},
			},
			UnitAmountExcludingTax: 8507,
		},
	}

	expected := []*bill.Line{
		{
			Quantity: num.MakeAmount(1, 0),
			Item: &org.Item{
				Name:     "Unused time on 2000 × Pro Plan after 08 Jan 2025",
				Currency: currency.EUR,
				Price:    num.MakeAmount(-11000, 2),
				Meta: cbc.Meta{
					goblstripe.MetaKeyDateFrom: cal.NewDate(2025, 1, 8).String(),
					goblstripe.MetaKeyDateTo:   cal.NewDate(2025, 2, 8).String(),
				},
			},
			Taxes: tax.Set{
				{
					Category: tax.CategoryVAT,
					Country:  "ES",
					Percent:  num.NewPercentage(210, 3),
				},
			},
		},
		{
			Quantity: num.MakeAmount(1, 0),
			Item: &org.Item{
				Name:     "Remaining time on 10000 × Pro Plan after 08 Jan 2025",
				Currency: currency.EUR,
				Price:    num.MakeAmount(19999, 2),
				Meta: cbc.Meta{
					goblstripe.MetaKeyDateFrom: cal.NewDate(2025, 1, 8).String(),
					goblstripe.MetaKeyDateTo:   cal.NewDate(2025, 2, 8).String(),
				},
			},
			Taxes: tax.Set{
				{
					Category: tax.CategoryVAT,
					Country:  "ES",
					Percent:  num.NewPercentage(210, 3),
				},
			},
		},
		{
			Quantity: num.MakeAmount(1, 0),
			Item: &org.Item{
				Name:     "Remaining time on Chargebee Addon after 08 Jan 2025",
				Currency: currency.EUR,
				Price:    num.MakeAmount(10000, 2),
				Meta: cbc.Meta{
					goblstripe.MetaKeyDateFrom: cal.NewDate(2025, 1, 8).String(),
					goblstripe.MetaKeyDateTo:   cal.NewDate(2025, 2, 8).String(),
				},
			},
			Taxes: tax.Set{
				{
					Category: tax.CategoryVAT,
					Country:  "ES",
					Percent:  num.NewPercentage(210, 3),
				},
			},
		},
		{
			Quantity: num.MakeAmount(3, 0),
			Item: &org.Item{
				Name:     "Chargebee Addon",
				Currency: currency.USD,
				Price:    num.MakeAmount(10294, 2),
				Meta: cbc.Meta{
					goblstripe.MetaKeyDateFrom: cal.NewDate(2025, 1, 8).String(),
					goblstripe.MetaKeyDateTo:   cal.NewDate(2025, 2, 8).String(),
				},
			},
			Taxes: tax.Set{
				{
					Category: tax.CategoryVAT,
					Country:  "ES",
					Percent:  num.NewPercentage(210, 3),
				},
			},
		},
	}

	result := goblstripe.FromLines(lines)
	assert.Equal(t, expected, result, "Converted lines should match expected")
	for i, line := range result {
		ctx := context.Background()
		// The index is calculated automatically in GOBL, so we need to set it manually
		line.Index = i + 1
		err := line.ValidateWithContext(ctx)
		assert.Nil(t, err, "Line should be valid")
	}

	assert.Equal(t, len(lines), len(result), "Number of converted lines should match input")
}

func TestFromLinePerUnit(t *testing.T) {
	line := &stripe.InvoiceLineItem{
		ID:                 "il_1Qf1WLQhcl5B85YleQz6Zutd",
		Amount:             30882,
		AmountExcludingTax: 25522,
		Currency:           stripe.CurrencyUSD,
		Quantity:           3,
		Price: &stripe.Price{
			BillingScheme: stripe.PriceBillingSchemePerUnit,
			Currency:      stripe.CurrencyUSD,
			TaxBehavior:   stripe.PriceTaxBehaviorInclusive,
			UnitAmount:    10294,
		},
		Period: &stripe.Period{
			Start: 1736351413,
			End:   1739029692,
		},
		Description: "Chargebee Addon",
		TaxAmounts: []*stripe.InvoiceTotalTaxAmount{
			{
				Amount:    5360,
				Inclusive: false,
				TaxRate: &stripe.TaxRate{
					TaxType:    stripe.TaxRateTaxTypeVAT,
					Country:    "ES",
					Percentage: 21.0,
				},
				TaxabilityReason: stripe.InvoiceTotalTaxAmountTaxabilityReasonStandardRated,
				TaxableAmount:    25522,
			},
		},
		UnitAmountExcludingTax: 8507,
	}

	result := goblstripe.FromLine(line)

	assert.NotNil(t, result, "Line conversion should not return nil")
	assert.Equal(t, result.Quantity, num.MakeAmount(3, 0), "Quantity should match line quantity")
	assert.Equal(t, "Chargebee Addon", result.Item.Name, "Item name should match line description")
	assert.Equal(t, currency.USD, result.Item.Currency, "Item currency should match line currency")
	assert.Equal(t, 102.94, result.Item.Price.Float64(), "Item price should match line amount")
}

func TestFromLineTiered(t *testing.T) {
	line := &stripe.InvoiceLineItem{
		ID:                 "il_1Qf1WLQhcl5B85YleQz6ZuEw",
		Amount:             19999,
		AmountExcludingTax: 19999,
		Currency:           stripe.CurrencyEUR,
		Discountable:       true,
		Quantity:           10000,
		Discounts: []*stripe.Discount{
			{Coupon: &stripe.Coupon{AmountOff: 5000, Currency: stripe.CurrencyEUR, Valid: true}},
		},
		Period: &stripe.Period{
			Start: 1736351413,
			End:   1739029692,
		},
		Price: &stripe.Price{
			BillingScheme: stripe.PriceBillingSchemeTiered,
			Currency:      stripe.CurrencyEUR,
			TaxBehavior:   stripe.PriceTaxBehaviorExclusive,
			UnitAmount:    0,
		},
		Description: "Remaining time on 10000 × Pro Plan after 08 Jan 2025",
		TaxAmounts: []*stripe.InvoiceTotalTaxAmount{
			{
				Amount:    4200,
				Inclusive: false,
				TaxRate: &stripe.TaxRate{
					TaxType:    stripe.TaxRateTaxTypeVAT,
					Country:    "ES",
					Percentage: 21.0,
				},
				TaxabilityReason: stripe.InvoiceTotalTaxAmountTaxabilityReasonStandardRated,
				TaxableAmount:    19999,
			},
		},
		UnitAmountExcludingTax: 2,
	}

	result := goblstripe.FromLine(line)

	assert.NotNil(t, result, "Line conversion should not return nil")
	assert.Equal(t, result.Quantity, num.MakeAmount(1, 0), "Quantity should match line quantity")
	assert.Equal(t, "Remaining time on 10000 × Pro Plan after 08 Jan 2025", result.Item.Name, "Item name should match line description")
	assert.Equal(t, currency.EUR, result.Item.Currency, "Item currency should match line currency")
	assert.Equal(t, 199.99, result.Item.Price.Float64(), "Item price should match line amount")
	assert.Equal(t, 50.0, result.Discounts[0].Amount.Float64(), "Discount amount should match line discount")
	assert.Equal(t, num.MakePercentage(210, 3), *result.Taxes[0].Percent, "Tax percentage should match line tax")
	assert.Equal(t, tax.CategoryVAT, result.Taxes[0].Category, "Tax category should match line tax")
	assert.Equal(t, l10n.ES.Tax(), result.Taxes[0].Country, "Tax country should match line tax")
}

func TestFromLineToItem(t *testing.T) {
	tests := []struct {
		name     string
		input    *stripe.InvoiceLineItem
		expected *org.Item
	}{
		{
			name: "per unit billing scheme",
			input: &stripe.InvoiceLineItem{
				Description: "Basic Plan",
				Currency:    "usd",
				Period: &stripe.Period{
					Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
					End:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC).Unix(),
				},
				Price: &stripe.Price{
					UnitAmount:    1000,
					Currency:      "usd",
					BillingScheme: stripe.PriceBillingSchemePerUnit,
				},
			},
			expected: &org.Item{
				Name:     "Basic Plan",
				Currency: "USD",
				Price:    num.MakeAmount(1000, 2),
				Meta: cbc.Meta{
					goblstripe.MetaKeyDateFrom: "2024-01-01",
					goblstripe.MetaKeyDateTo:   "2024-01-31",
				},
			},
		},
		{
			name: "tiered billing scheme",
			input: &stripe.InvoiceLineItem{
				Description: "Usage Plan",
				Currency:    "eur",
				Amount:      2500,
				Period: &stripe.Period{
					Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
					End:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC).Unix(),
				},
				Price: &stripe.Price{
					Currency:      "eur",
					BillingScheme: stripe.PriceBillingSchemeTiered,
				},
			},
			expected: &org.Item{
				Name:     "Usage Plan",
				Currency: "EUR",
				Price:    num.MakeAmount(2500, 2),
				Meta: cbc.Meta{
					goblstripe.MetaKeyDateFrom: "2024-01-01",
					goblstripe.MetaKeyDateTo:   "2024-01-31",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblstripe.FromLineToItem(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFromDiscount(t *testing.T) {
	tests := []struct {
		name     string
		input    *stripe.Discount
		expected *bill.LineDiscount
	}{
		{
			name: "percentage discount",
			input: &stripe.Discount{
				Coupon: &stripe.Coupon{
					Valid:      true,
					PercentOff: 15.0,
					Name:       "New Customer",
				},
			},
			expected: &bill.LineDiscount{
				Percent: num.NewPercentage(150, 3),
				Reason:  "New Customer",
			},
		},
		{
			name: "amount discount",
			input: &stripe.Discount{
				Coupon: &stripe.Coupon{
					Valid:     true,
					AmountOff: 1000,
					Currency:  "eur",
					Name:      "Welcome Bonus",
				},
			},
			expected: &bill.LineDiscount{
				Amount: num.MakeAmount(1000, 2),
				Reason: "Welcome Bonus",
			},
		},
		{
			name: "invalid coupon",
			input: &stripe.Discount{
				Coupon: &stripe.Coupon{
					Valid: false,
				},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblstripe.FromDiscount(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFromTaxAmountsToTaxSet(t *testing.T) {
	tests := []struct {
		name     string
		input    []*stripe.InvoiceTotalTaxAmount
		expected tax.Set
	}{
		{
			name: "multiple tax amounts",
			input: []*stripe.InvoiceTotalTaxAmount{
				{
					TaxRate: &stripe.TaxRate{
						Country:    "DE",
						TaxType:    stripe.TaxRateTaxTypeVAT,
						Percentage: 19.0,
						Created:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
					},
				},
				{
					TaxRate: &stripe.TaxRate{
						Country:    "US",
						TaxType:    stripe.TaxRateTaxTypeSalesTax,
						Percentage: 8.875,
						Created:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
					},
				},
			},
			expected: tax.Set{
				{
					Category: tax.CategoryVAT,
					Country:  "DE",
					Rate:     tax.RateStandard,
				},
				{
					Category: tax.CategoryST,
					Country:  "US",
					Percent:  num.NewPercentage(8875, 5),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblstripe.FromTaxAmountsToTaxSet(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
