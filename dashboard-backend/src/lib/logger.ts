type Level = 'info' | 'warn' | 'error' | 'debug';

export const logger = {
  info: (...a: unknown[]) => console.log('[INFO]', ...a),
  warn: (...a: unknown[]) => console.warn('[WARN]', ...a),
  error: (...a: unknown[]) => console.error('[ERROR]', ...a),
  debug: (...a: unknown[]) => console.debug('[DEBUG]', ...a),
  log: (level: Level, ...a: unknown[]) => console[level]('[' + level.toUpperCase() + ']', ...a),
};
