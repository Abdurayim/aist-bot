package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"aist-bot/internal/config"
	"aist-bot/internal/database"
	"aist-bot/internal/handlers"
	"aist-bot/internal/services"
	"aist-bot/internal/systemd"
)

func main() {
	log.SetFlags(log.LstdFlags)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	if err := database.Connect(&cfg.Database); err != nil {
		log.Fatalf("database error: %v", err)
	}
	defer func() { _ = database.Close() }()

	log.Printf("✓ database ready at %s", cfg.Database.Path)

	bot, err := connectWithRetry(cfg)
	if err != nil {
		log.Fatalf("telegram error: %v", err)
	}

	log.Printf("✓ connected as @%s", bot.Bot.Self.UserName)

	if err := bot.SyncAdmins(); err != nil {
		log.Fatalf("failed to sync admins: %v", err)
	}

	if err := bot.RegisterCommands(); err != nil {
		log.Printf("warning: %v", err)
	} else {
		log.Println("✓ command menu published in all three languages")
	}

	log.Printf("✓ %d admin phone number(s) configured", len(cfg.Admin.PhoneNumbers))
	reportUnlinkedAdmins(bot)

	// Abandoned half-filled forms are cleared once a day.
	stop := make(chan struct{})
	go runCleanup(bot, stop)

	// Tell the service manager the bot is up. Until this is sent the unit
	// counts as still starting, so a bot that cannot reach Telegram at boot
	// is reported as a failed start rather than a healthy one.
	if err := systemd.Ready(); err != nil {
		log.Printf("warning: %v", err)
	}

	if interval, running := systemd.StartWatchdog(stop); running {
		log.Printf("✓ watchdog active, reporting every %s", interval)
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	if cfg.Bot.WebhookURL != "" {
		runWebhook(cfg, bot, shutdown)
	} else {
		runPolling(bot, shutdown)
	}

	_ = systemd.Stopping()

	close(stop)
	log.Println("bot stopped")
}

// connectWithRetry opens the Telegram connection, retrying a few times
// before giving up.
//
// Reaching api.telegram.org is not always instant from where this bot runs,
// and a server reboot often brings the bot up before the network has
// settled. Without this, the first attempt fails, systemd restarts the
// process, and the journal fills with crashes for what is really just a
// slow link.
func connectWithRetry(cfg *config.Config) (*services.BotService, error) {
	const attempts = 6

	var lastErr error

	for attempt := 1; attempt <= attempts; attempt++ {
		bot, err := services.NewBotService(cfg, database.DB)
		if err == nil {
			return bot, nil
		}

		lastErr = err

		if attempt == attempts {
			break
		}

		wait := time.Duration(attempt*5) * time.Second
		log.Printf("could not reach Telegram (attempt %d/%d): %v — retrying in %s",
			attempt, attempts, err, wait)
		time.Sleep(wait)
	}

	return nil, fmt.Errorf("gave up after %d attempts: %w", attempts, lastErr)
}

// runPolling is the default mode: no public URL or TLS certificate needed,
// which is what a school deployment usually has.
func runPolling(bot *services.BotService, shutdown <-chan os.Signal) {
	if err := bot.RemoveWebhook(); err != nil {
		log.Printf("warning: %v", err)
	}

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 50

	updates := bot.Bot.GetUpdatesChan(updateConfig)

	log.Println("🔄 listening for messages (polling mode)")
	log.Println(strings.Repeat("─", 52))

	for {
		select {
		case update := <-updates:
			// Each update is handled on its own goroutine so a slow PDF
			// never blocks other parents.
			go handlers.HandleUpdate(bot, update)

		case <-shutdown:
			log.Println("shutting down...")
			bot.Bot.StopReceivingUpdates()
			return
		}
	}
}

// runWebhook serves Telegram directly, for deployments behind a public URL.
func runWebhook(cfg *config.Config, bot *services.BotService, shutdown <-chan os.Signal) {
	webhookURL := strings.TrimRight(cfg.Bot.WebhookURL, "/") + "/webhook"

	if err := bot.SetWebhook(webhookURL); err != nil {
		log.Fatalf("failed to set webhook: %v", err)
	}

	log.Printf("✓ webhook set to %s", webhookURL)

	mux := http.NewServeMux()

	mux.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var update tgbotapi.Update
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		// Answer Telegram immediately; the work happens in the background.
		go handlers.HandleUpdate(bot, update)

		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := database.HealthCheck(); err != nil {
			http.Error(w, "unhealthy", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("🌐 listening on :%s (webhook mode)", cfg.Server.Port)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-shutdown
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = server.Shutdown(shutdownCtx)
}

// runCleanup drops conversation states nobody came back to.
func runCleanup(bot *services.BotService, stop <-chan struct{}) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bot.Appeals.CleanupStates()
		case <-stop:
			return
		}
	}
}

// reportUnlinkedAdmins warns at startup about admins who have not opened the
// bot yet, because until they do they cannot receive a single appeal. This
// is the most common deployment mistake, so it is called out loudly.
func reportUnlinkedAdmins(bot *services.BotService) {
	admins, err := bot.Admins.GetAll()
	if err != nil {
		log.Printf("warning: failed to check admins: %v", err)
		return
	}

	var pending []string
	for _, admin := range admins {
		if !admin.IsLinked() {
			pending = append(pending, admin.PhoneNumber)
		}
	}

	if len(pending) == 0 {
		log.Println("✓ every admin is connected and will receive appeals")
		return
	}

	log.Printf("⚠ %d admin(s) have not opened the bot yet and will NOT receive appeals:", len(pending))
	for _, phone := range pending {
		log.Printf("    %s — ask them to send /start and share this number", phone)
	}
}
