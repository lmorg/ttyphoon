import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest';
import { disableNativeTextAssistanceIn } from './text-input.js';

// ─── Mocks ────────────────────────────────────────────────────────────────────

const notesSpellCheckMock = vi.fn(() => Promise.resolve([]));
const showLocalMenuMock = vi.fn();

vi.mock('./wails-bindings.js', () => ({
    NotesSpellCheck: notesSpellCheckMock,
}));

vi.mock('./popup_menu', () => ({
    showLocalMenu: showLocalMenuMock,
}));

// Canvas context mock — jsdom does not implement canvas drawing.
let ctxMock;
function makeCtxMock() {
    return {
        clearRect:   vi.fn(),
        save:        vi.fn(),
        restore:     vi.fn(),
        beginPath:   vi.fn(),
        rect:        vi.fn(),
        clip:        vi.fn(),
        moveTo:      vi.fn(),
        lineTo:      vi.fn(),
        stroke:      vi.fn(),
        strokeStyle: '',
        lineWidth:   0,
        lineJoin:    '',
        lineCap:     '',
    };
}

// ─── Helpers ──────────────────────────────────────────────────────────────

/** Create a fresh textarea appended to body. */
function makeTextarea(value = '') {
    const ta = document.createElement('textarea');
    ta.value = value;
    document.body.appendChild(ta);
    return ta;
}

/** Advance fake timers + flush microtask queue. */
async function flush(ms = 0) {
    vi.advanceTimersByTime(ms);
    await Promise.resolve();
    await Promise.resolve();
}

// ─── Setup ──────────────────────────────────────────────────────────────

let attachSpellCheck;

beforeEach(async () => {
    vi.useFakeTimers();
    vi.resetModules();
    notesSpellCheckMock.mockReset();
    notesSpellCheckMock.mockResolvedValue([]);
    document.body.innerHTML = '';

    ctxMock = makeCtxMock();
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(ctxMock);

    // jsdom does not implement Range.prototype.getBoundingClientRect — stub it.
    Range.prototype.getBoundingClientRect = vi.fn(() => ({
        left: 0, right: 0, top: 0, bottom: 0, width: 0, height: 16,
    }));

    ({ attachSpellCheck } = await import('./spellcheck.js'));
});

afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
    delete Range.prototype.getBoundingClientRect;
    document.body.innerHTML = '';
});

describe('spellcheck — suggestion interaction', () => {
    it('disables native text assistance on text controls without adding dictionary checking', () => {
        document.body.innerHTML = '<input id="plain"><input id="filter" type="search"><textarea id="path"></textarea><div id="cell" contenteditable="true"></div><input id="checkbox" type="checkbox">';
        disableNativeTextAssistanceIn(document.body);
        for (const id of ['plain', 'filter', 'path', 'cell']) {
            const control = document.getElementById(id);
            for (const [attribute, value] of [
                ['autocomplete', 'off'], ['autocorrect', 'off'], ['autocapitalize', 'off'],
                ['spellcheck', 'false'], ['writingsuggestions', 'false'],
            ]) expect(control.getAttribute(attribute)).toBe(value);
        }
        expect(document.getElementById('checkbox').hasAttribute('spellcheck')).toBe(false);
        expect(document.querySelector('canvas')).toBeNull();
        expect(notesSpellCheckMock).not.toHaveBeenCalled();
    });

    it('checks single-line inputs with native text assistance disabled', async () => {
        const input = document.createElement('input');
        input.type = 'text';
        input.value = 'helo';
        document.body.appendChild(input);
        notesSpellCheckMock.mockResolvedValue([
            { misspeltWord: 'helo', wordStart: 0, wordLength: 4, suggestions: ['hello'] },
        ]);
        const handle = attachSpellCheck(input);
        await flush(900);
        expect(notesSpellCheckMock).toHaveBeenCalledWith('helo');
        for (const [attribute, value] of [
            ['autocomplete', 'off'], ['autocorrect', 'off'], ['autocapitalize', 'off'],
            ['spellcheck', 'false'], ['writingsuggestions', 'false'],
        ]) expect(input.getAttribute(attribute)).toBe(value);
        expect(document.querySelector('canvas')).not.toBeNull();
        handle.detach();
        expect(document.querySelector('canvas')).toBeNull();
    });

    it('opens suggestions on right-click over a misspelling, not left-click', async () => {
        const textarea = makeTextarea('helo');
        notesSpellCheckMock.mockResolvedValue([
            { misspeltWord: 'helo', wordStart: 0, wordLength: 4, suggestions: ['hello'] },
        ]);
        attachSpellCheck(textarea);
        await flush(900);

        textarea.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, clientX: 0, clientY: 1 }));
        expect(showLocalMenuMock).not.toHaveBeenCalled();

        let bubbledContextMenu = false;
        document.body.addEventListener('contextmenu', () => { bubbledContextMenu = true; }, { once: true });
        const event = new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 0, clientY: 1 });
        textarea.dispatchEvent(event);

        expect(event.defaultPrevented).toBe(true);
        expect(bubbledContextMenu).toBe(false);
        expect(showLocalMenuMock).toHaveBeenCalledWith(expect.objectContaining({
            title: 'helo',
            options: ['hello'],
            x: 0,
            y: 18,
            showNextToMouseCursor: true,
        }));
    });

    it('leaves right-clicks outside misspellings to the normal context menu', async () => {
        const textarea = makeTextarea('hello');
        notesSpellCheckMock.mockResolvedValue([]);
        attachSpellCheck(textarea);
        await flush(900);
        let bubbledContextMenu = false;
        document.body.addEventListener('contextmenu', () => { bubbledContextMenu = true; }, { once: true });
        const event = new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 0, clientY: 1 });
        textarea.dispatchEvent(event);

        expect(event.defaultPrevented).toBe(false);
        expect(bubbledContextMenu).toBe(true);
        expect(showLocalMenuMock).not.toHaveBeenCalled();
    });
});

