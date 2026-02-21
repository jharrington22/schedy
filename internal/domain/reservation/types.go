package reservation

import "time"

type Platform string

const (
	PlatformResy      Platform = "resy"
	PlatformOpenTable Platform = "opentable"
)

type ReservationJob struct {
	ID        string
	Platform  Platform
	VenueID   string
	Date      time.Time
	PartySize int

	// Preferred times in restaurant-local time on Date. Strict ordering: earlier entries win.
	PreferredTimes []time.Time

	// Attempt window (inclusive start, exclusive end)
	WindowStart time.Time
	WindowEnd   time.Time
}
