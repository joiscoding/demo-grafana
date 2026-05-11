package utils

import (
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func makeConfigMap(group, kind, namespace, name string) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	cm.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Version: "v1",
		Kind:    kind,
	})
	return cm
}

func TestCalculateClusterWideUID_Stable(t *testing.T) {
	a := makeConfigMap("g", "K", "ns", "name")
	b := makeConfigMap("g", "K", "ns", "name")
	require.Equal(t, CalculateClusterWideUID(a), CalculateClusterWideUID(b))
}

func TestCalculateClusterWideUID_DifferentInputsDiffer(t *testing.T) {
	base := CalculateClusterWideUID(makeConfigMap("g", "K", "ns", "name"))

	cases := []runtime.Object{
		makeConfigMap("g2", "K", "ns", "name"),
		makeConfigMap("g", "K2", "ns", "name"),
		makeConfigMap("g", "K", "ns2", "name"),
		makeConfigMap("g", "K", "ns", "name2"),
	}
	for _, c := range cases {
		require.NotEqual(t, base, CalculateClusterWideUID(c))
	}
}

func TestCalculateClusterWideUID_OnlyLettersOrDigitsOrX(t *testing.T) {
	uid := string(CalculateClusterWideUID(makeConfigMap("group", "Kind", "ns", "name")))
	require.NotEmpty(t, uid)
	for _, r := range uid {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != 'X' {
			t.Fatalf("uid contains invalid rune %q in %s", r, uid)
		}
	}
}

func TestCalculateClusterWideUID_EmptyObject(t *testing.T) {
	uid := CalculateClusterWideUID(&corev1.ConfigMap{})
	require.NotEmpty(t, uid)
}

// fakeNoMeta intentionally fails meta.Accessor to exercise the err branch.
type fakeNoMeta struct{}

func (fakeNoMeta) GetObjectKind() schema.ObjectKind { return schema.EmptyObjectKind }
func (fakeNoMeta) DeepCopyObject() runtime.Object   { return fakeNoMeta{} }

func TestCalculateClusterWideUID_AccessorError(t *testing.T) {
	uid := CalculateClusterWideUID(fakeNoMeta{})
	require.NotEmpty(t, uid)
}
