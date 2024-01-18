package router

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubesphere/whizard/pkg/api/monitoring/v1alpha1"
	"github.com/kubesphere/whizard/pkg/constants"
	"github.com/kubesphere/whizard/pkg/controllers/monitoring/resources"
)

const (
	configDir       = "/etc/whizard"
	hashringsFile   = "hashrings.json"
	envoyConfigFile = "envoy.yaml"
)

type Router struct {
	resources.BaseReconciler
	router *v1alpha1.Router
}

func New(reconciler resources.BaseReconciler, r *v1alpha1.Router) (*Router, error) {
	if err := reconciler.SetService(r); err != nil {
		return nil, err
	}
	return &Router{
		BaseReconciler: reconciler,
		router:         r,
	}, nil
}

func (r *Router) labels() map[string]string {
	labels := r.BaseLabels()
	labels[constants.LabelNameAppName] = constants.AppNameRouter
	labels[constants.LabelNameAppManagedBy] = r.Service.Name
	return labels
}

func (r *Router) name(nameSuffix ...string) string {
	return r.QualifiedName(constants.AppNameRouter, r.router.Name, nameSuffix...)
}

func (r *Router) meta(name string) metav1.ObjectMeta {

	return metav1.ObjectMeta{
		Name:      name,
		Namespace: r.Service.Namespace,
		Labels:    r.labels(),
	}
}

func (r *Router) HttpAddr() string {
	return fmt.Sprintf("http://%s.%s.svc:%d",
		r.name(constants.ServiceNameSuffix), r.Service.Namespace, constants.HTTPPort)
}

func (r *Router) RemoteWriteAddr() string {
	return fmt.Sprintf("http://%s.%s.svc:%d",
		r.name(constants.ServiceNameSuffix), r.Service.Namespace, constants.RemoteWritePort)
}

func (r *Router) RemoteWriteHTTPSAddr() string {
	return fmt.Sprintf("https://%s.%s.svc:%d",
		r.name(constants.ServiceNameSuffix), r.Service.Namespace, constants.RemoteWritePort)
}

func (r *Router) Reconcile() error {
	return r.ReconcileResources([]resources.Resource{
		r.hashringsConfigMap,
		r.webConfigSecret,
		r.deployment,
		r.service,
	})
}
