package rsc

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/zbitech/common/pkg/utils"
	"github.com/zbitech/mgr/internal/helper"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	lwdConfig *config.InstanceResourceConfig
}

func NewLWDInstanceResourceManager(lwdConfig *config.InstanceResourceConfig) (interfaces.InstanceResourceManagerIF, error) {
	var mgr = &LWDInstanceResourceManager{lwdConfig: lwdConfig}

	for _, cfg := range mgr.lwdConfig.Versions {
		if err := cfg.Init(vars.ASSET_PATH_DIRECTORY, object.NO_FUNCS); err != nil {
			return nil, err
		}
	}

	return mgr, nil
}

func (lwd *LWDInstanceResourceManager) GetInstanceResources(version string) (*config.VersionedResourceConfig, bool) {
	resource, ok := lwd.lwdConfig.Versions[version]
	return resource, ok
}

func (lwd *LWDInstanceResourceManager) CreateInstanceRequest(ctx context.Context, iRequest interface{}) (object.InstanceRequestIF, error) {
	jsonStr, err := json.Marshal(iRequest)
	if err != nil {
		return nil, errs.ErrMarshalFailed
	}

	var lwdReq object.LWDInstanceRequest
	if err := json.Unmarshal(jsonStr, &lwdReq); err != nil {
		logger.Errorf(ctx, "Failed to unmarshal request - %s", err)
		return nil, errs.ErrMarshalFailed
	}

	return lwdReq, nil
}

func (lwd *LWDInstanceResourceManager) CreateInstance(ctx context.Context, project *entity.Project, request object.InstanceRequestIF) (entity.InstanceIF, error) {

	instResource, ok := lwd.GetInstanceResources(request.GetVersion())
	if !ok {
		logger.Errorf(ctx, "Lightwalletd resource not available for %s", request.GetVersion())
		return nil, errs.ErrInstanceResourceFailed
	}

	lwdRequest := request.(object.LWDInstanceRequest)
	dataVolume := instResource.Volumes[0]

	lwdInstance := entity.LWDInstance{
		Instance: entity.Instance{
			Project:        project.Name,
			Name:           request.GetName(),
			Version:        request.GetVersion(),
			Network:        project.Network,
			Description:    lwdRequest.Description,
			Owner:          project.GetOwner(),
			Status:         "New",
			Timestamp:      time.Now(),
			DataSourceType: request.GetDataSourceType(),
			DataSource:     request.GetDataSource(),
			InstanceType:   request.GetInstanceType(),
			Action:         "created",
			ActionTime:     time.Now(),
		},
		LWDDetails: entity.LWDDetails{
			ZcashInstance: lwdRequest.ZcashInstance,
			DataVolume:    entity.DataVolume{Name: dataVolume + "-" + lwdRequest.Name, Size: 10, Volume: dataVolume},
		},
	}

	return &lwdInstance, nil
}

func (lwd *LWDInstanceResourceManager) UpdateInstance(ctx context.Context, project *entity.Project, instance entity.InstanceIF, request object.InstanceRequestIF) error {

	lwdRequest := request.(object.LWDInstanceRequest)
	lwdInstance := instance.(*entity.LWDInstance)

	lwdInstance.Action = "updated"
	lwdInstance.ActionTime = time.Now()
	lwdInstance.Description = lwdRequest.Description
	lwdInstance.ZcashInstance = lwdRequest.ZcashInstance

	return nil
}

