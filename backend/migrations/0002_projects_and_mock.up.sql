-- Sentinel schema: 0002_projects_and_mock
-- Phase 2 M1 scope: projects, project_members, apis, mock_rules, mock_hits.
-- Aligns with plan §5 (tables 2-6) plus `extractor_json` on mock_rules per ADR-0007.
-- `schema_migrations` is managed by golang-migrate; do not write it here.

-- ─── projects ────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS projects (
  id                CHAR(26)        NOT NULL,
  name              VARCHAR(128)    NOT NULL,
  slug              VARCHAR(64)     NOT NULL,
  owner_id          CHAR(26)        NOT NULL,
  description       VARCHAR(512)    NOT NULL DEFAULT '',
  default_base_url  VARCHAR(512)    NOT NULL DEFAULT '',
  created_at        DATETIME(3)     NOT NULL,
  updated_at        DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_projects_slug (slug),
  KEY idx_projects_owner (owner_id),
  CONSTRAINT fk_projects_owner FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ─── project_members ─────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS project_members (
  project_id  CHAR(26)                                       NOT NULL,
  user_id     CHAR(26)                                       NOT NULL,
  role        ENUM('admin','engineer','viewer')              NOT NULL DEFAULT 'viewer',
  created_at  DATETIME(3)                                    NOT NULL,
  PRIMARY KEY (project_id, user_id),
  KEY idx_project_members_user (user_id),
  CONSTRAINT fk_project_members_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
  CONSTRAINT fk_project_members_user    FOREIGN KEY (user_id)    REFERENCES users(id)    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ─── apis ─────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS apis (
  id                     CHAR(26)                                  NOT NULL,
  project_id             CHAR(26)                                  NOT NULL,
  name                   VARCHAR(128)                              NOT NULL,
  method                 ENUM('GET','POST','PUT','PATCH','DELETE','HEAD','OPTIONS') NOT NULL,
  path                   VARCHAR(512)                              NOT NULL,
  operation_id           VARCHAR(128)                              NOT NULL DEFAULT '',
  tags_json              JSON                                      NULL,
  request_schema_json    JSON                                      NULL,
  response_schema_json   JSON                                      NULL,
  spec_json              JSON                                      NULL,
  source                 ENUM('openapi','manual')                  NOT NULL DEFAULT 'manual',
  spec_version           VARCHAR(32)                               NOT NULL DEFAULT '',
  deprecated             TINYINT(1)                                NOT NULL DEFAULT 0,
  created_at             DATETIME(3)                               NOT NULL,
  updated_at             DATETIME(3)                               NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_apis_project_method_path (project_id, method, path),
  KEY idx_apis_project (project_id),
  CONSTRAINT fk_apis_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ─── mock_rules ──────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mock_rules (
  id                       CHAR(26)       NOT NULL,
  api_id                   CHAR(26)       NOT NULL,
  name                     VARCHAR(128)   NOT NULL,
  -- match_json shape: {"query":{"k":"v"},"headers":{"K":"V"},"body":{...}}. Bad schema → 422 (not silent).
  match_json               JSON           NOT NULL,
  response_status          INT            NOT NULL DEFAULT 200,
  response_headers_json    JSON           NULL,
  response_body_json       JSON           NULL,
  -- extractor_json shape: [{"path":"$.x.y","as":"var_name","from":"response.body","type":"jsonpath|regex"}].
  extractor_json           JSON           NULL,
  priority                 INT            NOT NULL DEFAULT 100,
  delay_ms                 INT            NOT NULL DEFAULT 0,
  enabled                  TINYINT(1)     NOT NULL DEFAULT 1,
  hit_count                BIGINT         NOT NULL DEFAULT 0,
  created_at               DATETIME(3)    NOT NULL,
  updated_at               DATETIME(3)    NOT NULL,
  PRIMARY KEY (id),
  KEY idx_mock_rules_api_enabled_priority (api_id, enabled, priority),
  CONSTRAINT fk_mock_rules_api FOREIGN KEY (api_id) REFERENCES apis(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ─── mock_hits ───────────────────────────────────────────────────────────────
-- `mock_hits` write path is wired in M2 (recorder). The table is created now so
-- the engine can INSERT in M1 if it wants — but for M1 we leave a TODO.
CREATE TABLE IF NOT EXISTS mock_hits (
  id                   CHAR(26)        NOT NULL,
  mock_rule_id         CHAR(26)        NOT NULL,
  request_method       VARCHAR(16)     NOT NULL,
  request_path         VARCHAR(512)    NOT NULL,
  request_headers_json JSON            NULL,
  request_body_json    JSON            NULL,             -- truncated to 8KB by service layer
  response_status      INT             NOT NULL,
  response_body_json   JSON            NULL,
  duration_ms          INT             NOT NULL,
  created_at           DATETIME(3)     NOT NULL,
  PRIMARY KEY (id),
  KEY idx_mock_hits_rule_created (mock_rule_id, created_at DESC),
  CONSTRAINT fk_mock_hits_rule FOREIGN KEY (mock_rule_id) REFERENCES mock_rules(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
