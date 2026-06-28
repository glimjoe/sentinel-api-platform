-- Sentinel schema: 0003_audit_logs
-- Phase 2 M2: audit trail for key operations.

CREATE TABLE IF NOT EXISTS audit_logs (
  id            CHAR(26)        NOT NULL,
  user_id       CHAR(26)        NULL,
  action        VARCHAR(64)     NOT NULL,
  resource_type VARCHAR(64)     NOT NULL,
  resource_id   VARCHAR(64)     NOT NULL DEFAULT '',
  project_id    CHAR(26)        NULL,
  payload_json  JSON            NULL,
  ip            VARCHAR(45)     NOT NULL DEFAULT '',
  user_agent    VARCHAR(512)    NOT NULL DEFAULT '',
  created_at    DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  KEY idx_audit_user_created (user_id, created_at DESC),
  KEY idx_audit_resource (resource_type, resource_id),
  KEY idx_audit_project (project_id, created_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
