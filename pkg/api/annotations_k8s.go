package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	restclient "k8s.io/client-go/rest"

	annotationV0 "github.com/grafana/grafana/apps/annotation/pkg/apis/annotation/v0alpha1"
	"github.com/grafana/grafana/pkg/api/dtos"
	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/annotations"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/dashboards"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/util"
	"github.com/grafana/grafana/pkg/web"
)

// annotationsGVR is the GroupVersionResource served by the annotation App Platform
// installer at pkg/registry/apps/annotation.
var annotationsGVR = schema.GroupVersionResource{
	Group:    "annotation.grafana.app",
	Version:  "v0alpha1",
	Resource: "annotations",
}

// rerouteAnnotationsEnabled reports whether legacy /api/annotations* calls should be
// dispatched to the annotation.grafana.app resource API. The resource must also be
// installed (kubernetesAnnotations or config-based gate).
func (hs *HTTPServer) rerouteAnnotationsEnabled() bool {
	//nolint:staticcheck // not yet migrated to OpenFeature
	if !hs.Features.IsEnabledGlobally(featuremgmt.FlagAnnotationsRerouteLegacyCRUDAPIs) {
		return false
	}
	//nolint:staticcheck // not yet migrated to OpenFeature
	if hs.Features.IsEnabledGlobally(featuremgmt.FlagKubernetesAnnotations) {
		return true
	}
	return hs.Cfg != nil && hs.Cfg.AnnotationAppPlatform.Enabled
}

// wrapK8sOrLegacy returns either the k8s-backed handler or the legacy one, based
// on whether the reroute flag (and its prerequisites) are enabled at route-
// registration time.
func (hs *HTTPServer) wrapK8sOrLegacy(k8sFn, legacyFn func(c *contextmodel.ReqContext) response.Response) web.Handler {
	if hs.rerouteAnnotationsEnabled() {
		return routing.Wrap(k8sFn)
	}
	return routing.Wrap(legacyFn)
}

// Factory helpers selected at startup. Each returns the legacy handler when the
// reroute flag is off and the k8s-backed handler when it is on.

func (hs *HTTPServer) getK8sAnnotationsListHandler() web.Handler {
	return hs.wrapK8sOrLegacy(hs.getAnnotationsViaK8s, hs.GetAnnotations)
}

func (hs *HTTPServer) getK8sAnnotationHandler() web.Handler {
	return hs.wrapK8sOrLegacy(hs.getAnnotationByIDViaK8s, hs.GetAnnotationByID)
}

func (hs *HTTPServer) postK8sAnnotationHandler() web.Handler {
	return hs.wrapK8sOrLegacy(hs.postAnnotationViaK8s, hs.PostAnnotation)
}

func (hs *HTTPServer) putK8sAnnotationHandler() web.Handler {
	return hs.wrapK8sOrLegacy(hs.updateAnnotationViaK8s, hs.UpdateAnnotation)
}

func (hs *HTTPServer) patchK8sAnnotationHandler() web.Handler {
	return hs.wrapK8sOrLegacy(hs.patchAnnotationViaK8s, hs.PatchAnnotation)
}

func (hs *HTTPServer) deleteK8sAnnotationHandler() web.Handler {
	return hs.wrapK8sOrLegacy(hs.deleteAnnotationByIDViaK8s, hs.DeleteAnnotationByID)
}

func (hs *HTTPServer) postK8sGraphiteAnnotationHandler() web.Handler {
	return hs.wrapK8sOrLegacy(hs.postGraphiteAnnotationViaK8s, hs.PostGraphiteAnnotation)
}

func (hs *HTTPServer) postK8sMassDeleteAnnotationsHandler() web.Handler {
	return hs.wrapK8sOrLegacy(hs.massDeleteAnnotationsViaK8s, hs.MassDeleteAnnotations)
}

func (hs *HTTPServer) getK8sAnnotationTagsHandler() web.Handler {
	return hs.wrapK8sOrLegacy(hs.getAnnotationTagsViaK8s, hs.GetAnnotationTags)
}