// ─── Tests ──────────────────────────────────────────────────────────────

describe('spellcheck — canvas lifecycle', () => {
    it('appends a canvas element as a sibling of the textarea', () => {
        const ta = makeTextarea('hello');
        attachSpellCheck(ta);
        const canvas = ta.nextSibling;
        expect(canvas).not.toBeNull();
        expect(canvas.tagName).toBe('CANVAS');
        expect(canvas.getAttribute('aria-hidden')).toBe('true');
    });

    it('removes canvas on detach', () => {
        const ta = makeTextarea('hello');
        const sc = attachSpellCheck(ta);
        expect(document.body.querySelector('canvas[aria-hidden="true"]')).not.toBeNull();
        sc.detach();
        expect(document.body.querySelector('canvas[aria-hidden="true"]')).toBeNull();
    });

    it('detach is idempotent', () => {
        const ta = makeTextarea('hello');
        const sc = attachSpellCheck(ta);
        expect(() => { sc.detach(); sc.detach(); }).not.toThrow();
    });

    it('canvas is pointer-events:none', () => {
        const ta = makeTextarea('hello');
        attachSpellCheck(ta);
        const canvas = document.body.querySelector('canvas[aria-hidden="true"]');
        expect(canvas.style.pointerEvents).toBe('none');
    });
});

describe('spellcheck — check scheduling', () => {
    it('calls NotesSpellCheck after debounce', async () => {
        const ta = makeTextarea('helo world');
        attachSpellCheck(ta);
        await flush(900);
        expect(notesSpellCheckMock).toHaveBeenCalledWith('helo world');
    });

    it('debounces: only one call for rapid input events', async () => {
        const ta = makeTextarea('');
        attachSpellCheck(ta);
        await flush(900); // initial check

        notesSpellCheckMock.mockClear();
        for (const v of ['a', 'ab', 'abc']) {
            ta.value = v;
            ta.dispatchEvent(new Event('input', { bubbles: true }));
        }
        await flush(900);
        expect(notesSpellCheckMock).toHaveBeenCalledTimes(1);
        expect(notesSpellCheckMock).toHaveBeenCalledWith('abc');
    });

    it('does not re-check when text has not changed', async () => {
        const ta = makeTextarea('same text');
        attachSpellCheck(ta);
        await flush(900);
        const callsBefore = notesSpellCheckMock.mock.calls.length;

        ta.dispatchEvent(new Event('input', { bubbles: true }));
        await flush(900);
        expect(notesSpellCheckMock.mock.calls.length).toBe(callsBefore);
    });

    it('does not call backend after detach', async () => {
        const ta = makeTextarea('helo');
        const sc = attachSpellCheck(ta);
        sc.detach();
        notesSpellCheckMock.mockClear();
        await flush(900);
        expect(notesSpellCheckMock).not.toHaveBeenCalled();
    });

    it('check() forces an immediate check bypassing debounce', async () => {
        const ta = makeTextarea('helo');
        const sc = attachSpellCheck(ta);
        await flush(900); // initial
        notesSpellCheckMock.mockClear();

        ta.value = 'new text';
        sc.check();
        await Promise.resolve(); await Promise.resolve();
        expect(notesSpellCheckMock).toHaveBeenCalledWith('new text');
    });
});

