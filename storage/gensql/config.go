package gensql

import (
	"database/sql"
	"fmt"

	"github.com/micromdm/nanomdm/log"
)

type Config struct {
	driver DriverName
	dsn    string
	db     *sql.DB
	logger log.Logger
	rm     bool
}

func (c *Config) Db() *sql.DB {
	return c.db
}
func (c *Config) Logger() log.Logger {
	return c.logger
}
func (c *Config) Rm() bool {
	return c.rm
}

type Option func(*Config)

func WithLogger(logger log.Logger) Option {
	return func(c *Config) {
		c.logger = logger
	}
}

func WithDSN(dsn string) Option {
	return func(c *Config) {
		c.dsn = dsn
	}
}

func WithDriver(driver DriverName) Option {
	return func(c *Config) {
		c.driver = driver
	}
}

func WithDeleteCommands() Option {
	return func(c *Config) {
		c.rm = true
	}
}

type DriverName string

const (
	MysqlDriver DriverName = "mysql"
	PgDriver    DriverName = "postgres"
)

var SupportedSQLDrivers = map[DriverName]struct{}{MysqlDriver: {}, PgDriver: {}}

func NewDB(opts []Option) (*Config, error) {
	cfg := &Config{logger: log.NopLogger}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.driver == "" {
		return nil, ErrEmptyDriverName
	}
	if _, ok := SupportedSQLDrivers[cfg.driver]; !ok {
		return nil, fmt.Errorf("%s: %w", cfg.driver, ErrUnsupportedSQLDriver)
	}

	var err error
	if cfg.db == nil {
		cfg.db, err = sql.Open(string(cfg.driver), cfg.dsn)
		if err != nil {
			return nil, err
		}
	}
	if err = cfg.db.Ping(); err != nil {
		return nil, err
	}
	return cfg, nil
}
