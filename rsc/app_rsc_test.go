package rsc

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/zbitech/common/interfaces"
	"github.com/zbitech/common/pkg/model/entity"
	"github.com/zbitech/common/pkg/model/object"
	"github.com/zbitech/common/pkg/model/spec"
	"github.com/zbitech/common/pkg/model/ztypes"
	"github.com/zbitech/common/pkg/utils"
	"github.com/zbitech/common/pkg/vars"
	"github.com/zbitech/fake/data"
	"github.com/zbitech/fake/test"
	"github.com/zbitech/mgr/internal/helper"
	"testing"
)

func createProjectManager(iTypes ...ztypes.InstanceType) (interfaces.ProjectResourceManagerIF, error) {

	var instanceMap = map[ztypes.InstanceType]interfaces.InstanceResourceManagerIF{}

	for _, iType := range iTypes {
		switch iType {
		case ztypes.InstanceTypeZCASH:
			zcashCfg, ok := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.InstanceTypeZCASH)
			if !ok {
				return nil, errors.New("unable to create zcash resource manager")
			}
			zcashManager, err := NewZcashInstanceResourceManager(zcashCfg)
			if err != nil {
				return nil, err
			}
			instanceMap[iType] = zcashManager

		case ztypes.InstanceTypeLWD:
			lwdCfg, ok := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.InstanceTypeLWD)
			if !ok {
				return nil, errors.New("unable to create lwd resource manager")
			}
			lwdManager, err := NewLWDInstanceResourceManager(lwdCfg)
			if err != nil {
				return nil, err
			}
			instanceMap[iType] = lwdManager
		}
	}

	projectManager, err := NewProjectResourceManager(vars.ResourceConfig.Project, instanceMap)
	if err != nil {
		return nil, err
	}
	return projectManager, nil
}

func Test_NewIngressResourceManager(t *testing.T) {

	vars.DATABASE_FACTORY = "memory"

	ctx := context.Background()
	test.InitTest(ctx)
	//	factory.InitProjectResourceConfig(ctx)

	projectManager, err := createProjectManager()
	assert.NoErrorf(t, err, "Unable to create project resource manager")

	ingressManager, err := NewAppResourceManager(vars.ResourceConfig.App, projectManager)
	assert.NoErrorf(t, err, "Failed to create ingress manager")
	assert.NotNilf(t, ingressManager, "Failed to created ingress manager")
}

func Test_GetIngressResources(t *testing.T) {

	vars.DATABASE_FACTORY = "memory"
	ctx := context.Background()
	test.InitTest(ctx)

	projectManager, err := createProjectManager()
	assert.NoError(t, err)

	appManager, err := NewAppResourceManager(vars.ResourceConfig.App, projectManager)
	assert.NoError(t, err)
	assert.NotNil(t, appManager)

	config, ok := appManager.GetAppResources("v1")
	assert.True(t, ok)
	assert.NotNil(t, config)
}

func Test_CreateProjectIngressAsset(t *testing.T) {

	vars.DATABASE_FACTORY = "memory"
	ctx := context.Background()
	test.InitTest(ctx)

	projectManager, err := createProjectManager(ztypes.InstanceTypeZCASH)
	assert.NoError(t, err)

	appManager, err := NewAppResourceManager(vars.ResourceConfig.App, projectManager)
	assert.NoError(t, err)
	assert.NotNil(t, appManager)

	ingress, err := appManager.CreateProjectIngressAsset(ctx, nil, &data.Project1, data.Instance1, "add")
	assert.NoError(t, err)
	assert.NotNil(t, ingress)
	t.Logf("%s", utils.MarshalIndentObject(ingress))

	ingress, err = appManager.CreateProjectIngressAsset(ctx, ingress, &data.Project1, data.Instance2, "add")
	assert.NoError(t, err)
	assert.NotNil(t, ingress)
	t.Logf("%s", utils.MarshalIndentObject(ingress))

	ingress, err = appManager.CreateProjectIngressAsset(ctx, ingress, &data.Project1, data.Instance1, "remove")
	assert.NoError(t, err)
	assert.NotNil(t, ingress)
	t.Logf("%s", utils.MarshalIndentObject(ingress))

}

