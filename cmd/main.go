package main

import (
	"log"
	"net"
	"net/http"
	"path/filepath"

	"github.com/labstack/echo/v4"

	"stren/internal/controllers"
	"stren/internal/db"
	"stren/internal/models"
	"stren/internal/push"
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

	// Load and validate storage (R2) configuration
	if _, err := utils.LoadStorageConfig(); err != nil {
		log.Fatalf("Failed to load storage configuration: %v", err)
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
	weightRepo := models.NewWeightRepository(database)
	pushRepo := models.NewPushSubscriptionRepository(database)

	// Initialize auth service
	jwtService := utils.NewJWTService(cfg.JWT_SECRET)

	// Load or generate the VAPID keypair. Keys live alongside the
	// SQLite file in the same persistent directory so subscriptions
	// survive restarts. If the directory is wiped (e.g. a fresh
	// container) a new keypair is generated and existing
	// subscriptions become invalid until the user re-enables them.
	vapidDataDir := filepath.Dir(cfg.DB_PATH)
	keys, err := push.LoadOrGenerate(vapidDataDir)
	if err != nil {
		log.Fatalf("Failed to load or generate VAPID keys: %v", err)
	}
	vapidPublicKey := keys.PublicKeyString()
	log.Printf("vapid: loaded keypair from %s", vapidDataDir)

	// Wire the push client + fan-out service. A bounded HTTP client
	// timeout (30s) is applied via the package default; no need to
	// expose it.
	pushClient := push.NewClient(keys, push.ClientConfig{})
	pushService := push.NewService(pushClient, push.NewStoreAdapter(pushRepo), push.ServiceConfig{})

	// Initialize controllers
	authCtrl := controllers.NewAuthController(userRepo, jwtService)
	entryCtrl := controllers.NewEntryController(repo)
	adminCtrl := controllers.NewAdminController(repo)
	adminUserCtrl := controllers.NewAdminUserController(adminUserRepo)
	feedbackCtrl := controllers.NewFeedbackController(models.NewFeedbackRepository(database))
	timersCtrl := controllers.NewTimersController()
	weightCtrl := controllers.NewWeightController(weightRepo)
	pushCtrl := controllers.NewPushController(pushRepo)
	adminNotificationsCtrl := controllers.NewAdminNotificationsController(pushService)

	// Initialize route handlers
	validator := utils.NewValidator()
	h := routes.NewHandler(
		authCtrl,
		entryCtrl,
		adminCtrl,
		adminUserCtrl,
		feedbackCtrl,
		timersCtrl,
		weightCtrl,
		pushCtrl,
		adminNotificationsCtrl,
		pushRepo,
		vapidPublicKey,
		true, // pushConfigured — keys are always present after LoadOrGenerate
		userRepo,
		jwtService,
		validator,
	)

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

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
