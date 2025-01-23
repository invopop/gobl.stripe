package goblstripe

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/gobl/uuid"
	"github.com/stripe/stripe-go/v81"
)

type paymentMethodDef struct {
	Key         string
	Description string
	MeansKey    cbc.Key
}

// Payment method definitions to convert Stripe payment methods to GOBL payment means
var paymentMethodDefinitions = []paymentMethodDef{
	{"ach_debit", "ACH", pay.MeansKeyDirectDebit},
	{"acss_debit", "Canadian pre-authorized debit", pay.MeansKeyDirectDebit},
	{"amazon_pay", "Amazon Pay", pay.MeansKeyOnline},
	{"bacs_debit", "Bacs Direct Debit", pay.MeansKeyDirectDebit},
	{"au_becs_debit", "BECS Direct Debit", pay.MeansKeyDirectDebit},
	{"bancontact", "Bancontact", pay.MeansKeyOnline},
	{"boleto", "boleto", pay.MeansKeyOnline},
	{"card", "Card", pay.MeansKeyCard},
	{"cashapp", "Cash App Pay", pay.MeansKeyOnline},
	{"customer_balance", "Bank Transfer", pay.MeansKeyCreditTransfer},
	{"eps", "EPS", pay.MeansKeyOnline},
	{"fpx", "FPX", pay.MeansKeyOnline},
	{"giropay", "giropay", pay.MeansKeyOnline},
	{"grabpay", "GrabPay", pay.MeansKeyOnline},
	{"ideal", "iDEAL", pay.MeansKeyOnline},
	{"kakao_pay", "Kakao Pay", pay.MeansKeyOnline},
	{"konbini", "Konbini", pay.MeansKeyOnline},
	{"kr_card", "Korean Credit Card", pay.MeansKeyCard},
	{"link", "Link", pay.MeansKeyOnline},
	{"multibanco", "Multibanco", pay.MeansKeyOnline},
	{"naver_pay", "Naver Pay", pay.MeansKeyOnline},
	{"p24", "Przelewy24", pay.MeansKeyOnline},
	{"payco", "PAYCO", pay.MeansKeyOnline},
	{"paynow", "PayNow", pay.MeansKeyOnline},
	{"paypal", "PayPal", pay.MeansKeyOnline},
	{"promptpay", "PromptPay", pay.MeansKeyOnline},
	{"revolut_pay", "Revolut Pay", pay.MeansKeyOnline},
	{"sepa_debit", "SEPA Direct Debit", pay.MeansKeyDirectDebit},
	{"sofort", "Sofort", pay.MeansKeyOnline},
	{"us_bank_account", "ACH direct debit", pay.MeansKeyDirectDebit},
	{"wechat_pay", "WeChat Pay", pay.MeansKeyOnline},
}

// https://docs.stripe.com/currencies#zero-decimal
var zeroDecimalCurrencies = []currency.Code{currency.BIF, currency.CLF, currency.DJF, currency.GNF, currency.JPY, currency.KMF,
	currency.KRW, currency.MGA, currency.PYG, currency.RWF, currency.VND, currency.VUV, currency.XAF, currency.XOF, currency.XPF}

// ToInvoice converts a GOBL bill.Invoice into a stripe invoice object.
func ToInvoice(inv *bill.Invoice) (*stripe.Invoice, error) {
	return nil, nil
}

