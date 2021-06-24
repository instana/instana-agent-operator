/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"
	"log"
	"time"

	"github.com/go-logr/logr"
	"github.com/procyon-projects/chrono"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	LeaderElectionTask chrono.ScheduledTask
	agentPods          = make(map[string]coreV1.Pod)
)

type LeaderElector struct {
	Ctx    context.Context
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

type CoordinationRecord struct {
	Requested []string `json:"requested,omitempty"`
	Assigned  []string `json:"assigned,omitempty"`
}

func (l *LeaderElector) StartCoordination() error {
	l.Log = ctrl.Log.WithName("leaderelector").WithName("InstanaAgent")

	taskScheduler := chrono.NewDefaultTaskScheduler()
	LeaderElectionTask, err := taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
		if err := l.fetchPods(); err != nil {
			l.Log.Error(err, "Unable to fetch agent pods for doing election")
			return
		}
		l.pollAgentsAndAssignLeaders(agentPods)

		time.Sleep(5 * time.Second)
	}, 5*time.Second)

	log.Println(LeaderElectionTask)

	if err == nil {
		log.Print("Task has been scheduled successfully.")
		IsLeaderElecting = true
	}
	return err
}

func (l *LeaderElector) CancelLeaderElection() {
	LeaderElectionTask.Cancel()
	IsLeaderElecting = false
}

func (l *LeaderElector) fetchPods() error {
	podList := &coreV1.PodList{}
	lbs := map[string]string{
		"app.kubernetes.io/name": "instana-agent",
	}
	labelSelector := labels.SelectorFromSet(lbs)
	listOps := &client.ListOptions{Namespace: AgentNameSpace, LabelSelector: labelSelector}
	if err := l.Client.List(l.Ctx, podList, listOps); err != nil {
		return err
	}

	for _, pod := range podList.Items {
		agentPods[string(pod.GetObjectMeta().GetUID())] = pod
	}
	return nil
}

func (l *LeaderElector) pollAgentsAndAssignLeaders(pods map[string]coreV1.Pod) error {
	// activePods := agentPods
	// failedPods := []string{}
	l.pollLeadershipStatus(pods)
	return nil
}

func (l *LeaderElector) pollLeadershipStatus(pods map[string]coreV1.Pod) error {
	resourcesByPod := make(map[string]CoordinationRecord)
	coordinationApi := PodCoordinationHttpClient{}
	for uid, pod := range pods {
		if pod.Status.Phase == coreV1.PodRunning {
			coordinationRecord, err := coordinationApi.pollPod(pod)
			if err != nil {
				return err
			}
			resourcesByPod[uid] = *coordinationRecord
			log.Println(coordinationRecord)
		}
	}
	return nil
}

type LeadershipStatus struct {
	Status map[string]CoordinationRecord
}

func (s *LeadershipStatus) getResourceRequests() map[string][]string {
	return nil
}
