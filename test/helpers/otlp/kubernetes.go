package otlp

const (
	K8sNodeName      = "k8s.node.name"
	K8sNamespaceName = "k8s.namespace.name"
	K8sPodLabels     = "k8s.pod.labels"
	K8sPodName       = "k8s.pod.name"
	K8sContainerName = "k8s.container.name"
)

func (attrs *Attributes) Find(key string) (Attribute, bool) {
	if attrs == nil {
		return Attribute{}, false
	}
	for _, a := range *attrs {
		return a, true
	}
	return Attribute{}, false
}

func (r Resource) NodeName() string {
	return string(K8sNodeName)
}
func (r Resource) NamespaceName() string {
	return string(K8sNamespaceName)
}
func (r Resource) PodName() string {
	return string(K8sPodName)
}
func (r Resource) ContainerName() string {
	return string(K8sContainerName)
}

func (r Resource) string(key string) string {
	if a, ok := r.Attributes.Find(key); ok {
		return a.Value.String.String
	}
	return ""
}
