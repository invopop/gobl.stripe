package goblstripe

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/tax"
	"github.com/stripe/stripe-go/v81"
)

// Stripe tax types that the API documents but stripe-go has no constant for, see
// https://docs.stripe.com/api/tax_rates/object#tax_rate_object-tax_type.
const (
	taxRateTaxTypeAdmissionsTax         stripe.TaxRateTaxType = "admissions_tax"
	taxRateTaxTypeAttendanceTax         stripe.TaxRateTaxType = "attendance_tax"
	taxRateTaxTypeDigitalExciseTax      stripe.TaxRateTaxType = "digital_excise_tax"
	taxRateTaxTypeEntertainmentTax      stripe.TaxRateTaxType = "entertainment_tax"
	taxRateTaxTypeGrossReceiptsTax      stripe.TaxRateTaxType = "gross_receipts_tax"
	taxRateTaxTypeHospitalityTax        stripe.TaxRateTaxType = "hospitality_tax"
	taxRateTaxTypeLuxuryTax             stripe.TaxRateTaxType = "luxury_tax"
	taxRateTaxTypeMassTransitParkingTax stripe.TaxRateTaxType = "mass_transit_parking_tax"
	taxRateTaxTypeParkingTax            stripe.TaxRateTaxType = "parking_tax"
	taxRateTaxTypeResortTax             stripe.TaxRateTaxType = "resort_tax"
	taxRateTaxTypeTourismTax            stripe.TaxRateTaxType = "tourism_tax"
	taxRateTaxTypeUtilityUsersTax       stripe.TaxRateTaxType = "utility_users_tax"
)

// unsupportedTaxes maps every Stripe tax type that GOBL has no tax category for
// to the name Stripe documents for it, which describes the charge when the tax
// rate carries no display name. Taxes of these types are imported as line
// charges so that the invoice total still reconciles with Stripe.
var unsupportedTaxes = map[stripe.TaxRateTaxType]string{
	taxRateTaxTypeAdmissionsTax:            "Admissions Tax",
	stripe.TaxRateTaxTypeAmusementTax:      "Amusement Tax",
	taxRateTaxTypeAttendanceTax:            "Attendance Tax",
	stripe.TaxRateTaxTypeCommunicationsTax: "Communications Tax",
	taxRateTaxTypeDigitalExciseTax:         "Digital Excise Tax",
	taxRateTaxTypeEntertainmentTax:         "Entertainment Tax",
	taxRateTaxTypeGrossReceiptsTax:         "Gross Receipts Tax",
	taxRateTaxTypeHospitalityTax:           "Hospitality Tax",
	stripe.TaxRateTaxTypeJCT:               "Japanese Consumption Tax",
	stripe.TaxRateTaxTypeLeaseTax:          "Chicago Lease Tax",
	taxRateTaxTypeLuxuryTax:                "Luxury Tax",
	taxRateTaxTypeMassTransitParkingTax:    "Mass Transit Parking Tax",
	taxRateTaxTypeParkingTax:               "Parking Tax",
	stripe.TaxRateTaxTypeQST:               "Quebec Sales Tax",
	taxRateTaxTypeResortTax:                "Resort Tax",
	stripe.TaxRateTaxTypeRetailDeliveryFee: "Retail Delivery Fee",
	stripe.TaxRateTaxTypeRST:               "Retail Sales Tax",
	stripe.TaxRateTaxTypeServiceTax:        "Service Tax",
	taxRateTaxTypeTourismTax:               "Tourism Tax",
	taxRateTaxTypeUtilityUsersTax:          "Utility Users Tax",
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
	if name, ok := unsupportedTaxes[taxRate.TaxType]; ok {
		return name
	}
	return "Tax"
}