describe('spellcheck — canvas drawing', () => {
    it('calls clearRect on the canvas context after each check', async () => {
        const ta = makeTextarea('helo');
        notesSpellCheckMock.mockResolvedValue([
            { misspeltWord: 'helo', wordStart: 0, wordLength: 4, suggestions: ['hello'] },
        ]);
        attachSpellCheck(ta);
        await flush(900);
        expect(ctxMock.clearRect).toHaveBeenCalled();
    });

    it('calls stroke when misspellings are present', async () => {
        const ta = makeTextarea('helo');
        notesSpellCheckMock.mockResolvedValue([
            { misspeltWord: 'helo', wordStart: 0, wordLength: 4, suggestions: ['hello'] },
        ]);
        attachSpellCheck(ta);
        await flush(900);
        expect(ctxMock.stroke).toHaveBeenCalled();
    });

    it('does not call stroke when there are no misspellings', async () => {
        const ta = makeTextarea('hello world');
        notesSpellCheckMock.mockResolvedValue([]);
        attachSpellCheck(ta);
        await flush(900);
        expect(ctxMock.stroke).not.toHaveBeenCalled();
    });

    it('canvas is positioned using textarea offsetTop/offsetLeft', async () => {
        const ta = makeTextarea('hello');
        attachSpellCheck(ta);
        await flush(900);
        const canvas = document.body.querySelector('canvas[aria-hidden="true"]');
        expect(canvas.style.top).toBe(`${ta.offsetTop}px`);
        expect(canvas.style.left).toBe(`${ta.offsetLeft}px`);
    });
});

describe('spellcheck — exclusions', () => {
    it('does not draw for words in the initial exclusions list', async () => {
        const ta = makeTextarea('tset async');
        notesSpellCheckMock.mockResolvedValue([
            { misspeltWord: 'tset',  wordStart: 0, wordLength: 4, suggestions: ['test'] },
            { misspeltWord: 'async', wordStart: 5, wordLength: 5, suggestions: [] },
        ]);
        attachSpellCheck(ta, { exclusions: ['async'] });
        await flush(900);
        // stroke called once (for 'tset' only), not twice
        expect(ctxMock.stroke).toHaveBeenCalledTimes(1);
    });

    it('setExclusions re-filters without calling backend again', async () => {
        const ta = makeTextarea('tset async');
        notesSpellCheckMock.mockResolvedValue([
            { misspeltWord: 'tset',  wordStart: 0, wordLength: 4, suggestions: ['test'] },
            { misspeltWord: 'async', wordStart: 5, wordLength: 5, suggestions: [] },
        ]);
        const sc = attachSpellCheck(ta);
        await flush(900);
        const callsBefore = notesSpellCheckMock.mock.calls.length;

        ctxMock.stroke.mockClear();
        sc.setExclusions(['tset', 'async']);
        await Promise.resolve();

        expect(notesSpellCheckMock.mock.calls.length).toBe(callsBefore); // no new backend call
        expect(ctxMock.stroke).not.toHaveBeenCalled(); // both excluded, nothing drawn
    });

    it('setExclusions is case-insensitive', async () => {
        const ta = makeTextarea('Hello');
        notesSpellCheckMock.mockResolvedValue([
            { misspeltWord: 'Hello', wordStart: 0, wordLength: 5, suggestions: [] },
        ]);
        const sc = attachSpellCheck(ta);
        await flush(900);
        expect(ctxMock.stroke).toHaveBeenCalled();

        ctxMock.stroke.mockClear();
        sc.setExclusions(['hello']); // lowercase exclusion matches 'Hello'
        await Promise.resolve();
        expect(ctxMock.stroke).not.toHaveBeenCalled();
    });

    it('clearing exclusions restores filtered words', async () => {
        const ta = makeTextarea('helo');
        notesSpellCheckMock.mockResolvedValue([
            { misspeltWord: 'helo', wordStart: 0, wordLength: 4, suggestions: ['hello'] },
        ]);
        const sc = attachSpellCheck(ta, { exclusions: ['helo'] });
        await flush(900);
        expect(ctxMock.stroke).not.toHaveBeenCalled();

        ctxMock.stroke.mockClear();
        sc.setExclusions([]); // remove exclusion
        await Promise.resolve();
        expect(ctxMock.stroke).toHaveBeenCalled();
    });
});

