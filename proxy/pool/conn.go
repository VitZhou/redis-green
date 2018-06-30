package pool

import "net"

type Conn struct {
	netConn net.Conn
}
