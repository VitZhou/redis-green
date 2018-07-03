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
	cn.UpdateLastUseTime(time.Now())
	if timeout <= 0 {
		cn.NetConn.SetWriteDeadline(noTimeout)
	}
	cn.NetConn.SetWriteDeadline(time.Now().Add(timeout))
}

func (cn *Conn) SetReadTimeout(timeout time.Duration) {
	cn.UpdateLastUseTime(time.Now())
	if timeout <= 0 {
		cn.NetConn.SetReadDeadline(noTimeout)
	}
	cn.NetConn.SetReadDeadline(time.Now().Add(timeout))
}

func (cn *Conn) UpdateLastUseTime(time time.Time) {
	cn.lastUseTime.Store(time)
}

func (cn *Conn) IsExpired(timeout time.Duration) bool {
	return timeout > 0 && time.Since(cn.LastUseTime()) > timeout
}

func (cn *Conn) LastUseTime() time.Time {
	return cn.lastUseTime.Load().(time.Time)
}
