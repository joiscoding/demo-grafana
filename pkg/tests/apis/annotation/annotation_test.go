package annotation

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	annotationV0 "github.com/grafana/grafana/apps/annotation/pkg/apis/annotation/v0alpha1"
	grafanarest "github.com/grafana/grafana/pkg/apiserver/rest"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/tests/apis"
	"github.com/grafana/grafana/pkg/tests/testinfra"
	"github.com/grafana/grafana/pkg/tests/testsuite"
	"github.com/grafana/grafana/pkg/util/testutil"
)

func TestMain(m *testing.M) {
	testsuite.Run(m)
}

func TestIntegrationAnnotation(t *testing.T) {
	testutil.SkipIntegrationTestInShortMode(t)

	for _, mode := range []grafanarest.DualWriterMode{
		grafanarest.Mode0, // Only legacy for now
	} {
		t.Run(fmt.Sprintf("annotation (mode:%d)", mode), func(t *testing.T) {
			helper := apis.NewK8sTestHelper(t, testinfra.GrafanaOpts{
				DisableAnonymous:     true,
				EnableFeatureToggles: []string{featuremgmt.FlagKubernetesAnnotations},
				UnifiedStorageConfig: map[string]setting.UnifiedStorageConfig{
					"annotations.annotation.grafana.app": {
						DualWriterMode: mode,
					},
				},
			})

			ctx := context.Background()
			kind := annotationV0.AnnotationKind()
			annotationClient := helper.GetResourceClient(apis.ResourceClientArgs{
				User: helper.Org1.Admin,
				GVR:  kind.GroupVersionResource(),
			})

			// Create annotation via k8s API
			now := time.Now().UnixMilli()
			annotation := &annotationV0.Annotation{
				ObjectMeta: v1.ObjectMeta{
					GenerateName: "test-annotation-",
					Namespace:    annotationClient.Args.User.Identity.GetNamespace(),
				},
				Spec: annotationV0.AnnotationSpec{
					Text: "Test annotation created via k8s API",
					Time: now,
					Tags: []string{"test", "k8s"},
				},
			}

			unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(annotation)
			require.NoError(t, err)
			u := &unstructured.Unstructured{Object: unstructuredObj}

			created, err := annotationClient.Resource.Create(ctx, u, v1.CreateOptions{})
			require.NoError(t, err, "create annotation")
			require.NotEmpty(t, created.GetName(), "annotation name")

			// List annotations
			listResults, err := annotationClient.Resource.List(ctx, v1.ListOptions{})
			require.NoError(t, err, "list annotations")
			require.GreaterOrEqual(t, len(listResults.Items), 1, "should have at least one annotation")

			// Get the created annotation
			getResult, err := annotationClient.Resource.Get(ctx, created.GetName(), v1.GetOptions{})
			require.NoError(t, err, "get annotation")
			require.Equal(t, created.GetName(), getResult.GetName())

			// Update annotation
			spec, found, err := unstructured.NestedMap(getResult.Object, "spec")
			require.NoError(t, err)
			require.True(t, found, "spec should exist")
			spec["text"] = "Updated annotation text"
			err = unstructured.SetNestedMap(getResult.Object, spec, "spec")
			require.NoError(t, err)

			updated, err := annotationClient.Resource.Update(ctx, getResult, v1.UpdateOptions{})
			require.NoError(t, err, "update annotation")
			require.Equal(t, created.GetName(), updated.GetName())

			// Delete annotation
			err = annotationClient.Resource.Delete(ctx, created.GetName(), v1.DeleteOptions{})
			require.NoError(t, err, "delete annotation")
		})
	}
}
