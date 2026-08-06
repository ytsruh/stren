package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"stren/internal/models"
)

// --- Test helpers ---

// apiDo issues a request through the test Echo instance with
// an optional Bearer token and optional JSON body. Returns
// the recorder for assertion. The content-type is forced to
// application/json so the API handlers go down the JSON
// binding path.
func apiDo(t *testing.T, e interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reqBody = bytes.NewReader(raw)
	} else {
		reqBody = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// decodeAPI unmarshals a 2xx body into T. Fails the test on
// a non-matching status or a malformed body so callers only
// have to assert on the decoded shape.
func decodeAPI[T any](t *testing.T, rec *httptest.ResponseRecorder, wantStatus int) T {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, wantStatus, rec.Body.String())
	}
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body: %v (body=%s)", err, rec.Body.String())
	}
	return out
}

// decodeAPIError unmarshals a 4xx/5xx body into an APIError.
func decodeAPIError(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int) APIError {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, wantStatus, rec.Body.String())
	}
	var e APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("decode error body: %v (body=%s)", err, rec.Body.String())
	}
	return e
}

// loginUser is a thin wrapper that seeds a user into the
// in-memory user repo and returns the JWT. The iOS app
// exercises the same "register then login" flow, so the
// tests use the real auth controller rather than hand-rolling
// tokens.
func loginUser(t *testing.T, h *Handler, mockUser *mockUserRepository, email, name string) (string, *models.User) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	mockUser.users = append(mockUser.users, models.User{
		ID:           "user-" + email,
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		WeightUnit:   "kg",
	})
	user, token, err := h.authCtrl.Login(email, "secret123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	return token, user
}

// --- /auth/login ---

func TestAPILogin_Success(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	loginUser(t, h, mockUser, "test@example.com", "Test")

	rec := apiDo(t, e, http.MethodPost, "/api/v1/auth/login", "", LoginRequest{
		Email:    "test@example.com",
		Password: "secret123",
	})
	resp := decodeAPI[AuthResponse](t, rec, http.StatusOK)
	if resp.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if resp.User.Email != "test@example.com" {
		t.Fatalf("user email = %q, want test@example.com", resp.User.Email)
	}
	if resp.User.WeightUnit != "kg" {
		t.Fatalf("user weight_unit = %q, want kg", resp.User.WeightUnit)
	}
}

func TestAPILogin_BadCredentials(t *testing.T) {
	_, _, _, e := setupHandler(t)

	rec := apiDo(t, e, http.MethodPost, "/api/v1/auth/login", "", LoginRequest{
		Email:    "nope@example.com",
		Password: "wrong",
	})
	body := decodeAPIError(t, rec, http.StatusUnauthorized)
	if body.Error == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestAPILogin_ValidationError(t *testing.T) {
	_, _, _, e := setupHandler(t)

	rec := apiDo(t, e, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email": "not-an-email",
	})
	apiErr := decodeAPIError(t, rec, http.StatusBadRequest)
	if apiErr.Error == "" {
		t.Fatal("expected validation error message")
	}
}

// --- /auth/register ---

func TestAPIRegister_Success(t *testing.T) {
	_, _, _, e := setupHandler(t)

	rec := apiDo(t, e, http.MethodPost, "/api/v1/auth/register", "", RegisterRequest{
		Name:     "Alice",
		Email:    "alice@example.com",
		Password: "secret123",
	})
	resp := decodeAPI[AuthResponse](t, rec, http.StatusOK)
	if resp.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if resp.User.Name != "Alice" {
		t.Fatalf("user name = %q, want Alice", resp.User.Name)
	}
}

func TestAPIRegister_DuplicateEmail(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	// Seed an existing user with the same email the test
	// will try to register with. loginUser is the standard
	// seed helper; we discard the returned token because the
	// duplicate test doesn't need it.
	loginUser(t, h, mockUser, "dup@example.com", "Existing")

	rec := apiDo(t, e, http.MethodPost, "/api/v1/auth/register", "", RegisterRequest{
		Name:     "New",
		Email:    "dup@example.com",
		Password: "secret123",
	})
	apiErr := decodeAPIError(t, rec, http.StatusConflict)
	if apiErr.Error == "" {
		t.Fatal("expected conflict error message")
	}
}

