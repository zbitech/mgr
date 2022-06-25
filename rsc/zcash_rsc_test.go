package rsc

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/zbitech/common/factory"
	"github.com/zbitech/common/pkg/model/k8s"
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

func Test_NewZcashInstanceResourceManager(t *testing.T) {
	ctx := context.Background()
	factory.InitProjectResourceConfig(ctx)

	zcashConfig, ok := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.InstanceTypeZCASH)
	assert.Truef(t, ok, "zcash resource not configured")
	assert.NotNilf(t, zcashConfig, "Failed to get zcash resource")

	zcashResource, err := NewZcashInstanceResourceManager(zcashConfig)
	assert.NoErrorf(t, err, "Error creating zcash resource manager - %s", err)
	assert.NotNilf(t, zcashResource, "Failed to create zcash resource manager")
}

func Test_GetZcashInstanceResources(t *testing.T) {
	ctx := context.Background()
	factory.InitProjectResourceConfig(ctx)

	zcashConfig, ok := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.InstanceTypeZCASH)
	assert.Truef(t, ok, "zcash resource not configured")
	assert.NotNilf(t, zcashConfig, "Failed to get zcash resource")

	zcashResource, _ := NewZcashInstanceResourceManager(zcashConfig)
	v1Config, ok := zcashResource.GetInstanceResources("v1")
	assert.Truef(t, ok, "zcash resource for v1 not configured")
	assert.NotNilf(t, v1Config, "Failed to get zcash resource for v1")
}

func Test_CreateZcashInstance(t *testing.T) {
	ctx := context.Background()
	factory.InitProjectResourceConfig(ctx)

	zcashConfig, _ := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.InstanceTypeZCASH)
	zcashResource, _ := NewZcashInstanceResourceManager(zcashConfig)

	var project = data.Project1
	var request = object.ZcashNodeInstanceRequest{
		InstanceRequest: object.InstanceRequest{
			Name:           data.Instance1.Name,
			Version:        data.Instance1.Version,
			Description:    data.Instance1.Description,
			Methods:        true,
			DataSourceType: data.Instance1.DataSourceType,
			DataSource:     data.Instance1.DataSource,
		},
		TransactionIndex: false,
		Miner:            false,
		Peers:            []string{},
	}

	instance, err := zcashResource.CreateInstance(ctx, &project, request)
	assert.NoError(t, err)
	assert.NotNil(t, instance)
}

func Test_CreateZcashDeploymentResourceAssets(t *testing.T) {

	vars.DATABASE_FACTORY = "memory"
	ctx := context.Background()
	test.InitTest(ctx)

	appManager := vars.ManagerFactory.GetAppResourceManager(ctx).(*rsc.FakeAppResourceManager)

	zcashConfig, _ := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.InstanceTypeZCASH)
	zcashResource, _ := NewZcashInstanceResourceManager(zcashConfig)

	appManager.FakeCreateVolumeAsset = func(ctx context.Context, volumes ...spec.VolumeSpec) ([]*unstructured.Unstructured, error) {
		data, err := data.GetInstanceResources(ztypes.InstanceTypeZCASH, ztypes.ResourcePersistentVolumeClaim, 2)
		return data, err
	}

	objects, err := zcashResource.CreateDeploymentResourceAssets(ctx, data.Instance1)
	assert.NoError(t, err)
	assert.NotNil(t, objects)
	//	t.Logf("Objects: %s", utils.MarshalIndentObject(objects))
}

func Test_CreateIngressAsset(t *testing.T) {
	vars.DATABASE_FACTORY = "memory"
	ctx := context.Background()
	test.InitTest(ctx)

	projIngress, err := data.GetGenericResource(ztypes.ResourceHTTPProxy)
	utils.SetResourceField(projIngress, "spec.includes", []k8s.IngressInclude{})
	assert.NoError(t, err)
	assert.NotNil(t, projIngress)

	t.Logf("Ingress: %s", utils.MarshalObject(projIngress))

	zcashConfig, _ := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.InstanceTypeZCASH)
	zcashResource, _ := NewZcashInstanceResourceManager(zcashConfig)

	// 1. create instance, 2 routes
	projIngress, err = zcashResource.CreateIngressAsset(ctx, projIngress, data.Instance1, ztypes.EventActionCreate)
	t.Logf("Created Ingress for instance: %s", utils.MarshalObject(projIngress))

	// 2. stop instance, 2 routes
	projIngress, err = zcashResource.CreateIngressAsset(ctx, projIngress, data.Instance1, ztypes.EventActionStopInstance)
	t.Logf("Stopped Ingress for instance: %s", utils.MarshalObject(projIngress))

	// 3. start instance, 2 routes
	projIngress, err = zcashResource.CreateIngressAsset(ctx, projIngress, data.Instance1, ztypes.EventActionStartInstance)
	t.Logf("Started Ingress for instance: %s", utils.MarshalObject(projIngress))

	// 4. delete instance 1 route
	projIngress, err = zcashResource.CreateIngressAsset(ctx, projIngress, data.Instance1, ztypes.EventActionDelete)
	t.Logf("Deleted Ingress for instance: %s", utils.MarshalObject(projIngress))

}

func Test_CreateZcashStartResourceAssets(t *testing.T) {
	vars.DATABASE_FACTORY = "memory"
	ctx := context.Background()
	test.InitTest(ctx)

	zcashConfig, _ := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.InstanceTypeZCASH)
	zcashResource, _ := NewZcashInstanceResourceManager(zcashConfig)

	objects, err := zcashResource.CreateStartResourceAssets(ctx, data.Instance1)
	assert.NoError(t, err)
	assert.NotNil(t, objects)

	t.Logf("Objects: %s", utils.MarshalIndentObject(objects))
}

func Test_CreateZcashSnapshotAssets(t *testing.T) {
	vars.DATABASE_FACTORY = "memory"
	ctx := context.Background()
	test.InitTest(ctx)

	appManager := vars.ManagerFactory.GetAppResourceManager(ctx).(*rsc.FakeAppResourceManager)

	zcashConfig, _ := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.InstanceTypeZCASH)
	zcashResource, _ := NewZcashInstanceResourceManager(zcashConfig)

	appManager.FakeCreateSnapshotAsset = func(ctx context.Context, req *object.SnapshotRequest) ([]*unstructured.Unstructured, error) {
		return nil, nil
	}

	zcashResource.CreateSnapshotAssets(ctx, data.Instance1, data.Instance1.DataVolume.Volume)
}

func Test_UnmarshalBSONZcashDetails(t *testing.T) {

}
