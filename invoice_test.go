package goblstripe_test

import (
	"encoding/json"
	"os"
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
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v81"
)

func validStripeAccount() *stripe.Account {
	return &stripe.Account{
		ID: "acct_test",
		BusinessProfile: &stripe.AccountBusinessProfile{
			Name: "Test Account",
			SupportAddress: &stripe.Address{
				City:       "Munich",
				Country:    "DE",
				Line1:      "Test Street 123",
				PostalCode: "80331",
				State:      "BY",
			},
			SupportEmail: "support@testaccount.com",
			SupportPhone: "+4989123456",
		},
		Settings: &stripe.AccountSettings{
			Invoices: &stripe.AccountSettingsInvoices{
				DefaultAccountTaxIDs: []*stripe.TaxID{
					{
						Created: 1736351225,
						Type:    stripe.TaxIDTypeEUVAT,
						Value:   "DE813495425",
						Country: "DE",
					},
				},
			},
		},
	}
}

func minimalStripeInvoice() *stripe.Invoice {
	return &stripe.Invoice{
		ID:             "in_1QkqKVQhcl5B85YlT32LIsNm",
		AccountCountry: "DE",
		AccountName:    "Test Account",
		AmountDue:      0,
		AmountPaid:     2000,
		Created:        1737738363,
		EffectiveAt:    1737738364,
		Currency:       stripe.CurrencyEUR,
		Lines: &stripe.InvoiceLineItemList{
			Data: []*stripe.InvoiceLineItem{
				{
					Description:  "Test Item",
					Amount:       2000,
					Currency:     stripe.CurrencyEUR,
					Quantity:     1,
					Discountable: true,
					Period: &stripe.Period{
						Start: 1704067200, // 2024-01-01 00:00:00 UTC
						End:   1706745599, // 2024-01-31 23:59:59 UTC
					},
					Price: &stripe.Price{
						BillingScheme: stripe.PriceBillingSchemePerUnit,
						Currency:      stripe.CurrencyEUR,
						TaxBehavior:   stripe.PriceTaxBehaviorUnspecified,
						UnitAmount:    2000,
					},
					TaxAmounts: []*stripe.InvoiceTotalTaxAmount{},
				},
			},
		},
		CustomerTaxIDs:  []*stripe.InvoiceCustomerTaxID{},
		Total:           2000,
		TotalTaxAmounts: []*stripe.InvoiceTotalTaxAmount{},
	}
}

func completeStripeInvoice() *stripe.Invoice {
	taxIDType := stripe.TaxIDTypeEUVAT
	return &stripe.Invoice{
		ID:             "inv_123",
		AccountName:    "Test Account",
		AccountCountry: "DE",
		AccountTaxIDs: []*stripe.TaxID{
			{
				Created: 1736351225,
				Type:    "eu_vat",
				Value:   "DE813495425",
				Country: "DE",
			},
		},
		AmountDue:       22989,
		AmountPaid:      22989,
		AmountRemaining: 0,
		Total:           22609, // Sum of lines (18999) + tax (3610)
		Created:         time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		Currency:        "eur",
		CustomerAddress: &stripe.Address{
			City:       "Berlin",
			Country:    "DE",
			Line1:      "Unter den Linden 1",
			Line2:      "",
			PostalCode: "10117",
			State:      "BE",
		},
		CustomerEmail: "me.unselfish@me.com",
		CustomerName:  "Test Customer Invoice",
		CustomerPhone: "+4915155555555",
		CustomerShipping: &stripe.ShippingDetails{
			Address: &stripe.Address{
				City:       "Berlin",
				Country:    "DE",
				Line1:      "Unter den Linden 1",
				Line2:      "",
				PostalCode: "10117",
				State:      "BE",
			},
			Name:  "Test Customer",
			Phone: "+4915155555555",
		},
		CustomerTaxIDs: []*stripe.InvoiceCustomerTaxID{
			{
				Type:  &taxIDType,
				Value: "DE282741168",
			},
		},
		CustomerTaxExempt: nil,
		DueDate:           0,
		EffectiveAt:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		Issuer: &stripe.InvoiceIssuer{
			Type: stripe.InvoiceIssuerTypeSelf,
		},
		Lines: &stripe.InvoiceLineItemList{
			Data: []*stripe.InvoiceLineItem{
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
							Inclusive: false,
							TaxRate: &stripe.TaxRate{
								TaxType:             stripe.TaxRateTaxTypeVAT,
								Country:             "DE",
								EffectivePercentage: 19.0,
								Percentage:          19.0,
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
							Inclusive: false,
							TaxRate: &stripe.TaxRate{
								TaxType:             stripe.TaxRateTaxTypeVAT,
								Country:             "DE",
								EffectivePercentage: 19.0,
								Percentage:          19.0,
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
							Inclusive: false,
							TaxRate: &stripe.TaxRate{
								TaxType:             stripe.TaxRateTaxTypeVAT,
								Country:             "DE",
								EffectivePercentage: 19.0,
								Percentage:          19.0,
							},
							TaxabilityReason: stripe.InvoiceTotalTaxAmountTaxabilityReasonStandardRated,
							TaxableAmount:    10000,
						},
					},
					UnitAmountExcludingTax: 10000,
				},
			},
		},
		Livemode: false,
		Paid:     true,
		PaymentIntent: &stripe.PaymentIntent{
			Amount:             22609,
			Created:            1736351225,
			Currency:           stripe.CurrencyEUR,
			PaymentMethodTypes: []string{"card"},
		},
		TotalTaxAmounts: []*stripe.InvoiceTotalTaxAmount{
			{
				Amount:    3610,
				Inclusive: false,
				TaxRate: &stripe.TaxRate{
					TaxType:    stripe.TaxRateTaxTypeVAT,
					Country:    "DE",
					Percentage: 19.0,
					Created:    1736351225,
					Livemode:   false,
				},
				TaxabilityReason: stripe.InvoiceTotalTaxAmountTaxabilityReasonStandardRated,
				TaxableAmount:    18999,
			},
		},
	}
}

