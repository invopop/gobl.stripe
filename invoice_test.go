package goblstripe_test

import (
	"testing"
	"time"

	goblstripe "github.com/invopop/gobl.stripe"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/gobl/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v81"
)

const (
	namespace = "550e8400-e29b-41d4-a716-446655440000"
)

func validStripeInvoice() *stripe.Invoice {
	return &stripe.Invoice{
		ID:             "inv_123",
		AccountName:    "Test Account",
		AccountCountry: "DE",
		AccountTaxIDs: []*stripe.TaxID{
			{
				Type:    "eu_vat",
				Value:   "DE813495425",
				Country: "DE",
			},
		},
		AmountDue:       22989,
		AmountPaid:      22989,
		AmountRemaining: 0,
		Created:         time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		Currency:        "eur",
		Customer:        validStripeCustomer(),
		CustomerAddress: &stripe.Address{
			City:       "Berlin",
			Country:    "DE",
			Line1:      "Unter den Linden 1",
			Line2:      "",
			PostalCode: "10117",
			State:      "BE",
		},
		CustomerEmail: "me.unselfish@me.com",
		CustomerName:  "Test Customer",
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
								TaxType:    stripe.TaxRateTaxTypeVAT,
								Country:    "DE",
								Percentage: 19.0,
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
								TaxType:    stripe.TaxRateTaxTypeVAT,
								Country:    "DE",
								Percentage: 19.0,
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
								TaxType:    stripe.TaxRateTaxTypeVAT,
								Country:    "DE",
								Percentage: 19.0,
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

func TestBasicFieldsConversion(t *testing.T) {
	s := validStripeInvoice()
	gi, err := goblstripe.FromInvoice(s, uuid.MustParse(namespace))
	require.NoError(t, err)

	assert.Equal(t, "inv_123", gi.Code.String())
	assert.Equal(t, "Test Account", gi.Supplier.Name)
	assert.Equal(t, "86f836a6-47ee-302e-90df-2ad86a2f7060", gi.UUID.String())
	assert.Equal(t, cal.MakeDate(2024, 1, 1), gi.IssueDate)
	assert.Equal(t, cal.NewDate(2024, 1, 1), gi.OperationDate)
	assert.Equal(t, currency.EUR, gi.Currency)
	assert.Equal(t, "Test Customer", gi.Customer.Name)
	assert.Equal(t, l10n.DE.Tax(), gi.Customer.TaxID.Country)
	assert.Equal(t, "282741168", gi.Customer.TaxID.Code.String())
	assert.Equal(t, "Unused time on 2000 × Pro Plan after 08 Jan 2025", gi.Lines[0].Item.Name)
	assert.Equal(t, currency.EUR, gi.Lines[0].Item.Currency)
	assert.Equal(t, num.MakeAmount(-11000, 2), gi.Lines[0].Item.Price)
	assert.Nil(t, gi.Tax)
}

func TestSupplier(t *testing.T) {
	s := validStripeInvoice()
	gi, err := goblstripe.FromInvoice(s, uuid.MustParse(namespace))
	require.NoError(t, err)

	assert.Equal(t, "Test Account", gi.Supplier.Name)
	assert.Equal(t, "DE", gi.Supplier.TaxID.Country.String())
	assert.Equal(t, "813495425", gi.Supplier.TaxID.Code.String())
}

func TestCustomer(t *testing.T) {
	s := validStripeInvoice()
	gi, err := goblstripe.FromInvoice(s, uuid.MustParse(namespace))
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

func TestTaxInclusive(t *testing.T) {

	s := validStripeInvoice()

	// Check that tax are inclusive
	gi, err := goblstripe.FromInvoice(s, uuid.MustParse(namespace))
	require.NoError(t, err)
	assert.Nil(t, gi.Tax)

	s.TotalTaxAmounts[0].Inclusive = true
	gi, err = goblstripe.FromInvoice(s, uuid.MustParse(namespace))
	require.NoError(t, err)
	assert.Equal(t, tax.CategoryVAT, gi.Tax.PricesInclude)
}

func TestOrderingPeriod(t *testing.T) {
	s := validStripeInvoice()
	gi, err := goblstripe.FromInvoice(s, uuid.MustParse(namespace))
	require.NoError(t, err)

	assert.Equal(t, "2025-01-08", gi.Lines[0].Item.Meta[goblstripe.MetaKeyDateFrom])
	assert.Equal(t, "2025-02-08", gi.Lines[0].Item.Meta[goblstripe.MetaKeyDateTo])
}

func TestShippingDetails(t *testing.T) {
	s := validStripeInvoice()
	gi, err := goblstripe.FromInvoice(s, uuid.MustParse(namespace))
	require.NoError(t, err)

	assert.Equal(t, "Test Customer", gi.Delivery.Receiver.Name)
	assert.Equal(t, "Berlin", gi.Delivery.Receiver.Addresses[0].Locality)
	assert.Equal(t, "DE", gi.Delivery.Receiver.Addresses[0].Country.String())
	assert.Equal(t, "Unter den Linden 1", gi.Delivery.Receiver.Addresses[0].Street)
	assert.Equal(t, "10117", gi.Delivery.Receiver.Addresses[0].Code.String())
	assert.Equal(t, "BE", gi.Delivery.Receiver.Addresses[0].State.String())
}

func TestNoTermsWhenPaid(t *testing.T) {
	s := validStripeInvoice()
	gi, err := goblstripe.FromInvoice(s, uuid.MustParse(namespace))
	require.NoError(t, err)

	assert.Nil(t, gi.Payment)
}

func TestExchangeRateConversionDefault(t *testing.T) {
	s := validStripeInvoice()
	gi, err := goblstripe.FromInvoice(s, uuid.MustParse(namespace))
	require.NoError(t, err)

	assert.Nil(t, gi.ExchangeRates)

	s.Currency = "usd"
	gi, err = goblstripe.FromInvoice(s, uuid.MustParse(namespace))
	require.NoError(t, err)

	assert.Equal(t, num.MakeAmount(935, 3), gi.ExchangeRates[0].Amount)
}

func TestZeroDecimalCurrencies(t *testing.T) {
	s := validStripeInvoice()
	s.Currency = "jpy"
	s.Lines.Data[0].Currency = "jpy"
	gi, err := goblstripe.FromInvoice(s, uuid.MustParse(namespace))
	require.NoError(t, err)

	assert.Equal(t, currency.JPY, gi.Currency)
	assert.Equal(t, num.MakeAmount(-11000, 0), gi.Lines[0].Item.Price)
}

func TestCalculate(t *testing.T) {
	s := validStripeInvoice()
	gi, err := goblstripe.FromInvoice(s, uuid.MustParse(namespace))
	require.NoError(t, err)

	err = gi.Calculate()
	require.NoError(t, err)

	assert.Equal(t, num.MakeAmount(18999, 2), gi.Totals.Sum)
	assert.Equal(t, num.MakeAmount(3610, 2), gi.Totals.Taxes.Sum)
	assert.Equal(t, num.MakeAmount(3610, 2), gi.Totals.Taxes.Categories[0].Amount)
	assert.Equal(t, num.MakeAmount(22609, 2), gi.Totals.TotalWithTax)
}

func TestValidate(t *testing.T) {
	s := validStripeInvoice()
	gi, err := goblstripe.FromInvoice(s, uuid.MustParse(namespace))
	require.NoError(t, err)

	err = gi.Calculate()
	require.NoError(t, err)

	err = gi.Validate()
	require.NoError(t, err)
}

// Below is a full test suite for the FromInvoice function. It is commented out
// because there are some fields in GOBL that cannot be set by the function,
// e.g. total in Totals
/*func TestFromInvoice(t *testing.T) {
	tests := []struct {
		name      string
		invoice   *stripe.Invoice
		namespace uuid.UUID
		want      *bill.Invoice
		wantErr   bool
	}{
		{
			name: "Invoice Invopop",
			invoice: &stripe.Invoice{
				ID:             "inv_123",
				AccountName:    "Test Account",
				AccountCountry: "DE",
				AccountTaxIDs: []*stripe.TaxID{
					{
						Type:    "eu_vat",
						Value:   "DE12345678",
						Country: "DE",
					},
				},
				AmountDue:       22989,
				AmountPaid:      22989,
				AmountRemaining: 0,
				Created:         time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
				Currency:        "eur",
				Customer:        createSampleCustomer(),
				CustomerAddress: &stripe.Address{
					City:       "Berlin",
					Country:    "DE",
					Line1:      "Unter den Linden 1",
					Line2:      "",
					PostalCode: "10117",
					State:      "BE",
				},
				CustomerEmail: "me.unselfish@me.com",
				CustomerName:  "Test Customer",
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
									Amount:    -2310,
									Inclusive: false,
									TaxRate: &stripe.TaxRate{
										TaxType:    stripe.TaxRateTaxTypeVAT,
										Country:    "DE",
										Percentage: 19.0,
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
									Amount:    3800,
									Inclusive: false,
									TaxRate: &stripe.TaxRate{
										TaxType:    stripe.TaxRateTaxTypeVAT,
										Country:    "DE",
										Percentage: 19.0,
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
									Amount:    1900,
									Inclusive: false,
									TaxRate: &stripe.TaxRate{
										TaxType:    stripe.TaxRateTaxTypeVAT,
										Country:    "DE",
										Percentage: 19.0,
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
					Amount:             22989,
					Created:            1736351225,
					Currency:           stripe.CurrencyEUR,
					PaymentMethodTypes: []string{"card"},
				},
				Subtotal:             18999,
				SubtotalExcludingTax: 18999,
				Tax:                  3590,
				Total:                22989,
				TotalExcludingTax:    18999,
				TotalTaxAmounts: []*stripe.InvoiceTotalTaxAmount{
					{
						Amount:    3590,
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
			},
			namespace: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			want: &bill.Invoice{
				Identify: uuid.Identify{
					UUID: uuid.MustParse("86f836a6-47ee-302e-90df-2ad86a2f7060"),
				},
				Addons:        tax.Addons{},
				Type:          bill.InvoiceTypeStandard,
				Regime:        tax.WithRegime("DE"),
				Code:          "inv_123",
				Currency:      currency.EUR,
				IssueDate:     cal.MakeDate(2024, 1, 1),
				OperationDate: cal.NewDate(2024, 1, 1),
				Supplier: &org.Party{
					Name: "Test Account",
					TaxID: &tax.Identity{
						Country: l10n.DE.Tax(),
						Code:    "12345678",
					},
				},
				Customer: &org.Party{
					Name: "Test Customer",
					Addresses: []*org.Address{
						{
							Locality:    "Berlin",
							Country:     "DE",
							Street:      "Unter den Linden 1",
							StreetExtra: "",
							Code:        "10117",
							State:       "BE",
						},
					},
					Emails: []*org.Email{
						{
							Address: "me.unselfish@me.com",
						},
					},
					Telephones: []*org.Telephone{
						{
							Number: "+4915155555555",
						},
					},
					TaxID: &tax.Identity{
						Country: "DE",
						Code:    "282741168",
					},
				},
				Lines: []*bill.Line{
					{
						Index:    1,
						Quantity: num.MakeAmount(1, 0),
						Sum:      num.MakeAmount(-11000, 2),
						Total:    num.MakeAmount(-11000, 2),
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
								Percent:  num.NewPercentage(190, 3),
							},
						},
					},
					{
						Index:    2,
						Quantity: num.MakeAmount(1, 0),
						Sum:      num.MakeAmount(19999, 2),
						Total:    num.MakeAmount(19999, 2),
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
								Country:  "DE",
								Percent:  num.NewPercentage(190, 3),
							},
						},
					},
					{
						Index:    3,
						Quantity: num.MakeAmount(1, 0),
						Sum:      num.MakeAmount(10000, 2),
						Total:    num.MakeAmount(10000, 2),
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
								Country:  "DE",
								Percent:  num.NewPercentage(190, 3),
							},
						},
					},
				},
				Delivery: &bill.Delivery{
					Receiver: &org.Party{
						Name: "Test Customer",
						Addresses: []*org.Address{
							{
								Locality:    "Berlin",
								Country:     "DE",
								Street:      "Unter den Linden 1",
								StreetExtra: "",
								Code:        "10117",
								State:       "BE",
							},
						},
					},
				},
				Totals: &bill.Totals{
					Sum:   num.MakeAmount(18999, 2),
					Total: num.MakeAmount(18999, 2),
					Taxes: &tax.Total{
						Categories: []*tax.CategoryTotal{
							{
								Code: tax.CategoryVAT,
								Rates: []*tax.RateTotal{
									{
										Percent: num.NewPercentage(190, 3),
										Amount:  num.MakeAmount(3990, 2),
									},
								},
								Amount: num.MakeAmount(3990, 2),
							},
						},
						Sum: num.MakeAmount(3990, 2),
					},
					Tax:          num.MakeAmount(3990, 2),
					TotalWithTax: num.MakeAmount(22989, 2),
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := goblstripe.FromInvoice(tt.invoice, tt.namespace)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			err = got.Calculate()
			assert.NoError(t, err)

			err = got.Validate()
			assert.NoError(t, err)

			assert.Equal(t, tt.want, got)
		})
	}
}*/
