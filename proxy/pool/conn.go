package pool

import "net"

type Conn struct {
	NetConn net.Conn
}
