import type { Metric } from 'web-vitals';

interface WebVitalMeasurement {
  name: Metric['name'];
  value: number;
  delta: number;
  rating: Metric['rating'];
  navigationType: string;
}

interface WebVitalsSink {
  record(measurement: WebVitalMeasurement): void;
}

const noOpWebVitalsSink: WebVitalsSink = Object.freeze({
  record: () => undefined,
});

export function createWebVitalsReporter(sink: WebVitalsSink) {
  return (metric: Metric) => {
    sink.record({
      name: metric.name,
      value: metric.value,
      delta: metric.delta,
      rating: metric.rating,
      navigationType: metric.navigationType,
    });
  };
}

export function startWebVitals(sink: WebVitalsSink = noOpWebVitalsSink): void {
  if (!import.meta.env.PROD) return;

  void import('web-vitals')
    .then(({ onCLS, onFCP, onINP, onLCP, onTTFB }) => {
      const report = createWebVitalsReporter(sink);
      onCLS(report);
      onFCP(report);
      onINP(report);
      onLCP(report);
      onTTFB(report);
    })
    .catch(() => {
      // Monitoring must never make the application unavailable.
    });
}
