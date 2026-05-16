package authorizer

import (
	"context"
	"errors"

	authnlib "github.com/grafana/authlib/authn"
	"github.com/grafana/authlib/types"
	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/services/org"
	k8suser "k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
)

func ctxWithRequester(req *identity.StaticRequester) context.Context {
	return identity.WithRequester(context.Background(), req)
}

func ctxWithAuthInfo(req *identity.StaticRequester) context.Context {
	return types.WithAuthInfo(context.Background(), req)
}

func testRequester(opts func(*identity.StaticRequester)) *identity.StaticRequester {
	r := &identity.StaticRequester{
		Type:      types.TypeUser,
		UserID:    1,
		UserUID:   "u001",
		OrgID:     2,
		Namespace: "org-2",
		OrgRole:   org.RoleEditor,
	}
	if opts != nil {
		opts(r)
	}
	return r
}

func resourceAttrs(opts func(*authorizer.AttributesRecord)) authorizer.Attributes {
	attrs := &authorizer.AttributesRecord{
		User:            &k8suser.DefaultInfo{Name: "test"},
		Verb:            "get",
		Namespace:       "org-2",
		APIGroup:        "folder.grafana.app",
		APIVersion:      "v1beta1",
		Resource:        "folders",
		ResourceRequest: true,
		Path:            "/apis/folder.grafana.app/v1beta1/namespaces/org-2/folders",
	}
	if opts != nil {
		opts(attrs)
	}
	return attrs
}

type mockAccessChecker struct {
	checkFunc func(ctx context.Context, ident types.AuthInfo, req types.CheckRequest, folder string) (types.CheckResponse, error)
}

func (m *mockAccessChecker) Check(ctx context.Context, ident types.AuthInfo, req types.CheckRequest, folder string) (types.CheckResponse, error) {
	if m.checkFunc != nil {
		return m.checkFunc(ctx, ident, req, folder)
	}
	return types.CheckResponse{Allowed: true}, nil
}

func accessPolicyWithPermissions(perms ...string) *identity.StaticRequester {
	return &identity.StaticRequester{
		Type: types.TypeAccessPolicy,
		AccessTokenClaims: &authnlib.Claims[authnlib.AccessTokenClaims]{
			Rest: authnlib.AccessTokenClaims{
				Permissions: perms,
			},
		},
	}
}

func userWithDelegatedPermissions(perms ...string) *identity.StaticRequester {
	return &identity.StaticRequester{
		Type: types.TypeUser,
		AccessTokenClaims: &authnlib.Claims[authnlib.AccessTokenClaims]{
			Rest: authnlib.AccessTokenClaims{
				DelegatedPermissions: perms,
			},
		},
	}
}

var errCheckFailed = errors.New("access check failed")
