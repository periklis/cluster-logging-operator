package vector

import (
	"github.com/ViaQ/logerr/log"
	logging "github.com/openshift/cluster-logging-operator/apis/logging/v1"
	"github.com/openshift/cluster-logging-operator/internal/constants"
	"github.com/openshift/cluster-logging-operator/internal/generator"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector/helpers"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector/output/elasticsearch"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector/output/kafka"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector/output/loki"
	corev1 "k8s.io/api/core/v1"
)

func OutputFromPipelines(spec *logging.ClusterLogForwarderSpec, op generator.Options) logging.RouteMap {
	r := logging.RouteMap{}
	for _, p := range spec.Pipelines {
		for _, o := range p.OutputRefs {
			r.Insert(o, p.Name)
		}
	}
	return r
}

func Outputs(clspec *logging.ClusterLoggingSpec, secrets map[string]*corev1.Secret, clfspec *logging.ClusterLogForwarderSpec, op generator.Options) []generator.Element {
	outputs := []generator.Element{}
	ofp := OutputFromPipelines(clfspec, op)

	for _, o := range clfspec.Outputs {
		var secret *corev1.Secret
		if s, ok := secrets[o.Name]; ok {
			secret = s
			log.V(9).Info("Using secret configured in output: " + o.Name)
		} else {
			secret = secrets[constants.LogCollectorToken]
			if secret != nil {
				log.V(9).Info("Using secret configured in " + constants.LogCollectorToken)
			} else {
				log.V(9).Info("No Secret found in " + constants.LogCollectorToken)
			}
		}
		inputs := ofp[o.Name].List()
		switch o.Type {
		case logging.OutputTypeKafka:
			outputs = generator.MergeElements(outputs, kafka.Conf(o, inputs, secret, op))
		case logging.OutputTypeLoki:
			outputs = generator.MergeElements(outputs, loki.Conf(o, inputs, secret, op))
		case logging.OutputTypeElasticsearch:
			outputs = generator.MergeElements(outputs, elasticsearch.Conf(o, inputs, secret, op))
		}
	}
	outputs = append(outputs, PrometheusOutput(PrometheusOutputSinkName, []string{InternalMetricsSourceName}))
	return outputs
}

func PrometheusOutput(id string, inputs []string) generator.Element {
	return PrometheusExporter{
		ID:      id,
		Inputs:  helpers.MakeInputs(inputs...),
		Address: PrometheusExporterAddress,
	}
}
