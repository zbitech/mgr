package rsc

import (
	"context"
	"github.com/zbitech/common/pkg/utils"
	"time"

	"github.com/zbitech/common/interfaces"
	"github.com/zbitech/common/pkg/errs"
	"github.com/zbitech/common/pkg/model/config"
	"github.com/zbitech/common/pkg/model/entity"
	"github.com/zbitech/common/pkg/model/object"
	"github.com/zbitech/common/pkg/model/spec"
	"github.com/zbitech/common/pkg/vars"

	"github.com/zbitech/common/pkg/logger"
	"github.com/zbitech/common/pkg/model/ztypes"
	"github.com/zbitech/common/pkg/rctx"
	"go.mongodb.org/mongo-driver/bson"
)

type ProjectResourceManager struct {
	instances map[ztypes.InstanceType]interfaces.InstanceResourceManagerIF
}

func NewProjectResourceManager() (interfaces.ProjectResourceManagerIF, error) {
	return &ProjectResourceManager{
		instances: make(map[ztypes.InstanceType]interfaces.InstanceResourceManagerIF),
	}, nil
}

func (p *ProjectResourceManager) AddInstanceManager(ctx context.Context, i_type ztypes.InstanceType, manager interfaces.InstanceResourceManagerIF) {
	p.instances[i_type] = manager
}

func (p *ProjectResourceManager) GetProjectResources(ctx context.Context, version string) (config.VersionedResourceConfig, bool) {
	resource, ok := vars.ResourceConfig.Project.Versions[version] //p.resourceMap[version]
	return resource, ok
}

func (p *ProjectResourceManager) ValidateProjectRequest(ctx context.Context, request *object.ProjectRequest) error {

	return nil
}

func (p *ProjectResourceManager) CreateProject(ctx context.Context, req *object.ProjectRequest) (*entity.Project, error) {

	currUser := rctx.GetCurrentUser(ctx)
	if !currUser.IsAuthenticated() {
		logger.Errorf(ctx, "User not authenticated")
		return nil, errs.ErrUserNotAuth
	}

	owner := currUser.GetUserId()
	project := entity.Project{}

	project.Network = req.Network
	project.Name = req.Name
	project.Version = req.Version
	project.Description = req.Description
	project.Owner = owner
	project.Status = "NEW"
	project.Timestamp = time.Now()

	return &project, nil
}

func (p *ProjectResourceManager) createProjectSpecObject(proj_resources config.VersionedResourceConfig, project *entity.Project) (*spec.ProjectSpec, error) {

	authz_image := proj_resources.GetImage("authz") //resources.Images.GetImage("authz")
	if authz_image == nil {
		return nil, errs.ErrProjectResourceFailed
	}

	p_spec := spec.ProjectSpec{}
	p_spec.Project = *project
	p_spec.Namespace = project.GetNamespace()
	p_spec.ServiceAccountName = vars.AppConfig.Policy.ServiceAccount
	p_spec.AuthzServerImage = authz_image.URL
	p_spec.Domain = vars.AppConfig.Policy.Domain
	p_spec.CertName = vars.AppConfig.Policy.CertName

	return &p_spec, nil
}

func (p *ProjectResourceManager) CreateProjectSpec(ctx context.Context, project *entity.Project) ([]string, error) {

	proj_resources, ok := p.GetProjectResources(ctx, project.Version) // p.resourceMap[project.Version]
	if !ok {
		logger.Errorf(ctx, "Project resource not available for %s", project.Version)
		return nil, errs.ErrProjectResourceFailed
	}

	p_spec, err := p.createProjectSpecObject(proj_resources, project)
	if err != nil {
		logger.Errorf(ctx, "Failed to create project spec - %s", err)
		return nil, errs.ErrProjectResourceFailed
	}

	logger.Debugf(ctx, "Created project spec - %s", utils.MarshalObject(p_spec))

	file_template, err := proj_resources.GetFileTemplate(vars.ASSET_PATH_DIRECTORY)
	if err != nil {
		logger.Errorf(ctx, "Project templates not found for version %s - %s", project.Version, err)
		return nil, errs.ErrProjectResourceFailed
	}

	logger.Debugf(ctx, "Retrieved resource templates: %s", file_template.Content)

	spec_arr, err := file_template.ExecuteTemplates([]string{"NAMESPACE", "AUTHZ_DEPLOYMENT", "AUTHZ_SERVICE"}, p_spec)
	if err != nil {
		logger.Errorf(ctx, "Project templates for version %s failed - %s", project.Version, err)
		return nil, errs.ErrProjectResourceFailed
	}

	logger.Debugf(ctx, "Generated spec details - %s", spec_arr)

	return spec_arr, nil
}

