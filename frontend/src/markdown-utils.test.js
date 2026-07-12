import { describe, expect, it, vi } from 'vitest';

vi.mock('../wailsjs/go/main/WApp', () => ({
    GetImage: vi.fn(() => Promise.resolve('')),
    GetCustomRegexp: vi.fn(() => Promise.resolve([])),
    HyperlinkOpenWithDefault: vi.fn(() => Promise.resolve()),
}));

vi.mock('./fullscreen-image-overlay', () => ({
    showFullscreenImageOverlay: vi.fn(),
}));

vi.mock('mermaid', () => ({
    default: {
        initialize: vi.fn(),
        render: vi.fn(() => Promise.resolve({ svg: '<svg></svg>' })),
    },
}));

import { applyMarkdownImageAltSizing, parseMarkdownImageAltSizing } from './markdown-utils.js';

describe('markdown image alt sizing', () => {
    it('keeps alt text unchanged when no colon is present', () => {
        const parsed = parseMarkdownImageAltSizing('example');
        expect(parsed.altText).toBe('example');
        expect(parsed.sizing).toBeNull();
    });

    it('parses percentage sizing after a colon and preserves pre-colon alt text', () => {
        const parsed = parseMarkdownImageAltSizing('example:20%');
        expect(parsed.altText).toBe('example');
        expect(parsed.sizing).toEqual({ unit: '%', value: 20 });
    });

    it('parses pixel sizing after a colon and preserves pre-colon alt text', () => {
        const parsed = parseMarkdownImageAltSizing('example:20px');
        expect(parsed.altText).toBe('example');
        expect(parsed.sizing).toEqual({ unit: 'px', value: 20 });
    });

    it('treats unknown suffixes as no sizing while preserving pre-colon alt text', () => {
        const parsed = parseMarkdownImageAltSizing('example:unknown');
        expect(parsed.altText).toBe('example:unknown');
        expect(parsed.sizing).toBeNull();
    });

    it('supports empty alt text with sizing token', () => {
        const parsed = parseMarkdownImageAltSizing(':20%');
        expect(parsed.altText).toBe('');
        expect(parsed.sizing).toEqual({ unit: '%', value: 20 });
    });

    it('applies sizing styles and preserves aspect-ratio behavior', () => {
        const container = document.createElement('div');
        container.innerHTML = [
            '<img id="img-original" alt="example" src="example.jpg">',
            '<img id="img-percent" alt="example:20%" src="example.jpg">',
            '<img id="img-px" alt="example:20px" src="example.jpg">',
            '<img id="img-empty-alt" alt=":30%" src="example.jpg">',
            '<img id="img-invalid" alt="example:abc" src="example.jpg">',
        ].join('');

        applyMarkdownImageAltSizing(container);

        const original = container.querySelector('#img-original');
        expect(original.alt).toBe('example');
        expect(original.style.maxWidth).toBe('none');
        expect(original.style.maxHeight).toBe('none');
        expect(original.style.width).toBe('auto');
        expect(original.style.height).toBe('auto');

        const percent = container.querySelector('#img-percent');
        expect(percent.alt).toBe('example');
        expect(percent.style.maxWidth).toBe('20vw');
        expect(percent.style.maxHeight).toBe('20vh');
        expect(percent.style.width).toBe('auto');
        expect(percent.style.height).toBe('auto');

        const px = container.querySelector('#img-px');
        expect(px.alt).toBe('example');
        expect(px.style.maxWidth).toBe('20px');
        expect(px.style.maxHeight).toBe('20px');
        expect(px.style.width).toBe('auto');
        expect(px.style.height).toBe('auto');

        const emptyAlt = container.querySelector('#img-empty-alt');
        expect(emptyAlt.alt).toBe('');
        expect(emptyAlt.style.maxWidth).toBe('30vw');
        expect(emptyAlt.style.maxHeight).toBe('30vh');

        const invalid = container.querySelector('#img-invalid');
        expect(invalid.alt).toBe('example:abc');
        expect(invalid.style.maxWidth).toBe('none');
        expect(invalid.style.maxHeight).toBe('none');
    });
});
