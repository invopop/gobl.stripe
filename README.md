# GOBL <-> Stripe API Convertor

Convert GOBL documents to and from the Stripe API format.

## Naming

All method or definition names in the `goblstripe` package should primarily reference the Stripe API objects that will be generated or parsed, suffixed with the verb representing the intent, e.g.:

- `ToCustomer` implies that a GOBL object, defined in the method, will be converted into a stripe customer.
- `FromCustomer` expects a stripe object to be converted into GOBL.
- `ToInvoice` converts a GOBL Invoice into the Stripe equivalent.
- `FromInvoice` expects a stripe API object to convert into GOBL.
- `ToTaxID` converts a GOBL `tax.Identity` or `org.Identity` into the expected Stripe customer Tax ID object.

## Steps to include

When converting from Stripe to GOBL, the Stripe invoice does not include any exchange rates. We add some default ones. To have updated ones you should include the exchange rates step in the workflow.

## Notes
- Stripe invoices have suppliers/issuers, that normally are the account. In the invoice you get the country and name of the account, but for getting the account_tax_ids, you must do an extra call. Then, the account tax ids can be from any country. I assume that normally they will be from the same country as the account, but what if not? Should we use the account country as the regime or the account tax id country as the regime?.
- Related to the previous point, there is also an extra issue where a supplier could have multiple tax ids. Which one should we pick? I am picking the first one by default. What if we don't have a supplier tax id in the invoice (can happen)? Then we just should select one from Invopop.
- We have to add more cases for the tax ids
- For the moment, we don't consider proration 
- We don't support custom unit amount
- We are assuming everything is expanded
- In Stripe they do tax inclusive/exclusive product wise, while in Invopop we do it invoice wise. We assume they all have the same tag

## Things not included in first version
- There is a field in the Stripe invoice called Issuer, that states if the issuer is the own account or another account. It can only be another account if using Stripe Connects. For the moment, we assume that the issuer is the same as the account.

For tags and more we can use the `metadata` or `custom_fields` field