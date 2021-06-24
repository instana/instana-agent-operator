/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	coreV1 "k8s.io/api/core/v1"
)

type PodCoordinationApi interface {
	assign(pod coreV1.Pod, assignment []string) error
	pollPod(pod coreV1.Pod) (CoordinationRecord, error)
}

type PodCoordinationHttpClient struct {
}

func (c *PodCoordinationHttpClient) pollPod(pod coreV1.Pod) (*CoordinationRecord, error) {
	coordinationRecord := &CoordinationRecord{}
	url := c.getBaseUrl(pod)
	resp, err := http.Get(url)
	if err != nil {
		return &CoordinationRecord{}, errors.New("Unsuccessful request polling " + pod.GetObjectMeta().GetName() + ": " + err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return &CoordinationRecord{}, errors.New("Unsuccessful request polling " + pod.GetObjectMeta().GetName() + ": " + fmt.Sprint(resp.StatusCode))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &CoordinationRecord{}, errors.New("Error reading response of " + pod.GetObjectMeta().GetName() + ": " + err.Error())
	}

	if err := json.Unmarshal(body, coordinationRecord); err != nil {
		return &CoordinationRecord{}, errors.New("Error Unmarshaling response of " + pod.GetObjectMeta().GetName() + ": " + err.Error())
	}

	return coordinationRecord, nil
}

func (c *PodCoordinationHttpClient) getBaseUrl(pod coreV1.Pod) string {
	ip := pod.Status.HostIP
	return "http://" + ip + ":" + fmt.Sprint(AgentPort) + "/coordination"
}