func TestMinimalFieldsConversion(t *testing.T) {
	s := minimalStripeInvoice()
	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	assert.Equal(t, "in_1QkqKVQhcl5B85YlT32LIsNm", gi.Code.String())
	assert.Equal(t, "Test Account", gi.Supplier.Name)
	assert.Equal(t, cal.NewDate(2025, 1, 24), gi.OperationDate)
	assert.Equal(t, currency.EUR, gi.Currency)
	assert.Nil(t, gi.Customer)
	assert.Nil(t, gi.Tax)
}

func TestSupplier(t *testing.T) {
	s := minimalStripeInvoice()
	s.AccountTaxIDs = []*stripe.TaxID{
		{
			Created: 1736351225,
			Type:    "eu_vat",
			Value:   "DE813495425",
			Country: "DE",
		},
	}
	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	assert.Equal(t, "Test Account", gi.Supplier.Name)
	assert.Equal(t, "DE", gi.Supplier.TaxID.Country.String())
	assert.Equal(t, "813495425", gi.Supplier.TaxID.Code.String())
}

func TestCustomer(t *testing.T) {
	s := minimalStripeInvoice()
	s.CustomerAddress = &stripe.Address{
		City:       "Berlin",
		Country:    "DE",
		Line1:      "Unter den Linden 1",
		Line2:      "",
		PostalCode: "10117",
		State:      "BE",
	}
	s.CustomerEmail = "me.unselfish@me.com"
	s.CustomerName = "Test Customer"
	s.CustomerPhone = "+4915155555555"
	s.CustomerShipping = &stripe.ShippingDetails{
		Address: &stripe.Address{
			City:       "Berlin",
			Country:    "DE",
			Line1:      "Unter den Linden 1",
			Line2:      "",
			PostalCode: "10117",
			State:      "BE",
		},
		Name:  "Test Customer",
		Phone: "+4915155555555",
	}
	taxIDType := stripe.TaxIDTypeEUVAT
	s.CustomerTaxIDs = []*stripe.InvoiceCustomerTaxID{
		{
			Type:  &taxIDType,
			Value: "DE282741168",
		},
	}
	s.CustomerTaxExempt = nil
	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	assert.Equal(t, "Test Customer", gi.Customer.Name)
	assert.Equal(t, "DE", gi.Customer.TaxID.Country.String())
	assert.Equal(t, "282741168", gi.Customer.TaxID.Code.String())
	assert.Equal(t, "Unter den Linden 1", gi.Customer.Addresses[0].Street)
	assert.Equal(t, "10117", gi.Customer.Addresses[0].Code.String())
	assert.Equal(t, "Berlin", gi.Customer.Addresses[0].Locality)
	assert.Equal(t, "DE", gi.Customer.Addresses[0].Country.String())
	assert.Equal(t, "BE", gi.Customer.Addresses[0].State.String())
	assert.Equal(t, "me.unselfish@me.com", gi.Customer.Emails[0].Address)
}

func TestCustomerWithMetadata(t *testing.T) {
	s := minimalStripeInvoice()
	s.Customer = validStripeCustomer()
	s.Customer.Metadata = map[string]string{
		"gobl-customer-my-key": "my-value",
		"another-key":          "another-value",
	}

	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	c := gi.Customer
	require.NotNil(t, c)

	assert.Equal(t, "Test Customer", c.Name)
	assert.NotNil(t, c.TaxID)
	assert.Equal(t, "DE", c.TaxID.Country.String())
	assert.Equal(t, "282741168", c.TaxID.Code.String())
	assert.Equal(t, "Unter den Linden 1", c.Addresses[0].Street)
	assert.Equal(t, "10117", c.Addresses[0].Code.String())
	assert.Equal(t, "Berlin", c.Addresses[0].Locality)
	assert.Equal(t, "DE", c.Addresses[0].Country.String())
	assert.Equal(t, "BE", c.Addresses[0].State.String())
	assert.Equal(t, "me.unselfish@me.com", c.Emails[0].Address)
	assert.Equal(t, "+4915155555555", c.Telephones[0].Number)

	require.NotNil(t, c.Ext)
	assert.Equal(t, cbc.Code("my-value"), c.Ext[cbc.Key("my-key")])
	_, ok := c.Ext[cbc.Key("another-key")]
	assert.False(t, ok)
}

func TestCustomerMetadataCondition(t *testing.T) {
	t.Run("customer with empty metadata uses fallback", func(t *testing.T) {
		s := completeStripeInvoice()
		s.Customer = validStripeCustomer()
		s.Customer.Metadata = map[string]string{} // Empty metadata

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		// Should use newCustomerFromInvoice fallback
		c := gi.Customer
		require.NotNil(t, c)
		assert.Equal(t, "Test Customer Invoice", c.Name)
	})

	t.Run("customer with nil metadata uses fallback", func(t *testing.T) {
		s := completeStripeInvoice()
		s.Customer = validStripeCustomer()
		s.Customer.Metadata = nil // Nil metadata

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		// Should use newCustomerFromInvoice fallback
		c := gi.Customer
		require.NotNil(t, c)
		assert.Equal(t, "Test Customer Invoice", c.Name)
	})

	t.Run("customer with non-empty metadata uses FromCustomer", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.Customer = validStripeCustomer()
		s.Customer.Metadata = map[string]string{
			"gobl-customer-my-key": "my-value",
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		// Should use FromCustomer
		c := gi.Customer
		require.NotNil(t, c)
		assert.Equal(t, "Test Customer", c.Name)

		// Check that metadata was processed
		require.NotNil(t, c.Ext)
		assert.Equal(t, cbc.Code("my-value"), c.Ext[cbc.Key("my-key")])
	})

	t.Run("nil customer uses fallback", func(t *testing.T) {
		s := completeStripeInvoice()
		s.Customer = nil

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		// Should use newCustomerFromInvoice fallback
		c := gi.Customer
		require.NotNil(t, c)
		assert.Equal(t, "Test Customer Invoice", c.Name)
	})
}

