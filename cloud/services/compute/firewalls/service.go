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

package firewalls

import (
	"context"

	"google.golang.org/api/compute/v1"
	"sigs.k8s.io/cluster-api-provider-gcp/cloud"
)

type firewallsInterface interface {
	Get(ctx context.Context, key *cloud.MetaKey) (*compute.Firewall, error)
	Insert(ctx context.Context, key *cloud.MetaKey, obj *compute.Firewall) error
	Update(ctx context.Context, key *cloud.MetaKey, obj *compute.Firewall) error
	Delete(ctx context.Context, key *cloud.MetaKey) error
}

// Scope is an interfaces that hold used methods.
type Scope interface {
	cloud.ClusterGetter
	FirewallRulesSpec() []*compute.Firewall
}

// Service implements firewalls reconciler.
type Service struct {
	scope     Scope
	firewalls firewallsInterface
}

var _ cloud.Reconciler = &Service{}

// New returns Service from given scope.
func New(scope Scope) *Service {
	return &Service{
		scope:     scope,
		firewalls: scope.Cloud().Firewalls(),
	}
}
