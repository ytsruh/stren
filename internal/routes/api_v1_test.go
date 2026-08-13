package routes

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"stren/internal/controllers"
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

// --- /me PUT (update profile) ---

func TestAPIUpdateMe_Success(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "upd@example.com", "Old Name")

	rec := apiDo(t, e, http.MethodPut, "/api/v1/me", token, UpdateMeRequest{
		Name:         "New Name",
		TargetWeight: ptrFloat(85.5),
		WeightUnit:   "lbs",
	})
	dto := decodeAPI[UserDTO](t, rec, http.StatusOK)
	if dto.Name != "New Name" {
		t.Fatalf("name = %q, want New Name", dto.Name)
	}
	if dto.TargetWeight == nil || *dto.TargetWeight != 85.5 {
		t.Fatalf("target_weight = %v, want 85.5", dto.TargetWeight)
	}
	if dto.WeightUnit != "lbs" {
		t.Fatalf("weight_unit = %q, want lbs", dto.WeightUnit)
	}
	// Email + admin flag must not be writable from this endpoint —
	// they come from the JWT, not the request body.
	if dto.Email != "upd@example.com" {
		t.Fatalf("email = %q, want upd@example.com (unchanged)", dto.Email)
	}
	if dto.IsAdmin {
		t.Fatal("is_admin should be false for a freshly-registered user")
	}
}

func TestAPIUpdateMe_ClearTargetWeight(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "clr@example.com", "Clr")

	// Seed an existing target weight by updating the user
	// directly through the mock. The HTTP path then clears
	// it by sending target_weight: null.
	seeded := 80.0
	for i := range mockUser.users {
		if mockUser.users[i].Email == "clr@example.com" {
			mockUser.users[i].TargetWeight = &seeded
		}
	}

	rec := apiDo(t, e, http.MethodPut, "/api/v1/me", token, map[string]any{
		"name":          "Clr",
		"target_weight": nil,
		"weight_unit":   "kg",
	})
	dto := decodeAPI[UserDTO](t, rec, http.StatusOK)
	if dto.TargetWeight != nil {
		t.Fatalf("target_weight = %v, want nil after clear", *dto.TargetWeight)
	}
}

func TestAPIUpdateMe_DefaultsWeightUnit(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "def@example.com", "Def")

	// Omit weight_unit entirely — the handler should default
	// it to "kg" rather than write an empty string.
	rec := apiDo(t, e, http.MethodPut, "/api/v1/me", token, map[string]any{
		"name": "Def",
	})
	dto := decodeAPI[UserDTO](t, rec, http.StatusOK)
	if dto.WeightUnit != "kg" {
		t.Fatalf("weight_unit = %q, want kg (default)", dto.WeightUnit)
	}
}

func TestAPIUpdateMe_Validation_NameTooShort(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "ns@example.com", "Ns")

	rec := apiDo(t, e, http.MethodPut, "/api/v1/me", token, UpdateMeRequest{
		Name: "A",
	})
	apiErr := decodeAPIError(t, rec, http.StatusBadRequest)
	if apiErr.Error == "" {
		t.Fatal("expected validation error")
	}
}

func TestAPIUpdateMe_Validation_NameTooLong(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "nl@example.com", "Nl")

	rec := apiDo(t, e, http.MethodPut, "/api/v1/me", token, UpdateMeRequest{
		Name: strings.Repeat("x", 101),
	})
	decodeAPIError(t, rec, http.StatusBadRequest)
}

func TestAPIUpdateMe_Validation_BadUnit(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "bu@example.com", "Bu")

	rec := apiDo(t, e, http.MethodPut, "/api/v1/me", token, UpdateMeRequest{
		Name:       "Bu",
		WeightUnit: "stone",
	})
	apiErr := decodeAPIError(t, rec, http.StatusBadRequest)
	if apiErr.Error == "" {
		t.Fatal("expected validation error for unknown unit")
	}
}

func TestAPIUpdateMe_Validation_TargetTooHigh(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "th@example.com", "Th")

	rec := apiDo(t, e, http.MethodPut, "/api/v1/me", token, UpdateMeRequest{
		Name:         "Th",
		TargetWeight: ptrFloat(2000),
	})
	decodeAPIError(t, rec, http.StatusBadRequest)
}

