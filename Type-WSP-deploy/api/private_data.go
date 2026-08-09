package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

const (
	maxScheduleItems           = 2000
	maxScheduleTitle           = 300
	maxScheduleNote            = 2000
	maxInspirationText         = 700
	maxInspirationsPerUser     = 2000
	maxImageLabel              = 200
	maxScheduleRequestBytes    = 8 << 20
	maxInspirationRequestBytes = 16 << 10
	inspirationQuotaLock       = 0x494E5350
)

var errInspirationQuotaExceeded = errors.New("inspiration quota exceeded")

type scheduleTask struct {
	ID         int     `json:"id"`
	Title      string  `json:"title"`
	Note       *string `json:"note,omitempty"`
	Date       string  `json:"date"`
	Time       string  `json:"time"`
	EndTime    *string `json:"endTime,omitempty"`
	Priority   string  `json:"priority"`
	Importance *int    `json:"importance,omitempty"`
	Completed  bool    `json:"completed"`
	Order      int     `json:"order"`
}

type scheduleReminder struct {
	ID      int     `json:"id"`
	Title   string  `json:"title"`
	Date    string  `json:"date"`
	Time    string  `json:"time"`
	EndTime *string `json:"endTime,omitempty"`
	Note    string  `json:"note"`
}

type storedSchedule struct {
	Tasks     []scheduleTask     `json:"tasks"`
	Reminders []scheduleReminder `json:"reminders"`
}

type inspiration struct {
	ID         int64   `json:"id"`
	Date       string  `json:"date"`
	Text       string  `json:"text"`
	ImageLabel *string `json:"imageLabel,omitempty"`
}

type inspirationInput struct {
	Date       string  `json:"date"`
	Text       string  `json:"text"`
	ImageLabel *string `json:"imageLabel,omitempty"`
}

type inspirationUpdate struct {
	Text string `json:"text"`
}

func handleGetSchedule(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var raw []byte
	err := systemPool.QueryRow(r.Context(),
		"SELECT schedule FROM user_schedules WHERE user_id = $1", user.ID,
	).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusOK, storedSchedule{Tasks: []scheduleTask{}, Reminders: []scheduleReminder{}})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "load schedule failed"})
		return
	}

	var schedule storedSchedule
	if err := json.Unmarshal(raw, &schedule); err != nil || validateSchedule(schedule) != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "stored schedule is invalid"})
		return
	}
	writeJSON(w, http.StatusOK, schedule)
}

func handlePutSchedule(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxScheduleRequestBytes)
	var schedule storedSchedule
	if err := readJSON(r, &schedule); err != nil {
		if isMaxBytesError(err) {
			writeJSON(w, http.StatusRequestEntityTooLarge, M{"error": "schedule body is too large"})
			return
		}
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid JSON"})
		return
	}
	if err := validateSchedule(schedule); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": err.Error()})
		return
	}

	raw, err := json.Marshal(schedule)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid schedule"})
		return
	}
	user := currentUser(r)
	_, err = systemPool.Exec(r.Context(), `
		INSERT INTO user_schedules (user_id, schedule)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE
		SET schedule = EXCLUDED.schedule, updated_at = NOW()
	`, user.ID, raw)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "save schedule failed"})
		return
	}
	writeJSON(w, http.StatusOK, schedule)
}

func handleListInspirations(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	rows, err := systemPool.Query(r.Context(), inspirationListQuery(), user.ID, maxInspirationsPerUser)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "load inspirations failed"})
		return
	}
	defer rows.Close()

	items := make([]inspiration, 0)
	for rows.Next() {
		var item inspiration
		if err := rows.Scan(&item.ID, &item.Date, &item.Text, &item.ImageLabel); err != nil {
			writeJSON(w, http.StatusInternalServerError, M{"error": "load inspirations failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "load inspirations failed"})
		return
	}
	writeJSON(w, http.StatusOK, M{"items": items})
}

func inspirationListQuery() string {
	return `
		SELECT id, item_date::text, text, image_label
		FROM inspirations
		WHERE user_id = $1
		ORDER BY item_date DESC, id DESC
		LIMIT $2
	`
}

func handleCreateInspiration(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxInspirationRequestBytes)
	var input inspirationInput
	if err := readJSON(r, &input); err != nil {
		if isMaxBytesError(err) {
			writeJSON(w, http.StatusRequestEntityTooLarge, M{"error": "inspiration body is too large"})
			return
		}
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid JSON"})
		return
	}
	if err := validateInspirationInput(input); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": err.Error()})
		return
	}

	user := currentUser(r)
	var item inspiration
	err := WithTx(r.Context(), systemPool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock($1::integer, $2::integer)", inspirationQuotaLock, user.ID); err != nil {
			return err
		}
		var total int
		if err := tx.QueryRow(r.Context(), "SELECT COUNT(*) FROM inspirations WHERE user_id = $1", user.ID).Scan(&total); err != nil {
			return err
		}
		if !hasInspirationCapacity(total) {
			return errInspirationQuotaExceeded
		}
		return tx.QueryRow(r.Context(), `
			INSERT INTO inspirations (user_id, item_date, text, image_label)
			VALUES ($1, $2, $3, $4)
			RETURNING id, item_date::text, text, image_label
		`, user.ID, input.Date, strings.TrimSpace(input.Text), normalizeOptionalText(input.ImageLabel)).Scan(
			&item.ID, &item.Date, &item.Text, &item.ImageLabel,
		)
	})
	if err != nil {
		if errors.Is(err, errInspirationQuotaExceeded) {
			writeJSON(w, http.StatusTooManyRequests, M{"error": "inspiration quota exceeded"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, M{"error": "create inspiration failed"})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func hasInspirationCapacity(total int) bool {
	return total >= 0 && total < maxInspirationsPerUser
}

func handleUpdateInspiration(w http.ResponseWriter, r *http.Request) {
	id, ok := positivePathID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid inspiration id"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxInspirationRequestBytes)
	var input inspirationUpdate
	if err := readJSON(r, &input); err != nil {
		if isMaxBytesError(err) {
			writeJSON(w, http.StatusRequestEntityTooLarge, M{"error": "inspiration body is too large"})
			return
		}
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid JSON"})
		return
	}
	input.Text = strings.TrimSpace(input.Text)
	if input.Text == "" || utf8.RuneCountInString(input.Text) > maxInspirationText {
		writeJSON(w, http.StatusBadRequest, M{"error": "inspiration text must be 1 to 700 characters"})
		return
	}

	user := currentUser(r)
	command, err := systemPool.Exec(r.Context(), `
		UPDATE inspirations SET text = $1, updated_at = NOW()
		WHERE id = $2 AND user_id = $3
	`, input.Text, id, user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "update inspiration failed"})
		return
	}
	if command.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, M{"error": "inspiration not found"})
		return
	}
	writeJSON(w, http.StatusOK, M{"message": "inspiration updated"})
}

