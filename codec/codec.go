package codec

type Codec interface {
	//Encode 封包
	Encode([]byte) ([]byte, error)
	//Split 拆包（bufio使用的Split方法)
	Split(data []byte, atEOF bool) (advance int, token []byte, err error)
}
