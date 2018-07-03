package pool

import (
	"sync"
	"sync/atomic"
	"errors"
	"time"
	"log"
	"fmt"
)

var AlreadyClosed = errors.New("mesh: pool is closed")
var poolFull = errors.New("mesh: pool is full")
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

	connNum     int32 //atomic
	idleConnNum int32 //atomic
	closedFlag  int32 //atomic
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
	pool.initIdleCheckTicker(opt.IdleCheckFrequency)
	return pool, nil
}

func (pool *ConnPool) NewConn() (*Conn, error) {
	log.Printf("create conn")
	if atomic.LoadInt32(&pool.connNum) >= pool.opt.MaxPoolSize {
		return nil, poolFull
	}
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
	conn.UpdateLastUseTime(time.Now())
	pool.idleConns = append(pool.idleConns, conn)
	atomic.AddInt32(&pool.idleConnNum, 1)
	atomic.AddInt32(&pool.connNum, 1)
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

	var idle *Conn
	var err error
	if atomic.LoadInt32(&pool.idleConnNum) <= 0 {
		idle, err = pool.NewConn()
	} else {
		idle = pool.popIdle()
	}

	if err == nil {
		pool.runningConnsMu.Lock()
		pool.runningConns = append(pool.runningConns, idle)
		pool.runningConnsMu.Unlock()
	}
	return idle, err
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
	atomic.StoreInt32(&pool.idleConnNum, 0)
	return closeConnErr
}

func (pool *ConnPool) isClosed() bool {
	return atomic.LoadInt32(&pool.closedFlag) == 1
}

func (pool *ConnPool) popIdle() *Conn {
	pool.idleConnsMu.Lock()
	defer pool.idleConnsMu.Unlock()
	idx := len(pool.idleConns) - 1
	conn := pool.idleConns[idx]
	pool.idleConns = pool.idleConns[:idx]
	atomic.AddInt32(&pool.idleConnNum, -1)
	return conn
}

func (pool *ConnPool) removeConn(conn *Conn) {
	removed := false
	pool.idleConnsMu.Lock()
	for i, c := range pool.idleConns {
		if c == conn {
			pool.idleConns = append(pool.idleConns[:i], pool.idleConns[i+1:]...)
			removed = true
			atomic.AddInt32(&pool.connNum, -1)
			atomic.AddInt32(&pool.idleConnNum, -1)
			break
		}
	}
	pool.idleConnsMu.Unlock()

	if !removed {
		pool.removeIdleConn(conn)
	}
}

func (pool *ConnPool) initIdleCheckTicker(idleCheckFrequency time.Duration) {
	go func() {
		tick := time.NewTicker(idleCheckFrequency)
		defer tick.Stop()
		for range tick.C {
			if pool.isClosed() {
				return
			}
			pool.checkIdle()
		}
	}()
}

func (pool *ConnPool) checkIdle() {
	pool.idleConnsMu.Lock()
	if len(pool.idleConns) > 0 && pool.idleConns[0].IsExpired(pool.opt.IdleTimeout) {
		pool.idleConns = append(pool.idleConns[:0], pool.idleConns[1:]...)
		atomic.AddInt32(&pool.connNum, -1)
		atomic.AddInt32(&pool.idleConnNum, -1)
	}
	pool.idleConnsMu.Unlock()

	pool.runningConnsMu.Lock()
	for i, conn := range pool.runningConns {
		if len(pool.idleConns) > 0 &&  conn.IsExpired(pool.opt.IdleTimeout) {
			pool.runningConns = append(pool.runningConns[:i], pool.runningConns[i+1:]...)
			atomic.AddInt32(&pool.connNum, -1)
		}
	}
	pool.runningConnsMu.Unlock()
}

func (pool *ConnPool) Size() int32 {
	return atomic.LoadInt32(&pool.connNum)
}

func (pool *ConnPool) Idle(conn *Conn) {
	pool.runningConnsMu.Lock()
	for i, c := range pool.runningConns {
		if c == conn {
			pool.runningConns = append(pool.runningConns[:i], pool.runningConns[i+1:]...)
			break
		}
	}
	pool.runningConnsMu.Unlock()

	pool.idleConnsMu.Lock()
	pool.idleConns = append(pool.idleConns, conn)
	atomic.AddInt32(&pool.idleConnNum, 1)
	pool.idleConnsMu.Unlock()
	fmt.Println("num:", atomic.LoadInt32(&pool.connNum))
	fmt.Println("pool.idleConns:", atomic.LoadInt32(&pool.idleConnNum))
}

func (pool *ConnPool) removeIdleConn(conn *Conn) {
	pool.idleConnsMu.Lock()
	for i, c := range pool.idleConns {
		if c == conn {
			pool.idleConns = append(pool.idleConns[:i], pool.idleConns[i+1:]...)
			atomic.AddInt32(&pool.connNum, -1)
			atomic.AddInt32(&pool.idleConnNum, -1)
			break
		}
	}
	pool.idleConnsMu.Unlock()
}

func (pool *ConnPool) removeRunningConn(conn *Conn) {
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
