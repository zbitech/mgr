package rsc

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/zbitech/common/pkg/id"
	"github.com/zbitech/common/pkg/model/k8s"
	"github.com/zbitech/common/pkg/rctx"
	"github.com/zbitech/common/pkg/utils"
	"github.com/zbitech/mgr/internal/helper"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"text/template"
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

var (
	FUNCTIONS = template.FuncMap{
		"base64Encode": func(value string) string {
			return utils.Base64EncodeString(value)
		},
		"basicCredentials": func(username, password string) string {
			creds := fmt.Sprintf("%s:%s", username, password)
			return utils.Base64EncodeString(creds)
		},
	}
)

type ZcashInstanceResourceManager struct {
	rscConfig *config.InstanceResourceConfig
}

func NewZcashInstanceResourceManager(instanceConfig *config.InstanceResourceConfig) (interfaces.InstanceResourceManagerIF, error) {
	var mgr = &ZcashInstanceResourceManager{rscConfig: instanceConfig}

	for _, cfg := range mgr.rscConfig.Versions {
		if err := cfg.Init(vars.ASSET_PATH_DIRECTORY, FUNCTIONS); err != nil {
			logger.Errorf(rctx.CTX, "Failed to create manager: %s", err)
			return nil, err
		}
	}

	return mgr, nil
}

func (z *ZcashInstanceResourceManager) GetInstanceResources(version string) (*config.VersionedResourceConfig, bool) {
	resource, ok := z.rscConfig.Versions[version]
	return resource, ok
}

func (z *ZcashInstanceResourceManager) CreateInstanceRequest(ctx context.Context, iRequest interface{}) (object.InstanceRequestIF, error) {

	jsonStr, err := json.Marshal(iRequest)
	if err != nil {
		return nil, errs.ErrMarshalFailed
	}

	var zcashReq object.ZcashNodeInstanceRequest
	if err := json.Unmarshal(jsonStr, &zcashReq); err != nil {
		logger.Errorf(ctx, "Failed to unmarshal request - %s", err)
		return nil, errs.ErrMarshalFailed
	}

	return zcashReq, nil
}

func (z *ZcashInstanceResourceManager) CreateInstance(ctx context.Context, project *entity.Project, request object.InstanceRequestIF) (entity.InstanceIF, error) {

	instResource, ok := z.GetInstanceResources(request.GetVersion())
	if !ok {
		logger.Errorf(ctx, "Zcash resource not available for %s", request.GetVersion())
		return nil, errs.ErrInstanceResourceFailed
	}

	zcashRequest := request.(object.ZcashNodeInstanceRequest)
	dataVolume := instResource.Volumes[0]
	paramsVolume := instResource.Volumes[1]

	return &entity.ZcashInstance{
		Instance: entity.Instance{
			Project:        project.GetName(),
			Name:           zcashRequest.GetName(),
			Version:        zcashRequest.GetVersion(),
			Network:        project.GetNetwork(),
			Description:    zcashRequest.Description,
			Owner:          project.GetOwner(),
			Status:         "New",
			Timestamp:      time.Now(),
			DataSourceType: zcashRequest.GetDataSourceType(),
			DataSource:     zcashRequest.GetDataSource(),
			InstanceType:   zcashRequest.GetInstanceType(),
			Action:         "created",
			ActionTime:     time.Now(),
			Age:            "",
		},
		ZcashDetails: entity.ZcashDetails{
			TransactionIndex: zcashRequest.TransactionIndex,
			Miner:            zcashRequest.Miner,
			Peers:            zcashRequest.Peers,
			DataVolume:       entity.DataVolume{Name: dataVolume + "-" + zcashRequest.Name, Size: 10, Volume: dataVolume},
			ParamsVolume:     entity.DataVolume{Name: paramsVolume + "-" + zcashRequest.Name, Size: 3, Volume: paramsVolume},
		},
	}, nil

	//	return &zcash, nil
}

func (z *ZcashInstanceResourceManager) UpdateInstance(ctx context.Context, project *entity.Project, instance entity.InstanceIF, request object.InstanceRequestIF) error {

	zcashRequest := request.(object.ZcashNodeInstanceRequest)
	zcash := instance.(*entity.ZcashInstance)

	zcash.Action = "updated"
	zcash.ActionTime = time.Now()
	zcash.Description = zcashRequest.Description
	//detail := instance.InstanceDetail.(entity.ZcashInstanceDetail)
	zcash.Miner = zcashRequest.Miner
	zcash.TransactionIndex = zcashRequest.TransactionIndex
	zcash.Peers = zcashRequest.Peers

	return nil
}

