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

// Lookup map for country code to Stripe tax ID type
var taxIDMapGOBLToStripe = map[l10n.Code]stripe.TaxIDType{
	l10n.AL: stripe.TaxIDTypeAlTin,
	l10n.AD: stripe.TaxIDTypeADNRT,
	l10n.AO: stripe.TaxIDTypeAoTin,
	l10n.AR: stripe.TaxIDTypeARCUIT,
	l10n.AM: stripe.TaxIDTypeAmTin,
	// TODO: handle other Australian tax ID type
	l10n.AU: stripe.TaxIDTypeAUABN, // main code for businesses
	l10n.BS: stripe.TaxIDTypeBsTin,
	l10n.BH: stripe.TaxIDTypeBhVAT,
	l10n.BB: stripe.TaxIDTypeBbTin,
	l10n.BY: stripe.TaxIDTypeByTin,
	l10n.BO: stripe.TaxIDTypeBOTIN,
	l10n.BA: stripe.TaxIDTypeBaTin,
	l10n.BR: stripe.TaxIDTypeBRCNPJ,
	l10n.KH: stripe.TaxIDTypeKhTin,
	l10n.CA: stripe.TaxIDTypeCABN, // TODO: correctly handle GST, HST, and PST variants
	l10n.CL: stripe.TaxIDTypeCLTIN,
	l10n.CN: stripe.TaxIDTypeCNTIN,
	l10n.CO: stripe.TaxIDTypeCONIT,
	l10n.CD: stripe.TaxIDTypeCdNif,
	l10n.CR: stripe.TaxIDTypeCRTIN,
	l10n.DO: stripe.TaxIDTypeDORCN,
	l10n.EC: stripe.TaxIDTypeECRUC,
	l10n.EG: stripe.TaxIDTypeEGTIN,
	l10n.SV: stripe.TaxIDTypeSVNIT,
	l10n.GE: stripe.TaxIDTypeGEVAT,
	l10n.GN: stripe.TaxIDTypeGnNif,
	l10n.HK: stripe.TaxIDTypeHKBR,
	l10n.IS: stripe.TaxIDTypeISVAT,
	l10n.IN: stripe.TaxIDTypeINGST,
	l10n.ID: stripe.TaxIDTypeIDNPWP,
	l10n.IL: stripe.TaxIDTypeILVAT,
	l10n.JP: stripe.TaxIDTypeJPCN, // TODO: handle other Japanese tax ID types
	l10n.KZ: stripe.TaxIDTypeKzBin,
	l10n.KE: stripe.TaxIDTypeKEPIN,
	l10n.LI: stripe.TaxIDTypeLiVAT, // TODO handle Liechtenstein UID number
	l10n.MY: stripe.TaxIDTypeMYITN, // TODO handle Malaysian FRP and SST numbers
	l10n.MR: stripe.TaxIDTypeMrNif,
	l10n.MX: stripe.TaxIDTypeMXRFC,
	l10n.MD: stripe.TaxIDTypeMdVAT,
	l10n.ME: stripe.TaxIDTypeMePib,
	l10n.MA: stripe.TaxIDTypeMaVAT,
	l10n.NP: stripe.TaxIDTypeNpPan,
	l10n.NZ: stripe.TaxIDTypeNZGST,
	l10n.NG: stripe.TaxIDTypeNgTin,
	l10n.MK: stripe.TaxIDTypeMkVAT,
	l10n.OM: stripe.TaxIDTypeOmVAT,
	l10n.PE: stripe.TaxIDTypePERUC,
	l10n.PH: stripe.TaxIDTypePHTIN,
	l10n.RO: stripe.TaxIDTypeROTIN,
	l10n.RU: stripe.TaxIDTypeRUINN, // TODO handle KPP
	l10n.SA: stripe.TaxIDTypeSAVAT,
	l10n.SN: stripe.TaxIDTypeSnNinea,
	l10n.RS: stripe.TaxIDTypeRSPIB,
	l10n.SG: stripe.TaxIDTypeSGGST, // TODO: handle UEN
	l10n.SI: stripe.TaxIDTypeSITIN,
	l10n.ZA: stripe.TaxIDTypeZAVAT,
	l10n.KR: stripe.TaxIDTypeKRBRN,
	l10n.SR: stripe.TaxIDTypeSrFin,
	l10n.CH: stripe.TaxIDTypeCHVAT, // TODO: handle UID
	l10n.TW: stripe.TaxIDTypeTWVAT,
	l10n.TJ: stripe.TaxIDTypeTjTin,
	l10n.TZ: stripe.TaxIDTypeTzVAT,
	l10n.TH: stripe.TaxIDTypeTHVAT,
	l10n.TR: stripe.TaxIDTypeTRTIN,
	l10n.UG: stripe.TaxIDTypeUgTin,
	l10n.UA: stripe.TaxIDTypeUAVAT,
	l10n.AE: stripe.TaxIDTypeAETRN,
	l10n.GB: stripe.TaxIDTypeGBVAT,
	l10n.US: stripe.TaxIDTypeUSEIN, // TODO: determine if this should be here
	l10n.UY: stripe.TaxIDTypeUYRUC,
	l10n.UZ: stripe.TaxIDTypeUzTin,
	l10n.VE: stripe.TaxIDTypeVERIF,
	l10n.VN: stripe.TaxIDTypeVNTIN,
	l10n.ZM: stripe.TaxIDTypeZmTin,
	l10n.ZW: stripe.TaxIDTypeZwTin,
}

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

	if tID.Country.Code() == l10n.EU {
		return &stripe.CustomerTaxIDDataParams{
			Type:  stripe.String(string(stripe.TaxIDTypeEUOSSVAT)),
			Value: stripe.String(tID.String()), // Include the country code.
		}
	}

	if typ, ok := taxIDMapGOBLToStripe[tID.Country.Code()]; ok {
		return &stripe.CustomerTaxIDDataParams{
			Type:  stripe.String(string(typ)),
			Value: stripe.String(tID.Code.String()),
		}
	}

	return nil
}

