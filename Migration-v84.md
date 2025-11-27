# Migration Guide: Stripe API v81 → v84

This document outlines the migration from `stripe-go/v81` to `stripe-go/v84` and the breaking changes introduced by Stripe's Basil API version (2025-03-31.basil).

## Overview

The v84 update includes significant breaking changes to tax modeling on Invoices, Invoice Line Items, and Credit Notes. The Stripe API has moved from simple tax amounts to a more structured tax configuration system.

## Version Information

- **Previous Version**: `github.com/stripe/stripe-go/v81` v81.4.0
- **New Version**: `github.com/stripe/stripe-go/v84` v84.0.0
- **API Version**: 2025-03-31.basil

## Breaking Changes Summary

### 1. Invoice Tax Fields

**Removed:**
- `Invoice.TotalTaxAmounts []*InvoiceTotalTaxAmount`
- `Invoice.Paid bool`
- `Invoice.PaymentIntent *PaymentIntent`
- `Invoice.Charge *Charge`

**Added:**
- `Invoice.TotalTaxes []*InvoiceTotalTax`
- `Invoice.Status InvoiceStatus` (use `InvoiceStatusPaid` instead of `Paid`)
- `Invoice.Payments *InvoicePaymentList` (replaces direct PaymentIntent access)

### 2. Credit Note Tax Fields

**Removed:**
- `CreditNote.TaxAmounts []*CreditNoteTaxAmount`

**Added:**
- `CreditNote.TotalTaxes []*CreditNoteTotalTax`

### 3. Invoice Line Item Changes

**Removed:**
- `InvoiceLineItem.Price *Price`
- `InvoiceLineItem.Plan *Plan`
- `InvoiceLineItem.TaxAmounts []*InvoiceTotalTaxAmount`
- `InvoiceLineItem.AmountExcludingTax`
- `InvoiceLineItem.UnitAmountExcludingTax`

**Added:**
- `InvoiceLineItem.Pricing *InvoiceLineItemPricing`
- `InvoiceLineItem.Parent *InvoiceLineItemParent`
- `InvoiceLineItem.Taxes []*InvoiceLineItemTax`

### 4. Credit Note Line Item Changes

**Removed:**
- `CreditNoteLineItem.TaxAmounts []*CreditNoteTaxAmount`

**Added:**
- `CreditNoteLineItem.Taxes []*CreditNoteLineItemTax`

### 5. Tax Structure Changes

The new tax structures no longer contain full `TaxRate` objects. Instead, they use:
- `TaxRateDetails` with just a tax rate ID reference
- `TaxBehavior` enum instead of `Inclusive bool`
- `TaxabilityReason` enum values have been renamed

**Old constants:**
```go
stripe.InvoiceTotalTaxAmountTaxabilityReasonReverseCharge
stripe.CreditNoteTaxAmountTaxabilityReasonReverseCharge
```

**New constants:**
```go
stripe.InvoiceTotalTaxTaxabilityReasonReverseCharge
stripe.CreditNoteTotalTaxTaxabilityReasonReverseCharge
stripe.InvoiceLineItemTaxTaxabilityReasonReverseCharge
stripe.CreditNoteLineItemTaxTaxabilityReasonReverseCharge
```

### 6. Discount Structure Changes

**Old:**
```go
discount.Coupon.Name
```

**New:**
```go
discount.Source.Coupon.Name
```

### 7. Payment Access Changes

**Old:**
```go
if invoice.Paid {
    // ...
}
if invoice.PaymentIntent != nil {
    methods := invoice.PaymentIntent.PaymentMethodTypes
}
if invoice.Charge != nil {
    created := invoice.Charge.Created
}
```

**New:**
```go
if invoice.Status == stripe.InvoiceStatusPaid {
    // ...
}
if invoice.Payments != nil && len(invoice.Payments.Data) > 0 {
    payment := invoice.Payments.Data[0]
    if payment.Payment.PaymentIntent != nil {
        methods := payment.Payment.PaymentIntent.PaymentMethodTypes
    }
    if payment.Payment.Charge != nil {
        created := payment.Payment.Charge.Created
    }
}
```

## Important Considerations

### 1. Tax Rate Information

In v84, line item taxes no longer contain full `TaxRate` objects with country, percentage, and type information. This affects how you extract tax categories. You may need to:

- Expand tax rate details in API calls using `expand[]=lines.data.taxes.tax_rate_details`
- Store tax information at the line level rather than deriving it from tax amounts
- Calculate tax percentages from `Amount` and `TaxableAmount` fields

### 2. Billing Scheme Detection

`InvoiceLineItem.Price.BillingScheme` is no longer available. The quantity logic now:
- Uses `line.Quantity` directly
- Defaults to 1 if quantity is 0 (typical for tiered pricing)

### 3. Product Metadata

Product metadata is no longer directly accessible via `line.Price.Product.Metadata`. You need to:
- Expand the price/product in API calls
- Use line-level metadata instead (`line.Metadata`)

### 4. Tax Behavior vs Inclusive

