package ms

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/whatisfaker/ms/codec"
	"golang.org/x/sync/errgroup"
)

const (
	bufferDefaultInitSize  = 4096
	defaultConnMaxIdleTime = 0 * time.Second
)

type Server struct {
	opts               *serverOptions
	log                Log
	mu                 sync.Mutex
	msConns            map[*msConn]bool
	dispatchMap        map[string][]*msConn
	dispatchReverseMap map[*msConn]string
	routeMap           map[int][]func(*Context)
	globalMW           []func(*Context)
	inShutdown         int32
	statusServer       *http.Server
}

func NewServer(opts ...ServerOption) *Server {
	sOpts := &serverOptions{
		codec:             codec.NewLineCodec(),
		bufferInitialSize: bufferDefaultInitSize,
		bufferMax:         bufio.MaxScanTokenSize,
		connMaxIdleTime:   defaultConnMaxIdleTime,
		log:               NewDefaultLogger("info"),
	}
	for _, opt := range opts {
		opt.apply(sOpts)
	}
	//如果没有特殊指定,下发给客户端的数据用相同的编码/解码
	if sOpts.clientCodec == nil {
		sOpts.clientCodec = sOpts.codec
	}
	return &Server{
		msConns:            make(map[*msConn]bool),
		opts:               sOpts,
		log:                sOpts.log,
		dispatchMap:        make(map[string][]*msConn),
		dispatchReverseMap: make(map[*msConn]string),
		routeMap:           make(map[int][]func(*Context)),
	}
}

//Shutdown 关闭服务
func (c *Server) Shutdown(ctx context.Context) error {
	//TODO
	atomic.StoreInt32(&c.inShutdown, 1)
	defer atomic.StoreInt32(&c.inShutdown, 0)
	for conn := range c.msConns {
		c.removeConn(conn)
	}
	if c.statusServer != nil {
		return c.statusServer.Shutdown(ctx)
	}
	return nil
}

func (c *Server) shuttingDown() bool {
	return atomic.LoadInt32(&c.inShutdown) != 0
}

//Serve 开始服务(block func)
func (c *Server) Serve(ctx context.Context, lis net.Listener) error {
	grp, ctx := errgroup.WithContext(ctx)
	if c.opts.webEnabled {
		sh := NewStatusContainer(c)
		mux := http.NewServeMux()
		mux.HandleFunc("/", sh.StatusHandler)
		c.statusServer = &http.Server{
			Addr:    c.opts.webListen,
			Handler: mux,
		}
		grp.Go(func() error {
			quit := make(chan error)
			defer close(quit)
			go func() {
				c.log.Info(fmt.Sprintf("status web server start %s", c.opts.webListen))
				// service connections
				if err := c.statusServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					quit <- err
					return
				}
				quit <- nil
			}()
			select {
			case err := <-quit:
				if err != nil {
					c.log.Error("start status web server error", err)
					return err
				}
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}
	grp.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			conn, err := lis.Accept()
			if err != nil {
				return err
			}
			go c.handleConn(conn)
		}
	})
	return grp.Wait()
}

func (c *Server) broadCast(b []byte) {
	for conn := range c.msConns {
		//同时下发
		go conn.write(b)
	}
}

func (c *Server) sendToDispatchers(b []byte, dispatchers ...string) {
	//读取dispatcherMap加锁
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, dispatch := range dispatchers {
		if conns, ok := c.dispatchMap[dispatch]; ok {
			for _, conn := range conns {
				go conn.write(b)
			}
		}
	}
}

func (c *Server) registerDispatcher(dispatcherID string, conn *msConn) {
	c.mu.Lock()
	defer c.mu.Unlock()
	//清理旧conn,例如用户切换
	if oldDispatcherID, ok := c.dispatchReverseMap[conn]; ok {
		if oldDispatcherID == dispatcherID {
			return
		}
		delete(c.dispatchReverseMap, conn)
		c.removeDispatcherConn(oldDispatcherID, conn)
	}
	if v, ok := c.dispatchMap[dispatcherID]; ok {
		v := append(v, conn)
		c.dispatchMap[dispatcherID] = v
	} else {
		c.dispatchMap[dispatcherID] = []*msConn{conn}
	}
	c.dispatchReverseMap[conn] = dispatcherID
}

func (c *Server) addConn(conn *msConn) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.msConns == nil {
		conn.Close()
		return false
	}
	c.msConns[conn] = true
	return true
}

func (c *Server) removeConn(conn *msConn) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.msConns != nil {
		delete(c.msConns, conn)
	}
	conn.Close()
	if dispatcherID, ok := c.dispatchReverseMap[conn]; ok {
		delete(c.dispatchReverseMap, conn)
		c.removeDispatcherConn(dispatcherID, conn)
	}
}

