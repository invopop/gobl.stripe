package goblstripe_test

import (
	"testing"

	goblstripe "github.com/invopop/gobl.stripe"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/regimes/de"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	stripe "github.com/stripe/stripe-go/v81"
)

func TestToCustomerTaxIDDataParamsForTax(t *testing.T) {
	tests := []struct {
		name          string
		taxIdentity   *tax.Identity
		expectedType  string
		expectedValue string
		expectNil     bool
	}{
		{
			name: "Valid US tax ID",
			taxIdentity: &tax.Identity{
				Country: l10n.US.Tax(),
				Code:    "123456789",
			},
			expectedType:  string(stripe.TaxIDTypeUSEIN),
			expectedValue: "123456789",
			expectNil:     false,
		},
		{
			name: "Valid EU oss tax ID",
			taxIdentity: &tax.Identity{
				Country: l10n.EU.Tax(),
				Code:    "123456789",
			},
			expectedType:  string(stripe.TaxIDTypeEUOSSVAT),
			expectedValue: "EU123456789",
			expectNil:     false,
		},
		{
			name: "Valid EU tax ID - Germany",
			taxIdentity: &tax.Identity{
				Country: l10n.DE.Tax(),
				Code:    "123456789",
			},
			expectedType:  string(stripe.TaxIDTypeEUVAT),
			expectedValue: "DE123456789",
			expectNil:     false,
		},
		{
			name: "Valid Australia tax ID",
			taxIdentity: &tax.Identity{
				Country: l10n.AU.Tax(),
				Code:    "987654321",
			},
			expectedType:  string(stripe.TaxIDTypeAUABN),
			expectedValue: "987654321",
			expectNil:     false,
		},
		{
			name: "Unsupported country",
			taxIdentity: &tax.Identity{
				Country: l10n.CodeEmpty.Tax(),
				Code:    "unknown",
			},
			expectedType:  "",
			expectedValue: "",
			expectNil:     true,
		},
		{
			name:        "Nil tax identity",
			taxIdentity: nil,
			expectNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblstripe.ToCustomerTaxIDDataParamsForTax(tt.taxIdentity)
			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedType, stripe.StringValue(result.Type))
				assert.Equal(t, tt.expectedValue, stripe.StringValue(result.Value))
			}
		})
	}
}

