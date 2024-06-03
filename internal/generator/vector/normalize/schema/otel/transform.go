package otel

import (
	. "github.com/openshift/cluster-logging-operator/internal/generator/framework"
	"strings"

	. "github.com/openshift/cluster-logging-operator/internal/generator/vector/elements"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector/helpers"
)

type Reduce struct {
	ComponentID string
	Desc        string
	Inputs      string
}

func (r Reduce) Name() string {
	return "reduceTemplate"
}

func (r Reduce) Template() string {
	return `{{define "reduceTemplate" -}}
{{if .Desc -}}
# {{.Desc}}
{{end -}}
[transforms.{{.ComponentID}}]
type = "reduce"
inputs = {{.Inputs}}
expire_after_ms = 30000
max_events = 500
group_by = [".kubernetes.namespace_name",".kubernetes.pod_name",".kubernetes.container_name"]
merge_strategies.resource = "retain"
merge_strategies.logRecords = "array"
{{end}}
`
}

func GroupBy(id string, inputs []string) Element {
	return Reduce{
		ComponentID: id,
		Inputs:      helpers.MakeInputs(inputs...),
	}
}

func FormatBatch(id string, inputs []string) Element {
	return Remap{
		Desc:        "Remap to match otelp",
		ComponentID: id,
		Inputs:      helpers.MakeInputs(inputs...),
		VRL: strings.TrimSpace(`

. = {
      "resource": {
         "attributes": .resource.attributes,
      },
      "scopeLogs": [
        {"logRecords": .logRecords}
      ]
    }
`),
	}
}

func Transform(id string, inputs []string) Element {
	return Remap{
		Desc:        "Normalize log records to OTEL schema",
		ComponentID: id,
		Inputs:      helpers.MakeInputs(inputs...),
		VRL: strings.TrimSpace(`
# Tech preview, OTEL for application logs only
if .log_type == "application" {
# Remove some fields
meta = [
  "kubernetes.pod_name", 
  "kubernetes.pod_id",
  "kubernetes.namespace_name",
  "kubernetes.container_name",
  "openshift.cluster_id",
  "hostname"
]
replace = {
  "pod.id": "pod.uid",
  "cluster.id": "cluster.uid",
  "hostname": "node.name"
}

resource.attributes = []
for_each(meta) -> |index,value| {
  sub_key = value
  path = split(value,".")
  if length(path) > 1 {
    sub_key = replace!(path[1],"_",".")
  }
  if get!(replace, [sub_key]) != null {
    sub_key = string!(get!(replace, [sub_key]))
  }
  key = "k8s." + sub_key
  a ={
    "key": key,
            "value": {
              "stringValue": get!(.,path)
            }
          }
  resource.attributes = append(resource.attributes, [a])
}

for_each(object!(.kubernetes.labels)) -> |key,value|{  
  a ={
    "key": "k8s.pod.labels." + key,
    "value": {
        "stringValue": value
    }
  }
  resource.attributes = append(resource.attributes, [a])
}
resource.attributes = append(resource.attributes, [{
  "key": "openshift.log.type",
  "value": {
    "stringValue": .log_type
  }
  }]
)

# transform the record
r = {}
r.timeUnixNano = to_string(to_unix_timestamp(parse_timestamp!(.@timestamp, format:"%+"), unit:"nanoseconds"))
r.observedTimeUnixNano = to_string(to_unix_timestamp(now(), unit:"nanoseconds"))
.severityText = del(.level)
# Convert syslog severity keyword to number, default to 9 (unknown)
r.severityNumber = to_syslog_severity(.severityText) ?? 9
r.body = {"stringValue": string!(.message)}
. = {
  "kubernetes": .kubernetes,
  "resource": resource,
  "logRecords": r
 }
}
`),
	}
}
