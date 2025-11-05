package config

import (
	"fmt"
	wbfconfig "github.com/wb-go/wbf/config"
	"os"
	"time"
)

type AppConfig struct {
	ServerConfig   ServerConfig   `mapstructure:"server"`
	LoggerConfig   loggerConfig   `mapstructure:"logger"`
	RabbitmqConfig RabbitmqConfig `mapstructure:"rabbitmq"`
	RedisConfig    redisConfig    `mapstructure:"redis"`
	DBConfig       dbConfig       `mapstructure:"db_config"`
	TelegramConfig telegramConfig `mapstructure:"telegram"`
	MailConfig     mailConfig     `mapstructure:"mail"`
	RetrysConfig   RetrysConfig   `mapstructure:"retry_strategy"`
	GinConfig      ginConfig      `mapstructure:"gin"`
}

type RetrysConfig struct {
	Attempts int           `mapstructure:"attempts" default:"3"`
	Delay    time.Duration `mapstructure:"delay" default:"1s"`
	Backoffs float64       `mapstructure:"backoffs" default:"2"`
}

type ginConfig struct {
	Mode string `mapstructure:"mode" default:"debug"`
}

type ServerConfig struct {
	Host string `mapstructure:"host" default:"localhost"`
	Port int    `mapstructure:"port" default:"8080"`
}

type loggerConfig struct {
	Level string `mapstructure:"level" default:"info"`
}

type RabbitmqConfig struct {
	Host      string `mapstructure:"host" default:"localhost"`
	Port      int    `mapstructure:"port" default:"5672"`
	User      string `mapstructure:"user" default:"guest"`
	Password  string `mapstructure:"password" default:"guest"`
	Exchange  string `mapstructure:"exchange" default:"notifications"`
	QueueName string `mapstructure:"queue_name" default:"notifications_queue"`
}

type redisConfig struct {
	Host      string `mapstructure:"host" default:"localhost"`
	Port      int    `mapstructure:"port" default:"6379"`
	Password  string `mapstructure:"password" default:""`
	DB        int    `mapstructure:"db" default:"0"`
	TTL       string `mapstructure:"ttl" default:"30s"`
	CacheSize int    `mapstructure:"cache_size" default:"1000"`
}

type postgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"db_name"`
	SSLMode  string `mapstructure:"ssl_mode" default:"disable"`
}

type dbConfig struct {
	Master          postgresConfig   `mapstructure:"postgres"`
	Slaves          []postgresConfig `mapstructure:"slaves"`
	MaxOpenConns    int              `mapstructure:"maxOpenConns"`
	MaxIdleConns    int              `mapstructure:"maxIdleConns"`
	ConnMaxLifetime time.Duration    `mapstructure:"connMaxLifetime"`
}

type telegramConfig struct {
	BotToken string `mapstructure:"bot_token" default:""`
}

type mailConfig struct {
	SMTPHost     string `mapstructure:"smtp_host" default:""`
	SMTPPort     int    `mapstructure:"smtp_port" default:"587"`
	SMTPEmail    string `mapstructure:"smtp_user" default:""`
	SMTPPassword string `mapstructure:"smtp_password" default:""`
}

func NewAppConfig() (*AppConfig, error) {
	envFilePath := "./.env"
	appConfigFilePath := "./config/local.yaml"

	cfg := wbfconfig.New()

	// Загрузка .env файлов
	if err := cfg.LoadEnvFiles(envFilePath); err != nil {
		return nil, fmt.Errorf("failed to load env files: %w", err)
	}

	// Включение поддержки переменных окружения
	cfg.EnableEnv("")

	if err := cfg.LoadConfigFiles(appConfigFilePath); err != nil {
		return nil, fmt.Errorf("failed to load config files: %w", err)
	}

	var appCfg AppConfig
	if err := cfg.Unmarshal(&appCfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	appCfg.DBConfig.Master.DBName = os.Getenv("POSTGRES_DB")
	appCfg.DBConfig.Master.User = os.Getenv("POSTGRES_USER")
	appCfg.DBConfig.Master.Password = os.Getenv("POSTGRES_PASSWORD")

	appCfg.RabbitmqConfig.User = os.Getenv("RABBITMQ_USER")
	appCfg.RabbitmqConfig.Password = os.Getenv("RABBITMQ_PASSWORD")

	appCfg.RedisConfig.Password = os.Getenv("REDIS_PASSWORD")

	appCfg.TelegramConfig.BotToken = os.Getenv("TELEGRAM_BOT_TOKEN")

	appCfg.MailConfig.SMTPEmail = os.Getenv("MAIL_SMTP_USER")
	appCfg.MailConfig.SMTPPassword = os.Getenv("MAIL_SMTP_PASSWORD")
	fmt.Println(appCfg.DBConfig)

	return &appCfg, nil
}
