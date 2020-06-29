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
	log               Log
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

//WebStatus 是否开启HTTP访问服务器状态页(设置为空为关闭，默认关闭，例子:":8080" 访问http 8080端口)
func WebStatus(listen string) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		if listen != "" {
			o.webEnabled = true
			o.webListen = listen
		} else {
			o.webEnabled = false
			o.webListen = ""
		}
	})
}

//LogLevel 调试等级(debug, info, warn, error) 默认info
func LogLevel(level string) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.log.Level(level)
	})
}

//Logger 自定义日志记录
func Logger(logger Log) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.log = logger
	})
}

//BufferSize 数据读取的缓存区（初始化大小，第一次最大读取长度）默认4096字节
func BufferSize(s int, max ...int) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.bufferInitialSize = s
		if len(max) > 0 {
			o.bufferMax = max[0]
		}
	})
}

//Codec TCP数据拆包的数据解码器（默认LineCodec)
func Codec(cc codec.Codec) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.codec = cc
	})
}

//ClientCodec 发送给客户端封包的数据编码器 （如果不设置默认为和拆包Codec一致)
func ClientCodec(cc codec.Codec) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.clientCodec = cc
	})
}

//ConnMaxIdleTime 链接最大空闲时间(超过最大空闲时间,链接自动关闭,默认为0永不关闭)
func ConnMaxIdleTime(d time.Duration) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.connMaxIdleTime = d
	})
}

//RouterKeyExtract 数据解包对应的路由解析函数（可选项,和Route搭配使用)默认为空
func RouterKeyExtract(fn func([]byte) int) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.extractRouteKey = fn
	})
}
