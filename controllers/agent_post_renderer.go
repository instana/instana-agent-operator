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
			objMap, err = p.removeLeaderElectorContainer(objMap)
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

func (p *AgentPostRenderer) removeLeaderElectorContainer(objMap map[string]interface{}) (map[string]interface{}, error) {
	var ds = &appV1.DaemonSet{}
	runtime.DefaultUnstructuredConverter.FromUnstructured(objMap, ds)

	containerList := ds.Spec.Template.Spec.Containers
	for i, container := range containerList {
		if container.Name == "leader-elector" {
			containerList = append(containerList[:i], containerList[i+1:]...)
			p.Log.Info("Leader-elector sidecar container is now removed from daemonset")
			break
		}
	}
	ds.Spec.Template.Spec.Containers = containerList

	return runtime.DefaultUnstructuredConverter.ToUnstructured(ds)
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
