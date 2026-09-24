package models

import "github.com/shopspring/decimal"

// Money is stored with fixed-point decimal semantics. Database values use the
// DECIMAL/NUMERIC representation supported by both PostgreSQL and SQLite.
type Money = decimal.Decimal

var ZeroMoney = decimal.Zero

// MoneyFromMinor converts a fiat minor-unit amount (e.g. IDR rupiah) to Money
// without ever passing through a binary float.
func MoneyFromMinor(minor int64) Money {
	return decimal.NewFromInt(minor)
}

// DecimalFromFloat converts a fractional multiplier that is not itself money
// (line-item quantity, tax rate) to exact decimal for arithmetic. Money
// amounts must never round-trip through float64; use MoneyFromMinor instead.
func DecimalFromFloat(value float64) Money {
	return decimal.NewFromFloat(value)
}