func (lwd *LWDInstanceResourceManager) CreateDeploymentResourceAssets(ctx context.Context, instance entity.InstanceIF) ([]*unstructured.Unstructured, error) {
	lwdInstance := instance.(*entity.LWDInstance)
	instResource, ok := lwd.GetInstanceResources(lwdInstance.Version)
	if !ok {
		logger.Errorf(ctx, "Lightwallet resource not available for %s", lwdInstance.Version)
		return nil, errs.ErrInstanceResourceFailed
	}

	lwdImage := instResource.GetImage("lwd")
	if lwdImage == nil {
		return nil, errs.ErrInstanceResourceFailed
	}

	zcashInstance := fmt.Sprintf("zcashd-svc-%s.%s.svc.cluster.local", lwdInstance.ZcashInstance, lwdInstance.GetNamespace())

	zcashPort := lwd.lwdConfig.Ports["service"]
	zcashRsc, zcashOk := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.InstanceTypeZCASH)
	if zcashOk {
		zcashPort = zcashRsc.Ports["service"]
	}

	lwdSpec := spec.LWDInstanceSpec{
		InstanceSpec: spec.InstanceSpec{
			Name:               lwdInstance.Name,
			Project:            lwdInstance.Project,
			Version:            lwdInstance.Version,
			ServiceAccountName: vars.AppConfig.Policy.ServiceAccount,
			Namespace:          lwdInstance.GetNamespace(),
			Labels:             helper.CreateInstanceLabels(lwdInstance),
			DomainName:         vars.AppConfig.Policy.Domain,
			DomainSecret:       vars.AppConfig.Policy.CertName,
			DataSourceType:     lwdInstance.DataSourceType,
			DataSource:         lwdInstance.DataSource},
		ZcashInstanceName: lwdInstance.ZcashInstance,
		ZcashInstanceUrl:  zcashInstance,
		ZcashPort:         zcashPort,
		LightwalletImage:  lwdImage.URL,
		Port:              lwd.lwdConfig.Ports["service"],
		HttpPort:          lwd.lwdConfig.Ports["http"],
		LogLevel:          10,
		DataVolume:        lwdInstance.DataVolume.Name,
		Envoy:             helper.CreateEnvoySpec(lwd.lwdConfig.Ports["envoy"]),
	}

	var specArr []string
	var err error

	fileTemplate := instResource.GetFileTemplate()
	specArr, err = fileTemplate.ExecuteTemplates([]string{"LWD_CONF", "ZCASH_CONF", "ENVOY_CONF", "DEPLOYMENT", "SERVICE", "INGRESS"}, lwdSpec)
	if err != nil {
		logger.Errorf(ctx, "Lightwalletd templates for version %s failed - %s", lwdInstance.Version, err)
		return nil, errs.ErrInstanceResourceFailed
	}

	objects, err := helper.CreateYAMLObjects(specArr)
	if err != nil {
		logger.Errorf(ctx, "Lightwalletd templates for version %s failed - %s", lwdInstance.Version, err)
		return nil, errs.ErrInstanceResourceFailed
	}

	volumeDataSource := lwdInstance.DataSourceType == ztypes.VolumeDataSource
	snapshotDataSource := lwdInstance.DataSourceType == ztypes.SnapshotDataSource
	storageClass := vars.AppConfig.Policy.StorageClass

	var volumeSpecs = []spec.VolumeSpec{
		{Volume: lwdInstance.DataVolume.Volume, VolumeName: lwdInstance.DataVolume.Name, StorageClass: storageClass,
			Namespace: lwdInstance.GetNamespace(), SourceName: lwdInstance.DataSource, VolumeDataSource: volumeDataSource,
			SnapshotDataSource: snapshotDataSource, Size: lwdInstance.DataVolume.Size, Labels: lwdSpec.Labels},
	}

	appRsc := vars.ManagerFactory.GetAppResourceManager(ctx)
	volumes, err := appRsc.CreateVolumeAsset(ctx, volumeSpecs...)
	if err != nil {
		logger.Errorf(ctx, "Lightwalletd volume templates for version %s failed - %s", lwdInstance.Version, err)
		return nil, errs.ErrInstanceResourceFailed
	}

	objects = append(objects, volumes...)
	return objects, nil
}

