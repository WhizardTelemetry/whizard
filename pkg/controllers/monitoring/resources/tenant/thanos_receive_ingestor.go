package tenant

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	monitoringv1alpha1 "github.com/kubesphere/paodin/pkg/api/monitoring/v1alpha1"
	"github.com/kubesphere/paodin/pkg/controllers/monitoring/resources"
	"github.com/kubesphere/paodin/pkg/util"
)

func (t *Tenant) receiveIngestor() error {

	// finalizers check,  when tenant cr is deleted, ObjectMeta.GetDeletionTimestamp() is not nil, remove finalizers field and call removeTenantFromIngestorbyName()
	if t.tenant.ObjectMeta.GetDeletionTimestamp().IsZero() {
		if !containsString(t.tenant.ObjectMeta.Finalizers, monitoringv1alpha1.FinalizerMonitoringPaodin) {
			t.tenant.ObjectMeta.Finalizers = append(t.tenant.ObjectMeta.Finalizers, monitoringv1alpha1.FinalizerMonitoringPaodin)
			if err := t.Client.Update(t.Context, t.tenant); err != nil {
				return err
			}
		}
	} else {
		if containsString(t.tenant.ObjectMeta.Finalizers, monitoringv1alpha1.FinalizerMonitoringPaodin) {
			if t.tenant.Status.ThanosResource != nil && t.tenant.Status.ThanosResource.ThanosReceiveIngestor != nil {
				if err := t.removeTenantFromIngestorbyName(t.tenant.Status.ThanosResource.ThanosReceiveIngestor.Namespace, t.tenant.Status.ThanosResource.ThanosReceiveIngestor.Name); err != nil {
					return err
				}
				t.tenant.Status.ThanosResource.ThanosReceiveIngestor = nil
			}
			t.tenant.ObjectMeta.Finalizers = removeString(t.tenant.Finalizers, monitoringv1alpha1.FinalizerMonitoringPaodin)
			return t.Client.Update(t.Context, t.tenant)
		}
	}

	// Check if ingestor needs to be created or reset
	ingestor := &monitoringv1alpha1.ThanosReceiveIngestor{}
	if t.tenant.Status.ThanosResource != nil && t.tenant.Status.ThanosResource.ThanosReceiveIngestor != nil {
		err := t.Client.Get(t.Context, types.NamespacedName{
			Namespace: t.tenant.Status.ThanosResource.ThanosReceiveIngestor.Namespace,
			Name:      t.tenant.Status.ThanosResource.ThanosReceiveIngestor.Name,
		}, ingestor)
		if err != nil {
			if apierrors.IsNotFound(err) {
				klog.V(3).Infof("Cannot find ingestor [%s] for tenant [%s], create one", t.tenant.Status.ThanosResource.ThanosReceiveIngestor.Name, t.tenant.Name)
			} else {
				return err
			}
		} else {
			var needResetIngestor bool = false
			if ok := containsString(ingestor.Spec.Tenants, t.tenant.Spec.Tanant); !ok {
				klog.V(3).Infof("Tenant [%s] and ingestor [%s] mismatch, need to reset ingestor", t.tenant.Name, ingestor.Name)
				needResetIngestor = true
			}

			if v, ok := ingestor.Labels[monitoringv1alpha1.MonitoringPaodinService]; !ok || v != t.tenant.Labels[monitoringv1alpha1.MonitoringPaodinService] {
				klog.V(3).Infof("Tenant [%s] and ingestor [%s]'s Service mismatch, need to reset ingestor", t.tenant.Name, ingestor.Name)
				needResetIngestor = true
			}
			if _, ok := t.tenant.Labels[monitoringv1alpha1.MonitoringPaodinService]; ok && len(strings.Split(t.tenant.Labels[monitoringv1alpha1.MonitoringPaodinService], ".")) < 2 {
				return fmt.Errorf("Tenant [%s]'s Service field [%s] is invalid", t.tenant.Name, t.tenant.Labels[monitoringv1alpha1.MonitoringPaodinService])
			}

			if v, ok := ingestor.Labels[monitoringv1alpha1.MonitoringPaodinStorage]; !ok || v != t.tenant.Labels[monitoringv1alpha1.MonitoringPaodinStorage] {
				klog.V(3).Infof("Tenant [%s] and ingestor [%s]'s Storage mismatch, need to reset ingestor", t.tenant.Name, ingestor.Name)
				needResetIngestor = true

			}
			// todo: Storage deep check
			if t.tenant.Spec.Storage != nil {
				if v, ok := t.tenant.Spec.Storage.MatchLabels[monitoringv1alpha1.MonitoringPaodinStorage]; ok && v != ingestor.Labels[monitoringv1alpha1.MonitoringPaodinStorage] {
					klog.V(3).Infof("Tenant [%s]'s Storage update, need to reset ingestor", t.tenant.Name)
					needResetIngestor = true
					t.tenant.Labels[monitoringv1alpha1.MonitoringPaodinStorage] = t.tenant.Spec.Storage.MatchLabels[monitoringv1alpha1.MonitoringPaodinStorage]
					if err := t.Client.Update(t.Context, t.tenant); err != nil {
						return err
					}
				}
			} else {
				service := &monitoringv1alpha1.Service{}
				serviceNamespacedName := strings.Split(t.tenant.Labels[monitoringv1alpha1.MonitoringPaodinService], ".")
				if err := t.Client.Get(t.Context, types.NamespacedName{
					Namespace: serviceNamespacedName[0],
					Name:      serviceNamespacedName[1],
				}, service); err == nil {
					if service.Spec.Storage != nil && t.tenant.Labels[monitoringv1alpha1.MonitoringPaodinStorage] != service.Spec.Storage.MatchLabels[monitoringv1alpha1.MonitoringPaodinStorage] {
						klog.V(3).Infof("Tenant [%s]'s Storage update, need to reset ingestor", t.tenant.Name)
						needResetIngestor = true
						t.tenant.Labels[monitoringv1alpha1.MonitoringPaodinStorage] = service.Spec.Storage.MatchLabels[monitoringv1alpha1.MonitoringPaodinStorage]
						if err := t.Client.Update(t.Context, t.tenant); err != nil {
							return err
						}
					}
				}
			}

			if !needResetIngestor {
				return nil
			} else {
				klog.V(3).Infof("Reset ingestor [%s] for tenant [%s]", t.tenant.Name, ingestor.Name)
				err := t.removeTenantFromIngestorbyName(ingestor.Namespace, ingestor.Name)
				if err != nil {
					return err
				}

				return t.Client.Status().Update(t.Context, t.tenant)
			}
		}
	}

	// when tenant.Labels don't contain Service, remove the bindings to ingestor and ruler
	if v, ok := t.tenant.Labels[monitoringv1alpha1.MonitoringPaodinService]; !ok || v == "" {
		klog.V(3).Infof("Tenant [%s]'s Service is empty. thanosReceiveIngestor does not need to be created", t.tenant.Name)
		if t.tenant.Status.ThanosResource != nil && t.tenant.Status.ThanosResource.ThanosReceiveIngestor != nil {
			err := t.removeTenantFromIngestorbyName(t.tenant.Status.ThanosResource.ThanosReceiveIngestor.Namespace, t.tenant.Status.ThanosResource.ThanosReceiveIngestor.Name)
			if err != nil {
				return err
			}
			return t.Client.Status().Update(t.Context, t.tenant)
		}
		return nil
	} else if len(strings.Split(t.tenant.Labels[monitoringv1alpha1.MonitoringPaodinService], ".")) != 2 {
		return fmt.Errorf("Tenant [%s]'s Service field [%s] is invalid", t.tenant.Name, t.tenant.Labels[monitoringv1alpha1.MonitoringPaodinService])
	}

	// append Storage label
	if _, ok := t.tenant.Labels[monitoringv1alpha1.MonitoringPaodinStorage]; !ok {
		if t.tenant.Spec.Storage != nil {
			if _, ok := t.tenant.Spec.Storage.MatchLabels[monitoringv1alpha1.MonitoringPaodinStorage]; ok {
				t.tenant.Labels[monitoringv1alpha1.MonitoringPaodinStorage] = t.tenant.Spec.Storage.MatchLabels[monitoringv1alpha1.MonitoringPaodinStorage]
				if err := t.Client.Update(t.Context, t.tenant); err != nil {
					return err
				}
			}
		} else {
			service := &monitoringv1alpha1.Service{}
			serviceNamespacedName := strings.Split(t.tenant.Labels[monitoringv1alpha1.MonitoringPaodinService], ".")
			if err := t.Client.Get(t.Context, types.NamespacedName{
				Namespace: serviceNamespacedName[0],
				Name:      serviceNamespacedName[1],
			}, service); err == nil {
				if service.Spec.Storage != nil {
					t.tenant.Labels[monitoringv1alpha1.MonitoringPaodinStorage] = service.Spec.Storage.MatchLabels[monitoringv1alpha1.MonitoringPaodinStorage]
					if err := t.Client.Update(t.Context, t.tenant); err != nil {
						return err
					}
				}
			}
		}
	}

	// when tenant.Labels don't contain Storage, remove the bindings to ingestor
	if v, ok := t.tenant.Labels[monitoringv1alpha1.MonitoringPaodinStorage]; !ok || v == "" {
		klog.V(3).Infof("Tenant [%s]'s Storage is empty. thanosReceiveIngestor does not need to be created", t.tenant.Name)
		if t.tenant.Status.ThanosResource != nil && t.tenant.Status.ThanosResource.ThanosReceiveIngestor != nil {
			err := t.removeTenantFromIngestorbyName(t.tenant.Status.ThanosResource.ThanosReceiveIngestor.Namespace, t.tenant.Status.ThanosResource.ThanosReceiveIngestor.Name)
			if err != nil {
				return err
			}
			return t.Client.Status().Update(t.Context, t.tenant)
		}
		return nil
	}

	var ingestorList monitoringv1alpha1.ThanosReceiveIngestorList
	ls := make(map[string]string, 2)
	ls[monitoringv1alpha1.MonitoringPaodinService] = t.tenant.Labels[monitoringv1alpha1.MonitoringPaodinService]
	ls[monitoringv1alpha1.MonitoringPaodinStorage] = t.tenant.Labels[monitoringv1alpha1.MonitoringPaodinStorage]
	serviceNamespacedName := strings.Split(t.tenant.Labels[monitoringv1alpha1.MonitoringPaodinService], ".")
	err := t.Client.List(t.Context, &ingestorList, &client.ListOptions{
		Namespace:     serviceNamespacedName[0],
		LabelSelector: labels.SelectorFromSet(ls),
	})
	if err != nil {
		return err
	}

	ingestorMapping := make(map[string]*monitoringv1alpha1.ThanosReceiveIngestor, len(ingestorList.Items))
	for _, ingestorItem := range ingestorList.Items {
		ingestor := ingestorItem
		ingestorMapping[ingestorItem.Name] = &ingestor
		klog.V(3).Infof("Ingestor %s have Tenants: %v", ingestorItem.Name, ingestorItem.Labels[monitoringv1alpha1.MonitoringPaodinTenant])
	}

	// Check duplicates
	for _, ingestorItem := range ingestorMapping {
		if containsString(ingestorItem.Spec.Tenants, t.tenant.Spec.Tanant) {
			klog.V(3).Infof("Ingestor [%s] has tenant [%s] ,update status ", ingestorItem.Name, t.tenant.Name)
			if t.tenant.Status.ThanosResource == nil {
				t.tenant.Status.ThanosResource = &monitoringv1alpha1.ThanosResource{}
			}
			t.tenant.Status.ThanosResource.ThanosReceiveIngestor = &monitoringv1alpha1.ObjectReference{
				Namespace: ingestorItem.Namespace,
				Name:      ingestorItem.Name,
			}

			return t.Client.Status().Update(t.Context, t.tenant)
		}
	}

	// create or update ingestor instance.
	// traverse ingestorMapping according to the index, if it is currently empty, create a new instance,
	// otherwise check len(ingestorItem.Spec.Tenants) < t.DefaultTenantCountPerIngestor，if so, select the instance.
	for i := 0; i < len(ingestorMapping)+1; i++ {
		name := createIngestorInstanceName(t.tenant, strconv.Itoa(i))
		if ingestorItem, ok := ingestorMapping[name]; ok {
			if len(ingestorItem.Spec.Tenants) < t.DefaultTenantCountPerIngestor {
				ingestor = ingestorItem
				addTenantToIngestorInstance(t.tenant, ingestor)
				break
			}
		} else {
			ingestor = createIngestorInstance(name, t.tenant)
			break
		}
	}

	if t.tenant.Status.ThanosResource == nil {
		t.tenant.Status.ThanosResource = &monitoringv1alpha1.ThanosResource{}
	}
	t.tenant.Status.ThanosResource.ThanosReceiveIngestor = &monitoringv1alpha1.ObjectReference{
		Namespace: ingestor.Namespace,
		Name:      ingestor.Name,
	}

	if err := util.CreateOrUpdate(t.Context, t.Client, ingestor); err != nil {
		return err
	}
	return t.Client.Status().Update(t.Context, t.tenant)
}