func (z *ZcashInstanceResourceManager) CreateDeploymentResourceAssets(ctx context.Context, instance entity.InstanceIF) ([]*unstructured.Unstructured, error) {

	zcash := instance.(*entity.ZcashInstance)
	instResource, ok := z.GetInstanceResources(zcash.Version)
	if !ok {
		logger.Errorf(ctx, "Zcash resource not available for %s", zcash.Version)
		return nil, errs.ErrInstanceResourceFailed
	}

	nodeImage := instResource.GetImage("node")
	metricsImage := instResource.GetImage("metrics")
	if nodeImage == nil || metricsImage == nil {
		return nil, errs.ErrInstanceResourceFailed
	}

	conf := object.NewZcashConf(zcash.Network, zcash.TransactionIndex, zcash.Miner)
	conf.SetPort(nodeImage.Port)

	zcashSpec := spec.ZcashNodeInstanceSpec{
		InstanceSpec: spec.InstanceSpec{
			Name:               zcash.Name,
			Project:            zcash.Project,
			Version:            zcash.Version,
			ServiceAccountName: vars.AppConfig.Policy.ServiceAccount,
			Namespace:          zcash.GetNamespace(),
			Labels:             helper.CreateInstanceLabels(zcash),
			DomainName:         vars.AppConfig.Policy.Domain,
			DomainSecret:       vars.AppConfig.Policy.CertName,
			DataSourceType:     zcash.DataSourceType,
			DataSource:         zcash.DataSource},
		Username:     id.GenerateUserName(),
		Password:     id.GenerateSecurePassword(),
		ZcashConf:    conf.Value(),
		ZcashImage:   nodeImage.URL,
		MetricsImage: metricsImage.URL,
		Port:         z.rscConfig.Ports["service"],
		MetricsPort:  z.rscConfig.Ports["metrics"],
		DataVolume:   zcash.DataVolume.Name,
		ParamsVolume: zcash.ParamsVolume.Name,
		Envoy:        helper.CreateEnvoySpec(z.rscConfig.Ports["envoy"]),
	}

	//	logger.Infof(ctx, "Creating deployment assets for: %s", utils.MarshalObject(zcashSpec))

	var specArr []string
	var err error

	fileTemplate := instResource.GetFileTemplate()
	specArr, err = fileTemplate.ExecuteTemplates([]string{"ZCASH_CONF", "ENVOY_CONF", "CREDENTIALS", "DEPLOYMENT", "SERVICE"}, zcashSpec)
	if err != nil {
		logger.Errorf(ctx, "Zcash templates for version %s failed - %s", zcash.Version, err)
		return nil, errs.ErrInstanceResourceFailed
	}

	objects, err := helper.CreateYAMLObjects(specArr)
	if err != nil {
		logger.Errorf(ctx, "Zcash templates for version %s failed - %s", zcash.Version, err)
		return nil, errs.ErrInstanceResourceFailed
	}

	volumeDataSource := zcash.DataSourceType == ztypes.VolumeDataSource
	snapshotDataSource := zcash.DataSourceType == ztypes.SnapshotDataSource
	storageClass := vars.AppConfig.Policy.StorageClass

	var volumeSpecs = []spec.VolumeSpec{
		{Volume: zcash.DataVolume.Volume, VolumeName: zcash.DataVolume.Name, StorageClass: storageClass,
			Namespace: zcash.GetNamespace(), SourceName: zcash.DataSource, VolumeDataSource: volumeDataSource,
			SnapshotDataSource: snapshotDataSource, Size: zcash.DataVolume.Size, Labels: zcashSpec.Labels},
		{Volume: zcash.ParamsVolume.Volume, VolumeName: zcash.ParamsVolume.Name, StorageClass: storageClass,
			Namespace: zcash.GetNamespace(), SourceName: zcash.DataSource, VolumeDataSource: volumeDataSource,
			SnapshotDataSource: snapshotDataSource, Size: zcash.ParamsVolume.Size, Labels: zcashSpec.Labels},
	}

	appRsc := vars.ManagerFactory.GetAppResourceManager(ctx)
	volumes, err := appRsc.CreateVolumeAsset(ctx, volumeSpecs...)
	if err != nil {
		logger.Errorf(ctx, "Zcash volume templates for version %s failed - %s", zcash.Version, err)
		return nil, errs.ErrInstanceResourceFailed
	}

	objects = append(objects, volumes...)
	return objects, nil
}

