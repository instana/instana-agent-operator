/*
(c) Copyright IBM Corp. 2024
*/

package env

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

func TestEnvBuilderBuildPanicsWhenEnvVarNotExists(t *testing.T) {
	assertions := require.New(t)

	builder := NewEnvBuilder(&instanav1.InstanaAgent{}, nil)
	assertions.PanicsWithError(
		"unknown environment variable requested", func() {
			builder.Build(EnvVar(9999))
		},
	)
}

func TestEnvBuilderBuild(t *testing.T) {
	for _, test := range []struct {
		name     string
		agent    *instanav1.InstanaAgent
		zone     *instanav1.Zone
		envVars  []EnvVar
		expected []corev1.EnvVar
	}{
		{
			name: "Should produce all env vars with values from the Instana Agent Spec",
			agent: &instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					UseSecretMounts: pointer.To(false),
					Zone:            instanav1.Name{Name: "INSTANA_AGENT_SPEC_ZONE_NAME"},
					Cluster:         instanav1.Name{Name: "INSTANA_AGENT_SPEC_CLUSTER_NAME"},
					Agent: instanav1.BaseAgentSpec{
						EndpointHost:              "INSTANA_AGENT_ENDPOINT_HOST",
						EndpointPort:              "INSTANA_AGENT_ENDPOINT_PORT",
						MvnRepoUrl:                "INSTANA_MVN_REPOSITORY_URL",
						MvnRepoFeaturesPath:       "INSTANA_MVN_REPOSITORY_FEATURES_PATH",
						MvnRepoSharedPath:         "INSTANA_MVN_REPOSITORY_SHARED_PATH",
						MirrorReleaseRepoUrl:      "AGENT_RELEASE_REPOSITORY_MIRROR_URL",
						MirrorReleaseRepoUsername: "AGENT_RELEASE_REPOSITORY_MIRROR_USERNAME",
						MirrorReleaseRepoPassword: "AGENT_RELEASE_REPOSITORY_MIRROR_PASSWORD",
						MirrorSharedRepoUrl:       "INSTANA_SHARED_REPOSITORY_MIRROR_URL",
						MirrorSharedRepoUsername:  "INSTANA_SHARED_REPOSITORY_MIRROR_USERNAME",
						MirrorSharedRepoPassword:  "INSTANA_SHARED_REPOSITORY_MIRROR_PASSWORD",
						ProxyHost:                 "INSTANA_AGENT_PROXY_HOST",
						ProxyPort:                 "",
						ProxyProtocol:             "INSTANA_AGENT_PROXY_PROTOCOL",
						ProxyUser:                 "INSTANA_AGENT_PROXY_USER",
						ProxyPassword:             "INSTANA_AGENT_PROXY_PASSWORD",
						ProxyUseDNS:               true,
						ListenAddress:             "INSTANA_AGENT_HTTP_LISTEN",
						RedactKubernetesSecrets:   "INSTANA_KUBERNETES_REDACT_SECRETS",
						Mode:                      "INSTANA_BASE_AGENT_SPEC_MODE",
						KeysSecret:                "INSTANA_AGENT_KEYS_SECRET",
						Env: map[string]string{
							"USER_SPECIFIED_ENV_VAR1": "USER_SPECIFIED_ENV_VAR_VAL1",
							"USER_SPECIFIED_ENV_VAR2": "USER_SPECIFIED_ENV_VAR_VAL2",
						},
					},
				},
			},
			envVars: []EnvVar{
				AgentModeEnv,
				ZoneNameEnv,
				ClusterNameEnv,
				AgentEndpointEnv,
				AgentEndpointPortEnv,
				MavenRepoURLEnv,
				MavenRepoFeaturesPath,
				MavenRepoSharedPath,
				MirrorReleaseRepoUrlEnv,
				MirrorReleaseRepoUsernameEnv,
				MirrorReleaseRepoPasswordEnv,
				MirrorSharedRepoUrlEnv,
				MirrorSharedRepoUsernameEnv,
				MirrorSharedRepoPasswordEnv,
				ProxyHostEnv,
				ProxyPortEnv,
				ProxyProtocolEnv,
				ProxyUserEnv,
				ProxyPasswordEnv,
				ProxyUseDNSEnv,
				ListenAddressEnv,
				RedactK8sSecretsEnv,
				AgentZoneEnv,
				HTTPSProxyEnv,
				BackendURLEnv,
				NoProxyEnv,
				ConfigPathEnv,
				BackendEnv,
				InstanaAgentKeyEnv,
				AgentKeyEnv,
				DownloadKeyEnv,
				InstanaAgentPodNameEnv,
				PodNameEnv,
				PodIPEnv,
				PodUIDEnv,
				PodNamespaceEnv,
				K8sServiceDomainEnv,
				EntrypointSkipBackendTemplateGeneration,
			},
			expected: []corev1.EnvVar{
				{Name: "USER_SPECIFIED_ENV_VAR1", Value: "USER_SPECIFIED_ENV_VAR_VAL1"},
				{Name: "USER_SPECIFIED_ENV_VAR2", Value: "USER_SPECIFIED_ENV_VAR_VAL2"},
				{Name: "INSTANA_AGENT_MODE", Value: "INSTANA_BASE_AGENT_SPEC_MODE"},
				{Name: "INSTANA_ZONE", Value: "INSTANA_AGENT_SPEC_ZONE_NAME"},
				{Name: "INSTANA_KUBERNETES_CLUSTER_NAME", Value: "INSTANA_AGENT_SPEC_CLUSTER_NAME"},
				{Name: "INSTANA_AGENT_ENDPOINT", Value: "INSTANA_AGENT_ENDPOINT_HOST"},
				{Name: "INSTANA_AGENT_ENDPOINT_PORT", Value: "INSTANA_AGENT_ENDPOINT_PORT"},
				{Name: "INSTANA_MVN_REPOSITORY_URL", Value: "INSTANA_MVN_REPOSITORY_URL"},
				{Name: "INSTANA_MVN_REPOSITORY_FEATURES_PATH", Value: "INSTANA_MVN_REPOSITORY_FEATURES_PATH"},
				{Name: "INSTANA_MVN_REPOSITORY_SHARED_PATH", Value: "INSTANA_MVN_REPOSITORY_SHARED_PATH"},
				{Name: "AGENT_RELEASE_REPOSITORY_MIRROR_URL", Value: "AGENT_RELEASE_REPOSITORY_MIRROR_URL"},
				{Name: "AGENT_RELEASE_REPOSITORY_MIRROR_USERNAME", Value: "AGENT_RELEASE_REPOSITORY_MIRROR_USERNAME"},
				{Name: "AGENT_RELEASE_REPOSITORY_MIRROR_PASSWORD", Value: "AGENT_RELEASE_REPOSITORY_MIRROR_PASSWORD"},
				{Name: "INSTANA_SHARED_REPOSITORY_MIRROR_URL", Value: "INSTANA_SHARED_REPOSITORY_MIRROR_URL"},
				{Name: "INSTANA_SHARED_REPOSITORY_MIRROR_USERNAME", Value: "INSTANA_SHARED_REPOSITORY_MIRROR_USERNAME"},
				{Name: "INSTANA_SHARED_REPOSITORY_MIRROR_PASSWORD", Value: "INSTANA_SHARED_REPOSITORY_MIRROR_PASSWORD"},
				{Name: "INSTANA_AGENT_PROXY_HOST", Value: "INSTANA_AGENT_PROXY_HOST"},
				{Name: "INSTANA_AGENT_PROXY_PROTOCOL", Value: "INSTANA_AGENT_PROXY_PROTOCOL"},
				{Name: "INSTANA_AGENT_PROXY_USER", Value: "INSTANA_AGENT_PROXY_USER"},
				{Name: "INSTANA_AGENT_PROXY_PASSWORD", Value: "INSTANA_AGENT_PROXY_PASSWORD"},
				{Name: "INSTANA_AGENT_PROXY_USE_DNS", Value: "true"},
				{Name: "INSTANA_AGENT_HTTP_LISTEN", Value: "INSTANA_AGENT_HTTP_LISTEN"},
				{Name: "INSTANA_KUBERNETES_REDACT_SECRETS", Value: "INSTANA_KUBERNETES_REDACT_SECRETS"},
				{Name: "AGENT_ZONE", Value: "INSTANA_AGENT_SPEC_CLUSTER_NAME"},
				{Name: "HTTPS_PROXY", Value: "INSTANA_AGENT_PROXY_PROTOCOL://INSTANA_AGENT_PROXY_USER:INSTANA_AGENT_PROXY_PASSWORD@INSTANA_AGENT_PROXY_HOST:80"},
				{Name: "BACKEND_URL", Value: "https://$(BACKEND)"},
				{Name: "NO_PROXY", Value: "kubernetes.default.svc"},
				{Name: "CONFIG_PATH", Value: "/opt/instana/agent/etc/instana-config-yml"},
				{Name: "BACKEND", Value: ""},
				{Name: "INSTANA_AGENT_KEY", Value: ""},
				{Name: "AGENT_KEY", Value: ""},
				{Name: "INSTANA_DOWNLOAD_KEY", Value: ""},
				{Name: "INSTANA_AGENT_POD_NAME", Value: "metadata.name"},
				{Name: "POD_NAME", Value: "metadata.name"},
				{Name: "POD_IP", Value: "status.podIP"},
				{Name: "POD_UID", Value: "metadata.uid"},
				{Name: "POD_NAMESPACE", Value: "metadata.namespace"},
				{Name: "K8S_SERVICE_DOMAIN", Value: "-headless..svc"},
				{Name: "ENTRYPOINT_SKIP_BACKEND_TEMPLATE_GENERATION", Value: "true"},
			},
		},
		{
			name: "Should produce Instana Agent Zone mode as Instana Agent Mode when it exists",
			zone: &instanav1.Zone{
				Mode: "INSTANA_AGENT_ZONE_MODE",
				Name: instanav1.Name{Name: "INSTANA_AGENT_ZONE_NAME"},
			},
			agent: &instanav1.InstanaAgent{},
			envVars: []EnvVar{
				AgentModeEnv,
				ZoneNameEnv,
			},
			expected: []corev1.EnvVar{
				{Name: "INSTANA_AGENT_MODE", Value: "INSTANA_AGENT_ZONE_MODE"},
				{Name: "INSTANA_ZONE", Value: "INSTANA_AGENT_ZONE_NAME"},
			},
		},
		{
			name: "Should not include ProxyUseDNSEnv field (boolean type) that is set to false",
			zone: &instanav1.Zone{},
			agent: &instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					Agent: instanav1.BaseAgentSpec{
						ProxyUseDNS: false,
					},
				},
			},
			envVars: []EnvVar{
				ProxyUseDNSEnv,
			},
			expected: []corev1.EnvVar{},
		},
		{
			name: "Should allow http proxy without credentials, but different port",
			zone: &instanav1.Zone{},
			agent: &instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					UseSecretMounts: pointer.To(false),
					Agent: instanav1.BaseAgentSpec{
						ProxyHost: "INSTANA_AGENT_PROXY_HOST",
						ProxyPort: "8080",
					},
				},
			},
			envVars: []EnvVar{
				HTTPSProxyEnv,
			},
			expected: []corev1.EnvVar{
				{Name: "HTTPS_PROXY", Value: "http://INSTANA_AGENT_PROXY_HOST:8080"},
			},
		},
		{
			name: "Should allow http proxy with credentials, and different protocol",
			zone: &instanav1.Zone{},
			agent: &instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					UseSecretMounts: pointer.To(false),
					Agent: instanav1.BaseAgentSpec{
						ProxyHost:     "INSTANA_AGENT_PROXY_HOST",
						ProxyPort:     "443",
						ProxyUser:     "testuser",
						ProxyPassword: "testpassword",
						ProxyProtocol: "https",
					},
				},
			},
			envVars: []EnvVar{
				HTTPSProxyEnv,
			},
			expected: []corev1.EnvVar{
				{Name: "HTTPS_PROXY", Value: "https://testuser:testpassword@INSTANA_AGENT_PROXY_HOST:443"},
			},
		},
		{
			name: "Should skip secret environment variables when useSecretMounts is true",
			zone: &instanav1.Zone{},
			agent: &instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					UseSecretMounts: pointer.To(true),
					Agent: instanav1.BaseAgentSpec{
						ProxyUser:                 "INSTANA_AGENT_PROXY_USER",
						ProxyPassword:             "INSTANA_AGENT_PROXY_PASSWORD",
						MirrorReleaseRepoUsername: "AGENT_RELEASE_REPOSITORY_MIRROR_USERNAME",
						MirrorReleaseRepoPassword: "AGENT_RELEASE_REPOSITORY_MIRROR_PASSWORD",
						MirrorSharedRepoUsername:  "INSTANA_SHARED_REPOSITORY_MIRROR_USERNAME",
						MirrorSharedRepoPassword:  "INSTANA_SHARED_REPOSITORY_MIRROR_PASSWORD",
					},
				},
			},
			envVars: []EnvVar{
				ProxyUserEnv,
				ProxyPasswordEnv,
				MirrorReleaseRepoUsernameEnv,
				MirrorReleaseRepoPasswordEnv,
				MirrorSharedRepoUsernameEnv,
				MirrorSharedRepoPasswordEnv,
				InstanaAgentKeyEnv,
				AgentKeyEnv,
				DownloadKeyEnv,
			},
			expected: []corev1.EnvVar{
				// No environment variables should be included since all are secrets and useSecretMounts is true
			},
		},
		{
			name: "Should include secret environment variables when useSecretMounts is false",
			zone: &instanav1.Zone{},
			agent: &instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					UseSecretMounts: pointer.To(false),
					Agent: instanav1.BaseAgentSpec{
						ProxyUser:                 "INSTANA_AGENT_PROXY_USER",
						ProxyPassword:             "INSTANA_AGENT_PROXY_PASSWORD",
						MirrorReleaseRepoUsername: "AGENT_RELEASE_REPOSITORY_MIRROR_USERNAME",
						MirrorReleaseRepoPassword: "AGENT_RELEASE_REPOSITORY_MIRROR_PASSWORD",
					},
				},
			},
			envVars: []EnvVar{
				ProxyUserEnv,
				ProxyPasswordEnv,
				MirrorReleaseRepoUsernameEnv,
				MirrorReleaseRepoPasswordEnv,
			},
			expected: []corev1.EnvVar{
				{Name: "INSTANA_AGENT_PROXY_USER", Value: "INSTANA_AGENT_PROXY_USER"},
				{Name: "INSTANA_AGENT_PROXY_PASSWORD", Value: "INSTANA_AGENT_PROXY_PASSWORD"},
				{Name: "AGENT_RELEASE_REPOSITORY_MIRROR_USERNAME", Value: "AGENT_RELEASE_REPOSITORY_MIRROR_USERNAME"},
				{Name: "AGENT_RELEASE_REPOSITORY_MIRROR_PASSWORD", Value: "AGENT_RELEASE_REPOSITORY_MIRROR_PASSWORD"},
			},
		},
		{
			name: "Should skip secret environment variables when useSecretMounts is nil (default is true)",
			zone: &instanav1.Zone{},
			agent: &instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					UseSecretMounts: nil,
					Agent: instanav1.BaseAgentSpec{
						ProxyUser:                 "INSTANA_AGENT_PROXY_USER",
						ProxyPassword:             "INSTANA_AGENT_PROXY_PASSWORD",
						MirrorReleaseRepoUsername: "AGENT_RELEASE_REPOSITORY_MIRROR_USERNAME",
						MirrorReleaseRepoPassword: "AGENT_RELEASE_REPOSITORY_MIRROR_PASSWORD",
					},
				},
			},
			envVars: []EnvVar{
				ProxyUserEnv,
				ProxyPasswordEnv,
				MirrorReleaseRepoUsernameEnv,
				MirrorReleaseRepoPasswordEnv,
			},
			expected: []corev1.EnvVar{
				// No environment variables should be included since all are secrets and useSecretMounts default is true
			},
		},
		{
			name: "Should not set when CrdMonitoring is enabled",
			agent: &instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					K8sSensor: instanav1.K8sSpec{
						FeatureFlags: instanav1.K8sFeatureFlagsSpec{
							CrdMonitoring: pointer.To(true),
						},
					},
				},
			},
			envVars:  []EnvVar{CrdMonitoring},
			expected: []corev1.EnvVar{},
		},
		{
			name: "Should not set K8SENSOR_ENABLE_CRD_CR_MONITORING when CrdMonitoring is disabled",
			agent: &instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					K8sSensor: instanav1.K8sSpec{
						FeatureFlags: instanav1.K8sFeatureFlagsSpec{
							CrdMonitoring: pointer.To(false),
						},
					},
				},
			},
			envVars:  []EnvVar{CrdMonitoring},
			expected: []corev1.EnvVar{},
		},
		{
			name: "Should not set K8SENSOR_ENABLE_CRD_CR_MONITORING when CrdMonitoring is nil",
			agent: &instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					K8sSensor: instanav1.K8sSpec{
						FeatureFlags: instanav1.K8sFeatureFlagsSpec{
							CrdMonitoring: nil,
						},
					},
				},
			},
			envVars:  []EnvVar{CrdMonitoring},
			expected: []corev1.EnvVar{},
		},
		{
			name: "Should not set K8SENSOR_ENABLE_CRD_CR_MONITORING when K8sFeatureFlagsSpec is missing",
			agent: &instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					K8sSensor: instanav1.K8sSpec{},
				},
			},
			envVars:  []EnvVar{CrdMonitoring},
			expected: []corev1.EnvVar{},
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				builder := NewEnvBuilder(test.agent, test.zone)
				values := builder.Build(test.envVars...)

				var reducedEnvVars []corev1.EnvVar
				for _, value := range values {
					// Put the FieldPath as the value to verify FieldPath are correctly added if Value is not set
					if value.Value == "" && value.ValueFrom != nil && value.ValueFrom.FieldRef != nil {
						value.Value = value.ValueFrom.FieldRef.FieldPath
					}
					reducedEnvVars = append(reducedEnvVars, corev1.EnvVar{Name: value.Name, Value: value.Value})
				}

				assertions.ElementsMatch(test.expected, reducedEnvVars)
			},
		)
	}
}
