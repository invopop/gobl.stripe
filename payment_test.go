package goblstripe_test

import (
	"testing"

	goblstripe "github.com/invopop/gobl.stripe"
	"github.com/invopop/gobl/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoTermsWhenPaid(t *testing.T) {
	s := validStripeInvoice()
	gi, err := goblstripe.FromInvoice(s, uuid.MustParse(namespace))
	require.NoError(t, err)

	assert.Nil(t, gi.Payment)
}
