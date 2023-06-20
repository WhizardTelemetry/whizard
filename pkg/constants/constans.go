package constants

const (
	DefaultTenantHeader    = "WHIZARD-TENANT"
	DefaultTenantId        = "default-tenant"
	DefaultTenantLabelName = "tenant_id"

	ServiceLabelKey = "monitoring.whizard.io/service"
	StorageLabelKey = "monitoring.whizard.io/storage"
	TenantLabelKey  = "monitoring.whizard.io/tenant"

	ExclusiveLabelKey  = "monitoring.whizard.io/exclusive"
	SoftTenantLabelKey = "monitoring.whizard.io/soft-tenant"

	FinalizerIngester  = "finalizers.monitoring.whizard.io/ingester"
	FinalizerCompactor = "finalizers.monitoring.whizard.io/compactor"
	FinalizerDeletePVC = "finalizers.monitoring.whizard.io/deletePVC"

	DefaultStorage = "default"
	LocalStorage   = "local"

	GRPCPort        = 10901
	HTTPPort        = 10902
	RemoteWritePort = 19291

	GRPCPortName        = "grpc"
	HTTPPortName        = "http"
	RemoteWritePortName = "remote-write"

	ReceiveReplicaLabelName = "receive_replica"
	RulerReplicaLabelName   = "ruler_replica"

	AppNameGateway       = "gateway"
	AppNameQuery         = "query"
	AppNameQueryFrontend = "query-frontend"
	AppNameRouter        = "router"
	AppNameIngester      = "ingester"
	AppNameRuler         = "ruler"
	AppNameStore         = "store"
	AppNameCompactor     = "compactor"
	AppNameStorage       = "storage"
	AppNameBlockManager  = "block-manager"

	ServiceNameSuffix = "operated"

	LabelNameAppName      = "app.kubernetes.io/name"
	LabelNameAppManagedBy = "app.kubernetes.io/managed-by"
	LabelNameAppPartOf    = "app.kubernetes.io/part-of"

	LabelNameIngesterState        = "monitoring.whizard.io/ingester-state"
	LabelNameIngesterDeletingTime = "monitoring.whizard.io/ingester-deleting-time"

	LabelNameRulerShardSn = "monitoring.whizard.io/ruler-shard-sn"

	ConfigPath     = "/etc/whizard/"
	StorageDir     = "/whizard"
	TSDBVolumeName = "tsdb"

	WhizardConfigMountPath  = "/etc/whizard/config/"
	WhizardCertsMountPath   = "/etc/whizard/certs/"
	EnvoyConfigMountPath    = "/etc/envoy/config/"
	EnvoyCertsMountPath     = "/etc/envoy/certs/"
	EnvoyConfigMapMountPath = "/etc/envoy/configmap/"
	EnvoySecretMountPath    = "/etc/envoy/secret/"

	LabelNameStorageHash = "monitoring.whizard.io/storage-hash"
	LabelNameTenantHash  = "monitoring.whizard.io/tenant-hash"

	TenantHash  = "TENANT_HASH"
	StorageHash = "STORAGE_HASH"

	IngesterStateDeleting = "deleting"
	IngesterStateRunning  = "running"
)

const (
	StorageProviderFILESYSTEM string = "FILESYSTEM"
	StorageProviderGCS        string = "GCS"
	StorageProviderS3         string = "S3"
	StorageProviderAZURE      string = "AZURE"
	StorageProviderSWIFT      string = "SWIFT"
	StorageProviderCOS        string = "COS"
	StorageProviderALIYUNOSS  string = "ALIYUNOSS"
	StorageProviderBOS        string = "BOS"
)

// Port layout of single-node components.
// Used in envoy-sidecar proxy.
// https://thanos.io/tip/thanos/getting-started.md/#testing-thanos-on-single-host

const (
	CustomProxyPort            = "10900"
	SidecarGRPCPort            = "10901"
	SidecarHTTPPort            = "10902"
	QueryGRPCPort              = "10903"
	QueryHTTPPort              = "10904"
	StoreGRPCPort              = "10905"
	StoreHTTPPort              = "10906"
	ReceiveGRPCPort            = "10907"
	ReceiveHTTPRemoteWritePort = "10908"
	ReceiveHTTPPort            = "10909"
	RuleGRPCPort               = "10910"
	RuleHTTPPort               = "10911"
	CompactHTTPPort            = "10912"
	QueryFrontendHTTPPort      = "10913"
)
