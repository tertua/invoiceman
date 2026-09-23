package models

import "github.com/shopspring/decimal"

// Money is stored with fixed-point decimal semantics. Database values use the
// DECIMAL/NUMERIC representation supported by both PostgreSQL and SQLite.
type Money = decimal.Decimal

var ZeroMoney = decimal.Zero

func MoneyFromFloat(value float64) Money {
	return decimal.NewFromFloat(value)
}
