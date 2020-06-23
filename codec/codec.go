package codec

type Codec interface {
	Encode(in []byte) ([]byte, error)
	Split(data []byte, atEOF bool) (advance int, token []byte, err error)
}
