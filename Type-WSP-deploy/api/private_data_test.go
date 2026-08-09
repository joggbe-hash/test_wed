package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrivateDataRoutesRequireAuthentication(t *testing.T) {
	mux := newMuxWithSessionLoader(func(context.Context, string) (*User, error) {
		t.Fatal("session loader must not be called without a cookie")
		return nil, nil
	})

	for _, request := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/schedule"},
		{http.MethodPut, "/api/schedule"},
		{http.MethodGet, "/api/inspirations"},
		{http.MethodPost, "/api/inspirations"},
		{http.MethodPatch, "/api/inspirations/1"},
		{http.MethodDelete, "/api/inspirations/1"},
	} {
		response := httptest.NewRecorder()
		req := httptest.NewRequest(request.method, request.path, nil)
		if request.method != http.MethodGet {
			req.Header.Set(browserRequestHeader, "1")
		}
		mux.ServeHTTP(response, req)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s status = %d, want %d", request.method, request.path, response.Code, http.StatusUnauthorized)
		}
	}
}

func TestValidateScheduleAcceptsValidPrivateData(t *testing.T) {
	note := "private note"
	endTime := "10:30"
	importance := 5
	schedule := storedSchedule{
		Tasks: []scheduleTask{{
			ID: 1, Title: "task", Note: &note, Date: "2026-08-04", Time: "09:00",
			EndTime: &endTime, Priority: "high", Importance: &importance, Order: 1,
		}},
		Reminders: []scheduleReminder{{
			ID: 1, Title: "reminder", Date: "2026-08-04", Time: "08:30", Note: "note",
		}},
	}
	if err := validateSchedule(schedule); err != nil {
		t.Fatalf("validateSchedule returned %v", err)
	}
}

func TestValidateScheduleRejectsDuplicateIDs(t *testing.T) {
	schedule := storedSchedule{
		Tasks: []scheduleTask{
			{ID: 1, Title: "a", Date: "2026-08-04", Time: "09:00", Priority: "low"},
			{ID: 1, Title: "b", Date: "2026-08-04", Time: "10:00", Priority: "low"},
		},
		Reminders: []scheduleReminder{},
	}
	if err := validateSchedule(schedule); err == nil {
		t.Fatal("expected duplicate task ids to be rejected")
	}
}

func TestValidateInspirationInputRejectsInvalidValues(t *testing.T) {
	for _, input := range []inspirationInput{
		{Date: "2026-02-30", Text: "text"},
		{Date: "2026-08-04", Text: "   "},
	} {
		if err := validateInspirationInput(input); err == nil {
			t.Fatalf("expected input %#v to be rejected", input)
		}
	}
}

func TestScheduleRequestBodyIsBoundedBeforeJSONValidation(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPut,
		"/api/schedule",
		strings.NewReader(strings.Repeat(" ", maxScheduleRequestBytes+1)),
	)
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(context.WithValue(request.Context(), userCtxKey, &User{ID: 1}))
	response := httptest.NewRecorder()

	handlePutSchedule(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestInspirationRequestBodyIsBoundedBeforeJSONValidation(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/inspirations",
		strings.NewReader(strings.Repeat(" ", maxInspirationRequestBytes+1)),
	)
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(context.WithValue(request.Context(), userCtxKey, &User{ID: 1}))
	response := httptest.NewRecorder()

	handleCreateInspiration(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestInspirationUpdateRequestBodyIsBoundedBeforeJSONValidation(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPut,
		"/api/inspirations/1",
		strings.NewReader(strings.Repeat(" ", maxInspirationRequestBytes+1)),
	)
	request.Header.Set("Content-Type", "application/json")
	request.SetPathValue("id", "1")
	request = request.WithContext(context.WithValue(request.Context(), userCtxKey, &User{ID: 1}))
	response := httptest.NewRecorder()

	handleUpdateInspiration(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestInspirationCapacityBoundsCumulativePerUserContent(t *testing.T) {
	if !hasInspirationCapacity(maxInspirationsPerUser - 1) {
		t.Fatal("inspiration below the cumulative per-user limit was rejected")
	}
	if hasInspirationCapacity(maxInspirationsPerUser) {
		t.Fatal("inspiration at the cumulative per-user limit was accepted")
	}
}

func TestInspirationListQueryIsBounded(t *testing.T) {
	query := inspirationListQuery()
	if !strings.Contains(query, "LIMIT $2") {
		t.Fatalf("inspiration list query has no server-side bound: %s", query)
	}
}
