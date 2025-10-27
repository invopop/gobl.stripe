package goblstripe_test

import (
	"testing"

	goblstripe "github.com/invopop/gobl.stripe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoTermsWhenPaid(t *testing.T) {
	s := completeStripeInvoice()
	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	assert.Nil(t, gi.Payment)
}
