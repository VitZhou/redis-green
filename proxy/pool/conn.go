package pool

import (
	"net"
	"sync/atomic"
	"time"
)

var noTimeout = time.Time{}

type Conn struct {
	NetConn     net.Conn
	lastUseTime atomic.Value
}

func (cn *Conn) SetWriteTimeout(timeout time.Duration) {
	if timeout <= 0 {
		cn.NetConn.SetWriteDeadline(noTimeout)
	}
	cn.NetConn.SetWriteDeadline(time.Now().Add(timeout))
}

func (cn *Conn) SetReadTimeout(timeout time.Duration) {
	if timeout <= 0 {
		cn.NetConn.SetReadDeadline(noTimeout)
	}
	cn.NetConn.SetReadDeadline(time.Now().Add(timeout))
}

func (cn *Conn) UpdateLastUseTime() {
	cn.lastUseTime.Store(time.Now())
}

func (cn *Conn) IsExpired(timeout time.Duration) bool {
	return timeout > 0 && time.Since(cn.LastUseTime()) > timeout
}

func (cn *Conn) LastUseTime() time.Time {
	return cn.lastUseTime.Load().(time.Time)
}
