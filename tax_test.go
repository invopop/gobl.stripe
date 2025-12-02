package goblstripe_test

import (
	"testing"

	goblstripe "github.com/invopop/gobl.stripe"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v84"
)

func TestTaxInclusive(t *testing.T) {

	s := completeStripeInvoice()

	// Check that taxes are exclusive (default)
	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)
	assert.Nil(t, gi.Tax)

	// When tax is inclusive, the line amounts already include tax
	// So the Total should be the sum of line amounts (18999) not sum + tax (22609)
	s.TotalTaxAmounts[0].Inclusive = true
	// Update line tax amounts to also be inclusive
	for _, line := range s.Lines.Data {
		for _, tax := range line.TaxAmounts {
			tax.Inclusive = true
		}
	}
	s.Total = 18999 // Lines total when tax is already included in prices
	gi, err = goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)
	assert.Equal(t, tax.CategoryVAT, gi.Tax.PricesInclude)
}

func TestTaxFromInvoiceTaxAmounts(t *testing.T) {
	t.Run("empty tax amounts returns nil", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.TotalTaxAmounts = []*stripe.InvoiceTotalTaxAmount{}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)
		assert.Nil(t, gi.Tax)
	})

	t.Run("nil tax amounts returns nil", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.TotalTaxAmounts = nil

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)
		assert.Nil(t, gi.Tax)
	})

	t.Run("tax without type and display name returns nil", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.TotalTaxAmounts = []*stripe.InvoiceTotalTaxAmount{
			{
				Amount:    100,
				Inclusive: false,
				TaxRate: &stripe.TaxRate{
					TaxType:     "",
					DisplayName: "",
					Country:     "DE",
					Percentage:  19.0,
				},
			},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)
		assert.Nil(t, gi.Tax)
	})

	t.Run("exclusive tax returns nil", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.TotalTaxAmounts = []*stripe.InvoiceTotalTaxAmount{
			{
				Amount:    100,
				Inclusive: false,
				TaxRate: &stripe.TaxRate{
					TaxType:    stripe.TaxRateTaxTypeVAT,
					Country:    "DE",
					Percentage: 19.0,
				},
			},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)
		assert.Nil(t, gi.Tax)
	})

	t.Run("inclusive VAT tax creates tax object", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.TotalTaxAmounts = []*stripe.InvoiceTotalTaxAmount{
			{
				Amount:    100,
				Inclusive: true,
				TaxRate: &stripe.TaxRate{
					TaxType:    stripe.TaxRateTaxTypeVAT,
					Country:    "DE",
					Percentage: 19.0,
				},
			},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)
		require.NotNil(t, gi.Tax)
		assert.Equal(t, tax.CategoryVAT, gi.Tax.PricesInclude)
	})

	t.Run("inclusive Sales Tax creates tax object", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.TotalTaxAmounts = []*stripe.InvoiceTotalTaxAmount{
			{
				Amount:    100,
				Inclusive: true,
				TaxRate: &stripe.TaxRate{
					TaxType:    stripe.TaxRateTaxTypeSalesTax,
					Country:    "US",
					Percentage: 8.5,
				},
			},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)
		require.NotNil(t, gi.Tax)
		assert.Equal(t, tax.CategoryST, gi.Tax.PricesInclude)
	})

	t.Run("inclusive GST creates tax object", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.TotalTaxAmounts = []*stripe.InvoiceTotalTaxAmount{
			{
				Amount:    100,
				Inclusive: true,
				TaxRate: &stripe.TaxRate{
					TaxType:    stripe.TaxRateTaxTypeGST,
					Country:    "AU",
					Percentage: 10.0,
				},
			},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)
		require.NotNil(t, gi.Tax)
		assert.Equal(t, tax.CategoryGST, gi.Tax.PricesInclude)
	})
}

func TestTaxFromCreditNoteTaxAmounts(t *testing.T) {
	t.Run("empty tax amounts returns nil", func(t *testing.T) {
		s := validCreditNote()
		s.TaxAmounts = []*stripe.CreditNoteTaxAmount{}

		gi, err := goblstripe.FromCreditNote(s, validStripeAccount())
		require.NoError(t, err)
		assert.Nil(t, gi.Tax)
	})

	t.Run("nil tax amounts returns nil", func(t *testing.T) {
		s := validCreditNote()
		s.TaxAmounts = nil

		gi, err := goblstripe.FromCreditNote(s, validStripeAccount())
		require.NoError(t, err)
		assert.Nil(t, gi.Tax)
	})

	t.Run("tax without type and display name returns nil", func(t *testing.T) {
		s := validCreditNote()
		s.TaxAmounts = []*stripe.CreditNoteTaxAmount{
			{
				Amount:    100,
				Inclusive: false,
				TaxRate: &stripe.TaxRate{
					TaxType:     "",
					DisplayName: "",
					Country:     "DE",
					Percentage:  19.0,
				},
			},
		}

		gi, err := goblstripe.FromCreditNote(s, validStripeAccount())
		require.NoError(t, err)
		assert.Nil(t, gi.Tax)
	})

	t.Run("exclusive tax returns nil", func(t *testing.T) {
		s := validCreditNote()
		s.TaxAmounts = []*stripe.CreditNoteTaxAmount{
			{
				Amount:    100,
				Inclusive: false,
				TaxRate: &stripe.TaxRate{
					TaxType:    stripe.TaxRateTaxTypeVAT,
					Country:    "DE",
					Percentage: 19.0,
				},
			},
		}

		gi, err := goblstripe.FromCreditNote(s, validStripeAccount())
		require.NoError(t, err)
		assert.Nil(t, gi.Tax)
	})

	t.Run("inclusive VAT tax creates tax object", func(t *testing.T) {
		s := validCreditNote()
		s.TaxAmounts = []*stripe.CreditNoteTaxAmount{
			{
				Amount:    100,
				Inclusive: true,
				TaxRate: &stripe.TaxRate{
					TaxType:    stripe.TaxRateTaxTypeVAT,
					Country:    "DE",
					Percentage: 19.0,
				},
			},
		}
		// Update line tax amounts to also be inclusive
		for _, line := range s.Lines.Data {
			for _, tax := range line.TaxAmounts {
				tax.Inclusive = true
			}
		}
		// When tax is inclusive, Total = line amount (10294) since tax is already in the price
		s.Total = 10294

		gi, err := goblstripe.FromCreditNote(s, validStripeAccount())
		require.NoError(t, err)
		require.NotNil(t, gi.Tax)
		assert.Equal(t, tax.CategoryVAT, gi.Tax.PricesInclude)
	})

	t.Run("inclusive GST creates tax object", func(t *testing.T) {
		s := validCreditNote()
		s.TaxAmounts = []*stripe.CreditNoteTaxAmount{
			{
				Amount:    100,
				Inclusive: true,
				TaxRate: &stripe.TaxRate{
					TaxType:    stripe.TaxRateTaxTypeGST,
					Country:    "AU",
					Percentage: 10.0,
				},
			},
		}

		gi, err := goblstripe.FromCreditNote(s, validStripeAccount())
		require.NoError(t, err)
		require.NotNil(t, gi.Tax)
		assert.Equal(t, tax.CategoryGST, gi.Tax.PricesInclude)
	})
}

