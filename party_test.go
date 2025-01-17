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

func TestFromTaxIDToTax(t *testing.T) {
	tests := []struct {
		name     string
		taxID    *stripe.TaxID
		expected *tax.Identity
	}{
		{
			name: "EU VAT",
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
			name: "Unsupported Type",
			taxID: &stripe.TaxID{
				Type:  "us_ein",
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

func TestFromCustomer(t *testing.T) {
	taxType := stripe.TaxIDTypeEUVAT
	tests := []struct {
		name     string
		doc      *stripe.Invoice
		expected *org.Party
	}{
		{
			"complete customer data",
			&stripe.Invoice{
				CustomerAddress: &stripe.Address{
					City:       "Berlin",
					Country:    "DE",
					Line1:      "Street 123",
					Line2:      "Apt 4",
					PostalCode: "10115",
					State:      "BE",
				},
				CustomerEmail: "test@example.com",
				CustomerName:  "Test Customer",
				CustomerPhone: "+491234567890",
				CustomerTaxIDs: []*stripe.InvoiceCustomerTaxID{
					{Type: &taxType, Value: "DE123456789"},
				},
			},
			&org.Party{
				Addresses: []*org.Address{
					{
						Locality:    "Berlin",
						Country:     "DE",
						Street:      "Street 123",
						StreetExtra: "Apt 4",
						Code:        cbc.Code("10115"),
						State:       cbc.Code("BE"),
					},
				},
				Emails: []*org.Email{
					{Address: "test@example.com"},
				},
				Name: "Test Customer",
				Telephones: []*org.Telephone{
					{Number: "+491234567890"},
				},
				TaxID: &tax.Identity{
					Country: l10n.TaxCountryCode("DE"),
					Code:    cbc.Code("123456789"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblstripe.FromCustomer(tt.doc)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFromSupplier(t *testing.T) {
	invoice := &stripe.Invoice{
		AccountName: "Test Supplier",
		AccountTaxIDs: []*stripe.TaxID{
			{Type: "eu_vat", Value: "DE987654321"},
		},
	}

	expected := &org.Party{
		Name: "Test Supplier",
		TaxID: &tax.Identity{
			Country: l10n.TaxCountryCode("DE"),
			Code:    cbc.Code("987654321"),
		},
	}

	result := goblstripe.FromSupplier(invoice)
	if result.Name != expected.Name {
		t.Errorf("expected name %v, got %v", expected.Name, result.Name)
	}
	if result.TaxID == nil || *result.TaxID != *expected.TaxID {
		t.Errorf("expected tax ID %v, got %v", expected.TaxID, result.TaxID)
	}
}
