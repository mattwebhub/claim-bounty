/** @type {import('dependency-cruiser').IConfiguration} */
module.exports = {
  forbidden: [
    {
      name: 'WEB-ARCH-001-no-cycles',
      severity: 'error',
      comment: 'Dependencies must be acyclic. See ARCHITECTURE.md.',
      from: {},
      to: { circular: true },
    },
    {
      name: 'WEB-ARCH-001-shared-is-feature-blind',
      severity: 'error',
      comment: 'Shared cannot import app, routes, or features. See ARCHITECTURE.md.',
      from: { path: '^src/shared/' },
      to: { path: '^src/(app|routes|features)/' },
    },
    {
      name: 'WEB-ARCH-001-feature-does-not-import-routes-or-app',
      severity: 'error',
      comment: 'Features cannot import route or application composition. See ARCHITECTURE.md.',
      from: { path: '^src/features/' },
      to: { path: '^src/(app|routes)/' },
    },
    {
      name: 'WEB-ARCH-002-no-cross-feature-internals',
      severity: 'error',
      comment: 'Import another feature only through its public index. See ARCHITECTURE.md.',
      from: { path: '^src/features/([^/]+)/' },
      to: {
        path: '^src/features/[^/]+/.+',
        pathNot: '^(?:src/features/$1/|src/features/[^/]+/index\\.ts$)',
      },
    },
  ],
  options: {
    doNotFollow: { path: 'node_modules' },
    exclude: { path: '(^|/)node_modules/' },
    tsConfig: { fileName: 'tsconfig.app.json' },
    enhancedResolveOptions: {
      exportsFields: ['exports'],
      conditionNames: ['import', 'types', 'default'],
    },
    reporterOptions: {
      text: { highlightFocused: true },
    },
  },
};
