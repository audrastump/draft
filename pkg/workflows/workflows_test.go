package workflows

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func createTempManifest(path string) (string, error) {
	file, err := ioutil.TempFile("", "*.yaml")
	if err != nil {
		return "", err
	}
	defer file.Close()

	var source *os.File
	source, err = os.Open(path)
	if err != nil {
		return "", err
	}
	defer source.Close()

	_, err = io.Copy(file, source)
	if err != nil {
		return "", err
	}
	return file.Name(), nil
}

func TestWorkflowEmbed(t *testing.T) {
	workflow := &workflowType{
		deployPath:         "/charts",
		workflowFileSuffix: "-helm",
	}

	assert.NotEmptyf(t, getWorkflowFile(workflow), "workflow should be fetched from the embeded file system")
}

func TestWorkflowReplace(t *testing.T) {
	config := &WorkflowConfig{
		AcrName:           "test",
		AksClusterName:    "test",
		ContainerName:     "test",
		ResourceGroupName: "test",
		BranchName:        "test",

		chartsOverridePath: "testOverride",
		kustomizePath:      "testKustomize",
	}

	ghw := &GitHubWorkflow{}
	ghw.On.Push.Branches = []string{"branch"}
	replaceWorkflowVars("", config, ghw)
	assert.NotNil(t, ghw.Env, "check that replace will update a ghw's environment")

	workflow, ok := deployNameToWorkflow["manifests"]
	assert.True(t, ok)

	ghw = getWorkflowFile(workflow)
	origLen := len(ghw.Jobs["build"].Steps)
	replaceWorkflowVars("manifests", config, ghw)
	assert.Equal(t, origLen-1, len(ghw.Jobs["build"].Steps), "check step is deleted")

	workflow, ok = deployNameToWorkflow["helm"]
	assert.True(t, ok)

	ghw = getWorkflowFile(workflow)
	replaceWorkflowVars("helm", config, ghw)
	assert.Equal(t, "testOverride", ghw.Env["CHART_OVERRIDE_PATH"], "check helm envs are replaced")

	workflow, ok = deployNameToWorkflow["kustomize"]
	assert.True(t, ok)

	ghw = getWorkflowFile(workflow)
	replaceWorkflowVars("kustomize", config, ghw)
	assert.Equal(t, "testKustomize", ghw.Env["KUSTOMIZE_PATH"], "check kustomize envs are replaces")
}

func TestUpdateProductionDeployments(t *testing.T) {
	config := &WorkflowConfig{
		AcrName:           "test",
		ContainerName:     "test",
		ResourceGroupName: "test",
	}
	assert.Nil(t, updateProductionDeployments("", ".", config))

	helmFileName, _ := createTempManifest("../../test/templates/helm_prod_values.yaml")
	deploymentFileName, _ := createTempManifest("../../test/templates/deployment.yaml")
	defer os.Remove(helmFileName)
	defer os.Remove(deploymentFileName)

	assert.Nil(t, setHelmContainerImage(helmFileName, "testImage"))
	file, _ := ioutil.ReadFile(helmFileName)
	var helmDeploy HelmProductionYaml
	_ = yaml.Unmarshal(file, &helmDeploy)
	assert.Equal(t, "testImage", helmDeploy.ImageKey.Repository)

	assert.Nil(t, setDeploymentContainerImage(deploymentFileName, "testImage"))
	file, _ = ioutil.ReadFile(deploymentFileName)
	var deploy DeploymentYaml
	_ = yaml.Unmarshal(file, &deploy)
	assert.Equal(t, "testImage", deploy.Spec.Template.Spec.Containers[0].Image)
}