package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/whatisfaker/ms"
	"github.com/whatisfaker/ms/codec"
)

func main() {
	// lis, err := net.Listen("tcp", ":3033")
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// codec := codec.NewLengthFieldCodec(binary.BigEndian, 1, 2, 0, 0)
	// //codec := codec.NewLineCodec()
	// srv := ms.NewServer(
	// 	ms.Codec(codec),
	// 	ms.BufferSize(1024),
	// 	ms.LogLevel("debug"),
	// 	ms.WebStatus(":1234"),
	// 	//ms.ConnMaxIdleTime(10*time.Second),
	// 	ms.RouterKeyExtract(func(b []byte) int {
	// 		return int(b[0])
	// 	}),
	// )
	tcpLis, err := net.Listen("tcp", ":4033")
	if err != nil {
		fmt.Println(err)
	}
	codec := codec.NewLengthFieldCodec(binary.BigEndian, 0, 8, 0, 0)
	//codec := codec.NewLineCodec()
	tcpSrv := ms.NewServer(
		ms.Codec(codec),
		ms.BufferSize(1024),
		ms.LogLevel("debug"),
		//ms.ConnMaxIdleTime(10*time.Second),
		ms.RouterKeyExtract(func(b []byte) int {
			a := binary.BigEndian.Uint16(b[10:12])
			return int(a)
		}),
	)
	tcpSrv.Use(func(ctx *ms.Context) {
		fmt.Println("recv:", ctx.Payload())
		if len(ctx.Payload()) >= 12 {
			a := binary.BigEndian.Uint16(ctx.Payload()[10:12])
			if a == 101 {
				ctx.Next()
				return
			}
		}
		b := make([]byte, 2+2+36+1)
		copy(b[0:2], []byte("YM"))
		binary.BigEndian.PutUint16(b[2:4], 201)
		for i := 4; i < 40; i++ {
			b[i] = 255
		}
		b[40] = 1
		b, _ = codec.Encode(b)
		fmt.Println("reply data", b)
		ctx.Reply(b)
	})
	tcpSrv.Route(101, func(ctx *ms.Context) {
		b := make([]byte, 2+2+36+1)
		copy(b[0:2], []byte("YM"))
		binary.BigEndian.PutUint16(b[2:4], 203)
		for i := 4; i < 40; i++ {
			b[i] = 255
		}
		b[40] = 1
		fmt.Println("reply 101 data", b)
		ctx.Reply(b)
		ctx.Disconnect()
	})
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			b := make([]byte, 2+2+36+1)
			copy(b[0:2], []byte("YM"))
			binary.BigEndian.PutUint16(b[2:4], 202)
			for i := 4; i < 40; i++ {
				b[i] = 255
			}
			b[40] = 1
			msg := map[string]interface{}{
				"test1": 1,
				"test2": "aaa",
				"test3": 2.1,
				"test4": map[string]interface{}{
					"a": 1,
				},
			}
			nb, _ := json.Marshal(msg)
			b = append(b, nb...)
			//fmt.Println("broad cast:", b)
			tcpSrv.BroadCast(b)
		}
	}()
	err = tcpSrv.Serve(context.Background(), tcpLis)
	if err != nil {
		fmt.Println(err)
	}
}
