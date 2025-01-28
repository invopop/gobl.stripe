package goblstripe_test

import (
	"testing"

	goblstripe "github.com/invopop/gobl.stripe"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/gobl/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaxInclusive(t *testing.T) {

	s := validStripeInvoice()

	// Check that tax are inclusive
	gi, err := goblstripe.FromInvoice(s, uuid.MustParse(namespace))
	require.NoError(t, err)
	assert.Nil(t, gi.Tax)

	s.TotalTaxAmounts[0].Inclusive = true
	gi, err = goblstripe.FromInvoice(s, uuid.MustParse(namespace))
	require.NoError(t, err)
	assert.Equal(t, tax.CategoryVAT, gi.Tax.PricesInclude)
}
