package base

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	"github.com/WhizardTelemetry/whizard/pkg/api/monitoring/v1alpha1"
	"github.com/WhizardTelemetry/whizard/pkg/constants"
)

func TestComponentStatefulSetBuild(t *testing.T) {
	comp := &v1alpha1.Compactor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "comp",
			Namespace: "ns",
			UID:       types.UID("uid"),
		},
		Spec: v1alpha1.CompactorSpec{
			CommonSpec: v1alpha1.CommonSpec{
				Image:           "comp:1.0",
				ImagePullPolicy: corev1.PullIfNotPresent,
				NodeSelector:    map[string]string{"disk": "ssd"},
				Tolerations:     []corev1.Toleration{{Key: "key"}},
				Affinity:        &corev1.Affinity{},
				SecurityContext: &corev1.PodSecurityContext{FSGroup: pointer.Int64(2000)},
				ImagePullSecrets: []corev1.LocalObjectReference{
					{Name: "pull-secret"},
				},
				PodMetadata: &v1alpha1.EmbeddedObjectMetadata{
					Labels:      map[string]string{"pod": "label"},
					Annotations: map[string]string{"pod": "anno"},
				},
			},
			Tenants: []string{"tenant-a"},
		},
	}

	component := Component{
		AppName:    constants.AppNameCompactor,
		Instance:   comp,
		CommonSpec: comp.Spec.CommonSpec,
	}

	sts, err := component.MakeStatefulset()
	if err != nil {
		t.Fatalf("make statefulset: %v", err)
	}

	pvc := corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "data"}}
	livenessProbe := &corev1.Probe{TimeoutSeconds: 5}

	ApplyStatefulSet(sts,
		WithPodLabels(map[string]string{"extra": "label"}),
		WithPodAnnotations(map[string]string{"anno": "yes"}),
		WithPodSpec(
			ForContainer(constants.AppNameCompactor,
				WithLivenessProbe(livenessProbe),
				AppendPorts(corev1.ContainerPort{Name: "http", ContainerPort: 8080}),
			),
		),
		AppendVolumeClaimTemplates(pvc),
		WithPersistentVolumeClaimRetentionPolicy(&appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
			WhenDeleted: appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
		}),
	)

	if sts.Name != comp.Name {
		t.Fatalf("unexpected statefulset name: %s", sts.Name)
	}
	if sts.Namespace != comp.Namespace {
		t.Fatalf("unexpected statefulset namespace: %s", sts.Namespace)
	}
	if len(sts.OwnerReferences) != 1 || sts.OwnerReferences[0].UID != comp.UID {
		t.Fatalf("owner reference not set: %#v", sts.OwnerReferences)
	}

	labels := sts.Spec.Template.Labels
	if labels[constants.LabelNameAppName] != constants.AppNameCompactor || labels["extra"] != "label" || labels["pod"] != "label" {
		t.Fatalf("pod labels not applied: %#v", labels)
	}
	if sts.Spec.Template.Annotations["anno"] != "yes" || sts.Spec.Template.Annotations["pod"] != "anno" {
		t.Fatalf("pod annotations not applied: %#v", sts.Spec.Template.Annotations)
	}

	if len(sts.Spec.VolumeClaimTemplates) != 1 || sts.Spec.VolumeClaimTemplates[0].Name != "data" {
		t.Fatalf("volume claim templates not applied: %#v", sts.Spec.VolumeClaimTemplates)
	}

	if sts.Spec.Template.Spec.NodeSelector["disk"] != "ssd" {
		t.Fatalf("node selector not propagated: %#v", sts.Spec.Template.Spec.NodeSelector)
	}
	if len(sts.Spec.Template.Spec.Tolerations) != 1 || sts.Spec.Template.Spec.Tolerations[0].Key != "key" {
		t.Fatalf("tolerations not propagated: %#v", sts.Spec.Template.Spec.Tolerations)
	}
	if sts.Spec.Template.Spec.SecurityContext == nil || sts.Spec.Template.Spec.SecurityContext.FSGroup == nil || *sts.Spec.Template.Spec.SecurityContext.FSGroup != 2000 {
		t.Fatalf("security context not propagated: %#v", sts.Spec.Template.Spec.SecurityContext)
	}
	if len(sts.Spec.Template.Spec.ImagePullSecrets) != 1 || sts.Spec.Template.Spec.ImagePullSecrets[0].Name != "pull-secret" {
		t.Fatalf("image pull secrets not propagated: %#v", sts.Spec.Template.Spec.ImagePullSecrets)
	}

	var mainContainer *corev1.Container
	for i := range sts.Spec.Template.Spec.Containers {
		if sts.Spec.Template.Spec.Containers[i].Name == constants.AppNameCompactor {
			mainContainer = &sts.Spec.Template.Spec.Containers[i]
			break
		}
	}
	if mainContainer == nil {
		t.Fatalf("main container not found in pod spec")
	}
	if mainContainer.Image != "comp:1.0" || mainContainer.ImagePullPolicy != corev1.PullIfNotPresent {
		t.Fatalf("container image settings not propagated: %s %s", mainContainer.Image, mainContainer.ImagePullPolicy)
	}
	if mainContainer.LivenessProbe != livenessProbe {
		t.Fatalf("liveness probe not applied")
	}
	if len(mainContainer.Ports) != 1 || mainContainer.Ports[0].ContainerPort != 8080 {
		t.Fatalf("container ports not applied: %#v", mainContainer.Ports)
	}
}
