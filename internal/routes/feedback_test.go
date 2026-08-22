package routes

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"stren/internal/controllers"
	"stren/internal/models"
	"stren/internal/utils"
)

// setupFeedbackHandler wires a Handler with its own feedback mock
// (setupHandler doesn't expose the feedback repository) so tests
// can inspect exactly what SubmitFeedback persisted.
func setupFeedbackHandler(t *testing.T) (*Handler, *mockFeedbackRepository, *echo.Echo) {
	t.Helper()
	e := echo.New()
	mockUser := newMockUserRepository()
	mockFeedback := newMockFeedbackRepository()
	jwtService := utils.NewJWTService("test-secret")
	proc, upl := newFakeImagePipeline()
	h := NewHandler(
		controllers.NewAuthController(mockUser, jwtService, nil),
		controllers.NewAuthRecoveryController(mockUser, newMockAuthTokenRepo(), nil),
		controllers.NewExerciseEntryController(newMockRepository()),
		controllers.NewAdminController(newMockRepository()),
		controllers.NewAdminUserController(newMockAdminUserRepository()),
		controllers.NewFeedbackController(mockFeedback),
		controllers.NewWeightController(newMockWeightRepository(), nil),
		controllers.NewGoalsController(newMockGoalRepository()),
		mockUser, jwtService, utils.NewValidator(),
		proc, upl, DefaultExerciseImageConfig,
	)
	h.RegisterRoutes(e)
	return h, mockFeedback, e
}

func TestFeedbackForm_RendersPage(t *testing.T) {
	h, _, e := setupFeedbackHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/feedback", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.FeedbackForm(c); err != nil {
		t.Fatalf("FeedbackForm failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Submit Feedback",
		`hx-post="/feedback"`,
		`name="title"`,
		`name="message"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected page body to contain %q", want)
		}
	}
}

// feedbackForm builds the url.Values payload the /feedback POST
// route expects.
func feedbackForm(title, message string) url.Values {
	form := url.Values{}
	form.Set("title", title)
	form.Set("message", message)
	return form
}

func TestSubmitFeedback_HtmxSuccess(t *testing.T) {
	h, mockFeedback, e := setupFeedbackHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/feedback",
		strings.NewReader(feedbackForm("  iOS access please  ", "  I would like access to the iOS app.  ").Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.SubmitFeedback(c); err != nil {
		t.Fatalf("SubmitFeedback failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body = %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, "Thanks for your feedback") {
		t.Errorf("expected success toast in body, got: %s", body)
	}
	// The page listens for this event to reset the form fields.
	if trigger := rec.Header().Get("HX-Trigger"); !strings.Contains(trigger, "feedbackSubmitted") {
		t.Errorf("expected HX-Trigger feedbackSubmitted, got %q", trigger)
	}

	// The submission must be stored with trimmed fields.
	mockFeedback.mu.Lock()
	defer mockFeedback.mu.Unlock()
	if len(mockFeedback.feedback) != 1 {
		t.Fatalf("expected 1 stored feedback, got %d", len(mockFeedback.feedback))
	}
	stored := mockFeedback.feedback[0]
	if stored.Title != "iOS access please" || stored.Message != "I would like access to the iOS app." {
		t.Errorf("unexpected stored values: title=%q message=%q", stored.Title, stored.Message)
	}
	if stored.UserID != "user-1" {
		t.Errorf("expected user id from claims, got %q", stored.UserID)
	}
}

func TestSubmitFeedback_ValidationShowsErrorToast(t *testing.T) {
	h, mockFeedback, e := setupFeedbackHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/feedback",
		strings.NewReader(feedbackForm("hi", "This message is long enough.").Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.SubmitFeedback(c); err != nil {
		t.Fatalf("SubmitFeedback failed: %v", err)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Title must be at least 5 characters") {
		t.Errorf("expected validation error toast in body, got: %s", body)
	}
	if !strings.Contains(body, `data-category="error"`) {
		t.Error("expected error toast markup")
	}

	mockFeedback.mu.Lock()
	defer mockFeedback.mu.Unlock()
	if len(mockFeedback.feedback) != 0 {
		t.Errorf("expected nothing stored on validation failure, got %d items", len(mockFeedback.feedback))
	}
}

func TestSubmitFeedback_NonHtmxRedirectsToDashboard(t *testing.T) {
	h, mockFeedback, e := setupFeedbackHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/feedback",
		strings.NewReader(feedbackForm("No JS fallback", "Submitted without htmx enabled.").Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.SubmitFeedback(c); err != nil {
		t.Fatalf("SubmitFeedback failed: %v", err)
	}
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Errorf("expected redirect to /, got %q", loc)
	}

	mockFeedback.mu.Lock()
	defer mockFeedback.mu.Unlock()
	if len(mockFeedback.feedback) != 1 {
		t.Errorf("expected submission to be stored, got %d items", len(mockFeedback.feedback))
	}
}

func TestDashboard_ShowsIOSBanner(t *testing.T) {
	h, mock, _, e := setupHandler(t)
	mock.exerciseEntries = []models.ExerciseEntry{
		{ID: "entry-1", UserID: "user-1", ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.Dashboard(c); err != nil {
		t.Fatalf("Dashboard failed: %v", err)
	}
	body := rec.Body.String()
	for _, want := range []string{"Stren is on iPhone", `href="/feedback"`} {
		if !strings.Contains(body, want) {
			t.Errorf("expected dashboard to contain banner text %q", want)
		}
	}
}

func TestDashboard_EmptyState_ShowsIOSBanner(t *testing.T) {
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.Dashboard(c); err != nil {
		t.Fatalf("Dashboard failed: %v", err)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Stren is on iPhone") {
		t.Error("expected banner on empty dashboard too")
	}
	if !strings.Contains(body, "No workouts in the last 7 days") {
		t.Error("expected empty state alongside banner")
	}
}
