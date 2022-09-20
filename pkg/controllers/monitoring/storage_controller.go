/*
Copyright 2021 The KubeSphere authors.
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

package monitoring

import (
	"context"

	monitoringv1alpha1 "github.com/kubesphere/whizard/pkg/api/monitoring/v1alpha1"
	"github.com/kubesphere/whizard/pkg/controllers/monitoring/options"
	"github.com/kubesphere/whizard/pkg/controllers/monitoring/resources"
	"github.com/kubesphere/whizard/pkg/controllers/monitoring/resources/storage"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// StorageReconciler reconciles a Storage object
type StorageReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Context context.Context

	Options *options.StorageOptions
}

//+kubebuilder:rbac:groups=monitoring.whizard.io,resources=storages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Service object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *StorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx).WithValues("Storage", req.NamespacedName)

	l.Info("sync")

	instance := &monitoringv1alpha1.Storage{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	instance = r.validate(instance)

	baseReconciler := resources.BaseReconciler{
		Client:  r.Client,
		Log:     l,
		Scheme:  r.Scheme,
		Context: ctx,
	}
	if err := storage.New(baseReconciler, instance).Reconcile(); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1alpha1.Storage{}).
		Watches(&source.Kind{Type: &corev1.Secret{}},
			handler.EnqueueRequestsFromMapFunc(r.mapToStoragebySecretRefFunc)).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func (r *StorageReconciler) mapToStoragebySecretRefFunc(o client.Object) []reconcile.Request {
	var reqs []reconcile.Request
	var storageList monitoringv1alpha1.StorageList
	if err := r.List(r.Context, &storageList, client.InNamespace(o.GetNamespace())); err != nil {
		return reqs
	}

	name := o.GetName()
	for _, s := range storageList.Items {
		if s.Spec.S3 == nil {
			continue
		}

		s3 := s.Spec.S3
		tls := s3.HTTPConfig.TLSConfig
		if s3.AccessKey.Name == name ||
			s3.SecretKey.Name == name ||
			(tls.CA != nil && tls.CA.Name == name) ||
			(tls.Key != nil && tls.Key.Name == name) ||
			(tls.Cert != nil && tls.Cert.Name == name) {
			reqs = append(reqs, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: s.GetNamespace(),
					Name:      s.GetName(),
				}})
		}
	}

	return reqs
}

func (r *StorageReconciler) validate(storage *monitoringv1alpha1.Storage) *monitoringv1alpha1.Storage {

	if storage.Spec.BlockManager != nil && storage.Spec.BlockManager.Enable != nil && *(storage.Spec.BlockManager.Enable) {
		r.Options.BlockManager.Apply(&storage.Spec.BlockManager.CommonSpec)

		if storage.Spec.BlockManager.BlockSyncInterval == nil || storage.Spec.BlockManager.BlockSyncInterval.Duration == 0 {
			storage.Spec.BlockManager.BlockSyncInterval = r.Options.BlockManager.BlockSyncInterval
		}

		if storage.Spec.BlockManager.ServiceAccountName == "" {
			storage.Spec.BlockManager.ServiceAccountName = r.Options.BlockManager.ServiceAccountName
		}

		if storage.Spec.BlockManager.GC != nil &&
			storage.Spec.BlockManager.GC.Enable != nil &&
			*storage.Spec.BlockManager.GC.Enable {
			if storage.Spec.BlockManager.GC.Image == "" {
				storage.Spec.BlockManager.GC.Image = r.Options.BlockManager.GC.Image
			}
			if storage.Spec.BlockManager.GC.ImagePullPolicy == "" {
				storage.Spec.BlockManager.GC.ImagePullPolicy = r.Options.BlockManager.GC.ImagePullPolicy
			}
			if storage.Spec.BlockManager.GC.Resources.Limits == nil {
				storage.Spec.BlockManager.GC.Resources.Limits = r.Options.BlockManager.GC.Resources.Limits
			}
			if storage.Spec.BlockManager.GC.Resources.Requests == nil {
				storage.Spec.BlockManager.GC.Resources.Requests = r.Options.BlockManager.GC.Resources.Requests
			}
			if storage.Spec.BlockManager.GC.GCInterval == nil ||
				storage.Spec.BlockManager.GC.GCInterval.Duration == 0 {
				storage.Spec.BlockManager.GC.GCInterval = r.Options.BlockManager.GC.GCInterval
			}
			if storage.Spec.BlockManager.GC.CleanupTimeout == nil ||
				storage.Spec.BlockManager.GC.GCInterval.Duration == 0 {
				storage.Spec.BlockManager.GC.CleanupTimeout = r.Options.BlockManager.GC.CleanupTimeout
			}
		}
	}

	return storage
}
