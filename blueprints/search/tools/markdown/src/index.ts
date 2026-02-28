import { Hono } from 'hono';

const app = new Hono();
app.get('/', (c) => c.text('ok'));
export default app;