func (t *Tenant) removeTenantFromIngestorbyName(namespace, name string) error {
	ingestor := &monitoringv1alpha1.ThanosReceiveIngestor{}

	err := t.Client.Get(t.Context, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, ingestor)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		} else {
			return err
		}
	} else {
		if ok := containsString(ingestor.Spec.Tenants, t.tenant.Spec.Tanant); ok {
			klog.V(3).Infof("ingestor %s update, remove tenant %s", ingestor.Name, t.tenant.Name)
			ingestor.Spec.Tenants = removeString(ingestor.Spec.Tenants, t.tenant.Spec.Tanant)
			ingestor.Labels[monitoringv1alpha1.MonitoringPaodinTenant] = strings.Join(ingestor.Spec.Tenants, "_")

			if len(ingestor.Spec.Tenants) == 0 {
				annotation := ingestor.GetAnnotations()
				if annotation == nil {
					annotation = make(map[string]string)
				}
				annotation[resources.LabelNameReceiveIngestorState] = "deleting"
				annotation[resources.LabelNameReceiveIngestorDeletingTime] = strconv.Itoa(int(time.Now().Unix()))
				ingestor.Annotations = annotation

				time.AfterFunc(t.DefaultIngestorRetentionPeriod, func() {
					t.deleteIngestorInstance(ingestor.Namespace, ingestor.Name)
				})
			}

			if t.tenant.Status.ThanosResource != nil && t.tenant.Status.ThanosResource.ThanosReceiveIngestor != nil {
				t.tenant.Status.ThanosResource.ThanosReceiveIngestor = nil
			}

			return util.CreateOrUpdate(t.Context, t.Client, ingestor)
		}
	}
	return nil
}

