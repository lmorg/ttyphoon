import { afterEach, describe, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import postcss from 'postcss';
import { discardPrintDocument, printDocument } from './notes-print.js';

afterEach(() => {
    window.dispatchEvent(new Event('afterprint'));
    document.body.replaceChildren();
});

describe('Notes print document', () => {
    it('reduces code blocks and highlighted tokens by one extra point without nested em sizing', () => {
        const stylesheet = postcss.parse(readFileSync('./src/notes.css', 'utf8'));
        const selectors = ['#notes-print-root pre', '#notes-print-root pre code', '#notes-print-root pre code *'];
        const sizes = new Map();
        stylesheet.walkRules((rule) => {
            for (const selector of rule.selectors) {
                if (!selectors.includes(selector)) continue;
                rule.walkDecls('font-size', (declaration) => {
                    expect(rule.parent.params).toBe('print');
                    expect(declaration.important).toBe(true);
                    sizes.set(selector, declaration.value);
                });
            }
        });
        for (const selector of selectors) {
            expect(sizes.get(selector)).toBe('max(1pt, calc(var(--notes-print-font-size) - 4pt))');
        }
    });

    it('prints the source filename as a separate text line before the code', async () => {
        await printDocument({
            source: 'const value = 1;', filename: 'notes <draft> & review.md',
            dismissMenu: vi.fn(), print: vi.fn(),
        });
        const root = document.getElementById('notes-print-root');
        expect(root.firstElementChild.className).toBe('notes-print-filename');
        expect(root.firstElementChild.textContent).toBe('notes <draft> & review.md');
        expect(root.firstElementChild.nextElementSibling.tagName).toBe('PRE');
        expect(root.querySelector('code').textContent).toBe('const value = 1;');
        expect(root.querySelector('draft')).toBeNull();
    });

    it('prints checked and unchecked checkbox outlines in foreground without a fill', () => {
        const stylesheet = postcss.parse(readFileSync('./src/notes.css', 'utf8'));
        const rules = new Map();
        stylesheet.walkRules((rule) => {
            if (rule.selector.startsWith('#notes-print-root input[type="checkbox"]')) {
                expect(rule.parent.name).toBe('media');
                expect(rule.parent.params).toBe('print');
                rules.set(rule.selector, Object.fromEntries(rule.nodes.map((declaration) => [declaration.prop, declaration.value])));
            }
        });
        expect(rules.get('#notes-print-root input[type="checkbox"]')).toMatchObject({
            color: 'var(--fg)', 'border-color': 'var(--fg)', background: 'transparent', opacity: '1',
        });
        expect(rules.get('#notes-print-root input[type="checkbox"]:checked::after')).toMatchObject({
            color: 'var(--fg)', 'font-size': 'calc(1em + 2pt)',
        });
    });

    it('captures original sizes before applying a print-only three-point reduction', async () => {
        await printDocument({
            source: 'font sizes', dismissMenu: vi.fn(),
            renderMarkdown: async (container) => {
                container.style.fontSize = '18px';
                container.innerHTML = '<h1 style="font-size:32px">Heading <span style="font-size:32px">nested</span></h1><p style="font-size:18px">Body</p><pre style="font-size:16px"><code style="font-size:16px">source</code></pre>';
            },
            print: () => {
                const root = document.getElementById('notes-print-root');
                expect(root.style.getPropertyValue('--notes-print-font-size')).toBe('18px');
                expect(root.querySelector('h1').style.getPropertyValue('--notes-print-font-size')).toBe('32px');
                expect(root.querySelector('span').style.getPropertyValue('--notes-print-font-size')).toBe('32px');
                expect(root.querySelector('code').style.getPropertyValue('--notes-print-font-size')).toBe('16px');
            },
        });
        const stylesheet = postcss.parse(readFileSync('./src/notes.css', 'utf8'));
        let reduction;
        stylesheet.walkRules((rule) => {
            if (rule.selectors.includes('#notes-print-root *')) {
                expect(rule.parent.params).toBe('print');
                reduction = rule.nodes.find((declaration) => declaration.prop === 'font-size');
            }
        });
        expect(reduction.value).toBe('max(1pt, calc(var(--notes-print-font-size) - 3pt))');
        expect(reduction.important).toBe(true);
    });

    it('scopes the supplied Display-P3 palette to print media and adds page margins and padding', () => {
        const stylesheet = postcss.parse(readFileSync('./src/notes.css', 'utf8'));
        let palette;
        let pageMargin;
        let pageSize;
        let firstPageTopMargin;
        let padding;
        stylesheet.walkRules((rule) => {
            if (rule.selector === 'html.notes-print-ready') {
                expect(rule.parent.name).toBe('media');
                expect(rule.parent.params).toBe('print');
                palette = Object.fromEntries(rule.nodes.map((declaration) => [declaration.prop, declaration.value]));
            }
            if (rule.selector === 'html.notes-print-ready #notes-print-root') {
                padding = rule.nodes.find((declaration) => declaration.prop === 'padding')?.value;
            }
        });
        stylesheet.walkAtRules('page', (rule) => {
            if (rule.params === ':first') {
                firstPageTopMargin = rule.nodes.find((declaration) => declaration.prop === 'margin-top')?.value;
                expect(rule.nodes.filter((declaration) => declaration.type === 'decl')).toHaveLength(0);
                return;
            }
            pageMargin = rule.nodes.find((declaration) => declaration.prop === 'margin')?.value;
            pageSize = rule.nodes.find((declaration) => declaration.prop === 'size')?.value;
        });
        expect(palette['--fg']).toBe('color(display-p3 0 0 0)');
        expect(palette['--bg']).toBe('color(display-p3 0.99607843160629272 1 1)');
        expect(palette['--link']).toBe('color(display-p3 0.35340332984924316 0.60392862558364868 0.97110134363174438)');
        expect(palette['--red']).toBe('color(display-p3 0.73696267604827881 0.2651408314704895 0.21598273515701294)');
        expect(palette['--accent']).toBe('var(--blue)');
        expect(pageMargin).toBe('5mm 0');
        expect(firstPageTopMargin).toBeUndefined();
        expect(pageSize).toBe('portrait');
        expect(padding).toBe('5mm');
    });

    it('waits for menu dismissal and coalesces repeated print requests', async () => {
        let finishAnimation;
        const menu = document.createElement('div');
        menu.className = 'tty-menu';
        menu.getAnimations = () => [{ finished: new Promise((resolve) => { finishAnimation = resolve; }) }];
        document.body.appendChild(menu);
        const print = vi.fn();
        const request = printDocument({ source: 'first', dismissMenu: vi.fn(), print });
        expect(printDocument({ source: 'second', dismissMenu: vi.fn(), print })).toBe(request);
        await Promise.resolve();
        expect(print).not.toHaveBeenCalled();
        expect(document.getElementById('notes-print-root')).toBeNull();
        finishAnimation();
        await request;
        expect(print).toHaveBeenCalledOnce();
        expect(document.getElementById('notes-print-root').textContent).toBe('first');
        discardPrintDocument();
        expect(document.getElementById('notes-print-root')).toBeNull();
    });

    it('prints the full source as text without changing the live application', async () => {
        const app = document.createElement('main');
        app.textContent = 'Visible editor lines only';
        document.body.appendChild(app);
        const source = Array.from({ length: 1000 }, (_, index) => `line ${index}: <b>source</b>`).join('\n');
        const dismissMenu = vi.fn();
        const print = vi.fn(() => {
            expect(dismissMenu).toHaveBeenCalledOnce();
            expect(document.documentElement.classList.contains('notes-print-ready')).toBe(true);
            expect(document.querySelector('#notes-print-root code').textContent).toBe(source);
            expect(document.querySelector('#notes-print-root b')).toBeNull();
        });
        await printDocument({ source, dismissMenu, print });
        expect(print).toHaveBeenCalledOnce();
        expect(app.textContent).toBe('Visible editor lines only');
        expect(document.getElementById('notes-print-root')).not.toBeNull();
        window.dispatchEvent(new Event('afterprint'));
        expect(document.getElementById('notes-print-root')).toBeNull();
        expect(document.documentElement.classList.contains('notes-print-ready')).toBe(false);
    });

    it('retains content when preview fires afterprint before the native PDF operation finishes', async () => {
        let completeOperation;
        let previewReady;
        const preview = new Promise((resolve) => { previewReady = resolve; });
        const request = printDocument({
            source: 'Content required by both preview and saved PDF',
            dismissMenu: vi.fn(),
            print: () => new Promise((resolve) => {
                completeOperation = resolve;
                previewReady();
            }),
        });
        await preview;
        for (let cycle = 0; cycle < 2; cycle++) {
            window.dispatchEvent(new Event('afterprint'));
            expect(document.getElementById('notes-print-root')?.textContent).toBe('Content required by both preview and saved PDF');
            expect(document.documentElement.classList.contains('notes-print-ready')).toBe(true);
        }
        completeOperation();
        await request;
        window.dispatchEvent(new Event('afterprint'));
        expect(document.getElementById('notes-print-root')).toBeNull();
    });

    it('waits for complete Markdown rendering and image decoding before printing', async () => {
        const decoded = vi.fn();
        const renderMarkdown = vi.fn(async (container, source) => {
            expect(source).toBe('# Entire document');
            container.innerHTML = '<h1>Entire document</h1><p>Last paragraph</p><details><summary>More</summary>Collapsed content</details><img>';
            container.querySelector('img').decode = async () => { decoded(); };
        });
        await printDocument({
            source: '# Entire document', renderMarkdown, dismissMenu: vi.fn(),
            print: vi.fn(() => {
                expect(decoded).toHaveBeenCalledOnce();
                expect(document.querySelector('#notes-print-root p').textContent).toBe('Last paragraph');
                expect(document.querySelector('#notes-print-root details').open).toBe(true);
            }),
        });
    });

    it('cleans up failures and replaces retained native print documents on the next request', async () => {
        await printDocument({ source: 'first', dismissMenu: vi.fn(), print: vi.fn() });
        await printDocument({ source: 'second', dismissMenu: vi.fn(), print: vi.fn() });
        expect(document.querySelectorAll('#notes-print-root')).toHaveLength(1);
        expect(document.getElementById('notes-print-root').textContent).toBe('second');
        await expect(printDocument({
            source: 'failed', dismissMenu: vi.fn(), print: vi.fn(() => Promise.reject(new Error('print failed'))),
        })).rejects.toThrow('print failed');
        expect(document.getElementById('notes-print-root')).toBeNull();
        expect(document.documentElement.classList.contains('notes-print-ready')).toBe(false);
    });
});