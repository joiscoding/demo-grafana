package builder_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/features"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	featuregatetesting "k8s.io/component-base/featuregate/testing"

	"github.com/grafana/grafana/pkg/services/apiserver/builder"
)

func TestProvideScheme(t *testing.T) {
	s := builder.ProvideScheme()
	require.NotNil(t, s)

	gv := schema.GroupVersion{Group: "", Version: "v1"}
	statusGVK := gv.WithKind("Status")

	obj, err := s.New(statusGVK)
	require.NoError(t, err)
	_, ok := obj.(*metav1.Status)
	require.True(t, ok, "Status should be registered in scheme")

	watchGVK := gv.WithKind("WatchEvent")
	obj2, err := s.New(watchGVK)
	require.NoError(t, err)
	_, ok = obj2.(*metav1.WatchEvent)
	require.True(t, ok)
}

func TestProvideCodecFactory_NoFeatures(t *testing.T) {
	s := builder.ProvideScheme()
	cf := builder.ProvideCodecFactory(s)
	require.NotNil(t, cf)
	require.NotEmpty(t, cf.SupportedMediaTypes())
}

func TestProvideCodecFactory_WithFeatures(t *testing.T) {
	featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.CBORServingAndStorage, true)
	featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.StreamingCollectionEncodingToJSON, true)
	featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.StreamingCollectionEncodingToProtobuf, true)

	s := builder.ProvideScheme()
	cf := builder.ProvideCodecFactory(s)
	require.NotNil(t, cf)
	require.NotEmpty(t, cf.SupportedMediaTypes())
}
