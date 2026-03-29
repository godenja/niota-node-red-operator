package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodeRedInstanceSpec defines the desired state of a Node-RED instance.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=nri
// +kubebuilder:printcolumn:name="ID",type=string,JSONPath=".spec.id"
// +kubebuilder:printcolumn:name="Domain",type=string,JSONPath=".spec.domain"
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
type NodeRedInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeRedInstanceSpec   `json:"spec,omitempty"`
	Status NodeRedInstanceStatus `json:"status,omitempty"`
}

// NodeRedInstanceSpec defines the desired state of NodeRedInstance.
type NodeRedInstanceSpec struct {
	// ID is the unique Node-RED instance identifier. Used as the URL path prefix
	// and as the base name for all managed Kubernetes resources.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-f0-9]{32}$`
	ID string `json:"id"`

	// Domain is the public hostname under which the instance is accessible.
	// +kubebuilder:validation:Required
	Domain string `json:"domain"`

	// TLSSecretName is the name of the Kubernetes TLS Secret used by the
	// Traefik IngressRoute.
	// +kubebuilder:validation:Required
	TLSSecretName string `json:"tlsSecretName"`

	// IngressClass sets the kubernetes.io/ingress.class annotation on the
	// Traefik IngressRoute, allowing multi-Traefik setups to route correctly.
	// +kubebuilder:default=traefik
	// +optional
	IngressClass string `json:"ingressClass,omitempty"`

	// OAuth holds OAuth2 credentials used by the Node-RED passport strategy.
	// +kubebuilder:validation:Required
	OAuth OAuthSpec `json:"oauth"`

	// Image configures the Node-RED container image.
	// +kubebuilder:validation:Required
	Image ImageSpec `json:"image"`

	// StorageSize is the requested PVC capacity for Node-RED data.
	// Defaults to "1Gi".
	// +kubebuilder:default="1Gi"
	// +optional
	StorageSize string `json:"storageSize,omitempty"`

	// StorageClass specifies the StorageClass for the PVC.
	// When omitted the cluster default is used.
	// +optional
	StorageClass *string `json:"storageClass,omitempty"`

	// CredentialSecret is the encryption key Node-RED uses to secure its
	// credentials file (credentialSecret in settings.js). When omitted, the
	// operator auto-generates a random 32-byte key on first reconciliation.
	// Provide this field when migrating an existing Node-RED instance so that
	// already-encrypted credentials remain accessible.
	// The resulting Kubernetes Secret is immutable — it will not be rotated
	// by subsequent CR updates.
	// +optional
	CredentialSecret string `json:"credentialSecret,omitempty"`
}

// OAuthSpec contains the OAuth2 client configuration for Node-RED.
type OAuthSpec struct {
	// ClientID is the OAuth2 client identifier.
	// +kubebuilder:validation:Required
	ClientID string `json:"clientId"`

	// ClientSecret is the OAuth2 client secret in plaintext. The operator
	// creates (and keeps updated) a Kubernetes Secret from this value.
	// +kubebuilder:validation:Required
	ClientSecret string `json:"clientSecret"`

	// AuthURL is the OAuth2 authorization endpoint
	// (e.g. https://datahub.digital/api/v1/oauth2/authorize).
	// +kubebuilder:validation:Required
	AuthURL string `json:"authUrl"`

	// TokenURL is the OAuth2 token endpoint
	// (e.g. https://datahub.digital/api/v1/oauth2/token).
	// +kubebuilder:validation:Required
	TokenURL string `json:"tokenUrl"`
}

// ImageSpec configures the Node-RED container image.
type ImageSpec struct {
	// Repository is the container image repository
	// (e.g. "ghcr.io/my-org/node-red").
	// +kubebuilder:validation:Required
	Repository string `json:"repository"`

	// Tag is the image tag. Defaults to "latest".
	// +kubebuilder:default=latest
	// +optional
	Tag string `json:"tag,omitempty"`

	// PullSecretName is the name of the imagePullSecret used to pull the image.
	// +optional
	PullSecretName string `json:"pullSecretName,omitempty"`
}

// NodeRedInstanceStatus describes the observed state of NodeRedInstance.
type NodeRedInstanceStatus struct {
	// Ready indicates whether all managed resources are operational.
	// +optional
	Ready bool `json:"ready,omitempty"`

	// Conditions contains the latest observations of the instance's state.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
type NodeRedInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeRedInstance `json:"items"`
}