func (lwd *LWDInstanceResourceManager) CreateStartResourceAssets(ctx context.Context, instance entity.InstanceIF) ([]*unstructured.Unstructured, error) {
	lwdInstance := instance.(*entity.LWDInstance)
	instResource, ok := lwd.GetInstanceResources(lwdInstance.Version)
	if !ok {
		logger.Errorf(ctx, "Lightwalletd resource not available for %s", lwdInstance.Version)
		return nil, errs.ErrInstanceResourceFailed
	}

	lwdImage := instResource.GetImage("lwd")
	if lwdImage == nil {
		return nil, errs.ErrInstanceResourceFailed
	}
	zcashInstance := fmt.Sprintf("zcashd-svc-%s.%s.svc.cluster.local", lwdInstance.ZcashInstance, lwdInstance.GetNamespace())

	zcashPort := lwd.lwdConfig.Ports["service"]
	zcashRsc, zcashOk := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.InstanceTypeZCASH)
	if zcashOk {
		zcashPort = zcashRsc.Ports["service"]
	}

	lwdSpec := spec.LWDInstanceSpec{
		InstanceSpec: spec.InstanceSpec{
			Name:               lwdInstance.Name,
			Project:            lwdInstance.Project,
			Version:            lwdInstance.Version,
			ServiceAccountName: vars.AppConfig.Policy.ServiceAccount,
			Namespace:          lwdInstance.GetNamespace(),
			Labels:             helper.CreateInstanceLabels(lwdInstance),
			DomainName:         vars.AppConfig.Policy.Domain,
			DomainSecret:       vars.AppConfig.Policy.CertName,
			DataSourceType:     lwdInstance.DataSourceType,
			DataSource:         lwdInstance.DataSource},
		ZcashInstanceName: lwdInstance.ZcashInstance,
		ZcashInstanceUrl:  zcashInstance,
		ZcashPort:         zcashPort,
		LightwalletImage:  lwdImage.URL,
		Port:              lwd.lwdConfig.Ports["service"],
		HttpPort:          lwd.lwdConfig.Ports["http"],
		LogLevel:          10,
		DataVolume:        lwdInstance.DataVolume.Name,
		Envoy:             helper.CreateEnvoySpec(lwd.lwdConfig.Ports["envoy"]),
	}

	var specArr []string
	var err error

	fileTemplate := instResource.GetFileTemplate()
	specArr, err = fileTemplate.ExecuteTemplates([]string{"DEPLOYMENT", "SERVICE"}, lwdSpec)
	if err != nil {
		logger.Errorf(ctx, "Lightwalletd templates for version %s failed - %s", lwdInstance.Version, err)
		return nil, errs.ErrInstanceResourceFailed
	}

	objects, err := helper.CreateYAMLObjects(specArr)
	if err != nil {
		logger.Errorf(ctx, "Lightwalletd templates for version %s failed - %s", lwdInstance.Version, err)
		return nil, errs.ErrInstanceResourceFailed
	}

	return objects, nil
}

