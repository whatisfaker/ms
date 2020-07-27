package main

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"time"

	"github.com/whatisfaker/ms/client"
	"github.com/whatisfaker/ms/codec"
)

func main() {
	// header := []byte{'a', 'b'}
	// data := []byte("[这里才是一个完整的数据包]")

	// sendData := make([]byte, 0)
	// sendData = append(sendData, header...)
	// sendData = append(sendData, data...)
	fc := codec.NewLengthFieldCodec(binary.BigEndian, 0, 8, 0, 0)
	//fc := codec.NewLineCodec()
	for i := 1; i < 2; i++ {
		go func(i int) {
			clt := client.NewClient(
				"localhost:4033",
				client.AutoReconnect(false),
				client.Codec(fc),
				client.LogLevel("debug"),
				client.RouterKeyExtract(func(b []byte) int {
					a := binary.BigEndian.Uint16(b[10:12])
					return int(a)
				}),
			)
			//identity := make([]byte, 4)
			//binary.BigEndian.PutUint32(identity, uint32(i*10))
			clt.HandleRoute(201, func(b []byte) {
				fmt.Println("client", i, "route", 201, string(b[49:]))
				//fmt.Println(fmt.Sprintf("client %d route %d: %s", i, 2, string(b[3:])))
			})
			clt.HandleRoute(203, func(b []byte) {
				fmt.Println("client", i, "route", 203, string(b[49:]))
				//fmt.Println(fmt.Sprintf("client %d route %d: %s", i, 2, string(b[3:])))
			})
			clt.HandleRoute(202, func(b []byte) {
				fmt.Println("client", i, "route", 202, string(b[49:]))
				//fmt.Println(fmt.Sprintf("client %d route %d: %s", i, 2, string(b[3:])))
			})
			go func() {
				err := clt.Start()
				if err != nil {
					fmt.Println("start error", err)
				}
			}()
			// <-time.After(2 * time.Second)
			tick := time.NewTicker(time.Duration(rand.Intn(15)+1) * time.Second)
			// err := clt.Send(regist)
			// if err != nil {
			// 	fmt.Println("send error 1", err)
			// 	return
			// }
			for {
				<-tick.C
				if clt.IsClosed() {
					fmt.Println("client", i, "isClosed")
					break
				}
				b := make([]byte, 2+2+36+1)
				copy(b[0:2], []byte("YM"))
				binary.BigEndian.PutUint16(b[2:4], 101)
				for i := 4; i < 40; i++ {
					b[i] = 0
				}
				b[40] = 1
				err := clt.Send(b)
				if err != nil {
					fmt.Println("send error 2", err)
					break
				}
			}
		}(i)
	}
	select {}
}
