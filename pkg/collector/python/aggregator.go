// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

// +build python

package python

import (
	"github.com/DataDog/datadog-agent/pkg/aggregator"
	chk "github.com/DataDog/datadog-agent/pkg/collector/check"
	"github.com/DataDog/datadog-agent/pkg/metrics"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

/*
#include <datadog_agent_six.h>
#cgo !windows LDFLAGS: -ldatadog-agent-six -ldl
#cgo windows LDFLAGS: -ldatadog-agent-six -lstdc++ -static

char *getStringAddr(char **array, unsigned int idx);
*/
import "C"

// extractTags returns a slice with the contents of the char **tags.
func extractTags(tags **C.char) []string {
	if tags != nil {
		goTags := []string{}

		for i := 0; ; i++ {
			// Work around go vet raising issue about unsafe pointer
			tagPtr := C.getStringAddr(tags, C.uint(i))
			if tagPtr == nil {
				return goTags
			}
			tag := C.GoString(tagPtr)
			goTags = append(goTags, tag)
		}
	}
	return nil
}

// SubmitMetric is the method exposed to Python scripts to submit metrics
//export SubmitMetric
func SubmitMetric(checkID *C.char, metricType C.metric_type_t, metricName *C.char, value C.float, tags **C.char, hostname *C.char) {
	goCheckID := C.GoString(checkID)

	sender, err := aggregator.GetSender(chk.ID(goCheckID))
	if err != nil || sender == nil {
		log.Errorf("Error submitting metric to the Sender: %v", err)
		return
	}

	_name := C.GoString(metricName)
	_value := float64(value)
	_hostname := C.GoString(hostname)
	_tags := extractTags(tags)

	switch metricType {
	case C.DATADOG_AGENT_SIX_GAUGE:
		sender.Gauge(_name, _value, _hostname, _tags)
	case C.DATADOG_AGENT_SIX_RATE:
		sender.Rate(_name, _value, _hostname, _tags)
	case C.DATADOG_AGENT_SIX_COUNT:
		sender.Count(_name, _value, _hostname, _tags)
	case C.DATADOG_AGENT_SIX_MONOTONIC_COUNT:
		sender.MonotonicCount(_name, _value, _hostname, _tags)
	case C.DATADOG_AGENT_SIX_COUNTER:
		sender.Counter(_name, _value, _hostname, _tags)
	case C.DATADOG_AGENT_SIX_HISTOGRAM:
		sender.Histogram(_name, _value, _hostname, _tags)
	case C.DATADOG_AGENT_SIX_HISTORATE:
		sender.Historate(_name, _value, _hostname, _tags)
	}
}

// SubmitServiceCheck is the method exposed to Python scripts to submit service checks
//export SubmitServiceCheck
func SubmitServiceCheck(checkID *C.char, scName *C.char, status C.int, tags **C.char, hostname *C.char, message *C.char) {
	goCheckID := C.GoString(checkID)

	sender, err := aggregator.GetSender(chk.ID(goCheckID))
	if err != nil || sender == nil {
		log.Errorf("Error submitting metric to the Sender: %v", err)
		return
	}

	_name := C.GoString(scName)
	_status := metrics.ServiceCheckStatus(status)
	_tags := extractTags(tags)
	_hostname := C.GoString(hostname)
	_message := C.GoString(message)

	sender.ServiceCheck(_name, _status, _hostname, _tags, _message)
}

func eventParseString(value *C.char, fieldName string) string {
	if value == nil {
		log.Errorf("Can't parse value for key '%s' in event submitted from python check", fieldName)
		return ""
	}
	return C.GoString(value)
}

// SubmitEvent is the method exposed to Python scripts to submit events
//export SubmitEvent
func SubmitEvent(checkID *C.char, event *C.event_t) {
	goCheckID := C.GoString(checkID)

	sender, err := aggregator.GetSender(chk.ID(goCheckID))
	if err != nil || sender == nil {
		log.Errorf("Error submitting metric to the Sender: %v", err)
		return
	}

	_event := metrics.Event{
		Title:          eventParseString(event.title, "msg_title"),
		Text:           eventParseString(event.text, "msg_text"),
		Priority:       metrics.EventPriority(eventParseString(event.priority, "priority")),
		Host:           eventParseString(event.host, "host"),
		Tags:           extractTags(event.tags),
		AlertType:      metrics.EventAlertType(eventParseString(event.alert_type, "alert_type")),
		AggregationKey: eventParseString(event.aggregation_key, "aggregation_key"),
		SourceTypeName: eventParseString(event.source_type_name, "source_type_name"),
	}

	if event.ts == 0 {
		log.Errorf("Can't cast timestamp to integer in event submitted from python check")
	} else {
		_event.Ts = int64(event.ts)
	}

	sender.Event(_event)
	return
}
