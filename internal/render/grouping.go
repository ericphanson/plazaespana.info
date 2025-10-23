package render

import (
	"sort"
	"time"

	"github.com/ericphanson/madrid-events/internal/event"
	"github.com/ericphanson/madrid-events/internal/filter"
)

// TimeGroup represents a group of events within a time range.
type TimeGroup struct {
	Name      string // e.g., "Past Weekend", "Happening Now / Today"
	Icon      string // emoji icon for the group
	Events    []TemplateEvent
	CityCount int // Count of city events (visible by default)

	// Distance-filtered counts (for dynamic count display)
	CountPlaza  int // Events at Plaza de España (En Plaza filter)
	CountNearby int // All nearby events (default filter, same as len(Events))
	CityPlaza   int // City events at Plaza
	CityNearby  int // All city events (same as CityCount)
}

// incrementDistanceCounts updates the distance-filtered counts for a time group.
func (g *TimeGroup) incrementDistanceCounts(evt TemplateEvent, isCityEvent bool) {
	// All events count toward "Nearby"
	g.CountNearby++
	if isCityEvent {
		g.CityNearby++
	}

	// Only events at Plaza de España count toward "En Plaza"
	if evt.AtPlaza {
		g.CountPlaza++
		if isCityEvent {
			g.CityPlaza++
		}
	}
}

// GroupedTemplateData extends TemplateData with time-grouped events.
type GroupedTemplateData struct {
	Lang                string
	CSSHash             string
	LastUpdated         string
	TotalEvents         int
	TotalCityEvents     int
	TotalCulturalEvents int
	ShowCulturalDefault bool // Whether cultural events should be shown by default
	Groups              []TimeGroup
	OngoingEvents       []TemplateEvent
	OngoingCityCount    int // Count of city events in ongoing section

	// Distance-filtered counts for ongoing events
	OngoingPlaza      int
	OngoingNearby     int
	OngoingCityPlaza  int
	OngoingCityNearby int
}

