/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package leaderelection

import (
	"context"
	"log"
	"time"

	"github.com/go-logr/logr"
	"github.com/instana/instana-agent-operator/controllers/leaderelection/coordination_api"
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
	IsLeaderElecting   = false

	coordinationApi = coordination_api.New()
)

type LeaderElector struct {
	Ctx    context.Context
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

func (l *LeaderElector) StartCoordination(agentNameSpace string) error {
	l.Log = ctrl.Log.WithName("leaderelector").WithName("InstanaAgent")

	taskScheduler := chrono.NewDefaultTaskScheduler()
	_, err := taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
		if err := l.fetchPods(agentNameSpace); err != nil {
			l.Log.Error(err, "Unable to fetch agent pods for doing election")
			return
		}
		l.pollAgentsAndAssignLeaders(agentPods)

		time.Sleep(5 * time.Second)
	}, 5*time.Second)

	if err == nil {
		l.Log.Info("Task has been scheduled successfully.")
		IsLeaderElecting = true
	}
	return err
}

func (l *LeaderElector) CancelLeaderElection() {
	LeaderElectionTask.Cancel()
	IsLeaderElecting = false
}

func (l *LeaderElector) fetchPods(agentNameSpace string) error {
	podList := &coreV1.PodList{}
	lbs := map[string]string{
		"app.kubernetes.io/name": "instana-agent",
	}
	labelSelector := labels.SelectorFromSet(lbs)
	listOps := &client.ListOptions{Namespace: agentNameSpace, LabelSelector: labelSelector}
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
	leadershipStatus, err := l.pollLeadershipStatus(pods)
	if err != nil {
		return err
	}
	log.Println(leadershipStatus)

	return nil
}

func (l *LeaderElector) pollLeadershipStatus(pods map[string]coreV1.Pod) (*LeadershipStatus, error) {
	resourcesByPod := make(map[string]coordination_api.CoordinationRecord)
	for uid, pod := range pods {
		if pod.Status.Phase == coreV1.PodRunning {
			coordinationRecord, err := coordinationApi.PollPod(pod)
			if err != nil {
				return nil, err
			}
			resourcesByPod[uid] = *coordinationRecord
			log.Println(coordinationRecord)
		}
	}
	return &LeadershipStatus{Status: resourcesByPod}, nil
}

type LeadershipStatus struct {
	Status map[string]coordination_api.CoordinationRecord
}

// func (s *LeadershipStatus) getResourceRequests() map[string][]string {
// 	resourceRequests := make(map[string][]string)

// 	for
// 	return nil
// }
