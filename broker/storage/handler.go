package storage

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// The UploadHandler returns a HTTP handler, which takes the POSTed file
// and inserts it into the provider storage, where workers can fetch their
// binaries and zip files from.
func (fs *FileStorage) upload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// if there's something in err upon return, we should log that
		var err error
		defer func() {
			if err != nil {
				log.Printf("ERR: Upload [%s]: %s", r.RemoteAddr, err)
			}
		}()

		// check the content-type of the request: accept zip or wasm
		ft, err := CheckMediaType(r.Header.Get("content-type"))
		if err != nil {
			http.Error(w, "unsupported filetype", http.StatusUnsupportedMediaType)
			return
		}

		// read the entire body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "reading body failed", http.StatusUnprocessableEntity)
			err = fmt.Errorf("reading body failed: %w", err)
			return
		}

		// can have a friendly lookup-name as query parameter
		name := r.URL.Query().Get("name")

		// insert file in storage
		file, err := fs.Insert(name, ft, body)
		if err != nil {
			http.Error(w, "inserting file in storage failed", http.StatusInternalServerError)
			err = fmt.Errorf("inserting in storage failed: %w", err)
			return
		}

		// return the content address to client
		w.WriteHeader(http.StatusOK)
		w.Header().Add("content-type", "text/plain")
		w.Header().Add("x-wasimoff-ref", file.Ref())
		fmt.Fprintln(w, file.Ref())

	}
}

// Make the FileStorage a http.Handler, so it can serve files on web requests.
// Expects a path value '{filename}' to retrieve the correct file.
func (fs *FileStorage) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// get the filename from path pattern
	filename := r.PathValue("filename")
	if filename == "" {
		http.Error(w, "path pattern not found", http.StatusInternalServerError)
		return
	}

	// maybe delegate to upload handler
	// TODO: check and disallow other methods when not uploading
	if r.Method == http.MethodPost && filename == "upload" {
		fs.upload()(w, r)
		return
	}

	// retrieve the file from storage
	file := fs.Get(filename)
	if file == nil {
		http.Error(w, "File not Found in storage", http.StatusNotFound)
		return
	}

	// put known content-type in a header and serve the file
	w.Header().Add("content-type", file.Media)
	w.Header().Add("x-wasimoff-ref", file.Ref())
	http.ServeContent(w, r, "", zerotime, bytes.NewReader(file.Bytes))

}

// TODO: should store the upload time as modtime
var zerotime = time.UnixMilli(0)
