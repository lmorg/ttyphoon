import { beforeEach, describe, expect, it, vi } from 'vitest';

const eventHandlers = new Map();

const eventsOnMock = vi.fn((eventName, handler) => {
    eventHandlers.set(eventName, handler);
});

const screenGetAllMock = vi.fn(async () => ([
    { isCurrent: true, x: 0, y: 0, width: 1920, height: 1080 },
]));
const windowGetPositionMock = vi.fn(async () => ({ x: 100, y: 100 }));
const windowGetSizeMock = vi.fn(async () => ({ w: 1200, h: 800 }));
const windowIsMaximisedMock = vi.fn(async () => false);
const windowMaximiseMock = vi.fn();
const windowUnmaximiseMock = vi.fn();
const windowSetPositionMock = vi.fn();
const windowSetSizeMock = vi.fn();

const getWindowStyleMock = vi.fn(async () => ({
    statusBar: true,
    fontSize: 14,
    fontFamily: 'sans-serif',
    colors: {
        bg: { Red: 30, Green: 34, Blue: 40 },
        fg: { Red: 230, Green: 237, Blue: 243 },
    },
}));
const getAppTitleMock = vi.fn(async () => 'TTYphoon');
const terminalSetFocusMock = vi.fn(async () => {});
const terminalGetPanesMock = vi.fn(async () => []);
const terminalPaneZoomMock = vi.fn(async () => {});
const getNotesLayoutSizingMock = vi.fn(async () => ({}));
const setNotesLayoutRatioMock = vi.fn(async () => {});

vi.mock('./wails-runtime.js', () => ({
    EventsOn: eventsOnMock,
    ScreenGetAll: screenGetAllMock,
    WindowGetPosition: windowGetPositionMock,
    WindowGetSize: windowGetSizeMock,
    WindowIsMaximised: windowIsMaximisedMock,
    WindowMaximise: windowMaximiseMock,
    WindowUnmaximise: windowUnmaximiseMock,
    WindowSetPosition: windowSetPositionMock,
    WindowSetSize: windowSetSizeMock,
}));

vi.mock('./wails-bindings.js', () => ({
    GetWindowStyle: getWindowStyleMock,
    GetAppTitle: getAppTitleMock,
    TerminalSetFocus: terminalSetFocusMock,
    TerminalGetPanes: terminalGetPanesMock,
    TerminalPaneZoom: terminalPaneZoomMock,
    GetNotesLayoutSizing: getNotesLayoutSizingMock,
    SetNotesLayoutRatio: setNotesLayoutRatioMock,
}));

vi.mock('./notes.js', () => ({}));
vi.mock('./terminal.js', () => ({}));

function rect(width, height) {
    return {
        x: 0,
        y: 0,
        left: 0,
        top: 0,
        width,
        height,
        right: width,
        bottom: height,
        toJSON() {
            return this;
        },
    };
}

function flushPromises() {
    return new Promise((resolve) => setTimeout(resolve, 0));
}

async function importTtyphoon() {
    vi.resetModules();
    eventHandlers.clear();
    await import('./ttyphoon.js');
    await flushPromises();
    await flushPromises();
}

