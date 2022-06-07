package mysql

import (
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage/gensql"
)

func (s *MySQLStorage) StoreBootstrapToken(r *mdm.Request, msg *mdm.SetBootstrapToken) error {
	_, err := s.db.ExecContext(
		r.Context,
		`UPDATE devices SET bootstrap_token_b64 = ?, bootstrap_token_at = CURRENT_TIMESTAMP WHERE id = ? LIMIT 1;`,
		gensql.NullEmptyString(msg.BootstrapToken.BootstrapToken.String()),
		r.ID,
	)
	if err != nil {
		return err
	}
	return s.updateLastSeen(r)
}

func (s *MySQLStorage) RetrieveBootstrapToken(r *mdm.Request, _ *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	var tokenB64 string
	err := s.db.QueryRowContext(
		r.Context,
		`SELECT bootstrap_token_b64 FROM devices WHERE id = ?;`,
		r.ID,
	).Scan(&tokenB64)
	if err != nil {
		return nil, err
	}
	bsToken := new(mdm.BootstrapToken)
	err = bsToken.SetTokenString(tokenB64)
	if err == nil {
		err = s.updateLastSeen(r)
	}
	return bsToken, err
}
