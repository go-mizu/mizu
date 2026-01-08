-- PostgreSQL initialization script for Spreadsheet
-- This runs on first container startup

-- Enable btree_gist extension for exclusion constraints
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Create test database
CREATE DATABASE spreadsheet_test;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE spreadsheet TO spreadsheet;
GRANT ALL PRIVILEGES ON DATABASE spreadsheet_test TO spreadsheet;
