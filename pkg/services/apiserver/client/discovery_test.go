package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	discoveryfake "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/rest"
	clientgotesting "k8s.io/client-go/testing"
)

// preferredFakeDiscovery wraps the fake discovery client and overrides
// ServerPreferredResources because the upstream fake always returns nil.
type preferredFakeDiscovery struct {
	*discoveryfake.FakeDiscovery
	preferred []*metav1.APIResourceList
	prefErr   error
}

func (p *preferredFakeDiscovery) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	return p.preferred, p.prefErr
}

func newFakeDiscoveryClient(resources []*metav1.APIResourceList, preferred []*metav1.APIResourceList, prefErr error) *DiscoveryClientImpl {
	fake := &discoveryfake.FakeDiscovery{Fake: &clientgotesting.Fake{Resources: resources}}
	var iface discovery.DiscoveryInterface = &preferredFakeDiscovery{
		FakeDiscovery: fake,
		preferred:     preferred,
		prefErr:       prefErr,
	}
	return &DiscoveryClientImpl{
		restConfig:         &rest.Config{},
		DiscoveryInterface: iface,
	}
}

func TestNewDiscoveryClient(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c, err := NewDiscoveryClient(&rest.Config{Host: "http://localhost"})
		require.NoError(t, err)
		require.NotNil(t, c)
	})

	t.Run("error from invalid host", func(t *testing.T) {
		_, err := NewDiscoveryClient(&rest.Config{Host: "://bad-host"})
		require.Error(t, err)
	})
}

