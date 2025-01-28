package goblstripe

import (
	"slices"
	"strings"

	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
	"github.com/stripe/stripe-go/v81"
)

// https://docs.stripe.com/currencies#zero-decimal
var zeroDecimalCurrencies = []currency.Code{currency.BIF, currency.CLF, currency.DJF, currency.GNF, currency.JPY, currency.KMF,
	currency.KRW, currency.MGA, currency.PYG, currency.RWF, currency.VND, currency.VUV, currency.XAF, currency.XOF, currency.XPF}

// DefaultRates provides a list of default exchange rates based on the same currency (USD).
var DefaultRates = map[currency.Code]num.Amount{
	currency.USD: num.MakeAmount(1, 0),
	currency.EUR: num.MakeAmount(935, 3),
	currency.GBP: num.MakeAmount(788, 3),
	currency.BRL: num.MakeAmount(5379, 3),
	currency.MXN: num.MakeAmount(18467, 3),
	currency.COP: num.MakeAmount(4132934, 3),
}

// FromCurrency converts a stripe currency into a GOBL currency code.
func FromCurrency(curr stripe.Currency) currency.Code {
	return currency.Code(strings.ToUpper(string(curr)))
}

// currencyAmount creates a currency amount object from a value and a currency code.
func currencyAmount(val int64, curr currency.Code) num.Amount {
	var exp uint32 = 2
	if slices.Contains(zeroDecimalCurrencies, curr) {
		exp = 0
	}

	return num.MakeAmount(val, exp)
}

// DefaultRate returns the default exchange rate for any pair of currencies.
func DefaultRate(from, to currency.Code) num.Amount {
	fromRate, ok := DefaultRates[from]
	if !ok {
		return num.AmountZero
	}

	toRate, ok := DefaultRates[to]
	if !ok {
		return num.AmountZero
	}

	toRate = toRate.MatchPrecision(fromRate)

	return toRate.Divide(fromRate)
}

// newExchangeRates creates the exchange rates for the invoice.
func newExchangeRates(curr currency.Code, regime *tax.RegimeDef) []*currency.ExchangeRate {
	if curr == regime.Currency {
		// The invoice's and the regime's currency are the same. No exchange rate needed.
		return nil
	}

	// Stripe does not provide exchange rates. We will use the default rates and should be updated after the invoice is created.
	rate := &currency.ExchangeRate{
		From:   curr,
		To:     regime.Currency,
		Amount: DefaultRate(curr, regime.Currency),
	}

	return []*currency.ExchangeRate{rate}
}
