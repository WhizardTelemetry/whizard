package query

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/WhizardTelemetry/whizard/pkg/api/monitoring/v1alpha1"
	"github.com/WhizardTelemetry/whizard/pkg/controllers/components"
)

type Query struct {
	client   client.Client
	instance *v1alpha1.Query
	service  *v1alpha1.Service
	storage  *v1alpha1.Storage
}

var _ components.Operator = (*Query)(nil)

// New constructs a compactor operator from the given CR, service, and storage.
func New(client client.Client, instance client.Object, service *v1alpha1.Service, storage *v1alpha1.Storage) (components.Operator, error) {
	compactor, ok := instance.(*v1alpha1.Query)
	if !ok {
		return nil, fmt.Errorf("invalid instance type %T, expect *v1alpha1.Query", instance)
	}

	return &Query{
		client:   client,
		instance: compactor,
		service:  service,
		storage:  storage,
	}, nil
}

func (c *Query) Reconcile(ctx context.Context) error {
	return nil
}
