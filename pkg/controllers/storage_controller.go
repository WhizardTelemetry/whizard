/*
Copyright 2021 The WhizardTelemetry Authors.

This program is free software: you can redistribute it and/or modify
it under the terms of the Server Side Public License, version 1,
as published by MongoDB, Inc.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
Server Side Public License for more details.

You should have received a copy of the Server Side Public License
along with this program. If not, see
<http://www.mongodb.com/licensing/server-side-public-license>.

As a special exception, the copyright holders give permission to link the
code of portions of this program with the OpenSSL library under certain
conditions as described in each individual source file and distribute
linked combinations including the program with the OpenSSL library. You
must comply with the Server Side Public License in all respects for
all of the code used other than as permitted herein. If you modify file(s)
with this exception, you may extend this exception to your version of the
file(s), but you are not obligated to do so. If you do not wish to do so,
delete this exception statement from your version. If you delete this
exception statement from all source files in the program, then also delete
it in the license file.
*/

package controllers

import (
	"context"

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

	monitoringv1alpha1 "github.com/WhizardTelemetry/whizard/pkg/api/monitoring/v1alpha1"
	"github.com/WhizardTelemetry/whizard/pkg/controllers/resources"
	"github.com/WhizardTelemetry/whizard/pkg/controllers/resources/storage"
)

// StorageReconciler reconciles a Storage object
type StorageReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Context context.Context
}

//+kubebuilder:rbac:groups=monitoring.whizard.io,resources=storages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

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
		Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.mapToStoragebySecretRefFunc)).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func (r *StorageReconciler) mapToStoragebySecretRefFunc(ctx context.Context, o client.Object) []reconcile.Request {
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
