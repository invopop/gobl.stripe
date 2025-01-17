package goblstripe

import (
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
)

// DefaultRates provides a list of default exchange rates based on the same currency (USD).
var DefaultRates = map[currency.Code]num.Amount{
	currency.USD: num.MakeAmount(1, 0),
	currency.EUR: num.MakeAmount(935, 3),
	currency.GBP: num.MakeAmount(788, 3),
	currency.BRL: num.MakeAmount(5379, 3),
	currency.MXN: num.MakeAmount(18467, 3),
	currency.COP: num.MakeAmount(4132934, 3),
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
