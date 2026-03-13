import { createStructuredLogger } from './structuredLogging';

describe('createStructuredLogger', () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('writes a structured payload to the console sink in test environments', () => {
    const infoSpy = jest.spyOn(console, 'info').mockImplementation(() => {});
    const logger = createStructuredLogger('unit.test.logger', { feature: 'structured-logging' });

    logger.info('hello world', { id: 1 });

    expect(infoSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        level: 'info',
        source: 'unit.test.logger',
        message: 'hello world',
        context: { feature: 'structured-logging' },
        arguments: ['hello world', { id: 1 }],
      })
    );
  });

  it('records timer duration metadata when timeEnd is called', () => {
    const infoSpy = jest.spyOn(console, 'info').mockImplementation(() => {});
    const nowSpy = jest.spyOn(Date, 'now');
    nowSpy.mockReturnValueOnce(100).mockReturnValueOnce(160);

    const logger = createStructuredLogger('unit.test.timer');
    logger.time('phase-a');
    logger.timeEnd('phase-a');

    expect(infoSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        level: 'info',
        source: 'unit.test.timer',
        message: 'Structured timer end: phase-a',
        context: { label: 'phase-a', durationMs: 60 },
      })
    );
  });
});
