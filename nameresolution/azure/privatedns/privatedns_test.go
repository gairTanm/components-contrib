package privatedns

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dapr/components-contrib/nameresolution"
	"github.com/dapr/kit/logger"
)

const (
	aciDaprInternalPortString = "50006"
)

func TestNewResolver(t *testing.T) {
	logger := logger.NewLogger("test")
	resolver := NewResolver(logger)

	assert.NotNil(t, resolver)
}

func TestResolveID(t *testing.T) {
	resolver := NewResolver(logger.NewLogger("test"))
	request := nameresolution.ResolveRequest{ID: "dapr-service-a", Namespace: "", Port: aciDaprInternalPort}
	const expect = "dapr-service-a.private.dapr.com:" + aciDaprInternalPortString
	target, err := resolver.ResolveID(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, expect, target)
}

func TestInit(t *testing.T) {
	resolver := NewResolver(logger.NewLogger("test"))
	metadata := nameresolution.Metadata{
		Configuration: map[string]interface{}{
			"zoneName":       "private.tgair.com",
			"clientId":       "a1058e76-cb9f-4503-88ad-adce6f0b13c1",
			"subscriptionId": "da28f5e5-aa45-46fe-90c8-053ca49ab4b5",
			"daprPort":       "3500",
			"appId":          "service-a",
			"resourceGroup":  "tgair-rg",
		},
	}

	err := resolver.Init(context.Background(), metadata)

	assert.NoError(t, err)
}

func TestInitWithManagedIdentity(t *testing.T) {
	resolver := NewResolver(logger.NewLogger("test"))
	err := resolver.Init(context.Background(), nameresolution.Metadata{
		Configuration: map[string]interface{}{
			"zoneName":       "private.contoso.com",
			"clientId":       "8dc8d2b2-aad7-47a3-8f70-3f62d7685d19",
			"subscriptionId": "da28f5e5-aa45-46fe-90c8-053ca49ab4b5",
			"daprPort":       "3500",
			"appId":          "service-a",
			"resourceGroup":  "tgair-rg",
		},
	})

	assert.NoError(t, err)
}

func TestResolveIDWithInit(t *testing.T) {
	resolver := NewResolver(logger.NewLogger("test"))
	request := nameresolution.ResolveRequest{ID: "dapr-service-a", Namespace: "", Port: aciDaprInternalPort}

	_ = resolver.Init(context.Background(), nameresolution.Metadata{
		Configuration: map[string]interface{}{
			"zoneName":       "private.contoso.com",
			"clientId":       "a1058e76-cb9f-4503-88ad-adce6f0b13c1",
			"subscriptionId": "da28f5e5-aa45-46fe-90c8-053ca49ab4b5",
			"daprPort":       "3500",
			"appId":          "service-a",
			"resourceGroup":  "tgair-rg",
		},
	})

	const expect = "dapr-service-a.private.contoso.com:" + aciDaprInternalPortString
	target, err := resolver.ResolveID(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, expect, target)
}