// annotationsDynamicClient returns a dynamic client pinned to the annotations GVR
// and to the caller's org namespace, plus the rest.Config used to build the
// client so callers can access custom subresources that the dynamic client does
// not expose.
func (hs *HTTPServer) annotationsDynamicClient(c *contextmodel.ReqContext) (dynamic.NamespaceableResourceInterface, *restclient.Config, string, error) {
	cfg := hs.clientConfigProvider.GetDirectRestConfig(c)
	if cfg == nil {
		return nil, nil, "", fmt.Errorf("no rest config available for k8s annotations client")
	}
	client, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create k8s client: %w", err)
	}
	namespace := hs.namespacer(c.GetOrgID())
	return client.Resource(annotationsGVR), cfg, namespace, nil
}

// handleK8sAnnotationError converts a K8s API error to a legacy HTTP response.
func handleK8sAnnotationError(err error, fallback string) response.Response {
	if err == nil {
		return nil
	}
	statusErr := new(k8serrors.StatusError)
	if errors.As(err, &statusErr) {
		code := int(statusErr.Status().Code)
		msg := statusErr.Status().Message
		switch code {
		case http.StatusNotFound:
			return response.Error(http.StatusNotFound, "Annotation not found", nil)
		case http.StatusForbidden:
			return response.Error(http.StatusForbidden, "Access denied to annotation", err)
		case http.StatusBadRequest:
			return response.Error(http.StatusBadRequest, msg, err)
		default:
			if code == 0 {
				code = http.StatusInternalServerError
			}
			return response.Error(code, msg, err)
		}
	}
	return response.Error(http.StatusInternalServerError, fallback, err)
}

func legacyIDToName(id int64) string {
	return fmt.Sprintf("a-%d", id)
}

func nameToLegacyID(name string) (int64, error) {
	if len(name) < 3 || name[:2] != "a-" {
		return 0, fmt.Errorf("invalid annotation name %q", name)
	}
	return strconv.ParseInt(name[2:], 10, 64)
}

// unstructuredToAnnotation converts an unstructured response from the k8s API into
// a typed Annotation.
func unstructuredToAnnotation(u *unstructured.Unstructured) (*annotationV0.Annotation, error) {
	if u == nil {
		return nil, fmt.Errorf("empty unstructured object")
	}
	raw, err := u.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal annotation: %w", err)
	}
	var anno annotationV0.Annotation
	if err := json.Unmarshal(raw, &anno); err != nil {
		return nil, fmt.Errorf("failed to unmarshal annotation: %w", err)
	}
	return &anno, nil
}

func annotationToUnstructured(anno *annotationV0.Annotation) (*unstructured.Unstructured, error) {
	raw, err := json.Marshal(anno)
	if err != nil {
		return nil, err
	}
	u := &unstructured.Unstructured{}
	if err := u.UnmarshalJSON(raw); err != nil {
		return nil, err
	}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   annotationsGVR.Group,
		Version: annotationsGVR.Version,
		Kind:    "Annotation",
	})
	return u, nil
}

// annotationToItemDTO translates an App Platform Annotation into the legacy ItemDTO
// shape so existing HTTP clients observe the same JSON.
func annotationToItemDTO(anno *annotationV0.Annotation) *annotations.ItemDTO {
	if anno == nil {
		return nil
	}
	dto := &annotations.ItemDTO{
		Text: anno.Spec.Text,
		Time: anno.Spec.Time,
		Tags: anno.Spec.Tags,
	}
	if id, err := nameToLegacyID(anno.Name); err == nil {
		dto.ID = id
	}
	if anno.Spec.TimeEnd != nil {
		dto.TimeEnd = *anno.Spec.TimeEnd
	}
	if anno.Spec.DashboardUID != nil && *anno.Spec.DashboardUID != "" {
		uid := *anno.Spec.DashboardUID
		dto.DashboardUID = &uid
	}
	if anno.Spec.PanelID != nil {
		dto.PanelID = *anno.Spec.PanelID
	}
	if !anno.CreationTimestamp.IsZero() {
		dto.Created = anno.CreationTimestamp.UnixMilli()
	}
	return dto
}

