package goblstripe_test

import (
	"testing"

	goblstripe "github.com/invopop/gobl.stripe"
	"github.com/invopop/gobl/pay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v81"
)

func TestNoTermsWhenPaid(t *testing.T) {
	s := completeStripeInvoice()
	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	assert.Nil(t, gi.Payment)
}

func TestPaymentInstructionsFromDefaultPaymentMethod(t *testing.T) {
	t.Run("card payment method", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.Paid = false
		s.DueDate = 1737738363
		s.DefaultPaymentMethod = &stripe.PaymentMethod{
			Type: stripe.PaymentMethodTypeCard,
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Payment)
		require.NotNil(t, gi.Payment.Instructions)
		assert.Equal(t, pay.MeansKeyCard, gi.Payment.Instructions.Key)
		assert.Equal(t, "Card", gi.Payment.Instructions.Detail)
	})

	t.Run("sepa debit payment method", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.Paid = false
		s.DueDate = 1737738363
		s.DefaultPaymentMethod = &stripe.PaymentMethod{
			Type: stripe.PaymentMethodTypeSEPADebit,
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Payment)
		require.NotNil(t, gi.Payment.Instructions)
		assert.Equal(t, pay.MeansKeyDirectDebit, gi.Payment.Instructions.Key)
		assert.Equal(t, "SEPA Direct Debit", gi.Payment.Instructions.Detail)
	})

	t.Run("customer balance (bank transfer) payment method", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.Paid = false
		s.DueDate = 1737738363
		s.DefaultPaymentMethod = &stripe.PaymentMethod{
			Type: stripe.PaymentMethodTypeCustomerBalance,
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Payment)
		require.NotNil(t, gi.Payment.Instructions)
		assert.Equal(t, pay.MeansKeyCreditTransfer, gi.Payment.Instructions.Key)
		assert.Equal(t, "Bank Transfer", gi.Payment.Instructions.Detail)
	})

	t.Run("paypal payment method", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.Paid = false
		s.DueDate = 1737738363
		s.DefaultPaymentMethod = &stripe.PaymentMethod{
			Type: stripe.PaymentMethodTypePaypal,
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Payment)
		require.NotNil(t, gi.Payment.Instructions)
		assert.Equal(t, pay.MeansKeyOnline, gi.Payment.Instructions.Key)
		assert.Equal(t, "PayPal", gi.Payment.Instructions.Detail)
	})
}

func TestPaymentInstructionsFromCustomerInvoiceSettings(t *testing.T) {
	t.Run("card from customer settings", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.Paid = false
		s.DueDate = 1737738363
		s.Customer = &stripe.Customer{
			InvoiceSettings: &stripe.CustomerInvoiceSettings{
				DefaultPaymentMethod: &stripe.PaymentMethod{
					Type: stripe.PaymentMethodTypeCard,
				},
			},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Payment)
		require.NotNil(t, gi.Payment.Instructions)
		assert.Equal(t, pay.MeansKeyCard, gi.Payment.Instructions.Key)
		assert.Equal(t, "Card", gi.Payment.Instructions.Detail)
	})

	t.Run("sepa debit from customer settings", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.Paid = false
		s.DueDate = 1737738363
		s.Customer = &stripe.Customer{
			InvoiceSettings: &stripe.CustomerInvoiceSettings{
				DefaultPaymentMethod: &stripe.PaymentMethod{
					Type: stripe.PaymentMethodTypeSEPADebit,
				},
			},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Payment)
		require.NotNil(t, gi.Payment.Instructions)
		assert.Equal(t, pay.MeansKeyDirectDebit, gi.Payment.Instructions.Key)
		assert.Equal(t, "SEPA Direct Debit", gi.Payment.Instructions.Detail)
	})
}

