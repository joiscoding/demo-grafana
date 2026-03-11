import { screen } from '@testing-library/react';
import { lazy, ComponentType } from 'react';
import { render } from 'test/test-utils';

import { getThemeById } from '@grafana/data/internal';
import { config, setEchoSrv } from '@grafana/runtime';

import { Echo } from '../services/echo/Echo';

import { GrafanaRoute, Props } from './GrafanaRoute';
import { GrafanaRouteComponentProps } from './types';

const mockChangeTheme = jest.fn();
jest.mock('../services/theme', () => ({
  changeTheme: (...args: unknown[]) => mockChangeTheme(...args),
}));

const mockLocation = {
  search: '?query=hello&test=asd',
  pathname: '',
  state: undefined,
  hash: '',
};
function setup(overrides: Partial<Props>) {
  const props: Props = {
    location: mockLocation,
    route: {
      path: '/',
      component: () => <div />,
    },
    ...overrides,
  };

  render(<GrafanaRoute {...props} />);
}

describe('GrafanaRoute', () => {
  beforeEach(() => {
    setEchoSrv(new Echo());
    mockChangeTheme.mockReset();
    config.theme2 = getThemeById('light');
  });

  it('Parses search', () => {
    let capturedProps: GrafanaRouteComponentProps;
    const PageComponent = (props: GrafanaRouteComponentProps) => {
      capturedProps = props;
      return <div />;
    };

    setup({ route: { component: PageComponent, path: '' } });
    expect(capturedProps!.queryParams.query).toBe('hello');
  });

  it('Shows loading on lazy load', async () => {
    const PageComponent = lazy(() => {
      return new Promise<{ default: ComponentType }>(() => {});
    });

    setup({ route: { component: PageComponent, path: '' } });

    expect(await screen.findByLabelText('Loading')).toBeInTheDocument();
  });

  it('Shows error on page error', async () => {
    const PageComponent = () => {
      throw new Error('Page threw error');
    };

    const consoleError = jest.fn();
    jest.spyOn(console, 'error').mockImplementation(consoleError);

    setup({ route: { component: PageComponent, path: '' } });

    expect(await screen.findByRole('heading', { name: 'An unexpected error happened' })).toBeInTheDocument();
    expect(consoleError).toHaveBeenCalled();
  });

  it('Applies dark theme from URL when mode differs', () => {
    setup({ location: { ...mockLocation, search: '?theme=dark' } });
    expect(mockChangeTheme).toHaveBeenCalledWith('dark', true);
  });

  it('Does not apply theme when URL mode matches current mode', () => {
    config.theme2 = getThemeById('dark');
    setup({ location: { ...mockLocation, search: '?theme=dark' } });
    expect(mockChangeTheme).not.toHaveBeenCalled();
  });
});
