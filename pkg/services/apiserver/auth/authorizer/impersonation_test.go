package authorizer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	k8suser "k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
)

func TestImpersonationAuthorizer_Authorize(t *testing.T) {
	auth := impersonationAuthorizer{}

	t.Run("impersonate verb is denied", func(t *testing.T) {
		attrs := &fakeAttributes{verb: "impersonate"}

		authorized, reason, err := auth.Authorize(context.Background(), attrs)

		require.Equal(t, authorizer.DecisionDeny, authorized)
		require.Equal(t, "user impersonation is not supported", reason)
		require.NoError(t, err)
	})

	t.Run("other verbs return no opinion", func(t *testing.T) {
		for _, verb := range []string{"get", "list", "watch", "create", "update", "patch", "delete", ""} {
			attrs := &fakeAttributes{verb: verb}

			authorized, reason, err := auth.Authorize(context.Background(), attrs)

			require.Equal(t, authorizer.DecisionNoOpinion, authorized, "verb=%q", verb)
			require.Equal(t, "", reason, "verb=%q", verb)
			require.NoError(t, err, "verb=%q", verb)
		}
	})

	t.Run("constructor returns non-nil authorizer", func(t *testing.T) {
		require.NotNil(t, NewImpersonationAuthorizer())
	})
}

// fakeAttributes is a test helper implementing authorizer.Attributes by
// embedding the interface (so any unset method panics, surfacing accidental
// usage) and overriding the getters used by the authorizers under test.
type fakeAttributes struct {
	authorizer.Attributes
	verb              string
	apiGroup          string
	apiVersion        string
	namespace         string
	resource          string
	subresource       string
	name              string
	path              string
	isResourceRequest bool
	user              k8suser.Info
}

func (a fakeAttributes) GetVerb() string         { return a.verb }
func (a fakeAttributes) GetAPIGroup() string     { return a.apiGroup }
func (a fakeAttributes) GetAPIVersion() string   { return a.apiVersion }
func (a fakeAttributes) GetNamespace() string    { return a.namespace }
func (a fakeAttributes) GetResource() string     { return a.resource }
func (a fakeAttributes) GetSubresource() string  { return a.subresource }
func (a fakeAttributes) GetName() string         { return a.name }
func (a fakeAttributes) GetPath() string         { return a.path }
func (a fakeAttributes) IsResourceRequest() bool { return a.isResourceRequest }
func (a fakeAttributes) GetUser() k8suser.Info {
	if a.user == nil {
		return &k8suser.DefaultInfo{}
	}
	return a.user
}
