package goblstripe

import (
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
	"github.com/stripe/stripe-go/v81"
)

// taxFromInvoiceTaxAmounts creates a tax object from the tax amounts in an invoice.
func taxFromInvoiceTaxAmounts(taxAmounts []*stripe.InvoiceTotalTaxAmount) *bill.Tax {
	var t *bill.Tax

	if len(taxAmounts) == 0 {
		return nil
	}

	if taxAmounts[0].TaxRate.TaxType == "" && taxAmounts[0].TaxRate.DisplayName == "" {
		return nil
	}

	// We just check the first tax
	if taxAmounts[0].Inclusive {
		t = new(bill.Tax)
		t.PricesInclude = extractTaxCat(taxAmounts[0].TaxRate)
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

	if taxAmounts[0].TaxRate.TaxType == "" && taxAmounts[0].TaxRate.DisplayName == "" {
		return nil
	}

	// We just check the first tax
	if taxAmounts[0].Inclusive {
		t = new(bill.Tax)
		t.PricesInclude = extractTaxCat(taxAmounts[0].TaxRate)
		return t
	}

	return nil
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
