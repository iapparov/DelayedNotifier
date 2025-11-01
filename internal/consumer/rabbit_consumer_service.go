package consumer

import (
	"DelayedNotifier/internal/app"
	"DelayedNotifier/internal/config"
	"DelayedNotifier/internal/sender"
	"context"
	"encoding/json"
	"sync"

	wbrabbit "github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/retry"
	wbzlog "github.com/wb-go/wbf/zlog"
	"fmt"
)

type RabbitConsumerService struct {
	consumer *wbrabbit.Consumer
	cfg *config.RetrysConfig
	repo StorageProvider
	cache CacheProvider
	sender map[app.ChannelType]sender.Sender
}

type StorageProvider interface {
	UpdateNotificationStatus(id string, status app.StatusType) error
}

type CacheProvider interface {
	SaveNotification(notif *app.Notification) error
}

func NewConsumer (cfg *config.AppConfig, sender *sender.SenderRegistry, repo StorageProvider, cache CacheProvider) (*RabbitConsumerService, error){
	config := wbrabbit.ConsumerConfig{
		Queue: cfg.RabbitmqConfig.QueueName,
		Consumer: "",
		AutoAck: false,
		Exclusive: false,
		NoLocal: false,
		NoWait: false,
		Args: nil,
	}
		rabbitDSN := fmt.Sprintf(
		"amqp://%s:%s@%s:%d/",
		cfg.RabbitmqConfig.User,
		cfg.RabbitmqConfig.Password,
		cfg.RabbitmqConfig.Host,
		cfg.RabbitmqConfig.Port,
	)
	client, err := wbrabbit.Connect(rabbitDSN, cfg.RetrysConfig.Attempts, cfg.RetrysConfig.Delay)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to connect to RabbitMQ")
		return nil, err
	}

	ch, err := client.Channel()
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to open channel in RabbitMQ")
		return nil, err
	}

	return &RabbitConsumerService{consumer: wbrabbit.NewConsumer(ch, &config), cfg: &cfg.RetrysConfig, sender: sender.All(), repo: repo, cache: cache}, nil
}

func (c *RabbitConsumerService) Start(ctx context.Context){
	var wg sync.WaitGroup
	msgChan := make(chan []byte)
	wg.Add(1)
	go func(){
		defer wg.Done()
		defer close(msgChan)
		err := c.consumer.ConsumeWithRetry(msgChan, retry.Strategy{Attempts: c.cfg.Attempts, Delay: c.cfg.Delay, Backoff: c.cfg.Backoffs})
		if err != nil {
			wbzlog.Logger.Debug().Msg("Failed to Consume RabbitMQ")
		}
	}()
	
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				wbzlog.Logger.Info().Msg("Consumer stopped by context cancel")
				return

			case body, ok := <-msgChan:
				if !ok {
					wbzlog.Logger.Info().Msg("Message channel closed, exiting consumer loop")
					return
				}

				var notif app.Notification
				if err := json.Unmarshal(body, &notif); err != nil {
					wbzlog.Logger.Error().Err(err).Msg("Failed to unmarshal notification")
					continue
				}

				wbzlog.Logger.Info().
					Str("id", notif.ID.String()).
					Str("channel", string(notif.Channel)).
					Msg("Received notification from queue")

				s, ok := c.sender[notif.Channel]
				if !ok {
					wbzlog.Logger.Error().
						Str("channel", string(notif.Channel)).
						Msg("Unknown notification channel")

					c.repo.UpdateNotificationStatus(notif.ID.String(), app.Failed)
					notif.MarkAsFailed()
					c.cache.SaveNotification(&notif)
					continue
				}

				if err := s.Send(&notif); err != nil {
					wbzlog.Logger.Error().
						Err(err).
						Str("id", notif.ID.String()).
						Msg("Failed to send notification")

					c.repo.UpdateNotificationStatus(notif.ID.String(), app.Failed)
					notif.MarkAsFailed()
					c.cache.SaveNotification(&notif)
					continue
				}

				c.repo.UpdateNotificationStatus(notif.ID.String(), app.Sent)
				notif.MarkAsSent()
				c.cache.SaveNotification(&notif)

				wbzlog.Logger.Info().
					Str("id", notif.ID.String()).
					Msg("Notification successfully sent")
			}
		}
	}()
	
	wg.Wait()
}