func handleDeleteInspiration(w http.ResponseWriter, r *http.Request) {
	id, ok := positivePathID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid inspiration id"})
		return
	}
	user := currentUser(r)
	command, err := systemPool.Exec(r.Context(),
		"DELETE FROM inspirations WHERE id = $1 AND user_id = $2", id, user.ID,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "delete inspiration failed"})
		return
	}
	if command.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, M{"error": "inspiration not found"})
		return
	}
	writeJSON(w, http.StatusOK, M{"message": "inspiration deleted"})
}

func validateSchedule(schedule storedSchedule) error {
	if schedule.Tasks == nil || schedule.Reminders == nil {
		return errors.New("tasks and reminders are required")
	}
	if len(schedule.Tasks)+len(schedule.Reminders) > maxScheduleItems {
		return errors.New("schedule has too many items")
	}
	taskIDs := make(map[int]struct{}, len(schedule.Tasks))
	for _, task := range schedule.Tasks {
		if task.ID <= 0 || strings.TrimSpace(task.Title) == "" || utf8.RuneCountInString(task.Title) > maxScheduleTitle || !validDate(task.Date) || !validClock(task.Time) || !validOptionalClock(task.EndTime) || !validPriority(task.Priority) || !validOptionalImportance(task.Importance) || !validOptionalLength(task.Note, maxScheduleNote) {
			return errors.New("schedule contains an invalid task")
		}
		if _, exists := taskIDs[task.ID]; exists {
			return errors.New("schedule contains duplicate task ids")
		}
		taskIDs[task.ID] = struct{}{}
	}
	reminderIDs := make(map[int]struct{}, len(schedule.Reminders))
	for _, reminder := range schedule.Reminders {
		if reminder.ID <= 0 || strings.TrimSpace(reminder.Title) == "" || utf8.RuneCountInString(reminder.Title) > maxScheduleTitle || !validDate(reminder.Date) || !validClock(reminder.Time) || !validOptionalClock(reminder.EndTime) || utf8.RuneCountInString(reminder.Note) > maxScheduleNote {
			return errors.New("schedule contains an invalid reminder")
		}
		if _, exists := reminderIDs[reminder.ID]; exists {
			return errors.New("schedule contains duplicate reminder ids")
		}
		reminderIDs[reminder.ID] = struct{}{}
	}
	return nil
}

func validateInspirationInput(input inspirationInput) error {
	text := strings.TrimSpace(input.Text)
	if !validDate(input.Date) {
		return errors.New("invalid inspiration date")
	}
	if text == "" || utf8.RuneCountInString(text) > maxInspirationText {
		return errors.New("inspiration text must be 1 to 700 characters")
	}
	if !validOptionalLength(input.ImageLabel, maxImageLabel) {
		return errors.New("image label is too long")
	}
	return nil
}

func validDate(value string) bool {
	parsed, err := time.Parse("2006-01-02", value)
	return err == nil && parsed.Format("2006-01-02") == value
}

func validClock(value string) bool {
	_, err := time.Parse("15:04", value)
	return err == nil
}

func validOptionalClock(value *string) bool {
	return value == nil || *value == "" || validClock(*value)
}

func validPriority(value string) bool {
	return value == "high" || value == "medium" || value == "low"
}

func validOptionalImportance(value *int) bool {
	return value == nil || (*value >= 1 && *value <= 5)
}

func validOptionalLength(value *string, maximum int) bool {
	return value == nil || utf8.RuneCountInString(*value) <= maximum
}

func normalizeOptionalText(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func positivePathID(value string) (int64, bool) {
	id, err := strconv.ParseInt(value, 10, 64)
	return id, err == nil && id > 0
}
