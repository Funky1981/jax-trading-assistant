package approvals

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

const defaultTelegramAPIBaseURL = "https://api.telegram.org"

type NotificationOutboxStore interface {
	ClaimPendingNotificationOutboxItems(ctx context.Context, channel string, limit int) ([]*NotificationOutboxItem, error)
	MarkNotificationOutboxSent(ctx context.Context, id uuid.UUID, sentAt time.Time) error
	MarkNotificationOutboxFailed(ctx context.Context, id uuid.UUID, message string) error
}

type NotificationSender interface {
	SendNotification(ctx context.Context, item *NotificationOutboxItem) error
}

type NotificationDispatcher struct {
	store   NotificationOutboxStore
	sender  NotificationSender
	channel string
	limit   int
	now     func() time.Time
}

func NewNotificationDispatcher(store NotificationOutboxStore, sender NotificationSender, channel string, limit int) *NotificationDispatcher {
	if limit <= 0 {
		limit = 25
	}
	if strings.TrimSpace(channel) == "" {
		channel = "telegram"
	}
	return &NotificationDispatcher{
		store:   store,
		sender:  sender,
		channel: channel,
		limit:   limit,
		now:     func() time.Time { return time.Now().UTC() },
	}
}

func (d *NotificationDispatcher) DispatchOnce(ctx context.Context) (int, error) {
	if d == nil || d.store == nil || d.sender == nil {
		return 0, errors.New("notification dispatcher is not configured")
	}
	items, err := d.store.ClaimPendingNotificationOutboxItems(ctx, d.channel, d.limit)
	if err != nil {
		return 0, err
	}
	sent := 0
	for _, item := range items {
		if err := d.sender.SendNotification(ctx, item); err != nil {
			_ = d.store.MarkNotificationOutboxFailed(ctx, item.ID, err.Error())
			continue
		}
		if err := d.store.MarkNotificationOutboxSent(ctx, item.ID, d.now()); err != nil {
			return sent, err
		}
		sent++
	}
	return sent, nil
}

type TelegramNotificationSender struct {
	BotToken   string
	ChatID     string
	APIBaseURL string
	HTTPClient *http.Client
}

func NewTelegramNotificationSenderFromEnv() (*TelegramNotificationSender, bool) {
	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	chatID := strings.TrimSpace(os.Getenv("TELEGRAM_CHAT_ID"))
	if token == "" || chatID == "" {
		return nil, false
	}
	baseURL := strings.TrimSpace(os.Getenv("TELEGRAM_API_BASE_URL"))
	if baseURL == "" {
		baseURL = defaultTelegramAPIBaseURL
	}
	return &TelegramNotificationSender{
		BotToken:   token,
		ChatID:     chatID,
		APIBaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}, true
}

func (s *TelegramNotificationSender) SendNotification(ctx context.Context, item *NotificationOutboxItem) error {
	if s == nil || strings.TrimSpace(s.BotToken) == "" || strings.TrimSpace(s.ChatID) == "" {
		return errors.New("telegram notification sender is not configured")
	}
	client := s.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	chatID := s.ChatID
	if item.Recipient != nil && strings.TrimSpace(*item.Recipient) != "" {
		chatID = strings.TrimSpace(*item.Recipient)
	}
	body := map[string]any{
		"chat_id": chatID,
		"text":    item.Message,
	}
	if replyMarkup := telegramReplyMarkup(item.Payload); replyMarkup != nil {
		body["reply_markup"] = replyMarkup
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal telegram notification: %w", err)
	}
	baseURL := strings.TrimRight(s.APIBaseURL, "/")
	if baseURL == "" {
		baseURL = defaultTelegramAPIBaseURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/bot%s/sendMessage", baseURL, s.BotToken), bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send telegram notification: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram notification failed: status %d", resp.StatusCode)
	}
	return nil
}

func (s *TelegramNotificationSender) ConfigureWebhook(ctx context.Context, webhookURL string) error {
	webhookURL = strings.TrimSpace(webhookURL)
	if webhookURL == "" {
		return nil
	}
	if s == nil || strings.TrimSpace(s.BotToken) == "" {
		return errors.New("telegram notification sender is not configured")
	}
	client := s.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	bodyBytes, err := json.Marshal(map[string]any{"url": webhookURL})
	if err != nil {
		return fmt.Errorf("marshal telegram webhook request: %w", err)
	}
	baseURL := strings.TrimRight(s.APIBaseURL, "/")
	if baseURL == "" {
		baseURL = defaultTelegramAPIBaseURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/bot%s/setWebhook", baseURL, s.BotToken), bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create telegram webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("configure telegram webhook: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram webhook configuration failed: status %d", resp.StatusCode)
	}
	return nil
}

func telegramReplyMarkup(payload map[string]any) map[string]any {
	rawButtons, ok := payload["buttons"].([]any)
	if !ok || len(rawButtons) == 0 {
		return nil
	}
	row := []map[string]string{}
	for _, raw := range rawButtons {
		buttonMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		label, _ := buttonMap["label"].(string)
		callbackData, _ := buttonMap["callbackData"].(string)
		if label == "" || callbackData == "" {
			continue
		}
		row = append(row, map[string]string{
			"text":          label,
			"callback_data": callbackData,
		})
	}
	if len(row) == 0 {
		return nil
	}
	return map[string]any{"inline_keyboard": []any{row}}
}
