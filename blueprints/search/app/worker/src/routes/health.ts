import { Hono } from 'hono'

type Env = {
  Bindings: {
    SEARCH_KV: KVNamespace
    ENVIRONMENT: string
  }
}

const app = new Hono<Env>()

app.get('/', (c) => {
  return c.json({ status: 'ok' })
})

export default app
