import { beforeEach, describe, expect, it, vi } from 'vitest';

const getWindowStyleMock = vi.fn();
const getMarkdownMock = vi.fn();
const listFilesMock = vi.fn();
const saveFileMock = vi.fn(() => Promise.resolve());
const saveBinaryFileMock = vi.fn(() => Promise.resolve());
const deleteFileMock = vi.fn(() => Promise.resolve());
const renameFileMock = vi.fn(() => Promise.resolve());
const runNoteMock = vi.fn(() => Promise.resolve());
const stopNoteMock = vi.fn(() => Promise.resolve());
const sendIpcMock = vi.fn(() => Promise.resolve());
const sendToTerminalMock = vi.fn(() => Promise.resolve());
const getLanguageDescriptionsMock = vi.fn(() => Promise.resolve([]));
const getAllLanguageDescriptionsMock = vi.fn(() => Promise.resolve([]));
const terminalCopyImageDataURLMock = vi.fn(() => Promise.resolve(''));
const saveImageDialogMock = vi.fn(() => Promise.resolve(''));
const windowPrintMock = vi.fn(() => Promise.resolve());
const getClipboardDataMock = vi.fn(() => Promise.resolve({ text: '', image: '' }));
const swaggerRequestMock = vi.fn(() => Promise.resolve(''));
const resolveFilePathMock = vi.fn(() => Promise.resolve(''));
const getHyperlinkMenuActionsMock = vi.fn(() => Promise.resolve([]));
const runHyperlinkMenuActionMock = vi.fn(() => Promise.resolve());
const displayHyperlinkMenuMock = vi.fn(() => Promise.resolve());
const eventsOnMock = vi.fn();
const clipboardSetTextMock = vi.fn(() => Promise.resolve());
const showLocalMenuMock = vi.fn();

vi.mock('../wailsjs/go/main/WApp', () => ({
    GetWindowStyle: getWindowStyleMock,
    GetMarkdown: getMarkdownMock,
    ListFiles: listFilesMock,
    SaveFile: saveFileMock,
    SaveBinaryFile: saveBinaryFileMock,
    DeleteFile: deleteFileMock,
    RenameFile: renameFileMock,
    RunNote: runNoteMock,
    StopNote: stopNoteMock,
    SendIpc: sendIpcMock,
    SendToTerminal: sendToTerminalMock,
    GetLanguageDescriptions: getLanguageDescriptionsMock,
    GetAllLanguageDescriptions: getAllLanguageDescriptionsMock,
    TerminalCopyImageDataURL: terminalCopyImageDataURLMock,
    SaveImageDialog: saveImageDialogMock,
    WindowPrint: windowPrintMock,
    GetClipboardData: getClipboardDataMock,
    SwaggerRequest: swaggerRequestMock,
    ResolveFilePath: resolveFilePathMock,
    GetHyperlinkMenuActions: getHyperlinkMenuActionsMock,
    RunHyperlinkMenuAction: runHyperlinkMenuActionMock,
    DisplayHyperlinkMenu: displayHyperlinkMenuMock,
}));

vi.mock('../wailsjs/runtime/runtime', () => ({
    EventsOn: eventsOnMock,
    ClipboardSetText: clipboardSetTextMock,
}));

vi.mock('./popup_menu', () => ({
    showLocalMenu: showLocalMenuMock,
}));

vi.mock('./markdown-utils.js', () => ({
    configureMarked: vi.fn(),
    processMarkdownContainer: vi.fn(),
}));

vi.mock('./style-utils.js', () => ({
    getScrollbarStyles: vi.fn(() => ''),
    getMarkdownContentStyles: vi.fn(() => ''),
    getHighlightJsTheme: vi.fn(() => ''),
    getCheckboxStyles: vi.fn(() => ''),
    getMarkdownBaseTextSizeStyles: vi.fn(() => ''),
    getSwaggerUIStyles: vi.fn(() => ''),
}));

vi.mock('./swagger-utils.js', () => ({
    isStructuredDataFile: vi.fn((fileName) => /\.(json|ya?ml)$/i.test(fileName || '')),
    hasSwaggerKey: vi.fn(() => false),
    parseSwaggerSpec: vi.fn(() => null),
    generateRequestBuilderHTML: vi.fn(() => ''),
    generateResponseHTML: vi.fn(() => ''),
    extractPaths: vi.fn(() => []),
    generateEndpointListHTML: vi.fn(() => ''),
    buildRequestUrl: vi.fn(() => ''),
    generateLiveResponseHTML: vi.fn(() => ''),
    escapeInfoText: vi.fn((value) => String(value ?? '')),
}));

