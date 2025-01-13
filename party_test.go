package goblstripe_test

import (
	"testing"

	goblstripe "github.com/invopop/gobl.stripe"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
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
