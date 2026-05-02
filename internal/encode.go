package internal

import (
	"bytes"
	"encoding/gob"
)

func encode[T any](obj T) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(obj); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decode[T any](data []byte) (T, error) {
	var obj T
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&obj); err != nil {
		return obj, err
	}

	return obj, nil
}