func TestAPIUpdateMe_Validation_TargetNegative(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "tn@example.com", "Tn")

	rec := apiDo(t, e, http.MethodPut, "/api/v1/me", token, UpdateMeRequest{
		Name:         "Tn",
		TargetWeight: ptrFloat(-1),
	})
	decodeAPIError(t, rec, http.StatusBadRequest)
}

func TestAPIUpdateMe_Unauthorized(t *testing.T) {
	_, _, _, e := setupHandler(t)

	// No token — 401 with the same APIError shape /me uses.
	rec := apiDo(t, e, http.MethodPut, "/api/v1/me", "", UpdateMeRequest{
		Name: "Whatever",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401, body = %s", rec.Code, rec.Body.String())
	}
}

// ptrFloat is a tiny helper so the success-path test can
// write `ptrFloat(85.5)` instead of a fresh `f := 85.5; &f`
// dance on every literal.
func ptrFloat(v float64) *float64 { return &v }

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

// --- /goals ---

// goalsResponse mirrors the JSON shape returned by GET /api/v1/goals.
// The iOS client decodes this into a `[GoalDTO]`, but using a
// named struct here keeps the test assertion symmetric with the
// server contract.
type goalsResponse struct {
	Goals []GoalDTO `json:"goals"`
}

func TestAPIListGoals_Empty(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "gle@example.com", "GE")

	rec := apiDo(t, e, http.MethodGet, "/api/v1/goals", token, nil)
	resp := decodeAPI[goalsResponse](t, rec, http.StatusOK)
	if len(resp.Goals) != 0 {
		t.Fatalf("goals len = %d, want 0", len(resp.Goals))
	}
}

func TestAPIListGoals_OrderingActiveThenCompleted(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "glo@example.com", "GO")
	userID := "user-glo@example.com"

	now := time.Now()
	completed := now.Add(-24 * time.Hour)

	// Seed the repo directly so we control the active/completed
	// split and verify the ordering. Swap the controller's
	// repository before any other goal test runs (test order is
	// undefined so each test owns its own swap).
	repo := mustSwapGoalsRepo(t, h, newMockGoalRepository())

	// Insert in random order to prove the response reorders.
	repo.goals["g-done"] = &models.Goal{ID: "g-done", UserID: userID, Title: "Done", CompletedAt: &completed, CreatedAt: now}
	repo.goals["g-b"] = &models.Goal{ID: "g-b", UserID: userID, Title: "B", CreatedAt: now}
	repo.goals["g-a"] = &models.Goal{ID: "g-a", UserID: userID, Title: "A", CreatedAt: now.Add(-time.Hour)}

	rec := apiDo(t, e, http.MethodGet, "/api/v1/goals", token, nil)
	resp := decodeAPI[goalsResponse](t, rec, http.StatusOK)
	if len(resp.Goals) != 3 {
		t.Fatalf("goals len = %d, want 3", len(resp.Goals))
	}
	if resp.Goals[0].CompletedAt != nil {
		t.Fatalf("first goal should be active, got completed_at=%v", resp.Goals[0].CompletedAt)
	}
	if resp.Goals[1].CompletedAt != nil {
		t.Fatalf("second goal should be active, got completed_at=%v", resp.Goals[1].CompletedAt)
	}
	if resp.Goals[2].CompletedAt == nil {
		t.Fatal("third goal should be completed")
	}
}

func TestAPICreateGoal_Success(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "gc@example.com", "GC")

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	target := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	rec := apiDo(t, e, http.MethodPost, "/api/v1/goals", token, CreateGoalRequest{
		Title:       "Bench 100kg",
		Description: "Working sets",
		StartDate:   &start,
		TargetDate:  &target,
	})
	g := decodeAPI[GoalDTO](t, rec, http.StatusCreated)
	if g.ID == "" {
		t.Fatal("expected generated id")
	}
	if g.Title != "Bench 100kg" {
		t.Fatalf("title = %q, want Bench 100kg", g.Title)
	}
	if g.Description != "Working sets" {
		t.Fatalf("description = %q, want Working sets", g.Description)
	}
	if g.TargetDate == nil || !g.TargetDate.Equal(target) {
		t.Fatalf("target_date = %v, want %v", g.TargetDate, target)
	}
	if g.CompletedAt != nil {
		t.Fatalf("completed_at = %v, want nil", g.CompletedAt)
	}
}

