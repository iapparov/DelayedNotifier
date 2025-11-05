package broker

import (
	"context"
	"delayedNotifier/internal/app"
	"delayedNotifier/internal/config"
	"encoding/json"
	"fmt"
	wbrabbit "github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/retry"
	wbzlog "github.com/wb-go/wbf/zlog"
	"time"
)

type RabbitService struct {
	client    *wbrabbit.Connection
	channel   *wbrabbit.Channel
	publisher publisherIface
	cfg       *config.RetrysConfig
	repo      StorageProvider
}

type StorageProvider interface {
	GetNotifications(status app.StatusType, batchSize int, lastId string) ([]*app.Notification, error)
}

type publisherIface interface {
	PublishWithRetry(body []byte, routingKey, contentType string, strategy retry.Strategy, options ...wbrabbit.PublishingOptions) error
}

func NewRabbitProducerService(cfg *config.AppConfig, repo StorageProvider) (*RabbitService, error) {
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

	ex := wbrabbit.NewExchange(cfg.RabbitmqConfig.Exchange, "direct")
	ex.Durable = true
	if err := ex.BindToChannel(ch); err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to declare exchange in RabbitMQ")
		return nil, err
	}

	qm := wbrabbit.NewQueueManager(ch)
	queue, err := qm.DeclareQueue(cfg.RabbitmqConfig.QueueName, wbrabbit.QueueConfig{
		Durable: true,
	})
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to declare queue in RabbitMQ")
		return nil, err
	}

	if err = ch.QueueBind(
		queue.Name,
		"notify",
		ex.Name(),
		false,
		nil,
	); err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to bind queue in RabbitMQ")
		return nil, err
	}

	publisher := wbrabbit.NewPublisher(ch, ex.Name())

	wbzlog.Logger.Info().Msg("Connected to RabbitMQ")
	return &RabbitService{client: client, channel: ch, publisher: publisher, cfg: &cfg.RetrysConfig, repo: repo}, nil
}

func (s *RabbitService) Close() error {
	err := retry.Do(func() error {

		err := s.channel.Close()
		if err != nil {
			return err
		}
		err = s.client.Close()
		if err != nil {
			return err
		}
		return nil
	}, retry.Strategy{Attempts: s.cfg.Attempts, Delay: s.cfg.Delay, Backoff: s.cfg.Backoffs})
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to close RabbitMQ connection")
		return err
	}
	return nil
}

func (s *RabbitService) Publish(notification *app.Notification) error {
	body, err := json.Marshal(notification)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to marshal notification")
		return err
	}
	err = s.publisher.PublishWithRetry(
		body,
		"notify",
		"application/json",
		retry.Strategy{
			Attempts: s.cfg.Attempts,
			Delay:    s.cfg.Delay,
			Backoff:  s.cfg.Backoffs,
		},
	)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to publish notification")
		return err
	}

	wbzlog.Logger.Info().
		Str("queue", "notifications_queue").
		Str("id", notification.ID.String()).
		Msg("Notification published to RabbitMQ")
	return nil
}

func (s *RabbitService) Channel() *wbrabbit.Channel {
	return s.channel
}

func (s *RabbitService) UploadFromDB(ctx context.Context) {
	const batchSize = 100
	lastID := "00000000-0000-0000-0000-000000000000" // минимальный UUID для первого запроса

	for {
		select {
		case <-ctx.Done():
			wbzlog.Logger.Info().Msg("Graceful shutdown: stopping Rabbit producer")
			return
		default:
		}

		notifications, err := s.repo.GetNotifications(app.Pending, batchSize, lastID)
		if err != nil {
			wbzlog.Logger.Error().Err(err).Msg("Failed to get notifications from DB")
			time.Sleep(2 * time.Second)
			continue
		}

		if len(notifications) == 0 {
			time.Sleep(2 * time.Second)
			lastID = "00000000-0000-0000-0000-000000000000"
			continue
		}

		for _, n := range notifications {
			select {
			case <-ctx.Done():
				wbzlog.Logger.Info().Msg("Context canceled during publishing — exiting cleanly")
				return
			default:
			}

			if err := s.Publish(n); err != nil {
				wbzlog.Logger.Error().Err(err).Msg("Failed to publish notification")
				continue
			}

			// обновляем lastID для следующего батча
			lastID = n.ID.String()
		}
	}
}
