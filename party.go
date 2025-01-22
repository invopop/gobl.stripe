package goblstripe

import (
	"slices"
	"strings"

	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/regimes/de"
	"github.com/invopop/gobl/tax"
	"github.com/stripe/stripe-go/v81"
)

var orgIDKeys = []stripe.TaxIDType{"de_stn"}

// FromTaxIDToTax converts a stripe tax ID object into a GOBL tax identity, if possible.
func FromTaxIDToTax(taxID *stripe.TaxID) *tax.Identity {
	var tid *tax.Identity

	if taxID.Country != "" {
		tid = &tax.Identity{
			Country: l10n.TaxCountryCode(taxID.Country),
			Code:    cbc.Code(taxID.Value),
		}
		tid.Normalize()
		return tid
	}

	if taxID.Type == "eu_vat" {
		tid = &tax.Identity{
			Country: l10n.TaxCountryCode(taxID.Value[:2]),
			Code:    cbc.Code(taxID.Value[2:]),
		}
		tid.Normalize()
		return tid
	}

	// If there is no explicit country, we assume the country is the first part of the type
	country := strings.Split(string(taxID.Type), "_")[0]
	possibleCountry := l10n.TaxCountryCode(strings.ToUpper(country))
	if possibleCountry.Validate() == nil {
		tid = &tax.Identity{
			Country: possibleCountry,
			Code:    cbc.Code(taxID.Value),
		}
		tid.Normalize()
		return tid
	}

	return nil
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
	/*
		There are 2 options to get the customer from an invoice:
		- Info already in the invoice in fields like customer_name, customer_email, etc.
		- Request the invoice with the customer object expanded

		The difference between the fields is the following. For instance, the difference between customer.email and customer_email is
		"Until the invoice is finalized, this field will equal customer.email. Once the invoice is finalized, this field will no longer be updated."

		For this function we assume the second option
	*/
	var customerParty *org.Party

	if customer.Address != nil {
		customerParty = new(org.Party)
		customerParty.Addresses = append(customerParty.Addresses, FromAddress(customer.Address))
	}

	if customer.Email != "" {
		if customerParty == nil {
			customerParty = new(org.Party)
		}
		customerParty.Emails = append(customerParty.Emails, FromEmail(customer.Email))
	}

	if customer.Name != "" {
		if customerParty == nil {
			customerParty = new(org.Party)
		}
		customerParty.Name = customer.Name
	}

	if customer.Phone != "" {
		if customerParty == nil {
			customerParty = new(org.Party)
		}
		customerParty.Telephones = append(customerParty.Telephones, FromTelephone(customer.Phone))
	}

	if customer.TaxIDs != nil {
		if customerParty == nil {
			customerParty = new(org.Party)
		}

		if slices.Contains(orgIDKeys, customer.TaxIDs.Data[0].Type) {
			customerParty.Identities = make([]*org.Identity, 1)
			customerParty.Identities[0] = FromTaxIDToOrg(customer.TaxIDs.Data[0])
		} else {
			customerParty.TaxID = FromTaxIDToTax(customer.TaxIDs.Data[0])
		}
	}

	return customerParty
}

// newCustomer creates the customer data from the invoice object
func newCustomer(doc *stripe.Invoice) *org.Party {
	/*
		Here, we will assume that the customer data is already in the invoice object
		in fields like customer_name, customer_email, etc.
	*/

	customerParty := new(org.Party)
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

	if doc.CustomerTaxIDs != nil {
		// In Stripe there are 2 objects for the taxID: stripe.TaxID and stripe.CustomerTaxID
		stripeTaxId := &stripe.TaxID{
			Type:  *doc.CustomerTaxIDs[0].Type,
			Value: doc.CustomerTaxIDs[0].Value,
		}

		if slices.Contains(orgIDKeys, stripeTaxId.Type) {
			customerParty.Identities[0] = FromTaxIDToOrg(stripeTaxId)
		} else {
			customerParty.TaxID = FromTaxIDToTax(stripeTaxId)
		}
	}

	return customerParty
}

func newSupplier(doc *stripe.Invoice) *org.Party {
	/*
		Here we have 2 options:
		- Do a call to the API here to fetch the supplier (account) tax id
		- Assume the received invoice has the

		We will assume the second option for now
	*/
	// For the moment, we only support the issuer being the account itself
	var supplierParty *org.Party
	if doc.AccountName != "" {
		supplierParty = new(org.Party)
		supplierParty.Name = doc.AccountName
	}

	if doc.AccountTaxIDs != nil {
		if supplierParty == nil {
			supplierParty = new(org.Party)
		}
		// When we have several accounttaxids, we assume the first one is the main one
		accountTaxId := doc.AccountTaxIDs[0]
		if slices.Contains(orgIDKeys, accountTaxId.Type) {
			supplierParty.Identities[0] = FromTaxIDToOrg(accountTaxId)
		} else {
			supplierParty.TaxID = FromTaxIDToTax(accountTaxId)
		}
	}

	// TODO: If supplier is nil, we should get it from Invopop

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
