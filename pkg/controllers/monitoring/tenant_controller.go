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
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	monitoringv1alpha1 "github.com/kubesphere/paodin/pkg/api/monitoring/v1alpha1"
	"github.com/kubesphere/paodin/pkg/controllers/monitoring/options"
	"github.com/kubesphere/paodin/pkg/controllers/monitoring/resources"
	"github.com/kubesphere/paodin/pkg/controllers/monitoring/resources/tenant"
)

// TenantReconciler reconciles a Tenant object
type TenantReconciler struct {
	DefaulterValidator TenantDefaulterValidator
	client.Client
	Scheme  *runtime.Scheme
	Context context.Context

	DefaultTenantPerIngestor       int
	DefaultIngestorRetentionPeriod time.Duration
	DeleteIngestorEventChan        chan tenant.DeleteIngestorEvent
}

//+kubebuilder:rbac:groups=monitoring.paodin.io,resources=tenants,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.paodin.io,resources=tenants/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=monitoring.paodin.io,resources=tenants/finalizers,verbs=update
//+kubebuilder:rbac:groups=monitoring.paodin.io,resources=service,verbs=get;list;watch
//+kubebuilder:rbac:groups=monitoring.paodin.io,resources=storage,verbs=get;list;watch
//+kubebuilder:rbac:groups=monitoring.paodin.io,resources=thanosreceiveingestors,verbs=get;list;watch
//+kubebuilder:rbac:groups=monitoring.paodin.io,resources=thanosrulers,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Service object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile

func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx).WithValues("tenant", req.NamespacedName)
	l.Info("sync")

	instance := &monitoringv1alpha1.Tenant{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	instance, err = r.DefaulterValidator(instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	baseReconciler := resources.BaseReconciler{
		Client:  r.Client,
		Log:     l,
		Scheme:  r.Scheme,
		Context: ctx,
	}
	if err := tenant.New(baseReconciler, instance, r.DefaultTenantPerIngestor, r.DefaultIngestorRetentionPeriod, r.DeleteIngestorEventChan).Reconcile(); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1alpha1.Tenant{}).
		Watches(&source.Kind{Type: &monitoringv1alpha1.ThanosReceiveIngestor{}},
			handler.EnqueueRequestsFromMapFunc(r.mapToTenantbyLabelFunc)).
		Watches(&source.Kind{Type: &monitoringv1alpha1.Service{}},
			handler.EnqueueRequestsFromMapFunc(r.mapToTenantbyService)).
		Watches(&source.Kind{Type: &monitoringv1alpha1.Storage{}},
			handler.EnqueueRequestsFromMapFunc(r.mapToTenantbyStorage)).
		Owns(&monitoringv1alpha1.ThanosRuler{}).
		Complete(r)
}

func (r *TenantReconciler) Recycle() error {
	return tenant.NewIngestorRecycleController(r.Client, r.Scheme, r.Context, r.DeleteIngestorEventChan).Recycle()
}

func (r *TenantReconciler) mapToTenantbyLabelFunc(o client.Object) []reconcile.Request {

	labels := o.GetLabels()
	var tenantsName []string
	if tenants, ok := labels[monitoringv1alpha1.MonitoringPaodinTenant]; ok {
		tenantsName = strings.Split(tenants, ".")
	}

	var reqs []reconcile.Request
	for _, tenantName := range tenantsName {
		reqs = append(reqs, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: tenantName,
			},
		})
	}

	return reqs
}

func (r *TenantReconciler) mapToTenantbyService(o client.Object) []reconcile.Request {

	var tenantList monitoringv1alpha1.TenantList
	if err := r.Client.List(r.Context, &tenantList, client.MatchingLabels(monitoringv1alpha1.ManagedLabelByService(o))); err != nil {
		log.FromContext(r.Context).WithValues("tenantlist", "").Error(err, "")
		return nil
	}

	var reqs []reconcile.Request
	for _, ingestor := range tenantList.Items {
		reqs = append(reqs, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: ingestor.Name,
			},
		})
	}

	return reqs
}

func (r *TenantReconciler) mapToTenantbyStorage(o client.Object) []reconcile.Request {
	var tenantList monitoringv1alpha1.TenantList

	if err := r.Client.List(r.Context, &tenantList, client.MatchingLabels(monitoringv1alpha1.ManagedLabelByStorage(o))); err != nil {
		log.FromContext(r.Context).WithValues("tenantlist", "").Error(err, "")
		return nil
	}

	var reqs []reconcile.Request
	for _, ingestor := range tenantList.Items {
		reqs = append(reqs, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: ingestor.Name,
			},
		})
	}

	return reqs
}

type TenantDefaulterValidator func(tenant *monitoringv1alpha1.Tenant) (*monitoringv1alpha1.Tenant, error)

func CreateTenantDefaulterValidator(opt options.Options) TenantDefaulterValidator {
	return func(tenant *monitoringv1alpha1.Tenant) (*monitoringv1alpha1.Tenant, error) {
		if tenant.Spec.Tenant == "" {
			tenant.Spec.Tenant = tenant.Name
		}
		return tenant, nil
	}
}
