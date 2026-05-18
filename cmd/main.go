package main

import (
	"log"
	"net"
	"net/http"

	"github.com/labstack/echo/v4"

	"stren/internal/controllers"
	"stren/internal/db"
	"stren/internal/models"
	"stren/internal/routes"
	"stren/internal/utils"
	"stren/internal/views"
)

func main() {
	// Load and validate environment variables on startup
	cfg, err := utils.LoadAndValidateEnv()
	if err != nil {
		log.Fatalf("Failed to load environment variables: %v", err)
	}

	// Initialize database
	database, err := db.NewConnection(cfg.DB_PATH, cfg.TURSO_DATABASE_URL, cfg.TURSO_AUTH_TOKEN)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize repositories
	repo := models.NewExerciseRepository(database)
	userRepo := models.NewUserRepository(database)
	adminUserRepo := models.NewUserAdminRepository(database)

	// Initialize auth service
	jwtService := utils.NewJWTService(cfg.JWT_SECRET)

	// Initialize controllers
	authCtrl := controllers.NewAuthController(userRepo, jwtService)
	entryCtrl := controllers.NewEntryController(repo)
	adminCtrl := controllers.NewAdminController(repo)
	adminUserCtrl := controllers.NewAdminUserController(adminUserRepo)
	feedbackCtrl := controllers.NewFeedbackController(models.NewFeedbackRepository(database))
	timerCtrl := controllers.NewTimerController()
	emomCtrl := controllers.NewEMOMController()

	// Initialize route handlers
	validator := utils.NewValidator()
	h := routes.NewHandler(authCtrl, entryCtrl, adminCtrl, adminUserCtrl, feedbackCtrl, timerCtrl, emomCtrl, userRepo, jwtService, validator)

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true

	// Custom HTTP error handler - renders error pages instead of JSON
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		code := http.StatusInternalServerError
		message := "Something went wrong"

		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
			if he.Message != nil {
				if msg, ok := he.Message.(string); ok {
					message = msg
				}
			}
		}

		data := views.NewErrorData(code, message)
		views.ErrorPage(data).Render(c.Request().Context(), c.Response())
	}

	// Register routes
	h.RegisterRoutes(e)

	// Start server
	localIP := getLocalIP()
	if localIP != "" {
		log.Printf("Server starting on http://localhost:%s and http://%s:%s", cfg.PORT, localIP, cfg.PORT)
	} else {
		log.Printf("Server starting on http://localhost:%s", cfg.PORT)
	}
	if err := e.Start(":" + cfg.PORT); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// getLocalIP returns the first non-loopback IPv4 address of the machine.
// Returns an empty string if no suitable interface is found.
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	return ""
}