func TestGetResourceForKind(t *testing.T) {
	resources := []*metav1.APIResourceList{{
		GroupVersion: "example.com/v1",
		APIResources: []metav1.APIResource{
			{Name: "widgets", Kind: "Widget"},
			{Name: "gadgets", Kind: "Gadget"},
		},
	}}
	c := newFakeDiscoveryClient(resources, nil, nil)

	t.Run("found", func(t *testing.T) {
		gvr, err := c.GetResourceForKind(schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Widget"})
		require.NoError(t, err)
		require.Equal(t, "widgets", gvr.Resource)
		require.Equal(t, "example.com", gvr.Group)
		require.Equal(t, "v1", gvr.Version)
	})

	t.Run("not found in group version", func(t *testing.T) {
		_, err := c.GetResourceForKind(schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Missing"})
		require.Error(t, err)
	})

	t.Run("group version not registered", func(t *testing.T) {
		_, err := c.GetResourceForKind(schema.GroupVersionKind{Group: "other.com", Version: "v1", Kind: "Widget"})
		require.Error(t, err)
	})
}

func TestGetKindForResource(t *testing.T) {
	resources := []*metav1.APIResourceList{{
		GroupVersion: "example.com/v1",
		APIResources: []metav1.APIResource{
			{Name: "widgets", Kind: "Widget"},
		},
	}}
	c := newFakeDiscoveryClient(resources, nil, nil)

	t.Run("found", func(t *testing.T) {
		gvk, err := c.GetKindForResource(schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"})
		require.NoError(t, err)
		require.Equal(t, "Widget", gvk.Kind)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := c.GetKindForResource(schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "missing"})
		require.Error(t, err)
	})

	t.Run("group version not registered", func(t *testing.T) {
		_, err := c.GetKindForResource(schema.GroupVersionResource{Group: "nope", Version: "v1", Resource: "widgets"})
		require.Error(t, err)
	})
}

func TestGetPreferredVersion(t *testing.T) {
	preferred := []*metav1.APIResourceList{
		{
			GroupVersion: "example.com/v1",
			APIResources: []metav1.APIResource{
				{Name: "widgets", Kind: "Widget"},
			},
		},
		{
			GroupVersion: "other.com/v2",
			APIResources: []metav1.APIResource{
				{Name: "gadgets", Kind: "Gadget"},
			},
		},
	}
	c := newFakeDiscoveryClient(nil, preferred, nil)

	t.Run("found", func(t *testing.T) {
		gvr, gvk, err := c.GetPreferredVesion(schema.GroupResource{Group: "example.com", Resource: "widgets"})
		require.NoError(t, err)
		require.Equal(t, "widgets", gvr.Resource)
		require.Equal(t, "v1", gvr.Version)
		require.Equal(t, "Widget", gvk.Kind)
	})

	t.Run("group matches but resource missing", func(t *testing.T) {
		_, _, err := c.GetPreferredVesion(schema.GroupResource{Group: "example.com", Resource: "missing"})
		require.Error(t, err)
	})

	t.Run("group not present", func(t *testing.T) {
		_, _, err := c.GetPreferredVesion(schema.GroupResource{Group: "absent.com", Resource: "widgets"})
		require.Error(t, err)
	})

	t.Run("propagates ServerPreferredResources error", func(t *testing.T) {
		errClient := newFakeDiscoveryClient(nil, nil, errors.New("boom"))
		_, _, err := errClient.GetPreferredVesion(schema.GroupResource{Group: "example.com", Resource: "widgets"})
		require.Error(t, err)
	})
}

func TestGetPreferredVersionForKind(t *testing.T) {
	preferred := []*metav1.APIResourceList{
		{
			GroupVersion: "example.com/v1",
			APIResources: []metav1.APIResource{
				{Name: "widgets", Kind: "Widget"},
			},
		},
		{
			// Core API group has no slash in the GroupVersion.
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Name: "pods", Kind: "Pod"},
			},
		},
	}
	c := newFakeDiscoveryClient(nil, preferred, nil)

	t.Run("found in non-core group", func(t *testing.T) {
		gvr, gvk, err := c.GetPreferredVersionForKind(schema.GroupKind{Group: "example.com", Kind: "Widget"})
		require.NoError(t, err)
		require.Equal(t, "widgets", gvr.Resource)
		require.Equal(t, "example.com", gvr.Group)
		require.Equal(t, "v1", gvr.Version)
		require.Equal(t, "Widget", gvk.Kind)
	})

	t.Run("found in core group", func(t *testing.T) {
		gvr, gvk, err := c.GetPreferredVersionForKind(schema.GroupKind{Group: "", Kind: "Pod"})
		require.NoError(t, err)
		require.Equal(t, "pods", gvr.Resource)
		require.Equal(t, "", gvr.Group)
		require.Equal(t, "v1", gvr.Version)
		require.Equal(t, "Pod", gvk.Kind)
	})

	t.Run("group present, kind missing", func(t *testing.T) {
		_, _, err := c.GetPreferredVersionForKind(schema.GroupKind{Group: "example.com", Kind: "Missing"})
		require.Error(t, err)
	})

	t.Run("group missing", func(t *testing.T) {
		_, _, err := c.GetPreferredVersionForKind(schema.GroupKind{Group: "absent.com", Kind: "Widget"})
		require.Error(t, err)
	})

	t.Run("propagates ServerPreferredResources error", func(t *testing.T) {
		errClient := newFakeDiscoveryClient(nil, nil, errors.New("boom"))
		_, _, err := errClient.GetPreferredVersionForKind(schema.GroupKind{Group: "example.com", Kind: "Widget"})
		require.Error(t, err)
	})
}

func TestWaitForAvailability(t *testing.T) {
	// Shorten the defaults so the failure path completes quickly.
	origPoll := defaultPollInterval
	origTimeout := defaultAvailabilityTimeout
	defaultPollInterval = 5 * time.Millisecond
	defaultAvailabilityTimeout = 50 * time.Millisecond
	t.Cleanup(func() {
		defaultPollInterval = origPoll
		defaultAvailabilityTimeout = origTimeout
	})

	t.Run("returns nil when group is available", func(t *testing.T) {
		resources := []*metav1.APIResourceList{{
			GroupVersion: "example.com/v1",
			APIResources: []metav1.APIResource{{Name: "widgets", Kind: "Widget"}},
		}}
		c := newFakeDiscoveryClient(resources, nil, nil)
		err := c.WaitForAvailability(context.Background(), schema.GroupVersion{Group: "example.com", Version: "v1"})
		require.NoError(t, err)
	})

	t.Run("times out when group never becomes available", func(t *testing.T) {
		c := newFakeDiscoveryClient(nil, nil, nil)
		err := c.WaitForAvailability(context.Background(), schema.GroupVersion{Group: "missing.io", Version: "v1"})
		require.Error(t, err)
	})
}
