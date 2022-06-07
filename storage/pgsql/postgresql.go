// Package pgsql stores and retrieves MDM data from PostgresSQL
package pgsql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/micromdm/nanomdm/cryptoutil"
	"github.com/micromdm/nanomdm/log"
	"github.com/micromdm/nanomdm/log/ctxlog"
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
	"github.com/micromdm/nanomdm/storage/gensql"
)

type PgSQLStorage struct {
	logger log.Logger
	db     *sql.DB
	rm     bool
}

// Verify PgSQLStorage interface compliance at compile time
var _ storage.AllStorage = (*PgSQLStorage)(nil)

func New(opts ...gensql.Option) (*PgSQLStorage, error) {
	opts = append(opts, gensql.WithDriver(gensql.PgDriver))
	cfg, err := gensql.NewDB(opts)
	if err != nil {
		return nil, err
	}
	return &PgSQLStorage{db: cfg.Db(), logger: cfg.Logger(), rm: cfg.Rm()}, nil
}

func (s *PgSQLStorage) StoreAuthenticate(r *mdm.Request, msg *mdm.Authenticate) error {
	var pemCert []byte
	if r.Certificate != nil {
		pemCert = cryptoutil.PEMCertificate(r.Certificate.Raw)
	}
	_, err := s.db.ExecContext(
		r.Context, `
INSERT INTO devices
    (id, identity_cert, serial_number, authenticate, authenticate_at)
VALUES
    ($1, $2, $3, $4, CURRENT_TIMESTAMP)
ON CONFLICT ON CONSTRAINT devices_pkey DO
UPDATE SET
    identity_cert = EXCLUDED.identity_cert,
    serial_number = EXCLUDED.serial_number,
    authenticate = EXCLUDED.authenticate,
    authenticate_at = CURRENT_TIMESTAMP;`,
		r.ID, gensql.NullEmptyString(string(pemCert)), gensql.NullEmptyString(msg.SerialNumber), msg.Raw,
	)
	return err
}

func (s *PgSQLStorage) storeDeviceTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	query := `UPDATE devices SET token_update = $1, token_update_at = CURRENT_TIMESTAMP`
	where := ` WHERE id = $2;`
	args := []interface{}{msg.Raw}
	// separately store the Unlock Token per MDM spec
	if len(msg.UnlockToken) > 0 {
		query += `, unlock_token = $2, unlock_token_at = CURRENT_TIMESTAMP `
		args = append(args, msg.UnlockToken)
		where = ` WHERE id = $3;`
	}
	args = append(args, r.ID)
	_, err := s.db.ExecContext(r.Context, query+where, args...)
	return err
}

func (s *PgSQLStorage) storeUserTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	// there shouldn't be an Unlock Token on the user channel, but
	// complain if there is to warn an admin
	if len(msg.UnlockToken) > 0 {
		ctxlog.Logger(r.Context, s.logger).Info(
			"msg", "Unlock Token on user channel not stored",
		)
	}
	_, err := s.db.ExecContext(
		r.Context, `
INSERT INTO users
    (id, device_id, user_short_name, user_long_name, token_update, token_update_at)
VALUES
    ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
ON CONFLICT ON CONSTRAINT users_pkey DO UPDATE 
SET 
    device_id = EXCLUDED.device_id,
    user_short_name = EXCLUDED.user_short_name,
    user_long_name = EXCLUDED.user_long_name,
    token_update = EXCLUDED.token_update,
    token_update_at = CURRENT_TIMESTAMP;`,
		r.ID,
		r.ParentID,
		gensql.NullEmptyString(msg.UserShortName),
		gensql.NullEmptyString(msg.UserLongName),
		msg.Raw,
	)
	return err
}

func (s *PgSQLStorage) StoreTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	var err error
	var deviceId, userId string
	resolved := (&msg.Enrollment).Resolved()
	if err = resolved.Validate(); err != nil {
		return err
	}
	if resolved.IsUserChannel {
		deviceId = r.ParentID
		userId = r.ID
		err = s.storeUserTokenUpdate(r, msg)
	} else {
		deviceId = r.ID
		err = s.storeDeviceTokenUpdate(r, msg)
	}
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(
		r.Context, `
INSERT INTO enrollments
	(id, device_id, user_id, type, topic, push_magic, token_hex, last_seen_at, token_update_tally)
VALUES
	($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, 1)
ON CONFLICT ON CONSTRAINT enrollments_pkey DO UPDATE 
SET
    device_id = EXCLUDED.device_id,
    user_id = EXCLUDED.user_id,
    type = EXCLUDED.type,
    topic = EXCLUDED.topic,
    push_magic = EXCLUDED.push_magic,
    token_hex = EXCLUDED.token_hex,
	enabled = TRUE,
	last_seen_at = CURRENT_TIMESTAMP,
	token_update_tally = enrollments.token_update_tally + 1;`,
		r.ID,
		deviceId,
		gensql.NullEmptyString(userId),
		r.Type.String(),
		msg.Topic,
		msg.PushMagic,
		msg.Token.String(),
	)
	return err
}

func (s *PgSQLStorage) RetrieveTokenUpdateTally(ctx context.Context, id string) (int, error) {
	var tally int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT token_update_tally FROM enrollments WHERE id = $1;`,
		id,
	).Scan(&tally)
	return tally, err
}

func (s *PgSQLStorage) StoreUserAuthenticate(r *mdm.Request, msg *mdm.UserAuthenticate) error {
	colName := "user_authenticate"
	colAtName := "user_authenticate_at"
	// if the DigestResponse is empty then this is the first (of two)
	// UserAuthenticate messages depending on our response
	if msg.DigestResponse != "" {
		colName = "user_authenticate_digest"
		colAtName = "user_authenticate_digest_at"
	}
	_, err := s.db.ExecContext(
		r.Context, `
INSERT INTO users
    (id, device_id, user_short_name, user_long_name, `+colName+`, `+colAtName+`)
VALUES
    ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
ON CONFLICT ON CONSTRAINT users_pkey DO UPDATE 
SET
    device_id = EXCLUDED.device_id,
    user_short_name = EXCLUDED.user_short_name,
    user_long_name = EXCLUDED.user_long_name,
    `+colName+` = EXCLUDED.`+colName+`,
    `+colAtName+` = EXCLUDED.`+colAtName+`;`,
		r.ID,
		r.ParentID,
		gensql.NullEmptyString(msg.UserShortName),
		gensql.NullEmptyString(msg.UserLongName),
		msg.Raw,
	)
	if err != nil {
		return err
	}
	return s.updateLastSeen(r)
}

// Disable can be called for an Authenticate or CheckOut message
func (s *PgSQLStorage) Disable(r *mdm.Request) error {
	if r.ParentID != "" {
		return errors.New("can only disable a device channel")
	}
	_, err := s.db.ExecContext(
		r.Context,
		`UPDATE enrollments SET enabled = FALSE, token_update_tally = 0, last_seen_at = CURRENT_TIMESTAMP WHERE device_id = $1 AND enabled = TRUE;`,
		r.ID,
	)
	return err
}

func (s *PgSQLStorage) updateLastSeen(r *mdm.Request) (err error) {
	_, err = s.db.ExecContext(
		r.Context,
		`UPDATE enrollments SET last_seen_at = CURRENT_TIMESTAMP WHERE id = $1`,
		r.ID,
	)
	if err != nil {
		err = fmt.Errorf("updating last seen: %w", err)
	}
	return
}
