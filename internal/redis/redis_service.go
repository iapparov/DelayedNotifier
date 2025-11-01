package redis

import (
	"DelayedNotifier/internal/app"
	"DelayedNotifier/internal/config"
	"context"
	"errors"
	"fmt"

	wbredis "github.com/wb-go/wbf/redis"
	"github.com/wb-go/wbf/retry"
	wbzlog "github.com/wb-go/wbf/zlog"
)

type RedisService struct {
	client *wbredis.Client
	cfg *config.RetrysConfig
}

type StorageProvider interface {
	UploadCache(limit int) ([]*app.Notification, error)
}

func NewRedisService(cfg *config.AppConfig) (*RedisService, error) {
	redisAddr := fmt.Sprintf("%s:%d", cfg.RedisConfig.Host, cfg.RedisConfig.Port)
	client := wbredis.New(redisAddr, cfg.RedisConfig.Password, cfg.RedisConfig.DB)
	wbzlog.Logger.Info().Msg("Connected to Redis")
	return &RedisService{client: client, cfg: &cfg.RetrysConfig}, nil
}

func (r *RedisService) LoadCache(cfg *config.AppConfig, repo StorageProvider) error{
	notifications, err := repo.UploadCache(cfg.RedisConfig.CacheSize)
	if err != nil {
		wbzlog.Logger.Debug().Msg("Failed to upload cache")
		return err
	}
	for _, n := range notifications {
		err := r.SaveNotification(n)
		if err != nil {
			wbzlog.Logger.Debug().Msg("Failed to save notification to cache")
			return err
		}
	}
	wbzlog.Logger.Debug().Msg("Succesful load cache")
	return nil
}

func (r *RedisService) GetNotification(id string) (*app.Notification, error) {
	ctx := context.Background()
	status, err := r.client.GetWithRetry(ctx, retry.Strategy{Attempts: r.cfg.Attempts, Delay: r.cfg.Delay, Backoff: r.cfg.Backoffs}, id)
	if err != nil {
			if errors.Is(err, wbredis.NoMatches) {
				// Ключ просто отсутствует
				wbzlog.Logger.Debug().
					Str("id", id).
					Msg("Notification status not found in Redis")
				return nil, nil
			}
		wbzlog.Logger.Warn().Err(err).Msg("Failed to get status by id")
		return nil, err
	}
	var notif app.Notification
	notif.Status = app.StatusType(status)
	wbzlog.Logger.Debug().Str("id", id).Str("status", status).Msg("Fetched notification status from Redis")
	return &notif, nil
}

func (r *RedisService) DeleteNotification(id string) error {
	ctx := context.Background()
	err := r.client.DelWithRetry(ctx, retry.Strategy{Attempts: r.cfg.Attempts, Delay: r.cfg.Delay, Backoff: r.cfg.Backoffs}, id)
	if err != nil {
		if errors.Is(err, wbredis.NoMatches) {
				// Ключ просто отсутствует
				wbzlog.Logger.Debug().
					Str("id", id).
					Msg("Notification status not found in Redis")
				return nil
			}
		wbzlog.Logger.Warn().Err(err).Msg("Failed to del status by id")
		return err
	}
	return nil
}

func (r *RedisService) SaveNotification(notification *app.Notification) error {
	id := notification.ID.String()
	status := notification.Status
	ctx := context.Background()
	err := r.client.SetWithRetry(ctx, retry.Strategy{Attempts: r.cfg.Attempts, Delay: r.cfg.Delay, Backoff: r.cfg.Backoffs}, id, string(status))
	if err != nil {
		wbzlog.Logger.Warn().Err(err).Msg("Failed to set status by id")
		return err
	}
	return nil
}

func (r *RedisService) Close() error {
	err := r.client.Close()
	if err != nil {
		wbzlog.Logger.Debug().Msg("Failed to close Redis connection")
		return err
	}
	return nil
}
