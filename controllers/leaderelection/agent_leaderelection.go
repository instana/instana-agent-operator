/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package leaderelection

import (
	"context"
	"log"
	"math/rand"
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
	LeaderElectionTask         chrono.ScheduledTask
	IsLeaderElecting           = false
	KubernetesLeaderResourceId = "com.instana.plugin.kubernetes.leader"

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
		var activePods map[string]coreV1.Pod
		var err error
		if activePods, err = l.fetchPods(agentNameSpace); err != nil {
			l.Log.Error(err, "Unable to fetch agent pods for doing election")
			return
		}
		l.pollAgentsAndAssignLeaders(activePods)

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

func (l *LeaderElector) fetchPods(agentNameSpace string) (map[string]coreV1.Pod, error) {
	podList := &coreV1.PodList{}
	activePods := make(map[string]coreV1.Pod)
	lbs := map[string]string{
		"app.kubernetes.io/name": "instana-agent",
	}
	labelSelector := labels.SelectorFromSet(lbs)
	listOps := &client.ListOptions{Namespace: agentNameSpace, LabelSelector: labelSelector}
	if err := l.Client.List(l.Ctx, podList, listOps); err != nil {
		return nil, err
	}

	for _, pod := range podList.Items {
		if pod.Status.Phase == coreV1.PodRunning {
			activePods[string(pod.GetObjectMeta().GetUID())] = pod
		}
	}
	return activePods, nil
}

func (l *LeaderElector) pollAgentsAndAssignLeaders(pods map[string]coreV1.Pod) error {
	leadershipStatus, err := l.pollLeadershipStatus(pods)
	if err != nil {
		return err
	}
	if len(leadershipStatus.Status) > 0 {
		desiredPod := l.calculateAssignedPod(leadershipStatus)
		log.Println(desiredPod)
	}

	return nil
}

func (l *LeaderElector) assign(activePods map[string]coreV1.Pod, leadershipStatus LeadershipStatus, desiredPod string) {
}

func (l *LeaderElector) pollLeadershipStatus(pods map[string]coreV1.Pod) (*LeadershipStatus, error) {
	resourcesByPod := make(map[string]Coordination)
	for uid, pod := range pods {
		coordinationRecord, err := coordinationApi.PollPod(pod)
		if err != nil {
			return nil, err
		}
		var requested, assigned string
		if coordinationRecord.Requested != nil && len(coordinationRecord.Requested) > 0 {
			requested = coordinationRecord.Requested[0]
		}
		if coordinationRecord.Assigned != nil && len(coordinationRecord.Assigned) > 0 {
			assigned = coordinationRecord.Assigned[0]
		}
		resourcesByPod[uid] = Coordination{
			Requested: requested,
			Assigned:  assigned,
		}
	}
	return &LeadershipStatus{Status: resourcesByPod}, nil
}

func (l *LeaderElector) calculateAssignedPod(leadershipStatus *LeadershipStatus) string {
	requestedPods := leadershipStatus.getRequestedPods()
	desired := leadershipStatus.getCurrentLeaderPod()
	if len(desired) == 0 {
		desired = l.selectRandomPod(requestedPods)
	}
	return desired
}

func (l *LeaderElector) selectRandomPod(requestedPods []string) string {
	return requestedPods[rand.Intn(len(requestedPods))]
}

type LeadershipStatus struct {
	Status map[string]Coordination
}

func (s *LeadershipStatus) getRequestedPods() []string {
	var requestedPods []string

	for podUid, coordination := range s.Status {
		if len(coordination.Requested) > 0 && coordination.Requested == KubernetesLeaderResourceId {
			requestedPods = append(requestedPods, podUid)
		}
	}
	return requestedPods
}

func (s *LeadershipStatus) getCurrentLeaderPod() string {
	for podUid, podStatus := range s.Status {
		if podStatus.Assigned == KubernetesLeaderResourceId {
			return podUid
		}
	}
	return ""
}

type Coordination struct {
	Requested string
	Assigned  string
}
