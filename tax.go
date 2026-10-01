package goblstripe

import (
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/regimes/in"
	"github.com/invopop/gobl/tax"
	"github.com/stripe/stripe-go/v81"
)

// taxCategories maps the Stripe tax types that have an equivalent GOBL tax
// category. Types missing from this map have no category and are recorded as
// document charges, see unsupportedTaxes.
var taxCategories = map[stripe.TaxRateTaxType]cbc.Code{
	stripe.TaxRateTaxTypeVAT:      tax.CategoryVAT,
	stripe.TaxRateTaxTypeJCT:      tax.CategoryVAT,
	stripe.TaxRateTaxTypeGST:      tax.CategoryGST,
	stripe.TaxRateTaxTypeHST:      tax.CategoryGST,
	stripe.TaxRateTaxTypeIGST:     in.TaxCategoryIGST,
	stripe.TaxRateTaxTypeSalesTax: tax.CategoryST,
}

// displayNameTaxCategories maps the tax rate display names we recognise, used
// when Stripe reports no tax type at all.
var displayNameTaxCategories = map[string]cbc.Code{
	"vat":       tax.CategoryVAT,
	"iva":       tax.CategoryVAT,
	"sales tax": tax.CategoryST,
	"gst":       tax.CategoryGST,
}

// taxFromInvoiceTaxAmounts creates a tax object from the tax amounts in an invoice.
// When a tax category can't be determined from the root-level tax rate,
// it falls back to line-level tax amounts to find a valid category.
func taxFromInvoiceTaxAmounts(taxAmounts []*stripe.InvoiceTotalTaxAmount, lines []*stripe.InvoiceLineItem, regimeDef *tax.RegimeDef) *bill.Tax {
	if len(taxAmounts) == 0 {
		return nil
	}

	// We just check the first tax
	if !taxAmounts[0].Inclusive {
		return nil
	}

	cat := extractTaxCat(taxAmounts[0].TaxRate, regimeDef)
	if cat == "" {
		cat = taxCatFromInvoiceLines(lines, regimeDef)
	}
	if cat == "" {
		return nil
	}

	return &bill.Tax{PricesInclude: cat}
}

// taxFromCreditNoteTaxAmounts creates a tax object from the tax amounts in a credit note.
// When a tax category can't be determined from the root-level tax rate,
// it falls back to line-level tax amounts to find a valid category.
func taxFromCreditNoteTaxAmounts(taxAmounts []*stripe.CreditNoteTaxAmount, lines []*stripe.CreditNoteLineItem, regimeDef *tax.RegimeDef) *bill.Tax {
	if len(taxAmounts) == 0 {
		return nil
	}

	// We just check the first tax
	if !taxAmounts[0].Inclusive {
		return nil
	}

	cat := extractTaxCat(taxAmounts[0].TaxRate, regimeDef)
	if cat == "" {
		cat = taxCatFromCreditNoteLines(lines, regimeDef)
	}
	if cat == "" {
		return nil
	}

	return &bill.Tax{PricesInclude: cat}
}

// taxCatFromInvoiceLines iterates over invoice line items to find a valid tax category.
func taxCatFromInvoiceLines(lines []*stripe.InvoiceLineItem, regimeDef *tax.RegimeDef) cbc.Code {
	for _, line := range lines {
		for _, ta := range line.TaxAmounts {
			if cat := extractTaxCat(ta.TaxRate, regimeDef); cat != "" {
				return cat
			}
		}
	}
	return ""
}

// taxCatFromCreditNoteLines iterates over credit note line items to find a valid tax category.
func taxCatFromCreditNoteLines(lines []*stripe.CreditNoteLineItem, regimeDef *tax.RegimeDef) cbc.Code {
	for _, line := range lines {
		for _, ta := range line.TaxAmounts {
			if cat := extractTaxCat(ta.TaxRate, regimeDef); cat != "" {
				return cat
			}
		}
	}
	return ""
}

// extractTaxCat extracts the tax category from a Stripe tax rate, using the
// display name when the tax type is not set. It returns an empty code when the
// tax has no category in the regime that will validate it, in which case the
// caller records the tax as a document charge instead.
func extractTaxCat(taxRate *stripe.TaxRate, regimeDef *tax.RegimeDef) cbc.Code {
	if taxRate == nil {
		return ""
	}

	cat, ok := taxCategories[taxRate.TaxType]
	if !ok {
		cat = displayNameTaxCategories[strings.ToLower(strings.TrimSpace(taxRate.DisplayName))]
	}
	if cat == "" {
		return ""
	}

	if taxRegimeDef(taxRate, regimeDef).CategoryDef(cat) == nil {
		return ""
	}

	return cat
}

// taxRegimeDef resolves the regime that will validate a tax combo built from the
// given rate, which is the rate's own country whenever Stripe provides one.
func taxRegimeDef(taxRate *stripe.TaxRate, regimeDef *tax.RegimeDef) *tax.RegimeDef {
	if taxRate.Country != "" {
		return tax.RegimeDefFor(l10n.Code(taxRate.Country))
	}
	return regimeDef
}