// itemToAnnotation builds an Annotation resource from a legacy Item. Used by
// create/update reroute paths.
func itemToAnnotation(item annotations.Item, namespace string) *annotationV0.Annotation {
	anno := &annotationV0.Annotation{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    namespace,
			GenerateName: "a-",
		},
		Spec: annotationV0.AnnotationSpec{
			Text: item.Text,
			Time: item.Epoch,
			Tags: item.Tags,
		},
	}
	if item.ID != 0 {
		anno.Name = legacyIDToName(item.ID)
		anno.GenerateName = ""
	}
	if item.EpochEnd != 0 {
		end := item.EpochEnd
		anno.Spec.TimeEnd = &end
	}
	if item.DashboardUID != "" {
		uid := item.DashboardUID
		anno.Spec.DashboardUID = &uid
	}
	if item.PanelID != 0 {
		pid := item.PanelID
		anno.Spec.PanelID = &pid
	}
	return anno
}

// -- Handlers --

func (hs *HTTPServer) getAnnotationsViaK8s(c *contextmodel.ReqContext) response.Response {
	_, restCfg, namespace, err := hs.annotationsDynamicClient(c)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to create annotations client", err)
	}

	dashboardID := c.QueryInt64("dashboardId")
	dashboardUID := c.Query("dashboardUID")
	if dashboardID != 0 && dashboardUID == "" {
		dq := dashboards.GetDashboardQuery{ID: dashboardID, OrgID: c.GetOrgID()} // nolint:staticcheck
		dqResult, err := hs.DashboardService.GetDashboard(c.Req.Context(), &dq)
		if err != nil {
			return response.Error(http.StatusBadRequest, "Invalid dashboard ID in annotation request", err)
		}
		dashboardUID = dqResult.UID
	}

	params := url.Values{}
	if dashboardUID != "" {
		params.Set("dashboardUID", dashboardUID)
	}
	if panelID := c.QueryInt64("panelId"); panelID != 0 {
		params.Set("panelID", strconv.FormatInt(panelID, 10))
	}
	if from := c.QueryInt64("from"); from != 0 {
		params.Set("from", strconv.FormatInt(from, 10))
	}
	if to := c.QueryInt64("to"); to != 0 {
		params.Set("to", strconv.FormatInt(to, 10))
	}
	limit := c.QueryInt64("limit")
	if limit == 0 {
		limit = defaultAnnotationsLimit
	}
	params.Set("limit", strconv.FormatInt(limit, 10))
	for _, tag := range c.QueryStrings("tags") {
		params.Add("tag", tag)
	}
	if c.QueryBool("matchAny") {
		params.Set("tagsMatchAny", "true")
	}

	// Use the /search custom route so legacy filtering semantics apply.
	body, err := invokeAnnotationCustomRoute(c.Req.Context(), restCfg, namespace, "search", params)
	if err != nil {
		return handleK8sAnnotationError(err, "Failed to get annotations")
	}

	var list annotationV0.AnnotationList
	if err := json.Unmarshal(body, &list); err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to parse annotations response", err)
	}

	items := make([]*annotations.ItemDTO, 0, len(list.Items))
	for i := range list.Items {
		items = append(items, annotationToItemDTO(&list.Items[i]))
	}
	for _, item := range items {
		if item.Email != "" {
			item.AvatarURL = dtos.GetGravatarUrl(hs.Cfg, item.Email)
		}
	}

	return response.JSON(http.StatusOK, items)
}

