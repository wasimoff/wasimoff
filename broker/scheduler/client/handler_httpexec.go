package client

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/shlex"
	"wasi.team/broker/net/transport"
	wasimoff "wasi.team/proto/v1"
)

func HttpExecWasip1Handler(rpc *ConnectRpcServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		addr := transport.ProxiedAddr(r)

		// get executable name from matched pattern
		executable := r.PathValue("wasm")
		if executable == "" {
			http.Error(w, "executable name required", http.StatusBadRequest)
			return
		}

		// must be a POST request
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// create a task stub
		task := &wasimoff.Task_Wasip1_Params{
			Binary: &wasimoff.File{Ref: &executable},
			Args:   []string{executable},
			Envs:   []string{},
		}

		// parse arguments from X-Args header (with shell quoting rules)
		if argsHeader := r.Header.Get("X-Args"); argsHeader != "" {
			parsed, err := shlex.Split(argsHeader)
			if err != nil {
				http.Error(w, fmt.Sprintf("malformatted X-Args: %s", err), http.StatusBadRequest)
				return
			} else {
				// prepend binary name as argv[0]
				parsed = append([]string{executable}, parsed...)
				task.Args = parsed
			}
		}

		// parse environment variables
		for key, values := range r.Header {

			// set variables from X-Env-* headers (strip prefix)
			if strings.HasPrefix(key, "X-Env-") {
				if len(values) > 0 {
					// use the first header for each variable
					k := strings.TrimPrefix(key, "X-Env-")
					task.Envs = append(task.Envs, fmt.Sprintf("%s=%s", strings.ToUpper(k), values[0]))
				}
			}

			// add rootfs ref from header
			if key == "X-RootFS-Ref" {
				if v := values[0]; v != "" {
					task.Rootfs = &wasimoff.File{Ref: &v}
				}
			}

			// add artifact paths from header(s)
			if key == "X-Artifact" {
				task.Artifacts = values
			}

		}

		// pass through content-length and content-type specifically and always override
		if contentLength := r.Header.Get("Content-Length"); contentLength != "" {
			task.Envs = append(task.Envs, fmt.Sprintf("%s=%s", "CONTENT_LENGTH", contentLength))
		}
		if contentType := r.Header.Get("Content-Type"); contentType != "" {
			task.Envs = append(task.Envs, fmt.Sprintf("%s=%s", "CONTENT_TYPE", contentType))
		}

		// read stdin from request body
		if r.Body != nil {
			var stdin bytes.Buffer
			defer r.Body.Close()
			_, err := io.Copy(&stdin, r.Body)
			if err != nil {
				http.Error(w, "error reading request body", http.StatusBadRequest)
				return
			}
			task.Stdin = stdin.Bytes()
		}

		// log.Printf("{client %s} %s", addr, prototext.Format(task))
		request := connect.NewRequest(&wasimoff.Task_Wasip1_Request{
			Info:   &wasimoff.Task_Metadata{Requester: &addr},
			Params: task,
		})
		response, err := rpc.RunWasip1(r.Context(), request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if response.Msg == nil {
			http.Error(w, "empty rpc response", http.StatusInternalServerError)
			return
		}
		msg := response.Msg

		// add provider id in header
		if prov := msg.GetInfo().GetProvider(); prov != "" {
			w.Header().Set("X-Wasimoff-Provider", prov)
		}

		// result is an error
		if err := msg.GetError(); err != "" {
			w.Header().Set("X-Wasimoff-Result", "Error")
			w.Write([]byte(err))
			return
		}

		// result is ok
		if ok := msg.GetOk(); ok != nil {
			w.Header().Set("X-Wasimoff-Result", "Ok")

			// set the actual status code
			if ok.Status != nil {
				w.Header().Set("X-Wasimoff-Status", strconv.Itoa(int(*ok.Status)))
			}

			// base64-encode the artifact, let's hope nobody ever uses this for large files
			if ok.Artifacts != nil && ok.Artifacts.Blob != nil {
				blob := base64.StdEncoding.EncodeToString(ok.Artifacts.Blob)
				w.Header().Set("X-Wasimoff-Artifacts", blob)
			}

			// we use 200 OK even for non-zero status codes because "technically" the RPC succeeded
			w.WriteHeader(http.StatusOK)

			// no way to interleave stderr and stdout afterwards, so put stderr on top
			w.Write(ok.Stderr)
			w.Write(ok.Stdout)
			return

		}

		// oops, we shouldn't be here
		http.Error(w, "oops, emptty result", http.StatusInternalServerError)

	}
}
