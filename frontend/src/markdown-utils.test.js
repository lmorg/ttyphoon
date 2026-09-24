import { describe, expect, it, vi } from 'vitest';

const { getCustomRegexpMock, getImageMock } = vi.hoisted(() => ({
    getCustomRegexpMock: vi.fn(() => Promise.resolve([])),
    getImageMock: vi.fn(() => Promise.resolve('')),
}));

vi.mock('../wailsjs/go/main/WApp', () => ({
    GetImage: getImageMock,
    GetCustomRegexp: getCustomRegexpMock,
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

import { applyMarkdownImageAltSizing, autoHyperlink, notesAssetURL, parseMarkdownImageAltSizing, processWailsImages } from './markdown-utils.js';

describe('wails image asset rewriting', () => {
    const containerWith = (src) => {
        const container = document.createElement('div');
        container.innerHTML = `<img src="${src}">`;
        return container;
    };

    it('rewrites wails image urls onto the asset endpoint without touching the bridge', () => {
        const container = containerWith('wails://wails/images/diagram.png');

        processWailsImages(container);

        const img = container.querySelector('img');
        expect(img.getAttribute('src')).toBe('/__asset?path=images%2Fdiagram.png');
        expect(img.dataset.originalFilename).toBe('diagram.png');
        expect(getImageMock).not.toHaveBeenCalled();
    });

    it('decodes percent escapes so the handler receives the real filename', () => {
        const container = containerWith('wails://wails/my%20diagram.png');

        processWailsImages(container);

        expect(container.querySelector('img').getAttribute('src'))
            .toBe('/__asset?path=my%20diagram.png');
        expect(container.querySelector('img').dataset.originalFilename).toBe('my diagram.png');
    });

    it('is idempotent so repeated passes do not double-encode the path', () => {
        const container = containerWith('wails://wails.localhost:34115/images/diagram.png');

        processWailsImages(container);
        const first = container.querySelector('img').getAttribute('src');
        processWailsImages(container);

        expect(container.querySelector('img').getAttribute('src')).toBe(first);
    });

    it('leaves non-wails sources alone', () => {
        const container = containerWith('https://example.com/diagram.png');

        processWailsImages(container);

        expect(container.querySelector('img').getAttribute('src')).toBe('https://example.com/diagram.png');
    });

    it('encodes paths that would otherwise break the query string', () => {
        expect(notesAssetURL('a b/c&d.png')).toBe('/__asset?path=a%20b%2Fc%26d.png');
    });
});

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
        expect(original.style.maxWidth).toBe('100%');
        expect(original.style.maxHeight).toBe('none');
        expect(original.style.width).toBe('auto');
        expect(original.style.height).toBe('auto');

        const percent = container.querySelector('#img-percent');
        expect(percent.alt).toBe('example');
        expect(percent.style.maxWidth).toBe('min(100%, 20vw)');
        expect(percent.style.maxHeight).toBe('20vh');
        expect(percent.style.width).toBe('auto');
        expect(percent.style.height).toBe('auto');

        const px = container.querySelector('#img-px');
        expect(px.alt).toBe('example');
        expect(px.style.maxWidth).toBe('min(100%, 20px)');
        expect(px.style.maxHeight).toBe('20px');
        expect(px.style.width).toBe('auto');
        expect(px.style.height).toBe('auto');

        const emptyAlt = container.querySelector('#img-empty-alt');
        expect(emptyAlt.alt).toBe('');
        expect(emptyAlt.style.maxWidth).toBe('min(100%, 30vw)');
        expect(emptyAlt.style.maxHeight).toBe('30vh');

        const invalid = container.querySelector('#img-invalid');
        expect(invalid.alt).toBe('example:abc');
        expect(invalid.style.maxWidth).toBe('100%');
        expect(invalid.style.maxHeight).toBe('none');
    });
});

describe('custom markdown hyperlinks', () => {
    it('rewrites matching text to a ttyphoon URL', async () => {
        getCustomRegexpMock.mockResolvedValueOnce([{
            pattern: '(HAP-[0-9]+)',
            link: 'ttyphoon://ai?prompt=$1&tools=jira',
        }]);
        const container = document.createElement('div');
        container.textContent = 'Review HAP-13379 now';

        await autoHyperlink(container);

        const link = container.querySelector('a');
        expect(link).not.toBeNull();
        expect(link?.textContent).toBe('HAP-13379');
        expect(link?.getAttribute('href')).toBe('ttyphoon://ai?prompt=HAP-13379&tools=jira');
    });

    it('retries when the initial custom-regex lookup is empty', async () => {
        const callsBefore = getCustomRegexpMock.mock.calls.length;
        getCustomRegexpMock
            .mockResolvedValueOnce([])
            .mockResolvedValueOnce([{
                pattern: '(HAP-[0-9]+)',
                link: 'ttyphoon://ai?prompt=$1',
            }]);

        const first = document.createElement('div');
        first.textContent = 'HAP-1';
        await autoHyperlink(first);
        expect(first.querySelector('a')).toBeNull();

        const second = document.createElement('div');
        second.textContent = 'HAP-2';
        await autoHyperlink(second);
        expect(second.querySelector('a')?.getAttribute('href')).toBe('ttyphoon://ai?prompt=HAP-2');
        expect(getCustomRegexpMock).toHaveBeenCalledTimes(callsBefore + 2);
    });
});
