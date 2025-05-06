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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VirtualNodeConnectionSpec defines the desired state of VirtualNodeConnection.
type VirtualNodeConnectionSpec struct {
	VirtualNodeA string `json:"virtualNodeA"`
	KubeconfigA  string `json:"kubeconfigA"`
	VirtualNodeB string `json:"virtualNodeB"`
	KubeconfigB  string `json:"kubeconfigB"`
}

// VirtualNodeConnectionStatus defines the observed state of VirtualNodeConnection.
type VirtualNodeConnectionStatus struct {
	// IsConnected indica se i nodi sono connessi.
	IsConnected bool `json:"isConnected"`
	// LastUpdated rappresenta il timestamp dell'ultimo aggiornamento dello stato.
	LastUpdated string `json:"lastUpdated,omitempty"`
	// Phase rappresenta la fase corrente del processo di connessione.
	Phase string `json:"phase,omitempty"`
	// ErrorMessage contiene eventuali messaggi di errore.
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=liqo,shortName=vnc;vnconnection
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Connected",type=boolean,JSONPath=`.status.isConnected`
// VirtualNodeConnection is the Schema for the virtualnodeconnections API.
type VirtualNodeConnection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualNodeConnectionSpec   `json:"spec,omitempty"`
	Status VirtualNodeConnectionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VirtualNodeConnectionList contains a list of VirtualNodeConnection.
type VirtualNodeConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualNodeConnection `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualNodeConnection{}, &VirtualNodeConnectionList{})
}