func TestAPICreateGoal_MissingTitle(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "gcmt@example.com", "GMT")

	rec := apiDo(t, e, http.MethodPost, "/api/v1/goals", token, CreateGoalRequest{Title: ""})
	e2 := decodeAPIError(t, rec, http.StatusBadRequest)
	if e2.Error == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestAPICreateGoal_TitleTooLong(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "gctl@example.com", "GCTL")

	rec := apiDo(t, e, http.MethodPost, "/api/v1/goals", token, CreateGoalRequest{Title: strings.Repeat("x", 201)})
	decodeAPIError(t, rec, http.StatusBadRequest)
}

func TestAPIGetGoal_Success(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, user := loginUser(t, h, mockUser, "gg@example.com", "GG")

	// Create via the API so we exercise the real write path.
	start := time.Now()
	target := start.AddDate(0, 0, 14)
	created := decodeAPI[GoalDTO](t, apiDo(t, e, http.MethodPost, "/api/v1/goals", token, CreateGoalRequest{
		Title:      "Run a 5k",
		TargetDate: &target,
	}), http.StatusCreated)

	got := decodeAPI[GoalDTO](t, apiDo(t, e, http.MethodGet, "/api/v1/goals/"+created.ID, token, nil), http.StatusOK)
	if got.ID != created.ID {
		t.Fatalf("id = %q, want %q", got.ID, created.ID)
	}
	if got.Title != "Run a 5k" {
		t.Fatalf("title = %q, want Run a 5k", got.Title)
	}
	_ = user
}

func TestAPIGetGoal_NotFound(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "ggnf@example.com", "GGNF")

	rec := apiDo(t, e, http.MethodGet, "/api/v1/goals/missing-id", token, nil)
	decodeAPIError(t, rec, http.StatusNotFound)
}

func TestAPIUpdateGoal_Success(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "gu@example.com", "GU")

	created := decodeAPI[GoalDTO](t, apiDo(t, e, http.MethodPost, "/api/v1/goals", token, CreateGoalRequest{Title: "Original"}), http.StatusCreated)

	updated := decodeAPI[GoalDTO](t, apiDo(t, e, http.MethodPut, "/api/v1/goals/"+created.ID, token, UpdateGoalRequest{
		Title:       "Edited",
		Description: "now with notes",
	}), http.StatusOK)
	if updated.Title != "Edited" {
		t.Fatalf("title = %q, want Edited", updated.Title)
	}
	if updated.Description != "now with notes" {
		t.Fatalf("description = %q, want now with notes", updated.Description)
	}
}

func TestAPIUpdateGoal_NotFound(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "gunf@example.com", "GUNF")

	rec := apiDo(t, e, http.MethodPut, "/api/v1/goals/missing-id", token, UpdateGoalRequest{Title: "x"})
	decodeAPIError(t, rec, http.StatusNotFound)
}

func TestAPIMarkGoalComplete_Success(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "gmc@example.com", "GMC")

	created := decodeAPI[GoalDTO](t, apiDo(t, e, http.MethodPost, "/api/v1/goals", token, CreateGoalRequest{Title: "Do it"}), http.StatusCreated)
	if created.CompletedAt != nil {
		t.Fatal("fresh goal should not be complete")
	}

	done := decodeAPI[GoalDTO](t, apiDo(t, e, http.MethodPost, "/api/v1/goals/"+created.ID+"/complete", token, nil), http.StatusOK)
	if done.CompletedAt == nil {
		t.Fatal("expected completed_at to be set")
	}
}

func TestAPIMarkGoalComplete_Idempotent(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "gmci@example.com", "GMCI")

	created := decodeAPI[GoalDTO](t, apiDo(t, e, http.MethodPost, "/api/v1/goals", token, CreateGoalRequest{Title: "Do it"}), http.StatusCreated)
	first := decodeAPI[GoalDTO](t, apiDo(t, e, http.MethodPost, "/api/v1/goals/"+created.ID+"/complete", token, nil), http.StatusOK)
	second := decodeAPI[GoalDTO](t, apiDo(t, e, http.MethodPost, "/api/v1/goals/"+created.ID+"/complete", token, nil), http.StatusOK)

	if first.CompletedAt == nil || second.CompletedAt == nil {
		t.Fatal("expected completed_at to remain set")
	}
	if !second.CompletedAt.Equal(*first.CompletedAt) {
		t.Fatalf("completed_at drifted: first=%v second=%v (server should not overwrite on second call)", *first.CompletedAt, *second.CompletedAt)
	}
}

