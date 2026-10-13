package goblstripe_test

import (
	"testing"

	"github.com/invopop/gobl"
	goblstripe "github.com/invopop/gobl.stripe"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/regimes/ca"
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
	t.Run("lease tax becomes a line charge", func(t *testing.T) {
		s := taxedStripeInvoice(chicagoLeaseTaxRate(), 300, false)

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.Len(t, gi.Lines[0].Charges, 1)
		charge := gi.Lines[0].Charges[0]
		assert.Equal(t, bill.ChargeKeyTax, charge.Key)
		assert.Equal(t, "lease_tax", charge.Code.String())
		assert.Equal(t, "Chicago Lease Tax", charge.Reason)
		assert.Equal(t, "3.00", charge.Amount.String())

		assert.Empty(t, gi.Lines[0].Taxes)
		assert.Equal(t, "23.00", gi.Totals.TotalWithTax.String())
	})

	t.Run("tax type we do not know at all becomes a charge", func(t *testing.T) {
		s := taxedStripeInvoice(&stripe.TaxRate{
			TaxType:     stripe.TaxRateTaxType("some_future_tax"),
			DisplayName: "Some Future Tax",
			Country:     "US",
			Percentage:  5.0,
		}, 100, false)

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.Len(t, gi.Lines[0].Charges, 1)
		assert.Equal(t, "Some Future Tax", gi.Lines[0].Charges[0].Reason)
		assert.Equal(t, "1.00", gi.Lines[0].Charges[0].Amount.String())
	})

	t.Run("unknown tax type with no display name is named after the charge", func(t *testing.T) {
		s := taxedStripeInvoice(&stripe.TaxRate{
			TaxType:    stripe.TaxRateTaxType("gross_receipts_tax"),
			Country:    "US",
			Percentage: 5.0,
		}, 100, false)

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.Len(t, gi.Lines[0].Charges, 1)
		assert.Equal(t, "Gross Receipts Tax", gi.Lines[0].Charges[0].Reason)
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

		require.Len(t, gi.Lines[0].Charges, 1)
		assert.Equal(t, "GST", gi.Lines[0].Charges[0].Reason)
		assert.Empty(t, gi.Lines[0].Taxes)
	})

	t.Run("inclusive unsupported tax is not charged twice", func(t *testing.T) {
		s := taxedStripeInvoice(chicagoLeaseTaxRate(), 300, true)

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		assert.Empty(t, gi.Lines[0].Charges)
		assert.Equal(t, "20.00", gi.Totals.TotalWithTax.String())
	})

	t.Run("unexpanded tax rate is not charged", func(t *testing.T) {
		s := taxedStripeInvoice(&stripe.TaxRate{ID: "txr_unexpanded"}, 0, false)

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		assert.Empty(t, gi.Lines[0].Charges)
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

		assert.Empty(t, gi.Lines[0].Charges)
		require.Len(t, gi.Lines[0].Taxes, 1)
		assert.Equal(t, tax.CategoryVAT, gi.Lines[0].Taxes[0].Category)
	})

	t.Run("harmonized sales tax maps to the canadian HST category", func(t *testing.T) {
		s := taxedStripeInvoice(&stripe.TaxRate{
			TaxType:             stripe.TaxRateTaxTypeHST,
			DisplayName:         "HST",
			Country:             "CA",
			Percentage:          13.0,
			EffectivePercentage: 13.0,
		}, 260, false)

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		assert.Empty(t, gi.Lines[0].Charges)
		require.Len(t, gi.Lines[0].Taxes, 1)
		assert.Equal(t, ca.TaxCategoryHST, gi.Lines[0].Taxes[0].Category)
	})

	t.Run("provincial sales tax maps to the canadian PST category", func(t *testing.T) {
		s := taxedStripeInvoice(&stripe.TaxRate{
			TaxType:             stripe.TaxRateTaxTypePST,
			DisplayName:         "PST",
			Country:             "CA",
			Percentage:          7.0,
			EffectivePercentage: 7.0,
		}, 140, false)

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		assert.Empty(t, gi.Lines[0].Charges)
		require.Len(t, gi.Lines[0].Taxes, 1)
		assert.Equal(t, ca.TaxCategoryPST, gi.Lines[0].Taxes[0].Category)
	})

	t.Run("quebec sales tax has no GOBL category and becomes a charge", func(t *testing.T) {
		s := taxedStripeInvoice(&stripe.TaxRate{
			TaxType:     stripe.TaxRateTaxTypeQST,
			DisplayName: "QST",
			Country:     "CA",
			Percentage:  9.975,
		}, 200, false)

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		assert.Empty(t, gi.Lines[0].Taxes)
		require.Len(t, gi.Lines[0].Charges, 1)
		assert.Equal(t, "qst", gi.Lines[0].Charges[0].Code.String())
	})

	t.Run("japanese consumption tax becomes a charge", func(t *testing.T) {
		s := taxedStripeInvoice(&stripe.TaxRate{
			TaxType:             stripe.TaxRateTaxTypeJCT,
			DisplayName:         "JCT",
			Country:             "JP",
			Percentage:          10.0,
			EffectivePercentage: 10.0,
		}, 200, false)

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		assert.Empty(t, gi.Lines[0].Taxes)
		require.Len(t, gi.Lines[0].Charges, 1)
		assert.Equal(t, "jct", gi.Lines[0].Charges[0].Code.String())
		assert.Equal(t, "JCT", gi.Lines[0].Charges[0].Reason)
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

	require.Len(t, gi.Lines[0].Charges, 1)
	assert.Equal(t, bill.ChargeKeyTax, gi.Lines[0].Charges[0].Key)
	assert.Equal(t, "Chicago Lease Tax", gi.Lines[0].Charges[0].Reason)
	assert.Equal(t, "3.00", gi.Lines[0].Charges[0].Amount.String())
}

// documentedUnsupportedTaxes lists every tax type Stripe documents that GOBL has
// no tax category for, with the name Stripe gives it, taken from
// https://docs.stripe.com/api/tax_rates/object#tax_rate_object-tax_type.
var documentedUnsupportedTaxes = map[string]string{
	"admissions_tax":           "Admissions Tax",
	"amusement_tax":            "Amusement Tax",
	"attendance_tax":           "Attendance Tax",
	"communications_tax":       "Communications Tax",
	"digital_excise_tax":       "Digital Excise Tax",
	"entertainment_tax":        "Entertainment Tax",
	"gross_receipts_tax":       "Gross Receipts Tax",
	"hospitality_tax":          "Hospitality Tax",
	"jct":                      "Japanese Consumption Tax",
	"lease_tax":                "Chicago Lease Tax",
	"luxury_tax":               "Luxury Tax",
	"mass_transit_parking_tax": "Mass Transit Parking Tax",
	"parking_tax":              "Parking Tax",
	"qst":                      "Quebec Sales Tax",
	"resort_tax":               "Resort Tax",
	"retail_delivery_fee":      "Retail Delivery Fee",
	"rst":                      "Retail Sales Tax",
	"service_tax":              "Service Tax",
	"tourism_tax":              "Tourism Tax",
	"utility_users_tax":        "Utility Users Tax",
}

func TestEveryUnsupportedTaxTypeIsCharged(t *testing.T) {
	for taxType, name := range documentedUnsupportedTaxes {
		t.Run(taxType, func(t *testing.T) {
			s := taxedStripeInvoice(&stripe.TaxRate{
				TaxType:    stripe.TaxRateTaxType(taxType),
				Percentage: 10.0,
			}, 200, false)

			gi, err := goblstripe.FromInvoice(s, validStripeAccount())
			require.NoError(t, err)

			assert.Empty(t, gi.Lines[0].Taxes)
			require.Len(t, gi.Lines[0].Charges, 1)
			assert.Equal(t, taxType, gi.Lines[0].Charges[0].Code.String())
			assert.Equal(t, name, gi.Lines[0].Charges[0].Reason)
			assert.Equal(t, "22.00", gi.Totals.TotalWithTax.String())
		})
	}
}

// spacebringAccount mirrors the account that reported APP-708: a Polish supplier
// billing in euros.
func spacebringAccount() *stripe.Account {
	return &stripe.Account{
		ID: "acct_spacebring",
		BusinessProfile: &stripe.AccountBusinessProfile{
			Name: "Test Coworking",
			SupportAddress: &stripe.Address{
				City:       "Warsaw",
				Country:    "PL",
				Line1:      "Prosta 51",
				PostalCode: "00-838",
			},
			SupportEmail: "support@example.com",
		},
		Settings: &stripe.AccountSettings{
			Invoices: &stripe.AccountSettingsInvoices{
				DefaultAccountTaxIDs: []*stripe.TaxID{
					{
						Created: 1736351225,
						Type:    stripe.TaxIDTypeEUVAT,
						Value:   "PL9551893317",
						Country: "PL",
					},
				},
			},
		},
	}
}

// chicagoLeaseTaxRateExpanded is the tax rate Stripe Tax applies to a Chicago
// subscription, as it arrives when the rate is expanded in the request.
func chicagoLeaseTaxRateExpanded() *stripe.TaxRate {
	return &stripe.TaxRate{
		ID:                  "txr_1UAapFHGXGms0VOu7empu7kR",
		TaxType:             stripe.TaxRateTaxTypeLeaseTax,
		DisplayName:         "Chicago Lease Tax",
		Country:             "US",
		State:               "IL",
		Jurisdiction:        "Chicago",
		JurisdictionLevel:   stripe.TaxRateJurisdictionLevelCity,
		Percentage:          15.0,
		EffectivePercentage: 15.0,
	}
}

// spacebringLeaseTaxInvoice reproduces the invoice behind APP-708: a Polish
// account billing a discounted Chicago subscription in euros, where Stripe Tax
// applies the city lease tax that GOBL has no category for. The zero-quantity
// usage lines and the per-line discounts are part of the reported shape.
func spacebringLeaseTaxInvoice() *stripe.Invoice {
	taxExempt := stripe.CustomerTaxExemptExempt
	period := &stripe.Period{Start: 1789471094, End: 1792063094}
	leaseTax := func(amount, taxable int64) []*stripe.InvoiceTotalTaxAmount {
		return []*stripe.InvoiceTotalTaxAmount{
			{
				Amount:        amount,
				Inclusive:     false,
				TaxableAmount: taxable,
				TaxRate:       chicagoLeaseTaxRateExpanded(),
			},
		}
	}
	line := func(desc string, amount, discount, quantity, tax, taxable int64) *stripe.InvoiceLineItem {
		l := &stripe.InvoiceLineItem{
			Description:  desc,
			Amount:       amount,
			Currency:     stripe.CurrencyEUR,
			Quantity:     quantity,
			Discountable: true,
			Period:       period,
			Price: &stripe.Price{
				BillingScheme: stripe.PriceBillingSchemePerUnit,
				Currency:      stripe.CurrencyEUR,
				TaxBehavior:   stripe.PriceTaxBehaviorExclusive,
				UnitAmount:    amount,
			},
			TaxAmounts: leaseTax(tax, taxable),
		}
		if discount > 0 {
			l.DiscountAmounts = []*stripe.InvoiceLineItemDiscountAmount{
				{Amount: discount, Discount: &stripe.Discount{Coupon: &stripe.Coupon{AmountOff: discount, Currency: stripe.CurrencyEUR, Valid: true}}},
			}
		}
		return l
	}

	return &stripe.Invoice{
		ID:                "in_1UAapEHGXGms0VOu8C23cy1a",
		Number:            "51FEE774-0033",
		AccountCountry:    "PL",
		AccountName:       "Test Coworking",
		Created:           1789471094,
		Currency:          stripe.CurrencyEUR,
		CustomerName:      "John Doe",
		CustomerEmail:     "customer@example.com",
		CustomerTaxExempt: &taxExempt,
		CustomerAddress: &stripe.Address{
			City:       "Chicago",
			Country:    "US",
			Line1:      "1449 S Michigan Ave",
			PostalCode: "60605",
			State:      "IL",
		},
		CustomerTaxIDs: []*stripe.InvoiceCustomerTaxID{},
		Lines: &stripe.InvoiceLineItemList{
			Data: []*stripe.InvoiceLineItem{
				line("Business (monthly)", 17500, 16625, 1, 131, 875),
				line("Member mobile app (monthly)", 10000, 9500, 1, 75, 500),
				line("0 × Additional active users (at €75.00 per 50 units / month)", 0, 0, 0, 0, 0),
			},
		},
		Subtotal:        27500,
		Tax:             206,
		Total:           1581,
		AmountDue:       1581,
		TotalTaxAmounts: leaseTax(206, 1375),
	}
}

func TestSpacebringLeaseTaxInvoice(t *testing.T) {
	gi, err := goblstripe.FromInvoice(spacebringLeaseTaxInvoice(), spacebringAccount())
	require.NoError(t, err)

	require.Len(t, gi.Lines, 3)
	for i, line := range gi.Lines[:2] {
		require.Len(t, line.Charges, 1, "line %d", i)
		assert.Equal(t, bill.ChargeKeyTax, line.Charges[0].Key)
		assert.Equal(t, "lease_tax", line.Charges[0].Code.String())
		assert.Equal(t, "Chicago Lease Tax", line.Charges[0].Reason)
		assert.Empty(t, line.Taxes, "the lease tax has no GOBL category")
	}
	assert.Empty(t, gi.Lines[2].Charges, "a zero amount is not charged")

	assert.Equal(t, "1.31", gi.Lines[0].Charges[0].Amount.String())
	assert.Equal(t, "0.75", gi.Lines[1].Charges[0].Amount.String())
	assert.Equal(t, "15.81", gi.Totals.TotalWithTax.String(), "total must match Stripe")
	assert.Equal(t, "0.00", gi.Totals.Tax.String())

	// The import used to fail here with `422 invalid-category: 'Chicago Lease Tax'
	// not defined in regime`.
	env, err := gobl.Envelop(gi)
	require.NoError(t, err)
	require.NoError(t, env.Validate())
}
