/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package leaderelection

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"

	instanaV1 "github.com/instana/instana-agent-operator/api/v1"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/google/go-cmp/cmp"

	"github.com/go-logr/logr"
	"github.com/procyon-projects/chrono"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/instana/instana-agent-operator/controllers/leaderelection/coordination_api"
)

func NewLeaderElection(client client.Client, namespacedName types.NamespacedName) *LeaderElector {
	return &LeaderElector{
		client:                      client,
		log:                         logf.Log.WithName("agent.leaderelector"),
		coordinationApi:             coordination_api.New(),
		leaderElectionTaskScheduler: chrono.NewDefaultTaskScheduler(),
		namespacedName:              namespacedName,
	}
}

type LeaderElector struct {
	client                      client.Client
	log                         logr.Logger
	coordinationApi             coordination_api.PodCoordinationApi
	leaderElectionTaskScheduler chrono.TaskScheduler
	namespacedName              types.NamespacedName
	// Uninitialized fields from NewLeaderElection
	leaderElectionTask chrono.ScheduledTask
}

/*
StartCoordination starts scheduling of leader elector coordination between Instana Agents.

Agent Coordination works by requesting the "resources" for which an Agent would be leader. This could be one or more.
Different Agents might request similar or different "resources".

The Operator should, for every requested resource, appoint one specific Agent pod as leader by calling the
/coordination/assigned endpoint.
*/
func (l *LeaderElector) StartCoordination(agentNameSpace string) error {
	if l.IsLeaderElectionScheduled() {
		return errors.New("leader election coordination task has already been scheduled")
	}

	if task, err := l.leaderElectionTaskScheduler.ScheduleWithFixedDelay(
		func(ctx context.Context) {
			fetchPodsCtx, cancelFunc := context.WithTimeout(ctx, 5*time.Second)
			defer cancelFunc()

			activePods, err := l.fetchPods(fetchPodsCtx, agentNameSpace)
			if err != nil {
				l.log.Error(err, "Unable to fetch agent pods for doing election")
				return
			}

			assignCtx, cancelFunc := context.WithTimeout(ctx, 20*time.Second)
			defer cancelFunc()
			leaders := l.pollAgentsAndAssignLeaders(assignCtx, activePods)

			if leaders != nil {
				l.updateLeaderStatusInCustomResource(leaders, activePods)
			}

		}, 10*time.Second,
	); err != nil {
		return fmt.Errorf("failure scheduling leader elector coordination task: %w", err)
	} else {
		l.leaderElectionTask = task
		l.log.Info("Leader Election task has been scheduled successfully.")
	}

	return nil
}

func (l *LeaderElector) IsLeaderElectionScheduled() bool {
	return l.leaderElectionTask != nil && !l.leaderElectionTask.IsCancelled()
}

func (l *LeaderElector) CancelLeaderElection() {
	if l.leaderElectionTask != nil && !l.leaderElectionTask.IsCancelled() {
		l.leaderElectionTask.Cancel()
	}
	if !l.leaderElectionTaskScheduler.IsShutdown() {
		l.leaderElectionTaskScheduler.Shutdown()
	}
}

func (l *LeaderElector) fetchPods(ctx context.Context, agentNameSpace string) (map[string]coreV1.Pod, error) {
	podList := &coreV1.PodList{}
	activePods := make(map[string]coreV1.Pod)
	lbs := map[string]string{
		"app.kubernetes.io/name": l.namespacedName.Namespace,
	}
	labelSelector := labels.SelectorFromSet(lbs)
	listOps := &client.ListOptions{Namespace: agentNameSpace, LabelSelector: labelSelector}
	if err := l.client.List(ctx, podList, listOps); err != nil {
		return nil, err
	}

	for _, pod := range podList.Items {
		if pod.Status.Phase == coreV1.PodRunning {
			activePods[string(pod.GetObjectMeta().GetUID())] = pod
		}
	}
	return activePods, nil
}

