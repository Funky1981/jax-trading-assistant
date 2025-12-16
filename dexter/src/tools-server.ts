#!/usr/bin/env bun
import { config } from "dotenv";

config({ quiet: true });

type ToolRequest = { tool: string; input: unknown };

function jsonResponse(obj: unknown, status = 200): Response {
  return new Response(JSON.stringify(obj), {
    status,
    headers: { "content-type": "application/json" },
  });
}

function badRequest(message: string): Response {
  return jsonResponse({ error: message }, 400);
}

function getPort(): number {
  const raw = process.env.PORT || "3000";
  const port = Number(raw);
  return Number.isFinite(port) ? port : 3000;
}

function isMockMode(): boolean {
  return process.env.DEXTER_TOOLS_MOCK === "1" || process.env.DEXTER_TOOLS_MOCK === "true";
}

async function handleResearchCompany(input: any) {
  const ticker = String(input?.ticker ?? "").trim();
  const questions = Array.isArray(input?.questions) ? input.questions.map(String) : [];
  if (!ticker) throw new Error("ticker is required");
  if (questions.length === 0) throw new Error("questions is required");

  if (isMockMode()) {
    return {
      ticker,
      summary: `Mock research summary for ${ticker}.`,
      key_points: ["Mock point 1", "Mock point 2"],
      metrics: { pe_ratio: 28.5 },
      raw_markdown: `## ${ticker} Research\n\n- Mock point 1\n- Mock point 2\n`,
    };
  }

  // Real implementation can be wired to Dexter Agent later.
  return {
    ticker,
    summary: `Dexter tools server is running, but non-mock mode is not wired yet.`,
    key_points: questions.slice(0, 5).map((q: string) => `Question received: ${q}`),
    metrics: {},
    raw_markdown: `## ${ticker} Research\n\n${questions.map((q: string) => `- ${q}`).join("\n")}\n`,
  };
}

async function handleCompareCompanies(input: any) {
  const tickers = Array.isArray(input?.tickers) ? input.tickers.map(String).filter(Boolean) : [];
  const focus = String(input?.focus ?? "").trim();
  if (tickers.length < 2) throw new Error("tickers must include at least 2 tickers");
  if (!focus) throw new Error("focus is required");

  if (isMockMode()) {
    return {
      comparison_axis: focus,
      items: tickers.map((t: string) => ({ ticker: t, thesis: `Mock thesis for ${t}`, notes: [] })),
    };
  }

  return {
    comparison_axis: focus,
    items: tickers.map((t: string) => ({ ticker: t, thesis: `Placeholder thesis for ${t}`, notes: [] })),
  };
}

Bun.serve({
  port: getPort(),
  async fetch(req) {
    const url = new URL(req.url);

    if (req.method === "GET" && url.pathname === "/health") {
      return jsonResponse({ ok: true });
    }

    if (req.method !== "POST") {
      return jsonResponse({ error: "method not allowed" }, 405);
    }

    let body: ToolRequest;
    try {
      body = (await req.json()) as ToolRequest;
    } catch {
      return badRequest("invalid JSON body");
    }

    const tool = String(body?.tool ?? "").trim();
    if (url.pathname !== "/tools") {
      return jsonResponse({ error: "not found" }, 404);
    }
    if (!tool) {
      return badRequest("tool is required");
    }

    try {
      switch (tool) {
        case "dexter.research_company": {
          const output = await handleResearchCompany(body.input);
          return jsonResponse({ output });
        }
        case "dexter.compare_companies": {
          const output = await handleCompareCompanies(body.input);
          return jsonResponse({ output });
        }
        default:
          return badRequest(`unknown tool: ${tool}`);
      }
    } catch (e: any) {
      return badRequest(e?.message || "request failed");
    }
  },
});

console.log(`Dexter tools server listening on :${getPort()} (mock=${isMockMode()})`);

