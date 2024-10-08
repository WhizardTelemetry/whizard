/*
Copyright 2024 the Whizard Authors.

Licensed under Apache License, Version 2.0 with a few additional conditions.

You may obtain a copy of the License at

    https://github.com/WhizardTelemetry/whizard/blob/main/LICENSE
*/

package v1alpha1

import (
	"github.com/prometheus/common/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// StorageSpec defines the desired state of Storage
type StorageSpec struct {
	BlockManager *BlockManager `json:"blockManager,omitempty"`
	S3           *S3           `json:"S3,omitempty"`
}

type BlockManager struct {
	Enable     *bool `json:"enable,omitempty"`
	CommonSpec `json:",inline"`
	// ServiceAccountName is the name of the ServiceAccount to use to run bucket Pods.
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
	// NodePort is the port used to expose the bucket service.
	// If this is a valid node port, the gateway service type will be set to NodePort accordingly.
	NodePort int32 `json:"nodePort,omitempty"`
	// Interval to sync block metadata from object storage
	BlockSyncInterval *metav1.Duration `json:"blockSyncInterval,omitempty"`
	GC                *BlockGC         `json:"gc,omitempty"`
}

type BlockGC struct {
	Enable *bool `json:"enable,omitempty"`
	// Define resources requests and limits for main container.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// Image is the component image with tag/version.
	Image string `json:"image,omitempty"`
	// Image pull policy.
	// One of Always, Never, IfNotPresent.
	// Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.
	// Cannot be updated.
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	GCInterval     *metav1.Duration `json:"gcInterval,omitempty"`
	CleanupTimeout *metav1.Duration `json:"cleanupTimeout,omitempty"`

	// Default tenant ID to use when none is provided via a header.
	DefaultTenantId string `json:"defaultTenantId,omitempty"`
	// Label name through which the tenant will be announced.
	TenantLabelName string `json:"tenantLabelName,omitempty"`
}

// Config stores the configuration for s3 bucket.
// https://github.com/thanos-io/objstore/blob/main/providers/s3
type S3 struct {
	Bucket             string                    `yaml:"bucket,omitempty" json:"bucket"`
	Endpoint           string                    `yaml:"endpoint,omitempty" json:"endpoint"`
	Region             string                    `yaml:"region,omitempty" json:"region,omitempty"`
	DisableDualstack   bool                      `yaml:"disable_dualstack,omitempty" json:"disableDualstack,omitempty"`
	AWSSDKAuth         bool                      `yaml:"aws_sdk_auth,omitempty" json:"awsSdkAuth,omitempty"`
	AccessKey          *corev1.SecretKeySelector `yaml:"access_key" json:"accessKey"`
	Insecure           bool                      `yaml:"insecure,omitempty" json:"insecure,omitempty"`
	SignatureV2        bool                      `yaml:"signature_version2,omitempty" json:"signatureVersion2,omitempty"`
	SecretKey          *corev1.SecretKeySelector `yaml:"secret_key" json:"secretKey"`
	PutUserMetadata    map[string]string         `yaml:"put_user_metadata,omitempty" json:"putUserMetadata,omitempty"`
	HTTPConfig         S3HTTPConfig              `yaml:"http_config,omitempty" json:"httpConfig,omitempty"`
	TraceConfig        S3TraceConfig             `yaml:"trace,omitempty" json:"trace,omitempty"`
	ListObjectsVersion string                    `yaml:"list_objects_version,omitempty" json:"listObjectsVersion,omitempty"`
	SendContentMd5     bool                      `yaml:"send_content_md5,omitempty" json:"sendContentMd5,omitempty"`
	DisableMultipart   bool                      `yaml:"disable_multipart,omitempty" json:"disableMultipart,omitempty"`
	// PartSize used for multipart upload. Only used if uploaded object size is known and larger than configured PartSize.
	// NOTE we need to make sure this number does not produce more parts than 10 000.
	PartSize    uint64      `yaml:"part_size,omitempty" json:"partSize,omitempty"`
	SSEConfig   S3SSEConfig `yaml:"sse_config,omitempty" json:"sseConfig,omitempty"`
	STSEndpoint string      `yaml:"sts_endpoint,omitempty" json:"stsEndpoint,omitempty"`
}

// S3SSEConfig deals with the configuration of SSE for Minio. The following options are valid:
// kmsencryptioncontext == https://docs.aws.amazon.com/kms/latest/developerguide/services-s3.html#s3-encryption-context
type S3SSEConfig struct {
	Type                 string            `yaml:"type,omitempty" json:"type,omitempty"`
	KMSKeyID             string            `yaml:"kms_key_id,omitempty" json:"kmsKeyId,omitempty"`
	KMSEncryptionContext map[string]string `yaml:"kms_encryption_context,omitempty" json:"kmsEncryptionContext,omitempty"`
	EncryptionKey        string            `yaml:"encryption_key,omitempty" json:"encryptionKey,omitempty"`
}

type S3TraceConfig struct {
	Enable bool `yaml:"enable,omitempty" json:"enable,omitempty"`
}

// S3HTTPConfig stores the http.Transport configuration for the s3 minio client.
type S3HTTPConfig struct {
	IdleConnTimeout       model.Duration `yaml:"idle_conn_timeout,omitempty" json:"idleConnTimeout,omitempty"`
	ResponseHeaderTimeout model.Duration `yaml:"response_header_timeout,omitempty" json:"responseHeaderTimeout,omitempty"`
	InsecureSkipVerify    bool           `yaml:"insecure_skip_verify,omitempty" json:"insecureSkipVerify,omitempty"`

	TLSHandshakeTimeout   model.Duration `yaml:"tls_handshake_timeout,omitempty" json:"tlsHandshakeTimeout,omitempty"`
	ExpectContinueTimeout model.Duration `yaml:"expect_continue_timeout,omitempty" json:"expectContinueTimeout,omitempty"`
	MaxIdleConns          int            `yaml:"max_idle_conns,omitempty" json:"maxIdleConns,omitempty"`
	MaxIdleConnsPerHost   int            `yaml:"max_idle_conns_per_host,omitempty" json:"maxIdleConnsPerHost,omitempty"`
	MaxConnsPerHost       int            `yaml:"max_conns_per_host,omitempty" json:"maxConnsPerHost,omitempty"`

	TLSConfig          TLSConfig `yaml:"tls_config,omitempty" json:"tlsConfig,omitempty"`
	DisableCompression bool      `yaml:"disable_compression" json:"disableCompression"`
}

// TLSConfig configures the options for TLS connections.
type TLSConfig struct {
	// The secret that including the CA cert.
	CA *corev1.SecretKeySelector `yaml:"ca_file,omitempty" json:"ca,omitempty"`
	// The secret that including the client cert.
	Cert *corev1.SecretKeySelector `yaml:"cert_file,omitempty" json:"cert,omitempty"`
	// The secret that including the client key.
	Key *corev1.SecretKeySelector `yaml:"key_file,omitempty" json:"key,omitempty"`
	// Used to verify the hostname for the targets.
	ServerName string `yaml:"server_name,omitempty" json:"serverName,omitempty"`
	// Disable target certificate validation.
	InsecureSkipVerify bool `yaml:"insecure_skip_verify,omitempty" json:"insecureSkipVerify,omitempty"`
}

// StorageStatus defines the observed state of Storage
type StorageStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// The `Storage` custom resource definition (CRD) defines how to configure access to object storage.
// More info https://thanos.io/tip/thanos/storage.md/
// Current object storage client implementations: S3, other in progress.
type Storage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StorageSpec   `json:"spec,omitempty"`
	Status StorageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StorageList contains a list of Storage
type StorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Storage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Storage{}, &StorageList{})
}
