package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/proto"
)

// generic interface for output logfile writers
type TraceOutputEncoder interface {
	Write(proto.Message) error
	Close() error
}

// open an output encoder, either jsonl or protodelim based on filename
func OpenTraceOutputEncoder(filename string) (TraceOutputEncoder, error) {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("cannot open %q for logging: %v", filename, err)
	}
	switch {
	case strings.HasSuffix(file.Name(), ".jsonl"):
		return NewJsonLineEncoder(file), nil
	default:
		return NewProtoDelimEncoder(file), nil
	}
}

type ProtoDelimEncoder struct {
	mutex   sync.Mutex
	file    io.WriteCloser
	encoder *protodelim.MarshalOptions
}

// Write traces in varint-delimited protobuf format.
func NewProtoDelimEncoder(file io.WriteCloser) TraceOutputEncoder {

	return &ProtoDelimEncoder{
		mutex:   sync.Mutex{},
		file:    file,
		encoder: &protodelim.MarshalOptions{},
	}

}

func (enc *ProtoDelimEncoder) Write(msg proto.Message) error {
	enc.mutex.Lock()
	defer enc.mutex.Unlock()
	_, err := enc.encoder.MarshalTo(enc.file, msg)
	return err
}

func (enc *ProtoDelimEncoder) Close() error {
	enc.mutex.Lock()
	defer enc.mutex.Unlock()
	return enc.file.Close()
}
