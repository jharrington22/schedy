package reservation

import (
	"context"
	"time"
)

type Slot struct {
	Start time.Time
	Meta  map[string]string
}

type ReservationRequest struct {
	VenueID   string
	Date      time.Time
	PartySize int
	PreferredTimes []time.Time

	FirstName string
	LastName  string
	Email     string
	Phone     string
}

type BookingProvider interface {
	Name() string
	Ping(ctx context.Context) error
	FindSlots(ctx context.Context, req ReservationRequest) ([]Slot, error)
	Book(ctx context.Context, req ReservationRequest, slot Slot) (string, error)
}
