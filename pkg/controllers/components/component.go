package components

import (
	"context"

	"github.com/WhizardTelemetry/whizard/pkg/api/monitoring/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Operator interface {
	Reconcile(context.Context) error
}

// Constructor defines a uniform factory signature for component operators.
type Constructor func(client.Client, client.Object, *v1alpha1.Service, *v1alpha1.Storage) (Operator, error)
