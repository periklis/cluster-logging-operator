// go:build vector
package otlp

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	loggingv1 "github.com/openshift/cluster-logging-operator/api/logging/v1"
	"github.com/openshift/cluster-logging-operator/internal/runtime"
	"github.com/openshift/cluster-logging-operator/test/framework/functional"
	"github.com/openshift/cluster-logging-operator/test/helpers/otlp"
	. "github.com/openshift/cluster-logging-operator/test/matchers"
)

const (
	timestamp     = "2023-08-28T12:59:28.573159188+00:00"
	timestampNano = "1693227568573159188"
)

var _ = Describe("[Functional][Outputs][Otlp] Forward output to an OTEL Collector", func() {
	var (
		framework    *functional.CollectorFunctionalFramework
		appNamespace string
	)

	BeforeEach(func() {
		framework = functional.NewCollectorFunctionalFrameworkUsingCollector(loggingv1.LogCollectionTypeVector)
	})

	AfterEach(func() {
		framework.Cleanup()
	})

	It("should support application logs over OTLP with recommended kubernetes attributes", func() {
		functional.NewClusterLogForwarderBuilder(framework.Forwarder).
			FromInput(loggingv1.InputNameApplication).
			ToOutputWithVisitor(func(output *loggingv1.OutputSpec) {
				output.Name = loggingv1.OutputTypeOtlp
				output.Type = loggingv1.OutputTypeOtlp
				output.URL = "http://localhost:8090/v1/logs"
				output.OutputTypeSpec = loggingv1.OutputTypeSpec{
					Otlp: &loggingv1.Otlp{},
				}
			}, loggingv1.OutputTypeOtlp)

		ExpectOK(framework.DeployWithVisitor(func(b *runtime.PodBuilder) error {
			return framework.AddOTELCollector(b, loggingv1.OutputTypeOtlp)
		}))

		appNamespace = framework.Pod.Namespace

		// Write message to namespace
		crioLine := functional.NewCRIOLogMessage(timestamp, "Format me to OTLP!", false)
		Expect(framework.WriteMessagesToNamespace(crioLine, appNamespace, 1)).To(Succeed())
		crioLine = functional.NewCRIOLogMessage(timestamp, "My second Message", false)
		Expect(framework.WriteMessagesToNamespace(crioLine, appNamespace, 1)).To(Succeed())

		// Read log
		raw, err := framework.ReadRawApplicationLogsFrom(loggingv1.OutputTypeOtlp)
		Expect(err).To(BeNil(), "Expected no errors reading the logs for type")
		Expect(raw).ToNot(BeEmpty())

		// Parse log
		logs, err := otlp.ParseLogs(raw[0])
		Expect(err).To(BeNil(), "Expected no errors parsing the logs")

		// Validate
		resourceLog := logs.Logs[0]
		Expect(resourceLog.Resource.NamespaceName()).ToNot(Equal(appNamespace), "Expect namespace name to not be nested under k8s.namespace")
		Expect(resourceLog.ScopeLogs).ToNot(BeEmpty(), "Expected scope logs")
		Expect(resourceLog.ScopeLogs).To(HaveLen(1), "Expected a single scope")
		Expect(resourceLog.ScopeLogs[0].LogRecords).ToNot(BeEmpty(), "Expected log records for the scope")
		Expect(resourceLog.ScopeLogs[0].LogRecords).To(HaveLen(2), "Expected all log records for the scope")

		// Validate log records
		log := resourceLog.ScopeLogs[0].LogRecords[0]
		Expect(log.TimeUnixNano).To(Equal(timestampNano), "Expect timestamp to be converted into unix nano")
		Expect(log.SeverityText).To(BeEmpty(), "Expect severityText to be an empty string")
		Expect(log.SeverityNumber).To(Equal(9), "Expect severityNumber to be parsed to 9")
		Expect(log.Body.String).ToNot(BeEmpty())
	})

})
