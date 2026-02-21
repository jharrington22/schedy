package resy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client is a minimal Resy API client based on the request flow used by lgrees/resy-cli.
// It requires an API key and auth token captured from an authenticated browser session.
//
// References:
// - resy-cli uses these endpoints and headers for ping and booking. citeturn12view1turn13view0turn14view0
type Client struct {
	hc   *http.Client
	creds Credentials
}

type Credentials struct {
	APIKey    string
	AuthToken string
}

func New(creds Credentials) *Client {
	return &Client{
		hc: &http.Client{Timeout: 3 * time.Second},
		creds: creds,
	}
}

func (c *Client) Ping(ctx context.Context) error {
	_, status, body, err := c.do(ctx, http.MethodGet, "https://api.resy.com/2/user", "", nil, nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		// resy-cli prints a helpful message field if present. citeturn12view1
		var r struct{ Message string `json:"message"` }
		_ = json.Unmarshal(body, &r)
		if r.Message != "" {
			return fmt.Errorf("resy ping failed: %s (status=%d)", r.Message, status)
		}
		return fmt.Errorf("resy ping failed (status=%d)", status)
	}
	return nil
}

// Book tries to book a reservation for a specific venue/date/party size.
// preferredTimes are strings like "18:15" or "18:15:00".
// reservationTypes is a comma-separated list (optional), e.g. "Dining Room,Bar".
func (c *Client) Book(ctx context.Context, venueID string, partySize int, reservationDate time.Time, preferredTimes []string, reservationTypes string) error {
	bd := bookingDetails{
		VenueID:          venueID,
		PartySize:        partySize,
		ReservationDate:  reservationDate.Format("2006-01-02"),
		ReservationTimes: normalizeTimes(preferredTimes),
		ReservationTypes: splitCSV(reservationTypes),
	}

	slots, err := c.fetchSlots(ctx, bd)
	if err != nil {
		return err
	}
	matching := findMatches(bd, slots)
	if len(matching) == 0 {
		return errors.New("no matching slots")
	}
	for _, slot := range matching {
		if err := c.bookSlot(ctx, bd, slot); err == nil {
			return nil
		}
	}
	return errors.New("could not book any matching slots")
}

// --- internals (ported from resy-cli flow) ---

type bookingDetails struct {
	VenueID          string
	PartySize        int
	ReservationDate  string // YYYY-MM-DD
	ReservationTimes []string
	ReservationTypes []string
}

type slot struct {
	Date struct {
		Start string `json:"start"`
	} `json:"date"`
	Config struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	} `json:"config"`
}

type slots []slot

type findResponse struct {
	Results struct {
		Venues []struct {
			Slots slots `json:"slots"`
		} `json:"venues"`
	} `json:"results"`
}

type bookingConfig struct {
	ConfigId   string `json:"config_id"`
	Day        string `json:"day"`
	PartySize  int64  `json:"party_size"`
}

type detailsResponse struct {
	BookToken struct {
		Value string `json:"value"`
	} `json:"book_token"`
	User struct {
		PaymentMethods []struct {
			ID int64 `json:"id"`
		} `json:"payment_methods"`
	} `json:"user"`
}

func (c *Client) fetchSlots(ctx context.Context, bd bookingDetails) (slots, error) {
	params := map[string]string{
		"party_size": strconv.Itoa(bd.PartySize),
		"venue_id":   bd.VenueID,
		"day":        bd.ReservationDate,
		// resy-cli includes these (deprecated but seemingly required). citeturn13view0
		"lat":  "0",
		"long": "0",
	}
	_, status, body, err := c.do(ctx, http.MethodGet, "https://api.resy.com/4/find", "", params, nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("failed to fetch slots (status=%d)", status)
	}
	var res findResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	if len(res.Results.Venues) == 0 || len(res.Results.Venues[0].Slots) == 0 {
		return nil, errors.New("no slots for date")
	}
	return res.Results.Venues[0].Slots, nil
}

