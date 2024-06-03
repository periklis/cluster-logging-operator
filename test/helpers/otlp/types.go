package otlp

import (
	"encoding/json"
	log "github.com/ViaQ/logerr/v2/log/static"
)

type ResourceLogs struct {
	Logs []ResourceLog `json:"resourceLogs,omitempty"`
}
type Resource struct {
	Attributes Attributes `json:"attributes,omitempty"`
}

type ResourceLog struct {
	Resource  Resource   `json:"resource,omitempty"`
	ScopeLogs []ScopeLog `json:"scopeLogs,omitempty"`
}

type ScopeLog struct {
	Scope      Scope       `json:"scope,omitempty"`
	LogRecords []LogRecord `json:"logRecords,omitempty"`
}

type Scope struct {
	Name       string     `json:"name,omitempty"`
	Version    string     `json:"version,omitempty"`
	Attributes Attributes `json:"attributes,omitempty"`
}

type LogRecord struct {
	TimeUnixNano         string      `json:"timeUnixNano,omitempty"`
	ObservedTimeUnixNano string      `json:"observedTimeUnixNano,omitempty"`
	SeverityNumber       int         `json:"severityNumber,omitempty"`
	SeverityText         string      `json:"severityText,omitempty"`
	TraceID              string      `json:"traceId,omitempty"`
	SpanID               string      `json:"spanId,omitempty"`
	Body                 StringValue `json:"body,omitempty"`
	Attributes           []Attribute `json:"attributes,omitempty"`
}
type Attributes []Attribute

type Attribute struct {
	Key   string         `json:"key,omitempty"`
	Value AttributeValue `json:"value,omitempty"`
}

type AttributeValue struct {
	String StringValue `json:",inline,omitempty"`
	Bool   bool        `json:"boolValue,omitempty"`
	Int    int         `json:"intValue,omitempty"`
	Float  float64     `json:"doubleValue,omitempty"`
	Array  ArrayValue  `json:"arrayValue,omitempty"`
	Map    KVListValue `json:"kvlistValue,omitempty"`
}
type StringValue struct {
	String string `json:"stringValue,omitempty"`
}
type ArrayValue struct {
	Values []StringValue `json:"values,omitempty"`
}
type KVListValue struct {
	Values []AttributeValue `json:"values,omitempty"`
}

func ParseLogs(in string) (ResourceLogs, error) {
	logs := ResourceLogs{}
	if in == "" {
		return logs, nil
	}
	log.V(4).Info("OTEL ParseLogs", "in", in)
	err := json.Unmarshal([]byte(in), &logs)
	if err != nil {
		return logs, err
	}

	return logs, nil
}
