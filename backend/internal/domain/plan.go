package domain

import (
	"time"

	"github.com/google/uuid"
)

type DailyVerse struct {
	DayNumber   int    `json:"day"`                   // Day number within the plan (1-based)
	Reference   string `json:"reference"`             // e.g., "John 3:16-18"
	Text        string `json:"text"`                  // The actual verse text (provided by LLM)
	Explanation string `json:"explanation,omitempty"` // Optional brief explanation for the niece
}

type ReadingPlan struct {
	ID             uuid.UUID    `json:"id"`
	Topic          string       `json:"topic"`
	DurationDays   int          `json:"duration_days"`
	TargetAudience string       `json:"target_audience"` // Store the audience for context
	CreatedAt      time.Time    `json:"created_at"`
	DailyVerses    []DailyVerse `json:"daily_verses"` // Ordered list of verses for the plan
}

// Helper to get verse for a specific day (1-based index)
func (p *ReadingPlan) GetVerseForDay(day int) (DailyVerse, bool) {
	if day < 1 || day > len(p.DailyVerses) {
		return DailyVerse{}, false
	}
	// Assuming DailyVerses is sorted by DayNumber or index matches DayNumber-1
	// Let's rely on index matching DayNumber-1 for simplicity here
	if day-1 >= 0 && day-1 < len(p.DailyVerses) {
		// Double check the DayNumber matches, paranoia is good
		if p.DailyVerses[day-1].DayNumber == day {
			return p.DailyVerses[day-1], true
		}
		// Fallback: search if DayNumbers are not contiguous/ordered (less efficient)
		for _, dv := range p.DailyVerses {
			if dv.DayNumber == day {
				return dv, true
			}
		}
	}
	// If we reach here, something is wrong with the data or the requested day
	return DailyVerse{}, false
}
