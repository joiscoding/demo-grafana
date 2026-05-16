package storewrapper

import (
	"context"
	"errors"
	"testing"

	"github.com/grafana/authlib/types"
	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/apiserver/rest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

type testSetup struct {
	mockStore *rest.MockStorage
	mockAuth  *FakeAuthorizer
	wrapper   *Wrapper
	ctx       context.Context
}

func newTestSetup(t *testing.T) *testSetup {
	mockStore := rest.NewMockStorage(t)
	mockAuth := &FakeAuthorizer{}
	wrapper := New(mockStore, mockAuth)

	ctx := identity.WithRequester(
		context.Background(),
		&identity.StaticRequester{UserUID: "u001", Type: types.TypeUser},
	)

	return &testSetup{mockStore: mockStore, mockAuth: mockAuth, wrapper: wrapper, ctx: ctx}
}

func matchesOriginalUser() func(context.Context) bool {
	return func(ctx context.Context) bool {
		user, err := identity.GetRequester(ctx)
		return err == nil && user.GetUID() == "user:u001"
	}
}

func matchesServiceIdentity() func(context.Context) bool {
	return func(ctx context.Context) bool {
		return identity.IsServiceIdentity(ctx)
	}
}

func TestWrapper_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		setup := newTestSetup(t)

		obj := &fakeObject{}
		createOpts := &metaV1.CreateOptions{}
		expectedObj := &fakeObject{ObjectMeta: metaV1.ObjectMeta{Name: "created"}}

		// Verify original user identity is used for authorization
		setup.mockAuth.On("BeforeCreate", mock.MatchedBy(matchesOriginalUser()), obj).Return(nil)

		// Verify service identity is used to call the underlying store
		setup.mockStore.On("Create", mock.MatchedBy(matchesServiceIdentity()), obj, mock.Anything, createOpts).Return(expectedObj, nil)

		result, err := setup.wrapper.Create(setup.ctx, obj, nil, createOpts)

		require.NoError(t, err)
		assert.Equal(t, expectedObj, result)

		// Assert expectations
		setup.mockAuth.AssertExpectations(t)
		setup.mockStore.AssertExpectations(t)
	})
	t.Run("unauthorized", func(t *testing.T) {
		setup := newTestSetup(t)

		obj := &fakeObject{}
		createOpts := &metaV1.CreateOptions{}

		// Simulate unauthorized error from authorizer
		setup.mockAuth.On("BeforeCreate", mock.MatchedBy(matchesOriginalUser()), obj).Return(ErrUnauthorized)

		result, err := setup.wrapper.Create(setup.ctx, obj, nil, createOpts)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrUnauthorized, err)

		// Assert expectations
		setup.mockAuth.AssertExpectations(t)
		setup.mockStore.AssertNotCalled(t, "Create")
	})
}

func TestWrapper_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		setup := newTestSetup(t)
		version := "1"
		obj := &fakeObject{ObjectMeta: metaV1.ObjectMeta{Name: "to-delete"}}
		deleteOpts := &metaV1.DeleteOptions{Preconditions: &metaV1.Preconditions{ResourceVersion: &version}}
		expectedObj := &fakeObject{ObjectMeta: metaV1.ObjectMeta{Name: "deleted"}}

		// Mock Get to fetch the object before deletion
		setup.mockStore.On("Get", mock.MatchedBy(matchesServiceIdentity()), "to-delete", mock.Anything).Return(obj, nil)

		// Verify original user identity is used for authorization
		setup.mockAuth.On("BeforeDelete", mock.MatchedBy(matchesOriginalUser()), obj).Return(nil)

		// Verify service identity is used to call the underlying store
		setup.mockStore.On("Delete", mock.MatchedBy(matchesServiceIdentity()), "to-delete", mock.Anything, deleteOpts).Return(expectedObj, true, nil)

		result, deleted, err := setup.wrapper.Delete(setup.ctx, "to-delete", nil, deleteOpts)

		require.NoError(t, err)
		assert.Equal(t, expectedObj, result)
		assert.True(t, deleted)

		// Assert expectations
		setup.mockAuth.AssertExpectations(t)
		setup.mockStore.AssertExpectations(t)
	})
	t.Run("unauthorized", func(t *testing.T) {
		setup := newTestSetup(t)
		version := "1"
		obj := &fakeObject{ObjectMeta: metaV1.ObjectMeta{Name: "to-delete"}}
		deleteOpts := &metaV1.DeleteOptions{Preconditions: &metaV1.Preconditions{ResourceVersion: &version}}

		// Mock Get to fetch the object before deletion
		setup.mockStore.On("Get", mock.MatchedBy(matchesServiceIdentity()), "to-delete", mock.Anything).Return(obj, nil)

		// Simulate unauthorized error from authorizer
		setup.mockAuth.On("BeforeDelete", mock.MatchedBy(matchesOriginalUser()), obj).Return(ErrUnauthorized)

		result, deleted, err := setup.wrapper.Delete(setup.ctx, "to-delete", nil, deleteOpts)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.False(t, deleted)
		assert.Equal(t, ErrUnauthorized, err)

		// Assert expectations
		setup.mockAuth.AssertExpectations(t)
		setup.mockStore.AssertExpectations(t)
		setup.mockStore.AssertNotCalled(t, "Delete")
	})
}

