package validation

import (
	"errors"
	domain "github.com/reybrally/order-service/internal/domain/order"
)

func isValidItems(items []domain.Item) error {
	if len(items) == 0 {
		return errors.New("items must not be empty")
	}
	for _, item := range items {
		err := isValidItem(item)
		if err != nil {
			return err
		}
	}
	return nil
}
func isValidItem(item domain.Item) error {
	if item.ChrtId == "" {
		return errors.New("chrt id must not be empty")
	}
	if item.Name == "" {
		return errors.New("name must not be empty")
	}
	if item.Price <= 0 {
		return errors.New("price must be more than zero")
	}
	if item.TotalPrice <= 0 {
		return errors.New("total price must be more than zero")
	}
	if item.Brand == "" {
		return errors.New("brand must not be empty")
	}
	if item.NmId == "" {
		return errors.New("nm_id must not be empty")
	}
	if item.TrackNumber == "" {
		return errors.New("track number must not be empty")
	}
	return nil
}
