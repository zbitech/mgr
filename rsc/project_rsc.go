package rsc

import (
	"context"
	"encoding/json"
	"github.com/zbitech/common/pkg/model/k8s"
	"github.com/zbitech/common/pkg/utils"
	"github.com/zbitech/mgr/internal/helper"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	"go.mongodb.org/mongo-driver/bson"
)

type ProjectResourceManager struct {
	projectConfig *config.ProjectResourceConfig
	instances     map[ztypes.InstanceType]interfaces.InstanceResourceManagerIF
}

func NewProjectResourceManager(projectConfig *config.ProjectResourceConfig,
	instanceManagers map[ztypes.InstanceType]interfaces.InstanceResourceManagerIF) (interfaces.ProjectResourceManagerIF, error) {
	var projectManager = &ProjectResourceManager{
		projectConfig: projectConfig,
		instances:     make(map[ztypes.InstanceType]interfaces.InstanceResourceManagerIF),
	}

	for iType, iManager := range instanceManagers {
		projectManager.instances[iType] = iManager
	}

	for _, cfg := range projectManager.projectConfig.Versions {
		if err := cfg.Init(vars.ASSET_PATH_DIRECTORY, object.NO_FUNCS); err != nil {
			return nil, err
		}
	}
	return projectManager, nil
}

func (p *ProjectResourceManager) GetProjectResources(version string) (*config.VersionedResourceConfig, bool) {
	resource, ok := p.projectConfig.Versions[version]
	return resource, ok
}

func (p *ProjectResourceManager) ValidateProjectRequest(ctx context.Context, request *object.ProjectRequest) error {

	return nil
}

func (p *ProjectResourceManager) CreateProject(ctx context.Context, req *object.ProjectRequest) (*entity.Project, error) {

	owner := req.GetOwner()
	project := entity.Project{}

	project.Network = req.Network
	project.Name = req.Name
	project.Version = req.Version
	project.Description = req.Description
	project.TeamId = req.Team
	project.Owner = owner
	project.Status = "New"
	project.Action = "created"
	project.Timestamp = time.Now()

	return &project, nil
}

func (p *ProjectResourceManager) UpdateProject(ctx context.Context, project *entity.Project, request *object.ProjectRequest) error {

	project.TeamId = request.Team
	project.Description = request.Description
	project.Action = "updated"
	project.ActionTime = time.Now()

	return nil
}

func (p *ProjectResourceManager) CreateProjectAssets(ctx context.Context, project *entity.Project) ([]*unstructured.Unstructured, error) {

	projResources, ok := p.GetProjectResources(project.Version)
	if !ok {
		logger.Errorf(ctx, "Project resource not available for %s", project.Version)
		return nil, errs.ErrProjectResourceFailed
	}

	pSpec := spec.ProjectSpec{}
	//	pSpec.Project = *project
	pSpec.Namespace = project.GetNamespace()
	//	pSpec.ServiceAccountName = vars.AppConfig.Policy.ServiceAccount
	//	pSpec.Domain = vars.AppConfig.Policy.Domain
	//	pSpec.CertName = vars.AppConfig.Policy.CertName

	pSpec.Labels = helper.CreateProjectLabels(project)
	//pSpec.Data = utils.MarshalObject(project)
	//	pSpec.DomainName = vars.AppConfig.Policy.Domain
	//	pSpec.DomainSecret = vars.AppConfig.Policy.CertName

	logger.Debugf(ctx, "Created project spec - %s", utils.MarshalObject(pSpec))

	fileTemplate := projResources.GetFileTemplate()

	var templates []string
	if vars.AppConfig.Features.AccessAuthorizationEnabled {
		//		templates = []string{"NAMESPACE", "SERVICE", "AUTHZ_SERVICE", "AUTHZ_EXTENSION"}
		templates = []string{"NAMESPACE", "SERVICE"}
	} else {
		templates = []string{"NAMESPACE", "SERVICE"}
	}

	specArr, err := fileTemplate.ExecuteTemplates(templates, pSpec)
	if err != nil {
		logger.Errorf(ctx, "Project templates for version %s failed - %s", project.Version, err)
		return nil, errs.ErrProjectResourceFailed
	}

	logger.Debugf(ctx, "Generated spec details - %s", specArr)

	return helper.CreateYAMLObjects(specArr)
}

