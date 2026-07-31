import { describe, expect, it } from 'vitest';
import nginxConfig from '../../docker/nginx.conf?raw';

describe('System Safety research diagnostics proxy', () => {
  it('proxies only fixed, read-only research diagnostic paths', () => {
    expect(nginxConfig).toContain('location = /diagnostics/research/health');
    expect(nginxConfig).toContain('proxy_pass http://jax-research:8091/health;');
    expect(nginxConfig).toContain('location = /diagnostics/research/v1/memory/banks');
    expect(nginxConfig).toContain('location ^~ /diagnostics/research/v1/memory/banks/');
    expect(nginxConfig).toContain('location = /diagnostics/research/v1/memory/search');
    expect(nginxConfig).toContain('limit_except GET { deny all; }');
    expect(nginxConfig).not.toContain('location /diagnostics/research/ {');
    expect(nginxConfig).not.toContain('proxy_pass $');
  });
});