func (t *Tenant) deleteIngestorInstance(namespace, name string) error {
	ingestor := &monitoringv1alpha1.ThanosReceiveIngestor{}
	err := t.Client.Get(t.Context, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, ingestor)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		} else {
			return err
		}
	}

	annotations := ingestor.GetAnnotations()
	if annotations != nil {
		if v, ok := annotations[resources.LabelNameReceiveIngestorState]; ok && v == "deleting" {
			if v, ok := annotations[monitoringv1alpha1.MonitoringPaodinTenant]; !ok || len(v) == 0 {
				klog.V(3).Infof("ThanosReceiveIngestor %s will be deleted.")
				t.Client.Delete(t.Context, ingestor)
			}
		}
	}
	return nil
}

func createIngestorInstanceName(tenant *monitoringv1alpha1.Tenant, suffix ...string) string {
	serviceNamespacedName := strings.Split(tenant.Labels[monitoringv1alpha1.MonitoringPaodinService], ".")
	storageNamespacedName := strings.Split(tenant.Labels[monitoringv1alpha1.MonitoringPaodinStorage], ".")

	name := fmt.Sprintf("%s-%s-auto", serviceNamespacedName[1], storageNamespacedName[1])
	if len(suffix) > 0 {
		name += "-" + strings.Join(suffix, "-")
	}
	return name
}

