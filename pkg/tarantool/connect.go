package tarantool

import (
	"context"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
	tnt "github.com/tarantool/go-tarantool/v2"
)

type Client struct {
	conn           *tnt.Connection
	requestTimeout time.Duration
}

func InitTarantool(ctx context.Context) (*Client, error) {
	dialTimeout, _ := time.ParseDuration(utils.EnvString("TARANTOOL_DIAL_TIMEOUT", "3s"))
	requestTimeout, _ := time.ParseDuration(utils.EnvString("TARANTOOL_TIMEOUT", "500ms"))
	reconnectTimeout, _ := time.ParseDuration(utils.EnvString("TARANTOOL_RECONNECT_TIMEOUT", "1s"))

	connectCtx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()

	dialer := tnt.NetDialer{
		Address: fmt.Sprintf("%s:%s", utils.EnvString("TARANTOOL_HOST", "tarantool"), utils.EnvString("TARANTOOL_PORT", "3301")),
		User:    utils.EnvString("TARANTOOL_USER", "cache"),
		Password: utils.EnvString(
			"TARANTOOL_PASSWORD",
			"local-tarantool-password",
		),
	}

	conn, err := tnt.Connect(connectCtx, dialer, tnt.Opts{
		Timeout:       requestTimeout,
		Reconnect:     reconnectTimeout,
		MaxReconnects: uint(utils.EnvInt("TARANTOOL_MAX_RECONNECTS", 3)),
	})
	if err != nil {
		return nil, err
	}

	client := &Client{conn: conn, requestTimeout: requestTimeout}
	if err := client.Ping(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

func (c *Client) Ping(ctx context.Context) error {
	if err := c.ensure(); err != nil {
		return err
	}
	reqCtx, cancel := c.withTimeout(ctx)
	defer cancel()
	_, err := c.conn.Do(tnt.NewPingRequest().Context(reqCtx)).Get()
	return err
}

func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) ensure() error {
	if c == nil || c.conn == nil || c.conn.ClosedNow() {
		return ErrUnavailable
	}
	return nil
}

func (c *Client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.requestTimeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, c.requestTimeout)
}
