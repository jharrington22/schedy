package resy

import (
	"context"
	"errors"

	"github.com/example/resy-scheduler/internal/domain/reservation"
	"github.com/example/resy-scheduler/internal/infrastructure/config"
)

type Provider struct{ cfg config.Config }

func New(cfg config.Config) *Provider { return &Provider{cfg: cfg} }
func (p *Provider) Name() string { return "resy" }

func (p *Provider) Ping(ctx context.Context) error {
	return errors.New("resy provider not implemented in this refactor tarball yet")
}
func (p *Provider) FindSlots(ctx context.Context, req reservation.ReservationRequest) ([]reservation.Slot, error) {
	return nil, errors.New("resy provider not implemented in this refactor tarball yet")
}
func (p *Provider) Book(ctx context.Context, req reservation.ReservationRequest, slot reservation.Slot) (string, error) {
	return "", errors.New("resy provider not implemented in this refactor tarball yet")
}
