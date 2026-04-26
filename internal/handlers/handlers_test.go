package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"stren/internal/models"
)

// mockRepository is an in-memory implementation of models.Repository for testing.
type mockRepository struct {
	mu      sync.Mutex
	types   []models.ExerciseType
	entries []models.ExerciseEntry

	nextTypeID  int64
	nextEntryID int64

	errCreateType          error
	errGetTypeByName       error
	errListTypes           error
	errCreateEntry         error
	errGetEntry            error
	errUpdateEntry         error
	errUpdateEntryWithDate error
	errDeleteEntry         error
	errListEntries         error
	errGetEntriesByExercise error
	errGetEntriesByDateRange error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		nextTypeID:  1,
		nextEntryID: 1,
	}
}

func (m *mockRepository) CreateType(_ *sql.Tx, name string) (int64, error) {
	if m.errCreateType != nil {
		return 0, m.errCreateType
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.types {
		if t.Name == name {
			return t.ID, nil
		}
	}
	id := m.nextTypeID
	m.nextTypeID++
	m.types = append(m.types, models.ExerciseType{ID: id, Name: name})
	return id, nil
}

func (m *mockRepository) GetTypeByName(name string) (*models.ExerciseType, error) {
	if m.errGetTypeByName != nil {
		return nil, m.errGetTypeByName
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.types {
		if t.Name == name {
			cp := t
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) ListTypes() ([]models.ExerciseType, error) {
	if m.errListTypes != nil {
		return nil, m.errListTypes
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]models.ExerciseType, len(m.types))
	copy(result, m.types)
	return result, nil
}

func (m *mockRepository) CreateEntry(entry *models.ExerciseEntry) error {
	if m.errCreateEntry != nil {
		return m.errCreateEntry
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var typeID int64
	for _, t := range m.types {
		if t.Name == entry.ExerciseName {
			typeID = t.ID
			break
		}
	}
	if typeID == 0 {
		typeID = m.nextTypeID
		m.nextTypeID++
		m.types = append(m.types, models.ExerciseType{ID: typeID, Name: entry.ExerciseName})
	}
	entry.ExerciseTypeID = typeID
	entry.ID = m.nextEntryID
	m.nextEntryID++
	m.entries = append(m.entries, *entry)
	return nil
}

func (m *mockRepository) GetEntry(id int64) (*models.ExerciseEntry, error) {
	if m.errGetEntry != nil {
		return nil, m.errGetEntry
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.entries {
		if e.ID == id {
			cp := e
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) UpdateEntry(entry *models.ExerciseEntry) error {
	if m.errUpdateEntry != nil {
		return m.errUpdateEntry
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, e := range m.entries {
		if e.ID == entry.ID {
			m.entries[i] = *entry
			return nil
		}
	}
	return nil
}

func (m *mockRepository) UpdateEntryWithDate(entry *models.ExerciseEntry) error {
	if m.errUpdateEntryWithDate != nil {
		return m.errUpdateEntryWithDate
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, e := range m.entries {
		if e.ID == entry.ID {
			m.entries[i] = *entry
			return nil
		}
	}
	return nil
}

func (m *mockRepository) DeleteEntry(id int64) error {
	if m.errDeleteEntry != nil {
		return m.errDeleteEntry
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, e := range m.entries {
		if e.ID == id {
			m.entries = append(m.entries[:i], m.entries[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockRepository) ListEntries(limit int) ([]models.ExerciseEntry, error) {
	if m.errListEntries != nil {
		return nil, m.errListEntries
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]models.ExerciseEntry, len(m.entries))
	copy(result, m.entries)
	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockRepository) GetEntriesByExercise(exerciseName string) ([]models.ExerciseEntry, error) {
	if m.errGetEntriesByExercise != nil {
		return nil, m.errGetEntriesByExercise
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []models.ExerciseEntry
	for _, e := range m.entries {
		if e.ExerciseName == exerciseName {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockRepository) GetEntriesByDateRange(start, end time.Time) ([]models.ExerciseEntry, error) {
	if m.errGetEntriesByDateRange != nil {
		return nil, m.errGetEntriesByDateRange
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []models.ExerciseEntry
	for _, e := range m.entries {
		if !e.CreatedAt.Before(start) && !e.CreatedAt.After(end) {
			result = append(result, e)
		}
	}
	return result, nil
}

// setupHandler creates a Handler backed by a mock repository for testing.
func setupHandler(t *testing.T) (*Handler, *mockRepository, *echo.Echo) {
	t.Helper()
	e := echo.New()
	mock := newMockRepository()
	// Seed with some exercise types so forms render correctly.
	mock.types = []models.ExerciseType{
		{ID: 1, Name: "Squat"},
		{ID: 2, Name: "Bench Press"},
	}
	mock.nextTypeID = 3
	h := NewHandler(mock)
	return h, mock, e
}

// chdirToProjectRoot changes to the project root when running from internal/handlers.
func chdirToProjectRoot(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(wd) == "handlers" {
		if err := os.Chdir("../.."); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			os.Chdir(wd)
		})
	}
}

func TestDashboard(t *testing.T) {
	h, mock, e := setupHandler(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.Dashboard(c); err != nil {
		t.Fatalf("Dashboard failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() == "" {
		t.Fatal("expected non-empty body")
	}
}

func TestDashboard_RepositoryError(t *testing.T) {
	h, mock, e := setupHandler(t)
	mock.errListEntries = errors.New("db error")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.Dashboard(c)
	if err == nil {
		t.Fatal("expected error from repository, got nil")
	}
}

func TestNewEntryForm(t *testing.T) {
	h, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/entries/new", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.NewEntryForm(c); err != nil {
		t.Fatalf("NewEntryForm failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() == "" {
		t.Fatal("expected non-empty body")
	}
}

func TestCreateEntry(t *testing.T) {
	h, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_name", "Squat")
	form.Set("reps", "5")
	form.Set("weight", "100")

	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.CreateEntry(c); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "saved") {
		t.Fatalf("expected success message containing 'saved', got %q", body)
	}
}

func TestCreateEntry_HTMX(t *testing.T) {
	h, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_name", "Squat")
	form.Set("reps", "5")
	form.Set("weight", "100")

	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.CreateEntry(c); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "saved") {
		t.Fatalf("expected success message containing 'saved', got %q", body)
	}
}

func TestCreateEntry_InvalidReps(t *testing.T) {
	h, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_name", "Squat")
	form.Set("reps", "abc")
	form.Set("weight", "100")

	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.CreateEntry(c); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 (rendered error), got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Reps") {
		t.Fatalf("expected error message containing 'Reps', got %q", body)
	}
}

func TestCreateEntry_InvalidWeight(t *testing.T) {
	h, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_name", "Squat")
	form.Set("reps", "5")
	form.Set("weight", "-10")

	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.CreateEntry(c); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 (rendered error), got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Weight") {
		t.Fatalf("expected error message containing 'Weight', got %q", body)
	}
}

func TestCreateEntry_MissingName(t *testing.T) {
	h, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_name", "")
	form.Set("reps", "5")
	form.Set("weight", "100")

	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.CreateEntry(c); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 (rendered error), got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "name") && !strings.Contains(body, "Name") {
		t.Fatalf("expected error message containing 'name', got %q", body)
	}
}

func TestCreateEntry_RepositoryError(t *testing.T) {
	h, mock, e := setupHandler(t)
	mock.errCreateEntry = errors.New("db error")

	form := url.Values{}
	form.Set("exercise_name", "Squat")
	form.Set("reps", "5")
	form.Set("weight", "100")

	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.CreateEntry(c); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 (rendered error), got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Failed to save") {
		t.Fatalf("expected error message containing 'Failed to save', got %q", body)
	}
}

func TestEditEntryForm(t *testing.T) {
	h, mock, e := setupHandler(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	req := httptest.NewRequest(http.MethodGet, "/entries/1/edit", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	if err := h.EditEntryForm(c); err != nil {
		t.Fatalf("EditEntryForm failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() == "" {
		t.Fatal("expected non-empty body")
	}
}

func TestEditEntryForm_InvalidID(t *testing.T) {
	h, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/entries/abc/edit", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := h.EditEntryForm(c)
	if err == nil {
		t.Fatal("expected error for invalid ID")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request error, got %v", err)
	}
}

func TestEditEntryForm_NotFound(t *testing.T) {
	h, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/entries/999/edit", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")

	err := h.EditEntryForm(c)
	if err == nil {
		t.Fatal("expected error for missing entry")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusNotFound {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestGetEntry(t *testing.T) {
	h, mock, e := setupHandler(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	req := httptest.NewRequest(http.MethodGet, "/entries/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	if err := h.GetEntry(c); err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() == "" {
		t.Fatal("expected non-empty body")
	}
}

func TestGetEntry_InvalidID(t *testing.T) {
	h, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/entries/abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := h.GetEntry(c)
	if err == nil {
		t.Fatal("expected error for invalid ID")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request error, got %v", err)
	}
}

func TestGetEntry_NotFound(t *testing.T) {
	h, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/entries/999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")

	err := h.GetEntry(c)
	if err == nil {
		t.Fatal("expected error for missing entry")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusNotFound {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestUpdateEntry(t *testing.T) {
	h, mock, e := setupHandler(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: time.Now()},
	}

	form := url.Values{}
	form.Set("exercise_name", "Squat")
	form.Set("reps", "3")
	form.Set("weight", "120")
	form.Set("created_at", time.Now().Format("2006-01-02T15:04"))

	req := httptest.NewRequest(http.MethodPut, "/entries/1", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	if err := h.UpdateEntry(c); err != nil {
		t.Fatalf("UpdateEntry failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "updated") {
		t.Fatalf("expected success message containing 'updated', got %q", body)
	}
}

func TestUpdateEntry_Redirect(t *testing.T) {
	h, mock, e := setupHandler(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: time.Now()},
	}

	form := url.Values{}
	form.Set("exercise_name", "Squat")
	form.Set("reps", "3")
	form.Set("weight", "120")

	req := httptest.NewRequest(http.MethodPut, "/entries/1", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	if err := h.UpdateEntry(c); err != nil {
		t.Fatalf("UpdateEntry failed: %v", err)
	}
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc != "/" {
		t.Fatalf("expected redirect to '/', got %q", loc)
	}
}

func TestUpdateEntry_InvalidDate(t *testing.T) {
	h, mock, e := setupHandler(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: time.Now()},
	}

	form := url.Values{}
	form.Set("exercise_name", "Squat")
	form.Set("reps", "3")
	form.Set("weight", "120")
	form.Set("created_at", "not-a-date")

	req := httptest.NewRequest(http.MethodPut, "/entries/1", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	if err := h.UpdateEntry(c); err != nil {
		t.Fatalf("UpdateEntry failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 (rendered error), got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Invalid date") {
		t.Fatalf("expected error message containing 'Invalid date', got %q", body)
	}
}

func TestUpdateEntry_InvalidID(t *testing.T) {
	h, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodPut, "/entries/abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := h.UpdateEntry(c)
	if err == nil {
		t.Fatal("expected error for invalid ID")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request error, got %v", err)
	}
}

func TestDeleteEntry(t *testing.T) {
	h, mock, e := setupHandler(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	req := httptest.NewRequest(http.MethodDelete, "/entries/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	if err := h.DeleteEntry(c); err != nil {
		t.Fatalf("DeleteEntry failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if len(mock.entries) != 0 {
		t.Fatalf("expected entry to be deleted, got %d entries", len(mock.entries))
	}
}

func TestDeleteEntry_InvalidID(t *testing.T) {
	h, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/entries/abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := h.DeleteEntry(c)
	if err == nil {
		t.Fatal("expected error for invalid ID")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request error, got %v", err)
	}
}

func TestExerciseHistory(t *testing.T) {
	h, mock, e := setupHandler(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
		{ID: 2, ExerciseName: "Bench Press", Reps: 8, Weight: 80},
	}

	req := httptest.NewRequest(http.MethodGet, "/exercises/Squat", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("name")
	c.SetParamValues("Squat")

	if err := h.ExerciseHistory(c); err != nil {
		t.Fatalf("ExerciseHistory failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() == "" {
		t.Fatal("expected non-empty body")
	}
}

func TestListExerciseTypes(t *testing.T) {
	h, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/exercises", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.ListExerciseTypes(c); err != nil {
		t.Fatalf("ListExerciseTypes failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get(echo.HeaderContentType); !strings.Contains(ct, echo.MIMEApplicationJSON) {
		t.Fatalf("expected JSON content type, got %q", ct)
	}

	var result []models.ExerciseType
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 types, got %d", len(result))
	}
}

func TestListExerciseTypes_RepositoryError(t *testing.T) {
	h, mock, e := setupHandler(t)
	mock.errListTypes = errors.New("db error")

	req := httptest.NewRequest(http.MethodGet, "/api/exercises", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.ListExerciseTypes(c)
	if err == nil {
		t.Fatal("expected error from repository, got nil")
	}
}

func TestServeManifest(t *testing.T) {
	chdirToProjectRoot(t)

	h, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/manifest.json", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.ServeManifest(c); err != nil {
		t.Fatalf("ServeManifest failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	ct := rec.Header().Get(echo.HeaderContentType)
	if ct != "application/manifest+json" {
		t.Fatalf("expected content-type 'application/manifest+json', got %q", ct)
	}
}

func TestParseEntryForm_Valid(t *testing.T) {
	h, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_name", "Deadlift")
	form.Set("reps", "5")
	form.Set("weight", "180.5")
	form.Set("notes", "PR")

	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	entry, err := h.parseEntryForm(c)
	if err != nil {
		t.Fatalf("parseEntryForm failed: %v", err)
	}
	if entry.ExerciseName != "Deadlift" {
		t.Fatalf("expected exercise name 'Deadlift', got %q", entry.ExerciseName)
	}
	if entry.Reps != 5 {
		t.Fatalf("expected reps 5, got %d", entry.Reps)
	}
	if entry.Weight != 180.5 {
		t.Fatalf("expected weight 180.5, got %f", entry.Weight)
	}
	if entry.Notes != "PR" {
		t.Fatalf("expected notes 'PR', got %q", entry.Notes)
	}
}

func TestParseEntryForm_MissingName(t *testing.T) {
	h, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_name", "")
	form.Set("reps", "5")
	form.Set("weight", "100")

	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, err := h.parseEntryForm(c)
	if err == nil {
		t.Fatal("expected error for missing exercise name")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request error, got %v", err)
	}
}

func TestParseEntryForm_InvalidReps(t *testing.T) {
	tests := []struct {
		name string
		reps string
	}{
		{"non-numeric", "abc"},
		{"zero", "0"},
		{"negative", "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, e := setupHandler(t)

			form := url.Values{}
			form.Set("exercise_name", "Squat")
			form.Set("reps", tt.reps)
			form.Set("weight", "100")

			req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			_, err := h.parseEntryForm(c)
			if err == nil {
				t.Fatal("expected error for invalid reps")
			}
			httpErr, ok := err.(*echo.HTTPError)
			if !ok || httpErr.Code != http.StatusBadRequest {
				t.Fatalf("expected bad request error, got %v", err)
			}
		})
	}
}

func TestParseEntryForm_InvalidWeight(t *testing.T) {
	tests := []struct {
		name   string
		weight string
	}{
		{"non-numeric", "abc"},
		{"negative", "-10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, e := setupHandler(t)

			form := url.Values{}
			form.Set("exercise_name", "Squat")
			form.Set("reps", "5")
			form.Set("weight", tt.weight)

			req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			_, err := h.parseEntryForm(c)
			if err == nil {
				t.Fatal("expected error for invalid weight")
			}
			httpErr, ok := err.(*echo.HTTPError)
			if !ok || httpErr.Code != http.StatusBadRequest {
				t.Fatalf("expected bad request error, got %v", err)
			}
		})
	}
}
