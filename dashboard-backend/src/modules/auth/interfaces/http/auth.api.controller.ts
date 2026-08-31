import { z } from 'zod';
import { withErrorHandler, withValidation } from '../../../../lib/middleware/withErrorHandler.js';
import { withAuth } from '../../../../lib/middleware/withAuth.js';
import type { AuthService } from '../../application/auth.service.js';
import type { Request, Response } from 'express';

export const loginSchema = z.object({
  email: z.string().email(),
  password: z.string().min(1),
});

export function createAuthController(authService: AuthService) {
  return {
    login: withErrorHandler(
      withValidation(loginSchema)(async (req: Request, res: Response) => {
        const result = await authService.login(req.body);
        res.json(result);
      }),
    ),

    me: withErrorHandler(
      withAuth()(async (req: Request, res: Response) => {
        res.json({ id: req.auth!.sub, email: req.auth!.email, role: req.auth!.role });
      }),
    ),
  };
}
