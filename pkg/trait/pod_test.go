package trait

import (
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/kubernetes"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestConfigurePodTraitDoesSucceed(t *testing.T) {
	trait, environment, _ := createPodTest()
	configured, err := trait.Configure(environment)

	assert.False(t, configured)
	assert.NotNil(t, err)

	trait.Template = "{}"
	configured, err = trait.Configure(environment)

	assert.True(t, configured)
	assert.Nil(t, err)
}

func TestSimpleChange(t *testing.T) {
	templateString := "metadata:\n  name: template-test\n\nspec:\n  containers:\n    - name: second-container"
	template := testPodTemplateSpec(t, templateString)

	assert.Equal(t, "template-test", template.Name)
	assert.Equal(t, 3, len(template.Spec.Containers))
}

func TestMergeArrays(t *testing.T) {
	templateString := "{metadata: {name: test-template}, " +
		"spec: {containers: [{name: second-container, " +
		"env: [{name: SOME_VARIABLE, value: SOME_VALUE}, {name: SOME_VARIABLE2, value: SOME_VALUE2}]}, " +
		"{name: integration, env: [{name: TEST_ADDED_CUSTOM_VARIABLE, value: value}]}" +
		"]" +
		"}}"
	templateSpec := testPodTemplateSpec(t, templateString)

	assert.Equal(t, "test-template", templateSpec.Name)
	assert.NotNil(t, getContainer(templateSpec.Spec.Containers, "second-container"))
	assert.Equal(t, "SOME_VALUE", containsEnvVariables(templateSpec, "second-container", "SOME_VARIABLE"))
	assert.Equal(t, "SOME_VALUE2", containsEnvVariables(templateSpec, "second-container", "SOME_VARIABLE2"))
	assert.True(t, len(getContainer(templateSpec.Spec.Containers, "integration").Env) > 1)
	assert.Equal(t, "value", containsEnvVariables(templateSpec, "integration", "TEST_ADDED_CUSTOM_VARIABLE"))
}

func TestChangeEnvVariables(t *testing.T) {
	templateString := "{metadata: {name: test-template}, " +
		"spec: {containers: [" +
		"{name: second, env: [{name: TEST_VARIABLE, value: TEST_VALUE}]}, " +
		"{name: integration, env: [{name: CAMEL_K_DIGEST, value: new_value}]}" +
		"]}}"
	templateSpec := testPodTemplateSpec(t, templateString)

	//check if env var was added in second container
	assert.Equal(t, containsEnvVariables(templateSpec, "second", "TEST_VARIABLE"), "TEST_VALUE")
	assert.Equal(t, 3, len(getContainer(templateSpec.Spec.Containers, "second").Env))

	//check if env var was changed
	assert.Equal(t, containsEnvVariables(templateSpec, "integration", "CAMEL_K_DIGEST"), "new_value")
}
func TestRemoveArray(t *testing.T) {
	templateString := "{metadata: " +
		"{name: test-template}, " +
		"spec:" +
		" {containers: " +
		"[{name: second-container, env: [{name: SOME_VARIABLE, value: SOME_VALUE}, {name: SOME_VARIABLE2, value: SOME_VALUE2}]}, " +
		"{name: integration, env: null}" +
		"]}}"

	templateSpec := testPodTemplateSpec(t, templateString)
	assert.True(t, len(getContainer(templateSpec.Spec.Containers, "integration").Env) == 0)
}

