// Package routes provides HTTP route handlers for the strength tracker application.
// This file contains Echo middleware for authentication.
package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"stren/internal/utils"
)

// authContextKey is used to store auth claims in the Echo context.
const authContextKey = "auth_claims"

// AuthMiddleware returns an Echo middleware function that verifies the auth cookie
// and injects the user claims into the request context.
// If the token is missing or invalid, the request is redirected to /login.
func AuthMiddleware(jwtService *utils.JWTService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip auth for public routes
			path := c.Request().URL.Path
			if isPublicRoute(path) {
				return next(c)
			}

			cookie, err := c.Cookie(utils.CookieName)
			if err != nil {
				return redirectToLogin(c)
			}

			claims, err := jwtService.VerifyToken(cookie.Value)
			if err != nil {
				return redirectToLogin(c)
			}

			c.Set(authContextKey, claims)
			return next(c)
		}
	}
}

// isPublicRoute returns true for routes that don't require authentication.
func isPublicRoute(path string) bool {
	public := []string{
		"/login",
		"/register",
		"/css/",
		"/icons/",
		"/manifest.json",
		"/sw.js",
		"/favicon.ico",
	}
	for _, p := range public {
		if len(path) >= len(p) && path[:len(p)] == p {
			return true
		}
	}
	return false
}

func redirectToLogin(c echo.Context) error {
	// For htmx requests, return a client-side redirect instruction
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Redirect", "/login")
		return c.NoContent(http.StatusUnauthorized)
	}
	return c.Redirect(http.StatusSeeOther, "/login")
}

// GetClaims retrieves the authenticated user's claims from the Echo context.
// Returns nil if no claims are present (should not happen on protected routes).
func GetClaims(c echo.Context) *utils.Claims {
	claims, ok := c.Get(authContextKey).(*utils.Claims)
	if !ok {
		return nil
	}
	return claims
}
