DROP INDEX IF EXISTS idx_audit_logs_target;
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_action;
DROP INDEX IF EXISTS idx_audit_logs_user_id;
DROP TABLE IF EXISTS audit_logs;

DROP INDEX IF EXISTS idx_reports_created_at;
DROP INDEX IF EXISTS idx_reports_reporter_id;
DROP INDEX IF EXISTS idx_reports_user_id;
DROP INDEX IF EXISTS idx_reports_video_id;
DROP INDEX IF EXISTS idx_reports_status;
DROP TABLE IF EXISTS content_reports;

DROP TYPE IF EXISTS report_status;
DROP TYPE IF EXISTS report_type;
