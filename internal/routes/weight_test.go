package routes

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"stren/internal/controllers"
	"stren/internal/models"
)

// TestCreateWeight_HTMX_SetsTriggerAndSuccessToast locks in the create
// flow's response contract: an HTMX POST /weight must (a) set the
// HX-Trigger that drives the client-side redirect to /weight, and
// (b) render a toast with the create title "Weight saved!" — not the
// previously hardcoded "Weight entry updated!" which made the create
// toast look like an update.
func TestCreateWeight_HTMX_SetsTriggerAndSuccessToast(t *testing.T) {
	h, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("weight", "75.5")
	form.Set("notes", "morning")
	req := httptest.NewRequest(http.MethodPost, "/weight", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.CreateWeight(c); err != nil {
		t.Fatalf("CreateWeight failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("HX-Trigger"); got != `{"triggerRedirect": "/weight"}` {
		t.Errorf("HX-Trigger = %q, want %q", got, `{"triggerRedirect": "/weight"}`)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Weight saved!") {
		t.Errorf("expected create title %q in toast, got body: %s", "Weight saved!", body)
	}
	if strings.Contains(body, "Weight entry updated!") {
		t.Errorf("create toast should not carry the update title, got body: %s", body)
	}
}

// TestUpdateWeight_HTMX_SetsUpdateTitle confirms the update route still
// passes the update title through the now-parameterised toast helper.
func TestUpdateWeight_HTMX_SetsUpdateTitle(t *testing.T) {
	h, _, _, e := setupHandler(t)

	// Seed an existing entry to update.
	mockWeight := newMockWeightRepository()
	mockWeight.entries = []models.WeightEntry{
		{ID: "w1", UserID: "user-1", Weight: 80, Notes: "before", CreatedAt: time.Date(2026, 1, 9, 8, 0, 0, 0, time.UTC)},
	}
	h.weightCtrl = controllers.NewWeightController(mockWeight, nil)

	form := url.Values{}
	form.Set("weight", "79.0")
	form.Set("notes", "after")
	form.Set("created_at", "2026-01-09T08:00")
	req := httptest.NewRequest(http.MethodPut, "/weight/w1", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("w1")
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.UpdateWeight(c); err != nil {
		t.Fatalf("UpdateWeight failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Weight entry updated!") {
		t.Errorf("expected update title in toast, got body: %s", body)
	}
}

// TestCreateWeight_ThenList_ShowsRowAndNotEmptyState is the regression
// test for the "app still asks for a first entry after I added a
// weight" bug report. The flow is: POST /weight to insert, then
// GET /weight and assert the page renders the new row and hides the
// empty state. Locks in both the data flow and the empty-state
// conditional in list.templ.
func TestCreateWeight_ThenList_ShowsRowAndNotEmptyState(t *testing.T) {
	h, _, _, e := setupHandler(t)

	// 1) Create a weight entry as an HTMX request.
	form := url.Values{}
	form.Set("weight", "75.5")
	form.Set("notes", "first entry")
	createReq := httptest.NewRequest(http.MethodPost, "/weight", strings.NewReader(form.Encode()))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	createReq.Header.Set("HX-Request", "true")
	createRec := httptest.NewRecorder()
	createCtx := e.NewContext(createReq, createRec)
	setAuthContext(createCtx, "user-1", "test@example.com", "Test User", false)
	if err := h.CreateWeight(createCtx); err != nil {
		t.Fatalf("CreateWeight failed: %v", err)
	}
	if createRec.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d", createRec.Code)
	}

	// 2) GET /weight and assert the page no longer shows the empty
	//    state and does show the row we just inserted.
	getReq := httptest.NewRequest(http.MethodGet, "/weight", nil)
	getRec := httptest.NewRecorder()
	getCtx := e.NewContext(getReq, getRec)
	setAuthContext(getCtx, "user-1", "test@example.com", "Test User", false)
	if err := h.WeightPage(getCtx); err != nil {
		t.Fatalf("WeightPage failed: %v", err)
	}
	if getRec.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", getRec.Code)
	}
	body := getRec.Body.String()
	if strings.Contains(body, "Log Your First Weight") {
		t.Errorf("expected empty state to be hidden after creating a weight; body:\n%s", body)
	}
	if strings.Contains(body, "No weight entries yet") {
		t.Errorf("expected empty heading to be hidden after creating a weight; body:\n%s", body)
	}
	if !strings.Contains(body, "75.5") {
		t.Errorf("expected the new weight (75.5) to appear in the list; body:\n%s", body)
	}
}
