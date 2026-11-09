// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { createHyperlinkTooltip } from './hyperlink-tooltip';

afterEach(() => document.body.replaceChildren());

function setup(getTooltip = vi.fn().mockResolvedValue('<div>URL details</div>')) {
    const callbacks = {
        getTooltip, showTooltip: vi.fn(), closeTooltip: vi.fn(), reportError: vi.fn(),
    };
    const anchor = document.createElement('a');
    document.body.append(anchor);
    return { ...callbacks, anchor, tooltip: createHyperlinkTooltip(callbacks) };
}

describe('Notes hyperlink tooltip', () => {
    it('requests shared Go tooltips for mailto links and caches them', async () => {
        const state = setup();
        const href = 'mailto:person@example.com?subject=Hello#section';
        await state.tooltip.show(state.anchor, href, 'fallback');
        state.tooltip.close();
        await state.tooltip.show(state.anchor, href, 'fallback');
        expect(state.getTooltip).toHaveBeenCalledExactlyOnceWith(href);
        expect(state.showTooltip).toHaveBeenCalledWith('<div>URL details</div>');
    });

    it('caches per anchor and URL, and updates changed URLs', async () => {
        const state = setup();
        await state.tooltip.show(state.anchor, 'https://example.com', 'fallback');
        await state.tooltip.show(state.anchor, 'https://example.com', 'fallback');
        expect(state.getTooltip).toHaveBeenCalledTimes(1);
        expect(state.showTooltip).toHaveBeenCalledWith('<div>URL details</div>');
        state.tooltip.close();
        await state.tooltip.show(state.anchor, 'https://example.com', 'fallback');
        expect(state.getTooltip).toHaveBeenCalledTimes(1);
        await state.tooltip.show(state.anchor, 'https://example.com/changed', 'fallback');
        expect(state.getTooltip).toHaveBeenCalledTimes(2);
    });

    it('ignores a late response after leaving', async () => {
        let resolve;
        const state = setup(vi.fn(() => new Promise(r => { resolve = r; })));
        const pending = state.tooltip.show(state.anchor, 'https://example.com', '');
        await Promise.resolve();
        state.tooltip.close();
        resolve('late HTML');
        await pending;
        expect(state.showTooltip).not.toHaveBeenCalled();
    });

    it('ignores a late response after switching to an internal link', async () => {
        let resolve;
        const state = setup(vi.fn(() => new Promise(r => { resolve = r; })));
        const pending = state.tooltip.show(state.anchor, 'https://example.com', '');
        await Promise.resolve();
        const other = document.createElement('a');
        document.body.append(other);
        await state.tooltip.show(other, 'ttyphoon://ai?prompt=test', 'New AI query');
        resolve('late HTML');
        await pending;
        expect(state.showTooltip).toHaveBeenCalledExactlyOnceWith('New AI query');
    });

    it('reports errors and allows retry instead of caching failure', async () => {
        const state = setup(vi.fn().mockRejectedValueOnce(new Error('bridge error')).mockResolvedValue('HTML'));
        await state.tooltip.show(state.anchor, 'file:///tmp/a', '');
        expect(state.reportError).toHaveBeenCalled();
        await state.tooltip.show(state.anchor, 'file:///tmp/a', '');
        expect(state.getTooltip).toHaveBeenCalledTimes(2);
        expect(state.showTooltip).toHaveBeenCalledWith('HTML');
    });
});
