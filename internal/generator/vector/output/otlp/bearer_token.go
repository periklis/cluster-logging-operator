package otlp

import "github.com/openshift/cluster-logging-operator/internal/generator/vector/output/common"

type BearerToken common.BearerToken

func (bt BearerToken) Name() string {
	return "otlpBearerTokenTemplate"
}

func (bt BearerToken) Template() string {
	return `{{define "` + bt.Name() + `" -}}
strategy = "bearer"
token = "{{.Token}}"
{{end}}
`
}