func (z *ZcashInstanceResourceManager) CreateStartResourceAssets(ctx context.Context, instance entity.InstanceIF) ([]*unstructured.Unstructured, error) {
	zcash := instance.(*entity.ZcashInstance)
	instResource, ok := z.GetInstanceResources(zcash.Version)
	if !ok {
		logger.Errorf(ctx, "Zcash resource not available for %s", zcash.Version)
		return nil, errs.ErrInstanceResourceFailed
	}

	nodeImage := instResource.GetImage("node")
	metricsImage := instResource.GetImage("metrics")
	if nodeImage == nil || metricsImage == nil {
		return nil, errs.ErrInstanceResourceFailed
	}

	zcashSpec := spec.ZcashNodeInstanceSpec{
		InstanceSpec: spec.InstanceSpec{
			Name:               zcash.Name,
			ServiceAccountName: vars.AppConfig.Policy.ServiceAccount,
			Namespace:          zcash.GetNamespace(),
			Labels:             helper.CreateInstanceLabels(zcash),
			DomainName:         vars.AppConfig.Policy.Domain,
			DomainSecret:       vars.AppConfig.Policy.CertName,
			DataSourceType:     zcash.DataSourceType,
			DataSource:         zcash.DataSource},
		ZcashImage:   nodeImage.URL,
		MetricsImage: metricsImage.URL,
		Port:         z.rscConfig.Ports["service"],
		MetricsPort:  z.rscConfig.Ports["metrics"],
		DataVolume:   zcash.DataVolume.Name,
		ParamsVolume: zcash.ParamsVolume.Name,
		Envoy:        helper.CreateEnvoySpec(z.rscConfig.Ports["envoy"]),
	}
	var specArr []string
	var err error

	fileTemplate := instResource.GetFileTemplate()
	specArr, err = fileTemplate.ExecuteTemplates([]string{"DEPLOYMENT", "SERVICE"}, zcashSpec)
	if err != nil {
		logger.Errorf(ctx, "Zcash templates for version %s failed - %s", zcash.Version, err)
		return nil, errs.ErrInstanceResourceFailed
	}

	objects, err := helper.CreateYAMLObjects(specArr)
	if err != nil {
		logger.Errorf(ctx, "Zcash templates for version %s failed - %s", zcash.Version, err)
		return nil, errs.ErrInstanceResourceFailed
	}

	return objects, nil
}

func (z *ZcashInstanceResourceManager) CreateIngressAsset(ctx context.Context, projIngress *unstructured.Unstructured, instance entity.InstanceIF, action ztypes.EventAction) (*unstructured.Unstructured, error) {

	zcash := instance.(*entity.ZcashInstance)
	instResource, ok := z.GetInstanceResources(zcash.Version)
	if !ok {
		logger.Errorf(ctx, "Zcash resource not available for %s", zcash.Version)
		return nil, errs.ErrInstanceResourceFailed
	}

	zcashSpec := spec.ZcashNodeInstanceSpec{
		InstanceSpec: spec.InstanceSpec{
			Name:               zcash.Name,
			ServiceAccountName: vars.AppConfig.Policy.ServiceAccount,
			Namespace:          zcash.GetNamespace(),
			Labels:             helper.CreateInstanceLabels(zcash),
			DomainName:         vars.AppConfig.Policy.Domain,
			DomainSecret:       vars.AppConfig.Policy.CertName},
		Envoy: helper.CreateEnvoySpec(z.rscConfig.Ports["envoy"]),
	}
	var specObj string
	var err error

	fileTemplate := instResource.GetFileTemplate()
	if action == ztypes.EventActionStopInstance {
		specObj, err = fileTemplate.ExecuteTemplate("INGRESS_STOPPED", zcashSpec)
	} else {
		specObj, err = fileTemplate.ExecuteTemplate("INGRESS", zcashSpec)
	}
	if err != nil {
		logger.Errorf(ctx, "Zcash templates for version %s failed - %s", zcash.Version, err)
		return nil, errs.ErrInstanceResourceFailed
	}

	var route k8s.IngressRoute
	if err = json.Unmarshal([]byte(specObj), &route); err != nil {
		logger.Errorf(ctx, "Zcash route marshal failed - %s", err)
		return nil, errs.ErrIngressResourceFailed
	}
	logger.Infof(ctx, "Route: %s", utils.MarshalObject(route))

	//TODO - handle projIngress == nil - return error?
	utils.RemoveResourceField(projIngress, "metadata.managedFields")
	utils.RemoveResourceField(projIngress, "spec.status")

	routeData := utils.MarshalObject(utils.GetResourceField(projIngress, "spec.routes"))
	var routes []k8s.IngressRoute
	if err = json.Unmarshal([]byte(routeData), &routes); err != nil {
		logger.Errorf(ctx, "Error unmarshaling ingress routes - %s", err)
	}
	var updated = false
	for index, r := range routes {
		for _, condition := range r.Conditions {
			logger.Infof(ctx, "Comparing %s and %s at index %d ...", condition.Prefix, route.Conditions[0].Prefix, index)
			if condition.Prefix == route.Conditions[0].Prefix {
				if action == ztypes.EventActionDelete {
					routes = append(routes[:index], routes[index+1:]...)
				} else {
					routes = append(routes[:index], route)
					routes = append(routes, routes[index+1:]...)
				}
				updated = true
			}
		}
	}

	if !updated {
		routes = append(routes, route)
	}
	logger.Debugf(ctx, "Ingress routes: %s", utils.MarshalObject(routes))
	utils.SetResourceField(projIngress, "spec.routes", routes)

	return projIngress, nil
}

