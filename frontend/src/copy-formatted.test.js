// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { captureRenderedElement, captureRenderedElements, captureRenderedSelection, prepareRenderedCopy, createRenderedSelectionCopier, imageToPNG } from './copy-formatted';

function select(container, start = container, startOffset = 0, end = container, endOffset = container.childNodes.length) {
    const range = document.createRange();
    range.setStart(start, startOffset);
    range.setEnd(end, endOffset);
    const selection = window.getSelection();
    selection.removeAllRanges();
    selection.addRange(range);
    return selection;
}

function root(html) {
    const element = document.createElement('article');
    element.className = 'markdown-body';
    element.innerHTML = html;
    document.body.append(element);
    return element;
}

beforeEach(() => {
    window.getSelection().removeAllRanges();
    document.body.replaceChildren();
});

afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
});

it('rasterizes images and SVG diagrams to PNG through the real canvas path', async () => {
    class TestImage {
        naturalWidth = 40;
        naturalHeight = 20;
        set src(value) {
            this.source = value;
            queueMicrotask(() => this.onload?.());
        }
    }
    vi.stubGlobal('Image', TestImage);
    const drawImage = vi.fn();
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({ drawImage });
    vi.spyOn(HTMLCanvasElement.prototype, 'toDataURL').mockReturnValue('data:image/png;base64,PNG');
    for (const image of [{ source: 'diagram.png' }, { svg: '<svg xmlns="http://www.w3.org/2000/svg"><text>Diagram</text></svg>' }]) {
        expect(await imageToPNG(image)).toBe('data:image/png;base64,PNG');
    }
    expect(drawImage).toHaveBeenCalledTimes(2);
    expect(drawImage.mock.calls[1][0].source).toMatch(/^data:image\/svg\+xml;base64,/);
});

