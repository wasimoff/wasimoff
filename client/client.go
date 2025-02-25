package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"wasimoff/broker/net/transport"
	wasimoff "wasimoff/proto/v1"
	"wasimoff/proto/v1/wasimoffv1connect"

	"connectrpc.com/connect"
	"github.com/gabriel-vasile/mimetype"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var ( // config flags
	brokerUrl = "http://localhost:4080" // default broker base URL
	verbose   = false                   // be more verbose
	readstdin = false                   // read stdin for exec
	websock   = false                   // use websocket to send tasks
)

var ( // command flags, pick one
	cmdUpload     = ""    // upload this file
	cmdExec       = false // execute cmdline
	cmdRunWasip1  = ""    // run wasip1 job
	cmdRunPyodide = ""    // run python file
)

func init() {
	// get the Broker URL from env
	if url, ok := os.LookupEnv("BROKER"); ok {
		brokerUrl = strings.TrimRight(url, "/")
	}
}

func main() {

	// commandline parser
	flag.StringVar(&brokerUrl, "broker", brokerUrl, "URL to the Broker to use")
	flag.StringVar(&cmdUpload, "upload", "", "Upload a file (wasm or zip) to the Broker and receive its ref")
	flag.BoolVar(&cmdExec, "exec", false, "Execute an uploaded binary by passing all non-flag args")
	flag.StringVar(&cmdRunWasip1, "run", "", "Run a prepared JSON job file")
	flag.StringVar(&cmdRunPyodide, "runpy", "", "Run a Python script file with Pyodide")
	flag.BoolVar(&verbose, "verbose", verbose, "Be more verbose and print raw messages for -exec")
	flag.BoolVar(&readstdin, "stdin", readstdin, "Read and send stdin when using -exec (not streamed)")
	flag.BoolVar(&websock, "ws", websock, "Use a WebSocket to send -run job")
	flag.Parse()

	switch true {

	// upload a file, optionally take another argument as name alias
	case cmdUpload != "":
		alias := flag.Arg(0)
		UploadFile(cmdUpload, alias)

	// execute an ad-hoc command, as if you were to run it locally
	case cmdExec:
		envs := []string{} // TODO: read os.Environ?
		args := flag.Args()
		Execute(args, envs)

	// execute a prepared JSON job
	case cmdRunWasip1 != "":
		RunWasip1Job(cmdRunWasip1)

	// execute a python script task
	case cmdRunPyodide != "":
		RunPythonScript(cmdRunPyodide)

	// no command specified
	default:
		fmt.Fprintln(os.Stderr, "ERR: at least one of -upload, -exec, -run, -runpy must be used")
		flag.Usage()
		os.Exit(2)
	}

}

// open wasimoff + connectrpc client connection
func ConnectRpcClient() wasimoffv1connect.TasksClient {
	return wasimoffv1connect.NewTasksClient(http.DefaultClient, brokerUrl+"/api/client")
}

// upload a local file to the Broker
func UploadFile(filename, name string) {

	// read the file
	buf, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal("reading file: ", err)
	}

	// detect the mediatype from buf
	mt := mimetype.Detect(buf)

	// reuse basename as name if it's empty
	if name == "" {
		name = filepath.Base(filename)
	}

	// upload to the broker
	resp, err := http.Post(
		brokerUrl+"/api/storage/upload?name="+name, mt.String(), bytes.NewBuffer(buf))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// print the response and exit depending on statusCode
	body, _ := io.ReadAll(resp.Body)
	fmt.Fprint(os.Stdout, string(body))
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, resp.Status)
		os.Exit(1)
	}
	os.Exit(0)

}

