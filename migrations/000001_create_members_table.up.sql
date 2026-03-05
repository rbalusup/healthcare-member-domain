-- Migration: 000001_create_members_table
-- Creates the primary members table with all fields required by the Member domain entity.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS members (
    -- Identity
    member_id       UUID            NOT NULL DEFAULT uuid_generate_v4(),

    -- Demographics
    first_name      VARCHAR(100)    NOT NULL,
    last_name       VARCHAR(100)    NOT NULL,
    date_of_birth   DATE            NOT NULL,
    gender          VARCHAR(20)     NOT NULL DEFAULT 'UNSPECIFIED',

    -- Contact
    email           VARCHAR(255)    NOT NULL,
    phone           VARCHAR(30),
    street          VARCHAR(255),
    city            VARCHAR(100),
    state           VARCHAR(100),
    zip_code        VARCHAR(20),
    country         VARCHAR(100)    DEFAULT 'US',

    -- Enrollment
    plan_id             VARCHAR(100)    NOT NULL,
    enrollment_date     DATE            NOT NULL,
    termination_date    DATE,

    -- Status
    status          VARCHAR(20)     NOT NULL DEFAULT 'ACTIVE',

    -- Risk profile
    risk_score      NUMERIC(5, 2)   NOT NULL DEFAULT 0.00,
    care_level      VARCHAR(20)     NOT NULL DEFAULT 'STANDARD',
    risk_updated_at TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    -- Audit
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT members_pkey PRIMARY KEY (member_id),
    CONSTRAINT members_email_unique UNIQUE (email),
    CONSTRAINT members_status_check CHECK (status IN ('ACTIVE', 'INACTIVE', 'SUSPENDED')),
    CONSTRAINT members_gender_check CHECK (gender IN ('MALE', 'FEMALE', 'NON_BINARY', 'UNSPECIFIED')),
    CONSTRAINT members_care_level_check CHECK (care_level IN ('STANDARD', 'ENHANCED', 'COMPLEX')),
    CONSTRAINT members_risk_score_check CHECK (risk_score >= 0 AND risk_score <= 100)
);

-- Index for common query patterns
CREATE INDEX idx_members_email      ON members (email);
CREATE INDEX idx_members_plan_id    ON members (plan_id);
CREATE INDEX idx_members_status     ON members (status);
CREATE INDEX idx_members_care_level ON members (care_level);
CREATE INDEX idx_members_created_at ON members (created_at DESC);

-- Full-text search index on name fields
CREATE INDEX idx_members_name_search ON members USING gin(
    to_tsvector('english', first_name || ' ' || last_name)
);

-- Automatically update updated_at on row modification
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER members_updated_at
    BEFORE UPDATE ON members
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE members IS 'Core member (patient) records for the healthcare platform.';
COMMENT ON COLUMN members.member_id IS 'Unique identifier (UUID v4) for the member.';
COMMENT ON COLUMN members.risk_score IS 'Clinical risk score 0-100; higher values indicate greater risk.';
COMMENT ON COLUMN members.care_level IS 'Care intensity: STANDARD, ENHANCED, or COMPLEX.';
