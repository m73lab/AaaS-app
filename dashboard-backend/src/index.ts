import express from 'express';
import cors from 'cors';
import bodyParser from 'body-parser';
import { env } from './config/env.js';
import { buildContainer } from './app.bootstrap.js';
import { createRouter } from './routes/index.js';
import { withErrorHandler } from './lib/middleware/withErrorHandler.js';

const app = express();
app.use(cors({ origin: env.corsOrigins, credentials: true }));
app.use(bodyParser.json({ limit: '1mb' }));

const container = buildContainer();
app.use('/v1/aas', createRouter(container));

app.get('/health', (_req, res) => res.json({ status: 'ok', ts: Date.now() }));
app.use((_req, res) => res.status(404).json({ error: { code: 'NOT_FOUND', message: 'Route not found' } }));

app.use(withErrorHandler((_req, res) => {
  res.status(500).json({ error: { code: 'INTERNAL', message: 'Unhandled' } });
}));

const port = env.port;
app.listen(port, () => {
  console.log(`[AaaS Dashboard] listening on :${port}`);
});

export { app };
