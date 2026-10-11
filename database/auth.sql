-- Auth Service Database Schema
-- Central user storage for SSO across all platform services

-- Enable UUID extension (using pgcrypto for gen_random_uuid)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users table - Central source of truth for all user data
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  firebase_uid TEXT UNIQUE NOT NULL,

  -- Basic user info
  email TEXT UNIQUE NOT NULL,
  full_name TEXT,
  whatsapp_number TEXT,
  avatar_url TEXT,
  email_verified BOOLEAN DEFAULT FALSE,

  -- Personal details
  gender TEXT,
  birthday TEXT,

  -- Student information (for GDG Babcock)
  student_status TEXT, -- e.g., "undergraduate", "postgraduate", "alumni"
  matric_no TEXT,
  department TEXT,
  faculty TEXT,

  -- GDG tracks and skills
  primary_track TEXT, -- e.g., "web", "mobile", "cloud", "ai/ml"
  secondary_track TEXT,
  primary_skill_level TEXT, -- e.g., "beginner", "intermediate", "advanced"
  secondary_skill_level TEXT,
  teams TEXT[] DEFAULT '{}', -- Teams user belongs to

  -- Platform fields
  roles TEXT[] DEFAULT ARRAY['user']::TEXT[], -- "user", "admin", "moderator", "lead"

  -- Terms of Service
  tos_agreed BOOLEAN DEFAULT FALSE,
  tos_agreed_at TIMESTAMP WITH TIME ZONE,
  tos_version TEXT,

  -- Metadata
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  last_login_at TIMESTAMP WITH TIME ZONE,

  -- Soft delete
  is_active BOOLEAN DEFAULT TRUE,
  deleted_at TIMESTAMP WITH TIME ZONE
);

-- Refresh tokens table - Track active refresh tokens
CREATE TABLE IF NOT EXISTS refresh_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  revoked_at TIMESTAMP WITH TIME ZONE,
  is_active BOOLEAN DEFAULT TRUE,

  -- Track where token was issued (for security)
  ip_address INET,
  user_agent TEXT
);

-- User sessions table - Optional session tracking
CREATE TABLE IF NOT EXISTS user_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  session_token TEXT UNIQUE NOT NULL,
  expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  last_activity_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

  -- Session metadata
  ip_address INET,
  user_agent TEXT,
  device_info JSONB
);

-- Audit log for important auth events
CREATE TABLE IF NOT EXISTS auth_audit_log (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  event_type TEXT NOT NULL, -- "login", "logout", "signup", "profile_update", "role_change", etc.
  event_data JSONB,
  ip_address INET,
  user_agent TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_users_firebase_uid ON users(firebase_uid);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_active ON refresh_tokens(is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_token ON user_sessions(session_token);
CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON auth_audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON auth_audit_log(created_at);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = CURRENT_TIMESTAMP;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update updated_at
CREATE TRIGGER update_users_updated_at
  BEFORE UPDATE ON users
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

-- Function to clean up expired tokens
CREATE OR REPLACE FUNCTION cleanup_expired_tokens()
RETURNS void AS $$
BEGIN
  UPDATE refresh_tokens
  SET is_active = FALSE, revoked_at = CURRENT_TIMESTAMP
  WHERE expires_at < CURRENT_TIMESTAMP AND is_active = TRUE;

  DELETE FROM user_sessions
  WHERE expires_at < CURRENT_TIMESTAMP;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE users IS 'Central user table - single source of truth for all platform services';
COMMENT ON COLUMN users.firebase_uid IS 'Firebase UID - links to Firebase Auth identity';
COMMENT ON COLUMN users.id IS 'Internal user ID - used across all platform services';
COMMENT ON COLUMN users.teams IS 'Teams user belongs to (e.g., organizing team, tech team)';
COMMENT ON COLUMN users.roles IS 'User roles for authorization (user, admin, moderator, lead, etc.)';
COMMENT ON COLUMN users.primary_track IS 'User primary focus track (web, mobile, cloud, ai/ml, etc.)';
COMMENT ON COLUMN users.student_status IS 'Student status (undergraduate, postgraduate, alumni)';
COMMENT ON COLUMN users.tos_agreed IS 'Whether user has agreed to Terms of Service';
COMMENT ON TABLE refresh_tokens IS 'Refresh tokens for JWT rotation';
COMMENT ON TABLE auth_audit_log IS 'Audit trail for security and compliance';
