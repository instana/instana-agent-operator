/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package helm

import (
	"bytes"

	"github.com/go-logr/logr"
	instanaV1Beta1 "github.com/instana/instana-agent-operator/api/v1beta1"
	"helm.sh/helm/v3/pkg/action"

	appV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/resource"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

func NewAgentChartPostRenderer(h *HelmReconciliation, crdInstance *instanaV1Beta1.InstanaAgent) *AgentChartPostRenderer {
	return &AgentChartPostRenderer{
		scheme:      h.scheme,
		helmCfg:     h.helmCfg,
		crdInstance: crdInstance,
		log:         h.log.WithName("postrenderer"),
	}
}

type AgentChartPostRenderer struct {
	scheme      *runtime.Scheme
	crdInstance *instanaV1Beta1.InstanaAgent
	helmCfg     *action.Configuration
	log         logr.Logger
}

func (p *AgentChartPostRenderer) Run(in *bytes.Buffer) (*bytes.Buffer, error) {
	resourceList, err := p.helmCfg.KubeClient.Build(in, false)
	if err != nil {
		return nil, err
	}

	out := bytes.Buffer{}
	if err := resourceList.Visit(func(r *resource.Info, incomingErr error) error {
		// For any incoming (parsing) errors, return with error to stop processing
		if incomingErr != nil {
			return incomingErr
		}

		// Make sure we have Unstructured content (wrapped key-value map) to work with for the necessary modifications
		var modifiedResource *unstructured.Unstructured
		if objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(r.Object); err == nil {
			modifiedResource = &unstructured.Unstructured{Object: objMap}
		} else {
			return err
		}

		if r.ObjectName() == "daemonsets/instana-agent" {
			var err error // Shadow outer "err" because cannot use re-declaration
			if modifiedResource, err = p.adjustDaemonsetRemoveLeaderElection(modifiedResource); err != nil {
				return err
			}
			p.log.V(1).Info("Removing leader-elector sidecar from DaemonSet was successful")
		}

		if !(r.ObjectName() == "clusterroles/instana-agent" || r.ObjectName() == "clusterrolebindings/instana-agent") {
			if err := controllerutil.SetControllerReference(p.crdInstance, modifiedResource, p.scheme); err != nil {
				return err
			}
			p.log.V(1).Info("Setting controller reference for Object was successful", "ObjectName", r.ObjectName())
		}

		if err := p.writeToOutBuffer(modifiedResource, &out); err != nil {
			return err
		}
		return nil

	}); err != nil {
		return nil, err
	}

	return &out, nil
}

func (p *AgentChartPostRenderer) adjustDaemonsetRemoveLeaderElection(unstr *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	// Convert to a structured Go object which is easier to modify in this case
	var ds = &appV1.DaemonSet{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstr.Object, ds); err != nil {
		return nil, err
	}

	containerList := ds.Spec.Template.Spec.Containers
	containerList = p.removeLeaderElectorContainer(containerList)
	for i := range containerList {
		ds.Spec.Template.Spec.Containers[i].Env = p.replaceLeaderElectorEnvVar(ds.Spec.Template.Spec.Containers[i].Env)
	}
	ds.Spec.Template.Spec.Containers = containerList

	if objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(ds); err == nil {
		return &unstructured.Unstructured{Object: objMap}, nil
	} else {
		return nil, err
	}
}

func (p *AgentChartPostRenderer) replaceLeaderElectorEnvVar(envList []v1.EnvVar) []v1.EnvVar {
	for i, envVar := range envList {
		if envVar.Name == "INSTANA_AGENT_LEADER_ELECTOR_PORT" {
			envList = append(envList[:i], envList[i+1:]...)
		}
	}
	envList = append(envList, v1.EnvVar{Name: "INSTANA_OPERATOR_MANAGED", Value: "true"})
	return envList
}

func (p *AgentChartPostRenderer) removeLeaderElectorContainer(containerList []v1.Container) []v1.Container {
	for i, container := range containerList {
		if container.Name == "leader-elector" {
			p.log.V(1).Info("Found leader-elector sidecar container", "container", container)
			// Assume only a single 'leader elector' container so return immediately with the updated slice
			return append(containerList[:i], containerList[i+1:]...)
		}
	}
	return containerList
}

func (p *AgentChartPostRenderer) writeToOutBuffer(modifiedResource *unstructured.Unstructured, out *bytes.Buffer) error {
	outData, err := yaml.Marshal(modifiedResource)
	if err != nil {
		return err
	}
	if _, err := out.WriteString("---\n" + string(outData)); err != nil {
		return err
	}
	return nil
}
