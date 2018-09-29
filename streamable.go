package almon

import (
	"bytes"
	"encoding/gob"
)

// Streamable represents data that can be streamed to a channel
type Streamable interface {
	GetBytes() ([]byte, error)
}

// ByteStreamed implements the `GetBytes` method
type ByteStreamed struct{}

// GetBytes outputs the data as bytes
func (st ByteStreamed) GetBytes() ([]byte, error) {
	var data bytes.Buffer
	enc := gob.NewEncoder(&data)

	err := enc.Encode(st)
	if err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}
