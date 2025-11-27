package base

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

type ConfigMapOption func(*corev1.ConfigMap)
type SecretOption func(*corev1.Secret)

// ApplyConfigMapOptions updates an existing ConfigMap using the provided options.
func ApplyConfigMapOptions(cm *corev1.ConfigMap, opts ...ConfigMapOption) {
	if cm == nil {
		return
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(cm)
	}
}

// ApplySecretOptions updates an existing Secret using the provided options.
func ApplySecretOptions(secret *corev1.Secret, opts ...SecretOption) {
	if secret == nil {
		return
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(secret)
	}
}

func WithConfigMapData(data map[string]string) ConfigMapOption {
	return func(cm *corev1.ConfigMap) {
		cm.Data = copyStringMap(data)
	}
}

func WithConfigMapBinaryData(data map[string][]byte) ConfigMapOption {
	return func(cm *corev1.ConfigMap) {
		cm.BinaryData = copyBinaryMap(data)
	}
}

func WithConfigMapImmutable(immutable bool) ConfigMapOption {
	return func(cm *corev1.ConfigMap) {
		cm.Immutable = pointer.Bool(immutable)
	}
}

func WithSecretStringData(data map[string]string) SecretOption {
	return func(secret *corev1.Secret) {
		secret.StringData = copyStringMap(data)
	}
}

func WithSecretData(data map[string][]byte) SecretOption {
	return func(secret *corev1.Secret) {
		secret.Data = copyBinaryMap(data)
	}
}

func WithSecretType(secretType corev1.SecretType) SecretOption {
	return func(secret *corev1.Secret) {
		secret.Type = secretType
	}
}

func WithSecretImmutable(immutable bool) SecretOption {
	return func(secret *corev1.Secret) {
		secret.Immutable = pointer.Bool(immutable)
	}
}

func copyStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyBinaryMap(src map[string][]byte) map[string][]byte {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string][]byte, len(src))
	for k, v := range src {
		if v == nil {
			dst[k] = nil
			continue
		}
		buf := make([]byte, len(v))
		copy(buf, v)
		dst[k] = buf
	}
	return dst
}

func mergeStringMap(dst map[string]string, sources ...map[string]string) map[string]string {
	if dst == nil {
		dst = map[string]string{}
	}
	for _, src := range sources {
		for k, v := range src {
			dst[k] = v
		}
	}
	return dst
}
