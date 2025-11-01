package sender

import (
	"fmt"
	"log"
	"strconv"
	"DelayedNotifier/internal/config"
	"DelayedNotifier/internal/app"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	wbzlog "github.com/wb-go/wbf/zlog"
)

type TelegramChannel struct {
	bot *tgbotapi.BotAPI
}


func NewTelegramChannel(cfg *config.AppConfig) *TelegramChannel {
	bot, err := tgbotapi.NewBotAPI(cfg.TelegramConfig.BotToken)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to create Telegram bot")
		return nil
	}

	tc := &TelegramChannel{bot: bot}
	go tc.listenForStartCommand()
	return tc
}

// Send — реализация интерфейса Sender
func (t *TelegramChannel) Send(notification *app.Notification) error {
	chatId, err := strconv.Atoi(notification.Recipient)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}
	msg := tgbotapi.NewMessage(int64(chatId), notification.Message)
	_, err = t.bot.Send(msg)
	return err
}


func (t *TelegramChannel) listenForStartCommand() {
	log.Println("Telegram listener started...")
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := t.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.Text == "/start" {
			chatID := update.Message.Chat.ID
			username := update.Message.From.UserName

			// логируем и уведомляем пользователя
			log.Printf("[TG] User %s started bot, chat_id=%d", username, chatID)

			msg := tgbotapi.NewMessage(chatID,
				fmt.Sprintf("👋 Привет, %s!\n\nТвой chat_id: `%d`\nОтправь его в приложение, чтобы получать уведомления.",
					username, chatID))

			t.bot.Send(msg)
			
			//TODO: сохранить chatID и username в БД или кэш
			// _ = saveUserToDB(username, chatID)  
		}
	}
}