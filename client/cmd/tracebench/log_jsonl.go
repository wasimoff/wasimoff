package main

import (
	"log"
	"os"
	"sync"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Write traces in JSONL format (one trace per line).
type JsonLineEncoder struct {
	mutex    sync.Mutex
	file     *os.File
	protoenc *protojson.MarshalOptions
}

func NewJsonLineEncoder(filename string) *JsonLineEncoder {

	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("ERR: can't open %q for trace logging: %s", filename, err)
	}

	return &JsonLineEncoder{
		mutex: sync.Mutex{},
		file:  file,
		protoenc: &protojson.MarshalOptions{
			Multiline:      false,
			UseEnumNumbers: false,
		},
	}

}

func (enc *JsonLineEncoder) Write(msg proto.Message) error {
	buf, err := enc.protoenc.Marshal(msg)
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