func createPodTest() (*podTrait, *Environment, *appsv1.Deployment) {
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
		"{name: integration, command: [/bin/sh, '-c'], env: [{name: CAMEL_K_DIGEST, value: vO3wwJHC7-uGEiFFVac0jq6rZT5EZNw56Ae5gKKFZZsk}, {name: CAMEL_K_CONF, value: /etc/camel/conf/application.properties}, {name: CAMEL_K_CONF_D, value: /etc/camel/conf.d},{name: CAMEL_K_VERSION, value: 1.3.0-SNAPSHOT}, {name: CAMEL_K_INTEGRATION, value: test}, {name: CAMEL_K_RUNTIME_VERSION, value: 1.5.0}, {name: CAMEL_K_MOUNT_PATH_CONFIGMAPS, value: /etc/camel/conf.d/_configmaps}, {name: CAMEL_K_MOUNT_PATH_SECRETS, value: /etc/camel/conf.d/_secrets}, {name: NAMESPACE, valueFrom: {fieldRef: {apiVersion: v1, fieldPath: metadata.namespace}}}, {name: POD_NAME, valueFrom: {fieldRef: {apiVersion: v1, fieldPath: metadata.name}}}], imagePullPolicy: IfNotPresent, volumeMounts: [{name: i-source-000, mountPath: /etc/camel/sources/i-source-000}, {name: application-properties, mountPath: /etc/camel/conf}], terminationMessagePolicy: File, image: 'image-registry.openshift-image-registry.svc:5000/podtrait/camel-k-kit-bvd7utv170hult6ju26g@sha256:1c091437ef986f2852733da5f3fce7a5f48a5ea51e409f0bdcb0c13ff620e6b2', workingDir: /deployments, args: ['echo exec java -cp ./resources:/etc/camel/conf:/etc/camel/resources:/etc/camel/sources/i-source-000:dependencies/camel-k-integration-1.3.0-SNAPSHOT-runner.jar:dependencies/io.quarkus.arc.arc-1.8.0.Final.jar:dependencies/io.quarkus.quarkus-arc-1.8.0.Final.jar:dependencies/io.quarkus.quarkus-bootstrap-runner-1.8.0.Final.jar:dependencies/io.quarkus.quarkus-core-1.8.0.Final.jar:dependencies/io.quarkus.quarkus-development-mode-spi-1.8.0.Final.jar:dependencies/io.quarkus.quarkus-ide-launcher-1.8.0.Final.jar:dependencies/io.smallrye.common.smallrye-common-annotation-1.3.0.jar:dependencies/io.smallrye.common.smallrye-common-constraint-1.1.0.jar:dependencies/io.smallrye.common.smallrye-common-expression-1.1.0.jar:dependencies/io.smallrye.common.smallrye-common-function-1.1.0.jar:dependencies/io.smallrye.config.smallrye-config-1.8.6.jar:dependencies/io.smallrye.config.smallrye-config-common-1.8.6.jar:dependencies/jakarta.annotation.jakarta.annotation-api-1.3.5.jar:dependencies/jakarta.el.jakarta.el-api-3.0.3.jar:dependencies/jakarta.enterprise.jakarta.enterprise.cdi-api-2.0.2.jar:dependencies/jakarta.inject.jakarta.inject-api-1.0.jar:dependencies/jakarta.interceptor.jakarta.interceptor-api-1.2.5.jar:dependencies/jakarta.transaction.jakarta.transaction-api-1.3.3.jar:dependencies/org.apache.camel.camel-api-3.5.0.jar:dependencies/org.apache.camel.camel-base-3.5.0.jar:dependencies/org.apache.camel.camel-bean-3.5.0.jar:dependencies/org.apache.camel.camel-componentdsl-3.5.0.jar:dependencies/org.apache.camel.camel-core-catalog-3.5.0.jar:dependencies/org.apache.camel.camel-core-engine-3.5.0.jar:dependencies/org.apache.camel.camel-core-languages-3.5.0.jar:dependencies/org.apache.camel.camel-endpointdsl-3.5.0.jar:dependencies/org.apache.camel.camel-groovy-3.5.0.jar:dependencies/org.apache.camel.camel-log-3.5.0.jar:dependencies/org.apache.camel.camel-main-3.5.0.jar:dependencies/org.apache.camel.camel-management-api-3.5.0.jar:dependencies/org.apache.camel.camel-microprofile-config-3.5.0.jar:dependencies/org.apache.camel.camel-support-3.5.0.jar:dependencies/org.apache.camel.camel-timer-3.5.0.jar:dependencies/org.apache.camel.camel-tooling-model-3.5.0.jar:dependencies/org.apache.camel.camel-util-3.5.0.jar:dependencies/org.apache.camel.camel-util-json-3.5.0.jar:dependencies/org.apache.camel.k.camel-k-loader-groovy-1.5.0.jar:dependencies/org.apache.camel.k.camel-k-quarkus-core-1.5.0.jar:dependencies/org.apache.camel.k.camel-k-quarkus-loader-groovy-1.5.0.jar:dependencies/org.apache.camel.k.camel-k-runtime-core-1.5.0.jar:dependencies/org.apache.camel.k.camel-k-runtime-quarkus-1.5.0.jar:dependencies/org.apache.camel.quarkus.camel-quarkus-bean-1.1.0.jar:dependencies/org.apache.camel.quarkus.camel-quarkus-core-1.1.0.jar:dependencies/org.apache.camel.quarkus.camel-quarkus-endpointdsl-1.1.0.jar:dependencies/org.apache.camel.quarkus.camel-quarkus-log-1.1.0.jar:dependencies/org.apache.camel.quarkus.camel-quarkus-main-1.1.0.jar:dependencies/org.apache.camel.quarkus.camel-quarkus-support-common-1.1.0.jar:dependencies/org.apache.camel.quarkus.camel-quarkus-timer-1.1.0.jar:dependencies/org.codehaus.groovy.groovy-3.0.5.jar:dependencies/org.eclipse.microprofile.config.microprofile-config-api-1.4.jar:dependencies/org.eclipse.microprofile.context-propagation.microprofile-context-propagation-api-1.0.1.jar:dependencies/org.graalvm.sdk.graal-sdk-20.2.0.jar:dependencies/org.jboss.logging.jboss-logging-3.3.2.Final.jar:dependencies/org.jboss.logging.jboss-logging-annotations-2.1.0.Final.jar:dependencies/org.jboss.logmanager.jboss-logmanager-embedded-1.0.4.jar:dependencies/org.jboss.slf4j.slf4j-jboss-logging-1.2.0.Final.jar:dependencies/org.jboss.threads.jboss-threads-3.1.1.Final.jar:dependencies/org.slf4j.slf4j-api-1.7.30.jar:dependencies/org.wildfly.common.wildfly-common-1.5.4.Final-format-001.jar io.quarkus.runner.GeneratedMain && exec java -cp ./resources:/etc/camel/conf:/etc/camel/resources:/etc/camel/sources/i-source-000:dependencies/camel-k-integration-1.3.0-SNAPSHOT-runner.jar:dependencies/io.quarkus.arc.arc-1.8.0.Final.jar:dependencies/io.quarkus.quarkus-arc-1.8.0.Final.jar:dependencies/io.quarkus.quarkus-bootstrap-runner-1.8.0.Final.jar:dependencies/io.quarkus.quarkus-core-1.8.0.Final.jar:dependencies/io.quarkus.quarkus-development-mode-spi-1.8.0.Final.jar:dependencies/io.quarkus.quarkus-ide-launcher-1.8.0.Final.jar:dependencies/io.smallrye.common.smallrye-common-annotation-1.3.0.jar:dependencies/io.smallrye.common.smallrye-common-constraint-1.1.0.jar:dependencies/io.smallrye.common.smallrye-common-expression-1.1.0.jar:dependencies/io.smallrye.common.smallrye-common-function-1.1.0.jar:dependencies/io.smallrye.config.smallrye-config-1.8.6.jar:dependencies/io.smallrye.config.smallrye-config-common-1.8.6.jar:dependencies/jakarta.annotation.jakarta.annotation-api-1.3.5.jar:dependencies/jakarta.el.jakarta.el-api-3.0.3.jar:dependencies/jakarta.enterprise.jakarta.enterprise.cdi-api-2.0.2.jar:dependencies/jakarta.inject.jakarta.inject-api-1.0.jar:dependencies/jakarta.interceptor.jakarta.interceptor-api-1.2.5.jar:dependencies/jakarta.transaction.jakarta.transaction-api-1.3.3.jar:dependencies/org.apache.camel.camel-api-3.5.0.jar:dependencies/org.apache.camel.camel-base-3.5.0.jar:dependencies/org.apache.camel.camel-bean-3.5.0.jar:dependencies/org.apache.camel.camel-componentdsl-3.5.0.jar:dependencies/org.apache.camel.camel-core-catalog-3.5.0.jar:dependencies/org.apache.camel.camel-core-engine-3.5.0.jar:dependencies/org.apache.camel.camel-core-languages-3.5.0.jar:dependencies/org.apache.camel.camel-endpointdsl-3.5.0.jar:dependencies/org.apache.camel.camel-groovy-3.5.0.jar:dependencies/org.apache.camel.camel-log-3.5.0.jar:dependencies/org.apache.camel.camel-main-3.5.0.jar:dependencies/org.apache.camel.camel-management-api-3.5.0.jar:dependencies/org.apache.camel.camel-microprofile-config-3.5.0.jar:dependencies/org.apache.camel.camel-support-3.5.0.jar:dependencies/org.apache.camel.camel-timer-3.5.0.jar:dependencies/org.apache.camel.camel-tooling-model-3.5.0.jar:dependencies/org.apache.camel.camel-util-3.5.0.jar:dependencies/org.apache.camel.camel-util-json-3.5.0.jar:dependencies/org.apache.camel.k.camel-k-loader-groovy-1.5.0.jar:dependencies/org.apache.camel.k.camel-k-quarkus-core-1.5.0.jar:dependencies/org.apache.camel.k.camel-k-quarkus-loader-groovy-1.5.0.jar:dependencies/org.apache.camel.k.camel-k-runtime-core-1.5.0.jar:dependencies/org.apache.camel.k.camel-k-runtime-quarkus-1.5.0.jar:dependencies/org.apache.camel.quarkus.camel-quarkus-bean-1.1.0.jar:dependencies/org.apache.camel.quarkus.camel-quarkus-core-1.1.0.jar:dependencies/org.apache.camel.quarkus.camel-quarkus-endpointdsl-1.1.0.jar:dependencies/org.apache.camel.quarkus.camel-quarkus-log-1.1.0.jar:dependencies/org.apache.camel.quarkus.camel-quarkus-main-1.1.0.jar:dependencies/org.apache.camel.quarkus.camel-quarkus-support-common-1.1.0.jar:dependencies/org.apache.camel.quarkus.camel-quarkus-timer-1.1.0.jar:dependencies/org.codehaus.groovy.groovy-3.0.5.jar:dependencies/org.eclipse.microprofile.config.microprofile-config-api-1.4.jar:dependencies/org.eclipse.microprofile.context-propagation.microprofile-context-propagation-api-1.0.1.jar:dependencies/org.graalvm.sdk.graal-sdk-20.2.0.jar:dependencies/org.jboss.logging.jboss-logging-3.3.2.Final.jar:dependencies/org.jboss.logging.jboss-logging-annotations-2.1.0.Final.jar:dependencies/org.jboss.logmanager.jboss-logmanager-embedded-1.0.4.jar:dependencies/org.jboss.slf4j.slf4j-jboss-logging-1.2.0.Final.jar:dependencies/org.jboss.threads.jboss-threads-3.1.1.Final.jar:dependencies/org.slf4j.slf4j-api-1.7.30.jar:dependencies/org.wildfly.common.wildfly-common-1.5.4.Final-format-001.jar io.quarkus.runner.GeneratedMain']}], restartPolicy: Always, terminationGracePeriodSeconds: 90, dnsPolicy: ClusterFirst, securityContext: {}, schedulerName: default-scheduler}}"
	var template corev1.PodTemplateSpec
	_ = yaml.Unmarshal([]byte(specTemplateYamlString), &template)

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
	trait, environment, _ := createPodTest()
	trait.Template = template

	_, err := trait.Configure(environment)
	assert.Nil(t, err)

	err = trait.Apply(environment)
	assert.Nil(t, err)

	deployment := environment.Resources.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == "pod-template-test-integration"
	})

	return deployment.Spec.Template
}