func TestCalculate(t *testing.T) {
	s := minimalStripeInvoice()
	s.Total = 22609 // Sum of lines (18999) + tax (3610)
	s.Lines = &stripe.InvoiceLineItemList{
		Data: []*stripe.InvoiceLineItem{
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
						Inclusive: false,
						TaxRate: &stripe.TaxRate{
							TaxType:             stripe.TaxRateTaxTypeVAT,
							Country:             "DE",
							EffectivePercentage: 19.0,
							Percentage:          19.0,
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
						Inclusive: false,
						TaxRate: &stripe.TaxRate{
							TaxType:             stripe.TaxRateTaxTypeVAT,
							Country:             "DE",
							EffectivePercentage: 19.0,
							Percentage:          19.0,
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
						Inclusive: false,
						TaxRate: &stripe.TaxRate{
							TaxType:             stripe.TaxRateTaxTypeVAT,
							Country:             "DE",
							EffectivePercentage: 19.0,
							Percentage:          19.0,
						},
						TaxabilityReason: stripe.InvoiceTotalTaxAmountTaxabilityReasonStandardRated,
						TaxableAmount:    10000,
					},
				},
				UnitAmountExcludingTax: 10000,
			},
		},
	}
	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	err = gi.Calculate()
	require.NoError(t, err)

	assert.Equal(t, num.MakeAmount(18999, 2), gi.Totals.Sum)
	assert.Equal(t, num.MakeAmount(3610, 2), gi.Totals.Taxes.Sum)
	assert.Equal(t, num.MakeAmount(3610, 2), gi.Totals.Taxes.Categories[0].Amount)
	assert.Equal(t, num.MakeAmount(22609, 2), gi.Totals.TotalWithTax)
}

func TestValidate(t *testing.T) {
	s := completeStripeInvoice()
	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	err = gi.Calculate()
	require.NoError(t, err)

	err = gi.Validate()
	require.NoError(t, err)
}

func TestReverseCharge(t *testing.T) {
	s := minimalStripeInvoice()
	s.Total = 10000 // Line amount with reverse charge (0% tax)
	customerReverse := stripe.CustomerTaxExemptReverse
	s.CustomerTaxExempt = &customerReverse
	taxIDType := stripe.TaxIDTypeEUVAT
	s.CustomerTaxIDs = []*stripe.InvoiceCustomerTaxID{
		{
			Type:  &taxIDType,
			Value: "DE282741168",
		},
	}
	s.Lines = &stripe.InvoiceLineItemList{
		Data: []*stripe.InvoiceLineItem{
			{
				ID:                 "il_1Qf1WLQhcl5B85YleQz6ZuEfd",
				Amount:             10000,
				AmountExcludingTax: 10000,
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
				Description: "Time on 2000 × Pro Plan after 08 Jan 2025",
				TaxAmounts: []*stripe.InvoiceTotalTaxAmount{
					{
						Inclusive: false,
						TaxRate: &stripe.TaxRate{
							TaxType:             stripe.TaxRateTaxTypeVAT,
							Country:             "DE",
							EffectivePercentage: 0.0,
							Percentage:          19.0,
						},
						TaxabilityReason: stripe.InvoiceTotalTaxAmountTaxabilityReasonReverseCharge,
						TaxableAmount:    10000,
					},
				},
				UnitAmountExcludingTax: 5,
			},
		},
	}

	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	assert.Equal(t, tax.TagReverseCharge, gi.Tags.List[0])
	assert.Equal(t, gi.Lines[0].Taxes[0].Category, tax.CategoryVAT)

	err = gi.Calculate()
	require.NoError(t, err)

	assert.Equal(t, num.MakeAmount(10000, 2), gi.Totals.Sum)
	assert.Equal(t, num.MakeAmount(0, 2), gi.Totals.Taxes.Sum)
	assert.Equal(t, num.MakeAmount(0, 2), gi.Totals.Taxes.Categories[0].Amount)
	assert.Equal(t, num.MakeAmount(10000, 2), gi.Totals.TotalWithTax)
}

func TestSimplifiedInvoiceWhenNoCustomerTaxID(t *testing.T) {
	// Test with no customer tax IDs
	s := minimalStripeInvoice()
	s.CustomerTaxIDs = nil

	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	assert.Len(t, gi.Tags.List, 1)
	assert.Equal(t, tax.TagSimplified, gi.Tags.List[0])
}

func TestSimplifiedInvoiceWhenEmptyCustomerTaxID(t *testing.T) {
	// Test with empty customer tax IDs slice
	s := minimalStripeInvoice()
	s.CustomerTaxIDs = []*stripe.InvoiceCustomerTaxID{}

	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	assert.Len(t, gi.Tags.List, 1)
	assert.Equal(t, tax.TagSimplified, gi.Tags.List[0])
}

