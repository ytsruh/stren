package routes

import (
	"context"
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
	"stren/internal/views"
)

type mockRepository struct {
	mu        sync.Mutex
	exercises []models.Exercise
	entries   []models.ExerciseEntry

	errCreate                       error
	errGetByName                    error
	errGetByID                      error
	errUpdate                       error
	errList                         error
	errCreateEntry                  error
	errGetEntry                     error
	errUpdateEntry                  error
	errUpdateEntryWithDate          error
	errDeleteEntry                  error
	errListEntries                  error
	errGetEntriesByExercisePaginated error
	errGetEntriesByDateRange        error
	errGetExerciseByID              error
	errGetMaxWeightByExercise       error
	errGetLastSetByExercise         error
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

func (m *mockRepository) GetEntriesByExercisePaginated(exerciseID string, userID string, limit, offset int) ([]models.ExerciseEntry, error) {
	if m.errGetEntriesByExercisePaginated != nil {
		return nil, m.errGetEntriesByExercisePaginated
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var all []models.ExerciseEntry
	for _, e := range m.entries {
		if e.ExerciseID == exerciseID && e.UserID == userID {
			all = append(all, e)
		}
	}
	if offset >= len(all) {
		return []models.ExerciseEntry{}, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}

func (m *mockRepository) GetMaxWeightByExercise(exerciseID string, userID string) (float64, error) {
	if m.errGetMaxWeightByExercise != nil {
		return 0, m.errGetMaxWeightByExercise
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var max float64
	for _, e := range m.entries {
		if e.ExerciseID == exerciseID && e.UserID == userID && e.Weight > max {
			max = e.Weight
		}
	}
	return max, nil
}

func (m *mockRepository) GetLastSetByExercise(exerciseID string, userID string) (*models.ExerciseEntry, error) {
	if m.errGetLastSetByExercise != nil {
		return nil, m.errGetLastSetByExercise
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var last *models.ExerciseEntry
	for i, e := range m.entries {
		if e.ExerciseID != exerciseID || e.UserID != userID {
			continue
		}
		if last == nil || e.CreatedAt.After(last.CreatedAt) {
			cp := m.entries[i]
			last = &cp
		}
	}
	return last, nil
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

// mockPushSubscriptionRepository satisfies the models.PushSubscriptionRepo
// interface for the route tests. The push routes only use a subset of
// the methods, but every method on the interface is implemented so
// the compile-time check holds.
type mockPushSubscriptionRepository struct {
	mu    sync.Mutex
	rows  map[string]models.PushSubscription
	errOp error
}

func newMockPushSubscriptionRepository() *mockPushSubscriptionRepository {
	return &mockPushSubscriptionRepository{rows: map[string]models.PushSubscription{}}
}

func (m *mockPushSubscriptionRepository) UpsertForUser(_ context.Context, userID string, sub models.PushSubscription) (*models.PushSubscription, error) {
	if m.errOp != nil {
		return nil, m.errOp
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.rows[sub.Endpoint]; ok {
		existing.P256dh = sub.P256dh
		existing.Auth = sub.Auth
		m.rows[sub.Endpoint] = existing
		out := existing
		return &out, nil
	}
	row := sub
	row.UserID = userID
	m.rows[sub.Endpoint] = row
	out := row
	return &out, nil
}

func (m *mockPushSubscriptionRepository) ListForUser(_ context.Context, userID string) ([]models.PushSubscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []models.PushSubscription{}
	for _, r := range m.rows {
		if r.UserID == userID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockPushSubscriptionRepository) ListAll(_ context.Context) ([]models.PushSubscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]models.PushSubscription, 0, len(m.rows))
	for _, r := range m.rows {
		out = append(out, r)
	}
	return out, nil
}

func (m *mockPushSubscriptionRepository) DeleteByEndpoint(_ context.Context, endpoint string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rows, endpoint)
	return nil
}

func (m *mockPushSubscriptionRepository) CountForUser(_ context.Context, userID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var n int64
	for _, r := range m.rows {
		if r.UserID == userID {
			n++
		}
	}
	return n, nil
}

func setupHandler(t *testing.T) (*Handler, *mockRepository, *mockUserRepository, *echo.Echo) {
	t.Helper()
	e := echo.New()
	mock := newMockRepository()
	mockUser := newMockUserRepository()
	mockAdminUser := newMockAdminUserRepository()
	mockFeedback := newMockFeedbackRepository()
	mockWeight := newMockWeightRepository()
	mockPush := newMockPushSubscriptionRepository()
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
	pushCtrl := controllers.NewPushController(mockPush)
	adminNotificationsCtrl := controllers.NewAdminNotificationsController(nil)
	validator := utils.NewValidator()
	h := NewHandler(
		authCtrl, entryCtrl, adminCtrl, adminUserCtrl,
		feedbackCtrl, timersCtrl, weightCtrl,
		pushCtrl, adminNotificationsCtrl,
		mockPush, "", false,
		mockUser, jwtService, validator,
	)
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
	form.Set("sets[0][reps]", "5")
	form.Set("sets[0][weight]", "100")

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
	form.Set("sets[0][reps]", "5")
	form.Set("sets[0][weight]", "100")

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

func TestCreateEntry_HTMX_MultipleSets(t *testing.T) {
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_id", "ex-1")
	form.Set("sets[0][reps]", "5")
	form.Set("sets[0][weight]", "100")
	form.Set("sets[1][reps]", "5")
	form.Set("sets[1][weight]", "100")
	form.Set("sets[2][reps]", "5")
	form.Set("sets[2][weight]", "95")

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
	body := rec.Body.String()
	if !strings.Contains(body, "3 sets saved") {
		t.Fatalf("expected toast with '3 sets saved', got body: %s", body)
	}
}

func TestCreateEntry_InvalidReps(t *testing.T) {
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_id", "ex-1")
	form.Set("sets[0][reps]", "abc")
	form.Set("sets[0][weight]", "100")

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
	if !strings.Contains(body, "reps") {
		t.Fatalf("expected error message containing 'reps', got %q", body)
	}
}

func TestCreateEntry_InvalidWeight(t *testing.T) {
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_id", "ex-1")
	form.Set("sets[0][reps]", "5")
	form.Set("sets[0][weight]", "-10")

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

func TestCreateEntry_NoSets(t *testing.T) {
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_id", "ex-1")
	// sets[0][reps] is intentionally absent — user submitted no valid sets

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
	if !strings.Contains(body, "at least one set") {
		t.Fatalf("expected error message about sets, got %q", body)
	}
}

func TestCreateEntry_TooManySets(t *testing.T) {
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_id", "ex-1")
	// MaxSetsPerEntry + 1 rows
	for i := 0; i <= views.MaxSetsPerEntry; i++ {
		form.Set(fmt.Sprintf("sets[%d][reps]", i), "5")
		form.Set(fmt.Sprintf("sets[%d][weight]", i), "100")
	}

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
	if !strings.Contains(body, "Maximum") {
		t.Fatalf("expected error message about max sets, got %q", body)
	}
}

func TestCreateEntry_MissingExercise(t *testing.T) {
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_id", "")
	form.Set("sets[0][reps]", "5")
	form.Set("sets[0][weight]", "100")

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
	form.Set("sets[0][reps]", "5")
	form.Set("sets[0][weight]", "100")

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

// TestCreateEntry_HTMX_WithCustomDate verifies that a created_at value
// submitted via the form is parsed (UTC, matching the edit form) and stored
// on the persisted row. The test fixes a back-dated timestamp that is far
// from "now" so the assertion can't accidentally pass via the time.Now()
// fallback.
func TestCreateEntry_HTMX_WithCustomDate(t *testing.T) {
	h, mock, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_id", "ex-1")
	form.Set("created_at", "2024-06-15T14:30")
	form.Set("sets[0][reps]", "5")
	form.Set("sets[0][weight]", "100")
	form.Set("sets[1][reps]", "5")
	form.Set("sets[1][weight]", "95")

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

	want := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	if len(mock.entries) != 2 {
		t.Fatalf("expected 2 entries in mock, got %d", len(mock.entries))
	}
	for i, got := range mock.entries {
		if !got.CreatedAt.Equal(want) {
			t.Errorf("entry %d: expected CreatedAt %v, got %v", i, want, got.CreatedAt)
		}
	}
}

// TestCreateEntry_HTMX_EmptyDateFallsBackToNow verifies that an absent
// created_at field falls back to time.Now() — this is the "default to now"
// behaviour when a user submits the form without touching the date input.
func TestCreateEntry_HTMX_EmptyDateFallsBackToNow(t *testing.T) {
	h, mock, _, e := setupHandler(t)

	before := time.Now()

	form := url.Values{}
	form.Set("exercise_id", "ex-1")
	// created_at intentionally omitted
	form.Set("sets[0][reps]", "5")
	form.Set("sets[0][weight]", "100")

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

	after := time.Now()
	if len(mock.entries) != 1 {
		t.Fatalf("expected 1 entry in mock, got %d", len(mock.entries))
	}
	got := mock.entries[0].CreatedAt
	// Allow a 5-second window for the fallback to be considered "now".
	if got.Before(before.Add(-5*time.Second)) || got.After(after.Add(5*time.Second)) {
		t.Errorf("expected CreatedAt near now (%v..%v), got %v", before, after, got)
	}
}

// TestCreateEntry_InvalidDateFormat verifies that a malformed created_at
// value produces a user-visible error rather than silently being ignored.
func TestCreateEntry_InvalidDateFormat(t *testing.T) {
	h, mock, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("exercise_id", "ex-1")
	form.Set("created_at", "not-a-date")
	form.Set("sets[0][reps]", "5")
	form.Set("sets[0][weight]", "100")

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
		t.Fatalf("expected status 200 (rendered error), got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Invalid date format") {
		t.Errorf("expected error message 'Invalid date format', got %q", body)
	}
	if len(mock.entries) != 0 {
		t.Errorf("expected no entries persisted on parse error, got %d", len(mock.entries))
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

func TestExerciseHistory_PageQuery(t *testing.T) {
	h, mock, _, e := setupHandler(t)
	mock.exercises = []models.Exercise{
		{ID: "uuid-exercise-1", Name: "Squat"},
	}
	now := time.Now()
	var entries []models.ExerciseEntry
	for i := 0; i < 30; i++ {
		entries = append(entries, models.ExerciseEntry{
			ID:         fmt.Sprintf("entry-%d", i),
			UserID:     "user-1",
			ExerciseID: "uuid-exercise-1",
			Reps:       5,
			Weight:     float64(100 + i),
			CreatedAt:  now.Add(time.Duration(i) * time.Minute),
		})
	}
	mock.entries = entries

	req := httptest.NewRequest(http.MethodGet, "/exercises/uuid-exercise-1?page=2", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("uuid-exercise-1")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.ExerciseHistory(c); err != nil {
		t.Fatalf("ExerciseHistory (?page=2) failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Page 2") {
		t.Error("expected 'Page 2' indicator in full-page response")
	}
}

func TestExerciseHistory_HtmxFragment(t *testing.T) {
	h, mock, _, e := setupHandler(t)
	mock.exercises = []models.Exercise{
		{ID: "uuid-exercise-1", Name: "Squat"},
	}
	now := time.Now()
	var entries []models.ExerciseEntry
	for i := 0; i < 30; i++ {
		entries = append(entries, models.ExerciseEntry{
			ID:         fmt.Sprintf("entry-%d", i),
			UserID:     "user-1",
			ExerciseID: "uuid-exercise-1",
			Reps:       5,
			Weight:     float64(100 + i),
			CreatedAt:  now.Add(time.Duration(i) * time.Minute),
		})
	}
	mock.entries = entries

	req := httptest.NewRequest(http.MethodGet, "/exercises/uuid-exercise-1?page=2", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("uuid-exercise-1")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.ExerciseHistory(c); err != nil {
		t.Fatalf("ExerciseHistory (htmx) failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if vary := rec.Header().Get("Vary"); !strings.Contains(vary, "HX-Request") {
		t.Errorf("expected Vary: HX-Request header, got %q", vary)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "history-table-wrap") {
		t.Error("expected fragment to contain swappable wrap id")
	}
	// Fragment must NOT include the full page chrome.
	if strings.Contains(body, "<html") {
		t.Error("fragment response should not include the full HTML document")
	}
	if !strings.Contains(body, "Page 2") {
		t.Error("expected 'Page 2' indicator inside the fragment")
	}
}

// TestExerciseChart verifies the chart placeholder route returns a 200
// with the expected sub-view links and the chart sub-view marked as
// active. Mirrors the TestExerciseHistory happy-path pattern.
func TestExerciseChart(t *testing.T) {
	h, mock, _, e := setupHandler(t)
	mock.exercises = []models.Exercise{
		{ID: "uuid-exercise-1", Name: "Squat"},
	}

	req := httptest.NewRequest(http.MethodGet, "/exercises/uuid-exercise-1/chart", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("uuid-exercise-1")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.ExerciseChart(c); err != nil {
		t.Fatalf("ExerciseChart failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `href="/exercises/uuid-exercise-1/chart"`) {
		t.Error("expected Chart link in button group")
	}
	if !strings.Contains(body, `href="/exercises/uuid-exercise-1/chart" class="btn" aria-current="page"`) {
		t.Error("expected Chart link to be the active sub-view")
	}
	if !strings.Contains(body, `href="/exercises/uuid-exercise-1/chart/advanced"`) {
		t.Error("expected Advanced link in button group")
	}
	if !strings.Contains(body, `href="/exercises/uuid-exercise-1"`) {
		t.Error("expected History (back) link in button group")
	}
}

// TestExerciseChart_NotFound asserts the chart route returns a 404 when
// the exercise ID does not match a row the user can see. We surface this
// explicitly (rather than letting an unhandled nil panic) so a bad URL
// fails cleanly in the browser.
func TestExerciseChart_NotFound(t *testing.T) {
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/exercises/missing/chart", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("missing")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	err := h.ExerciseChart(c)
	if err == nil {
		t.Fatal("expected error for missing exercise")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusNotFound {
		t.Fatalf("expected not found error, got %v", err)
	}
}

// TestExerciseChart_Populated asserts the dedicated chart route renders
// the full-width chart card and the dedicated exercise-chart canvas
// when the user has at least 2 unique days of data. Multiple sets on
// the same day are aggregated to a single point on the line.
func TestExerciseChart_Populated(t *testing.T) {
	h, mock, _, e := setupHandler(t)
	mock.exercises = []models.Exercise{
		{ID: "uuid-exercise-1", Name: "Squat"},
	}
	day1 := time.Date(2025, 6, 10, 9, 0, 0, 0, time.UTC)
	day2 := time.Date(2025, 6, 17, 9, 0, 0, 0, time.UTC)
	mock.entries = []models.ExerciseEntry{
		{ID: "e1", ExerciseID: "uuid-exercise-1", UserID: "user-1", Reps: 5, Weight: 100, CreatedAt: day1},
		{ID: "e2", ExerciseID: "uuid-exercise-1", UserID: "user-1", Reps: 3, Weight: 105, CreatedAt: day1.Add(8 * time.Hour)},
		{ID: "e3", ExerciseID: "uuid-exercise-1", UserID: "user-1", Reps: 5, Weight: 110, CreatedAt: day2},
		{ID: "e4", ExerciseID: "uuid-exercise-1", UserID: "user-1", Reps: 3, Weight: 108, CreatedAt: day2.Add(8 * time.Hour)},
		// Other-exercise entries must be ignored.
		{ID: "e5", ExerciseID: "uuid-exercise-other", UserID: "user-1", Reps: 5, Weight: 200, CreatedAt: day1},
		// Other-user entries must be ignored.
		{ID: "e6", ExerciseID: "uuid-exercise-1", UserID: "user-other", Reps: 5, Weight: 999, CreatedAt: day1},
	}

	req := httptest.NewRequest(http.MethodGet, "/exercises/uuid-exercise-1/chart", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("uuid-exercise-1")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.ExerciseChart(c); err != nil {
		t.Fatalf("ExerciseChart failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// Page header copy.
	if !strings.Contains(body, "Squat Progression") {
		t.Error("expected page header with exercise name + Progression")
	}
	if !strings.Contains(body, "Max weight per training day") {
		t.Error("expected subtitle describing the chart aggregation")
	}
	// Full-width chart card chrome.
	if !strings.Contains(body, `<div class="card p-4">`) {
		t.Error("expected full-width card container")
	}
	if !strings.Contains(body, `class="h-[60vh] min-h-96"`) {
		t.Error("expected tall fixed-height chart wrapper")
	}
	// Distinct canvas id and JSON data block.
	if !strings.Contains(body, `<canvas id="exercise-chart">`) {
		t.Error("expected canvas with id exercise-chart")
	}
	if !strings.Contains(body, `id="exercise-chart-data"`) {
		t.Error("expected exercise-chart-data JSON payload")
	}
	// Dataset label reflects the exercise name.
	if !strings.Contains(body, "Squat (kg)") {
		t.Error("expected dataset label to include the exercise name + (kg)")
	}
	// Aggregation: only 2 unique days -> 2 chart points. Heaviest per
	// day wins, so day1 = 105 and day2 = 110.
	re := regexp.MustCompile(`<script id="exercise-chart-data" type="application/json">([\s\S]*?)</script>`)
	m := re.FindStringSubmatch(body)
	if len(m) < 2 {
		t.Fatal("could not find exercise-chart-data script block")
	}
	var parsed struct {
		Labels   []string `json:"labels"`
		Datasets []struct {
			Values []float64 `json:"values"`
		} `json:"datasets"`
	}
	if err := json.Unmarshal([]byte(m[1]), &parsed); err != nil {
		t.Fatalf("data block is not valid JSON: %v\ncontent: %s", err, m[1])
	}
	if len(parsed.Labels) != 2 {
		t.Errorf("expected 2 day-bucketed labels, got %d (%v)", len(parsed.Labels), parsed.Labels)
	}
	if len(parsed.Datasets) != 1 {
		t.Fatalf("expected 1 dataset, got %d", len(parsed.Datasets))
	}
	if len(parsed.Datasets[0].Values) != 2 || parsed.Datasets[0].Values[0] != 105 || parsed.Datasets[0].Values[1] != 110 {
		t.Errorf("expected max-weight-per-day values [105, 110], got %v", parsed.Datasets[0].Values)
	}
	// Empty state must NOT be rendered in the populated case.
	if strings.Contains(body, "Log at least 2 sessions") {
		t.Error("did not expect empty-state message in populated chart view")
	}
}

// TestExerciseChart_EmptyState asserts the empty-state message is
// rendered (and the chart canvas is not) when the user has fewer than
// 2 unique days of data for the exercise.
func TestExerciseChart_EmptyState(t *testing.T) {
	h, mock, _, e := setupHandler(t)
	mock.exercises = []models.Exercise{
		{ID: "uuid-exercise-1", Name: "Squat"},
	}
	// Single entry, single day — below the 2-day threshold.
	mock.entries = []models.ExerciseEntry{
		{ID: "e1", ExerciseID: "uuid-exercise-1", UserID: "user-1", Reps: 5, Weight: 100, CreatedAt: time.Date(2025, 6, 10, 9, 0, 0, 0, time.UTC)},
	}

	req := httptest.NewRequest(http.MethodGet, "/exercises/uuid-exercise-1/chart", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("uuid-exercise-1")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.ExerciseChart(c); err != nil {
		t.Fatalf("ExerciseChart failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "Log at least 2 sessions to see your progression.") {
		t.Error("expected friendly empty-state message in the empty-state route test")
	}
	if strings.Contains(body, `<canvas id="exercise-chart">`) {
		t.Error("did not expect chart canvas in empty-state route response")
	}
	// The page header copy should still be rendered so the user
	// understands which exercise the empty state applies to.
	if !strings.Contains(body, "Squat Progression") {
		t.Error("expected page header copy even in the empty state")
	}
}

// TestExerciseChartAdvanced asserts the dedicated advanced route returns
// 200 OK, the Advanced link is marked active, the line chart link is
// NOT marked active, and all three sub-view links are present in the
// shared button group.
//
// Two sets of data are passed so the chart (and its subtitle) is
// rendered. The empty-state case is covered by
// TestExerciseChartAdvanced_EmptyState.
func TestExerciseChartAdvanced(t *testing.T) {
	h, mock, _, e := setupHandler(t)
	mock.exercises = []models.Exercise{
		{ID: "uuid-exercise-1", Name: "Squat"},
	}
	day1 := time.Date(2025, 6, 10, 9, 0, 0, 0, time.UTC)
	day2 := time.Date(2025, 6, 17, 9, 0, 0, 0, time.UTC)
	mock.entries = []models.ExerciseEntry{
		{ID: "e1", ExerciseID: "uuid-exercise-1", UserID: "user-1", Reps: 5, Weight: 100, CreatedAt: day1},
		{ID: "e2", ExerciseID: "uuid-exercise-1", UserID: "user-1", Reps: 5, Weight: 105, CreatedAt: day2},
	}

	req := httptest.NewRequest(http.MethodGet, "/exercises/uuid-exercise-1/chart/advanced", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("uuid-exercise-1")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.ExerciseChartAdvanced(c); err != nil {
		t.Fatalf("ExerciseChartAdvanced failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `href="/exercises/uuid-exercise-1/chart/advanced" class="btn" aria-current="page"`) {
		t.Error("expected Advanced link to be the active sub-view")
	}
	if !strings.Contains(body, `href="/exercises/uuid-exercise-1/chart"`) {
		t.Error("expected Chart link in button group")
	}
	if !strings.Contains(body, `href="/exercises/uuid-exercise-1"`) {
		t.Error("expected History (back) link in button group")
	}
	// Page header copy is rendered on the advanced view too.
	if !strings.Contains(body, "Squat Volume") {
		t.Error("expected page header with exercise name + Volume")
	}
	if !strings.Contains(body, "Every set plotted by reps and weight") {
		t.Error("expected subtitle describing the scatter plot")
	}
	// Scatter canvas must be present (2 entries -> 2 dots).
	if !strings.Contains(body, `<canvas id="exercise-chart-advanced">`) {
		t.Error("expected scatter canvas when entries exist")
	}
	if strings.Contains(body, "Log at least 2 sets to see your volume profile.") {
		t.Error("did not expect empty-state message when entries exist")
	}
}

// TestExerciseChartAdvanced_Populated asserts the dedicated advanced
// route renders the full-width scatter card and the dedicated
// exercise-chart-advanced canvas when the user has at least 2 sets of
// data. Each set is plotted as its own (reps, weight) point — no
// per-day collapse.
func TestExerciseChartAdvanced_Populated(t *testing.T) {
	h, mock, _, e := setupHandler(t)
	mock.exercises = []models.Exercise{
		{ID: "uuid-exercise-1", Name: "Squat"},
	}
	day1 := time.Date(2025, 6, 10, 9, 0, 0, 0, time.UTC)
	day2 := time.Date(2025, 6, 17, 9, 0, 0, 0, time.UTC)
	mock.entries = []models.ExerciseEntry{
		{ID: "e1", ExerciseID: "uuid-exercise-1", UserID: "user-1", Reps: 5, Weight: 100, CreatedAt: day1},
		{ID: "e2", ExerciseID: "uuid-exercise-1", UserID: "user-1", Reps: 3, Weight: 110, CreatedAt: day1.Add(8 * time.Hour)},
		{ID: "e3", ExerciseID: "uuid-exercise-1", UserID: "user-1", Reps: 5, Weight: 115, CreatedAt: day2},
		// Other-exercise entries must be ignored.
		{ID: "e4", ExerciseID: "uuid-exercise-other", UserID: "user-1", Reps: 5, Weight: 200, CreatedAt: day1},
		// Other-user entries must be ignored.
		{ID: "e5", ExerciseID: "uuid-exercise-1", UserID: "user-other", Reps: 5, Weight: 999, CreatedAt: day1},
	}

	req := httptest.NewRequest(http.MethodGet, "/exercises/uuid-exercise-1/chart/advanced", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("uuid-exercise-1")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.ExerciseChartAdvanced(c); err != nil {
		t.Fatalf("ExerciseChartAdvanced failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// Full-width scatter card chrome.
	if !strings.Contains(body, `<div class="card p-4">`) {
		t.Error("expected full-width card container")
	}
	if !strings.Contains(body, `class="h-[60vh] min-h-96"`) {
		t.Error("expected tall fixed-height scatter wrapper")
	}
	// Distinct canvas id from the line chart.
	if !strings.Contains(body, `<canvas id="exercise-chart-advanced">`) {
		t.Error("expected canvas with id exercise-chart-advanced")
	}
	if strings.Contains(body, `<canvas id="exercise-chart">`) {
		t.Error("did not expect the line-chart canvas id on the advanced view")
	}
	if !strings.Contains(body, `id="exercise-chart-advanced-data"`) {
		t.Error("expected exercise-chart-advanced-data JSON payload")
	}
	// Y-axis label reflects the exercise name.
	if !strings.Contains(body, "Squat (kg)") {
		t.Error("expected y-axis label to include the exercise name + (kg)")
	}
	// No per-day collapse: 3 owned sets -> 3 scatter points.
	re := regexp.MustCompile(`<script id="exercise-chart-advanced-data" type="application/json">([\s\S]*?)</script>`)
	m := re.FindStringSubmatch(body)
	if len(m) < 2 {
		t.Fatal("could not find exercise-chart-advanced-data script block")
	}
	var parsed struct {
		XLabel   string `json:"xLabel"`
		YLabel   string `json:"yLabel"`
		Points   []struct {
			X    float64 `json:"x"`
			Y    float64 `json:"y"`
			Date string  `json:"date"`
		} `json:"points"`
		HideAxes bool `json:"hideAxes"`
	}
	if err := json.Unmarshal([]byte(m[1]), &parsed); err != nil {
		t.Fatalf("data block is not valid JSON: %v\ncontent: %s", err, m[1])
	}
	if parsed.XLabel != "Reps" {
		t.Errorf("expected xLabel 'Reps', got %q", parsed.XLabel)
	}
	if parsed.YLabel != "Squat (kg)" {
		t.Errorf("expected yLabel 'Squat (kg)', got %q", parsed.YLabel)
	}
	if parsed.HideAxes {
		t.Error("expected hideAxes=false at full width so axis tick labels are visible")
	}
	if len(parsed.Points) != 3 {
		t.Errorf("expected 3 per-set scatter points, got %d", len(parsed.Points))
	}
	// Empty state must NOT be rendered in the populated case.
	if strings.Contains(body, "Log at least 2 sets") {
		t.Error("did not expect empty-state message in populated advanced view")
	}
}

// TestExerciseChartAdvanced_EmptyState asserts the empty-state message
// is rendered (and the scatter canvas is not) when the user has fewer
// than 2 sets for the exercise.
func TestExerciseChartAdvanced_EmptyState(t *testing.T) {
	h, mock, _, e := setupHandler(t)
	mock.exercises = []models.Exercise{
		{ID: "uuid-exercise-1", Name: "Squat"},
	}
	// Single set — below the 2-set threshold.
	mock.entries = []models.ExerciseEntry{
		{ID: "e1", ExerciseID: "uuid-exercise-1", UserID: "user-1", Reps: 5, Weight: 100, CreatedAt: time.Date(2025, 6, 10, 9, 0, 0, 0, time.UTC)},
	}

	req := httptest.NewRequest(http.MethodGet, "/exercises/uuid-exercise-1/chart/advanced", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("uuid-exercise-1")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.ExerciseChartAdvanced(c); err != nil {
		t.Fatalf("ExerciseChartAdvanced failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "Log at least 2 sets to see your volume profile.") {
		t.Error("expected friendly empty-state message in the empty-state route test")
	}
	if strings.Contains(body, `<canvas id="exercise-chart-advanced">`) {
		t.Error("did not expect scatter canvas in empty-state route response")
	}
	// The page header copy should still be rendered so the user
	// understands which exercise the empty state applies to.
	if !strings.Contains(body, "Squat Volume") {
		t.Error("expected page header copy even in the empty state")
	}
}

// TestExerciseChartAdvanced_NotFound asserts the advanced route also
// returns 404 for unknown exercise IDs.
func TestExerciseChartAdvanced_NotFound(t *testing.T) {
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/exercises/missing/chart/advanced", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("missing")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	err := h.ExerciseChartAdvanced(c)
	if err == nil {
		t.Fatal("expected error for missing exercise")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusNotFound {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestParsePage(t *testing.T) {
	tests := []struct {
		raw      string
		expected int
	}{
		{"", 1},
		{"0", 1},
		{"-5", 1},
		{"abc", 1},
		{"1", 1},
		{"3", 3},
		{"42", 42},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			if got := parsePage(tt.raw); got != tt.expected {
				t.Errorf("parsePage(%q) = %d, want %d", tt.raw, got, tt.expected)
			}
		})
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

func TestParseEntrySets_Single(t *testing.T) {
	form := url.Values{}
	form.Set("sets[0][reps]", "5")
	form.Set("sets[0][weight]", "100")
	form.Set("sets[0][rest_time]", "90")

	c := newEchoContextWithForm(t, form)
	sets, err := parseEntrySets(c, utils.NewValidator())
	if err != nil {
		t.Fatalf("parseEntrySets failed: %v", err)
	}
	if len(sets) != 1 {
		t.Fatalf("expected 1 set, got %d", len(sets))
	}
	if sets[0].Reps != 5 || sets[0].Weight != 100 || sets[0].RestTime != 90 {
		t.Errorf("unexpected set: %+v", sets[0])
	}
}

func TestParseEntrySets_Multiple(t *testing.T) {
	form := url.Values{}
	form.Set("sets[0][reps]", "5")
	form.Set("sets[0][weight]", "100")
	form.Set("sets[1][reps]", "5")
	form.Set("sets[1][weight]", "100")
	form.Set("sets[2][reps]", "5")
	form.Set("sets[2][weight]", "95")

	c := newEchoContextWithForm(t, form)
	sets, err := parseEntrySets(c, utils.NewValidator())
	if err != nil {
		t.Fatalf("parseEntrySets failed: %v", err)
	}
	if len(sets) != 3 {
		t.Fatalf("expected 3 sets, got %d", len(sets))
	}
	if sets[2].Weight != 95 {
		t.Errorf("expected last set weight 95, got %v", sets[2].Weight)
	}
}

func TestParseEntrySets_SkipsEmptyRows(t *testing.T) {
	form := url.Values{}
	form.Set("sets[0][reps]", "5")
	form.Set("sets[0][weight]", "100")
	// sets[1] intentionally empty (no reps)
	form.Set("sets[2][reps]", "5")
	form.Set("sets[2][weight]", "100")

	c := newEchoContextWithForm(t, form)
	sets, err := parseEntrySets(c, utils.NewValidator())
	if err != nil {
		t.Fatalf("parseEntrySets failed: %v", err)
	}
	if len(sets) != 2 {
		t.Fatalf("expected 2 sets (empty row skipped), got %d", len(sets))
	}
}

func TestParseEntrySets_PreservesOrder(t *testing.T) {
	// Submit out of order to verify server sorts by index.
	form := url.Values{}
	form.Set("sets[2][reps]", "5")
	form.Set("sets[2][weight]", "90")
	form.Set("sets[0][reps]", "5")
	form.Set("sets[0][weight]", "100")
	form.Set("sets[1][reps]", "5")
	form.Set("sets[1][weight]", "95")

	c := newEchoContextWithForm(t, form)
	sets, err := parseEntrySets(c, utils.NewValidator())
	if err != nil {
		t.Fatalf("parseEntrySets failed: %v", err)
	}
	if len(sets) != 3 {
		t.Fatalf("expected 3 sets, got %d", len(sets))
	}
	expected := []float64{100, 95, 90}
	for i, want := range expected {
		if sets[i].Weight != want {
			t.Errorf("set %d: expected weight %v, got %v", i, want, sets[i].Weight)
		}
	}
}

func TestParseEntrySets_NoSets(t *testing.T) {
	form := url.Values{}
	c := newEchoContextWithForm(t, form)
	sets, err := parseEntrySets(c, utils.NewValidator())
	if err != nil {
		t.Fatalf("parseEntrySets failed: %v", err)
	}
	if len(sets) != 0 {
		t.Errorf("expected 0 sets, got %d", len(sets))
	}
}

func TestParseEntrySets_EmptyRepsTreatedAsBlank(t *testing.T) {
	form := url.Values{}
	form.Set("sets[0][reps]", " ")
	form.Set("sets[0][weight]", "100")

	c := newEchoContextWithForm(t, form)
	sets, err := parseEntrySets(c, utils.NewValidator())
	if err != nil {
		t.Fatalf("parseEntrySets failed: %v", err)
	}
	if len(sets) != 0 {
		t.Errorf("expected whitespace reps to be skipped, got %d sets", len(sets))
	}
}

func TestParseEntrySets_TooManySets(t *testing.T) {
	form := url.Values{}
	// Submit MaxSetsPerEntry + 1 rows (all non-empty) to trigger the cap.
	for i := 0; i <= views.MaxSetsPerEntry; i++ {
		form.Set(fmt.Sprintf("sets[%d][reps]", i), "5")
		form.Set(fmt.Sprintf("sets[%d][weight]", i), "100")
	}

	c := newEchoContextWithForm(t, form)
	_, err := parseEntrySets(c, utils.NewValidator())
	if err == nil {
		t.Fatal("expected error for too many sets")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %v", err)
	}
	if !strings.Contains(httpErr.Message.(string), "Maximum") {
		t.Errorf("expected 'Maximum' in error, got %q", httpErr.Message)
	}
}

func TestParseEntrySets_EmptyRowsCountTowardCap(t *testing.T) {
	// A malicious client could send sets[0..99999] with empty reps. Even though
	// parseEntrySets would skip them on processing, the count of submitted
	// indices should still trip the cap before we waste cycles.
	form := url.Values{}
	for i := 0; i <= views.MaxSetsPerEntry; i++ {
		form.Set(fmt.Sprintf("sets[%d][weight]", i), "100")
		// No reps on any row.
	}

	c := newEchoContextWithForm(t, form)
	_, err := parseEntrySets(c, utils.NewValidator())
	if err == nil {
		t.Fatal("expected error when too many empty rows submitted")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %v", err)
	}
}

func TestParseEntrySets_InvalidReps(t *testing.T) {
	form := url.Values{}
	form.Set("sets[0][reps]", "abc")
	form.Set("sets[0][weight]", "100")

	c := newEchoContextWithForm(t, form)
	_, err := parseEntrySets(c, utils.NewValidator())
	if err == nil {
		t.Fatal("expected error for non-numeric reps")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %v", err)
	}
	if !strings.Contains(httpErr.Message.(string), "reps") {
		t.Errorf("expected 'reps' in error, got %q", httpErr.Message)
	}
}

func TestParseEntrySets_InvalidWeight(t *testing.T) {
	form := url.Values{}
	form.Set("sets[0][reps]", "5")
	form.Set("sets[0][weight]", "-10")

	c := newEchoContextWithForm(t, form)
	_, err := parseEntrySets(c, utils.NewValidator())
	if err == nil {
		t.Fatal("expected error for negative weight")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %v", err)
	}
	if !strings.Contains(httpErr.Message.(string), "Weight") {
		t.Errorf("expected 'Weight' in error, got %q", httpErr.Message)
	}
}

func TestParseEntrySets_InvalidRestTime(t *testing.T) {
	form := url.Values{}
	form.Set("sets[0][reps]", "5")
	form.Set("sets[0][weight]", "100")
	form.Set("sets[0][rest_time]", "abc")

	c := newEchoContextWithForm(t, form)
	_, err := parseEntrySets(c, utils.NewValidator())
	if err == nil {
		t.Fatal("expected error for non-numeric rest time")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %v", err)
	}
}

func TestParseEntrySets_RestTimeDefaultsToZero(t *testing.T) {
	form := url.Values{}
	form.Set("sets[0][reps]", "5")
	form.Set("sets[0][weight]", "100")
	// rest_time intentionally absent

	c := newEchoContextWithForm(t, form)
	sets, err := parseEntrySets(c, utils.NewValidator())
	if err != nil {
		t.Fatalf("parseEntrySets failed: %v", err)
	}
	if len(sets) != 1 || sets[0].RestTime != 0 {
		t.Errorf("expected rest_time 0, got %+v", sets[0])
	}
}

func TestParseEntrySets_PerRowErrorIncludesIndex(t *testing.T) {
	// Second set is bad — error message should reference set 2 (1-indexed for users).
	form := url.Values{}
	form.Set("sets[0][reps]", "5")
	form.Set("sets[0][weight]", "100")
	form.Set("sets[1][reps]", "5")
	form.Set("sets[1][weight]", "-5")

	c := newEchoContextWithForm(t, form)
	_, err := parseEntrySets(c, utils.NewValidator())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Set 2") {
		t.Errorf("expected 'Set 2' in error, got %q", err.Error())
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