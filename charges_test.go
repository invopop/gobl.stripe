package goblstripe_test

import (
	"testing"

	goblstripe "github.com/invopop/gobl.stripe"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v81"
)

// taxedStripeInvoice builds a minimal invoice whose single line carries the given
// tax, with the totals Stripe would report for it.
func taxedStripeInvoice(taxRate *stripe.TaxRate, amount int64, inclusive bool) *stripe.Invoice {
	s := minimalStripeInvoice()
	ta := &stripe.InvoiceTotalTaxAmount{
		Amount:        amount,
		Inclusive:     inclusive,
		TaxableAmount: 2000,
		TaxRate:       taxRate,
	}
	s.TotalTaxAmounts = []*stripe.InvoiceTotalTaxAmount{ta}
	s.Lines.Data[0].TaxAmounts = []*stripe.InvoiceTotalTaxAmount{ta}
	if !inclusive {
		s.Total = 2000 + amount
	}
	return s
}

func chicagoLeaseTaxRate() *stripe.TaxRate {
	return &stripe.TaxRate{
		TaxType:             stripe.TaxRateTaxTypeLeaseTax,
		DisplayName:         "Chicago Lease Tax",
		Country:             "US",
		State:               "IL",
		Jurisdiction:        "Chicago",
		Percentage:          15.0,
		EffectivePercentage: 15.0,
	}
}

func TestChargesFromUnsupportedTaxes(t *testing.T) {
	t.Run("lease tax becomes a document charge", func(t *testing.T) {
		s := taxedStripeInvoice(chicagoLeaseTaxRate(), 300, false)

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.Len(t, gi.Charges, 1)
		charge := gi.Charges[0]
		assert.Equal(t, bill.ChargeKeyTax, charge.Key)
		assert.Equal(t, "lease_tax", charge.Code.String())
		assert.Equal(t, "Chicago Lease Tax", charge.Reason)
		assert.Equal(t, "3.00", charge.Amount.String())

		assert.Empty(t, gi.Lines[0].Taxes)
		assert.Equal(t, "23.00", gi.Totals.TotalWithTax.String())
	})

	t.Run("tax type without an SDK constant becomes a charge", func(t *testing.T) {
		s := taxedStripeInvoice(&stripe.TaxRate{
			TaxType:     stripe.TaxRateTaxType("gross_receipts_tax"),
			DisplayName: "Gross Receipts Tax",
			Country:     "US",
			Percentage:  5.0,
		}, 100, false)

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.Len(t, gi.Charges, 1)
		assert.Equal(t, "Gross Receipts Tax", gi.Charges[0].Reason)
		assert.Equal(t, "1.00", gi.Charges[0].Amount.String())
	})

	t.Run("category missing from the regime becomes a charge", func(t *testing.T) {
		s := taxedStripeInvoice(&stripe.TaxRate{
			TaxType:     stripe.TaxRateTaxTypeGST,
			DisplayName: "GST",
			Country:     "US",
			Percentage:  10.0,
		}, 200, false)

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.Len(t, gi.Charges, 1)
		assert.Equal(t, "GST", gi.Charges[0].Reason)
		assert.Empty(t, gi.Lines[0].Taxes)
	})

	t.Run("inclusive unsupported tax is not charged twice", func(t *testing.T) {
		s := taxedStripeInvoice(chicagoLeaseTaxRate(), 300, true)

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		assert.Empty(t, gi.Charges)
		assert.Equal(t, "20.00", gi.Totals.TotalWithTax.String())
	})

	t.Run("unexpanded tax rate is not charged", func(t *testing.T) {
		s := taxedStripeInvoice(&stripe.TaxRate{ID: "txr_unexpanded"}, 0, false)

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		assert.Empty(t, gi.Charges)
	})

	t.Run("supported tax is still a tax", func(t *testing.T) {
		s := taxedStripeInvoice(&stripe.TaxRate{
			TaxType:             stripe.TaxRateTaxTypeVAT,
			DisplayName:         "VAT",
			Country:             "DE",
			Percentage:          19.0,
			EffectivePercentage: 19.0,
		}, 380, false)

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		assert.Empty(t, gi.Charges)
		require.Len(t, gi.Lines[0].Taxes, 1)
		assert.Equal(t, tax.CategoryVAT, gi.Lines[0].Taxes[0].Category)
	})

	t.Run("japanese consumption tax maps to VAT", func(t *testing.T) {
		s := taxedStripeInvoice(&stripe.TaxRate{
			TaxType:             stripe.TaxRateTaxTypeJCT,
			DisplayName:         "JCT",
			Country:             "JP",
			Percentage:          10.0,
			EffectivePercentage: 10.0,
		}, 200, false)

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		assert.Empty(t, gi.Charges)
		require.Len(t, gi.Lines[0].Taxes, 1)
		assert.Equal(t, tax.CategoryVAT, gi.Lines[0].Taxes[0].Category)
	})
}

func TestChargesFromCreditNoteUnsupportedTaxes(t *testing.T) {
	s := validCreditNote()
	ta := &stripe.CreditNoteTaxAmount{
		Amount:        300,
		Inclusive:     false,
		TaxableAmount: 10294,
		TaxRate:       chicagoLeaseTaxRate(),
	}
	s.TaxAmounts = []*stripe.CreditNoteTaxAmount{ta}
	for _, line := range s.Lines.Data {
		line.TaxAmounts = []*stripe.CreditNoteTaxAmount{ta}
	}
	s.Total = 10294 + 300

	gi, err := goblstripe.FromCreditNote(s, validStripeAccount())
	require.NoError(t, err)

	require.Len(t, gi.Charges, 1)
	assert.Equal(t, bill.ChargeKeyTax, gi.Charges[0].Key)
	assert.Equal(t, "Chicago Lease Tax", gi.Charges[0].Reason)
	assert.Equal(t, "3.00", gi.Charges[0].Amount.String())
}
