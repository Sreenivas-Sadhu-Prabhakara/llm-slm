CREATE TABLE IF NOT EXISTS conversations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid, user_id uuid,
  mode text NOT NULL DEFAULT 'customer',
  channel text NOT NULL DEFAULT 'web',
  status text NOT NULL DEFAULT 'open',
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS messages (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id uuid NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  tenant_id uuid, role text NOT NULL, content text NOT NULL,
  retrieved_chunk_ids uuid[], model text, latency_ms int,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS feedback (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  message_id uuid NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  tenant_id uuid, user_id uuid,
  rating text NOT NULL, solved boolean, note text,
  created_at timestamptz NOT NULL DEFAULT now()
);
