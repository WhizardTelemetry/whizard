package store

import (
	"github.com/kubesphere/paodin/pkg/api/monitoring/v1alpha1"
	"github.com/kubesphere/paodin/pkg/controllers/monitoring/options"
	"github.com/kubesphere/paodin/pkg/controllers/monitoring/resources"
	"github.com/kubesphere/paodin/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

type Store struct {
	resources.BaseReconciler
	store *v1alpha1.Store
	options.StoreOptions
}

func New(reconciler resources.BaseReconciler, instance *v1alpha1.Store, o options.StoreOptions) *Store {
	return &Store{
		BaseReconciler: reconciler,
		store:          instance,
		StoreOptions:   o,
	}
}

func (r *Store) labels() map[string]string {
	labels := r.BaseLabels()
	labels[resources.LabelNameAppName] = resources.AppNameStore
	labels[resources.LabelNameAppManagedBy] = r.store.Name
	util.AppendLabel(labels, r.store.Labels)
	return labels
}

func (r *Store) meta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:            name,
		Namespace:       r.store.Namespace,
		Labels:          r.labels(),
		OwnerReferences: r.OwnerReferences(),
	}
}

func (r *Store) OwnerReferences() []metav1.OwnerReference {
	return []metav1.OwnerReference{
		{
			APIVersion: r.store.APIVersion,
			Kind:       r.store.Kind,
			Name:       r.store.Name,
			UID:        r.store.UID,
			Controller: pointer.BoolPtr(true),
		},
	}
}

func (r *Store) Reconcile() error {
	return r.ReconcileResources([]resources.Resource{
		r.statefulSet,
		r.horizontalPodAutoscaler,
		r.service,
	})
}