func (hs *HTTPServer) getAnnotationByIDViaK8s(c *contextmodel.ReqContext) response.Response {
	annotationID, err := strconv.ParseInt(web.Params(c.Req)[":annotationId"], 10, 64)
	if err != nil {
		return response.Error(http.StatusBadRequest, "annotationId is invalid", err)
	}

	client, _, namespace, err := hs.annotationsDynamicClient(c)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to create annotations client", err)
	}

	got, err := client.Namespace(namespace).Get(c.Req.Context(), legacyIDToName(annotationID), metav1.GetOptions{})
	if err != nil {
		return handleK8sAnnotationError(err, "Failed to get annotation")
	}

	anno, err := unstructuredToAnnotation(got)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to decode annotation", err)
	}

	dto := annotationToItemDTO(anno)
	if dto.Email != "" {
		dto.AvatarURL = dtos.GetGravatarUrl(hs.Cfg, dto.Email)
	}
	return response.JSON(http.StatusOK, dto)
}

func (hs *HTTPServer) postAnnotationViaK8s(c *contextmodel.ReqContext) response.Response {
	cmd := dtos.PostAnnotationsCmd{}
	if err := web.Bind(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}

	if cmd.DashboardUID != "" {
		query := dashboards.GetDashboardQuery{OrgID: c.GetOrgID(), UID: cmd.DashboardUID}
		if dash, err := hs.DashboardService.GetDashboard(c.Req.Context(), &query); err == nil {
			cmd.DashboardId = dash.ID
		}
	}
	if cmd.DashboardId != 0 && cmd.DashboardUID == "" {
		query := dashboards.GetDashboardQuery{OrgID: c.GetOrgID(), ID: cmd.DashboardId}
		dash, err := hs.DashboardService.GetDashboard(c.Req.Context(), &query)
		if err != nil {
			return response.Error(http.StatusBadRequest, "Invalid dashboard ID in annotation request", err)
		}
		cmd.DashboardUID = dash.UID
	}

	if canSave, err := hs.canCreateAnnotation(c, cmd.DashboardUID); err != nil || !canSave {
		if err != nil {
			return response.Error(http.StatusInternalServerError, "Error while checking annotation permissions", err)
		}
		return response.Error(http.StatusForbidden, "Access denied to save the annotation", nil)
	}

	if cmd.Text == "" {
		return response.Error(http.StatusBadRequest, "Failed to save annotation", &AnnotationError{"text field should not be empty"})
	}

	client, _, namespace, err := hs.annotationsDynamicClient(c)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to create annotations client", err)
	}

	userID, _ := identity.UserIdentifier(c.GetID())
	item := annotations.Item{
		OrgID:        c.GetOrgID(),
		UserID:       userID,
		DashboardID:  cmd.DashboardId,
		DashboardUID: cmd.DashboardUID,
		PanelID:      cmd.PanelId,
		Epoch:        cmd.Time,
		EpochEnd:     cmd.TimeEnd,
		Text:         cmd.Text,
		Tags:         cmd.Tags,
	}
	anno := itemToAnnotation(item, namespace)
	u, err := annotationToUnstructured(anno)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to encode annotation", err)
	}

	created, err := client.Namespace(namespace).Create(c.Req.Context(), u, metav1.CreateOptions{})
	if err != nil {
		return handleK8sAnnotationError(err, "Failed to save annotation")
	}

	createdAnno, err := unstructuredToAnnotation(created)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to decode annotation", err)
	}
	id, _ := nameToLegacyID(createdAnno.Name)

	return response.JSON(http.StatusOK, util.DynMap{
		"message": "Annotation added",
		"id":      id,
	})
}

