CREATE TABLE __table__ (
  id	TEXT,
  parent_id TEXT,
  type TEXT,
  props jsonb DEFAULT '{}'::jsonb,
  created_at TIMESTAMP with time zone,
  updated_at TIMESTAMP with time zone,
  PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx___table___updated_at ON __table__ (updated_at);
CREATE INDEX IF NOT EXISTS idx___table___parent ON __table__ (parent_id);
CREATE INDEX IF NOT EXISTS idx___table___type ON __table__ (type);
CREATE INDEX IF NOT EXISTS idx___table___props_gin ON __table__ USING gin (props);
CREATE OR REPLACE FUNCTION jsonb_shallow_merge(a jsonb, b jsonb)
  returns jsonb language sql as $$
select
  jsonb_object_agg(
      coalesce(ka, kb),
      case
      when va isnull then vb
      when vb isnull then va
      else va || vb
      end
  )
from jsonb_each(a) e1(ka, va)
  full join jsonb_each(b) e2(kb, vb) on ka = kb
$$;