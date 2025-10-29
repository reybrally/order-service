package validation

import (
	"errors"
	domain "github.com/reybrally/order-service/internal/domain/order"
	"net/mail"
)

func isValidDelivery(delivery domain.Delivery) error {
	if delivery.Name == "" {
		return errors.New("delivery name is required")
	}
	if delivery.Email == "" {
		return IsValidMail(delivery.Email)
	}
	if delivery.Phone == "" {
		return errors.New("delivery phone is required")
	}
	if delivery.Address == "" {
		return errors.New("delivery address is required")
	}
	if delivery.City == "" {
		return errors.New("delivery city is required")
	}
	if delivery.Region == "" {
		return errors.New("delivery region is required")
	}
	return nil
}

func IsValidMail(mai string) error {
	if mai == "" {
		return errors.New("mail is required")
	}
	_, err := mail.ParseAddress(mai)
	return err
}
