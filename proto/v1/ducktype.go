package wasimoffv1

import "google.golang.org/protobuf/proto"

type Task_Request interface {
	proto.Message
	GetInfo() *Task_Metadata
	GetQos() *Task_QoS
	// GetParams() proto.Message
}

type Task_Response interface {
	proto.Message
	GetInfo() *Task_Metadata
	// GetResult() proto.Message
	// GetOK() proto.Message
	GetError() string
}
