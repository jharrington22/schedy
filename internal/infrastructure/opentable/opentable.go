package opentable

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/example/resy-scheduler/internal/domain/reservation"
	"github.com/example/resy-scheduler/internal/infrastructure/config"
)

const defaultBaseURL = "https://www.opentable.com/dapi"
const defaultUA = "Mozilla/5.0 (X11; Linux x86_64) resy-scheduler/1.0"
const defaultPersistedQuerySHA256 = "e6b87083b2dfc66e11d26f9bd6e98b8f6a9f4a3b7d0e9a2f33c9f1f6a0b9f2a1"

type Provider struct {
	http *http.Client
	cfg  config.Config

	base string
	ua   string
	hash string
}

func New(cfg config.Config) *Provider {
	base := defaultBaseURL
	ua := defaultUA
	hash := defaultPersistedQuerySHA256
	if strings.TrimSpace(cfg.OpenTablePersistedQuerySHA256) != "" {
		hash = cfg.OpenTablePersistedQuerySHA256
	}
	return &Provider{
		http: &http.Client{Timeout: 20 * time.Second},
		cfg:  cfg,
		base: strings.TrimRight(base, "/"),
		ua:   ua,
		hash: hash,
	}
}

func (p *Provider) Name() string { return "opentable" }

func (p *Provider) Ping(ctx context.Context) error {
	if strings.TrimSpace(p.cfg.OpenTableToken) == "" {
		return errors.New("OPENTABLE_TOKEN is empty")
	}
	return nil
}

func (p *Provider) FindSlots(ctx context.Context, req reservation.ReservationRequest) ([]reservation.Slot, error) {
	if err := p.Ping(ctx); err != nil {
		return nil, err
	}
	if req.VenueID == "" {
		return nil, errors.New("VenueID (restaurantId) is required")
	}
	if req.PartySize <= 0 {
		return nil, errors.New("PartySize must be > 0")
	}

	dateStr := req.Date.Format("2006-01-02")

	payload := map[string]any{
		"operationName": "RestaurantsAvailability",
		"variables": map[string]any{
			"restaurantIds": []string{req.VenueID},
			"partySize":     req.PartySize,
			"dateTime":      dateStr + "T19:00:00.000",
			"forwardDays":   1,
			"includeOffers": true,
		},
		"extensions": map[string]any{
			"persistedQuery": map[string]any{
				"version":    1,
				"sha256Hash": p.hash,
			},
		},
	}

	b, _ := json.Marshal(payload)
	url := p.base + "/fe/gql?optype=query&opname=RestaurantsAvailability"
	hreq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	hreq.Header.Set("content-type", "application/json")
	hreq.Header.Set("user-agent", p.ua)
	hreq.Header.Set("x-csrf-token", p.cfg.OpenTableToken)

	hresp, err := p.http.Do(hreq)
	if err != nil {
		return nil, err
	}
	defer hresp.Body.Close()

	body, _ := io.ReadAll(hresp.Body)
	if hresp.StatusCode < 200 || hresp.StatusCode >= 300 {
		return nil, fmt.Errorf("opentable availability http %d: %s", hresp.StatusCode, string(body))
	}

	var parsed struct {
		Data struct {
			Availability []struct {
				AvailabilityDays []struct {
					Slots []struct {
						IsAvailable           bool   `json:"isAvailable"`
						ReservationDateTime   string `json:"reservationDateTime"`
						SlotAvailabilityToken string `json:"slotAvailabilityToken"`
						SlotHash              string `json:"slotHash"`
					} `json:"slots"`
				} `json:"availabilityDays"`
			} `json:"availability"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("opentable parse availability: %w", err)
	}

	out := make([]reservation.Slot, 0, 16)
	for _, a := range parsed.Data.Availability {
		for _, d := range a.AvailabilityDays {
			for _, s := range d.Slots {
				if !s.IsAvailable {
					continue
				}
				t, err := time.Parse(time.RFC3339Nano, s.ReservationDateTime)
				if err != nil {
					t, err = time.Parse(time.RFC3339, s.ReservationDateTime)
					if err != nil {
						continue
					}
				}
				out = append(out, reservation.Slot{
					Start: t,
					Meta: map[string]string{
						"slotAvailabilityToken": s.SlotAvailabilityToken,
						"slotHash":              s.SlotHash,
					},
				})
			}
		}
	}
	return out, nil
}

func (p *Provider) Book(ctx context.Context, req reservation.ReservationRequest, slot reservation.Slot) (string, error) {
	if err := p.Ping(ctx); err != nil {
		return "", err
	}
	tok := slot.Meta["slotAvailabilityToken"]
	hash := slot.Meta["slotHash"]
	if tok == "" || hash == "" {
		return "", errors.New("slot missing slotAvailabilityToken/slotHash")
	}

	first := req.FirstName
	last := req.LastName
	email := req.Email
	phone := req.Phone
	if first == "" { first = p.cfg.FirstName }
	if last == "" { last = p.cfg.LastName }
	if email == "" { email = p.cfg.Email }
	if phone == "" { phone = p.cfg.Phone }
	if first == "" || last == "" || email == "" || phone == "" {
		return "", errors.New("missing contact info; set BOOKING_* env vars or pass per-request")
	}

	payload := map[string]any{
		"restaurantId":           req.VenueID,
		"partySize":              req.PartySize,
		"reservationDateTime":    slot.Start.Format(time.RFC3339),
		"slotAvailabilityToken":  tok,
		"slotHash":               hash,
		"firstName":              first,
		"lastName":               last,
		"email":                  email,
		"phoneNumber":            phone,
	}
	b, _ := json.Marshal(payload)

	url := p.base + "/booking/make-reservation"
	hreq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	hreq.Header.Set("content-type", "application/json")
	hreq.Header.Set("user-agent", p.ua)
	hreq.Header.Set("x-csrf-token", p.cfg.OpenTableToken)

	hresp, err := p.http.Do(hreq)
	if err != nil {
		return "", err
	}
	defer hresp.Body.Close()

	respBody, _ := io.ReadAll(hresp.Body)
	if hresp.StatusCode < 200 || hresp.StatusCode >= 300 {
		return "", fmt.Errorf("opentable book http %d: %s", hresp.StatusCode, string(respBody))
	}
	return string(respBody), nil
}
