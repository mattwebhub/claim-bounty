export const commonEn = {
  appName: 'React Template',
  actions: {
    goHome: 'Go home',
    reload: 'Reload page',
    retry: 'Try again',
    toggleTheme: 'Toggle color theme',
  },
  errors: {
    unexpectedTitle: 'Something went wrong',
    unexpectedDescription: 'The page could not be displayed. Reload it to try again.',
  },
  home: {
    eyebrow: 'Production-shaped foundation',
    title: 'Build the feature, not the scaffolding.',
    description:
      'A strict, accessible React and Vite baseline with explicit state ownership and fast feedback.',
    explore: 'Explore the foundation',
    foundationsTitle: 'Template foundations',
    foundations: {
      architecture: {
        title: 'Visible boundaries',
        description: 'Feature-first ownership with mechanically enforced dependency direction.',
        command: 'pnpm architecture:check',
      },
      state: {
        title: 'One state owner',
        description: 'React Query, URL state, forms, and Zustand each have one clear job.',
        command: 'pnpm typecheck',
      },
      quality: {
        title: 'Fast proof',
        description: 'Strict checks and behavior-focused tests run through one command.',
        command: 'pnpm check:fast',
      },
    },
  },
  notFound: {
    title: 'Page not found',
    description: 'The requested route does not exist in this application.',
  },
  states: {
    loading: 'Loading',
    offline: 'You are offline. Changes will not sync until the connection returns.',
    save: {
      idle: 'No changes',
      dirty: 'Unsaved changes',
      saving: 'Saving changes',
      saved: 'Changes saved',
      error: 'Changes could not be saved',
      conflict: 'Another version was saved first',
      offline: 'Waiting for connection',
    },
  },
} as const;
