import { beforeEach, describe, expect, it, vi } from 'vitest';

const terminalInputBoxSubmitMock = vi.fn(() => Promise.resolve());
const eventsOnMock = vi.fn();
const showLocalMenuMock = vi.fn();

vi.mock('../wailsjs/go/main/WApp', () => ({
    TerminalInputBoxSubmit: terminalInputBoxSubmitMock,
}));

vi.mock('../wailsjs/runtime/runtime', () => ({
    EventsOn: eventsOnMock,
}));

vi.mock('./popup_menu', () => ({
    showLocalMenu: showLocalMenuMock,
}));

describe('inputbox', () => {
    beforeEach(() => {
        document.body.innerHTML = '<div id="terminal-app"></div>';
        eventsOnMock.mockReset();
        showLocalMenuMock.mockReset();
        terminalInputBoxSubmitMock.mockReset();
        terminalInputBoxSubmitMock.mockResolvedValue();
    });

    it('shows a history popup button and writes the selected item into the input', async () => {
        const { initInputBox } = await import('./inputbox.js');
        const canvas = document.createElement('canvas');
        document.body.appendChild(canvas);

        initInputBox(canvas);

        expect(eventsOnMock).toHaveBeenCalledWith('terminalInputBox', expect.any(Function));

        const openHandler = eventsOnMock.mock.calls[0][1];
        openHandler({
            id: 7,
            title: 'Rename file',
            defaultValue: 'draft.txt',
            placeholder: 'Enter a file name',
            multiline: false,
            history: ['notes.txt', 'archive.txt'],
        });

        const input = document.querySelector('.inputbox-input');
        const historyButton = document.getElementById('inputbox-history-btn');

        expect(input).not.toBeNull();
        expect(historyButton).not.toBeNull();
        expect(historyButton.style.display).toBe('inline-flex');

        historyButton.click();

        expect(showLocalMenuMock).toHaveBeenCalledWith(expect.objectContaining({
            title: 'History',
            options: ['notes.txt', 'archive.txt'],
            onSelect: expect.any(Function),
        }));

        const { onSelect } = showLocalMenuMock.mock.calls[0][0];
        onSelect(1);

        expect(input.value).toBe('archive.txt');
    });

    it('does not close when text selection drag ends on backdrop', async () => {
        const { initInputBox } = await import('./inputbox.js');
        const canvas = document.createElement('canvas');
        document.body.appendChild(canvas);

        initInputBox(canvas);

        const openHandler = eventsOnMock.mock.calls[0][1];
        openHandler({
            id: 9,
            title: 'Search',
            defaultValue: 'selected text',
            placeholder: 'Type...',
            multiline: false,
            history: [],
        });

        const overlay = document.getElementById('terminal-inputbox');
        const input = document.querySelector('.inputbox-input');

        expect(overlay.style.display).toBe('flex');

        // Simulate text selection that starts inside input and ends outside dialog.
        input.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }));
        overlay.dispatchEvent(new MouseEvent('pointerup', { bubbles: true }));

        expect(overlay.style.display).toBe('flex');
        expect(terminalInputBoxSubmitMock).not.toHaveBeenCalled();
    });

    it('closes when pointer down/up both occur on backdrop', async () => {
        const { initInputBox } = await import('./inputbox.js');
        const canvas = document.createElement('canvas');
        document.body.appendChild(canvas);

        initInputBox(canvas);

        const openHandler = eventsOnMock.mock.calls[0][1];
        openHandler({
            id: 10,
            title: 'Search',
            defaultValue: '',
            placeholder: 'Type...',
            multiline: false,
            history: [],
        });

        const overlay = document.getElementById('terminal-inputbox');

        overlay.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }));
        overlay.dispatchEvent(new MouseEvent('pointerup', { bubbles: true }));

        expect(overlay.style.display).toBe('none');
        expect(terminalInputBoxSubmitMock).toHaveBeenCalledWith(10, '', false);
    });
});