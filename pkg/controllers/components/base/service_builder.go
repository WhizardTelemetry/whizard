package base

import (
	corev1 "k8s.io/api/core/v1"
)

type ServiceOption func(*corev1.Service)

// ApplyServiceOptions applies options to an existing Service instance.
func ApplyServiceOptions(svc *corev1.Service, opts ...ServiceOption) {
	if svc == nil {
		return
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(svc)
	}
}

func WithServiceType(serviceType corev1.ServiceType) ServiceOption {
	return func(service *corev1.Service) {
		service.Spec.Type = serviceType
	}
}

func WithServiceSelector(selector map[string]string) ServiceOption {
	return func(service *corev1.Service) {
		service.Spec.Selector = mergeStringMap(service.Spec.Selector, selector)
	}
}

func WithServicePorts(ports ...corev1.ServicePort) ServiceOption {
	return func(service *corev1.Service) {
		service.Spec.Ports = append([]corev1.ServicePort{}, ports...)
	}
}

// WithHeadless configures the service as headless.
func WithHeadless() ServiceOption {
	return func(service *corev1.Service) {
		service.Spec.ClusterIP = corev1.ClusterIPNone
	}
}

func WithServiceLabels(labels map[string]string) ServiceOption {
	return func(service *corev1.Service) {
		service.Labels = mergeStringMap(service.Labels, labels)
	}
}

func WithServiceAnnotations(annotations map[string]string) ServiceOption {
	return func(service *corev1.Service) {
		service.Annotations = mergeStringMap(service.Annotations, annotations)
	}
}
