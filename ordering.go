package goblstripe

import (
	"time"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/stripe/stripe-go/v81"
)

// newOrdering creates an ordering object from an invoice.
func newOrdering(doc *stripe.Invoice) *bill.Ordering {
	ordering := new(bill.Ordering)
	ordering.Period = newOrderingPeriod(doc.Lines.Data)
	return ordering
}

// newOrderingPeriod creates an ordering period from invoice line items.
func newOrderingPeriod(lines []*stripe.InvoiceLineItem) *cal.Period {
	from := lines[0].Period.Start
	to := lines[0].Period.End

	for _, line := range lines {
		if from != line.Period.Start || to != line.Period.End {
			//Different periods in the same invoice
			return nil
		}
	}

	return &cal.Period{
		Start: cal.DateOf(time.Unix(from, 0).UTC()),
		End:   cal.DateOf(time.Unix(to, 0).UTC()),
	}
}
