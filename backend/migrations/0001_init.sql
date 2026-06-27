-- Sentinel schema: 0001_init
-- Phase 1 scope: users + refresh_tokens only.
-- Other tables (projects, apis, mock_rules, etc.) arrive in Phase 2+.

CREATE TABLE IF NOT EXISTS users (
  id              CHAR(26)        NOT NULL,
  email           VARCHAR(255)    NOT NULL,
  password_hash   VARCHAR(255)    NOT NULL,
  display_name    VARCHAR(64)     NOT NULL DEFAULT '',
  role            ENUM('admin','engineer','viewer') NOT NULL DEFAULT 'viewer',
  is_active       TINYINT(1)      NOT NULL DEFAULT 1,
  created_at      DATETIME(3)     NOT NULL,
  updated_at      DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_users_email (email),
  KEY idx_users_role (role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS refresh_tokens (
  id              CHAR(26)        NOT NULL,
  user_id         CHAR(26)        NOT NULL,
  token_hash      VARCHAR(255)    NOT NULL,
  expires_at      DATETIME(3)     NOT NULL,
  revoked_at      DATETIME(3)     NULL,
  ip              VARCHAR(45)     NOT NULL DEFAULT '',
  user_agent      VARCHAR(255)    NOT NULL DEFAULT '',
  created_at      DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_refresh_tokens_hash (token_hash),
  KEY idx_refresh_tokens_user (user_id),
  KEY idx_refresh_tokens_expires (expires_at),
  CONSTRAINT fk_refresh_tokens_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