func (p *ProjectResourceManager) CreateProjectIngressAsset(ctx context.Context, appIngress *unstructured.Unstructured, project *entity.Project, action ztypes.EventAction) ([]*unstructured.Unstructured, error) {
	projResources, ok := p.GetProjectResources(project.Version)
	if !ok {
		logger.Errorf(ctx, "Project resource not available for %s", project.Version)
		return nil, errs.ErrProjectResourceFailed
	}

	pSpec := spec.ProjectSpec{}
	pSpec.Namespace = project.GetNamespace()
	pSpec.Labels = helper.CreateProjectLabels(project)
	//pSpec.Data = utils.MarshalObject(project)
	//pSpec.DomainName = vars.AppConfig.Policy.Domain
	//pSpec.DomainSecret = vars.AppConfig.Policy.CertName
	//pSpec.Envoy = helper.CreateEnvoySpec()

	logger.Debugf(ctx, "Created project spec - %s", utils.MarshalObject(pSpec))

	fileTemplate := projResources.GetFileTemplate()

	specObj, err := fileTemplate.ExecuteTemplates([]string{"INGRESS", "INGRESS_INCLUDE"}, pSpec)
	if err != nil {
		logger.Errorf(ctx, "Project templates for version %s failed - %s", project.Version, err)
		return nil, errs.ErrProjectResourceFailed
	}

	logger.Debugf(ctx, "Generated spec details - %s", specObj)

	var ingressObj unstructured.Unstructured
	if err = helper.DecodeJSON(specObj[0], &ingressObj); err != nil {
		logger.Errorf(ctx, "Controller app template failed - %s", err)
		return nil, errs.ErrIngressResourceFailed
	}

	var includeObj k8s.IngressInclude
	if err = json.Unmarshal([]byte(specObj[1]), &includeObj); err != nil {
		logger.Errorf(ctx, "Controller app template failed - %s", err)
		return nil, errs.ErrIngressResourceFailed
	}

	//TODO - handle appIngress == nil - Return error?

	utils.RemoveResourceField(appIngress, "metadata.managedFields")
	utils.RemoveResourceField(appIngress, "spec.status")

	var includes []k8s.IngressInclude
	includeData := utils.GetResourceField(appIngress, "spec.includes")
	if includeData == nil {
		includes = make([]k8s.IngressInclude, 0)
	} else if err = json.Unmarshal([]byte(includeData.(string)), &includes); err != nil {
		logger.Errorf(ctx, "Error unmarshaling ingress routes - %s", err)
	}

	var updated = false
	for index, include := range includes {
		if include.Namespace == includeObj.Namespace {
			if action == ztypes.EventActionDelete {
				includes = append(includes[:index], includes[index+1:]...)
			} else {
				includes = append(includes[:index], includeObj)
				includes = append(includes, includes[index+1:]...)
			}
			updated = true
		}
	}

	if !updated && action != "remove" {
		includes = append(includes, includeObj)
	}
	logger.Debugf(ctx, "Ingress includes - %s", utils.MarshalObject(includes))
	utils.SetResourceField(appIngress, "spec.includes", includes)

	return []*unstructured.Unstructured{appIngress, &ingressObj}, nil
}

func (p *ProjectResourceManager) GetInstanceResources(iType ztypes.InstanceType, version string) (*config.VersionedResourceConfig, bool) {
	dataManager, ok := p.instances[iType]
	if !ok {
		return nil, false
	}

	return dataManager.GetInstanceResources(version)
}

func (p *ProjectResourceManager) CreateInstanceRequest(ctx context.Context, iType ztypes.InstanceType, iRequest interface{}) (object.InstanceRequestIF, error) {
	rscManager, ok := p.instances[iType]
	if !ok {
		return nil, errs.ErrInstanceResourceFailed
	}

	return rscManager.CreateInstanceRequest(ctx, iRequest)
}

