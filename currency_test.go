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

	// When changing invoice currency to USD, update line currencies too
	// to maintain consistency (otherwise Calculate() fails with exchange rate errors)
	s.Currency = "usd"
	for _, line := range s.Lines.Data {
		line.Currency = "usd"
	}
	// Total stays the same since we're just relabeling the currency
	gi, err = goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	// Exchange rate is created for the regime (DE) to convert USD to EUR
	assert.Equal(t, num.MakeAmount(935, 3), gi.ExchangeRates[0].Amount)
}

func TestZeroDecimalCurrencies(t *testing.T) {
	s := completeStripeInvoice()
	s.Currency = "jpy"
	// Update all lines to JPY and set matching Total
	for _, line := range s.Lines.Data {
		line.Currency = "jpy"
	}
	// JPY is a zero-decimal currency, so Total is the face value
	// Sum of lines: -11000 + 19999 + 10000 = 18999
	// Tax at 19%: ~3610
	// Total: ~22609 (but in JPY it's just the integer value)
	s.Total = 22609
	gi, err := goblstripe.FromInvoice(s, validStripeAccount())
	require.NoError(t, err)

	assert.Equal(t, currency.JPY, gi.Currency)
	assert.Equal(t, num.NewAmount(11000, 0), gi.Lines[0].Item.Price)
}

func TestDefaultRate(t *testing.T) {
	tests := []struct {
		name     string
		from     currency.Code
		to       currency.Code
		expected num.Amount
	}{
		{
			name:     "USD to EUR",
			from:     currency.USD,
			to:       currency.EUR,
			expected: num.MakeAmount(935, 3),
		},
		{
			name:     "USD to PLN",
			from:     currency.USD,
			to:       currency.PLN,
			expected: num.MakeAmount(4015, 3),
		},
		{
			name:     "EUR to PLN",
			from:     currency.EUR,
			to:       currency.PLN,
			expected: num.MakeAmount(4294, 3),
		},
		{
			name:     "USD to AUD",
			from:     currency.USD,
			to:       currency.AUD,
			expected: num.MakeAmount(1577, 3),
		},
		{
			name:     "USD to JPY",
			from:     currency.USD,
			to:       currency.JPY,
			expected: num.MakeAmount(149, 0),
		},
		{
			name:     "USD to CHF",
			from:     currency.USD,
			to:       currency.CHF,
			expected: num.MakeAmount(883, 3),
		},
		{
			name:     "USD to SEK",
			from:     currency.USD,
			to:       currency.SEK,
			expected: num.MakeAmount(10512, 3),
		},
		{
			name:     "USD to ZAR",
			from:     currency.USD,
			to:       currency.ZAR,
			expected: num.MakeAmount(18234, 3),
		},
		{
			name:     "unknown from currency returns 0",
			from:     currency.Code("XXX"),
			to:       currency.EUR,
			expected: num.AmountZero,
		},
		{
			name:     "unknown to currency returns 0",
			from:     currency.USD,
			to:       currency.Code("XXX"),
			expected: num.AmountZero,
		},
		{
			name:     "both unknown returns 0",
			from:     currency.Code("AAA"),
			to:       currency.Code("BBB"),
			expected: num.AmountZero,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := goblstripe.DefaultRate(test.from, test.to)
			assert.Equal(t, test.expected, result)
		})
	}
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