describe('rendered selection capture', () => {
    it('keeps partially selected semantic ancestors, not unselected text', async () => {
        const container = root('<h2>Title</h2><p>before <strong>bold text</strong> after</p><p>outside</p>');
        const bold = container.querySelector('strong').firstChild;
        const tail = container.querySelector('p').lastChild;
        select(container, bold, 5, tail, 3);
        const result = await prepareRenderedCopy(captureRenderedSelection(container));
        expect(result.html).toBe('<p><strong>text</strong> af</p>');
    });

    it('keeps table DOM order and computed values, excludes filtered rows and editing UI', async () => {
        const container = root(`<table><thead><tr><th>Value</th><th>Name</th></tr></thead><tbody>
            <tr><td data-formula="=1+1"><span class="notes-table-cell-wrap"><span>2</span><span class="notes-cellref">A1</span></span></td><td>Beta</td></tr>
            <tr class="notes-table-row-filtered"><td>99</td><td>Hidden</td></tr>
            <tr><td>1</td><td>Alpha</td></tr></tbody></table><div class="notes-table-sql-filter-footer">SQL expression</div>`);
        select(container);
        const { html } = await prepareRenderedCopy(captureRenderedSelection(container));
        const output = document.createElement('div');
        output.innerHTML = html;
        expect([...output.querySelectorAll('tr')].map((row) => row.textContent)).toEqual(['ValueName', '2Beta', '1Alpha']);
        expect(html).not.toMatch(/data-formula|=1\+1|A1|Hidden|SQL expression/);
    });

    it('copies just selected table cells, with their table structure', async () => {
        const container = root('<table><tbody><tr><td>first</td><td>second</td><td>third</td></tr><tr><td>outside</td></tr></tbody></table>');
        const text = container.querySelectorAll('td')[1].firstChild;
        select(container, text, 1, text, 5);
        const { html } = await prepareRenderedCopy(captureRenderedSelection(container));
        expect(html).toBe('<table><tbody><tr><td>econ</td></tr></tbody></table>');
    });

    it('retains empty selected cells so later columns do not shift', async () => {
        const container = root('<table><tbody><tr><td></td><td>Middle</td><td></td></tr><tr><td></td><td></td><td></td></tr></tbody></table>');
        select(container);
        const { html } = await prepareRenderedCopy(captureRenderedSelection(container));
        expect(html).toBe(container.innerHTML);
    });

    it('preserves ordered list numbering and code whitespace while omitting modal line numbers', async () => {
        const container = root('<ol start="4"><li>outside</li><li>selected</li></ol><div class="markdown-modal-code-lines">1\n2</div><pre><code> a\n b</code></pre>');
        const text = container.querySelectorAll('li')[1].firstChild;
        select(container, text, 0, container, container.childNodes.length);
        const { html } = await prepareRenderedCopy(captureRenderedSelection(container));
        expect(html).toBe('<ol start="5"><li>selected</li></ol><pre><code> a\n b</code></pre>');
    });

    it('exports SVG viewBox dimensions rather than the scaled panel size', () => {
        const container = root('<svg viewBox="0 0 1800 1200" width="100%" style="max-width: 100%; width: 300px; height: 200px"><text>Diagram</text></svg>');
        const svg = container.querySelector('svg');
        vi.spyOn(svg, 'getBoundingClientRect').mockReturnValue({ width: 300, height: 200 });
        const snapshot = captureRenderedElement(svg);
        const exported = new DOMParser().parseFromString(snapshot.images[0].svg, 'image/svg+xml').documentElement;
        expect(exported.getAttribute('width')).toBe('1800');
        expect(exported.getAttribute('height')).toBe('1200');
        expect(exported.style.width).toBe('');
        expect(exported.style.maxWidth).toBe('');
        expect(exported.getAttribute('viewBox')).toBe('0 0 1800 1200');
    });

    it('uses explicit SVG dimensions when no valid viewBox exists', () => {
        const container = root('<svg width="1600px" height="900" viewBox="invalid"></svg>');
        const svg = container.querySelector('svg');
        vi.spyOn(svg, 'getBoundingClientRect').mockReturnValue({ width: 320, height: 180 });
        const snapshot = captureRenderedElement(svg);
        const exported = new DOMParser().parseFromString(snapshot.images[0].svg, 'image/svg+xml').documentElement;
        expect(exported.getAttribute('width')).toBe('1600');
        expect(exported.getAttribute('height')).toBe('900');
    });

    it('captures Mermaid and graph images atomically and inlines PNGs', async () => {
        const container = root('<div class="mermaid-diagram"><svg viewBox="0 0 40 20"><title>Flow</title><text>A</text></svg></div><img alt="Chart" src="chart.svg">');
        select(container);
        const rasterize = vi.fn().mockResolvedValue('data:image/png;base64,PNG');
        const snapshot = captureRenderedSelection(container);
        expect(snapshot.images).toHaveLength(2);
        const result = await prepareRenderedCopy(snapshot, rasterize);
        expect(rasterize.mock.calls[0][0].svg).toContain('<svg');
        expect(result.html).not.toContain('<svg');
        expect(result.html.match(/data:image\/png;base64,PNG/g)).toHaveLength(2);
        expect(result.warnings).toEqual([]);
    });

    it('uses image alt text/URL and reports failures without discarding other text', async () => {
        const container = root('<p>Keep this <img alt="Diagram" src="diagram.png"></p>');
        select(container);
        const result = await prepareRenderedCopy(captureRenderedSelection(container), vi.fn().mockRejectedValue(new Error('Offline')));
        expect(result.html).toBe('<p>Keep this ![Diagram](diagram.png)</p>');
        expect(result.warnings).toEqual(['Diagram: Offline']);
    });

    it('returns no snapshot for collapsed or unrelated selections', () => {
        const container = root('<p>Text</p>');
        expect(captureRenderedSelection(container)).toBeNull();
        const other = root('<p>Other</p>');
        select(other);
        expect(captureRenderedSelection(container)).toBeNull();
    });
});

