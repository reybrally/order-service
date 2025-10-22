package validation

import (
	"errors"
	domain "github.com/reybrally/order-service/internal/domain/order"
	"time"
)

func IsValidOrder(order domain.Order) error {
	if err := validateOrderFields(order); err != nil {
		return err
	}
	if err := isValidPayment(order.Payment); err != nil {
		return err
	}
	if err := isValidItems(order.Items); err != nil {
		return err
	}
	if err := isValidDelivery(order.Delivery); err != nil {
		return err
	}
	return nil
}

func isValidDate(date time.Time) bool {
	if date.IsZero() {
		return false
	}
	if date.After(time.Now()) || date.Before(time.Now().AddDate(-1, 0, 0)) {
		return false
	}
	return true
}

func validateOrderFields(order domain.Order) error {
	if order.DeliveryService == "" {
		return errors.New("delivery service is required")
	}
	if order.TrackNumber == "" {
		return errors.New("track number is required")
	}
	if order.Entry == "" {
		return errors.New("entry is required")
	}
	if order.Locale == "" {
		return errors.New("locale is required")
	}
	if !isValidDate(order.DateCreated) {
		return errors.New("invalid date created")
	}
	if order.CustomerId == "" {
		return errors.New("invalid customer id")
	}
	return nil
}
