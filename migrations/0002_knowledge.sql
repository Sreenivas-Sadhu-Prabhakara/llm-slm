CREATE TABLE IF NOT EXISTS knowledge_documents (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid,                       -- NULL = shared/global
  title text NOT NULL,
  source_type text NOT NULL,
  source_uri text,
  audience text NOT NULL DEFAULT 'customer',
  product text, brand text, language text DEFAULT 'taglish',
  version int NOT NULL DEFAULT 1,
  content_hash text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS knowledge_chunks (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id uuid NOT NULL REFERENCES knowledge_documents(id) ON DELETE CASCADE,
  tenant_id uuid,
  chunk_index int NOT NULL,
  content text NOT NULL,
  embedding vector(1024) NOT NULL,
  token_count int NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_chunks_embedding_hnsw
  ON knowledge_chunks USING hnsw (embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_chunks_tenant_audience
  ON knowledge_chunks (tenant_id);
