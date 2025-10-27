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
	"github.com/spf13/cobra"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/creditnote"
	"github.com/stripe/stripe-go/v81/invoice"
	"github.com/stripe/stripe-go/v81/webhook"
)

type listenOpts struct {
	*rootOpts
	port          string
	stripeKey     string
	webhookSecret string
	convertToGOBL bool
	//directory string
}

func listen(o *rootOpts) *listenOpts {
	return &listenOpts{rootOpts: o}
}

func (l *listenOpts) cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "listen",
		Short: "Listen to Stripe invoice/credit_note events",
		Long:  "Listen to Stripe invoice/credit_note events, save them as Stripe Invoice/ Credit Note JSON and convert it to GOBL json",
		RunE:  l.runE,
	}

	cmd.Flags().StringVarP(&l.port, "port", "p", "8080", "Port to listen for Stripe webhooks")
	cmd.Flags().StringVarP(&l.stripeKey, "stripe-key", "k", " ", "Stripe secret key")
	cmd.Flags().StringVarP(&l.webhookSecret, "webhook-secret", "s", " ", "Stripe webhook secret")
	cmd.Flags().BoolVarP(&l.convertToGOBL, "convert", "c", true, "Convert Stripe invoices to GOBL format")

	return cmd
}

func (l *listenOpts) runE(_ *cobra.Command, _ []string) error {
	server := &http.Server{
		Addr:    ":" + l.port,
		Handler: http.DefaultServeMux,
	}

	err := l.loadSecrets()
	if err != nil {
		return fmt.Errorf("failed to load secrets: %w", err)
	}
	stripe.Key = l.stripeKey

	http.HandleFunc("/webhook", l.handleWebhook)

	// Channel to listen for termination signals (control + c)
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	// Goroutine to start the server
	go func() {
		log.Printf("Listening on port %s\n", l.port)
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

func (l *listenOpts) handleWebhook(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		handleError(w, "Failed to read request body", err, http.StatusInternalServerError)
		return
	}

	event, err := webhook.ConstructEvent(body, req.Header.Get("Stripe-Signature"), l.webhookSecret)
	if err != nil {
		handleError(w, "Failed to construct event", err, http.StatusBadRequest)
		return
	}

	switch event.Type {
	case "invoice.created", "invoice.finalized":
		l.processInvoice(w, event)
	case "credit_note.created":
		l.processCreditNote(w, event)
	default:
		log.Printf("Unhandled event type: %s\n", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

// loadSecrets loads the Stripe secret key and webhook secret first from the arguments,
// then from the environment variables
func (l *listenOpts) loadSecrets() error {
	if l.stripeKey == " " {
		l.stripeKey = os.Getenv("STRIPE_SECRET_KEY")
		if l.stripeKey == "" {
			return fmt.Errorf("stripe secret key must be provided as an argument or in the STRIPE_SECRET_KEY environment variable")
		}
	}

	if l.webhookSecret == " " {
		l.webhookSecret = os.Getenv("STRIPE_WEBHOOK_SECRET")
		if l.webhookSecret == "" {
			return fmt.Errorf("stripe webhook secret must be provided as an argument or in the STRIPE_WEBHOOK_SECRET environment variable")
		}
	}
	return nil
}

func handleError(w http.ResponseWriter, message string, err error, statusCode int) {
	log.Printf("%s: %v", message, err)
	http.Error(w, fmt.Sprintf("%s: %v", message, err), statusCode)
}

func (l *listenOpts) processInvoice(w http.ResponseWriter, event stripe.Event) {
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

	if l.convertToGOBL {
		gi, err := convertInvoiceToGOBL(invoiceExpanded)
		if err != nil {
			handleError(w, "Failed to convert invoice to GOBL", err, http.StatusInternalServerError)
			return
		}

		if err := saveJSON(gi); err != nil {
			handleError(w, "Failed to save GOBL JSON", err, http.StatusInternalServerError)
		}
	}
}

func createInvoiceExpandParams() *stripe.InvoiceParams {
	params := &stripe.InvoiceParams{}
	params.AddExpand("account_tax_ids")
	//params.AddExpand("customer.tax_ids")
	params.AddExpand("lines.data.discounts")
	params.AddExpand("lines.data.tax_amounts.tax_rate")
	params.AddExpand("lines.data.price.product")
	params.AddExpand("total_tax_amounts.tax_rate")
	params.AddExpand("payment_intent")
	return params
}

func convertInvoiceToGOBL(invoiceNew *stripe.Invoice) (*bill.Invoice, error) {
	gi, err := goblstripe.FromInvoice(invoiceNew, nil)
	if err != nil {
		return nil, err
	}

	if err := gi.Calculate(); err != nil {
		return nil, err
	}

	return gi, nil
}

func (l *listenOpts) processCreditNote(w http.ResponseWriter, event stripe.Event) {
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

	if l.convertToGOBL {
		gi, err := convertCreditNoteToGOBL(creditNoteExpanded)
		if err != nil {
			handleError(w, "Failed to convert invoice to GOBL", err, http.StatusInternalServerError)
			return
		}

		if err := saveJSON(gi); err != nil {
			handleError(w, "Failed to save GOBL JSON", err, http.StatusInternalServerError)
		}
	}

}

func convertCreditNoteToGOBL(creditNoteNew *stripe.CreditNote) (*bill.Invoice, error) {
	gi, err := goblstripe.FromCreditNote(creditNoteNew, nil)
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
