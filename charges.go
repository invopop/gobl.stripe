package goblstripe

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/tax"
	"github.com/stripe/stripe-go/v81"
)

// unsupportedTax describes a Stripe tax type that GOBL has no tax category for.
// Taxes of these types are imported as line charges so that the invoice total
// still reconciles with Stripe.
type unsupportedTax struct {
	// TaxType is the value Stripe reports in the tax rate's tax_type field.
	TaxType stripe.TaxRateTaxType
	// Name describes the tax, and is used as the charge reason when the Stripe
	// tax rate carries no display name.
	Name string
}

// unsupportedTaxes lists the Stripe tax types that stripe-go defines and GOBL has
// no tax category for. Stripe reports further types that are charged just the
// same, using the display name on the tax rate to describe them.
var unsupportedTaxes = []unsupportedTax{
	{TaxType: stripe.TaxRateTaxTypeAmusementTax, Name: "Amusement Tax"},
	{TaxType: stripe.TaxRateTaxTypeCommunicationsTax, Name: "Communications Tax"},
	{TaxType: stripe.TaxRateTaxTypeLeaseTax, Name: "Chicago Lease Tax"},
	{TaxType: stripe.TaxRateTaxTypePST, Name: "Provincial Sales Tax"},
	{TaxType: stripe.TaxRateTaxTypeQST, Name: "Quebec Sales Tax"},
	{TaxType: stripe.TaxRateTaxTypeRetailDeliveryFee, Name: "Retail Delivery Fee"},
	{TaxType: stripe.TaxRateTaxTypeRST, Name: "Retail Sales Tax"},
	{TaxType: stripe.TaxRateTaxTypeServiceTax, Name: "Service Tax"},
}

// unsupportedTaxFor returns the definition of a Stripe tax type that cannot be
// expressed as a GOBL tax category, or nil when the type is not listed.
func unsupportedTaxFor(taxType stripe.TaxRateTaxType) *unsupportedTax {
	for _, ut := range unsupportedTaxes {
		if ut.TaxType == taxType {
			return &ut
		}
	}
	return nil
}

// lineChargesFromInvoiceTaxAmounts converts the taxes on an invoice line that
// GOBL cannot express as tax combos into line charges.
func lineChargesFromInvoiceTaxAmounts(taxAmounts []*stripe.InvoiceTotalTaxAmount, curr currency.Code, regimeDef *tax.RegimeDef) []*bill.LineCharge {
	var charges []*bill.LineCharge
	for _, ta := range taxAmounts {
		charge := newLineChargeFromTax(ta.TaxRate, ta.Amount, ta.Inclusive, curr, regimeDef)
		if charge != nil {
			charges = append(charges, charge)
		}
	}
	return charges
}

// lineChargesFromCreditNoteTaxAmounts converts the taxes on a credit note line
// that GOBL cannot express as tax combos into line charges.
func lineChargesFromCreditNoteTaxAmounts(taxAmounts []*stripe.CreditNoteTaxAmount, curr currency.Code, regimeDef *tax.RegimeDef) []*bill.LineCharge {
	var charges []*bill.LineCharge
	for _, ta := range taxAmounts {
		charge := newLineChargeFromTax(ta.TaxRate, ta.Amount, ta.Inclusive, curr, regimeDef)
		if charge != nil {
			charges = append(charges, charge)
		}
	}
	return charges
}

// newLineChargeFromTax builds a line charge for a Stripe tax with no GOBL tax
// category. It returns nil when the tax maps to a category, when the amount is
// already part of the line price, or when there is nothing to charge.
func newLineChargeFromTax(taxRate *stripe.TaxRate, amount int64, inclusive bool, curr currency.Code, regimeDef *tax.RegimeDef) *bill.LineCharge {
	if taxRate == nil || amount == 0 || inclusive {
		return nil
	}

	if taxRate.TaxType == "" && taxRate.DisplayName == "" {
		// The rate says nothing about the tax it represents, typically because it
		// was not expanded in the request, so there is nothing to charge for.
		return nil
	}

	if extractTaxCat(taxRate, regimeDef) != "" {
		return nil
	}

	return &bill.LineCharge{
		Key:    bill.ChargeKeyTax,
		Code:   cbc.Code(taxRate.TaxType),
		Reason: chargeReasonFromTax(taxRate),
		Amount: CurrencyAmount(amount, curr),
	}
}

// chargeReasonFromTax describes the tax being charged, preferring the name Stripe
// shows to the customer over the one defined for the tax type.
func chargeReasonFromTax(taxRate *stripe.TaxRate) string {
	if taxRate.DisplayName != "" {
		return taxRate.DisplayName
	}
	if ut := unsupportedTaxFor(taxRate.TaxType); ut != nil {
		return ut.Name
	}
	return "Tax"
}
