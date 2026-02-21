package usecases

import (
	"context"
	"fmt"

	"github.com/example/resy-scheduler/internal/domain/reservation"
)

type FindAndBook struct {
	Provider reservation.BookingProvider
}

func (u FindAndBook) Execute(ctx context.Context, req reservation.ReservationRequest) (string, error) {
	if u.Provider == nil {
		return "", fmt.Errorf("provider is nil")
	}
	slots, err := u.Provider.FindSlots(ctx, req)
	if err != nil {
		return "", err
	}
	slot, ok := reservation.ChooseSlotStrict(req.PreferredTimes, slots)
	if !ok {
		return "", fmt.Errorf("no matching slots")
	}
	return u.Provider.Book(ctx, req, slot)
}