func TestWrapper_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		setup := newTestSetup(t)

		obj := &fakeObject{ObjectMeta: metaV1.ObjectMeta{Name: "fetched"}}

		// Verify service identity is used to call the underlying store
		setup.mockStore.On("Get", mock.MatchedBy(matchesServiceIdentity()), "fetched", mock.Anything).Return(obj, nil)

		// Verify original user identity is used for after-get authorization
		setup.mockAuth.On("AfterGet", mock.MatchedBy(matchesOriginalUser()), obj).Return(nil)

		result, err := setup.wrapper.Get(setup.ctx, "fetched", &metaV1.GetOptions{})

		require.NoError(t, err)
		assert.Equal(t, obj, result)

		// Assert expectations
		setup.mockAuth.AssertExpectations(t)
		setup.mockStore.AssertExpectations(t)
	})
	t.Run("unauthorized", func(t *testing.T) {
		setup := newTestSetup(t)

		obj := &fakeObject{ObjectMeta: metaV1.ObjectMeta{Name: "fetched"}}

		// Verify service identity is used to call the underlying store
		setup.mockStore.On("Get", mock.MatchedBy(matchesServiceIdentity()), "fetched", mock.Anything).Return(obj, nil)

		// Simulate unauthorized error from after-get authorizer
		setup.mockAuth.On("AfterGet", mock.MatchedBy(matchesOriginalUser()), obj).Return(ErrUnauthorized)

		result, err := setup.wrapper.Get(setup.ctx, "fetched", &metaV1.GetOptions{})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrUnauthorized, err)

		// Assert expectations
		setup.mockAuth.AssertExpectations(t)
		setup.mockStore.AssertExpectations(t)
	})
}

func TestWrapper_List(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		setup := newTestSetup(t)

		listObj := &metaV1.List{Items: []runtime.RawExtension{
			{Object: &fakeObject{ObjectMeta: metaV1.ObjectMeta{Name: "item1"}}},
			{Object: &fakeObject{ObjectMeta: metaV1.ObjectMeta{Name: "item2"}}},
		}}

		filteredListObj := &metaV1.List{Items: []runtime.RawExtension{
			{Object: &fakeObject{ObjectMeta: metaV1.ObjectMeta{Name: "item1"}}},
		}}

		// Verify service identity is used to call the underlying store
		setup.mockStore.On("List", mock.MatchedBy(matchesServiceIdentity()), mock.Anything).Return(listObj, nil)

		// Verify original user identity is used for filtering the list
		setup.mockAuth.On("FilterList", mock.MatchedBy(matchesOriginalUser()), listObj).Return(filteredListObj, nil)

		result, err := setup.wrapper.List(setup.ctx, &internalversion.ListOptions{})

		require.NoError(t, err)
		assert.Equal(t, filteredListObj, result)

		// Assert expectations
		setup.mockAuth.AssertExpectations(t)
		setup.mockStore.AssertExpectations(t)
	})
	t.Run("unauthorized", func(t *testing.T) {
		setup := newTestSetup(t)

		listObj := &metaV1.List{Items: []runtime.RawExtension{
			{Object: &fakeObject{ObjectMeta: metaV1.ObjectMeta{Name: "item1"}}},
			{Object: &fakeObject{ObjectMeta: metaV1.ObjectMeta{Name: "item2"}}},
		}}

		// Verify service identity is used to call the underlying store
		setup.mockStore.On("List", mock.MatchedBy(matchesServiceIdentity()), mock.Anything).Return(listObj, nil)

		// Simulate unauthorized error from FilterList authorizer
		setup.mockAuth.On("FilterList", mock.MatchedBy(matchesOriginalUser()), listObj).Return(nil, ErrUnauthorized)

		result, err := setup.wrapper.List(setup.ctx, &internalversion.ListOptions{})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrUnauthorized, err)

		// Assert expectations
		setup.mockAuth.AssertExpectations(t)
		setup.mockStore.AssertExpectations(t)
	})
}

