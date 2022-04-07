package rsc

import (
	"context"
	"github.com/zbitech/common/pkg/id"
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

type ZcashInstanceResourceManager struct {
}

func NewZcashInstanceResourceManager() (interfaces.InstanceResourceManagerIF, error) {
	return &ZcashInstanceResourceManager{}, nil
}

func (z *ZcashInstanceResourceManager) GetInstanceResources(ctx context.Context, version string) (*config.VersionedResourceConfig, bool) {

	zcash_rsc, ok := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.ZCASH_INSTANCE)
	if !ok {
		return nil, false
	}

	resource, ok := zcash_rsc.Versions[version] //p.resourceMap[version]
	return &resource, ok
}

func (z *ZcashInstanceResourceManager) ValidateInstanceRequest(ctx context.Context, request ztypes.InstanceRequestIF) error {

	return nil
}

func (z *ZcashInstanceResourceManager) CreateInstance(ctx context.Context, project *entity.Project, request ztypes.InstanceRequestIF) (*entity.Instance, error) {

	instance := entity.Instance{}

	instance.Project = project.GetName()
	instance.Owner = project.GetOwner()
	instance.Network = project.GetNetwork()
	instance.Version = request.GetVersion()
	instance.Status = "NEW"
	instance.Timestamp = time.Now()
	instance.InstanceType = request.GetInstanceType()

	//	var paramSource *entity.VolumeSource = nil
	//	var dataSource *entity.VolumeSource = nil

	zcash_request := request.(*object.ZcashNodeInstanceRequest)

	instance.Version = zcash_request.Version
	instance.Name = zcash_request.Name
	instance.Description = zcash_request.Description

	detail := entity.ZcashInstanceDetail{}
	detail.Miner = zcash_request.Miner
	detail.TransactionIndex = zcash_request.TransactionIndex
	detail.Peers = zcash_request.Peers

	//detail.Conf = entity.NewZcashConf(zcash.Network, zcash_request.TransactionIndex, zcash_request.Miner)
	//	detail.ParamVolumeSource = paramSource
	//	detail.DataVolumeSource = dataSource

	instance.InstanceDetail = &detail

	return &instance, nil
}

func (z *ZcashInstanceResourceManager) CreateInstanceSpec(ctx context.Context, project *entity.Project, instance *entity.Instance) ([]string, error) {

	serviceAccountName := vars.AppConfig.Policy.ServiceAccount
	storageClass := "hostpath"

	inst_resource, ok := z.GetInstanceResources(ctx, instance.Version)
	if !ok {
		logger.Errorf(ctx, "Zcash resource not available for %s", instance.Version)
		return nil, errs.ErrInstanceResourceFailed
	}

	namespace := project.GetNamespace()

	node_image := inst_resource.GetImage("node")
	metrics_image := inst_resource.GetImage("metrics")
	if node_image == nil || metrics_image == nil {
		return nil, errs.ErrInstanceResourceFailed
	}

	z_spec := spec.ZcashNodeInstanceSpec{}

	detail := instance.InstanceDetail.(*entity.ZcashInstanceDetail)

	conf := object.NewZcashConf(instance.Network, detail.TransactionIndex, detail.Miner)
	conf.SetPort(node_image.Port)

	z_spec.InstanceSpec = spec.InstanceSpec{
		Project:            instance.Project,
		Name:               instance.Name,
		Version:            instance.Version,
		Network:            instance.Network,
		Owner:              instance.Owner,
		InstanceType:       instance.InstanceType,
		ServiceAccountName: serviceAccountName,
		Namespace:          namespace,
		StorageClass:       storageClass}

	z_spec.ZcashConf = conf.Value()
	z_spec.ZcashImage = node_image.URL
	z_spec.MetricsImage = metrics_image.URL
	z_spec.Port = node_image.Port
	z_spec.MetricsPort = metrics_image.Port
	z_spec.Username = id.GenerateUserName()
	z_spec.Password = id.GenerateSecurePassword()
	z_spec.Timeout = "2.0"

	z_spec.Username = utils.Base64EncodedString(z_spec.Username)
	z_spec.Password = utils.Base64EncodedString(z_spec.Password)

	fileTemplate, err := inst_resource.GetFileTemplate(vars.ASSET_PATH_DIRECTORY)
	if err != nil {
		logger.Errorf(ctx, "Zcash templates not found for version %s - %s", instance.Version, err)
		return nil, errs.ErrInstanceResourceFailed
	}

	specArr, err := fileTemplate.ExecuteTemplates(inst_resource.Templates.Keys, z_spec)
	if err != nil {
		logger.Errorf(ctx, "Zcash templates for version %s failed - %s", instance.Version, err)
		return nil, errs.ErrInstanceResourceFailed
	}

	return specArr, nil
}

func (z *ZcashInstanceResourceManager) UnmarshalBSONDetails(ctx context.Context, value bson.RawValue) (ztypes.InstanceDetailIF, error) {
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
