package client

import (
	"context"
	"errors"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/rest"
	clientgotesting "k8s.io/client-go/testing"

	"github.com/stretchr/testify/require"
)

func newTestDiscoveryClient(t *testing.T, resources []*metav1.APIResourceList) *DiscoveryClientImpl {
	t.Helper()

	fake := &fakediscovery.FakeDiscovery{Fake: &clientgotesting.Fake{}}
	fake.Resources = resources

	return &DiscoveryClientImpl{
		restConfig:         &rest.Config{Host: "http://test"},
		DiscoveryInterface: fake,
	}
}

// stubDiscovery implements discovery methods used by DiscoveryClientImpl for preferred-resource tests.
type stubDiscovery struct {
	*fakediscovery.FakeDiscovery
	preferred []*metav1.APIResourceList
	preferredErr error
}

func (s *stubDiscovery) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	if s.preferredErr != nil {
		return nil, s.preferredErr
	}
	return s.preferred, nil
}

func dashboardResources() []*metav1.APIResourceList {
	return []*metav1.APIResourceList{
		{
			GroupVersion: "dashboard.grafana.app/v1beta1",
			APIResources: []metav1.APIResource{
				{Name: "dashboards", Kind: "Dashboard"},
			},
		},
	}
}

func TestNewDiscoveryClient(t *testing.T) {
	t.Run("creates client with valid config", func(t *testing.T) {
		client, err := NewDiscoveryClient(&rest.Config{Host: "http://localhost"})
		require.NoError(t, err)
		require.NotNil(t, client)
	})

}

func TestGetResourceForKind(t *testing.T) {
	resources := dashboardResources()
	client := newTestDiscoveryClient(t, resources)

	t.Run("returns resource for matching kind", func(t *testing.T) {
		gvk := schema.GroupVersionKind{Group: "dashboard.grafana.app", Version: "v1beta1", Kind: "Dashboard"}
		gvr, err := client.GetResourceForKind(gvk)
		require.NoError(t, err)
		require.Equal(t, "dashboard.grafana.app", gvr.Group)
		require.Equal(t, "v1beta1", gvr.Version)
		require.Equal(t, "dashboards", gvr.Resource)
	})

	t.Run("returns error when group version not found", func(t *testing.T) {
		gvk := schema.GroupVersionKind{Group: "missing", Version: "v1", Kind: "Foo"}
		_, err := client.GetResourceForKind(gvk)
		require.Error(t, err)
	})

	t.Run("returns error when kind not found", func(t *testing.T) {
		gvk := schema.GroupVersionKind{Group: "dashboard.grafana.app", Version: "v1beta1", Kind: "NotFound"}
		_, err := client.GetResourceForKind(gvk)
		require.Error(t, err)
		require.Contains(t, err.Error(), "resource not found")
	})

	t.Run("returns error from discovery", func(t *testing.T) {
		fake := &fakediscovery.FakeDiscovery{Fake: &clientgotesting.Fake{}}
		fake.PrependReactor("*", "*", func(action clientgotesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("discovery failed")
		})
		client := &DiscoveryClientImpl{DiscoveryInterface: fake}
		_, err := client.GetResourceForKind(schema.GroupVersionKind{Group: "g", Version: "v1", Kind: "K"})
		require.Error(t, err)
	})
}

func TestGetKindForResource(t *testing.T) {
	client := newTestDiscoveryClient(t, dashboardResources())

	t.Run("returns kind for matching resource", func(t *testing.T) {
		gvr := schema.GroupVersionResource{Group: "dashboard.grafana.app", Version: "v1beta1", Resource: "dashboards"}
		gvk, err := client.GetKindForResource(gvr)
		require.NoError(t, err)
		require.Equal(t, "Dashboard", gvk.Kind)
	})

	t.Run("returns error when resource not found", func(t *testing.T) {
		gvr := schema.GroupVersionResource{Group: "dashboard.grafana.app", Version: "v1beta1", Resource: "missing"}
		_, err := client.GetKindForResource(gvr)
		require.Error(t, err)
		require.Contains(t, err.Error(), "kind not found")
	})
}

