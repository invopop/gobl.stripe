# GOBL <-> Stripe API Convertor

Convert GOBL documents to and from the Stripe API format.

## Naming

All method or definition names in the `goblstripe` package should primarily reference the Stripe API objects that will be generated or parsed, suffixed with the verb representing the intent, e.g.:

- `ToCustomer` implies that a GOBL object, defined in the method, will be converted into a stripe customer.
- `FromCustomer` expects a stripe object to be converted into GOBL.
- `ToInvoice` converts a GOBL Invoice into the Stripe equivalent.
- `FromInvoice` expects a stripe API object to convert into GOBL.
- `ToTaxID` converts a GOBL `tax.Identity` or `org.Identity` into the expected Stripe customer Tax ID object.

In some cases, a new GOBL object is created that does not correspond to any specific or similar object in Stripe. For such cases, the method names include the prefix `new`, e.g.:
- `newOrdering` creates a new `Ordering` object not from a similar object but from the whole `Invoice`.

## Expanded fields
To get all the required information, they request to the invoice must be done expanding the following fields.

### For Invoices
- account_tax_ids
- customer.tax_ids
- lines.data.discounts
- lines.data.tax_amounts.tax_rate
- total_amounts.tax_rate
- payment_intent

### For Credit Notes
- invoice.account_tax_ids
- customer.tax_ids
- lines.data.tax_amounts.tax_rate

## Assumptions/Things to consider for future versions

### Supplier
For the invoice supplier we are currently assuming the following:
- The `supplier` is always the Stripe `account`. This is false if using Stripe connect. The field `issuer` informs you if the issuer is the own account (`self`) or another account (`account`).
    - If the `issuer` is not `self` we don't have access to the tax ids as they are not displayed on the `account` object.
- The `supplier` has exactly 1 tax id. In Stripe, the account can have several tax ids or none. If it has several, we assume that the first one is the correct one. If it has none, we would need to get an Invopop supplier like in Chargebee.

A solution for these problems could be directly getting the supplier from Invopop like Chargebee does.

### Tax ids
- We must add more values for the tax ids mapping from Stripe to GOBL.

### Tax included/excluded
Tax is included/excluded is treated differently in GOBL and Stripe. In GOBL, the included flag is for all the taxes in the invoice, while in Stripe it is considered that the invoice can have several taxes. Our assumption is that there is going to be only 1 tax type (VAT, SGT, ...) per invoice.

### Discounts
For the moment, we consider there are no discounts on the general invoice, but only on the line items. 

### Payment
- For the moment, we are not including the payment instructions for already paid invoices. We could add it by expanding the `payment_method` field in `charge`.
- For the advances there is no a straightforward way to get them as another API request is required. Currently we are handling it as a unique advancement on the `amount_paid`.

### Not supported yet
- Proration
- Charges (*bill.Charge). If needed for delivering goods we could add the shipping charges. 

For tags and more we can use the `metadata` or `custom_fields` field

We still need to define what to do with the naming of fields and tags and where to include them. My hypothesis is that we can create the app in Stripe and add the fields there.

### Handling extensions
For the moment GOBL fields `$addons` and `$tags` are not being mapped as we need to have the complete integration with Stripe to understand how can we pass these fields. Some ideas are to include them on the `metadata` field that several Stripe fields have. We could use the `Invoice templates` to define some common `custom fields` per invoice.

## Useful Notes
- `livemode` field states wether the generated invoice is in testing or live. `True` means it is live and `False` testing. Currently not being used.
- - For tax there is a field that is default_tax_rates, but it is normally empty as not specified by the user. To check the rates we need to check the total_tax_amounts. 
- For the `regime`, the `account_country` is always used.
- We assume the attribute `has_more` in lines is false. This attribute is used to state if there are more line pages in the invoice that we can fetch.
- Amount is always charged in the smallest possible unit (cents in euros, yens in yens, ...)


## Steps to include in Workflows

When converting from Stripe to GOBL, the Stripe invoice does not include any exchange rates. We add some default ones. To have updated ones you should include the exchange rates step in the workflow.