package trait

import (
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestConfigurePodTraitDoesSucceed(t *testing.T) {
	trait, environment, _ := createPodTest("")
	configured, err := trait.Configure(environment)

	assert.True(t, configured)
	assert.Nil(t, err)

	configured, err = trait.Configure(environment)

	assert.True(t, configured)
	assert.Nil(t, err)
}

func TestSimpleChange(t *testing.T) {
	templateString := `containers:
  - name: second-container
    env:
      - name: test
        value: test`
	template := testPodTemplateSpec(t, templateString)

	assert.Equal(t, 3, len(template.Spec.Containers))
}

func TestMergeArrays(t *testing.T) {
	templateString :=
		"{containers: [{name: second-container, " +
		"env: [{name: SOME_VARIABLE, value: SOME_VALUE}, {name: SOME_VARIABLE2, value: SOME_VALUE2}]}, " +
		"{name: integration, env: [{name: TEST_ADDED_CUSTOM_VARIABLE, value: value}]}" +
		"]" +
		"}"
	templateSpec := testPodTemplateSpec(t, templateString)

	assert.NotNil(t, getContainer(templateSpec.Spec.Containers, "second-container"))
	assert.Equal(t, "SOME_VALUE", containsEnvVariables(templateSpec, "second-container", "SOME_VARIABLE"))
	assert.Equal(t, "SOME_VALUE2", containsEnvVariables(templateSpec, "second-container", "SOME_VARIABLE2"))
	assert.True(t, len(getContainer(templateSpec.Spec.Containers, "integration").Env) > 1)
	assert.Equal(t, "value", containsEnvVariables(templateSpec, "integration", "TEST_ADDED_CUSTOM_VARIABLE"))
}

func TestChangeEnvVariables(t *testing.T) {
	templateString := "{containers: [" +
		"{name: second, env: [{name: TEST_VARIABLE, value: TEST_VALUE}]}, " +
		"{name: integration, env: [{name: CAMEL_K_DIGEST, value: new_value}]}" +
		"]}"
	templateSpec := testPodTemplateSpec(t, templateString)

	//check if env var was added in second container
	assert.Equal(t, containsEnvVariables(templateSpec, "second", "TEST_VARIABLE"), "TEST_VALUE")
	assert.Equal(t, 3, len(getContainer(templateSpec.Spec.Containers, "second").Env))

	//check if env var was changed
	assert.Equal(t, containsEnvVariables(templateSpec, "integration", "CAMEL_K_DIGEST"), "new_value")
}

func createPodTest(templateString string) (*podTrait, *Environment, *appsv1.Deployment) {
	trait := newPodTrait().(*podTrait)
	enabled := true
	trait.Enabled = &enabled

	specTemplateYamlString := "{metadata: {name: example-template, creationTimestamp: null, " +
		"labels: {camel.apache.org/integration: test}}, " +
		"spec: {volumes: [" +
		"{name: i-source-000," + "configMap: {name: test-source-000, items: [{key: content, path: test.groovy}], defaultMode: 420}}, " +
		"{name: application-properties, configMap: {name: test-application-properties, items: [{key: application.properties, path: application.properties}], defaultMode: 420}}], " +
		"containers: [" +
		"{name: second, env: [{name: SOME_VARIABLE, value: SOME_VALUE}, {name: SOME_VARIABLE2, value: SOME_VALUE2}]}," +
		"{name: integration, command: [/bin/sh, '-c'], env: [{name: CAMEL_K_DIGEST, value: vO3wwJHC7-uGEiFFVac0jq6rZT5EZNw56Ae5gKKFZZsk}, {name: CAMEL_K_CONF, value: /etc/camel/conf/application.properties}, {name: CAMEL_K_CONF_D, value: /etc/camel/conf.d},{name: CAMEL_K_VERSION, value: 1.3.0-SNAPSHOT}, {name: CAMEL_K_INTEGRATION, value: test}, {name: CAMEL_K_RUNTIME_VERSION, value: 1.5.0}, {name: CAMEL_K_MOUNT_PATH_CONFIGMAPS, value: /etc/camel/conf.d/_configmaps}, {name: CAMEL_K_MOUNT_PATH_SECRETS, value: /etc/camel/conf.d/_secrets}, {name: NAMESPACE, valueFrom: {fieldRef: {apiVersion: v1, fieldPath: metadata.namespace}}}, {name: POD_NAME, valueFrom: {fieldRef: {apiVersion: v1, fieldPath: metadata.name}}}], imagePullPolicy: IfNotPresent, volumeMounts: [{name: i-source-000, mountPath: /etc/camel/sources/i-source-000}, {name: application-properties, mountPath: /etc/camel/conf}], terminationMessagePolicy: File, image: 'image-registry.openshift-image-registry.svc:5000/podtrait/camel-k-kit-bvd7utv170hult6ju26g@sha256:1c091437ef986f2852733da5f3fce7a5f48a5ea51e409f0bdcb0c13ff620e6b2', workingDir: /deployments, args: ['echo exec java -cp ./resources:/etc/camel/conf:/etc/camel/resources:/etc/camel/sources/i-source-000 io.quarkus.runner.GeneratedMain']}], restartPolicy: Always, terminationGracePeriodSeconds: 90, dnsPolicy: ClusterFirst, securityContext: {}, schedulerName: default-scheduler}}"
	var template corev1.PodTemplateSpec
	_ = yaml.Unmarshal([]byte(specTemplateYamlString), &template)

	var podTemplateIt v1.PodSpec
	if templateString != "" {
		_ = yaml.Unmarshal([]byte(templateString), &podTemplateIt)
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod-template-test-integration",
		},
		Spec: appsv1.DeploymentSpec{
			Template: template,
		},
	}

	environment := &Environment{
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pod-template-test-integration",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				PodTemplate: &v1.PodSpecTemplate{
					Spec: podTemplateIt,
				},
			},
		},

		Resources: kubernetes.NewCollection(deployment),
	}
	return trait, environment, deployment
}

func containsEnvVariables(template corev1.PodTemplateSpec, containerName string, name string) string {
	container := getContainer(template.Spec.Containers, containerName)

	for i := range container.Env {
		envv := container.Env[i]
		if envv.Name == name {
			return envv.Value
		}
	}
	return "not found!"
}

func getContainer(containers []corev1.Container, name string) *corev1.Container {
	for i := range containers {
		if containers[i].Name == name {
			return &containers[i]
		}
	}
	return nil
}

func testPodTemplateSpec(t *testing.T, template string) corev1.PodTemplateSpec {
	trait, environment, _ := createPodTest(template)
	//trait.Template = template

	_, err := trait.Configure(environment)
	assert.Nil(t, err)

	err = trait.Apply(environment)
	assert.Nil(t, err)

	deployment := environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == "pod-template-test-integration"
	})

	return deployment.Spec.Template
}
