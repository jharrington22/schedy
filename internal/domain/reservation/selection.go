package reservation

import "time"

// ChooseSlotStrict returns the first available slot that matches the preferred times in order.
// Matching is minute-granularity equality on the provided timestamp.
// If PreferredTimes is empty, returns the earliest available slot.
func ChooseSlotStrict(preferred []time.Time, available []Slot) (Slot, bool) {
	if len(available) == 0 {
		return Slot{}, false
	}
	if len(preferred) == 0 {
		best := available[0]
		for _, s := range available[1:] {
			if s.Start.Before(best.Start) {
				best = s
			}
		}
		return best, true
	}

	// Build lookup map from available slots, minute-rounded RFC3339 key.
	m := make(map[string]Slot, len(available))
	for _, s := range available {
		k := s.Start.Truncate(time.Minute).Format(time.RFC3339)
		// Keep earliest if duplicates
		if existing, ok := m[k]; ok {
			if s.Start.Before(existing.Start) {
				m[k] = s
			}
			continue
		}
		m[k] = s
	}
	for _, p := range preferred {
		k := p.Truncate(time.Minute).Format(time.RFC3339)
		if s, ok := m[k]; ok {
			return s, true
		}
	}
	return Slot{}, false
}