func (p *ProjectResourceManager) CreateProjectIngressSpec(ctx context.Context, project *entity.Project, instances []entity.Instance) ([]string, error) {

	proj_resources, ok := p.GetProjectResources(ctx, project.Version) // p.resourceMap[project.Version]
	if !ok {
		logger.Errorf(ctx, "Project resource not available for %s", project.Version)
		return nil, errs.ErrProjectResourceFailed
	}
	p_spec, err := p.createProjectSpecObject(proj_resources, project)
	p_spec.Instances = instances
	if err != nil {
		return nil, err
	}

	file_template, err := proj_resources.GetFileTemplate(vars.ASSET_PATH_DIRECTORY)
	if err != nil {
		logger.Errorf(ctx, "Project templates not found for version %s - %s", project.Version, err)
		return nil, errs.ErrProjectResourceFailed
	}

	spec_arr, err := file_template.ExecuteTemplates([]string{"HTTP-PROXY"}, p_spec)
	if err != nil {
		logger.Errorf(ctx, "Project templates for version %s failed - %s", project.Version, err)
		return nil, errs.ErrProjectResourceFailed
	}

	return spec_arr, nil
}

func (p *ProjectResourceManager) GetInstanceResources(ctx context.Context, iType ztypes.InstanceType, version string) (*config.VersionedResourceConfig, bool) {
	data_manager, ok := p.instances[iType]
	if !ok {
		return nil, false
	}

	return data_manager.GetInstanceResources(ctx, version)
}

func (p *ProjectResourceManager) ValidateInstanceRequest(ctx context.Context, request ztypes.InstanceRequestIF) error {
	rscManager, ok := p.instances[request.GetInstanceType()]
	if !ok {
		return errs.ErrInstanceResourceFailed
	}
	return rscManager.ValidateInstanceRequest(ctx, request)
}

func (p *ProjectResourceManager) CreateInstance(ctx context.Context, project *entity.Project, req ztypes.InstanceRequestIF) (*entity.Instance, error) {
	data_manager, ok := p.instances[req.GetInstanceType()]
	if !ok {
		return nil, errs.ErrInstanceDataFailed
	}

	return data_manager.CreateInstance(ctx, project, req)
}

func (p *ProjectResourceManager) CreateInstanceSpec(ctx context.Context, project *entity.Project, instance *entity.Instance) ([]string, error) {
	data_manager, ok := p.instances[instance.InstanceType]
	if !ok {
		return nil, errs.ErrInstanceDataFailed
	}

	return data_manager.CreateInstanceSpec(ctx, project, instance)
}

func (p *ProjectResourceManager) UnmarshalBSONInstance(ctx context.Context, data []byte) (*entity.Instance, error) {

	instance := entity.Instance{}

	var r bson.Raw
	if err := bson.Unmarshal(data, &r); err != nil {
		return nil, err
	}

	r.Lookup("project").Unmarshal(&instance.Project)
	r.Lookup("name").Unmarshal(&instance.Name)
	r.Lookup("version").Unmarshal(&instance.Version)
	r.Lookup("network").Unmarshal(&instance.Network)
	r.Lookup("owner").Unmarshal(&instance.Owner)
	r.Lookup("status").Unmarshal(&instance.Status)
	r.Lookup("timestamp").Unmarshal(&instance.Timestamp)
	r.Lookup("instancetype").Unmarshal(&instance.InstanceType)

	//	logger.Debugf(ctx, "Instance Type - %s", instance.InstanceType)

	details, err := p.UnmarshalBSONDetails(ctx, instance.InstanceType, r.Lookup("instancedetail"))
	if err != nil {
		logger.Errorf(ctx, "Unable to get instance entity - %s", err)
		return nil, errs.ErrInstanceDataFailed
	}

	instance.InstanceDetail = &details

	return &instance, nil

}

