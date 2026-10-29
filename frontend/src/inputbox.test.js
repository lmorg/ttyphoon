import { beforeEach, describe, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';

const inputboxCss = readFileSync('./src/inputbox.css', 'utf8');
const imageBytes = readFileSync('./src/assets/images/icon-question.png');
const imageBase64 = imageBytes.toString('base64');

const terminalInputBoxSubmitMock = vi.fn(() => Promise.resolve());
const terminalInputBoxHistoryImagesMock = vi.fn(() => Promise.resolve([]));
const getClipboardDataMock = vi.fn(() => Promise.resolve({ text: '', image: '' }));
const terminalTextInputMock = vi.fn(() => Promise.resolve());
const terminalKeyPressMock = vi.fn(() => Promise.resolve());
const eventsOnMock = vi.fn();
const showLocalMenuMock = vi.fn();
const showSharedTooltipMock = vi.fn(() => Promise.resolve());
const closeSharedTooltipMock = vi.fn();

vi.mock('../wailsjs/go/main/WApp', () => ({
    TerminalInputBoxSubmit: terminalInputBoxSubmitMock,
    TerminalInputBoxHistoryImages: terminalInputBoxHistoryImagesMock,
    GetClipboardData: getClipboardDataMock,
    TerminalTextInput: terminalTextInputMock,
    TerminalKeyPress: terminalKeyPressMock,
}));

vi.mock('../wailsjs/runtime/runtime', () => ({
    EventsOn: eventsOnMock,
}));

vi.mock('./popup_menu', () => ({
    showLocalMenu: showLocalMenuMock,
}));

vi.mock('./shared_tooltip', () => ({
    bindSharedTooltipMouseTracking: vi.fn(),
    showSharedTooltip: showSharedTooltipMock,
    closeSharedTooltip: closeSharedTooltipMock,
}));

vi.mock('./vim-mode', () => ({
    attachVimMode: vi.fn(() => ({ detach: vi.fn(), getMode: vi.fn(() => 'insert') })),
}));

vi.mock('./spellcheck', () => ({
    attachSpellCheck: vi.fn(() => ({ detach: vi.fn(), setExclusions: vi.fn(), check: vi.fn() })),
}));

describe('inputbox', () => {
    beforeEach(() => {
        document.body.innerHTML = '<div id="terminal-app"></div>';
        vi.resetModules();
        eventsOnMock.mockReset();
        showLocalMenuMock.mockReset();
        showSharedTooltipMock.mockClear();
        closeSharedTooltipMock.mockClear();
        terminalInputBoxSubmitMock.mockReset();
        terminalInputBoxSubmitMock.mockResolvedValue();
        terminalInputBoxHistoryImagesMock.mockReset();
        terminalInputBoxHistoryImagesMock.mockResolvedValue([]);
        terminalTextInputMock.mockClear();
        terminalKeyPressMock.mockClear();
        getClipboardDataMock.mockReset();
        getClipboardDataMock.mockResolvedValue({ text: '', image: '' });
    });

    async function openImageInputbox(options = {}) {
        const { initInputBox } = await import('./inputbox.js');
        initInputBox(document.createElement('canvas'));
        const open = eventsOnMock.mock.calls.find(([event]) => event === 'terminalInputBox')[1];
        open({ id: 101, title: 'Ask AI', multiline: true, ...options });
        return open;
    }

    function pasteImage(file) {
        const event = new Event('paste', { bubbles: true, cancelable: true });
        Object.defineProperty(event, 'clipboardData', { value: {
            items: file ? [{ kind: 'file', type: file.type, getAsFile: () => file }] : [],
            getData: () => '',
        } });
        document.querySelector('.inputbox-input').dispatchEvent(event);
        return event;
    }

    it('disables native assistance on single-line prompts and string variables', async () => {
        const { attachSpellCheck } = await import('./spellcheck');
        attachSpellCheck.mockClear();
        await openImageInputbox({
            multiline: false,
            variables: [{ name: 'path', label: 'Path', type: 'string', default: '/work' }],
        });
        const controls = document.querySelectorAll('.inputbox-input');
        expect(controls).toHaveLength(2);
        for (const control of controls) {
            for (const [attribute, value] of [
                ['autocomplete', 'off'], ['autocorrect', 'off'], ['autocapitalize', 'off'],
                ['spellcheck', 'false'], ['writingsuggestions', 'false'],
            ]) expect(control.getAttribute(attribute)).toBe(value);
        }
        expect(attachSpellCheck).not.toHaveBeenCalled();
    });

    it('uses shared tooltips for actions, variable descriptions and image removal', async () => {
        await openImageInputbox({
            canSaveDraft: true, allowImagePaste: true, history: ['Previous query'],
            variables: [{ name: 'count', label: 'Count', type: 'integer', description: 'Number of results', default: '1' }],
            draft: { value: 'Prompt', variables: {}, images: [{ mimeType: 'image/png', base64: imageBase64 }] },
        });
        const controls = [
            [document.getElementById('inputbox-draft'), 'Close, but store draft'],
            [document.getElementById('inputbox-history-btn'), 'History'],
            [document.querySelector('.inputbox-variable-label'), 'Number of results'],
            [document.querySelector('.inputbox-image-remove'), 'Remove image 1'],
        ];
        for (const [control, text] of controls) {
            expect(control.hasAttribute('title')).toBe(false);
            control.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }));
            expect(showSharedTooltipMock).toHaveBeenLastCalledWith(text);
            closeSharedTooltipMock.mockClear();
            control.dispatchEvent(new MouseEvent('mouseout', { bubbles: true }));
            expect(closeSharedTooltipMock).toHaveBeenCalled();
        }
        const historyButton = document.getElementById('inputbox-history-btn');
        historyButton.focus();
        expect(showSharedTooltipMock).toHaveBeenLastCalledWith('History');
        closeSharedTooltipMock.mockClear();
        historyButton.blur();
        expect(closeSharedTooltipMock).toHaveBeenCalled();
        closeSharedTooltipMock.mockClear();
        document.getElementById('inputbox-cancel').click();
        expect(closeSharedTooltipMock).toHaveBeenCalled();
        expect(document.querySelector('.inputbox-variable-tooltip')).toBeNull();
    });

    it('restores and saves a complete draft with false, zero, empty, list and image values', async () => {
        await openImageInputbox({
            canSaveDraft: true, allowImagePaste: true, defaultValue: 'Default prompt',
            variables: [
                { name: 'text', type: 'string', default: 'Default text' },
                { name: 'count', type: 'integer', default: '5' },
                { name: 'enabled', type: 'boolean', default: 'true' },
                { name: 'mode', type: 'list', default: 'first,second' },
            ],
            draft: {
                value: '', variables: { text: '', count: 0, enabled: false, mode: 'second' },
                images: [{ mimeType: 'image/png', base64: imageBase64 }],
            },
        });
        expect(document.querySelector('textarea.inputbox-input').value).toBe('');
        const rows = document.querySelectorAll('.inputbox-variable-row');
        expect(rows[0].querySelector('input').value).toBe('');
        expect(rows[1].querySelector('input').value).toBe('0');
        expect(rows[2].querySelector('input').checked).toBe(false);
        expect(rows[3].querySelector('button').textContent).toBe('second');
        expect(document.querySelectorAll('.inputbox-image')).toHaveLength(1);
        const draftButton = document.getElementById('inputbox-draft');
        expect(draftButton.previousElementSibling.id).toBe('inputbox-ok');
        expect(draftButton.textContent.codePointAt(0)).toBe(0xe698);
        expect(draftButton.getAttribute('aria-label')).toBe('Draft');
        expect(draftButton.dataset.tooltip).toBe('Close, but store draft');
        expect(draftButton.hasAttribute('title')).toBe(false);
        const historyButton = document.getElementById('inputbox-history-btn');
        expect(historyButton.textContent.codePointAt(0)).toBe(0xf1da);
        expect(historyButton.getAttribute('aria-label')).toBe('History');
        expect(historyButton.dataset.tooltip).toBe('History');
        expect(historyButton.hasAttribute('title')).toBe(false);
        expect(Array.from(document.querySelectorAll('.inputbox-buttons button')).map((button) => button.id))
            .toEqual(['inputbox-ok', 'inputbox-draft', 'inputbox-history-btn', 'inputbox-cancel']);
        expect(draftButton.children).toHaveLength(0);
        document.getElementById('inputbox-draft').click();
        await vi.waitFor(() => expect(document.getElementById('terminal-inputbox').style.display).toBe('none'));
        expect(terminalInputBoxSubmitMock).toHaveBeenCalledOnce();
        expect(terminalInputBoxSubmitMock).toHaveBeenCalledWith(101, {
            value: '', variables: { text: '', count: 0, enabled: false, mode: 'second' },
            images: [{ mimeType: 'image/png', base64: imageBase64 }], saveDraft: true,
        }, false);
    });

    it('waits for draft persistence before closing and prevents duplicate saves', async () => {
        let complete;
        terminalInputBoxSubmitMock.mockReturnValue(new Promise((resolve) => { complete = resolve; }));
        await openImageInputbox({ canSaveDraft: true, defaultValue: 'Work in progress' });
        document.getElementById('inputbox-draft').click();
        expect(document.getElementById('terminal-inputbox').style.display).toBe('flex');
        expect(document.getElementById('inputbox-draft').disabled).toBe(true);
        expect(document.getElementById('inputbox-ok').disabled).toBe(true);
        expect(document.getElementById('inputbox-cancel').disabled).toBe(true);
        document.getElementById('inputbox-draft').click();
        expect(terminalInputBoxSubmitMock).toHaveBeenCalledOnce();
        complete();
        await vi.waitFor(() => expect(document.getElementById('terminal-inputbox').style.display).toBe('none'));
    });

    it('keeps edited fields open when saving a draft fails and leaves Cancel unchanged', async () => {
        terminalInputBoxSubmitMock.mockRejectedValueOnce(new Error('Bridge unavailable'));
        await openImageInputbox({ canSaveDraft: true });
        document.querySelector('.inputbox-input').value = 'Unsaved work';
        document.getElementById('inputbox-draft').click();
        await vi.waitFor(() => expect(document.getElementById('inputbox-image-error').textContent).toContain('Could not save draft'));
        expect(document.getElementById('terminal-inputbox').style.display).toBe('flex');
        expect(document.querySelector('.inputbox-input').value).toBe('Unsaved work');
        expect(document.getElementById('inputbox-draft').disabled).toBe(false);
        document.getElementById('inputbox-cancel').click();
        expect(terminalInputBoxSubmitMock).toHaveBeenLastCalledWith(101, { value: 'Unsaved work', variables: {} }, false);
    });

    it('does not offer draft saving for confirmation dialogs', async () => {
        await openImageInputbox({ canSaveDraft: true, confirmation: true });
        const draftButton = document.getElementById('inputbox-draft');
        expect(draftButton.hidden).toBe(true);
        draftButton.click();
        expect(terminalInputBoxSubmitMock).not.toHaveBeenCalled();
    });

    it('does not intercept image paste without the opt-in flag', async () => {
        await openImageInputbox();
        const event = pasteImage(new File([imageBytes], 'image.png', { type: 'image/png' }));
        expect(event.defaultPrevented).toBe(false);
        expect(getClipboardDataMock).not.toHaveBeenCalled();
        expect(document.getElementById('inputbox-images').hidden).toBe(true);
        document.getElementById('inputbox-ok').click();
        expect(terminalInputBoxSubmitMock).toHaveBeenCalledWith(101, { value: '', variables: {} }, true);
    });

    it('previews and submits pasted browser image files alongside text and variables', async () => {
        await openImageInputbox({ allowImagePaste: true, defaultValue: 'Describe this', variables: [
            { name: 'model', label: 'Model', type: 'string', default: 'vision' },
        ] });
        const file = new File([imageBytes], 'image.png', { type: 'image/png' });
        expect(pasteImage(file).defaultPrevented).toBe(true);
        expect(document.getElementById('inputbox-ok').disabled).toBe(true);
        await vi.waitFor(() => expect(document.querySelectorAll('.inputbox-image img')).toHaveLength(1));
        expect(document.getElementById('inputbox-ok').disabled).toBe(false);
        document.getElementById('inputbox-ok').click();
        expect(terminalInputBoxSubmitMock).toHaveBeenCalledWith(101, {
            value: 'Describe this', variables: { model: 'vision' },
            images: [{ mimeType: 'image/png', base64: imageBase64 }],
        }, true);
    });

    it('uses the native clipboard fallback and lets an attachment be removed', async () => {
        getClipboardDataMock.mockResolvedValue({ text: '', image: imageBase64 });
        await openImageInputbox({ allowImagePaste: true });
        pasteImage();
        await vi.waitFor(() => expect(document.querySelectorAll('.inputbox-image img')).toHaveLength(1));
        expect(getClipboardDataMock).toHaveBeenCalledOnce();
        document.querySelector('.inputbox-image-remove').click();
        expect(document.querySelectorAll('.inputbox-image')).toHaveLength(0);
        document.getElementById('inputbox-ok').click();
        expect(terminalInputBoxSubmitMock.mock.calls[0][1].images).toBeUndefined();
    });

    it('ignores clipboard results from a closed inputbox and discards cancelled attachments', async () => {
        let complete;
        getClipboardDataMock.mockReturnValue(new Promise((resolve) => { complete = resolve; }));
        const open = await openImageInputbox({ allowImagePaste: true });
        pasteImage();
        document.getElementById('inputbox-cancel').click();
        expect(terminalInputBoxSubmitMock).toHaveBeenCalledWith(101, { value: '', variables: {} }, false);
        open({ id: 102, title: 'Another query', multiline: true, allowImagePaste: true });
        complete({ image: imageBase64 });
        await Promise.resolve();
        await Promise.resolve();
        expect(document.querySelectorAll('.inputbox-image')).toHaveLength(0);
        expect(document.getElementById('inputbox-ok').disabled).toBe(false);
    });

    it('limits pasted image count and keeps text-only paste native', async () => {
        getClipboardDataMock.mockResolvedValue({ image: imageBase64 });
        await openImageInputbox({ allowImagePaste: true });
        for (let index = 0; index < 7; index++) pasteImage();
        await vi.waitFor(() => expect(document.querySelectorAll('.inputbox-image')).toHaveLength(6));
        expect(document.getElementById('inputbox-image-error').textContent).toContain('at most 6');
        const textPaste = new Event('paste', { bubbles: true, cancelable: true });
        Object.defineProperty(textPaste, 'clipboardData', { value: { items: [], getData: () => 'ordinary text' } });
        document.querySelector('.inputbox-input').dispatchEvent(textPaste);
        expect(textPaste.defaultPrevented).toBe(false);
    });

    it.each([
        ['image/svg+xml', 10, 'PNG, JPEG, GIF, or WebP'],
        ['image/png', 8 * 1024 * 1024 + 1, 'at most 8 MiB'],
    ])('rejects unsupported or oversized pasted files (%s)', async (type, size, message) => {
        await openImageInputbox({ allowImagePaste: true });
        pasteImage(new File([new Uint8Array(size)], 'image', { type }));
        await vi.waitFor(() => expect(document.getElementById('inputbox-image-error').textContent).toContain(message));
        expect(document.querySelectorAll('.inputbox-image')).toHaveLength(0);
        expect(document.getElementById('inputbox-ok').disabled).toBe(false);
    });

    it('restores history images lazily and includes their bytes on submission', async () => {
        const attachment = { mimeType: 'image/png', base64: imageBase64 };
        terminalInputBoxHistoryImagesMock.mockResolvedValue([attachment]);
        await openImageInputbox({ allowImagePaste: true,
            historyEntries: [{ value: 'With image', variables: {}, images: [{ path: '/saved/image.png' }] }],
        });
        expect(terminalInputBoxHistoryImagesMock).not.toHaveBeenCalled();
        document.getElementById('inputbox-history-btn').click();
        const menu = showLocalMenuMock.mock.calls.at(-1)[0];
        expect(menu.options[0]).toContain('1 image');
        menu.onSelect(0);
        expect(document.getElementById('inputbox-ok').disabled).toBe(true);
        expect(terminalInputBoxHistoryImagesMock).toHaveBeenCalledWith(101, 0);
        await vi.waitFor(() => expect(document.querySelectorAll('.inputbox-image img')).toHaveLength(1));
        expect(document.querySelector('.inputbox-image img').src).toBe(`data:image/png;base64,${imageBase64}`);
        document.getElementById('inputbox-ok').click();
        expect(terminalInputBoxSubmitMock).toHaveBeenCalledWith(101, { value: 'With image', variables: {}, images: [attachment] }, true);
    });

    it('clears history images for text-only selections but preserves them on field-only recall', async () => {
        const attachment = { mimeType: 'image/png', base64: imageBase64 };
        await openImageInputbox({ allowImagePaste: true, canSaveDraft: true,
            draft: { value: '', variables: {}, images: [attachment] },
            historyEntries: [{ value: 'Text only', variables: {} }],
        });
        document.querySelector('textarea').dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowUp', bubbles: true, cancelable: true }));
        showLocalMenuMock.mock.calls.at(-1)[0].onSelect(0);
        expect(document.querySelectorAll('.inputbox-image')).toHaveLength(1);
        document.getElementById('inputbox-history-btn').click();
        showLocalMenuMock.mock.calls.at(-1)[0].onSelect(0);
        expect(document.querySelectorAll('.inputbox-image')).toHaveLength(0);
        expect(terminalInputBoxHistoryImagesMock).not.toHaveBeenCalled();
    });

    it('ignores late history image results after another selection and prevents concurrent paste', async () => {
        let complete;
        terminalInputBoxHistoryImagesMock.mockReturnValue(new Promise((resolve) => { complete = resolve; }));
        await openImageInputbox({ allowImagePaste: true, historyEntries: [
            { value: 'With image', images: [{ path: '/saved/image.png' }] }, { value: 'Text only' },
        ] });
        document.getElementById('inputbox-history-btn').click();
        showLocalMenuMock.mock.calls.at(-1)[0].onSelect(0);
        pasteImage(new File([imageBytes], 'image.png', { type: 'image/png' }));
        expect(document.getElementById('inputbox-image-error').textContent).toContain('finish loading');
        document.getElementById('inputbox-history-btn').click();
        showLocalMenuMock.mock.calls.at(-1)[0].onSelect(1);
        complete([{ mimeType: 'image/png', base64: imageBase64 }]);
        await Promise.resolve();
        await Promise.resolve();
        expect(document.querySelector('textarea').value).toBe('Text only');
        expect(document.querySelectorAll('.inputbox-image')).toHaveLength(0);
        expect(document.getElementById('inputbox-ok').disabled).toBe(false);
    });

    it('ignores history image results after cancelling and reopening the dialog', async () => {
        let complete;
        terminalInputBoxHistoryImagesMock.mockReturnValue(new Promise((resolve) => { complete = resolve; }));
        const open = await openImageInputbox({ allowImagePaste: true,
            historyEntries: [{ value: 'With image', images: [{ path: '/saved/image.png' }] }],
        });
        document.getElementById('inputbox-history-btn').click();
        showLocalMenuMock.mock.calls.at(-1)[0].onSelect(0);
        document.getElementById('inputbox-cancel').click();
        open({ id: 102, title: 'Another dialog', multiline: true, allowImagePaste: true });
        complete([{ mimeType: 'image/png', base64: imageBase64 }]);
        await Promise.resolve();
        await Promise.resolve();
        expect(document.querySelectorAll('.inputbox-image')).toHaveLength(0);
        expect(document.getElementById('inputbox-ok').disabled).toBe(false);
    });

    it('shows an error for missing history images without blocking the restored text', async () => {
        terminalInputBoxHistoryImagesMock.mockRejectedValue(new Error('Saved image is missing'));
        await openImageInputbox({ allowImagePaste: true,
            historyEntries: [{ value: 'With image', images: [{ path: '/missing.png' }] }],
        });
        document.getElementById('inputbox-history-btn').click();
        showLocalMenuMock.mock.calls.at(-1)[0].onSelect(0);
        await vi.waitFor(() => expect(document.getElementById('inputbox-image-error').textContent).toContain('missing'));
        expect(document.querySelector('textarea').value).toBe('With image');
        expect(document.getElementById('inputbox-ok').disabled).toBe(false);
    });

    it('does not restore history images into image-disabled dialogs', async () => {
        await openImageInputbox({ historyEntries: [{ value: 'With image', images: [{ path: '/saved/image.png' }] }] });
        document.getElementById('inputbox-history-btn').click();
        showLocalMenuMock.mock.calls.at(-1)[0].onSelect(0);
        expect(terminalInputBoxHistoryImagesMock).not.toHaveBeenCalled();
        expect(document.querySelectorAll('.inputbox-image')).toHaveLength(0);
    });

    it('restores all typed fields from a selected full-history entry', async () => {
        await openImageInputbox({ defaultValue: 'Current prompt',
            variables: [
                { name: 'text', type: 'string', default: 'Default text' },
                { name: 'count', type: 'integer', default: '9' },
                { name: 'enabled', type: 'boolean', default: 'true' },
                { name: 'model', type: 'list', options: ['first', 'second'], default: 'first' },
                { name: 'newField', type: 'string', default: 'New default' },
            ],
            historyEntries: [
                { value: 'Same prompt', variables: { text: 'older', count: 4, enabled: true, model: 'first' } },
                { value: 'Same prompt', variables: { text: '', count: 0, enabled: false, model: 'second' } },
            ],
        });
        const rows = document.querySelectorAll('.inputbox-variable-row');
        rows[4].querySelector('input').value = 'Do not retain this';
        document.getElementById('inputbox-history-btn').click();
        const menu = showLocalMenuMock.mock.calls.at(-1)[0];
        expect(menu.options).toHaveLength(2);
        expect(menu.options[0]).not.toBe(menu.options[1]);
        expect(menu.options[1]).toContain('model: second');
        menu.onSelect(1);
        expect(document.querySelector('textarea').value).toBe('Same prompt');
        expect(rows[0].querySelector('input').value).toBe('');
        expect(rows[1].querySelector('input').value).toBe('0');
        expect(rows[2].querySelector('input').checked).toBe(false);
        expect(rows[3].querySelector('button').textContent).toBe('second');
        expect(rows[4].querySelector('input').value).toBe('New default');
        document.getElementById('inputbox-ok').click();
        expect(terminalInputBoxSubmitMock).toHaveBeenCalledWith(101, {
            value: 'Same prompt', variables: { text: '', count: 0, enabled: false, model: 'second', newField: 'New default' },
        }, true);
    });

    it('uses distinct field-only history for Up Arrow in a variable text field', async () => {
        await openImageInputbox({ defaultValue: 'Keep this prompt',
            variables: [
                { name: 'text', type: 'string', default: '' },
                { name: 'other', type: 'string', default: 'Keep this field' },
            ],
            historyEntries: [
                { value: 'New prompt', variables: { text: 'new value', other: 'new other' } },
                { value: 'Middle prompt', variables: { text: 'new value', other: 'middle other' } },
                { value: 'Old prompt', variables: { text: 'old value', other: 'old other' } },
                { value: 'Legacy prompt' },
            ],
        });
        const rows = document.querySelectorAll('.inputbox-variable-row');
        const field = rows[0].querySelector('input');
        const event = new KeyboardEvent('keydown', { key: 'ArrowUp', bubbles: true, cancelable: true });
        field.dispatchEvent(event);
        expect(event.defaultPrevented).toBe(true);
        const menu = showLocalMenuMock.mock.calls.at(-1)[0];
        expect(menu.options).toEqual(['new value', 'old value']);
        menu.onSelect(1);
        expect(field.value).toBe('old value');
        expect(rows[1].querySelector('input').value).toBe('Keep this field');
        expect(document.querySelector('textarea').value).toBe('Keep this prompt');
    });

    it('uses distinct prompt-only history for Up Arrow in the main input', async () => {
        await openImageInputbox({
            variables: [{ name: 'text', type: 'string', default: 'Keep variable' }],
            historyEntries: [
                { value: 'Same prompt', variables: { text: 'one' } },
                { value: 'Same prompt', variables: { text: 'two' } },
                { value: 'Other prompt', variables: { text: 'three' } },
            ],
        });
        document.querySelector('textarea').dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowUp', bubbles: true, cancelable: true }));
        const menu = showLocalMenuMock.mock.calls.at(-1)[0];
        expect(menu.options).toEqual(['Same prompt', 'Other prompt']);
        menu.onSelect(1);
        expect(document.querySelector('textarea').value).toBe('Other prompt');
        expect(document.querySelector('.inputbox-variable-row input').value).toBe('Keep variable');
    });

    it('restores empty prompt history and preserves fields for legacy text-only entries', async () => {
        await openImageInputbox({ defaultValue: 'Current prompt',
            variables: [{ name: 'text', type: 'string', default: 'Keep variable' }],
            historyEntries: [{ value: '' }, { value: 'Legacy prompt' }],
        });
        document.getElementById('inputbox-history-btn').click();
        const menu = showLocalMenuMock.mock.calls.at(-1)[0];
        menu.onSelect(0);
        expect(document.querySelector('textarea').value).toBe('');
        expect(document.querySelector('.inputbox-variable-row input').value).toBe('Keep variable');
        document.getElementById('inputbox-history-btn').click();
        showLocalMenuMock.mock.calls.at(-1)[0].onSelect(1);
        expect(document.querySelector('textarea').value).toBe('Legacy prompt');
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

        expect(document.querySelector('#inputbox-title h3')?.textContent).toBe('Rename file');
        expect(document.getElementById('inputbox-title').classList.contains('markdown-body')).toBe(true);
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

    it('uses themed Kill and Cancel buttons and lets Escape cancel a confirmation', async () => {
        const { initInputBox } = await import('./inputbox.js');
        initInputBox(document.createElement('canvas'));
        const openHandler = eventsOnMock.mock.calls.find(([name]) => name === 'terminalInputBox')[1];
        openHandler({
            id: 12,
            title: 'Confirm pane close',
            message: 'Are you sure you wish to kill the active pane?\n\nID: %7\nName: shell',
            confirmation: true,
            confirmLabel: 'Kill',
            cancelLabel: 'Cancel',
            confirmIsDanger: true,
        });

        expect(document.querySelector('#inputbox-title h3')?.textContent).toBe('Confirm pane close');
        expect(document.querySelector('.inputbox-confirmation-message')?.textContent).toContain('ID: %7\nName: shell');
        expect(document.querySelector('.inputbox-input')).toBeNull();
        expect(document.getElementById('inputbox-ok').textContent).toBe('Kill');
        expect(document.getElementById('inputbox-ok').classList.contains('inputbox-danger')).toBe(true);
        expect(document.getElementById('inputbox-cancel').textContent).toBe('Cancel');
        expect(document.getElementById('inputbox-escape-hint').textContent).toBe('Escape to cancel');
        const visibleButtons = Array.from(document.querySelectorAll('.inputbox-buttons button'))
            .filter((button) => button.style.display !== 'none');
        expect(visibleButtons.map((button) => button.id)).toEqual(['inputbox-ok', 'inputbox-cancel']);

        await new Promise((resolve) => setTimeout(resolve, 0));
        expect(document.activeElement).toBe(document.getElementById('inputbox-cancel'));
        const tabEvent = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true });
        document.dispatchEvent(tabEvent);
        expect(tabEvent.defaultPrevented).toBe(true);
        expect(document.activeElement).toBe(document.getElementById('inputbox-ok'));
        const reverseTabEvent = new KeyboardEvent('keydown', {
            key: 'Tab', shiftKey: true, bubbles: true, cancelable: true,
        });
        document.dispatchEvent(reverseTabEvent);
        expect(reverseTabEvent.defaultPrevented).toBe(true);
        expect(document.activeElement).toBe(document.getElementById('inputbox-cancel'));

        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }));
        expect(document.activeElement).toBe(document.getElementById('inputbox-ok'));
        const enterEvent = new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true });
        document.dispatchEvent(enterEvent);
        expect(enterEvent.defaultPrevented).toBe(true);
        expect(terminalInputBoxSubmitMock).toHaveBeenCalledWith(12, { value: '', variables: {} }, true);

        terminalInputBoxSubmitMock.mockClear();
        openHandler({
            id: 12,
            title: 'Confirm pane close',
            message: 'Are you sure you wish to kill the active pane?\n\nID: %7\nName: shell',
            confirmation: true,
            confirmLabel: 'Kill',
            cancelLabel: 'Cancel',
            confirmIsDanger: true,
        });
        await new Promise((resolve) => setTimeout(resolve, 0));
        for (const keyEvent of [{ key: 'x' }, { key: 'c', ctrlKey: true }]) {
            const event = new KeyboardEvent('keydown', { ...keyEvent, bubbles: true, cancelable: true });
            document.dispatchEvent(event);
            expect(event.defaultPrevented).toBe(true);
        }
        expect(terminalInputBoxSubmitMock).not.toHaveBeenCalled();
        expect(document.getElementById('terminal-inputbox').style.display).toBe('flex');

        const spaceEvent = new KeyboardEvent('keydown', { key: ' ', bubbles: true, cancelable: true });
        document.dispatchEvent(spaceEvent);
        expect(spaceEvent.defaultPrevented).toBe(true);
        expect(terminalInputBoxSubmitMock).toHaveBeenCalledWith(12, { value: '', variables: {} }, false);

        terminalInputBoxSubmitMock.mockClear();
        openHandler({
            id: 12,
            title: 'Confirm pane close',
            message: 'Are you sure you wish to kill the active pane?\n\nID: %7\nName: shell',
            confirmation: true,
            confirmLabel: 'Kill',
            cancelLabel: 'Cancel',
            confirmIsDanger: true,
        });

        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }));
        expect(terminalInputBoxSubmitMock).toHaveBeenCalledWith(12, { value: '', variables: {} }, false);
    });

    it('uses the shared yellow Jura heading style for dialog titles', () => {
        const style = document.createElement('style');
        style.textContent = inputboxCss;
        document.head.appendChild(style);
        try {
            const title = Array.from(style.sheet.cssRules).find((rule) => rule.selectorText === '.inputbox-title').style;
            expect(title.getPropertyValue('color')).toContain('--yellow');
            expect(title.getPropertyValue('font-family')).toContain('Jura');
            expect(title.getPropertyValue('font-size')).toContain('--notes-font-size');
        } finally {
            style.remove();
        }
    });

    it('keeps the Kill background transparent at rest and red-tinted on hover', () => {
        const style = document.createElement('style');
        style.textContent = inputboxCss;
        document.head.appendChild(style);
        try {
            const rules = Array.from(style.sheet.cssRules);
            const idle = rules.find((rule) => rule.selectorText === '.inputbox-danger').style;
            const hover = rules.find((rule) => rule.selectorText === '.inputbox-danger:hover').style;
            const cancel = rules.find((rule) => rule.selectorText === '.inputbox-cancel').style;
            expect(idle.getPropertyValue('background')).toBe('transparent');
            expect(idle.getPropertyValue('border-color')).toBe(cancel.getPropertyValue('border-color'));
            expect(idle.getPropertyValue('color')).toBe(cancel.getPropertyValue('color'));
            expect(hover.getPropertyValue('border-color')).toContain('--red');
            expect(hover.getPropertyValue('background')).toContain('--terminal-red-20');
        } finally {
            style.remove();
        }
    });

    it('visibly outlines the focused modal button', () => {
        const style = document.createElement('style');
        style.textContent = inputboxCss;
        document.head.appendChild(style);
        try {
            const focus = Array.from(style.sheet.cssRules)
                .find((rule) => rule.selectorText === '.inputbox-btn:focus').style;
            expect(focus.getPropertyValue('outline')).toContain('var(--accent');
            expect(focus.getPropertyValue('outline-offset')).toBe('2px');
        } finally {
            style.remove();
        }
    });

    it('submits a confirmation only when Kill is activated', async () => {
        const { initInputBox } = await import('./inputbox.js');
        initInputBox(document.createElement('canvas'));
        const openHandler = eventsOnMock.mock.calls.find(([name]) => name === 'terminalInputBox')[1];
        openHandler({ id: 13, title: 'Confirm pane close', confirmation: true, confirmLabel: 'Kill' });
        document.getElementById('inputbox-ok').click();
        expect(terminalInputBoxSubmitMock).toHaveBeenCalledWith(13, { value: '', variables: {} }, true);
    });

    it('blocks terminal input during confirmation and supports keyboard button activation', async () => {
        const { initInputBox } = await import('./inputbox.js');
        const { wireKeyboardEvents } = await import('./events.js');
        const canvas = document.createElement('canvas');
        document.body.appendChild(canvas);
        wireKeyboardEvents(canvas);
        initInputBox(canvas);
        const openHandler = eventsOnMock.mock.calls.find(([name]) => name === 'terminalInputBox')[1];
        openHandler({ id: 14, title: 'Confirm pane close', confirmation: true, confirmLabel: 'Kill' });

        const kill = document.getElementById('inputbox-ok');
        const cancel = document.getElementById('inputbox-cancel');
        cancel.focus();
        window.terminalFocusedState = true;
        for (const key of ['x', 'ArrowUp']) {
            for (const type of ['keydown', 'keypress', 'keyup']) {
                const event = new KeyboardEvent(type, { key, bubbles: true, cancelable: true });
                cancel.dispatchEvent(event);
                expect(event.defaultPrevented).toBe(true);
            }
        }
        cancel.dispatchEvent(new KeyboardEvent('keydown', { key: 'v', ctrlKey: true, bubbles: true, cancelable: true }));
        expect(terminalTextInputMock).not.toHaveBeenCalled();
        expect(terminalKeyPressMock).not.toHaveBeenCalled();
        expect(terminalInputBoxSubmitMock).not.toHaveBeenCalled();

        cancel.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }));
        expect(document.activeElement).toBe(kill);
        kill.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }));
        expect(document.activeElement).toBe(cancel);
        cancel.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, bubbles: true, cancelable: true }));
        expect(document.activeElement).toBe(kill);

        kill.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true }));
        expect(terminalInputBoxSubmitMock).toHaveBeenCalledWith(14, { value: '', variables: {} }, true);
        terminalInputBoxSubmitMock.mockClear();

        openHandler({ id: 14, title: 'Confirm pane close', confirmation: true, confirmLabel: 'Kill' });
        kill.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }));
        expect(terminalInputBoxSubmitMock).toHaveBeenCalledWith(14, { value: '', variables: {} }, false);
        expect(terminalKeyPressMock).not.toHaveBeenCalled();
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

    it('shows list default from options array when provided', async () => {
        const { initInputBox } = await import('./inputbox.js');
        const canvas = document.createElement('canvas');
        document.body.appendChild(canvas);

        initInputBox(canvas);

        const openHandler = eventsOnMock.mock.calls[0][1];
        openHandler({
            id: 23,
            title: 'Select model',
            defaultValue: 'prompt',
            placeholder: '',
            multiline: false,
            history: [],
            variables: [
                {
                    name: 'model',
                    label: 'Model',
                    type: 'list',
                    default: 'Claude: claude-3-5-sonnet',
                    options: [
                        'ChatGPT: gpt-4o',
                        'Claude: claude-3-5-sonnet',
                        'Claude: claude-3-haiku',
                    ],
                },
            ],
        });

        const listButton = document.querySelector('.inputbox-variable-list-btn');
        expect(listButton).toBeTruthy();
        expect(listButton.textContent).toBe('Claude: claude-3-5-sonnet');

        listButton.click();
        expect(showLocalMenuMock).toHaveBeenLastCalledWith(expect.objectContaining({
            title: 'Model',
            options: [
                'ChatGPT: gpt-4o',
                'Claude: claude-3-5-sonnet',
                'Claude: claude-3-haiku',
            ],
            onSelect: expect.any(Function),
        }));
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