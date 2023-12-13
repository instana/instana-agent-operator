/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package coordination_api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	coreV1 "k8s.io/api/core/v1"
)

type podCoordinationHttpClient struct {
}

func (c *podCoordinationHttpClient) Assign(ctx context.Context, pod coreV1.Pod, assignment []string) error {
	url := c.getBaseUrl(pod) + "/assigned"

	body, err := json.Marshal(assignment)
	if err != nil {
		return fmt.Errorf("error marshaling assignment list for %v: %w", pod.GetObjectMeta().GetName(), err)
	}

	request, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("invalid Http request for assigning leadership to %v: %w", pod.GetObjectMeta().GetName(), err)
	}
	request.Header.Add("content-type", "application/json")

	resp, err := http.DefaultClient.Do(request.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("unsuccessful request assigning leadership to %v: %w", pod.GetObjectMeta().GetName(), err)
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("leadership assignment request to %v resulted in HTTP error response %v",
			pod.GetObjectMeta().GetName(), resp.Status)
	}

	return nil
}

func (c *podCoordinationHttpClient) PollPod(ctx context.Context, pod coreV1.Pod) (*CoordinationRecord, error) {
	coordinationRecord := &CoordinationRecord{}
	url := c.getBaseUrl(pod)

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid Http request for querying leadership to %v: %w", pod.GetObjectMeta().GetName(), err)
	}

	resp, err := http.DefaultClient.Do(request.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("unsuccessful request polling %v: %w", pod.GetObjectMeta().GetName(), err)
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("request polling %v resulted in HTTP error response %v",
			pod.GetObjectMeta().GetName(), resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading /coordination response of %v: %w", pod.GetObjectMeta().GetName(), err)
	}

	if err := json.Unmarshal(body, coordinationRecord); err != nil {
		return nil, fmt.Errorf("error unmarshalling response of %v: %w", pod.GetObjectMeta().GetName(), err)
	}

	return coordinationRecord, nil
}

func (c *podCoordinationHttpClient) getBaseUrl(pod coreV1.Pod) string {
	return fmt.Sprintf("http://%v:%d/coordination", pod.Status.HostIP, AgentPort)
}
