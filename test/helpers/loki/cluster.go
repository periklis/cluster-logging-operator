package loki

import (
    "fmt"

    "github.com/openshift/cluster-logging-operator/pkg/k8shandler"
    apps "k8s.io/api/apps/v1"
    v1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/resource"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/util/intstr"
)

const (
    // Service ports
    lokiComponentMetricsPort = 80
    lokiComponentGrpcPort    = 9095

    // Components
    lokiDistributorName   = "loki-dst"
    lokiIngesterName      = "loki-ing"
    lokiQuerierName       = "loki-qrr"
    lokiQueryFrontendName = "loki-qrf"
    lokiTableManagerName  = "loki-tmg"
    lokiConfigName        = "loki-cm"

    // Container Config
    lokiImageName = "quay.io/periklis/loki:flush-boltdb-to-object-store-795555c"
    lokiLogLevel  = "debug"
)

var (
    replicas    int32 = 2
    termination int64 = 30
)

func NewDistributorDeployment(namespace string) *apps.Deployment {
    return &apps.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      lokiDistributorName,
            Namespace: namespace,
            Labels: map[string]string{
                "app":       lokiDistributorName,
                "component": lokiComponent,
                "provider":  lokiProvider,
            },
        },
        Spec: apps.DeploymentSpec{
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{
                    "app": lokiDistributorName,
                },
            },
            Replicas: &replicas,
            Template: v1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{
                        "app":       lokiDistributorName,
                        "component": lokiComponent,
                        "provider":  lokiProvider,
                    },
                },
                Spec: v1.PodSpec{
                    TerminationGracePeriodSeconds: &termination,
                    Containers: []v1.Container{
                        {
                            Name:            lokiDistributorName,
                            Image:           lokiImageName,
                            ImagePullPolicy: v1.PullIfNotPresent,
                            Args: []string{
                                "-config.file=/etc/loki/config/config.yaml",
                                "-limits.per-user-override-config=/etc/loki/config/overrides.yaml",
                                fmt.Sprintf("-log.level=%s", lokiLogLevel),
                                "-target=distributor",
                            },
                            Ports: []v1.ContainerPort{
                                {
                                    ContainerPort: lokiComponentMetricsPort,
                                    Name:          fmt.Sprintf("%s-http", lokiDistributorName),
                                },
                                {
                                    ContainerPort: lokiComponentGrpcPort,
                                    Name:          fmt.Sprintf("%s-grpc", lokiDistributorName),
                                },
                            },
                            ReadinessProbe: &v1.Probe{
                                Handler: v1.Handler{
                                    HTTPGet: &v1.HTTPGetAction{
                                        Path: "/ready",
                                        Port: intstr.FromInt(80),
                                    },
                                },
                                InitialDelaySeconds: 15,
                                TimeoutSeconds:      1,
                            },
                            Resources: v1.ResourceRequirements{
                                Requests: v1.ResourceList{
                                    v1.ResourceCPU:    resource.MustParse("500m"),
                                    v1.ResourceMemory: resource.MustParse("512Mi"),
                                },
                                Limits: v1.ResourceList{
                                    v1.ResourceCPU:    resource.MustParse("1000m"),
                                    v1.ResourceMemory: resource.MustParse("1Gi"),
                                },
                            },
                            VolumeMounts: []v1.VolumeMount{
                                {
                                    Name:      "config",
                                    MountPath: "/etc/loki/config",
                                },
                                {
                                    Name:      "storage",
                                    MountPath: "/data",
                                },
                            },
                        },
                    },
                    Volumes: []v1.Volume{
                        {
                            Name: "config",
                            VolumeSource: v1.VolumeSource{
                                ConfigMap: &v1.ConfigMapVolumeSource{
                                    LocalObjectReference: v1.LocalObjectReference{
                                        Name: lokiConfigName,
                                    },
                                },
                            },
                        },
                        {
                            Name: "storage",
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

func NewDistributorService(namespace string) *v1.Service {
    return newComponentService(lokiDistributorName, namespace)
}

func NewIngesterDeployment(namespace string) *apps.Deployment {
    return &apps.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      lokiIngesterName,
            Namespace: namespace,
            Labels: map[string]string{
                "app":       lokiIngesterName,
                "component": lokiComponent,
                "provider":  lokiProvider,
            },
        },
        Spec: apps.DeploymentSpec{
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{
                    "app": lokiIngesterName,
                },
            },
            Replicas: &replicas,
            Template: v1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{
                        "app":       lokiIngesterName,
                        "component": lokiComponent,
                        "provider":  lokiProvider,
                    },
                },
                Spec: v1.PodSpec{
                    TerminationGracePeriodSeconds: &termination,
                    Containers: []v1.Container{
                        {
                            Name:            lokiIngesterName,
                            Image:           lokiImageName,
                            ImagePullPolicy: v1.PullIfNotPresent,
                            Args: []string{
                                "-config.file=/etc/loki/config/config.yaml",
                                "-limits.per-user-override-config=/etc/loki/config/overrides.yaml",
                                fmt.Sprintf("-log.level=%s", lokiLogLevel),
                                "-target=ingester",
                            },
                            Ports: []v1.ContainerPort{
                                {
                                    ContainerPort: lokiComponentMetricsPort,
                                    Name:          fmt.Sprintf("%s-http", lokiIngesterName),
                                },
                                {
                                    ContainerPort: lokiComponentGrpcPort,
                                    Name:          fmt.Sprintf("%s-grpc", lokiIngesterName),
                                },
                            },
                            Resources: v1.ResourceRequirements{
                                Requests: v1.ResourceList{
                                    v1.ResourceCPU:    resource.MustParse("1000m"),
                                    v1.ResourceMemory: resource.MustParse("2Gi"),
                                },
                                Limits: v1.ResourceList{
                                    v1.ResourceCPU:    resource.MustParse("2000m"),
                                    v1.ResourceMemory: resource.MustParse("4Gi"),
                                },
                            },
                            ReadinessProbe: &v1.Probe{
                                Handler: v1.Handler{
                                    HTTPGet: &v1.HTTPGetAction{
                                        Path: "/ready",
                                        Port: intstr.FromInt(80),
                                    },
                                },
                                InitialDelaySeconds: 15,
                                TimeoutSeconds:      1,
                            },
                            VolumeMounts: []v1.VolumeMount{
                                {
                                    Name:      "config",
                                    MountPath: "/etc/loki/config",
                                },
                                {
                                    Name:      "storage",
                                    MountPath: "/data",
                                },
                            },
                        },
                    },
                    Volumes: []v1.Volume{
                        {
                            Name: "config",
                            VolumeSource: v1.VolumeSource{
                                ConfigMap: &v1.ConfigMapVolumeSource{
                                    LocalObjectReference: v1.LocalObjectReference{
                                        Name: lokiConfigName,
                                    },
                                },
                            },
                        },
                        {
                            Name: "storage",
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

func NewIngesterService(namespace string) *v1.Service {
    return newComponentService(lokiIngesterName, namespace)
}

func NewQuerierDeployment(namespace string) *apps.Deployment {
    return &apps.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      lokiQuerierName,
            Namespace: namespace,
            Labels: map[string]string{
                "app":       lokiQuerierName,
                "component": lokiComponent,
                "provider":  lokiProvider,
            },
        },
        Spec: apps.DeploymentSpec{
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{
                    "app": lokiQuerierName,
                },
            },
            Replicas: &replicas,
            Template: v1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{
                        "app":       lokiQuerierName,
                        "component": lokiComponent,
                        "provider":  lokiProvider,
                    },
                },
                Spec: v1.PodSpec{
                    TerminationGracePeriodSeconds: &termination,
                    Containers: []v1.Container{
                        {
                            Name:            lokiQuerierName,
                            Image:           lokiImageName,
                            ImagePullPolicy: v1.PullIfNotPresent,
                            Args: []string{
                                "-config.file=/etc/loki/config/config.yaml",
                                "-limits.per-user-override-config=/etc/loki/config/overrides.yaml",
                                fmt.Sprintf("-log.level=%s", lokiLogLevel),
                                "-target=querier",
                            },
                            Ports: []v1.ContainerPort{
                                {
                                    ContainerPort: lokiComponentMetricsPort,
                                    Name:          fmt.Sprintf("%s-http", lokiQuerierName),
                                },
                                {
                                    ContainerPort: lokiComponentGrpcPort,
                                    Name:          fmt.Sprintf("%s-grpc", lokiQuerierName),
                                },
                            },
                            ReadinessProbe: &v1.Probe{
                                Handler: v1.Handler{
                                    HTTPGet: &v1.HTTPGetAction{
                                        Path: "/ready",
                                        Port: intstr.FromInt(80),
                                    },
                                },
                                InitialDelaySeconds: 15,
                                TimeoutSeconds:      1,
                            },
                            VolumeMounts: []v1.VolumeMount{
                                {
                                    Name:      "config",
                                    MountPath: "/etc/loki/config",
                                },
                                {
                                    Name:      "storage",
                                    MountPath: "/data",
                                },
                            },
                        },
                    },
                    Volumes: []v1.Volume{
                        {
                            Name: "config",
                            VolumeSource: v1.VolumeSource{
                                ConfigMap: &v1.ConfigMapVolumeSource{
                                    LocalObjectReference: v1.LocalObjectReference{
                                        Name: lokiConfigName,
                                    },
                                },
                            },
                        },
                        {
                            Name: "storage",
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

func NewQuerierService(namespace string) *v1.Service {
    return newComponentService(lokiQuerierName, namespace)
}

func NewQueryFrontendDeployment(namespace string) *apps.Deployment {
    return &apps.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      lokiQueryFrontendName,
            Namespace: namespace,
            Labels: map[string]string{
                "app":       lokiQueryFrontendName,
                "component": lokiComponent,
                "provider":  lokiProvider,
            },
        },
        Spec: apps.DeploymentSpec{
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{
                    "app": lokiQueryFrontendName,
                },
            },
            Replicas: &replicas,
            Template: v1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{
                        "app":       lokiQueryFrontendName,
                        "component": lokiComponent,
                        "provider":  lokiProvider,
                    },
                },
                Spec: v1.PodSpec{
                    TerminationGracePeriodSeconds: &termination,
                    Containers: []v1.Container{
                        {
                            Name:            lokiQueryFrontendName,
                            Image:           lokiImageName,
                            ImagePullPolicy: v1.PullIfNotPresent,
                            Args: []string{
                                "-config.file=/etc/loki/config/config.yaml",
                                "-limits.per-user-override-config=/etc/loki/config/overrides.yaml",
                                fmt.Sprintf("-log.level=%s", lokiLogLevel),
                                "-target=query-frontend",
                            },
                            Ports: []v1.ContainerPort{
                                {
                                    ContainerPort: lokiComponentMetricsPort,
                                    Name:          fmt.Sprintf("%s-http", lokiQueryFrontendName),
                                },
                                {
                                    ContainerPort: lokiComponentGrpcPort,
                                    Name:          fmt.Sprintf("%s-grpc", lokiQueryFrontendName),
                                },
                            },
                            Resources: v1.ResourceRequirements{
                                Requests: v1.ResourceList{
                                    v1.ResourceCPU:    resource.MustParse("1000m"),
                                    v1.ResourceMemory: resource.MustParse("600Mi"),
                                },
                                Limits: v1.ResourceList{
                                    v1.ResourceCPU:    resource.MustParse("2000m"),
                                    v1.ResourceMemory: resource.MustParse("1200Mi"),
                                },
                            },
                            VolumeMounts: []v1.VolumeMount{
                                {
                                    Name:      "config",
                                    MountPath: "/etc/loki/config",
                                },
                                {
                                    Name:      "storage",
                                    MountPath: "/data",
                                },
                            },
                        },
                    },
                    Volumes: []v1.Volume{
                        {
                            Name: "config",
                            VolumeSource: v1.VolumeSource{
                                ConfigMap: &v1.ConfigMapVolumeSource{
                                    LocalObjectReference: v1.LocalObjectReference{
                                        Name: lokiConfigName,
                                    },
                                },
                            },
                        },
                        {
                            Name: "storage",
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

func NewQueryFrontendService(namespace string) *v1.Service {
    return newComponentService(lokiQueryFrontendName, namespace)
}

func NewTableManagerDeployment(namespace string) *apps.Deployment {
    return &apps.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      lokiTableManagerName,
            Namespace: namespace,
            Labels: map[string]string{
                "app":       lokiTableManagerName,
                "component": lokiComponent,
                "provider":  lokiProvider,
            },
        },
        Spec: apps.DeploymentSpec{
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{
                    "app": lokiTableManagerName,
                },
            },
            Replicas: &replicas,
            Template: v1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{
                        "app":       lokiTableManagerName,
                        "component": lokiComponent,
                        "provider":  lokiProvider,
                    },
                },
                Spec: v1.PodSpec{
                    TerminationGracePeriodSeconds: &termination,
                    Containers: []v1.Container{
                        {
                            Name:            lokiTableManagerName,
                            Image:           lokiImageName,
                            ImagePullPolicy: v1.PullIfNotPresent,
                            Args: []string{
                                "-config.file=/etc/loki/config/config.yaml",
                                "-limits.per-user-override-config=/etc/loki/config/overrides.yaml",
                                fmt.Sprintf("-log.level=%s", lokiLogLevel),
                                "-target=table-manager",
                            },
                            Ports: []v1.ContainerPort{
                                {
                                    ContainerPort: lokiComponentMetricsPort,
                                    Name:          fmt.Sprintf("%s-http", lokiTableManagerName),
                                },
                                {
                                    ContainerPort: lokiComponentGrpcPort,
                                    Name:          fmt.Sprintf("%s-grpc", lokiTableManagerName),
                                },
                            },
                            Resources: v1.ResourceRequirements{
                                Requests: v1.ResourceList{
                                    v1.ResourceCPU:    resource.MustParse("100m"),
                                    v1.ResourceMemory: resource.MustParse("100Mi"),
                                },
                                Limits: v1.ResourceList{
                                    v1.ResourceCPU:    resource.MustParse("200m"),
                                    v1.ResourceMemory: resource.MustParse("200Mi"),
                                },
                            },
                            ReadinessProbe: &v1.Probe{
                                Handler: v1.Handler{
                                    HTTPGet: &v1.HTTPGetAction{
                                        Path: "/ready",
                                        Port: intstr.FromInt(80),
                                    },
                                },
                                InitialDelaySeconds: 15,
                                TimeoutSeconds:      1,
                            },
                            VolumeMounts: []v1.VolumeMount{
                                {
                                    Name:      "config",
                                    MountPath: "/etc/loki/config",
                                },
                                {
                                    Name:      "storage",
                                    MountPath: "/data",
                                },
                            },
                        },
                    },
                    Volumes: []v1.Volume{
                        {
                            Name: "config",
                            VolumeSource: v1.VolumeSource{
                                ConfigMap: &v1.ConfigMapVolumeSource{
                                    LocalObjectReference: v1.LocalObjectReference{
                                        Name: lokiConfigName,
                                    },
                                },
                            },
                        },
                        {
                            Name: "storage",
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

func NewTableManagerService(namespace string) *v1.Service {
    return newComponentService(lokiTableManagerName, namespace)
}

func NewClusterConfigMap(namespace string) *v1.ConfigMap {
    data := map[string]string{
        "config.yaml":    lokiClusterConfig,
        "overrides.yaml": lokiClusterOverrides,
    }
    return k8shandler.NewConfigMap(lokiConfigName, namespace, data)
}

func newComponentService(name, namespace string) *v1.Service {
    ports := []v1.ServicePort{
        {
            Name: fmt.Sprintf("%s-http", name),
            Port: lokiComponentMetricsPort,
        },
        {
            Name: fmt.Sprintf("%s-grpc", name),
            Port: lokiComponentGrpcPort,
        },
    }

    svc := k8shandler.NewService(name, namespace, lokiComponent, ports)
    svc.Spec.Selector["app"] = name
    return svc
}