func TestNoSimplifiedTagWhenCustomerHasTaxID(t *testing.T) {
	// Test that simplified tag is NOT added when customer has a tax ID
	s := minimalStripeInvoice()
	taxIDType := stripe.TaxIDTypeEUVAT
	s.CustomerTaxIDs = []*stripe.InvoiceCustomerTaxID{
		{
			Type:  &taxIDType,
			Value: "DE123456789",
		},
	}

	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	// Should not have the simplified tag
	assert.Len(t, gi.Tags.List, 0)
}

func TestBothReverseChargeAndSimplifiedTags(t *testing.T) {
	// Test that both reverse-charge and simplified tags are added when:
	// - Customer has reverse charge status
	// - Customer has no tax ID
	s := minimalStripeInvoice()
	customerReverse := stripe.CustomerTaxExemptReverse
	s.CustomerTaxExempt = &customerReverse
	s.CustomerTaxIDs = nil // No tax ID

	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	// Should have both tags
	assert.Len(t, gi.Tags.List, 2)
	assert.Contains(t, gi.Tags.List, tax.TagReverseCharge)
	assert.Contains(t, gi.Tags.List, tax.TagSimplified)
}

func TestReverseChargeFromTaxAmountWithoutCustomerTaxID(t *testing.T) {
	// Test that both tags are added when reverse charge comes from tax amount
	// and customer has no tax ID
	s := minimalStripeInvoice()
	s.CustomerTaxIDs = nil // No tax ID
	s.TotalTaxAmounts = []*stripe.InvoiceTotalTaxAmount{
		{
			TaxabilityReason: stripe.InvoiceTotalTaxAmountTaxabilityReasonReverseCharge,
			TaxRate: &stripe.TaxRate{
				TaxType:    stripe.TaxRateTaxTypeVAT,
				Country:    "DE",
				Percentage: 19.0,
			},
		},
	}

	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	// Should have both tags
	assert.Len(t, gi.Tags.List, 2)
	assert.Contains(t, gi.Tags.List, tax.TagReverseCharge)
	assert.Contains(t, gi.Tags.List, tax.TagSimplified)
}

func TestReverseChargeOnlyWhenCustomerHasTaxID(t *testing.T) {
	// Test that only reverse-charge tag is added when customer has tax ID
	s := minimalStripeInvoice()
	customerReverse := stripe.CustomerTaxExemptReverse
	s.CustomerTaxExempt = &customerReverse
	taxIDType := stripe.TaxIDTypeEUVAT
	s.CustomerTaxIDs = []*stripe.InvoiceCustomerTaxID{
		{
			Type:  &taxIDType,
			Value: "DE123456789",
		},
	}

	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	// Should only have reverse-charge tag
	assert.Len(t, gi.Tags.List, 1)
	assert.Equal(t, tax.TagReverseCharge, gi.Tags.List[0])
}

func TestOrderingPeriod(t *testing.T) {
	s := minimalStripeInvoice()
	s.PeriodStart = 1737738363
	s.PeriodEnd = 1737738363
	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	assert.Equal(t, "2024-01-01", gi.Ordering.Period.Start.String())
	assert.Equal(t, "2024-01-31", gi.Ordering.Period.End.String())
}

func TestNewOrdering(t *testing.T) {
	t.Run("basic period dates", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.PeriodStart = 1704067200 // 2024-01-01 00:00:00 UTC
		s.PeriodEnd = 1704067200   // 2024-01-01 23:59:59 UTC

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Ordering)
		require.NotNil(t, gi.Ordering.Period)
		assert.Equal(t, "2024-01-01", gi.Ordering.Period.Start.String())
		assert.Equal(t, "2024-01-31", gi.Ordering.Period.End.String())
		assert.Empty(t, gi.Ordering.Code)
	})

	t.Run("with PO number in custom fields", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.PeriodStart = 1704067200
		s.PeriodEnd = 1704067200
		s.CustomFields = []*stripe.InvoiceCustomField{
			{
				Name:  "po number",
				Value: "PO-12345",
			},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Ordering)
		require.NotNil(t, gi.Ordering.Period)
		assert.Equal(t, "2024-01-01", gi.Ordering.Period.Start.String())
		assert.Equal(t, "2024-01-31", gi.Ordering.Period.End.String())
		assert.Equal(t, cbc.Code("PO-12345"), gi.Ordering.Code)
	})

	t.Run("case insensitive PO number field matching", func(t *testing.T) {
		testCases := []struct {
			name        string
			fieldName   string
			expected    string
			shouldMatch bool
		}{
			{"lowercase", "po number", "PO-12345", true},
			{"uppercase", "PO NUMBER", "PO-12345", true},
			{"mixed case", "Po Number", "PO-12345", true},
			{"with spaces", "  po number  ", "PO-12345", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				s := minimalStripeInvoice()
				s.PeriodStart = 1704067200
				s.PeriodEnd = 1706745599
				s.CustomFields = []*stripe.InvoiceCustomField{
					{
						Name:  tc.fieldName,
						Value: tc.expected,
					},
				}

				gi, err := goblstripe.FromInvoice(s, validStripeAccount())
				require.NoError(t, err)

				require.NotNil(t, gi.Ordering)
				if tc.shouldMatch {
					assert.Equal(t, cbc.Code(tc.expected), gi.Ordering.Code)
				} else {
					assert.Empty(t, gi.Ordering.Code)
				}
			})
		}
	})

	t.Run("no custom fields", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.PeriodStart = 1704067200
		s.PeriodEnd = 1706745599
		s.CustomFields = nil

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Ordering)
		require.NotNil(t, gi.Ordering.Period)
		assert.Equal(t, "2024-01-01", gi.Ordering.Period.Start.String())
		assert.Equal(t, "2024-01-31", gi.Ordering.Period.End.String())
		assert.Empty(t, gi.Ordering.Code)
	})

	t.Run("empty custom fields", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.PeriodStart = 1704067200
		s.PeriodEnd = 1706745599
		s.CustomFields = []*stripe.InvoiceCustomField{}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Ordering)
		require.NotNil(t, gi.Ordering.Period)
		assert.Equal(t, "2024-01-01", gi.Ordering.Period.Start.String())
		assert.Equal(t, "2024-01-31", gi.Ordering.Period.End.String())
		assert.Empty(t, gi.Ordering.Code)
	})
}