// execute an ad-hoc command line
func Execute(args, envs []string) {
	if len(args) == 0 {
		log.Fatal("need at least one argument")
	}

	// connect the client
	client := ConnectRpcClient()

	// prepare the request
	request := &wasimoff.Task_Wasip1_Request{
		Params: &wasimoff.Task_Wasip1_Params{
			Binary: &wasimoff.File{Ref: proto.String(args[0])},
			Args:   args,
			Envs:   envs,
		},
	}
	// optionally read stdin
	if readstdin {
		stdin, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERR: failed reading stdin:", err)
			os.Exit(1)
		}
		request.Params.Stdin = stdin
	}

	// make the request
	maybeDumpJson("[RunWasip1] run:", request)
	response, err := client.RunWasip1(
		context.Background(),
		connect.NewRequest(request),
	)

	// print the result
	if err != nil {
		fmt.Fprintf(os.Stderr, "[RunWasip1] ERR: %s\n", err.Error())
		os.Exit(1)
	} else {
		r := response.Msg
		if r.GetError() != "" {
			fmt.Fprintf(os.Stderr, "[RunWasip1] FAIL: %s\n", r.GetError())
			os.Exit(1)
		} else {
			ok := r.GetOk()
			maybeDumpJson("[RunWasip1] result:", ok)
			if len(ok.GetStderr()) != 0 {
				fmt.Fprintf(os.Stderr, "\033[31m%s\033[0m\n", string(ok.GetStderr()))
			}
			fmt.Fprintln(os.Stdout, string(ok.GetStdout()))
			os.Exit(int(ok.GetStatus()))
		}
	}

}

// run a prepared job configuration from file
func RunWasip1Job(config string) {

	// read the file
	buf, err := os.ReadFile(config)
	if err != nil {
		log.Fatal("reading file: ", err)
	}

	// decode with protojson and report any errors locally
	job := &wasimoff.Task_Wasip1_JobRequest{}
	if err = protojson.Unmarshal(buf, job); err != nil {
		log.Fatal("unmarshal job: ", err)
	}

	// run the job
	var results *wasimoff.Task_Wasip1_JobResponse
	if websock {
		results = runWasip1JobOnWebSocket(job)
	} else {
		results = runWasip1JobOnRpc(job)
	}

	// print all task results
	if results.GetError() != "" {
		fmt.Fprintf(os.Stderr, "[RunWasip1Job] FAIL: %s\n", results.GetError())
		os.Exit(1)
	}
	for i, task := range results.GetTasks() {
		if task.GetError() != "" {
			fmt.Fprintf(os.Stderr, "[task %d] FAIL: %s\n", i, task.GetError())
		} else {
			r := task.GetOk()
			fmt.Fprintf(os.Stderr, "[task %d] exit:%d\n", i, *r.Status)
			if r.Artifacts != nil {
				fmt.Fprintf(os.Stderr, "artifact: %s\n", base64.StdEncoding.EncodeToString(r.Artifacts.GetBlob()))
			}
			if len(r.GetStderr()) != 0 {
				fmt.Fprintf(os.Stderr, "\033[31m%s\033[0m\n", string(r.GetStderr()))
			}
			fmt.Fprintln(os.Stdout, string(r.GetStdout()))
		}
	}

}

// run a prepared job configuration from proto message
func runWasip1JobOnRpc(job *wasimoff.Task_Wasip1_JobRequest) *wasimoff.Task_Wasip1_JobResponse {

	// connect the rpc client
	client := ConnectRpcClient()

	// make the request
	response, err := client.RunWasip1Job(
		context.Background(),
		connect.NewRequest(job),
	)

	// print failure or return response
	if err != nil {
		fmt.Fprintf(os.Stderr, "[RunWasip1Job] ERR: %s\n", err.Error())
		os.Exit(1)
	}
	return response.Msg

}