// GroupEventsByTime groups events into time-based buckets relative to now.
// Events are filtered to show only:
// - Past: From most recent Saturday through now
// - Future: Up to 30 days from now
//
// Time groups:
// - Past Weekend: Most recent Sat-Sun
// - Happening Now / Today: Current day
// - This Weekend: Upcoming/current Fri-Sun
// - This Week: Next 7 days
// - Later This Month: Rest of current calendar month
// - Ongoing: Events lasting 5+ days (returned separately)
func GroupEventsByTime(events []event.CulturalEvent, now time.Time) (groups []TimeGroup, ongoing []TemplateEvent) {
	// Define time boundaries
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfToday := startOfToday.Add(24 * time.Hour)

	// Past Weekend: Most recent Sat-Sun that has already passed
	// If today is Mon-Fri: last Sat-Sun
	// If today is Sat-Sun: previous Sat-Sun (not current one)
	var pastWeekendStart, pastWeekendEnd time.Time
	if now.Weekday() == time.Saturday {
		// Today is Saturday - "past weekend" is 7 days ago (last Sat-Sun)
		pastWeekendStart = startOfToday.AddDate(0, 0, -7)
		pastWeekendEnd = pastWeekendStart.Add(48 * time.Hour)
	} else if now.Weekday() == time.Sunday {
		// Today is Sunday - "past weekend" is 8 days ago for Sat (last Sat-Sun)
		pastWeekendStart = startOfToday.AddDate(0, 0, -8)
		pastWeekendEnd = pastWeekendStart.Add(48 * time.Hour)
	} else {
		// Mon-Fri: "past weekend" is most recent Sat-Sun
		daysToLastSunday := int(now.Weekday())                             // Mon=1, Tue=2, etc.
		pastWeekendStart = startOfToday.AddDate(0, 0, -daysToLastSunday-1) // Go to last Saturday
		pastWeekendEnd = pastWeekendStart.Add(48 * time.Hour)              // Sat + Sun
	}

	// This weekend: next or current Fri-Sun
	var thisWeekendStart, thisWeekendEnd time.Time
	if now.Weekday() >= time.Friday {
		// If today is Fri/Sat/Sun, "this weekend" is the current Fri-Sun
		daysToFriday := int(time.Friday - now.Weekday())
		if daysToFriday < 0 {
			daysToFriday += 7
		}
		thisWeekendStart = startOfToday.AddDate(0, 0, -(int(now.Weekday()) - int(time.Friday)))
		thisWeekendEnd = thisWeekendStart.Add(72 * time.Hour) // Fri + Sat + Sun
	} else {
		// Mon-Thu: "this weekend" is upcoming Fri-Sun
		daysToFriday := int(time.Friday - now.Weekday())
		thisWeekendStart = startOfToday.AddDate(0, 0, daysToFriday)
		thisWeekendEnd = thisWeekendStart.Add(72 * time.Hour)
	}

	// This week: next 7 days from now
	thisWeekEnd := startOfToday.AddDate(0, 0, 7)

	// Later this month: rest of current calendar month
	endOfMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())

	// Future limit: 30 days from now
	futureLimit := now.AddDate(0, 0, 30)

	// Initialize groups
	pastWeekend := TimeGroup{Name: "Past Weekend", Icon: "📅", Events: []TemplateEvent{}}
	happeningNow := TimeGroup{Name: "Happening Now / Today", Icon: "⏰", Events: []TemplateEvent{}}
	thisWeekend := TimeGroup{Name: "This Weekend", Icon: "🎉", Events: []TemplateEvent{}}
	thisWeek := TimeGroup{Name: "This Week", Icon: "📆", Events: []TemplateEvent{}}
	laterThisMonth := TimeGroup{Name: "Later This Month", Icon: "📅", Events: []TemplateEvent{}}
	ongoingEvents := []TemplateEvent{}

	// Hard filter: Don't show events older than 60 days to prevent stale data
	oldEventCutoff := now.AddDate(0, 0, -60)

	// Filter and group events
	for _, evt := range events {
		// Calculate event duration
		endTime := evt.EndTime
		if endTime.IsZero() {
			endTime = evt.StartTime.Add(2 * time.Hour) // Assume 2-hour duration if no end time
		}
		duration := endTime.Sub(evt.StartTime)

		// Skip events that ended before the past weekend
		if endTime.Before(pastWeekendStart) {
			continue
		}

		// Skip very old events (data quality filter)
		if evt.StartTime.Before(oldEventCutoff) {
			continue
		}

		// Skip events that start after the future limit
		if evt.StartTime.After(futureLimit) {
			continue
		}

		// Convert to template event
		// Format time: only show HH:MM if not midnight (00:00 likely means "no time specified")
		timeFormat := "02/01/2006"
		if evt.StartTime.Hour() != 0 || evt.StartTime.Minute() != 0 {
			timeFormat = "02/01/2006 15:04"
		}

		templateEvt := TemplateEvent{
			IDEvento:          evt.ID,
			Titulo:            evt.Title,
			StartHuman:        evt.StartTime.Format(timeFormat),
			StartTime:         evt.StartTime,
			NombreInstalacion: evt.VenueName,
			ContentURL:        evt.DetailsURL,
			Description:       TruncateText(evt.Description, 150),
			EventType:         "cultural", // Default for this function
		}

		// Classify: ongoing events (5+ days) go to separate section
		if duration >= 5*24*time.Hour {
			ongoingEvents = append(ongoingEvents, templateEvt)
			continue
		}

		// Otherwise, assign to time groups (can appear in multiple)
		added := false

		// Past Weekend
		if evt.StartTime.Before(pastWeekendEnd) && endTime.After(pastWeekendStart) {
			pastWeekend.Events = append(pastWeekend.Events, templateEvt)
			added = true
		}

		// Happening Now / Today
		if evt.StartTime.Before(endOfToday) && endTime.After(startOfToday) {
			happeningNow.Events = append(happeningNow.Events, templateEvt)
			added = true
		}

		// This Weekend
		if evt.StartTime.Before(thisWeekendEnd) && evt.StartTime.After(thisWeekendStart) {
			thisWeekend.Events = append(thisWeekend.Events, templateEvt)
			added = true
		}

		// This Week (but not already in "This Weekend" or "Today")
		if !added && evt.StartTime.Before(thisWeekEnd) && evt.StartTime.After(endOfToday) {
			thisWeek.Events = append(thisWeek.Events, templateEvt)
			added = true
		}

		// Later This Month
		if !added && evt.StartTime.Before(endOfMonth) && evt.StartTime.After(thisWeekEnd) {
			laterThisMonth.Events = append(laterThisMonth.Events, templateEvt)
		}
	}

	// Build groups list (only include non-empty groups)
	groups = []TimeGroup{}
	if len(pastWeekend.Events) > 0 {
		groups = append(groups, pastWeekend)
	}
	if len(happeningNow.Events) > 0 {
		groups = append(groups, happeningNow)
	}
	if len(thisWeekend.Events) > 0 {
		groups = append(groups, thisWeekend)
	}
	if len(thisWeek.Events) > 0 {
		groups = append(groups, thisWeek)
	}
	if len(laterThisMonth.Events) > 0 {
		groups = append(groups, laterThisMonth)
	}

	return groups, ongoingEvents
}

