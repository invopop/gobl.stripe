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

	// Payment should exist with advances (since AmountPaid > 0), but no terms or instructions
	require.NotNil(t, gi.Payment)
	assert.Nil(t, gi.Payment.Terms, "Paid invoice should not have payment terms")
	assert.Nil(t, gi.Payment.Instructions, "Paid invoice should not have payment instructions")
	assert.NotNil(t, gi.Payment.Advances, "Paid invoice should have advances")
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

func TestPaymentAdvancesZeroAmountPaid(t *testing.T) {
	// When AmountPaid == 0, should return nil (no advances)
	s := minimalStripeInvoice()
	s.AmountPaid = 0
	s.Paid = false
	s.DueDate = 1737738363

	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	// Payment should still exist (due to terms/instructions) but without advances
	if gi.Payment != nil {
		assert.Nil(t, gi.Payment.Advances)
	}
}

func TestPaymentAdvancesWithoutCharge(t *testing.T) {
	// When AmountPaid > 0 and Charge is nil, should create advance with default description
	s := minimalStripeInvoice()
	s.AmountPaid = 5000
	s.Charge = nil

	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	require.NotNil(t, gi.Payment)
	require.NotNil(t, gi.Payment.Advances)
	require.Len(t, gi.Payment.Advances, 1)

	advance := gi.Payment.Advances[0]
	assert.Equal(t, 50.0, advance.Amount.Float64(), "Advance amount should be 50.00 EUR")
	assert.Equal(t, "Advance payment", advance.Description, "Should use default description")
	assert.Nil(t, advance.Date, "Date should be nil when no Charge")
	assert.Empty(t, advance.Key, "Key should be empty when no Charge")
}

func TestPaymentAdvancesWithCharge(t *testing.T) {
	// When AmountPaid > 0 and Charge is set, should use Charge details
	s := minimalStripeInvoice()
	s.AmountPaid = 7500
	s.Charge = &stripe.Charge{
		Created:     1737738363, // 2025-01-24
		Description: "Payment for Invoice INV-123",
	}

	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	require.NotNil(t, gi.Payment)
	require.NotNil(t, gi.Payment.Advances)
	require.Len(t, gi.Payment.Advances, 1)

	advance := gi.Payment.Advances[0]
	assert.Equal(t, 75.0, advance.Amount.Float64(), "Advance amount should be 75.00 EUR")
	assert.Equal(t, "Payment for Invoice INV-123", advance.Description, "Should use Charge description")
	require.NotNil(t, advance.Date, "Date should be set from Charge.Created")
	assert.Equal(t, "2025-01-24", advance.Date.String(), "Date should match Charge.Created")
	assert.Empty(t, advance.Key, "Key should be empty when no PaymentMethodDetails")
}

func TestPaymentAdvancesWithChargeEmptyDescription(t *testing.T) {
	// When Charge.Description is empty, should keep default description
	s := minimalStripeInvoice()
	s.AmountPaid = 8000
	s.Charge = &stripe.Charge{
		Created:     1737738363,
		Description: "", // Empty description
	}

	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	require.NotNil(t, gi.Payment)
	require.NotNil(t, gi.Payment.Advances)
	require.Len(t, gi.Payment.Advances, 1)

	advance := gi.Payment.Advances[0]
	assert.Equal(t, 80.0, advance.Amount.Float64())
	assert.Equal(t, "Advance payment", advance.Description, "Should use default description when Charge.Description is empty")
	require.NotNil(t, advance.Date, "Date should still be set from Charge.Created")
}

func TestPaymentAdvancesWithChargeAndPaymentMethodCard(t *testing.T) {
	// When Charge has PaymentMethodDetails of type card
	s := minimalStripeInvoice()
	s.AmountPaid = 10000
	s.Charge = &stripe.Charge{
		Created:     1737738363,
		Description: "Card payment",
		PaymentMethodDetails: &stripe.ChargePaymentMethodDetails{
			Type: stripe.ChargePaymentMethodDetailsTypeCard,
		},
	}

	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	require.NotNil(t, gi.Payment)
	require.NotNil(t, gi.Payment.Advances)
	require.Len(t, gi.Payment.Advances, 1)

	advance := gi.Payment.Advances[0]
	assert.Equal(t, 100.0, advance.Amount.Float64())
	assert.Equal(t, "Card payment", advance.Description)
	assert.Equal(t, pay.MeansKeyCard, advance.Key, "Key should be card for card payment")
}

