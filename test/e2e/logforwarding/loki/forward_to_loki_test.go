package loki

import (
	"fmt"
	"runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	loggingv1 "github.com/openshift/cluster-logging-operator/pkg/apis/logging/v1"
	v1 "github.com/openshift/cluster-logging-operator/pkg/apis/logging/v1"
	"github.com/openshift/cluster-logging-operator/pkg/logger"
	"github.com/openshift/cluster-logging-operator/test/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("LogForwarding", func() {
	_, filename, _, _ := runtime.Caller(0)
	logger.Infof("Running %s", filename)

	const (
		lokiOutputName = "loki-receiver"
	)

	var (
		e2e = helpers.NewE2ETestFramework()
	)

	BeforeEach(func() {
		if err := e2e.DeployLogGenerator(); err != nil {
			logger.Errorf("unable to deploy log generator. E: %s", err.Error())
		}
	})

	Describe("when ClusterLogging is configured with 'forwarding' to an administrator managed", func() {
		Describe("Loki single server", func() {
			BeforeEach(func() {
				if err := e2e.DeployLokiReceiver(); err != nil {
					Fail(fmt.Sprintf("Unable to deploy loki receiver: %v", err))
				}

				cr := helpers.NewClusterLogging(helpers.ComponentTypeCollector)
				if err := e2e.CreateClusterLogging(cr); err != nil {
					Fail(fmt.Sprintf("Unable to create an instance of cluster logging: %v", err))
				}
				forwarding := &loggingv1.ClusterLogForwarder{
					TypeMeta: metav1.TypeMeta{
						Kind:       loggingv1.ClusterLogForwarderKind,
						APIVersion: loggingv1.SchemeGroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "instance",
					},
					Spec: loggingv1.ClusterLogForwarderSpec{
						Outputs: []loggingv1.OutputSpec{
							loggingv1.OutputSpec{
								Name: lokiOutputName,
								Type: logforward.OutputTypeLoki,
								URL: fmt.Sprintf(
									"http://%s",
									e2e.LogStore.ClusterLocalEndpoint(),
								),
							},
						},
						Pipelines: []loggingv1.PipelineSpec{
							{
								Name:       "test-app",
								InputRefs:  []string{v1.InputNameApplication},
								OutputRefs: []string{lokiOutputName},
							},
							{
								Name:       "test-audit",
								InputRefs:  []string{v1.InputNameAudit},
								OutputRefs: []string{lokiOutputName},
							},
							{
								Name:       "test-infra",
								InputRefs:  []string{v1.InputNameInfrastructure},
								OutputRefs: []string{lokiOutputName},
							},
						},
					},
				}
				if err := e2e.CreateClusterLogForwarder(forwarding); err != nil {
					Fail(fmt.Sprintf("Unable to create an instance of ClusterLogForwarder: %v", err))
				}
				components := []helpers.LogComponentType{helpers.ComponentTypeCollector}
				for _, component := range components {
					if err := e2e.WaitFor(component); err != nil {
						Fail(fmt.Sprintf("Failed waiting for component %s to be ready: %v", component, err))
					}
				}

			})

			It("should send logs to the logstore", func() {
				Expect(e2e.LogStore.HasInfraStructureLogs(helpers.DefaultWaitForLogsTimeout)).To(BeTrue(), "Expected to find stored infrastructure logs")
				Expect(e2e.LogStore.HasApplicationLogs(helpers.DefaultWaitForLogsTimeout)).To(BeTrue(), "Expected to find stored application logs")
				Expect(e2e.LogStore.HasAuditLogs(helpers.DefaultWaitForLogsTimeout)).To(BeTrue(), "Expected to find stored audit logs")
			})

			AfterEach(func() {
				e2e.Cleanup()
			})
		})

		Describe("Loki cluster", func() {
			BeforeEach(func() {
				if err := e2e.DeployLokiClusterReceiver(); err != nil {
					Fail(fmt.Sprintf("Unable to deploy loki receiver: %v", err))
				}

				cr := helpers.NewClusterLogging(helpers.ComponentTypeCollector)
				if err := e2e.CreateClusterLogging(cr); err != nil {
					Fail(fmt.Sprintf("Unable to create an instance of cluster logging: %v", err))
				}
				forwarding := &loggingv1.ClusterLogForwarder{
					TypeMeta: metav1.TypeMeta{
						Kind:       loggingv1.ClusterLogForwarderKind,
						APIVersion: loggingv1.SchemeGroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "instance",
					},
					Spec: loggingv1.ClusterLogForwarderSpec{
						Outputs: []loggingv1.OutputSpec{
							{
								Name: lokiOutputName,
								Type: logforward.OutputTypeLoki,
								URL:  "http://loki-dst.openshift-logging.svc.cluster.local",
							},
						},
						Pipelines: []loggingv1.PipelineSpec{
							{
								Name:       "test-app",
								InputRefs:  []string{v1.InputNameApplication},
								OutputRefs: []string{lokiOutputName},
							},
							{
								Name:       "test-audit",
								InputRefs:  []string{v1.InputNameAudit},
								OutputRefs: []string{lokiOutputName},
							},
							{
								Name:       "test-infra",
								InputRefs:  []string{v1.InputNameInfrastructure},
								OutputRefs: []string{lokiOutputName},
							},
						},
					},
				}
				if err := e2e.CreateClusterLogForwarder(forwarding); err != nil {
					Fail(fmt.Sprintf("Unable to create an instance of logforwarding: %v", err))
				}
				components := []helpers.LogComponentType{helpers.ComponentTypeCollector}
				for _, component := range components {
					if err := e2e.WaitFor(component); err != nil {
						Fail(fmt.Sprintf("Failed waiting for component %s to be ready: %v", component, err))
					}
				}

			})

			It("should send logs to the logstore", func() {
				Expect(e2e.LogStore.HasInfraStructureLogs(helpers.DefaultWaitForLogsTimeout)).To(BeTrue(), "Expected to find stored infrastructure logs")
				Expect(e2e.LogStore.HasApplicationLogs(helpers.DefaultWaitForLogsTimeout)).To(BeTrue(), "Expected to find stored application logs")
				Expect(e2e.LogStore.HasAuditLogs(helpers.DefaultWaitForLogsTimeout)).To(BeTrue(), "Expected to find stored audit logs")
			})

			AfterEach(func() {
				e2e.Cleanup()
			})
		})
	})

})
