package common

import (
	"crypto/tls"
	"time"

	"github.com/RichardKnop/machinery/v1/config"
	"github.com/gomodule/redigo/redis"
	"strings"
	"strconv"
	neturl "net/url"
	"errors"
)

var (
	defaultConfig = &config.RedisConfig{
		MaxIdle:                10,
		MaxActive:              100,
		IdleTimeout:            300,
		Wait:                   true,
		ReadTimeout:            15,
		WriteTimeout:           15,
		ConnectTimeout:         15,
		NormalTasksPollPeriod:  1000,
		DelayedTasksPollPeriod: 20,
	}
)

type OtherGoRedisOptions struct {
	MaxRetries      int
	MinRetryBackoff time.Duration
	MaxRetryBackoff time.Duration
}

// RedisConnector ...
type RedisConnector struct{}

// NewPool returns a new pool of Redis connections
func (rc *RedisConnector) NewPool(socketPath, host, password string, db int, cnf *config.RedisConfig, tlsConfig *tls.Config) *redis.Pool {
	if cnf == nil {
		cnf = defaultConfig
	}
	return &redis.Pool{
		MaxIdle:     cnf.MaxIdle,
		IdleTimeout: time.Duration(cnf.IdleTimeout) * time.Second,
		MaxActive:   cnf.MaxActive,
		Wait:        cnf.Wait,
		Dial: func() (redis.Conn, error) {
			c, err := rc.open(socketPath, host, password, db, cnf, tlsConfig)
			if err != nil {
				return nil, err
			}

			if db != 0 {
				_, err = c.Do("SELECT", db)
				if err != nil {
					return nil, err
				}
			}

			return c, err
		},
		// PINGs connections that have been idle more than 10 seconds
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Duration(10*time.Second) {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}

// Open a new Redis connection
func (rc *RedisConnector) open(socketPath, host, password string, db int, cnf *config.RedisConfig, tlsConfig *tls.Config) (redis.Conn, error) {
	var opts = []redis.DialOption{
		redis.DialDatabase(db),
		redis.DialReadTimeout(time.Duration(cnf.ReadTimeout) * time.Second),
		redis.DialWriteTimeout(time.Duration(cnf.WriteTimeout) * time.Second),
		redis.DialConnectTimeout(time.Duration(cnf.ConnectTimeout) * time.Second),
	}

	if tlsConfig != nil {
		opts = append(opts, redis.DialTLSConfig(tlsConfig), redis.DialUseTLS(true))
	}

	if password != "" {
		opts = append(opts, redis.DialPassword(password))
	}

	if socketPath != "" {
		return redis.Dial("unix", socketPath, opts...)
	}

	return redis.Dial("tcp", host, opts...)
}

// ParseRedisURL ...
func ParseRedisURL(url string) (host, password string, db int, err error) {
	// redis://pwd@host/db

	var u *neturl.URL
	u, err = neturl.Parse(url)
	if err != nil {
		return
	}
	if u.Scheme != "redis" {
		err = errors.New("No redis scheme found")
		return
	}

	if u.User != nil {
		var exists bool
		password, exists = u.User.Password()
		if !exists {
			password = u.User.Username()
		}
	}

	host = u.Host

	parts := strings.Split(u.Path, "/")
	if len(parts) == 1 {
		db = 0 //default redis db
	} else {
		db, err = strconv.Atoi(parts[1])
		if err != nil {
			db, err = 0, nil //ignore err here
		}
	}

	return
}