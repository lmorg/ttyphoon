import { GetWindowStyle, GetMarkdown, GetParameters } from '../wailsjs/go/main/WApp';
import { BrowserOpenURL } from '../wailsjs/runtime/runtime';

import { marked } from "marked";
import { gfmHeadingId } from "marked-gfm-heading-id";
import hljs from "highlight.js/lib/common";

const app = document.getElementById('app') || (() => {
    const root = document.createElement('div');
    root.id = 'app';
    document.body.appendChild(root);
    return root;
})();

document.title = 'Notes';

app.innerHTML = `
    <div id="notes-app">
        <aside id="notes-sidebar">
            <div id="notes-sidebar-header">
                <div id="notes-title">Notes</div>
                <div id="notes-actions">
                    <button id="notes-new" type="button">New</button>
                    <button id="notes-refresh" type="button">Refresh</button>
                    <button id="notes-save" type="button">Save</button>
                </div>
            </div>
            <div id="notes-list" role="list"></div>
            <div id="notes-status" role="status"></div>
        </aside>
        <main id="notes-main">
            <div id="notes-tabs" role="tablist">
                <button id="notes-tab-editor" type="button" role="tab" aria-selected="true">Editor</button>
                <button id="notes-tab-viewer" type="button" role="tab" aria-selected="false">Viewer</button>
            </div>
            <div id="notes-panel">
                <div id="notes-editor-wrap" role="tabpanel">
                    <textarea id="notes-editor" spellcheck="false"></textarea>
                </div>
                <div id="notes-preview-wrap" class="markdown-body" role="tabpanel">
                    <div id="notes-preview"></div>
                </div>
            </div>
        </main>
    </div>
    <div id="notes-modal" data-open="false" aria-hidden="true">
        <div id="notes-modal-card" role="dialog" aria-modal="true" aria-labelledby="notes-modal-title">
            <div id="notes-modal-title">New note name</div>
            <input id="notes-modal-input" type="text" placeholder="example-note" autocomplete="off" />
            <div id="notes-modal-actions">
                <button id="notes-modal-cancel" type="button">Cancel</button>
                <button id="notes-modal-create" type="button">Create</button>
            </div>
        </div>
    </div>
`;

const elements = {
    list: document.getElementById('notes-list'),
    editor: document.getElementById('notes-editor'),
    preview: document.getElementById('notes-preview'),
    status: document.getElementById('notes-status'),
    newFile: document.getElementById('notes-new'),
    save: document.getElementById('notes-save'),
    refresh: document.getElementById('notes-refresh'),
    tabEditor: document.getElementById('notes-tab-editor'),
    tabViewer: document.getElementById('notes-tab-viewer'),
    editorWrap: document.getElementById('notes-editor-wrap'),
    previewWrap: document.getElementById('notes-preview-wrap'),
    modal: document.getElementById('notes-modal'),
    modalInput: document.getElementById('notes-modal-input'),
    modalCancel: document.getElementById('notes-modal-cancel'),
    modalCreate: document.getElementById('notes-modal-create')
};

const state = {
    files: [],
    currentFile: '',
    dirty: false,
    renderTimer: null,
    viewMode: 'editor'
};

marked.use(gfmHeadingId({}));

function getWailsFunction(name) {
    const fn = window && window.go && window.go.main && window.go.main.WApp && window.go.main.WApp[name];
    return typeof fn === 'function' ? fn : null;
}

function setStatus(message, isError) {
    elements.status.textContent = message || '';
    elements.status.dataset.state = isError ? 'error' : 'ok';
}

function renderMarkdown() {
    const markdown = elements.editor.value || '';
    elements.preview.innerHTML = marked.parse(markdown);

    elements.preview.querySelectorAll('pre code').forEach((block) => {
        hljs.highlightElement(block);
    });

    elements.preview.querySelectorAll('a').forEach((link) => {
        link.addEventListener('click', (event) => {
            event.preventDefault();
            BrowserOpenURL(link.href);
        });
    });
}

function scheduleRender() {
    if (state.renderTimer) {
        clearTimeout(state.renderTimer);
    }
    state.renderTimer = setTimeout(() => {
        state.renderTimer = null;
        renderMarkdown();
    }, 120);
}

function setDirty(isDirty) {
    state.dirty = isDirty;
    const label = state.currentFile ? state.currentFile : 'No file selected';
    elements.status.textContent = isDirty ? `${label} (unsaved)` : label;
}

