package goblstripe

import (
	"fmt"
	"strings"
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

// Meta constants used in the Stripe to GOBL conversion
const (
	MetaKeyStripeDocID   = "stripe-document-id"
	MetaKeyStripeDocType = "stripe-document-type"
	MetaKeyStripeEnv     = "stripe-env" // The environment in which the invoice was created
)

// Document type constants used in the Stripe to GOBL conversion
const (
	StripeDocTypeInvoice    = "invoice"
	StripeDocTypeCreditNote = "credit_note"
)

// Custom field constants used in the Stripe to GOBL conversion
const (
	CustomFieldPONumber = "po number"
)

// ToInvoice converts a GOBL bill.Invoice into a stripe invoice object.
// TODO: Implement
/*func ToInvoice(inv *bill.Invoice) (*stripe.Invoice, error) {
	return nil, nil
}*/

// FromInvoice converts a stripe invoice object into a GOBL bill.Invoice.
func FromInvoice(doc *stripe.Invoice) (*bill.Invoice, error) {
	inv := new(bill.Invoice)
	inv.Type = bill.InvoiceTypeStandard

	regimeDef, err := regimeFromInvoice(doc)
	if err != nil {
		return nil, err
	}

	inv.UUID = uuid.V7() // Generated randomly, but you can modify afterwards for the specific use case.

	if doc.Number != "" {
		// Split the invoice number by "-" to separate series and code
		parts := strings.Split(doc.Number, "-")
		if len(parts) > 1 {
			inv.Series = cbc.Code(parts[0])
			inv.Code = cbc.Code(parts[1]) // Sequential code used to identify this invoice in tax declarations.
		} else {
			inv.Code = cbc.Code(doc.Number) // No separator found, use the whole number as code
		}
	} else {
		inv.Code = cbc.Code(doc.ID)
	}

	inv.Meta = cbc.Meta{
		MetaKeyStripeDocID:   doc.ID,
		MetaKeyStripeDocType: StripeDocTypeInvoice,
	}

	if doc.EffectiveAt != 0 {
		inv.OperationDate = newDateFromTS(doc.EffectiveAt) // Date when the operation defined by the invoice became effective
	}

	inv.Currency = FromCurrency(doc.Currency)
	inv.ExchangeRates = newExchangeRates(inv.Currency, regimeDef)

	inv.Supplier = newSupplierFromInvoice(doc)
	if doc.Customer != nil && len(doc.Customer.Metadata) != 0 {
		inv.Customer = FromCustomer(doc.Customer)
	} else {
		inv.Customer = newCustomerFromInvoice(doc)
	}

	inv.Tags = newTags(doc)

	inv.Lines = FromInvoiceLines(doc.Lines.Data, regimeDef)
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
func FromCreditNote(doc *stripe.CreditNote) (*bill.Invoice, error) {
	inv := new(bill.Invoice)
	inv.Type = bill.InvoiceTypeCreditNote

	regimeDef, err := regimeFromInvoice(doc.Invoice)
	if err != nil {
		return nil, err
	}

	inv.UUID = uuid.V4() // Generated randomly, but you can modify afterwards for the specific use case.

	if doc.Number != "" {
		inv.Code = cbc.Code(doc.Number)
	} else {
		inv.Code = cbc.Code(doc.ID)
	}

	inv.Meta = cbc.Meta{
		MetaKeyStripeDocID:   doc.ID,
		MetaKeyStripeDocType: StripeDocTypeCreditNote,
	}

	if doc.EffectiveAt != 0 {
		inv.OperationDate = newDateFromTS(doc.EffectiveAt) // Date when the operation defined by the credit note became effective
	}

	inv.Currency = FromCurrency(doc.Currency)
	inv.ExchangeRates = newExchangeRates(inv.Currency, regimeDef)

	inv.Supplier = newSupplierFromInvoice(doc.Invoice)
	if doc.Customer != nil {
		inv.Customer = FromCustomer(doc.Customer)
	}

	inv.Lines = FromCreditNoteLines(doc.Lines.Data, inv.Currency, regimeDef)
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
func newTags(doc *stripe.Invoice) tax.Tags {
	if doc.CustomerTaxExempt != nil {
		if *doc.CustomerTaxExempt == stripe.CustomerTaxExemptReverse {
			return tax.WithTags(tax.TagReverseCharge)
		}
	}

	for _, taxAmount := range doc.TotalTaxAmounts {
		if taxAmount.TaxabilityReason == stripe.InvoiceTotalTaxAmountTaxabilityReasonReverseCharge {
			return tax.WithTags(tax.TagReverseCharge)
		}
	}

	return tax.Tags{}
}

// newOrdering creates an ordering object from an invoice.
func newOrdering(doc *stripe.Invoice) *bill.Ordering {
	ordering := &bill.Ordering{
		Period: &cal.Period{
			Start: *newDateFromTS(doc.PeriodStart),
			End:   *newDateFromTS(doc.PeriodEnd),
		},
	}
	if doc.CustomFields != nil {
		for _, field := range doc.CustomFields {
			if strings.ToLower(strings.TrimSpace(field.Name)) == CustomFieldPONumber {
				ordering.Code = cbc.Code(field.Value)
				break
			}
		}
	}
	return ordering
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

// ExpectedInvoiceTotal returns the expected total of an invoice.
func ExpectedInvoiceTotal(doc *stripe.Invoice) num.Amount {
	return currencyAmount(doc.Total, FromCurrency(doc.Currency))
}

// ExpectedCreditNoteTotal returns the expected total of a credit note.
func ExpectedCreditNoteTotal(doc *stripe.CreditNote) num.Amount {
	return currencyAmount(doc.Total, FromCurrency(doc.Currency))
}
