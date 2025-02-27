package main

import (
	"encoding/json"
	"fmt"
	"io"

	goblstripe "github.com/invopop/gobl.stripe"
	"github.com/invopop/gobl/bill"
	"github.com/spf13/cobra"
	"github.com/stripe/stripe-go/v81"
)

type convertOpts struct {
	*rootOpts
	direction string // "to-gobl" or "to-stripe"
}

func convert(o *rootOpts) *convertOpts {
	return &convertOpts{rootOpts: o}
}

func (c *convertOpts) cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert <infile>",
		Short: "Convert between Stripe and GOBL formats",
		Long:  "Convert invoice files between Stripe JSON format and GOBL format",
		RunE:  c.runE,
	}

	cmd.Flags().StringVarP(&c.direction, "direction", "d", "from-stripe", "Direction of conversion: 'to-stripe' or 'from-stripe'")

	cmd.MarkFlagRequired("input")

	return cmd
}

func (c *convertOpts) runE(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || len(args) > 1 {
		return fmt.Errorf("expected only one argument, the command usage is `gobl.stripe convert <infile>`")
	}

	// Validate direction flag
	if c.direction != "from-stripe" && c.direction != "to-stripe" {
		return fmt.Errorf("direction must be either 'from-stripe' or 'to-stripe'")
	}

	if c.direction == "to-stripe" {
		return fmt.Errorf("conversion to Stripe format is not yet supported")
	}

	input, err := openInput(cmd, args)
	if err != nil {
		return err
	}
	defer input.Close() // nolint:errcheck

	inData, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	// Currently only supporting Stripe to GOBL conversion
	return c.stripeToGobl(inData)
}

func (c *convertOpts) stripeToGobl(data []byte) error {
	// Determine if this is an invoice or credit note
	var objMap map[string]interface{}
	if err := json.Unmarshal(data, &objMap); err != nil {
		return fmt.Errorf("failed to parse JSON: %v", err)
	}

	objType, ok := objMap["object"].(string)
	if !ok {
		return fmt.Errorf("could not determine object type from Stripe JSON")
	}

	var goblInvoice *bill.Invoice
	var err error

	switch objType {
	case "invoice":
		var stripeInvoice stripe.Invoice
		if err := json.Unmarshal(data, &stripeInvoice); err != nil {
			return fmt.Errorf("failed to parse Stripe invoice: %v", err)
		}
		goblInvoice, err = goblstripe.FromInvoice(&stripeInvoice)
		if err != nil {
			return fmt.Errorf("failed to convert to GOBL: %v", err)
		}
	case "credit_note":
		var stripeCreditNote stripe.CreditNote
		if err := json.Unmarshal(data, &stripeCreditNote); err != nil {
			return fmt.Errorf("failed to parse Stripe credit note: %v", err)
		}
		goblInvoice, err = goblstripe.FromCreditNote(&stripeCreditNote)
		if err != nil {
			return fmt.Errorf("failed to convert to GOBL: %v", err)
		}
	default:
		return fmt.Errorf("unsupported Stripe object type: %s", objType)
	}

	// Calculate the GOBL invoice
	if err := goblInvoice.Calculate(); err != nil {
		return fmt.Errorf("failed to calculate GOBL invoice: %v", err)
	}

	// Write to output file
	return saveJSON(goblInvoice)
}
