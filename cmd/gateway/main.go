package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"tenant-gateway/internal/admin"
	"tenant-gateway/internal/auth"
	"tenant-gateway/internal/config"
	"tenant-gateway/internal/database"
	"tenant-gateway/internal/proxy"
)

func main() {
	configPath := flag.String("config", "", "Path to config file")
	migrate := flag.Bool("migrate", false, "Run database migrations and exit")
	bootstrap := flag.String("bootstrap", "", "Create initial admin user with given username")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to database
	db, err := database.New(ctx, cfg.Database.URL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(ctx); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	if *migrate {
		log.Println("Migrations completed successfully")
		return
	}

	// Bootstrap admin user if requested
	if *bootstrap != "" {
		if err := bootstrapAdmin(ctx, db, *bootstrap, cfg.Auth.TokenHashCost); err != nil {
			log.Fatalf("Failed to bootstrap admin: %v", err)
		}
		return
	}

	// Create authenticator
	authenticator := auth.NewAuthenticator(db)

	// Create permission checker
	permissions := proxy.NewPermissionChecker(cfg.Endpoints.Read, cfg.Endpoints.Write)

	// Create proxy
	p, err := proxy.New(cfg.Upstream.URL, cfg.Upstream.Timeout, permissions)
	if err != nil {
		log.Fatalf("Failed to create proxy: %v", err)
	}

	// Create admin handlers
	adminHandlers := admin.NewHandlers(db, cfg.Auth.TokenHashCost)

	// Set up router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Health check (unauthenticated)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Swagger UI (unauthenticated)
	r.Get("/swagger", http.RedirectHandler("/swagger/", http.StatusMovedPermanently).ServeHTTP)
	r.Get("/swagger/", admin.SwaggerUI())
	r.Get("/swagger/openapi.yaml", admin.OpenAPISpec())

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(authenticator.Middleware)

		// Admin API
		r.Mount("/admin", admin.Routes(adminHandlers))

		// Proxy all other requests
		r.HandleFunc("/*", p.Handler().ServeHTTP)
	})

	// Create server
	server := &http.Server{
		Addr:    cfg.Server.Listen,
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting server on %s", cfg.Server.Listen)
		log.Printf("Proxying to %s", cfg.Upstream.URL)
		log.Printf("Swagger UI available at %s/swagger/", cfg.Server.Listen)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

func bootstrapAdmin(ctx context.Context, db *database.DB, username string, hashCost int) error {
	// Check if user already exists
	existing, err := db.GetUserByUsername(ctx, username)
	if err == nil {
		fmt.Printf("User '%s' already exists (ID: %s)\n", existing.Username, existing.ID)
		return nil
	}
	if err != database.ErrNotFound {
		return fmt.Errorf("checking existing user: %w", err)
	}

	// Create admin user
	user, err := db.CreateUser(ctx, username, true)
	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}

	fmt.Printf("Created admin user '%s' (ID: %s)\n", user.Username, user.ID)

	// Generate API key
	plaintext, hash, prefix, err := auth.GenerateToken(hashCost)
	if err != nil {
		return fmt.Errorf("generating API key: %w", err)
	}

	_, err = db.CreateAPIKey(ctx, user.ID, hash, prefix, "bootstrap", nil)
	if err != nil {
		return fmt.Errorf("creating API key: %w", err)
	}

	fmt.Printf("\nAPI Key (save this - it won't be shown again):\n%s\n", plaintext)

	return nil
}