func TestExtractTaxCat(t *testing.T) {
	// Test extractTaxCat via FromInvoice since the function is not exportable
	t.Run("VAT from tax type", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.TotalTaxAmounts = []*stripe.InvoiceTotalTaxAmount{
			{
				Amount:    100,
				Inclusive: true,
				TaxRate: &stripe.TaxRate{
					TaxType:    stripe.TaxRateTaxTypeVAT,
					Country:    "DE",
					Percentage: 19.0,
				},
			},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)
		require.NotNil(t, gi.Tax)
		assert.Equal(t, tax.CategoryVAT, gi.Tax.PricesInclude)
	})

	t.Run("Sales Tax from tax type", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.TotalTaxAmounts = []*stripe.InvoiceTotalTaxAmount{
			{
				Amount:    100,
				Inclusive: true,
				TaxRate: &stripe.TaxRate{
					TaxType:    stripe.TaxRateTaxTypeSalesTax,
					Country:    "US",
					Percentage: 8.5,
				},
			},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)
		require.NotNil(t, gi.Tax)
		assert.Equal(t, tax.CategoryST, gi.Tax.PricesInclude)
	})

	t.Run("GST from tax type", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.TotalTaxAmounts = []*stripe.InvoiceTotalTaxAmount{
			{
				Amount:    100,
				Inclusive: true,
				TaxRate: &stripe.TaxRate{
					TaxType:    stripe.TaxRateTaxTypeGST,
					Country:    "AU",
					Percentage: 10.0,
				},
			},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)
		require.NotNil(t, gi.Tax)
		assert.Equal(t, tax.CategoryGST, gi.Tax.PricesInclude)
	})

	t.Run("VAT from display name", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.TotalTaxAmounts = []*stripe.InvoiceTotalTaxAmount{
			{
				Amount:    100,
				Inclusive: true,
				TaxRate: &stripe.TaxRate{
					TaxType:     "",
					DisplayName: "VAT",
					Country:     "DE",
					Percentage:  19.0,
				},
			},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)
		require.NotNil(t, gi.Tax)
		assert.Equal(t, tax.CategoryVAT, gi.Tax.PricesInclude)
	})

	t.Run("IVA from display name", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.TotalTaxAmounts = []*stripe.InvoiceTotalTaxAmount{
			{
				Amount:    100,
				Inclusive: true,
				TaxRate: &stripe.TaxRate{
					TaxType:     "",
					DisplayName: "IVA",
					Country:     "ES",
					Percentage:  21.0,
				},
			},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)
		require.NotNil(t, gi.Tax)
		assert.Equal(t, tax.CategoryVAT, gi.Tax.PricesInclude)
	})

	t.Run("Sales Tax from display name", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.TotalTaxAmounts = []*stripe.InvoiceTotalTaxAmount{
			{
				Amount:    100,
				Inclusive: true,
				TaxRate: &stripe.TaxRate{
					TaxType:     "",
					DisplayName: "Sales Tax",
					Country:     "US",
					Percentage:  8.5,
				},
			},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)
		require.NotNil(t, gi.Tax)
		assert.Equal(t, tax.CategoryST, gi.Tax.PricesInclude)
	})

	t.Run("GST from display name", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.TotalTaxAmounts = []*stripe.InvoiceTotalTaxAmount{
			{
				Amount:    100,
				Inclusive: true,
				TaxRate: &stripe.TaxRate{
					TaxType:     "",
					DisplayName: "GST",
					Country:     "AU",
					Percentage:  10.0,
				},
			},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)
		require.NotNil(t, gi.Tax)
		assert.Equal(t, tax.CategoryGST, gi.Tax.PricesInclude)
	})

	t.Run("unknown tax type and display name", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.TotalTaxAmounts = []*stripe.InvoiceTotalTaxAmount{
			{
				Amount:    100,
				Inclusive: true,
				TaxRate: &stripe.TaxRate{
					TaxType:     "",
					DisplayName: "Unknown Tax",
					Country:     "XX",
					Percentage:  15.0,
				},
			},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)
		require.NotNil(t, gi.Tax)
		// Should return empty code for unknown tax types
		assert.Equal(t, "Unknown Tax", string(gi.Tax.PricesInclude))
	})
}
