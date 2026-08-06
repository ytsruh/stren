package main

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v4"

	"stren/internal/controllers"
	"stren/internal/db"
	"stren/internal/email"
	"stren/internal/export"
	"stren/internal/imaging"
	"stren/internal/models"
	"stren/internal/push"
	"stren/internal/reminders"
	"stren/internal/routes"
	"stren/internal/utils"
	"stren/internal/views"
)

// weightReminderCronSpec is the cron expression for the
// hourly "who is due for a weight reminder?" tick.
// "0 * * * *" is minute=0, every hour, every day. Each
// tick finds every user whose next_fire_at is at or before
// now and fires their chosen channels. The per-user
// schedule (frequency, day-of-week, time) lives in the
// users table — this is just the heartbeat.
//
// Interpreted in UTC by the cron wrapper so the schedule
// is the same regardless of where the server happens to
// be deployed. Per project policy (AGENTS.md) the spec is
// hard-coded rather than driven by an env var: the tick
// cadence is a product decision, not a per-deployment knob.
const weightReminderCronSpec = "0 * * * *"

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
	authTokenRepo := models.NewAuthTokenRepository(database)
	goalsRepo := models.NewGoalRepository(database)

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

	// Wire the email service. The SMTP client targets Cloudflare's
	// implicit-TLS endpoint (smtp.mx.cloudflare.net:465); the
	// default From address, port, and timeout are set by
	// email.NewClient from the APIToken alone. Email is required at
	// startup (the env var is in the required list), so a failure
	// to construct the client is fatal. The PUBLIC_URL env var is
	// the base URL threaded into every link the email contains
	// (dashboard button, password-reset URL, footer link).
	emailClient, err := email.NewClient(email.ClientConfig{
		APIToken: cfg.CLOUDFLARE_EMAIL_TOKEN,
	})
	if err != nil {
		log.Fatalf("Failed to initialize email client: %v", err)
	}
	emailService, err := email.NewService(emailClient, cfg.PUBLIC_URL)
	if err != nil {
		log.Fatalf("Failed to initialize email service: %v", err)
	}

	// Initialize controllers
	authCtrl := controllers.NewAuthController(userRepo, jwtService, emailService)
	exerciseEntryCtrl := controllers.NewExerciseEntryController(repo)
	adminCtrl := controllers.NewAdminController(repo)
	adminUserCtrl := controllers.NewAdminUserController(adminUserRepo)
	feedbackCtrl := controllers.NewFeedbackController(models.NewFeedbackRepository(database))
	timersCtrl := controllers.NewTimersController()
	weightCtrl := controllers.NewWeightController(weightRepo, r2PhotoGetter{})
	pushCtrl := controllers.NewPushController(pushRepo)
	authRecoveryCtrl := controllers.NewAuthRecoveryController(userRepo, authTokenRepo, emailService)
	goalsCtrl := controllers.NewGoalsController(goalsRepo)

	// Initialize the per-user weight-reminder orchestrator here
	// (above the admin notifications controller it is wired
	// into) so the controller constructor can take a non-nil
	// pointer. The orchestrator's own scheduler is started
	// further down, after the Echo routes are registered, so
	// an init failure here is the same as an init failure
	// there (log.Fatal).
	weightReminder, err := reminders.NewUserReminder(
		userRepo,
		emailService,
		pushService,
		reminders.UserReminderConfig{},
	)
	if err != nil {
		log.Fatalf("Failed to initialize weight reminder: %v", err)
	}

	adminNotificationsCtrl := controllers.NewAdminNotificationsController(pushService, weightReminder)

	// Initialize route handlers
	validator := utils.NewValidator()
	h := routes.NewHandler(
		authCtrl,
		authRecoveryCtrl,
		exerciseEntryCtrl,
		adminCtrl,
		adminUserCtrl,
		feedbackCtrl,
		timersCtrl,
		weightCtrl,
		pushCtrl,
		adminNotificationsCtrl,
		goalsCtrl,
		pushRepo,
		vapidPublicKey,
		true, // pushConfigured — keys are always present after LoadOrGenerate
		userRepo,
		jwtService,
		validator,
		imaging.NewStdProcessor(),
		utilsR2Uploader{},
		routes.DefaultExerciseImageConfig,
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

	// Start the hourly weight-reminder scheduler. The
	// orchestrator was constructed above and is also
	// shared with the admin notifications controller
	// (the "send all due reminders now" button). The cron
	// wrapper is the only place in the codebase that
	// imports the third-party scheduling library, so
	// a future "swap for a queue / system cron" change
	// is a one-package diff. A bad spec (e.g. a typo
	// in "0 * * * *") fails startup rather than
	// silently never firing.
	scheduler, err := reminders.NewCronScheduler(
		weightReminderCronSpec,
		time.UTC,
		// The cron job discards the TickResult — every
		// useful field is already written to the server
		// log. The admin "send all due reminders" route
		// calls Run directly and renders the result
		// into a result card; the cron path
		// intentionally does not duplicate that.
		func(ctx context.Context) { _, _ = weightReminder.Run(ctx) },
	)
	if err != nil {
		log.Fatalf("Failed to initialize reminder scheduler: %v", err)
	}
	scheduler.Start()
	defer scheduler.Stop()

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

// r2PhotoGetter adapts utils.GetObject to the export.PhotoGetter
// interface. Kept as its own type so a test fake can be substituted in
// if/when the export package is exercised via the controller.
type r2PhotoGetter struct{}

func (r2PhotoGetter) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return utils.GetObject(ctx, key)
}

// Compile-time check: r2PhotoGetter satisfies export.PhotoGetter.
var _ export.PhotoGetter = r2PhotoGetter{}

// utilsR2Uploader adapts utils.PutObject to the
// routes.exerciseImageUploader interface. Same shape as
// r2PhotoGetter (the read-side adapter used by the export flow);
// keeping it as a named type makes it easy to swap a fake in tests
// when the admin image route eventually needs end-to-end coverage.
type utilsR2Uploader struct{}

func (utilsR2Uploader) PutObject(ctx context.Context, key, contentType string, body io.Reader) error {
	return utils.PutObject(ctx, key, contentType, body)
}

// Compile-time check: utilsR2Uploader satisfies the
// routes.exerciseImageUploader interface.
var _ interface {
	PutObject(ctx context.Context, key, contentType string, body io.Reader) error
} = utilsR2Uploader{}
