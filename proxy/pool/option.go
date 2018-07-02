package pool

import (
	"net"
	"time"
	"crypto/tls"
)

type Option interface {
}

type Options struct {
	dialer  func() (net.Conn, error)
	OnClose func(*Conn) error

	Address            string
	Pwd                string
	DbIndex            int
	InitialPoolSize    int
	MaxPoolSize        int32
	PoolTimeout        time.Duration
	DialTimeout        time.Duration
	IdleTimeout        time.Duration
	IdleCheckFrequency time.Duration
	ReadDeadline       time.Duration
	WriteDeadline      time.Duration

	TlsConfig *tls.Config
}

func (opt *Options) init() {
	if opt.DialTimeout == 0 {
		opt.DialTimeout = 10 * time.Second
	}

	if opt.dialer == nil {
		opt.dialer = func() (net.Conn, error) {
			conn, e := net.DialTimeout("tcp", opt.Address, opt.DialTimeout)
			if e != nil {
				return nil, e
			}
			if opt.TlsConfig != nil {
				t := tls.Client(conn, opt.TlsConfig)
				return t, t.Handshake()
			}
			return conn, nil
		}
	}

	if opt.InitialPoolSize == 0 {
		opt.InitialPoolSize = 1
	}
	if opt.MaxPoolSize == 0 {
		opt.MaxPoolSize = 10
	}
	if opt.IdleTimeout == 0 {
		opt.IdleTimeout = 5 * time.Minute
	}
	if opt.IdleCheckFrequency == 0 {
		opt.IdleCheckFrequency = 3 * time.Second
	}
	if opt.WriteDeadline == -1 {
		opt.WriteDeadline = 0
	}
	if opt.WriteDeadline == 0 {
		opt.WriteDeadline = 3 * time.Minute
	}

	if opt.ReadDeadline == -1 {
		opt.ReadDeadline = 0
	}
	if opt.ReadDeadline == 0 {
		opt.ReadDeadline = 10 * time.Second
	}
	if opt.PoolTimeout == 0 {
		opt.PoolTimeout = opt.ReadDeadline + time.Second
	}
}