vi.mock('./json-viewer.js', () => ({
    attachJsonViewerEditHandler: vi.fn(),
    renderJsonViewer: vi.fn(),
}));

const theme = {
    colors: {
        fg: { Red: 230, Green: 237, Blue: 243 },
        bg: { Red: 30, Green: 34, Blue: 40 },
        yellow: { Red: 226, Green: 200, Blue: 92 },
        link: { Red: 110, Green: 170, Blue: 240 },
        red: { Red: 220, Green: 80, Blue: 80 },
        green: { Red: 61, Green: 127, Blue: 199 },
        blue: { Red: 80, Green: 110, Blue: 200 },
        magenta: { Red: 180, Green: 100, Blue: 210 },
        cyan: { Red: 90, Green: 180, Blue: 220 },
        redBright: { Red: 255, Green: 120, Blue: 120 },
        greenBright: { Red: 130, Green: 220, Blue: 160 },
        yellowBright: { Red: 245, Green: 220, Blue: 120 },
        blueBright: { Red: 140, Green: 170, Blue: 210 },
        magentaBright: { Red: 220, Green: 140, Blue: 240 },
        cyanBright: { Red: 130, Green: 220, Blue: 240 },
        selection: { Red: 49, Green: 109, Blue: 176 },
        error: { Red: 255, Green: 90, Blue: 90 },
    },
    fontFamily: 'sans-serif',
    fontSize: 14,
};

function flushPromises() {
    return new Promise((resolve) => {
        setTimeout(resolve, 0);
    });
}

async function importNotesModule() {
    vi.resetModules();
    await import('./notes.js');
    await flushPromises();
    await flushPromises();
}

