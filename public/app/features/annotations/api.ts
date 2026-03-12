import { AnnotationEvent, DataFrame, toDataFrame } from '@grafana/data';
import { config, getBackendSrv } from '@grafana/runtime';
import { getAPIBaseURL } from 'app/api/utils';
import { StateHistoryItem } from 'app/types/unified-alerting';

import { AnnotationTagsResponse } from './types';

export interface AnnotationServer {
  query(params: Record<string, unknown>, requestId: string): Promise<DataFrame>;
  forAlert(alertUID: string): Promise<StateHistoryItem[]>;
  save(annotation: AnnotationEvent): Promise<AnnotationEvent>;
  update(annotation: AnnotationEvent): Promise<unknown>;
  delete(annotation: AnnotationEvent): Promise<unknown>;
  tags(): Promise<Array<{ term: string; count: number }>>;
}

class LegacyAnnotationServer implements AnnotationServer {
  query(params: unknown, requestId: string): Promise<DataFrame> {
    return getBackendSrv()
      .get('/api/annotations', params, requestId)
      .then((v) => toDataFrame(v));
  }

  forAlert(alertUID: string) {
    return getBackendSrv().get('/api/annotations', {
      alertUID,
    });
  }

  save(annotation: AnnotationEvent) {
    return getBackendSrv().post('/api/annotations', annotation);
  }

  update(annotation: AnnotationEvent) {
    return getBackendSrv().put(`/api/annotations/${annotation.id}`, annotation);
  }

  delete(annotation: AnnotationEvent) {
    return getBackendSrv().delete(`/api/annotations/${annotation.id}`);
  }

  async tags() {
    const response = await getBackendSrv().get<AnnotationTagsResponse>('/api/annotations/tags?limit=1000');
    return response.result.tags.map(({ tag, count }) => ({
      term: tag,
      count,
    }));
  }
}

const ANNOTATIONS_API_BASE_URL = getAPIBaseURL('annotation.grafana.app', 'v0alpha1');
const ANNOTATION_DEFAULT_LIMIT = 100;

interface AnnotationResource {
  metadata: {
    name?: string;
    namespace?: string;
    resourceVersion?: string;
    generateName?: string;
  };
  spec: {
    dashboardUID?: string;
    panelID?: number;
    tags?: string[];
    text: string;
    time: number;
    timeEnd?: number;
  };
}

interface AnnotationListResponse {
  items: AnnotationResource[];
}

interface AnnotationTagsV0Response {
  tags: Array<{
    tag: string;
    count: number;
  }>;
}

type QueryParams = Record<string, unknown>;

function hasUnsupportedKubernetesQueryParams(params: QueryParams) {
  return (
    params.alertId !== undefined ||
    params.alertUID !== undefined ||
    params.dashboardId !== undefined ||
    params.userId !== undefined ||
    params.tags !== undefined ||
    params.matchAny !== undefined ||
    params.type !== undefined
  );
}

function buildFieldSelector(params: QueryParams) {
  const selectors: string[] = [];

  if (typeof params.dashboardUID === 'string' && params.dashboardUID.length > 0) {
    selectors.push(`spec.dashboardUID=${params.dashboardUID}`);
  }

  if (typeof params.panelId === 'number' && Number.isFinite(params.panelId) && params.panelId > 0) {
    selectors.push(`spec.panelID=${params.panelId}`);
  }

  if (typeof params.from === 'number' && Number.isFinite(params.from) && params.from > 0) {
    selectors.push(`spec.time=${params.from}`);
  }

  if (typeof params.to === 'number' && Number.isFinite(params.to) && params.to > 0) {
    selectors.push(`spec.timeEnd=${params.to}`);
  }

  return selectors.length > 0 ? selectors.join(',') : undefined;
}

function parseAnnotationName(name?: string): string | undefined {
  if (!name) {
    return undefined;
  }

  if (name.startsWith('a-')) {
    return name.slice(2);
  }

  return name;
}

function toAnnotationName(id: AnnotationEvent['id']): string {
  if (!id) {
    throw new Error('Annotation id is required');
  }

  const asString = String(id);
  return asString.startsWith('a-') ? asString : `a-${asString}`;
}

