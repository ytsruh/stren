package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPushSubscribe_HappyPath exercises the JSON body path: a
// browser-side pushManager.subscribe serialises its object to JSON
// and the server parses it, stores it, and returns a success toast.
func TestPushSubscribe_HappyPath(t *testing.T) {
	h, _, _, e := setupHandler(t)

	body := `{"endpoint":"https://push.example/abc","p256dh":"BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4AyfIuF8u0cKPC3tB8S5d3VbV5J8K3A8B7B4e1d2c3a4b5c6d7e8f9a0b1c2","auth":"BxSKS5R8b4wCY0Uo7w8jVA"}`
	req := httptest.NewRequest(http.MethodPost, "/api/push/subscribe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.PushSubscribe(c); err != nil {
		t.Fatalf("PushSubscribe: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	// The response is the templ-rendered success toast.
	if !strings.Contains(rec.Body.String(), "Notifications enabled") {
		t.Fatalf("expected success toast, got: %s", rec.Body.String())
	}
}

// TestPushSubscribe_BadJSONReturnsError makes sure an invalid JSON
// payload still produces a clean 200 response (HTMX renders the
// toast in either case) with the error message visible.
func TestPushSubscribe_BadJSON(t *testing.T) {
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/push/subscribe", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.PushSubscribe(c); err != nil {
		t.Fatalf("PushSubscribe: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Invalid subscription payload") {
		t.Fatalf("expected error toast, got: %s", rec.Body.String())
	}
}

func TestPushUnsubscribe_HappyPath(t *testing.T) {
	h, _, _, e := setupHandler(t)

	body := `{"endpoint":"https://push.example/abc"}`
	req := httptest.NewRequest(http.MethodDelete, "/api/push/unsubscribe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "user-1", "test@example.com", "Test User", false)

	if err := h.PushUnsubscribe(c); err != nil {
		t.Fatalf("PushUnsubscribe: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestAdminNotificationsForm_AdminAllowed(t *testing.T) {
	h, _, _, e := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/notifications", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "admin-1", "admin@example.com", "Admin", true)

	if err := h.AdminNotificationsForm(c); err != nil {
		t.Fatalf("AdminNotificationsForm: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Send to all subscribers") {
		t.Fatalf("expected form to render, got: %s", rec.Body.String())
	}
}

func TestAdminNotificationsSend_NilServiceReturnsErrorCard(t *testing.T) {
	// The setupHandler wires the admin controller with a nil push
	// service. The route should still return a clean 200 with an
	// error card rather than crashing the handler.
	h, _, _, e := setupHandler(t)

	form := strings.NewReader("title=Hi&body=World")
	req := httptest.NewRequest(http.MethodPost, "/admin/notifications/send", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, "admin-1", "admin@example.com", "Admin", true)

	if err := h.AdminNotificationsSend(c); err != nil {
		t.Fatalf("AdminNotificationsSend: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "push notifications are not configured") {
		t.Fatalf("expected error card, got: %s", rec.Body.String())
	}
}
