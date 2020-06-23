package client

import (
	"time"

	"github.com/whatisfaker/ms/codec"
)

type clientOptions struct {
	codec             codec.Codec
	serverCodec       codec.Codec
	bufferInitialSize int
	bufferMax         int
	connectionTimeout time.Duration
	extractRouteKey   func([]byte) int
	loglevel          string
	autoReconnect     bool
}

type ClientOption interface {
	apply(*clientOptions)
}

type funcClientOption struct {
	f func(*clientOptions)
}

func (fdo *funcClientOption) apply(do *clientOptions) {
	fdo.f(do)
}

func newFuncClientOption(f func(*clientOptions)) *funcClientOption {
	return &funcClientOption{
		f: f,
	}
}

func BufferSize(s int, max ...int) ClientOption {
	return newFuncClientOption(func(o *clientOptions) {
		o.bufferInitialSize = s
		if len(max) > 0 {
			o.bufferMax = max[0]
		}
	})
}

func AutoReconnect(rec bool) ClientOption {
	return newFuncClientOption(func(o *clientOptions) {
		o.autoReconnect = rec
	})
}

func LogLevel(level string) ClientOption {
	return newFuncClientOption(func(o *clientOptions) {
		o.loglevel = level
	})
}

func Codec(cc codec.Codec) ClientOption {
	return newFuncClientOption(func(o *clientOptions) {
		o.codec = cc
	})
}

func ServerCodec(cc codec.Codec) ClientOption {
	return newFuncClientOption(func(o *clientOptions) {
		o.serverCodec = cc
	})
}

func ConnectTimeout(d time.Duration) ClientOption {
	return newFuncClientOption(func(o *clientOptions) {
		o.connectionTimeout = d
	})
}

func RouterKeyExtract(fn func([]byte) int) ClientOption {
	return newFuncClientOption(func(o *clientOptions) {
		o.extractRouteKey = fn
	})
}
