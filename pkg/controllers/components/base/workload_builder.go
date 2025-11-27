package base

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type PodTemplateMutator func(*corev1.PodTemplateSpec)
type PodSpecMutator func(*corev1.PodSpec)
type ContainerMutator func(*corev1.Container)
type StatefulSetMutator func(*appsv1.StatefulSet)
type DeploymentMutator func(*appsv1.Deployment)

type DeploymentTemplateMutator interface {
	applyDeployment(*appsv1.Deployment)
}

type StatefulSetTemplateMutator interface {
	apply(*appsv1.StatefulSet)
}

func (m PodTemplateMutator) applyDeployment(dep *appsv1.Deployment) {
	mutatePodTemplate(&dep.Spec.Template, m)
}

func (m PodTemplateMutator) apply(sts *appsv1.StatefulSet) {
	mutatePodTemplate(&sts.Spec.Template, m)
}

func (m StatefulSetMutator) apply(sts *appsv1.StatefulSet) {
	if m == nil {
		return
	}
	m(sts)
}

func (m DeploymentMutator) applyDeployment(dep *appsv1.Deployment) {
	if m == nil {
		return
	}
	m(dep)
}

// ApplyDeployment applies mutators to a Deployment's spec/template.
func ApplyDeployment(dep *appsv1.Deployment, mutators ...DeploymentTemplateMutator) {
	for _, mutate := range mutators {
		if mutate == nil {
			continue
		}
		mutate.applyDeployment(dep)
	}
}

// ApplyStatefulSet applies mutators to a StatefulSet's pod template/spec.
func ApplyStatefulSet(sts *appsv1.StatefulSet, mutators ...StatefulSetTemplateMutator) {
	for _, mutate := range mutators {
		if mutate == nil {
			continue
		}
		mutate.apply(sts)
	}
}

// WithPodLabels merges labels into the pod template.
func WithPodLabels(labels map[string]string) PodTemplateMutator {
	return func(t *corev1.PodTemplateSpec) {
		t.Labels = mergeStringMap(t.Labels, labels)
	}
}

// WithPodAnnotations merges annotations into the pod template.
func WithPodAnnotations(annotations map[string]string) PodTemplateMutator {
	return func(t *corev1.PodTemplateSpec) {
		t.Annotations = mergeStringMap(t.Annotations, annotations)
	}
}

// WithPodSpec applies PodSpec-level mutations.
func WithPodSpec(mutators ...PodSpecMutator) PodTemplateMutator {
	return func(t *corev1.PodTemplateSpec) {
		for _, mutate := range mutators {
			if mutate == nil {
				continue
			}
			mutate(&t.Spec)
		}
	}
}

// WithNodeSelector merges node selector constraints.
func WithNodeSelector(selector map[string]string) PodSpecMutator {
	return func(spec *corev1.PodSpec) {
		spec.NodeSelector = mergeStringMap(spec.NodeSelector, selector)
	}
}

// WithTolerations overwrites tolerations with the provided list.
func WithTolerations(tolerations []corev1.Toleration) PodSpecMutator {
	return func(spec *corev1.PodSpec) {
		spec.Tolerations = append([]corev1.Toleration{}, tolerations...)
	}
}

// WithAffinity sets pod affinity rules.
func WithAffinity(affinity *corev1.Affinity) PodSpecMutator {
	return func(spec *corev1.PodSpec) {
		spec.Affinity = affinity
	}
}

// WithSecurityContext sets the pod security context.
func WithSecurityContext(ctx *corev1.PodSecurityContext) PodSpecMutator {
	return func(spec *corev1.PodSpec) {
		spec.SecurityContext = ctx
	}
}

// WithImagePullSecrets overwrites image pull secrets.
func WithImagePullSecrets(secrets []corev1.LocalObjectReference) PodSpecMutator {
	return func(spec *corev1.PodSpec) {
		spec.ImagePullSecrets = append([]corev1.LocalObjectReference{}, secrets...)
	}
}

