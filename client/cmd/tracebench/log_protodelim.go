package main

import (
	"log"
	"os"
	"sync"

	"github.com/klauspost/compress/zstd"
	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/proto"
)

// generic interface for output logfile writers
type TraceOutputEncoder interface {
	Write(proto.Message) error
	Close() error
}

// Write traces in varint-delimited protobuf format.
type ProtoDelimEncoder struct {
	mutex sync.Mutex
	file  *os.File
	zenc  *zstd.Encoder
}

func NewProtoDelimEncoder(filename string) *ProtoDelimEncoder {

	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("ERR: can't open %q for trace logging: %s", filename, err)
	}

	zenc, err := zstd.NewWriter(file)
	if err != nil {
		log.Fatalf("ERR: can't open zstd writer on file: %s", err)
	}

	return &ProtoDelimEncoder{
		mutex: sync.Mutex{},
		file:  file,
		zenc:  zenc,
	}

}

func (enc *ProtoDelimEncoder) Write(msg proto.Message) error {
	enc.mutex.Lock()
	defer enc.mutex.Unlock()
	_, err := protodelim.MarshalTo(enc.zenc, msg)
	return err
}

func (enc *ProtoDelimEncoder) Close() error {
	enc.mutex.Lock()
	defer enc.mutex.Unlock()
	if err := enc.zenc.Flush(); err != nil {
		enc.file.Close()
		return err
	}
	return enc.file.Close()
}