function toAnnotationResource(annotation: AnnotationEvent): AnnotationResource {
  const metadata: AnnotationResource['metadata'] = {};
  if (annotation.id) {
    metadata.name = toAnnotationName(annotation.id);
  } else {
    metadata.generateName = 'a-';
  }

  const spec: AnnotationResource['spec'] = {
    text: annotation.text ?? '',
    time: annotation.time ?? Date.now(),
    tags: annotation.tags ?? [],
  };

  if (annotation.dashboardUID) {
    spec.dashboardUID = annotation.dashboardUID;
  }

  if (annotation.panelId !== undefined) {
    spec.panelID = annotation.panelId;
  }

  if (annotation.timeEnd !== undefined && annotation.timeEnd > 0) {
    spec.timeEnd = annotation.timeEnd;
  }

  return {
    metadata,
    spec,
  };
}

function fromAnnotationResource(resource: AnnotationResource): AnnotationEvent {
  return {
    id: parseAnnotationName(resource.metadata.name),
    dashboardUID: resource.spec.dashboardUID ?? null,
    panelId: resource.spec.panelID,
    tags: resource.spec.tags ?? [],
    text: resource.spec.text,
    time: resource.spec.time,
    timeEnd: resource.spec.timeEnd,
  };
}

class KubernetesAnnotationServer implements AnnotationServer {
  constructor(private readonly fallback: AnnotationServer) {}

  async query(params: QueryParams, requestId: string): Promise<DataFrame> {
    if (hasUnsupportedKubernetesQueryParams(params)) {
      return this.fallback.query(params, requestId);
    }

    const fieldSelector = buildFieldSelector(params);
    const limit =
      typeof params.limit === 'number' && Number.isFinite(params.limit) && params.limit > 0
        ? params.limit
        : ANNOTATION_DEFAULT_LIMIT;
    const response = await getBackendSrv().get<AnnotationListResponse>(
      `${ANNOTATIONS_API_BASE_URL}/annotations`,
      {
        fieldSelector,
        limit,
      },
      requestId
    );

    return toDataFrame(response.items.map(fromAnnotationResource));
  }

  forAlert(alertUID: string): Promise<StateHistoryItem[]> {
    return this.fallback.forAlert(alertUID);
  }

  async save(annotation: AnnotationEvent): Promise<AnnotationEvent> {
    const response = await getBackendSrv().post<AnnotationResource>(
      `${ANNOTATIONS_API_BASE_URL}/annotations`,
      toAnnotationResource(annotation)
    );
    return fromAnnotationResource(response);
  }

  async update(annotation: AnnotationEvent): Promise<unknown> {
    const name = toAnnotationName(annotation.id);
    const existing = await getBackendSrv().get<AnnotationResource>(`${ANNOTATIONS_API_BASE_URL}/annotations/${name}`);
    const updateBody: AnnotationResource = {
      ...existing,
      spec: {
        ...existing.spec,
        ...toAnnotationResource(annotation).spec,
      },
    };
    return getBackendSrv().put(`${ANNOTATIONS_API_BASE_URL}/annotations/${name}`, updateBody);
  }

  delete(annotation: AnnotationEvent): Promise<unknown> {
    const name = toAnnotationName(annotation.id);
    return getBackendSrv().delete(`${ANNOTATIONS_API_BASE_URL}/annotations/${name}`);
  }

  async tags() {
    const response = await getBackendSrv().get<AnnotationTagsV0Response>(`${ANNOTATIONS_API_BASE_URL}/tags`, {
      limit: 1000,
    });

    return response.tags.map(({ tag, count }) => ({
      term: tag,
      count,
    }));
  }
}

let instance: AnnotationServer | null = null;

export function annotationServer(): AnnotationServer {
  if (!instance) {
    const legacy = new LegacyAnnotationServer();
    instance = config.featureToggles.kubernetesAnnotations ? new KubernetesAnnotationServer(legacy) : legacy;
  }
  return instance;
}
