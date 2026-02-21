package usecases

import (
	"context"
	"fmt"

	"github.com/example/resy-scheduler/internal/domain/reservation"
)

type PingProvider struct {
	Provider reservation.BookingProvider
}

func (u PingProvider) Execute(ctx context.Context) error {
	if u.Provider == nil {
		return fmt.Errorf("provider is nil")
	}
	return u.Provider.Ping(ctx)
}
