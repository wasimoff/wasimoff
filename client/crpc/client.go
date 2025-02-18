package main

import (
	"context"
	"log"
	"net/http"
	wasimoffv1 "wasimoff/proto/v1"
	"wasimoff/proto/v1/wasimoffv1connect"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

func main() {

	client := wasimoffv1connect.NewWasimoffClient(
		http.DefaultClient,
		"http://localhost:4080",
		connect.WithGRPC(),
	)

	res, err := client.RunWasip1(
		context.Background(), connect.NewRequest(&wasimoffv1.Task_Wasip1_Params{
			Binary: &wasimoffv1.File{Ref: proto.String("tsp.wasm")},
			Args:   []string{"bin", "rand", "10"},
		}))
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(prototext.Format(res.Msg))

}
