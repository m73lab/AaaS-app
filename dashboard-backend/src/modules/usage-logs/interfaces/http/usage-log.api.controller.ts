import { z } from 'zod';
import { withErrorHandler, withValidation } from '../../../../lib/middleware/withErrorHandler.js';
import { withAuth } from '../../../../lib/middleware/withAuth.js';
import type { UsageLogService } from '../../application/usage-log.service.js';
import type { Request, Response } from 'express';

export const ingestSchema = z.object({
  tenantId: z.string().min(1),
  sessionId: z.string().min(1),
  model: z.string().optional(),
  provider: z.string().optional(),
  format: z.string().optional(),
  promptTokens: z.number().int().min(0).optional(),
  completionTokens: z.number().int().min(0).optional(),
  entitiesDetected: z.number().int().min(0).optional(),
  categories: z.record(z.string(), z.number().int().min(0)).optional(),
  action: z.string().min(1),
  latencyMs: z.number().int().min(0).optional(),
});

export function createUsageLogController(service: UsageLogService) {
  return {
    ingest: withErrorHandler(
      withValidation(ingestSchema)(async (req: Request, res: Response) => {
        await service.ingest(req.body);
        res.status(202).json({ accepted: true });
      }),
    ),

    list: withErrorHandler(
      withAuth({ requireAdmin: true })(async (req: Request, res: Response) => {
        const tenantId = req.query.tenantId as string | undefined;
        const from = req.query.from as string | undefined;
        const to = req.query.to as string | undefined;
        const limit = req.query.limit ? parseInt(req.query.limit as string, 10) : 100;
        res.json(await service.query({ tenantId, from, to, limit }));
      }),
    ),
  };
}
