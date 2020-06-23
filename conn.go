package ms

import (
	"net"
	"sync"

	"github.com/whatisfaker/ms/codec"
	"github.com/whatisfaker/zaptrace/log"
	"go.uber.org/zap"
)

type msConn struct {
	mu     sync.Mutex
	conn   net.Conn
	codec  codec.Codec
	send   chan []byte
	log    *log.Factory
	closed bool
}

func newConn(srv *Server, conn net.Conn, codec codec.Codec, log *log.Factory) *msConn {
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
	return c.closed
}

func (c *msConn) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.log.Normal().Debug("close conn")
	c.closed = true
	close(c.send)
	c.conn.Close()
}

func (c *msConn) write(b []byte) {
	if c.closed {
		//已关闭的链接无法写入
		return
	}
	dt, err := c.codec.Encode(b)
	if err != nil {
		c.log.Normal().Error("msConn data encode error", zap.Error(err))
		return
	}
	c.send <- dt
}

func (c *msConn) writeProc() {
	for {
		if b, ok := <-c.send; ok {
			_, err := c.conn.Write(b)
			if err != nil {
				c.log.Normal().Error("send error", zap.Error(err))
				break
			}
		} else {
			break
		}
	}
}
