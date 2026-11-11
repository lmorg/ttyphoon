import { beforeEach, describe, expect, it, vi } from 'vitest';
import { attachMonacoEditorHooks, attachMonacoFontMetricsRefresh, attachMonacoSpellcheckContextMenu } from './monaco_adapter';
import { showSpellcheckSuggestionsPopup } from './spellcheck';

vi.mock('./spellcheck', () => ({ showSpellcheckSuggestionsPopup: vi.fn() }));
vi.mock('./vim-mode', () => ({ MODE_LABELS: {} }));

describe('Monaco Escape priority', () => {
    let node;
    let editor;
    let hover;
    let options;
    let toggleVim;
    let nativeKeydown;
    let handle;

    beforeEach(() => {
        document.body.innerHTML = '<div id="editor"><textarea></textarea><div class="suggest-widget visible"></div></div>';
        node = document.getElementById('editor');
        hover = { isHoverVisible: true, hideContentHover: vi.fn(() => { hover.isHoverVisible = false; }) };
        editor = {
            getDomNode: () => node,
            getContribution: () => hover,
            onKeyDown: () => ({ dispose: vi.fn() }),
            onDidType: () => ({ dispose: vi.fn() }),
            trigger: vi.fn(() => node.querySelector('.suggest-widget').classList.remove('visible')),
        };
        options = { onEscape: vi.fn(() => false) };
        toggleVim = vi.fn(() => true);
        nativeKeydown = vi.fn();
        node.querySelector('textarea').addEventListener('keydown', nativeKeydown);
        handle = attachMonacoEditorHooks(editor, options, toggleVim);
    });

    const escape = () => {
        const event = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true });
        document.querySelector('#editor textarea').dispatchEvent(event);
        return event;
    };

    it('dismisses hover, then suggestions, then toggles Vim on separate keypresses', () => {
        expect(escape().defaultPrevented).toBe(true);
        expect(hover.hideContentHover).toHaveBeenCalledOnce();
        expect(editor.trigger).not.toHaveBeenCalled();
        expect(toggleVim).not.toHaveBeenCalled();
        expect(escape().defaultPrevented).toBe(true);
        expect(editor.trigger).toHaveBeenCalledWith('keyboard', 'hideSuggestWidget', null);
        expect(toggleVim).not.toHaveBeenCalled();
        expect(escape().defaultPrevented).toBe(true);
        expect(toggleVim).toHaveBeenCalledOnce();
        expect(nativeKeydown).not.toHaveBeenCalled();
        handle.dispose();
    });

    it('dismisses external Notes tooltips before suggestions without reaching Vim', () => {
        hover.isHoverVisible = false;
        options.onEscape.mockReturnValueOnce(true);
        escape();
        expect(options.onEscape).toHaveBeenCalledOnce();
        expect(editor.trigger).not.toHaveBeenCalled();
        expect(toggleVim).not.toHaveBeenCalled();
        expect(nativeKeydown).not.toHaveBeenCalled();
        escape();
        expect(editor.trigger).toHaveBeenCalledWith('keyboard', 'hideSuggestWidget', null);
        handle.dispose();
    });

    it('dismisses parameter hints before suggestions', () => {
        hover.isHoverVisible = false;
        node.insertAdjacentHTML('beforeend', '<div class="parameter-hints-widget visible"></div>');
        escape();
        expect(editor.trigger).toHaveBeenCalledWith('keyboard', 'closeParameterHints', null);
        expect(toggleVim).not.toHaveBeenCalled();
        expect(nativeKeydown).not.toHaveBeenCalled();
        handle.dispose();
    });

    it('allows unhandled editor keys through and removes its capture handler on disposal', () => {
        hover.isHoverVisible = false;
        node.querySelector('.suggest-widget').remove();
        toggleVim.mockReturnValue(false);
        expect(escape().defaultPrevented).toBe(false);
        expect(nativeKeydown).toHaveBeenCalledOnce();
        handle.dispose();
        toggleVim.mockClear();
        escape();
        expect(toggleVim).not.toHaveBeenCalled();
        expect(nativeKeydown).toHaveBeenCalledTimes(2);
    });
});

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