describe('copy actions', () => {
    it('shows upload fallback warnings returned by the Go clipboard writer', async () => {
        const container = root('<p>Remaining content</p>');
        const notify = vi.fn();
        const copier = createRenderedSelectionCopier({
            write: vi.fn().mockResolvedValue(['Chart: S3 image upload failed (HTTP 403)']),
            notify,
        });
        await copier.copy(captureRenderedElement(container));
        expect(notify).toHaveBeenCalledWith(
            'Copied with image fallback: Chart: S3 image upload failed (HTTP 403)', 'warn',
        );
    });

    it('combines full rendered response blocks without copying intervening content', async () => {
        const container = root('<section><p>First</p><img alt="Chart" src="chart.png"></section><aside>Tool output</aside><section><p>Second</p></section>');
        const snapshot = captureRenderedElements(container.querySelectorAll('section'));
        const result = await prepareRenderedCopy(snapshot, vi.fn().mockResolvedValue('data:image/png;base64,test'));
        expect(result.html).toBe('<p>First</p><img alt="Chart" data-copy-source="chart.png" src="data:image/png;base64,test"><p>Second</p>');
        expect(captureRenderedElements([])).toBeNull();
    });

    it('uses the table writer without changing selection writes or their shared order', async () => {
        const container = root('<p>Selected</p>');
        const calls = [];
        const write = vi.fn(async () => { calls.push('selection'); });
        const tableWrite = vi.fn(async () => { calls.push('table'); });
        const copier = createRenderedSelectionCopier({ write, notify: vi.fn() });
        const first = copier.copy(captureRenderedElement(container), tableWrite);
        const second = copier.copy(captureRenderedElement(container));
        await Promise.all([first, second]);
        expect(calls).toEqual(['table', 'selection']);
        expect(tableWrite).toHaveBeenCalledOnce();
        expect(write).toHaveBeenCalledOnce();
    });

    it('snapshots the selection when a context menu opens', async () => {
        const container = root('<p><strong>Selected</strong> outside</p>');
        const text = container.querySelector('strong').firstChild;
        select(container, text, 0, text, text.length);
        const write = vi.fn().mockResolvedValue();
        const copier = createRenderedSelectionCopier({ write, notify: vi.fn() });
        const menu = copier.menuItem(container);
        window.getSelection().removeAllRanges();
        container.innerHTML = '<p>Changed</p>';
        await menu.onSelect();
        expect(write).toHaveBeenCalledExactlyOnceWith('<p><strong>Selected</strong></p>');
    });

    it('handles native copy only for rendered selections, leaving editors alone', async () => {
        const container = root('<p><em>Selected</em></p><div contenteditable="true">Editor</div>');
        const write = vi.fn().mockResolvedValue();
        const copier = createRenderedSelectionCopier({ write, notify: vi.fn() });
        const text = container.querySelector('em').firstChild;
        select(container, text, 0, text, text.length);
        const event = new Event('copy', { cancelable: true });
        copier.handleCopy(event);
        await copier.copy(null);
        await new Promise((resolve) => setTimeout(resolve, 0));
        expect(event.defaultPrevented).toBe(true);
        expect(write).toHaveBeenCalledExactlyOnceWith('<p><em>Selected</em></p>');
        const editorText = container.querySelector('[contenteditable]').firstChild;
        select(container, editorText, 0, editorText, editorText.length);
        const editorEvent = new Event('copy', { cancelable: true });
        copier.handleCopy(editorEvent);
        expect(editorEvent.defaultPrevented).toBe(false);
    });

    it('handles selections crossing nested rendered blocks', async () => {
        const container = root('<section class="markdown-body"><p>First</p></section><section class="markdown-body"><p>Second</p></section>');
        const paragraphs = container.querySelectorAll('p');
        select(container, paragraphs[0].firstChild, 0, paragraphs[1].firstChild, 6);
        const write = vi.fn().mockResolvedValue();
        const copier = createRenderedSelectionCopier({ write, notify: vi.fn() });
        copier.handleCopy(new Event('copy', { cancelable: true }));
        await new Promise((resolve) => setTimeout(resolve, 0));
        expect(write).toHaveBeenCalledExactlyOnceWith('<p>First</p><p>Second</p>');
    });

    it('does not steal editor copy when an old rendered selection remains', () => {
        const container = root('<p>Old selection</p><textarea>Editor text</textarea>');
        const text = container.querySelector('p').firstChild;
        select(container, text, 0, text, text.length);
        const write = vi.fn();
        const copier = createRenderedSelectionCopier({ write, notify: vi.fn() });
        const editor = container.querySelector('textarea');
        const event = new Event('copy', { cancelable: true });
        Object.defineProperty(event, 'target', { value: editor });
        copier.handleCopy(event);
        expect(event.defaultPrevented).toBe(false);
        expect(write).not.toHaveBeenCalled();
    });

    it('warns about image fallback and reports clipboard failures', async () => {
        const container = root('<p>Text <img alt="Image" src="broken.png"></p>');
        select(container);
        const notify = vi.fn();
        const write = vi.fn().mockResolvedValueOnce().mockRejectedValueOnce(new Error('Clipboard unavailable'));
        const copier = createRenderedSelectionCopier({
            write, notify, rasterize: vi.fn().mockRejectedValue(new Error('Image unavailable')),
        });
        await copier.copy(captureRenderedSelection(container));
        expect(notify).toHaveBeenCalledWith(expect.stringContaining('Copied with image fallback'), 'warn');
        await copier.copy(captureRenderedSelection(container));
        expect(notify).toHaveBeenCalledWith('Failed to copy selection: Clipboard unavailable', 'error');
    });
});
