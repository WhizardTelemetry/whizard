package app

import (
	"context"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/kubesphere/paodin/cmd/controller-manager/app/options"
	"github.com/kubesphere/paodin/pkg/client/k8s"
	"github.com/kubesphere/paodin/pkg/controllers/monitoring"
	"github.com/kubesphere/paodin/pkg/informers"
)

func addControllers(mgr manager.Manager, client k8s.Client, informerFactory informers.InformerFactory,
	cmOptions *options.PaodinControllerManagerOptions, ctx context.Context) error {

	if err := (&monitoring.ServiceReconciler{
		DefaulterValidator: monitoring.CreateServiceDefaulterValidator(*cmOptions.MonitoringOptions),
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
		Context:            ctx,
	}).SetupWithManager(mgr); err != nil {
		klog.Errorf("Unable to create Service controller: %v", err)
		return err
	}

	if err := (&monitoring.StoreReconciler{
		DefaulterValidator: monitoring.CreateStoreDefaulterValidator(*cmOptions.MonitoringOptions),
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
		Context:            ctx,
		Options:            cmOptions.MonitoringOptions.Store,
	}).SetupWithManager(mgr); err != nil {
		klog.Errorf("Unable to create Store controller: %v", err)
		return err
	}

	if err := (&monitoring.CompactorReconciler{
		DefaulterValidator: monitoring.CreateCompactorDefaulterValidator(*cmOptions.MonitoringOptions),
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
		Context:            ctx,
		Options:            cmOptions.MonitoringOptions.Compactor,
	}).SetupWithManager(mgr); err != nil {
		klog.Errorf("Unable to create Compactor controller: %v", err)
		return err
	}

	if err := (&monitoring.IngesterReconciler{
		DefaulterValidator: monitoring.CreateIngesterDefaulterValidator(*cmOptions.MonitoringOptions),
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
		Context:            ctx,
	}).SetupWithManager(mgr); err != nil {
		klog.Errorf("Unable to create Ingester controller: %v", err)
		return err
	}

	if err := (&monitoring.RulerReconciler{
		DefaulterValidator:    monitoring.CreateRulerDefaulterValidator(*cmOptions.MonitoringOptions),
		ReloaderConfig:        cmOptions.MonitoringOptions.PrometheusConfigReloader,
		RulerQueryProxyConfig: cmOptions.MonitoringOptions.RulerQueryProxy,
		Client:                mgr.GetClient(),
		Scheme:                mgr.GetScheme(),
		Context:               ctx,
	}).SetupWithManager(mgr); err != nil {
		klog.Errorf("Unable to create Ruler controller: %v", err)
		return err
	}

	if err := (&monitoring.TenantReconciler{
		DefaulterValidator: monitoring.CreateTenantDefaulterValidator(*cmOptions.MonitoringOptions),
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
		Context:            ctx,
		Options:            cmOptions.MonitoringOptions,
	}).SetupWithManager(mgr); err != nil {
		klog.Errorf("Unable to create Tenant controller: %v", err)
		return err
	}

	if err := (&monitoring.StorageReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Context: ctx,
	}).SetupWithManager(mgr); err != nil {
		klog.Errorf("Unable to create Storage controller: %v", err)
		return err
	}

	if cmOptions.MonitoringOptions.EnableKubeSphereAdapter {
		if err := (&monitoring.ClusterReconciler{
			Client:                          mgr.GetClient(),
			Scheme:                          mgr.GetScheme(),
			Context:                         ctx,
			KubesphereAdapterDefaultService: cmOptions.MonitoringOptions.KubeSphereAdapterService,
		}).SetupWithManager(mgr); err != nil {
			klog.Errorf("Unable to create Cluster controller: %v", err)
			return err
		}
	}
	return nil
}
