package pool

import (
	"sync"
	"sync/atomic"
	"errors"
)

var AlreadyClosed = errors.New("mesh: pool is closed")
var PoolFull = errors.New("mesh: pool is full")
var NoIdle = errors.New("mesh: no idle conn")

type Pool interface {
	NewConn() (*Conn, error)
	CloseConn(*Conn) error
	Close() error
	Get() (*Conn, error)
}

type ConnPool struct {
	opt         *Options
	idleConns   []*Conn
	idleConnsMu sync.Mutex

	runningConns   []*Conn
	runningConnsMu sync.Mutex

	connNum    int32 //atomic
	closedFlag int32 //atomic
}

func NewConnPool(opt *Options) (*ConnPool, error) {
	opt.init()
	pool := &ConnPool{
		opt:       opt,
		idleConns: make([]*Conn, 0, opt.MaxPoolSize),
	}
	for i := 0; i < pool.opt.InitialPoolSize; i++ {
		_, e := pool.NewConn()
		if e != nil {
			return pool, e
		}
	}
	return pool, nil
}

func (pool *ConnPool) NewConn() (*Conn, error) {
	if atomic.LoadInt32(&pool.connNum) >= pool.opt.MaxPoolSize {
		return nil, PoolFull
	}
	atomic.AddInt32(&pool.connNum, 1)

	if pool.isClosed() {
		return nil, AlreadyClosed
	}
	pool.idleConnsMu.Lock()
	defer pool.idleConnsMu.Unlock()
	dialer, e := pool.opt.dialer()
	if e != nil {
		return nil, e
	}
	conn := &Conn{NetConn: dialer}
	pool.idleConns = append(pool.idleConns, conn)
	return conn, nil
}

func (pool *ConnPool) CloseConn(conn *Conn) error {
	if pool.isClosed() {
		return AlreadyClosed
	}
	pool.removeConn(conn)
	return conn.NetConn.Close()
}

func (pool *ConnPool) Get() (*Conn, error) {
	if pool.isClosed() {
		return nil, AlreadyClosed
	}
	idle := pool.popIdle()
	if idle == nil {
		return nil, NoIdle
	}
	pool.runningConnsMu.Lock()
	defer pool.runningConnsMu.Unlock()
	pool.runningConns = append(pool.runningConns, idle)
	return idle, nil
}

func (pool *ConnPool) Close() error {
	if !atomic.CompareAndSwapInt32(&pool.closedFlag, 0, 1) {
		return AlreadyClosed
	}

	var closeConnErr error
	pool.runningConnsMu.Lock()
	for _, cn := range pool.runningConns {
		if err := cn.NetConn.Close(); err != nil && closeConnErr == nil {
			closeConnErr = err
		}
	}
	pool.runningConns = nil
	pool.runningConnsMu.Unlock()

	pool.idleConnsMu.Lock()
	for _, cn := range pool.idleConns {
		if err := cn.NetConn.Close(); err != nil && closeConnErr == nil {
			closeConnErr = err
		}
	}
	pool.idleConns = nil
	pool.idleConnsMu.Unlock()
	atomic.StoreInt32(&pool.connNum, 0)
	return closeConnErr
}

func (pool *ConnPool) isClosed() bool {
	return atomic.LoadInt32(&pool.closedFlag) == 1
}

func (pool *ConnPool) popIdle() *Conn {
	if len(pool.idleConns) == 0 {
		return nil
	}

	pool.idleConnsMu.Lock()
	defer pool.idleConnsMu.Unlock()
	idx := len(pool.idleConns) - 1
	cn := pool.idleConns[idx]
	pool.idleConns = pool.idleConns[:idx]
	return cn
}

func (pool *ConnPool) removeConn(conn *Conn) {
	removed := false
	pool.idleConnsMu.Lock()
	for i, c := range pool.idleConns {
		if c == conn {
			pool.idleConns = append(pool.idleConns[:i], pool.idleConns[i+1:]...)
			removed = true
			atomic.AddInt32(&pool.connNum, -1)
			break
		}
	}
	pool.idleConnsMu.Unlock()

	if !removed {
		pool.runningConnsMu.Lock()
		for i, c := range pool.runningConns {
			if c == conn {
				pool.runningConns = append(pool.runningConns[:i], pool.runningConns[i+1:]...)
				atomic.AddInt32(&pool.connNum, -1)
				break
			}
		}
		pool.runningConnsMu.Unlock()
	}
}
