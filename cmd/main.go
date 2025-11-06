// @title           DelayedNotifier API
// @version         1.0
// @description     API для сервиса отложенных уведомлений
// @BasePath        /

package main

import (
	"delayedNotifier/internal/broker"
	"delayedNotifier/internal/config"
	"delayedNotifier/internal/consumer"
	"delayedNotifier/internal/db"
	"delayedNotifier/internal/di"
	"delayedNotifier/internal/redis"
	"delayedNotifier/internal/sender"
	"delayedNotifier/internal/web"
	wbzlog "github.com/wb-go/wbf/zlog"
	"go.uber.org/fx"
)

func main() {
	wbzlog.Init()
	app := fx.New(
		fx.Provide(
			config.NewAppConfig,
			db.NewPostgres,

			redis.NewRedisService,
			func(db *db.Postgres) redis.StorageProvider {
				return db
			},

			func(db *db.Postgres) broker.StorageProvider {
				return db
			},

			broker.NewRabbitProducerService,

			sender.NewSenderRegistry,

			consumer.NewConsumer,
			func(db *db.Postgres) consumer.StorageProvider {
				return db
			},
			func(redis *redis.RedisService) consumer.CacheProvider {
				return redis
			},

			web.NewNotifyHandler,
			func(db *db.Postgres) web.StorageProvider {
				return db
			},
			func(redis *redis.RedisService) web.CacheProvider {
				return redis
			},
		),
		fx.Invoke(
			di.StartHTTPServer,
			di.LoadCacheOnStart,
			di.StartRabitProducer,
			di.StartRabbitConsumer,
			di.ClosePostgresOnStop,
		),
	)

	app.Run()
}
