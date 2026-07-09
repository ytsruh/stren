package routes

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// req builds a plain GET (or other verb) http.Request for the
// test server. Kept as a tiny helper so the route tests below
// are not buried in httptest boilerplate.
func req(method, target string) *http.Request {
	return httptest.NewRequest(method, target, nil)
}

// reqForm builds a POST request with an x-www-form-urlencoded
// body. Echo's c.FormValue reads the body, so a request with
// the right Content-Type and body is what the handler actually
// sees.
func reqForm(method, target string, form url.Values) *http.Request {
	r := httptest.NewRequest(method, target, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

// newRecorder returns a fresh httptest.ResponseRecorder. Kept
// as a named helper so the test body can stay focused on the
// assertions.
func newRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

// TestForgotForm_Renders confirms GET /forgot returns 200
// and contains the form. Acts as a smoke test for the route
// registration: if /forgot is not registered, the test
// surfaces a 404 instead of a render error.
func TestForgotForm_Renders(t *testing.T) {
	_, _, _, e := setupHandler(t)

	rec := newRecorder()
	e.ServeHTTP(rec, req("GET", "/forgot"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Forgot password") {
		t.Errorf("body missing 'Forgot password' heading: %q", body)
	}
	if !strings.Contains(body, `name="email"`) {
		t.Errorf("body missing email input: %q", body)
	}
}

func TestForgotPost_UnknownEmailStillReturnsOK(t *testing.T) {
	// A POST for a non-existent email must return the
	// "submitted" page (no enumeration leak), with the same
	// shape as the success response. The non-HTMX path
	// renders the form page in submitted state.
	_, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("email", "ghost@example.com")

	rec := newRecorder()
	e.ServeHTTP(rec, reqForm("POST", "/forgot", form))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "we've sent a reset link") {
		t.Errorf("body missing 'sent a reset link' confirmation: %q", body)
	}
}

func TestForgotPost_HTMXRedirects(t *testing.T) {
	// HTMX requests get a 204 + HX-Redirect header so the
	// client navigates to the same page in "submitted" mode.
	_, _, _, e := setupHandler(t)

	form := url.Values{}
	form.Set("email", "ghost@example.com")

	r := reqForm("POST", "/forgot", form)
	r.Header.Set("HX-Request", "true")

	rec := newRecorder()
	e.ServeHTTP(rec, r)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("HX-Redirect"); !strings.HasPrefix(got, "/forgot") {
		t.Errorf("HX-Redirect = %q, want /forgot prefix", got)
	}
}

func TestForgotPost_EmptyEmailShowsError(t *testing.T) {
	// A POST with no email field must not silently succeed.
	// The form's required attribute will block this in the
	// browser, but the server must not depend on that.
	_, _, _, e := setupHandler(t)

	rec := newRecorder()
	e.ServeHTTP(rec, reqForm("POST", "/forgot", url.Values{}))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Email is required") {
		t.Errorf("body missing 'Email is required': %q", rec.Body.String())
	}
}

func TestResetForm_NoTokenRedirectsToForgot(t *testing.T) {
	// GET /reset with no token is a user navigating directly
	// to the page. Redirect them to /forgot so they can
	// request a link.
	_, _, _, e := setupHandler(t)
	rec := newRecorder()
	e.ServeHTTP(rec, req("GET", "/reset"))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 (redirect to /forgot)", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/forgot" {
		t.Errorf("Location = %q, want /forgot", loc)
	}
}

func TestResetForm_RendersWithToken(t *testing.T) {
	// GET /reset?token=... renders the reset form with the
	// token preserved in a hidden input.
	_, _, _, e := setupHandler(t)
	rec := newRecorder()
	e.ServeHTTP(rec, req("GET", "/reset?token=abc123"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `value="abc123"`) {
		t.Errorf("body missing hidden token: %q", body)
	}
	if !strings.Contains(body, `name="password"`) {
		t.Errorf("body missing password input: %q", body)
	}
	if !strings.Contains(body, `name="confirm"`) {
		t.Errorf("body missing confirm input: %q", body)
	}
}

func TestResetPost_EmptyTokenRedirectsToForgot(t *testing.T) {
	// A POST with no token (or a token that fails the
	// controller's check) is a programming error from the
	// browser side; we redirect to /forgot for safety.
	_, _, _, e := setupHandler(t)
	form := url.Values{}
	form.Set("password", "newpass1")
	form.Set("confirm", "newpass1")

	rec := newRecorder()
	e.ServeHTTP(rec, reqForm("POST", "/reset", form))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
}

func TestResetPost_PasswordMismatchShowsError(t *testing.T) {
	// Password and confirm must match. The form's
	// minlength attribute is a UX hint, but the server
	// must enforce it independently.
	_, _, _, e := setupHandler(t)
	form := url.Values{}
	form.Set("token", "any")
	form.Set("password", "longenough1")
	form.Set("confirm", "different1")

	rec := newRecorder()
	e.ServeHTTP(rec, reqForm("POST", "/reset", form))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Passwords do not match") {
		t.Errorf("body missing mismatch error: %q", rec.Body.String())
	}
}
