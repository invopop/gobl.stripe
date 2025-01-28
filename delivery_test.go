package goblstripe_test

import (
	"testing"

	goblstripe "github.com/invopop/gobl.stripe"
	"github.com/invopop/gobl/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShippingDetails(t *testing.T) {
	s := validStripeInvoice()
	gi, err := goblstripe.FromInvoice(s, uuid.MustParse(namespace))
	require.NoError(t, err)

	assert.Equal(t, "Test Customer", gi.Delivery.Receiver.Name)
	assert.Equal(t, "Berlin", gi.Delivery.Receiver.Addresses[0].Locality)
	assert.Equal(t, "DE", gi.Delivery.Receiver.Addresses[0].Country.String())
	assert.Equal(t, "Unter den Linden 1", gi.Delivery.Receiver.Addresses[0].Street)
	assert.Equal(t, "10117", gi.Delivery.Receiver.Addresses[0].Code.String())
	assert.Equal(t, "BE", gi.Delivery.Receiver.Addresses[0].State.String())
}
