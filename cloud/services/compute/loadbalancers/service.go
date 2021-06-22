/*
Copyright 2021 The Kubernetes Authors.

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

package loadbalancers

import (
	"context"

	"google.golang.org/api/compute/v1"
	"sigs.k8s.io/cluster-api-provider-gcp/cloud"
)

type addressesInterface interface {
	Get(ctx context.Context, key *cloud.MetaKey) (*compute.Address, error)
	Insert(ctx context.Context, key *cloud.MetaKey, obj *compute.Address) error
	Delete(ctx context.Context, key *cloud.MetaKey) error
}

type backendservicesInterface interface {
	Get(ctx context.Context, key *cloud.MetaKey) (*compute.BackendService, error)
	Insert(ctx context.Context, key *cloud.MetaKey, obj *compute.BackendService) error
	Update(context.Context, *cloud.MetaKey, *compute.BackendService) error
	Delete(ctx context.Context, key *cloud.MetaKey) error
}

type forwardingrulesInterface interface {
	Get(ctx context.Context, key *cloud.MetaKey) (*compute.ForwardingRule, error)
	Insert(ctx context.Context, key *cloud.MetaKey, obj *compute.ForwardingRule) error
	Delete(ctx context.Context, key *cloud.MetaKey) error
}

type healthchecksInterface interface {
	Get(ctx context.Context, key *cloud.MetaKey) (*compute.HealthCheck, error)
	Insert(ctx context.Context, key *cloud.MetaKey, obj *compute.HealthCheck) error
	Delete(ctx context.Context, key *cloud.MetaKey) error
}

type instancegroupsInterface interface {
	Get(ctx context.Context, key *cloud.MetaKey) (*compute.InstanceGroup, error)
	List(ctx context.Context, zone string, fl *cloud.Filter) ([]*compute.InstanceGroup, error)
	Insert(ctx context.Context, key *cloud.MetaKey, obj *compute.InstanceGroup) error
	Delete(ctx context.Context, key *cloud.MetaKey) error
}

type targettcpproxiesInterface interface {
	Get(ctx context.Context, key *cloud.MetaKey) (*compute.TargetTcpProxy, error)
	Insert(ctx context.Context, key *cloud.MetaKey, obj *compute.TargetTcpProxy) error
	Delete(ctx context.Context, key *cloud.MetaKey) error
}

// Scope is an interfaces that hold used methods.
type Scope interface {
	cloud.Cluster
	AddressSpec() *compute.Address
	BackendServiceSpec() *compute.BackendService
	ForwardingRuleSpec() *compute.ForwardingRule
	HealthCheckSpec() *compute.HealthCheck
	InstanceGroupSpec(zone string) *compute.InstanceGroup
	TargetTCPProxySpec() *compute.TargetTcpProxy
}

// Service implements loadbalancers reconciler.
type Service struct {
	scope            Scope
	addresses        addressesInterface
	backendservices  backendservicesInterface
	forwardingrules  forwardingrulesInterface
	healthchecks     healthchecksInterface
	instancegroups   instancegroupsInterface
	targettcpproxies targettcpproxiesInterface
}

var _ cloud.Reconciler = &Service{}

// New returns Service from given scope.
func New(scope Scope) *Service {
	return &Service{
		scope:            scope,
		addresses:        scope.Cloud().GlobalAddresses(),
		backendservices:  scope.Cloud().BackendServices(),
		forwardingrules:  scope.Cloud().GlobalForwardingRules(),
		healthchecks:     scope.Cloud().HealthChecks(),
		instancegroups:   scope.Cloud().InstanceGroups(),
		targettcpproxies: scope.Cloud().TargetTcpProxies(),
	}
}
