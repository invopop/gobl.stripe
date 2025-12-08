package goblstripe_test

import (
	"testing"
	"time"

	goblstripe "github.com/invopop/gobl.stripe"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-go/v81"
)

func validInvoiceLine() *stripe.InvoiceLineItem {
	return &stripe.InvoiceLineItem{
		ID:           "il_1Qf1WLQhcl5B85YleQz6Zutd",
		Object:       "invoiceitem",
		Amount:       25522,
		Currency:     stripe.CurrencyUSD,
		Proration:    false,
		Type:         stripe.InvoiceLineItemTypeInvoiceItem,
		Discountable: true,
		Discounts:    []*stripe.Discount{},
		Livemode:     false,
		Period: &stripe.Period{
			Start: 1736351413,
			End:   1739029692,
		},
		TaxAmounts: []*stripe.InvoiceTotalTaxAmount{
			{
				Amount:    5360,
				Inclusive: false,
				TaxRate: &stripe.TaxRate{
					TaxType:             stripe.TaxRateTaxTypeVAT,
					Country:             "ES",
					EffectivePercentage: 21.0,
					Percentage:          21.0,
				},
				TaxabilityReason: stripe.InvoiceTotalTaxAmountTaxabilityReasonStandardRated,
				TaxableAmount:    25522,
			},
		},
		TaxRates: []*stripe.TaxRate{},
	}
}

func TestBasicFields(t *testing.T) {
	line := validInvoiceLine()
	result := goblstripe.FromInvoiceLine(line, tax.RegimeDefFor(l10n.DE))

	assert.NotNil(t, result, "Line conversion should not return nil")
	assert.Equal(t, currency.USD, result.Item.Currency, "Item currency should match line currency")
	assert.Equal(t, tax.CategoryVAT, result.Taxes[0].Category, "Tax category should match line tax")
	assert.Equal(t, l10n.ES.Tax(), result.Taxes[0].Country, "Tax country should match line tax")
	assert.Equal(t, num.NewPercentage(210, 3), result.Taxes[0].Percent, "Tax percentage should match line tax")

}

