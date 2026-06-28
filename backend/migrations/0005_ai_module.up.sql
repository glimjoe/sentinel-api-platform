-- Sentinel: 0005_ai_module (Phase 4)
-- AI usage tracking for budget enforcement and audit.

CREATE TABLE IF NOT EXISTS ai_usage (
  id                CHAR(26) NOT NULL,
  model             VARCHAR(64) NOT NULL,
  function          ENUM('attribution','completion','prioritization') NOT NULL,
  prompt_tokens     INT NOT NULL DEFAULT 0,
  completion_tokens INT NOT NULL DEFAULT 0,
  cost_usd          DECIMAL(12,8) NOT NULL DEFAULT 0,
  project_id        CHAR(26) NULL,
  created_at        DATETIME(3) NOT NULL,
  PRIMARY KEY (id),
  KEY idx_ai_usage_created (created_at),
  KEY idx_ai_usage_function (function),
  KEY idx_ai_usage_project (project_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
