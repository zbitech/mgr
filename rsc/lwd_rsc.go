package rsc

import (
	"context"
	"github.com/zbitech/common/pkg/utils"
	"time"

	"github.com/zbitech/common/interfaces"
	"github.com/zbitech/common/pkg/errs"
	"github.com/zbitech/common/pkg/logger"
	"github.com/zbitech/common/pkg/model/config"
	"github.com/zbitech/common/pkg/model/entity"
	"github.com/zbitech/common/pkg/model/object"
	"github.com/zbitech/common/pkg/model/spec"
	"github.com/zbitech/common/pkg/model/ztypes"
	"github.com/zbitech/common/pkg/vars"
	"go.mongodb.org/mongo-driver/bson"
)

type LWDInstanceResourceManager struct {
}

func NewLWDInstanceResourceManager() (interfaces.InstanceResourceManagerIF, error) {
	return &LWDInstanceResourceManager{}, nil
}

func (lwd *LWDInstanceResourceManager) GetInstanceResources(ctx context.Context, version string) (*config.VersionedResourceConfig, bool) {

	zcash_rsc, ok := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.ZCASH_INSTANCE)
	if !ok {
		return nil, false
	}

	resource, ok := zcash_rsc.Versions[version] //p.resourceMap[version]
	return &resource, ok
}

func (lwd *LWDInstanceResourceManager) CreateInstance(ctx context.Context, project *entity.Project, request ztypes.InstanceRequestIF) (*entity.Instance, error) {

	instance := entity.Instance{}
	instance.Project = project.GetName()

	instance.Owner = project.GetOwner()
	instance.Network = project.GetNetwork()
	instance.Version = request.GetVersion()
	instance.Status = "NEW"
	instance.Timestamp = time.Now()
	instance.InstanceType = request.GetInstanceType()

	lwd_request := request.(*object.LWDInstanceRequest)

	instance.Name = lwd_request.Name
	instance.Version = lwd_request.Version

	detail := entity.LWDInstanceDetail{}
	detail.ZcashInstance = lwd_request.ZcashInstance

	instance.InstanceDetail = &detail

	return &instance, nil
}

func (lwd *LWDInstanceResourceManager) ValidateInstanceRequest(ctx context.Context, request ztypes.InstanceRequestIF) error {

	return nil
}

func (lwd *LWDInstanceResourceManager) CreateInstanceSpec(ctx context.Context, project *entity.Project, instance *entity.Instance) ([]string, error) {

	serviceAccountName := vars.AppConfig.Policy.ServiceAccount
	storageClass := "hostpath"

	inst_resource, ok := lwd.GetInstanceResources(ctx, instance.Version)
	if !ok {
		logger.Errorf(ctx, "Lightwallet resource not available for %s", instance.Version)
		return nil, errs.ErrInstanceResourceFailed
	}

	lwd_image := inst_resource.GetImage("lwd")
	if lwd_image == nil {
		return nil, errs.ErrInstanceResourceFailed
	}

	namespace := project.GetNamespace()

	detail := instance.InstanceDetail.(*entity.LWDInstanceDetail)

	lw_spec := spec.LWDInstanceSpec{}

	lw_spec.InstanceSpec = spec.InstanceSpec{
		Project:            instance.Project,
		Name:               instance.Name,
		Version:            instance.Version,
		Network:            instance.Network,
		Owner:              instance.Owner,
		InstanceType:       instance.InstanceType,
		ServiceAccountName: serviceAccountName,
		Namespace:          namespace,
		StorageClass:       storageClass}

	lw_spec.LightwalletImage = lwd_image.URL
	lw_spec.Port = int32(9067)
	lw_spec.HttPort = int32(9068)
	lw_spec.LogLevel = int32(7)
	lw_spec.ZcashInstance = detail.ZcashInstance
	lw_spec.Timeout = "2.0"

	logger.Infof(ctx, "Creating instance from %s - %s", utils.MarshalIndentObject(instance), utils.MarshalIndentObject(lw_spec))

	// TODO - Validate spec requirements - StorageClass, Namespace, etc
	// ValidateInstanceType()
	// ValidateName()
	// ValidateNamespace()
	// ValidateVersion()
	// ValidateStorageClass()
	// ValidateSubscriptionPlan()

	file_template, err := inst_resource.GetFileTemplate(vars.ASSET_PATH_DIRECTORY)
	if err != nil {
		logger.Errorf(ctx, "LWD templates not found for version %s - %s", project.Version, err)
		return nil, errs.ErrInstanceResourceFailed
	}

	spec_arr, err := file_template.ExecuteTemplates(inst_resource.Templates.Keys, lw_spec)
	if err != nil {
		logger.Errorf(ctx, "LWD templates for version %s failed - %s", project.Version, err)
		return nil, errs.ErrInstanceResourceFailed
	}

	return spec_arr, nil
}

func (lwd *LWDInstanceResourceManager) UnmarshalBSONDetails(ctx context.Context, value bson.RawValue) (ztypes.InstanceDetailIF, error) {
	logger.Debugf(ctx, "Unmarshaling Zcash instance details ............. %s", value.String())
	detail := entity.ZcashInstanceDetail{}
	//err := json.Unmarshal([]byte(value.String()), &detail)
	err := value.Unmarshal(&detail)
	if err != nil {
		logger.Errorf(ctx, "Unable to unmarshal zcash details - %s", err)
		return nil, err
	}

	logger.Debugf(ctx, "Unmarshaled zcash details - %s", utils.MarshalObject(detail))
	return &detail, nil
}
