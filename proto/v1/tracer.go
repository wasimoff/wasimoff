package wasimoffv1

// Additional tracing helper on task metadata.

import (
	"time"

	"google.golang.org/protobuf/proto"
)

// Add a traced event to task metadata.
func (t *Task_Metadata) TraceEvent(ev Task_TraceEvent_EventType) {
	// only append if metadata contains a trace message
	if t != nil && t.Trace != nil {
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

// Return the component of a logged event based on its enum range
func (e Task_TraceEvent_EventType) Component() Task_TraceEvent_Component {
	switch {
	case 10 <= e && e < 20:
		return Task_TraceEvent_Component_Client
	case 20 <= e && e < 30:
		return Task_TraceEvent_Component_Broker
	case 30 <= e && e < 40:
		return Task_TraceEvent_Component_Provider
	default:
		return Task_TraceEvent_Component_Unknown
	}
}

type Task_TraceEvent_Component int

const (
	Task_TraceEvent_Component_Unknown Task_TraceEvent_Component = iota
	Task_TraceEvent_Component_Client
	Task_TraceEvent_Component_Broker
	Task_TraceEvent_Component_Provider
)

// Try to correct for clock skew by "centering" logical spans inside their surrounding events.
// Since we can't rely on all components to have exact time (though NTP alone is admitteldy pretty good),
// this tries to correct skew by "centering" child spans within their parents (like the Provider events
// between Broker transmit and receive). It assumes equal latency front and back, which isn't quite correct
// because it also includes en/decoding steps. Could probably be amended with actual latency measurements,
// once known.
func (t *Task_Trace) ClockSkewCorrection() {
	events := t.Events

	// need at least three events
	if len(events) < 3 {
		return
	}

	// center brokers within client
	centerSpans(events,
		Task_TraceEvent_ClientTransmitRequest,
		Task_TraceEvent_ClientReceivedResponse,
		Task_TraceEvent_Component_Client,
	)

	// center providers within brokers
	centerSpans(events,
		Task_TraceEvent_BrokerTransmitProviderTask,
		Task_TraceEvent_BrokerReceivedProviderResult,
		Task_TraceEvent_Component_Broker,
	)

	// store result
	t.Events = events

}

func centerSpans(events []*Task_TraceEvent, start, end Task_TraceEvent_EventType, parent Task_TraceEvent_Component) {
	// hold parent/child start/end instants
	var (
		parentStart int64 = 0
		childStart  int64 = 0
		childEnd    int64 = 0
		parentEnd   int64 = 0
	)

	// collect the events to adjustEvents
	adjustEvents := make([]*Task_TraceEvent, 0)

	// search for broker spans within client
	for _, e := range events {

		// parent start found
		if *e.Event == start {
			parentStart = *e.Unixnano
			childStart = 0
			childEnd = 0
			parentEnd = 0
			adjustEvents = make([]*Task_TraceEvent, 0)
			continue
		}

		// parent end found
		if *e.Event == end {
			if parentStart == 0 {
				panic("parent end without parent start")
			}
			parentEnd = *e.Unixnano
			if len(adjustEvents) > 0 {

				childDuration := childEnd - childStart
				parentDuration := parentEnd - parentStart
				latency := (parentDuration - childDuration) / 2
				if latency < 0 {
					panic("negative latency, cannot center child span")
				}

				var previous int64
				for j, a := range adjustEvents {
					if j == 0 {
						previous = *a.Unixnano
						adjustedNano := parentStart + latency
						a.Unixnano = &adjustedNano
						// fmt.Println("adjusted", a.Event, ":", adjustedNano-previous)
					} else {
						adjustedNano := (*a.Unixnano - previous) + *adjustEvents[j-1].Unixnano
						previous = *a.Unixnano
						a.Unixnano = &adjustedNano
					}
				}

			}
			parentStart = 0
			continue
		}

		// collect events in between
		if parentStart != 0 {

			// found any other child event in between, that's unexpected
			if e.Event.Component() == parent {
				// if the child list is empty, just reset
				if len(adjustEvents) == 0 {
					parentStart = 0
					continue
				}
				panic("found unexpected parent event between start and the next end")
			}

			adjustEvents = append(adjustEvents, e)
			if childStart == 0 {
				childStart = *e.Unixnano
			}
			childEnd = *e.Unixnano
		}

	}
}
