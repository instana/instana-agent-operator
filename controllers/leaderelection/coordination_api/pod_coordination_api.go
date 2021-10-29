/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package coordination_api

import (
	"context"

	coreV1 "k8s.io/api/core/v1"
)

const AgentPort = 42699

type PodCoordinationApi interface {
	Assign(ctx context.Context, pod coreV1.Pod, assignment []string) error
	PollPod(ctx context.Context, pod coreV1.Pod) (*CoordinationRecord, error)
}

type CoordinationRecord struct {
	Requested []string `json:"requested,omitempty"`
	Assigned  []string `json:"assigned,omitempty"`
}

func New() PodCoordinationApi {
	return &podCoordinationHttpClient{}
}
