package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	approvalsmod "jax-trading-assistant/internal/modules/approvals"

	"github.com/jackc/pgx/v5/pgxpool"
)

func startMobileNotificationDispatcher(ctx context.Context, pool *pgxpool.Pool) {
	sender, ok := approvalsmod.NewTelegramNotificationSenderFromEnv()
	if !ok {
		log.Println("mobile notifications disabled: TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID are not configured")
		return
	}
	if webhookURL := strings.TrimSpace(os.Getenv("TELEGRAM_WEBHOOK_URL")); webhookURL != "" {
		if err := sender.ConfigureWebhook(ctx, webhookURL); err != nil {
			log.Printf("mobile notification webhook configuration warning: %v", err)
		} else {
			log.Println("mobile notification webhook configured for telegram")
		}
	}

	dispatcher := approvalsmod.NewNotificationDispatcher(approvalsmod.NewStore(pool), sender, "telegram", 25)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	log.Println("mobile notification dispatcher started for telegram")

	for {
		sent, err := dispatcher.DispatchOnce(ctx)
		if err != nil {
			log.Printf("mobile notification dispatch error: %v", err)
		} else if sent > 0 {
			log.Printf("mobile notification dispatcher sent %d notification(s)", sent)
		}

		select {
		case <-ctx.Done():
			log.Println("mobile notification dispatcher stopped")
			return
		case <-ticker.C:
		}
	}
}