func TestGetPreferredVesion(t *testing.T) {
	preferred := dashboardResources()
	client := &DiscoveryClientImpl{
		DiscoveryInterface: &stubDiscovery{
			FakeDiscovery: &fakediscovery.FakeDiscovery{Fake: &clientgotesting.Fake{}},
			preferred:     preferred,
		},
	}

	t.Run("returns preferred GVR and GVK", func(t *testing.T) {
		gr := schema.GroupResource{Group: "dashboard.grafana.app", Resource: "dashboards"}
		gvr, gvk, err := client.GetPreferredVesion(gr)
		require.NoError(t, err)
		require.Equal(t, "dashboards", gvr.Resource)
		require.Equal(t, "Dashboard", gvk.Kind)
	})

	t.Run("returns error when preferred resources fail", func(t *testing.T) {
		client := &DiscoveryClientImpl{
			DiscoveryInterface: &stubDiscovery{
				FakeDiscovery: &fakediscovery.FakeDiscovery{Fake: &clientgotesting.Fake{}},
				preferredErr:  errors.New("preferred failed"),
			},
		}
		_, _, err := client.GetPreferredVesion(schema.GroupResource{Group: "g", Resource: "r"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "getting server's preferred resources")
	})

	t.Run("returns error when resource not in preferred list", func(t *testing.T) {
		client := &DiscoveryClientImpl{
			DiscoveryInterface: &stubDiscovery{
				FakeDiscovery: &fakediscovery.FakeDiscovery{Fake: &clientgotesting.Fake{}},
				preferred:     dashboardResources(),
			},
		}
		_, _, err := client.GetPreferredVesion(schema.GroupResource{Group: "dashboard.grafana.app", Resource: "unknown"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "preferred version not found")
	})
}

func TestGetPreferredVersionForKind(t *testing.T) {
	t.Run("returns preferred GVR and GVK for namespaced API", func(t *testing.T) {
		client := &DiscoveryClientImpl{
			DiscoveryInterface: &stubDiscovery{
				FakeDiscovery: &fakediscovery.FakeDiscovery{Fake: &clientgotesting.Fake{}},
				preferred:     dashboardResources(),
			},
		}
		gk := schema.GroupKind{Group: "dashboard.grafana.app", Kind: "Dashboard"}
		gvr, gvk, err := client.GetPreferredVersionForKind(gk)
		require.NoError(t, err)
		require.Equal(t, "dashboards", gvr.Resource)
		require.Equal(t, "Dashboard", gvk.Kind)
	})

	t.Run("handles core API group version format", func(t *testing.T) {
		coreResources := []*metav1.APIResourceList{
			{
				GroupVersion: "v1",
				APIResources: []metav1.APIResource{
					{Name: "configmaps", Kind: "ConfigMap"},
				},
			},
		}
		client := &DiscoveryClientImpl{
			DiscoveryInterface: &stubDiscovery{
				FakeDiscovery: &fakediscovery.FakeDiscovery{Fake: &clientgotesting.Fake{}},
				preferred:     coreResources,
			},
		}
		gk := schema.GroupKind{Group: "", Kind: "ConfigMap"}
		gvr, gvk, err := client.GetPreferredVersionForKind(gk)
		require.NoError(t, err)
		require.Equal(t, "", gvr.Group)
		require.Equal(t, "v1", gvr.Version)
		require.Equal(t, "configmaps", gvr.Resource)
		require.Equal(t, "ConfigMap", gvk.Kind)
	})

	t.Run("returns error when kind not found", func(t *testing.T) {
		client := &DiscoveryClientImpl{
			DiscoveryInterface: &stubDiscovery{
				FakeDiscovery: &fakediscovery.FakeDiscovery{Fake: &clientgotesting.Fake{}},
				preferred:     dashboardResources(),
			},
		}
		_, _, err := client.GetPreferredVersionForKind(schema.GroupKind{Group: "dashboard.grafana.app", Kind: "Missing"})
		require.Error(t, err)
	})
}

func TestWaitForAvailability(t *testing.T) {
	t.Run("returns immediately when API is available", func(t *testing.T) {
		client := newTestDiscoveryClient(t, dashboardResources())
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		err := client.WaitForAvailability(ctx, schema.GroupVersion{Group: "dashboard.grafana.app", Version: "v1beta1"})
		require.NoError(t, err)
	})

	t.Run("returns context error when API never becomes available", func(t *testing.T) {
		fake := &fakediscovery.FakeDiscovery{Fake: &clientgotesting.Fake{}}
		fake.Resources = nil
		client := &DiscoveryClientImpl{DiscoveryInterface: fake}

		oldInterval := defaultPollInterval
		oldTimeout := defaultAvailabilityTimeout
		defaultPollInterval = 10 * time.Millisecond
		defaultAvailabilityTimeout = 50 * time.Millisecond
		t.Cleanup(func() {
			defaultPollInterval = oldInterval
			defaultAvailabilityTimeout = oldTimeout
		})

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		err := client.WaitForAvailability(ctx, schema.GroupVersion{Group: "missing", Version: "v1"})
		require.Error(t, err)
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})
}
