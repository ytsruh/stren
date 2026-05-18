package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"stren/internal/utils"
)

func TestIsPublicRoute(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/login", true},
		{"/register", true},
		{"/css/styles.css", true},
		{"/icons/icon-192.png", true},
		{"/manifest.json", true},
		{"/sw.js", true},
		{"/favicon.ico", true},
		{"/", false},
		{"/entries", false},
		{"/entries/new", false},
		{"/exercises/Squat", false},
		{"/api/exercises", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isPublicRoute(tt.path); got != tt.expected {
				t.Errorf("isPublicRoute(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestRedirectToLogin_NonHTMX(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/entries", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := redirectToLogin(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Fatalf("expected redirect to /login, got %q", loc)
	}
}

func TestRedirectToLogin_HTMX(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/entries", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := redirectToLogin(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	if hx := rec.Header().Get("HX-Redirect"); hx != "/login" {
		t.Fatalf("expected HX-Redirect /login, got %q", hx)
	}
}

func TestAuthMiddleware_PublicRoute(t *testing.T) {
	jwtService := utils.NewJWTService("test-secret")
	middleware := AuthMiddleware(jwtService)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	next := func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "ok")
	}

	err := middleware(next)(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected next handler to be called for public route")
	}
}

func TestAuthMiddleware_MissingCookie(t *testing.T) {
	jwtService := utils.NewJWTService("test-secret")
	middleware := AuthMiddleware(jwtService)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/entries", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	next := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	err := middleware(next)(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect status, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Fatalf("expected redirect to /login, got %q", loc)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	jwtService := utils.NewJWTService("test-secret")
	middleware := AuthMiddleware(jwtService)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/entries", nil)
	req.AddCookie(&http.Cookie{
		Name:  utils.CookieName,
		Value: "invalid-token",
	})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	next := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	err := middleware(next)(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect status, got %d", rec.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	jwtService := utils.NewJWTService("test-secret")
	token, err := jwtService.GenerateToken("user-42", "alice@example.com", "Alice", false)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	middleware := AuthMiddleware(jwtService)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/entries", nil)
	req.AddCookie(&http.Cookie{
		Name:  utils.CookieName,
		Value: token,
	})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	next := func(c echo.Context) error {
		called = true

		// Verify claims were injected
		claims := GetClaims(c)
		if claims == nil {
			t.Fatal("expected claims to be set in context")
		}
if claims.UserID != "user-42" {
		t.Fatalf("expected user_id 'user-42', got %q", claims.UserID)
	}
		if claims.Email != "alice@example.com" {
			t.Fatalf("expected email 'alice@example.com', got %q", claims.Email)
		}
		if claims.Name != "Alice" {
			t.Fatalf("expected name 'Alice', got %q", claims.Name)
		}
		if claims.IsAdmin {
			t.Fatal("expected is_admin to be false")
		}

		return c.String(http.StatusOK, "ok")
	}

	err = middleware(next)(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected next handler to be called")
	}
}

func TestGetClaims_Missing(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/entries", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	claims := GetClaims(c)
	if claims != nil {
		t.Fatalf("expected nil claims, got %+v", claims)
	}
}

func TestGetClaims_Present(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/entries", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	expected := &utils.Claims{
		UserID:  "user-7",
		Email:   "bob@example.com",
		Name:    "Bob",
		IsAdmin: true,
	}
	c.Set(authContextKey, expected)

	claims := GetClaims(c)
	if claims == nil {
		t.Fatal("expected claims, got nil")
	}
	if claims.UserID != expected.UserID {
		t.Fatalf("expected user_id %q, got %q", expected.UserID, claims.UserID)
	}
	if claims.Email != expected.Email {
		t.Fatalf("expected email %q, got %q", expected.Email, claims.Email)
	}
	if claims.IsAdmin != expected.IsAdmin {
		t.Fatalf("expected is_admin %v, got %v", expected.IsAdmin, claims.IsAdmin)
	}
}

func TestAuthMiddleware_ValidToken_HTMX(t *testing.T) {
	jwtService := utils.NewJWTService("test-secret")
	token, err := jwtService.GenerateToken("user-1", "test@example.com", "Test", false)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	middleware := AuthMiddleware(jwtService)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/entries", nil)
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{
		Name:  utils.CookieName,
		Value: token,
	})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	next := func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "ok")
	}

	err = middleware(next)(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected next handler to be called for valid HTMX request")
	}
}

func TestAuthMiddleware_MissingCookie_HTMX(t *testing.T) {
	jwtService := utils.NewJWTService("test-secret")
	middleware := AuthMiddleware(jwtService)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/entries", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	next := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	err := middleware(next)(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d for HTMX, got %d", http.StatusUnauthorized, rec.Code)
	}
	if hx := rec.Header().Get("HX-Redirect"); hx != "/login" {
		t.Fatalf("expected HX-Redirect /login, got %q", hx)
	}
}
