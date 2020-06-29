# TCP server/client

## Server

根据自定义的解码方式,对TCP数据包进行解析。

```golang
LengthFieldCodec //参考java netty LengthFieldBasedFrameDecoder
LineCodec //bufio.ScanLines
```

## 例子

1. 最简server

```golang
lis, _ := net.Listen("tcp", ":3033")
srv := ms.NewServer()
srv.Serve(context.TODO(), lis)
```