The boolean `Inclusive` field has been replaced with a `TaxBehavior` enum:

```go
// Old
if taxAmount.Inclusive {
    // ...
}

// New
if totalTax.TaxBehavior == stripe.InvoiceTotalTaxTaxBehaviorInclusive {
    // ...
}
```

### 5. Payment Intent Expansion

If you need payment intent details, you must expand the payments field:

```go
params := &stripe.InvoiceParams{}
params.AddExpand("payments.data.payment.payment_intent")
```

## Migration Steps

1. **Update go.mod**
   ```bash
   go get github.com/stripe/stripe-go/v84@v84.0.0
   go mod tidy
   ```

2. **Update imports**
   Replace all `github.com/stripe/stripe-go/v81` with `github.com/stripe/stripe-go/v84`

3. **Update tax field access**
   - Replace `TotalTaxAmounts` with `TotalTaxes`
   - Replace `TaxAmounts` with `Taxes`
   - Update type references to the new tax types

4. **Update payment status checks**
   - Replace `invoice.Paid` with `invoice.Status == stripe.InvoiceStatusPaid`
   - Update payment intent access to use `invoice.Payments`

5. **Update discount access**
   - Change `discount.Coupon` to `discount.Source.Coupon`

6. **Update test fixtures**
   - Remove deprecated fields
   - Update struct literals to use new field names
   - Update type references

7. **Review API expansion requirements**
   - Add necessary `expand[]` parameters to get full object details
   - Update params for invoice and credit note retrieval

## Code Examples

### Invoice Conversion

**Before:**
```go
inv.Tax = taxFromInvoiceTaxAmounts(doc.TotalTaxAmounts)
invLine.Taxes = FromInvoiceTaxAmountsToTaxSet(line.TaxAmounts, regimeDef)
```

**After:**
```go
inv.Tax = taxFromInvoiceTotalTaxes(doc.TotalTaxes)
invLine.Taxes = FromInvoiceLineItemTaxesToTaxSet(line.Taxes, regimeDef)
```

### Reverse Charge Detection

**Before:**
```go
for _, taxAmount := range doc.TotalTaxAmounts {
    if taxAmount.TaxabilityReason == stripe.InvoiceTotalTaxAmountTaxabilityReasonReverseCharge {
        return true
    }
}
```

**After:**
```go
for _, taxAmount := range doc.TotalTaxes {
    if taxAmount.TaxabilityReason == stripe.InvoiceTotalTaxTaxabilityReasonReverseCharge {
        return true
    }
}
```

### Line Item Quantity

**Before:**
```go
if line.Price == nil {
    return num.AmountZero
}
switch line.Price.BillingScheme {
case stripe.PriceBillingSchemePerUnit:
    return num.MakeAmount(line.Quantity, 0)
case stripe.PriceBillingSchemeTiered:
    return num.MakeAmount(1, 0)
}
```

**After:**
```go
if line.Quantity == 0 {
    return num.MakeAmount(1, 0)
}
return num.MakeAmount(line.Quantity, 0)
```

## Testing

### Test Data Updates

Update test fixtures to use new structures:

```go
// Old
&stripe.InvoiceLineItem{
    Price: &stripe.Price{
        BillingScheme: stripe.PriceBillingSchemePerUnit,
    },
    TaxAmounts: []*stripe.InvoiceTotalTaxAmount{},
}

// New
&stripe.InvoiceLineItem{
    Pricing: &stripe.InvoiceLineItemPricing{
        UnitAmountDecimal: 2000,
    },
    Taxes: []*stripe.InvoiceLineItemTax{},
}
```

### API Expansion in Tests

When testing with real Stripe data, update your expansion parameters:

```go
params := &stripe.InvoiceParams{}
params.AddExpand("payments.data.payment.payment_intent")
params.AddExpand("payments.data.payment.charge")
params.AddExpand("lines.data.taxes.tax_rate_details")
```

## Known Limitations

1. **Tax Category Detection**: Without full TaxRate objects, determining tax categories (VAT, GST, Sales Tax) from line items is more difficult. You may need to expand tax rates or use alternative approaches.

2. **Product Information**: Product names and metadata require additional API calls or expansions if not present in the line description.

3. **Historical Data**: Invoices created before the API version upgrade may have different data structures. Consider versioning your data processing logic.

## References

- [Stripe API Changelog: Invoice Tax Configurations](https://docs.stripe.com/changelog/basil/2025-03-31/invoice-tax-configurations)
- [Stripe Basil Changelog](https://docs.stripe.com/changelog/basil)
- [stripe-go v82 Migration Guide](https://github.com/stripe/stripe-go/wiki/Migration-guide-for-v82)
- [Stripe API Versioning](https://docs.stripe.com/api/versioning?lang=go)

## Support

For issues specific to this codebase, refer to the updated source files:
- `invoice.go` - Invoice and CreditNote conversion
- `lines.go` - Line item processing
- `tax.go` - Tax extraction and conversion
- `payment.go` - Payment handling

For Stripe API questions, consult the [official Stripe documentation](https://docs.stripe.com/).
