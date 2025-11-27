package base

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/WhizardTelemetry/whizard/pkg/api/monitoring/v1alpha1"
	"github.com/WhizardTelemetry/whizard/pkg/constants"
	"github.com/WhizardTelemetry/whizard/pkg/util"
	"github.com/prometheus-operator/prometheus-operator/pkg/k8sutil"
)

type Component struct {
	AppName    string
	Instance   client.Object
	CommonSpec v1alpha1.CommonSpec
}

func (c *Component) MakeObjectMeta() metav1.ObjectMeta {
	gvk := c.Instance.GetObjectKind().GroupVersionKind()
	return metav1.ObjectMeta{
		Name:      c.Instance.GetName(),
		Namespace: c.Instance.GetNamespace(),
		Labels: map[string]string{
			constants.LabelNameAppName:      c.AppName,
			constants.LabelNameAppManagedBy: c.Instance.GetName(),
		},
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion: gvk.GroupVersion().String(),
				Kind:       gvk.Kind,
				Name:       c.Instance.GetName(),
				UID:        c.Instance.GetUID(),
				Controller: ptr.To(true),
			},
		},
	}
}

func (c *Component) MakeDeployment() (*appsv1.Deployment, error) {
	objectmeta := c.MakeObjectMeta()
	podTemplate, err := c.makePodTemplate()
	if err != nil {
		return nil, err
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: objectmeta,
		Spec: appsv1.DeploymentSpec{
			Replicas: c.CommonSpec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: objectmeta.GetLabels(),
			},
			Template: podTemplate,
		},
	}

	return deploy, nil
}

func (c *Component) MakeStatefulset() (*appsv1.StatefulSet, error) {
	objectmeta := c.MakeObjectMeta()
	service := c.MakeService()
	podTemplate, err := c.makePodTemplate()
	if err != nil {
		return nil, err
	}
	sts := &appsv1.StatefulSet{
		ObjectMeta: objectmeta,
		Spec: appsv1.StatefulSetSpec{
			Replicas: c.CommonSpec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: objectmeta.GetLabels(),
			},
			Template:             podTemplate,
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{},
			ServiceName:          service.GetName(),
		},
	}

	return sts, nil
}

func (c *Component) makePodTemplate() (corev1.PodTemplateSpec, error) {

	objectmeta := c.MakeObjectMeta()
	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				constants.LabelDefaultContainer: c.AppName,
			},
			Labels: objectmeta.GetLabels(),
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{},
			Containers: []corev1.Container{
				{
					Name:            c.AppName,
					Image:           c.CommonSpec.Image,
					ImagePullPolicy: c.CommonSpec.ImagePullPolicy,
					Resources:       c.CommonSpec.Resources,
					VolumeMounts:    []corev1.VolumeMount{},
					LivenessProbe:   &corev1.Probe{},
				},
			},
			NodeSelector:     c.CommonSpec.NodeSelector,
			SecurityContext:  c.CommonSpec.SecurityContext,
			ImagePullSecrets: c.CommonSpec.ImagePullSecrets,
			Tolerations:      c.CommonSpec.Tolerations,
			Affinity:         c.CommonSpec.Affinity,
		},
	}

	if c.CommonSpec.PodMetadata != nil {
		mergeStringMap(podTemplate.Annotations, c.CommonSpec.PodMetadata.Annotations)
		mergeStringMap(podTemplate.Labels, c.CommonSpec.PodMetadata.Labels)
	}

	if len(c.CommonSpec.Containers.Raw) > 0 {
		embeddedContainers, err := util.DecodeRawToContainers(c.CommonSpec.Containers)
		if err != nil {
			return podTemplate, fmt.Errorf("failed to decode containers: %w", err)
		}
		containers, err := k8sutil.MergePatchContainers(podTemplate.Spec.Containers, embeddedContainers)
		if err != nil {
			return podTemplate, fmt.Errorf("failed to merge containers spec: %w", err)
		}
		podTemplate.Spec.Containers = containers
	}

	return podTemplate, nil
}

func (c *Component) MakeService() *corev1.Service {
	objectmeta := c.MakeObjectMeta()
	objectmeta.SetName(objectmeta.GetName() + constants.ServiceNameSuffix)
	return &corev1.Service{
		ObjectMeta: objectmeta,
		Spec: corev1.ServiceSpec{
			Selector: objectmeta.Labels,
			Ports:    []corev1.ServicePort{},
		},
	}
}

func (c *Component) MakeConfigMap(data map[string]string) *corev1.ConfigMap {
	objectmeta := c.MakeObjectMeta()
	return &corev1.ConfigMap{
		ObjectMeta: objectmeta,
		Data:       copyStringMap(data),
	}
}

func (c *Component) MakeSecret(data map[string][]byte) *corev1.Secret {
	objectmeta := c.MakeObjectMeta()
	return &corev1.Secret{
		ObjectMeta: objectmeta,
		Data:       copyBinaryMap(data),
	}
}
