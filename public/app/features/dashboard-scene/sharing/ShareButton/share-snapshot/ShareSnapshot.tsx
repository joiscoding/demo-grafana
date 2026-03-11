import { useEffect, useState } from 'react';
import useAsyncFn from 'react-use/lib/useAsyncFn';

import { selectors as e2eSelectors } from '@grafana/e2e-selectors';
import { Trans, t } from '@grafana/i18n';
import { config } from '@grafana/runtime';
import { SceneComponentProps } from '@grafana/scenes';
import { Alert, Button, ClipboardButton, Spinner, Stack, TextLink } from '@grafana/ui';
import { contextSrv } from 'app/core/services/context_srv';
import { AccessControlAction } from 'app/types/accessControl';

import { SnapshotSharingOptions } from '../../../../dashboard/services/SnapshotSrv';
import { ShareDrawerConfirmAction } from '../../ShareDrawer/ShareDrawerConfirmAction';
import { ShareSnapshotTab } from '../../ShareSnapshotTab';
import { ShareView } from '../../types';

import { UpsertSnapshot } from './UpsertSnapshot';

const selectors = e2eSelectors.pages.ShareDashboardDrawer.ShareSnapshot;
const SNAPSHOT_LINK_STORAGE_KEY = 'grafana.share.snapshot.latest';

interface PersistedSnapshotLinkState {
  storageId: string;
  key: string;
  url: string;
}

interface SnapshotLinkState {
  key: string;
  url: string;
}

const isPersistedSnapshotLinkState = (value: unknown): value is PersistedSnapshotLinkState => {
  if (!value || typeof value !== 'object') {
    return false;
  }

  return (
    typeof Reflect.get(value, 'storageId') === 'string' &&
    typeof Reflect.get(value, 'key') === 'string' &&
    typeof Reflect.get(value, 'url') === 'string'
  );
};

export class ShareSnapshot extends ShareSnapshotTab implements ShareView {
  static Component = ShareSnapshotRenderer;

  public getTabLabel() {
    return t('share-dashboard.menu.share-snapshot-title', 'Share snapshot');
  }
}

function ShareSnapshotRenderer({ model }: SceneComponentProps<ShareSnapshot>) {
  const [showDeleteConfirmation, setShowDeleteConfirmation] = useState(false);
  const [showDeletedAlert, setShowDeletedAlert] = useState(false);
  const [step, setStep] = useState(1);

  const { snapshotName, snapshotSharingOptions, selectedExpireOption, panelRef, onDismiss, dashboardRef } =
    model.useState();
  const storageID = `${dashboardRef.resolve().state.uid || 'dashboard'}:${panelRef?.resolve()?.getPathId() || 'all'}`;
  const [activeSnapshot, setActiveSnapshot] = useState<SnapshotLinkState | undefined>();

  const persistSnapshotLinkState = (snapshotState: PersistedSnapshotLinkState) => {
    try {
      window.sessionStorage.setItem(SNAPSHOT_LINK_STORAGE_KEY, JSON.stringify(snapshotState));
    } catch {
      // ignore storage errors
    }
  };

  const getPersistedSnapshotLinkState = (): PersistedSnapshotLinkState | undefined => {
    try {
      const raw = window.sessionStorage.getItem(SNAPSHOT_LINK_STORAGE_KEY);
      if (!raw) {
        return undefined;
      }
      const parsed = JSON.parse(raw);
      if (!isPersistedSnapshotLinkState(parsed)) {
        return undefined;
      }
      return parsed;
    } catch {
      return undefined;
    }
  };

  const clearPersistedSnapshotLinkState = () => {
    try {
      window.sessionStorage.removeItem(SNAPSHOT_LINK_STORAGE_KEY);
    } catch {
      // ignore storage errors
    }
  };

  useEffect(() => {
    const persisted = getPersistedSnapshotLinkState();
    if (!persisted || persisted.storageId !== storageID) {
      return;
    }

    setActiveSnapshot({ key: persisted.key, url: persisted.url });
    setStep(2);
  }, [storageID]);

  const [snapshotResult, createSnapshot] = useAsyncFn(async (external = false) => {
    const response = await model.onSnapshotCreate(external);
    setActiveSnapshot({ key: response.key, url: response.url });
    persistSnapshotLinkState({ storageId: storageID, key: response.key, url: response.url });
    setStep(2);
    return response;
  });
  const [deleteSnapshotResult, deleteSnapshot] = useAsyncFn(async (url: string) => {
    const response = await model.onSnapshotDelete(url);
    clearPersistedSnapshotLinkState();
    setActiveSnapshot(undefined);
    setStep(1);
    setShowDeleteConfirmation(false);
    setShowDeletedAlert(true);
    return response;
  });

  const onCancelClick = () => {
    onDismiss?.();
  };

  const reset = () => {
    model.onSnasphotNameChange(dashboardRef.resolve().state.title);
    setStep(1);
  };

  const onDeleteSnapshotClick = async () => {
    if (!activeSnapshot?.key) {
      return;
    }
    await deleteSnapshot(activeSnapshot.key);
    reset();
  };

  if (showDeleteConfirmation) {
    return (
      <ShareDrawerConfirmAction
        title={t('snapshot.share.delete-title', 'Delete snapshot')}
        confirmButtonLabel={t('snapshot.share.delete-button', 'Delete snapshot')}
        onConfirm={onDeleteSnapshotClick}
        onDismiss={() => setShowDeleteConfirmation(false)}
        description={t('snapshot.share.delete-description', 'Are you sure you want to delete this snapshot?')}
        isActionLoading={deleteSnapshotResult.loading}
      />
    );
  }

  return (
    <div data-testid={selectors.container}>
      <>
        {step === 1 && showDeletedAlert && (
          <Alert severity="info" title={''} onRemove={() => setShowDeletedAlert(false)}>
            <Trans i18nKey="snapshot.share.deleted-alert">
              Snapshot deleted. It could take an hour to be cleared from CDN caches.
            </Trans>
          </Alert>
        )}
        <UpsertSnapshot
          name={snapshotName ?? ''}
          selectedExpireOption={selectedExpireOption}
          onNameChange={model.onSnasphotNameChange}
          onExpireChange={model.onExpireChange}
          disableInputs={step === 2}
          panelRef={panelRef}
        >
          <Stack justifyContent="space-between" gap={{ xs: 2 }} direction={{ xs: 'column', xl: 'row' }}>
            {step === 1 ? (
              <CreateSnapshotActions
                onCreateClick={createSnapshot}
                isLoading={snapshotResult.loading}
                onCancelClick={onCancelClick}
                sharingOptions={snapshotSharingOptions}
              />
            ) : (
              step === 2 &&
              activeSnapshot && (
                <UpsertSnapshotActions
                  url={activeSnapshot.url}
                  onDeleteClick={() => setShowDeleteConfirmation(true)}
                  onNewSnapshotClick={reset}
                />
              )
            )}
            <TextLink icon="external-link-alt" href={`${config.appSubUrl || ''}/dashboard/snapshots`} external>
              {t('snapshot.share.view-all-button', 'View all snapshots')}
            </TextLink>
          </Stack>
        </UpsertSnapshot>
      </>
    </div>
  );
}

