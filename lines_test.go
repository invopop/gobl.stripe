package goblstripe_test

import (
	"context"
	"testing"

	goblstripe "github.com/invopop/gobl.stripe"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
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

	issueDate := cal.NewDate(2024, 12, 25)
	result := goblstripe.FromLines(lines, false, *issueDate)
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

	issueDate := cal.NewDate(2024, 12, 25)
	result := goblstripe.FromLine(line, false, *issueDate)

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
			{Amount: 5000},
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

	issueDate := cal.NewDate(2024, 12, 25)
	result := goblstripe.FromLine(line, false, *issueDate)

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
