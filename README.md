# GOBL <-> Stripe API Convertor

Convert GOBL documents to and from the Stripe API format.

## Naming

All method or definition names in the `goblstripe` package should primarily reference the Stripe API objects that will be generated or parsed, suffixed with the verb representing the intent, e.g.:

- `ToCustomer` implies that a GOBL object, defined in the method, will be converted into a stripe customer.
- `FromCustomer` expects a stripe object to be converted into GOBL.
- `ToInvoice` converts a GOBL Invoice into the Stripe equivalent.
- `FromInvoice` expects a stripe API object to convert into GOBL.
- `ToTaxID` converts a GOBL `tax.Identity` or `org.Identity` into the expected Stripe customer Tax ID object.
