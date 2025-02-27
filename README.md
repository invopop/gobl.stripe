# GOBL <-> Stripe API Convertor

Convert GOBL documents to and from the Stripe API format.

Copyright [Invopop Ltd.](https://invopop.com) 2025. Released publicly under the [Apache License Version 2.0](LICENSE). For commercial licenses please contact the [dev team at invopop](mailto:dev@invopop.com). In order to accept contributions to this library we will require transferring copyrights to Invopop Ltd.

## Table of Contents

- [GOBL <-> Stripe API Convertor](#gobl-<->-stripe-api-convertor)
  - [Usage](#usage)
    - [Go Package](#go-package)
      - [Stripe -> GOBL conversion](#stripe-->-gobl-conversion)
    - [Command line](#command-line)
      - [Listen to Stripe Events + Stripe -> GOBL conversion](#listen-to-stripe-events-+-stripe-->-gobl-conversion)
  - [Naming](#naming)
  - [Expanded fields](#expanded-fields)
    - [For Invoices](#for-invoices)
    - [For Credit Notes](#for-credit-notes)
  - [Assumptions/Things to consider for future versions](#assumptions/things-to-consider-for-future-versions)
    - [Supplier](#supplier)
    - [Tax included/excluded](#tax-included/excluded)
    - [Discounts](#discounts)
    - [Payment](#payment)
    - [Not supported yet](#not-supported-yet)
  - [Handling tags/extensions](#handling-tags/extensions)
  - [Useful Notes](#useful-notes)
  - [Steps to include in Workflows](#steps-to-include-in-workflows)


## Usage

### Go Package

#### Stripe -> GOBL conversion

Invoice: 
```go
package main

import (
    "fmt"
    "os"
    "json"

    goblstripe "github.com/invopop/gobl.stripe"
    "github.com/invopop/gobl/uuid"
    "github.com/stripe/stripe-go/v81"
)

func main{
    data, _ := os.ReadFile("examples/stripe.gobl/stripe_basic_invoice.json")

    s := new(stripe.Invoice)
    if err := json.Unmarshal(data, s); err != nil {
		fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

    gi, err := goblstripe.FromInvoice(s)
    if err != nil {
        fmt.Errorf("error in conversion: %w", err)
    }

    // Now you can use the GOBL invoice (*bill.Invoice object) returned.
}

```

Credit Note: 
```go
package main

import (
    "fmt"
    "os"
    "json"

    goblstripe "github.com/invopop/gobl.stripe"
    "github.com/invopop/gobl/uuid"
    "github.com/stripe/stripe-go/v81"
)

func main{
    data, _ := os.ReadFile("examples/stripe.gobl/stripe_basic_invoice.json")

    s := new(stripe.CreditNote)
    if err := json.Unmarshal(data, s); err != nil {
		fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

    gi, err := goblstripe.FromCreditNote(s)
    if err != nil {
        fmt.Errorf("error in conversion: %w", err)
    }

    // Now you can use the GOBL invoice (*bill.Invoice object) returned.
}

```

### Command line
The GOBL <-> Stripe package also includes a command line helper. You can install it manually in your Go environment (from this main directory) with:

```bash
go install ./cmd/gobl.stripe
```

#### Listen to Stripe Events + Stripe -> GOBL conversion

With the `listen` command you will be able to:  
1. Listen to Stripe events (e.g., invoice finalized or credit note created).  
2. Save the invoice received in the event as a JSON file.  
3. (optional) Convert to GOBL and save as a JSON file.

To test this functionality locally, use the **Stripe CLI**, which you can install by following these [instructions](https://docs.stripe.com/stripe-cli).

---

1. **Set Up Stripe CLI to Forward Events**

Choose a port to forward events to (e.g., port `5276` in this example), and run the following command:  

```bash
stripe listen --forward-to localhost:5276/webhook
```

2. **Start the `listen` command**

Before running the `listen` command, you need to configure 2 secrets: Stripe secret API key and the webhook secret. This can be done in 2 ways:

- With environment variables: You would need to set the `STRIPE_SECRET_KEY` and `STRIPE_WEBHOOK_SECRET` on your environment. Then you run the command like: 

```bash
gobl.stripe listen -p 5276
```

- As arguments in the command:
```bash
gobl.stripe listen -p 5276 -k sk_test_afjsadf44332... -s whsec_jdferfwerif329...
```

3. **Trigger a Stripe Event**

To test, trigger a Stripe event. You can do these in several ways:

- *From the command line*

For example, to trigger an `invoice.finalized` event, run:

```bash
stripe trigger invoice.finalized
```

This will generate the most basic type of invoice. 

- *From the Stripe Dashboard*

You can manually create and finalize invoices from the [Stripe Dashboard](https://support.stripe.com/topics/dashboard).

- *From the Stripe API*

You can also create and finalize invoices using the [Stripe API](https://docs.stripe.com/api/invoices/object).

4. **Review the Output**

Once an event is triggered, the `listen` command generates 2 JSON files in the same directory:
- `stripe_{id}.json`: Contains the Stripe event data
- `gobl_{id}.json`: Contains the corresponding GOBL-converted data.

If you just want to get the stripe JSON file you can set the flag of convert to false:

```bash
gobl.stripe listen -p 5276 -c=false
```

#### Stripe to GOBL conversion only
If you already have a Stripe invoice in JSON format and want to convert it to GOBL, you can use the convert command:

```bash
gobl.stripe convert stripe_in_1QxASYQhcl5B85Ylo18UfypW.json
```


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
To get all the required information, the request to the invoice must be done expanding the following fields.

### For Invoices
- account_tax_ids
- lines.data.discounts
- lines.data.tax_amounts.tax_rate
- lines.data.price.product
- total_tax_amounts.tax_rate
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
- The `supplier` has exactly 1 tax id. In Stripe, the account can have several tax ids or none. If it has several, we assume that the first one is the correct one. If it has none, we would need to get an Invopop supplier.

A solution for these problems could be directly getting the supplier from Invopop.

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

## Handling tags/extensions
To handle tags and extensions different approaches are possible:
- If creating the invoice via API, some tags can be included in the `custom_fields` or `metadata`.
- If creating the invoice from the Stripe Dashboard, you can create a `template` with up to 4 custom fields.
- When we have the Invopop app in Stripe, we could use it to add some fields. 

## Useful Notes
- `livemode` field states wether the generated invoice is in testing or live. `True` means it is live and `False` testing. Currently not being used.
- For tax there is a field that is `default_tax_rates`, but it is normally empty as not specified by the user. To check the rates we need to check the `total_tax_amounts`. 
- For the `regime`, the `account_country` is always used.
- We assume the attribute `has_more` in lines is false. This attribute is used to state if there are more line pages in the invoice that we can fetch.
- Amount is always charged in the smallest possible unit (cents in euros, yens in yens, ...)
- The UUID generated is random, if you need a specific UUID, you can check the ones in the [gobl/uuid package](https://github.com/invopop/gobl/tree/main/uuid).


## Steps to include in Workflows

- If using Stripe Webhooks to fetch the invoice, the payload contains some unexpanded fields. To expand these fields, we recommend to include the `Expand with Stripe data` step.
- When converting from Stripe to GOBL, the Stripe invoice does not include any exchange rates. We add some default ones. To have updated ones you should include the exchange rates step in the workflow.
