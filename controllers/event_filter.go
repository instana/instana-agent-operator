package controllers

import (
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
	instanaclient "github.com/instana/instana-agent-operator/pkg/k8s/client"
)

func wasModifiedByOther(objectNew client.Object, objectOld client.Object) bool {
	var lastModifiedBySelf time.Time

	for _, mfe := range objectNew.GetManagedFields() {
		if mfe.Manager == instanaclient.FieldOwnerName {
			if mfe.Time == nil {
				continue
			}
			lastModifiedBySelf = mfe.Time.Time
			break
		}
	}

	if lastModifiedBySelf.IsZero() {
		return true
	}

	for _, mfe := range objectNew.GetManagedFields() {
		if mfe.Manager == instanaclient.FieldOwnerName {
			continue
		} else if mfe.Time == nil && !list.NewDeepContainsElementChecker(objectOld.GetManagedFields()).Contains(mfe) {
			return true
		} else if lastModifiedBySelf.Before(mfe.Time.Time) {
			return true
		}
	}

	return false
}

// Create generic filter for all events, that removes some chattiness mainly when only the Status field has been updated.
func filterPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			switch createEvent.Object.(type) {
			case *instanav1.InstanaAgent:
				return true
			default:
				return false
			}
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			switch e.ObjectOld.(type) {
			case *instanav1.InstanaAgent:
				return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
			default:
				return wasModifiedByOther(e.ObjectNew, e.ObjectOld)
			}
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			switch e.Object.(type) {
			case *instanav1.InstanaAgent:
				return !e.DeleteStateUnknown
			default:
				return true
			}
		},
	}
}
