package client

import (
	"bufio"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/whatisfaker/ms/codec"
	"github.com/whatisfaker/zaptrace/log"
	"go.uber.org/zap"
)

const (
	bufferDefaultInitSize = 4096
	defaultTimeout        = 30 * time.Second
)

type Client struct {
	mu       sync.Mutex
	opts     *clientOptions
	conn     net.Conn
	addr     string
	handler  func([]byte)
	handlers map[int]func([]byte)
	log      *log.Factory
	closed   uint32
}

func NewClient(addr string, opts ...ClientOption) *Client {
	sOpts := &clientOptions{
		codec:             codec.NewLineCodec(),
		bufferInitialSize: bufferDefaultInitSize,
		bufferMax:         bufio.MaxScanTokenSize,
		connectionTimeout: defaultTimeout,
		loglevel:          "info",
	}
	for _, opt := range opts {
		opt.apply(sOpts)
	}
	//如果没有特殊指定,下发给客户端的数据用相同的编码/解码
	if sOpts.serverCodec == nil {
		sOpts.serverCodec = sOpts.codec
	}
	c := &Client{
		handlers: make(map[int]func([]byte)),
		handler:  func([]byte) {},
		log:      log.NewStdLogger(sOpts.loglevel),
		opts:     sOpts,
		addr:     addr,
	}
	c.close()
	return c
}

func (c *Client) close() {
	atomic.StoreUint32(&c.closed, 1)
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Client) start() {
	atomic.StoreUint32(&c.closed, 0)
}

func (c *Client) isClosed() bool {
	return atomic.LoadUint32(&c.closed) == 1
}

func (c *Client) recv() error {
	scanner := bufio.NewScanner(c.conn)
	scanner.Split(c.opts.codec.Split)
	var err error
	for {
		if ok := scanner.Scan(); !ok {
			err = scanner.Err()
			break
		} else {
			d := scanner.Bytes()
			if c.opts.extractRouteKey != nil {
				k := c.opts.extractRouteKey(d)
				if h, ok := c.handlers[k]; ok {
					h(d)
					continue
				}
			}
			if c.handler != nil {
				c.handler(d)
			}
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) connect() error {
	var err error
	c.start()
	c.conn, err = net.DialTimeout("tcp", c.addr, c.opts.connectionTimeout)
	if err != nil {
		//closed
		c.close()
		return err
	}
	//openned
	c.start()
	err = c.recv()
	if err != nil {
		c.close()
		return err
	}
	return nil
}

func (c *Client) HandleRoute(key int, handler func([]byte)) {
	c.mu.Lock()
	c.handlers[key] = handler
	c.mu.Unlock()
}

func (c *Client) Handle(handler func([]byte)) {
	c.handler = handler
}

func (c *Client) Start() error {
	err := c.connect()
	if err != nil {
		c.log.Normal().Error("connect error", zap.Error(err))
	}
	if c.opts.autoReconnect {
		c.log.Normal().Info("try reconnect 1")
		timer := time.NewTimer(5 * time.Second)
		<-timer.C
		err = c.connect()
		if err != nil {
			c.log.Normal().Error("connect error", zap.Error(err))
		}
		c.log.Normal().Info("try reconnect 2")
		timer = time.NewTimer(10 * time.Second)
		<-timer.C
		err = c.connect()
		if err != nil {
			c.log.Normal().Error("connect error", zap.Error(err))
		}
	}
	c.log.Normal().Info("disconnect")
	return err
}

func (c *Client) Send(b []byte) error {
	dt, err := c.opts.serverCodec.Encode(b)
	if err != nil {
		return err
	}
	return c.rawSend(dt)
}

func (c *Client) rawSend(b []byte) error {
	//已经关闭
	if c.isClosed() {
		return nil
	}
	if c.conn != nil {
		_, err := c.conn.Write(b)
		if err != nil {
			if !c.isClosed() {
				c.close()
			}
			return err
		}
	}
	return nil
}
