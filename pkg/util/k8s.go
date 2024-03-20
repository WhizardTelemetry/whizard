package util

import (
	"bytes"
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateOrUpdateStatefulSet(ctx context.Context, cli client.Client, desired *appsv1.StatefulSet) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var current = &appsv1.StatefulSet{}
		err := cli.Get(ctx, client.ObjectKeyFromObject(desired), current)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
			return cli.Create(ctx, desired)
		}

		mergeMetadata(&desired.ObjectMeta, &current.ObjectMeta)

		return cli.Update(ctx, desired)
	})
}

func CreateOrUpdateDeployment(ctx context.Context, cli client.Client, desired *appsv1.Deployment) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var current = &appsv1.Deployment{}
		err := cli.Get(ctx, client.ObjectKeyFromObject(desired), current)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
			return cli.Create(ctx, desired)
		}

		mergeMetadata(&desired.ObjectMeta, &current.ObjectMeta)

		return cli.Update(ctx, desired)
	})
}

func CreateOrUpdateService(ctx context.Context, cli client.Client, desired *corev1.Service) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var current = &corev1.Service{}
		err := cli.Get(ctx, client.ObjectKeyFromObject(desired), current)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
			return cli.Create(ctx, desired)
		}

		// Apply immutable fields from the existing service.
		desired.Spec.IPFamilies = current.Spec.IPFamilies
		desired.Spec.IPFamilyPolicy = current.Spec.IPFamilyPolicy
		desired.Spec.ClusterIP = current.Spec.ClusterIP
		desired.Spec.ClusterIPs = current.Spec.ClusterIPs

		mergeMetadata(&desired.ObjectMeta, &current.ObjectMeta)

		return cli.Update(ctx, desired)
	})
}

func CreateOrUpdateConfigMap(ctx context.Context, cli client.Client, desired *corev1.ConfigMap) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var current = &appsv1.Deployment{}
		err := cli.Get(ctx, client.ObjectKeyFromObject(desired), current)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
			return cli.Create(ctx, desired)
		}

		mergeMetadata(&desired.ObjectMeta, &current.ObjectMeta)

		if apiequality.Semantic.DeepEqual(current, desired) {
			return nil
		}

		return cli.Update(ctx, desired)
	})
}

func CreateOrUpdateServiceAccount(ctx context.Context, cli client.Client, desired *corev1.ServiceAccount) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var current = &corev1.ServiceAccount{}
		err := cli.Get(ctx, client.ObjectKeyFromObject(desired), current)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
			return cli.Create(ctx, desired)
		}

		mergeMetadata(&desired.ObjectMeta, &current.ObjectMeta)

		if apiequality.Semantic.DeepEqual(current, desired) {
			return nil
		}

		desired.Secrets = current.Secrets // ignoring secrets update

		return cli.Update(ctx, desired)
	})
}

func CreateOrUpdate(ctx context.Context, cli client.Client, desired client.Object) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var current = desired.DeepCopyObject().(client.Object)
		err := cli.Get(ctx, client.ObjectKeyFromObject(desired), current)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
			return cli.Create(ctx, desired)
		}

		desired.SetResourceVersion(current.GetResourceVersion())
		annotation := labels.Merge(current.GetAnnotations(), desired.GetAnnotations())
		desired.SetAnnotations(annotation)
		ls := labels.Merge(current.GetLabels(), desired.GetLabels())
		desired.SetLabels(ls)

		if apiequality.Semantic.DeepEqual(current, desired) {
			return nil
		}
		return cli.Update(ctx, desired)
	})
}

func CreateIfNotExists(ctx context.Context, cli client.Client, desired client.Object) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var current = desired.DeepCopyObject().(client.Object)
		err := cli.Get(ctx, client.ObjectKeyFromObject(desired), current)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
			return cli.Create(ctx, desired)
		}
		return nil
	})
}

func mergeMetadata(new, old *metav1.ObjectMeta) {
	new.ResourceVersion = old.ResourceVersion
	new.Labels = labels.Merge(old.Labels, new.Labels)
	new.Annotations = labels.Merge(old.Annotations, new.Annotations)
}

// IndexOwnerRef returns the index of the owner reference in the slice if found, or -1.
func IndexOwnerRef(ownerReferences []metav1.OwnerReference, ref metav1.OwnerReference) int {
	for index, r := range ownerReferences {
		if referSameObject(r, ref) {
			return index
		}
	}
	return -1
}

// Returns true if a and b point to the same object.
func referSameObject(a, b metav1.OwnerReference) bool {
	aGV, err := schema.ParseGroupVersion(a.APIVersion)
	if err != nil {
		return false
	}

	bGV, err := schema.ParseGroupVersion(b.APIVersion)
	if err != nil {
		return false
	}

	return aGV.Group == bGV.Group && a.Kind == b.Kind && a.Name == b.Name
}

func DeletePVC(ctx context.Context, c client.Client, obj runtime.Object) error {

	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      accessor.GetName(),
			Namespace: accessor.GetNamespace(),
		},
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(sts), sts); err != nil {
		return err
	}

	replicas := 1
	if sts.Spec.Replicas != nil {
		replicas = int(*sts.Spec.Replicas)
	}
	for _, item := range sts.Spec.VolumeClaimTemplates {
		for index := 0; index < replicas; index++ {
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%s-%d", item.Name, sts.Name, index),
					Namespace: sts.Namespace,
				},
			}

			if err := c.Delete(ctx, pvc); err != nil {
				if !IsNotFound(err) {
					return err
				}
			}
		}
	}

	return nil
}

func DecodeRawToContainers(raw runtime.RawExtension) ([]corev1.Container, error) {
	var containers []corev1.Container

	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(raw.Raw), len(raw.Raw))
	if err := decoder.Decode(&containers); err != nil {
		return nil, err
	}
	return containers, nil
}
