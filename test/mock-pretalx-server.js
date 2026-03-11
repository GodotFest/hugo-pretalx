#!/usr/bin/env node
/**
 * Mock Pretalx API server for tests.
 * Serves fixture JSON at /api/events/:event/talks/ and /api/events/:event/speakers/
 * in Pretalx paginated format. Accepts any Authorization: Token (e.g. test-token).
 *
 * Usage: node test/mock-pretalx-server.js
 *        From repo root. Fixtures read from test/fixtures/.
 */

const http = require('http');
const fs = require('fs');
const path = require('path');

const PORT = Number(process.env.PRETALX_MOCK_PORT) || 9876;
const FIXTURES_DIR = path.join(__dirname, 'fixtures');

function readJson(name) {
  const p = path.join(FIXTURES_DIR, name);
  try {
    return JSON.parse(fs.readFileSync(p, 'utf8'));
  } catch (e) {
    return [];
  }
}

const talks = readJson('talks.json');
const speakers = readJson('speakers.json');

function send(res, status, body) {
  res.writeHead(status, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify(body));
}

function paginated(results) {
  return { count: results.length, next: null, previous: null, results };
}

const server = http.createServer((req, res) => {
  const url = new URL(req.url || '', `http://localhost:${PORT}`);
  const pathname = url.pathname;

  // GET /api/events/:event/talks/ or /api/events/:event/speakers/
  const match = pathname.match(/^\/api\/events\/([^/]+)\/(talks|speakers)\/?$/);
  if (req.method === 'GET' && match) {
    const [, event, resource] = match;
    const data = resource === 'talks' ? talks : speakers;
    send(res, 200, paginated(data));
    return;
  }

  send(res, 404, { detail: 'Not found' });
});

server.listen(PORT, '127.0.0.1', () => {
  console.log(`Mock Pretalx API: http://127.0.0.1:${PORT}`);
});
