package goblstripe

import (
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
		// Default is used for unexpanded invoices to prevent invalids
		return tax.CategoryVAT
	}
}
