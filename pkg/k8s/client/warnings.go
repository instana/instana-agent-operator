/*
(c) Copyright IBM Corp. 2025

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"context"
	"strings"

	"k8s.io/client-go/rest"
	crlog "sigs.k8s.io/controller-runtime/pkg/log"
)

const sessionAffinityHeadlessWarning = "spec.SessionAffinity is ignored for headless services"

// ConfigureWarningHandler replaces the default warning handler on the provided REST config with
// one that suppresses noisy warnings we already account for while delegating everything else to
// controller-runtime's default logger.
func ConfigureWarningHandler(cfg *rest.Config) {
	delegate := crlog.NewKubeAPIWarningLogger(crlog.KubeAPIWarningLoggerOptions{})
	cfg.WarningHandlerWithContext = &filteringWarningHandler{
		delegate:        delegate,
		ignoredMessages: []string{sessionAffinityHeadlessWarning},
	}
}

type filteringWarningHandler struct {
	delegate        rest.WarningHandlerWithContext
	ignoredMessages []string
}

func (f *filteringWarningHandler) HandleWarningHeaderWithContext(
	ctx context.Context,
	code int,
	agent string,
	message string,
) {
	for _, ignored := range f.ignoredMessages {
		if strings.Contains(message, ignored) {
			return
		}
	}
	if f.delegate != nil {
		f.delegate.HandleWarningHeaderWithContext(ctx, code, agent, message)
	}
}
