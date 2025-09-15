package wasimoffv1

import (
	"time"

	"google.golang.org/protobuf/proto"
)

// Additional helpers on the generated types.

// Fill any unset (nil) task parameters from a parent task specification.
func (wt *Task_Wasip1_Params) InheritNil(parent *Task_Wasip1_Params) *Task_Wasip1_Params {
	if parent == nil {
		// nothing to do when parent is nil
		return wt
	}
	if wt.Binary == nil {
		wt.Binary = parent.Binary
	}
	if wt.Args == nil {
		wt.Args = parent.Args
	}
	if wt.Envs == nil {
		wt.Envs = parent.Envs
	}
	if wt.Stdin == nil {
		wt.Stdin = parent.Stdin
	}
	if wt.Rootfs == nil {
		wt.Rootfs = parent.Rootfs
	}
	if wt.Artifacts == nil {
		wt.Artifacts = parent.Artifacts
	}
	return wt
}

// Return a string list of needed files for a task request.
func (tr *Task_Wasip1_Request) GetRequiredFiles() (files []string) {
	files = make([]string, 0, 2) // usually max. binary + rootfs
	p := tr.Params

	if p.Binary != nil && p.Binary.GetRef() != "" {
		files = append(files, *p.Binary.Ref)
	}
	if p.Rootfs != nil && p.Rootfs.GetRef() != "" {
		files = append(files, *p.Rootfs.Ref)
	}

	return files
}

// Add a traced event to task metadata.
func (t *Task_Metadata) TraceEvent(ev Task_TraceEvent_EventType) {
	// only append if metadata contains a trace message
	if t.Trace != nil {
		// prepare list with some capacity when empty
		if t.Trace.Events == nil {
			t.Trace.Events = make([]*Task_TraceEvent, 0, 20)
		}
		t.Trace.Events = append(t.Trace.Events, &Task_TraceEvent{
			Unixnano: proto.Int64(time.Now().UnixNano()),
			Event:    ev.Enum(),
		})
	}
}
