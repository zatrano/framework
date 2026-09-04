package money

import (
	"fmt"
	"math"
	"strings"
)

// Money is an integer minor-unit amount (e.g. cents) with currency.
type Money struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

// Of creates money from minor units.
func Of(amount int64, currency string) Money {
	return Money{Amount: amount, Currency: strings.ToUpper(strings.TrimSpace(currency))}
}

// FromMajor converts a major-unit float (e.g. 12.34) into minor units.
func FromMajor(major float64, currency string, precision ...int) Money {
	p := 2
	if len(precision) > 0 && precision[0] >= 0 {
		p = precision[0]
	}
	scale := math.Pow10(p)
	return Of(int64(math.Round(major*scale)), currency)
}

// Major returns the major-unit float.
func (m Money) Major(precision ...int) float64 {
	p := 2
	if len(precision) > 0 && precision[0] >= 0 {
		p = precision[0]
	}
	return float64(m.Amount) / math.Pow10(p)
}

// Add returns m + other (same currency).
func (m Money) Add(other Money) (Money, error) {
	if err := m.sameCurrency(other); err != nil {
		return Money{}, err
	}
	return Of(m.Amount+other.Amount, m.Currency), nil
}

// Sub returns m - other.
func (m Money) Sub(other Money) (Money, error) {
	if err := m.sameCurrency(other); err != nil {
		return Money{}, err
	}
	return Of(m.Amount-other.Amount, m.Currency), nil
}

// Mul multiplies by a factor.
func (m Money) Mul(factor float64) Money {
	return Of(int64(math.Round(float64(m.Amount)*factor)), m.Currency)
}

// Allocate splits amount into n parts without losing cents.
func (m Money) Allocate(n int) ([]Money, error) {
	if n <= 0 {
		return nil, fmt.Errorf("money: n must be positive")
	}
	base := m.Amount / int64(n)
	rem := m.Amount % int64(n)
	out := make([]Money, n)
	for i := 0; i < n; i++ {
		amt := base
		if int64(i) < rem {
			amt++
		}
		out[i] = Of(amt, m.Currency)
	}
	return out, nil
}

// Format renders a simple currency string.
func (m Money) Format(symbol string, precision ...int) string {
	p := 2
	if len(precision) > 0 && precision[0] >= 0 {
		p = precision[0]
	}
	if symbol == "" {
		symbol = m.Currency + " "
	}
	return fmt.Sprintf("%s%.*f", symbol, p, m.Major(p))
}

func (m Money) sameCurrency(other Money) error {
	if strings.EqualFold(m.Currency, other.Currency) {
		return nil
	}
	return fmt.Errorf("money: currency mismatch %s vs %s", m.Currency, other.Currency)
}