func TestWrapper_Update(t *testing.T) {
	setup := newTestSetup(t)

	oldObj := &fakeObject{ObjectMeta: metaV1.ObjectMeta{
		Name: "to-update", ResourceVersion: "2", Labels: map[string]string{"updated": "false"},
	}}
	objInfo := &fakeUpdatedObjectInfo{obj: oldObj}
	updateOpts := &metaV1.UpdateOptions{}

	var authzInfo *authorizedUpdateInfo

	// Verify service identity is used to call the underlying store
	setup.mockStore.On("Update",
		mock.MatchedBy(matchesServiceIdentity()),
		"to-update",
		mock.MatchedBy(func(info *authorizedUpdateInfo) bool {
			// Capture the authorizedUpdateInfo for later verification
			authzInfo = info
			return true
		}),
		mock.Anything,
		mock.Anything,
		false,
		updateOpts).Return(oldObj, true, nil)

	result, updated, err := setup.wrapper.Update(setup.ctx, "to-update", objInfo, nil, nil, false, updateOpts)
	require.NoError(t, err)
	assert.Equal(t, oldObj, result)
	assert.True(t, updated)

	// Now verify that the authorization is performed inside UpdatedObject
	setup.mockAuth.On("BeforeUpdate", mock.MatchedBy(matchesOriginalUser()), oldObj).Return(nil)
	obj, err := authzInfo.UpdatedObject(context.Background(), oldObj)
	require.NoError(t, err)
	assert.Equal(t, oldObj, obj)

	// Assert expectations
	setup.mockAuth.AssertExpectations(t)
	setup.mockStore.AssertExpectations(t)
}

func TestWrapper_Delete_getError(t *testing.T) {
	setup := newTestSetup(t)
	deleteOpts := &metaV1.DeleteOptions{}
	getErr := errors.New("not found")

	setup.mockStore.On("Get", mock.MatchedBy(matchesServiceIdentity()), "missing", mock.Anything).Return(nil, getErr)

	result, deleted, err := setup.wrapper.Delete(setup.ctx, "missing", nil, deleteOpts)

	require.Error(t, err)
	assert.Equal(t, getErr, err)
	assert.Nil(t, result)
	assert.False(t, deleted)
	setup.mockAuth.AssertNotCalled(t, "BeforeDelete")
}

func TestWrapper_Get_storeError(t *testing.T) {
	setup := newTestSetup(t)
	getErr := errors.New("not found")

	setup.mockStore.On("Get", mock.MatchedBy(matchesServiceIdentity()), "missing", mock.Anything).Return(nil, getErr)

	result, err := setup.wrapper.Get(setup.ctx, "missing", &metaV1.GetOptions{})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, getErr, err)
	setup.mockAuth.AssertNotCalled(t, "AfterGet")
}

func TestWrapper_List_storeError(t *testing.T) {
	setup := newTestSetup(t)
	listErr := errors.New("list failed")

	setup.mockStore.On("List", mock.MatchedBy(matchesServiceIdentity()), mock.Anything).Return(nil, listErr)

	result, err := setup.wrapper.List(setup.ctx, &internalversion.ListOptions{})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, listErr, err)
	setup.mockAuth.AssertNotCalled(t, "FilterList")
}

