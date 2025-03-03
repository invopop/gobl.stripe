package goblstripe_test

import (
	"testing"

	goblstripe "github.com/invopop/gobl.stripe"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExchangeRateConversionDefault(t *testing.T) {
	s := completeStripeInvoice()
	gi, err := goblstripe.FromInvoice(s)
	require.NoError(t, err)

	assert.Nil(t, gi.ExchangeRates)

	s.Currency = "usd"
	gi, err = goblstripe.FromInvoice(s)
	require.NoError(t, err)

	assert.Equal(t, num.MakeAmount(935, 3), gi.ExchangeRates[0].Amount)
}

func TestZeroDecimalCurrencies(t *testing.T) {
	s := completeStripeInvoice()
	s.Currency = "jpy"
	s.Lines.Data[0].Currency = "jpy"
	gi, err := goblstripe.FromInvoice(s)
	require.NoError(t, err)

	assert.Equal(t, currency.JPY, gi.Currency)
	assert.Equal(t, num.MakeAmount(-11000, 0), gi.Lines[0].Item.Price)
}
