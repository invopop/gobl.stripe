package goblstripe

import (
	"fmt"
	"time"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/gobl/uuid"
	"github.com/stripe/stripe-go/v81"
)

// ToInvoice converts a GOBL bill.Invoice into a stripe invoice object.
// TODO: Implement
/*func ToInvoice(inv *bill.Invoice) (*stripe.Invoice, error) {
	return nil, nil
}*/

// FromInvoice converts a stripe invoice object into a GOBL bill.Invoice.
// The namespace is the UUID of the enrollment.
func FromInvoice(doc *stripe.Invoice, namespace uuid.UUID) (*bill.Invoice, error) {
	inv := new(bill.Invoice)
	inv.Type = bill.InvoiceTypeStandard

	// TODO: Add support for reverse charge invoices

	regimeDef, err := regimeFromInvoice(doc)
	if err != nil {
		return nil, err
	}

	inv.UUID, err = uuidFromInvoice(doc, namespace)
	if err != nil {
		return nil, err
	}

	inv.Code = cbc.Code(doc.ID) //Sequential code used to identify this invoice in tax declarations.

	inv.IssueDate = cal.DateOf(time.Unix(doc.Created, 0).UTC()) //Date when the invoice was created
	if doc.EffectiveAt != 0 {
		inv.OperationDate = newDateFromTS(doc.EffectiveAt) // Date when the operation defined by the invoice became effective
	}

	inv.Currency = FromCurrency(doc.Currency)
	inv.ExchangeRates = newExchangeRates(inv.Currency, regimeDef)

	inv.Supplier = newSupplierFromInvoice(doc)
	if doc.Customer != nil {
		inv.Customer = FromCustomer(doc.Customer)
		inv.Tags = newTags(doc.Customer)
	} else {
		inv.Customer = newCustomerFromInvoice(doc)
	}

	inv.Lines = FromInvoiceLines(doc.Lines.Data)
	inv.Tax = taxFromInvoiceTaxAmounts(doc.TotalTaxAmounts)
	inv.Ordering = newOrdering(doc)
	inv.Delivery = newDelivery(doc)
	inv.Payment = newPayment(doc)

	//Remaining fields
	//Addons: TODO
	//Tags: TODO
	//Discounts: for the moment not considered in general (only in lines)

	return inv, nil
}

// FromCreditNote converts a stripe credit note object into a GOBL bill.Invoice.
// The namespace is the UUID of the enrollment.
func FromCreditNote(doc *stripe.CreditNote, namespace uuid.UUID) (*bill.Invoice, error) {
	inv := new(bill.Invoice)
	inv.Type = bill.InvoiceTypeCreditNote

	regimeDef, err := regimeFromInvoice(doc.Invoice)
	if err != nil {
		return nil, err
	}

	inv.UUID, err = uuidFromInvoice(doc.Invoice, namespace)
	if err != nil {
		return nil, err
	}

	inv.Code = cbc.Code(doc.ID) //Sequential code used to identify this credit note in tax declarations.

	inv.IssueDate = cal.DateOf(time.Unix(doc.Created, 0).UTC()) //Date when the credit note was created
	if doc.EffectiveAt != 0 {
		inv.OperationDate = newDateFromTS(doc.EffectiveAt) // Date when the operation defined by the credit note became effective
	}

	inv.Currency = FromCurrency(doc.Currency)
	inv.ExchangeRates = newExchangeRates(inv.Currency, regimeDef)

	inv.Supplier = newSupplierFromInvoice(doc.Invoice)
	if doc.Customer != nil {
		inv.Customer = FromCustomer(doc.Customer)
	}

	inv.Lines = FromCreditNoteLines(doc.Lines.Data, inv.Currency)
	inv.Tax = taxFromCreditNoteTaxAmounts(doc.TaxAmounts)
	inv.Preceding = []*org.DocumentRef{newPrecedingFromInvoice(doc.Invoice, string(doc.Reason))}

	return inv, nil
}

// uuidFromInvoice generates a UUID for an invoice based on the namespace, site, and stripe Invoice ID.
func uuidFromInvoice(doc *stripe.Invoice, namespace uuid.UUID) (uuid.UUID, error) {
	if doc.AccountName == "" {
		return uuid.Empty, fmt.Errorf("missing account name")
	}
	return invoiceUUID(namespace, doc.AccountName, doc.ID), nil
}

// invoiceUUID generates a UUID for a UUID based on the namespace, site, and stripe Invoice ID.
func invoiceUUID(ns uuid.UUID, site string, stID string) uuid.UUID {
	if ns == uuid.Empty {
		return uuid.Empty
	}

	base := site + ":" + stID
	return uuid.V3(ns, []byte(base))
}

// newDateFromTS creates a cal date object from a Unix timestamp.
func newDateFromTS(ts int64) *cal.Date {
	d := cal.DateOf(time.Unix(ts, 0).UTC())
	return &d
}

// regimeFromInvoice creates a tax regime definition from a Stripe invoice.
func regimeFromInvoice(doc *stripe.Invoice) (*tax.RegimeDef, error) {
	if doc.AccountCountry == "" {
		return nil, fmt.Errorf("missing account country")
	}
	regime := tax.WithRegime(l10n.TaxCountryCode(doc.AccountCountry)) //The country of the business associated with this invoice, most often the business creating the invoice.
	if regime.RegimeDef() == nil {
		return nil, fmt.Errorf("missing regime definition for %s", doc.AccountCountry)
	}

	return regime.RegimeDef(), nil
}

// newPrecedingFromInvoice creates a document reference from a Stripe invoice.
func newPrecedingFromInvoice(doc *stripe.Invoice, reason string) *org.DocumentRef {
	return &org.DocumentRef{
		Type:      bill.InvoiceTypeStandard,
		IssueDate: newDateFromTS(doc.Created),
		Code:      cbc.Code(doc.ID),
		Reason:    reason,
	}
}

func newTags(customer *stripe.Customer) tax.Tags {
	if customer.TaxExempt == stripe.CustomerTaxExemptReverse {
		return tax.WithTags(tax.TagReverseCharge)
	}

	return tax.Tags{}
}
