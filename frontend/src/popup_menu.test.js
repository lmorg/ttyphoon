import { beforeEach, describe, expect, it, vi } from 'vitest';

const terminalMenuHighlightMock = vi.fn(() => Promise.resolve());
const terminalMenuSelectMock = vi.fn(() => Promise.resolve());
const terminalMenuCancelMock = vi.fn(() => Promise.resolve());
const terminalRequestRedrawMock = vi.fn(() => Promise.resolve());
const eventsOnMock = vi.fn();

vi.mock('../wailsjs/go/main/WApp', () => ({
    TerminalMenuHighlight: terminalMenuHighlightMock,
    TerminalMenuSelect: terminalMenuSelectMock,
    TerminalMenuCancel: terminalMenuCancelMock,
    TerminalRequestRedraw: terminalRequestRedrawMock,
}));

vi.mock('../wailsjs/runtime/runtime', () => ({
    EventsOn: eventsOnMock,
}));

describe('popup menu hide/show transitions', () => {
    beforeEach(() => {
        document.body.innerHTML = '';

        Object.defineProperty(HTMLCanvasElement.prototype, 'getContext', {
            configurable: true,
            value: vi.fn(() => ({
                font: '',
                measureText: (text) => ({ width: String(text).length * 8 }),
            })),
        });

        eventsOnMock.mockReset();
        terminalMenuHighlightMock.mockReset();
        terminalMenuSelectMock.mockReset();
        terminalMenuCancelMock.mockReset();
        terminalRequestRedrawMock.mockReset();
        terminalMenuHighlightMock.mockImplementation(() => Promise.resolve());
        terminalMenuSelectMock.mockImplementation(() => Promise.resolve());
        terminalMenuCancelMock.mockImplementation(() => Promise.resolve());
        terminalRequestRedrawMock.mockImplementation(() => Promise.resolve());
        vi.resetModules();
    });

    it('ignores stale hide animation completion when a new menu is opened', async () => {
        const { initTerminalPopupMenu } = await import('./popup_menu.js');

        const canvas = document.createElement('canvas');
        document.body.appendChild(canvas);
        initTerminalPopupMenu(canvas);

        const listMenuEvent = eventsOnMock.mock.calls.find(([eventName]) => eventName === 'terminalListMenu');
        expect(listMenuEvent).toBeTruthy();

        const showMenu = listMenuEvent[1];

        showMenu({ menuId: 1, title: 'Settings', options: ['Terminal.ColorTheme'] });

        const listRoot = document.getElementById('ttyphoon-listbox-menu');
        expect(listRoot).not.toBeNull();
        expect(listRoot.style.display).toBe('block');

        const row = listRoot.querySelector('.tty-menu-row');
        expect(row).not.toBeNull();
        row.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));

        // Emulate backend callback opening a submenu before the previous hide animation ends.
        showMenu({ menuId: 2, title: 'Select a theme', options: ['solarized.itermcolors'] });
        expect(listRoot.style.display).toBe('block');

        listRoot.dispatchEvent(new Event('animationend', { bubbles: true }));

        expect(listRoot.style.display).toBe('block');
        expect(listRoot.querySelector('.tty-menu-title')?.textContent).toBe('Select a theme');
    });

    it('still hides the menu when the current hide animation completes', async () => {
        const { initTerminalPopupMenu } = await import('./popup_menu.js');

        const canvas = document.createElement('canvas');
        document.body.appendChild(canvas);
        initTerminalPopupMenu(canvas);

        const listMenuEvent = eventsOnMock.mock.calls.find(([eventName]) => eventName === 'terminalListMenu');
        expect(listMenuEvent).toBeTruthy();

        const showMenu = listMenuEvent[1];
        showMenu({ menuId: 3, title: 'Settings', options: ['Terminal.ColorTheme'] });

        const listRoot = document.getElementById('ttyphoon-listbox-menu');
        expect(listRoot.style.display).toBe('block');

        document.body.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
        listRoot.dispatchEvent(new Event('animationend', { bubbles: true }));

        expect(listRoot.style.display).toBe('none');
    });
});
