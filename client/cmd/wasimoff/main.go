package main

import (
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

	"wasi.team/client"
	wasimoff "wasi.team/proto/v1"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var ( // config flags
	brokerUrl = "http://localhost:4080" // default broker base URL
	verbose   = false                   // be more verbose
	readstdin = false                   // read stdin for exec
	websock   = false                   // use websocket to send tasks
	rootfs    = ""                      // include a rootfs in exec
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

var c client.WasimoffClient

func main() {

	// commandline parser
	flag.StringVar(&brokerUrl, "broker", brokerUrl, "URL to the Broker to use")
	flag.StringVar(&cmdUpload, "upload", "", "Upload a file (wasm or zip) to the Broker and receive its ref")
	flag.BoolVar(&cmdExec, "exec", false, "Execute an uploaded binary by passing all non-flag args")
	flag.StringVar(&cmdRunWasip1, "run", "", "Run a prepared JSON job file")
	flag.StringVar(&cmdRunPyodide, "runpy", "", "Run a Python script file with Pyodide")
	flag.BoolVar(&verbose, "verbose", verbose, "Be more verbose and print raw messages for -exec")
	flag.BoolVar(&readstdin, "stdin", readstdin, "Read and send stdin when using -exec (not streamed)")
	flag.BoolVar(&websock, "ws", websock, "Use a WebSocket to connect to Broker")
	flag.StringVar(&rootfs, "rootfs", rootfs, "Use a rootfs ZIP in -exec task")
	flag.Parse()

	// establish a connection to the broker
	if websock {
		wc, err := client.NewWasimoffWebsocketClient(context.Background(), brokerUrl)
		if err != nil {
			log.Fatalf("ERR: can't connect to Broker: %s", err)
		} else {
			c = wc
		}
	} else {
		c = client.NewWasimoffConnectRpcClient(http.DefaultClient, brokerUrl)
	}

	switch true {

	// upload a file, optionally take another argument as name alias
	case cmdUpload != "":
		alias := flag.Arg(0)
		UploadFile(cmdUpload, alias)

	// execute an ad-hoc command, as if you were to run it locally
	case cmdExec:
		envs := []string{}
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

// upload a local file to the Broker
func UploadFile(filename, name string) {

	// read the file
	buf, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal("reading file: ", err)
	}

	// reuse basename as name if it's empty
	if name == "" {
		name = filepath.Base(filename)
	}

	ref, err := c.Upload(buf, name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, ref)
	os.Exit(0)

}

// execute an ad-hoc command line
func Execute(args, envs []string) {
	if len(args) == 0 {
		log.Fatal("need at least one argument")
	}

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

	// optionall add rootfs ref
	if rootfs != "" {
		request.Params.Rootfs = &wasimoff.File{Ref: &rootfs}
	}

	// make the request
	maybeDumpJson("[RunWasip1] run:", request)
	response, err := c.RunWasip1(context.Background(), request)

	// check for errors
	if err != nil {
		fmt.Fprintf(os.Stderr, "[RunWasip1] ERR: %s\n", err.Error())
		os.Exit(1)
	}
	if response.GetError() != "" {
		fmt.Fprintf(os.Stderr, "[RunWasip1] FAIL: %s\n", response.GetError())
		os.Exit(1)
	}

	// print the result
	ok := response.GetOk()
	maybeDumpJson("[RunWasip1] result:", ok)
	if len(ok.GetStderr()) != 0 {
		fmt.Fprintf(os.Stderr, "\033[31m%s\033[0m\n", string(ok.GetStderr()))
	}
	fmt.Fprintln(os.Stdout, string(ok.GetStdout()))
	os.Exit(int(ok.GetStatus()))

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
	response, err := c.RunWasip1Job(context.Background(), job)

	// check for errors
	if err != nil {
		fmt.Fprintf(os.Stderr, "[RunWasip1Job] ERR: %s\n", err.Error())
		os.Exit(1)
	}
	if response.GetError() != "" {
		fmt.Fprintf(os.Stderr, "[RunWasip1Job] FAIL: %s\n", response.GetError())
		os.Exit(1)
	}

	// print all task results
	for i, task := range response.GetTasks() {
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

// run a python script from file
func RunPythonScript(script string) {

	// read the file
	buf, err := os.ReadFile(script)
	if err != nil {
		log.Fatal("reading file: ", err)
	}
	script = string(buf)

	// prepare a request using this file
	request := &wasimoff.Task_Pyodide_Request{
		Params: &wasimoff.Task_Pyodide_Params{
			Run: &wasimoff.Task_Pyodide_Params_Script{
				Script: script,
			},
		},
	}

	// make the request
	maybeDumpJson("[RunPyodide] run:", request)
	response, err := c.RunPyodide(context.Background(), request)

	// check for errors
	if err != nil {
		fmt.Fprintf(os.Stderr, "[RunPyodide] ERR: %s\n", err.Error())
		os.Exit(1)
	}
	if response.GetError() != "" {
		fmt.Fprintf(os.Stderr, "[RunPyodide] FAIL: %s\n", response.GetError())
		os.Exit(1)
	}

	// print task result
	ok := response.GetOk()
	fmt.Fprintf(os.Stderr, "# Pyodide v%s: https://pyodide.org/en/%s/usage/packages-in-pyodide.html\n", ok.GetVersion(), ok.GetVersion())
	if len(ok.GetStderr()) != 0 {
		fmt.Fprintf(os.Stderr, "\033[31m%s\033[0m\n", string(ok.GetStderr()))
	}
	fmt.Fprintln(os.Stdout, string(ok.GetStdout()))
	if ok.Pickle != nil {
		fmt.Fprintf(os.Stderr, "\nresult pickle: %s\n", base64.StdEncoding.EncodeToString(ok.GetPickle()))
	}

}

func maybeDumpJson(pre string, m proto.Message) {
	if verbose {
		js, _ := protojson.Marshal(m)
		log.Println(pre, string(js))
	}
}