// GroupCityEventsByTime groups city events into time-based buckets relative to now.
// Similar to GroupEventsByTime but for CityEvent type.
func GroupCityEventsByTime(events []event.CityEvent, now time.Time) (groups []TimeGroup, ongoing []TemplateEvent) {
	// Convert CityEvents to CulturalEvents for reuse of grouping logic
	culturalEvents := make([]event.CulturalEvent, len(events))
	for i, evt := range events {
		culturalEvents[i] = event.CulturalEvent{
			ID:          evt.ID,
			Title:       evt.Title,
			Description: evt.Description,
			StartTime:   evt.StartDate,
			EndTime:     evt.EndDate,
			VenueName:   evt.Venue,
			DetailsURL:  evt.WebURL,
		}
	}

	return GroupEventsByTime(culturalEvents, now)
}

// GroupMixedEventsByTime groups both city and cultural events into time-based buckets.
// Events are merged and sorted chronologically (city events first on ties).
// Cultural events are marked with EventType="cultural" for CSS filtering.
// Returns groups, ongoing events, and counts for ongoing section.
// Calculates and formats distance from reference point (typically Plaza de España).
func GroupMixedEventsByTime(cityEvents []event.CityEvent, culturalEvents []event.CulturalEvent, now time.Time, refLat, refLon float64) (groups []TimeGroup, ongoing []TemplateEvent, ongoingCityCount, ongoingPlaza, ongoingNearby, ongoingCityPlaza, ongoingCityNearby int) {
	// Convert both types to a common internal type with metadata
	type eventWithType struct {
		evt       event.CulturalEvent
		eventType string // "city" or "cultural"
	}

	// Combine both event lists
	allEvents := make([]eventWithType, 0, len(cityEvents)+len(culturalEvents))

	for _, evt := range cityEvents {
		allEvents = append(allEvents, eventWithType{
			evt: event.CulturalEvent{
				ID:          evt.ID,
				Title:       evt.Title,
				Description: evt.Description,
				StartTime:   evt.StartDate,
				EndTime:     evt.EndDate,
				Latitude:    evt.Latitude,
				Longitude:   evt.Longitude,
				VenueName:   evt.Venue,
				DetailsURL:  evt.WebURL,
			},
			eventType: "city",
		})
	}

	for _, evt := range culturalEvents {
		allEvents = append(allEvents, eventWithType{
			evt:       evt,
			eventType: "cultural",
		})
	}

	// Sort: chronological order, city first on ties
	sort.Slice(allEvents, func(i, j int) bool {
		if allEvents[i].evt.StartTime.Equal(allEvents[j].evt.StartTime) {
			// Tie: city events come first
			return allEvents[i].eventType == "city" && allEvents[j].eventType == "cultural"
		}
		return allEvents[i].evt.StartTime.Before(allEvents[j].evt.StartTime)
	})

	// Use the same time grouping logic
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfToday := startOfToday.Add(24 * time.Hour)

	// Past Weekend calculation (same as GroupEventsByTime)
	var pastWeekendStart, pastWeekendEnd time.Time
	if now.Weekday() == time.Saturday {
		pastWeekendStart = startOfToday.AddDate(0, 0, -7)
		pastWeekendEnd = pastWeekendStart.Add(48 * time.Hour)
	} else if now.Weekday() == time.Sunday {
		pastWeekendStart = startOfToday.AddDate(0, 0, -8)
		pastWeekendEnd = pastWeekendStart.Add(48 * time.Hour)
	} else {
		daysToLastSunday := int(now.Weekday())
		pastWeekendStart = startOfToday.AddDate(0, 0, -daysToLastSunday-1)
		pastWeekendEnd = pastWeekendStart.Add(48 * time.Hour)
	}

	// This weekend calculation
	var thisWeekendStart, thisWeekendEnd time.Time
	if now.Weekday() >= time.Friday {
		thisWeekendStart = startOfToday.AddDate(0, 0, -(int(now.Weekday()) - int(time.Friday)))
		thisWeekendEnd = thisWeekendStart.Add(72 * time.Hour)
	} else {
		daysToFriday := int(time.Friday - now.Weekday())
		thisWeekendStart = startOfToday.AddDate(0, 0, daysToFriday)
		thisWeekendEnd = thisWeekendStart.Add(72 * time.Hour)
	}

	thisWeekEnd := startOfToday.AddDate(0, 0, 7)
	endOfMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	futureLimit := now.AddDate(0, 0, 30)
	oldEventCutoff := now.AddDate(0, 0, -60)

	// Initialize groups
	pastWeekend := TimeGroup{Name: "Past Weekend", Icon: "📅", Events: []TemplateEvent{}}
	happeningNow := TimeGroup{Name: "Happening Now / Today", Icon: "⏰", Events: []TemplateEvent{}}
	thisWeekend := TimeGroup{Name: "This Weekend", Icon: "🎉", Events: []TemplateEvent{}}
	thisWeek := TimeGroup{Name: "This Week", Icon: "📆", Events: []TemplateEvent{}}
	laterThisMonth := TimeGroup{Name: "Later This Month", Icon: "📅", Events: []TemplateEvent{}}
	ongoingEvents := []TemplateEvent{}
	ongoingCityCount = 0

	// Group events
	for _, ewt := range allEvents {
		evt := ewt.evt

		// Calculate duration
		endTime := evt.EndTime
		if endTime.IsZero() {
			endTime = evt.StartTime.Add(2 * time.Hour)
		}
		duration := endTime.Sub(evt.StartTime)

		// Skip old/future events
		if endTime.Before(pastWeekendStart) || evt.StartTime.Before(oldEventCutoff) || evt.StartTime.After(futureLimit) {
			continue
		}

		// Format time
		timeFormat := "02/01/2006"
		if evt.StartTime.Hour() != 0 || evt.StartTime.Minute() != 0 {
			timeFormat = "02/01/2006 15:04"
		}

		// Calculate distance from reference point (only for valid coordinates)
		var distanceStr string
		var distanceMeters int
		var atPlaza bool
		if evt.Latitude != 0 || evt.Longitude != 0 {
			distanceKm := filter.HaversineDistance(refLat, refLon, evt.Latitude, evt.Longitude)
			distanceStr = FormatDistance(distanceKm)
			distanceMeters = int(distanceKm * 1000) // Convert to meters

			// Check if event is at Plaza de España (distance ~0m OR text mentions it)
			if distanceMeters <= 50 || filter.MatchesPlazaEspana(evt.Title, evt.VenueName, evt.Address, evt.Description) {
				atPlaza = true
			}
		} else {
			// No coordinates - check all text fields for Plaza de España mentions
			atPlaza = filter.MatchesPlazaEspana(evt.Title, evt.VenueName, evt.Address, evt.Description)
			if atPlaza {
				distanceMeters = 0 // Treat as 0 meters if venue name matches
			}
		}

		// Convert to template event
		templateEvt := TemplateEvent{
			IDEvento:          evt.ID,
			Titulo:            evt.Title,
			StartHuman:        evt.StartTime.Format(timeFormat),
			StartTime:         evt.StartTime,
			NombreInstalacion: evt.VenueName,
			ContentURL:        evt.DetailsURL,
			Description:       TruncateText(evt.Description, 150),
			EventType:         ewt.eventType,
			DistanceHuman:     distanceStr,
			DistanceMeters:    distanceMeters,
			AtPlaza:           atPlaza,
		}

		// Ongoing events (5+ days)
		if duration >= 5*24*time.Hour {
			ongoingEvents = append(ongoingEvents, templateEvt)
			isCityEvent := ewt.eventType == "city"
			if isCityEvent {
				ongoingCityCount++
			}

			// Track distance-filtered counts for ongoing events
			ongoingNearby++
			if isCityEvent {
				ongoingCityNearby++
			}
			if templateEvt.AtPlaza {
				ongoingPlaza++
				if isCityEvent {
					ongoingCityPlaza++
				}
			}

			continue
		}

		// Assign to time groups
		added := false
		isCityEvent := ewt.eventType == "city"

		if evt.StartTime.Before(pastWeekendEnd) && endTime.After(pastWeekendStart) {
			pastWeekend.Events = append(pastWeekend.Events, templateEvt)
			pastWeekend.incrementDistanceCounts(templateEvt, isCityEvent)
			if isCityEvent {
				pastWeekend.CityCount++
			}
			added = true
		}

		if evt.StartTime.Before(endOfToday) && endTime.After(startOfToday) {
			happeningNow.Events = append(happeningNow.Events, templateEvt)
			happeningNow.incrementDistanceCounts(templateEvt, isCityEvent)
			if isCityEvent {
				happeningNow.CityCount++
			}
			added = true
		}

		if evt.StartTime.Before(thisWeekendEnd) && evt.StartTime.After(thisWeekendStart) {
			thisWeekend.Events = append(thisWeekend.Events, templateEvt)
			thisWeekend.incrementDistanceCounts(templateEvt, isCityEvent)
			if isCityEvent {
				thisWeekend.CityCount++
			}
			added = true
		}

		if !added && evt.StartTime.Before(thisWeekEnd) && evt.StartTime.After(endOfToday) {
			thisWeek.Events = append(thisWeek.Events, templateEvt)
			thisWeek.incrementDistanceCounts(templateEvt, isCityEvent)
			if isCityEvent {
				thisWeek.CityCount++
			}
			added = true
		}

		if !added && evt.StartTime.Before(endOfMonth) && evt.StartTime.After(thisWeekEnd) {
			laterThisMonth.Events = append(laterThisMonth.Events, templateEvt)
			laterThisMonth.incrementDistanceCounts(templateEvt, isCityEvent)
			if isCityEvent {
				laterThisMonth.CityCount++
			}
		}
	}

	// Build groups list (always include non-empty groups, even if all events are cultural/hidden)
	groups = []TimeGroup{}
	if len(pastWeekend.Events) > 0 {
		groups = append(groups, pastWeekend)
	}
	if len(happeningNow.Events) > 0 {
		groups = append(groups, happeningNow)
	}
	if len(thisWeekend.Events) > 0 {
		groups = append(groups, thisWeekend)
	}
	if len(thisWeek.Events) > 0 {
		groups = append(groups, thisWeek)
	}
	if len(laterThisMonth.Events) > 0 {
		groups = append(groups, laterThisMonth)
	}

	return groups, ongoingEvents, ongoingCityCount, ongoingPlaza, ongoingNearby, ongoingCityPlaza, ongoingCityNearby
}
