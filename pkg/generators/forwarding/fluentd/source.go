package fluentd

import (
	"fmt"

	logging "github.com/openshift/cluster-logging-operator/pkg/apis/logging/v1"
	"github.com/openshift/cluster-logging-operator/pkg/constants"
	"k8s.io/apimachinery/pkg/util/sets"
)

func (engine *ConfigGenerator) generateSource(sources sets.String) (results []string, err error) {
	// Order of templates matters.
	var templates []string
	if sources.Has(logging.InputNameInfrastructure) {
		templates = append(templates, "inputSourceJournalTemplate")
	}
	if sources.Has(logging.InputNameApplication) || sources.Has(logging.InputNameInfrastructure) {
		templates = append(templates, "inputSourceContainerTemplate")
	}
	if sources.Has(logging.InputNameAudit) {
		templates = append(templates, "inputSourceHostAuditTemplate")
		templates = append(templates, "inputSourceK8sAuditTemplate")
		templates = append(templates, "inputSourceOpenShiftAuditTemplate")
	}
	if len(templates) == 0 {
		return results, fmt.Errorf("No recognized input types: %v", sources.List())
	}
	data := struct {
		LoggingNamespace           string
		CollectorPodNamePrefix     string
		LogStorePodNamePrefix      string
		VisualizationPodNamePrefix string
	}{
		constants.OpenshiftNS,
		constants.FluentdName,
		constants.ElasticsearchName,
		constants.KibanaName,
	}
	for _, template := range templates {
		result, err := engine.Execute(template, data)
		if err != nil {
			return results, fmt.Errorf("Error processing template %s: %v", template, err)
		}
		results = append(results, result)
	}
	return results, nil
}
