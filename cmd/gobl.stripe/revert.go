package main

// Change name to listen?
import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	goblstripe "github.com/invopop/gobl.stripe"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/uuid"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/creditnote"
	"github.com/stripe/stripe-go/v81/invoice"
	"github.com/stripe/stripe-go/v81/webhook"
)

type revertOpts struct {
	*rootOpts
	port string
	//directory string
}

func revert(o *rootOpts) *revertOpts {
	return &revertOpts{rootOpts: o}
}

func (c *revertOpts) cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revert",
		Short: "Receive a finalized Stripe Invoice/Credit Note and convert it to a GOBL JSON document",
		RunE:  c.runE,
	}

	cmd.Flags().StringVarP(&c.port, "port", "p", "8080", "Port to listen for Stripe webhooks")
	//cmd.Flags().StringVarP(&c.directory, "directory", "d", ".", "Directory to save GOBL JSON files")

	return cmd
}

func (c *revertOpts) runE(cmd *cobra.Command, args []string) error {
	server := &http.Server{
		Addr:    ":" + c.port,
		Handler: http.DefaultServeMux,
	}

	http.HandleFunc("/webhook", c.handleWebhook)

	// Channel to listen for termination signals (control + c)
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	// Goroutine to start the server
	go func() {
		log.Printf("Listening on port %s\n", c.port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v\n", err)
		}
	}()

	// Terminate the server when a signal is received
	<-stopChan
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v\n", err)
	}
	log.Println("Server stopped gracefully.")
	return nil
}

func (c revertOpts) handleWebhook(w http.ResponseWriter, r *http.Request) {
	webhookSecret, err := loadSecrets()
	if err != nil {
		handleError(w, "Failed to load webhook secret", err, http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		handleError(w, "Failed to read request body", err, http.StatusInternalServerError)
		return
	}

	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), webhookSecret)
	if err != nil {
		handleError(w, "Failed to construct event", err, http.StatusBadRequest)
		return
	}

	switch event.Type {
	case "invoice.created", "invoice.finalized":
		processInvoice(w, event)
	case "credit_note.created":
		processCreditNote(w, event)
	default:
		log.Printf("Unhandled event type: %s\n", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

// loadSecrets loads the Stripe secret key and webhook secret from the .env file
func loadSecrets() (string, error) {
	err := godotenv.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load .env file: %w", err)
	}
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	return os.Getenv("STRIPE_WEBHOOK_SECRET"), nil
}

func handleError(w http.ResponseWriter, message string, err error, statusCode int) {
	log.Printf("%s: %v", message, err)
	http.Error(w, fmt.Sprintf("%s: %v", message, err), statusCode)
}

func processInvoice(w http.ResponseWriter, event stripe.Event) {
	var invoiceReceived stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoiceReceived); err != nil {
		handleError(w, "Failed to parse invoice", err, http.StatusBadRequest)
		return
	}

	params := createInvoiceExpandParams()

	invoiceExpanded, err := invoice.Get(invoiceReceived.ID, params)
	if err != nil {
		handleError(w, "Failed to get invoice", err, http.StatusBadRequest)
		return
	}

	if err := saveJSON(invoiceExpanded); err != nil {
		handleError(w, "Failed to save Stripe JSON", err, http.StatusInternalServerError)
	}

	gi, err := convertInvoiceToGOBL(invoiceExpanded)
	if err != nil {
		handleError(w, "Failed to convert invoice to GOBL", err, http.StatusInternalServerError)
		return
	}

	if err := saveJSON(gi); err != nil {
		handleError(w, "Failed to save GOBL JSON", err, http.StatusInternalServerError)
	}
}

func createInvoiceExpandParams() *stripe.InvoiceParams {
	params := &stripe.InvoiceParams{}
	params.AddExpand("account_tax_ids")
	params.AddExpand("customer.tax_ids")
	params.AddExpand("lines.data.discounts")
	params.AddExpand("lines.data.tax_amounts.tax_rate")
	params.AddExpand("total_tax_amounts.tax_rate")
	params.AddExpand("payment_intent")
	return params
}

func convertInvoiceToGOBL(invoiceNew *stripe.Invoice) (*bill.Invoice, error) {
	namespace := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	gi, err := goblstripe.FromInvoice(invoiceNew, namespace)
	if err != nil {
		return nil, err
	}

	if err := gi.Calculate(); err != nil {
		return nil, err
	}

	return gi, nil
}

func processCreditNote(w http.ResponseWriter, event stripe.Event) {
	var creditNoteReceived stripe.CreditNote
	if err := json.Unmarshal(event.Data.Raw, &creditNoteReceived); err != nil {
		handleError(w, "Failed to parse credit note", err, http.StatusBadRequest)
		return
	}

	params := createCreditNoteExpandParams()

	creditNoteExpanded, err := creditnote.Get(creditNoteReceived.ID, params)
	if err != nil {
		handleError(w, "Failed to get credit note", err, http.StatusBadRequest)
		return
	}

	if err := saveJSON(creditNoteExpanded); err != nil {
		handleError(w, "Failed to save Stripe JSON", err, http.StatusInternalServerError)
	}

	gi, err := convertCreditNoteToGOBL(creditNoteExpanded)
	if err != nil {
		handleError(w, "Failed to convert invoice to GOBL", err, http.StatusInternalServerError)
		return
	}

	if err := saveJSON(gi); err != nil {
		handleError(w, "Failed to save GOBL JSON", err, http.StatusInternalServerError)
	}

}

func convertCreditNoteToGOBL(creditNoteNew *stripe.CreditNote) (*bill.Invoice, error) {
	namespace := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	gi, err := goblstripe.FromCreditNote(creditNoteNew, namespace)
	if err != nil {
		return nil, err
	}

	if err := gi.Calculate(); err != nil {
		return nil, err
	}

	return gi, nil
}

func createCreditNoteExpandParams() *stripe.CreditNoteParams {
	params := &stripe.CreditNoteParams{}
	params.AddExpand("customer.tax_ids")
	params.AddExpand("invoice.account_tax_ids")
	params.AddExpand("lines.data.tax_amounts.tax_rate")
	return params
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
	defer file.Close()

	if _, err := file.Write(fullJSON); err != nil {
		return fmt.Errorf("failed to write to file: %v", err)
	}

	log.Printf("%s JSON saved to %s\n", prefix, filename)
	return nil
}
