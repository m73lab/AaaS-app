import dotenv from 'dotenv';
dotenv.config();

function required(name: string, value: string | undefined): string {
  if (!value) throw new Error(`Missing required env: ${name}`);
  return value;
}

export const env = {
  nodeEnv: process.env.NODE_ENV || 'development',
  isProduction: (process.env.NODE_ENV || 'development') === 'production',
  port: parseInt(process.env.PORT || '3001', 10),
  supabaseUrl: required('SUPABASE_URL', process.env.SUPABASE_URL),
  supabaseServiceKey: required('SUPABASE_SERVICE_KEY', process.env.SUPABASE_SERVICE_KEY),
  jwtSecret: required('JWT_SECRET', process.env.JWT_SECRET),
  redisUrl: process.env.REDIS_URL || 'redis://localhost:6379',
  corsOrigins: (process.env.CORS_ORIGINS || 'http://localhost:3000').split(','),
};