func Test_CreateControllerIngressAsset(t *testing.T) {

	vars.DATABASE_FACTORY = "memory"
	ctx := context.Background()
	test.InitTest(ctx)

	vars.AppConfig.Policy.Domain = "api.zbitech.local"
	vars.AppConfig.Policy.CertName = "kube-system/zbi-tls"

	projectManager, err := createProjectManager()
	assert.NoErrorf(t, err, "Unable to create project resource manager")

	appManager, err := NewAppResourceManager(vars.ResourceConfig.App, projectManager)
	assert.NoError(t, err)
	assert.NotNil(t, appManager)

	var project = &entity.Project{Name: "project1", Version: "v1", Network: ztypes.NetworkTypeTest, Owner: "admin", Status: "NEW"}

	ingress, err := appManager.CreateControllerIngressAsset(ctx, nil, project, "")
	assert.NoError(t, err)
	assert.NotNil(t, ingress)
	t.Logf("%s", utils.MarshalIndentObject(ingress))
}

func Test_CreateSnapshotAsset(t *testing.T) {

	//ctx := context.Background()
	//factory.InitProjectResourceConfig(ctx)

	vars.DATABASE_FACTORY = "memory"

	ctx := context.Background()
	test.InitTest(ctx)

	appManager, err := NewAppResourceManager(vars.ResourceConfig.App, nil)
	assert.NoErrorf(t, err, "Failed to create ingress manager")
	assert.NotNilf(t, appManager, "Failed to create ingress manager")

	instance := data.Instance1
	var req = &object.SnapshotRequest{Version: "v1", Volume: instance.DataVolume.Volume, VolumeName: instance.DataVolume.Name,
		Namespace: instance.GetNamespace(), Labels: helper.CreateInstanceLabels(instance)}
	specArr, err := appManager.CreateSnapshotAsset(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, specArr)
	assert.Len(t, specArr, 1)
}

func Test_CreateSnapshotScheduleAsset(t *testing.T) {
	vars.DATABASE_FACTORY = "memory"

	ctx := context.Background()
	test.InitTest(ctx)

	appManager, err := NewAppResourceManager(vars.ResourceConfig.App, nil)
	assert.NoError(t, err)
	assert.NotNil(t, appManager)

	instance := data.Instance1
	var req = &object.SnapshotScheduleRequest{Schedule: ztypes.DailySnapshotSchedule, Version: "v1", Volume: instance.DataVolume.Volume,
		VolumeName: instance.DataVolume.Name, Namespace: instance.GetNamespace(), Labels: helper.CreateInstanceLabels(instance)}
	objects, err := appManager.CreateSnapshotScheduleAsset(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, objects)
	assert.Len(t, objects, 1)

	t.Logf("Asset: %s", utils.MarshalIndentObject(objects))
}

func Test_CreateVolumeAsset(t *testing.T) {
	vars.DATABASE_FACTORY = "memory"

	ctx := context.Background()
	test.InitTest(ctx)

	appManager, err := NewAppResourceManager(vars.ResourceConfig.App, nil)
	assert.NoError(t, err)
	assert.NotNil(t, appManager)

	zcash := data.Instance1
	labels := helper.CreateInstanceLabels(zcash)

	storageClass := vars.AppConfig.Policy.StorageClass
	volumeDataSource := zcash.DataSourceType == ztypes.VolumeDataSource
	snapshotDataSource := zcash.DataSourceType == ztypes.SnapshotDataSource

	var volumeSpecs = []spec.VolumeSpec{
		{Volume: zcash.DataVolume.Volume, VolumeName: zcash.DataVolume.Name, StorageClass: storageClass, Namespace: zcash.GetNamespace(),
			SourceName: zcash.DataSource, VolumeDataSource: volumeDataSource, SnapshotDataSource: snapshotDataSource, Size: 10,
			Labels: labels},
		{Volume: zcash.ParamsVolume.Volume, VolumeName: zcash.ParamsVolume.Name, StorageClass: storageClass, Namespace: zcash.GetNamespace(),
			SourceName: zcash.DataSource, VolumeDataSource: volumeDataSource, SnapshotDataSource: snapshotDataSource, Size: 3,
			Labels: labels},
	}

	objects, err := appManager.CreateVolumeAsset(ctx, volumeSpecs...)
	assert.NoError(t, err)
	assert.NotNil(t, objects)
	assert.Len(t, objects, 2)

	t.Logf("Asset: %s", utils.MarshalIndentObject(objects))
}
