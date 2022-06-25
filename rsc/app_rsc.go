package rsc

import (
	"context"
	"github.com/zbitech/common/interfaces"
	"github.com/zbitech/common/pkg/errs"
	"github.com/zbitech/common/pkg/logger"
	"github.com/zbitech/common/pkg/model/config"
	"github.com/zbitech/common/pkg/model/object"
	"github.com/zbitech/common/pkg/model/spec"
	"github.com/zbitech/common/pkg/vars"
	"github.com/zbitech/mgr/internal/helper"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type AppResourceManager struct {
	resourceConfig *config.AppResourceConfig
	projManager    interfaces.ProjectResourceManagerIF
}

func NewAppResourceManager(resourceConfig *config.AppResourceConfig,
	projManager interfaces.ProjectResourceManagerIF) (interfaces.AppResourceManagerIF, error) {
	var mgr = &AppResourceManager{
		resourceConfig: resourceConfig,
		projManager:    projManager,
	}

	for _, cfg := range mgr.resourceConfig.Versions {
		if err := cfg.Init(vars.ASSET_PATH_DIRECTORY, object.NO_FUNCS); err != nil {
			return nil, err
		}
	}

	return mgr, nil
}

func (app *AppResourceManager) GetAppResources(version string) (*config.VersionedResourceConfig, bool) {
	resource, ok := app.resourceConfig.Versions[version]
	return resource, ok
}

//func (app *AppResourceManager) CreateControllerIngressAsset(ctx context.Context, obj *unstructured.Unstructured, project *entity.Project, action string) (*unstructured.Unstructured, error) {
//
//	appRsc, ok := app.GetAppResources(app.resourceConfig.Version)
//	if !ok {
//		logger.Errorf(ctx, "Controller App resource not available for %s", app.resourceConfig.Version)
//		return nil, errs.ErrIngressResourceFailed
//	}
//
//	fileTemplate := appRsc.GetFileTemplate()
//
//	if obj == nil {
//		obj = new(unstructured.Unstructured)
//
//		ctrlSpec := spec.ControllerIngressSpec{
//			ControllerDomain: vars.AppConfig.Policy.Domain,
//			CertSecret:       vars.AppConfig.Policy.CertName,
//		}
//
//		ctrlObj, err := fileTemplate.ExecuteTemplate("CONTROLLER_INGRESS", ctrlSpec)
//		if err != nil {
//			logger.Errorf(ctx, "Controller app template failed - %s", err)
//			return nil, errs.ErrIngressResourceFailed
//		}
//
//		if err = helper.DecodeJSON(ctrlObj, obj); err != nil {
//			logger.Errorf(ctx, "Controller app template failed - %s", err)
//			return nil, errs.ErrIngressResourceFailed
//		}
//	} else {
//		utils.RemoveResourceField(obj, "metadata.managedFields")
//		utils.RemoveResourceField(obj, "spec.status")
//	}
//
//	if action == "remove" {
//		includes, ok := utils.GetResourceField(obj, "spec.includes").([]k8s.IngressInclude)
//		if ok {
//			for index, include := range includes {
//				if include.Name == project.Name && include.Namespace == project.GetNamespace() {
//					includes = append(includes[:index], includes[index+1:]...)
//					utils.SetResourceField(obj, "spec.includes", includes)
//				}
//			}
//		} else {
//			logger.Errorf(ctx, "Unable to cast object as k8s.IngressInclude")
//		}
//	} else {
//		projObj, err := fileTemplate.ExecuteTemplate("PROJECT_CONTROLLER_INGRESS", project)
//		if err != nil {
//			logger.Errorf(ctx, "Project app template failed - %s", err)
//			return nil, errs.ErrIngressResourceFailed
//		}
//
//		var projIngress map[string]interface{}
//		if err = json.Unmarshal([]byte(projObj), &projIngress); err != nil {
//			logger.Errorf(ctx, "Project app template failed - %s", err)
//			return nil, errs.ErrIngressResourceFailed
//		}
//
//		utils.AddResourceField(obj, "spec.includes", projIngress)
//	}
//
//	return obj, nil
//
//}

//func (app *AppResourceManager) CreateProjectIngressAsset(ctx context.Context, obj *unstructured.Unstructured, project *entity.Project, instance entity.InstanceIF, action string) (*unstructured.Unstructured, error) {
//
//	appRsc, ok := app.GetAppResources(app.resourceConfig.Version)
//	if !ok {
//		logger.Errorf(ctx, "app resource not available for %s", app.resourceConfig.Version)
//		return nil, errs.ErrIngressResourceFailed
//	}
//
//	fileTemplate := appRsc.GetFileTemplate()
//
//	if obj == nil {
//		obj = new(unstructured.Unstructured)
//
//		projObj, err := fileTemplate.ExecuteTemplate("PROJECT_INGRESS", project)
//		if err != nil {
//			logger.Errorf(ctx, "Project app template failed - %s", err)
//			return nil, errs.ErrIngressResourceFailed
//		}
//
//		if err = helper.DecodeJSON(projObj, obj); err != nil {
//			logger.Errorf(ctx, "Controller app template failed - %s", err)
//			return nil, errs.ErrIngressResourceFailed
//		}
//		utils.SetResourceField(obj, "metadata.labels", helper.CreateProjectLabels(project))
//	} else {
//		utils.RemoveResourceField(obj, "metadata.managedFields")
//		utils.RemoveResourceField(obj, "spec.status")
//	}
//
//	iResource, ok := app.projManager.GetInstanceResources(instance.GetInstanceType(), instance.GetVersion())
//	if ok {
//		instanceSpec := spec.InstanceIngressSpec{Name: instance.GetName(), Version: instance.GetVersion(),
//			ServicePrefix: iResource.Service.Prefix, ServicePort: iResource.Service.ProxyPort}
//
//		instanceObj, err := fileTemplate.ExecuteTemplate("INSTANCE_PROJECT_INGRESS", instanceSpec)
//		if err != nil {
//			logger.Errorf(ctx, "Project app template failed - %s", err)
//			return nil, errs.ErrIngressResourceFailed
//		}
//
//		//		var route map[string]interface{}
//		var route k8s.IngressRoute
//		if err = json.Unmarshal([]byte(instanceObj), &route); err != nil {
//			logger.Errorf(ctx, "Project app template failed - %s", err)
//			return nil, errs.ErrIngressResourceFailed
//		}
//		logger.Infof(ctx, "Route: %s", utils.MarshalObject(route))
//
//		if action == "remove" {
//
//			routeData := utils.MarshalObject(utils.GetResourceField(obj, "spec.routes"))
//			var routes []k8s.IngressRoute
//			if err = json.Unmarshal([]byte(routeData), &routes); err != nil {
//				logger.Errorf(ctx, "Error unmarshaling ingress routes - %s", err)
//			} else {
//				for _, r := range routes {
//					for index, condition := range r.Conditions {
//						if condition.Prefix == route.Conditions[0].Prefix {
//							routes = append(routes[:index], routes[index+1:]...)
//							utils.SetResourceField(obj, "spec.routes", routes)
//						}
//					}
//				}
//			}
//		} else {
//			utils.AddResourceField(obj, "spec.routes", route)
//		}
//	} else {
//		//TODO - log failure to get resource
//		logger.Errorf(ctx, "Unable to get resource for instance %s version %s", instance.GetInstanceType(), instance.GetVersion())
//	}
//
//	return obj, nil
//}

func (app *AppResourceManager) CreateVolumeAsset(ctx context.Context, volumes ...spec.VolumeSpec) ([]*unstructured.Unstructured, error) {

	appRsc, ok := app.GetAppResources(app.resourceConfig.Version)
	if !ok {
		logger.Errorf(ctx, "app resource not available for %s", app.resourceConfig.Version)
		return nil, errs.ErrIngressResourceFailed
	}

	fileTemplate := appRsc.GetFileTemplate()

	var objects = make([]*unstructured.Unstructured, 0, len(volumes))
	for _, volume := range volumes {
		data, err := fileTemplate.ExecuteTemplate("VOLUME", volume)
		if err != nil {
			logger.Errorf(ctx, "volume template failed - %s", err)
			return nil, errs.ErrIngressResourceFailed
		}

		object, err := helper.CreateYAMLObject(data)
		if err != nil {
			logger.Errorf(ctx, "volume template failed - %s", err)
			return nil, errs.ErrIngressResourceFailed
		}
		objects = append(objects, object)
	}

	return objects, nil
}

//func (app *AppResourceManager) CreateConfigMapAsset(ctx context.Context, configs ...spec.DataSpec) ([]*unstructured.Unstructured, error) {
//	appRsc, ok := app.GetAppResources(app.version)
//	if !ok {
//		logger.Errorf(ctx, "app resource not available for %s", app.version)
//		return nil, errs.ErrIngressResourceFailed
//	}
//
//	fileTemplate := appRsc.GetFileTemplate()
//	var objects = make([]*unstructured.Unstructured, 0, len(configs))
//	for _, cfg := range configs {
//		data, err := fileTemplate.ExecuteTemplate("CONFIG", cfg)
//		if err != nil {
//			logger.Errorf(ctx, "configmap template failed - %s", err)
//			return nil, errs.ErrIngressResourceFailed
//		}
//
//		object, err := helper.CreateYAMLObject(data)
//		objects = append(objects, object)
//	}
//
//	return objects, nil
//}

//func (app *AppResourceManager) CreateSecretAsset(ctx context.Context, configs ...spec.DataSpec) ([]*unstructured.Unstructured, error) {
//
//	appRsc, ok := app.GetAppResources(app.version)
//	if !ok {
//		logger.Errorf(ctx, "app resource not available for %s", app.version)
//		return nil, errs.ErrIngressResourceFailed
//	}
//
//	fileTemplate := appRsc.GetFileTemplate()
//	var objects = make([]*unstructured.Unstructured, 0, len(configs))
//	for _, cfg := range configs {
//		data, err := fileTemplate.ExecuteTemplate("SECRET", cfg)
//		if err != nil {
//			logger.Errorf(ctx, "secret template failed - %s", err)
//			return nil, errs.ErrIngressResourceFailed
//		}
//
//		object, err := helper.CreateYAMLObject(data)
//		objects = append(objects, object)
//	}
//
//	return objects, nil
//}

func (app *AppResourceManager) CreateSnapshotAsset(ctx context.Context, req *object.SnapshotRequest) ([]*unstructured.Unstructured, error) {

	appRsc, ok := app.GetAppResources(req.Version)
	if !ok {
		logger.Errorf(ctx, "app resource not available for %s", req.Version)
		return nil, errs.ErrIngressResourceFailed
	}

	fileTemplate := appRsc.GetFileTemplate()

	snapshotClass := vars.AppConfig.Policy.SnapshotClass

	var specArr []string
	var err error

	snapshotSpec := spec.SnapshotSpec{
		Name:          req.VolumeName,
		Namespace:     req.Namespace,
		Volume:        req.Volume,
		SnapshotClass: snapshotClass,
		Labels:        req.Labels,
	}

	specArr, err = fileTemplate.ExecuteTemplates([]string{"SNAPSHOT"}, snapshotSpec)

	if err != nil {
		logger.Errorf(ctx, "backup templates for version %s failed - %s", req.Labels["version"], err)
		return nil, errs.ErrInstanceResourceFailed
	}

	return helper.CreateYAMLObjects(specArr)
}

func (app *AppResourceManager) CreateSnapshotScheduleAsset(ctx context.Context, req *object.SnapshotScheduleRequest) ([]*unstructured.Unstructured, error) {
	appRsc, ok := app.GetAppResources(req.Version)
	if !ok {
		logger.Errorf(ctx, "app resource not available for %s", req.Version)
		return nil, errs.ErrIngressResourceFailed
	}

	fileTemplate := appRsc.GetFileTemplate()

	snapshotClass := vars.AppConfig.Policy.SnapshotClass
	expiration := vars.AppConfig.Policy.BackupExpiration
	maxBackupCount := vars.AppConfig.Policy.MaxBackupCount

	var specArr []string
	var err error

	snapshotSpec := spec.SnapshotScheduleSpec{
		Name:             req.VolumeName,
		Namespace:        req.Namespace,
		Volume:           req.Volume,
		SnapshotClass:    snapshotClass,
		BackupExpiration: expiration,
		MaxBackupCount:   maxBackupCount,
		ScheduleType:     req.Schedule,
		Schedule:         helper.CreateSnapshotSchedule(req.Schedule),
		Labels:           req.Labels,
	}

	specArr, err = fileTemplate.ExecuteTemplates([]string{"SCHEDULE_SNAPSHOT"}, snapshotSpec)

	if err != nil {
		logger.Errorf(ctx, "backup templates for version %s failed - %s", req.Labels["version"], err)
		return nil, errs.ErrInstanceResourceFailed
	}

	return helper.CreateYAMLObjects(specArr)
}
