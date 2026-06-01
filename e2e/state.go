/*
 * (c) Copyright IBM Corp. 2025
 */

package e2e

import (
	"sync"
)

type suiteState struct {
	mu                 sync.Mutex
	fullResetRequested bool
	fullResetReason    string
}

var currentSuiteState = suiteState{}

// MarkFullResetRequired allows tests to mark the environment as dirty, forcing the next test to perform a full cleanup.
func MarkFullResetRequired(reason string) {
	currentSuiteState.mu.Lock()
	defer currentSuiteState.mu.Unlock()
	currentSuiteState.fullResetRequested = true
	currentSuiteState.fullResetReason = reason
}

// FullResetRequested returns whether a full cleanup has been requested together with the reason.
func FullResetRequested() (bool, string) {
	currentSuiteState.mu.Lock()
	defer currentSuiteState.mu.Unlock()
	return currentSuiteState.fullResetRequested, currentSuiteState.fullResetReason
}

// ClearFullResetRequest clears a previously recorded full cleanup request.
func ClearFullResetRequest() {
	currentSuiteState.mu.Lock()
	defer currentSuiteState.mu.Unlock()
	currentSuiteState.fullResetRequested = false
	currentSuiteState.fullResetReason = ""
}

// Made with Bob
