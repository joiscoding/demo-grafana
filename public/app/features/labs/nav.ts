import { NavLinkDTO } from '@grafana/data';
import { t } from '@grafana/i18n';
import config from 'app/core/config';
import { contextSrv } from 'app/core/services/context_srv';

export const LABS_NAV_ID = 'labs';
const ADMIN_NAV_ID = 'cfg';

export function getLabsNavItem(): NavLinkDTO {
  return {
    id: LABS_NAV_ID,
    text: t('labs-page.title', 'Labs'),
    subTitle: t('labs-page.sidebar-subtitle', 'Inspect the current state of Grafana feature flags'),
    icon: 'adjust-circle',
    url: `${config.appSubUrl}/labs`,
  };
}

export function withLabsNavItem(navTree: NavLinkDTO[]): NavLinkDTO[] {
  const isAdmin = contextSrv.hasRole('Admin') || contextSrv.isGrafanaAdmin;
  if (!config.bootData.user.isSignedIn || !isAdmin || navTree.some((item) => item.id === LABS_NAV_ID)) {
    return navTree;
  }

  const labsNav = getLabsNavItem();
  const adminIndex = navTree.findIndex((item) => item.id === ADMIN_NAV_ID);

  if (adminIndex === -1) {
    return [...navTree, labsNav];
  }

  return [...navTree.slice(0, adminIndex), labsNav, ...navTree.slice(adminIndex)];
}