// pollAgentsAndAssignLeaders will first get all "requested resources" from every Agent Pod. It will then calculate new
// assignments, prioritizing any Pod that already holds that assignment.
// The function is executed in a loop, so that should assignments fail for any Pod, determining assignments starts over from scratch.
func (l *LeaderElector) pollAgentsAndAssignLeaders(
	ctx context.Context,
	pods map[string]coreV1.Pod,
) map[string][]string {
outer:
	for {
		// Safeguard to prevent infinite loop should we fail to assign all pods
		if len(pods) == 0 {
			return nil
		}

		leadershipStatus := l.pollLeadershipStatus(ctx, pods)
		// if leadershipStatus.Status is empty list that means we were not able to get leadership status for any pod
		// so we can't proceed with leadership assigning process and we should return
		if len(leadershipStatus.Status) == 0 {
			return nil
		}

		desiredPodWithAssignments := l.calculateDesiredAssignments(leadershipStatus)

		// As there can be multiple pods with different resource assignments, have to try Pod assignment one by one.
		// If one fails, the pod needs to be removed and assignments re-calculated
		for pod, assignments := range desiredPodWithAssignments {
			if result := l.assign(ctx, pods, leadershipStatus, pod, assignments); !result {
				// Failure for assigning to one of the pods, need to re-evaluate and re-calculate coordination
				delete(pods, pod)
				continue outer
			}
		}

		// Because of a bug in the Agent code, it could happen we have Pods with _no_ requests but _with_ assignments, clean
		// these up although we're not interested failures as the Pod might get restarted
		for _, pod := range leadershipStatus.getPodsWithAssignmentsNoRequests() {
			c := pods[pod]
			l.log.Info(
				fmt.Sprintf(
					"Pod with UID %v has assignments but no requests. Resetting.",
					c.GetObjectMeta().GetName(),
				),
			)
			l.assign(ctx, pods, leadershipStatus, pod, []string{})
		}

		// All assignments finished correctly, so exit
		return desiredPodWithAssignments
	}
}

func (l *LeaderElector) assign(
	ctx context.Context,
	activePods map[string]coreV1.Pod,
	leadershipStatus *LeadershipStatus,
	desiredPod string,
	assignments []string,
) bool {
	less := func(a, b string) bool { return a < b }
	if !cmp.Equal(assignments, leadershipStatus.getAssignmentsForPod(desiredPod), cmpopts.SortSlices(less)) {
		// Only need to update if desired assignments are not yet equal to actual assignments
		pod := activePods[desiredPod]
		if err := l.coordinationApi.Assign(ctx, pod, assignments); err != nil {
			l.log.Error(
				err,
				fmt.Sprintf("Failed to assign leadership %v to pod: %v", assignments, pod.GetObjectMeta().GetName()),
			)
			return false
		}
		l.log.Info(fmt.Sprintf("Assigned leadership of %v to pod: %v", assignments, pod.GetObjectMeta().GetName()))
	}
	return true
}

func (l *LeaderElector) pollLeadershipStatus(ctx context.Context, pods map[string]coreV1.Pod) *LeadershipStatus {
	resourcesMutex := sync.Mutex{}
	resourcesByPod := make(map[string]*coordination_api.CoordinationRecord)

	group := sync.WaitGroup{}

	for uid, pod := range pods {

		group.Add(1)
		go func(uid string, pod coreV1.Pod) {
			defer group.Done()

			coordinationRecord, err := l.coordinationApi.PollPod(ctx, pod)
			if err != nil {
				// Logging on Info level because could just happen that Pod is not ready
				l.log.Info(
					fmt.Sprintf(
						"Unable to poll coordination status for Pod %v: %v",
						pod.GetObjectMeta().GetName(),
						err,
					),
				)
			} else {
				l.log.V(1).Info(
					fmt.Sprintf(
						"Coordination status was successfully polled for Pod %v",
						pod.GetObjectMeta().GetName(),
					),
				)
				resourcesMutex.Lock()
				resourcesByPod[uid] = coordinationRecord
				resourcesMutex.Unlock()
			}
		}(uid, pod)
	}

	group.Wait()
	return &LeadershipStatus{Status: resourcesByPod}
}

