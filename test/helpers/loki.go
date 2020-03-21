package helpers

import (
	"fmt"
	"strings"
	"time"

	"github.com/openshift/cluster-logging-operator/pkg/logger"
	"github.com/openshift/cluster-logging-operator/test/helpers/loki"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type lokiReceiverMode string

const (
	singleServerMode lokiReceiverMode = "single-server"
	clusterMode      lokiReceiverMode = "cluster"
)

type lokiReceiver struct {
	tc   *E2ETestFramework
	mode lokiReceiverMode
}

func (lr *lokiReceiver) ApplicationLogs(timeout time.Duration) (string, error) {
	res, err := lr.tc.lokiLogs(ProjectIndexPrefix)
	if err != nil {
		return "", err
	}
	return res.ToString(), nil
}

func (lr *lokiReceiver) HasInfraStructureLogs(timeout time.Duration) (bool, error) {
	err := wait.Poll(defaultRetryInterval, timeout, func() (done bool, err error) {
		res, err := lr.tc.lokiLogs(InfraIndexPrefix)
		if err != nil {
			return false, err
		}
		return res.NonEmpty(), nil
	})
	return true, err
}

func (lr *lokiReceiver) HasApplicationLogs(timeout time.Duration) (bool, error) {
	err := wait.Poll(defaultRetryInterval, timeout, func() (done bool, err error) {
		res, err := lr.tc.lokiLogs(ProjectIndexPrefix)
		if err != nil {
			return false, err
		}
		return res.NonEmpty(), nil
	})
	return true, err
}

func (lr *lokiReceiver) HasAuditLogs(timeout time.Duration) (bool, error) {
	err := wait.Poll(defaultRetryInterval, timeout, func() (done bool, err error) {
		res, err := lr.tc.lokiLogs(AuditIndexPrefix)
		if err != nil {
			return false, err
		}
		return res.NonEmpty(), nil
	})
	return true, err
}

func (kr *lokiReceiver) GrepLogs(expr string, timeToWait time.Duration) (string, error) {
	return "Not Found", fmt.Errorf("Not implemented")
}

func (kr *lokiReceiver) ClusterLocalEndpoint() string {
	switch kr.mode {
	case singleServerMode:
		return loki.SingleServerEndpoint(OpenshiftLoggingNS)
	case clusterMode:
		return loki.ClusterEndpoint(OpenshiftLoggingNS)
	default:
		return ""
	}
}

func (tc *E2ETestFramework) lokiLogs(indexName string) (*loki.Response, error) {
	pod, err := tc.KubeClient.Core().Pods(OpenshiftLoggingNS).Get(loki.UtilName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	logger.Debugf("Pod %s", pod.GetName())

	indexName = fmt.Sprintf("%swrite", indexName)
	cmd := []string{"/bin/sh", "/data/loki_util", tc.LogStore.ClusterLocalEndpoint(), indexName}
	stdout, err := tc.PodExec(OpenshiftLoggingNS, loki.UtilName, loki.UtilName, cmd)
	if err != nil {
		fmt.Println(strings.Join(cmd, " "))
		return nil, err
	}

	res, err := loki.Parse(stdout)
	if err != nil {
		fmt.Println(strings.Join(cmd, " "))
		return nil, err
	}

	return &res, nil
}

func (tc *E2ETestFramework) DeployLokiReceiver() error {
	if err := tc.createLokiConfigMap(); err != nil {
		return err
	}

	if err := tc.createLokiStatefulSet(); err != nil {
		return err
	}

	if err := tc.createLokiService(); err != nil {
		return err
	}

	if err := tc.createLokiQueryUtil(); err != nil {
		return err
	}

	receiver := &lokiReceiver{
		tc:   tc,
		mode: singleServerMode,
	}
	tc.LogStore = receiver

	return nil
}

func (tc *E2ETestFramework) createLokiConfigMap() error {
	cm := loki.NewConfigMap(OpenshiftLoggingNS)

	tc.AddCleanup(func() error {
		var zerograce int64
		return tc.KubeClient.Core().ConfigMaps(OpenshiftLoggingNS).Delete(cm.GetName(), metav1.NewDeleteOptions(zerograce))
	})

	if _, err := tc.KubeClient.Core().ConfigMaps(OpenshiftLoggingNS).Create(cm); err != nil {
		return err
	}

	return nil
}

func (tc *E2ETestFramework) createLokiStatefulSet() error {
	app := loki.NewStatefulSet(OpenshiftLoggingNS)

	tc.AddCleanup(func() error {
		var zerograce int64
		return tc.KubeClient.Apps().StatefulSets(OpenshiftLoggingNS).Delete(app.GetName(), metav1.NewDeleteOptions(zerograce))
	})

	_, err := tc.KubeClient.Apps().StatefulSets(OpenshiftLoggingNS).Create(app)
	if err != nil {
		return err
	}

	return tc.waitForStatefulSet(OpenshiftLoggingNS, app.GetName(), defaultRetryInterval, defaultTimeout)
}

func (tc *E2ETestFramework) createLokiService() error {
	svc := loki.NewService(OpenshiftLoggingNS)

	tc.AddCleanup(func() error {
		var zerograce int64
		return tc.KubeClient.CoreV1().Services(OpenshiftLoggingNS).Delete(svc.GetName(), metav1.NewDeleteOptions(zerograce))
	})

	if _, err := tc.KubeClient.CoreV1().Services(OpenshiftLoggingNS).Create(svc); err != nil {
		return err
	}

	return nil
}

func (tc *E2ETestFramework) createLokiQueryUtil() error {
	cm := loki.NewLokiUtilConfigMap(OpenshiftLoggingNS)
	tc.AddCleanup(func() error {
		return tc.KubeClient.Core().ConfigMaps(OpenshiftLoggingNS).Delete(cm.GetName(), nil)
	})
	_, err := tc.KubeClient.Core().ConfigMaps(OpenshiftLoggingNS).Create(cm)
	if err != nil {
		return err
	}

	pod := loki.NewLokiUtilPod(OpenshiftLoggingNS)
	tc.AddCleanup(func() error {
		return tc.KubeClient.Apps().Deployments(OpenshiftLoggingNS).Delete(pod.GetName(), nil)
	})
	pod, err = tc.KubeClient.Core().Pods(OpenshiftLoggingNS).Create(pod)
	if err != nil {
		return err
	}
	return tc.waitForPod(OpenshiftLoggingNS, loki.UtilName, defaultRetryInterval, defaultTimeout)
}

func (tc *E2ETestFramework) DeployLokiClusterReceiver() error {
	receiver := &lokiReceiver{
		tc:   tc,
		mode: clusterMode,
	}
	tc.LogStore = receiver

	if err := tc.createLokiClusterConfigMap(); err != nil {
		return err
	}

	// Etcd StatefulSet
	if err := tc.createLokiKvStore(); err != nil {
		return err
	}

	// Ingester Deplopyment
	if err := tc.createLokiIngester(); err != nil {
		return err
	}

	// Distributor Deployment
	if err := tc.createLokiDistributor(); err != nil {
		return err
	}

	// Querier Deployment
	if err := tc.createLokiQuerier(); err != nil {
		return err
	}

	// Query-Frontend Deployment
	if err := tc.createLokiQueryFrontend(); err != nil {
		return err
	}

	// Table-Manager Deployment
	if err := tc.createLokiTableManager(); err != nil {
		return err
	}

	// Helper pod to query loki
	if err := tc.createLokiQueryUtil(); err != nil {
		return err
	}

	return nil
}

func (tc *E2ETestFramework) createLokiClusterConfigMap() error {
	cm := loki.NewClusterConfigMap(OpenshiftLoggingNS)

	tc.AddCleanup(func() error {
		var zerograce int64
		return tc.KubeClient.Core().ConfigMaps(OpenshiftLoggingNS).Delete(cm.GetName(), metav1.NewDeleteOptions(zerograce))
	})

	if _, err := tc.KubeClient.Core().ConfigMaps(OpenshiftLoggingNS).Create(cm); err != nil {
		return err
	}

	return nil
}

func (tc *E2ETestFramework) createLokiKvStore() error {
	// Etcd Scripts
	cm := loki.NewEtcdConfigMap(OpenshiftLoggingNS)
	tc.AddCleanup(func() error {
		return tc.KubeClient.Core().ConfigMaps(OpenshiftLoggingNS).Delete(cm.GetName(), nil)
	})

	_, err := tc.KubeClient.Core().ConfigMaps(OpenshiftLoggingNS).Create(cm)
	if err != nil {
		return err
	}

	// Etcd Client Service
	svc := loki.NewEtcdService(OpenshiftLoggingNS)
	tc.AddCleanup(func() error {
		var zerograce int64
		return tc.KubeClient.CoreV1().Services(OpenshiftLoggingNS).Delete(svc.GetName(), metav1.NewDeleteOptions(zerograce))
	})

	if _, err := tc.KubeClient.CoreV1().Services(OpenshiftLoggingNS).Create(svc); err != nil {
		return err
	}

	// Etcd StatefulSet
	app := loki.NewEtcdStatefulSet(OpenshiftLoggingNS)
	tc.AddCleanup(func() error {
		var zerograce int64
		return tc.KubeClient.Apps().StatefulSets(OpenshiftLoggingNS).Delete(app.GetName(), metav1.NewDeleteOptions(zerograce))
	})

	_, err = tc.KubeClient.Apps().StatefulSets(OpenshiftLoggingNS).Create(app)
	if err != nil {
		return err
	}

	return tc.waitForStatefulSet(OpenshiftLoggingNS, app.GetName(), defaultRetryInterval, defaultTimeout)
}

func (tc *E2ETestFramework) createLokiDistributor() error {
	// Loki Distributor Service
	svc := loki.NewDistributorService(OpenshiftLoggingNS)
	tc.AddCleanup(func() error {
		var zerograce int64
		return tc.KubeClient.CoreV1().Services(OpenshiftLoggingNS).Delete(svc.GetName(), metav1.NewDeleteOptions(zerograce))
	})

	if _, err := tc.KubeClient.CoreV1().Services(OpenshiftLoggingNS).Create(svc); err != nil {
		return err
	}

	// Loki Distributor Deployment
	app := loki.NewDistributorDeployment(OpenshiftLoggingNS)
	tc.AddCleanup(func() error {
		var zerograce int64
		return tc.KubeClient.Apps().Deployments(OpenshiftLoggingNS).Delete(app.GetName(), metav1.NewDeleteOptions(zerograce))
	})

	_, err = tc.KubeClient.Apps().Deployments(OpenshiftLoggingNS).Create(app)
	if err != nil {
		return err
	}

	return tc.waitForDeployment(OpenshiftLoggingNS, app.GetName(), defaultRetryInterval, defaultTimeout)
}

func (tc *E2ETestFramework) createLokiIngester() error {
	// Loki Ingester Service
	svc := loki.NewIngesterService(OpenshiftLoggingNS)
	tc.AddCleanup(func() error {
		var zerograce int64
		return tc.KubeClient.CoreV1().Services(OpenshiftLoggingNS).Delete(svc.GetName(), metav1.NewDeleteOptions(zerograce))
	})

	if _, err := tc.KubeClient.CoreV1().Services(OpenshiftLoggingNS).Create(svc); err != nil {
		return err
	}

	// Loki Ingester Deployment
	app := loki.NewIngesterDeployment(OpenshiftLoggingNS)
	tc.AddCleanup(func() error {
		var zerograce int64
		return tc.KubeClient.Apps().Deployments(OpenshiftLoggingNS).Delete(app.GetName(), metav1.NewDeleteOptions(zerograce))
	})

	_, err = tc.KubeClient.Apps().Deployments(OpenshiftLoggingNS).Create(app)
	if err != nil {
		return err
	}

	return tc.waitForDeployment(OpenshiftLoggingNS, app.GetName(), defaultRetryInterval, defaultTimeout)
}

func (tc *E2ETestFramework) createLokiQuerier() error {
	// Loki Querier Service
	svc := loki.NewQuerierService(OpenshiftLoggingNS)
	tc.AddCleanup(func() error {
		var zerograce int64
		return tc.KubeClient.CoreV1().Services(OpenshiftLoggingNS).Delete(svc.GetName(), metav1.NewDeleteOptions(zerograce))
	})

	if _, err := tc.KubeClient.CoreV1().Services(OpenshiftLoggingNS).Create(svc); err != nil {
		return err
	}

	// Loki Querier Deployment
	app := loki.NewQuerierDeployment(OpenshiftLoggingNS)
	tc.AddCleanup(func() error {
		var zerograce int64
		return tc.KubeClient.Apps().Deployments(OpenshiftLoggingNS).Delete(app.GetName(), metav1.NewDeleteOptions(zerograce))
	})

	_, err = tc.KubeClient.Apps().Deployments(OpenshiftLoggingNS).Create(app)
	if err != nil {
		return err
	}

	return tc.waitForDeployment(OpenshiftLoggingNS, app.GetName(), defaultRetryInterval, defaultTimeout)
}

func (tc *E2ETestFramework) createLokiQueryFrontend() error {
	// Loki Query Frontend Service
	svc := loki.NewQueryFrontendService(OpenshiftLoggingNS)
	tc.AddCleanup(func() error {
		var zerograce int64
		return tc.KubeClient.CoreV1().Services(OpenshiftLoggingNS).Delete(svc.GetName(), metav1.NewDeleteOptions(zerograce))
	})

	if _, err := tc.KubeClient.CoreV1().Services(OpenshiftLoggingNS).Create(svc); err != nil {
		return err
	}

	// Loki Query Frontend Deployment
	app := loki.NewQueryFrontendDeployment(OpenshiftLoggingNS)
	tc.AddCleanup(func() error {
		var zerograce int64
		return tc.KubeClient.Apps().Deployments(OpenshiftLoggingNS).Delete(app.GetName(), metav1.NewDeleteOptions(zerograce))
	})

	_, err = tc.KubeClient.Apps().Deployments(OpenshiftLoggingNS).Create(app)
	if err != nil {
		return err
	}

	return tc.waitForDeployment(OpenshiftLoggingNS, app.GetName(), defaultRetryInterval, defaultTimeout)
}

func (tc *E2ETestFramework) createLokiTableManager() error {
	// Loki Table Manager Service
	svc := loki.NewTableManagerService(OpenshiftLoggingNS)
	tc.AddCleanup(func() error {
		var zerograce int64
		return tc.KubeClient.CoreV1().Services(OpenshiftLoggingNS).Delete(svc.GetName(), metav1.NewDeleteOptions(zerograce))
	})

	if _, err := tc.KubeClient.CoreV1().Services(OpenshiftLoggingNS).Create(svc); err != nil {
		return err
	}

	// Loki Query Frontend Deployment
	app := loki.NewTableManagerDeployment(OpenshiftLoggingNS)
	tc.AddCleanup(func() error {
		var zerograce int64
		return tc.KubeClient.Apps().Deployments(OpenshiftLoggingNS).Delete(app.GetName(), metav1.NewDeleteOptions(zerograce))
	})

	_, err = tc.KubeClient.Apps().Deployments(OpenshiftLoggingNS).Create(app)
	if err != nil {
		return err
	}

	return tc.waitForDeployment(OpenshiftLoggingNS, app.GetName(), defaultRetryInterval, defaultTimeout)
}
