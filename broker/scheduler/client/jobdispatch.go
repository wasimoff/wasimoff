package client

import (
	"context"
	"errors"
	"fmt"
	"wasi.team/broker/provider"
	wasimoff "wasi.team/proto/v1"

	"google.golang.org/protobuf/proto"
)

//
// ----------> offloading jobs

// An OffloadingJob holds the pb.OffloadWasiArgs from the request along with
// some internal information about the requesting client.
type OffloadingJob struct {
	JobID      string // used to track all tasks of this request
	ClientAddr string // remote address of the requesting client
	JobSpec    *wasimoff.Task_Wasip1_JobRequest
}

// dispatchJob takes a run configuration, generates individual tasks from it,
// schedules them in the queue and eventually returns with the results of all
// those tasks.
func dispatchJob(
	ctx context.Context,
	store *provider.ProviderStore,
	job *OffloadingJob,
	queue chan *provider.AsyncTask,
) *wasimoff.Task_Wasip1_JobResponse {

	// go through all the *pb.Files in parent and tasks to resolve names from storage
	errs := []error{}
	if job.JobSpec.Parent != nil {
		errs = append(errs, store.Storage.ResolvePbFile(job.JobSpec.Parent.Binary))
		errs = append(errs, store.Storage.ResolvePbFile(job.JobSpec.Parent.Rootfs))
	}
	for _, task := range job.JobSpec.Tasks {
		errs = append(errs, store.Storage.ResolvePbFile(task.Binary))
		errs = append(errs, store.Storage.ResolvePbFile(task.Rootfs))
	}
	if err := errors.Join(errs...); err != nil {
		return &wasimoff.Task_Wasip1_JobResponse{
			Error: proto.String(err.Error()),
		}
	}

	// create slice for queued tasks and a sufficiently large channel for done signals
	pending := make([]*provider.AsyncTask, len(job.JobSpec.Tasks))
	doneChan := make(chan *provider.AsyncTask, len(pending)+10)

	// dispatch each task from slice
	for i, spec := range job.JobSpec.Tasks {

		// create the request+response for remote procedure call
		response := wasimoff.Task_Wasip1_Response{}
		request := wasimoff.Task_Wasip1_Request{
			// common task metadata with index counter
			Info: &wasimoff.Task_Metadata{
				Id:        proto.String(fmt.Sprintf("%s/%d", job.JobID, i)),
				Requester: &job.ClientAddr,
			},
			// inherit empty parameters from the parent job
			Params: spec.InheritNil(job.JobSpec.Parent),
		}

		// create the async task with the common done channel and queue it for dispatch
		task := provider.NewAsyncTask(ctx, &request, &response, doneChan)
		pending[i] = task
		queue <- task
	}

	// wait for all tasks to finish
	done := 0
	for t := range doneChan {
		done++
		if t.Error == nil {
			// store.RateTick()
		}
		if done == len(pending) {
			break
		}
	}

	// collect the task responses
	jobResponse := &wasimoff.Task_Wasip1_JobResponse{
		Tasks: make([]*wasimoff.Task_Wasip1_Response, len(pending)),
	}
	for i, task := range pending {

		// internal scheduling error
		if task.Error != nil {
			jobResponse.Tasks[i] = &wasimoff.Task_Wasip1_Response{
				Result: &wasimoff.Task_Wasip1_Response_Error{
					Error: task.Error.Error(),
				},
			}
			continue
		}

		// cast the response type
		r, ok := task.Response.(*wasimoff.Task_Wasip1_Response)
		if !ok {
			jobResponse.Tasks[i] = &wasimoff.Task_Wasip1_Response{
				Result: &wasimoff.Task_Wasip1_Response_Error{
					Error: "unexpected result type",
				},
			}
			continue
		}

		// otherwise just pass it through
		jobResponse.Tasks[i] = r

	}

	return jobResponse
}
