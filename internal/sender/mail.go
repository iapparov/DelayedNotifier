package sender

import (
	"delayedNotifier/internal/app"
	"delayedNotifier/internal/config"
	"fmt"
	"net/smtp"
)

type EmailChannel struct {
	smtpHost  string
	smtpPort  int
	smtpEmail string
	smtp      string
}

func NewEmailChannel(cfg *config.AppConfig) *EmailChannel {
	return &EmailChannel{
		smtpHost:  cfg.MailConfig.SMTPHost,
		smtpPort:  cfg.MailConfig.SMTPPort,
		smtpEmail: cfg.MailConfig.SMTPEmail,
		smtp:      cfg.MailConfig.SMTPPassword,
	}
}

func (s *EmailChannel) Send(notification *app.Notification) error {
	auth := smtp.PlainAuth("", s.smtpEmail, s.smtp, s.smtpHost)
	to := []string{notification.Recipient}
	msg := []byte("To: " + notification.Recipient + "\r\n" +
		"Subject: Notification" + "\r\n" +
		"\r\n" +
		notification.Message + "\r\n")
	addr := s.smtpHost + ":" + fmt.Sprint(s.smtpPort)
	err := smtp.SendMail(addr, auth, s.smtpEmail, to, msg)
	if err != nil {
		return err
	}
	return nil
}
