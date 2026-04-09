/*
(c) Copyright IBM Corp. 2026

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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

// MockETCDDiscoverer is a mock implementation of ETCDDiscoverer for testing
type MockETCDDiscoverer struct {
	ShouldSkipDiscoveryFunc func(ctx context.Context, agent *instanav1.InstanaAgent) (bool, error)
	FindETCDServiceFunc     func(ctx context.Context, log logr.Logger) (*corev1.Service, error)
	GetServiceWithLabelFunc func(
		ctx context.Context,
		name, labelKey, labelValue string,
	) (*corev1.Service, error)
	GetServiceByNameFunc          func(ctx context.Context, name string) (*corev1.Service, error)
	FindMetricsPortAndSchemeFunc  func(service *corev1.Service) (*int32, string)
	BuildTargetsFromEndpointsFunc func(
		ctx context.Context,
		service *corev1.Service,
		metricsPort int32,
		scheme string,
	) ([]string, error)
	BuildTargetsFromLegacyEndpointsFunc func(
		ctx context.Context,
		service *corev1.Service,
		metricsPort int32,
		scheme string,
	) ([]string, error)
	CheckCASecretExistsFunc func(ctx context.Context, agent *instanav1.InstanaAgent) bool
}

func (m *MockETCDDiscoverer) ShouldSkipDiscovery(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
) (bool, error) {
	if m.ShouldSkipDiscoveryFunc != nil {
		return m.ShouldSkipDiscoveryFunc(ctx, agent)
	}
	return false, nil
}

func (m *MockETCDDiscoverer) FindETCDService(
	ctx context.Context,
	log logr.Logger,
) (*corev1.Service, error) {
	if m.FindETCDServiceFunc != nil {
		return m.FindETCDServiceFunc(ctx, log)
	}
	return nil, nil
}

func (m *MockETCDDiscoverer) GetServiceWithLabel(
	ctx context.Context,
	name, labelKey, labelValue string,
) (*corev1.Service, error) {
	if m.GetServiceWithLabelFunc != nil {
		return m.GetServiceWithLabelFunc(ctx, name, labelKey, labelValue)
	}
	return nil, nil
}

func (m *MockETCDDiscoverer) GetServiceByName(
	ctx context.Context,
	name string,
) (*corev1.Service, error) {
	if m.GetServiceByNameFunc != nil {
		return m.GetServiceByNameFunc(ctx, name)
	}
	return nil, nil
}

func (m *MockETCDDiscoverer) FindMetricsPortAndScheme(service *corev1.Service) (*int32, string) {
	if m.FindMetricsPortAndSchemeFunc != nil {
		return m.FindMetricsPortAndSchemeFunc(service)
	}
	return nil, ""
}

func (m *MockETCDDiscoverer) BuildTargetsFromEndpoints(
	ctx context.Context,
	service *corev1.Service,
	metricsPort int32,
	scheme string,
) ([]string, error) {
	if m.BuildTargetsFromEndpointsFunc != nil {
		return m.BuildTargetsFromEndpointsFunc(ctx, service, metricsPort, scheme)
	}
	return nil, nil
}

func (m *MockETCDDiscoverer) BuildTargetsFromLegacyEndpoints(
	ctx context.Context,
	service *corev1.Service,
	metricsPort int32,
	scheme string,
) ([]string, error) {
	if m.BuildTargetsFromLegacyEndpointsFunc != nil {
		return m.BuildTargetsFromLegacyEndpointsFunc(ctx, service, metricsPort, scheme)
	}
	return nil, nil
}

func (m *MockETCDDiscoverer) CheckCASecretExists(
	ctx context.Context, agent *instanav1.InstanaAgent,
) bool {
	if m.CheckCASecretExistsFunc != nil {
		return m.CheckCASecretExistsFunc(ctx, agent)
	}
	return false
}
