/*
Copyright 2024 the Whizard Authors.

Licensed under Apache License, Version 2.0 with a few additional conditions.

You may obtain a copy of the License at

    https://github.com/WhizardTelemetry/whizard/blob/main/LICENSE
*/

package controllers

import (
	"context"

	"dario.cat/mergo"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	monitoringv1alpha1 "github.com/WhizardTelemetry/whizard/pkg/api/monitoring/v1alpha1"
	"github.com/WhizardTelemetry/whizard/pkg/constants"
	"github.com/WhizardTelemetry/whizard/pkg/controllers/resources"
	"github.com/WhizardTelemetry/whizard/pkg/controllers/resources/queryfrontend"
	"github.com/WhizardTelemetry/whizard/pkg/util"
)

// QueryFrontendReconciler reconciles a Service object
type QueryFrontendReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Context context.Context
}

//+kubebuilder:rbac:groups=monitoring.whizard.io,resources=queryfrontends,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.whizard.io,resources=queryfrontends/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=monitoring.whizard.io,resources=queryfrontends/finalizers,verbs=update
//+kubebuilder:rbac:groups=monitoring.whizard.io,resources=services,verbs=get;list;watch
//+kubebuilder:rbac:groups=monitoring.whizard.io,resources=queries,verbs=get;list;watch
//+kubebuilder:rbac:groups=monitoring.whizard.io,resources=tenants,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=services;configmaps;secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the Service object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *QueryFrontendReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx).WithValues("query-frontend", req.NamespacedName)

	l.Info("sync")

	instance := &monitoringv1alpha1.QueryFrontend{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if instance.Labels == nil ||
		instance.Labels[constants.ServiceLabelKey] == "" {
		return ctrl.Result{}, nil
	}

	service := &monitoringv1alpha1.Service{}
	if err := r.Get(ctx, *util.ServiceNamespacedName(&instance.ObjectMeta), service); err != nil {
		return ctrl.Result{}, err
	}

	if _, err := r.applyConfigurationFromQueryFrontendTemplateSpec(instance, resources.ApplyDefaults(service).Spec.QueryFrontendTemplateSpec); err != nil {
		return ctrl.Result{}, err
	}

	queryFrontendReconciler, err := queryfrontend.New(
		resources.BaseReconciler{
			Client:  r.Client,
			Log:     l,
			Scheme:  r.Scheme,
			Context: ctx,
		},
		instance,
	)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, queryFrontendReconciler.Reconcile()
}

// SetupWithManager sets up the controller with the Manager.
func (r *QueryFrontendReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1alpha1.QueryFrontend{}).
		Watches(&monitoringv1alpha1.Service{},
			handler.EnqueueRequestsFromMapFunc(r.mapFuncBySelectorFunc(util.ManagedLabelByService))).
		Watches(&monitoringv1alpha1.Query{},
			handler.EnqueueRequestsFromMapFunc(r.mapFuncBySelectorFunc(util.ManagedLabelBySameService))).
		Watches(&monitoringv1alpha1.Tenant{},
			handler.EnqueueRequestsFromMapFunc(r.mapFuncBySelectorFunc(util.ManagedLabelBySameService))).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

func (r *QueryFrontendReconciler) mapFuncBySelectorFunc(fn func(metav1.Object) map[string]string) handler.MapFunc {
	return func(ctx context.Context, o client.Object) []reconcile.Request {
		queryFrontendList := &monitoringv1alpha1.QueryFrontendList{}
		if err := r.Client.List(r.Context, queryFrontendList, client.MatchingLabels(fn(o))); err != nil {
			log.FromContext(r.Context).WithValues("queryFrontendList", "").Error(err, "")
			return nil
		}

		var reqs []reconcile.Request
		for _, item := range queryFrontendList.Items {
			reqs = append(reqs, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: item.Namespace,
					Name:      item.Name,
				},
			})
		}

		return reqs
	}
}

func (r *QueryFrontendReconciler) applyConfigurationFromQueryFrontendTemplateSpec(queryFrontend *monitoringv1alpha1.QueryFrontend, queryFrontendTemplateSpec monitoringv1alpha1.QueryFrontendSpec) (*monitoringv1alpha1.QueryFrontend, error) {

	err := mergo.Merge(&queryFrontend.Spec, queryFrontendTemplateSpec)

	return queryFrontend, err
}
