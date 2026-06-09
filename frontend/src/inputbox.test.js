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
        vi.resetModules();
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
        expect(terminalInputBoxSubmitMock).toHaveBeenCalledWith(10, {
            value: '',
            variables: {},
        }, false);
    });

    it('maps Home and End to current line boundaries in multiline input', async () => {
        const { initInputBox } = await import('./inputbox.js');
        const canvas = document.createElement('canvas');
        document.body.appendChild(canvas);

        initInputBox(canvas);

        const openHandler = eventsOnMock.mock.calls[0][1];
        openHandler({
            id: 11,
            title: 'Multiline',
            defaultValue: 'one line\ntwo line\nthree line',
            placeholder: '',
            multiline: true,
            history: [],
        });

        const input = document.querySelector('.inputbox-input');
        const secondLineStart = input.value.indexOf('two line');
        const secondLineEnd = secondLineStart + 'two line'.length;

        input.selectionStart = secondLineStart + 4;
        input.selectionEnd = secondLineStart + 4;
        input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Home', bubbles: true, cancelable: true }));
        expect(input.selectionStart).toBe(secondLineStart);
        expect(input.selectionEnd).toBe(secondLineStart);

        input.selectionStart = secondLineStart + 1;
        input.selectionEnd = secondLineStart + 1;
        input.dispatchEvent(new KeyboardEvent('keydown', { key: 'End', bubbles: true, cancelable: true }));
        expect(input.selectionStart).toBe(secondLineEnd);
        expect(input.selectionEnd).toBe(secondLineEnd);
    });

    it('submits value and typed variable map when dynamic fields are present', async () => {
        const { initInputBox } = await import('./inputbox.js');
        const canvas = document.createElement('canvas');
        document.body.appendChild(canvas);

        initInputBox(canvas);

        const openHandler = eventsOnMock.mock.calls[0][1];
        openHandler({
            id: 22,
            title: 'Create thing',
            defaultValue: 'base-value',
            placeholder: '',
            multiline: false,
            history: [],
            variables: [
                {
                    name: 'textVar',
                    label: 'Text value',
                    type: 'string',
                    default: 'hello',
                    description: 'string description',
                },
                {
                    name: 'count',
                    label: 'Count',
                    type: 'int',
                    default: '7',
                },
                {
                    name: 'enabled',
                    label: 'Enabled',
                    type: 'bool',
                    default: 'false',
                },
                {
                    name: 'mode',
                    label: 'Mode',
                    type: 'list',
                    default: 'alpha, beta, gamma',
                    description: 'Pick one',
                },
            ],
        });

        const inputs = Array.from(document.querySelectorAll('.inputbox-input'));
        const mainInput = inputs[0];
        const textVarInput = inputs[1];
        const intInput = document.querySelector('input[type="number"]');
        const checkbox = document.querySelector('.inputbox-variable-checkbox');
        const listButton = document.querySelector('.inputbox-variable-list-btn');

        expect(mainInput.value).toBe('base-value');
        expect(textVarInput.value).toBe('hello');
        expect(intInput.value).toBe('7');
        expect(checkbox.checked).toBe(false);
        expect(listButton.textContent).toBe('alpha');

        listButton.click();
        expect(showLocalMenuMock).toHaveBeenLastCalledWith(expect.objectContaining({
            title: 'Mode',
            options: ['alpha', 'beta', 'gamma'],
            onSelect: expect.any(Function),
        }));
        const { onSelect } = showLocalMenuMock.mock.calls[0][0];
        onSelect(2);
        expect(listButton.textContent).toBe('gamma');

        mainInput.value = 'updated-base';
        textVarInput.value = 'updated-string';
        intInput.value = '42';
        checkbox.checked = true;

        document.getElementById('inputbox-ok').click();

        expect(terminalInputBoxSubmitMock).toHaveBeenCalledWith(22, {
            value: 'updated-base',
            variables: {
                textVar: 'updated-string',
                count: 42,
                enabled: true,
                mode: 'gamma',
            },
        }, true);
    });

    it('keeps Tab navigation inside the inputbox modal while open', async () => {
        const { initInputBox } = await import('./inputbox.js');
        const canvas = document.createElement('canvas');
        document.body.appendChild(canvas);

        const outsideButton = document.createElement('button');
        outsideButton.type = 'button';
        outsideButton.id = 'outside-focus-target';
        document.body.appendChild(outsideButton);

        initInputBox(canvas);

        const openHandler = eventsOnMock.mock.calls[0][1];
        openHandler({
            id: 31,
            title: 'Tab trap',
            defaultValue: 'first',
            placeholder: '',
            multiline: false,
            history: [],
            variables: [
                {
                    name: 'second',
                    label: 'Second',
                    type: 'string',
                    default: 'value',
                },
            ],
        });

        const inputs = Array.from(document.querySelectorAll('.inputbox-input'));
        const firstField = inputs[0];
        const secondField = inputs[1];

        await new Promise((resolve) => setTimeout(resolve, 0));

        firstField.focus();
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }));
        expect(document.activeElement).toBe(secondField);

        // Tab from second field wraps back to first (buttons excluded from tab cycle)
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }));
        expect(document.activeElement).toBe(firstField);

        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, bubbles: true, cancelable: true }));
        expect(document.activeElement).toBe(secondField);

        outsideButton.focus();
        expect(document.activeElement).toBe(firstField);
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }));
        expect(document.activeElement).toBe(secondField);
    });

    it('prevents focus from escaping to outside elements while inputbox is open', async () => {
        const { initInputBox } = await import('./inputbox.js');
        const canvas = document.createElement('canvas');
        document.body.appendChild(canvas);

        const outsideInput = document.createElement('input');
        outsideInput.type = 'text';
        outsideInput.id = 'outside-input-target';
        document.body.appendChild(outsideInput);

        initInputBox(canvas);

        const openHandler = eventsOnMock.mock.calls[0][1];
        openHandler({
            id: 32,
            title: 'Focus lock',
            defaultValue: 'inside',
            placeholder: '',
            multiline: false,
            history: [],
            variables: [],
        });

        const insideInput = document.querySelector('.inputbox-input');
        await new Promise((resolve) => setTimeout(resolve, 0));
        expect(document.activeElement).toBe(insideInput);

        outsideInput.focus();
        expect(document.activeElement).toBe(insideInput);
    });

    it('uses shared terminal focus state function while inputbox is open', async () => {
        const { initInputBox } = await import('./inputbox.js');
        const canvas = document.createElement('canvas');
        document.body.appendChild(canvas);

        const setFocusMock = vi.fn();
        window.ttyphoonSetTerminalFocusState = setFocusMock;
        window.terminalFocusedState = true;

        initInputBox(canvas);

        const openHandler = eventsOnMock.mock.calls[0][1];
        openHandler({
            id: 33,
            title: 'Shared focus',
            defaultValue: 'abc',
            placeholder: '',
            multiline: false,
            history: [],
            variables: [],
        });

        expect(setFocusMock).toHaveBeenCalledWith(false, { focusVisible: false, force: true });

        document.getElementById('inputbox-cancel').click();

        expect(setFocusMock).toHaveBeenCalledWith(true, { focusVisible: false, force: true });
        delete window.ttyphoonSetTerminalFocusState;
    });

    it('consumes Ctrl+Tab while inputbox is open', async () => {
        const { initInputBox } = await import('./inputbox.js');
        const canvas = document.createElement('canvas');
        document.body.appendChild(canvas);

        initInputBox(canvas);

        const openHandler = eventsOnMock.mock.calls[0][1];
        openHandler({
            id: 34,
            title: 'Ctrl+Tab guard',
            defaultValue: 'inside',
            placeholder: '',
            multiline: false,
            history: [],
            variables: [],
        });

        await new Promise((resolve) => setTimeout(resolve, 0));
        const insideInput = document.querySelector('.inputbox-input');
        expect(document.activeElement).toBe(insideInput);

        const evt = new KeyboardEvent('keydown', {
            key: 'Tab',
            ctrlKey: true,
            bubbles: true,
            cancelable: true,
        });

        const dispatchResult = document.dispatchEvent(evt);
        expect(dispatchResult).toBe(false);
        expect(document.activeElement).toBe(insideInput);
    });
});