package main

import (
	"io"
	"sync"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type JsonLineEncoder struct {
	mutex   sync.Mutex
	file    io.WriteCloser
	encoder *protojson.MarshalOptions
}

// Write traces in JSONL format (one trace per line).
func NewJsonLineEncoder(file io.WriteCloser) TraceOutputEncoder {

	return &JsonLineEncoder{
		mutex: sync.Mutex{},
		file:  file,
		encoder: &protojson.MarshalOptions{
			Multiline:      false,
			UseEnumNumbers: false,
		},
	}

}

func (enc *JsonLineEncoder) Write(msg proto.Message) error {
	buf, err := enc.encoder.Marshal(msg)
	if err != nil {
		return err
	}
	buf = append(buf, '\n')
	enc.mutex.Lock()
	defer enc.mutex.Unlock()
	_, err = enc.file.Write(buf)
	return err
}

func (enc *JsonLineEncoder) Close() error {
	enc.mutex.Lock()
	defer enc.mutex.Unlock()
	return enc.file.Close()
}