func createIngestorInstance(name string, tenant *monitoringv1alpha1.Tenant) *monitoringv1alpha1.ThanosReceiveIngestor {
	klog.V(3).Infof("create new ingestor %s for tenant %s", name, tenant.Name)
	label := make(map[string]string, 2)
	label[monitoringv1alpha1.MonitoringPaodinService] = tenant.Labels[monitoringv1alpha1.MonitoringPaodinService]
	label[monitoringv1alpha1.MonitoringPaodinStorage] = tenant.Labels[monitoringv1alpha1.MonitoringPaodinStorage]
	label[monitoringv1alpha1.MonitoringPaodinTenant] = tenant.Name

	namespacedName := strings.Split(tenant.Labels[monitoringv1alpha1.MonitoringPaodinService], ".")
	// todo: ingestor config
	return &monitoringv1alpha1.ThanosReceiveIngestor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespacedName[0],
			Labels:    label,
		},
		Spec: monitoringv1alpha1.ThanosReceiveIngestorSpec{
			Tenants: []string{tenant.Spec.Tanant},
		},
	}
}

func addTenantToIngestorInstance(tenant *monitoringv1alpha1.Tenant, ingestor *monitoringv1alpha1.ThanosReceiveIngestor) {
	klog.V(3).Infof("Ingestor %s update, add tenant %s", ingestor.Name, tenant.Name)

	ingestor.Spec.Tenants = append(ingestor.Spec.Tenants, tenant.Spec.Tanant)

	label := ingestor.GetLabels()
	if v, ok := label[monitoringv1alpha1.MonitoringPaodinTenant]; !ok || len(v) == 0 {
		label[monitoringv1alpha1.MonitoringPaodinTenant] = tenant.Name
	} else {
		label[monitoringv1alpha1.MonitoringPaodinTenant] = label[monitoringv1alpha1.MonitoringPaodinTenant] + "." + tenant.Name
	}
	ingestor.Labels = label

	annotation := ingestor.GetAnnotations()
	if v, ok := annotation[resources.LabelNameReceiveIngestorState]; ok && v == "deleting" {
		annotation[resources.LabelNameReceiveIngestorState] = "running"
		annotation[resources.LabelNameReceiveIngestorDeletingTime] = ""
	}
	ingestor.Annotations = annotation
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
