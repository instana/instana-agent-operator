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
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/google/go-cmp/cmp"

	"github.com/go-logr/logr"
	"github.com/instana/instana-agent-operator/controllers/leaderelection/coordination_api"
	"github.com/procyon-projects/chrono"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// TODO move to struct, should not be exposed externally
	LeaderElectionTask          chrono.ScheduledTask
	LeaderElectionTaskScheduler chrono.TaskScheduler

	coordinationApi = coordination_api.New()
)

type LeaderElector struct {
	Ctx    context.Context
	Client client.Client
	Log    logr.Logger
}

/*
	StartCoordination starts scheduling of leader elector coordination between Instana Agents.

	Agent Coordination works by requesting the "resources" for which an Agent would be leader. This could be one or more.
	Different Agents might request similar or different "resources".

	The Operator should, for every requested resource, appoint one specific Agent pod as leader by calling the
	/coordination/assigned endpoint.
*/
func (l *LeaderElector) StartCoordination(agentNameSpace string) error {
	LeaderElectionTaskScheduler = chrono.NewDefaultTaskScheduler()
	var err error
	LeaderElectionTask, err = LeaderElectionTaskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {

		activePods, err := l.fetchPods(agentNameSpace)
		if err != nil {
			l.Log.Error(err, "Unable to fetch agent pods for doing election")
			return
		}
		l.pollAgentsAndAssignLeaders(activePods)

	}, 5*time.Second)
	if err != nil {
		l.Log.Error(err, "Failure scheduling Leader Elector task")
		return errors.New("failure starting leader elector coordination")
	}
	l.Log.Info("Leader Election task has been scheduled successfully.")

	return nil
}

func (l *LeaderElector) CancelLeaderElection() {
	if LeaderElectionTask != nil && !LeaderElectionTask.IsCancelled() {
		LeaderElectionTask.Cancel()
	}
	if !LeaderElectionTaskScheduler.IsShutdown() {
		LeaderElectionTaskScheduler.Shutdown()
	}
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

// pollAgentsAndAssignLeaders will first get all "requested resources" from every Agent Pod. It will then calculate new
// assignments, prioritizing any Pod that already holds that assignment.
// The function is executed in a loop, so that should assignments fail for any Pod, determining assignments starts over from scratch.
func (l *LeaderElector) pollAgentsAndAssignLeaders(pods map[string]coreV1.Pod) {
outer:
	for {
		// Safeguard to prevent infinite loop should we fail to assign all pods
		if len(pods) == 0 {
			return
		}

		leadershipStatus := l.pollLeadershipStatus(pods)
		// if leadershipStatus.Status is empty list that means we were not able to get leadership status for any pod
		// so we can't proceed with leadership assigning process and we should return
		if len(leadershipStatus.Status) == 0 {
			return
		}

		desiredPodWithAssignments := l.calculateDesiredAssignments(leadershipStatus)

		// As there can be multiple pods with different resource assignments, have to try Pod assignment one by one.
		// If one fails, the pod needs to be removed and assignments re-calculated
		for pod, assignments := range desiredPodWithAssignments {
			if result := l.assign(pods, leadershipStatus, pod, assignments); !result {
				// Failure for assigning to one of the pods, need to re-evaluate and re-calculate coordination
				delete(pods, pod)
				continue outer
			}
		}

		// Because of a bug in the Agent code, it could happen we have Pods with _no_ requests but _with_ assignments, clean
		// these up although we're not interested failures as the Pod might get restarted
		for _, pod := range leadershipStatus.getPodsWithAssignmentsNoRequests() {
			c := pods[pod]
			l.Log.Info(fmt.Sprintf("Pod with UID %v has assignments but no requests. Resetting.", c.GetObjectMeta().GetName()))
			l.assign(pods, leadershipStatus, pod, []string{})
		}

		// All assignments finished correctly, so exit
		return
	}
}

func (l *LeaderElector) assign(activePods map[string]coreV1.Pod, leadershipStatus *LeadershipStatus, desiredPod string, assignments []string) bool {
	less := func(a, b string) bool { return a < b }
	if !cmp.Equal(assignments, leadershipStatus.getAssignmentsForPod(desiredPod), cmpopts.SortSlices(less)) {
		// Only need to update if desired assignments are not yet equal to actual assignments
		pod := activePods[desiredPod]
		if err := coordinationApi.Assign(pod, assignments); err != nil {
			l.Log.Error(err, fmt.Sprintf("Failed to assign leadership %v to pod: %v", assignments, pod.GetObjectMeta().GetName()))
			return false
		}
		l.Log.Info(fmt.Sprintf("Assigned leadership of %v to pod: %v", assignments, pod.GetObjectMeta().GetName()))
	}
	return true
}

func (l *LeaderElector) pollLeadershipStatus(pods map[string]coreV1.Pod) *LeadershipStatus {
	resourcesByPod := make(map[string]*coordination_api.CoordinationRecord)
	for uid, pod := range pods {
		coordinationRecord, err := coordinationApi.PollPod(pod)
		if err != nil {
			// Logging on Info level because could just happen that Pod is not ready
			l.Log.Info(fmt.Sprintf("Unable to poll coordination status for Pod %v: %v", pod.GetObjectMeta().GetName(), err))
		} else {
			l.Log.V(1).Info(fmt.Sprintf("Coordination status was successfully polled for Pod %v", pod.GetObjectMeta().GetName()))
			resourcesByPod[uid] = coordinationRecord
		}

	}
	return &LeadershipStatus{Status: resourcesByPod}
}

func (l *LeaderElector) calculateDesiredAssignments(leadershipStatus *LeadershipStatus) map[string][]string {
	desiredPodWithAssignments := make(map[string][]string)

	// For every requested resource, determine if there is a 'leader' already, if not, pick a random Pod from the eligible ones
	for resource, podsList := range leadershipStatus.getRequestedResourcesWithPods() {
		var desiredPod string
		var ok bool

		if desiredPod, ok = leadershipStatus.getCurrentLeaderPodForResource(resource); !ok {
			// We can be certain len(podsList) > 0 and so invocation rand.Intn() is safe
			desiredPod = podsList[rand.Intn(len(podsList))]
		}

		if assignments, contains := desiredPodWithAssignments[desiredPod]; contains {
			desiredPodWithAssignments[desiredPod] = append(assignments, resource)
		} else {
			desiredPodWithAssignments[desiredPod] = []string{resource}
		}
	}

	return desiredPodWithAssignments
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
