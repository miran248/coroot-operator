package v1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	DefaultMetricRefreshInterval = "15s"
)

type CommunityEditionSpec struct {
	Version string `json:"version,omitempty"`
}

type EnterpriseEditionSpec struct {
	Version    string `json:"version,omitempty"`
	LicenseKey string `json:"licenseKey,omitempty"`
}

type AgentsOnlySpec struct {
	CorootURL string `json:"corootURL,omitempty"`
}

type ServiceSpec struct {
	Type     corev1.ServiceType `json:"type,omitempty"`
	Port     int32              `json:"port,omitempty"`
	NodePort int32              `json:"nodePort,omitempty"`
}

type StorageSpec struct {
	Size      resource.Quantity `json:"size,omitempty"`
	ClassName *string           `json:"className,omitempty"`
}

type NodeAgentSpec struct {
	Version string `json:"version,omitempty"`

	PriorityClassName string                         `json:"priorityClassName,omitempty"`
	UpdateStrategy    appsv1.DaemonSetUpdateStrategy `json:"update_strategy,omitempty"`
	Affinity          *corev1.Affinity               `json:"affinity,omitempty"`
	Resources         corev1.ResourceRequirements    `json:"resources,omitempty"`
	Env               []corev1.EnvVar                `json:"env,omitempty"`
}

type ClusterAgentSpec struct {
	Version string `json:"version,omitempty"`

	Affinity  *corev1.Affinity            `json:"affinity,omitempty"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	Env       []corev1.EnvVar             `json:"env,omitempty"`
}

type PrometheusSpec struct {
	Affinity  *corev1.Affinity            `json:"affinity,omitempty"`
	Storage   StorageSpec                 `json:"storage,omitempty"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

type ClickhouseSpec struct {
	Shards   int `json:"shards,omitempty"`
	Replicas int `json:"replicas,omitempty"`

	Affinity  *corev1.Affinity            `json:"affinity,omitempty"`
	Storage   StorageSpec                 `json:"storage,omitempty"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	Keeper ClickhouseKeeperSpec `json:"keeper,omitempty"`
}

type ClickhouseKeeperSpec struct {
	Affinity  *corev1.Affinity            `json:"affinity,omitempty"`
	Storage   StorageSpec                 `json:"storage,omitempty"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

type ExternalClickhouseSpec struct {
	Address  string `json:"address,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	Database string `json:"database,omitempty"`
}

type PostgresSpec struct {
	ConnectionString string `json:"connectionString,omitempty"`
}

type CorootSpec struct {
	ApiKey                     string          `json:"apiKey,omitempty"`
	MetricsRefreshInterval     metav1.Duration `json:"metricsRefreshInterval,omitempty"`
	CacheTTL                   metav1.Duration `json:"cacheTTL,omitempty"`
	AuthAnonymousRole          string          `json:"authAnonymousRole,omitempty"`
	AuthBootstrapAdminPassword string          `json:"authBootstrapAdminPassword,omitempty"`
	Env                        []corev1.EnvVar `json:"env,omitempty"`

	CommunityEdition  CommunityEditionSpec   `json:"communityEdition,omitempty"`
	EnterpriseEdition *EnterpriseEditionSpec `json:"enterpriseEdition,omitempty"`
	AgentsOnly        *AgentsOnlySpec        `json:"agentsOnly,omitempty"`

	Replicas  int                         `json:"replicas,omitempty"`
	Service   ServiceSpec                 `json:"service,omitempty"`
	Affinity  *corev1.Affinity            `json:"affinity,omitempty"`
	Storage   StorageSpec                 `json:"storage,omitempty"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	NodeAgent    NodeAgentSpec    `json:"nodeAgent,omitempty"`
	ClusterAgent ClusterAgentSpec `json:"clusterAgent,omitempty"`

	Prometheus PrometheusSpec `json:"prometheus,omitempty"`

	Clickhouse         ClickhouseSpec          `json:"clickhouse,omitempty"`
	ExternalClickhouse *ExternalClickhouseSpec `json:"externalClickhouse,omitempty"`

	Postgres *PostgresSpec `json:"postgres,omitempty"`
}

type CorootStatus struct { // TODO
	// Represents the observations of a Coroot's current state.
	// Coroot.status.conditions.type are: "Available", "Progressing", and "Degraded"
	// Coroot.status.conditions.status are one of True, False, Unknown.
	// Coroot.status.conditions.reason the value should be a CamelCase string and producers of specific
	// condition types may define expected values and meanings for this field, and whether the values
	// are considered a guaranteed API.
	// Coroot.status.conditions.Message is a human-readable message indicating details about the transition.
	// For further information see: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type Coroot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CorootSpec   `json:"spec,omitempty"`
	Status CorootStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type CorootList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Coroot `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Coroot{}, &CorootList{})
}
