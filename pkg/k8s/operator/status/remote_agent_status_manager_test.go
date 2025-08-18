/*
(c) Copyright IBM Corp. 2024
*/

package status

import (
	"context"
	"testing"

	"github.com/go-errors/errors"
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/internal/testmocks"

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

	writer := new(testmocks.MockSubResourceWriter)
	writer.On("Patch",
		mock.Anything, // ctx
		mock.Anything, // obj
		mock.Anything, // patch
		mock.Anything, // opts
	).Return(errors.New("FAILURE"))

	instanaAgentClient := new(testmocks.MockInstanaAgentClient)
	instanaAgentClient.On("GetAsResult",
		mock.Anything, // ctx
		mock.Anything, // key
		mock.Anything, // obj
		mock.Anything, // opts
	).Return(result.Of[k8sclient.Object](&unstructured.Unstructured{}, nil))

	instanaAgentClient.On("Status").Return(writer)

	agentStatusManager := NewInstanaAgentRemoteStatusManager(instanaAgentClient, record.NewFakeRecorder(10))
	agentStatusManager.SetAgentOld(&instanav1.InstanaAgentRemote{})
	agentStatusManager.SetAgentSecretConfig(types.NamespacedName{
		Name:      "SetAgentSecretConfigName",
		Namespace: "SetAgentSecretConfigNamespace",
	})
	err := agentStatusManager.UpdateAgentStatus(ctx, nil)
	assertions.NotNil(err)

	writer.AssertExpectations(t)
	instanaAgentClient.AssertExpectations(t)
}

// Note: The original file had a large commented-out test function.
// If this test is needed, it should be uncommented and migrated.
// For now, we're keeping it commented out as it was in the original file.

// func TestUpdateInstanaAgentRemoteStatus(t *testing.T) {
// 	instanaAgent := instanav1.InstanaAgentRemote{}
// 	configSecret := types.NamespacedName{
// 		Name:      "SetAgentSecretConfigName",
// 		Namespace: "SetAgentSecretConfigNamespace",
// 	}
// 	remoteDeployment := &types.NamespacedName{Name: "test_remote_deployment", Namespace: "test_namespace_deployment"}

// 	num := int64(1)
// 	semVer := instanav1.SemanticVersion{}
// 	brokenOperatorVersion := "not_a_real_version_number"

// 	for _, test := range []struct {
// 		name                  string
// 		getAsResultErrors     []error
// 		agent                 *instanav1.InstanaAgentRemote
// 		configSecret          *types.NamespacedName
// 		instanaAgentRemoteDeployment *types.NamespacedName
// 		reconciliationErrors  error
// 		expected              string
// 		envVarOperatorVersion *string
// 	}{
// 		// Test cases would go here
// 	} {
// 		t.Run(
// 			test.name, func(t *testing.T) {
// 				if test.envVarOperatorVersion != nil {
// 					os.Setenv("OPERATOR_VERSION", *test.envVarOperatorVersion)
// 					defer func() {
// 						os.Setenv("OPERATOR_VERSION", "")
// 					}()
// 				}

// 				assertions := require.New(t)
// 				ctx, cancel := context.WithCancel(context.Background())
// 				defer cancel()

// 				writer := new(testmocks.MockSubResourceWriter)
// 				writer.On("Patch",
// 					mock.Anything, // ctx
// 					mock.Anything, // obj
// 					mock.Anything, // patch
// 					mock.Anything, // opts
// 				).Return(nil)

// 				instanaAgentClient := new(testmocks.MockInstanaAgentClient)
//
// 				// Set up expectations for each error in the test case
// 				for _, val := range test.getAsResultErrors {
// 					instanaAgentClient.On("GetAsResult",
// 						mock.Anything, // ctx
// 						mock.Anything, // key
// 						mock.Anything, // obj
// 						mock.Anything, // opts
// 					).Return(result.Of[k8sclient.Object](&unstructured.Unstructured{}, val)).Once()
// 				}
//
// 				instanaAgentClient.On("Status").Return(writer)

// 				agentStatusManager := NewInstanaAgentRemoteStatusManager(instanaAgentClient, record.NewFakeRecorder(10))

// 				if test.agent != nil {
// 					agentStatusManager.SetAgentOld(test.agent)
// 				}
// 				if test.configSecret != nil {
// 					agentStatusManager.SetAgentSecretConfig(*test.configSecret)
// 				}
// 				if test.instanaAgentRemoteDeployment != nil {
// 					agentStatusManager.AddAgentDeployment(*test.instanaAgentRemoteDeployment)
// 				}

// 				err := agentStatusManager.UpdateAgentStatus(ctx, test.reconciliationErrors)

// 				if test.expected != "" {
// 					assertions.Equal(test.expected, err.Error())
// 				} else {
// 					assertions.Nil(err)
// 				}
//
// 				writer.AssertExpectations(t)
// 				instanaAgentClient.AssertExpectations(t)
// 			},
// 		)
// 	}
// }

// Made with Bob
