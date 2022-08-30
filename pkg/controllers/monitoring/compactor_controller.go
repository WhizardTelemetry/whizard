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
	"github.com/kubesphere/whizard/pkg/constants"
	"github.com/kubesphere/whizard/pkg/controllers/monitoring/options"
	"github.com/kubesphere/whizard/pkg/controllers/monitoring/resources"
	"github.com/kubesphere/whizard/pkg/controllers/monitoring/resources/compactor"
	"github.com/kubesphere/whizard/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CompactorReconciler reconciles a compactor object
type CompactorReconciler struct {
	DefaulterValidator CompactorDefaulterValidator
	client.Client
	Scheme  *runtime.Scheme
	Context context.Context

	Options options.CompactorOptions
}

//+kubebuilder:rbac:groups=monitoring.whizard.io,resources=compactors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.whizard.io,resources=compactors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=monitoring.whizard.io,resources=compactors/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Service object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *CompactorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx).WithValues("compactor", req.NamespacedName)

	l.Info("sync")

	instance := &monitoringv1alpha1.Compactor{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if instance.GetDeletionTimestamp().IsZero() {
		if len(instance.Finalizers) == 0 {
			instance.Finalizers = append(instance.Finalizers, constants.FinalizerDeletePVC)
		}
	} else {
		if err := util.DeletePVC(r.Context, r.Client, instance); err != nil {
			return ctrl.Result{}, err
		}

		instance.Finalizers = nil
		return ctrl.Result{}, r.Client.Update(r.Context, instance)
	}

	instance, err = r.DefaulterValidator(instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(instance.Spec.Tenants) == 0 {
		klog.V(3).Infof("ignore compactor %s/%s because of empty tenants", instance.Name, instance.Namespace)
		return ctrl.Result{}, nil
	}

	baseReconciler := resources.BaseReconciler{
		Client:  r.Client,
		Log:     l,
		Scheme:  r.Scheme,
		Context: ctx,
	}

	if err := compactor.New(baseReconciler, instance).Reconcile(); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CompactorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1alpha1.Compactor{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

type CompactorDefaulterValidator func(compactor *monitoringv1alpha1.Compactor) (*monitoringv1alpha1.Compactor, error)

func CreateCompactorDefaulterValidator(opt options.Options) CompactorDefaulterValidator {
	var replicas int32 = 1

	return func(compactor *monitoringv1alpha1.Compactor) (*monitoringv1alpha1.Compactor, error) {

		if compactor.Spec.Image == "" {
			compactor.Spec.Image = opt.WhizardImage
		}

		if compactor.Spec.ImagePullPolicy == "" {
			compactor.Spec.ImagePullPolicy = opt.Compactor.ImagePullPolicy
		}

		compactor.Spec.Replicas = &replicas

		if compactor.Spec.Affinity == nil {
			compactor.Spec.Affinity = opt.Compactor.Affinity
		}

		if compactor.Spec.Tolerations == nil {
			compactor.Spec.Tolerations = opt.Compactor.Tolerations
		}

		if compactor.Spec.NodeSelector == nil {
			compactor.Spec.NodeSelector = opt.Compactor.NodeSelector
		}

		if compactor.Spec.Resources.Requests == nil {
			compactor.Spec.Resources.Requests = opt.Compactor.Resources.Requests
		}

		if compactor.Spec.Resources.Limits == nil {
			compactor.Spec.Resources.Limits = opt.Compactor.Resources.Limits
		}

		if compactor.Spec.LogLevel == "" {
			compactor.Spec.LogLevel = opt.Compactor.LogLevel
		}

		if compactor.Spec.LogFormat == "" {
			compactor.Spec.LogFormat = opt.Compactor.LogFormat
		}

		if compactor.Spec.Flags == nil {
			compactor.Spec.Flags = opt.Compactor.Flags
		}

		if compactor.Spec.DataVolume != nil {
			compactor.Spec.DataVolume = opt.Compactor.DataVolume
		}

		return compactor, nil
	}
}
