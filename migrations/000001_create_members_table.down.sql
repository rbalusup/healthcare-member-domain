-- Rollback: 000001_create_members_table

DROP TRIGGER IF EXISTS members_updated_at ON members;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS members;
