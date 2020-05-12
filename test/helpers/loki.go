package helpers

import (
	"fmt"
	"strings"
	"time"

	"github.com/openshift/cluster-logging-operator/pkg/logger"
	"github.com/openshift/cluster-logging-operator/test/helpers/loki"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type lokiReceiver struct {
	deployment *apps.StatefulSet
	tc         *E2ETestFramework
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
	return loki.ClusterLocalEndpoint(OpenshiftLoggingNS)

}

func (tc *E2ETestFramework) lokiLogs(indexName string) (*loki.Response, error) {
	pod, err := tc.KubeClient.Core().Pods(OpenshiftLoggingNS).Get(loki.QuerierName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	logger.Debugf("Pod %s", pod.GetName())

	indexName = fmt.Sprintf("%swrite", indexName)
	cmd := []string{"/bin/sh", "/data/loki_util", tc.LogStore.ClusterLocalEndpoint(), indexName}
	stdout, err := tc.PodExec(OpenshiftLoggingNS, loki.QuerierName, loki.QuerierName, cmd)
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

	app, err := tc.createLokiStatefulSet()
	if err != nil {
		return err
	}

	if err := tc.createLokiService(); err != nil {
		return err
	}

	if err := tc.createLokiQuerier(); err != nil {
		return err
	}

	receiver := &lokiReceiver{
		tc:         tc,
		deployment: app,
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

func (tc *E2ETestFramework) createLokiStatefulSet() (*apps.StatefulSet, error) {
	app := loki.NewStatefulSet(OpenshiftLoggingNS)

	tc.AddCleanup(func() error {
		var zerograce int64
		return tc.KubeClient.Apps().StatefulSets(OpenshiftLoggingNS).Delete(app.GetName(), metav1.NewDeleteOptions(zerograce))
	})

	app, err := tc.KubeClient.Apps().StatefulSets(OpenshiftLoggingNS).Create(app)
	if err != nil {
		return nil, err
	}

	return app, tc.waitForStatefulSet(OpenshiftLoggingNS, app.GetName(), defaultRetryInterval, defaultTimeout)
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

func (tc *E2ETestFramework) createLokiQuerier() error {
	cm := loki.NewQuerierConfigMap(OpenshiftLoggingNS)
	tc.AddCleanup(func() error {
		return tc.KubeClient.Core().ConfigMaps(OpenshiftLoggingNS).Delete(cm.GetName(), nil)
	})
	_, err := tc.KubeClient.Core().ConfigMaps(OpenshiftLoggingNS).Create(cm)
	if err != nil {
		return err
	}

	pod := loki.NewQuerierPod(OpenshiftLoggingNS)
	tc.AddCleanup(func() error {
		return tc.KubeClient.Apps().Deployments(OpenshiftLoggingNS).Delete(pod.GetName(), nil)
	})
	pod, err = tc.KubeClient.Core().Pods(OpenshiftLoggingNS).Create(pod)
	if err != nil {
		return err
	}
	return tc.waitForPod(OpenshiftLoggingNS, loki.QuerierName, defaultRetryInterval, defaultTimeout)
}