func (p *ProjectResourceManager) UnmarshalBSONKubernetesProjectResource(ctx context.Context, data []byte) (*entity.KubernetesProjectResource, error) {

	var r bson.Raw
	if err := bson.Unmarshal(data, &r); err != nil {
		return nil, err
	}

	var resource = entity.KubernetesProjectResource{}

	rsc, err := p.UnmarshalBSONKubernetesResource(ctx, data)
	if err != nil {
		return nil, err
	}

	resource.KubernetesResource = *rsc
	r.Lookup("project").Unmarshal(&resource.Project)

	return &resource, nil
}

func (p *ProjectResourceManager) UnmarshalBSONKubernetesInstanceResource(ctx context.Context, data []byte) (*entity.KubernetesInstanceResource, error) {

	var r bson.Raw
	if err := bson.Unmarshal(data, &r); err != nil {
		return nil, err
	}

	var resource = entity.KubernetesInstanceResource{}

	rsc, err := p.UnmarshalBSONKubernetesResource(ctx, data)
	if err != nil {
		return nil, err
	}

	resource.KubernetesResource = *rsc
	r.Lookup("project").Unmarshal(&resource.Project)
	r.Lookup("instance").Unmarshal(&resource.Instance)

	return &resource, nil
}

func (p *ProjectResourceManager) UnmarshalBSONKubernetesResource(ctx context.Context, r bson.Raw) (*entity.KubernetesResource, error) {

	// var r bson.Raw
	// if err := bson.Unmarshal(rsc, &r); err != nil {
	// 	return nil, err
	// }

	var resource = entity.KubernetesResource{}
	r.Lookup("_id").Unmarshal(&resource.Id)
	// r.Lookup("project").Unmarshal(&resource.Project)
	// r.Lookup("instance").Unmarshal(&resource.Instance)
	r.Lookup("name").Unmarshal(&resource.Name)
	r.Lookup("namespace").Unmarshal(&resource.Namespace)
	r.Lookup("type").Unmarshal(&resource.Type)
	r.Lookup("gvr").Unmarshal(&resource.GVR)
	r.Lookup("timestamp").Unmarshal(&resource.Timestamp)

	stateValue := r.Lookup("state")
	var state ztypes.ResourceStateIF
	var err error

	switch ztypes.ResourceObjectType(resource.Type) {
	case ztypes.DEPLOYMENT_RESOURCE:
		state = &entity.DeploymentState{}
		err = stateValue.Unmarshal(state)

	case ztypes.POD_RESOURCE:
		state = &entity.PodState{}
		err = stateValue.Unmarshal(state)

	case ztypes.PERSISTENT_VOLUME_RESOURCE:
		state = &entity.DeploymentState{}
		err = stateValue.Unmarshal(state)

	case ztypes.PERSISTENT_VOLUME_CLAIM_RESOURCE:
		state = &entity.DeploymentState{}
		err = stateValue.Unmarshal(state)

	case ztypes.VOLUME_SNAPHOT_RESOURCE:
		state = &entity.DeploymentState{}
		err = stateValue.Unmarshal(state)

	default:
		state = &entity.ResourceState{}
		err = stateValue.Unmarshal(state)
	}

	if err != nil {
		return nil, err
	}

	resource.State = state

	return &resource, nil
}

func (p *ProjectResourceManager) UnmarshalBSONDetails(ctx context.Context, iType ztypes.InstanceType, value bson.RawValue) (ztypes.InstanceDetailIF, error) {
	data_manager, ok := p.instances[iType]
	if !ok {
		return nil, errs.ErrInstanceDataFailed
	}

	return data_manager.UnmarshalBSONDetails(ctx, value)
}
