package helper

import (
	"bytes"
	"fmt"
	"github.com/zbitech/common/pkg/model/entity"
	"github.com/zbitech/common/pkg/model/spec"
	"github.com/zbitech/common/pkg/model/ztypes"
	"github.com/zbitech/common/pkg/utils"
	"github.com/zbitech/common/pkg/vars"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	JSONSerializer = k8sjson.NewSerializerWithOptions(k8sjson.DefaultMetaFactory, scheme.Scheme, scheme.Scheme, k8sjson.SerializerOptions{Pretty: true})
	YAMLSerializer = k8sjson.NewSerializerWithOptions(k8sjson.DefaultMetaFactory, scheme.Scheme, scheme.Scheme, k8sjson.SerializerOptions{Yaml: true})
)

func DecodeYAML(yaml string, object *unstructured.Unstructured) error {
	_, _, err := YAMLSerializer.Decode([]byte(yaml), nil, object)
	if err != nil {
		return err
	}

	return nil
}

func EncodeYAML(object *unstructured.Unstructured) (string, error) {
	var buffer = new(bytes.Buffer)
	if err := YAMLSerializer.Encode(object, buffer); err != nil {
		return "", err
	}

	return buffer.String(), nil
}

func DecodeJSON(data string, object *unstructured.Unstructured) error {
	_, _, err := JSONSerializer.Decode([]byte(data), nil, object)
	if err != nil {
		return err
	}

	return nil
}

func EncodeJSON(object *unstructured.Unstructured) (string, error) {
	var buffer = new(bytes.Buffer)
	if err := JSONSerializer.Encode(object, buffer); err != nil {
		return "", err
	}

	return buffer.String(), nil
}

func CreateYAMLObjects(specArr []string) ([]*unstructured.Unstructured, error) {
	var objects = make([]*unstructured.Unstructured, len(specArr))
	for index, yamlString := range specArr {
		objects[index] = new(unstructured.Unstructured)
		if err := DecodeYAML(yamlString, objects[index]); err != nil {
			return nil, err
		}
	}

	return objects, nil
}

func CreateYAMLObject(yamlString string) (*unstructured.Unstructured, error) {
	var object = new(unstructured.Unstructured)
	if err := DecodeYAML(yamlString, object); err != nil {
		return nil, err
	}

	return object, nil
}

func CreateProjectLabels(project *entity.Project) map[string]string {
	return map[string]string{
		"platform": "zbi",
		"project":  project.Name,
		"version":  project.Version,
		"owner":    project.Owner,
		"network":  string(project.Network),
	}
}

func CreateInstanceLabels(instance entity.InstanceIF) map[string]string {
	return map[string]string{
		"platform": "zbi",
		"project":  instance.GetProject(),
		"instance": instance.GetName(),
		"type":     string(instance.GetInstanceType()),
		"version":  instance.GetVersion(),
	}
}

func CreateEnvoySpec(envoyServicePort int32) spec.EnvoySpec {
	return spec.EnvoySpec{
		Image:                 vars.AppConfig.Envoy.Image,
		Command:               utils.MarshalObject(vars.AppConfig.Envoy.Command),
		Port:                  envoyServicePort,
		Timeout:               vars.AppConfig.Envoy.Timeout,
		AccessAuthorization:   vars.AppConfig.Features.AccessAuthorizationEnabled,
		AuthServerURL:         vars.AppConfig.Envoy.AuthServerURL,
		AuthServerPort:        vars.AppConfig.Envoy.AuthServerPort,
		AuthenticationEnabled: vars.AppConfig.Envoy.AuthenticationEnabled,
	}
}

func CreateSnapshotSchedule(schedule ztypes.ZBIBackupScheduleType) string {
	if schedule == ztypes.DailySnapshotSchedule {
		hour := 5
		min := 1
		return fmt.Sprintf("%d %d * * *", min, hour)
	} else if schedule == ztypes.WeeklySnapshotSchedule {
		weekDay := 1
		return fmt.Sprintf("* * * * %d", weekDay)
	} else if schedule == ztypes.MonthlySnapshotSchedule {
		day := 1
		month := 1
		return fmt.Sprintf("* * %d %d *", day, month)
	}

	return ""
}