func (l *LeaderElector) calculateDesiredAssignments(leadershipStatus *LeadershipStatus) map[string][]string {
	desiredPodWithAssignments := make(map[string][]string)

	// For every requested resource, determine if there is a 'leader' already, if not, pick a random Pod from the eligible ones
	for resource, podsList := range leadershipStatus.getRequestedResourcesWithPods() {
		var desiredPod string
		var ok bool

		if desiredPod, ok = leadershipStatus.getCurrentLeaderPodForResource(resource); !ok {
			// We can be certain len(podsList) > 0 and so invocation rand.IntnRange(1, ) is safe
			desiredPod = podsList[rand.IntnRange(1, len(podsList))]
		}

		if assignments, contains := desiredPodWithAssignments[desiredPod]; contains {
			desiredPodWithAssignments[desiredPod] = append(assignments, resource)
		} else {
			desiredPodWithAssignments[desiredPod] = []string{resource}
		}
	}

	return desiredPodWithAssignments
}

func (l *LeaderElector) updateLeaderStatusInCustomResource(leaders map[string][]string, pods map[string]coreV1.Pod) {
	// Don't do any error handling. Updating the Status field is not critical ATM and will be retried each cycle

	// Convert from map pods -> statuses to status -> ResourceInfo(pod)
	leadershipStatus := make(map[string]instanaV1.ResourceInfo, len(leaders))
	for podUID, leaderTypes := range leaders {
		for _, leaderType := range leaderTypes {

			var podName string
			if pod, ok := pods[podUID]; ok {
				podName = pod.Name
			} else {
				podName = "<unknown>"
			}

			leadershipStatus[leaderType] = instanaV1.ResourceInfo{
				Name: podName,
				UID:  podUID,
			}
		}
	}

	crdInstance := &instanaV1.InstanaAgent{}
	if err := l.client.Get(context.Background(), l.namespacedName, crdInstance); err != nil {
		l.log.Error(err, "Failure querying InstanaAgent CR to update LeaderShip Status field")
		return
	}

	// Only update the CR if anything actually changed
	less := func(a, b string) bool { return a < b }
	if crdInstance.Status.LeadingAgentPod == nil || !cmp.Equal(
		leadershipStatus,
		crdInstance.Status.LeadingAgentPod,
		cmpopts.SortSlices(less),
	) {

		crdInstance.Status.LeadingAgentPod = leadershipStatus
		if err := l.client.Status().Update(context.Background(), crdInstance); err != nil {
			l.log.Error(err, "Failed updating CR with LeaderShip Status")
		}
	}
}

type LeadershipStatus struct {
	Status map[string]*coordination_api.CoordinationRecord
}

func (s *LeadershipStatus) getRequestedResourcesWithPods() map[string][]string {
	requestedResourcesByPods := make(map[string][]string)

	// From a Map with podUid -> [] requested resources, transform to a map of 'requested resource' -> [] podUids
	for podUid, coordinationRecord := range s.Status {
		if len(coordinationRecord.Requested) > 0 {
			for _, resource := range coordinationRecord.Requested {
				if elem, contains := requestedResourcesByPods[resource]; contains {
					requestedResourcesByPods[resource] = append(elem, podUid)
				} else {
					requestedResourcesByPods[resource] = []string{podUid}
				}
			}
		}
	}

	return requestedResourcesByPods
}

func (s *LeadershipStatus) getCurrentLeaderPodForResource(resource string) (string, bool) {
	for podUid, coordinationRecord := range s.Status {
		if len(coordinationRecord.Assigned) > 0 {
			for _, assignedResource := range coordinationRecord.Assigned {
				if resource == assignedResource {
					return podUid, true
				}
			}
		}
	}
	return "", false
}

func (s *LeadershipStatus) getAssignmentsForPod(podUid string) []string {
	if coordinationRecord, ok := s.Status[podUid]; ok {
		return coordinationRecord.Assigned
	} else {
		return nil
	}
}

func (s *LeadershipStatus) getPodsWithAssignmentsNoRequests() []string {
	podsWithoutRequests := make([]string, 0, len(s.Status))

	// From a Map with podUid -> [] requested resources, transform to a map of 'requested resource' -> [] podUids
	for podUid, coordinationRecord := range s.Status {
		if len(coordinationRecord.Assigned) > 0 && len(coordinationRecord.Requested) == 0 {
			podsWithoutRequests = append(podsWithoutRequests, podUid)
		}
	}

	return podsWithoutRequests
}
