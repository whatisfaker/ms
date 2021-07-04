package ms

import (
	"net"
	"sync"
	"sync/atomic"

	"github.com/whatisfaker/ms/codec"
)

type msConn struct {
	mu     sync.Mutex
	conn   net.Conn
	codec  codec.Codec
	send   chan []byte
	log    Log
	closed int32
}

func newConn(srv *Server, conn net.Conn, codec codec.Codec, log Log) *msConn {
	con := &msConn{
		conn:  conn,
		codec: codec,
		send:  make(chan []byte, 5),
		log:   log,
	}
	go con.writeProc()
	return con
}

func (c *msConn) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) == 1
}

func (c *msConn) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.IsClosed() {
		return
	}
	c.log.Debug("close conn")
	atomic.AddInt32(&c.closed, 1)
	close(c.send)
}

func (c *msConn) write(b []byte) {
	if c.IsClosed() {
		//已关闭的链接无法写入
		return
	}
	dt, err := c.codec.Encode(b)
	if err != nil {
		c.log.Error("msConn data encode error", err)
		return
	}
	c.send <- dt
}

func (c *msConn) writeProc() {
	for {
		if b, ok := <-c.send; ok {
			_, err := c.conn.Write(b)
			if err != nil {
				c.log.Error("send error", err)
				break
			}
		} else {
			c.conn.Close()
			break
		}
	}
}
