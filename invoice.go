package goblstripe

import (
	"strings"
	"time"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/gobl/uuid"
	"github.com/stripe/stripe-go/v81"
)

type Reverter struct {
	regime         *tax.RegimeDef
	namespace      uuid.UUID
	pricesInclude  bool
	CustomerExempt bool
	IssueDate      cal.Date
}

// ToInvoice converts a GOBL bill.Invoice into a stripe invoice object.
func ToInvoice(inv *bill.Invoice) (*stripe.Invoice, error) {
	return nil, nil
}

// FromInvoice converts a stripe invoice object into a GOBL bill.Invoice.
// The namespace is the UUID of the enrollment.
// The site is the name of the stripe account.
func FromInvoice(doc *stripe.Invoice, namespace uuid.UUID) (*bill.Invoice, error) {
	inv := new(bill.Invoice)
	// livemode false means it is a test invoice, true means it is a real invoice
	inv.Type = bill.InvoiceTypeStandard
	inv.Regime = tax.WithRegime(l10n.TaxCountryCode(doc.AccountCountry)) //The country of the business associated with this invoice, most often the business creating the invoice.
	// Here we could prefer using the tax id from the supplier (account) instead of the account country. Check the Notes in the README.md.

	inv.UUID = invoiceUUID(namespace, doc.AccountName, doc.ID) //The UUID of the invoice
	//addons: TODO
	//tags: TODO
	inv.Code = cbc.Code(doc.ID)                                 //The ID of the invoice
	inv.IssueDate = cal.DateOf(time.Unix(doc.Created, 0).UTC()) //The time received is in seconds since the Unix epoch.
	//Operation Date: TODO, might be the same as the period_end or the issue date. Not sure.
	inv.Currency = currency.Code(strings.ToUpper(string(doc.Currency)))        //The currency of the invoice (it comes in lowercase)
	inv.ExchangeRates = newExchangeRates(inv.Currency, inv.Regime.RegimeDef()) //The exchange rate of the invoice
	// Tax: TODO
	inv.Supplier = FromSupplier(doc) //The supplier of the invoice
	inv.Customer = FromCustomer(doc) //The customer of the invoice

	customerExempt := *doc.CustomerTaxExempt == stripe.CustomerTaxExemptExempt || *doc.CustomerTaxExempt == stripe.CustomerTaxExemptReverse

	// We are assuming the attribute has_more is false, it is used if there is another page of items after this one to fetch
	inv.Lines = FromLines(doc.Lines.Data, customerExempt, inv.IssueDate)
	inv.Tax = FromTax(doc)

	// For the tax we need to check if price includes tax or not

	return inv, nil
}

func FromCreditNote(doc *stripe.CreditNote) (*bill.Invoice, error) {
	inv := new(bill.Invoice)
	inv.Type = bill.InvoiceTypeCreditNote

	return inv, nil
}

func invoiceUUID(ns uuid.UUID, site string, cbID string) uuid.UUID {
	if ns == uuid.Empty {
		return uuid.Empty
	}

	base := site + ":" + cbID
	return uuid.V3(ns, []byte(base))
}

func newExchangeRates(curr currency.Code, regime *tax.RegimeDef) []*currency.ExchangeRate {
	if curr == regime.Currency {
		// The invoice's and the regime's currency are the same. No exchange rate needed.
		return nil
	}

	// Stripe does not provide exchange rates. We will use the default rates and should be updated after the invoice is created.
	rate := &currency.ExchangeRate{
		From:   curr,
		To:     regime.Currency,
		Amount: DefaultRate(curr, regime.Currency),
	}

	return []*currency.ExchangeRate{rate}
}

func FromTax(doc *stripe.Invoice) *bill.Tax {
	var t *bill.Tax

	if len(doc.TotalTaxAmounts) == 0 {
		return nil
	}

	if len(doc.TotalTaxAmounts) == 1 {
		t = new(bill.Tax)
		t.PricesInclude = extractTaxCat(doc.TotalTaxAmounts[0].TaxRate.TaxType)
		return t
	}

	for _, taxes := range doc.TotalTaxAmounts {
		if taxes.Inclusive {
			if extractTaxCat(taxes.TaxRate.TaxType) == tax.CategoryVAT {
				t = new(bill.Tax)
				t.PricesInclude = tax.CategoryVAT
				return t
			}
		}
	}

	return nil
}

// extractTaxCat extracts the tax category from a Stripe tax type.
func extractTaxCat(taxType stripe.TaxRateTaxType) cbc.Code {
	switch taxType {
	case stripe.TaxRateTaxTypeVAT:
		return tax.CategoryVAT
	case stripe.TaxRateTaxTypeSalesTax:
		return tax.CategoryST
	case stripe.TaxRateTaxTypeGST:
		return tax.CategoryGST
	default:
		return ""
	}
}
