import { describe, expect, it, vi } from 'vitest';
import { createWebVitalsReporter } from './web-vitals';

describe('createWebVitalsReporter', () => {
  it('forwards only a bounded, vendor-neutral measurement', () => {
    const record = vi.fn();
    const report = createWebVitalsReporter({ record });
    report({
      name: 'LCP',
      value: 1_200,
      delta: 40,
      rating: 'good',
      navigationType: 'navigate',
      navigationId: 42,
      id: 'metric-id-not-forwarded',
      entries: [],
    });

    expect(record).toHaveBeenCalledWith({
      name: 'LCP',
      value: 1_200,
      delta: 40,
      rating: 'good',
      navigationType: 'navigate',
    });
  });
});
