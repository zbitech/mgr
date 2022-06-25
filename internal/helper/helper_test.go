package helper

import (
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

var (
	//	DeploymentYAML = "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: nginx-deployment\n  namespace: project\n  labels:\n    platform: zbi\n    project: project\n    instance: instance\n    app: nginx\nspec:\n  replicas: 1\n  selector:\n    matchLabels:\n      platform: zbi\n      project: project\n      instance: instance\n      app: nginx\n  template:\n    metadata:\n      labels:\n        platform: zbi\n        project: project\n        instance: instance\n        app: nginx\n    spec:\n      containers:\n        - name: nginx\n          image: nginx:1.14.2\n          ports:\n            - containerPort: 80\n"

	DeploymentYAML = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: project
  labels:
    platform: zbi
    project: project
    instance: instance
    app: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      platform: zbi
      project: project
      instance: instance
      app: nginx
  template:
    metadata:
      labels:
        platform: zbi
        project: project
        instance: instance
        app: nginx
    spec:
      containers:
        - name: nginx
          image: nginx:1.14.2
          ports:
            - containerPort: 80
`

	IngressJSON = `
{
  "apiVersion": "projectcontour.io/v1",
  "kind": "HTTPProxy",
  "metadata": {
    "name": "zbi-proxy",
    "namespace": "zbi",
    "labels": {
      "platform": "zbi"
    }
  },
  "spec": {
    "virtualhost": {
      "fqdn": "api.zbitech.local",
      "tls": {
        "secretName": "kube-system/zbi-tls"
      },
      "includes": []
    }
  }
}
`
)

func Test_DecodeYAML(t *testing.T) {

	var obj = new(unstructured.Unstructured)
	err := DecodeYAML(DeploymentYAML, obj)
	assert.NoErrorf(t, err, "Failed to generate object from YAML - %s", err)
	assert.NotNilf(t, obj, "Failed to generate object from YAML")
}

func Test_EncodeYAML(t *testing.T) {
	var obj = new(unstructured.Unstructured)
	DecodeYAML(DeploymentYAML, obj)
	data, err := EncodeYAML(obj)
	assert.NoErrorf(t, err, "Failed to generate YAML from object - %s", err)
	assert.NotNilf(t, data, "Failed to convert to YAML")
}

func Test_DecodeJSON(t *testing.T) {

	var obj = new(unstructured.Unstructured)
	err := DecodeJSON(IngressJSON, obj)
	assert.NoErrorf(t, err, "Failed to generate object from JSON - %s", err)
	assert.NotNilf(t, obj, "Failed to generate object from JSON")

	content := obj.UnstructuredContent()
	spec := content["spec"].(map[string]interface{})
	vhost := spec["virtualhost"].(map[string]interface{})
	includes := vhost["includes"].([]interface{})
	includes = append(includes, "ABCDEFG")
	vhost["includes"] = includes

	assert.Lenf(t, vhost["includes"], 1, "Failed to add content")
}

func Test_EncodeJSON(t *testing.T) {
	var obj = new(unstructured.Unstructured)
	DecodeJSON(IngressJSON, obj)
	data, err := EncodeJSON(obj)
	assert.NoErrorf(t, err, "Failed to generate JSON from object - %s", err)
	assert.NotNilf(t, data, "Failed to convert to JSON")
}
