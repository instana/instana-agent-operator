/*
 * (c) Copyright IBM Corp. 2024
 * (c) Copyright Instana Inc. 2024
 */

package e2e

import (
	"os"

	"github.com/joho/godotenv"
	log "k8s.io/klog/v2"
	"sigs.k8s.io/e2e-framework/pkg/utils"
)

type InstanaTestConfig struct {
	ContainerRegistry *ContainerRegistry
	InstanaBackend    *InstanaBackend
	OperatorImage     *OperatorImage
}

type ContainerRegistry struct {
	Name     string
	User     string
	Host     string
	Password string
}

type InstanaBackend struct {
	EndpointHost string
	EndpointPort int
	AgentKey     string
}

type OperatorImage struct {
	Name string
	Tag  string
}

var InstanaTestCfg InstanaTestConfig

const InstanaNamespace string = "instana-agent"
const InstanaOperatorOldDeploymentName string = "controller-manager"
const InstanaOperatorOldClusterRoleName string = "manager-role"
const InstanaOperatorOldClusterRoleBindingName string = "manager-rolebinding"
const InstanaOperatorDeploymentName string = "instana-agent-controller-manager"
const AgentDaemonSetName string = "instana-agent"
const AgentCustomResourceName string = "instana-agent"
const K8sensorDeploymentName string = "instana-agent-k8sensor"
const InstanaAgentConfigSecretName string = "instana-agent-config"
const InstanaAgentStaticImage string = "containers.instana.io/instana/release/agent/static"

func init() {
	var instanaApiKey, containerRegistryUser, containerRegistryPassword, containerRegistryHost, endpointHost, operatorImageName, operatorImageTag string
	var found, fatal bool

	err := godotenv.Load(".env")
	if err != nil {
		log.Warningln(
			"Warning: an error occurred while attempting to load dotenv-file expected in location: BASE_DIR/e2e/.env",
			err,
		)
	}

	instanaApiKey, found = os.LookupEnv("INSTANA_API_KEY")
	if !found {
		log.Errorln("Required: $INSTANA_API_KEY not defined")
		fatal = true
	}

	containerRegistryUser, found = os.LookupEnv("ICR_USERNAME")
	if !found {
		log.Errorln("Required: $ICR_USERNAME not defined")
		fatal = true
	}

	containerRegistryPassword, found = os.LookupEnv("ICR_PASSWORD")
	if !found {
		log.Errorln("Required: $ICR_PASSWORD not defined")
		fatal = true
	}

	containerRegistryHost, found = os.LookupEnv("ICR_HOST")
	if !found {
		log.Warningln("Optional: $ICR_HOST not defined, using default ICR host icr.io")
		containerRegistryHost = "icr.io"
	}
	endpointHost, found = os.LookupEnv("INSTANA_ENDPOINT_HOST")
	if !found {
		log.Warningln("Optional: $INSTANA_ENDPOINT_HOST not defined, using default")
		endpointHost = "ingress-red-saas.instana.io"
	}
	operatorImageName, found = os.LookupEnv("OPERATOR_IMAGE_NAME")
	if !found {
		log.Warningln("Optional: $OPERATOR_IMAGE_NAME not defined, using default")
		operatorImageName = "icr.io/instana-agent-dev/instana-agent-operator"
	}

	operatorImageTag, found = os.LookupEnv("OPERATOR_IMAGE_TAG")
	if !found {
		log.Warningln("Optional: $OPERATOR_IMAGE_TAG not defined, falling back to $GIT_COMMIT")
		operatorImageTag, found = os.LookupEnv("GIT_COMMIT")
		if !found {
			log.Warningln(
				"Optional: $GIT_COMMIT is not defined, falling back to git cli to resolve last commit",
			)
			p := utils.RunCommand("git rev-parse HEAD")
			if p.Err() != nil {
				log.Warningf(
					"Error while getting git commit via cli: %v, %v, %v, %v\n",
					p.Command(),
					p.Err(),
					p.Out(),
					p.ExitCode(),
				)
				log.Fatalln(
					"Required: Either $OPERATOR_IMAGE_TAG or $GIT_COMMIT must be set to be able to deploy a custom operator build",
				)
				fatal = true
			}
			// using short commit as tag (default)
			operatorImageTag = p.Result()[0:7]
		}
	}

	if fatal {
		log.Fatalln(
			"Fatal: Required configuration is missing, tests woud not work without those settings, terminating execution",
		)
	}

	InstanaTestCfg = InstanaTestConfig{
		ContainerRegistry: &ContainerRegistry{
			Name:     "icr-io-pull-secret",
			User:     containerRegistryUser,
			Password: containerRegistryPassword,
			Host:     containerRegistryHost,
		},
		InstanaBackend: &InstanaBackend{
			EndpointHost: endpointHost,
			EndpointPort: 443,
			AgentKey:     instanaApiKey,
		},
		OperatorImage: &OperatorImage{
			Name: operatorImageName,
			Tag:  operatorImageTag,
		},
	}
}
