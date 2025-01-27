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

// For more details on Customer Tax IDs in Stripe, see: https://docs.stripe.com/billing/customer/tax-ids

var orgIDKeys = []stripe.TaxIDType{"de_stn"}

// ToCustomerTaxIDDataParamsForTax takes a GOBL tax Identity and converts it into
// the expected stripe tax ID object. GOBL tax identities are much more restrictive than
// those used in Stripe and only accept a specific code type for the given country. Countries
// defined in Stripe will multiple types will need to use the Org Identity conversion method
// also if they need to include all options.
func ToCustomerTaxIDDataParamsForTax(tID *tax.Identity) *stripe.CustomerTaxIDDataParams {
	if tID == nil {
		return nil
	}
	eu := l10n.Unions().Code(l10n.EU)
	if eu.HasMember(tID.Country.Code()) {
		return &stripe.CustomerTaxIDDataParams{
			Type:  stripe.String(string(stripe.TaxIDTypeEUVAT)),
			Value: stripe.String(tID.String()), // Include the country code.
		}
	}
	var typ stripe.TaxIDType
	switch tID.Country.Code() {
	case l10n.EU:
		return &stripe.CustomerTaxIDDataParams{
			Type:  stripe.String(string(stripe.TaxIDTypeEUOSSVAT)),
			Value: stripe.String(tID.String()), // Include the country code.
		}
	case l10n.AL:
		typ = stripe.TaxIDTypeAlTin
	case l10n.AD:
		typ = stripe.TaxIDTypeADNRT
	case l10n.AO:
		typ = stripe.TaxIDTypeAoTin
	case l10n.AR:
		typ = stripe.TaxIDTypeARCUIT
	case l10n.AM:
		typ = stripe.TaxIDTypeAmTin
	case l10n.AU:
		// TODO: handle other Australian tax ID type
		typ = stripe.TaxIDTypeAUABN // main code for businesses
	case l10n.BS:
		typ = stripe.TaxIDTypeBsTin
	case l10n.BH:
		typ = stripe.TaxIDTypeBhVAT
	case l10n.BB:
		typ = stripe.TaxIDTypeBbTin
	case l10n.BY:
		typ = stripe.TaxIDTypeByTin
	case l10n.BO:
		typ = stripe.TaxIDTypeBOTIN
	case l10n.BA:
		typ = stripe.TaxIDTypeBaTin
	case l10n.BR:
		typ = stripe.TaxIDTypeBRCNPJ
	case l10n.KH:
		typ = stripe.TaxIDTypeKhTin
	case l10n.CA:
		// TODO: correctly handle GST, HST, and PST variants
		typ = stripe.TaxIDTypeCABN
	case l10n.CL:
		typ = stripe.TaxIDTypeCLTIN
	case l10n.CN:
		typ = stripe.TaxIDTypeCNTIN
	case l10n.CO:
		typ = stripe.TaxIDTypeCONIT
	case l10n.CD:
		typ = stripe.TaxIDTypeCdNif
	case l10n.CR:
		typ = stripe.TaxIDTypeCRTIN
	case l10n.DO:
		typ = stripe.TaxIDTypeDORCN
	case l10n.EC:
		typ = stripe.TaxIDTypeECRUC
	case l10n.EG:
		typ = stripe.TaxIDTypeEGTIN
	case l10n.SV:
		typ = stripe.TaxIDTypeSVNIT
	case l10n.GE:
		typ = stripe.TaxIDTypeGEVAT
	case l10n.GN:
		typ = stripe.TaxIDTypeGnNif
	case l10n.HK:
		typ = stripe.TaxIDTypeHKBR
	case l10n.IS:
		typ = stripe.TaxIDTypeISVAT
	case l10n.IN:
		typ = stripe.TaxIDTypeINGST
	case l10n.ID:
		typ = stripe.TaxIDTypeIDNPWP
	case l10n.IL:
		typ = stripe.TaxIDTypeILVAT
	case l10n.JP:
		// TODO handle other Japanese tax ID types
		typ = stripe.TaxIDTypeJPCN
	case l10n.KZ:
		typ = stripe.TaxIDTypeKzBin
	case l10n.KE:
		typ = stripe.TaxIDTypeKEPIN
	case l10n.LI:
		// TODO handle Liechtenstein UID number
		typ = stripe.TaxIDTypeLiVAT
	case l10n.MY:
		// TODO handle Malaysian FRP and SST numbers
		typ = stripe.TaxIDTypeMYITN
	case l10n.MR:
		typ = stripe.TaxIDTypeMrNif
	case l10n.MX:
		typ = stripe.TaxIDTypeMXRFC
	case l10n.MD:
		typ = stripe.TaxIDTypeMdVAT
	case l10n.ME:
		typ = stripe.TaxIDTypeMePib
	case l10n.MA:
		typ = stripe.TaxIDTypeMaVAT
	case l10n.NP:
		typ = stripe.TaxIDTypeNpPan
	case l10n.NZ:
		typ = stripe.TaxIDTypeNZGST
	case l10n.NG:
		typ = stripe.TaxIDTypeNgTin
	case l10n.MK:
		typ = stripe.TaxIDTypeMkVAT
	case l10n.OM:
		typ = stripe.TaxIDTypeOmVAT
	case l10n.PE:
		typ = stripe.TaxIDTypePERUC
	case l10n.PH:
		typ = stripe.TaxIDTypePHTIN
	case l10n.RO:
		typ = stripe.TaxIDTypeROTIN
	case l10n.RU:
		// TODO handle KPP
		typ = stripe.TaxIDTypeRUINN
	case l10n.SA:
		typ = stripe.TaxIDTypeSAVAT
	case l10n.SN:
		typ = stripe.TaxIDTypeSnNinea
	case l10n.RS:
		typ = stripe.TaxIDTypeRSPIB
	case l10n.SG:
		// TODO: handle UEN
		typ = stripe.TaxIDTypeSGGST
	case l10n.SI:
		typ = stripe.TaxIDTypeSITIN
	case l10n.ZA:
		typ = stripe.TaxIDTypeZAVAT
	case l10n.KR:
		typ = stripe.TaxIDTypeKRBRN
	case l10n.SR:
		typ = stripe.TaxIDTypeSrFin
	case l10n.CH:
		// TODO: handle UID
		typ = stripe.TaxIDTypeCHVAT
	case l10n.TW:
		typ = stripe.TaxIDTypeTWVAT
	case l10n.TJ:
		typ = stripe.TaxIDTypeTjTin
	case l10n.TZ:
		typ = stripe.TaxIDTypeTzVAT
	case l10n.TH:
		typ = stripe.TaxIDTypeTHVAT
	case l10n.TR:
		typ = stripe.TaxIDTypeTRTIN
	case l10n.UG:
		typ = stripe.TaxIDTypeUgTin
	case l10n.UA:
		typ = stripe.TaxIDTypeUAVAT
	case l10n.AE:
		typ = stripe.TaxIDTypeAETRN
	case l10n.GB:
		typ = stripe.TaxIDTypeGBVAT
	case l10n.US:
		// TODO: determine if this should be here
		typ = stripe.TaxIDTypeUSEIN
	case l10n.UY:
		typ = stripe.TaxIDTypeUYRUC
	case l10n.UZ:
		typ = stripe.TaxIDTypeUzTin
	case l10n.VE:
		typ = stripe.TaxIDTypeVERIF
	case l10n.VN:
		typ = stripe.TaxIDTypeVNTIN
	case l10n.ZM:
		typ = stripe.TaxIDTypeZmTin
	case l10n.ZW:
		typ = stripe.TaxIDTypeZwTin
	}
	if typ != "" {
		return &stripe.CustomerTaxIDDataParams{
			Type:  stripe.String(string(typ)),
			Value: stripe.String(tID.Code.String()),
		}
	}
	return nil
}

