package proxy

import (
	"net"
	"log"
	"io"
	"github.com/VitZhou/redis-green/proxy/protocol"
	"github.com/VitZhou/redis-green/proxy/pool"
	"time"
	"context"
)

type Proxy struct {
	pool *pool.ConnPool
}

func NewSocketProxy() {
	listener, e := net.Listen("tcp", ":33333")
	if e != nil {
		log.Fatal("can't listen address,", e)
	}
	defer listener.Close()
	log.Println("tcp server started on port 20880 waitting for clients")
	opt := &pool.Options{
		Address:            ":6379",
		IdleCheckFrequency: time.Second,
		IdleTimeout:        50 * time.Second,
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
		go proxy.forward(conn)
	}
}

func (p *Proxy) forward(srcConn net.Conn) {
	ctx := context.Background()
	reader := protocol.NewRESPReader(srcConn)
	dstConn, e := p.pool.Get()
	if e != nil {
		log.Fatal("get conn fail:", e)
	}
	go p.resolve(ctx, reader, dstConn, srcConn)
	go p.copy(ctx, srcConn, dstConn)
}

func (p *Proxy) resolve(ctx context.Context, reader *protocol.RESPReader, dstConn *pool.Conn, srcConn net.Conn) {
	for {
		bytes, err := reader.ReadObject()
		conn := dstConn.NetConn
		if err != nil {
			if err == io.EOF {
				context.WithCancel(ctx)
				srcConn.Close()
				//客户端连接断开后,向targetConn写入一个字节,以便targetConn退出io.copy
				conn.Write([]byte("close\r\n"))
				return
			}
		}
		conn.Write(bytes)
	}
}

type readerFunc func(p []byte) (n int, err error)

func (rf readerFunc) Read(p []byte) (n int, err error) { return rf(p) }

func (p *Proxy) copy(ctx context.Context, dst io.Writer, src *pool.Conn) error {
	_, err := io.Copy(dst, readerFunc(func(p []byte) (int, error) {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
			return src.NetConn.Read(p)
		}
	}))
	p.pool.Idle(src)
	return err
}

