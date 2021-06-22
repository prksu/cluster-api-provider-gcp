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
	"reflect"

	"google.golang.org/api/compute/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"
	infrav1 "sigs.k8s.io/cluster-api-provider-gcp/api/v1alpha4"
	"sigs.k8s.io/cluster-api-provider-gcp/cloud"
	"sigs.k8s.io/cluster-api-provider-gcp/cloud/gcperrors"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Reconcile reconcile cluster network components.
func (s *Service) Reconcile(ctx context.Context) error {
	network, err := s.createOrGetNetwork(ctx)
	if err != nil {
		return err
	}

	if network.Description == infrav1.ClusterTagKey(s.scope.Name()) {
		router, err := s.createOrGetRouter(ctx, network)
		if err != nil {
			return err
		}

		s.scope.Network().Router = pointer.String(router.SelfLink)
	}

	s.scope.Network().SelfLink = pointer.String(network.SelfLink)
	return s.createOrPatchSubnet(ctx, network)
}

// Delete delete cluster network components.
func (s *Service) Delete(ctx context.Context) error {
	log := log.FromContext(ctx)
	if err := s.deleteOrPatchSubnetwork(ctx); err != nil {
		return err
	}

	networkKey := cloud.GlobalKey(s.scope.NetworkName())
	log.V(2).Info("Looking for network before deleting", "name", networkKey)
	network, err := s.networks.Get(ctx, networkKey)
	if err != nil {
		return gcperrors.IgnoreNotFound(err)
	}

	if network.Description != infrav1.ClusterTagKey(s.scope.Name()) {
		return nil
	}

	log.V(2).Info("Found network created by capg", "name", s.scope.NetworkName())

	routerSpec := s.scope.NatRouterSpec()
	routerKey := cloud.RegionalKey(routerSpec.Name, s.scope.Region())
	log.V(2).Info("Looking for cloudnat router before deleting", "name", routerSpec.Name)
	router, err := s.routers.Get(ctx, routerKey)
	if err != nil && !gcperrors.IsNotFound(err) {
		return err
	}

	if router != nil && router.Description == infrav1.ClusterTagKey(s.scope.Name()) {
		if err := s.routers.Delete(ctx, routerKey); err != nil && !gcperrors.IsNotFound(err) {
			return err
		}
	}

	if err := s.networks.Delete(ctx, networkKey); err != nil {
		log.Error(err, "Error deleting a network", "name", s.scope.NetworkName())
		return err
	}

	s.scope.Network().Router = nil
	s.scope.Network().SelfLink = nil
	return nil
}

// createOrGetNetwork creates a network if not exist otherwise return existing network.
func (s *Service) createOrGetNetwork(ctx context.Context) (*compute.Network, error) {
	log := log.FromContext(ctx)
	log.V(2).Info("Looking for network", "name", s.scope.NetworkName())
	networkKey := cloud.GlobalKey(s.scope.NetworkName())
	network, err := s.networks.Get(ctx, networkKey)
	if err != nil {
		if !gcperrors.IsNotFound(err) {
			log.Error(err, "Error looking for network", "name", s.scope.NetworkName())
			return nil, err
		}

		log.V(2).Info("Creating a network", "name", s.scope.NetworkName())
		if err := s.networks.Insert(ctx, networkKey, s.scope.NetworkSpec()); err != nil {
			log.Error(err, "Error creating a network", "name", s.scope.NetworkName())
			return nil, err
		}

		network, err = s.networks.Get(ctx, networkKey)
		if err != nil {
			return nil, err
		}
	}

	return network, nil
}

// createOrGetRouter creates a cloudnat router if not exist otherwise return the existing.
func (s *Service) createOrGetRouter(ctx context.Context, network *compute.Network) (*compute.Router, error) {
	log := log.FromContext(ctx)
	spec := s.scope.NatRouterSpec()
	log.V(2).Info("Looking for cloudnat router", "name", spec.Name)
	routerKey := cloud.RegionalKey(spec.Name, s.scope.Region())
	router, err := s.routers.Get(ctx, routerKey)
	if err != nil {
		if !gcperrors.IsNotFound(err) {
			log.Error(err, "Error looking for cloudnat router", "name", spec.Name)
			return nil, err
		}

		spec.Network = network.SelfLink
		spec.Description = infrav1.ClusterTagKey(s.scope.Name())
		log.V(2).Info("Creating a cloudnat router", "name", spec.Name)
		if err := s.routers.Insert(ctx, routerKey, spec); err != nil {
			log.Error(err, "Error creating a cloudnat router", "name", spec.Name)
			return nil, err
		}

		router, err = s.routers.Get(ctx, routerKey)
		if err != nil {
			return nil, err
		}
	}

	return router, nil
}

