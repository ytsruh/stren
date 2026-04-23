package main

import (
	"log"
	"net"
	"os"

	"github.com/labstack/echo/v4"

	"stren/internal/db"
	"stren/internal/handlers"
	"stren/internal/models"
)

func main() {
	// Get database path from environment or use default
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "strength_tracker.db"
	}

	// Initialize database
	database, err := db.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize repository
	repo := models.NewExerciseRepository(database)

	// Initialize handlers
	h := handlers.NewHandler(repo)

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true

	// Register routes
	h.RegisterRoutes(e)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	localIP := getLocalIP()
	if localIP != "" {
		log.Printf("Server starting on http://localhost:%s and http://%s:%s", port, localIP, port)
	} else {
		log.Printf("Server starting on http://localhost:%s", port)
	}
	if err := e.Start(":" + port); err != nil {
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
