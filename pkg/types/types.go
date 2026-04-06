package types

import "time"

// Pod represents a containerized application unit
type Pod struct {
	Metadata   ObjectMeta            `json:"metadata"`
	Spec       PodSpec               `json:"spec"`
	Status     PodStatus             `json:"status"`
	CreatedAt  time.Time             `json:"createdAt"`
	UpdatedAt  time.Time             `json:"updatedAt"`
}

// ObjectMeta contains metadata for Kubernetes objects
type ObjectMeta struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels"`
	UID       string            `json:"uid"`
}

// PodSpec defines the specification for a Pod
type PodSpec struct {
	Containers    []Container           `json:"containers"`
	RestartPolicy RestartPolicy         `json:"restartPolicy"`
	NodeName      string                `json:"nodeName,omitempty"`
	Resources     ResourceRequirements  `json:"resources,omitempty"`
}

// Container defines a container within a Pod
type Container struct {
	Name            string               `json:"name"`
	Image           string               `json:"image"`
	Ports           []ContainerPort      `json:"ports,omitempty"`
	Env             []EnvVar             `json:"env,omitempty"`
	Resources       ResourceRequirements `json:"resources,omitempty"`
	RestartPolicy   RestartPolicy        `json:"restartPolicy"`
}

// ContainerPort defines a port exposed by a container
type ContainerPort struct {
	Name          string `json:"name,omitempty"`
	ContainerPort int32  `json:"containerPort"`
	Protocol      string `json:"protocol,omitempty"`
}

// EnvVar defines an environment variable
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ResourceRequirements defines resource limits and requests
type ResourceRequirements struct {
	Requests ResourceList `json:"requests,omitempty"`
	Limits   ResourceList `json:"limits,omitempty"`
}

// ResourceList is a map of resource quantities
type ResourceList map[string]string

// RestartPolicy defines when a container should be restarted
type RestartPolicy string

const (
	RestartPolicyAlways    RestartPolicy = "Always"
	RestartPolicyOnFailure RestartPolicy = "OnFailure"
	RestartPolicyNever     RestartPolicy = "Never"
)

// PodStatus represents the current status of a Pod
type PodStatus struct {
	Phase             PodPhase     `json:"phase"`
	ContainerStatuses []ContainerStatus `json:"containerStatuses,omitempty"`
	Message           string       `json:"message,omitempty"`
	HostIP            string       `json:"hostIP,omitempty"`
	PodIP             string       `json:"podIP,omitempty"`
}

// PodPhase represents the phase of a Pod
type PodPhase string

const (
	PodPending   PodPhase = "Pending"
	PodRunning   PodPhase = "Running"
	PodSucceeded PodPhase = "Succeeded"
	PodFailed    PodPhase = "Failed"
	PodUnknown   PodPhase = "Unknown"
)

// ContainerStatus represents the status of a container
type ContainerStatus struct {
	Name        string `json:"name"`
	ContainerID string `json:"containerID,omitempty"`
	Ready       bool   `json:"ready"`
	Running     bool   `json:"running"`
	State       string `json:"state"`
}

// Node represents a worker node in the cluster
type Node struct {
	Metadata      ObjectMeta        `json:"metadata"`
	Status        NodeStatus        `json:"status"`
	Capacity      ResourceList      `json:"capacity"`
	Allocatable   ResourceList      `json:"allocatable"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
	LastHeartbeat time.Time         `json:"lastHeartbeat"`
}

// NodeStatus represents the status of a Node
type NodeStatus struct {
	Phase   NodePhase `json:"phase"`
	Ready   bool      `json:"ready"`
	Message string    `json:"message,omitempty"`
}

// NodePhase represents the phase of a Node
type NodePhase string

const (
	NodePending   NodePhase = "Pending"
	NodeReady     NodePhase = "Ready"
	NodeNotReady  NodePhase = "NotReady"
	NodeUnknown   NodePhase = "Unknown"
)

// Service represents a network service
type Service struct {
	Metadata ObjectMeta   `json:"metadata"`
	Spec     ServiceSpec  `json:"spec"`
	Status   ServiceStatus `json:"status"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
}

// ServiceSpec defines the specification for a Service
type ServiceSpec struct {
	Selector map[string]string `json:"selector"`
	Ports    []ServicePort     `json:"ports"`
	Type     ServiceType       `json:"type"`
}

// ServicePort defines a port for a Service
type ServicePort struct {
	Name       string `json:"name,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	Port       int32  `json:"port"`
	TargetPort int32  `json:"targetPort"`
}

// ServiceType represents the type of Service
type ServiceType string

const (
	ServiceTypeClusterIP   ServiceType = "ClusterIP"
	ServiceTypeNodePort    ServiceType = "NodePort"
	ServiceTypeLoadBalancer ServiceType = "LoadBalancer"
)

// ServiceStatus represents the status of a Service
type ServiceStatus struct {
	ClusterIP string `json:"clusterIP,omitempty"`
	Message   string `json:"message,omitempty"`
}

// Deployment represents a deployment resource
type Deployment struct {
	Metadata ObjectMeta       `json:"metadata"`
	Spec     DeploymentSpec   `json:"spec"`
	Status   DeploymentStatus `json:"status"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

// DeploymentSpec defines the specification for a Deployment
type DeploymentSpec struct {
	Replicas int32             `json:"replicas"`
	Selector map[string]string `json:"selector"`
	Template PodTemplateSpec   `json:"template"`
}

// PodTemplateSpec defines a template for creating Pods
type PodTemplateSpec struct {
	Metadata ObjectMeta `json:"metadata"`
	Spec     PodSpec    `json:"spec"`
}

// DeploymentStatus represents the status of a Deployment
type DeploymentStatus struct {
	Replicas          int32  `json:"replicas"`
	UpdatedReplicas   int32  `json:"updatedReplicas"`
	AvailableReplicas int32  `json:"availableReplicas"`
	Message           string `json:"message,omitempty"`
}
