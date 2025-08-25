/*
(c) Copyright IBM Corp. 2024
*/

package status

import (
	"context"
	"testing"

	"github.com/go-errors/errors"
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/internal/mocks"

	"github.com/instana/instana-agent-operator/pkg/result"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestUpdateInstanaAgentRemoteStatusReturnsErrorOnPatchFailure(t *testing.T) {
	assertions := require.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	writer := &mocks.MockSubResourceWriter{}
	defer writer.AssertExpectations(t)
	writer.On("Patch", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("FAILURE"))

	instanaAgentClient := &mocks.MockInstanaAgentClient{}
	defer instanaAgentClient.AssertExpectations(t)
	instanaAgentClient.On("GetAsResult", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(result.Of[k8sclient.Object](&unstructured.Unstructured{}, nil))
	instanaAgentClient.On("Status").
		Return(writer)

	agentStatusManager := NewInstanaAgentRemoteStatusManager(instanaAgentClient, record.NewFakeRecorder(10))
	agentStatusManager.SetAgentOld(&instanav1.InstanaAgentRemote{})
	agentStatusManager.SetAgentSecretConfig(types.NamespacedName{
		Name:      "SetAgentSecretConfigName",
		Namespace: "SetAgentSecretConfigNamespace",
	})
	err := agentStatusManager.UpdateAgentStatus(ctx, nil)
	assertions.NotNil(err)
}
