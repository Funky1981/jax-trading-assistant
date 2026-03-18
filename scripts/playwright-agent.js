#!/usr/bin/env node
// scripts/playwright-agent.js
// Lightweight HTTP agent that runs Playwright E2E tests on the host machine
// on behalf of the Docker-based jax-trader backend.
//
// The jax-trader container cannot run `npx playwright test` because Node.js
// is not installed inside the image.  This agent listens on port 9092 and
// is reachable from the container via http://host.docker.internal:9092.
//
// Endpoints:
//   POST /run?spec=<name>  Run tests (optionally scoped to one spec file).
//                          Blocks until tests complete, returns JSON result.
//   GET  /health           Returns {"status":"ok"}.
//
// Usage: automatically started by start.ps1; stopped by stop.ps1.

'use strict';

const http = require('http');
const { spawnSync } = require('child_process');
const path = require('path');
const fs = require('fs');
const url = require('url');

const PORT = parseInt(process.env.PLAYWRIGHT_AGENT_PORT || '9092', 10);
const REPO_ROOT = path.resolve(__dirname, '..');
const FRONTEND_DIR = path.join(REPO_ROOT, 'frontend');
const RUNTIME_DIR = path.join(REPO_ROOT, '.runtime');
const LOG_FILE = path.join(REPO_ROOT, 'logs', 'playwright-agent.log');
const PID_FILE = path.join(RUNTIME_DIR, 'playwright-agent.pid');
const PLAYWRIGHT_CLI = path.join(FRONTEND_DIR, 'node_modules', 'playwright', 'cli.js');

// Ensure dirs exist.
[RUNTIME_DIR, path.dirname(LOG_FILE)].forEach(d => {
  if (!fs.existsSync(d)) fs.mkdirSync(d, { recursive: true });
});

// Write PID file.
fs.writeFileSync(PID_FILE, String(process.pid));

function log(msg) {
  const line = `${new Date().toISOString()}  ${msg}\n`;
  process.stdout.write(line);
  fs.appendFileSync(LOG_FILE, line, { encoding: 'utf8' });
}

function sendJson(res, code, obj) {
  const body = JSON.stringify(obj);
  res.writeHead(code, { 'Content-Type': 'application/json; charset=utf-8', 'Content-Length': Buffer.byteLength(body) });
  res.end(body);
}

if (!fs.existsSync(PLAYWRIGHT_CLI)) {
  log(`ERROR: Playwright CLI not found at ${PLAYWRIGHT_CLI} -- run 'npm install' in frontend/ first.`);
  process.exit(1);
}

const server = http.createServer((req, res) => {
  const parsed = url.parse(req.url, true);
  const method = req.method.toUpperCase();
  const pathname = parsed.pathname;

  if (pathname === '/health' && method === 'GET') {
    sendJson(res, 200, { status: 'ok' });
    return;
  }

  if (pathname === '/run' && method === 'POST') {
    // Sanitise spec parameter.
    let spec = parsed.query.spec || '';
    let specArgs = [];
    if (spec) {
      const cleaned = path.basename(spec).replace(/\.\./g, '').replace(/[/\\]/g, '');
      const noExt = cleaned.replace(/\.spec\.ts$/, '').replace(/\.spec$/, '');
      if (noExt) {
        specArgs = [`e2e/${noExt}.spec.ts`];
        log(`Running spec: ${noExt}`);
      } else {
        log(`Skipped invalid spec '${spec}', running full suite`);
      }
    } else {
      log('Running full test suite');
    }

    const args = [PLAYWRIGHT_CLI, 'test', '--reporter=list', ...specArgs];
    const startedAt = new Date();

    const result = spawnSync(process.execPath, args, {
      cwd: FRONTEND_DIR,
      encoding: 'utf8',
      maxBuffer: 32 * 1024 * 1024,
      timeout: 10 * 60 * 1000, // 10 minutes
    });

    const completedAt = new Date();
    const durationMs = completedAt - startedAt;
    const exitCode = result.status !== null ? result.status : -1;
    const status = exitCode === 0 ? 'passed' : 'failed';

    let output = (result.stdout || '') + (result.stderr || '');
    const MAX = 24000;
    if (output.length > MAX) output = output.slice(0, MAX) + '\n... [truncated]';

    log(`Run finished: ${status} (exit ${exitCode}) in ${Math.round(durationMs / 1000)}s`);

    sendJson(res, 200, {
      startedAt: startedAt.toISOString(),
      completedAt: completedAt.toISOString(),
      durationMs,
      exitCode,
      status,
      spec: spec || '',
      output,
    });
    return;
  }

  res.writeHead(404);
  res.end();
});

server.listen(PORT, '0.0.0.0', () => {
  log(`Playwright agent listening on http://localhost:${PORT} (PID ${process.pid})`);
});

server.on('error', err => {
  log(`Server error: ${err.message}`);
  process.exit(1);
});

// Clean up PID file on exit.
process.on('exit', () => {
  try { fs.unlinkSync(PID_FILE); } catch (_) {}
});
['SIGINT', 'SIGTERM'].forEach(sig => {
  process.on(sig, () => process.exit(0));
});
