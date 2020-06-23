package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/whatisfaker/ms"
	"github.com/whatisfaker/ms/codec"
	"github.com/whatisfaker/zaptrace/log"
	"go.uber.org/zap"
)

func main() {
	lis, err := net.Listen("tcp", ":3033")
	if err != nil {
		fmt.Println(err)
	}
	log := log.NewStdLogger("debug")
	codec := codec.NewLengthFieldCodec(binary.BigEndian, 1, 2, 0, 0)
	//codec := codec.NewLineCodec()
	srv := ms.NewServer(
		ms.Codec(codec),
		ms.BufferSize(1024),
		ms.LogLevel("debug"),
		ms.ConnMaxIdleTime(10*time.Second),
		ms.RouterKeyExtract(func(b []byte) int {
			return int(b[0])
		}),
	)
	log.Normal().Info("server start")
	srv.Use(func(ctx *ms.Context) {
		//fmt.Println("mw:")
		ctx.Next()
	})
	srv.Route(1, func(ctx *ms.Context) {
		log.Normal().Info("route 1", zap.String("msg", string(ctx.Payload()[7:])))
		identity := binary.BigEndian.Uint32(ctx.Payload()[3:7])
		ctx.RegisterDispatcher(int(identity))
		data := append([]byte{2}, []byte("reg reply")...)
		ctx.Reply(data)
	})
	srv.Route(2, func(ctx *ms.Context) {
		log.Normal().Info("route 2", zap.String("msg", string(ctx.Payload()[7:])))
		identity := binary.BigEndian.Uint32(ctx.Payload()[3:7])
		if identity/10 == 2 {
			ctx.ReplyToGroup(append([]byte{2}, []byte("20 to group 10")...), 10)
		}
		data := append([]byte{2}, []byte("broadcast")...)
		ctx.BroadCast(data)
	})
	err = srv.Serve(lis)
	if err != nil {
		log.Normal().Error("serve err", zap.Error(err))
	}
}