func TestFromTaxIDToTax(t *testing.T) {
	tests := []struct {
		name     string
		taxID    *stripe.TaxID
		expected *tax.Identity
	}{
		{
			name: "with explicit country",
			taxID: &stripe.TaxID{
				Country: "DE",
				Value:   "123456789",
			},
			expected: &tax.Identity{
				Country: "DE",
				Code:    "123456789",
			},
		},
		{
			name: "with country from type",
			taxID: &stripe.TaxID{
				Type:  "es_cif",
				Value: "A12345678",
			},
			expected: &tax.Identity{
				Country: "ES",
				Code:    "A12345678",
			},
		},
		{
			name: "EU VAT - no explicit country",
			taxID: &stripe.TaxID{
				Type:  "eu_vat",
				Value: "DE123456789",
			},
			expected: &tax.Identity{
				Country: l10n.DE.Tax(),
				Code:    cbc.Code("123456789"),
			},
		},
		{
			name: "EU OSS VAT",
			taxID: &stripe.TaxID{
				Type:  "eu_oss_vat",
				Value: "EU123456789",
			},
			expected: &tax.Identity{
				Country: l10n.EU.Tax(),
				Code:    cbc.Code("123456789"),
			},
		},
		{
			name: "with invalid type",
			taxID: &stripe.TaxID{
				Type:  "invalid_type",
				Value: "123456789",
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblstripe.FromTaxIDToTax(tt.taxID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFromTaxIDToOrg(t *testing.T) {
	tests := []struct {
		name     string
		taxID    *stripe.TaxID
		expected *org.Identity
	}{
		{
			"valid DE STN",
			&stripe.TaxID{Type: "de_stn", Value: "123456789"},
			&org.Identity{Key: de.IdentityKeyTaxNumber, Code: cbc.Code("123456789")},
		},
		{
			"invalid type",
			&stripe.TaxID{Type: "unknown", Value: "123456789"},
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblstripe.FromTaxIDToOrg(tt.taxID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func validStripeCustomer() *stripe.Customer {
	return &stripe.Customer{
		ID:     "cus_RY7TdAXokervKd",
		Object: "customer",
		Address: &stripe.Address{
			City:       "Berlin",
			Country:    "DE",
			Line1:      "Unter den Linden 1",
			Line2:      "",
			PostalCode: "10117",
			State:      "BE",
		},
		Balance:       0,
		Created:       1736350312,
		Currency:      "eur",
		Delinquent:    false,
		Description:   "Test account",
		Email:         "me.unselfish@me.com",
		InvoicePrefix: "255RTCB4",
		Livemode:      false,
		Metadata:      map[string]string{},
		Name:          "Test Customer",
		Phone:         "+4915155555555",
		TaxExempt:     "none",
		PreferredLocales: []string{
			"en-GB",
		},
		Shipping: &stripe.ShippingDetails{
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
		TaxIDs: &stripe.TaxIDList{
			Data: []*stripe.TaxID{
				{
					ID:       "txi_1Qf1TJQhcl5B85YlZjE6rVvJ",
					Object:   "tax_id",
					Country:  "DE",
					Created:  1736351225,
					Livemode: false,
					Type:     "eu_vat",
					Value:    "DE282741168",
				},
			},
		},
	}
}

func TestFromCustomer(t *testing.T) {
	tests := []struct {
		name     string
		input    *stripe.Customer
		expected *org.Party
	}{
		{
			name: "with all fields",
			input: &stripe.Customer{
				Name:  "Test Company",
				Email: "test@example.com",
				Phone: "+1234567890",
				Address: &stripe.Address{
					City:       "Berlin",
					Country:    "DE",
					Line1:      "Test Street 1",
					Line2:      "Floor 2",
					PostalCode: "12345",
					State:      "Berlin",
				},
				TaxIDs: &stripe.TaxIDList{
					Data: []*stripe.TaxID{
						{
							Type:  "de_stn",
							Value: "123456789",
						},
					},
				},
			},
			expected: &org.Party{
				Name: "Test Company",
				Addresses: []*org.Address{
					{
						Locality:    "Berlin",
						Country:     "DE",
						Street:      "Test Street 1",
						StreetExtra: "Floor 2",
						Code:        "12345",
						State:       "Berlin",
					},
				},
				Emails: []*org.Email{
					{
						Address: "test@example.com",
					},
				},
				Telephones: []*org.Telephone{
					{
						Number: "+1234567890",
					},
				},
				Identities: []*org.Identity{
					{
						Key:  de.IdentityKeyTaxNumber,
						Code: "123456789",
					},
				},
			},
		},
		{
			name:  "German customer",
			input: validStripeCustomer(),
			expected: &org.Party{
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
		},
		{
			name: "with minimal fields",
			input: &stripe.Customer{
				Name: "Test Company",
			},
			expected: &org.Party{
				Name: "Test Company",
			},
		},
		{
			name: "with extensions",
			input: &stripe.Customer{
				Name: "Test Company",
				Metadata: map[string]string{
					"gobl-customer-foo": "bar",
				},
			},
			expected: &org.Party{
				Name: "Test Company",
				Ext: tax.Extensions{
					"foo": "bar",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblstripe.FromCustomer(tt.input, l10n.DE.Tax())
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFromAddress(t *testing.T) {
	tests := []struct {
		name     string
		input    *stripe.Address
		expected *org.Address
	}{
		{
			name: "with all fields",
			input: &stripe.Address{
				City:       "Berlin",
				Country:    "DE",
				Line1:      "Test Street 1",
				Line2:      "Floor 2",
				PostalCode: "12345",
				State:      "Berlin",
			},
			expected: &org.Address{
				Locality:    "Berlin",
				Country:     "DE",
				Street:      "Test Street 1",
				StreetExtra: "Floor 2",
				Code:        "12345",
				State:       "Berlin",
			},
		},
		{
			name: "with minimal fields",
			input: &stripe.Address{
				City:    "Berlin",
				Country: "DE",
			},
			expected: &org.Address{
				Locality: "Berlin",
				Country:  "DE",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblstripe.FromAddress(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFromEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *org.Email
	}{
		{
			name:  "valid email",
			input: "test@example.com",
			expected: &org.Email{
				Address: "test@example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblstripe.FromEmail(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFromTelephone(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *org.Telephone
	}{
		{
			name:  "valid phone number",
			input: "+1234567890",
			expected: &org.Telephone{
				Number: "+1234567890",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblstripe.FromTelephone(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFromCustomerItalyTaxIDLogic(t *testing.T) {
	tests := []struct {
		name          string
		customer      *stripe.Customer
		regimeCountry l10n.TaxCountryCode
		expected      *org.Party
	}{
		{
			name: "Italy regime with foreign customer - should add tax ID with customer country",
			customer: &stripe.Customer{
				Name: "Foreign Customer",
				Address: &stripe.Address{
					Country: "DE",
					City:    "Berlin",
				},
				// No TaxIDs provided
			},
			regimeCountry: l10n.IT.Tax(),
			expected: &org.Party{
				Name: "Foreign Customer",
				Addresses: []*org.Address{
					{
						Country:  "DE",
						Locality: "Berlin",
					},
				},
				TaxID: &tax.Identity{
					Country: l10n.DE.Tax(),
				},
			},
		},
		{
			name: "Italy regime with Italian customer - should not add tax ID",
			customer: &stripe.Customer{
				Name: "Italian Customer",
				Address: &stripe.Address{
					Country: "IT",
					City:    "Rome",
				},
			},
			regimeCountry: l10n.IT.Tax(),
			expected: &org.Party{
				Name: "Italian Customer",
				Addresses: []*org.Address{
					{
						Country:  "IT",
						Locality: "Rome",
					},
				},
			},
		},
		{
			name: "Non-Italy regime with foreign customer - should not add tax ID",
			customer: &stripe.Customer{
				Name: "Foreign Customer",
				Address: &stripe.Address{
					Country: "DE",
					City:    "Berlin",
				},
			},
			regimeCountry: l10n.DE.Tax(),
			expected: &org.Party{
				Name: "Foreign Customer",
				Addresses: []*org.Address{
					{
						Country:  "DE",
						Locality: "Berlin",
					},
				},
			},
		},
		{
			name: "Italy regime with existing tax ID - should not override",
			customer: &stripe.Customer{
				Name: "Customer with Tax ID",
				Address: &stripe.Address{
					Country: "DE",
					City:    "Berlin",
				},
				TaxIDs: &stripe.TaxIDList{
					Data: []*stripe.TaxID{
						{
							Type:  "eu_vat",
							Value: "DE123456789",
						},
					},
				},
			},
			regimeCountry: l10n.IT.Tax(),
			expected: &org.Party{
				Name: "Customer with Tax ID",
				Addresses: []*org.Address{
					{
						Country:  "DE",
						Locality: "Berlin",
					},
				},
				TaxID: &tax.Identity{
					Country: "DE",
					Code:    "123456789",
				},
			},
		},
		{
			name: "Italy regime with existing org identity - should not override",
			customer: &stripe.Customer{
				Name: "Customer with Org Identity",
				Address: &stripe.Address{
					Country: "DE",
					City:    "Berlin",
				},
				TaxIDs: &stripe.TaxIDList{
					Data: []*stripe.TaxID{
						{
							Type:  "de_stn",
							Value: "123456789",
						},
					},
				},
			},
			regimeCountry: l10n.IT.Tax(),
			expected: &org.Party{
				Name: "Customer with Org Identity",
				Addresses: []*org.Address{
					{
						Country:  "DE",
						Locality: "Berlin",
					},
				},
				Identities: []*org.Identity{
					{
						Key:  de.IdentityKeyTaxNumber,
						Code: "123456789",
					},
				},
			},
		},
		{
			name: "Italy regime with no address - should not add tax ID",
			customer: &stripe.Customer{
				Name: "Customer without Address",
			},
			regimeCountry: l10n.IT.Tax(),
			expected: &org.Party{
				Name: "Customer without Address",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblstripe.FromCustomer(tt.customer, tt.regimeCountry)
			assert.Equal(t, tt.expected, result)
		})
	}
}