func TestUnexpandedTax(t *testing.T) {
	data, _ := os.ReadFile("examples/stripe.gobl/unexpanded_invoice.json")
	s := new(stripe.Invoice)
	err := json.Unmarshal(data, s)
	require.NoError(t, err)
	s.AccountCountry = "DE"
	customerExempt := stripe.CustomerTaxExemptNone
	s.CustomerTaxExempt = &customerExempt

	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	err = gi.Calculate()
	require.NoError(t, err)

	assert.Equal(t, goblstripe.ExpectedInvoiceTotal(s), gi.Totals.Payable)

	err = gi.Validate()
	require.NoError(t, err)
}

func TestStripeCoupon(t *testing.T) {
	data, _ := os.ReadFile("examples/stripe.gobl/stripe_coupon.json")
	s := new(stripe.Invoice)
	err := json.Unmarshal(data, s)
	require.NoError(t, err)

	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	err = gi.Calculate()
	require.NoError(t, err)

	assert.Equal(t, goblstripe.ExpectedInvoiceTotal(s), gi.Totals.Payable)

	err = gi.Validate()
	require.NoError(t, err)
}

func TestAdjustRounding(t *testing.T) {
	t.Run("no rounding adjustment needed", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.Total = 2000 // Matches the line total exactly

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		// No rounding should be set when totals match
		assert.Nil(t, gi.Totals.Rounding)
		assert.Equal(t, num.MakeAmount(2000, 2), gi.Totals.TotalWithTax)
	})

	t.Run("rounding adjustment needed with multiple lines", func(t *testing.T) {
		s := minimalStripeInvoice()
		// Add 3 lines to increase tolerance (3 * 0.005 = 0.015, allowing 1 cent difference)
		s.Lines = &stripe.InvoiceLineItemList{
			Data: []*stripe.InvoiceLineItem{
				{
					Description: "Item 1",
					Amount:      1000,
					Currency:    stripe.CurrencyEUR,
					Quantity:    1,
					Period:      &stripe.Period{Start: 1704067200, End: 1706745599},
					Price: &stripe.Price{
						BillingScheme: stripe.PriceBillingSchemePerUnit,
						Currency:      stripe.CurrencyEUR,
						TaxBehavior:   stripe.PriceTaxBehaviorUnspecified,
						UnitAmount:    1000,
					},
					TaxAmounts: []*stripe.InvoiceTotalTaxAmount{},
				},
				{
					Description: "Item 2",
					Amount:      500,
					Currency:    stripe.CurrencyEUR,
					Quantity:    1,
					Period:      &stripe.Period{Start: 1704067200, End: 1706745599},
					Price: &stripe.Price{
						BillingScheme: stripe.PriceBillingSchemePerUnit,
						Currency:      stripe.CurrencyEUR,
						TaxBehavior:   stripe.PriceTaxBehaviorUnspecified,
						UnitAmount:    500,
					},
					TaxAmounts: []*stripe.InvoiceTotalTaxAmount{},
				},
				{
					Description: "Item 3",
					Amount:      500,
					Currency:    stripe.CurrencyEUR,
					Quantity:    1,
					Period:      &stripe.Period{Start: 1704067200, End: 1706745599},
					Price: &stripe.Price{
						BillingScheme: stripe.PriceBillingSchemePerUnit,
						Currency:      stripe.CurrencyEUR,
						TaxBehavior:   stripe.PriceTaxBehaviorUnspecified,
						UnitAmount:    500,
					},
					TaxAmounts: []*stripe.InvoiceTotalTaxAmount{},
				},
			},
		}
		s.Total = 1999 // 1 cent less than calculated total (20.00)

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		// Rounding should be set to account for the difference
		require.NotNil(t, gi.Totals.Rounding)
		assert.Equal(t, num.MakeAmount(-1, 2), *gi.Totals.Rounding)
	})

	t.Run("rounding adjustment positive difference", func(t *testing.T) {
		s := minimalStripeInvoice()
		// Add 3 lines to increase tolerance
		s.Lines = &stripe.InvoiceLineItemList{
			Data: []*stripe.InvoiceLineItem{
				{
					Description: "Item 1",
					Amount:      1000,
					Currency:    stripe.CurrencyEUR,
					Quantity:    1,
					Period:      &stripe.Period{Start: 1704067200, End: 1706745599},
					Price: &stripe.Price{
						BillingScheme: stripe.PriceBillingSchemePerUnit,
						Currency:      stripe.CurrencyEUR,
						TaxBehavior:   stripe.PriceTaxBehaviorUnspecified,
						UnitAmount:    1000,
					},
					TaxAmounts: []*stripe.InvoiceTotalTaxAmount{},
				},
				{
					Description: "Item 2",
					Amount:      500,
					Currency:    stripe.CurrencyEUR,
					Quantity:    1,
					Period:      &stripe.Period{Start: 1704067200, End: 1706745599},
					Price: &stripe.Price{
						BillingScheme: stripe.PriceBillingSchemePerUnit,
						Currency:      stripe.CurrencyEUR,
						TaxBehavior:   stripe.PriceTaxBehaviorUnspecified,
						UnitAmount:    500,
					},
					TaxAmounts: []*stripe.InvoiceTotalTaxAmount{},
				},
				{
					Description: "Item 3",
					Amount:      500,
					Currency:    stripe.CurrencyEUR,
					Quantity:    1,
					Period:      &stripe.Period{Start: 1704067200, End: 1706745599},
					Price: &stripe.Price{
						BillingScheme: stripe.PriceBillingSchemePerUnit,
						Currency:      stripe.CurrencyEUR,
						TaxBehavior:   stripe.PriceTaxBehaviorUnspecified,
						UnitAmount:    500,
					},
					TaxAmounts: []*stripe.InvoiceTotalTaxAmount{},
				},
			},
		}
		s.Total = 2001 // 1 cent more than calculated total (20.00)

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		// Rounding should be set to account for the difference
		require.NotNil(t, gi.Totals.Rounding)
		assert.Equal(t, num.MakeAmount(1, 2), *gi.Totals.Rounding)
	})

	t.Run("rounding error too high for single line", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.Total = 1999 // 1 cent difference, but only 1 line (tolerance is 0.005)

		_, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.Error(t, err)

		// Verify error is of ErrRounding type
		assert.ErrorIs(t, err, goblstripe.ErrRounding)
		assert.Contains(t, err.Error(), "rounding error in totals too high")
	})

	t.Run("rounding error too high - large difference", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.Total = 1900 // 1 EUR less - too much difference

		_, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.Error(t, err)

		assert.ErrorIs(t, err, goblstripe.ErrRounding)
		assert.Contains(t, err.Error(), "rounding error in totals too high")
	})
}

