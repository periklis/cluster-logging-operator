package loki

import (
    "fmt"

    "github.com/openshift/cluster-logging-operator/pkg/k8shandler"
    apps "k8s.io/api/apps/v1"
    v1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/resource"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
    lokiKvStoreName = "loki-kvstore"

    etcdClientPort = 2379
    etcdPeerPort   = 2380
)

func NewEtcdStatefulSet(namespace string) *apps.StatefulSet {
    var (
        replicas    int32 = 1
        termination int64 = 30
        defaultMode int32 = 0777
    )
    return &apps.StatefulSet{
        ObjectMeta: metav1.ObjectMeta{
            Name:      lokiKvStoreName,
            Namespace: namespace,
            Labels: map[string]string{
                "app":       lokiKvStoreName,
                "component": lokiComponent,
                "provider":  lokiProvider,
            },
        },
        Spec: apps.StatefulSetSpec{
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{
                    "app": lokiKvStoreName,
                },
            },
            Replicas:    &replicas,
            ServiceName: lokiKvStoreName,
            Template: v1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{
                        "app":       lokiKvStoreName,
                        "component": lokiComponent,
                        "provider":  lokiProvider,
                    },
                },
                Spec: v1.PodSpec{
                    TerminationGracePeriodSeconds: &termination,
                    Containers: []v1.Container{
                        {
                            Name:            lokiKvStoreName,
                            Image:           "k8s.gcr.io/etcd-amd64:3.2.26",
                            ImagePullPolicy: v1.PullIfNotPresent,
                            Ports: []v1.ContainerPort{
                                {
                                    ContainerPort: etcdClientPort,
                                    Name:          "etcd-client",
                                },
                                {
                                    ContainerPort: etcdPeerPort,
                                    Name:          "etcd-server",
                                },
                            },
                            Resources: v1.ResourceRequirements{
                                Requests: v1.ResourceList{
                                    v1.ResourceCPU:    resource.MustParse("100m"),
                                    v1.ResourceMemory: resource.MustParse("128Mi"),
                                },
                                Limits: v1.ResourceList{
                                    v1.ResourceCPU:    resource.MustParse("200m"),
                                    v1.ResourceMemory: resource.MustParse("256Mi"),
                                },
                            },
                            Env: []v1.EnvVar{
                                {
                                    Name:  "ETCDCTL_API",
                                    Value: "3",
                                },
                                {
                                    Name:  "INITIAL_CLUSTER_SIZE",
                                    Value: fmt.Sprintf("%d", replicas),
                                },
                                {
                                    Name:  "SET_NAME",
                                    Value: lokiKvStoreName,
                                },
                            },
                            VolumeMounts: []v1.VolumeMount{
                                {
                                    Name:      "datadir",
                                    MountPath: "/var/run/etcd",
                                },
                                {
                                    Name:      "scripts",
                                    MountPath: "/scripts",
                                },
                            },
                            Lifecycle: &v1.Lifecycle{
                                PreStop: &v1.Handler{
                                    Exec: &v1.ExecAction{
                                        Command: []string{
                                            "/bin/sh",
                                            "-ec",
                                            "/scripts/etcd-pre-stop.sh",
                                        },
                                    },
                                },
                            },
                            Command: []string{
                                "/bin/sh",
                                "-ec",
                                "/scripts/etcd-server.sh",
                            },
                        },
                    },
                    Volumes: []v1.Volume{
                        {
                            Name: "scripts",
                            VolumeSource: v1.VolumeSource{
                                ConfigMap: &v1.ConfigMapVolumeSource{
                                    LocalObjectReference: v1.LocalObjectReference{
                                        Name: lokiKvStoreName,
                                    },
                                    DefaultMode: &defaultMode,
                                },
                            },
                        },
                        {
                            Name: "datadir",
                            VolumeSource: v1.VolumeSource{
                                EmptyDir: &v1.EmptyDirVolumeSource{
                                    Medium: v1.StorageMediumMemory,
                                },
                            },
                        },
                    },
                },
            },
        },
    }
}

func NewEtcdService(namespace string) *v1.Service {
    ports := []v1.ServicePort{
        {
            Port: etcdPeerPort,
            Name: "etcd-server",
        },
        {
            Port: etcdClientPort,
            Name: "etcd-client",
        },
    }

    return k8shandler.NewService(lokiKvStoreName, namespace, lokiComponent, ports)
}

func NewEtcdConfigMap(namespace string) *v1.ConfigMap {
    data := map[string]string{
        "etcd-pre-stop.sh": etcdPreStopCmd,
        "etcd-server.sh":   etcdCmd,
    }
    return k8shandler.NewConfigMap(lokiKvStoreName, namespace, data)
}
