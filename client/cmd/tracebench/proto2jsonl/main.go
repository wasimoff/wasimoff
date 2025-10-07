package main

import (
	"bufio"
	"io"
	"log"
	"os"

	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/encoding/protojson"
	wasimoffv1 "wasi.team/proto/v1"
)

func main() {

	// struct to reserialize each message
	msg := &wasimoffv1.Task_Metadata{}

	// configure a json encoder without newlines
	jsonenc := &protojson.MarshalOptions{
		Multiline:      false,
		UseEnumNumbers: false,
	}

	// open stdin in a buffered reader
	stdin := bufio.NewReader(os.Stdin)

	for i := 0; ; i++ {
		// read the protobuf log file from stdin ...
		err := protodelim.UnmarshalFrom(stdin, msg)
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatalf("deserialize message %d: protodelim: %v", i, err)
		}
		// and output json-formatted trace per line
		buf, err := jsonenc.Marshal(msg)
		if err != nil {
			log.Fatalf("reserialize message %d: protojson: %v", i, err)
		}
		buf = append(buf, '\n')
		_, err = os.Stdout.Write(buf)
		if err != nil {
			log.Fatalf("reserialize message %d: stdout: %v", i, err)
		}
	}

}
