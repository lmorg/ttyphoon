import { beforeEach, describe, expect, it, vi } from 'vitest';

const getWindowStyleMock = vi.fn();
const getFileMock = vi.fn();
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
const getCurrentProjectMock = vi.fn(() => Promise.resolve(''));
const getFileMetaMarkdownMock = vi.fn(() => Promise.resolve([
    '# note.md',
    '',
    '## Attributes',
    '',
    '- Size: `0`',
    '- Path: `/tmp/note.md`',
    '',
    '## Owners',
    '',
    '- User: `user`',
    '- Group: `group`',
    '',
    '## Permissions',
    '',
    '- Unix: `0644`',
    '- User: `rw-`',
    '- Group: `r--`',
    '- Other: `r--`',
].join('\n')));
const resolveFilePathMock = vi.fn(() => Promise.resolve(''));
const getHyperlinkMenuActionsMock = vi.fn(() => Promise.resolve([]));
const runHyperlinkMenuActionMock = vi.fn(() => Promise.resolve());
const displayHyperlinkMenuMock = vi.fn(() => Promise.resolve());
const eventsOnMock = vi.fn();
const clipboardSetTextMock = vi.fn(() => Promise.resolve());
const showLocalMenuMock = vi.fn();

vi.mock('../wailsjs/go/main/WApp', () => ({
    GetWindowStyle: getWindowStyleMock,
    GetFile: getFileMock,
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
    GetCurrentProject: getCurrentProjectMock,
    GetFileMetaMarkdown: getFileMetaMarkdownMock,
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
    DARKEN_BACKGROUND_OVERLAY: 'rgba(0, 0, 0, 0.2)',
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
        getFileMock.mockReset();
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
        getCurrentProjectMock.mockReset();
        getFileMetaMarkdownMock.mockReset();
        resolveFilePathMock.mockReset();
        getHyperlinkMenuActionsMock.mockReset();
        runHyperlinkMenuActionMock.mockReset();
        displayHyperlinkMenuMock.mockReset();
        eventsOnMock.mockReset();
        clipboardSetTextMock.mockClear();
        showLocalMenuMock.mockReset();

        getWindowStyleMock.mockResolvedValue(theme);
        getFileMock.mockResolvedValue({ contents: '', text: '', error: '' });
        getCurrentProjectMock.mockResolvedValue('');
        getFileMetaMarkdownMock.mockResolvedValue([
            '# note.md',
            '',
            '## Attributes',
            '',
            '- Size: `0`',
            '- Path: `/tmp/note.md`',
            '',
            '## Owners',
            '',
            '- User: `user`',
            '- Group: `group`',
            '',
            '## Permissions',
            '',
            '- Unix: `0644`',
            '- User: `rw-`',
            '- Group: `r--`',
            '- Other: `r--`',
        ].join('\n'));
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
        const clearButton = document.getElementById('notes-list-filter-clear');
        expect(clearButton?.dataset.visible).toBe('false');

        filterInput.value = 'guide';
        filterInput.dispatchEvent(new Event('input', { bubbles: true }));
        expect(clearButton?.dataset.visible).toBe('true');

        const visibleFiles = Array.from(document.querySelectorAll('.notes-file')).map((node) => node.dataset.file);
        expect(visibleFiles).toEqual(['$GLOBAL/docs/guide.md']);
        expect(document.querySelector('.notes-tree-folder')?.dataset.expanded).toBe('true');

        filterInput.value = 'missing';
        filterInput.dispatchEvent(new Event('input', { bubbles: true }));

        expect(document.getElementById('notes-empty')?.textContent).toBe('No matching files.');

        clearButton.click();
        expect(filterInput.value).toBe('');
        expect(clearButton?.dataset.visible).toBe('false');

        const restoredAfterClear = Array.from(document.querySelectorAll('.notes-file')).map((node) => node.dataset.file);
        expect(restoredAfterClear).toEqual(expect.arrayContaining([
            '$GLOBAL/docs/guide.md',
            '$GLOBAL/images/logo.png',
            '$NOTES/todo.md',
        ]));

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
        getFileMock.mockResolvedValue({ contents: '# Hello Notes', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/todo.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        expect(getFileMock).toHaveBeenCalledWith('$NOTES/todo.md');
        expect(document.getElementById('notes-preview')?.textContent).toContain('Hello Notes');
        expect(document.querySelector('[data-file="$NOTES/todo.md"]')?.dataset.active).toBe('true');
    });

    it('renames notes using the exact modal path and extension without forcing .md', async () => {
        listFilesMock
            .mockResolvedValueOnce(['$GLOBAL/docs/todo.md'])
            .mockResolvedValueOnce(['$PROJECT/docs/todo.txt']);
        getFileMock.mockResolvedValue({ contents: '# Hello Notes', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$GLOBAL/docs/todo.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        document.getElementById('notes-rename').click();
        await flushPromises();

        const modalInput = document.getElementById('notes-modal-input');
        expect(modalInput.value).toBe('$GLOBAL/docs/todo.md');

        modalInput.value = '$PROJECT/docs/todo.txt';
        document.getElementById('notes-modal-create').click();
        await flushPromises();
        await flushPromises();

        expect(renameFileMock).toHaveBeenCalledWith('$GLOBAL/docs/todo.md', '$PROJECT/docs/todo.txt');
        expect(renameFileMock).not.toHaveBeenCalledWith('$GLOBAL/docs/todo.md', '$PROJECT/docs/todo.txt.md');
    });

    it('focuses the textarea whenever an Edit view becomes active', async () => {
        listFilesMock.mockResolvedValue([
            '$NOTES/script.go',
            '$NOTES/readme.md',
            '$NOTES/spec.yaml',
        ]);

        getFileMock.mockImplementation(async (file) => {
            if (file.endsWith('.go')) {
                return { contents: 'package main\n\nfunc main() {}', text: '', error: '' };
            }

            if (file.endsWith('.yaml')) {
                return { contents: 'openapi: 3.0.0\ninfo:\n  title: Sample', text: '', error: '' };
            }

            return { contents: '# Markdown note', text: '', error: '' };
        });

        await importNotesModule();

        const notesEditor = document.getElementById('notes-editor');
        const clickFile = async (filePath) => {
            const fileButton = document.querySelector(`[data-file="${filePath}"]`);
            fileButton.click();
            await flushPromises();
            await flushPromises();
        };

        await clickFile('$NOTES/script.go');
        expect(document.activeElement).toBe(notesEditor);

        await clickFile('$NOTES/readme.md');

        document.getElementById('notes-tab-editor').click();
        await flushPromises();
        expect(document.activeElement).toBe(notesEditor);

        await clickFile('$NOTES/spec.yaml');

        document.getElementById('notes-tab-swagger-edit').click();
        await flushPromises();
        expect(document.activeElement).toBe(notesEditor);

        document.getElementById('notes-tab-swagger-view').click();
        await flushPromises();
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        await flushPromises();
        expect(document.activeElement).toBe(notesEditor);
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

    it('shows a folder-tree menu on category headers and applies collapse/expand actions', async () => {
        listFilesMock.mockResolvedValue([
            '$PROJECT/docs/guide.md',
            '$PROJECT/docs/reference/api.md',
            '$PROJECT/images/logo.png',
        ]);

        await importNotesModule();

        const projectHeader = document.querySelector('.notes-category-header[data-category="$PROJECT"]');
        projectHeader.dispatchEvent(new MouseEvent('contextmenu', {
            bubbles: true,
            cancelable: true,
            clientX: 64,
            clientY: 128,
        }));

        expect(showLocalMenuMock).toHaveBeenCalledTimes(1);
        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        expect(menuConfig.options).toEqual(['Collapse Folders', 'Expand Folders']);
        expect(menuConfig.icons).toEqual([0xf146, 0xf0fe]);

        menuConfig.onSelect(0);
        await flushPromises();

        const collapsedFolders = Array.from(document.querySelectorAll('.notes-tree-folder'));
        expect(collapsedFolders.length).toBeGreaterThan(0);
        expect(collapsedFolders.every((folder) => folder.dataset.expanded === 'false')).toBe(true);

        menuConfig.onSelect(1);
        await flushPromises();

        const expandedFolders = Array.from(document.querySelectorAll('.notes-tree-folder'));
        expect(expandedFolders.length).toBeGreaterThan(0);
        expect(expandedFolders.every((folder) => folder.dataset.expanded === 'true')).toBe(true);
    });

    it('shows a folder-tree menu on directory items and applies actions to child folders only', async () => {
        listFilesMock.mockResolvedValue([
            '$PROJECT/docs/reference/api.md',
            '$PROJECT/docs/readme.md',
            '$PROJECT/images/logo.png',
        ]);

        await importNotesModule();

        const docsFolder = document.querySelector('.notes-tree-folder[data-folder-key="$PROJECT/docs"]');
        docsFolder.dispatchEvent(new MouseEvent('contextmenu', {
            bubbles: true,
            cancelable: true,
            clientX: 72,
            clientY: 140,
        }));

        expect(showLocalMenuMock).toHaveBeenCalledTimes(1);
        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        expect(menuConfig.options).toEqual(['Collapse Folders', 'Expand Folders']);

        menuConfig.onSelect(0);
        await flushPromises();

        expect(document.querySelector('.notes-tree-folder[data-folder-key="$PROJECT/docs"]')?.dataset.expanded).toBe('true');
        expect(document.querySelector('.notes-tree-folder[data-folder-key="$PROJECT/docs/reference"]')?.dataset.expanded).toBe('false');
        expect(document.querySelector('.notes-tree-folder[data-folder-key="$PROJECT/images"]')?.dataset.expanded).toBe('true');

        menuConfig.onSelect(1);
        await flushPromises();

        expect(document.querySelector('.notes-tree-folder[data-folder-key="$PROJECT/docs/reference"]')?.dataset.expanded).toBe('true');
    });

    it('shows a hyperlink context menu when right-clicking an anchor in the markdown preview', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide', text: '', error: '' });
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
        getFileMock.mockResolvedValue({ contents: '# Readme', text: '', error: '' });
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
        getFileMock.mockResolvedValue({ contents: '# Guide', text: '', error: '' });

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

        getFileMock.mockImplementation(async (file) => {
            if (file.endsWith('.md')) {
                return { contents: '# Markdown note', text: '', error: '' };
            }

            if (file.endsWith('.yaml')) {
                return { contents: 'openapi: 3.0.0\ninfo:\n  title: Sample', text: '', error: '' };
            }

            return { contents: 'package main\n\nfunc main() {}', text: '', error: '' };
        });

        await importNotesModule();

        const tabEditor = document.getElementById('notes-tab-editor');
        const tabViewer = document.getElementById('notes-tab-viewer');
        const tabJupyter = document.getElementById('notes-tab-jupyter');
        const tabMeta = document.getElementById('notes-tab-meta');
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
        expect(tabMeta.style.display).toBe('');

        await clickFile('$NOTES/readme.md');
        expect(tabEditor.style.display).toBe('');
        expect(tabViewer.style.display).toBe('');
        expect(tabJupyter.style.display).toBe('');
        expect(tabSwaggerView.style.display).toBe('none');
        expect(tabSwaggerEdit.style.display).toBe('none');
        expect(tabSwaggerRun.style.display).toBe('none');
        expect(tabMeta.style.display).toBe('');

        await clickFile('$NOTES/spec.yaml');
        expect(tabEditor.style.display).toBe('none');
        expect(tabViewer.style.display).toBe('none');
        expect(tabJupyter.style.display).toBe('none');
        expect(tabSwaggerView.style.display).toBe('');
        expect(tabSwaggerEdit.style.display).toBe('');
        expect(tabSwaggerRun.style.display).toBe('none');
        expect(tabMeta.style.display).toBe('');
    });

    it('shows a Hex tab for binary files and renders hexdump output', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/app.bin']);
        getFileMock.mockResolvedValue({
            contents: 'Y2YgAAAAAABoZXh5CgAAAA==',
            binary: true,
            error: '',
        });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/app.bin"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const tabEditor = document.getElementById('notes-tab-editor');
        const tabViewer = document.getElementById('notes-tab-viewer');
        const tabMeta = document.getElementById('notes-tab-meta');
        const hexWrap = document.getElementById('notes-hex-wrap');
        const hexRoot = document.getElementById('notes-hex');
        const editorWrap = document.getElementById('notes-editor-wrap');
        const hexHeader = document.querySelector('.notes-hex-header');
        const offsetInput = document.querySelector('.notes-hex-offset-input');
        const goButton = document.querySelector('.notes-hex-offset-go');

        expect(tabEditor.textContent).toBe('Hex');
        expect(tabEditor.getAttribute('aria-selected')).toBe('true');
        expect(tabViewer.style.display).toBe('none');
        expect(tabMeta.style.display).toBe('');
        expect(hexWrap.dataset.active).toBe('true');
        expect(editorWrap.dataset.active).toBe('false');
        expect(hexRoot.textContent).toContain('00000000');
        expect(hexRoot.textContent).toContain('63 66 20 00 00 00 00 00');
        expect(hexRoot.textContent).toContain('|cf .....hexy....|');
        expect(hexHeader.textContent).toContain('Offset');
        expect(offsetInput).toBeTruthy();
        expect(goButton).toBeTruthy();
    });

    it('shows meta pane only when meta tab is selected and renders template-provided markdown', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/readme.md']);
        getFileMock.mockResolvedValue({ contents: '# Markdown note', text: '', error: '' });
        getFileMetaMarkdownMock.mockResolvedValue([
            '# readme.md',
            '',
            '## Attributes',
            '',
            '- Size: `123Bb`',
            '- Path:',
            '  ```',
            '  /tmp/readme.md',
            '  ```',
            '',
            '## Owners',
            '',
            '- User: `user`',
            '- Group: `group`',
            '',
            '## Permissions',
            '',
            '- Unix: `0644`',
            '- User: `rw-`',
            '- Group: `r--`',
            '- Other: `r--`',
        ].join('\n'));

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/readme.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const tabViewer = document.getElementById('notes-tab-viewer');
        const tabMeta = document.getElementById('notes-tab-meta');
        const previewWrap = document.getElementById('notes-preview-wrap');
        const metaWrap = document.getElementById('notes-meta-wrap');
        const metaRoot = document.getElementById('notes-meta');

        expect(tabViewer.getAttribute('aria-selected')).toBe('true');
        expect(metaWrap.dataset.active).toBe('false');
        expect(previewWrap.dataset.active).toBe('true');

        tabMeta.click();
        await flushPromises();

        expect(tabMeta.getAttribute('aria-selected')).toBe('true');
        expect(metaWrap.dataset.active).toBe('true');
        expect(previewWrap.dataset.active).toBe('false');
        expect(metaRoot.querySelector('pre')).toBeTruthy();
        expect(metaRoot.querySelectorAll('code').length).toBeGreaterThan(0);
    });

    it('keeps the code-style highlight layer as wide as the scrollable editor content', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/config.json']);
        getFileMock.mockResolvedValue({ contents: '{"longPropertyName": "value"}', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/config.json"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const notesEditor = document.getElementById('notes-editor');
        const editorHighlight = document.getElementById('notes-editor-highlight');

        Object.defineProperty(notesEditor, 'scrollWidth', { configurable: true, value: 640 });
        Object.defineProperty(notesEditor, 'clientWidth', { configurable: true, value: 320 });
        Object.defineProperty(notesEditor, 'scrollHeight', { configurable: true, value: 240 });
        Object.defineProperty(notesEditor, 'clientHeight', { configurable: true, value: 180 });

        notesEditor.value = '{"veryLongPropertyNameThatForcesHorizontalScrolling": "value"}';
        notesEditor.dispatchEvent(new Event('input', { bubbles: true }));

        expect(editorHighlight.style.minWidth).toBe('640px');
        expect(editorHighlight.style.minHeight).toBe('240px');
    });

    it('cycles visible notes tabs with ctrl+tab', async () => {
        listFilesMock.mockResolvedValue([
            '$NOTES/readme.md',
            '$NOTES/spec.yaml',
        ]);

        getFileMock.mockImplementation(async (file) => {
            if (file.endsWith('.yaml')) {
                return { contents: 'openapi: 3.0.0\ninfo:\n  title: Sample', text: '', error: '' };
            }
            return { contents: '# Markdown note', text: '', error: '' };
        });

        await importNotesModule();

        const clickFile = async (filePath) => {
            const fileButton = document.querySelector(`[data-file="${filePath}"]`);
            fileButton.click();
            await flushPromises();
            await flushPromises();
        };

        // Markdown defaults to View, then cycles View -> Edit -> Run -> Meta -> View.
        await clickFile('$NOTES/readme.md');
        const tabViewer = document.getElementById('notes-tab-viewer');
        const tabEditor = document.getElementById('notes-tab-editor');
        const tabJupyter = document.getElementById('notes-tab-jupyter');
        const tabMeta = document.getElementById('notes-tab-meta');

        expect(tabViewer.getAttribute('aria-selected')).toBe('true');
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        expect(tabEditor.getAttribute('aria-selected')).toBe('true');
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        expect(tabJupyter.getAttribute('aria-selected')).toBe('true');
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        expect(tabMeta.getAttribute('aria-selected')).toBe('true');
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        expect(tabViewer.getAttribute('aria-selected')).toBe('true');

        // YAML defaults to structured View, then cycles View -> Edit -> Meta -> View (Run hidden without swagger key).
        await clickFile('$NOTES/spec.yaml');
        const tabSwaggerView = document.getElementById('notes-tab-swagger-view');
        const tabSwaggerEdit = document.getElementById('notes-tab-swagger-edit');
        const tabSwaggerRun = document.getElementById('notes-tab-swagger-run');

        expect(tabSwaggerRun.style.display).toBe('none');
        tabSwaggerView.click();
        await flushPromises();
        const selectedBefore = tabSwaggerView.getAttribute('aria-selected') === 'true' ? 'view' : 'edit';
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        const selectedAfterFirst = tabSwaggerView.getAttribute('aria-selected') === 'true'
            ? 'view'
            : (tabSwaggerEdit.getAttribute('aria-selected') === 'true' ? 'edit' : 'meta');
        expect(selectedAfterFirst).not.toBe(selectedBefore);
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        const selectedAfterSecond = tabSwaggerView.getAttribute('aria-selected') === 'true'
            ? 'view'
            : (tabSwaggerEdit.getAttribute('aria-selected') === 'true' ? 'edit' : 'meta');
        expect(selectedAfterSecond).not.toBe(selectedBefore);
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        const selectedAfterThird = tabSwaggerView.getAttribute('aria-selected') === 'true'
            ? 'view'
            : (tabSwaggerEdit.getAttribute('aria-selected') === 'true' ? 'edit' : 'meta');
        expect(selectedAfterThird).toBe(selectedBefore);
    });

    it('disables grammar helpers and keeps spellcheck enabled on note editors', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/readme.md']);
        getFileMock.mockResolvedValue({ contents: '# Note\n\n```js\nconsole.log("hello")\n```', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/readme.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const notesEditor = document.getElementById('notes-editor');
        const jupyterEditor = document.querySelector('.jupyter-code-editable');

        expect(notesEditor.getAttribute('autocorrect')).toBe('off');
        expect(notesEditor.getAttribute('autocapitalize')).toBe('off');
        expect(notesEditor.getAttribute('autocomplete')).toBe('off');
        expect(notesEditor.getAttribute('data-gramm')).toBe('false');
        expect(notesEditor.getAttribute('data-gramm_editor')).toBe('false');
        expect(notesEditor.getAttribute('data-enable-grammarly')).toBe('false');
        expect(notesEditor.getAttribute('spellcheck')).toBe('false');

        expect(jupyterEditor).toBeTruthy();
        expect(jupyterEditor.getAttribute('autocorrect')).toBe('off');
        expect(jupyterEditor.getAttribute('autocapitalize')).toBe('off');
        expect(jupyterEditor.getAttribute('autocomplete')).toBe('off');
        expect(jupyterEditor.getAttribute('data-gramm')).toBe('false');
        expect(jupyterEditor.getAttribute('data-gramm_editor')).toBe('false');
        expect(jupyterEditor.getAttribute('data-enable-grammarly')).toBe('false');
        expect(jupyterEditor.getAttribute('spellcheck')).toBeNull();
    });

    it('edits markdown table cells on double click in Run tab', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/table.md']);
        getFileMock.mockResolvedValue({ contents: [
            '# Table',
            '',
            '| Name | Value |',
            '| --- | --- |',
            '| Alpha | 1 |',
            '| Beta | 2 |',
        ].join('\n'), text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/table.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const runTab = document.getElementById('notes-tab-jupyter');
        runTab.click();
        await flushPromises();
        await flushPromises();

        const targetCell = document.querySelector('#notes-jupyter tbody tr td');
        expect(targetCell).toBeTruthy();

        targetCell.dispatchEvent(new MouseEvent('dblclick', { bubbles: true, cancelable: true }));
        await flushPromises();

        expect(targetCell.getAttribute('contenteditable')).toBe('true');

        targetCell.textContent = 'Gamma';
        targetCell.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true }));
        await flushPromises();
        await flushPromises();

        const notesEditor = document.getElementById('notes-editor');
        expect(notesEditor.value).toContain('| Gamma | 1 |');
    });
});