// WithServiceAccount sets the service account name if provided.
func WithServiceAccount(name string) PodSpecMutator {
	return func(spec *corev1.PodSpec) {
		if name != "" {
			spec.ServiceAccountName = name
		}
	}
}

// AppendVolumes appends volumes to the pod spec.
func AppendVolumes(volumes ...corev1.Volume) PodSpecMutator {
	return func(spec *corev1.PodSpec) {
		spec.Volumes = append(spec.Volumes, volumes...)
	}
}

// AppendVolumeClaimTemplates appends volume claim templates to the StatefulSet spec.
func AppendVolumeClaimTemplates(templates ...corev1.PersistentVolumeClaim) StatefulSetMutator {
	return func(sts *appsv1.StatefulSet) {
		sts.Spec.VolumeClaimTemplates = append(sts.Spec.VolumeClaimTemplates, templates...)
	}
}

// WithPersistentVolumeClaimRetentionPolicy sets the PVC retention policy on the StatefulSet.
func WithPersistentVolumeClaimRetentionPolicy(policy *appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy) StatefulSetMutator {
	return func(sts *appsv1.StatefulSet) {
		sts.Spec.PersistentVolumeClaimRetentionPolicy = policy
	}
}

// WithDeploymentStrategy sets Deployment update strategy.
func WithDeploymentStrategy(strategy appsv1.DeploymentStrategy) DeploymentMutator {
	return func(dep *appsv1.Deployment) {
		dep.Spec.Strategy = strategy
	}
}

// AppendInitContainers appends init containers to the pod spec.
func AppendInitContainers(containers ...corev1.Container) PodSpecMutator {
	return func(spec *corev1.PodSpec) {
		spec.InitContainers = append(spec.InitContainers, containers...)
	}
}

// AppendContainers appends containers to the pod spec.
func AppendContainers(containers ...corev1.Container) PodSpecMutator {
	return func(spec *corev1.PodSpec) {
		spec.Containers = append(spec.Containers, containers...)
	}
}

// ForContainer applies container-level mutations to the named container.
func ForContainer(name string, mutators ...ContainerMutator) PodSpecMutator {
	return func(spec *corev1.PodSpec) {
		for i := range spec.Containers {
			if spec.Containers[i].Name != name {
				continue
			}
			for _, mutate := range mutators {
				if mutate == nil {
					continue
				}
				mutate(&spec.Containers[i])
			}
		}
	}
}

// AppendEnv appends env vars to a container.
func AppendEnv(envs ...corev1.EnvVar) ContainerMutator {
	return func(c *corev1.Container) {
		c.Env = append(c.Env, envs...)
	}
}

// WithLivenessProbe sets the container liveness probe.
func WithLivenessProbe(probe *corev1.Probe) ContainerMutator {
	return func(c *corev1.Container) {
		c.LivenessProbe = probe
	}
}

// WithReadinessProbe sets the container readiness probe.
func WithReadinessProbe(probe *corev1.Probe) ContainerMutator {
	return func(c *corev1.Container) {
		c.ReadinessProbe = probe
	}
}

// WithStartupProbe sets the container startup probe.
func WithStartupProbe(probe *corev1.Probe) ContainerMutator {
	return func(c *corev1.Container) {
		c.StartupProbe = probe
	}
}

// AppendPorts appends container ports.
func AppendPorts(ports ...corev1.ContainerPort) ContainerMutator {
	return func(c *corev1.Container) {
		c.Ports = append(c.Ports, ports...)
	}
}

// AppendVolumeMounts appends volume mounts to a container.
func AppendVolumeMounts(mounts ...corev1.VolumeMount) ContainerMutator {
	return func(c *corev1.Container) {
		c.VolumeMounts = append(c.VolumeMounts, mounts...)
	}
}

// AppendArgs appends command-line arguments to a container.
func AppendArgs(args ...string) ContainerMutator {
	return func(c *corev1.Container) {
		c.Args = append(c.Args, args...)
	}
}

func mutatePodTemplate(t *corev1.PodTemplateSpec, mutators ...PodTemplateMutator) {
	for _, mutate := range mutators {
		if mutate == nil {
			continue
		}
		mutate(t)
	}
}
