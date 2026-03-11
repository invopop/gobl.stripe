package goblstripe

import (
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
	"github.com/stripe/stripe-go/v81"
)

// taxFromInvoiceTaxAmounts creates a tax object from the tax amounts in an invoice.
// When a tax category can't be determined from the root-level tax rate,
// it falls back to line-level tax amounts to find a valid category.
func taxFromInvoiceTaxAmounts(taxAmounts []*stripe.InvoiceTotalTaxAmount, lines []*stripe.InvoiceLineItem) *bill.Tax {
	if len(taxAmounts) == 0 {
		return nil
	}

	// We just check the first tax
	if !taxAmounts[0].Inclusive {
		return nil
	}

	cat := extractTaxCat(taxAmounts[0].TaxRate)
	if cat == "" {
		cat = taxCatFromInvoiceLines(lines)
	}
	if cat == "" {
		return nil
	}

	return &bill.Tax{PricesInclude: cat}
}

// taxFromCreditNoteTaxAmounts creates a tax object from the tax amounts in a credit note.
// When a tax category can't be determined from the root-level tax rate,
// it falls back to line-level tax amounts to find a valid category.
func taxFromCreditNoteTaxAmounts(taxAmounts []*stripe.CreditNoteTaxAmount, lines []*stripe.CreditNoteLineItem) *bill.Tax {
	if len(taxAmounts) == 0 {
		return nil
	}

	// We just check the first tax
	if !taxAmounts[0].Inclusive {
		return nil
	}

	cat := extractTaxCat(taxAmounts[0].TaxRate)
	if cat == "" {
		cat = taxCatFromCreditNoteLines(lines)
	}
	if cat == "" {
		return nil
	}

	return &bill.Tax{PricesInclude: cat}
}

// taxCatFromInvoiceLines iterates over invoice line items to find a valid tax category.
func taxCatFromInvoiceLines(lines []*stripe.InvoiceLineItem) cbc.Code {
	for _, line := range lines {
		for _, ta := range line.TaxAmounts {
			if cat := extractTaxCat(ta.TaxRate); cat != "" {
				return cat
			}
		}
	}
	return ""
}

// taxCatFromCreditNoteLines iterates over credit note line items to find a valid tax category.
func taxCatFromCreditNoteLines(lines []*stripe.CreditNoteLineItem) cbc.Code {
	for _, line := range lines {
		for _, ta := range line.TaxAmounts {
			if cat := extractTaxCat(ta.TaxRate); cat != "" {
				return cat
			}
		}
	}
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
