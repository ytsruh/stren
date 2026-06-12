package routes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
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

type mockRepository struct {
	mu        sync.Mutex
	exercises []models.Exercise
	entries   []models.ExerciseEntry

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
	errGetExerciseByID       error
}

func newMockRepository() *mockRepository {
	return &mockRepository{}
}

func (m *mockRepository) Create(_ *sql.Tx, name string) (string, error) {
	if m.errCreate != nil {
		return "", m.errCreate
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.exercises {
		if e.Name == name {
			return e.ID, nil
		}
	}
	id := "ex-" + name
	m.exercises = append(m.exercises, models.Exercise{ID: id, Name: name})
	return id, nil
}

func (m *mockRepository) CreateNoTx(params models.CreateExerciseParams) (string, error) {
	if m.errCreate != nil {
		return "", m.errCreate
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.exercises {
		if e.Name == params.Name {
			return e.ID, nil
		}
	}
	id := "ex-" + params.Name
	m.exercises = append(m.exercises, models.Exercise{ID: id, Name: params.Name, Description: params.Description, VideoURL: params.VideoURL, ImgURL: params.ImgURL, Type: params.Type})
	return id, nil
}

func (m *mockRepository) GetByID(id string) (*models.Exercise, error) {
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

func (m *mockRepository) GetExerciseByID(id string, userID string) (*models.Exercise, error) {
	if m.errGetExerciseByID != nil {
		return nil, m.errGetExerciseByID
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

func (m *mockRepository) Update(id string, params models.UpdateExerciseParams) (*models.Exercise, error) {
	if m.errUpdate != nil {
		return nil, m.errUpdate
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, e := range m.exercises {
		if e.ID == id {
			m.exercises[i].Name = params.Name
			m.exercises[i].Description = params.Description
			m.exercises[i].VideoURL = params.VideoURL
			m.exercises[i].ImgURL = params.ImgURL
			m.exercises[i].Type = params.Type
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

	var exerciseID string
	for _, e := range m.exercises {
		if e.Name == entry.ExerciseName {
			exerciseID = e.ID
			break
		}
	}
	if exerciseID == "" {
		exerciseID = "ex-" + entry.ExerciseName
		m.exercises = append(m.exercises, models.Exercise{ID: exerciseID, Name: entry.ExerciseName})
	}
	entry.ExerciseID = exerciseID
	entry.ID = "entry-" + entry.ExerciseName
	m.entries = append(m.entries, *entry)
	return nil
}

func (m *mockRepository) GetEntry(id string, userID string) (*models.ExerciseEntry, error) {
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

func (m *mockRepository) UpdateEntry(entry *models.ExerciseEntry, userID string) error {
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

func (m *mockRepository) UpdateEntryWithDate(entry *models.ExerciseEntry, userID string) error {
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

func (m *mockRepository) DeleteEntry(id string, userID string) error {
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

func (m *mockRepository) ListEntries(userID string, limit int) ([]models.ExerciseEntry, error) {
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

func (m *mockRepository) GetEntriesByExercise(exerciseID string, userID string) ([]models.ExerciseEntry, error) {
	if m.errGetEntriesByExercise != nil {
		return nil, m.errGetEntriesByExercise
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []models.ExerciseEntry
	for _, e := range m.entries {
		if e.ExerciseID == exerciseID && e.UserID == userID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockRepository) GetEntriesByDateRange(start, end time.Time, userID string) ([]models.ExerciseEntry, error) {
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

func (m *mockRepository) ListEntriesLast7Days(userID string) ([]models.ExerciseEntry, error) {
	if m.errListEntries != nil {
		return nil, m.errListEntries
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	var result []models.ExerciseEntry
	for _, e := range m.entries {
		if e.CreatedAt.After(sevenDaysAgo) && e.UserID == userID {
			result = append(result, e)
		}
	}
	return result, nil
}

type mockUserRepository struct {
	mu    sync.Mutex
	users []models.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{}
}

func (m *mockUserRepository) CreateUser(user *models.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == user.Email {
			return errors.New("email already exists")
		}
	}
	user.ID = "user-" + user.Email
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

func (m *mockUserRepository) GetUserByID(id string) (*models.User, error) {
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

func (m *mockUserRepository) UpdateUser(user *models.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, u := range m.users {
		if u.ID == user.ID {
			m.users[i] = *user
			return nil
		}
	}
	return errors.New("user not found")
}

type mockAdminUserRepository struct {
	mu    sync.Mutex
	users []models.User
}

func newMockAdminUserRepository() *mockAdminUserRepository {
	return &mockAdminUserRepository{}
}

func (m *mockAdminUserRepository) ListUsers() ([]models.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.users, nil
}

type mockFeedbackRepository struct {
	mu       sync.Mutex
	feedback []*models.Feedback

	errCreate   error
	errGetAll   error
	errGetByID  error
	errUpdate   error
}

func newMockFeedbackRepository() *mockFeedbackRepository {
	return &mockFeedbackRepository{}
}

func (m *mockFeedbackRepository) Create(feedback *models.Feedback) error {
	if m.errCreate != nil {
		return m.errCreate
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	feedback.ID = "fb-" + feedback.Title
	m.feedback = append(m.feedback, feedback)
	return nil
}

func (m *mockFeedbackRepository) GetAll(filter string) ([]*models.Feedback, error) {
	if m.errGetAll != nil {
		return nil, m.errGetAll
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []*models.Feedback
	for _, f := range m.feedback {
		switch filter {
		case "open":
			if !f.IsClosed {
				result = append(result, f)
			}
		case "closed":
			if f.IsClosed {
				result = append(result, f)
			}
		default:
			result = append(result, f)
		}
	}
	return result, nil
}

func (m *mockFeedbackRepository) GetByID(id string) (*models.Feedback, error) {
	if m.errGetByID != nil {
		return nil, m.errGetByID
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, f := range m.feedback {
		if f.ID == id {
			return f, nil
		}
	}
	return nil, nil
}

func (m *mockFeedbackRepository) UpdateStatus(id string, isClosed bool) error {
	if m.errUpdate != nil {
		return m.errUpdate
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, f := range m.feedback {
		if f.ID == id {
			f.IsClosed = isClosed
			return nil
		}
	}
	return nil
}

type mockWeightRepository struct {
	mu       sync.Mutex
	entries  []models.WeightEntry

	errCreate    error
	errGetByID   error
	errList      error
	errUpdate    error
	errDelete    error
}

func newMockWeightRepository() *mockWeightRepository {
	return &mockWeightRepository{}
}

func (m *mockWeightRepository) Create(entry *models.WeightEntry) error {
	if m.errCreate != nil {
		return m.errCreate
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	entry.ID = "weight-" + fmt.Sprintf("%d", len(m.entries)+1)
	m.entries = append(m.entries, *entry)
	return nil
}

func (m *mockWeightRepository) GetByID(id string, userID string) (*models.WeightEntry, error) {
	if m.errGetByID != nil {
		return nil, m.errGetByID
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.entries {
		if e.ID == id && e.UserID == userID {
			return &e, nil
		}
	}
	return nil, nil
}

func (m *mockWeightRepository) List(userID string) ([]models.WeightEntry, error) {
	if m.errList != nil {
		return nil, m.errList
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []models.WeightEntry
	for _, e := range m.entries {
		if e.UserID == userID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockWeightRepository) Update(entry *models.WeightEntry, userID string) error {
	if m.errUpdate != nil {
		return m.errUpdate
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

func (m *mockWeightRepository) Delete(id string, userID string) error {
	if m.errDelete != nil {
		return m.errDelete
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

func setupHandler(t *testing.T) (*Handler, *mockRepository, *mockUserRepository, *echo.Echo) {
	t.Helper()
	e := echo.New()
	mock := newMockRepository()
	mockUser := newMockUserRepository()
	mockAdminUser := newMockAdminUserRepository()
	mockFeedback := newMockFeedbackRepository()
	mockWeight := newMockWeightRepository()
	jwtService := utils.NewJWTService("test-secret")
	mock.exercises = []models.Exercise{
		{ID: "ex-1", Name: "Squat"},
		{ID: "ex-2", Name: "Bench Press"},
	}
	authCtrl := controllers.NewAuthController(mockUser, jwtService)
	entryCtrl := controllers.NewEntryController(mock)
	adminCtrl := controllers.NewAdminController(mock)
	adminUserCtrl := controllers.NewAdminUserController(mockAdminUser)
	feedbackCtrl := controllers.NewFeedbackController(mockFeedback)
	timersCtrl := controllers.NewTimersController()
	weightCtrl := controllers.NewWeightController(mockWeight)
	validator := utils.NewValidator()
	h := NewHandler(authCtrl, entryCtrl, adminCtrl, adminUserCtrl, feedbackCtrl, timersCtrl, weightCtrl, mockUser, jwtService, validator)
	return h, mock, mockUser, e
}

func setAuthContext(c echo.Context, userID string, email, name string, isAdmin bool) {
	claims := &utils.Claims{
		UserID:  userID,
		Email:   email,
		Name:    name,
		IsAdmin: isAdmin,
	}
	c.Set("auth_claims", claims)
}

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
		{ID: "entry-1", UserID: "user-1", ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

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
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

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
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

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

func TestNewEntryForm_Preselected(t *testing.T) {
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/exercises/ex-1/new", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("ex-1")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.NewEntryForm(c); err != nil {
		t.Fatalf("NewEntryForm failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<option value="ex-1" selected>Squat</option>`) {
		t.Fatalf("expected Squat option to be preselected, got body: %s", body)
	}
	if strings.Contains(body, `<option value="ex-2" selected>Bench Press</option>`) {
		t.Fatalf("expected Bench Press option NOT to be preselected, got body: %s", body)
	}
}

func TestNewEntryForm_PreselectedInvalid(t *testing.T) {
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/exercises/non-existent/new", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("non-existent")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.NewEntryForm(c); err != nil {
		t.Fatalf("NewEntryForm failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 (graceful fallback), got %d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, `selected>`) {
		t.Fatalf("expected no preselected option for unknown exercise, got body: %s", body)
	}
}

func TestCreateEntry(t *testing.T) {
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_id", "ex-1")
	form.Set("reps", "5")
	form.Set("weight", "100")

	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.CreateEntry(c); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Fatalf("expected redirect to /, got %q", loc)
	}
}

func TestCreateEntry_HTMX(t *testing.T) {
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_id", "ex-1")
	form.Set("reps", "5")
	form.Set("weight", "100")

	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.CreateEntry(c); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if hxTrigger := rec.Header().Get("HX-Trigger"); hxTrigger != `{"triggerRedirect": "/"}` {
		t.Fatalf("expected HX-Trigger to be set, got %q", hxTrigger)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Entry saved") {
		t.Fatalf("expected toast with 'Entry saved', got body: %s", body)
	}
}

func TestCreateEntry_InvalidReps(t *testing.T) {
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_id", "ex-1")
	form.Set("reps", "abc")
	form.Set("weight", "100")

	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

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
	form.Set("exercise_id", "ex-1")
	form.Set("reps", "5")
	form.Set("weight", "-10")

	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

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

func TestCreateEntry_MissingExercise(t *testing.T) {
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_id", "")
	form.Set("reps", "5")
	form.Set("weight", "100")

	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.CreateEntry(c); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 (rendered error), got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Exercise") {
		t.Fatalf("expected error message containing 'Exercise', got %q", body)
	}
}

func TestCreateEntry_RepositoryError(t *testing.T) {
	h, mock, _, e := setupHandler(t)
	mock.errCreateEntry = errors.New("db error")

	form := url.Values{}
	form.Set("exercise_id", "ex-1")
	form.Set("reps", "5")
	form.Set("weight", "100")

	req := httptest.NewRequest(http.MethodPost, "/entries", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

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
		{ID: "entry-1", UserID: "user-1", ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	req := httptest.NewRequest(http.MethodGet, "/entries/entry-1/edit", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("entry-1")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

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

func TestEditEntryForm_NotFound(t *testing.T) {
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/entries/non-existent/edit", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("non-existent")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

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
		{ID: "entry-1", UserID: "user-1", ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	req := httptest.NewRequest(http.MethodGet, "/entries/entry-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("entry-1")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

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

func TestGetEntry_NotFound(t *testing.T) {
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/entries/non-existent", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("non-existent")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

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
		{ID: "entry-1", UserID: "user-1", ExerciseID: "ex-1", ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: time.Now()},
	}

	form := url.Values{}
	form.Set("exercise_id", "ex-1")
	form.Set("reps", "3")
	form.Set("weight", "120")
	form.Set("created_at", time.Now().Format("2006-01-02T15:04"))

	req := httptest.NewRequest(http.MethodPut, "/entries/entry-1", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("entry-1")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

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
		{ID: "entry-1", UserID: "user-1", ExerciseID: "ex-1", ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: time.Now()},
	}

	form := url.Values{}
	form.Set("exercise_id", "ex-1")
	form.Set("reps", "3")
	form.Set("weight", "120")

	req := httptest.NewRequest(http.MethodPut, "/entries/entry-1", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("entry-1")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

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
		{ID: "entry-1", UserID: "user-1", ExerciseID: "ex-1", ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: time.Now()},
	}

	form := url.Values{}
	form.Set("exercise_id", "ex-1")
	form.Set("reps", "3")
	form.Set("weight", "120")
	form.Set("created_at", "not-a-date")

	req := httptest.NewRequest(http.MethodPut, "/entries/entry-1", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("entry-1")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

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

func TestDeleteEntry(t *testing.T) {
	h, mock, _, e := setupHandler(t)
	mock.entries = []models.ExerciseEntry{
		{ID: "entry-1", UserID: "user-1", ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	req := httptest.NewRequest(http.MethodDelete, "/entries/entry-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("entry-1")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.DeleteEntry(c); err != nil {
		t.Fatalf("DeleteEntry failed: %v", err)
	}
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303, got %d", rec.Code)
	}
	if len(mock.entries) != 0 {
		t.Fatalf("expected entry to be deleted, got %d entries", len(mock.entries))
	}
}

func TestExerciseHistory(t *testing.T) {
	h, mock, _, e := setupHandler(t)
	mock.exercises = []models.Exercise{
		{ID: "uuid-exercise-1", Name: "Squat"},
	}
	mock.entries = []models.ExerciseEntry{
		{ID: "entry-1", UserID: "user-1", ExerciseName: "Squat", Reps: 5, Weight: 100},
		{ID: "entry-2", UserID: "user-1", ExerciseName: "Bench Press", Reps: 8, Weight: 80},
	}

	req := httptest.NewRequest(http.MethodGet, "/exercises/uuid-exercise-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("uuid-exercise-1")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

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
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.ListExercisesJSON(c); err != nil {
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
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	err := h.ListExercisesJSON(c)
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
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

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
	if entry.ExerciseName != "" {
		t.Fatalf("expected exercise name '', got %q", entry.ExerciseName)
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

func TestTimerPage(t *testing.T) {
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/timer", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.TimerPage(c); err != nil {
		t.Fatalf("TimerPage failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Timer") {
		t.Fatalf("expected timer page, got %q", body)
	}
	if !strings.Contains(body, `role="tablist"`) {
		t.Fatalf("expected tablist on timer page, got %q", body)
	}
	if !strings.Contains(body, `id="timers-tab-timer"`) {
		t.Fatalf("expected Timer tab button, got %q", body)
	}
	if !strings.Contains(body, `id="timers-tab-emom"`) {
		t.Fatalf("expected EMOM tab button, got %q", body)
	}
	if !strings.Contains(body, `id="timers-panel-timer"`) {
		t.Fatalf("expected Timer tab panel, got %q", body)
	}
	if !strings.Contains(body, `id="timers-panel-emom"`) {
		t.Fatalf("expected EMOM tab panel, got %q", body)
	}
	// Timer tab must be the active one when on /timer
	timerActiveRegex := regexp.MustCompile(`id="timers-tab-timer"[^>]*aria-selected="true"`)
	emomActiveRegex := regexp.MustCompile(`id="timers-tab-emom"[^>]*aria-selected="true"`)
	if !timerActiveRegex.MatchString(body) {
		t.Fatalf("expected Timer tab to be active on /timer, got %q", body)
	}
	if emomActiveRegex.MatchString(body) {
		t.Fatalf("expected EMOM tab to be inactive on /timer, got %q", body)
	}
}

func TestTimerValidationError(t *testing.T) {
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/timer/error", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.TimerValidationError(c); err != nil {
		t.Fatalf("TimerValidationError failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Invalid Duration") {
		t.Fatalf("expected toast error, got %q", body)
	}
}

func TestRegisterAndLogin(t *testing.T) {
	h, _, _, e := setupHandler(t)

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

	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	mockUser.users = append(mockUser.users, models.User{
		ID:           "user-1",
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
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

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

	cookies := rec.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == utils.CookieName && cookie.MaxAge >= 0 {
			t.Fatal("expected auth cookie to be cleared")
		}
	}
}

func TestEMOMPage(t *testing.T) {
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/timer/emom", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.EMOMPage(c); err != nil {
		t.Fatalf("EMOMPage failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "EMOM") {
		t.Fatalf("expected emom page, got %q", body)
	}
	if !strings.Contains(body, `role="tablist"`) {
		t.Fatalf("expected tablist on emom page, got %q", body)
	}
	if !strings.Contains(body, `id="timers-tab-timer"`) {
		t.Fatalf("expected Timer tab button, got %q", body)
	}
	if !strings.Contains(body, `id="timers-tab-emom"`) {
		t.Fatalf("expected EMOM tab button, got %q", body)
	}
	// EMOM tab must be the active one when on /timer/emom
	timerActiveRegex := regexp.MustCompile(`id="timers-tab-timer"[^>]*aria-selected="true"`)
	emomActiveRegex := regexp.MustCompile(`id="timers-tab-emom"[^>]*aria-selected="true"`)
	if !emomActiveRegex.MatchString(body) {
		t.Fatalf("expected EMOM tab to be active on /timer/emom, got %q", body)
	}
	if timerActiveRegex.MatchString(body) {
		t.Fatalf("expected Timer tab to be inactive on /timer/emom, got %q", body)
	}
}

func TestEMOMValidationError(t *testing.T) {
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/timer/emom/error", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.EMOMValidationError(c); err != nil {
		t.Fatalf("EMOMValidationError failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Invalid Rounds") {
		t.Fatalf("expected toast error, got %q", body)
	}
}

func TestEMOMRoundToast(t *testing.T) {
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("round", "3")

	req := httptest.NewRequest(http.MethodPost, "/timer/emom/round", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.EMOMRoundToast(c); err != nil {
		t.Fatalf("EMOMRoundToast failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Round 3 Complete") {
		t.Fatalf("expected toast with round number, got %q", body)
	}
}