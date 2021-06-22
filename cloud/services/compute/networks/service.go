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

package networks

import (
	"context"

	"google.golang.org/api/compute/v1"

	"sigs.k8s.io/cluster-api-provider-gcp/cloud"
)

type networksInterface interface {
	Get(ctx context.Context, key *cloud.MetaKey) (*compute.Network, error)
	Insert(ctx context.Context, key *cloud.MetaKey, obj *compute.Network) error
	Delete(ctx context.Context, key *cloud.MetaKey) error
}

type routersInterface interface {
	Get(ctx context.Context, key *cloud.MetaKey) (*compute.Router, error)
	Insert(ctx context.Context, key *cloud.MetaKey, obj *compute.Router) error
	Delete(ctx context.Context, key *cloud.MetaKey) error
}

type subnetworksInterface interface {
	Get(ctx context.Context, key *cloud.MetaKey) (*compute.Subnetwork, error)
	Insert(ctx context.Context, key *cloud.MetaKey, obj *compute.Subnetwork) error
	Delete(ctx context.Context, key *cloud.MetaKey) error
	Patch(ctx context.Context, key *cloud.MetaKey, obj *compute.Subnetwork) error
}

// Scope is an interfaces that hold used methods.
type Scope interface {
	cloud.Cluster
	NetworkSpec() *compute.Network
	NatRouterSpec() *compute.Router
	SubnetworksSpec() []*compute.Subnetwork
}

// Service implements networks reconciler.
type Service struct {
	scope       Scope
	networks    networksInterface
	routers     routersInterface
	subnetworks subnetworksInterface
}

var _ cloud.Reconciler = &Service{}

// New returns Service from given scope.
func New(scope Scope) *Service {
	return &Service{
		scope:       scope,
		networks:    scope.Cloud().Networks(),
		routers:     scope.Cloud().Routers(),
		subnetworks: scope.Cloud().Subnetworks(),
	}
}