// --- /me ---

func TestAPIMe_Success(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, user := loginUser(t, h, mockUser, "me@example.com", "Me")

	rec := apiDo(t, e, http.MethodGet, "/api/v1/me", token, nil)
	dto := decodeAPI[UserDTO](t, rec, http.StatusOK)
	if dto.ID != user.ID {
		t.Fatalf("id = %q, want %q", dto.ID, user.ID)
	}
}

func TestAPIMe_Unauthorized(t *testing.T) {
	_, _, _, e := setupHandler(t)

	// No token — should return JSON 401, not a redirect to /login.
	rec := apiDo(t, e, http.MethodGet, "/api/v1/me", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401, body = %s", rec.Code, rec.Body.String())
	}
	var body APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error != "unauthorized" {
		t.Fatalf("error = %q, want unauthorized", body.Error)
	}
}

func TestAPIMe_BadToken(t *testing.T) {
	_, _, _, e := setupHandler(t)

	rec := apiDo(t, e, http.MethodGet, "/api/v1/me", "this-is-not-a-jwt", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401, body = %s", rec.Code, rec.Body.String())
	}
}

// --- /exercises ---

func TestAPIListExercises(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "ex@example.com", "Ex")

	rec := apiDo(t, e, http.MethodGet, "/api/v1/exercises", token, nil)
	exs := decodeAPI[[]ExerciseDTO](t, rec, http.StatusOK)
	if len(exs) != 2 {
		t.Fatalf("len = %d, want 2", len(exs))
	}
	// Assert the seeded exercises are both present. Ordering
	// is a repository concern (sqlc ORDER BY), not the API
	// surface, so the test should not depend on it.
	names := map[string]bool{}
	for _, e := range exs {
		names[e.Name] = true
		if e.ID == "" {
			t.Fatalf("exercise %q has empty ID", e.Name)
		}
	}
	if !names["Squat"] || !names["Bench Press"] {
		t.Fatalf("missing expected exercise, got %v", names)
	}
}

// --- /exercise-entries ---

func TestAPICreateExerciseEntries_MultiSet(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "create@example.com", "Create")

	rec := apiDo(t, e, http.MethodPost, "/api/v1/exercise-entries", token, CreateExerciseEntriesRequest{
		ExerciseID: "ex-1",
		Notes:      "Felt strong",
		Sets: []CreateSetInput{
			{Reps: 5, Weight: 100, RestTime: 120},
			{Reps: 5, Weight: 100, RestTime: 120},
			{Reps: 5, Weight: 100, RestTime: 120},
		},
	})
	entries := decodeAPI[[]ExerciseEntryDTO](t, rec, http.StatusCreated)
	if len(entries) != 3 {
		t.Fatalf("len = %d, want 3", len(entries))
	}
	for i, entry := range entries {
		if entry.Reps != 5 || entry.Weight != 100 {
			t.Fatalf("entry %d: reps=%d weight=%.1f, want 5/100", i, entry.Reps, entry.Weight)
		}
	}
}

func TestAPICreateExerciseEntries_ValidationError(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "v@example.com", "V")

	// Zero reps is rejected by the validator.
	rec := apiDo(t, e, http.MethodPost, "/api/v1/exercise-entries", token, CreateExerciseEntriesRequest{
		ExerciseID: "ex-1",
		Sets:       []CreateSetInput{{Reps: 0, Weight: 100, RestTime: 0}},
	})
	apiErr := decodeAPIError(t, rec, http.StatusBadRequest)
	if apiErr.Error == "" {
		t.Fatal("expected validation error message")
	}
}

func TestAPICreateExerciseEntries_UnknownExercise(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "u@example.com", "U")

	rec := apiDo(t, e, http.MethodPost, "/api/v1/exercise-entries", token, CreateExerciseEntriesRequest{
		ExerciseID: "does-not-exist",
		Sets:       []CreateSetInput{{Reps: 5, Weight: 100, RestTime: 0}},
	})
	apiErr := decodeAPIError(t, rec, http.StatusBadRequest)
	if !strings.Contains(apiErr.Error, "exercise") {
		t.Fatalf("error = %q, want it to mention exercise", apiErr.Error)
	}
}

