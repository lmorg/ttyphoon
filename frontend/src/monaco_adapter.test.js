import { beforeEach, describe, expect, it, vi } from 'vitest';
import { attachMonacoFontMetricsRefresh, attachMonacoSpellcheckContextMenu } from './monaco_adapter';
import { showSpellcheckSuggestionsPopup } from './spellcheck';

vi.mock('./spellcheck', () => ({ showSpellcheckSuggestionsPopup: vi.fn() }));
vi.mock('./vim-mode', () => ({ MODE_LABELS: {} }));

describe('Monaco webfont metrics', () => {
    let fonts;
    let finishLoading;
    let monaco;
    let editor;

    beforeEach(() => {
        fonts = new EventTarget();
        fonts.ready = new Promise((resolve) => { finishLoading = resolve; });
        monaco = { editor: { remeasureFonts: vi.fn() } };
        editor = { layout: vi.fn() };
    });

    it('invalidates cached fallback widths before laying out once fonts are ready', async () => {
        const handle = attachMonacoFontMetricsRefresh(monaco, editor, fonts);
        expect(monaco.editor.remeasureFonts).not.toHaveBeenCalled();
        finishLoading();
        await fonts.ready;
        expect(monaco.editor.remeasureFonts).toHaveBeenCalledOnce();
        expect(editor.layout).toHaveBeenCalledOnce();
        expect(monaco.editor.remeasureFonts.mock.invocationCallOrder[0])
            .toBeLessThan(editor.layout.mock.invocationCallOrder[0]);
        handle.dispose();
    });

    it('remeasures later font loads and stops listening on disposal', () => {
        const handle = attachMonacoFontMetricsRefresh(monaco, editor, fonts);
        fonts.dispatchEvent(new Event('loadingdone'));
        fonts.dispatchEvent(new Event('loadingdone'));
        expect(monaco.editor.remeasureFonts).toHaveBeenCalledTimes(2);
        expect(editor.layout).toHaveBeenCalledTimes(2);
        handle.dispose();
        fonts.dispatchEvent(new Event('loadingdone'));
        expect(editor.layout).toHaveBeenCalledTimes(2);
    });

    it('does not refresh an editor disposed before initial font loading finishes', async () => {
        const handle = attachMonacoFontMetricsRefresh(monaco, editor, fonts);
        handle.dispose();
        finishLoading();
        await fonts.ready;
        expect(monaco.editor.remeasureFonts).not.toHaveBeenCalled();
        expect(editor.layout).not.toHaveBeenCalled();
    });
});

describe('Monaco spelling context menu', () => {
    let node;
    let editor;
    let findMisspelling;
    let replaceMisspelling;
    let normalMenu;
    let handle;
    const typo = { wordStart: 0, wordLength: 7, misspeltWord: 'mispelt', suggestions: ['misspelt'] };

    beforeEach(() => {
        vi.clearAllMocks();
        document.body.innerHTML = '<div id="editor"></div>';
        node = document.getElementById('editor');
        editor = {
            getDomNode: () => node,
            getTargetAtClientPoint: vi.fn(() => ({ position: { lineNumber: 1, column: 3 } })),
            getModel: () => ({ getOffsetAt: () => 2 }),
        };
        findMisspelling = vi.fn(() => typo);
        replaceMisspelling = vi.fn();
        normalMenu = vi.fn();
        node.addEventListener('contextmenu', normalMenu);
        handle = attachMonacoSpellcheckContextMenu(editor, findMisspelling, replaceMisspelling);
    });

    it('leaves left-click caret placement and selection alone', () => {
        for (const type of ['mousedown', 'click']) {
            const event = new MouseEvent(type, { button: 0, bubbles: true, cancelable: true });
            node.dispatchEvent(event);
            expect(event.defaultPrevented).toBe(false);
        }
        expect(showSpellcheckSuggestionsPopup).not.toHaveBeenCalled();
        expect(findMisspelling).not.toHaveBeenCalled();
    });

    it('opens suggestions on right-click instead of the general editor menu', () => {
        const event = new MouseEvent('contextmenu', { button: 2, clientX: 40, clientY: 60, bubbles: true, cancelable: true });
        node.dispatchEvent(event);
        expect(editor.getTargetAtClientPoint).toHaveBeenCalledWith(40, 60);
        expect(findMisspelling).toHaveBeenCalledWith(2);
        expect(event.defaultPrevented).toBe(true);
        expect(normalMenu).not.toHaveBeenCalled();
        expect(showSpellcheckSuggestionsPopup).toHaveBeenCalledWith(40, 60, typo, expect.any(Function));
        showSpellcheckSuggestionsPopup.mock.calls[0][3]('misspelt');
        expect(replaceMisspelling).toHaveBeenCalledWith(typo, 'misspelt');
    });

    it('passes right-clicks outside misspellings through to the normal menu', () => {
        findMisspelling.mockReturnValue(null);
        const event = new MouseEvent('contextmenu', { button: 2, bubbles: true, cancelable: true });
        node.dispatchEvent(event);
        expect(event.defaultPrevented).toBe(false);
        expect(normalMenu).toHaveBeenCalledOnce();
        expect(showSpellcheckSuggestionsPopup).not.toHaveBeenCalled();
    });

    it('ignores editor margins and removes its listener on disposal', () => {
        editor.getTargetAtClientPoint.mockReturnValue(null);
        node.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true }));
        expect(findMisspelling).not.toHaveBeenCalled();
        handle.dispose();
        editor.getTargetAtClientPoint.mockClear();
        node.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true }));
        expect(editor.getTargetAtClientPoint).not.toHaveBeenCalled();
        expect(normalMenu).toHaveBeenCalledTimes(2);
    });
});