func TestAdjustRoundingCreditNote(t *testing.T) {
	t.Run("no rounding adjustment needed", func(t *testing.T) {
		s := validCreditNote()
		// Credit note line: Amount=10294, Tax=2162 (21%)
		// Total should be 10294 + 2162 = 12456
		s.Total = 12456

		gi, err := goblstripe.FromCreditNote(s, validStripeAccount())
		require.NoError(t, err)

		assert.Nil(t, gi.Totals.Rounding)
		assert.Equal(t, num.MakeAmount(12456, 2), gi.Totals.TotalWithTax)
	})

	t.Run("rounding error too high - single line", func(t *testing.T) {
		s := validCreditNote()
		// 1 cent difference exceeds tolerance for single line (0.005)
		s.Total = 12455

		_, err := goblstripe.FromCreditNote(s, validStripeAccount())
		require.Error(t, err)

		assert.ErrorIs(t, err, goblstripe.ErrRounding)
		assert.Contains(t, err.Error(), "rounding error in totals too high")
	})

	t.Run("rounding error too high - large difference", func(t *testing.T) {
		s := validCreditNote()
		s.Total = 12356 // 1 EUR less

		_, err := goblstripe.FromCreditNote(s, validStripeAccount())
		require.Error(t, err)

		assert.ErrorIs(t, err, goblstripe.ErrRounding)
		assert.Contains(t, err.Error(), "rounding error in totals too high")
	})

	t.Run("rounding adjustment with multiple lines", func(t *testing.T) {
		s := validCreditNote()
		// Add more lines to increase tolerance
		s.Lines = &stripe.CreditNoteLineItemList{
			Data: []*stripe.CreditNoteLineItem{
				{
					ID:          "cnli_1",
					Amount:      5000,
					Description: "Item 1",
					TaxAmounts: []*stripe.CreditNoteTaxAmount{
						{
							Amount:        1050,
							Inclusive:     false,
							TaxRate:       &stripe.TaxRate{TaxType: stripe.TaxRateTaxTypeVAT, Country: "DE", EffectivePercentage: 21.0, Percentage: 21.0},
							TaxableAmount: 5000,
						},
					},
					Type: stripe.CreditNoteLineItemTypeInvoiceLineItem,
				},
				{
					ID:          "cnli_2",
					Amount:      3000,
					Description: "Item 2",
					TaxAmounts: []*stripe.CreditNoteTaxAmount{
						{
							Amount:        630,
							Inclusive:     false,
							TaxRate:       &stripe.TaxRate{TaxType: stripe.TaxRateTaxTypeVAT, Country: "DE", EffectivePercentage: 21.0, Percentage: 21.0},
							TaxableAmount: 3000,
						},
					},
					Type: stripe.CreditNoteLineItemTypeInvoiceLineItem,
				},
				{
					ID:          "cnli_3",
					Amount:      2294,
					Description: "Item 3",
					TaxAmounts: []*stripe.CreditNoteTaxAmount{
						{
							Amount:        482,
							Inclusive:     false,
							TaxRate:       &stripe.TaxRate{TaxType: stripe.TaxRateTaxTypeVAT, Country: "DE", EffectivePercentage: 21.0, Percentage: 21.0},
							TaxableAmount: 2294,
						},
					},
					Type: stripe.CreditNoteLineItemTypeInvoiceLineItem,
				},
			},
		}
		// Update tax amounts to match
		s.TaxAmounts = []*stripe.CreditNoteTaxAmount{
			{
				Amount:        2162,
				Inclusive:     false,
				TaxRate:       &stripe.TaxRate{TaxType: stripe.TaxRateTaxTypeVAT, Country: "DE", EffectivePercentage: 21.0, Percentage: 21.0},
				TaxableAmount: 10294,
			},
		}
		// Total = 5000 + 3000 + 2294 + 1050 + 630 + 482 = 12456
		// With 3 lines, tolerance is 0.015, so 1 cent is within range
		s.Total = 12455

		gi, err := goblstripe.FromCreditNote(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Totals.Rounding)
		assert.Equal(t, num.MakeAmount(-1, 2), *gi.Totals.Rounding)
	})
}

