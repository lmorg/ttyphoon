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
});