package utils

import (
	"bytes"
	"encoding/binary"
)

type uintNum interface {
	uint | uint8 | uint16 | uint32 | uint64
}
type intNum interface {
	uintNum | int | int8 | int16 | int32 | int64
}
type floatNum interface{ float32 | float64 }
type anyNum interface{ intNum | floatNum }

func Remap[T anyNum](val, fromMin, fromMax, toMin, toMax T) T {
	return toMin + (val-fromMin)*(toMax-toMin)/(fromMax-fromMin)
}

func ToByteArray[T any](s T) (b []byte) {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, s)
	b = buf.Bytes()
	return
}
