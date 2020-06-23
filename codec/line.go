package codec

import "bufio"

type LineCodec struct {
}

func NewLineCodec() Codec {
	return &LineCodec{}
}

func (c *LineCodec) Split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	return bufio.ScanLines(data, atEOF)
}

func (c *LineCodec) Encode(in []byte) ([]byte, error) {
	out := append([]byte{}, in...)
	out = append(out, '\n')
	return out, nil
}
