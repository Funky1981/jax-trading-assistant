DROP TRIGGER IF EXISTS trg_protect_ai_shadow_results ON ai_shadow_benchmark_results;
DROP TRIGGER IF EXISTS trg_protect_ai_shadow_attempts ON ai_shadow_benchmark_attempts;
DROP FUNCTION IF EXISTS protect_ai_shadow_append_only();
DROP TABLE IF EXISTS ai_shadow_benchmark_results;
DROP TABLE IF EXISTS ai_shadow_benchmark_attempts;
DROP TABLE IF EXISTS ai_shadow_benchmark_runs;
