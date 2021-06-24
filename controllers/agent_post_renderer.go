/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"bytes"
	"fmt"

	"github.com/go-logr/logr"
	instanaV1Beta1 "github.com/instana/instana-agent-operator/api/v1beta1"

	appV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/resource"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

type AgentPostRenderer struct {
	client.Client
	Scheme      *runtime.Scheme
	Log         logr.Logger
	CrdInstance *instanaV1Beta1.InstanaAgent
}

func (p *AgentPostRenderer) Run(in *bytes.Buffer) (*bytes.Buffer, error) {
	p.Log = ctrl.Log.WithName("postrenderer").WithName("InstanaAgent")
	resourceList, err := HelmCfg.KubeClient.Build(in, false)
	if err != nil {
		return nil, err
	}
	out := bytes.Buffer{}
	err = resourceList.Visit(func(r *resource.Info, err error) error {

		if err != nil {
			return err
		}

		objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(r.Object)
		if err != nil {
			return err
		}

		if r.ObjectName() == "daemonsets/instana-agent" {
			objMap, err = p.adjustDaemonsetForLeaderElection(objMap)
			if err != nil {
				return err
			}
		}
		u := &unstructured.Unstructured{Object: objMap}
		if !(r.ObjectName() == "clusterroles/instana-agent" || r.ObjectName() == "clusterrolebindings/instana-agent") {
			if err = controllerutil.SetControllerReference(p.CrdInstance, u, p.Scheme); err != nil {
				return err
			}
			p.Log.Info(fmt.Sprintf("Set controller reference for %s was successful", r.ObjectName()))
		}
		if err = p.writeToOutBuffer(u.Object, &out); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (p *AgentPostRenderer) adjustDaemonsetForLeaderElection(objMap map[string]interface{}) (map[string]interface{}, error) {
	var ds = &appV1.DaemonSet{}
	runtime.DefaultUnstructuredConverter.FromUnstructured(objMap, ds)

	containerList := ds.Spec.Template.Spec.Containers
	ds.Spec.Template.Spec.Containers = p.removeLeaderElectorContainer(containerList)
	envList := ds.Spec.Template.Spec.Containers[0].Env
	ds.Spec.Template.Spec.Containers[0].Env = p.replaceLeaderElectorEnvVar(envList)
	return runtime.DefaultUnstructuredConverter.ToUnstructured(ds)
}

func (p *AgentPostRenderer) replaceLeaderElectorEnvVar(envList []v1.EnvVar) []v1.EnvVar {
	envList = append(envList, v1.EnvVar{Name: "INSTANA_OPERATOR_MANAGED", Value: "true"})
	for i, envVar := range envList {
		if envVar.Name == "INSTANA_AGENT_LEADER_ELECTOR_PORT" {
			envList = append(envList[:i], envList[i+1:]...)
		}
	}
	return envList
}
func (p *AgentPostRenderer) removeLeaderElectorContainer(containerList []v1.Container) []v1.Container {
	for i, container := range containerList {
		if container.Name == "leader-elector" {
			containerList = append(containerList[:i], containerList[i+1:]...)
			p.Log.Info("Leader-elector sidecar container is now removed from daemonset")
			break
		}
	}
	return containerList
}
func (p *AgentPostRenderer) writeToOutBuffer(modifiedResource interface{}, out *bytes.Buffer) error {
	outData, err := yaml.Marshal(modifiedResource)
	if err != nil {
		return err
	}
	if _, err := out.WriteString("---\n" + string(outData)); err != nil {
		return err
	}
	return nil
}
