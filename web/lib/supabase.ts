// Supabase browser client for Realtime (inbox updates)
// Lazy-initialized to avoid build-time errors when env vars aren't set
import { createClient, SupabaseClient } from '@supabase/supabase-js';

let client: SupabaseClient | null = null;

function getClient(): SupabaseClient | null {
  if (client) return client;

  const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL;
  const supabaseAnonKey = process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY;

  if (!supabaseUrl || !supabaseAnonKey) {
    return null;
  }

  client = createClient(supabaseUrl, supabaseAnonKey);
  return client;
}

// Subscribe to new WhatsApp messages for realtime inbox updates
export function subscribeToMessages(
  tenantId: string,
  onMessage: (payload: any) => void
) {
  const sb = getClient();
  if (!sb) {
    // No Supabase configured - return a no-op subscription
    return {
      unsubscribe: () => {},
    };
  }

  return sb
    .channel(`wa_messages:${tenantId}`)
    .on(
      'postgres_changes',
      {
        event: 'INSERT',
        schema: 'public',
        table: 'wa_messages',
        filter: `tenant_id=eq.${tenantId}`,
      },
      onMessage
    )
    .subscribe();
}

// Subscribe to conversation updates
export function subscribeToConversations(
  tenantId: string,
  onUpdate: (payload: any) => void
) {
  const sb = getClient();
  if (!sb) {
    return {
      unsubscribe: () => {},
    };
  }

  return sb
    .channel(`wa_conversations:${tenantId}`)
    .on(
      'postgres_changes',
      {
        event: '*',
        schema: 'public',
        table: 'wa_conversations',
        filter: `tenant_id=eq.${tenantId}`,
      },
      onUpdate
    )
    .subscribe();
}