func (hs *HTTPServer) postGraphiteAnnotationViaK8s(c *contextmodel.ReqContext) response.Response {
	cmd := dtos.PostGraphiteAnnotationsCmd{}
	if err := web.Bind(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}
	if cmd.What == "" {
		return response.Error(http.StatusBadRequest, "Failed to save Graphite annotation", &AnnotationError{"what field should not be empty"})
	}

	text := formatGraphiteAnnotation(cmd.What, cmd.Data)

	var tagsArray []string
	switch tags := cmd.Tags.(type) {
	case string:
		if tags != "" {
			tagsArray = strings.Split(tags, " ")
		} else {
			tagsArray = []string{}
		}
	case []any:
		for _, t := range tags {
			if tagStr, ok := t.(string); ok {
				tagsArray = append(tagsArray, tagStr)
			} else {
				return response.Error(http.StatusBadRequest, "Failed to save Graphite annotation", &AnnotationError{"tag should be a string"})
			}
		}
	default:
		return response.Error(http.StatusBadRequest, "Failed to save Graphite annotation", &AnnotationError{"unsupported tags format"})
	}

	client, _, namespace, err := hs.annotationsDynamicClient(c)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to create annotations client", err)
	}

	item := annotations.Item{
		OrgID: c.GetOrgID(),
		Epoch: cmd.When * 1000,
		Text:  text,
		Tags:  tagsArray,
	}
	anno := itemToAnnotation(item, namespace)
	u, err := annotationToUnstructured(anno)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to encode annotation", err)
	}

	created, err := client.Namespace(namespace).Create(c.Req.Context(), u, metav1.CreateOptions{})
	if err != nil {
		return handleK8sAnnotationError(err, "Failed to save Graphite annotation")
	}
	createdAnno, err := unstructuredToAnnotation(created)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to decode annotation", err)
	}
	id, _ := nameToLegacyID(createdAnno.Name)

	return response.JSON(http.StatusOK, util.DynMap{
		"message": "Graphite annotation added",
		"id":      id,
	})
}

func (hs *HTTPServer) updateAnnotationViaK8s(c *contextmodel.ReqContext) response.Response {
	cmd := dtos.UpdateAnnotationsCmd{}
	if err := web.Bind(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}
	annotationID, err := strconv.ParseInt(web.Params(c.Req)[":annotationId"], 10, 64)
	if err != nil {
		return response.Error(http.StatusBadRequest, "annotationId is invalid", err)
	}

	client, _, namespace, err := hs.annotationsDynamicClient(c)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to create annotations client", err)
	}

	name := legacyIDToName(annotationID)
	existingU, err := client.Namespace(namespace).Get(c.Req.Context(), name, metav1.GetOptions{})
	if err != nil {
		return handleK8sAnnotationError(err, "Failed to find annotation")
	}
	existing, err := unstructuredToAnnotation(existingU)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to decode annotation", err)
	}

	existing.Spec.Text = cmd.Text
	existing.Spec.Tags = cmd.Tags
	existing.Spec.Time = cmd.Time
	if cmd.TimeEnd != 0 {
		end := cmd.TimeEnd
		existing.Spec.TimeEnd = &end
	} else {
		existing.Spec.TimeEnd = nil
	}

	updated, err := annotationToUnstructured(existing)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to encode annotation", err)
	}
	if _, err := client.Namespace(namespace).Update(c.Req.Context(), updated, metav1.UpdateOptions{}); err != nil {
		return handleK8sAnnotationError(err, "Failed to update annotation")
	}
	return response.Success("Annotation updated")
}

func (hs *HTTPServer) patchAnnotationViaK8s(c *contextmodel.ReqContext) response.Response {
	cmd := dtos.PatchAnnotationsCmd{}
	if err := web.Bind(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}
	annotationID, err := strconv.ParseInt(web.Params(c.Req)[":annotationId"], 10, 64)
	if err != nil {
		return response.Error(http.StatusBadRequest, "annotationId is invalid", err)
	}

	client, _, namespace, err := hs.annotationsDynamicClient(c)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to create annotations client", err)
	}

	name := legacyIDToName(annotationID)
	existingU, err := client.Namespace(namespace).Get(c.Req.Context(), name, metav1.GetOptions{})
	if err != nil {
		return handleK8sAnnotationError(err, "Failed to find annotation")
	}
	existing, err := unstructuredToAnnotation(existingU)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to decode annotation", err)
	}

	if cmd.Tags != nil {
		existing.Spec.Tags = cmd.Tags
	}
	if cmd.Text != "" && cmd.Text != existing.Spec.Text {
		existing.Spec.Text = cmd.Text
	}
	if cmd.Time > 0 && cmd.Time != existing.Spec.Time {
		existing.Spec.Time = cmd.Time
	}
	var currentTimeEnd int64
	if existing.Spec.TimeEnd != nil {
		currentTimeEnd = *existing.Spec.TimeEnd
	}
	if cmd.TimeEnd > 0 && cmd.TimeEnd != currentTimeEnd {
		end := cmd.TimeEnd
		existing.Spec.TimeEnd = &end
	}

	updated, err := annotationToUnstructured(existing)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to encode annotation", err)
	}
	if _, err := client.Namespace(namespace).Update(c.Req.Context(), updated, metav1.UpdateOptions{}); err != nil {
		return handleK8sAnnotationError(err, "Failed to update annotation")
	}
	return response.Success("Annotation patched")
}

