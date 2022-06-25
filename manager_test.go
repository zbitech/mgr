package mgr

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/zbitech/common/factory"
	"testing"
)

func Test_NewResourceManagerFactory(t *testing.T) {
	var f = NewResourceManagerFactory()
	assert.NotNilf(t, f, "Failed to create resource manager factory")
}

func Test_Init(t *testing.T) {
	ctx := context.Background()
	factory.InitProjectResourceConfig(ctx)
	var f = NewResourceManagerFactory()
	err := f.Init(ctx)
	assert.NoErrorf(t, err, "Error while initializing resource manager factory - %s", err)
}

func Test_GetIngressResourceManager(t *testing.T) {
	ctx := context.Background()
	factory.InitProjectResourceConfig(ctx)
	var f = NewResourceManagerFactory()
	f.Init(ctx)
	assert.NotNilf(t, f.GetAppResourceManager(ctx), "Failed to create ingress resource manager")
}

func Test_GetProjectDataManager(t *testing.T) {
	ctx := context.Background()
	factory.InitProjectResourceConfig(ctx)
	var f = NewResourceManagerFactory()
	f.Init(ctx)
	assert.NotNilf(t, f.GetProjectDataManager(ctx), "Failed to create project resource manager")
}
