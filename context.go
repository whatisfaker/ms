package ms

import "context"

type Context struct {
	context.Context
	srv                 *Server
	conn                *msConn
	payload             []byte
	hanlders            []func(*Context)
	currentHandlerIndex int
	handlersMax         int
}

func newContext(srv *Server, conn *msConn, payload []byte) *Context {
	return &Context{
		Context:             context.TODO(),
		srv:                 srv,
		payload:             payload,
		currentHandlerIndex: -1,
		handlersMax:         0,
		conn:                conn,
		hanlders:            make([]func(*Context), 0),
	}
}

func (c *Context) addChainHandlers(hls ...func(*Context)) {
	c.hanlders = append(c.hanlders, hls...)
	c.handlersMax = len(c.hanlders)
}

//Payload 数据包原始内容
func (c *Context) Payload() []byte {
	return c.payload
}

//RegisterDispatcher 注册分发标识
func (c *Context) RegisterDispatcher(identity int) {
	c.srv.registerDispatcher(identity, c.conn)
}

//Next 传递下一个处理器处理
func (c *Context) Next() {
	if c.currentHandlerIndex >= c.handlersMax-1 {
		return
	}
	c.currentHandlerIndex++
	c.hanlders[c.currentHandlerIndex](c)
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
func (c *Context) ReplyToGroup(b []byte, identities ...int) {
	c.srv.sendToDispatchers(b, identities...)
}
