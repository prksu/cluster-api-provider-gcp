package cloud

import (
	"context"
	"time"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/filter"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"google.golang.org/api/compute/v1"
	"k8s.io/client-go/util/flowcontrol"
)

type (
	// Cloud is alias for *cloud.GCE.
	Cloud = *cloud.GCE

	// MetaKey is alias for meta.Key
	// key for a GCP resource.
	MetaKey = meta.Key

	// Filter is alias for filter.F
	// is a filter to be used with List() operations.
	Filter = filter.F
)

var (
	// ZonalKey is alias for meta.ZonalKey
	// returns the key for a zonal resource.
	ZonalKey = meta.ZonalKey

	// RegionalKey is alias for meta.RegionalKey
	// returns the key for a regional resource.
	RegionalKey = meta.RegionalKey

	// GlobalKey is alias for meta.GlobalKey
	// returns the key for a global resource.
	GlobalKey = meta.GlobalKey
)

var (
	// FilterNone is alias for filter.None.
	FilterNone = filter.None
	// FilterRegexp is alias for filter.Regexp
	// returns a filter for fieldName eq regexp v.
	FilterRegexp = filter.Regexp
)

// rateLimiter implements cloud.RateLimiter.
type rateLimiter struct{}

// Accept blocks until the operation can be performed.
func (rl *rateLimiter) Accept(ctx context.Context, key *cloud.RateLimitKey) error {
	if key.Operation == "Get" && key.Service == "Operations" {
		// Wait a minimum amount of time regardless of rate limiter.
		rl := &cloud.MinimumRateLimiter{
			// Convert flowcontrol.RateLimiter into cloud.RateLimiter
			RateLimiter: &cloud.AcceptRateLimiter{
				Acceptor: flowcontrol.NewTokenBucketRateLimiter(5, 5), // 5
			},
			Minimum: time.Second,
		}
		return rl.Accept(ctx, key)
	}
	return nil
}

// NewCloud instantiates *cloud.GCE from given service and projectID.
func NewCloud(service *compute.Service, projectID string) Cloud {
	return cloud.NewGCE(&cloud.Service{
		GA:            service,
		ProjectRouter: &cloud.SingleProjectRouter{ID: projectID},
		RateLimiter:   &rateLimiter{},
	})
}