function setViewMode(mode) {
    state.viewMode = mode === 'viewer' ? 'viewer' : 'editor';
    const isEditor = state.viewMode === 'editor';
    elements.tabEditor.setAttribute('aria-selected', isEditor ? 'true' : 'false');
    elements.tabViewer.setAttribute('aria-selected', isEditor ? 'false' : 'true');
    elements.editorWrap.dataset.active = isEditor ? 'true' : 'false';
    elements.previewWrap.dataset.active = isEditor ? 'false' : 'true';
}

async function refreshFiles() {
    const listFn = getWailsFunction('ListFiles');
    if (!listFn) {
        setStatus('ListFiles is not available.', true);
        return;
    }

    try {
        const files = await listFn();
        state.files = Array.isArray(files) ? files : [];
        renderFileList();
    } catch (err) {
        setStatus('Failed to load file list.', true);
        console.error(err);
    }
}

function renderFileList() {
    elements.list.innerHTML = '';

    if (state.files.length === 0) {
        const empty = document.createElement('div');
        empty.id = 'notes-empty';
        empty.textContent = 'No notes found.';
        elements.list.appendChild(empty);
        return;
    }

    state.files.forEach((file) => {
        const item = document.createElement('button');
        item.type = 'button';
        item.className = 'notes-file';
        item.textContent = file;
        item.dataset.file = file;
        if (file === state.currentFile) {
            item.dataset.active = 'true';
        }
        item.addEventListener('click', () => {
            loadFile(file);
        });
        elements.list.appendChild(item);
    });
}

async function loadFile(file) {
    if (!file) {
        return;
    }

    try {
        const doc = await GetMarkdown(file);
        state.currentFile = file;
        elements.editor.value = doc || '';
        renderMarkdown();
        setDirty(false);
        renderFileList();
    } catch (err) {
        setStatus(`Failed to load ${file}.`, true);
        console.error(err);
    }
}

async function saveFile() {
    if (!state.currentFile) {
        setStatus('Select a note before saving.', true);
        return;
    }

    const saveFn = getWailsFunction('SaveFile');
    if (!saveFn) {
        setStatus('SaveFile is not available.', true);
        return;
    }

    try {
        await saveFn(state.currentFile, elements.editor.value);
        setDirty(false);
    } catch (err) {
        setStatus(`Failed to save ${state.currentFile}.`, true);
        console.error(err);
    }
}

function openNewFilePrompt() {
    elements.modal.dataset.open = 'true';
    elements.modal.setAttribute('aria-hidden', 'false');
    elements.modalInput.value = '';
    setTimeout(() => {
        elements.modalInput.focus();
    }, 0);
}

function closeNewFilePrompt() {
    elements.modal.dataset.open = 'false';
    elements.modal.setAttribute('aria-hidden', 'true');
    elements.newFile.focus();
}

function normalizeNoteName(rawName) {
    const trimmed = (rawName || '').trim();
    if (trimmed === '') {
        return '';
    }

    if (trimmed.toLowerCase().endsWith('.md')) {
        return trimmed;
    }

    return `${trimmed}.md`;
}

async function createNewFile() {
    const fileName = normalizeNoteName(elements.modalInput.value);
    if (fileName === '') {
        setStatus('File name cannot be empty.', true);
        return;
    }

    const exists = state.files.some((file) => file === fileName);
    if (exists) {
        closeNewFilePrompt();
        await loadFile(fileName);
        setStatus(`${fileName} already exists.`, false);
        return;
    }

    const saveFn = getWailsFunction('SaveFile');
    if (!saveFn) {
        setStatus('SaveFile is not available.', true);
        return;
    }

    try {
        await saveFn(fileName, '');
        await refreshFiles();
        await loadFile(fileName);
        setViewMode('editor');
        closeNewFilePrompt();
        setStatus(`Created ${fileName}.`, false);
    } catch (err) {
        setStatus(`Failed to create ${fileName}.`, true);
        console.error(err);
    }
}

