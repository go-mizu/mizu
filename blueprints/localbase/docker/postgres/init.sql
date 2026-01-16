-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Create schemas
CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS storage;
CREATE SCHEMA IF NOT EXISTS functions;
CREATE SCHEMA IF NOT EXISTS realtime;

-- Grant permissions
GRANT ALL ON SCHEMA auth TO localbase;
GRANT ALL ON SCHEMA storage TO localbase;
GRANT ALL ON SCHEMA functions TO localbase;
GRANT ALL ON SCHEMA realtime TO localbase;
