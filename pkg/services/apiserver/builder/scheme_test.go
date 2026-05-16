package builder_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/grafana/grafana/pkg/services/apiserver/builder"
)

func TestProvideScheme(t *testing.T) {
	t.Parallel()

	scheme := builder.ProvideScheme()
	require.NotNil(t, scheme)

	gv := schema.GroupVersion{Group: "", Version: "v1"}
	kinds, _, err := scheme.ObjectKinds(&metav1.Status{})
	require.NoError(t, err)
	require.NotEmpty(t, kinds)

	for _, kind := range []string{"Status", "WatchEvent", "APIVersions", "APIGroupList", "APIGroup", "APIResourceList", "PartialObjectMetadata", "PartialObjectMetadataList"} {
		gvk := gv.WithKind(kind)
		require.True(t, scheme.Recognizes(gvk), "expected scheme to recognize %s", gvk)
	}
}

func TestProvideCodecFactory(t *testing.T) {
	t.Parallel()

	scheme := builder.ProvideScheme()
	codecs := builder.ProvideCodecFactory(scheme)
	require.NotNil(t, codecs)

	mediaTypes := codecs.SupportedMediaTypes()
	require.NotEmpty(t, mediaTypes)
	require.NotNil(t, mediaTypes[0].Serializer)
}
