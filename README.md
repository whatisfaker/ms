# TCP server/client

## Server

根据自定义的解码方式,对TCP数据包进行解析。

```golang
LengthFieldCodec //参考java netty LengthFieldBasedFrameDecoder
LineCodec //bufio.ScanLines
```
