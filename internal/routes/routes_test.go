package routes

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
	"golang.org/x/crypto/bcrypt"

	"stren/internal/controllers"
	"stren/internal/models"
	"stren/internal/utils"
)

// mockRepository is an in-memory implementation of models.Repository for testing.
type mockRepository struct {
	mu        sync.Mutex
	exercises []models.Exercise
	entries   []models.ExerciseEntry

	nextExerciseID int64
	nextEntryID    int64

	errCreate              error
	errGetByName           error
	errGetByID             error
	errUpdate              error
	errList                error
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
		nextExerciseID: 1,
		nextEntryID:    1,
	}
}

func (m *mockRepository) Create(_ *sql.Tx, name string) (int64, error) {
	if m.errCreate != nil {
		return 0, m.errCreate
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.exercises {
		if e.Name == name {
			return e.ID, nil
		}
	}
	id := m.nextExerciseID
	m.nextExerciseID++
	m.exercises = append(m.exercises, models.Exercise{ID: id, Name: name})
	return id, nil
}

func (m *mockRepository) CreateNoTx(name string) (int64, error) {
	return m.Create(nil, name)
}

func (m *mockRepository) GetByID(id int64) (*models.Exercise, error) {
	if m.errGetByID != nil {
		return nil, m.errGetByID
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.exercises {
		if e.ID == id {
			cp := e
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) GetByName(name string) (*models.Exercise, error) {
	if m.errGetByName != nil {
		return nil, m.errGetByName
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.exercises {
		if e.Name == name {
			cp := e
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) Update(id int64, name string) (*models.Exercise, error) {
	if m.errUpdate != nil {
		return nil, m.errUpdate
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, e := range m.exercises {
		if e.ID == id {
			m.exercises[i].Name = name
			cp := m.exercises[i]
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) List() ([]models.Exercise, error) {
	if m.errList != nil {
		return nil, m.errList
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]models.Exercise, len(m.exercises))
	copy(result, m.exercises)
	return result, nil
}

func (m *mockRepository) CreateEntry(entry *models.ExerciseEntry) error {
	if m.errCreateEntry != nil {
		return m.errCreateEntry
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var exerciseID int64
	for _, e := range m.exercises {
		if e.Name == entry.ExerciseName {
			exerciseID = e.ID
			break
		}
	}
	if exerciseID == 0 {
		exerciseID = m.nextExerciseID
		m.nextExerciseID++
		m.exercises = append(m.exercises, models.Exercise{ID: exerciseID, Name: entry.ExerciseName})
	}
	entry.ExerciseID = exerciseID
	entry.ID = m.nextEntryID
	m.nextEntryID++
	m.entries = append(m.entries, *entry)
	return nil
}

func (m *mockRepository) GetEntry(id int64, userID int64) (*models.ExerciseEntry, error) {
	if m.errGetEntry != nil {
		return nil, m.errGetEntry
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.entries {
		if e.ID == id && e.UserID == userID {
			cp := e
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) UpdateEntry(entry *models.ExerciseEntry, userID int64) error {
	if m.errUpdateEntry != nil {
		return m.errUpdateEntry
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, e := range m.entries {
		if e.ID == entry.ID && e.UserID == userID {
			m.entries[i] = *entry
			return nil
		}
	}
	return nil
}

func (m *mockRepository) UpdateEntryWithDate(entry *models.ExerciseEntry, userID int64) error {
	if m.errUpdateEntryWithDate != nil {
		return m.errUpdateEntryWithDate
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, e := range m.entries {
		if e.ID == entry.ID && e.UserID == userID {
			m.entries[i] = *entry
			return nil
		}
	}
	return nil
}

func (m *mockRepository) DeleteEntry(id int64, userID int64) error {
	if m.errDeleteEntry != nil {
		return m.errDeleteEntry
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, e := range m.entries {
		if e.ID == id && e.UserID == userID {
			m.entries = append(m.entries[:i], m.entries[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockRepository) ListEntries(userID int64, limit int) ([]models.ExerciseEntry, error) {
	if m.errListEntries != nil {
		return nil, m.errListEntries
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []models.ExerciseEntry
	for _, e := range m.entries {
		if e.UserID == userID {
			result = append(result, e)
		}
	}
	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockRepository) GetEntriesByExercise(exerciseName string, userID int64) ([]models.ExerciseEntry, error) {
	if m.errGetEntriesByExercise != nil {
		return nil, m.errGetEntriesByExercise
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []models.ExerciseEntry
	for _, e := range m.entries {
		if e.ExerciseName == exerciseName && e.UserID == userID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockRepository) GetEntriesByDateRange(start, end time.Time, userID int64) ([]models.ExerciseEntry, error) {
	if m.errGetEntriesByDateRange != nil {
		return nil, m.errGetEntriesByDateRange
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []models.ExerciseEntry
	for _, e := range m.entries {
		if !e.CreatedAt.Before(start) && !e.CreatedAt.After(end) && e.UserID == userID {
			result = append(result, e)
		}
	}
	return result, nil
}

// mockUserRepository is an in-memory implementation of models.UserRepo for testing.
type mockUserRepository struct {
	mu     sync.Mutex
	users  []models.User
	nextID int64
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{nextID: 1}
}

func (m *mockUserRepository) CreateUser(user *models.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == user.Email {
			return errors.New("email already exists")
		}
	}
	user.ID = m.nextID
	m.nextID++
	m.users = append(m.users, *user)
	return nil
}

func (m *mockUserRepository) GetUserByEmail(email string) (*models.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == email {
			cp := u
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepository) GetUserByID(id int64) (*models.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.ID == id {
			cp := u
			return &cp, nil
		}
	}
	return nil, nil
}

// setupHandler creates a Handler backed by mock repositories for testing.
func setupHandler(t *testing.T) (*Handler, *mockRepository, *mockUserRepository, *echo.Echo) {
	t.Helper()
	e := echo.New()
	mock := newMockRepository()
	mockUser := newMockUserRepository()
	jwtService := utils.NewJWTService("test-secret")
	mock.exercises = []models.Exercise{
		{ID: 1, Name: "Squat"},
		{ID: 2, Name: "Bench Press"},
	}
	mock.nextExerciseID = 3
	authCtrl := controllers.NewAuthController(mockUser, jwtService)
	entryCtrl := controllers.NewEntryController(mock)
	adminCtrl := controllers.NewAdminController(mock)
	validator := utils.NewValidator()
	h := NewHandler(authCtrl, entryCtrl, adminCtrl, jwtService, validator)
	return h, mock, mockUser, e
}

// setAuthContext adds an authenticated user to the Echo context for testing.
func setAuthContext(c echo.Context, userID int64, email, name string, isAdmin bool) {
	claims := &utils.Claims{
		UserID:  userID,
		Email:   email,
		Name:    name,
		IsAdmin: isAdmin,
	}
	c.Set("auth_claims", claims)
}

// chdirToProjectRoot changes to the project root when running from internal/routes.
func chdirToProjectRoot(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(wd) == "routes" {
		if err := os.Chdir("../.."); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			os.Chdir(wd)
		})
	}
}

func TestDashboard(t *testing.T) {
	h, mock, _, e := setupHandler(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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
	h, mock, _, e := setupHandler(t)
	mock.errListEntries = errors.New("db error")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, 1, "test@example.com", "Test User", false)

	err := h.Dashboard(c)
	if err == nil {
		t.Fatal("expected error from repository, got nil")
	}
}

func TestNewEntryForm(t *testing.T) {
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/entries/new", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_name", "Squat")
	form.Set("reps", "5")
	form.Set("weight", "100")

	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_name", "Squat")
	form.Set("reps", "5")
	form.Set("weight", "100")

	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_name", "Squat")
	form.Set("reps", "abc")
	form.Set("weight", "100")

	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_name", "Squat")
	form.Set("reps", "5")
	form.Set("weight", "-10")

	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_name", "")
	form.Set("reps", "5")
	form.Set("weight", "100")

	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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
	h, mock, _, e := setupHandler(t)
	mock.errCreateEntry = errors.New("db error")

	form := url.Values{}
	form.Set("exercise_name", "Squat")
	form.Set("reps", "5")
	form.Set("weight", "100")

	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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
	h, mock, _, e := setupHandler(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	req := httptest.NewRequest(http.MethodGet, "/entries/1/edit", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/entries/abc/edit", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/entries/999/edit", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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
	h, mock, _, e := setupHandler(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	req := httptest.NewRequest(http.MethodGet, "/entries/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/entries/abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/entries/999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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
	h, mock, _, e := setupHandler(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: time.Now()},
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
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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
	h, mock, _, e := setupHandler(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: time.Now()},
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
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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
	h, mock, _, e := setupHandler(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: time.Now()},
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
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodPut, "/entries/abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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
	h, mock, _, e := setupHandler(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	req := httptest.NewRequest(http.MethodDelete, "/entries/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/entries/abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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
	h, mock, _, e := setupHandler(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
		{ID: 2, UserID: 1, ExerciseName: "Bench Press", Reps: 8, Weight: 80},
	}

	req := httptest.NewRequest(http.MethodGet, "/exercises/Squat", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("name")
	c.SetParamValues("Squat")
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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

func TestListExercises(t *testing.T) {
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/exercises", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, 1, "test@example.com", "Test User", false)

	if err := h.ListExercises(c); err != nil {
		t.Fatalf("ListExercises failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get(echo.HeaderContentType); !strings.Contains(ct, echo.MIMEApplicationJSON) {
		t.Fatalf("expected JSON content type, got %q", ct)
	}

	var result []models.Exercise
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 exercises, got %d", len(result))
	}
}

func TestListExercises_RepositoryError(t *testing.T) {
	h, mock, _, e := setupHandler(t)
	mock.errList = errors.New("db error")

	req := httptest.NewRequest(http.MethodGet, "/api/exercises", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, 1, "test@example.com", "Test User", false)

	err := h.ListExercises(c)
	if err == nil {
		t.Fatal("expected error from repository, got nil")
	}
}

func TestServeManifest(t *testing.T) {
	chdirToProjectRoot(t)

	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/manifest.json", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, 1, "test@example.com", "Test User", false)

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

func newEchoContextWithForm(t *testing.T, form url.Values) echo.Context {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec)
}

func TestParseEntryForm_Valid(t *testing.T) {
	form := url.Values{}
	form.Set("exercise_name", "Deadlift")
	form.Set("reps", "5")
	form.Set("weight", "180.5")
	form.Set("notes", "PR")

	c := newEchoContextWithForm(t, form)
	entry, err := parseEntryForm(c, utils.NewValidator())
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
	form := url.Values{}
	form.Set("exercise_name", "")
	form.Set("reps", "5")
	form.Set("weight", "100")

	c := newEchoContextWithForm(t, form)
	_, err := parseEntryForm(c, utils.NewValidator())
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
			form := url.Values{}
			form.Set("exercise_name", "Squat")
			form.Set("reps", tt.reps)
			form.Set("weight", "100")

			c := newEchoContextWithForm(t, form)
			_, err := parseEntryForm(c, utils.NewValidator())
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
			form := url.Values{}
			form.Set("exercise_name", "Squat")
			form.Set("reps", "5")
			form.Set("weight", tt.weight)

			c := newEchoContextWithForm(t, form)
			_, err := parseEntryForm(c, utils.NewValidator())
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

// --- Auth Handler Tests ---

func TestLoginForm(t *testing.T) {
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.LoginForm(c); err != nil {
		t.Fatalf("LoginForm failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Login") {
		t.Fatalf("expected login form, got %q", body)
	}
}

func TestRegisterForm(t *testing.T) {
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.RegisterForm(c); err != nil {
		t.Fatalf("RegisterForm failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Register") {
		t.Fatalf("expected register form, got %q", body)
	}
}

func TestRegisterAndLogin(t *testing.T) {
	h, _, _, e := setupHandler(t)

	// Register
	form := url.Values{}
	form.Set("name", "Alice")
	form.Set("email", "alice@example.com")
	form.Set("password", "secret123")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.Register(c); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect after register, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc != "/" {
		t.Fatalf("expected redirect to '/', got %q", loc)
	}

	// Verify cookie was set
	cookies := rec.Result().Cookies()
	var hasAuth bool
	for _, cookie := range cookies {
		if cookie.Name == utils.CookieName {
			hasAuth = true
			break
		}
	}
	if !hasAuth {
		t.Fatal("expected auth cookie to be set after registration")
	}

	// Login with same credentials
	form = url.Values{}
	form.Set("email", "alice@example.com")
	form.Set("password", "secret123")

	req = httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if err := h.Login(c); err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect after login, got %d", rec.Code)
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)

	// Pre-seed a user
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	mockUser.users = append(mockUser.users, models.User{
		ID:           1,
		Name:         "Bob",
		Email:        "bob@example.com",
		PasswordHash: string(hash),
	})

	form := url.Values{}
	form.Set("email", "bob@example.com")
	form.Set("password", "wrong")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.Login(c); err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 (rendered error), got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Invalid") {
		t.Fatalf("expected error message, got %q", body)
	}
}

func TestLogin_InvalidEmail(t *testing.T) {
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("email", "not-an-email")
	form.Set("password", "password123")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.Login(c); err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 (rendered error), got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "email") {
		t.Fatalf("expected email validation error, got %q", body)
	}
}

func TestLogin_EmptyFields(t *testing.T) {
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("email", "")
	form.Set("password", "")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.Login(c); err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 (rendered error), got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "required") {
		t.Fatalf("expected required field error, got %q", body)
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("name", "Alice")
	form.Set("email", "not-an-email")
	form.Set("password", "secret123")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.Register(c); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 (rendered error), got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "email") {
		t.Fatalf("expected email validation error, got %q", body)
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("name", "Alice")
	form.Set("email", "alice@example.com")
	form.Set("password", "short")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.Register(c); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 (rendered error), got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Password") {
		t.Fatalf("expected password validation error, got %q", body)
	}
}

func TestRegister_EmptyFields(t *testing.T) {
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("name", "")
	form.Set("email", "")
	form.Set("password", "")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.Register(c); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 (rendered error), got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "required") {
		t.Fatalf("expected required field error, got %q", body)
	}
}

func TestLogout(t *testing.T) {
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, 1, "test@example.com", "Test User", false)

	if err := h.Logout(c); err != nil {
		t.Fatalf("Logout failed: %v", err)
	}
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect after logout, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc != "/login" {
		t.Fatalf("expected redirect to '/login', got %q", loc)
	}

	// Verify cookie is cleared
	cookies := rec.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == utils.CookieName && cookie.MaxAge >= 0 {
			t.Fatal("expected auth cookie to be cleared")
		}
	}
}