package db

import (
	"context"
	"database/sql"
	"delayedNotifier/internal/app"
	"delayedNotifier/internal/config"
	"fmt"
	wbdb "github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/retry"
	wbzlog "github.com/wb-go/wbf/zlog"
	"time"
)

type Postgres struct {
	db  *wbdb.DB
	cfg *config.RetrysConfig
}

func NewPostgres(cfg *config.AppConfig) (*Postgres, error) {
	masterDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBConfig.Master.Host,
		cfg.DBConfig.Master.Port,
		cfg.DBConfig.Master.User,
		cfg.DBConfig.Master.Password,
		cfg.DBConfig.Master.DBName,
	)

	slaveDSNs := make([]string, 0, len(cfg.DBConfig.Slaves))
	for _, slave := range cfg.DBConfig.Slaves {
		dsn := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			slave.Host,
			slave.Port,
			slave.User,
			slave.Password,
			slave.DBName,
		)
		slaveDSNs = append(slaveDSNs, dsn)
	}
	var opts wbdb.Options
	opts.ConnMaxLifetime = cfg.DBConfig.ConnMaxLifetime
	opts.MaxIdleConns = cfg.DBConfig.MaxIdleConns
	opts.MaxOpenConns = cfg.DBConfig.MaxOpenConns
	db, err := wbdb.New(masterDSN, slaveDSNs, &opts)
	if err != nil {
		wbzlog.Logger.Debug().Msg("Failed to connect to Postgres")
		return nil, err
	}
	wbzlog.Logger.Info().Msg("Connected to Postgres")
	return &Postgres{db: db, cfg: &cfg.RetrysConfig}, nil
}

func (p *Postgres) Close() error {
	err := p.db.Master.Close()
	if err != nil {
		wbzlog.Logger.Debug().Msg("Failed to close Postgres connection")
		return err
	}
	for _, slave := range p.db.Slaves {
		if slave != nil {
			err := slave.Close()
			if err != nil {
				wbzlog.Logger.Debug().Msg("Failed to close Postgres slave connection")
				return err
			}
		}
	}
	return nil
}

func (p *Postgres) SaveNotification(notification *app.Notification) error {

	ctx := context.Background()

	query := `
		INSERT INTO notifications (id, channel, message, send_at, status, created_at, updated_at, recipient)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := p.db.ExecWithRetry(ctx, retry.Strategy{Attempts: p.cfg.Attempts, Delay: p.cfg.Delay, Backoff: p.cfg.Backoffs}, query,
		notification.ID,
		notification.Channel,
		notification.Message,
		notification.SendAt,
		notification.Status,
		notification.CreatedAt,
		notification.UpdatedAt,
		notification.Recipient,
	)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to execute insert notification query")
		return err
	}
	return nil

}

func (p *Postgres) GetNotifications(status app.StatusType, batchSize int, lastID string) ([]*app.Notification, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		UPDATE notifications
		SET status = 'processing'
		WHERE id IN (
			SELECT id
			FROM notifications
			WHERE status = $1
			AND id > $2
			AND send_at <= $4
			ORDER BY id ASC
			LIMIT $3
		)
		RETURNING id, channel, message, send_at, status, created_at, updated_at, recipient;
	`

	rows, err := p.db.QueryWithRetry(ctx, retry.Strategy{
		Attempts: p.cfg.Attempts,
		Delay:    p.cfg.Delay,
		Backoff:  p.cfg.Backoffs,
	}, query, status, lastID, batchSize, time.Now().Add(3*time.Second))
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to execute select notifications query")
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			wbzlog.Logger.Error().Err(err).Msg("Failed to close rows")
		}
	}()

	var notifications []*app.Notification
	for rows.Next() {
		var n app.Notification
		if err := rows.Scan(
			&n.ID,
			&n.Channel,
			&n.Message,
			&n.SendAt,
			&n.Status,
			&n.CreatedAt,
			&n.UpdatedAt,
			&n.Recipient,
		); err != nil {
			wbzlog.Logger.Error().Err(err).Msg("Failed to scan notification row")
			return nil, err
		}
		notifications = append(notifications, &n)
	}

	if err := rows.Err(); err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Row iteration error")
		return nil, err
	}

	return notifications, nil
}

func (p *Postgres) GetNotification(id string) (*app.Notification, error) {
	ctx := context.Background()
	query := `
		SELECT id, channel, message, send_at, status, created_at, updated_at, recipient
		FROM notifications
		WHERE id = $1
	`

	row, err := p.db.QueryRowWithRetry(ctx, retry.Strategy{Attempts: p.cfg.Attempts, Delay: p.cfg.Delay, Backoff: p.cfg.Backoffs}, query, id)

	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to execute select notifications query")
		return nil, err
	}
	var notification app.Notification
	err = row.Scan(
		&notification.ID,
		&notification.Channel,
		&notification.Message,
		&notification.SendAt,
		&notification.Status,
		&notification.CreatedAt,
		&notification.UpdatedAt,
		&notification.Recipient,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			wbzlog.Logger.Info().Str("id", id).Msg("Notification not found")
			return nil, nil // можно вернуть nil, nil, чтобы явно показать «не найдено»
		}
		wbzlog.Logger.Error().Err(err).Msg("Failed to scan notification row")
		return nil, err
	}
	if err = row.Err(); err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Row iteration error")
		return nil, err
	}
	return &notification, nil
}

func (p *Postgres) UpdateNotificationStatus(id string, status app.StatusType) error {
	ctx := context.Background()

	query := `
		UPDATE notifications
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := p.db.ExecWithRetry(ctx, retry.Strategy{Attempts: p.cfg.Attempts, Delay: p.cfg.Delay, Backoff: p.cfg.Backoffs}, query,
		status,
		time.Now(),
		id,
	)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to execute update notification status query")
		return err
	}
	return nil
}

func (p *Postgres) DeleteNotification(id string) error {
	ctx := context.Background()

	query := `
		DELETE FROM notifications
		WHERE id = $1
	`

	_, err := p.db.ExecWithRetry(ctx, retry.Strategy{Attempts: p.cfg.Attempts, Delay: p.cfg.Delay, Backoff: p.cfg.Backoffs}, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			wbzlog.Logger.Info().Str("id", id).Msg("Notification not found")
			return nil // можно вернуть nil, чтобы явно показать «не найдено»
		}
		wbzlog.Logger.Error().Err(err).Msg("Failed to execute delete notification query")
		return err
	}
	return nil
}

func (p *Postgres) UploadCache(limit int) ([]*app.Notification, error) {

	ctx := context.Background()

	query := `
		SELECT id, channel, message, send_at, status, created_at, updated_at, recipient
		FROM notifications
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := p.db.QueryWithRetry(ctx, retry.Strategy{Attempts: p.cfg.Attempts, Delay: p.cfg.Delay, Backoff: p.cfg.Backoffs}, query,
		limit,
	)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to execute select notifications query")
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			wbzlog.Logger.Error().Err(err).Msg("Failed to close rows")
		}
	}()

	var notifications []*app.Notification
	for rows.Next() {
		var n app.Notification
		err := rows.Scan(
			&n.ID,
			&n.Channel,
			&n.Message,
			&n.SendAt,
			&n.Status,
			&n.CreatedAt,
			&n.UpdatedAt,
			&n.Recipient,
		)
		if err != nil {
			wbzlog.Logger.Error().Err(err).Msg("Failed to scan notification row")
			return nil, err
		}
		notifications = append(notifications, &n)
	}

	if err = rows.Err(); err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Row iteration error")
		return nil, err
	}

	return notifications, nil
}
