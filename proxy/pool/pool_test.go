package pool

import "testing"
import (
	"github.com/stretchr/testify/assert"
	"net"
)

func TestNewConnPool(t *testing.T) {
	t.Run("成功创建连接池", func(t *testing.T) {
		opt := &Options{Address: ":8080", dialer: func() (net.Conn, error) {
			_, conn := net.Pipe()
			return conn, nil
		}}
		connPool, e := NewConnPool(opt)

		assert.NoError(t, e)
		assert.NotNil(t, connPool)

	})
}

func TestConnPool_NewConn(t *testing.T) {
	t.Run("创建连接成功", func(t *testing.T) {
		opt := &Options{Address: ":8080", dialer: func() (net.Conn, error) {
			_, conn := net.Pipe()
			return conn, nil
		}}
		connPool, _ := NewConnPool(opt)

		conn, e := connPool.NewConn()
		assert.NoError(t, e)
		assert.NotNil(t, conn)
		if connPool.connNum != 2 {
			t.Errorf("连接数不正确")
		}
	})

	t.Run("超过连接池最大限制,创建失败", func(t *testing.T) {
		opt := &Options{Address: ":8080", dialer: func() (net.Conn, error) {
			_, conn := net.Pipe()
			return conn, nil
		}, MaxPoolSize: 1}
		connPool, _ := NewConnPool(opt)

		conn, e := connPool.NewConn()
		assert.Equal(t, poolFull, e)
		assert.Nil(t, conn)
	})
}

func TestConnPool_RemoveConn(t *testing.T) {
	t.Run("成功移除", func(t *testing.T) {
		opt := &Options{Address: ":8080", dialer: func() (net.Conn, error) {
			_, conn := net.Pipe()
			return conn, nil
		}, MaxPoolSize: 1}
		connPool, _ := NewConnPool(opt)
		conn, _ := connPool.Get()
		connPool.removeConn(conn)
	})
}

func TestConnPool_Get(t *testing.T) {
	t.Run("有连接,成功获取", func(t *testing.T) {
		opt := &Options{Address: ":8080", dialer: func() (net.Conn, error) {
			_, conn := net.Pipe()
			return conn, nil
		}, MaxPoolSize: 1}
		connPool, _ := NewConnPool(opt)
		conn, _ := connPool.Get()
		assert.NotNil(t, conn)
	})

	t.Run("无连接,获取失败", func(t *testing.T) {
		opt := &Options{Address: ":8080", dialer: func() (net.Conn, error) {
			_, conn := net.Pipe()
			return conn, nil
		}, MaxPoolSize: 1}
		connPool, _ := NewConnPool(opt)

		_, _ = connPool.Get()
		_, e := connPool.Get()
		assert.Equal(t, e, NoIdle)
	})

	t.Run("连接池已关闭,无法获取", func(t *testing.T) {
		opt := &Options{Address: ":8080", dialer: func() (net.Conn, error) {
			_, conn := net.Pipe()
			return conn, nil
		}, MaxPoolSize: 1}
		connPool, _ := NewConnPool(opt)

		connPool.Close()
		_, e := connPool.Get()
		assert.Equal(t, e, AlreadyClosed)
	})
}

func TestConnPool_Close(t *testing.T) {
	t.Run("正确关闭连接池", func(t *testing.T) {
		opt := &Options{Address: ":8080", dialer: func() (net.Conn, error) {
			_, conn := net.Pipe()
			return conn, nil
		}, MaxPoolSize: 1}
		connPool, _ := NewConnPool(opt)

		e := connPool.Close()
		assert.Nil(t, e)

		if connPool.connNum != 0 {
			t.Errorf("连接数没有归0")
		}
	})

	t.Run("重复关闭", func(t *testing.T) {
		opt := &Options{Address: ":8080", dialer: func() (net.Conn, error) {
			_, conn := net.Pipe()
			return conn, nil
		}, MaxPoolSize: 1}
		connPool, _ := NewConnPool(opt)

		_ = connPool.Close()
		e := connPool.Close()
		assert.Error(t, e, AlreadyClosed)
	})
}

func TestConnPool_CloseConn(t *testing.T) {
	t.Run("正确关闭连接", func(t *testing.T) {
		opt := &Options{Address: ":8080", dialer: func() (net.Conn, error) {
			_, conn := net.Pipe()
			return conn, nil
		}, MaxPoolSize: 1}
		connPool, _ := NewConnPool(opt)

		conn, _ := connPool.Get()
		e := connPool.CloseConn(conn)
		assert.Nil(t, e)

		if connPool.connNum != 0 {
			t.Errorf("连接数没有归0")
		}
	})

	t.Run("连接池已关闭,无法单独关闭连接", func(t *testing.T) {
		opt := &Options{Address: ":8080", dialer: func() (net.Conn, error) {
			_, conn := net.Pipe()
			return conn, nil
		}, MaxPoolSize: 1}
		connPool, _ := NewConnPool(opt)

		conn, _ := connPool.Get()
		_ = connPool.Close()
		e := connPool.CloseConn(conn)
		assert.Error(t, e, AlreadyClosed)
	})
}
