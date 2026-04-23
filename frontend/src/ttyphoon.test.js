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

vi.mock('../wailsjs/runtime/runtime', () => ({
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

vi.mock('../wailsjs/go/main/WApp', () => ({
    GetWindowStyle: getWindowStyleMock,
    GetAppTitle: getAppTitleMock,
    TerminalSetFocus: terminalSetFocusMock,
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
});
