import { handle } from 'hono/vercel';
import { createApp } from '../src/app';

export const config = {
  runtime: 'edge',
};

const app = createApp();

export default handle(app);
