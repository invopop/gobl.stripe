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
	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	assert.Nil(t, gi.ExchangeRates)

	s.Currency = "usd"
	gi, err = goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	assert.Equal(t, num.MakeAmount(935, 3), gi.ExchangeRates[0].Amount)
}

func TestZeroDecimalCurrencies(t *testing.T) {
	s := completeStripeInvoice()
	s.Currency = "jpy"
	s.Lines.Data[0].Currency = "jpy"
	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	assert.Equal(t, currency.JPY, gi.Currency)
	assert.Equal(t, num.NewAmount(-11000, 0), gi.Lines[0].Item.Price)
}

func TestToStripeInt(t *testing.T) {
	tests := []struct {
		amount   *num.Amount
		curr     currency.Code
		expected int64
	}{
		{
			amount:   num.NewAmount(11000, 2),
			curr:     currency.EUR,
			expected: 11000,
		},
		{
			amount:   num.NewAmount(1234567, 4),
			curr:     currency.EUR,
			expected: 12346,
		},
		{
			amount:   num.NewAmount(1234, 1),
			curr:     currency.EUR,
			expected: 12340,
		},
		{
			amount:   num.NewAmount(11000, 0),
			curr:     currency.JPY,
			expected: 11000,
		},
		{
			amount:   num.NewAmount(11000, 2),
			curr:     currency.JPY,
			expected: 110,
		},
		{
			amount:   num.NewAmount(11230, 2),
			curr:     currency.JPY,
			expected: 112,
		},
	}
	for _, test := range tests {
		assert.Equal(t, test.expected, goblstripe.ToStripeInt(test.amount, test.curr))
	}
}
