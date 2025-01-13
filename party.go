package goblstripe

import (
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/regimes/de"
	"github.com/invopop/gobl/tax"
	"github.com/stripe/stripe-go/v81"
)

// FromTaxIDToTax converts a stripe tax ID object into a GOBL tax identity, if possible.
func FromTaxIDToTax(taxID *stripe.TaxID) *tax.Identity {
	var tid *tax.Identity
	switch taxID.Type {
	case "eu_vat", "eu_oss_vat":
		tid = &tax.Identity{
			Country: l10n.TaxCountryCode(taxID.Value[:2]),
			Code:    cbc.Code(taxID.Value),
		}
	}
	tid.Normalize()
	return tid
}

// FromTaxIDToOrg converts a stripe tax ID object into a GOBL organization identity, if possible.
func FromTaxIDToOrg(taxID *stripe.TaxID) *org.Identity {
	var oid *org.Identity
	switch taxID.Type {
	case "de_stn":
		oid = &org.Identity{
			Key:  de.IdentityKeyTaxNumber,
			Code: cbc.Code(taxID.Value),
		}
	}
	oid.Normalize(nil)
	return oid
}

// ToCustomer converts a GOBL org.Party into a stripe customer object.
func ToCustomer(party *org.Party) *stripe.Customer {

	return nil
}

// FromCustomer converts a stripe customer object into a GOBL org.Party.
func FromCustomer(customer *stripe.Customer) *org.Party {

	return nil
}
