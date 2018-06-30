package proxy

import (
	"net"
	"log"
	"io"
	"fmt"
	"os"
	"github.com/VitZhou/redis-green/proxy/protocol"
)

type Proxy struct {
	targetConn net.Conn
}

func NewSocketProxy() {
	listener, e := net.Listen("tcp", ":33333")
	CheckError(e)
	defer listener.Close()
	log.Println("tcp server started on port 20880 waitting for clients")
	for {
		conn, i := listener.Accept()

		if i != nil {
			continue
		}
		reader := protocol.NewRESPReader(conn)
		proxy := &Proxy{}
		go proxy.forward(reader, conn)
	}
}

func (p *Proxy) forward(reader *protocol.RESPReader, clientConn net.Conn) {
	targetConn, e := p.getTargetConn()
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
					targetConn.Write([]byte{1})
					return
				}
			}
			log.Print("##########", string(bytes))
			targetConn.Write(bytes)
		}
	}()
	go io.Copy(clientConn, targetConn)
}

func (p *Proxy) getTargetConn() (net.Conn, error) {
	if p.targetConn == nil {
		dial, e := net.Dial("tcp", ":6379")
		if dial == nil || e != nil {
			return nil, e
		}
		p.targetConn = dial
	}
	return p.targetConn, nil
}

func CheckError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
