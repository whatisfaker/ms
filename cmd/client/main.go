package main

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"time"

	"github.com/whatisfaker/ms/client"
	"github.com/whatisfaker/ms/codec"
	"github.com/whatisfaker/zaptrace/log"
	"go.uber.org/zap"
)

func main() {
	log := log.NewStdLogger("debug")
	// header := []byte{'a', 'b'}
	// data := []byte("[这里才是一个完整的数据包]")

	// sendData := make([]byte, 0)
	// sendData = append(sendData, header...)
	// sendData = append(sendData, data...)
	fc := codec.NewLengthFieldCodec(binary.BigEndian, 1, 2, 0, 0)
	//fc := codec.NewLineCodec()
	for i := 1; i < 2; i++ {
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
				log.Normal().Info("client recv", zap.Int("client", i), zap.Int("route", 2), zap.String("msg", string(b[3:])))
				//fmt.Println(fmt.Sprintf("client %d route %d: %s", i, 2, string(b[3:])))
			})
			go func() {
				err := clt.Start()
				if err != nil {
					log.Normal().Error("start error", zap.Error(err))
				}
			}()
			<-time.After(2 * time.Second)
			tick := time.NewTicker(time.Duration(rand.Intn(15)+1) * time.Second)
			regist := append([]byte{1}, identity...)
			err := clt.Send(regist)
			if err != nil {
				log.Normal().Error("send error 1", zap.Error(err))
				return
			}
			for {
				<-tick.C
				data := append([]byte{2}, identity...)
				data = append(data, []byte(fmt.Sprintf("client: %d %s", i, time.Now().Format(time.RFC3339)))...)
				err := clt.Send(data)
				if err != nil {
					log.Normal().Error("send error 2", zap.Error(err))
					break
				}
			}
		}(i)
	}
	select {}
}
