// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { marked } from 'marked';
import { createAIPipelineFormatter } from './ai_pipeline_formatter';
import { captureRenderedElements } from './copy-formatted';

afterEach(() => {
    document.body.replaceChildren();
    vi.unstubAllGlobals();
});

describe('rich AI response export', () => {
    it('captures live top-level answers with current table values and filters', async () => {
        const container = document.createElement('div');
        document.body.append(container);
        const formatter = createAIPipelineFormatter(container, { marked });
        formatter.startJob('');
        for (const block of [
            { blockId: 'thinking', kind: 'thinking', content: 'Excluded reasoning' },
            { blockId: 'tool', kind: 'tool-call', content: 'Excluded tool' },
            { blockId: 'nested', kind: 'text', parentId: 'tool', content: 'Excluded child' },
            { blockId: 'one', kind: 'text', content: 'First answer' },
            { blockId: 'two', kind: 'text', content: 'Second answer' },
        ]) {
            formatter.appendBlock({ ...block, status: 'closed' });
        }
        const elements = await formatter.outputElements();
        expect(elements).toHaveLength(2);
        elements[0].innerHTML = '<table><tr><th>Value</th><th>Name</th></tr><tr><td data-formula="=A1+1">42</td><td>Visible</td></tr><tr class="notes-table-row-filtered"><td>99</td><td>Filtered</td></tr></table>';
        const html = captureRenderedElements(await formatter.outputElements()).fragment.innerHTML;
        expect(html).toContain('<th>Value</th><th>Name</th>');
        expect(html).toContain('<td>42</td><td>Visible</td>');
        expect(html).toContain('Second answer');
        expect(html).not.toMatch(/Excluded|Filtered|99|data-formula/);
    });

    it('renders unloaded historical answers without expanding or mutating the screen', async () => {
        vi.stubGlobal('IntersectionObserver', class {
            observe() {}
            disconnect() {}
        });
        const container = document.createElement('div');
        document.body.append(container);
        const load = vi.fn(async (block) => block.blockId === 'first' ? '**Historical first**' : 'Final');
        const process = vi.fn(async () => {});
        const formatter = createAIPipelineFormatter(container, { marked, processMarkdownContainer: process });
        formatter.setBlocks([
            { blockId: 'first', kind: 'text', status: 'closed', size: 20 },
            { blockId: 'last', kind: 'text', status: 'closed', size: 5 },
        ], load, { collapseIntermediate: true, prompt: 'Prompt' });
        const shell = container.querySelector('[data-block-id="first"]');
        expect(shell.textContent).toBe('');
        const elements = await formatter.outputElements();
        const html = captureRenderedElements(elements).fragment.innerHTML;
        expect(html).toContain('<strong>Historical first</strong>');
        expect(html).toContain('Final');
        expect(shell.textContent).toBe('');
        expect(container.querySelector('details').open).toBe(false);
        expect(process).toHaveBeenCalledWith(elements[0]);
    });

    it('propagates historical loading errors rather than copying an empty response', async () => {
        vi.stubGlobal('IntersectionObserver', class {
            observe() {}
            disconnect() {}
        });
        const container = document.createElement('div');
        document.body.append(container);
        const formatter = createAIPipelineFormatter(container, { marked });
        formatter.setBlocks([
            { blockId: 'first', kind: 'text', status: 'closed', size: 20 },
            { blockId: 'last', kind: 'text', status: 'closed', size: 5 },
        ], async () => { throw new Error('History unavailable'); }, { collapseIntermediate: true });
        await expect(formatter.outputElements()).rejects.toThrow('History unavailable');
    });
});
