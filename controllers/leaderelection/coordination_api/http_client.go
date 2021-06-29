/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package coordination_api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	coreV1 "k8s.io/api/core/v1"
)

type podCoordinationHttpClient struct {
}

func (c *podCoordinationHttpClient) Assign(pod coreV1.Pod, assignment []string) error {
	url := c.getBaseUrl(pod) + "/assigned"
	body, err := json.Marshal(assignment)
	if err != nil {
		return err
	}

	request, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Add("content-type", "application/json")
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	log.Println("assing response :")
	log.Println(resp)

	return nil
}
func (c *podCoordinationHttpClient) PollPod(pod coreV1.Pod) (*CoordinationRecord, error) {
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

func (c *podCoordinationHttpClient) getBaseUrl(pod coreV1.Pod) string {
	ip := pod.Status.HostIP
	return "http://" + ip + ":" + fmt.Sprint(AgentPort) + "/coordination"
}