func TestMaxRoundingError(t *testing.T) {
	t.Run("single line invoice", func(t *testing.T) {
		gi := &bill.Invoice{
			Currency: currency.EUR,
			Lines: []*bill.Line{
				{Quantity: num.MakeAmount(1, 0)},
			},
		}
		// Max error should be 0.005 (0.5 cents) for 1 line
		maxErr := goblstripe.MaxRoundingError(gi)
		assert.Equal(t, num.MakeAmount(5, 3), maxErr)
	})

	t.Run("multiple lines invoice", func(t *testing.T) {
		gi := &bill.Invoice{
			Currency: currency.EUR,
			Lines: []*bill.Line{
				{Quantity: num.MakeAmount(1, 0)},
				{Quantity: num.MakeAmount(1, 0)},
				{Quantity: num.MakeAmount(1, 0)},
				{Quantity: num.MakeAmount(1, 0)},
				{Quantity: num.MakeAmount(1, 0)},
			},
		}
		// Max error should be 0.025 (0.5 cents * 5 lines)
		maxErr := goblstripe.MaxRoundingError(gi)
		assert.Equal(t, num.MakeAmount(25, 3), maxErr)
	})
}

// Credit Notes

func validCreditNote() *stripe.CreditNote {
	return &stripe.CreditNote{
		ID:       "cn_123",
		Currency: stripe.CurrencyEUR,
		Invoice: &stripe.Invoice{
			ID:             "inv_123",
			Number:         "INV-123",
			AccountName:    "Test Account",
			AccountCountry: "DE",
			AccountTaxIDs: []*stripe.TaxID{
				{
					Type:    "eu_vat",
					Value:   "DE813495425",
					Country: "DE",
				},
			},
			Created: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		},
		Created:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EffectiveAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		Amount:      22989,
		Total:       12456, // 10294 (line amount) + 2162 (tax) = 12456
		Customer:    validStripeCustomer(),
		Lines: &stripe.CreditNoteLineItemList{
			Data: []*stripe.CreditNoteLineItem{
				{
					ID:              "cnli_1Qk7SqQhcl5B85YlliBbmPiN",
					Object:          "credit_note_line_item",
					Amount:          10294,
					Description:     "Unused time on 2000 × Pro Plan after 08 Jan 2025",
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
				},
			},
		},
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
		Reason: stripe.CreditNoteReasonOrderChange,
	}
}

func TestFromCreditNote(t *testing.T) {
	s := validCreditNote()
	gi, err := goblstripe.FromCreditNote(s, validStripeAccount())
	require.NoError(t, err)

	assert.Equal(t, "cn_123", gi.Code.String())
	assert.Equal(t, "Test Account", gi.Supplier.Name)
	assert.Equal(t, cal.NewDate(2024, 1, 1), gi.OperationDate)
	assert.Equal(t, currency.EUR, gi.Currency)
	assert.Equal(t, "Test Customer", gi.Customer.Name)
	assert.Equal(t, l10n.DE.Tax(), gi.Customer.TaxID.Country)
	assert.Equal(t, "282741168", gi.Customer.TaxID.Code.String())
	assert.Equal(t, "Unused time on 2000 × Pro Plan after 08 Jan 2025", gi.Lines[0].Item.Name)
	assert.Equal(t, currency.EUR, gi.Lines[0].Item.Currency)
	assert.Equal(t, num.NewAmount(10294, 2), gi.Lines[0].Item.Price)
	assert.Equal(t, "order_change", gi.Preceding[0].Reason)
	assert.Equal(t, "123", gi.Preceding[0].Code.String())
	assert.Equal(t, "INV", gi.Preceding[0].Series.String())
	assert.Nil(t, gi.Tax)
}

func TestCreditNoteSimplifiedWhenNoCustomer(t *testing.T) {
	s := validCreditNote()
	s.Customer = nil // No customer

	gi, err := goblstripe.FromCreditNote(s, validStripeAccount())
	require.NoError(t, err)

	assert.Len(t, gi.Tags.List, 1)
	assert.Equal(t, tax.TagSimplified, gi.Tags.List[0])
}

func TestCreditNoteSimplifiedWhenCustomerHasNoTaxID(t *testing.T) {
	s := validCreditNote()
	s.Customer = &stripe.Customer{
		Name:  "Customer Without Tax ID",
		Email: "notax@example.com",
	}

	gi, err := goblstripe.FromCreditNote(s, validStripeAccount())
	require.NoError(t, err)

	assert.Len(t, gi.Tags.List, 1)
	assert.Equal(t, tax.TagSimplified, gi.Tags.List[0])
}

func TestCreditNoteNoSimplifiedWhenCustomerHasTaxID(t *testing.T) {
	s := validCreditNote()
	// validCreditNote already has validStripeCustomer() which has TaxIDs

	gi, err := goblstripe.FromCreditNote(s, validStripeAccount())
	require.NoError(t, err)

	// Should not have the simplified tag
	assert.Len(t, gi.Tags.List, 0)
}

func TestCreditNoteReverseChargeFromCustomer(t *testing.T) {
	s := validCreditNote()
	s.Customer.TaxExempt = stripe.CustomerTaxExemptReverse

	gi, err := goblstripe.FromCreditNote(s, validStripeAccount())
	require.NoError(t, err)

	// Should have only reverse-charge tag (customer has tax ID)
	assert.Len(t, gi.Tags.List, 1)
	assert.Equal(t, tax.TagReverseCharge, gi.Tags.List[0])
}

