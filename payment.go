package goblstripe

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
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

// newPayment creates a GOBL payment object from a Stripe invoice.
func newPayment(doc *stripe.Invoice, regimeDef *tax.RegimeDef) *bill.PaymentDetails {
	var p *bill.PaymentDetails

	if terms := newPaymentTerms(doc, regimeDef); terms != nil {
		p = &bill.PaymentDetails{
			Terms: terms,
		}
	}

	if instructions := newPaymentInstructions(doc); instructions != nil {
		if p == nil {
			p = new(bill.PaymentDetails)
		}
		p.Instructions = instructions
	}

	if advances := newPaymentAdvances(doc, regimeDef); advances != nil {
		if p == nil {
			p = new(bill.PaymentDetails)
		}
		p.Advances = advances
	}

	return p
}

// newPaymentTerms creates a payment terms object from a Stripe invoice.
func newPaymentTerms(doc *stripe.Invoice, regimeDef *tax.RegimeDef) *pay.Terms {
	if doc.Paid || doc.DueDate == 0 {
		return nil
	}

	return &pay.Terms{
		DueDates: []*pay.DueDate{
			{
				Date:    newDateFromTS(doc.DueDate, regimeDef.TimeLocation()),
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

	// We first check the charge in case it is a direct debit (it is not paid but it has a charge)
	if doc.Charge != nil && doc.Charge.PaymentMethodDetails != nil {
		for _, def := range paymentMethodDefinitions {
			if string(doc.Charge.PaymentMethodDetails.Type) == def.Key {
				instructions := &pay.Instructions{
					Key:    def.MeansKey,
					Detail: def.Description,
				}
				if doc.Charge.PaymentMethodDetails.Type == stripe.ChargePaymentMethodDetailsTypeSEPADebit {
					if doc.Charge.PaymentMethodDetails.SEPADebit != nil {
						instructions.DirectDebit = &pay.DirectDebit{
							Ref: doc.Charge.PaymentMethodDetails.SEPADebit.Mandate,
						}
					}
				}
				return instructions
			}
		}
	}

	// First check for default payment method
	if doc.DefaultPaymentMethod != nil {
		for _, def := range paymentMethodDefinitions {
			if string(doc.DefaultPaymentMethod.Type) == def.Key {
				return &pay.Instructions{
					Key:    def.MeansKey,
					Detail: def.Description,
				}
			}
		}
	}

	// Then check for the default payment method in customer settings
	if doc.Customer != nil && doc.Customer.InvoiceSettings != nil && doc.Customer.InvoiceSettings.DefaultPaymentMethod != nil {
		for _, def := range paymentMethodDefinitions {
			if string(doc.Customer.InvoiceSettings.DefaultPaymentMethod.Type) == def.Key {
				return &pay.Instructions{
					Key:    def.MeansKey,
					Detail: def.Description,
				}
			}
		}
	}

	if doc.PaymentIntent == nil {
		return nil
	}

	if doc.PaymentIntent.PaymentMethodTypes == nil {
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
					if !instructions.Key.Has(def.MeansKey) {
						instructions.Key = instructions.Key.With(def.MeansKey)
					}
					instructions.Detail += ", " + def.Description
				}
			}
		}
	}

	return instructions
}

// newPaymentAdvances creates a payment advances object from a Stripe invoice.
func newPaymentAdvances(doc *stripe.Invoice, regimeDef *tax.RegimeDef) []*pay.Advance {
	if doc.AmountPaid == 0 {
		return nil
	}

	// For the moment we can create an advance object with the amount paid
	advance := &pay.Advance{
		Amount:      CurrencyAmount(doc.AmountPaid, FromCurrency(doc.Currency)),
		Description: "Advance payment",
	}

	if doc.Charge != nil {
		advance.Date = newDateFromTS(doc.Charge.Created, regimeDef.TimeLocation())
		if doc.Charge.Description != "" {
			advance.Description = doc.Charge.Description
		}

		if doc.Charge.PaymentMethodDetails != nil {
			for _, def := range paymentMethodDefinitions {
				if string(doc.Charge.PaymentMethodDetails.Type) == def.Key {
					advance.Key = def.MeansKey
					break
				}
			}
		}
	}

	return []*pay.Advance{advance}
}
