package outputs

import (
	"testing"

	logging "github.com/openshift/cluster-logging-operator/pkg/apis/logging/v1"
	"github.com/openshift/cluster-logging-operator/test/functional"
	"github.com/openshift/cluster-logging-operator/test/helpers/loki"
	"github.com/openshift/cluster-logging-operator/test/runtime"
	"github.com/stretchr/testify/require"
)

// NOTE: This test demonstrates how Ginkgo-like nesting & naming can be
// accomplished using the standard go testing package plus testify require/assert.
//
// The test uses the Ginkgo-like name it() to hammer home the point,
// that isn't a requirement for tests in general.
func TestLokiOutput(t *testing.T) {
	var ff *functional.FluentdFunctionalFramework

	it := func(name string, f func(t *testing.T)) {
		ff = functional.NewFluentdFunctionalFramework() // Before
		t.Log("FIXME skipping cleanup")
		//		defer ff.Cleanup()                              // After
		t.Run(name, f)
	}

	deploy := func(t *testing.T, lokiSpec *logging.Loki) *loki.Client {
		t.Helper()
		require := require.New(t)
		ff.Forwarder.Spec.Outputs = append(ff.Forwarder.Spec.Outputs,
			logging.OutputSpec{
				Name: "loki",
				Type: "loki",
				URL:  "http://localhost:3100",
				OutputTypeSpec: logging.OutputTypeSpec{
					Loki: lokiSpec,
				},
				// FIXME secret tests
				// Secret: &logging.OutputSecretSpec{},
			})
		ff.Forwarder.Spec.Pipelines = append(ff.Forwarder.Spec.Pipelines,
			logging.PipelineSpec{
				OutputRefs: []string{"loki"},
				InputRefs:  []string{"application"},
			})
		require.NoError(ff.DeployWithVisitor(func(b *runtime.PodBuilder) error {
			b.Pod.Spec.Containers = append(b.Pod.Spec.Containers, loki.NewContainer("loki"))
			return nil
		}))
		// FIXME: expose loki service for query.
		c, err := loki.NewClient("http://localhost:3100")
		require.NoError(err)
		return c
	}

	it("sends all log types to loki with expected labels", func(t *testing.T) {
		require := require.New(t)
		c := deploy(t, nil)
		require.NoError(ff.WriteMessagesToApplicationLog("application", 1))
		require.NoError(ff.WriteMessagesToAuditLog("audit", 1))
		require.NoError(ff.WriteMessagesTok8sAuditLog("k8s audit", 1))
		require.NoError(ff.WriteMessagesToAuditLog("openshift audit", 1))

		r, err := c.QueryAll(loki.Selector{}, 4)
		require.NoError(err)
		require.Len(r, 4) //FIXME?
		require.Equal(loki.Selector{"STREAMS": "FORLOGS"}, r[0].Stream)
		require.Equal([]string{"logs"}, r[0].Logs())
		// ...
	})

	it("sets tenant ID from namespace", func(t *testing.T) {
		require := require.New(t)
		c := deploy(t, &logging.Loki{TenantID: "kubernetes.namespace_name"})
		require.NoError(ff.WritesApplicationLogs(1))
		_, err := c.QueryAll(loki.Selector{"index": "what?"}, 1)
		require.NoError(err)
		// FIXME check tenant
	})

	it("sets tenant ID from label", func(t *testing.T) {
		require := require.New(t)
		c := deploy(t, &logging.Loki{TenantID: "kubernetes.namespace_name"})
		require.NoError(ff.WritesApplicationLogs(1))
		_, err := c.QueryAll(loki.Selector{"index": "what?"}, 1)
		require.NoError(err)
		// FIXME check tenant
	})

	it("sends modified labels", func(t *testing.T) {
		require := require.New(t)
		c := deploy(t, &logging.Loki{TenantID: "kubernetes.namespace_name"})
		require.NoError(ff.WritesApplicationLogs(1))
		_, err := c.QueryAll(loki.Selector{"index": "what?"}, 1)
		require.NoError(err)
		// FIXME check labels
	})
}

// TODO benchmark?