func TestAPIListExerciseEntries_DefaultDays(t *testing.T) {
	h, mock, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "l@example.com", "L")

	// Seed two entries: one inside the 7-day window, one outside it.
	now := time.Now()
	mock.exerciseEntries = []models.ExerciseEntry{
		{ID: "e1", UserID: "user-l@example.com", ExerciseID: "ex-1", ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: now.AddDate(0, 0, -2)},
		{ID: "e2", UserID: "user-l@example.com", ExerciseID: "ex-1", ExerciseName: "Squat", Reps: 5, Weight: 90, CreatedAt: now.AddDate(0, 0, -30)},
	}

	rec := apiDo(t, e, http.MethodGet, "/api/v1/exercise-entries", token, nil)
	entries := decodeAPI[[]ExerciseEntryDTO](t, rec, http.StatusOK)
	if len(entries) != 1 {
		t.Fatalf("len = %d, want 1 (only the entry inside the 7-day window)", len(entries))
	}
	if entries[0].ID != "e1" {
		t.Fatalf("id = %q, want e1", entries[0].ID)
	}
}

func TestAPIListExerciseEntries_CustomDays(t *testing.T) {
	h, mock, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "l2@example.com", "L2")

	now := time.Now()
	mock.exerciseEntries = []models.ExerciseEntry{
		{ID: "e1", UserID: "user-l2@example.com", ExerciseID: "ex-1", ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: now.AddDate(0, 0, -2)},
		{ID: "e2", UserID: "user-l2@example.com", ExerciseID: "ex-1", ExerciseName: "Squat", Reps: 5, Weight: 90, CreatedAt: now.AddDate(0, 0, -30)},
	}

	rec := apiDo(t, e, http.MethodGet, "/api/v1/exercise-entries?days=60", token, nil)
	entries := decodeAPI[[]ExerciseEntryDTO](t, rec, http.StatusOK)
	if len(entries) != 2 {
		t.Fatalf("len = %d, want 2", len(entries))
	}
}

func TestAPIListExerciseEntries_BadDays(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "bd@example.com", "BD")

	rec := apiDo(t, e, http.MethodGet, "/api/v1/exercise-entries?days=-1", token, nil)
	apiErr := decodeAPIError(t, rec, http.StatusBadRequest)
	if apiErr.Error == "" {
		t.Fatal("expected validation error")
	}
}

func TestAPIGetExerciseEntry_NotFound(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "nf@example.com", "NF")

	rec := apiDo(t, e, http.MethodGet, "/api/v1/exercise-entries/missing", token, nil)
	decodeAPIError(t, rec, http.StatusNotFound)
}

func TestAPIUpdateExerciseEntry(t *testing.T) {
	h, mock, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "up@example.com", "Up")

	mock.exerciseEntries = []models.ExerciseEntry{
		{ID: "e1", UserID: "user-up@example.com", ExerciseID: "ex-1", ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: time.Now()},
	}

	rec := apiDo(t, e, http.MethodPut, "/api/v1/exercise-entries/e1", token, UpdateExerciseEntryRequest{
		ExerciseID: "ex-1",
		Reps:       8,
		Weight:     110,
		RestTime:   90,
	})
	entry := decodeAPI[ExerciseEntryDTO](t, rec, http.StatusOK)
	if entry.Reps != 8 || entry.Weight != 110 {
		t.Fatalf("reps=%d weight=%.1f, want 8/110", entry.Reps, entry.Weight)
	}
}

func TestAPIDeleteExerciseEntry(t *testing.T) {
	h, mock, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "del@example.com", "Del")

	mock.exerciseEntries = []models.ExerciseEntry{
		{ID: "e1", UserID: "user-del@example.com", ExerciseID: "ex-1", ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: time.Now()},
	}

	rec := apiDo(t, e, http.MethodDelete, "/api/v1/exercise-entries/e1", token, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if len(mock.exerciseEntries) != 0 {
		t.Fatalf("entries len = %d, want 0 after delete", len(mock.exerciseEntries))
	}
}