function applyWindowStyle(result) {
    document.body.style.color = `rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue})`;
    document.body.style.backgroundColor = `rgb(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue})`;

    const style = document.createElement('style');
    style.textContent = `
        :root {
            --bg: rgb(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue});
            --fg: rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue});
            --accent: rgb(${result.colors.yellow.Red}, ${result.colors.yellow.Green}, ${result.colors.yellow.Blue});
            --link: rgb(${result.colors.link.Red}, ${result.colors.link.Green}, ${result.colors.link.Blue});
            --green: rgb(${result.colors.green.Red}, ${result.colors.green.Green}, ${result.colors.green.Blue});
            --magenta: rgb(${result.colors.magenta.Red}, ${result.colors.magenta.Green}, ${result.colors.magenta.Blue});
            --cyan: rgb(${result.colors.cyan.Red}, ${result.colors.cyan.Green}, ${result.colors.cyan.Blue});
            --red: rgb(${result.colors.red.Red}, ${result.colors.red.Green}, ${result.colors.red.Blue});
            --blue-bright: rgb(${result.colors.blueBright.Red}, ${result.colors.blueBright.Green}, ${result.colors.blueBright.Blue});
            --selection: rgb(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue});
            --error: rgb(${result.colors.error.Red}, ${result.colors.error.Green}, ${result.colors.error.Blue});
        }

        * {
            box-sizing: border-box;
            font-family: ${result.fontFamily};
        }

        body {
            margin: 0;
            padding: 0;
        }

        ::selection {
            background-color: var(--selection);
        }

        #notes-app {
            display: grid;
            grid-template-columns: 260px 1fr;
            height: 100vh;
            color: var(--fg);
            background: var(--bg);
        }

        #notes-sidebar {
            display: flex;
            flex-direction: column;
            border-right: 2px solid var(--fg);
            padding: 16px;
            gap: 12px;
        }

        #notes-sidebar-header {
            display: flex;
            flex-direction: column;
            gap: 12px;
        }

        #notes-title {
            font-size: calc(${result.fontSize}px + 4px);
            color: var(--accent);
        }

        #notes-actions {
            display: flex;
            gap: 10px;
        }

        #notes-actions button {
            border-radius: 0;
            border: 2px solid var(--fg);
            background: transparent;
            color: var(--fg);
            padding: 6px 10px;
            cursor: pointer;
        }

        #notes-actions button:hover {
            border-color: var(--accent);
            color: var(--accent);
        }

        #notes-modal {
            position: fixed;
            inset: 0;
            display: none;
            align-items: center;
            justify-content: center;
            background: rgba(0, 0, 0, 0.45);
            z-index: 999;
        }

        #notes-modal[data-open="true"] {
            display: flex;
        }

        #notes-modal-card {
            min-width: 360px;
            max-width: 80vw;
            border: 2px solid var(--fg);
            background: var(--bg);
            color: var(--fg);
            padding: 14px;
            display: flex;
            flex-direction: column;
            gap: 10px;
        }

        #notes-modal-title {
            color: var(--accent);
            font-size: ${result.fontSize}px;
        }

        #notes-modal-input {
            border-radius: 0;
            border: 2px solid var(--fg);
            background: transparent;
            color: var(--fg);
            padding: 8px;
            font-size: ${result.fontSize}px;
            outline: none;
        }

        #notes-modal-actions {
            display: flex;
            gap: 10px;
            justify-content: flex-end;
        }

        #notes-modal-actions button {
            border-radius: 0;
            border: 2px solid var(--fg);
            background: transparent;
            color: var(--fg);
            padding: 6px 10px;
            cursor: pointer;
        }

        #notes-modal-actions button:hover {
            border-color: var(--accent);
            color: var(--accent);
        }

        #notes-list {
            display: flex;
            flex-direction: column;
            gap: 6px;
            overflow-y: auto;
            flex: 1;
        }

        .notes-file {
            text-align: left;
            border-radius: 0;
            border: 2px solid transparent;
            background: transparent;
            color: var(--fg);
            padding: 6px 8px;
            cursor: pointer;
        }

        .notes-file[data-active="true"] {
            border-color: var(--accent);
            color: var(--accent);
        }

        .notes-file:hover {
            border-color: var(--fg);
        }

        #notes-empty {
            opacity: 0.7;
        }

        #notes-status {
            font-size: calc(${result.fontSize}px - 2px);
            opacity: 0.8;
        }

        #notes-status[data-state="error"] {
            color: var(--error);
        }

        #notes-main {
            display: flex;
            flex-direction: column;
            gap: 12px;
            padding: 16px;
            height: 100vh;
        }

        #notes-tabs {
            display: inline-flex;
            gap: 8px;
            border-bottom: 2px solid var(--fg);
            padding-bottom: 6px;
        }

        #notes-tabs button {
            border-radius: 0;
            border: 2px solid transparent;
            background: transparent;
            color: var(--fg);
            padding: 6px 12px;
            cursor: pointer;
        }

        #notes-tabs button[aria-selected="true"] {
            border-color: var(--accent);
            color: var(--accent);
        }

        #notes-panel {
            position: relative;
            flex: 1;
            min-height: 0;
        }

        #notes-editor-wrap,
        #notes-preview-wrap {
            position: absolute;
            inset: 0;
            display: none;
            min-height: 0;
        }

        #notes-editor-wrap[data-active="true"],
        #notes-preview-wrap[data-active="true"] {
            display: block;
        }

        #notes-editor {
            width: 100%;
            height: 100%;
            resize: none;
            border-radius: 0;
            border: 2px solid var(--fg);
            background: transparent;
            color: var(--fg);
            padding: 10px;
            font-size: ${result.fontSize}px;
            line-height: 1.4;
        }

        #notes-preview-wrap {
            overflow-y: auto;
            border-left: 2px solid var(--fg);
            padding-left: 16px;
        }

        .markdown-body h1,
        .markdown-body h2,
        .markdown-body h3,
        .markdown-body h4,
        .markdown-body h5,
        .markdown-body h6 {
            color: var(--accent);
        }

        .markdown-body a {
            text-decoration: none;
            color: var(--link);
        }

        .markdown-body a:hover {
            text-decoration: underline;
        }

        .markdown-body pre,
        .markdown-body code {
            color: var(--green);
        }

        .markdown-body pre {
            border: 0;
            border-left: 2px solid var(--fg);
            margin: 0;
            padding: 10px 10px 10px 20px;
            overflow-x: auto;
        }

        .markdown-body blockquote {
            border: 0;
            border-left: 2px solid var(--fg);
            margin: 0;
            padding: 1px 1px 1px 20px;
            color: var(--magenta);
        }

        .markdown-body details {
            opacity: 0.5;
            width: 100%;
            border-radius: 0;
            border-width: 2px;
            border-style: solid;
            padding: 5px;
            margin-top: 5px;
        }

        .markdown-body summary {
            cursor: pointer;
        }

        pre code.hljs {
            display: block;
            overflow-x: auto;
            background: transparent;
            color: var(--fg);
        }

        .hljs-comment,
        .hljs-quote {
            color: var(--blue-bright);
            font-style: italic;
        }

        .hljs-keyword,
        .hljs-selector-tag,
        .hljs-subst {
            color: var(--magenta);
            font-weight: bold;
        }

        .hljs-string,
        .hljs-title,
        .hljs-name,
        .hljs-type,
        .hljs-attribute,
        .hljs-symbol,
        .hljs-bullet,
        .hljs-addition,
        .hljs-built_in {
            color: var(--green);
        }

        .hljs-number,
        .hljs-literal,
        .hljs-variable,
        .hljs-template-variable {
            color: var(--accent);
        }

        .hljs-section,
        .hljs-meta,
        .hljs-function,
        .hljs-class,
        .hljs-title.class_ {
            color: var(--cyan);
        }

        .hljs-deletion,
        .hljs-regexp,
        .hljs-link {
            color: var(--red);
        }

        .hljs-punctuation,
        .hljs-tag {
            color: var(--fg);
        }

        @media (max-width: 980px) {
            #notes-app {
                grid-template-columns: 1fr;
                grid-template-rows: auto 1fr;
            }

            #notes-sidebar {
                border-right: none;
                border-bottom: 2px solid var(--fg);
            }

            #notes-preview-wrap {
                border-left: none;
                border-top: 2px solid var(--fg);
                padding-left: 0;
                padding-top: 16px;
            }
        }
    `;

    document.head.appendChild(style);
}

