package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type LengthFieldCodec struct {
	byteOrder   binary.ByteOrder
	fieldOffset uint
	fieldLength uint
	adjustment  int
	trip        uint
	dataOffset  int
}

func NewLengthFieldCodec(byteOrder binary.ByteOrder, fieldOffset uint, fieldLength uint, adjustment int, trip uint) Codec {
	return &LengthFieldCodec{
		byteOrder:   byteOrder,
		fieldOffset: fieldOffset,
		fieldLength: fieldLength,
		adjustment:  adjustment,
		trip:        trip,
		dataOffset:  int(fieldOffset + fieldLength),
	}
}

func (c *LengthFieldCodec) Split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if !atEOF {
		dl := len(data)
		if dl <= c.dataOffset {
			return 0, nil, nil
		}
		var l int
		switch c.fieldLength {
		case 1:
			l = int(data[c.fieldOffset : c.fieldOffset+1][0])
		case 2:
			l = int(c.byteOrder.Uint16(data[c.fieldOffset : c.fieldOffset+2]))
		case 4:
			l = int(c.byteOrder.Uint32(data[c.fieldOffset : c.fieldOffset+4]))
		case 8:
			l = int(c.byteOrder.Uint64(data[c.fieldOffset : c.fieldOffset+8]))
		default:
			err = fmt.Errorf("lengthField's fieldLength must be either 1, 2, 4, or 8")
			return
		}
		pl := l + c.dataOffset + c.adjustment
		if pl <= dl {
			advance = pl
			token = data[c.trip:pl]
			return
		}
	}
	return
}

func (c *LengthFieldCodec) Encode(in []byte) ([]byte, error) {
	w := new(bytes.Buffer)
	if c.fieldOffset > 0 {
		offsetData := in[:c.fieldOffset]
		err := binary.Write(w, c.byteOrder, &offsetData)
		if err != nil {
			return nil, err
		}
	}
	sizeData := make([]byte, c.fieldLength)
	dataLength := len(in) - int(c.fieldOffset) - c.adjustment
	switch c.fieldLength {
	case 1:
		sizeData[0] = byte(dataLength)
	case 2:
		c.byteOrder.PutUint16(sizeData, uint16(dataLength))
	case 4:
		c.byteOrder.PutUint32(sizeData, uint32(dataLength))
	case 8:
		c.byteOrder.PutUint64(sizeData, uint64(dataLength))
	default:
		return nil, fmt.Errorf("lengthField's fieldLength must be either 1, 2, 4, or 8")
	}
	err := binary.Write(w, c.byteOrder, &sizeData)
	if err != nil {
		return nil, err
	}
	otherData := in[c.fieldOffset:]
	err = binary.Write(w, c.byteOrder, &otherData)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}
