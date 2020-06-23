package ms

import (
	"time"

	"github.com/whatisfaker/ms/codec"
)

type serverOptions struct {
	codec             codec.Codec
	clientCodec       codec.Codec
	bufferInitialSize int
	bufferMax         int
	connMaxIdleTime   time.Duration
	extractRouteKey   func([]byte) int
	loglevel          string
	webEnabled        bool
	webListen         string
}

type ServerOption interface {
	apply(*serverOptions)
}

type funcServerOption struct {
	f func(*serverOptions)
}

func (fdo *funcServerOption) apply(do *serverOptions) {
	fdo.f(do)
}

func newFuncServerOption(f func(*serverOptions)) *funcServerOption {
	return &funcServerOption{
		f: f,
	}
}

//WebStatus 是否开启HTTP访问服务器状态页
func WebStatus(enabled bool, listen string) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.webEnabled = enabled
		o.webListen = listen
	})
}

//LogLevel 调试等级(debug, info, warn, error, fatal)
func LogLevel(level string) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.loglevel = level
	})
}

//BufferSize 数据读取的缓存区（初始化大小，最大长度 ）
func BufferSize(s int, max ...int) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.bufferInitialSize = s
		if len(max) > 0 {
			o.bufferMax = max[0]
		}
	})
}

//Codec TCP数据拆包的数据编码/解码器
func Codec(cc codec.Codec) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.codec = cc
	})
}

//ClientCodec 发送给客户端的数据编码/解码器 （如果不设置默认为和拆包Codec一致)
func ClientCodec(cc codec.Codec) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.clientCodec = cc
	})
}

//ConnMaxIdleTime 链接最大空闲时间
func ConnMaxIdleTime(d time.Duration) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.connMaxIdleTime = d
	})
}

//RouterKeyExtract 数据解包对应的路由解析函数
func RouterKeyExtract(fn func([]byte) int) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.extractRouteKey = fn
	})
}
