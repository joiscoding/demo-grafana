import { PayloadAction } from '@reduxjs/toolkit';
import configureMockStore from 'redux-mock-store';
import { thunk } from 'redux-thunk';

import * as structuredLogging from '../../../../scripts/helpers/structuredLogging';
const { createStructuredLogger } = structuredLogging;
const structuredLogger = createStructuredLogger('public/test/core/thunk/thunkTester');

const mockStore = configureMockStore([thunk]);

export interface ThunkGiven {
  givenThunk: (thunkFunction: any) => ThunkWhen;
}

export interface ThunkWhen {
  whenThunkIsDispatched: (...args: unknown[]) => Promise<Array<PayloadAction<any>>>;
}

export const thunkTester = (initialState: unknown, debug?: boolean): ThunkGiven => {
  const store = mockStore(initialState);
  let thunkUnderTest: any = null;
  let dispatchedActions: PayloadAction[] = [];

  const givenThunk = (thunkFunction: any): ThunkWhen => {
    thunkUnderTest = thunkFunction;

    return instance;
  };

  const whenThunkIsDispatched = async (...args: unknown[]): Promise<PayloadAction[]> => {
    await store.dispatch(thunkUnderTest(...args));

    dispatchedActions = store.getActions();
    if (debug) {
      structuredLogger.log('resultingActions:', JSON.stringify(dispatchedActions, null, 2));
    }

    return dispatchedActions;
  };

  const instance = {
    givenThunk,
    whenThunkIsDispatched,
  };

  return instance;
};