describe('spellcheck — scroll and redraw', () => {
    it('redraws (clearRect) when textarea fires a scroll event', async () => {
        const ta = makeTextarea('hello');
        attachSpellCheck(ta);
        await flush(900);

        ctxMock.clearRect.mockClear();
        ta.dispatchEvent(new Event('scroll'));
        expect(ctxMock.clearRect).toHaveBeenCalled();
    });

    it('does not redraw after detach', async () => {
        const ta = makeTextarea('hello');
        const sc = attachSpellCheck(ta);
        await flush(900);
        sc.detach();

        ctxMock.clearRect.mockClear();
        ta.dispatchEvent(new Event('scroll'));
        expect(ctxMock.clearRect).not.toHaveBeenCalled();
    });

    it('canvas is a sibling of textarea so it inherits parent visibility', () => {
        const ta = makeTextarea('helo');
        attachSpellCheck(ta);
        expect(ta.nextSibling.tagName).toBe('CANVAS');
        expect(ta.parentNode).toBe(ta.nextSibling.parentNode);
    });
});

describe('spellcheck — aspell unavailable', () => {
    it('silently ignores backend errors and draws nothing', async () => {
        const ta = makeTextarea('helo');
        notesSpellCheckMock.mockRejectedValue(new Error('aspell not found'));
        attachSpellCheck(ta);
        await flush(900);
        expect(ctxMock.stroke).not.toHaveBeenCalled();
    });
});

describe('spellcheck — external (typos-lsp) mode', () => {
    it('does not call the aspell backend when started in external mode', async () => {
        const ta = makeTextarea('helo world');
        attachSpellCheck(ta, { mode: 'external' });
        await flush(900);
        expect(notesSpellCheckMock).not.toHaveBeenCalled();
    });

    it('renders pushed misspellings in external mode', async () => {
        const ta = makeTextarea('helo world');
        const sc = attachSpellCheck(ta, { mode: 'external' });
        await flush(900);

        ctxMock.stroke.mockClear();
        sc.setMisspellings([
            { misspeltWord: 'helo', wordStart: 0, wordLength: 4, suggestions: ['hello'] },
        ]);
        expect(ctxMock.stroke).toHaveBeenCalled();
    });

    it('ignores setMisspellings while in aspell mode', async () => {
        const ta = makeTextarea('helo world');
        const sc = attachSpellCheck(ta);
        await flush(900);

        ctxMock.stroke.mockClear();
        sc.setMisspellings([
            { misspeltWord: 'helo', wordStart: 0, wordLength: 4, suggestions: ['hello'] },
        ]);
        expect(ctxMock.stroke).not.toHaveBeenCalled();
    });

    it('setMode switches sources: external suppresses aspell, aspell restores it', async () => {
        const ta = makeTextarea('helo world');
        const sc = attachSpellCheck(ta);
        await flush(900);
        expect(sc.getMode()).toBe('aspell');

        sc.setMode('external');
        expect(sc.getMode()).toBe('external');
        notesSpellCheckMock.mockClear();
        ta.value = 'changed text';
        ta.dispatchEvent(new Event('input', { bubbles: true }));
        await flush(900);
        expect(notesSpellCheckMock).not.toHaveBeenCalled();

        sc.setMode('aspell');
        await flush(900);
        expect(notesSpellCheckMock).toHaveBeenCalledWith('changed text');
    });

    it('switching back to aspell clears externally pushed misspellings', async () => {
        const ta = makeTextarea('helo world');
        const sc = attachSpellCheck(ta, { mode: 'external' });
        await flush(900);
        sc.setMisspellings([
            { misspeltWord: 'helo', wordStart: 0, wordLength: 4, suggestions: ['hello'] },
        ]);

        ctxMock.clearRect.mockClear();
        ctxMock.stroke.mockClear();
        notesSpellCheckMock.mockResolvedValue([]);
        sc.setMode('aspell');
        // Underlines cleared immediately on mode switch.
        expect(ctxMock.clearRect).toHaveBeenCalled();
        await flush(900);
        expect(ctxMock.stroke).not.toHaveBeenCalled();
    });
});


