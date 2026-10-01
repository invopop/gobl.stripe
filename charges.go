package goblstripe

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/tax"
	"github.com/stripe/stripe-go/v81"
)

// unsupportedTax describes a Stripe tax type that GOBL has no tax category for.
// Taxes of these types are imported as document charges so that the invoice
// total still reconciles with Stripe.
type unsupportedTax struct {
	// TaxType is the value Stripe reports in the tax rate's tax_type field.
	TaxType stripe.TaxRateTaxType
	// Name describes the tax, and is used as the charge reason when the Stripe
	// tax rate carries no display name.
	Name string
}

// unsupportedTaxes lists every Stripe tax type without a GOBL tax category.
// The types missing a constant in stripe-go are declared as literals, as Stripe
// returns them regardless of whether the SDK knows about them.
var unsupportedTaxes = []unsupportedTax{
	{TaxType: "admissions_tax", Name: "Admissions Tax"},
	{TaxType: stripe.TaxRateTaxTypeAmusementTax, Name: "Amusement Tax"},
	{TaxType: "attendance_tax", Name: "Attendance Tax"},
	{TaxType: stripe.TaxRateTaxTypeCommunicationsTax, Name: "Communications Tax"},
	{TaxType: "entertainment_tax", Name: "Entertainment Tax"},
	{TaxType: "gross_receipts_tax", Name: "Gross Receipts Tax"},
	{TaxType: "hospitality_tax", Name: "Hospitality Tax"},
	{TaxType: stripe.TaxRateTaxTypeLeaseTax, Name: "Chicago Lease Tax"},
	{TaxType: "luxury_tax", Name: "Luxury Tax"},
	{TaxType: "mass_transit_parking_tax", Name: "Mass Transit Parking Tax"},
	{TaxType: "parking_tax", Name: "Parking Tax"},
	{TaxType: stripe.TaxRateTaxTypePST, Name: "Provincial Sales Tax"},
	{TaxType: stripe.TaxRateTaxTypeQST, Name: "Quebec Sales Tax"},
	{TaxType: "resort_tax", Name: "Resort Tax"},
	{TaxType: stripe.TaxRateTaxTypeRetailDeliveryFee, Name: "Retail Delivery Fee"},
	{TaxType: stripe.TaxRateTaxTypeRST, Name: "Retail Sales Tax"},
	{TaxType: stripe.TaxRateTaxTypeServiceTax, Name: "Service Tax"},
	{TaxType: "tourism_tax", Name: "Tourism Tax"},
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

// chargesFromInvoiceTaxAmounts converts the Stripe taxes that GOBL cannot express
// as tax combos into document charges.
func chargesFromInvoiceTaxAmounts(taxAmounts []*stripe.InvoiceTotalTaxAmount, curr currency.Code, regimeDef *tax.RegimeDef) []*bill.Charge {
	var charges []*bill.Charge
	for _, ta := range taxAmounts {
		charge := newChargeFromTax(ta.TaxRate, ta.Amount, ta.Inclusive, curr, regimeDef)
		if charge != nil {
			charges = append(charges, charge)
		}
	}
	return charges
}

// chargesFromCreditNoteTaxAmounts converts the Stripe taxes that GOBL cannot
// express as tax combos into document charges.
func chargesFromCreditNoteTaxAmounts(taxAmounts []*stripe.CreditNoteTaxAmount, curr currency.Code, regimeDef *tax.RegimeDef) []*bill.Charge {
	var charges []*bill.Charge
	for _, ta := range taxAmounts {
		charge := newChargeFromTax(ta.TaxRate, ta.Amount, ta.Inclusive, curr, regimeDef)
		if charge != nil {
			charges = append(charges, charge)
		}
	}
	return charges
}

// newChargeFromTax builds a document charge for a Stripe tax with no GOBL tax
// category. It returns nil when the tax maps to a category, when the amount is
// already part of the line prices, or when there is nothing to charge.
func newChargeFromTax(taxRate *stripe.TaxRate, amount int64, inclusive bool, curr currency.Code, regimeDef *tax.RegimeDef) *bill.Charge {
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

	return &bill.Charge{
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