func TestPaymentAdvancesWithChargeAndPaymentMethodSEPA(t *testing.T) {
	// When Charge has PaymentMethodDetails of type SEPA debit
	s := minimalStripeInvoice()
	s.AmountPaid = 20000
	s.Charge = &stripe.Charge{
		Created:     1737738363,
		Description: "SEPA Direct Debit payment",
		PaymentMethodDetails: &stripe.ChargePaymentMethodDetails{
			Type: stripe.ChargePaymentMethodDetailsTypeSEPADebit,
		},
	}

	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	require.NotNil(t, gi.Payment)
	require.NotNil(t, gi.Payment.Advances)
	require.Len(t, gi.Payment.Advances, 1)

	advance := gi.Payment.Advances[0]
	assert.Equal(t, 200.0, advance.Amount.Float64())
	assert.Equal(t, "SEPA Direct Debit payment", advance.Description)
	assert.Equal(t, pay.MeansKeyDirectDebit, advance.Key, "Key should be direct-debit for SEPA")
}

func TestPaymentAdvancesWithChargeAndPaymentMethodOnline(t *testing.T) {
	// When Charge has PaymentMethodDetails of type bancontact (online payment)
	s := minimalStripeInvoice()
	s.AmountPaid = 15000
	s.Charge = &stripe.Charge{
		Created:     1737738363,
		Description: "Bancontact payment",
		PaymentMethodDetails: &stripe.ChargePaymentMethodDetails{
			Type: stripe.ChargePaymentMethodDetailsTypeBancontact,
		},
	}

	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	require.NotNil(t, gi.Payment)
	require.NotNil(t, gi.Payment.Advances)
	require.Len(t, gi.Payment.Advances, 1)

	advance := gi.Payment.Advances[0]
	assert.Equal(t, 150.0, advance.Amount.Float64())
	assert.Equal(t, "Bancontact payment", advance.Description)
	assert.Equal(t, pay.MeansKeyOnline, advance.Key, "Key should be online for Bancontact")
}

func TestPaymentAdvancesWithChargeAndPaymentMethodACHDebit(t *testing.T) {
	// When Charge has PaymentMethodDetails of type ach_debit (direct debit)
	s := minimalStripeInvoice()
	s.AmountPaid = 50000
	s.Charge = &stripe.Charge{
		Created:     1737738363,
		Description: "ACH Debit payment",
		PaymentMethodDetails: &stripe.ChargePaymentMethodDetails{
			Type: stripe.ChargePaymentMethodDetailsTypeACHDebit,
		},
	}

	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	require.NotNil(t, gi.Payment)
	require.NotNil(t, gi.Payment.Advances)
	require.Len(t, gi.Payment.Advances, 1)

	advance := gi.Payment.Advances[0]
	assert.Equal(t, 500.0, advance.Amount.Float64())
	assert.Equal(t, "ACH Debit payment", advance.Description)
	assert.Equal(t, pay.MeansKeyDirectDebit, advance.Key, "Key should be direct-debit for ACH")
}

func TestPaymentAdvancesWithChargeAndUnknownPaymentMethod(t *testing.T) {
	// When Charge has PaymentMethodDetails of an unknown type
	s := minimalStripeInvoice()
	s.AmountPaid = 3000
	s.Charge = &stripe.Charge{
		Created:     1737738363,
		Description: "Unknown method payment",
		PaymentMethodDetails: &stripe.ChargePaymentMethodDetails{
			Type: "some_unknown_type",
		},
	}

	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	require.NotNil(t, gi.Payment)
	require.NotNil(t, gi.Payment.Advances)
	require.Len(t, gi.Payment.Advances, 1)

	advance := gi.Payment.Advances[0]
	assert.Equal(t, 30.0, advance.Amount.Float64())
	assert.Equal(t, "Unknown method payment", advance.Description)
	assert.Empty(t, advance.Key, "Key should be empty for unknown payment method type")
}