// FromInvoice converts a stripe invoice object into a GOBL bill.Invoice.
// The namespace is the UUID of the enrollment.
func FromInvoice(doc *stripe.Invoice, namespace uuid.UUID) (*bill.Invoice, error) {
	inv := new(bill.Invoice)
	inv.Type = bill.InvoiceTypeStandard

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

// FromCurrency converts a stripe currency into a GOBL currency code.
func FromCurrency(curr stripe.Currency) currency.Code {
	return currency.Code(strings.ToUpper(string(curr)))
}

// newExchangeRates creates the exchange rates for the invoice.
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

// taxFromInvoiceTaxAmounts creates a tax object from the tax amounts in an invoice.
func taxFromInvoiceTaxAmounts(taxAmounts []*stripe.InvoiceTotalTaxAmount) *bill.Tax {
	var t *bill.Tax

	if len(taxAmounts) == 0 {
		return nil
	}

	// We just check the first tax
	if taxAmounts[0].Inclusive {
		t = new(bill.Tax)
		t.PricesInclude = extractTaxCat(taxAmounts[0].TaxRate.TaxType)
		return t
	}

	return nil
}

// taxFromCreditNoteTaxAmounts creates a tax object from the tax amounts in a credit note.
func taxFromCreditNoteTaxAmounts(taxAmounts []*stripe.CreditNoteTaxAmount) *bill.Tax {
	var t *bill.Tax

	if len(taxAmounts) == 0 {
		return nil
	}

	// We just check the first tax
	if taxAmounts[0].Inclusive {
		t = new(bill.Tax)
		t.PricesInclude = extractTaxCat(taxAmounts[0].TaxRate.TaxType)
		return t
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

// newDelivery creates a delivery object from an invoice.
func newDelivery(doc *stripe.Invoice) *bill.Delivery {
	if doc.ShippingDetails != nil {
		return FromShippingDetailsToDelivery(doc.ShippingDetails)
	}

	if doc.CustomerShipping != nil {
		return FromShippingDetailsToDelivery(doc.CustomerShipping)
	}

	// If no shipping details are provided, return nil
	return nil
}

// FromShippingDetailsToDelivery converts a stripe shipping details object into a GOBL delivery object.
func FromShippingDetailsToDelivery(shipping *stripe.ShippingDetails) *bill.Delivery {
	receiver := newReceiver(shipping)
	if receiver.Validate() != nil {
		return nil
	}
	return &bill.Delivery{
		Receiver: receiver,
	}
}

// newReceiver creates a receiver object from shipping details.
func newReceiver(shipping *stripe.ShippingDetails) *org.Party {
	return &org.Party{
		Name:       shipping.Name,
		Addresses:  []*org.Address{FromAddress(shipping.Address)},
		Telephones: []*org.Telephone{FromTelephone(shipping.Phone)},
	}
}

// newPayment creates a GOBL payment object from a Stripe invoice.
func newPayment(doc *stripe.Invoice) *bill.Payment {
	var p *bill.Payment

	if terms := newPaymentTerms(doc); terms != nil {
		p = &bill.Payment{
			Terms: terms,
		}
	}

	if instructions := newPaymentInstructions(doc); instructions != nil {
		if p == nil {
			p = new(bill.Payment)
		}
		p.Instructions = instructions
	}

	if advances := newPaymentAdvances(doc); advances != nil {
		if p == nil {
			p = new(bill.Payment)
		}
		p.Advances = advances
	}

	return p
}

// newPaymentTerms creates a payment terms object from a Stripe invoice.
func newPaymentTerms(doc *stripe.Invoice) *pay.Terms {
	if doc.Paid {
		return nil
	}

	return &pay.Terms{
		DueDates: []*pay.DueDate{
			{
				Date:    newDateFromTS(doc.DueDate),
				Percent: num.NewPercentage(1, 0),
			},
		},
	}
}

// newPaymentInstructions creates a payment instructions object from a Stripe invoice.
func newPaymentInstructions(doc *stripe.Invoice) *pay.Instructions {
	if doc.Paid {
		return nil
	}

	var instructions *pay.Instructions
	for _, method := range doc.PaymentIntent.PaymentMethodTypes {
		for _, def := range paymentMethodDefinitions {
			if method == def.Key {
				if instructions == nil {
					instructions = new(pay.Instructions)
					instructions.Key = def.MeansKey
					instructions.Detail = def.Description
				} else {
					instructions.Key = instructions.Key.With(def.MeansKey)
					instructions.Detail += ", " + def.Description
				}
			}
		}
	}

	return instructions
}

// newPaymentAdvances creates a payment advances object from a Stripe invoice.
func newPaymentAdvances(doc *stripe.Invoice) []*pay.Advance {
	if doc.Paid || doc.AmountPaid == 0 {
		return nil
	}

	//TODO: How can we get previous payments? I have not seen any examples on these cases
	// I believe it would be better to wait for a use case to implement this

	// We could get all charges by doing a get to the following endpoint:
	// https://api.stripe.com/v1/charges?invoice={invoice_id}

	// For the moment we can create an advance object with the amount paid
	advance := &pay.Advance{
		Amount:      currencyAmount(doc.AmountPaid, FromCurrency(doc.Currency)),
		Description: "Advance payment",
		Date:        newDateFromTS(doc.Charge.Created),
	}
	return []*pay.Advance{advance}
}

// newDateFromTS creates a cal date object from a Unix timestamp.
func newDateFromTS(ts int64) *cal.Date {
	d := cal.DateOf(time.Unix(ts, 0).UTC())
	return &d
}

// currencyAmount creates a currency amount object from a value and a currency code.
func currencyAmount(val int64, curr currency.Code) num.Amount {
	var exp uint32 = 2
	if slices.Contains(zeroDecimalCurrencies, curr) {
		exp = 0
	}

	return num.MakeAmount(val, exp)
}

// isCustomerExempt checks if a customer is exempt from taxes.
func isCustomerExempt(doc *stripe.Invoice) bool {
	return *doc.CustomerTaxExempt == stripe.CustomerTaxExemptExempt || *doc.CustomerTaxExempt == stripe.CustomerTaxExemptReverse
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
