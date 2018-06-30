package proxy

import (
	"net"
	"log"
	"io"
	"fmt"
	"os"
	"github.com/VitZhou/redis-green/proxy/protocol"
	"redis-green/proxy/pool"
)

type Proxy struct {
	pool *pool.ConnPool
}

func NewSocketProxy() {
	listener, e := net.Listen("tcp", ":33333")
	CheckError(e)
	defer listener.Close()
	log.Println("tcp server started on port 20880 waitting for clients")
	opt := &pool.Options{
		Address: ":6379",
	}
	connPool, i := pool.NewConnPool(opt)
	if i != nil {
		log.Fatal("init fail", i)
	}
	proxy := &Proxy{pool: connPool}
	for {
		conn, i := listener.Accept()

		if i != nil {
			continue
		}
		reader := protocol.NewRESPReader(conn)

		go proxy.forward(reader, conn)
	}
}

func (p *Proxy) forward(reader *protocol.RESPReader, clientConn net.Conn) {
	targetConn, e := p.pool.Get()
	if e != nil {
		log.Fatal("remote targetConn failed:", e)
	}
	go func() {
		for {
			//buf := make([]byte, 2048)
			//clientConn.Read(buf)
			bytes, err := reader.ReadObject()
			if err != nil {
				if err == io.EOF {
					clientConn.Close()
					//客户端连接断开后,向targetConn写入一个字节,以便targetConn退出io.copy
					targetConn.NetConn.Write([]byte{1})
					return
				}
			}
			log.Print("##########", string(bytes))
			targetConn.NetConn.Write(bytes)
		}
	}()
	go io.Copy(clientConn, targetConn.NetConn)
}


func CheckError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