// ToTaxIDForOrg expects a GOBL org Identity and attempts to convert it into a
// Stripe tax ID object.
func ToTaxIDForOrg(id *org.Identity) *stripe.TaxID {
	if id == nil {
		return nil
	}
	switch id.Key {
	case de.IdentityKeyTaxNumber:
		return &stripe.TaxID{
			Type:  stripe.TaxIDTypeDEStn,
			Value: id.Code.String(),
		}
	}
	return nil
}

// FromTaxIDToTax converts a stripe tax ID object into a GOBL tax identity, if possible.
func FromTaxIDToTax(taxID *stripe.TaxID) *tax.Identity {
	if taxID == nil {
		return nil
	}
	var tid *tax.Identity

	if taxID.Country != "" {
		tid = &tax.Identity{
			Country: l10n.TaxCountryCode(taxID.Country),
			Code:    cbc.Code(taxID.Value),
		}
		tid.Normalize()
		return tid
	}

	switch taxID.Type {
	case stripe.TaxIDTypeEUVAT, stripe.TaxIDTypeEUOSSVAT:
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

// FromTaxIDForOrg converts a stripe tax ID object into a GOBL organization identity, if possible.
func FromTaxIDToOrg(taxID *stripe.TaxID) *org.Identity {
	if taxID == nil {
		return nil
	}
	var oid *org.Identity
	switch taxID.Type {
	case stripe.TaxIDTypeDEStn:
		oid = &org.Identity{
			Key:  de.IdentityKeyTaxNumber,
			Code: cbc.Code(taxID.Value),
		}
	}
	oid.Normalize(nil)
	return oid
}

// ToCustomerParams converts a GOBL org.Party into a stripe customer object suitable for
// sending to the Stripe API.
func ToCustomerParams(party *org.Party) *stripe.CustomerParams {
	if party == nil {
		return nil
	}
	cus := &stripe.CustomerParams{
		Name:     stripe.String(party.Name),
		Metadata: make(map[string]string),
	}
	if len(party.Emails) > 0 {
		cus.Email = stripe.String(party.Emails[0].Address)
	}
	if len(party.Addresses) > 0 {
		cus.Address = ToAddressParams(party.Addresses[0])
	}
	if party.TaxID != nil {
		tID := ToCustomerTaxIDDataParamsForTax(party.TaxID)
		if tID != nil {
			cus.TaxIDData = []*stripe.CustomerTaxIDDataParams{
				tID,
			}
		}
	}
	return cus
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

	if customer.TaxIDs != nil && customer.TaxIDs.Data != nil && len(customer.TaxIDs.Data) > 0 {
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

// newCustomerFromInvoice creates the customer data from the invoice object
func newCustomerFromInvoice(doc *stripe.Invoice) *org.Party {
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
		stripeTaxID := &stripe.TaxID{
			Type:  *doc.CustomerTaxIDs[0].Type,
			Value: doc.CustomerTaxIDs[0].Value,
		}

		if slices.Contains(orgIDKeys, stripeTaxID.Type) {
			customerParty.Identities[0] = FromTaxIDToOrg(stripeTaxID)
		} else {
			customerParty.TaxID = FromTaxIDToTax(stripeTaxID)
		}
	}

	return customerParty
}

func newSupplierFromInvoice(doc *stripe.Invoice) *org.Party {
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
		accountTaxID := doc.AccountTaxIDs[0]
		if slices.Contains(orgIDKeys, accountTaxID.Type) {
			supplierParty.Identities[0] = FromTaxIDToOrg(accountTaxID)
		} else {
			supplierParty.TaxID = FromTaxIDToTax(accountTaxID)
		}
	}

	// TODO: If supplier is nil, we should get it from Invopop

	return supplierParty
}

// FromAddress converts a stripe address object into a GOBL address object.
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

// FromEmail converts a string into a GOBL email object.
func FromEmail(email string) *org.Email {
	return &org.Email{
		Address: email,
	}
}

// FromTelephone converts a string into a GOBL telephone object.
func FromTelephone(phone string) *org.Telephone {
	return &org.Telephone{
		Number: phone,
	}
}

// ToAddress converts the GOBL address object into a stripe address object.
func ToAddressParams(addr *org.Address) *stripe.AddressParams {
	if addr == nil {
		return nil
	}
	return &stripe.AddressParams{
		Line1:      stripe.String(addr.Street),
		Line2:      stripe.String(addr.StreetExtra),
		City:       stripe.String(addr.Locality),
		State:      stripe.String(addr.Region),
		PostalCode: stripe.String(addr.Code.String()),
		Country:    stripe.String(addr.Country.String()),
	}
}
