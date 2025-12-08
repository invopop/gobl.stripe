package goblstripe_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	goblstripe "github.com/invopop/gobl.stripe"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v81"
)

var skipExamplePaths = []string{
	"build/",
	".out.",
	"/out/",
	"data/",
	".github",
	".golangci.yaml",
	"wasm/",
}

var updateExamples = flag.Bool("update", false, "Update the examples in the repository")

// TestConvertExamplesToJSON finds all of the `.json` and `.yaml` files in the
// package and attempts to convert the to JSON Envelopes.
func TestConvertExamplesToJSON(t *testing.T) {
	// Find all .yaml files in subdirectories
	var files []string
	err := filepath.Walk("./", func(path string, _ os.FileInfo, _ error) error {
		switch filepath.Ext(path) {
		case ".json":
			for _, skip := range skipExamplePaths {
				if strings.Contains(path, skip) {
					return nil
				}
			}
			files = append(files, path)
		}
		return nil
	})
	require.NoError(t, err)

	for _, path := range files {
		assert.NoError(t, processFile(t, path))
	}
}

func processFile(t *testing.T, path string) error {
	t.Helper()
	t.Logf("processing file: %v", path)

	// attempt to load and convert
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	var objMap map[string]interface{}
	if err := json.Unmarshal(data, &objMap); err != nil {
		return fmt.Errorf("failed to parse JSON: %v", err)
	}

	objType, ok := objMap["object"].(string)
	if !ok {
		return fmt.Errorf("could not determine object type from Stripe JSON")
	}

	var goblInvoice *bill.Invoice

	switch objType {
	case "invoice":
		var stripeInvoice stripe.Invoice
		if err := json.Unmarshal(data, &stripeInvoice); err != nil {
			return fmt.Errorf("failed to parse Stripe invoice: %v", err)
		}
		goblInvoice, err = goblstripe.FromInvoice(&stripeInvoice, validStripeAccount())
		if err != nil && goblInvoice == nil {
			return fmt.Errorf("failed to convert to GOBL: %v", err)
		}
	case "credit_note":
		var stripeCreditNote stripe.CreditNote
		if err := json.Unmarshal(data, &stripeCreditNote); err != nil {
			return fmt.Errorf("failed to parse Stripe credit note: %v", err)
		}
		goblInvoice, err = goblstripe.FromCreditNote(&stripeCreditNote, validStripeAccount())
		if err != nil && goblInvoice == nil {
			return fmt.Errorf("failed to convert to GOBL: %v", err)
		}
	default:
		return fmt.Errorf("unsupported Stripe object type: %s", objType)
	}

	// override the document UUID for consistent test results
	goblInvoice.UUID = uuid.MustParse("019860fc-7d4c-7922-a371-e848ca5141d3")

	env, err := gobl.Envelop(goblInvoice)
	if err != nil {
		return fmt.Errorf("failed to create envelop: %v", err)
	}

	// override the envelope UUID for consistent test results
	env.Head.UUID = uuid.MustParse("8a51fd30-2a27-11ee-be56-0242ac120002")

	if err := env.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Output to the filesystem in the /out/ directory
	out, err := json.MarshalIndent(env, "", "	")
	if err != nil {
		return fmt.Errorf("marshalling output: %w", err)
	}

	dir := filepath.Join(filepath.Dir(path), "out")
	baseName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	// Replace "stripe_" prefix with "gobl_" for output files
	if strings.HasPrefix(baseName, "stripe_") {
		baseName = "gobl_" + strings.TrimPrefix(baseName, "stripe_")
	}
	of := baseName + ".json"
	np := filepath.Join(dir, of)
	if _, err := os.Stat(np); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("checking file: %s: %w", np, err)
		}
		if !*updateExamples {
			return fmt.Errorf("output file missing, run tests with `--update` flag to create")
		}
	}

	if *updateExamples {
		if err := os.WriteFile(np, out, 0644); err != nil {
			return fmt.Errorf("saving file data: %w", err)
		}
		t.Logf("wrote file: %v", np)
	} else {
		// Compare to existing file
		existing, err := os.ReadFile(np)
		if err != nil {
			return fmt.Errorf("reading existing file: %w", err)
		}
		t.Run(np, func(t *testing.T) {
			assert.JSONEq(t, string(existing), string(out), "output file does not match, run tests with `--update` flag to update")
		})
	}

	return nil
}
