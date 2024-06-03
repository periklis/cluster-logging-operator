package otlp

import (
	"github.com/openshift/cluster-logging-operator/internal/generator/framework"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector/helpers"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	logging "github.com/openshift/cluster-logging-operator/api/logging/v1"
	"github.com/openshift/cluster-logging-operator/internal/generator/utils"
	. "github.com/openshift/cluster-logging-operator/test/matchers"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Generate vector config", func() {
	DescribeTable("for Otlp output... (TODO: complex spec and legacy http tests)", func(output logging.OutputSpec, secret *corev1.Secret, op framework.Options, exp string) {
		conf := New(helpers.MakeOutputID(output.Name), output, []string{"input_application_viaq_logtype"}, secret, nil, op) //, includeNS, excludes)
		Expect(exp).To(EqualConfigFrom(conf))
	},
		Entry("",
			logging.OutputSpec{
				Type: logging.OutputTypeOtlp,
				Name: "otel-collector",
				URL:  "http://localhost:4318/v1/logs",
			},
			nil,
			framework.NoOptions,
			`
[transforms.output_otel_collector_normalize]
type = "remap"
inputs = ["input_application_viaq_logtype"]
source = '''
  del(.file)
'''

[transforms.output_otel_collector_dedot]
type = "remap"
inputs = ["output_otel_collector_normalize"]
source = '''
  .openshift.sequence = to_unix_timestamp(now(), unit: "nanoseconds")
  if exists(.kubernetes.namespace_labels) {
      for_each(object!(.kubernetes.namespace_labels)) -> |key,value| { 
        newkey = replace(key, r'[\./]', "_") 
        .kubernetes.namespace_labels = set!(.kubernetes.namespace_labels,[newkey],value)
        if newkey != key {
          .kubernetes.namespace_labels = remove!(.kubernetes.namespace_labels,[key],true)
        }
      }
  }
  if exists(.kubernetes.labels) {
      for_each(object!(.kubernetes.labels)) -> |key,value| { 
        newkey = replace(key, r'[\./]', "_") 
        .kubernetes.labels = set!(.kubernetes.labels,[newkey],value)
        if newkey != key {
          .kubernetes.labels = remove!(.kubernetes.labels,[key],true)
        }
      }
  }
'''

# Normalize log records to OTLP schema
[transforms.output_otel_collector_pre_otlp]
type = "remap"
inputs = ["output_otel_collector_dedot"]
source = '''
  # OTLP for application logs only
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
'''

[transforms.output_otel_collector_reduce]
type = "reduce"
inputs = ["output_otel_collector_pre_otlp"]
expire_after_ms = 30000
max_events = 500
group_by = [".kubernetes.namespace_name",".kubernetes.pod_name",".kubernetes.container_name"]
merge_strategies.resource = "retain"
merge_strategies.logRecords = "array"

# Remap to match OTEL protocol
[transforms.output_otel_collector_post_otlp]
type = "remap"
inputs = ["output_otel_collector_reduce"]
source = '''
  . = {
        "resource": {
           "attributes": .resource.attributes,
        },
        "scopeLogs": [
          {"logRecords": .logRecords}
        ]
      }
'''

[sinks.output_otel_collector]
type = "http"
inputs = ["output_otel_collector_post_otlp"]
uri = "http://localhost:4318/v1/logs"
method = "post"
payload_prefix = "{\"resourceLogs\":"
payload_suffix = "}"
encoding.codec = "json"

[sinks.output_otel_collector.request]
headers = {"Content-Type"="application/json"}
`,
		),
	)
})

func TestHeaders(t *testing.T) {
	h := map[string]string{
		"k1": "v1",
		"k2": "v2",
	}
	expected := `{"k1"="v1","k2"="v2"}`
	got := utils.ToHeaderStr(h, "%q=%q")
	if got != expected {
		t.Logf("got: %s, expected: %s", got, expected)
		t.Fail()
	}
}

func TestVectorConfGenerator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vector Conf Generation")
}