func findMatches(bd bookingDetails, ss slots) (matches slots) {
	for _, s := range ss {
		if isSlotMatch(bd, s) {
			matches = append(matches, s)
		}
	}
	return matches
}

func isSlotMatch(bd bookingDetails, s slot) bool {
	pieces := strings.Split(s.Date.Start, " ")
	if len(pieces) < 2 {
		return false
	}
	slotTime := pieces[1]
	slotType := strings.ToLower(s.Config.Type)

	isTypeMatch := len(bd.ReservationTypes) == 0
	isTimeMatch := false
	for _, t := range bd.ReservationTimes {
		if t == slotTime {
			isTimeMatch = true
			break
		}
	}
	for _, rt := range bd.ReservationTypes {
		if strings.ToLower(rt) == slotType {
			isTypeMatch = true
			break
		}
	}
	return isTimeMatch && isTypeMatch
}

func (c *Client) bookSlot(ctx context.Context, bd bookingDetails, s slot) error {
	bc := bookingConfig{ConfigId: s.Config.Token, Day: bd.ReservationDate, PartySize: int64(bd.PartySize)}
	jb, err := json.Marshal(bc)
	if err != nil {
		return err
	}
	_, status, body, err := c.do(ctx, http.MethodPost, "https://api.resy.com/3/details", "application/json", nil, jb)
	if err != nil {
		return err
	}
	if status >= 400 || body == nil {
		return fmt.Errorf("failed to get booking details (status=%d)", status)
	}
	var details detailsResponse
	_ = json.Unmarshal(body, &details)

	// book
	token := fmt.Sprintf("book_token=%s", url.PathEscape(details.BookToken.Value))
	var paymentDetails string
	if details.User.PaymentMethods != nil && len(details.User.PaymentMethods) != 0 {
		pb, _ := json.Marshal(struct {
			ID int64 `json:"id"`
		}{ID: details.User.PaymentMethods[0].ID})
		paymentDetails = fmt.Sprintf("struct_payment_method=%s", url.PathEscape(string(pb)))
	}
	form := token
	if paymentDetails != "" {
		form = strings.Join([]string{token, paymentDetails}, "&")
	}
	_, status, _, err = c.do(ctx, http.MethodPost, "https://api.resy.com/3/book", "application/x-www-form-urlencoded", nil, []byte(form))
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("failed to book reservation (status=%d)", status)
	}
	return nil
}

func (c *Client) do(ctx context.Context, method, rawURL, contentType string, query map[string]string, body []byte) (*http.Response, int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, bytes.NewReader(body))
	if err != nil {
		return nil, 0, nil, err
	}
	// resy-cli sets these headers. citeturn14view0
	req.Header.Add("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36")
	req.Header.Add("origin", "https://resy.com")
	req.Header.Add("referrer", "https://resy.com")
	req.Header.Add("x-origin", "https://resy.com")
	req.Header.Add("cache-control", "no-cache")
	if contentType != "" {
		req.Header.Add("content-type", contentType)
	}
	req.Header.Add("authorization", fmt.Sprintf(`ResyAPI api_key="%s"`, c.creds.APIKey))
	req.Header.Add("x-resy-auth-token", c.creds.AuthToken)
	req.Header.Add("x-resy-universal-auth", c.creds.AuthToken)

	if query != nil {
		q := req.URL.Query()
		for k, v := range query {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	res, err := c.hc.Do(req)
	if err != nil {
		return nil, 500, nil, err
	}
	if res == nil {
		return nil, 0, nil, nil
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return res, res.StatusCode, nil, err
	}
	return res, res.StatusCode, b, nil
}

func normalizeTimes(in []string) []string {
	out := make([]string, 0, len(in))
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if len(t) == 5 && strings.Count(t, ":") == 1 {
			t = t + ":00"
		}
		out = append(out, t)
	}
	return out
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
