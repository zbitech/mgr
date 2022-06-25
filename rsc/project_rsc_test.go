package rsc

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/zbitech/common/factory"
	"github.com/zbitech/common/interfaces"
	"github.com/zbitech/common/pkg/model/object"
	"github.com/zbitech/common/pkg/model/ztypes"
	"github.com/zbitech/common/pkg/vars"
	"testing"
)

func Test_NewProjectResourceManager(t *testing.T) {
	ctx := context.Background()
	factory.InitProjectResourceConfig(ctx)

	var projManager interfaces.ProjectResourceManagerIF
	var err error

	projManager, err = NewProjectResourceManager(vars.ResourceConfig.Project, nil)
	assert.NoErrorf(t, err, "Failed to create project resource manager")
	assert.NotNilf(t, projManager, "Failed to create project resource manager")

	projManager, err = NewProjectResourceManager(vars.ResourceConfig.Project, map[ztypes.InstanceType]interfaces.InstanceResourceManagerIF{})
	assert.NoErrorf(t, err, "Failed to create project resource manager")
	assert.NotNilf(t, projManager, "Failed to create project resource manager")
}

func Test_GetProjectResources(t *testing.T) {
	ctx := context.Background()
	factory.InitProjectResourceConfig(ctx)

	var projManager interfaces.ProjectResourceManagerIF
	var err error

	projManager, err = NewProjectResourceManager(vars.ResourceConfig.Project, nil)
	assert.NoErrorf(t, err, "Failed to create project resource manager")

	_, ok := projManager.GetProjectResources("v1")
	assert.Truef(t, ok, "Failed to get project resources")
}

func Test_CreateProject(t *testing.T) {
	ctx := context.Background()
	factory.InitProjectResourceConfig(ctx)

	var projManager interfaces.ProjectResourceManagerIF
	var err error

	projManager, err = NewProjectResourceManager(vars.ResourceConfig.Project, nil)
	assert.NoErrorf(t, err, "Failed to create project resource manager")

	var request = object.ProjectRequest{Name: "sample", Version: "v1", Network: ztypes.NetworkTypeTest, Description: "Sample Project", Team: "team1"}
	project, err := projManager.CreateProject(ctx, &request)
	assert.NoErrorf(t, err, "Failed to create project - %s", err)
	assert.NotNilf(t, project, "Failed to create project")
}

func Test_CreateProjectAsserts(t *testing.T) {
	ctx := context.Background()
	factory.InitProjectResourceConfig(ctx)

	var projManager interfaces.ProjectResourceManagerIF
	var err error

	projManager, err = NewProjectResourceManager(vars.ResourceConfig.Project, nil)
	assert.NoErrorf(t, err, "Failed to create project resource manager")

	var request = object.ProjectRequest{Name: "sample", Version: "v1", Network: ztypes.NetworkTypeTest, Description: "Sample Project", Team: "team1"}
	request.SetOwner("admin")
	project, _ := projManager.CreateProject(ctx, &request)

	objects, err := projManager.CreateProjectAssets(ctx, project)
	assert.NoErrorf(t, err, "Failed to generate project assets")
	assert.Lenf(t, objects, 1, "Failed to generate 1 resource")
}

func Test_CreateProjectSpec(t *testing.T) {

}

func Test_GetInstanceResources(t *testing.T) {

}

func Test_CreateInstance(t *testing.T) {

}

func Test_CreateInstanceSpec(t *testing.T) {

}

func Test_UnmarshalBSONInstance(t *testing.T) {

}

func Test_UnmarshalBSONDetails(t *testing.T) {

}
