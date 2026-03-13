package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/services/org/orgtest"
	tempuser "github.com/grafana/grafana/pkg/services/temp_user"
	"github.com/grafana/grafana/pkg/services/temp_user/tempusertest"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/services/user/usertest"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/web/webtest"
)

func TestOrgInvitesAPIEndpoint_RBAC(t *testing.T) {
	type testCase struct {
		desc         string
		body         string
		permissions  []accesscontrol.Permission
		expectedCode int
	}

	tests := []testCase{
		{
			desc: "should be able to invite user to org with correct permissions",
			body: `{"loginOrEmail": "new user", "role": "Viewer"}`,
			permissions: []accesscontrol.Permission{
				{Action: accesscontrol.ActionOrgUsersAdd, Scope: "users:id:1"},
			},
			expectedCode: http.StatusOK,
		},
		{
			desc:         "should not be able to invite user to org without correct permissions",
			body:         `{"loginOrEmail": "new user", "role": "Viewer"}`,
			permissions:  []accesscontrol.Permission{},
			expectedCode: http.StatusForbidden,
		},
		{
			desc: "should not be able to invite user to org with wrong scope",
			body: `{"loginOrEmail": "new user", "role": "Viewer"}`,
			permissions: []accesscontrol.Permission{
				{Action: accesscontrol.ActionOrgUsersAdd, Scope: "users:id:2"},
			},
			expectedCode: http.StatusForbidden,
		},
		{
			desc: "should not be able to invite user to org with higher role then requester",
			body: `{"loginOrEmail": "new user", "role": "Admin"}`,
			permissions: []accesscontrol.Permission{
				{Action: accesscontrol.ActionOrgUsersAdd, Scope: "users:id:1"},
			},
			expectedCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			server := SetupAPITestServer(t, func(hs *HTTPServer) {
				hs.Cfg = setting.NewCfg()
				hs.orgService = orgtest.NewOrgServiceFake()
				hs.userService = &usertest.FakeUserService{
					ExpectedUser: &user.User{ID: 1},
				}
			})

			req := webtest.RequestWithSignedInUser(server.NewPostRequest("/api/org/invites", strings.NewReader(tt.body)), userWithPermissions(1, tt.permissions))
			res, err := server.SendJSON(req)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCode, res.StatusCode)
			require.NoError(t, res.Body.Close())
		})
	}
}

func TestGetInviteInfoByCode(t *testing.T) {
	t.Run("should return 410 Gone for expired invites", func(t *testing.T) {
		fakeTempUserService := &tempusertest.FakeTempUserService{
			GetTempUserByCodeFN: func(ctx context.Context, query *tempuser.GetTempUserByCodeQuery) (*tempuser.TempUserDTO, error) {
				return &tempuser.TempUserDTO{
					Email:  "expired@example.com",
					Name:   "Expired User",
					OrgID:  1,
					Status: tempuser.TmpUserExpired,
					Code:   query.Code,
				}, nil
			},
		}

		fakeOrgService := &orgtest.FakeOrgService{
			ExpectedOrg: &org.Org{
				ID:   1,
				Name: "Test Org",
			},
		}

		server := SetupAPITestServer(t, func(hs *HTTPServer) {
			hs.Cfg = setting.NewCfg()
			hs.tempUserService = fakeTempUserService
			hs.orgService = fakeOrgService
		})

		req := server.NewGetRequest("/api/user/invite/expired-code")
		res, err := server.Send(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusGone, res.StatusCode)

		var body map[string]any
		err = json.NewDecoder(res.Body).Decode(&body)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())

		assert.Equal(t, "This invitation has expired", body["message"])
		assert.Equal(t, "Expired", body["status"])
		assert.Equal(t, "expired@example.com", body["email"])
		assert.Equal(t, "Test Org", body["orgName"])
	})

	t.Run("should return 404 for completed invites", func(t *testing.T) {
		fakeTempUserService := &tempusertest.FakeTempUserService{
			GetTempUserByCodeFN: func(ctx context.Context, query *tempuser.GetTempUserByCodeQuery) (*tempuser.TempUserDTO, error) {
				return &tempuser.TempUserDTO{
					Email:  "completed@example.com",
					OrgID:  1,
					Status: tempuser.TmpUserCompleted,
					Code:   query.Code,
				}, nil
			},
		}

		server := SetupAPITestServer(t, func(hs *HTTPServer) {
			hs.Cfg = setting.NewCfg()
			hs.tempUserService = fakeTempUserService
		})

		req := server.NewGetRequest("/api/user/invite/completed-code")
		res, err := server.Send(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		require.NoError(t, res.Body.Close())
	})

	t.Run("should return 200 for pending invites", func(t *testing.T) {
		fakeTempUserService := &tempusertest.FakeTempUserService{
			GetTempUserByCodeFN: func(ctx context.Context, query *tempuser.GetTempUserByCodeQuery) (*tempuser.TempUserDTO, error) {
				return &tempuser.TempUserDTO{
					Email:  "pending@example.com",
					Name:   "Pending User",
					OrgID:  1,
					Status: tempuser.TmpUserInvitePending,
					Code:   query.Code,
				}, nil
			},
		}

		fakeOrgService := &orgtest.FakeOrgService{
			ExpectedOrg: &org.Org{
				ID:   1,
				Name: "Test Org",
			},
		}

		server := SetupAPITestServer(t, func(hs *HTTPServer) {
			hs.Cfg = setting.NewCfg()
			hs.tempUserService = fakeTempUserService
			hs.orgService = fakeOrgService
		})

		req := server.NewGetRequest("/api/user/invite/pending-code")
		res, err := server.Send(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		require.NoError(t, res.Body.Close())
	})
}