func (hs *HTTPServer) deleteAnnotationByIDViaK8s(c *contextmodel.ReqContext) response.Response {
	annotationID, err := strconv.ParseInt(web.Params(c.Req)[":annotationId"], 10, 64)
	if err != nil {
		return response.Error(http.StatusBadRequest, "annotationId is invalid", err)
	}

	client, _, namespace, err := hs.annotationsDynamicClient(c)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to create annotations client", err)
	}

	if err := client.Namespace(namespace).Delete(c.Req.Context(), legacyIDToName(annotationID), metav1.DeleteOptions{}); err != nil {
		return handleK8sAnnotationError(err, "Failed to delete annotation")
	}
	return response.Success("Annotation deleted")
}

func (hs *HTTPServer) massDeleteAnnotationsViaK8s(c *contextmodel.ReqContext) response.Response {
	cmd := dtos.MassDeleteAnnotationsCmd{}
	if err := web.Bind(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}

	if cmd.DashboardUID != "" {
		query := dashboards.GetDashboardQuery{OrgID: c.GetOrgID(), UID: cmd.DashboardUID}
		if dash, err := hs.DashboardService.GetDashboard(c.Req.Context(), &query); err == nil {
			cmd.DashboardId = dash.ID
		}
	}
	if cmd.DashboardId != 0 && cmd.DashboardUID == "" {
		query := dashboards.GetDashboardQuery{OrgID: c.GetOrgID(), ID: cmd.DashboardId}
		dash, err := hs.DashboardService.GetDashboard(c.Req.Context(), &query)
		if err != nil {
			return response.Error(http.StatusBadRequest, "Invalid dashboard ID in annotation request", err)
		}
		cmd.DashboardUID = dash.UID
	}

	if (cmd.DashboardId != 0 && cmd.PanelId == 0) || (cmd.PanelId != 0 && cmd.DashboardId == 0) {
		return response.Error(http.StatusBadRequest, "bad request data",
			&AnnotationError{message: "DashboardId and PanelId are both required for mass delete"})
	}

	client, restCfg, namespace, err := hs.annotationsDynamicClient(c)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to create annotations client", err)
	}

	var dashboardUID string
	var targetNames []string

	if cmd.AnnotationId != 0 {
		got, err := client.Namespace(namespace).Get(c.Req.Context(), legacyIDToName(cmd.AnnotationId), metav1.GetOptions{})
		if err != nil {
			return handleK8sAnnotationError(err, "Failed to find annotation")
		}
		anno, err := unstructuredToAnnotation(got)
		if err != nil {
			return response.Error(http.StatusInternalServerError, "Failed to decode annotation", err)
		}
		if anno.Spec.DashboardUID != nil {
			dashboardUID = *anno.Spec.DashboardUID
		}
		targetNames = []string{legacyIDToName(cmd.AnnotationId)}
	} else {
		dashboardUID = cmd.DashboardUID
		params := url.Values{}
		params.Set("dashboardUID", cmd.DashboardUID)
		params.Set("panelID", strconv.FormatInt(cmd.PanelId, 10))
		params.Set("limit", "1000")
		body, err := invokeAnnotationCustomRoute(c.Req.Context(), restCfg, namespace, "search", params)
		if err != nil {
			return handleK8sAnnotationError(err, "Failed to look up annotations")
		}
		var list annotationV0.AnnotationList
		if err := json.Unmarshal(body, &list); err != nil {
			return response.Error(http.StatusInternalServerError, "Failed to parse annotations response", err)
		}
		for _, item := range list.Items {
			targetNames = append(targetNames, item.Name)
		}
	}

	canSave, err := hs.canMassDeleteAnnotations(c, dashboardUID)
	if err != nil || !canSave {
		if err != nil {
			return response.Error(http.StatusInternalServerError, "Error while checking annotation permissions", err)
		}
		return response.Error(http.StatusForbidden, "Access denied to mass delete annotations", nil)
	}

	for _, name := range targetNames {
		if err := client.Namespace(namespace).Delete(c.Req.Context(), name, metav1.DeleteOptions{}); err != nil {
			if k8serrors.IsNotFound(err) {
				continue
			}
			return handleK8sAnnotationError(err, "Failed to delete annotations")
		}
	}
	return response.Success("Annotations deleted")
}

