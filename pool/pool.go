package pool

import (
	"time"
	"net"
)

type Options struct {
	Dialer  func() (net.Conn, error)
	OnClose func(*Conn) error

	PoolSize           int
	PoolTimeout        time.Duration
	IdleTimeout        time.Duration
	IdleCheckFrequency time.Duration
}

type Pool interface {
	NewConn() (*Conn, error)
	CloseConn(*Conn) error
	Put(conn *Conn)
	Get() (*Conn, error)
}

type ConnPool struct {
	Options *Options
}


