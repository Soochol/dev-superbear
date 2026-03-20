-- Test seed data for monitoring E2E tests
INSERT INTO users (id, email, name) VALUES
  ('00000000-0000-0000-0000-000000000001', 'test@test.com', 'Test User')
ON CONFLICT DO NOTHING;

INSERT INTO pipelines (id, user_id, name) VALUES
  ('00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', 'Test Pipeline')
ON CONFLICT DO NOTHING;

INSERT INTO agent_blocks (id, user_id, name, instruction) VALUES
  ('00000000-0000-0000-0000-000000000020', '00000000-0000-0000-0000-000000000001', 'Test Block', '매일 시황 분석')
ON CONFLICT DO NOTHING;

INSERT INTO cases (id, user_id, pipeline_id, symbol, status, event_date, event_snapshot, success_script, failure_script) VALUES
  ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000010', '005930', 'LIVE', '2026-03-20', '{"high":55000,"low":50000,"close":53000}', 'close >= event_high * 2', 'close < pre_event_ma(120)')
ON CONFLICT DO NOTHING;

INSERT INTO monitor_blocks (id, pipeline_id, block_id, cron, enabled, case_id) VALUES
  ('00000000-0000-0000-0000-000000000200', '00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000020', '0 9 * * 1-5', true, '00000000-0000-0000-0000-000000000100')
ON CONFLICT DO NOTHING;