func (z *ZcashInstanceResourceManager) CreateSnapshotAssets(ctx context.Context, instance entity.InstanceIF, volume string) ([]*unstructured.Unstructured, error) {

	var req object.SnapshotRequest

	zcash := instance.(*entity.ZcashInstance)
	appRsc := vars.ManagerFactory.GetAppResourceManager(ctx)
	req.Namespace = zcash.GetNamespace()
	if volume == "zcash-data" {
		req.VolumeName = zcash.DataVolume.Name
	} else if volume == "zcash-params" {
		req.VolumeName = zcash.ParamsVolume.Name
	}
	req.Labels = helper.CreateInstanceLabels(zcash)

	return appRsc.CreateSnapshotAsset(ctx, &req)
}

func (z *ZcashInstanceResourceManager) CreateSnapshotScheduleAssets(ctx context.Context, instance entity.InstanceIF, volume string, scheduleType ztypes.ZBIBackupScheduleType) ([]*unstructured.Unstructured, error) {

	var req object.SnapshotScheduleRequest

	zcash := instance.(*entity.ZcashInstance)

	appRsc := vars.ManagerFactory.GetAppResourceManager(ctx)
	req.Namespace = zcash.GetNamespace()
	req.Schedule = scheduleType
	if volume == "zcash-data" {
		req.VolumeName = zcash.DataVolume.Name
	} else if volume == "zcash-params" {
		req.VolumeName = zcash.ParamsVolume.Name
	}
	req.Labels = helper.CreateInstanceLabels(zcash)

	return appRsc.CreateSnapshotScheduleAsset(ctx, &req)
}

func (z *ZcashInstanceResourceManager) CreateRotationAssets(ctx context.Context, instance entity.InstanceIF) ([]*unstructured.Unstructured, error) {
	zcash := instance.(*entity.ZcashInstance)
	instResource, ok := z.GetInstanceResources(zcash.Version)
	if !ok {
		logger.Errorf(ctx, "Zcash resource not available for %s", zcash.Version)
		return nil, errs.ErrInstanceResourceFailed
	}

	zcashSpec := spec.ZcashNodeInstanceSpec{
		InstanceSpec: spec.InstanceSpec{
			Name:               zcash.Name,
			ServiceAccountName: vars.AppConfig.Policy.ServiceAccount,
			Namespace:          zcash.GetNamespace(),
			Labels:             helper.CreateInstanceLabels(zcash),
			DataSourceType:     zcash.DataSourceType,
			DataSource:         zcash.DataSource},
		Username: id.GenerateUserName(),
		Password: id.GenerateSecurePassword(),
		Envoy:    helper.CreateEnvoySpec(z.rscConfig.Ports["envoy"]),
	}

	var specArr []string
	var err error

	fileTemplate := instResource.GetFileTemplate()
	specArr, err = fileTemplate.ExecuteTemplates([]string{"ENVOY_CONF", "CREDENTIALS"}, zcashSpec)
	if err != nil {
		logger.Errorf(ctx, "Zcash templates for version %s failed - %s", zcash.Version, err)
		return nil, errs.ErrInstanceResourceFailed
	}

	return helper.CreateYAMLObjects(specArr)
}

func (z *ZcashInstanceResourceManager) UnmarshalBSONDetails(ctx context.Context, value bson.Raw) (entity.InstanceIF, error) {

	logger.Tracef(ctx, "Unmarshaling Zcash instance details ............. %s", value.String())

	var zcash entity.ZcashInstance
	if err := bson.Unmarshal(value, &zcash); err != nil {
		return nil, err
	}

	logger.Debugf(ctx, "Unmarshaled zcash details - %s", utils.MarshalObject(zcash))
	return &zcash, nil

	//	detail := entity.ZcashInstance{}
	//	if err := value.Unmarshal(&detail); err != nil {
	//		logger.Errorf(ctx, "Unable to unmarshal zcash details - %s", err)
	//		return nil, err
	//	}
	//
	//	logger.Debugf(ctx, "Unmarshaled zcash details - %s", utils.MarshalObject(detail))
	//	return detail, nil
}