func TestPaymentInstructionsDefaultPaymentMethodPriority(t *testing.T) {
	t.Run("invoice default takes priority over customer settings", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.Paid = false
		s.DueDate = 1737738363
		// Invoice has card payment method
		s.DefaultPaymentMethod = &stripe.PaymentMethod{
			Type: stripe.PaymentMethodTypeCard,
		}
		// Customer settings has SEPA debit
		s.Customer = &stripe.Customer{
			InvoiceSettings: &stripe.CustomerInvoiceSettings{
				DefaultPaymentMethod: &stripe.PaymentMethod{
					Type: stripe.PaymentMethodTypeSEPADebit,
				},
			},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Payment)
		require.NotNil(t, gi.Payment.Instructions)
		// Should use invoice's DefaultPaymentMethod (card), not customer's (sepa_debit)
		assert.Equal(t, pay.MeansKeyCard, gi.Payment.Instructions.Key)
		assert.Equal(t, "Card", gi.Payment.Instructions.Detail)
	})

	t.Run("falls back to customer settings when invoice default is nil", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.Paid = false
		s.DueDate = 1737738363
		s.DefaultPaymentMethod = nil // No invoice default
		s.Customer = &stripe.Customer{
			InvoiceSettings: &stripe.CustomerInvoiceSettings{
				DefaultPaymentMethod: &stripe.PaymentMethod{
					Type: stripe.PaymentMethodTypeSEPADebit,
				},
			},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Payment)
		require.NotNil(t, gi.Payment.Instructions)
		// Should fall back to customer's DefaultPaymentMethod
		assert.Equal(t, pay.MeansKeyDirectDebit, gi.Payment.Instructions.Key)
		assert.Equal(t, "SEPA Direct Debit", gi.Payment.Instructions.Detail)
	})

	t.Run("falls back to payment intent when no defaults set", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.Paid = false
		s.DueDate = 1737738363
		s.DefaultPaymentMethod = nil
		s.Customer = nil
		s.PaymentIntent = &stripe.PaymentIntent{
			PaymentMethodTypes: []string{"card", "sepa_debit"},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Payment)
		require.NotNil(t, gi.Payment.Instructions)
		// Should use payment intent methods
		assert.True(t, gi.Payment.Instructions.Key.Has(pay.MeansKeyCard))
		assert.True(t, gi.Payment.Instructions.Key.Has(pay.MeansKeyDirectDebit))
	})
}

func TestPaymentInstructionsNilCustomerInvoiceSettings(t *testing.T) {
	t.Run("customer with nil invoice settings", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.Paid = false
		s.DueDate = 1737738363
		s.DefaultPaymentMethod = nil
		s.Customer = &stripe.Customer{
			Name:            "Test",
			InvoiceSettings: nil, // No invoice settings
		}
		s.PaymentIntent = &stripe.PaymentIntent{
			PaymentMethodTypes: []string{"card"},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Payment)
		require.NotNil(t, gi.Payment.Instructions)
		// Should fall through to payment intent
		assert.Equal(t, pay.MeansKeyCard, gi.Payment.Instructions.Key)
	})

	t.Run("customer invoice settings with nil default payment method", func(t *testing.T) {
		s := minimalStripeInvoice()
		s.Paid = false
		s.DueDate = 1737738363
		s.DefaultPaymentMethod = nil
		s.Customer = &stripe.Customer{
			Name: "Test",
			InvoiceSettings: &stripe.CustomerInvoiceSettings{
				DefaultPaymentMethod: nil, // No default payment method
			},
		}
		s.PaymentIntent = &stripe.PaymentIntent{
			PaymentMethodTypes: []string{"card"},
		}

		gi, err := goblstripe.FromInvoice(s, validStripeAccount())
		require.NoError(t, err)

		require.NotNil(t, gi.Payment)
		require.NotNil(t, gi.Payment.Instructions)
		// Should fall through to payment intent
		assert.Equal(t, pay.MeansKeyCard, gi.Payment.Instructions.Key)
	})
}
