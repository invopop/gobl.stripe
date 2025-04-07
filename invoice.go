package goblstripe

import (
	"fmt"
	"time"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
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
func FromInvoice(doc *stripe.Invoice) (*bill.Invoice, error) {
	inv := new(bill.Invoice)
	inv.Type = bill.InvoiceTypeStandard

	regimeDef, err := regimeFromInvoice(doc)
	if err != nil {
		return nil, err
	}

	inv.UUID = uuid.V7() // Generated randomly, but you can modify afterwards for the specific use case.

	inv.Code = cbc.Code(doc.ID) //Sequential code used to identify this invoice in tax declarations.

	inv.IssueDate = cal.DateOf(time.Unix(doc.Created, 0).UTC()) //Date when the invoice was created
	if doc.EffectiveAt != 0 {
		inv.OperationDate = newDateFromTS(doc.EffectiveAt) // Date when the operation defined by the invoice became effective
	}

	inv.Currency = FromCurrency(doc.Currency)
	inv.ExchangeRates = newExchangeRates(inv.Currency, regimeDef)

	inv.Supplier = newSupplierFromInvoice(doc)
	inv.Customer = newCustomerFromInvoice(doc)
	if doc.CustomerTaxExempt != nil {
		inv.Tags = newTags(*doc.CustomerTaxExempt)
	}

	inv.Lines = FromInvoiceLines(doc.Lines.Data)
	inv.Tax = taxFromInvoiceTaxAmounts(doc.TotalTaxAmounts)
	inv.Ordering = newOrdering(doc)
	inv.Delivery = newDelivery(doc)
	inv.Payment = newPayment(doc)

	//Remaining fields
	//Addons: TODO
	//Discounts: for the moment not considered in general (only in lines)

	return inv, nil
}

// FromCreditNote converts a stripe credit note object into a GOBL bill.Invoice.
// The namespace is the UUID of the enrollment.
func FromCreditNote(doc *stripe.CreditNote) (*bill.Invoice, error) {
	inv := new(bill.Invoice)
	inv.Type = bill.InvoiceTypeCreditNote

	regimeDef, err := regimeFromInvoice(doc.Invoice)
	if err != nil {
		return nil, err
	}

	inv.UUID = uuid.V4() // Generated randomly, but you can modify afterwards for the specific use case.

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
	docRef := new(org.DocumentRef)

	if doc.Number != "" {
		parts := strings.Split(doc.Number, "-")
		if len(parts) > 1 {
			docRef.Series = cbc.Code(parts[0])
			docRef.Code = cbc.Code(parts[1]) // Sequential code used to identify this invoice in tax declarations.
		} else {
			docRef.Code = cbc.Code(doc.Number)
		}
	} else {
		docRef.Code = cbc.Code(doc.ID)
	}

	docRef.IssueDate = newDateFromTS(doc.Created)
	docRef.Type = bill.InvoiceTypeStandard
	docRef.Reason = reason

	return docRef
}

// newTags creates a tax tags object from a customer tax exempt status.
func newTags(customerExempt stripe.CustomerTaxExempt) tax.Tags {
	if customerExempt == stripe.CustomerTaxExemptReverse {
		return tax.WithTags(tax.TagReverseCharge)
	}

	return tax.Tags{}
}

// newOrdering creates an ordering object from an invoice.
func newOrdering(doc *stripe.Invoice) *bill.Ordering {
	return &bill.Ordering{
		Period: &cal.Period{
			Start: *newDateFromTS(doc.PeriodStart),
			End:   *newDateFromTS(doc.PeriodEnd),
		},
	}
}

// AdjustRounding checks and, if need be, adjusts the rounding in the GOBL invoice to match the
// Stripe payable total. Stripe calculates totals by rounding each line and then summing
// which can lead to a mismatch with the total amount in GOBL.
func AdjustRounding(gi *bill.Invoice, total int64, curr stripe.Currency) error {
	// Calculate the difference between the expected and the calculated totals
	exp := currencyAmount(total, FromCurrency(curr))
	diff := exp.Subtract(gi.Totals.Payable)
	if diff.IsZero() {
		// No difference. No adjustment needed
		return nil
	}

	// Check if the difference can be attributed to rounding
	maxErr := MaxRoundingError(gi)
	if diff.Abs().Compare(maxErr) == 1 {
		// Too much difference. Report the error
		return fmt.Errorf("rounding error in totals too high: %s", diff)
	}

	gi.Totals.Rounding = &diff

	return nil
}

// MaxRoundingError returns the maximum error that can be attributed to rounding in an invoice.
func MaxRoundingError(gi *bill.Invoice) num.Amount {
	// 0.5 of the smallest subunit of the currency per line
	return num.MakeAmount(5*int64(len(gi.Lines)), gi.Currency.Def().Subunits+1)
}
