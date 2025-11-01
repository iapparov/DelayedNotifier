package di

import (
	rabbit "DelayedNotifier/internal/broker"
	"DelayedNotifier/internal/config"
	"DelayedNotifier/internal/consumer"
	"DelayedNotifier/internal/redis"
	"DelayedNotifier/internal/web"
	"context"
	"fmt"
	"log"
	"net/http"

	wbgin "github.com/wb-go/wbf/ginext"
	"go.uber.org/fx"
)

func StartHTTPServer(lc fx.Lifecycle, notifyHandler *web.NotifyHandler, config *config.AppConfig) {
	router := wbgin.New(config.GinConfig.Mode)
	
	router.Use(wbgin.Logger(), wbgin.Recovery())
		router.Use(func(c *wbgin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
    c.Next()
})

	web.RegisterRoutes(router, notifyHandler)

	addres := fmt.Sprintf("%s:%d", config.ServerConfig.Host, config.ServerConfig.Port)
	server := &http.Server{
		Addr:    addres,
		Handler: router.Engine, 
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Printf("Server started")
			go func() {
				if err := server.ListenAndServe(); err != nil {
					log.Printf("ListenAndServe error: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Printf("Shutting down server...")
			return server.Close()
		},
	})
}

func LoadCacheOnStart(lc fx.Lifecycle, c *redis.RedisService, repo redis.StorageProvider, cfg *config.AppConfig) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Println("Loading cache from DB on startup...")
			if err := c.LoadCache(cfg, repo); err != nil {
				log.Printf("Failed to load cache: %v", err)
				return err
			}
			log.Println("Cache loaded successfully")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("Closing Redis connection")
			return c.Close()
		},
	})
}

func StartRabitProducer(lc fx.Lifecycle, r *rabbit.RabbitService) {
		lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Println("Start Rabbit Producer...")
			producerCtx, cancel := context.WithCancel(context.Background())
			go r.UploadFromDB(producerCtx)

			lc.Append(fx.Hook{
                OnStop: func(ctx context.Context) error {
                    log.Println("Stopping Rabit producer")
                    cancel() // отменяем consumer
                    return nil
                },
            })
			log.Println("Rabbit Producer started successfully")
			return nil
		},
	})
}

func StartRabbitConsumer(lc fx.Lifecycle, r *consumer.RabbitConsumerService) {
    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            log.Println("Start Rabbit Consumer...")

            // Используем отдельный контекст для долгоживущей горутины
            consumerCtx, cancel := context.WithCancel(context.Background())
            go r.Start(consumerCtx)

            // Сохраняем cancel для корректного завершения
            lc.Append(fx.Hook{
                OnStop: func(ctx context.Context) error {
                    log.Println("Stopping Rabbit Consumer...")
                    cancel() // отменяем consumer
                    return nil
                },
            })

            log.Println("Rabbit Consumer started successfully")
            return nil
        },
    })
}

