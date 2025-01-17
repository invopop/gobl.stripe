package goblstripe

import (
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/regimes/de"
	"github.com/invopop/gobl/tax"
	"github.com/stripe/stripe-go/v81"
)

/*
Not used (for now)
const (
	SelfIssuer  = "self"
	OtherIssuer = "account"
)
*/

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
func FromCustomer(doc *stripe.Invoice) *org.Party {
	/*
		There are 2 options here:
		- The Stripe invoice document contains plenty of info on the customer itself, we can extract it directly from the doc
		- We can assume that the doc has been extended with the customer info already

		We will assume the first option for now as it is more generic
		The difference between the fields is the following. For instance, the difference between customer.email and customer_email is
		"Until the invoice is finalized, this field will equal customer.email. Once the invoice is finalized, this field will no longer be updated."
	*/

	customerParty := new(org.Party)
	// I want to create an addresses object but just with one address obtained from the function FromAddress
	if doc.CustomerAddress != nil {
		customerParty.Addresses = append(customerParty.Addresses, FromAddress(doc.CustomerAddress))
	}
	if doc.CustomerEmail != "" {
		customerParty.Emails = append(customerParty.Emails, FromEmail(doc.CustomerEmail))
	}

	if doc.CustomerName != "" {
		customerParty.Name = doc.CustomerName
	}

	if doc.CustomerPhone != "" {
		customerParty.Telephones = append(customerParty.Telephones, FromTelephone(doc.CustomerPhone))
	}

	// Think what to do with doc.CustomerTaxExempt
	/*
		// Simpler solution: We could do this if we are expanding the doc with the customer data
		if doc.Customer.TaxIds != nil {
			customerParty.TaxID = FromTaxIDToTax(doc.Customer.TaxIDs.Data[0])
		}
	*/

	// More complex solution: If we are not demanding the data expanding
	if doc.CustomerTaxIDs != nil {
		// The problem is that there are 2 different objects in Stripe: stripe.TaxID and stripe.CustomerTaxID
		stripeTaxId := &stripe.TaxID{
			Type:  *doc.CustomerTaxIDs[0].Type,
			Value: doc.CustomerTaxIDs[0].Value,
		}
		customerParty.TaxID = FromTaxIDToTax(stripeTaxId)
	}

	// We can add custom fields in the customer but only in the customer field of Stripe, thus we will need to extend the doc
	return customerParty
}

func FromSupplier(doc *stripe.Invoice) *org.Party {
	/*
		Here we have 2 options:
		- Do a call to the API here to fetch the supplier (account) tax id
		- Assume the doc is already extended because the call to the API for this has already been done and the tax id is already in the doc

		We will assume the second option for now
	*/
	// For the moment, we only support the issuer being the account itself
	// We are assuming that there is at least one supplier (account) tax id
	supplierParty := new(org.Party)
	supplierParty.Name = doc.AccountName
	accountTaxId := doc.AccountTaxIDs[0]
	supplierParty.TaxID = FromTaxIDToTax(accountTaxId)
	return supplierParty
}

func FromAddress(address *stripe.Address) *org.Address {
	return &org.Address{
		Locality:    address.City,
		Country:     l10n.ISOCountryCode(address.Country),
		Street:      address.Line1,
		StreetExtra: address.Line2,
		Code:        cbc.Code(address.PostalCode),
		State:       cbc.Code(address.State),
	}
}

func FromEmail(email string) *org.Email {
	return &org.Email{
		Address: email,
	}
}

func FromTelephone(phone string) *org.Telephone {
	return &org.Telephone{
		Number: phone,
	}
}
