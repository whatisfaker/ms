package ms

import (
	"context"
	"fmt"
)

type Context struct {
	context.Context
	srv                 *Server
	conn                *msConn
	payload             []byte
	handlers            []func(*Context)
	currentHandlerIndex int
	handlersMax         int
	values              map[string]interface{}
}

func newContext(srv *Server, conn *msConn, payload []byte) *Context {
	return &Context{
		Context:             context.TODO(),
		srv:                 srv,
		payload:             payload,
		currentHandlerIndex: -1,
		handlersMax:         0,
		conn:                conn,
		handlers:            make([]func(*Context), 0),
		values:              map[string]interface{}{},
	}
}

func (c *Context) addChainHandlers(hls ...func(*Context)) {
	c.handlers = append(c.handlers, hls...)
	c.handlersMax = len(c.handlers)
}

//Payload 数据包原始内容
func (c *Context) Payload() []byte {
	return c.payload
}

func (c *Context) Set(k string, v interface{}) {
	c.values[k] = v
}

func (c *Context) Get(k string) (interface{}, bool) {
	if v, ok := c.values[k]; ok {
		return v, ok
	}
	return nil, false
}

func (c *Context) MustGet(k string) interface{} {
	if v, ok := c.values[k]; ok {
		return v
	} else {
		panic(fmt.Sprintf("no key(%s) found in context", k))
	}
}

//RegisterDispatcher 注册分发标识
func (c *Context) RegisterDispatcher(identity string) {
	c.srv.registerDispatcher(identity, c.conn)
}

//Next 传递下一个处理器处理
func (c *Context) Next() {
	if c.currentHandlerIndex >= c.handlersMax-1 {
		return
	}
	c.currentHandlerIndex++
	c.handlers[c.currentHandlerIndex](c)
}

//Reply 回复给当前连接发送方
func (c *Context) Reply(b []byte) {
	c.conn.write(b)
}

//BroadCast 广播消息给所有连接
func (c *Context) BroadCast(b []byte) {
	c.srv.broadCast(b)
}

//ReplyToGroup 发送消息给指定标识的连接（通过RegisterDispatcher注册的标识）
func (c *Context) ReplyToGroup(b []byte, identities ...string) {
	c.srv.sendToDispatchers(b, identities...)
}