func TestSeveralLines(t *testing.T) {
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
						TaxType:             stripe.TaxRateTaxTypeVAT,
						Country:             "ES",
						EffectivePercentage: 21.0,
						Percentage:          21.0,
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
						TaxType:             stripe.TaxRateTaxTypeVAT,
						Country:             "ES",
						EffectivePercentage: 21.0,
						Percentage:          21.0,
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
						TaxType:             stripe.TaxRateTaxTypeVAT,
						Country:             "ES",
						EffectivePercentage: 21.0,
						Percentage:          21.0,
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
					Inclusive: true,
					TaxRate: &stripe.TaxRate{
						TaxType:             stripe.TaxRateTaxTypeVAT,
						Country:             "ES",
						EffectivePercentage: 21.0,
						Percentage:          21.0,
					},
					TaxabilityReason: stripe.InvoiceTotalTaxAmountTaxabilityReasonStandardRated,
					TaxableAmount:    25522,
				},
			},
			UnitAmountExcludingTax: 8507,
		},
		{
			ID:                 "il_1Qf1WLQhcl5B85YleQz6Zusf",
			Amount:             5000,
			AmountExcludingTax: 5000,
			Currency:           stripe.CurrencyEUR,
			Quantity:           1,
			Price: &stripe.Price{
				BillingScheme: stripe.PriceBillingSchemePerUnit,
				Currency:      stripe.CurrencyEUR,
				TaxBehavior:   stripe.PriceTaxBehaviorExclusive,
				UnitAmount:    5000,
				Product: &stripe.Product{
					Metadata: map[string]string{
						"gobl-item-foo": "bar",
					},
				},
			},
			Period: &stripe.Period{
				Start: 1736351413,
				End:   1739029692,
			},
			Description: "Line with extension",
		},
	}

	expected := []*bill.Line{
		{
			Quantity: num.MakeAmount(1, 0),
			Item: &org.Item{
				Name:     "Unused time on 2000 × Pro Plan after 08 Jan 2025",
				Currency: currency.EUR,
				Price:    num.NewAmount(-11000, 2),
			},
			Taxes: tax.Set{
				{
					Category: tax.CategoryVAT,
					Country:  "ES",
					Percent:  num.NewPercentage(210, 3),
				},
			},
			Period: &cal.Period{
				Start: *cal.NewDate(2025, 1, 8),
				End:   *cal.NewDate(2025, 2, 8),
			},
		},
		{
			Quantity: num.MakeAmount(1, 0),
			Item: &org.Item{
				Name:     "Remaining time on 10000 × Pro Plan after 08 Jan 2025",
				Currency: currency.EUR,
				Price:    num.NewAmount(19999, 2),
			},
			Taxes: tax.Set{
				{
					Category: tax.CategoryVAT,
					Country:  "ES",
					Percent:  num.NewPercentage(210, 3),
				},
			},
			Period: &cal.Period{
				Start: *cal.NewDate(2025, 1, 8),
				End:   *cal.NewDate(2025, 2, 8),
			},
		},
		{
			Quantity: num.MakeAmount(1, 0),
			Item: &org.Item{
				Name:     "Remaining time on Chargebee Addon after 08 Jan 2025",
				Currency: currency.EUR,
				Price:    num.NewAmount(10000, 2),
			},
			Taxes: tax.Set{
				{
					Category: tax.CategoryVAT,
					Country:  "ES",
					Percent:  num.NewPercentage(210, 3),
				},
			},
			Period: &cal.Period{
				Start: *cal.NewDate(2025, 1, 8),
				End:   *cal.NewDate(2025, 2, 8),
			},
		},
		{
			Quantity: num.MakeAmount(3, 0),
			Item: &org.Item{
				Name:     "Chargebee Addon",
				Currency: currency.USD,
				Price:    num.NewAmount(10294, 2),
			},
			Taxes: tax.Set{
				{
					Category: tax.CategoryVAT,
					Country:  "ES",
					Percent:  num.NewPercentage(210, 3),
				},
			},
			Period: &cal.Period{
				Start: *cal.NewDate(2025, 1, 8),
				End:   *cal.NewDate(2025, 2, 8),
			},
		},
		{
			Quantity: num.MakeAmount(1, 0),
			Item: &org.Item{
				Name:     "Line with extension",
				Currency: currency.EUR,
				Price:    num.NewAmount(5000, 2),
				Ext: tax.Extensions{
					"foo": "bar",
				},
			},
			Taxes: nil,
			Period: &cal.Period{
				Start: *cal.NewDate(2025, 1, 8),
				End:   *cal.NewDate(2025, 2, 8),
			},
		},
	}

	result := goblstripe.FromInvoiceLines(lines, tax.RegimeDefFor(l10n.DE))
	assert.Equal(t, expected, result, "Converted lines should match expected")

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
				Inclusive: true,
				TaxRate: &stripe.TaxRate{
					TaxType:             stripe.TaxRateTaxTypeVAT,
					Country:             "ES",
					EffectivePercentage: 21.0,
					Percentage:          21.0,
				},
				TaxabilityReason: stripe.InvoiceTotalTaxAmountTaxabilityReasonStandardRated,
				TaxableAmount:    25522,
			},
		},
		UnitAmountExcludingTax: 8507,
	}

	result := goblstripe.FromInvoiceLine(line, tax.RegimeDefFor(l10n.DE))

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
		DiscountAmounts: []*stripe.InvoiceLineItemDiscountAmount{
			{Amount: 5000, Discount: &stripe.Discount{Coupon: &stripe.Coupon{AmountOff: 5000, Currency: stripe.CurrencyEUR, Valid: true}}},
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
					TaxType:             stripe.TaxRateTaxTypeVAT,
					Country:             "ES",
					EffectivePercentage: 21.0,
					Percentage:          21.0,
				},
				TaxabilityReason: stripe.InvoiceTotalTaxAmountTaxabilityReasonStandardRated,
				TaxableAmount:    19999,
			},
		},
		UnitAmountExcludingTax: 2,
	}

	result := goblstripe.FromInvoiceLine(line, tax.RegimeDefFor(l10n.DE))

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

