package fluentd

import (
	logging "github.com/openshift/cluster-logging-operator/pkg/apis/logging/v1"
	. "github.com/openshift/cluster-logging-operator/pkg/generator"
	. "github.com/openshift/cluster-logging-operator/pkg/generator/fluentd/elements"
	"github.com/openshift/cluster-logging-operator/pkg/generator/fluentd/helpers"
)

const PrometheusMonitorTemplate = `
{{define "PrometheusMonitor" -}}
# {{.Desc}}
<source>
  @type prometheus
  bind "#{ENV['POD_IP']}"
  <ssl>
    enable true
    certificate_path "#{ENV['METRICS_CERT'] || '/etc/fluent/metrics/tls.crt'}"
    private_key_path "#{ENV['METRICS_KEY'] || '/etc/fluent/metrics/tls.key'}"
  </ssl>
</source>

<source>
  @type prometheus_monitor
  <labels>
    hostname ${hostname}
  </labels>
</source>

# excluding prometheus_tail_monitor
# since it leaks namespace/pod info
# via file paths

# tail_monitor plugin which publishes log_collected_bytes_total
<source>
  @type collected_tail_monitor
  <labels>
    hostname ${hostname}
  </labels>
</source>

# This is considered experimental by the repo
<source>
  @type prometheus_output_monitor
  <labels>
    hostname ${hostname}
  </labels>
</source>
{{end}}
`

type PrometheusMonitor = ConfLiteral

func PrometheusMetrics(spec *logging.ClusterLogForwarderSpec, o Options) []Element {
	return []Element{
		Pipeline{
			InLabel: helpers.LabelName("MEASURE"),
			Desc:    "Increment Prometheus metrics",
			SubElements: []Element{
				ConfLiteral{
					TemplateName: "EmitMetrics",
					TemplateStr:  EmitMetrics,
				},
				Match{
					Desc:      "Journal Logs go to INGRESS pipeline",
					MatchTags: "journal",
					MatchElement: Relabel{
						OutLabel: helpers.LabelName("INGRESS"),
					},
				},
				Match{
					Desc:      "Audit Logs go to INGRESS pipeline",
					MatchTags: "*audit.log",
					MatchElement: Relabel{
						OutLabel: helpers.LabelName("INGRESS"),
					},
				},
				Match{
					Desc:      "Kubernetes Logs go to CONCAT pipeline",
					MatchTags: "kubernetes.**",
					MatchElement: Relabel{
						OutLabel: helpers.LabelName("CONCAT"),
					},
				},
			},
		},
	}
}

const EmitMetrics string = `
{{define "EmitMetrics" -}}
<filter **>
  @type record_transformer
  enable_ruby
  <record>
    msg_size ${record.to_s.length}
  </record>
</filter>

<filter **>
  @type prometheus
  <metric>
    name cluster_logging_collector_input_record_total
    type counter
    desc The total number of incoming records
    <labels>
      tag ${tag}
      hostname ${hostname}
    </labels>
  </metric>
</filter>

<filter **>
  @type prometheus
  <metric>
    name cluster_logging_collector_input_record_bytes
    type counter
    desc The total bytes of incoming records
    key msg_size
    <labels>
      tag ${tag}
      hostname ${hostname}
    </labels>
  </metric>
</filter>

<filter **>
  @type record_transformer
  remove_keys msg_size
</filter>
{{end}}
`