const CreateSnapshotActions = ({
  isLoading,
  onCreateClick,
  onCancelClick,
  sharingOptions,
}: {
  isLoading: boolean;
  sharingOptions?: SnapshotSharingOptions;
  onCancelClick: () => void;
  onCreateClick: (isExternal?: boolean) => void;
}) => (
  <Stack gap={1} flex={1} direction={{ xs: 'column', sm: 'row' }}>
    <Button
      variant="primary"
      disabled={isLoading}
      onClick={() => onCreateClick()}
      data-testid={selectors.publishSnapshot}
    >
      <Trans i18nKey="snapshot.share.local-button">Publish snapshot</Trans>
    </Button>
    {sharingOptions?.externalEnabled && (
      <Button variant="secondary" disabled={isLoading} onClick={() => onCreateClick(true)}>
        {sharingOptions?.externalSnapshotName}
      </Button>
    )}
    <Button variant="secondary" fill="outline" onClick={onCancelClick}>
      <Trans i18nKey="snapshot.share.cancel-button">Cancel</Trans>
    </Button>
    {isLoading && <Spinner />}
  </Stack>
);

const UpsertSnapshotActions = ({
  url,
  onDeleteClick,
  onNewSnapshotClick,
}: {
  url: string;
  onDeleteClick: () => void;
  onNewSnapshotClick: () => void;
}) => {
  const hasDeletePermission = contextSrv.hasPermission(AccessControlAction.SnapshotsDelete);
  const deleteTooltip = hasDeletePermission
    ? ''
    : t('snapshot.share.delete-permission-tooltip', "You don't have permission to delete snapshots");

  return (
    <Stack justifyContent="flex-start" gap={1} direction={{ xs: 'column', sm: 'row' }}>
      <ClipboardButton
        icon="link"
        variant="primary"
        fill="outline"
        getText={() => url}
        data-testid={selectors.copyUrlButton}
      >
        <Trans i18nKey="snapshot.share.copy-link-button">Copy link</Trans>
      </ClipboardButton>
      <Button
        icon="trash-alt"
        variant="destructive"
        fill="outline"
        onClick={onDeleteClick}
        disabled={!hasDeletePermission}
        tooltip={deleteTooltip}
      >
        <Trans i18nKey="snapshot.share.disable-link-button">Disable link</Trans>
      </Button>
      <Button variant="secondary" fill="solid" onClick={onNewSnapshotClick}>
        <Trans i18nKey="snapshot.share.new-snapshot-button">New snapshot</Trans>
      </Button>
    </Stack>
  );
};