func (hs *HTTPServer) getAnnotationTagsViaK8s(c *contextmodel.ReqContext) response.Response {
	_, restCfg, namespace, err := hs.annotationsDynamicClient(c)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to create annotations client", err)
	}

	params := url.Values{}
	if tag := c.Query("tag"); tag != "" {
		params.Set("prefix", tag)
	}
	if limit := c.QueryInt64("limit"); limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}

	body, err := invokeAnnotationCustomRoute(c.Req.Context(), restCfg, namespace, "tags", params)
	if err != nil {
		return handleK8sAnnotationError(err, "Failed to find annotation tags")
	}

	var tagsResp annotationV0.GetTagsResponse
	if err := json.Unmarshal(body, &tagsResp); err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to parse tags response", err)
	}

	result := annotations.FindTagsResult{Tags: make([]*annotations.TagsDTO, 0, len(tagsResp.Tags))}
	for _, t := range tagsResp.Tags {
		tag, count := readTagBody(t)
		result.Tags = append(result.Tags, &annotations.TagsDTO{Tag: tag, Count: count})
	}
	return response.JSON(http.StatusOK, annotations.GetAnnotationTagsResponse{Result: result})
}

// invokeAnnotationCustomRoute dispatches a GET against a namespace-scoped custom
// route on the annotation app (e.g. /search or /tags). The dynamic client does
// not expose these directly, so we issue a raw HTTP GET using the same transport.
func invokeAnnotationCustomRoute(ctx context.Context, cfg *restclient.Config, namespace, route string, params url.Values) ([]byte, error) {
	if cfg == nil {
		return nil, fmt.Errorf("no rest config available")
	}
	httpClient, err := restclient.HTTPClientFor(cfg)
	if err != nil {
		return nil, err
	}
	// The annotation rest config has no Host because requests are handled by an
	// in-process round-tripper; use a placeholder scheme to keep URL parsing
	// happy.
	host := cfg.Host
	if host == "" {
		host = "http://annotations.local"
	}
	u := host + path.Join("/apis", annotationsGVR.Group, annotationsGVR.Version, "namespaces", namespace, route)
	if len(params) > 0 {
		u = u + "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return body, &k8serrors.StatusError{ErrStatus: metav1.Status{
			Code:    int32(resp.StatusCode),
			Message: string(body),
			Reason:  metav1.StatusReasonUnknown,
		}}
	}
	return body, nil
}

// readTagBody extracts the Tag name and Count from a generated tag body struct.
// The SDK generator produces fields with unexported backing, so we round-trip
// through JSON to stay resilient to name changes.
func readTagBody(t annotationV0.GetTagsV0alpha1BodyTags) (string, int64) {
	raw, err := json.Marshal(t)
	if err != nil {
		return "", 0
	}
	aux := struct {
		Tag   string `json:"tag"`
		Count int64  `json:"count"`
	}{}
	_ = json.Unmarshal(raw, &aux)
	return aux.Tag, aux.Count
}

// Ensure accesscontrol import isn't dropped by linter (scopes referenced via
// helpers in annotations.go).
var _ = accesscontrol.ActionAnnotationsRead
