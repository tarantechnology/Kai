package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const googleCalendarEventsURL = "https://www.googleapis.com/calendar/v3/calendars/primary/events"

type googleCalendarListResponse struct {
	Items []googleCalendarEvent `json:"items"`
}

type googleCalendarEvent struct {
	ID      string                 `json:"id"`
	Status  string                 `json:"status"`
	Summary string                 `json:"summary"`
	Start   googleCalendarDateTime `json:"start"`
	End     googleCalendarDateTime `json:"end"`
}

type googleCalendarDateTime struct {
	Date     string `json:"date"`
	DateTime string `json:"dateTime"`
	TimeZone string `json:"timeZone"`
}

type googleCalendarCreateRequest struct {
	Title      string `json:"title"`
	StartLabel string `json:"startLabel"`
	EndLabel   string `json:"endLabel"`
}

type frontendEvent struct {
	ID         string `json:"id"`
	ExternalID string `json:"externalId,omitempty"`
	Title      string `json:"title"`
	StartAt    string `json:"startAt"`
	EndAt      string `json:"endAt"`
	StartLabel string `json:"startLabel"`
	EndLabel   string `json:"endLabel"`
	Day        string `json:"day"`
	DateLabel  string `json:"dateLabel"`
	Track      int    `json:"track"`
	Accent     string `json:"accent"`
	Source     string `json:"source"`
}

type googleCalendarEventsResponse struct {
	Events []frontendEvent `json:"events"`
}

type CalendarHandler struct {
	auth *AuthHandler
}

func NewCalendarHandler(auth *AuthHandler) *CalendarHandler {
	return &CalendarHandler{auth: auth}
}

func (h *CalendarHandler) ListGoogleEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := h.requireGoogleSession()
		if err != nil {
			writeJSON(w, http.StatusBadGateway, authStatusResponse{Provider: "google", Status: "error", Message: err.Error()})
			return
		}

		if session == nil {
			writeJSON(w, http.StatusUnauthorized, authStatusResponse{Provider: "google", Status: "error", Message: "Connect Google Calendar before syncing events."})
			return
		}

		events, err := h.fetchGoogleEvents(session.GoogleProviderToken)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, authStatusResponse{Provider: "google", Status: "error", Message: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, googleCalendarEventsResponse{Events: events})
	}
}

func (h *CalendarHandler) CreateGoogleEvent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := h.requireGoogleSession()
		if err != nil {
			writeJSON(w, http.StatusBadGateway, authStatusResponse{Provider: "google", Status: "error", Message: err.Error()})
			return
		}

		if session == nil {
			writeJSON(w, http.StatusUnauthorized, authStatusResponse{Provider: "google", Status: "error", Message: "Connect Google Calendar before creating events."})
			return
		}

		var payload googleCalendarCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, authStatusResponse{Provider: "google", Status: "error", Message: "Invalid event payload."})
			return
		}

		payload.Title = strings.TrimSpace(payload.Title)
		payload.StartLabel = strings.TrimSpace(payload.StartLabel)
		payload.EndLabel = strings.TrimSpace(payload.EndLabel)
		if payload.Title == "" || payload.StartLabel == "" || payload.EndLabel == "" {
			writeJSON(w, http.StatusBadRequest, authStatusResponse{Provider: "google", Status: "error", Message: "Title, start time, and end time are required."})
			return
		}

		event, err := h.createGoogleEvent(session.GoogleProviderToken, payload)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, authStatusResponse{Provider: "google", Status: "error", Message: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, event)
	}
}

