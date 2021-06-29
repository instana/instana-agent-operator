/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package coordination_api

import (
	coreV1 "k8s.io/api/core/v1"
)

const AgentPort = 42699

type PodCoordinationApi interface {
	Assign(pod coreV1.Pod, assignment []string) error
	PollPod(pod coreV1.Pod) (*CoordinationRecord, error)
}

type CoordinationRecord struct {
	Requested []string `json:"requested,omitempty"`
	Assigned  []string `json:"assigned,omitempty"`
}

func New() PodCoordinationApi {
	return &podCoordinationHttpClient{}
}
