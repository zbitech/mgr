package rsc

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/zbitech/common/pkg/model/object"
	"github.com/zbitech/common/pkg/model/spec"
	"github.com/zbitech/common/pkg/model/ztypes"
	"github.com/zbitech/common/pkg/utils"
	"github.com/zbitech/common/pkg/vars"
	"github.com/zbitech/fake/data"
	"github.com/zbitech/fake/mgr/rsc"
	"github.com/zbitech/fake/test"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func Test_NewLWDInstanceResourceManager(t *testing.T) {
	vars.DATABASE_FACTORY = "memory"
	ctx := context.Background()
	test.InitTest(ctx)

	lwdConfig, ok := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.InstanceTypeLWD)
	assert.True(t, ok)
	assert.NotNil(t, lwdConfig)

	lwdResource, err := NewLWDInstanceResourceManager(lwdConfig)
	assert.NoError(t, err)
	assert.NotNil(t, lwdResource)
}

func Test_GetLWDInstanceResources(t *testing.T) {
	vars.DATABASE_FACTORY = "memory"
	ctx := context.Background()
	test.InitTest(ctx)

	lwdConfig, _ := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.InstanceTypeLWD)
	lwdResource, _ := NewLWDInstanceResourceManager(lwdConfig)
	lwdManager := lwdResource.(*LWDInstanceResourceManager)
	for version, _ := range lwdManager.lwdConfig.Versions {
		cfg, ok := lwdResource.GetInstanceResources(version)
		assert.True(t, ok)
		assert.NotNil(t, cfg)
	}
}

func Test_CreateLWDInstanceRequest(t *testing.T) {
	vars.DATABASE_FACTORY = "memory"
	ctx := context.Background()
	test.InitTest(ctx)

	lwdConfig, _ := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.InstanceTypeLWD)
	lwdResource, _ := NewLWDInstanceResourceManager(lwdConfig)
	//	lwdManager := lwdResource.(*LWDInstanceResourceManager)

	var input struct {
		Name           string
		Version        string
		Description    string
		Methods        bool
		DataSourceType ztypes.DataSourceType
		DataSource     string
		ZcashInstance  string
		ZcashPort      int32
	}

	input.Name = "lwd-instance"
	input.Version = "v1"
	input.Description = "LWD Instance"
	input.Methods = true
	input.DataSourceType = ztypes.NoDataSource
	input.DataSource = ""
	input.ZcashInstance = "zcash-main-1.project.svc.cluster.local"
	input.ZcashPort = vars.AppConfig.Envoy.Port

	lwdReq, err := lwdResource.CreateInstanceRequest(ctx, input)
	assert.NoError(t, err)
	assert.NotNil(t, lwdReq)
	assert.Equal(t, lwdReq.GetInstanceType(), ztypes.InstanceTypeLWD)
}

func Test_CreateLWDInstance(t *testing.T) {
	vars.DATABASE_FACTORY = "memory"
	ctx := context.Background()
	test.InitTest(ctx)

	lwdConfig, _ := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.InstanceTypeLWD)
	lwdResource, _ := NewLWDInstanceResourceManager(lwdConfig)

	var lwdReq = object.LWDInstanceRequest{
		InstanceRequest: object.InstanceRequest{
			Name:           "lwd-instance",
			Version:        "v1",
			Description:    "Lightwalletd instance",
			Methods:        true,
			DataSourceType: ztypes.NoDataSource,
			DataSource:     "",
		},
		ZcashInstance: "zcash-main-1.project.svc.cluster.local",
	}

	lwdInstance, err := lwdResource.CreateInstance(ctx, &data.Project1, lwdReq)
	assert.NoError(t, err)
	assert.NotNil(t, lwdInstance)
	t.Logf("LWD Instance: %s", utils.MarshalIndentObject(lwdInstance))
}

func Test_UpdateLWDInstance(t *testing.T) {

}

func Test_CreateLWDDeploymentResourceAssets(t *testing.T) {
	vars.DATABASE_FACTORY = "memory"
	ctx := context.Background()
	test.InitTest(ctx)

	lwdConfig, _ := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.InstanceTypeLWD)
	lwdResource, _ := NewLWDInstanceResourceManager(lwdConfig)

	appManager := vars.ManagerFactory.GetAppResourceManager(ctx).(*rsc.FakeAppResourceManager)
	appManager.FakeCreateVolumeAsset = func(ctx context.Context, volumes ...spec.VolumeSpec) ([]*unstructured.Unstructured, error) {
		data, err := data.GetInstanceResources(ztypes.InstanceTypeLWD, ztypes.ResourcePersistentVolumeClaim, 1)
		return data, err
	}

	//	t.Logf("LWD Instance: %s", utils.MarshalIndentObject(data.LwdInstance1))
	objects, err := lwdResource.CreateDeploymentResourceAssets(ctx, data.LwdInstance1)
	assert.NoError(t, err)
	assert.NotNil(t, objects)
	assert.Greater(t, len(objects), 0)

	//	t.Logf("LWD Objects: %s", utils.MarshalIndentObject(objects))
}
