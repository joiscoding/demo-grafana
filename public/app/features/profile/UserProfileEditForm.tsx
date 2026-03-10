import { selectors } from '@grafana/e2e-selectors';
import { useGetUserPreferencesQuery } from '@grafana/api-clients/rtkq/legacy/preferences';
import { Trans, t } from '@grafana/i18n';
import { Button, Field, FieldSet, Icon, Input, Switch, Tooltip } from '@grafana/ui';
import { Controller } from 'react-hook-form';
import { Form } from 'app/core/components/Form/Form';
import config from 'app/core/config';
import { UserDTO } from 'app/types/user';

import { ProfileUpdateFields } from './types';

export interface Props {
  user: UserDTO | null;
  isSavingUser: boolean;
  updateProfile: (payload: ProfileUpdateFields) => void;
}

const { disableLoginForm } = config;

export const UserProfileEditForm = ({ user, isSavingUser, updateProfile }: Props) => {
  const { data: preferences } = useGetUserPreferencesQuery();
  const compactMode = Boolean(preferences?.compactMode);
  const formKey = `${user?.id ?? 'anonymous'}-${compactMode ? 'compact' : 'standard'}`;

  const onSubmitProfileUpdate = (data: ProfileUpdateFields) => {
    updateProfile(data);
  };

  // check if authLabels is longer than 0 otherwise false
  const isExternalUser: boolean = (user && user.isExternal) ?? false;
  let authSource = isExternalUser && user && user.authLabels ? user.authLabels[0] : '';
  if (user?.isProvisioned) {
    authSource = 'SCIM';
  }
  const lockMessage = authSource ? ` (Synced via ${authSource})` : '';
  const disabledEdit = disableLoginForm || isExternalUser;

  return (
    <Form
      key={formKey}
      onSubmit={onSubmitProfileUpdate}
      validateOn="onBlur"
      defaultValues={{
        name: user?.name ?? '',
        email: user?.email ?? '',
        login: user?.login ?? '',
        compactMode,
      }}
    >
      {({ register, errors, control }) => {
        return (
          <>
            <FieldSet>
              <Field
                label={t('user-profile.fields.name-label', 'Name') + lockMessage}
                invalid={!!errors.name}
                error={<Trans i18nKey="user-profile.fields.name-error">Name is required</Trans>}
                disabled={disabledEdit}
              >
                <Input
                  {...register('name', { required: true })}
                  id="edit-user-profile-name"
                  placeholder={t('user-profile.fields.name-label', 'Name')}
                  suffix={<InputSuffix />}
                />
              </Field>

              <Field
                label={t('user-profile.fields.email-label', 'Email') + lockMessage}
                invalid={!!errors.email}
                error={<Trans i18nKey="user-profile.fields.email-error">Email is required</Trans>}
                disabled={disabledEdit}
              >
                <Input
                  {...register('email', { required: true })}
                  id="edit-user-profile-email"
                  placeholder={t('user-profile.fields.email-label', 'Email')}
                  suffix={<InputSuffix />}
                />
              </Field>

              <Field label={t('user-profile.fields.username-label', 'Username') + lockMessage} disabled={disabledEdit}>
                <Input
                  {...register('login')}
                  id="edit-user-profile-username"
                  placeholder={t('user-profile.fields.username-label', 'Username') + lockMessage}
                  suffix={<InputSuffix />}
                />
              </Field>

              <Field
                label={t('user-profile.fields.compact-mode-label', 'Compact mode')}
                description={t(
                  'user-profile.fields.compact-mode-description',
                  'Use a compact navigation layout with an undocked menu.'
                )}
              >
                <Controller
                  name="compactMode"
                  control={control}
                  render={({ field: { ref, value, ...field } }) => (
                    <Switch
                      {...field}
                      id="edit-user-profile-compact-mode"
                      data-testid="edit-user-profile-compact-mode"
                      ref={ref}
                      value={Boolean(value)}
                    />
                  )}
                />
              </Field>
            </FieldSet>
            <Button
              variant="primary"
              disabled={isSavingUser || disabledEdit}
              data-testid={selectors.components.UserProfile.profileSaveButton}
              type="submit"
            >
              <Trans i18nKey="common.save">Save</Trans>
            </Button>
          </>
        );
      }}
    </Form>
  );
};

export default UserProfileEditForm;

const InputSuffix = () => {
  return disableLoginForm ? (
    <Tooltip
      content={t(
        'profile.input-suffix.content-login-details-locked-because-managed-another',
        'Login details locked because they are managed in another system.'
      )}
    >
      <Icon name="lock" />
    </Tooltip>
  ) : null;
};
