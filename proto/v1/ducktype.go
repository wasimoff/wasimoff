package wasimoffv1

import "google.golang.org/protobuf/proto"

// Duck typing is a dynamic typing concept where an object's methods and properties
// determine its type at runtime. It follows the principle: "If it walks like a
// duck and quacks like a duck, it's a duck."

// Common duck-typed interface of possible task requests.
type Task_Request interface {
	proto.Message
	GetInfo() *Task_Metadata
	GetQos() *Task_QoS
	// GetParams() proto.Message
}

// Common duck-typed interface of possible task responses.
type Task_Response interface {
	proto.Message
	GetInfo() *Task_Metadata
	// GetResult() proto.Message
	// GetOK() proto.Message
	GetError() string
}