describe('notes rendering', () => {
    beforeEach(() => {
        document.body.innerHTML = '<div id="notes-status"></div><div id="app"></div>';

        if (typeof window.ResizeObserver === 'undefined') {
            window.ResizeObserver = class {
                observe() {}
                disconnect() {}
                unobserve() {}
            };
        }

        getWindowStyleMock.mockReset();
        getMarkdownMock.mockReset();
        listFilesMock.mockReset();
        saveFileMock.mockClear();
        saveBinaryFileMock.mockClear();
        deleteFileMock.mockClear();
        renameFileMock.mockClear();
        runNoteMock.mockClear();
        stopNoteMock.mockClear();
        sendIpcMock.mockClear();
        sendToTerminalMock.mockClear();
        getLanguageDescriptionsMock.mockClear();
        getAllLanguageDescriptionsMock.mockClear();
        terminalCopyImageDataURLMock.mockClear();
        saveImageDialogMock.mockClear();
        windowPrintMock.mockClear();
        getClipboardDataMock.mockClear();
        swaggerRequestMock.mockClear();
        resolveFilePathMock.mockReset();
        getHyperlinkMenuActionsMock.mockReset();
        runHyperlinkMenuActionMock.mockReset();
        displayHyperlinkMenuMock.mockReset();
        eventsOnMock.mockReset();
        clipboardSetTextMock.mockClear();
        showLocalMenuMock.mockReset();

        getWindowStyleMock.mockResolvedValue(theme);
        getMarkdownMock.mockResolvedValue('');
        resolveFilePathMock.mockResolvedValue('');
        getHyperlinkMenuActionsMock.mockResolvedValue([]);
        clipboardSetTextMock.mockResolvedValue();
        runHyperlinkMenuActionMock.mockResolvedValue();
    });

    it('renders grouped note categories and nested files from the Wails file list', async () => {
        listFilesMock.mockResolvedValue([
            '$GLOBAL/docs/guide.md',
            '$NOTES/todo.md',
            '$PROJECT/apis/openapi.yaml',
            '$HISTORY/archive.md',
        ]);

        await importNotesModule();

        const categoryHeaders = Array.from(document.querySelectorAll('.notes-category-header')).map((node) => node.dataset.category);

        expect(categoryHeaders).toEqual(['$GLOBAL', '$NOTES', '$PROJECT', '$HISTORY']);
        expect(document.querySelector('.notes-tree-folder .notes-tree-label')?.textContent).toBe('docs');
        expect(document.querySelector('[data-file="$NOTES/todo.md"]')?.textContent).toContain('todo.md');
        expect(document.querySelector('[data-file="$PROJECT/apis/openapi.yaml"]')?.textContent).toContain('openapi.yaml');
    });

    it('filters the file list from the sidebar input and shows a no-match empty state', async () => {
        listFilesMock.mockResolvedValue([
            '$GLOBAL/docs/guide.md',
            '$GLOBAL/images/logo.png',
            '$NOTES/todo.md',
        ]);

        await importNotesModule();

        const filterInput = document.getElementById('notes-list-filter');
        filterInput.value = 'guide';
        filterInput.dispatchEvent(new Event('input', { bubbles: true }));

        const visibleFiles = Array.from(document.querySelectorAll('.notes-file')).map((node) => node.dataset.file);
        expect(visibleFiles).toEqual(['$GLOBAL/docs/guide.md']);
        expect(document.querySelector('.notes-tree-folder')?.dataset.expanded).toBe('true');

        filterInput.value = 'missing';
        filterInput.dispatchEvent(new Event('input', { bubbles: true }));

        expect(document.getElementById('notes-empty')?.textContent).toBe('No matching files.');

        filterInput.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));

        const restoredFiles = Array.from(document.querySelectorAll('.notes-file')).map((node) => node.dataset.file);
        expect(restoredFiles).toEqual(expect.arrayContaining([
            '$GLOBAL/docs/guide.md',
            '$GLOBAL/images/logo.png',
            '$NOTES/todo.md',
        ]));
    });

    it('loads markdown content when a rendered file entry is clicked', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/todo.md']);
        getMarkdownMock.mockResolvedValue('# Hello Notes');

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/todo.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        expect(getMarkdownMock).toHaveBeenCalledWith('$NOTES/todo.md');
        expect(document.getElementById('notes-preview')?.textContent).toContain('Hello Notes');
        expect(document.querySelector('[data-file="$NOTES/todo.md"]')?.dataset.active).toBe('true');
    });

    it('shows a file context menu with copy actions and Go-provided file handlers', async () => {
        listFilesMock.mockResolvedValue(['$PROJECT/docs/api.json']);
        resolveFilePathMock.mockResolvedValue('/tmp/project/docs/api.json');
        getHyperlinkMenuActionsMock.mockResolvedValue([
            { title: 'Copy file path to clipboard', icon: 0xf0c1, action: '0' },
            { title: 'Open file with Visual Studio Code', icon: 0xf08e, action: '3' },
            { title: '-', icon: 0, action: '' },
            { title: 'Rename file', icon: 0xf044, action: '4' },
            { title: 'Delete file', icon: 0xf1f8, action: '5' },
        ]);

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$PROJECT/docs/api.json"]');
        fileButton.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 48, clientY: 96 }));
        await flushPromises();

        expect(showLocalMenuMock).toHaveBeenCalledTimes(1);

        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        expect(menuConfig.title).toBe('api.json');
        expect(menuConfig.options).toEqual([
            'Copy file path to clipboard',
            'Open file with Visual Studio Code',
            '-',
            'Rename file',
            'Delete file',
        ]);

        expect(getHyperlinkMenuActionsMock).toHaveBeenCalledWith('file:///tmp/project/docs/api.json', 'api.json');

        menuConfig.onSelect(1);
        await flushPromises();
        expect(runHyperlinkMenuActionMock).toHaveBeenCalledWith('file:///tmp/project/docs/api.json', 'api.json', '3');

        menuConfig.onSelect(3);
        await flushPromises();
        expect(runHyperlinkMenuActionMock).toHaveBeenCalledWith('file:///tmp/project/docs/api.json', 'api.json', '4');

        menuConfig.onSelect(4);
        await flushPromises();
        expect(runHyperlinkMenuActionMock).toHaveBeenCalledWith('file:///tmp/project/docs/api.json', 'api.json', '5');
    });

    it('shows a hyperlink context menu when right-clicking an anchor in the markdown preview', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getMarkdownMock.mockResolvedValue('# Guide');
        displayHyperlinkMenuMock.mockResolvedValue();

        await importNotesModule();

        // Load the markdown file so viewMode becomes 'viewer'
        const fileButton = document.querySelector('[data-file="$NOTES/guide.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        // Inject an anchor directly into the rendered preview container
        const preview = document.getElementById('notes-preview');
        const anchor = document.createElement('a');
        anchor.href = 'https://example.com/docs';
        anchor.textContent = 'Docs site';
        preview.appendChild(anchor);

        anchor.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 100, clientY: 200 }));
        await flushPromises();
        await flushPromises();

        expect(displayHyperlinkMenuMock).toHaveBeenCalledWith('https://example.com/docs', 'Docs site');
        expect(showLocalMenuMock).not.toHaveBeenCalled();
    });

    it('uses href as fallback label when right-clicking an empty anchor label', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/readme.md']);
        getMarkdownMock.mockResolvedValue('# Readme');
        displayHyperlinkMenuMock.mockResolvedValue();

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/readme.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const preview = document.getElementById('notes-preview');
        const anchor = document.createElement('a');
        anchor.href = 'https://go.dev';
        anchor.textContent = '';
        preview.appendChild(anchor);

        anchor.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 50, clientY: 80 }));
        await flushPromises();
        await flushPromises();

        expect(displayHyperlinkMenuMock).toHaveBeenCalledWith('https://go.dev/', 'https://go.dev');
        expect(showLocalMenuMock).not.toHaveBeenCalled();
    });

    it('auto-copies markdown viewer selection when highlighted', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getMarkdownMock.mockResolvedValue('# Guide');

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/guide.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const textNode = document.createTextNode('Selected markdown text');
        document.getElementById('notes-preview').appendChild(textNode);

        const selectionMock = {
            rangeCount: 1,
            anchorNode: textNode,
            focusNode: textNode,
            toString: () => 'Selected markdown text',
        };
        const originalGetSelection = window.getSelection;
        window.getSelection = vi.fn(() => selectionMock);

        document.dispatchEvent(new MouseEvent('mouseup', { bubbles: true, button: 0 }));
        await flushPromises();
        expect(clipboardSetTextMock).toHaveBeenCalledWith('Selected markdown text');
        expect(sendIpcMock).toHaveBeenCalledWith('terminal-notify', {
            level: 'info',
            message: 'Selection copied to clipboard',
        });

        // Repeat with the same selection should not re-copy.
        document.dispatchEvent(new MouseEvent('mouseup', { bubbles: true, button: 0 }));
        await flushPromises();
        expect(clipboardSetTextMock).toHaveBeenCalledTimes(1);
        expect(sendIpcMock).toHaveBeenCalledTimes(1);

        window.getSelection = originalGetSelection;
    });

    it('shows single Edit tab only for code files and preserves markdown/yaml tabs', async () => {
        listFilesMock.mockResolvedValue([
            '$NOTES/readme.md',
            '$NOTES/script.go',
            '$NOTES/spec.yaml',
        ]);

        getMarkdownMock.mockImplementation(async (file) => {
            if (file.endsWith('.md')) {
                return '# Markdown note';
            }

            if (file.endsWith('.yaml')) {
                return 'openapi: 3.0.0\ninfo:\n  title: Sample';
            }

            return 'package main\n\nfunc main() {}';
        });

        await importNotesModule();

        const tabEditor = document.getElementById('notes-tab-editor');
        const tabViewer = document.getElementById('notes-tab-viewer');
        const tabJupyter = document.getElementById('notes-tab-jupyter');
        const tabSwaggerView = document.getElementById('notes-tab-swagger-view');
        const tabSwaggerEdit = document.getElementById('notes-tab-swagger-edit');
        const tabSwaggerRun = document.getElementById('notes-tab-swagger-run');

        const clickFile = async (filePath) => {
            const fileButton = document.querySelector(`[data-file="${filePath}"]`);
            fileButton.click();
            await flushPromises();
            await flushPromises();
        };

        await clickFile('$NOTES/script.go');
        expect(tabEditor.style.display).toBe('');
        expect(tabViewer.style.display).toBe('none');
        expect(tabJupyter.style.display).toBe('none');
        expect(tabSwaggerView.style.display).toBe('none');
        expect(tabSwaggerEdit.style.display).toBe('none');
        expect(tabSwaggerRun.style.display).toBe('none');

        await clickFile('$NOTES/readme.md');
        expect(tabEditor.style.display).toBe('');
        expect(tabViewer.style.display).toBe('');
        expect(tabJupyter.style.display).toBe('');
        expect(tabSwaggerView.style.display).toBe('none');
        expect(tabSwaggerEdit.style.display).toBe('none');
        expect(tabSwaggerRun.style.display).toBe('none');

        await clickFile('$NOTES/spec.yaml');
        expect(tabEditor.style.display).toBe('none');
        expect(tabViewer.style.display).toBe('none');
        expect(tabJupyter.style.display).toBe('none');
        expect(tabSwaggerView.style.display).toBe('');
        expect(tabSwaggerEdit.style.display).toBe('');
        expect(tabSwaggerRun.style.display).toBe('none');
    });

    it('cycles visible notes tabs with ctrl+tab', async () => {
        listFilesMock.mockResolvedValue([
            '$NOTES/readme.md',
            '$NOTES/spec.yaml',
        ]);

        getMarkdownMock.mockImplementation(async (file) => {
            if (file.endsWith('.yaml')) {
                return 'openapi: 3.0.0\ninfo:\n  title: Sample';
            }
            return '# Markdown note';
        });

        await importNotesModule();

        const clickFile = async (filePath) => {
            const fileButton = document.querySelector(`[data-file="${filePath}"]`);
            fileButton.click();
            await flushPromises();
            await flushPromises();
        };

        // Markdown defaults to View, then cycles View -> Edit -> Run -> View.
        await clickFile('$NOTES/readme.md');
        const tabViewer = document.getElementById('notes-tab-viewer');
        const tabEditor = document.getElementById('notes-tab-editor');
        const tabJupyter = document.getElementById('notes-tab-jupyter');

        expect(tabViewer.getAttribute('aria-selected')).toBe('true');
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        expect(tabEditor.getAttribute('aria-selected')).toBe('true');
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        expect(tabJupyter.getAttribute('aria-selected')).toBe('true');
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        expect(tabViewer.getAttribute('aria-selected')).toBe('true');

        // YAML defaults to structured View, then cycles View <-> Edit (Run hidden without swagger key).
        await clickFile('$NOTES/spec.yaml');
        const tabSwaggerView = document.getElementById('notes-tab-swagger-view');
        const tabSwaggerEdit = document.getElementById('notes-tab-swagger-edit');
        const tabSwaggerRun = document.getElementById('notes-tab-swagger-run');

        expect(tabSwaggerRun.style.display).toBe('none');
        tabSwaggerView.click();
        await flushPromises();
        const selectedBefore = tabSwaggerView.getAttribute('aria-selected') === 'true' ? 'view' : 'edit';
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        const selectedAfterFirst = tabSwaggerView.getAttribute('aria-selected') === 'true' ? 'view' : 'edit';
        expect(selectedAfterFirst).not.toBe(selectedBefore);
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        const selectedAfterSecond = tabSwaggerView.getAttribute('aria-selected') === 'true' ? 'view' : 'edit';
        expect(selectedAfterSecond).toBe(selectedBefore);
    });

    it('disables grammar helpers and keeps spellcheck enabled on note editors', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/readme.md']);
        getMarkdownMock.mockResolvedValue('# Note\n\n```js\nconsole.log("hello")\n```');

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/readme.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const notesEditor = document.getElementById('notes-editor');
        const swaggerEditor = document.getElementById('notes-swagger-editor');
        const jupyterEditor = document.querySelector('.jupyter-code-editable');

        expect(notesEditor.getAttribute('autocorrect')).toBe('off');
        expect(notesEditor.getAttribute('autocapitalize')).toBe('off');
        expect(notesEditor.getAttribute('autocomplete')).toBe('off');
        expect(notesEditor.getAttribute('data-gramm')).toBe('false');
        expect(notesEditor.getAttribute('data-gramm_editor')).toBe('false');
        expect(notesEditor.getAttribute('data-enable-grammarly')).toBe('false');
        expect(notesEditor.getAttribute('spellcheck')).toBeNull();

        expect(swaggerEditor.getAttribute('autocorrect')).toBe('off');
        expect(swaggerEditor.getAttribute('autocapitalize')).toBe('off');
        expect(swaggerEditor.getAttribute('autocomplete')).toBe('off');
        expect(swaggerEditor.getAttribute('data-gramm')).toBe('false');
        expect(swaggerEditor.getAttribute('data-gramm_editor')).toBe('false');
        expect(swaggerEditor.getAttribute('data-enable-grammarly')).toBe('false');
        expect(swaggerEditor.getAttribute('spellcheck')).toBeNull();

        expect(jupyterEditor).toBeTruthy();
        expect(jupyterEditor.getAttribute('autocorrect')).toBe('off');
        expect(jupyterEditor.getAttribute('autocapitalize')).toBe('off');
        expect(jupyterEditor.getAttribute('autocomplete')).toBe('off');
        expect(jupyterEditor.getAttribute('data-gramm')).toBe('false');
        expect(jupyterEditor.getAttribute('data-gramm_editor')).toBe('false');
        expect(jupyterEditor.getAttribute('data-enable-grammarly')).toBe('false');
        expect(jupyterEditor.getAttribute('spellcheck')).toBeNull();
    });
});