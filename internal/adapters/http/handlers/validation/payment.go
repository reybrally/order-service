package validation

import (
	"errors"
	domain "github.com/reybrally/order-service/internal/domain/order"
)

func isValidPayment(payment domain.Payment) error {
	if payment.Transaction == "" {
		return errors.New("transaction payment is required")
	}
	if payment.Amount < 0 {
		return errors.New("amount is negative")
	}
	if payment.Currency == "" {
		return errors.New("currency is required")
	}
	if payment.PaymentDt.IsZero() {
		return errors.New("payment dt is required")
	}
	return nil
}
