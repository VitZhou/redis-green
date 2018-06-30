package protocol

import (
	"testing"
	"net"
	bytes2 "bytes"
)

func TestReadLine(t *testing.T) {
	t.Run("正确读取", func(t *testing.T) {
		server, client := net.Pipe()
		defer client.Close()
		defer server.Close()

		go func() {
			reader := NewRESPReader(server)
			result, e := reader.readLine()
			if e != nil {
				t.Error("fail")
			}
			expect := []byte("*3\r\n")
			if bytes2.Compare(result, expect) != 0 {
				t.Errorf("excpect=" + string(expect) + ",but result=" + string(result))
			}
		}()
		client.Write([]byte("*3\r\n$3\r\nSET\r\n$5\r\nmykey\r\n$8\r\nmy value\r\n"))
	})

	t.Run("没有正确结尾符", func(t *testing.T) {
		server, client := net.Pipe()
		defer client.Close()
		defer server.Close()

		go func() {
			reader := NewRESPReader(server)
			_, e := reader.readLine()
			if e == nil {
				t.Error("fail", e)
			}
		}()
		appen := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		client.Write(appen)
	})

	t.Run("空数据", func(t *testing.T) {
		server, client := net.Pipe()
		defer client.Close()
		defer server.Close()

		go func() {
			reader := NewRESPReader(server)
			_, e := reader.readLine()
			if e == nil {
				t.Error("fail", e)
			}
		}()
		appen := []byte("")
		client.Write(appen)
	})

	t.Run("nil", func(t *testing.T) {
		server, client := net.Pipe()
		defer client.Close()
		defer server.Close()

		go func() {
			reader := NewRESPReader(server)
			_, e := reader.readLine()
			if e == nil {
				t.Error("fail", e)
			}
		}()
		appen := []byte{SIMPLE_STRING}
		client.Write(append(appen, []byte("-1")...))
	})

	t.Run("格式错误", func(t *testing.T) {
		server, client := net.Pipe()
		defer client.Close()
		defer server.Close()

		go func() {
			reader := NewRESPReader(server)
			_, e := reader.readLine()
			if e == nil {
				t.Error("fail", e)
			}
		}()
		appen := []byte("$2\n")
		client.Write(appen)
	})
}

func TestGetCount(t *testing.T) {
	t.Run("正确获取", func(t *testing.T) {
		server, client := net.Pipe()
		defer client.Close()
		defer server.Close()

		reader := NewRESPReader(server)
		count, e := reader.getCount([]byte("$3\r\nget\r\n44\r\n"))
		if e != nil {
			t.Error("fail", e)
		}
		if count != 3{
			t.Error("获取次数失败")
		}
	})

	t.Run("协议错误,读取失败", func(t *testing.T) {
		server, client := net.Pipe()
		defer client.Close()
		defer server.Close()

		reader := NewRESPReader(server)
		_, e := reader.getCount([]byte("r"))
		if e == nil {
			t.Error("fail", e)
		}
	})
}

