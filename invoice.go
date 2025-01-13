package goblstripe

import (
	"github.com/invopop/gobl/bill"
	"github.com/stripe/stripe-go/v81"
)

// ToInvoice converts a GOBL bill.Invoice into a stripe invoice object.
func ToInvoice(inv *bill.Invoice) (*stripe.Invoice, error) {
	return nil, nil
}

// FromInvoice converts a stripe invoice object into a GOBL bill.Invoice.
func FromInvoice(doc *stripe.Invoice) (*bill.Invoice, error) {
	inv := new(bill.Invoice)
	inv.Type = bill.InvoiceTypeStandard

	return inv, nil
}

func FromCreditNote(doc *stripe.CreditNote) (*bill.Invoice, error) {
	inv := new(bill.Invoice)
	inv.Type = bill.InvoiceTypeCreditNote

	return inv, nil
}