func TestAPIMarkGoalComplete_NotFound(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "gmcnf@example.com", "GMCNF")

	rec := apiDo(t, e, http.MethodPost, "/api/v1/goals/missing-id/complete", token, nil)
	decodeAPIError(t, rec, http.StatusNotFound)
}

func TestAPIReopenGoal_Success(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "gr@example.com", "GR")

	created := decodeAPI[GoalDTO](t, apiDo(t, e, http.MethodPost, "/api/v1/goals", token, CreateGoalRequest{Title: "Do it"}), http.StatusCreated)
	_ = decodeAPI[GoalDTO](t, apiDo(t, e, http.MethodPost, "/api/v1/goals/"+created.ID+"/complete", token, nil), http.StatusOK)
	reopened := decodeAPI[GoalDTO](t, apiDo(t, e, http.MethodPost, "/api/v1/goals/"+created.ID+"/reopen", token, nil), http.StatusOK)
	if reopened.CompletedAt != nil {
		t.Fatalf("expected completed_at nil after reopen, got %v", reopened.CompletedAt)
	}
}

func TestAPIReopenGoal_NotFound(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "grnf@example.com", "GRNF")

	rec := apiDo(t, e, http.MethodPost, "/api/v1/goals/missing-id/reopen", token, nil)
	decodeAPIError(t, rec, http.StatusNotFound)
}

func TestAPIDeleteGoal_Success(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "gd@example.com", "GD")

	created := decodeAPI[GoalDTO](t, apiDo(t, e, http.MethodPost, "/api/v1/goals", token, CreateGoalRequest{Title: "Do it"}), http.StatusCreated)
	rec := apiDo(t, e, http.MethodDelete, "/api/v1/goals/"+created.ID, token, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body = %s", rec.Code, rec.Body.String())
	}

	// Subsequent GET must 404.
	decodeAPIError(t, apiDo(t, e, http.MethodGet, "/api/v1/goals/"+created.ID, token, nil), http.StatusNotFound)
}

func TestAPIDeleteGoal_NotFound(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "gdnf@example.com", "GDNF")

	rec := apiDo(t, e, http.MethodDelete, "/api/v1/goals/missing-id", token, nil)
	decodeAPIError(t, rec, http.StatusNotFound)
}

func TestAPIGoals_RequiresAuth(t *testing.T) {
	_, _, _, e := setupHandler(t)

	rec := apiDo(t, e, http.MethodGet, "/api/v1/goals", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

// mustSwapGoalsRepo replaces h.goalsCtrl with a new controller
// backed by the supplied mockGoalRepository. Used by ordering
// tests that need to seed goals without a real DB. We mutate
// the controller in place (the controller has no exported
// setter) via the same package.
func mustSwapGoalsRepo(t *testing.T, h *Handler, repo *mockGoalRepository) *mockGoalRepository {
	t.Helper()
	h.goalsCtrl = controllers.NewGoalsController(repo)
	return repo
}

// --- /feedback ---

// mustSwapFeedbackRepo replaces h.feedbackCtrl with a new
// controller backed by the supplied mockFeedbackRepository.
// Used by the feedback tests that need to seed/inspect
// persisted rows. Mutates the controller in place via the
// same package, mirroring mustSwapGoalsRepo.
func mustSwapFeedbackRepo(t *testing.T, h *Handler, repo *mockFeedbackRepository) *mockFeedbackRepository {
	t.Helper()
	h.feedbackCtrl = controllers.NewFeedbackController(repo)
	return repo
}

func TestAPISubmitFeedback_Success(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "fb@example.com", "Feedbacker")
	repo := mustSwapFeedbackRepo(t, h, newMockFeedbackRepository())

	rec := apiDo(t, e, http.MethodPost, "/api/v1/feedback", token, SubmitFeedbackRequest{
		Title:   "Loving the new dashboard",
		Message: "Just wanted to say thanks for the recent changes.",
	})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body = %s", rec.Code, rec.Body.String())
	}
	if len(repo.feedback) != 1 {
		t.Fatalf("feedback len = %d, want 1", len(repo.feedback))
	}
	got := repo.feedback[0]
	// user_id must come from the JWT, not from the request
	// body — clients cannot submit feedback on someone else's
	// behalf.
	if got.UserID != "user-fb@example.com" {
		t.Fatalf("user_id = %q, want user-fb@example.com", got.UserID)
	}
	if got.Title != "Loving the new dashboard" {
		t.Fatalf("title = %q, want %q", got.Title, "Loving the new dashboard")
	}
	if got.Message != "Just wanted to say thanks for the recent changes." {
		t.Fatalf("message = %q, want %q", got.Message, "Just wanted to say thanks for the recent changes.")
	}
}

