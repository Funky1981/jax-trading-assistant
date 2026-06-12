package approvals

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeNotificationStore struct {
	claimed []*NotificationOutboxItem
	sent    []uuid.UUID
	failed  map[uuid.UUID]string
}

func (s *fakeNotificationStore) ClaimPendingNotificationOutboxItems(ctx context.Context, channel string, limit int) ([]*NotificationOutboxItem, error) {
	return s.claimed, nil
}

func (s *fakeNotificationStore) MarkNotificationOutboxSent(ctx context.Context, id uuid.UUID, sentAt time.Time) error {
	s.sent = append(s.sent, id)
	return nil
}

func (s *fakeNotificationStore) MarkNotificationOutboxFailed(ctx context.Context, id uuid.UUID, message string) error {
	if s.failed == nil {
		s.failed = map[uuid.UUID]string{}
	}
	s.failed[id] = message
	return nil
}

type fakeNotificationSender struct {
	fail bool
}

func (s fakeNotificationSender) SendNotification(ctx context.Context, item *NotificationOutboxItem) error {
	if s.fail {
		return errFakeNotificationSend
	}
	return nil
}

var errFakeNotificationSend = &fakeNotificationError{}

type fakeNotificationError struct{}

func (e *fakeNotificationError) Error() string { return "fake send failure" }

func TestNotificationDispatcherDispatchOnceMarksSentAndFailed(t *testing.T) {
	firstID := uuid.New()
	secondID := uuid.New()
	store := &fakeNotificationStore{
		claimed: []*NotificationOutboxItem{
			{ID: firstID, Channel: "telegram", Message: "first"},
			{ID: secondID, Channel: "telegram", Message: "second"},
		},
	}

	dispatcher := NewNotificationDispatcher(store, fakeNotificationSender{}, "telegram", 10)
	sent, err := dispatcher.DispatchOnce(context.Background())
	if err != nil {
		t.Fatalf("dispatch once: %v", err)
	}
	if sent != 2 || len(store.sent) != 2 {
		t.Fatalf("sent count = %d, marked sent = %d", sent, len(store.sent))
	}

	store = &fakeNotificationStore{claimed: []*NotificationOutboxItem{{ID: firstID, Channel: "telegram", Message: "first"}}}
	dispatcher = NewNotificationDispatcher(store, fakeNotificationSender{fail: true}, "telegram", 10)
	sent, err = dispatcher.DispatchOnce(context.Background())
	if err != nil {
		t.Fatalf("dispatch once should continue after send failure: %v", err)
	}
	if sent != 0 || store.failed[firstID] != "fake send failure" {
		t.Fatalf("failed dispatch not recorded: sent=%d failed=%v", sent, store.failed)
	}
}

func TestTelegramNotificationSenderPostsMessageAndButtons(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bottest-token/sendMessage" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode telegram request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	sender := &TelegramNotificationSender{
		BotToken:   "test-token",
		ChatID:     "chat-1",
		APIBaseURL: server.URL,
		HTTPClient: server.Client(),
	}
	item := &NotificationOutboxItem{
		ID:      uuid.New(),
		Channel: "telegram",
		Message: "ETF: SPY\nAction: BUY",
		Payload: map[string]any{
			"buttons": []any{
				map[string]any{"label": "Approve", "callbackData": "approve:token"},
			},
		},
	}

	if err := sender.SendNotification(context.Background(), item); err != nil {
		t.Fatalf("send notification: %v", err)
	}
	if received["chat_id"] != "chat-1" || received["text"] != item.Message {
		t.Fatalf("unexpected telegram payload: %#v", received)
	}
	if received["reply_markup"] == nil {
		t.Fatalf("telegram payload should include reply markup: %#v", received)
	}
}

func TestTelegramNotificationSenderConfiguresWebhook(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bottest-token/setWebhook" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode telegram request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	sender := &TelegramNotificationSender{
		BotToken:   "test-token",
		ChatID:     "chat-1",
		APIBaseURL: server.URL,
		HTTPClient: server.Client(),
	}

	if err := sender.ConfigureWebhook(context.Background(), "https://jax.example.com/api/v1/mobile/telegram/webhook"); err != nil {
		t.Fatalf("configure webhook: %v", err)
	}
	if received["url"] != "https://jax.example.com/api/v1/mobile/telegram/webhook" {
		t.Fatalf("unexpected webhook payload: %#v", received)
	}
}
