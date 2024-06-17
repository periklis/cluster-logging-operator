package otlp

import "github.com/openshift/cluster-logging-operator/internal/generator/framework"

type BasicAuthConf framework.ConfLiteral

func (t BasicAuthConf) Name() string {
	return "otlpBasicAuthConf"
}

func (t BasicAuthConf) Template() string {
	return `
{{define "otlpBasicAuthConf" -}}
# {{.Desc}}
[sinks.{{.ComponentID}}.auth]
{{- end}}`
}