describe('ttyphoon focus handoff', () => {
    beforeEach(() => {
        document.body.innerHTML = '<div id="app"></div>';

        eventHandlers.clear();
        eventsOnMock.mockClear();

        screenGetAllMock.mockClear();
        windowGetPositionMock.mockClear();
        windowGetSizeMock.mockClear();
        windowIsMaximisedMock.mockClear();
        windowMaximiseMock.mockClear();
        windowUnmaximiseMock.mockClear();
        windowSetPositionMock.mockClear();
        windowSetSizeMock.mockClear();

        getWindowStyleMock.mockClear();
        getAppTitleMock.mockClear();
        terminalSetFocusMock.mockClear();
        terminalGetPanesMock.mockReset().mockResolvedValue([]);
        terminalPaneZoomMock.mockReset().mockResolvedValue();
        getNotesLayoutSizingMock.mockReset();
        getNotesLayoutSizingMock.mockResolvedValue({});
        setNotesLayoutRatioMock.mockClear();

        if (typeof window.requestAnimationFrame !== 'function') {
            window.requestAnimationFrame = (cb) => setTimeout(() => cb(0), 0);
        }

        vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function mockRect() {
            if (this.id === 'app') {
                return rect(1200, 800);
            }
            if (this.id === 'notes-pane') {
                return rect(600, 800);
            }
            if (this.id === 'notes-terminal-split') {
                return rect(8, 800);
            }
            return rect(1200, 800);
        });
    });

    it('clears terminal focus when collapsing notes into terminal tab mode', async () => {
        await importTtyphoon();

        const toggleNotesPane = eventHandlers.get('toggleNotesPane');
        expect(typeof toggleNotesPane).toBe('function');

        toggleNotesPane();
        await flushPromises();
        await flushPromises();

        expect(terminalSetFocusMock).toHaveBeenCalledWith(false);
        expect(window.terminalFocusedState).toBe(false);
    });

    it('asserts terminal focus on startup', async () => {
        await importTtyphoon();

        expect(terminalSetFocusMock).toHaveBeenCalledWith(true);
        expect(window.terminalFocusedState).toBe(true);
    });

    it('restores the cached outer split and saves only a completed user resize', async () => {
        getNotesLayoutSizingMock.mockResolvedValue({ notesRatio: 0.65 });
        await importTtyphoon();
        const notesPane = document.getElementById('notes-pane');
        const splitter = document.getElementById('notes-terminal-split');
        expect(Number.parseFloat(notesPane.style.width)).toBeCloseTo(65);
        expect(setNotesLayoutRatioMock).not.toHaveBeenCalled();

        splitter.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, button: 0 }));
        window.dispatchEvent(new MouseEvent('mousemove', { clientX: 720 }));
        expect(setNotesLayoutRatioMock).not.toHaveBeenCalled();
        window.dispatchEvent(new MouseEvent('mouseup'));
        await flushPromises();
        expect(setNotesLayoutRatioMock).toHaveBeenCalledWith('notesRatio', 0.6);

        window.dispatchEvent(new CustomEvent('ttyphoon-terminal-fullsize-toggle'));
        window.dispatchEvent(new Event('resize'));
        expect(setNotesLayoutRatioMock).toHaveBeenCalledTimes(1);
        window.dispatchEvent(new CustomEvent('ttyphoon-terminal-fullsize-toggle'));
        expect(Number.parseFloat(notesPane.style.width)).toBeCloseTo(60);
    });

    it('preserves a cached ratio when Notes is collapsed and restored', async () => {
        getNotesLayoutSizingMock.mockResolvedValue({ notesRatio: 0.65 });
        await importTtyphoon();
        const toggle = eventHandlers.get('toggleNotesPane');
        await toggle();
        await flushPromises();
        await toggle();
        await flushPromises();
        await flushPromises();
        expect(Number.parseFloat(document.getElementById('notes-pane').style.width)).toBeCloseTo(65);
        expect(setNotesLayoutRatioMock).not.toHaveBeenCalled();
    });

    // The terminal pane's base `flex:1` sets flex-basis:0%/flex-grow:1, which
    // beats `width` inside the overlay and removed its left/right margins.
    it('neutralises flex growth on a maximized pane so its width applies', async () => {
        await importTtyphoon();

        const terminalPane = document.getElementById('terminal-pane');
        const notesPane = document.getElementById('notes-pane');
        expect(terminalPane.style.flex).toBe('1');

        window.dispatchEvent(new CustomEvent('ttyphoon-terminal-fullsize-toggle'));
        await flushPromises();

        expect(terminalPane.style.flex).toBe('0 0 auto');
        expect(terminalPane.style.width).toBe('calc(100vw - 100px)');
        expect(terminalPane.style.height).toBe('calc(100vh - 100px)');
        // The pane left behind fills the window.
        expect(notesPane.style.width).toBe('100%');
        expect(document.getElementById('notes-terminal-split').style.display).toBe('none');

        window.dispatchEvent(new CustomEvent('ttyphoon-terminal-fullsize-toggle'));
        await flushPromises();

        expect(terminalPane.style.flex).toBe('1');
        expect(notesPane.style.width).toBe('50%');
        expect(document.getElementById('notes-terminal-split').style.display).not.toBe('none');
    });

    it('unzooms the tmux pane before exiting terminal fullsize from the backdrop', async () => {
        terminalGetPanesMock.mockResolvedValue([{ id: '%17', zoomed: true }]);
        await importTtyphoon();

        const terminalPane = document.getElementById('terminal-pane');
        window.dispatchEvent(new CustomEvent('ttyphoon-terminal-fullsize-toggle'));
        await flushPromises();
        expect(terminalPane.parentElement.id).toBe('terminal-fullsize-overlay');
        const overlay = terminalPane.parentElement;

        terminalPaneZoomMock.mockImplementation(async () => {
            expect(terminalPane.parentElement.id).toBe('terminal-fullsize-overlay');
        });
        overlay.dispatchEvent(new MouseEvent('click', { bubbles: true }));
        await flushPromises();
        await flushPromises();

        expect(terminalGetPanesMock).toHaveBeenCalledOnce();
        expect(terminalPaneZoomMock).toHaveBeenCalledWith('%17');
        await vi.waitFor(() => {
            expect(terminalPane.parentElement.id).toBe('content-wrapper');
            expect(overlay.isConnected).toBe(false);
        });
    });

    it('toggles the terminal fullsize overlay from the zoom hotkey event', async () => {
        await importTtyphoon();
        const terminalPane = document.getElementById('terminal-pane');
        const toggle = eventHandlers.get('terminalFullsizeToggle');
        expect(toggle).toBeTypeOf('function');

        toggle();
        expect(terminalPane.dataset.fullsize).toBe('true');
        expect(terminalPane.parentElement.id).toBe('terminal-fullsize-overlay');
        expect(terminalPane.style.width).toBe('calc(100vw - 100px)');
        expect(terminalPane.style.height).toBe('calc(100vh - 100px)');

        toggle();
        expect(terminalPane.dataset.fullsize).toBeUndefined();
        expect(terminalPane.parentElement.id).toBe('content-wrapper');
        expect(terminalPane.style.flex).toBe('1');
    });

    it('maximizes notes with the same geometry as the terminal', async () => {
        await importTtyphoon();

        const notesPane = document.getElementById('notes-pane');
        const terminalPane = document.getElementById('terminal-pane');

        window.dispatchEvent(new CustomEvent('ttyphoon-notes-fullsize-toggle'));
        await flushPromises();

        expect(notesPane.style.width).toBe('calc(100vw - 100px)');
        expect(notesPane.style.height).toBe('calc(100vh - 100px)');
        expect(notesPane.style.flex).toBe('0 0 auto');
        expect(terminalPane.style.width).toBe('100%');

        window.dispatchEvent(new CustomEvent('ttyphoon-notes-fullsize-toggle'));
        await flushPromises();

        expect(notesPane.style.width).toBe('50%');
    });

    it('does not re-focus terminal when clicking embedded notes content', async () => {
        await importTtyphoon();

        const toggleNotesPane = eventHandlers.get('toggleNotesPane');
        toggleNotesPane();
        await flushPromises();
        await flushPromises();

        terminalSetFocusMock.mockClear();

        const notesPane = document.getElementById('notes-pane');
        expect(notesPane).not.toBeNull();

        notesPane.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));
        notesPane.dispatchEvent(new Event('focusin', { bubbles: true }));
        await flushPromises();

        expect(terminalSetFocusMock).not.toHaveBeenCalledWith(true);
    });

    it('still focuses terminal when clicking terminal pane directly', async () => {
        await importTtyphoon();

        const toggleNotesPane = eventHandlers.get('toggleNotesPane');
        toggleNotesPane();
        await flushPromises();
        await flushPromises();

        terminalSetFocusMock.mockClear();

        const terminalPane = document.getElementById('terminal-pane');
        expect(terminalPane).not.toBeNull();

        terminalPane.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));

        expect(terminalSetFocusMock).toHaveBeenCalledWith(true);
    });

    it('does not focus terminal when clicking the embedded Notes tab button', async () => {
        await importTtyphoon();

        const terminalPane = document.getElementById('terminal-pane');
        const notesTabButton = document.createElement('button');
        notesTabButton.dataset.windowId = '__jupyter__';
        terminalPane.appendChild(notesTabButton);

        terminalSetFocusMock.mockClear();

        notesTabButton.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));
        notesTabButton.dispatchEvent(new Event('focusin', { bubbles: true }));

        expect(terminalSetFocusMock).not.toHaveBeenCalledWith(true);
    });

    it('hands ctrl+tab focus to notes and requests embedded notes tab activation', async () => {
        await importTtyphoon();

        const terminalPane = document.getElementById('terminal-pane');
        const notesPane = document.getElementById('notes-pane');
        const embeddedHost = document.createElement('div');
        embeddedHost.id = 'terminal-jupyter-host';
        terminalPane.appendChild(embeddedHost);
        embeddedHost.appendChild(notesPane);

        const activations = [];
        window.addEventListener('ttyphoon-activate-notes-tab', () => {
            activations.push('notes');
        });

        terminalPane.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));

        terminalSetFocusMock.mockClear();

        window.dispatchEvent(new CustomEvent('ttyphoon-focus-notes'));
        await flushPromises();

        expect(terminalSetFocusMock).toHaveBeenCalledWith(false);
        expect(window.terminalFocusedState).toBe(false);
        expect(activations).toEqual(['notes']);
    });

    it('uses alt+tab to hand off to notes and the shared terminal focus path to switch back, including embedded tabs', async () => {
        await importTtyphoon();

        const terminalPane = document.getElementById('terminal-pane');
        const notesPane = document.getElementById('notes-pane');
        const embeddedHost = document.createElement('div');
        embeddedHost.id = 'terminal-jupyter-host';
        terminalPane.appendChild(embeddedHost);
        embeddedHost.appendChild(notesPane);

        const activations = [];
        window.addEventListener('ttyphoon-activate-notes-tab', () => {
            activations.push('notes');
        });
        window.addEventListener('ttyphoon-activate-terminal-tab', () => {
            activations.push('terminal');
        });

        terminalSetFocusMock.mockClear();

        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', altKey: true, bubbles: true, cancelable: true }));
        await flushPromises();

        expect(terminalSetFocusMock).toHaveBeenCalledWith(false);
        expect(window.terminalFocusedState).toBe(false);
        expect(activations.at(-1)).toBe('notes');

        terminalSetFocusMock.mockClear();

        window.dispatchEvent(new CustomEvent('ttyphoon-focus-terminal'));
        await flushPromises();

        expect(terminalSetFocusMock).toHaveBeenCalledWith(true);
        expect(window.terminalFocusedState).toBe(true);
        expect(activations).toContain('terminal');
    });
});
