#!/usr/bin/env node
// Nidus Mail MCP Server v0.1.0
// Provides email sending tools to opencode and other MCP clients

const https = require('https');
const http = require('http');
const readline = require('readline');

const API_BASE = process.env.NIDUS_API_URL || 'http://localhost:3001';
const API_TOKEN = process.env.NIDUS_API_TOKEN || '';

async function apiRequest(method, path, body) {
  const url = new URL(API_BASE + path);
  const isHttps = url.protocol === 'https:';
  const mod = isHttps ? https : http;

  return new Promise((resolve, reject) => {
    const options = {
      hostname: url.hostname,
      port: url.port || (isHttps ? 443 : 80),
      path: url.pathname,
      method,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${API_TOKEN}`,
      },
    };

    const req = mod.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => data += chunk);
      res.on('end', () => {
        try {
          resolve({ status: res.statusCode, data: JSON.parse(data || '{}') });
        } catch {
          resolve({ status: res.statusCode, data: { raw: data } });
        }
      });
    });

    req.on('error', (e) => reject({ status: 0, error: e.message }));
    
    if (body) req.write(JSON.stringify(body));
    req.end();
  });
}

const TOOLS = [
  {
    name: 'mail_send',
    description: 'Send an email using Nidus Mail. Supports templates with variable substitution.',
    inputSchema: {
      type: 'object',
      properties: {
        to: { type: 'string', description: 'Recipient email address' },
        to_name: { type: 'string', description: 'Recipient name (optional)' },
        subject: { type: 'string', description: 'Email subject' },
        template_id: { type: 'string', description: 'Template ID (welcome, deploy-success, deploy-failed, password-reset, db-credentials)' },
        html: { type: 'string', description: 'HTML body (overrides template)' },
        text: { type: 'string', description: 'Plain text body' },
        vars: { type: 'object', description: 'Template variables, e.g. {"name":"John","url":"https://..."}' },
      },
      required: ['to'],
    },
  },
  {
    name: 'mail_templates',
    description: 'List all available Nidus email templates.',
    inputSchema: { type: 'object', properties: {} },
  },
  {
    name: 'mail_render',
    description: 'Preview a template with variables without sending.',
    inputSchema: {
      type: 'object',
      properties: {
        template_id: { type: 'string', description: 'Template ID to render' },
        vars: { type: 'object', description: 'Template variables' },
      },
      required: ['template_id'],
    },
  },
  {
    name: 'mail_logs',
    description: 'View recent email send logs.',
    inputSchema: { type: 'object', properties: {} },
  },
];

async function handleToolCall(name, args) {
  switch (name) {
    case 'mail_send': {
      const result = await apiRequest('POST', '/api/mail/send', {
        to: args.to,
        to_name: args.to_name || args.to,
        subject: args.subject,
        template_id: args.template_id,
        html: args.html,
        text: args.text,
        vars: args.vars || {},
      });
      return result;
    }
    case 'mail_templates': {
      const result = await apiRequest('GET', '/api/mail/templates');
      return result;
    }
    case 'mail_render': {
      // We get the template, then show how variables would render
      const result = await apiRequest('GET', '/api/mail/templates');
      if (result.status === 200 && Array.isArray(result.data)) {
        const tmpl = result.data.find(t => t.id === args.template_id);
        if (tmpl) {
          let html = tmpl.html;
          let text = tmpl.text;
          let subject = tmpl.subject;
          if (args.vars) {
            for (const [k, v] of Object.entries(args.vars)) {
              const re = new RegExp(`{{${k}}}`, 'g');
              html = html.replace(re, v);
              text = text.replace(re, v);
              subject = subject.replace(re, v);
            }
          }
          return { status: 200, data: { template_id: tmpl.id, subject, html, text } };
        }
        return { status: 404, data: { error: 'Template not found' } };
      }
      return result;
    }
    case 'mail_logs': {
      const result = await apiRequest('GET', '/api/mail/logs');
      return result;
    }
    default:
      return { status: 404, error: `Unknown tool: ${name}` };
  }
}

// ─── MCP Protocol ─────────────────────────────────────────────────────

const rl = readline.createInterface({ input: process.stdin, terminal: false });
let initialized = false;

function send(msg) {
  process.stdout.write(JSON.stringify(msg) + '\n');
}

rl.on('line', async (line) => {
  let msg;
  try { msg = JSON.parse(line); } catch { return; }

  const { id, method, params } = msg;

  switch (method) {
    case 'initialize':
      send({
        jsonrpc: '2.0', id,
        result: {
          protocolVersion: '2024-11-05',
          capabilities: { tools: {} },
          serverInfo: { name: 'nidus-mail', version: '0.1.0' },
        },
      });
      break;

    case 'notifications/initialized':
      initialized = true;
      // MCP doesn't send a response for notifications
      break;

    case 'tools/list':
      send({ jsonrpc: '2.0', id, result: { tools: TOOLS } });
      break;

    case 'tools/call':
      try {
        const args = params.arguments || {};
        const result = await handleToolCall(params.name, args);
        send({
          jsonrpc: '2.0', id,
          result: {
            content: [{ type: 'text', text: JSON.stringify(result.data || result, null, 2) }],
          },
        });
      } catch (e) {
        send({
          jsonrpc: '2.0', id,
          error: { code: -32000, message: e.message || String(e) },
        });
      }
      break;

    default:
      send({
        jsonrpc: '2.0', id,
        error: { code: -32601, message: `Method not found: ${method}` },
      });
  }
});

// Heartbeat: keep process alive
setInterval(() => {}, 60000);