func TestFromDiscount(t *testing.T) {
	tests := []struct {
		name     string
		input    *stripe.InvoiceLineItemDiscountAmount
		expected *bill.LineDiscount
	}{
		{
			name: "percentage discount",
			input: &stripe.InvoiceLineItemDiscountAmount{
				Amount: 1500,
				Discount: &stripe.Discount{
					Coupon: &stripe.Coupon{
						Valid:      true,
						PercentOff: 15.0,
						Name:       "New Customer",
					},
				},
			},
			expected: &bill.LineDiscount{
				Amount: num.MakeAmount(1500, 2),
				Reason: "New Customer",
			},
		},
		{
			name: "amount discount",
			input: &stripe.InvoiceLineItemDiscountAmount{
				Amount: 1000,
				Discount: &stripe.Discount{
					Coupon: &stripe.Coupon{
						Valid:     true,
						AmountOff: 1000,
						Currency:  "eur",
						Name:      "Welcome Bonus",
					},
				},
			},
			expected: &bill.LineDiscount{
				Amount: num.MakeAmount(1000, 2),
				Reason: "Welcome Bonus",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblstripe.FromInvoiceLineDiscount(tt.input, stripe.CurrencyEUR)
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
						Country:             "DE",
						TaxType:             stripe.TaxRateTaxTypeVAT,
						EffectivePercentage: 19.0,
						Percentage:          19.0,
						Created:             time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
					},
				},
				{
					TaxRate: &stripe.TaxRate{
						Country:             "ES",
						TaxType:             stripe.TaxRateTaxTypeVAT,
						EffectivePercentage: 8.875,
						Percentage:          8.875,
						Created:             time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
					},
				},
			},
			expected: tax.Set{
				{
					Category: tax.CategoryVAT,
					Country:  "DE",
					Rate:     tax.RateGeneral,
				},
				{
					Category: tax.CategoryVAT,
					Country:  "ES",
					Percent:  num.NewPercentage(8875, 5),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblstripe.FromInvoiceTaxAmountsToTaxSet(tt.input, tax.RegimeDefFor(l10n.DE))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFromTaxAmountsExempt(t *testing.T) {
	tests := []struct {
		name     string
		input    []*stripe.InvoiceTotalTaxAmount
		expected tax.Set
	}{
		{
			name: "tax amount exempt",
			input: []*stripe.InvoiceTotalTaxAmount{
				{
					TaxRate: &stripe.TaxRate{
						Country:             "DE",
						TaxType:             stripe.TaxRateTaxTypeVAT,
						EffectivePercentage: 0.0,
						Percentage:          19.0,
						Created:             time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
					},
				},
			},
			expected: tax.Set{
				{
					Category: tax.CategoryVAT,
					Country:  "DE",
					Percent:  num.NewPercentage(0, 3),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblstripe.FromInvoiceTaxAmountsToTaxSet(tt.input, tax.RegimeDefFor(l10n.DE))
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFromInvoiceTaxAmountNoStripeTax tests the fallback to Percentage when
// EffectivePercentage is 0 but there is a non-zero tax amount (Stripe tax not used).
func TestFromInvoiceTaxAmountNoStripeTax(t *testing.T) {
	tests := []struct {
		name     string
		input    []*stripe.InvoiceTotalTaxAmount
		expected tax.Set
	}{
		{
			name: "effective percentage zero but amount non-zero uses percentage",
			input: []*stripe.InvoiceTotalTaxAmount{
				{
					Amount: 1900, // Non-zero tax amount
					TaxRate: &stripe.TaxRate{
						Country:             "DE",
						TaxType:             stripe.TaxRateTaxTypeVAT,
						EffectivePercentage: 0.0, // Stripe tax not used
						Percentage:          19.0,
						Created:             time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
					},
				},
			},
			expected: tax.Set{
				{
					Category: tax.CategoryVAT,
					Country:  "DE",
					Rate:     tax.RateGeneral,
				},
			},
		},
		{
			name: "effective percentage zero and amount zero uses effective percentage",
			input: []*stripe.InvoiceTotalTaxAmount{
				{
					Amount: 0, // Zero tax amount (truly exempt)
					TaxRate: &stripe.TaxRate{
						Country:             "DE",
						TaxType:             stripe.TaxRateTaxTypeVAT,
						EffectivePercentage: 0.0,
						Percentage:          19.0,
						Created:             time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
					},
				},
			},
			expected: tax.Set{
				{
					Category: tax.CategoryVAT,
					Country:  "DE",
					Percent:  num.NewPercentage(0, 3),
				},
			},
		},
		{
			name: "non-standard percentage fallback when effective is zero",
			input: []*stripe.InvoiceTotalTaxAmount{
				{
					Amount: 500, // Non-zero tax amount
					TaxRate: &stripe.TaxRate{
						Country:             "ES",
						TaxType:             stripe.TaxRateTaxTypeVAT,
						EffectivePercentage: 0.0,  // Stripe tax not used
						Percentage:          21.0, // Standard rate
						Created:             time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
					},
				},
			},
			expected: tax.Set{
				{
					Category: tax.CategoryVAT,
					Country:  "ES",
					Rate:     tax.RateGeneral,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblstripe.FromInvoiceTaxAmountsToTaxSet(tt.input, tax.RegimeDefFor(l10n.DE))
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFromCreditNoteTaxAmountNoStripeTax tests the fallback to Percentage when
// EffectivePercentage is 0 but there is a non-zero tax amount (Stripe tax not used).
func TestFromCreditNoteTaxAmountNoStripeTax(t *testing.T) {
	tests := []struct {
		name     string
		input    []*stripe.CreditNoteTaxAmount
		expected tax.Set
	}{
		{
			name: "effective percentage zero but amount non-zero uses percentage",
			input: []*stripe.CreditNoteTaxAmount{
				{
					Amount: 1900, // Non-zero tax amount
					TaxRate: &stripe.TaxRate{
						Country:             "DE",
						TaxType:             stripe.TaxRateTaxTypeVAT,
						EffectivePercentage: 0.0, // Stripe tax not used
						Percentage:          19.0,
						Created:             time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
					},
				},
			},
			expected: tax.Set{
				{
					Category: tax.CategoryVAT,
					Country:  "DE",
					Rate:     tax.RateGeneral,
				},
			},
		},
		{
			name: "effective percentage zero and amount zero uses effective percentage",
			input: []*stripe.CreditNoteTaxAmount{
				{
					Amount: 0, // Zero tax amount (truly exempt)
					TaxRate: &stripe.TaxRate{
						Country:             "DE",
						TaxType:             stripe.TaxRateTaxTypeVAT,
						EffectivePercentage: 0.0,
						Percentage:          19.0,
						Created:             time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
					},
				},
			},
			expected: tax.Set{
				{
					Category: tax.CategoryVAT,
					Country:  "DE",
					Percent:  num.NewPercentage(0, 3),
				},
			},
		},
		{
			name: "non-standard percentage fallback when effective is zero",
			input: []*stripe.CreditNoteTaxAmount{
				{
					Amount: 500, // Non-zero tax amount
					TaxRate: &stripe.TaxRate{
						Country:             "ES",
						TaxType:             stripe.TaxRateTaxTypeVAT,
						EffectivePercentage: 0.0,  // Stripe tax not used
						Percentage:          21.0, // Standard rate
						Created:             time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
					},
				},
			},
			expected: tax.Set{
				{
					Category: tax.CategoryVAT,
					Country:  "ES",
					Rate:     tax.RateGeneral,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblstripe.FromCreditNoteTaxAmountsToTaxSet(tt.input, tax.RegimeDefFor(l10n.DE))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFromInvoiceTaxAmountsReverseCharge(t *testing.T) {
	tests := []struct {
		name     string
		input    []*stripe.InvoiceTotalTaxAmount
		regime   l10n.Code
		expected tax.Set
	}{
		{
			name: "reverse charge with exempt rate available",
			input: []*stripe.InvoiceTotalTaxAmount{
				{
					TaxRate: &stripe.TaxRate{
						Country:             "GB",
						TaxType:             stripe.TaxRateTaxTypeVAT,
						EffectivePercentage: 0.0,
						Percentage:          20.0,
						Created:             time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
					},
					TaxabilityReason: stripe.InvoiceTotalTaxAmountTaxabilityReasonReverseCharge,
				},
			},
			regime: l10n.DE, // Germany regime looking at GB reverse charge
			expected: tax.Set{
				{
					Category: tax.CategoryVAT,
					Country:  l10n.DE.Tax(),
					Key:      tax.KeyReverseCharge,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblstripe.FromInvoiceTaxAmountsToTaxSet(tt.input, tax.RegimeDefFor(tt.regime))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFromCreditNoteTaxAmountsReverseCharge(t *testing.T) {
	tests := []struct {
		name     string
		input    []*stripe.CreditNoteTaxAmount
		regime   l10n.Code
		expected tax.Set
	}{
		{
			name: "credit note reverse charge with exempt rate available",
			input: []*stripe.CreditNoteTaxAmount{
				{
					TaxRate: &stripe.TaxRate{
						Country:             "GB",
						TaxType:             stripe.TaxRateTaxTypeVAT,
						EffectivePercentage: 0.0,
						Percentage:          20.0,
						Created:             time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
					},
					TaxabilityReason: stripe.CreditNoteTaxAmountTaxabilityReasonReverseCharge,
				},
			},
			regime: l10n.DE, // Germanyregime looking at GB reverse charge
			expected: tax.Set{
				{
					Category: tax.CategoryVAT,
					Country:  l10n.DE.Tax(),
					Key:      tax.KeyReverseCharge,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblstripe.FromCreditNoteTaxAmountsToTaxSet(tt.input, tax.RegimeDefFor(tt.regime))
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Credit Notes

func validCreditNoteLine() *stripe.CreditNoteLineItem {
	return &stripe.CreditNoteLineItem{
		ID:              "cnli_1Qk7SqQhcl5B85YlliBbmPiN",
		Object:          "credit_note_line_item",
		Amount:          10294,
		DiscountAmounts: []*stripe.CreditNoteLineItemDiscountAmount{},
		Livemode:        false,
		TaxAmounts: []*stripe.CreditNoteTaxAmount{
			{
				Amount:    2162,
				Inclusive: false,
				TaxRate: &stripe.TaxRate{
					TaxType:             stripe.TaxRateTaxTypeVAT,
					Country:             "ES",
					EffectivePercentage: 21.0,
					Percentage:          21.0,
				},
				TaxableAmount: 10294,
			},
		},
		TaxRates: []*stripe.TaxRate{},
		Type:     stripe.CreditNoteLineItemTypeInvoiceLineItem,
	}
}

func TestBasicFieldsCreditNote(t *testing.T) {
	line := validCreditNoteLine()
	result := goblstripe.FromCreditNoteLine(line, currency.EUR, tax.RegimeDefFor(l10n.DE))

	assert.NotNil(t, result, "Line conversion should not return nil")
	assert.Equal(t, currency.EUR, result.Item.Currency, "Item currency should match line currency")
	assert.Equal(t, tax.CategoryVAT, result.Taxes[0].Category, "Tax category should match line tax")
	assert.Equal(t, l10n.ES.Tax(), result.Taxes[0].Country, "Tax country should match line tax")
	assert.Equal(t, num.NewPercentage(210, 3), result.Taxes[0].Percent, "Tax percentage should match line tax")
	assert.Equal(t, 102.94, result.Item.Price.Float64(), "Item price should match line amount")
	assert.Equal(t, num.MakeAmount(1, 0), result.Quantity, "Quantity should be 1")
}

func TestCNLinePerUnit(t *testing.T) {
	line := validCreditNoteLine()
	line.UnitAmount = 5147
	line.Quantity = 2
	line.Description = "Stripe Addon"

	result := goblstripe.FromCreditNoteLine(line, currency.EUR, tax.RegimeDefFor(l10n.DE))

	assert.NotNil(t, result, "Line conversion should not return nil")
	assert.Equal(t, num.MakeAmount(2, 0), result.Quantity, "Quantity should match line quantity")
	assert.Equal(t, 51.47, result.Item.Price.Float64(), "Item price should match line amount")
	assert.Equal(t, "Stripe Addon", result.Item.Name, "Item name should match line description")
}

func TestCNLineTiered(t *testing.T) {
	line := validCreditNoteLine()
	line.UnitAmount = 10294
	line.Description = "Invopops"

	result := goblstripe.FromCreditNoteLine(line, currency.EUR, tax.RegimeDefFor(l10n.DE))

	assert.NotNil(t, result, "Line conversion should not return nil")
	assert.Equal(t, num.MakeAmount(1, 0), result.Quantity, "Quantity should match line quantity")
	assert.Equal(t, 102.94, result.Item.Price.Float64(), "Item price should match line amount")
	assert.Equal(t, "Invopops", result.Item.Name, "Item name should match line description")
}

func TestCNDiscounts(t *testing.T) {
	line := validCreditNoteLine()
	line.DiscountAmounts = []*stripe.CreditNoteLineItemDiscountAmount{
		{
			Amount: 1000,
		},
	}

	result := goblstripe.FromCreditNoteLine(line, currency.EUR, tax.RegimeDefFor(l10n.DE))
	assert.NotNil(t, result, "Line conversion should not return nil")
	assert.Equal(t, 10.0, result.Discounts[0].Amount.Float64(), "Discount amount should match line discount")

}

func TestUnitAmountNil(t *testing.T) {
	line := validCreditNoteLine()
	line.Quantity = 2

	result := goblstripe.FromCreditNoteLine(line, currency.EUR, tax.RegimeDefFor(l10n.DE))
	assert.NotNil(t, result, "Line conversion should not return nil")

	assert.Equal(t, num.MakeAmount(2, 0), result.Quantity, "Quantity should match line quantity")
	assert.Equal(t, 51.47, result.Item.Price.Float64(), "Item price should match line amount")
}

func TestFromCNTaxAmountsToTaxSet(t *testing.T) {
	tests := []struct {
		name     string
		input    []*stripe.CreditNoteTaxAmount
		expected tax.Set
	}{
		{
			name: "multiple tax amounts",
			input: []*stripe.CreditNoteTaxAmount{
				{
					TaxRate: &stripe.TaxRate{
						Country:             "DE",
						TaxType:             stripe.TaxRateTaxTypeVAT,
						EffectivePercentage: 19.0,
						Percentage:          19.0,
						Created:             time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
					},
				},
				{
					TaxRate: &stripe.TaxRate{
						Country:             "ES",
						TaxType:             stripe.TaxRateTaxTypeVAT,
						EffectivePercentage: 8.875,
						Percentage:          8.875,
						Created:             time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
					},
				},
			},
			expected: tax.Set{
				{
					Category: tax.CategoryVAT,
					Country:  "DE",
					Rate:     tax.RateGeneral,
				},
				{
					Category: tax.CategoryVAT,
					Country:  "ES",
					Percent:  num.NewPercentage(8875, 5),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblstripe.FromCreditNoteTaxAmountsToTaxSet(tt.input, tax.RegimeDefFor(l10n.DE))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFreeTrialLine(t *testing.T) {
	line := &stripe.InvoiceLineItem{
		ID:                 "il_1234567890abcd",
		Amount:             0,
		AmountExcludingTax: 0,
		Currency:           stripe.CurrencyMXN,
		Quantity:           1,
		Price: &stripe.Price{
			BillingScheme: stripe.PriceBillingSchemePerUnit,
			Currency:      stripe.CurrencyMXN,
			TaxBehavior:   stripe.PriceTaxBehaviorUnspecified,
			UnitAmount:    172840,
		},
		Period: &stripe.Period{
			Start: 1750690831,
			End:   1752505231,
		},
		Description: "Período de prueba para Plan Avanzado",
		TaxAmounts: []*stripe.InvoiceTotalTaxAmount{
			{
				Amount:    0,
				Inclusive: true,
				TaxRate: &stripe.TaxRate{
					TaxType:             stripe.TaxRateTaxTypeVAT,
					Country:             "MX",
					EffectivePercentage: 16.0,
					Percentage:          16.0,
				},
				TaxabilityReason: stripe.InvoiceTotalTaxAmountTaxabilityReasonStandardRated,
				TaxableAmount:    0,
			},
		},
		UnitAmountExcludingTax: 0,
	}

	regimeDef := tax.RegimeDefFor(l10n.MX)

	result := goblstripe.FromInvoiceLine(line, regimeDef)

	assert.NotNil(t, result, "Line conversion should not return nil")
	assert.Equal(t, num.MakeAmount(1, 0), result.Quantity, "Quantity should match line quantity")
	assert.Equal(t, "Período de prueba para Plan Avanzado", result.Item.Name, "Item name should match line description")
	assert.Equal(t, currency.MXN, result.Item.Currency, "Item currency should match line currency")
	assert.Equal(t, 0.0, result.Item.Price.Float64(), "Item price should be 0 for free trial")
	assert.Equal(t, tax.CategoryVAT, result.Taxes[0].Category, "Tax category should match line tax")
	assert.Equal(t, l10n.MX.Tax(), result.Taxes[0].Country, "Tax country should match line tax")
}

// Test resolveInvoiceLinePrice edge cases (lines 112-120)
func TestResolveInvoiceLinePriceZeroQuantityWithPrice(t *testing.T) {
	// When quantity is zero but Price is available, should use Price.UnitAmount
	line := &stripe.InvoiceLineItem{
		ID:          "il_zero_qty_with_price",
		Amount:      0,
		Currency:    stripe.CurrencyEUR,
		Description: "Zero quantity item with price",
		Price: &stripe.Price{
			BillingScheme: stripe.PriceBillingSchemePerUnit,
			Currency:      stripe.CurrencyEUR,
			UnitAmount:    5000,
		},
		Period: &stripe.Period{
			Start: 1736351413,
			End:   1739029692,
		},
	}

	result := goblstripe.FromInvoiceLine(line, tax.RegimeDefFor(l10n.DE))

	assert.NotNil(t, result, "Line conversion should not return nil")
	assert.Equal(t, num.AmountZero, result.Quantity, "Quantity should be zero")
	assert.Equal(t, 50.0, result.Item.Price.Float64(), "Item price should use UnitAmount when quantity is zero")
}

func TestResolveInvoiceLinePriceZeroQuantityNilPrice(t *testing.T) {
	// When quantity is zero and Price is nil, should return AmountZero
	line := &stripe.InvoiceLineItem{
		ID:          "il_zero_qty_nil_price",
		Amount:      5000,
		Currency:    stripe.CurrencyEUR,
		Description: "Zero quantity item without price",
		Price:       nil, // No price object
		Period: &stripe.Period{
			Start: 1736351413,
			End:   1739029692,
		},
	}

	result := goblstripe.FromInvoiceLine(line, tax.RegimeDefFor(l10n.DE))

	assert.NotNil(t, result, "Line conversion should not return nil")
	assert.Equal(t, num.AmountZero, result.Quantity, "Quantity should be zero")
	assert.Equal(t, 0.0, result.Item.Price.Float64(), "Item price should be zero when quantity is zero and no price")
}

func TestResolveInvoiceLinePriceNonZeroQuantity(t *testing.T) {
	// When quantity is not zero, should divide Amount by quantity
	line := &stripe.InvoiceLineItem{
		ID:       "il_nonzero_qty",
		Amount:   10000,
		Currency: stripe.CurrencyEUR,
		Quantity: 4,
		Price: &stripe.Price{
			BillingScheme: stripe.PriceBillingSchemePerUnit,
			Currency:      stripe.CurrencyEUR,
			UnitAmount:    2500,
		},
		Description: "Item with quantity",
		Period: &stripe.Period{
			Start: 1736351413,
			End:   1739029692,
		},
	}

	result := goblstripe.FromInvoiceLine(line, tax.RegimeDefFor(l10n.DE))

	assert.NotNil(t, result, "Line conversion should not return nil")
	assert.Equal(t, num.MakeAmount(4, 0), result.Quantity, "Quantity should be 4")
	assert.Equal(t, 25.0, result.Item.Price.Float64(), "Item price should be Amount/Quantity = 10000/4 = 2500 cents = 25.00")
}

// Test FromInvoiceLineDiscount with zero amount (lines 136-138)
func TestFromInvoiceLineDiscountZeroAmount(t *testing.T) {
	discount := &stripe.InvoiceLineItemDiscountAmount{
		Amount: 0, // Zero amount discount
		Discount: &stripe.Discount{
			Coupon: &stripe.Coupon{
				Name: "Zero Discount",
			},
		},
	}

	result := goblstripe.FromInvoiceLineDiscount(discount, stripe.CurrencyEUR)

	assert.Nil(t, result, "Discount with zero amount should return nil")
}

// Test FromInvoiceLineDiscounts filtering zero-amount discounts
func TestFromInvoiceLineDiscountsFiltersZeroAmount(t *testing.T) {
	discounts := []*stripe.InvoiceLineItemDiscountAmount{
		{
			Amount: 1000,
			Discount: &stripe.Discount{
				Coupon: &stripe.Coupon{Name: "Valid Discount"},
			},
		},
		{
			Amount: 0, // Should be filtered out
			Discount: &stripe.Discount{
				Coupon: &stripe.Coupon{Name: "Zero Discount"},
			},
		},
		{
			Amount: 500,
			Discount: &stripe.Discount{
				Coupon: &stripe.Coupon{Name: "Another Valid Discount"},
			},
		},
	}

	result := goblstripe.FromInvoiceLineDiscounts(discounts, stripe.CurrencyEUR)

	assert.Len(t, result, 2, "Should have 2 discounts after filtering zero-amount")
	assert.Equal(t, "Valid Discount", result[0].Reason)
	assert.Equal(t, "Another Valid Discount", result[1].Reason)
}

// Test FromInvoiceLines filtering (lines 23-28)
func TestFromInvoiceLinesEmptyInput(t *testing.T) {
	lines := []*stripe.InvoiceLineItem{}

	result := goblstripe.FromInvoiceLines(lines, tax.RegimeDefFor(l10n.DE))

	assert.NotNil(t, result, "Result should not be nil")
	assert.Len(t, result, 0, "Result should be empty for empty input")
}

func TestFromInvoiceLinesAllValid(t *testing.T) {
	lines := []*stripe.InvoiceLineItem{
		{
			ID:       "il_1",
			Amount:   5000,
			Currency: stripe.CurrencyEUR,
			Quantity: 1,
			Price: &stripe.Price{
				BillingScheme: stripe.PriceBillingSchemePerUnit,
				Currency:      stripe.CurrencyEUR,
				UnitAmount:    5000,
			},
			Description: "First line",
			Period: &stripe.Period{
				Start: 1736351413,
				End:   1739029692,
			},
		},
		{
			ID:       "il_2",
			Amount:   3000,
			Currency: stripe.CurrencyEUR,
			Quantity: 1,
			Price: &stripe.Price{
				BillingScheme: stripe.PriceBillingSchemePerUnit,
				Currency:      stripe.CurrencyEUR,
				UnitAmount:    3000,
			},
			Description: "Second line",
			Period: &stripe.Period{
				Start: 1736351413,
				End:   1739029692,
			},
		},
	}

	result := goblstripe.FromInvoiceLines(lines, tax.RegimeDefFor(l10n.DE))

	assert.Len(t, result, 2, "Should have 2 converted lines")
	assert.Equal(t, "First line", result[0].Item.Name)
	assert.Equal(t, "Second line", result[1].Item.Name)
}

// Test FromCreditNoteLineDiscount with zero amount (lines 289-291)
func TestFromCreditNoteLineDiscountZeroAmount(t *testing.T) {
	discount := &stripe.CreditNoteLineItemDiscountAmount{
		Amount: 0, // Zero amount discount
		Discount: &stripe.Discount{
			Coupon: &stripe.Coupon{
				Name: "Zero Discount",
			},
		},
	}

	result := goblstripe.FromCreditNoteLineDiscount(discount, currency.EUR)

	assert.Nil(t, result, "Credit note discount with zero amount should return nil")
}

// Test FromCreditNoteLineDiscounts filtering zero-amount discounts (lines 278-283)
func TestFromCreditNoteLineDiscountsFiltersZeroAmount(t *testing.T) {
	discounts := []*stripe.CreditNoteLineItemDiscountAmount{
		{
			Amount: 1500,
			Discount: &stripe.Discount{
				Coupon: &stripe.Coupon{Name: "CN Valid Discount"},
			},
		},
		{
			Amount: 0, // Should be filtered out
			Discount: &stripe.Discount{
				Coupon: &stripe.Coupon{Name: "CN Zero Discount"},
			},
		},
		{
			Amount: 750,
			Discount: &stripe.Discount{
				Coupon: &stripe.Coupon{Name: "CN Another Valid Discount"},
			},
		},
	}

	result := goblstripe.FromCreditNoteLineDiscounts(discounts, currency.EUR)

	assert.Len(t, result, 2, "Should have 2 discounts after filtering zero-amount")
	assert.Equal(t, "CN Valid Discount", result[0].Reason)
	assert.Equal(t, "CN Another Valid Discount", result[1].Reason)
}

func TestFromCreditNoteLineDiscountsEmptyInput(t *testing.T) {
	discounts := []*stripe.CreditNoteLineItemDiscountAmount{}

	result := goblstripe.FromCreditNoteLineDiscounts(discounts, currency.EUR)

	assert.NotNil(t, result, "Result should not be nil")
	assert.Len(t, result, 0, "Result should be empty for empty input")
}

func TestFromCreditNoteLineDiscountsAllZeroAmount(t *testing.T) {
	discounts := []*stripe.CreditNoteLineItemDiscountAmount{
		{
			Amount: 0,
			Discount: &stripe.Discount{
				Coupon: &stripe.Coupon{Name: "Zero 1"},
			},
		},
		{
			Amount: 0,
			Discount: &stripe.Discount{
				Coupon: &stripe.Coupon{Name: "Zero 2"},
			},
		},
	}

	result := goblstripe.FromCreditNoteLineDiscounts(discounts, currency.EUR)

	assert.NotNil(t, result, "Result should not be nil")
	assert.Len(t, result, 0, "Result should be empty when all discounts are zero")
}
