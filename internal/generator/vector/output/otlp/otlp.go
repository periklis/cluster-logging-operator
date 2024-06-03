package otlp

import (
	logging "github.com/openshift/cluster-logging-operator/api/logging/v1"
	. "github.com/openshift/cluster-logging-operator/internal/generator/framework"
	genhelper "github.com/openshift/cluster-logging-operator/internal/generator/helpers"
	. "github.com/openshift/cluster-logging-operator/internal/generator/vector/elements"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector/helpers"
	vectorhelpers "github.com/openshift/cluster-logging-operator/internal/generator/vector/helpers"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector/normalize"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector/output/common"
	corev1 "k8s.io/api/core/v1"
)

type Otlp struct {
	ComponentID string
	Inputs      string
	URI         string
	common.RootMixin
}

func (p Otlp) Name() string {
	return "vectorOtlpTemplate"
}

func (p Otlp) Template() string {
	return `{{define "` + p.Name() + `" -}}
[sinks.{{.ComponentID}}]
type = "http"
inputs = {{.Inputs}}
uri = "{{.URI}}"
method = "post"
payload_prefix = "{\"resourceLogs\":"
payload_suffix = "}"
encoding.codec = "json"
{{.Compression}}
{{end}}
`
}

func (p *Otlp) SetCompression(algo string) {
	p.Compression.Value = algo
}

func New(id string, o logging.OutputSpec, inputs []string, secret *corev1.Secret, strategy common.ConfigStrategy, op Options) []Element {
	dedottedID := vectorhelpers.MakeID(id, "dedot")
	if genhelper.IsDebugOutput(op) {
		return []Element{
			Debug(helpers.MakeID(id, "debug"), dedottedID),
		}
	}
	var els []Element
	els = append(els, normalize.DedotLabels(dedottedID, inputs))

	rerouteID := vectorhelpers.MakeID(id, "reroute")
	transformContainerID := vectorhelpers.MakeID(id, "pre_otlp_container")
	transformJournalID := vectorhelpers.MakeID(id, "pre_otlp_journal")
	reduceContainerID := vectorhelpers.MakeID(id, "group_by_container")
	reduceJournalID := vectorhelpers.MakeID(id, "group_by_node")
	formatID := vectorhelpers.MakeID(id, "post_otlp")
	els = append(els, Remapping(rerouteID, []string{dedottedID}))
	els = append(els, TransformContainer(transformContainerID, []string{rerouteID + ".container"}))
	els = append(els, GroupByContainer(reduceContainerID, []string{transformContainerID}))
	els = append(els, TransformJournal(transformJournalID, []string{rerouteID + ".journal"}))
	els = append(els, GroupByNode(reduceJournalID, []string{transformJournalID}))
	els = append(els, FormatBatch(formatID, []string{reduceContainerID, reduceJournalID}))

	sink := Output(id, o, []string{formatID}, secret, op)
	if strategy != nil {
		strategy.VisitSink(sink)
	}
	return MergeElements(
		els,
		[]Element{
			sink,
			common.NewAcknowledgments(id, strategy),
			common.NewBatch(id, strategy),
			common.NewBuffer(id, strategy),
			Request(id, o, strategy),
		},
		common.TLS(id, o, secret, op),
	)
}

func Output(id string, o logging.OutputSpec, inputs []string, secret *corev1.Secret, op Options) *Otlp {
	return &Otlp{
		ComponentID: id,
		Inputs:      vectorhelpers.MakeInputs(inputs...),
		URI:         o.URL,
		RootMixin:   common.NewRootMixin(nil),
	}
}

func Request(id string, o logging.OutputSpec, strategy common.ConfigStrategy) *common.Request {
	req := common.NewRequest(id, strategy)
	if o.Otlp != nil && o.Otlp.Timeout != 0 {
		req.TimeoutSecs.Value = o.Otlp.Timeout
	}
	headers := map[string]string{}
	if o.Otlp != nil && len(o.Otlp.Headers) != 0 {
		headers = o.Http.Headers
	}
	// required
	headers["Content-Type"] = "application/json"
	req.SetHeaders(headers)
	return req
}
