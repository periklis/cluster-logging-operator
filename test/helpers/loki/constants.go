package loki

import "fmt"

const (
	UtilName = "loki-util"
)

func SingleServerEndpoint(namespace string) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local:%d", lokiSingleServerName, namespace, lokiSingleServerPort)
}

func ClusterEndpoint(namespace string) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", lokiQuerierName, namespace)
}
