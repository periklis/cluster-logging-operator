package functional

import (
	log "github.com/ViaQ/logerr/v2/log/static"
	"github.com/openshift/cluster-logging-operator/internal/runtime"
	corev1 "k8s.io/api/core/v1"
	"strings"
)

const (
	OTELReceiverConf = `
exporters:
  debug:
    verbosity: detailed
  file:
    path: /tmp/app-logs
    format: json
    flush_interval: 1s
receivers:
  otlp:
    protocols:
      http:
        endpoint: localhost:8090
service:
  pipelines:
    logs:
      receivers: [otlp]
      exporters: [file,debug]
  telemetry:
    logs:
      level: debug
      development: false
      output_paths: [stdout]
      error_output_paths: [stdout]
`
	OTELCollectorImage = "quay.io/openshift-logging/opentelemetry-collector:0.96.0"
)

func (f *CollectorFunctionalFramework) AddOTELCollector(b *runtime.PodBuilder, outputName string) error {
	log.V(3).Info("Adding OTEL collector", "name", outputName)
	name := strings.ToLower(outputName)

	config := runtime.NewConfigMap(b.Pod.Namespace, name, map[string]string{
		"config.yaml": OTELReceiverConf,
	})
	log.V(2).Info("Creating configmap", "namespace", config.Namespace, "name", config.Name, "config.yaml", OTELReceiverConf)
	if err := f.Test.Client.Create(config); err != nil {
		return err
	}

	log.V(2).Info("Adding container", "name", name, "image", OTELCollectorImage)
	b.AddContainer(name, OTELCollectorImage).
		AddVolumeMount(config.Name, "/etc/otel", "", true).
		//AddVolumeMount("tmp", "/tmp", "", false).
		WithImagePullPolicy(corev1.PullAlways).
		End().
		//AddEmptyDirVolume("tmp").
		AddConfigMapVolume(config.Name, config.Name)

	return nil
}
