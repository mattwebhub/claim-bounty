import { setupServer } from 'msw/node';
import { handlers } from '@/shared/test/mocks/handlers';

export const mockServer = setupServer(...handlers);