func (h *CalendarHandler) DeleteGoogleEvent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := h.requireGoogleSession()
		if err != nil {
			writeJSON(w, http.StatusBadGateway, authStatusResponse{Provider: "google", Status: "error", Message: err.Error()})
			return
		}

		if session == nil {
			writeJSON(w, http.StatusUnauthorized, authStatusResponse{Provider: "google", Status: "error", Message: "Connect Google Calendar before deleting events."})
			return
		}

		eventID := strings.TrimSpace(chi.URLParam(r, "eventID"))
		if eventID == "" {
			writeJSON(w, http.StatusBadRequest, authStatusResponse{Provider: "google", Status: "error", Message: "Event id is required."})
			return
		}

		if err := h.deleteGoogleEvent(session.GoogleProviderToken, eventID); err != nil {
			writeJSON(w, http.StatusBadGateway, authStatusResponse{Provider: "google", Status: "error", Message: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": eventID})
	}
}

func (h *CalendarHandler) requireGoogleSession() (*authSession, error) {
	session, err := h.auth.ensureSessionFresh()
	if err != nil {
		return nil, err
	}
	if session == nil || !session.GoogleConnected {
		return nil, nil
	}
	if strings.TrimSpace(session.GoogleProviderToken) == "" {
		return nil, fmt.Errorf("google calendar needs to be reconnected before events can sync")
	}
	return session, nil
}

func (h *CalendarHandler) fetchGoogleEvents(accessToken string) ([]frontendEvent, error) {
	query := url.Values{}
	query.Set("singleEvents", "true")
	query.Set("orderBy", "startTime")
	query.Set("timeMin", beginningOfWeek(time.Now()).Format(time.RFC3339))
	query.Set("timeMax", beginningOfWeek(time.Now()).AddDate(0, 0, 30).Format(time.RFC3339))
	query.Set("maxResults", "250")

	var payload googleCalendarListResponse
	if err := h.googleCalendarRequest(http.MethodGet, googleCalendarEventsURL+"?"+query.Encode(), accessToken, nil, &payload); err != nil {
		return nil, err
	}

	events := make([]frontendEvent, 0, len(payload.Items))
	for _, event := range payload.Items {
		if event.Status == "cancelled" {
			continue
		}

		frontendEvent, ok := mapGoogleCalendarEvent(event)
		if ok {
			events = append(events, frontendEvent)
		}
	}

	return events, nil
}

func (h *CalendarHandler) createGoogleEvent(accessToken string, payload googleCalendarCreateRequest) (*frontendEvent, error) {
	startAt, endAt, err := buildTodayEventWindow(payload.StartLabel, payload.EndLabel)
	if err != nil {
		return nil, err
	}

	requestBody := map[string]any{
		"summary": payload.Title,
		"start": map[string]string{
			"dateTime": startAt.Format(time.RFC3339),
			"timeZone": startAt.Location().String(),
		},
		"end": map[string]string{
			"dateTime": endAt.Format(time.RFC3339),
			"timeZone": endAt.Location().String(),
		},
	}

	var created googleCalendarEvent
	if err := h.googleCalendarRequest(http.MethodPost, googleCalendarEventsURL, accessToken, requestBody, &created); err != nil {
		return nil, err
	}

	mapped, ok := mapGoogleCalendarEvent(created)
	if !ok {
		return nil, fmt.Errorf("google calendar returned an event Kai could not render")
	}

	return &mapped, nil
}

func (h *CalendarHandler) deleteGoogleEvent(accessToken string, eventID string) error {
	endpoint := googleCalendarEventsURL + "/" + url.PathEscape(eventID)
	return h.googleCalendarRequest(http.MethodDelete, endpoint, accessToken, nil, nil)
}

func (h *CalendarHandler) googleCalendarRequest(method string, endpoint string, accessToken string, body any, target any) error {
	var payload io.Reader
	if body != nil {
		buffer, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to encode Google Calendar request: %w", err)
		}
		payload = strings.NewReader(string(buffer))
	}

	request, err := http.NewRequest(method, endpoint, payload)
	if err != nil {
		return fmt.Errorf("failed to create Google Calendar request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+accessToken)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := h.auth.client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to reach Google Calendar: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read Google Calendar response: %w", err)
	}

	if response.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("google calendar request failed with HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	if target == nil || len(responseBody) == 0 {
		return nil
	}

	if err := json.Unmarshal(responseBody, target); err != nil {
		return fmt.Errorf("failed to decode Google Calendar response: %w", err)
	}

	return nil
}

func mapGoogleCalendarEvent(event googleCalendarEvent) (frontendEvent, bool) {
	startAt, err := googleDateTime(event.Start)
	if err != nil {
		return frontendEvent{}, false
	}
	endAt, err := googleDateTime(event.End)
	if err != nil {
		return frontendEvent{}, false
	}

	return frontendEvent{
		ID:         event.ID,
		ExternalID: event.ID,
		Title:      firstNonEmpty(strings.TrimSpace(event.Summary), "Untitled Event"),
		StartAt:    startAt.Format(time.RFC3339),
		EndAt:      endAt.Format(time.RFC3339),
		StartLabel: startAt.Format("3:04 PM"),
		EndLabel:   endAt.Format("3:04 PM"),
		Day:        startAt.Format("Mon"),
		DateLabel:  startAt.Format("2"),
		Track:      0,
		Accent:     "blue",
		Source:     "google",
	}, true
}

func googleDateTime(value googleCalendarDateTime) (time.Time, error) {
	if strings.TrimSpace(value.DateTime) != "" {
		return time.Parse(time.RFC3339, value.DateTime)
	}
	if strings.TrimSpace(value.Date) != "" {
		return time.Parse("2006-01-02", value.Date)
	}
	return time.Time{}, fmt.Errorf("google calendar event had no start/end datetime")
}

func buildTodayEventWindow(startLabel string, endLabel string) (time.Time, time.Time, error) {
	now := time.Now()
	startAt, err := parseClockLabelOnDate(now, startLabel)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	endAt, err := parseClockLabelOnDate(now, endLabel)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if !endAt.After(startAt) {
		return time.Time{}, time.Time{}, fmt.Errorf("event end time must be after start time")
	}
	return startAt, endAt, nil
}

func parseClockLabelOnDate(date time.Time, label string) (time.Time, error) {
	parsed, err := time.ParseInLocation("3:04 PM", label, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time label %q", label)
	}
	return time.Date(date.Year(), date.Month(), date.Day(), parsed.Hour(), parsed.Minute(), 0, 0, time.Local), nil
}

func beginningOfWeek(now time.Time) time.Time {
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return start.AddDate(0, 0, -(weekday - 1))
}
