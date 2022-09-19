package router

import (
	"github.com/kubesphere/whizard/pkg/constants"
	"github.com/kubesphere/whizard/pkg/controllers/monitoring/resources"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *Router) service() (runtime.Object, resources.Operation, error) {
	var s = &corev1.Service{ObjectMeta: r.meta(r.name(constants.ServiceNameSuffix))}

	if r.router == nil {
		return s, resources.OperationDelete, nil
	}

	s.Spec = corev1.ServiceSpec{
		Type:     corev1.ServiceTypeClusterIP,
		Selector: r.labels(),
		Ports: []corev1.ServicePort{
			{
				Protocol: corev1.ProtocolTCP,
				Name:     constants.GRPCPortName,
				Port:     constants.GRPCPort,
			},
			{
				Protocol: corev1.ProtocolTCP,
				Name:     constants.HTTPPortName,
				Port:     constants.HTTPPort,
			},
			{
				Protocol: corev1.ProtocolTCP,
				Name:     constants.RemoteWritePortName,
				Port:     constants.RemoteWritePort,
			},
		},
	}
	return s, resources.OperationCreateOrUpdate, ctrl.SetControllerReference(r.router, s, r.Scheme)
}