// --- /exercises/:id/history ---

func TestAPIGetExerciseHistory(t *testing.T) {
	h, mock, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "h@example.com", "H")

	now := time.Now()
	mock.exerciseEntries = []models.ExerciseEntry{
		{ID: "e1", UserID: "user-h@example.com", ExerciseID: "ex-1", ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: now},
		{ID: "e2", UserID: "user-h@example.com", ExerciseID: "ex-1", ExerciseName: "Squat", Reps: 5, Weight: 110, CreatedAt: now.AddDate(0, 0, -1)},
	}

	rec := apiDo(t, e, http.MethodGet, "/api/v1/exercises/ex-1/history", token, nil)
	page := decodeAPI[HistoryPageDTO](t, rec, http.StatusOK)
	if len(page.Entries) != 2 {
		t.Fatalf("entries len = %d, want 2", len(page.Entries))
	}
	if page.Stats.MaxWeight != 110 {
		t.Fatalf("max weight = %.1f, want 110", page.Stats.MaxWeight)
	}
	if page.Page != 1 {
		t.Fatalf("page = %d, want 1", page.Page)
	}
	if page.HasPrev {
		t.Fatal("has_prev should be false on the first page")
	}
}

func TestAPIGetExerciseHistory_NotFound(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "hnf@example.com", "HNF")

	rec := apiDo(t, e, http.MethodGet, "/api/v1/exercises/does-not-exist/history", token, nil)
	decodeAPIError(t, rec, http.StatusNotFound)
}

// --- /exercises/:id/chart ---

func TestAPIGetExerciseChartData(t *testing.T) {
	h, mock, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "c@example.com", "C")

	now := time.Now()
	mock.exerciseEntries = []models.ExerciseEntry{
		{ID: "e1", UserID: "user-c@example.com", ExerciseID: "ex-1", ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: now},
		{ID: "e2", UserID: "user-c@example.com", ExerciseID: "ex-1", ExerciseName: "Squat", Reps: 5, Weight: 110, CreatedAt: now.AddDate(0, 0, -7)},
	}

	rec := apiDo(t, e, http.MethodGet, "/api/v1/exercises/ex-1/chart", token, nil)
	entries := decodeAPI[[]ExerciseEntryDTO](t, rec, http.StatusOK)
	if len(entries) != 2 {
		t.Fatalf("len = %d, want 2", len(entries))
	}
}

// --- Cookie auth (web app) still works for the API ---

func TestAPIAuthMiddleware_CookieFallback(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "ck@example.com", "CK")

	// Send the request with the token in a cookie instead of
	// the Authorization header. The web app and any other
	// cookie-auth client should still be able to call the API.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}
	var dto UserDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &dto); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if dto.Email != "ck@example.com" {
		t.Fatalf("email = %q, want ck@example.com", dto.Email)
	}
}

// --- /auth/logout (stateless) ---

func TestAPILogout(t *testing.T) {
	_, _, _, e := setupHandler(t)

	rec := apiDo(t, e, http.MethodPost, "/api/v1/auth/logout", "", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

// --- Smoke: error bodies are always APIError ---

func TestAPIErrorBodyShape(t *testing.T) {
	// Hit a 401 path and assert the response is a clean
	// APIError JSON, not an HTML error page.
	_, _, _, e := setupHandler(t)

	rec := apiDo(t, e, http.MethodGet, "/api/v1/me", "", nil)
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("401 content-type = %q, want application/json", ct)
	}
	var e401 APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &e401); err != nil {
		t.Fatalf("401 decode: %v (body=%s)", err, rec.Body.String())
	}
	if e401.Error != "unauthorized" {
		t.Fatalf("401 error = %q, want unauthorized", e401.Error)
	}
}

// --- The web app's redirect-to-login is preserved for HTML routes ---

func TestHTMLLogin_StillRedirectsOnMissingToken(t *testing.T) {
	_, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 (redirect to /login)", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Fatalf("location = %q, want /login", loc)
	}
}