func TestCreditNoteReverseChargeFromTaxAmounts(t *testing.T) {
	s := validCreditNote()
	s.TaxAmounts = []*stripe.CreditNoteTaxAmount{
		{
			TaxabilityReason: stripe.CreditNoteTaxAmountTaxabilityReasonReverseCharge,
			TaxRate: &stripe.TaxRate{
				TaxType:    stripe.TaxRateTaxTypeVAT,
				Country:    "DE",
				Percentage: 19.0,
			},
		},
	}

	gi, err := goblstripe.FromCreditNote(s, validStripeAccount())
	require.NoError(t, err)

	// Should have only reverse-charge tag (customer has tax ID)
	assert.Len(t, gi.Tags.List, 1)
	assert.Equal(t, tax.TagReverseCharge, gi.Tags.List[0])
}

func TestCreditNoteBothReverseChargeAndSimplified(t *testing.T) {
	s := validCreditNote()
	// Use TaxAmounts for reverse charge since Customer will be nil
	s.TaxAmounts = []*stripe.CreditNoteTaxAmount{
		{
			TaxabilityReason: stripe.CreditNoteTaxAmountTaxabilityReasonReverseCharge,
			TaxRate: &stripe.TaxRate{
				TaxType:    stripe.TaxRateTaxTypeVAT,
				Country:    "DE",
				Percentage: 19.0,
			},
		},
	}
	s.Customer = nil // No customer (no tax ID)

	gi, err := goblstripe.FromCreditNote(s, validStripeAccount())
	require.NoError(t, err)

	// Should have both tags
	assert.Len(t, gi.Tags.List, 2)
	assert.Contains(t, gi.Tags.List, tax.TagReverseCharge)
	assert.Contains(t, gi.Tags.List, tax.TagSimplified)
}

func TestNotesInInvoiceConversion(t *testing.T) {
	t.Run("invoice with description only", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.Description = "Monthly subscription invoice"

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Notes)
		require.Len(t, gi.Notes, 1)
		assert.Equal(t, org.NoteKeyGeneral, gi.Notes[0].Key)
		assert.Equal(t, cbc.Key("stripe"), gi.Notes[0].Src)
		assert.Equal(t, "Monthly subscription invoice", gi.Notes[0].Text)
	})

	t.Run("invoice with footer only", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.Footer = "Thank you for your business\nPlease pay within 30 days"

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Notes)
		require.Len(t, gi.Notes, 1)
		assert.Empty(t, gi.Notes[0].Key)
		assert.Equal(t, cbc.Key("stripe"), gi.Notes[0].Src)
		assert.Equal(t, "Thank you for your business<br>Please pay within 30 days", gi.Notes[0].Text)
	})

	t.Run("invoice with both description and footer", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.Description = "Monthly subscription invoice"
		s.Footer = "Thank you for your business"

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Notes)
		require.Len(t, gi.Notes, 2)

		// First note: description with key "general"
		assert.Equal(t, org.NoteKeyGeneral, gi.Notes[0].Key)
		assert.Equal(t, cbc.Key("stripe"), gi.Notes[0].Src)
		assert.Equal(t, "Monthly subscription invoice", gi.Notes[0].Text)

		// Second note: footer without key
		assert.Empty(t, gi.Notes[1].Key)
		assert.Equal(t, cbc.Key("stripe"), gi.Notes[1].Src)
		assert.Equal(t, "Thank you for your business", gi.Notes[1].Text)
	})

	t.Run("invoice without description or footer", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.Description = ""
		s.Footer = ""

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		assert.Nil(t, gi.Notes)
	})

	t.Run("invoice with multiline footer", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.Footer = "Terms:\n1. Payment due in 30 days\n2. Late fees apply\n\nThank you!"

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Notes)
		require.Len(t, gi.Notes, 1)
		assert.Empty(t, gi.Notes[0].Key)
		assert.Equal(t, cbc.Key("stripe"), gi.Notes[0].Src)
		assert.Equal(t, "Terms:<br>1. Payment due in 30 days<br>2. Late fees apply<br><br>Thank you!", gi.Notes[0].Text)
	})
}

func TestNotesInCreditNoteConversion(t *testing.T) {
	t.Run("credit note with memo", func(t *testing.T) {
		s := validCreditNote()
		s.Memo = "Refund for cancelled subscription"

		gi, err := goblstripe.FromCreditNote(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Notes)
		require.Len(t, gi.Notes, 1)
		assert.Equal(t, org.NoteKeyGeneral, gi.Notes[0].Key)
		assert.Equal(t, cbc.Key("stripe"), gi.Notes[0].Src)
		assert.Equal(t, "Refund for cancelled subscription", gi.Notes[0].Text)
	})

	t.Run("credit note without memo", func(t *testing.T) {
		s := validCreditNote()
		s.Memo = ""

		gi, err := goblstripe.FromCreditNote(s, validStripeAccount())
		require.NoError(t, err)

		assert.Nil(t, gi.Notes)
	})

	t.Run("credit note with multiline memo", func(t *testing.T) {
		s := validCreditNote()
		s.Memo = "Refund reason:\n- Product defect\n- Customer request"

		gi, err := goblstripe.FromCreditNote(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Notes)
		require.Len(t, gi.Notes, 1)
		assert.Equal(t, org.NoteKeyGeneral, gi.Notes[0].Key)
		assert.Equal(t, cbc.Key("stripe"), gi.Notes[0].Src)
		assert.Equal(t, "Refund reason:<br>- Product defect<br>- Customer request", gi.Notes[0].Text)
	})
}
