package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-styler/internal/telegram"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
)

func main() {
	// Load configuration
	cfg, err := telegram.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Telegram.BotToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required. Please set it in your environment or .env file")
	}

	log.Printf("Starting Telegram bot in %s mode...", cfg.Telegram.Env)

	// Initialize database
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Redis
	redisClient, err := initRedis(cfg)
	if err != nil {
		log.Printf("Warning: Failed to initialize Redis: %v (continuing without Redis)", err)
		redisClient = nil
	}
	if redisClient != nil {
		defer redisClient.Close()
	}

	// Initialize storage
	storage, err := telegram.NewStorage(db, redisClient)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer storage.Close()

	// Initialize session manager
	sessionMgr := telegram.NewSessionManager(storage)

	// Initialize API client
	apiClient := telegram.NewAPIClient(cfg.API.BaseURL, cfg.API.APIKey, cfg.API.Timeout)

	// Initialize rate limiter
	rateLimiter := telegram.NewRateLimiter(redisClient)

	// Initialize handlers
	handlers := telegram.NewHandlers(nil, apiClient, sessionMgr, rateLimiter, cfg)

	// Initialize bot
	bot, err := telegram.NewBot(cfg, handlers)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Set bot in handlers
	handlers.SetBot(bot.GetBot())

	// Start health server
	healthServer := telegram.NewHealthServer(db, redisClient, cfg.Server.HealthPort)
	go func() {
		log.Printf("Starting health server on port %d", cfg.Server.HealthPort)
		if err := healthServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Printf("Health server error: %v", err)
		}
	}()

	// Start bot in a goroutine
	go func() {
		log.Printf("Starting Telegram bot...")
		if err := bot.Start(); err != nil {
			log.Fatalf("Bot failed: %v", err)
		}
	}()

	// Give bot time to start
	time.Sleep(2 * time.Second)
	log.Printf("✅ Bot service started successfully!")
	log.Printf("📱 Send /start to your bot to test it")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down bot...")

	// Stop bot
	bot.Stop()

	log.Println("Bot stopped")
}

// initDatabase initializes PostgreSQL connection
func initDatabase(cfg *telegram.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.Database.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// initRedis initializes Redis connection
func initRedis(cfg *telegram.Config) (*redis.Client, error) {
	opts, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		// Fallback to individual settings
		opts = &redis.Options{
			Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		}
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return client, nil
}

