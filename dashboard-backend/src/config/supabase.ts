import { createClient, SupabaseClient } from '@supabase/supabase-js';
import { env } from '../config/env.js';

export function createSupabaseClient(): SupabaseClient {
  return createClient(env.supabaseUrl, env.supabaseServiceKey, {
    auth: { persistSession: false },
  });
}
