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

	if doc.AccountCountry == "" {
		return nil, fmt.Errorf("missing account country")
	}
	inv.Regime = tax.WithRegime(l10n.TaxCountryCode(doc.AccountCountry)) //The country of the business associated with this invoice, most often the business creating the invoice.
	if inv.Regime.RegimeDef() == nil {
		return nil, fmt.Errorf("missing regime definition for %s", doc.AccountCountry)
	}

	if doc.AccountName == "" {
		return nil, fmt.Errorf("missing account name")
	}
	inv.UUID = invoiceUUID(namespace, doc.AccountName, doc.ID)

	inv.Code = cbc.Code(doc.ID) //Sequential code used to identify this invoice in tax declarations.

	inv.IssueDate = cal.DateOf(time.Unix(doc.Created, 0).UTC()) //Date when the invoice was created
	if doc.EffectiveAt != 0 {
		inv.OperationDate = newDateFromTS(doc.EffectiveAt) // Date when the operation defined by the invoice became effective
	}

	inv.Currency = FromCurrency(doc.Currency)
	inv.ExchangeRates = newExchangeRates(inv.Currency, inv.Regime.RegimeDef())

	inv.Supplier = newSupplier(doc)
	if doc.Customer != nil {
		inv.Customer = FromCustomer(doc.Customer)
	} else {
		inv.Customer = newCustomer(doc)
	}

	inv.Lines = FromLines(doc.Lines.Data)
	inv.Tax = FromTax(doc)
	inv.Ordering = newOrdering(doc)
	inv.Delivery = newDelivery(doc)
	inv.Payment = newPayment(doc)

	//Remaining fields
	//Addons: TODO
	//Tags: TODO
	//Discounts: for the moment not considered in general (only in lines)

	return inv, nil
}

func FromCreditNote(doc *stripe.CreditNote) (*bill.Invoice, error) {
	// Credit note: post_payment_credit_notes_amount, pre_payment_credit_notes_amount
	// Preceding field
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

	// We just check the first tax
	if doc.TotalTaxAmounts[0].Inclusive {
		t = new(bill.Tax)
		t.PricesInclude = extractTaxCat(doc.TotalTaxAmounts[0].TaxRate.TaxType)
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

func newOrdering(doc *stripe.Invoice) *bill.Ordering {
	ordering := new(bill.Ordering)
	ordering.Period = newOrderingPeriod(doc.Lines.Data)
	return ordering
}

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

func FromShippingDetailsToDelivery(shipping *stripe.ShippingDetails) *bill.Delivery {
	receiver := newReceiver(shipping)
	if receiver.Validate() != nil {
		return nil
	}
	return &bill.Delivery{
		Receiver: receiver,
	}
}

func newReceiver(shipping *stripe.ShippingDetails) *org.Party {
	return &org.Party{
		Name:       shipping.Name,
		Addresses:  []*org.Address{FromAddress(shipping.Address)},
		Telephones: []*org.Telephone{FromTelephone(shipping.Phone)},
	}
}

func newPayment(doc *stripe.Invoice) *bill.Payment {
	// Collection method: Either charge_automatically, or send_invoice. When
	//charging automatically, Stripe will attempt to pay this invoice using the
	//default source attached to the customer. When sending an invoice, Stripe
	//will email this invoice to the customer with payment instructions.
	// due_date: The date on which payment for this invoice is due. This value
	//will be null for invoices where collection_method=charge_automatically.
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

func newDateFromTS(ts int64) *cal.Date {
	d := cal.DateOf(time.Unix(ts, 0).UTC())
	return &d
}

func currencyAmount(val int64, curr currency.Code) num.Amount {
	var exp uint32 = 2
	if slices.Contains(zeroDecimalCurrencies, curr) {
		exp = 0
	}

	return num.MakeAmount(val, exp)
}

func FromCurrency(curr stripe.Currency) currency.Code {
	return currency.Code(strings.ToUpper(string(curr)))
}

func isCustomerExempt(doc *stripe.Invoice) bool {
	return *doc.CustomerTaxExempt == stripe.CustomerTaxExemptExempt || *doc.CustomerTaxExempt == stripe.CustomerTaxExemptReverse
}
