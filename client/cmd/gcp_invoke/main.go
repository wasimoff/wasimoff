package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"google.golang.org/api/idtoken"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	wasimoffv1 "wasi.team/proto/v1"
)

var (
	credentials = flag.String("credentials", "account.json", "path to service account json")
	function    = flag.String("function", "https://wasimoff-runner-308704998937.europe-west10.run.app", "cloud run function url to invoke")
	readstdin   = flag.Bool("stdin", false, "read from stdin and send with task")
	verbose     = flag.Bool("v", false, "be more verbose and print messages")
)

func main() {

	// parse commandline argument, expect exec args in positional
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		log.Fatalln("args: must give at least a binary name")
	}

	// create an idtoken client
	client, err := idtoken.NewClient(context.Background(), *function, idtoken.WithCredentialsFile(*credentials))
	if err != nil {
		log.Fatalf("Failed to create idtoken client: %s", err)
	}

	// create the request
	request := &wasimoffv1.Task_Wasip1_Request{
		Info: &wasimoffv1.Task_Metadata{
			Id: proto.String("gcloud-invoke"),
		},
		Params: &wasimoffv1.Task_Wasip1_Params{
			Binary: &wasimoffv1.File{Ref: proto.String(args[0])},
			Args:   args,
		},
	}
	// optionally read stdin
	if *readstdin {
		stdin, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalln("failed reading stdin:", err)
		}
		request.Params.Stdin = stdin
	}

	// serialize the request
	body, err := proto.Marshal(request)
	if err != nil {
		log.Fatalf("Failed to serialize Protobuf request: %s", err)
	}
	if *verbose {
		fmt.Printf("\033[36;1mRequest --> \033[0;36m%s\033[0m\n", prototext.Format(request))
	}

	// do the request
	resp, err := client.Post(*function, "application/proto", bytes.NewBuffer(body))
	if err != nil {
		log.Fatalf("Failed request: %s", err)
	}
	body, bodyerr := io.ReadAll(resp.Body)
	if bodyerr != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}
	if resp.StatusCode != 200 {
		log.Fatalf("Request failed: %s\n%s", resp.Status, string(body))
	}
	if *verbose {
		fmt.Printf("\033[33;1m%s --> \033[0;33m%q\033[0m\n\n", resp.Status, string(body))
	}

	// decode the response
	response := &wasimoffv1.Task_Wasip1_Response{}
	if err := proto.Unmarshal(body, response); err != nil {
		log.Fatalf("Failed to decode body as Task_Wasip1_Response: %s", err)
	} else {
		ok := response.GetOk()
		if len(ok.GetStderr()) != 0 {
			fmt.Fprintf(os.Stderr, "\033[31m%s\033[0m\n", string(ok.GetStderr()))
		}
		fmt.Fprint(os.Stdout, string(ok.GetStdout()))
		os.Exit(int(ok.GetStatus()))
	}

}
