package compactor

import (
	"fmt"
	"strings"

	"github.com/kubesphere/whizard/pkg/api/monitoring/v1alpha1"
	"github.com/kubesphere/whizard/pkg/constants"
	"github.com/kubesphere/whizard/pkg/controllers/monitoring/resources"
	"github.com/kubesphere/whizard/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	mainContainerName = "compactor"
)

var (
	// sliceArgs is the args that can be set repeatedly.
	// An error will occur if a non-slice arg is set repeatedly.
	sliceArgs = []string{
		"--deduplication.replica-label",
	}
)

func (r *Compactor) statefulSet() (runtime.Object, resources.Operation, error) {
	var sts = &appsv1.StatefulSet{ObjectMeta: r.meta(r.compactor.Name)}
	if err := r.Client.Get(r.Context, client.ObjectKeyFromObject(sts), sts); err != nil {
		if !util.IsNotFound(err) {
			return nil, "", err
		}
	}

	sts.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: r.labels(),
	}

	sts.Spec.Replicas = r.compactor.Spec.Replicas
	sts.Spec.Template.Labels = r.labels()
	sts.Spec.Template.Spec.Affinity = r.compactor.Spec.Affinity
	sts.Spec.Template.Spec.NodeSelector = r.compactor.Spec.NodeSelector
	sts.Spec.Template.Spec.Volumes = []corev1.Volume{}
	sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{}

	var container *corev1.Container
	for i := 0; i < len(sts.Spec.Template.Spec.Containers); i++ {
		if sts.Spec.Template.Spec.Containers[i].Name == mainContainerName {
			container = &sts.Spec.Template.Spec.Containers[i]
		}
	}

	needToAppend := false
	if container == nil {
		container = &corev1.Container{
			Name:      mainContainerName,
			Image:     r.compactor.Spec.Image,
			Resources: r.compactor.Spec.Resources,
			Ports: []corev1.ContainerPort{
				{
					Name:          constants.HTTPPortName,
					ContainerPort: constants.HTTPPort,
					Protocol:      corev1.ProtocolTCP,
				},
			},
		}

		needToAppend = true
	}

	container.VolumeMounts = []corev1.VolumeMount{}
	resources.AddTSDBVolume(sts, container, r.compactor.Spec.DataVolume)

	volumes, volumeMounts, err := resources.VolumesAndVolumeMountsForStorage(r.Context, r.Client, r.compactor.Labels[constants.StorageLabelKey])
	if err != nil {
		return nil, "", err
	}
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, volumes...)
	container.VolumeMounts = append(container.VolumeMounts, volumeMounts...)

	if container.LivenessProbe == nil {
		container.LivenessProbe = resources.DefaultLivenessProbe()
	}

	if container.ReadinessProbe == nil {
		container.ReadinessProbe = resources.DefaultReadinessProbe()
	}

	container.Resources = r.compactor.Spec.Resources

	hashCode, err := resources.GetStorageHash(r.Context, r.Client, r.compactor.Labels[constants.StorageLabelKey])
	if err != nil {
		return nil, "", err
	}
	env := corev1.EnvVar{
		Name:  constants.StorageHash,
		Value: hashCode,
	}
	replaced := util.ReplaceInSlice(container.Env, func(v interface{}) bool {
		return v.(corev1.EnvVar).Name == constants.StorageHash
	}, env)
	if !replaced {
		container.Env = append(container.Env, env)
	}

	if args, err := r.megerArgs(); err != nil {
		return nil, "", err
	} else {
		container.Args = args
	}

	if needToAppend {
		sts.Spec.Template.Spec.Containers = append(sts.Spec.Template.Spec.Containers, *container)
	}

	return sts, resources.OperationCreateOrUpdate, ctrl.SetControllerReference(r.compactor, sts, r.Scheme)
}

type relabelConfig struct {
	Action        string   `yaml:"action"`
	SourceLablels []string `yaml:"source_labels"`
	Regex         string   `yaml:"regex"`
}

func (r *Compactor) createRelabelConfig() (string, error) {

	namespacedName := strings.Split(r.compactor.Labels[constants.ServiceLabelKey], ".")
	svc := &v1alpha1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName[1],
			Namespace: namespacedName[0],
		},
	}
	if err := r.Client.Get(r.Context, client.ObjectKeyFromObject(svc), svc); err != nil {
		return "", err
	}

	label := svc.Spec.TenantLabelName
	if len(label) == 0 {
		label = constants.DefaultTenantLabelName
	}

	regex := ""
	for _, tenant := range r.compactor.Spec.Tenants {
		regex = fmt.Sprintf("%s|^%s$", regex, tenant)
	}

	return util.YamlMarshal([]relabelConfig{
		{Action: "keep",
			SourceLablels: []string{label},
			Regex:         strings.TrimPrefix(regex, "|"),
		},
	})
}

func (r *Compactor) megerArgs() ([]string, error) {

	storageConfig, err := resources.GetStorageConfig(r.Context, r.Client, r.compactor.Labels[constants.StorageLabelKey])
	if err != nil {
		return nil, err
	}

	rc, err := r.createRelabelConfig()
	if err != nil {
		return nil, err
	}

	defaultArgs := []string{
		"compact",
		"--wait",
		fmt.Sprintf("--data-dir=%s", constants.StorageDir),
		"--objstore.config=" + string(storageConfig),
		fmt.Sprintf("--selector.relabel-config=%s", rc),
		fmt.Sprintf("--deduplication.replica-label=%s", constants.ReceiveReplicaLabelName),
		fmt.Sprintf("--deduplication.replica-label=%s", constants.RulerReplicaLabelName),
	}

	if r.compactor.Spec.LogLevel != "" {
		defaultArgs = append(defaultArgs, "--log.level="+r.compactor.Spec.LogLevel)
	}
	if r.compactor.Spec.LogFormat != "" {
		defaultArgs = append(defaultArgs, "--log.format="+r.compactor.Spec.LogFormat)
	}
	if r.compactor.Spec.DownsamplingDisable != nil {
		defaultArgs = append(defaultArgs, fmt.Sprintf("--downsampling.disable=%v", r.compactor.Spec.DownsamplingDisable))
	}
	if retention := r.compactor.Spec.Retention; retention != nil {
		if retention.RetentionRaw != "" {
			defaultArgs = append(defaultArgs, fmt.Sprintf("--retention.resolution-raw=%s", retention.RetentionRaw))
		}
		if retention.Retention5m != "" {
			defaultArgs = append(defaultArgs, fmt.Sprintf("--retention.resolution-5m=%s", retention.Retention5m))
		}
		if retention.Retention5m != "" {
			defaultArgs = append(defaultArgs, fmt.Sprintf("--retention.resolution-1h=%s", retention.Retention5m))
		}
	}

	for _, flag := range r.compactor.Spec.Flags {
		if arg := util.GetArgName(flag); util.Contains(sliceArgs, arg) {
			defaultArgs = append(defaultArgs, flag)
			continue
		}

		replaced := util.ReplaceInSlice(defaultArgs, func(v interface{}) bool {
			return util.GetArgName(v.(string)) == util.GetArgName(flag)
		}, flag)
		if !replaced {
			defaultArgs = append(defaultArgs, flag)
		}
	}

	return defaultArgs, nil
}
