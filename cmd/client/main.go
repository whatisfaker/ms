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
	fc := codec.NewLengthFieldCodec(binary.BigEndian, 1, 2, 0, 0)
	//fc := codec.NewLineCodec()
	for i := 1; i < 4; i++ {
		go func(i int) {
			clt := client.NewClient(
				":3033",
				client.AutoReconnect(false),
				client.Codec(fc),
				client.LogLevel("debug"),
				client.RouterKeyExtract(func(b []byte) int {
					return int(b[0])
				}),
			)
			identity := make([]byte, 4)
			binary.BigEndian.PutUint32(identity, uint32(i*10))
			clt.HandleRoute(2, func(b []byte) {
				fmt.Println("client", i, "route", 2, string(b[3:]))
				//fmt.Println(fmt.Sprintf("client %d route %d: %s", i, 2, string(b[3:])))
			})
			go func() {
				err := clt.Start()
				if err != nil {
					fmt.Println("start error", err)
				}
			}()
			<-time.After(2 * time.Second)
			tick := time.NewTicker(time.Duration(rand.Intn(15)+1) * time.Second)
			regist := append([]byte{1}, identity...)
			err := clt.Send(regist)
			if err != nil {
				fmt.Println("send error 1", err)
				return
			}
			for {
				<-tick.C
				if clt.IsClosed() {
					fmt.Println("client", i, "isClosed")
					break
				}
				data := append([]byte{2}, identity...)
				data = append(data, []byte(fmt.Sprintf("client: %d %s", i, time.Now().Format(time.RFC3339)))...)
				err := clt.Send(data)
				if err != nil {
					fmt.Println("send error 2", err)
					break
				}
			}
		}(i)
	}
	select {}
}
