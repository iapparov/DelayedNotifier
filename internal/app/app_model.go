package app

import (
	"github.com/google/uuid"
	wbzlog "github.com/wb-go/wbf/zlog"
	"time"
)

type StatusType string

const (
	Pending  StatusType = "pending"
	Sent     StatusType = "sent"
	Failed   StatusType = "failed"
	Canceled StatusType = "canceled"
)

type ChannelType string

const (
	Email    ChannelType = "email"
	Telegram ChannelType = "telegram"
)

type Notification struct {
	ID        uuid.UUID   `db:"id" json:"id"`
	Channel   ChannelType `db:"channel" json:"channel"`
	Recipient string      `db:"recipient" json:"recipient"`
	Message   string      `db:"message" json:"message"`
	SendAt    time.Time   `db:"send_at" json:"send_at"`
	Status    StatusType  `db:"status" json:"status"` // pending, sent, failed, canceled
	CreatedAt time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt time.Time   `db:"updated_at" json:"updated_at"`
}

type NotificationRequest struct {
	Channel   string `json:"channel" binding:"required,oneof=email telegram"`
	Message   string `json:"message" binding:"required"`
	Recipient string `json:"recipient" binding:"required"`
	SendAt    string `json:"send_at" binding:"required,datetime=2006-01-02T15:04:05Z07:00"`
}

func NewNotification(req NotificationRequest) (*Notification, error) {
	sendAt, error := time.Parse(time.RFC3339, req.SendAt)

	if error != nil {
		wbzlog.Logger.Error().Err(error).Msg("Failed to parse send_at time, defaulting to now")
		sendAt = time.Now()
	}
	if req.Channel != "email" && req.Channel != "telegram" {
		wbzlog.Logger.Error().Str("channel", req.Channel).Msg("Invalid channel type, defaulting to email")
		req.Channel = "email"
	}

	if messageLen := len(req.Message); messageLen == 0 || messageLen > 1000 {
		wbzlog.Logger.Error().Int("message_length", messageLen).Msg("Invalid message length, defaulting to 'No message provided'")
		req.Message = "No message provided"
	}

	channel := ChannelType(req.Channel)

	// Log the channel type for debugging
	wbzlog.Logger.Debug().Str("channel", string(channel)).Msg("Creating new notification")
	return &Notification{
		ID:        uuid.New(),
		Channel:   channel,
		Message:   req.Message,
		Recipient: req.Recipient,
		SendAt:    sendAt,
		Status:    Pending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (n *Notification) MarkAsSent() {
	n.Status = Sent
	n.UpdatedAt = time.Now()
}

func (n *Notification) MarkAsFailed() {
	n.Status = Failed
	n.UpdatedAt = time.Now()
}

func (n *Notification) MarkAsCanceled() {
	n.Status = Canceled
	n.UpdatedAt = time.Now()
}

func IsValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