func (c *Server) removeDispatcherConn(dispatcherID string, conn *msConn) {
	//在锁内
	if conns, ok := c.dispatchMap[dispatcherID]; ok {
		l := len(conns)
		newConns := make([]*msConn, 0, l)
		for _, v := range conns {
			if v != conn {
				newConns = append(newConns, v)
			}
		}
		l2 := len(conns)
		if l2 == 0 {
			delete(c.dispatchMap, dispatcherID)
		} else {
			c.dispatchMap[dispatcherID] = newConns
		}
	}
}

//Use 设置全局处理器
func (c *Server) Use(fn ...func(*Context)) {
	c.globalMW = append(c.globalMW, fn...)
}

//Route 设置指定路由的处理器(key通过 ExtractRouterKey(ServerOption)获取)
func (c *Server) Route(key int, fn ...func(*Context)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.routeMap[key] = fn
}

func (c *Server) handleConn(conn net.Conn) {
	//正在关闭,拒绝链接
	if c.shuttingDown() {
		conn.Close()
		return
	}
	scanner := bufio.NewScanner(conn)
	scanner.Split(c.opts.codec.Split)
	buf := make([]byte, c.opts.bufferInitialSize)
	scanner.Buffer(buf, c.opts.bufferMax)
	msConn := newConn(c, conn, c.opts.clientCodec, c.log)
	if !c.addConn(msConn) {
		c.log.Warn("add msConn false")
		return
	}
	messages := make(chan []byte)
	//处理数据和超时的协程
	go c.handleData(msConn, messages)

	//重试次数
	var try int
	for {
		if ok := scanner.Scan(); !ok {
			if err := scanner.Err(); err != nil {
				//忽略因为关闭链接导致的错误
				if msConn.IsClosed() {
					break
				}
				if netErr, ok := err.(net.Error); ok {
					if !netErr.Temporary() || try > 3 {
						c.log.Error("scanner scan error", err)
					} else {
						try++
						c.log.Warn("scanner scan error temporary", "retry", try)
						continue
					}
				} else {
					c.log.Error("scanner scan error", err)
				}
			}
			//解析失败，或者关闭链接都是直接移除conn
			c.removeConn(msConn)
			break
		}
		messages <- scanner.Bytes()
	}
}

func (c *Server) handleData(msConn *msConn, msg <-chan []byte) {
	if c.opts.connMaxIdleTime > 0 {
		timer := time.NewTimer(c.opts.connMaxIdleTime)
		defer timer.Stop()
	Loop:
		for {
			select {
			case <-timer.C:
				//超时
				if !msConn.IsClosed() {
					c.log.Debug("timeout")
					c.removeConn(msConn)
				}
				break Loop
			case data, ok := <-msg:
				if !ok {
					break Loop
				}
				//有消息的时候,重置计时器
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(c.opts.connMaxIdleTime)
				ctx := newContext(c, msConn, data)
				if len(c.globalMW) > 0 {
					ctx.addChainHandlers(c.globalMW...)
				}
				if c.opts.extractRouteKey != nil {
					routerID := c.opts.extractRouteKey(data)
					if rhls, ok := c.routeMap[routerID]; ok {
						ctx.addChainHandlers(rhls...)
					}
				}
				ctx.Next()
			}
		}
	} else {
		//无超时长连接
		for data := range msg {
			ctx := newContext(c, msConn, data)
			if len(c.globalMW) > 0 {
				ctx.addChainHandlers(c.globalMW...)
			}
			if c.opts.extractRouteKey != nil {
				routerID := c.opts.extractRouteKey(data)
				if rhls, ok := c.routeMap[routerID]; ok {
					ctx.addChainHandlers(rhls...)
				}
			}
			ctx.Next()
		}
	}
}

//Status 获取当前服务器状态
func (c *Server) Status() interface{} {
	v := make(map[string]struct {
		Ld string
	})

	for k := range c.msConns {
		v[k.conn.RemoteAddr().String()] = struct {
			Ld string
		}{
			Ld: k.conn.LocalAddr().String(),
		}
	}
	return v
}

//BroadCast 广播消息给所有连接
func (c *Server) BroadCast(b []byte) {
	c.broadCast(b)
}

//ReplyToGroup 发送消息给指定标识的连接（通过RegisterDispatcher注册的标识）
func (c *Server) ReplyToGroup(b []byte, identities ...string) {
	c.sendToDispatchers(b, identities...)
}

//IsOnline 返回还在线的identidies
func (c *Server) IsOnline(identities ...string) []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	ret := make([]string, 0, len(identities))
	for _, dispatch := range identities {
		if _, ok := c.dispatchMap[dispatch]; ok {
			ret = append(ret, dispatch)
		}
	}
	return ret
}

//Onlines 返回在线人数
func (c *Server) Onlines() int {
	return len(c.dispatchMap)
}