func TestAPISubmitFeedback_TrimsWhitespace(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "fbt@example.com", "FbT")
	repo := mustSwapFeedbackRepo(t, h, newMockFeedbackRepository())

	rec := apiDo(t, e, http.MethodPost, "/api/v1/feedback", token, SubmitFeedbackRequest{
		Title:   "  Trimmed Title  ",
		Message: "   Trimmed message here.   ",
	})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body = %s", rec.Code, rec.Body.String())
	}
	if repo.feedback[0].Title != "Trimmed Title" {
		t.Fatalf("title = %q, want trimmed %q", repo.feedback[0].Title, "Trimmed Title")
	}
	if repo.feedback[0].Message != "Trimmed message here." {
		t.Fatalf("message = %q, want trimmed %q", repo.feedback[0].Message, "Trimmed message here.")
	}
}

func TestAPISubmitFeedback_TitleTooShort(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "fbts@example.com", "FbTS")

	rec := apiDo(t, e, http.MethodPost, "/api/v1/feedback", token, SubmitFeedbackRequest{
		Title:   "Hi",
		Message: "This is a long enough message body.",
	})
	apiErr := decodeAPIError(t, rec, http.StatusBadRequest)
	if apiErr.Error == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestAPISubmitFeedback_TitleTooLong(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "fbtl@example.com", "FbTL")

	rec := apiDo(t, e, http.MethodPost, "/api/v1/feedback", token, SubmitFeedbackRequest{
		Title:   strings.Repeat("x", 101),
		Message: "This is a long enough message body.",
	})
	decodeAPIError(t, rec, http.StatusBadRequest)
}

func TestAPISubmitFeedback_MessageTooShort(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "fbms@example.com", "FbMS")

	rec := apiDo(t, e, http.MethodPost, "/api/v1/feedback", token, SubmitFeedbackRequest{
		Title:   "Valid Title Here",
		Message: "Short",
	})
	apiErr := decodeAPIError(t, rec, http.StatusBadRequest)
	if apiErr.Error == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestAPISubmitFeedback_MessageTooLong(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "fbml@example.com", "FbML")

	rec := apiDo(t, e, http.MethodPost, "/api/v1/feedback", token, SubmitFeedbackRequest{
		Title:   "Valid Title Here",
		Message: strings.Repeat("x", 1001),
	})
	decodeAPIError(t, rec, http.StatusBadRequest)
}

func TestAPISubmitFeedback_InvalidBody(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "fbib@example.com", "FbIB")

	// Send raw garbage so the JSON decoder in c.Bind rejects it.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	apiErr := decodeAPIError(t, rec, http.StatusBadRequest)
	if apiErr.Error == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestAPISubmitFeedback_RepoError(t *testing.T) {
	h, _, mockUser, e := setupHandler(t)
	token, _ := loginUser(t, h, mockUser, "fbre@example.com", "FbRE")
	repo := mustSwapFeedbackRepo(t, h, newMockFeedbackRepository())
	repo.errCreate = errors.New("database error")

	rec := apiDo(t, e, http.MethodPost, "/api/v1/feedback", token, SubmitFeedbackRequest{
		Title:   "Valid Title Here",
		Message: "Valid message body — long enough.",
	})
	apiErr := decodeAPIError(t, rec, http.StatusInternalServerError)
	if apiErr.Error == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestAPISubmitFeedback_RequiresAuth(t *testing.T) {
	_, _, _, e := setupHandler(t)

	rec := apiDo(t, e, http.MethodPost, "/api/v1/feedback", "", SubmitFeedbackRequest{
		Title:   "Anonymous Title",
		Message: "Anonymous message body that is long enough.",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401, body = %s", rec.Code, rec.Body.String())
	}
}
