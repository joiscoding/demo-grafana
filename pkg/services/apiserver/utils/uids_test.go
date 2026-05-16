package utils

import (
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type objectWithoutMeta struct {
	metav1.TypeMeta
}

func (o *objectWithoutMeta) DeepCopyObject() runtime.Object {
	return &objectWithoutMeta{TypeMeta: o.TypeMeta}
}

func TestCalculateClusterWideUID_stableForSameObject(t *testing.T) {
	obj := &metav1.PartialObjectMetadata{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "folder.grafana.app/v1alpha1",
			Kind:       "Folder",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "org-1",
			Name:      "my-folder",
		},
	}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "folder.grafana.app",
		Version: "v1alpha1",
		Kind:    "Folder",
	})

	uid1 := CalculateClusterWideUID(obj)
	uid2 := CalculateClusterWideUID(obj)

	require.NotEmpty(t, uid1)
	require.Equal(t, uid1, uid2)
}

func TestCalculateClusterWideUID_differsByGVKNamespaceAndName(t *testing.T) {
	base := func(ns, name string, gvk schema.GroupVersionKind) *metav1.PartialObjectMetadata {
		obj := &metav1.PartialObjectMetadata{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      name,
			},
		}
		obj.SetGroupVersionKind(gvk)
		return obj
	}
	gvk := schema.GroupVersionKind{Group: "test.grafana.app", Version: "v1", Kind: "Thing"}

	uidBase := CalculateClusterWideUID(base("ns-a", "name-a", gvk))
	uidOtherNS := CalculateClusterWideUID(base("ns-b", "name-a", gvk))
	uidOtherName := CalculateClusterWideUID(base("ns-a", "name-b", gvk))
	uidOtherKind := CalculateClusterWideUID(base("ns-a", "name-a", schema.GroupVersionKind{
		Group: "test.grafana.app", Version: "v1", Kind: "Other",
	}))

	require.NotEqual(t, uidBase, uidOtherNS)
	require.NotEqual(t, uidBase, uidOtherName)
	require.NotEqual(t, uidBase, uidOtherKind)
}

func TestCalculateClusterWideUID_onlyAlphanumericOrX(t *testing.T) {
	obj := &metav1.PartialObjectMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "resource-with-symbols",
		},
	}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group: "grafana.app", Version: "v1", Kind: "Resource",
	})

	uid := CalculateClusterWideUID(obj)
	for _, r := range string(uid) {
		require.True(t,
			unicode.IsLetter(r) || unicode.IsDigit(r) || r == 'X',
			"unexpected rune %q in UID %q", r, uid,
		)
	}
}

func TestCalculateClusterWideUID_withoutObjectMetaUsesGVKOnly(t *testing.T) {
	obj := &objectWithoutMeta{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "test.grafana.app/v1",
			Kind:       "NoMeta",
		},
	}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group: "test.grafana.app", Version: "v1", Kind: "NoMeta",
	})

	uidWithGVK := CalculateClusterWideUID(obj)
	require.NotEmpty(t, uidWithGVK)

	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group: "other.grafana.app", Version: "v1", Kind: "NoMeta",
	})
	uidOtherGroup := CalculateClusterWideUID(obj)
	require.NotEqual(t, uidWithGVK, uidOtherGroup)
}
