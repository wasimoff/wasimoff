package main

import (
	"encoding/json"
	"log"
	"os"
	"sync"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type JsonLineEncoder struct {
	mutex    sync.Mutex
	file     *os.File
	encoder  *json.Encoder
	protoenc *protojson.MarshalOptions
}

func OpenOutputLog(filename string) *JsonLineEncoder {

	if filename == "" {
		return nil
	}

	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("ERR: can't open %q for trace logging: %s", filename, err)
	}

	return &JsonLineEncoder{
		mutex:   sync.Mutex{},
		file:    file,
		encoder: json.NewEncoder(file),
		protoenc: &protojson.MarshalOptions{
			Multiline:      false,
			UseEnumNumbers: false,
		},
	}

}

func (jl *JsonLineEncoder) EncodeJson(v any) error {
	jl.mutex.Lock()
	defer jl.mutex.Unlock()
	return jl.encoder.Encode(v)
}

func (jl *JsonLineEncoder) EncodeProto(msg proto.Message) error {
	buf, err := jl.protoenc.Marshal(msg)
	if err != nil {
		return err
	}
	buf = append(buf, '\n')
	jl.mutex.Lock()
	defer jl.mutex.Unlock()
	_, err = jl.file.Write(buf)
	return err
}

func (jl *JsonLineEncoder) Close() error {
	jl.mutex.Lock()
	defer jl.mutex.Unlock()
	return jl.file.Close()
}