// ToTaxIDFromOrg expects a GOBL org Identity and attempts to convert it into a
// Stripe tax ID object.
func ToTaxIDFromOrg(id *org.Identity) *stripe.TaxID {
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

// FromTaxIDToOrg converts a stripe tax ID object into a GOBL organization identity, if possible.
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
	oid.Normalize()
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

	if customer.Email != "" {
		customerParty = new(org.Party)
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

	if customer.Address != nil {
		if customerParty == nil {
			customerParty = new(org.Party)
		}
		customerParty.Addresses = append(customerParty.Addresses, FromAddress(customer.Address))
	}

	if len(customer.Metadata) != 0 {
		customerParty.Ext = newExtensionsWithPrefix(customer.Metadata, customDataCustomerExt)
	}

	return customerParty
}

// newCustomerFromInvoice creates the customer data from the invoice object
func newCustomerFromInvoice(doc *stripe.Invoice) *org.Party {
	/*
		Here, we will assume that the customer data is already in the invoice object
		in fields like customer_name, customer_email, etc.
	*/

	var customerParty *org.Party

	if doc.CustomerEmail != "" {
		customerParty = new(org.Party)
		customerParty.Emails = append(customerParty.Emails, FromEmail(doc.CustomerEmail))
	}

	if doc.CustomerName != "" {
		if customerParty == nil {
			customerParty = new(org.Party)
		}
		customerParty.Name = doc.CustomerName
	}

	if doc.CustomerPhone != "" {
		if customerParty == nil {
			customerParty = new(org.Party)
		}
		customerParty.Telephones = append(customerParty.Telephones, FromTelephone(doc.CustomerPhone))
	}

	if doc.CustomerTaxIDs != nil {
		if len(doc.CustomerTaxIDs) > 0 {
			if customerParty == nil {
				customerParty = new(org.Party)
			}
			// In Stripe there are 2 objects for the taxID: stripe.TaxID and stripe.CustomerTaxID
			stripeTaxID := &stripe.TaxID{
				Type:  *doc.CustomerTaxIDs[0].Type,
				Value: doc.CustomerTaxIDs[0].Value,
			}

			if slices.Contains(orgIDKeys, stripeTaxID.Type) {
				customerParty.Identities = []*org.Identity{FromTaxIDToOrg(stripeTaxID)}
			} else {
				customerParty.TaxID = FromTaxIDToTax(stripeTaxID)
			}
		}
	}

	if doc.CustomerAddress != nil {
		if customerParty == nil {
			customerParty = new(org.Party)
		}
		customerParty.Addresses = append(customerParty.Addresses, FromAddress(doc.CustomerAddress))
	}

	return customerParty
}

// newSupplierFromInvoice creates a basic supplier from the invoice account data
func newSupplierFromInvoice(doc *stripe.Invoice) *org.Party {
	if doc == nil {
		return nil
	}
	var supplierParty *org.Party

	if doc.AccountName != "" {
		supplierParty = &org.Party{
			Name: doc.AccountName,
		}
	}

	if len(doc.AccountTaxIDs) > 0 {
		// Only process if the tax ID has a created timestamp (indicating it's expanded)
		if doc.AccountTaxIDs[0].Created != 0 {
			if supplierParty == nil {
				supplierParty = new(org.Party)
			}
			if slices.Contains(orgIDKeys, doc.AccountTaxIDs[0].Type) {
				supplierParty.Identities = []*org.Identity{FromTaxIDToOrg(doc.AccountTaxIDs[0])}
			} else {
				supplierParty.TaxID = FromTaxIDToTax(doc.AccountTaxIDs[0])
			}
		}
	}

	return supplierParty
}

// NewSupplierFromAccount creates a GOBL supplier from the Stripe account object
func NewSupplierFromAccount(account *stripe.Account) *org.Party {
	if account == nil {
		return nil
	}
	var supplierParty *org.Party
	if account.BusinessProfile != nil {
		if account.BusinessProfile.Name != "" {
			supplierParty = &org.Party{
				Name: account.BusinessProfile.Name,
			}
		}

		if account.BusinessProfile.SupportAddress != nil {
			if supplierParty == nil {
				supplierParty = new(org.Party)
			}
			supplierParty.Addresses = append(supplierParty.Addresses, FromAddress(account.BusinessProfile.SupportAddress))
		}

		if account.BusinessProfile.SupportEmail != "" {
			if supplierParty == nil {
				supplierParty = new(org.Party)
			}
			supplierParty.Emails = append(supplierParty.Emails, FromEmail(account.BusinessProfile.SupportEmail))
		}

		if account.BusinessProfile.SupportPhone != "" {
			if supplierParty == nil {
				supplierParty = new(org.Party)
			}
			supplierParty.Telephones = append(supplierParty.Telephones, FromTelephone(account.BusinessProfile.SupportPhone))
		}
	}

	if account.Settings != nil {
		if account.Settings.Invoices != nil {
			if account.Settings.Invoices.DefaultAccountTaxIDs != nil {
				if len(account.Settings.Invoices.DefaultAccountTaxIDs) > 0 {
					accountTaxID := account.Settings.Invoices.DefaultAccountTaxIDs[0]

					if accountTaxID.Created != 0 {
						if supplierParty == nil {
							supplierParty = new(org.Party)
						}
						if slices.Contains(orgIDKeys, accountTaxID.Type) {
							supplierParty.Identities = []*org.Identity{FromTaxIDToOrg(accountTaxID)}
						} else {
							supplierParty.TaxID = FromTaxIDToTax(accountTaxID)
						}
					}
				}
			}
		}

	}

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

// ToAddressParams converts the GOBL address object into a stripe address object.
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
