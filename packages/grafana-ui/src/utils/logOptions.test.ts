import { logOptions } from './logOptions';


const RECOMMENDED_AMOUNT = 10;

describe('logOptions', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should not log anything if amount is less than or equal to recommendedAmount', () => {
    const warnSpy = jest.spyOn(console, 'warn').mockImplementation(() => {});

    logOptions(5, RECOMMENDED_AMOUNT, 'test-id', 'test-aria');

    expect(warnSpy).not.toHaveBeenCalled();
  });

  it('should log a warning if amount exceeds recommendedAmount', () => {
    const warnSpy = jest.spyOn(console, 'warn').mockImplementation(() => {});

    logOptions(15, RECOMMENDED_AMOUNT, 'test-id', 'test-aria');

    expect(warnSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        message: '[Combobox] Items exceed the recommended amount 10.',
        arguments: [
          '[Combobox] Items exceed the recommended amount 10.',
          expect.objectContaining({
            itemsCount: '15',
            recommendedAmount: '10',
            'aria-labelledby': 'test-aria',
            id: 'test-id',
          }),
        ],
      })
    );
  });
});