func (lwd *LWDInstanceResourceManager) CreateIngressAsset(ctx context.Context, projIngress *unstructured.Unstructured, instance entity.InstanceIF, action ztypes.EventAction) (*unstructured.Unstructured, error) {
	lwdInstance := instance.(*entity.LWDInstance)
	instResource, ok := lwd.GetInstanceResources(lwdInstance.Version)
	if !ok {
		logger.Errorf(ctx, "Lightwalletd resource not available for %s", lwdInstance.Version)
		return nil, errs.ErrInstanceResourceFailed
	}

	zcashInstance := fmt.Sprintf("zcashd-svc-%s.%s.svc.cluster.local", lwdInstance.ZcashInstance, lwdInstance.GetNamespace())
	zcashPort := lwd.lwdConfig.Ports["service"]
	zcashRsc, zcashOk := vars.ResourceConfig.GetInstanceResourceConfig(ztypes.InstanceTypeZCASH)
	if zcashOk {
		zcashPort = zcashRsc.Ports["service"]
	}

	lwdSpec := spec.LWDInstanceSpec{
		InstanceSpec: spec.InstanceSpec{
			Name:               lwdInstance.Name,
			Project:            lwdInstance.Project,
			Version:            lwdInstance.Version,
			ServiceAccountName: vars.AppConfig.Policy.ServiceAccount,
			Namespace:          lwdInstance.GetNamespace(),
			Labels:             helper.CreateInstanceLabels(lwdInstance),
			DomainName:         vars.AppConfig.Policy.Domain,
			DomainSecret:       vars.AppConfig.Policy.CertName},
		ZcashInstanceName: lwdInstance.ZcashInstance,
		ZcashInstanceUrl:  zcashInstance,
		ZcashPort:         zcashPort,
		Envoy:             helper.CreateEnvoySpec(lwd.lwdConfig.Ports["envoy"]),
	}

	var specObj string
	var err error

	fileTemplate := instResource.GetFileTemplate()
	if action == "stopped" {
		specObj, err = fileTemplate.ExecuteTemplate("INGRESS_STOPPED", lwdSpec)
	} else {
		specObj, err = fileTemplate.ExecuteTemplate("INGRESS", lwdSpec)
	}
	if err != nil {
		logger.Errorf(ctx, "Lightwalletd templates for version %s failed - %s", lwdInstance.Version, err)
		return nil, errs.ErrInstanceResourceFailed
	}

	object, err := helper.CreateYAMLObject(specObj)
	if err != nil {
		logger.Errorf(ctx, "Lightwalletd templates for version %s failed - %s", lwdInstance.Version, err)
		return nil, errs.ErrInstanceResourceFailed
	}

	return object, nil
}

func (lwd *LWDInstanceResourceManager) CreateSnapshotAssets(ctx context.Context, instance entity.InstanceIF, volume string) ([]*unstructured.Unstructured, error) {

	var req object.SnapshotRequest

	lwdInstance := instance.(*entity.LWDInstance)
	appRsc := vars.ManagerFactory.GetAppResourceManager(ctx)
	req.Namespace = lwdInstance.GetNamespace()
	req.Volume = volume
	req.VolumeName = lwdInstance.DataVolume.Name
	req.Labels = helper.CreateInstanceLabels(lwdInstance)

	return appRsc.CreateSnapshotAsset(ctx, &req)
}

func (lwd *LWDInstanceResourceManager) CreateSnapshotScheduleAssets(ctx context.Context, instance entity.InstanceIF, volume string, scheduleType ztypes.ZBIBackupScheduleType) ([]*unstructured.Unstructured, error) {

	var req object.SnapshotScheduleRequest

	lwdInstance := instance.(*entity.LWDInstance)
	appRsc := vars.ManagerFactory.GetAppResourceManager(ctx)
	req.Namespace = lwdInstance.GetNamespace()
	req.Volume = volume
	req.Schedule = scheduleType
	req.VolumeName = lwdInstance.DataVolume.Name
	req.Labels = helper.CreateInstanceLabels(lwdInstance)

	return appRsc.CreateSnapshotScheduleAsset(ctx, &req)
}

func (lwd *LWDInstanceResourceManager) CreateRotationAssets(ctx context.Context, instance entity.InstanceIF) ([]*unstructured.Unstructured, error) {
	return []*unstructured.Unstructured{}, nil
}

func (lwd *LWDInstanceResourceManager) UnmarshalBSONDetails(ctx context.Context, value bson.Raw) (entity.InstanceIF, error) {
	logger.Tracef(ctx, "Unmarshaling Lightwalletd server instance details ............. %s", value.String())

	var lwdInstance entity.LWDInstance
	if err := bson.Unmarshal(value, &lwdInstance); err != nil {
		return nil, err
	}

	logger.Debugf(ctx, "Unmarshaled Lightwalletd server details - %s", utils.MarshalObject(lwdInstance))
	return &lwdInstance, nil
}
