import { z } from 'zod';
import { withErrorHandler, withValidation } from '../../../../lib/middleware/withErrorHandler.js';
import { withAuth } from '../../../../lib/middleware/withAuth.js';
import type { TenantService } from '../../application/tenant.service.js';
import type { Request, Response } from 'express';

export const createTenantSchema = z.object({
  name: z.string().min(1),
  email: z.string().email(),
  plan: z.string().optional(),
  rateLimitPerMin: z.number().int().positive().optional(),
  rateLimitPerHour: z.number().int().positive().optional(),
});

export const updateTenantSchema = z.object({
  name: z.string().min(1).optional(),
  email: z.string().email().optional(),
  plan: z.string().optional(),
  rateLimitPerMin: z.number().int().positive().optional(),
  rateLimitPerHour: z.number().int().positive().optional(),
  active: z.boolean().optional(),
});

export function createTenantController(service: TenantService) {
  return {
    list: withErrorHandler(
      withAuth({ requireAdmin: true })(async (_req: Request, res: Response) => {
        res.json(await service.list());
      }),
    ),

    get: withErrorHandler(
      withAuth({ requireAdmin: true })(async (req: Request, res: Response) => {
        res.json(await service.getById(req.params.id as string));
      }),
    ),

    create: withErrorHandler(
      withAuth({ requireAdmin: true })(
        withValidation(createTenantSchema)(async (req: Request, res: Response) => {
          res.status(201).json(await service.create(req.body));
        }),
      ),
    ),

    update: withErrorHandler(
      withAuth({ requireAdmin: true })(
        withValidation(updateTenantSchema)(async (req: Request, res: Response) => {
          res.json(await service.update(req.params.id as string, req.body));
        }),
      ),
    ),

    remove: withErrorHandler(
      withAuth({ requireAdmin: true })(async (req: Request, res: Response) => {
        await service.remove(req.params.id as string);
        res.status(204).end();
      }),
    ),
  };
}
