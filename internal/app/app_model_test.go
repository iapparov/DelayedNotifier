package app

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewNotificationValidInput(t *testing.T) {
	req := NotificationRequest{
		Channel:   "email",
		Message:   "Hello, world!",
		Recipient: "test@example.com",
		SendAt:    time.Now().UTC().Format(time.RFC3339),
	}

	notification, err := NewNotification(req)

	assert.NoError(t, err)
	assert.NotNil(t, notification)
	assert.Equal(t, Email, notification.Channel)
	assert.Equal(t, req.Message, notification.Message)
	assert.Equal(t, req.Recipient, notification.Recipient)
	assert.Equal(t, Pending, notification.Status)
	assert.True(t, notification.SendAt.Before(time.Now().Add(1*time.Second)))
	assert.NotZero(t, notification.ID)
}

func TestNewNotificationInvalidSendAt(t *testing.T) {
	req := NotificationRequest{
		Channel:   "email",
		Message:   "Invalid time test",
		Recipient: "test@example.com",
		SendAt:    "not-a-time",
	}

	n, err := NewNotification(req)
	assert.NoError(t, err)
	assert.WithinDuration(t, time.Now(), n.SendAt, time.Second*2, "Should default to now when send_at invalid")
}

func TestNewNotificationInvalidChannel(t *testing.T) {
	req := NotificationRequest{
		Channel:   "whatsapp",
		Message:   "Test message",
		Recipient: "test@example.com",
		SendAt:    time.Now().Format(time.RFC3339),
	}

	n, err := NewNotification(req)
	assert.NoError(t, err)
	assert.Equal(t, Email, n.Channel, "Should default to email when channel invalid")
}

func TestNewNotificationInvalidMessageLength(t *testing.T) {
	req := NotificationRequest{
		Channel:   "email",
		Message:   "", // пустое сообщение
		Recipient: "test@example.com",
		SendAt:    time.Now().Format(time.RFC3339),
	}

	n, err := NewNotification(req)
	assert.NoError(t, err)
	assert.Equal(t, "No message provided", n.Message)
}

func TestNewNotificationLongMessage(t *testing.T) {
	longMsg := make([]byte, 1001)
	for i := range longMsg {
		longMsg[i] = 'a'
	}

	req := NotificationRequest{
		Channel:   "email",
		Message:   string(longMsg),
		Recipient: "test@example.com",
		SendAt:    time.Now().Format(time.RFC3339),
	}

	n, err := NewNotification(req)
	assert.NoError(t, err)
	assert.Equal(t, "No message provided", n.Message)
}

func TestMarkAsSent(t *testing.T) {
	n := &Notification{Status: Pending}
	n.MarkAsSent()
	assert.Equal(t, Sent, n.Status)
	assert.WithinDuration(t, time.Now(), n.UpdatedAt, time.Second)
}

func TestMarkAsFailed(t *testing.T) {
	n := &Notification{Status: Pending}
	n.MarkAsFailed()
	assert.Equal(t, Failed, n.Status)
	assert.WithinDuration(t, time.Now(), n.UpdatedAt, time.Second)
}

func TestMarkAsCanceled(t *testing.T) {
	n := &Notification{Status: Pending}
	n.MarkAsCanceled()
	assert.Equal(t, Canceled, n.Status)
	assert.WithinDuration(t, time.Now(), n.UpdatedAt, time.Second)
}

func TestIsValidUUID(t *testing.T) {
	valid := "550e8400-e29b-41d4-a716-446655440000"
	invalid := "not-a-uuid"

	assert.True(t, IsValidUUID(valid))
	assert.False(t, IsValidUUID(invalid))
}
