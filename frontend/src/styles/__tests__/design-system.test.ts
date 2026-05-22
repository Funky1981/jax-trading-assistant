import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import { tokens } from '../tokens';

const frontendRoot = resolve(__dirname, '../../..');

function readFrontendFile(path: string) {
  return readFileSync(resolve(frontendRoot, path), 'utf8');
}

describe('design-system foundation', () => {
  it('keeps runtime CSS variables as the token value source', () => {
    const css = readFrontendFile('src/index.css');

    for (const variableName of tokens.cssVariableNames) {
      expect(css).toContain(`--${variableName}:`);
    }

    expect(css).toContain('--chart-up:');
    expect(css).toContain('--chart-down:');
    expect(css).toContain('--risk:');
    expect(css).toContain('--elevation-sm:');
    expect(css).toContain('--font-sans:');
  });

  it('keeps TypeScript tokens mapped to CSS variables instead of duplicate hex values', () => {
    expect(tokens.colors.bg).toBe('hsl(var(--background))');
    expect(tokens.colors.surface).toBe('hsl(var(--card))');
    expect(tokens.colors.accent).toBe('hsl(var(--accent))');
    expect(tokens.chart.up).toBe('hsl(var(--chart-up))');
    expect(tokens.elevation.sm).toBe('var(--elevation-sm)');
  });

  it('declares Storybook scripts and installed addons', () => {
    const packageJson = JSON.parse(readFrontendFile('package.json')) as {
      scripts: Record<string, string>;
      devDependencies?: Record<string, string>;
    };

    expect(packageJson.scripts.storybook).toBe('storybook dev -p 6006');
    expect(packageJson.scripts['build-storybook']).toBe('storybook build');

    expect(packageJson.devDependencies).toMatchObject({
      storybook: expect.any(String),
      '@storybook/react-vite': expect.any(String),
      '@storybook/addon-essentials': expect.any(String),
      '@storybook/addon-interactions': expect.any(String),
      '@storybook/addon-a11y': expect.any(String),
    });
  });
});
