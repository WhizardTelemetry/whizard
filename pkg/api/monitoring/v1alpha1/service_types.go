/*
Copyright 2024 the Whizard Authors.

Licensed under Apache License, Version 2.0 with a few additional conditions.

You may obtain a copy of the License at

    https://github.com/WhizardTelemetry/whizard/blob/main/LICENSE
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceSpec defines the desired state of Service
type ServiceSpec struct {
	// HTTP header to determine tenant for remote write requests.
	TenantHeader string `json:"tenantHeader,omitempty"`
	// Default tenant ID to use when none is provided via a header.
	DefaultTenantId string `json:"defaultTenantId,omitempty"`
	// Label name through which the tenant will be announced.
	TenantLabelName string `json:"tenantLabelName,omitempty"`

	Storage *ObjectReference `json:"storage,omitempty"`

	// RemoteWrites is the list of remote write configurations.
	// If it is configured, its targets will receive write requests from the Gateway and the Ruler.
	RemoteWrites []RemoteWriteSpec `json:"remoteWrites,omitempty"`
	// RemoteQuery is the remote query configuration and the remote target should have prometheus-compatible Query APIs.
	// If not configured, the Gateway will proxy all read requests through the QueryFrontend to the Query,
	// If configured, the Gateway will proxy metrics read requests through the QueryFrontend to the remote target,
	// but proxy rules read requests directly to the Query.
	RemoteQuery *RemoteQuerySpec `json:"remoteQuery,omitempty"`

	// GatewayTemplateSpec defines the Gateway configuration template.
	GatewayTemplateSpec GatewaySpec `json:"gatewayTemplateSpec"`
	// QueryFrontendTemplateSpec defines the QueryFrontend configuration template.
	QueryFrontendTemplateSpec QueryFrontendSpec `json:"queryFrontendTemplateSpec"`
	// QueryTemplateSpec defines the Query configuration template.
	QueryTemplateSpec QuerySpec `json:"queryTemplateSpec"`
	// RulerTemplateSpec defines the Ruler configuration template.
	RulerTemplateSpec RulerTemplateSpec `json:"rulerTemplateSpec"`
	// RouterTemplateSpec defines the Router configuration template.
	RouterTemplateSpec RouterSpec `json:"routerTemplateSpec"`
	// IngesterTemplateSpec defines the Ingester configuration template.
	IngesterTemplateSpec IngesterTemplateSpec `json:"ingesterTemplateSpec"`
	// StoreTemplateSpec defines the Store configuration template.
	StoreTemplateSpec StoreSpec `json:"storeTemplateSpec"`
	// CompactorTemplateSpec defines the Compactor configuration template.
	CompactorTemplateSpec CompactorTemplateSpec `json:"compactorTemplateSpec"`
}

type CompactorTemplateSpec struct {
	CompactorSpec `json:",inline"`

	// DefaultTenantsPerIngester Whizard default tenant count per ingester.
	// Default: 10
	// +kubebuilder:default:=10
	DefaultTenantsPerCompactor int `json:"defaultTenantsPerCompactor,omitempty"`
}

type IngesterTemplateSpec struct {
	IngesterSpec `json:",inline"`

	// DefaultTenantsPerIngester Whizard default tenant count per ingester.
	//
	// Default: 3
	// +kubebuilder:default:=3
	DefaultTenantsPerIngester int `json:"defaultTenantsPerIngester,omitempty"`
	// DefaultIngesterRetentionPeriod Whizard default ingester retention period when it has no tenant.
	//
	// Default: "3h"
	// +kubebuilder:default:="3h"
	DefaultIngesterRetentionPeriod Duration `json:"defaultIngesterRetentionPeriod,omitempty"`
	// DisableTSDBCleanup Disable the TSDB cleanup of ingester.
	// The cleanup will delete the blocks that belong to deleted tenants in the data directory of ingester TSDB.
	//
	// Default: true
	// +kubebuilder:default:=true
	DisableTSDBCleanup *bool `json:"disableTsdbCleanup,omitempty"`
}

type RulerTemplateSpec struct {
	RulerSpec `json:",inline"`

	// DisableAlertingRulesAutoSelection disable auto select alerting rules in tenant ruler
	//
	// Default: true
	// +kubebuilder:default:=true
	DisableAlertingRulesAutoSelection *bool `json:"disableAlertingRulesAutoSelection,omitempty"`
}

// RemoteQuerySpec defines the configuration to query from remote service
// which should have prometheus-compatible Query APIs.
type RemoteQuerySpec struct {
	Name             string `json:"name,omitempty"`
	URL              string `json:"url"`
	HTTPClientConfig `json:",inline"`
}

// RemoteWriteSpec defines the remote write configuration.
type RemoteWriteSpec struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url"`
	// Custom HTTP headers to be sent along with each remote write request.
	Headers map[string]string `json:"headers,omitempty"`
	// Timeout for requests to the remote write endpoint.
	RemoteTimeout Duration `json:"remoteTimeout,omitempty"`

	HTTPClientConfig `json:",inline"`
}

// ServiceStatus defines the observed state of Service
type ServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// The `Service` custom resource definition (CRD) defines the Whizard service configuration.
// The `ServiceSpec“ has component configuration templates. Some components scale based on the number of tenants and load service configurations
type Service struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceSpec   `json:"spec,omitempty"`
	Status ServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceList contains a list of Service
type ServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Service `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Service{}, &ServiceList{})
}
