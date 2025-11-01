package sender

import (
	"DelayedNotifier/internal/app"
	"DelayedNotifier/internal/config"
)

type SenderRegistry struct {
    senders map[app.ChannelType]Sender
}

type Sender interface {
	Send(notification *app.Notification) error
}

func NewSenderRegistry(cfg *config.AppConfig) *SenderRegistry {
	senders := make(map[app.ChannelType]Sender)

	senders[app.Telegram] = NewTelegramChannel(cfg)
	senders[app.Email] = NewEmailChannel(cfg)
    return &SenderRegistry{
        senders: senders,
    }
}

func (r *SenderRegistry) All() map[app.ChannelType]Sender {
    return r.senders
}