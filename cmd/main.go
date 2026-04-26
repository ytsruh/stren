package main

import (
	"log"
	"net"

	"github.com/labstack/echo/v4"

	"stren/internal/controllers"
	"stren/internal/db"
	"stren/internal/models"
	"stren/internal/routes"
	"stren/internal/utils"
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

	// Initialize auth service
	jwtService := utils.NewJWTService(cfg.JWT_SECRET)

	// Initialize controllers
	authCtrl := controllers.NewAuthController(userRepo, jwtService)
	entryCtrl := controllers.NewEntryController(repo)

	// Initialize route handlers
	h := routes.NewHandler(authCtrl, entryCtrl, jwtService)

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true

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
