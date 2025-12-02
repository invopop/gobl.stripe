package goblstripe

import (
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
	"github.com/stripe/stripe-go/v84"
)

// taxFromInvoiceTotalTaxes creates a tax object from the total taxes in an invoice.
func taxFromInvoiceTotalTaxes(totalTaxes []*stripe.InvoiceTotalTax) *bill.Tax {
	var t *bill.Tax

	if len(totalTaxes) == 0 {
		return nil
	}

	// We just check the first tax
	if totalTaxes[0].TaxBehavior == stripe.InvoiceTotalTaxTaxBehaviorInclusive {
		t = new(bill.Tax)
		t.PricesInclude = extractTaxCatFromInvoiceTotalTax(totalTaxes[0])
		return t
	}

	return nil
}

// taxFromCreditNoteTotalTaxes creates a tax object from the total taxes in a credit note.
func taxFromCreditNoteTotalTaxes(totalTaxes []*stripe.CreditNoteTotalTax) *bill.Tax {
	var t *bill.Tax

	if len(totalTaxes) == 0 {
		return nil
	}

	// We just check the first tax
	if totalTaxes[0].TaxBehavior == stripe.CreditNoteTotalTaxTaxBehaviorInclusive {
		t = new(bill.Tax)
		t.PricesInclude = extractTaxCatFromCreditNoteTotalTax(totalTaxes[0])
		return t
	}

	return nil
}

// extractTaxCatFromInvoiceTotalTax extracts the tax category from an InvoiceTotalTax.
// Since v84 doesn't have direct access to TaxRate details in the TotalTax,
// we need to look at the TaxRateDetails if available.
func extractTaxCatFromInvoiceTotalTax(totalTax *stripe.InvoiceTotalTax) cbc.Code {
	if totalTax == nil {
		return ""
	}

	// In v84, we don't have direct access to TaxRate type or display name in TotalTax
	// We'll return an empty code for now, as the tax category should be determined from line items
	return ""
}

// extractTaxCatFromCreditNoteTotalTax extracts the tax category from a CreditNoteTotalTax.
func extractTaxCatFromCreditNoteTotalTax(totalTax *stripe.CreditNoteTotalTax) cbc.Code {
	if totalTax == nil {
		return ""
	}

	// In v84, we don't have direct access to TaxRate type or display name in TotalTax
	// We'll return an empty code for now, as the tax category should be determined from line items
	return ""
}

// extractTaxCat extracts the tax category from a Stripe tax rate.
// If the tax type is not set, we use the display name to determine the tax category.
func extractTaxCat(taxRate *stripe.TaxRate) cbc.Code {
	if taxRate == nil {
		return ""
	}
	switch taxRate.TaxType {
	case stripe.TaxRateTaxTypeVAT:
		return tax.CategoryVAT
	case stripe.TaxRateTaxTypeSalesTax:
		return tax.CategoryST
	case stripe.TaxRateTaxTypeGST:
		return tax.CategoryGST
	}

	switch strings.ToLower(strings.TrimSpace(taxRate.DisplayName)) {
	case "vat", "iva":
		return tax.CategoryVAT
	case "sales tax":
		return tax.CategoryST
	case "gst":
		return tax.CategoryGST
	}

	return cbc.Code(taxRate.DisplayName)
}