func TestWrapper_Watch(t *testing.T) {
	t.Run("not supported", func(t *testing.T) {
		setup := newTestSetup(t)

		_, err := setup.wrapper.Watch(setup.ctx, &internalversion.ListOptions{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "watch is not supported")
	})

	t.Run("success", func(t *testing.T) {
		mockStore := rest.NewMockStorage(t)
		watchStore := &watchableStorage{MockStorage: mockStore}
		wrapper := New(watchStore, &NoopAuthorizer{})
		ctx := identity.WithRequester(
			context.Background(),
			&identity.StaticRequester{UserUID: "u001", Type: types.TypeUser},
		)
		fakeWatch := watch.NewFake()
		mockStore.On("Watch", mock.MatchedBy(matchesServiceIdentity()), mock.Anything).Return(fakeWatch, nil)

		result, err := wrapper.Watch(ctx, &internalversion.ListOptions{})

		require.NoError(t, err)
		assert.Equal(t, fakeWatch, result)
		mockStore.AssertExpectations(t)
	})
}

func TestWrapper_Update_errors(t *testing.T) {
	t.Run("inner UpdatedObject error", func(t *testing.T) {
		setup := newTestSetup(t)
		updateErr := errors.New("update failed")
		objInfo := &fakeUpdatedObjectInfo{err: updateErr}
		updateOpts := &metaV1.UpdateOptions{}

		var authzInfo *authorizedUpdateInfo
		setup.mockStore.On("Update",
			mock.MatchedBy(matchesServiceIdentity()),
			"obj",
			mock.MatchedBy(func(info *authorizedUpdateInfo) bool {
				authzInfo = info
				return true
			}),
			mock.Anything,
			mock.Anything,
			false,
			updateOpts).Return(nil, false, nil)

		_, _, err := setup.wrapper.Update(setup.ctx, "obj", objInfo, nil, nil, false, updateOpts)
		require.NoError(t, err)

		_, err = authzInfo.UpdatedObject(context.Background(), &fakeObject{})
		require.Error(t, err)
		assert.Equal(t, updateErr, err)
		setup.mockAuth.AssertNotCalled(t, "BeforeUpdate")
	})

	t.Run("unauthorized", func(t *testing.T) {
		setup := newTestSetup(t)
		oldObj := &fakeObject{ObjectMeta: metaV1.ObjectMeta{Name: "obj"}}
		objInfo := &fakeUpdatedObjectInfo{obj: oldObj}
		updateOpts := &metaV1.UpdateOptions{}

		var authzInfo *authorizedUpdateInfo
		setup.mockStore.On("Update",
			mock.MatchedBy(matchesServiceIdentity()),
			"obj",
			mock.MatchedBy(func(info *authorizedUpdateInfo) bool {
				authzInfo = info
				return true
			}),
			mock.Anything,
			mock.Anything,
			false,
			updateOpts).Return(nil, false, nil)

		_, _, err := setup.wrapper.Update(setup.ctx, "obj", objInfo, nil, nil, false, updateOpts)
		require.NoError(t, err)

		setup.mockAuth.On("BeforeUpdate", mock.MatchedBy(matchesOriginalUser()), oldObj).Return(ErrUnauthorized)
		_, err = authzInfo.UpdatedObject(context.Background(), oldObj)
		require.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
	})
}

func TestAuthorizedUpdateInfo_Preconditions(t *testing.T) {
	rv := "1"
	inner := &fakeUpdatedObjectInfo{
		preconditions: &metaV1.Preconditions{ResourceVersion: &rv},
	}
	authzInfo := &authorizedUpdateInfo{inner: inner}

	require.Equal(t, inner.Preconditions(), authzInfo.Preconditions())
}

func TestStoreCtx_withoutUserUID(t *testing.T) {
	mockStore := rest.NewMockStorage(t)
	wrapper := New(mockStore, &NoopAuthorizer{})
	ctx := identity.WithRequester(
		context.Background(),
		&identity.StaticRequester{Type: types.TypeUser},
	)
	obj := &fakeObject{}

	mockStore.On("Create", mock.MatchedBy(matchesServiceIdentity()), obj, mock.Anything, mock.Anything).Return(obj, nil)

	_, err := wrapper.Create(ctx, obj, nil, &metaV1.CreateOptions{})
	require.NoError(t, err)
}

func TestNoopAuthorizer(t *testing.T) {
	authz := &NoopAuthorizer{}
	ctx := context.Background()
	obj := &fakeObject{}
	list := &metaV1.List{}

	require.NoError(t, authz.BeforeCreate(ctx, obj))
	require.NoError(t, authz.BeforeUpdate(ctx, obj))
	require.NoError(t, authz.BeforeDelete(ctx, obj))
	require.NoError(t, authz.AfterGet(ctx, obj))
	filtered, err := authz.FilterList(ctx, list)
	require.NoError(t, err)
	assert.Equal(t, list, filtered)
}

func TestDenyAuthorizer(t *testing.T) {
	authz := &DenyAuthorizer{}
	ctx := context.Background()
	obj := &fakeObject{}
	list := &metaV1.List{}

	require.Equal(t, ErrUnauthorized, authz.BeforeCreate(ctx, obj))
	require.Equal(t, ErrUnauthorized, authz.BeforeUpdate(ctx, obj))
	require.Equal(t, ErrUnauthorized, authz.BeforeDelete(ctx, obj))
	require.Equal(t, ErrUnauthorized, authz.AfterGet(ctx, obj))
	filtered, err := authz.FilterList(ctx, list)
	require.Equal(t, ErrUnauthorized, err)
	assert.Nil(t, filtered)
}

func TestWrapper_DeleteCollection(t *testing.T) {
	setup := newTestSetup(t)

	result, err := setup.wrapper.DeleteCollection(setup.ctx, nil, &metaV1.DeleteOptions{}, &internalversion.ListOptions{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "bulk delete operations are not supported")
	assert.Nil(t, result)
}

func TestWrapper_PassthroughMethods(t *testing.T) {
	setup := newTestSetup(t)

	t.Run("New", func(t *testing.T) {
		obj := &fakeObject{}
		setup.mockStore.On("New").Return(obj).Once()
		assert.Equal(t, obj, setup.wrapper.New())
	})

	t.Run("NewList", func(t *testing.T) {
		obj := &fakeObject{}
		setup.mockStore.On("NewList").Return(obj).Once()
		assert.Equal(t, obj, setup.wrapper.NewList())
	})

	t.Run("GetSingularName", func(t *testing.T) {
		setup.mockStore.On("GetSingularName").Return("fake").Once()
		assert.Equal(t, "fake", setup.wrapper.GetSingularName())
	})

	t.Run("NamespaceScoped", func(t *testing.T) {
		setup.mockStore.On("NamespaceScoped").Return(true).Once()
		assert.True(t, setup.wrapper.NamespaceScoped())
	})

	t.Run("Destroy", func(t *testing.T) {
		setup.mockStore.On("Destroy").Once()
		setup.wrapper.Destroy()
	})

	t.Run("ConvertToTable", func(t *testing.T) {
		obj := &fakeObject{}
		table := &metaV1.Table{}
		setup.mockStore.On("ConvertToTable", setup.ctx, obj, mock.Anything).Return(table, nil).Once()
		result, err := setup.wrapper.ConvertToTable(setup.ctx, obj, nil)
		require.NoError(t, err)
		assert.Equal(t, table, result)
	})

	setup.mockStore.AssertExpectations(t)
}

// -----
// Fakes
// -----

type FakeAuthorizer struct {
	mock.Mock
}

func (f *FakeAuthorizer) BeforeCreate(ctx context.Context, obj runtime.Object) error {
	args := f.Called(ctx, obj)
	return args.Error(0)
}

func (f *FakeAuthorizer) BeforeUpdate(ctx context.Context, obj runtime.Object) error {
	args := f.Called(ctx, obj)
	return args.Error(0)
}

func (f *FakeAuthorizer) BeforeDelete(ctx context.Context, obj runtime.Object) error {
	args := f.Called(ctx, obj)
	return args.Error(0)
}

func (f *FakeAuthorizer) AfterGet(ctx context.Context, obj runtime.Object) error {
	args := f.Called(ctx, obj)
	return args.Error(0)
}

func (f *FakeAuthorizer) FilterList(ctx context.Context, list runtime.Object) (runtime.Object, error) {
	args := f.Called(ctx, list)
	var res runtime.Object
	if args.Get(0) != nil {
		res = args.Get(0).(runtime.Object)
	}
	return res, args.Error(1)
}

type fakeObject struct {
	metaV1.TypeMeta
	metaV1.ObjectMeta
}

func (f *fakeObject) DeepCopyObject() runtime.Object {
	return &fakeObject{
		TypeMeta:   f.TypeMeta,
		ObjectMeta: f.ObjectMeta,
	}
}

// fakeUpdatedObjectInfo implements k8srest.UpdatedObjectInfo for testing
type fakeUpdatedObjectInfo struct {
	obj           runtime.Object
	err           error
	preconditions *metaV1.Preconditions
}

func (f *fakeUpdatedObjectInfo) Preconditions() *metaV1.Preconditions {
	return f.preconditions
}

func (f *fakeUpdatedObjectInfo) UpdatedObject(ctx context.Context, oldObj runtime.Object) (runtime.Object, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.obj, nil
}

type watchableStorage struct {
	*rest.MockStorage
}

func (w *watchableStorage) Watch(ctx context.Context, options *internalversion.ListOptions) (watch.Interface, error) {
	args := w.MockStorage.Mock.MethodCalled("Watch", ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(watch.Interface), args.Error(1)
}