GetWindowStyle().then((result) => {
    applyWindowStyle(result);
});

GetParameters().then((params) => {
    if (params && params.path) {
        loadFile(params.path);
    }
});

refreshFiles();

if (elements.editor) {
    elements.editor.addEventListener('input', () => {
        setDirty(true);
        scheduleRender();
    });
}

elements.tabEditor.addEventListener('click', () => {
    setViewMode('editor');
});

elements.tabViewer.addEventListener('click', () => {
    setViewMode('viewer');
});

elements.newFile.addEventListener('click', () => {
    openNewFilePrompt();
});

elements.modalCancel.addEventListener('click', () => {
    closeNewFilePrompt();
});

elements.modalCreate.addEventListener('click', () => {
    createNewFile();
});

elements.save.addEventListener('click', () => {
    saveFile();
});

elements.refresh.addEventListener('click', () => {
    refreshFiles();
});

document.addEventListener('keydown', (event) => {
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 's') {
        event.preventDefault();
        saveFile();
    }

    if (event.key === 'Escape' && elements.modal.dataset.open === 'true') {
        event.preventDefault();
        closeNewFilePrompt();
    }
});

elements.modalInput.addEventListener('keydown', (event) => {
    if (event.key === 'Enter') {
        event.preventDefault();
        createNewFile();
    }
});

setViewMode('editor');