// alternatively, run a job by sending each task over websocket
func runWasip1JobOnWebSocket(job *wasimoff.Task_Wasip1_JobRequest) *wasimoff.Task_Wasip1_JobResponse {

	// open a websocket to the broker
	socket, err := transport.DialWebSocketTransport(context.TODO(), brokerUrl+"/api/client/ws")
	if err != nil {
		log.Printf("[WebSocket] ERR: dial: %s", err)
	}
	// wrap it in a messenger for RPC
	messenger := transport.NewMessengerInterface(socket)
	defer messenger.Close(nil)

	// chan and list to collect responses
	ntasks := len(job.GetTasks())
	done := make(chan *transport.PendingCall, ntasks)
	responses := make([]*wasimoff.Task_Wasip1_Response, ntasks)

	// submit all tasks
	for i, task := range job.GetTasks() {
		task.InheritNil(job.Parent)
		if verbose {
			log.Printf("[WebSocket] submit task %d", i)
		}

		// store index in context
		ctx := context.WithValue(context.TODO(), ctxJobIndex{}, i)

		// assemble wrapped task and fire it off
		tr := &wasimoff.Task_Wasip1_Request{
			Params: task,
		}
		messenger.SendRequest(ctx, tr, &wasimoff.Task_Wasip1_Response{}, done)
	}

	// wait for all responses
	for ntasks > 0 {
		call := <-done
		ntasks -= 1
		i := call.Context.Value(ctxJobIndex{}).(int)
		if verbose {
			log.Printf("[WebSocket] received result %d: err=%v", i, call.Error)
		}

		if call.Error != nil {
			responses[i].Result = &wasimoff.Task_Wasip1_Response_Error{
				Error: call.Error.Error(),
			}
		} else {
			if resp, ok := call.Response.(*wasimoff.Task_Wasip1_Response); ok {
				responses[i] = resp
			} else {
				responses[i] = &wasimoff.Task_Wasip1_Response{
					Result: &wasimoff.Task_Wasip1_Response_Error{
						Error: "failed to parse the response as pb.Task_Response",
					},
				}
			}
		}
	}

	// return the results
	return &wasimoff.Task_Wasip1_JobResponse{
		Tasks: responses,
	}
}

// run a python script from file
func RunPythonScript(script string) {

	// read the file
	buf, err := os.ReadFile(script)
	if err != nil {
		log.Fatal("reading file: ", err)
	}
	script = string(buf)

	// open wasimoff client connection
	client := ConnectRpcClient()

	// prepare a request using this file
	request := &wasimoff.Task_Pyodide_Request{
		Params: &wasimoff.Task_Pyodide_Params{
			Script: &script,
		},
	}

	// make the request
	maybeDumpJson("[RunPyodide] run:", request)
	response, err := client.RunPyodide(
		context.Background(),
		connect.NewRequest(request),
	)

	// print task result
	if err != nil {
		fmt.Fprintf(os.Stderr, "[RunPyodide ERR] %s\n", err.Error())
		os.Exit(1)
	} else {
		r := response.Msg
		if r.GetError() != "" {
			fmt.Fprintf(os.Stderr, "[RunPyodide FAIL] %s\n", r.GetError())
			os.Exit(1)
		} else {
			ok := r.GetOk()
			fmt.Fprintf(os.Stderr, "# Pyodide v%s: https://pyodide.org/en/%s/usage/packages-in-pyodide.html\n", ok.GetVersion(), ok.GetVersion())
			if len(ok.GetStderr()) != 0 {
				fmt.Fprintf(os.Stderr, "\033[31m%s\033[0m\n", string(ok.GetStderr()))
			}
			fmt.Fprintln(os.Stdout, string(ok.GetStdout()))
			if ok.Pickle != nil {
				fmt.Fprintf(os.Stderr, "\nresult pickle: %s\n", base64.StdEncoding.EncodeToString(ok.GetPickle()))
			}
		}
	}

}

func maybeDumpJson(pre string, m proto.Message) {
	if verbose {
		js, _ := protojson.Marshal(m)
		log.Println(pre, string(js))
	}
}

// typed key to store index in context
type ctxJobIndex struct{}
