// Package main provides a CLI interface for the library
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/invopop/gobl/bill"
	"github.com/joho/godotenv"
	"github.com/stripe/stripe-go/v84"
)

// build data provided by goreleaser and mage setup
var (
	name    = "gobl.stripe"
	version = "dev"
	date    = ""
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := godotenv.Load(".env"); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to load .env file: %w", err)
		}
	}

	return root().cmd().ExecuteContext(ctx)
}

func saveJSON(data interface{}) error {
	var filename string
	var prefix string

	switch v := data.(type) {
	case *bill.Invoice:
		filename = "gobl_" + v.Code.String() + ".json"
		prefix = "GOBL Invoice"
	case *stripe.Invoice:
		filename = "stripe_" + v.ID + ".json"
		prefix = "Stripe Invoice"
	case *stripe.CreditNote:
		filename = "stripe_" + v.ID + ".json"
		prefix = "Stripe Credit Note"
	default:
		return fmt.Errorf("unsupported type for JSON saving")
	}

	fullJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal %s: %v", prefix, err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close() // nolint: errcheck

	if _, err := file.Write(fullJSON); err != nil {
		return fmt.Errorf("failed to write to file: %v", err)
	}

	log.Printf("%s JSON saved to %s\n", prefix, filename)
	return nil
}
