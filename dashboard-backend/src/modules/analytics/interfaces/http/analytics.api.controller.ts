import { withErrorHandler } from '../../../../lib/middleware/withErrorHandler.js';
import { withAuth } from '../../../../lib/middleware/withAuth.js';
import type { AnalyticsService } from '../../application/analytics.service.js';
import type { Request, Response } from 'express';

function range(req: Request): { from: string; to: string } {
  const to = (req.query.to as string) ?? new Date().toISOString();
  const from = (req.query.from as string) ?? new Date(Date.now() - 24 * 3600 * 1000).toISOString();
  return { from, to };
}

export function createAnalyticsController(service: AnalyticsService) {
  return {
    overview: withErrorHandler(
      withAuth({ requireAdmin: true })(async (req: Request, res: Response) => {
        const { from, to } = range(req);
        res.json(await service.overview(from, to));
      }),
    ),

    series: withErrorHandler(
      withAuth({ requireAdmin: true })(async (req: Request, res: Response) => {
        const { from, to } = range(req);
        const tenantId = req.query.tenantId as string | undefined;
        res.json(await service.usageSeries(from, to, tenantId));
      }),
    ),
  };
}