// createOrPatchSubnet creates a subnet if not exist and patch if subnet already exist but
// does not have secondary ip ranges mentioned in the spec.
func (s *Service) createOrPatchSubnet(ctx context.Context, network *compute.Network) error {
	log := log.FromContext(ctx)
	for _, spec := range s.scope.SubnetworksSpec() {
		log.V(2).Info("Found additional spec for subnet", "name", spec.Name)
		subnetName := spec.Name
		subnetKey := cloud.RegionalKey(subnetName, s.scope.Region())
		log.V(2).Info("Looking for subnet", "name", subnetName)
		subnet, err := s.subnetworks.Get(ctx, subnetKey)
		if err != nil {
			if !gcperrors.IsNotFound(err) {
				log.Error(err, "Error looking for subnet", "name", subnetName)
				return err
			}

			spec.Network = network.SelfLink
			spec.Description = infrav1.ClusterTagKey(s.scope.Name())
			log.V(2).Info("Creating a subnet", "name", subnetName)
			if err := s.subnetworks.Insert(ctx, subnetKey, spec); err != nil {
				log.Error(err, "Error creating a subnet", "name", subnetName)
				return err
			}

			subnet, err = s.subnetworks.Get(ctx, subnetKey)
			if err != nil {
				return err
			}
		}

		// Try to add secondary ip ranges from spec to existing subnet
		// in the case user want to use secondary ip range for ip alias.
		secondaryIPRange := subnet.SecondaryIpRanges
		secondaryIPSets := sets.NewString()
		for _, ipRange := range subnet.SecondaryIpRanges {
			secondaryIPSets.Insert(ipRange.RangeName)
		}

		for _, ipRangeFromSpec := range spec.SecondaryIpRanges {
			if !secondaryIPSets.Has(ipRangeFromSpec.RangeName) {
				secondaryIPRange = append(secondaryIPRange, ipRangeFromSpec)
			}
		}

		if !reflect.DeepEqual(secondaryIPRange, subnet.SecondaryIpRanges) {
			log.V(2).Info("Patch a secondary ip ranges for subnet", "name", subnetName)
			subnet.SecondaryIpRanges = secondaryIPRange
			if err := s.subnetworks.Patch(ctx, subnetKey, subnet); err != nil {
				return err
			}
		}
	}

	return nil
}

// deleteOrPatchSubnetwork deletes the subnet if created by capg and patch the subnet
// to restore additional secondary ip range added by capg.
func (s *Service) deleteOrPatchSubnetwork(ctx context.Context) error {
	log := log.FromContext(ctx)
	specs := s.scope.SubnetworksSpec()
	for _, spec := range specs {
		subnetName := spec.Name
		subnetKey := cloud.RegionalKey(subnetName, s.scope.Region())
		log.V(2).Info("Looking for subnet before deleting", "name", subnetName)
		subnet, err := s.subnetworks.Get(ctx, subnetKey)
		if err != nil {
			if !gcperrors.IsNotFound(err) {
				continue
			}

			return err
		}

		if subnet.Description == infrav1.ClusterTagKey(s.scope.Name()) {
			log.V(2).Info("Found subnet created by capg. Deleting", "name", subnetName)
			if err := s.subnetworks.Delete(ctx, subnetKey); err != nil {
				log.Error(err, "Error deleting subnet", "name", subnetName)
				return err
			}

			continue
		}

		// Try to restore secondary ip ranges by removing additional secondary ip range.
		secondaryIPRange := make([]*compute.SubnetworkSecondaryRange, 0, len(subnet.SecondaryIpRanges))
		secondaryIPSets := sets.NewString()
		for _, ipRangeFromSpec := range spec.SecondaryIpRanges {
			secondaryIPSets.Insert(ipRangeFromSpec.RangeName)
		}

		for _, ipRange := range subnet.SecondaryIpRanges {
			// Insert ipRange into secondaryIPRange only if secondaryIPSets above does not have ipRange
			if !secondaryIPSets.Has(ipRange.RangeName) {
				secondaryIPRange = append(secondaryIPRange, ipRange)
			}
		}

		if !reflect.DeepEqual(secondaryIPRange, subnet.SecondaryIpRanges) {
			log.V(2).Info("Patch a secondary ip ranges for subnet", "name", subnetName)
			subnet.SecondaryIpRanges = secondaryIPRange
			if err := s.subnetworks.Patch(ctx, subnetKey, subnet); err != nil {
				return err
			}
		}
	}

	return nil
}