func (p *ProjectResourceManager) CreateInstance(ctx context.Context, project *entity.Project, req object.InstanceRequestIF) (entity.InstanceIF, error) {
	dataManager, ok := p.instances[req.GetInstanceType()]
	if !ok {
		return nil, errs.ErrInstanceDataFailed
	}

	return dataManager.CreateInstance(ctx, project, req)
}

func (p *ProjectResourceManager) UpdateInstance(ctx context.Context, project *entity.Project, instance entity.InstanceIF, request object.InstanceRequestIF) error {
	resourceManager, ok := p.instances[instance.GetInstanceType()]
	if !ok {
		return errs.ErrInstanceDataFailed
	}

	return resourceManager.UpdateInstance(ctx, project, instance, request)
}

func (p *ProjectResourceManager) CreateDeploymentResourceAssets(ctx context.Context, instance entity.InstanceIF) ([]*unstructured.Unstructured, error) {
	dataManager, ok := p.instances[instance.GetInstanceType()]
	if !ok {
		return nil, errs.ErrInstanceDataFailed
	}

	return dataManager.CreateDeploymentResourceAssets(ctx, instance)
}

func (p *ProjectResourceManager) CreateIngressAsset(ctx context.Context, projIngress *unstructured.Unstructured, instance entity.InstanceIF, action ztypes.EventAction) (*unstructured.Unstructured, error) {
	resourceManager, ok := p.instances[instance.GetInstanceType()]
	if !ok {
		return nil, errs.ErrInstanceDataFailed
	}

	return resourceManager.CreateIngressAsset(ctx, projIngress, instance, action)
}

func (p *ProjectResourceManager) CreateStartResourceAssets(ctx context.Context, instance entity.InstanceIF) ([]*unstructured.Unstructured, error) {
	resourceManager, ok := p.instances[instance.GetInstanceType()]
	if !ok {
		return nil, errs.ErrInstanceDataFailed
	}

	return resourceManager.CreateStartResourceAssets(ctx, instance)
}

func (p *ProjectResourceManager) CreateSnapshotAssets(ctx context.Context, instance entity.InstanceIF, volume string) ([]*unstructured.Unstructured, error) {
	resourceManager, ok := p.instances[instance.GetInstanceType()]
	if !ok {
		return nil, errs.ErrInstanceDataFailed
	}

	return resourceManager.CreateSnapshotAssets(ctx, instance, volume)
}

func (p *ProjectResourceManager) CreateSnapshotScheduleAssets(ctx context.Context, instance entity.InstanceIF, volume string, schedule ztypes.ZBIBackupScheduleType) ([]*unstructured.Unstructured, error) {
	resourceManager, ok := p.instances[instance.GetInstanceType()]
	if !ok {
		return nil, errs.ErrInstanceDataFailed
	}

	return resourceManager.CreateSnapshotScheduleAssets(ctx, instance, volume, schedule)
}

func (p *ProjectResourceManager) CreateRotationAssets(ctx context.Context, instance entity.InstanceIF) ([]*unstructured.Unstructured, error) {
	resourceManager, ok := p.instances[instance.GetInstanceType()]
	if !ok {
		return nil, errs.ErrInstanceDataFailed
	}

	return resourceManager.CreateRotationAssets(ctx, instance)
}

func (p *ProjectResourceManager) UnmarshalBSONInstance(ctx context.Context, data bson.Raw) (entity.InstanceIF, error) {
	var iType ztypes.InstanceType
	data.Lookup("instancetype").Unmarshal(&iType)

	return p.UnmarshalBSONDetails(ctx, iType, data)
}

func (p *ProjectResourceManager) UnmarshalBSONDetails(ctx context.Context, iType ztypes.InstanceType, value bson.Raw) (entity.InstanceIF, error) {
	resourceManager, ok := p.instances[iType]
	if !ok {
		return nil, errs.ErrInstanceDataFailed
	}

	return resourceManager.UnmarshalBSONDetails(ctx, value)
}
