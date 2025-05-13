/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ForeignClusterConnectionSpec defines the desired state of ForeignClusterConnection.
type ForeignClusterConnectionSpec struct {
	ForeignClusterA string           `json:"foreignClusterA"`
	ForeignClusterB string           `json:"foreignClusterB"`
	Networking      NetworkingConfig `json:"networking"`
}

// NetworkingConfig describes configuration flags for setting up the virtual connection.
type NetworkingConfig struct {
	MTU                int32 `json:"mtu"`
	DisableSharingKeys bool  `json:"disableSharingKeys"`

	ServerGatewayType       string `json:"serverGatewayType"`
	ServerTemplateName      string `json:"serverTemplateName"`
	ServerTemplateNamespace string `json:"serverTemplateNamespace"`
	ServerServiceType       string `json:"serverServiceType"`
	ServerServicePort       int32  `json:"serverServicePort"`

	ClientGatewayType       string `json:"clientGatewayType"`
	ClientTemplateName      string `json:"clientTemplateName"`
	ClientTemplateNamespace string `json:"clientTemplateNamespace"`

	TimeoutSeconds int32 `json:"timeoutSeconds"`
	Wait           bool  `json:"wait"`
}

// ForeignClusterConnectionStatus defines the observed state of ForeignClusterConnection.
type ForeignClusterConnectionStatus struct {
	IsConnected  bool   `json:"isConnected"`
	LastUpdated  string `json:"lastUpdated,omitempty"`
	Phase        string `json:"phase,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`

	RemoteClusterA ClusterNetworkingStatus `json:"remoteClusterA,omitempty"`
	RemoteClusterB ClusterNetworkingStatus `json:"remoteClusterB,omitempty"`
}

// ClusterNetworkingStatus describes resolved values for CIDR handling between clusters.
type ClusterNetworkingStatus struct {
	PodCIDR         string `json:"podCIDR,omitempty"`
	ExternalCIDR    string `json:"externalCIDR,omitempty"`
	RemappedPodCIDR string `json:"remappedPodCIDR,omitempty"`
	RemappedExtCIDR string `json:"remappedExternalCIDR,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:singular=foreignclusterconnection,categories=liqo,shortName=fcc;fcconnection
// +kubebuilder:printcolumn:name="ClusterA",type=string,JSONPath=`.spec.foreignClusterA`
// +kubebuilder:printcolumn:name="ClusterB",type=string,JSONPath=`.spec.foreignClusterB`
// +kubebuilder:printcolumn:name="Connected",type=boolean,JSONPath=`.status.isConnected`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="A-CIDR",type=string,JSONPath=`.status.remoteClusterA.podCIDR`
// +kubebuilder:printcolumn:name="B-CIDR",type=string,JSONPath=`.status.remoteClusterB.podCIDR`

// ForeignClusterConnection is the Schema for the foreignclusterconnections API.
type ForeignClusterConnection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ForeignClusterConnectionSpec   `json:"spec,omitempty"`
	Status ForeignClusterConnectionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ForeignClusterConnectionList contains a list of ForeignClusterConnection.
type ForeignClusterConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ForeignClusterConnection `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ForeignClusterConnection{}, &ForeignClusterConnectionList{})
}
