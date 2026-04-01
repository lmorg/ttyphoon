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
        eventsOnMock.mockReset();
        clipboardSetTextMock.mockReset();
        showLocalMenuMock.mockReset();

        getWindowStyleMock.mockResolvedValue(theme);
        getMarkdownMock.mockResolvedValue('');
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
});