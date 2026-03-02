import { GetWindowStyle, GetMarkdown, GetParameters, GetImage, SendIpc } from '../wailsjs/go/main/WApp';
import { BrowserOpenURL, WindowHide } from '../wailsjs/runtime/runtime';

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
            </div>
            <div id="notes-list" role="list"></div>
        </aside>
        <main id="notes-main">
            <div id="notes-tabs" role="tablist">
                <button id="notes-tab-viewer" type="button" role="tab" aria-selected="true">Viewer</button>
                <button id="notes-tab-editor" type="button" role="tab" aria-selected="false">Editor</button>
                <button id="notes-new" type="button">New</button>
                <button id="notes-delete" type="button" title="Delete current note">Delete</button>
                <div id="notes-status" role="status"></div>
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
    <div id="notes-delete-modal" data-open="false" aria-hidden="true">
        <div id="notes-delete-modal-card" role="dialog" aria-modal="true" aria-labelledby="notes-delete-modal-title">
            <div id="notes-delete-modal-title">Delete note</div>
            <div id="notes-delete-modal-body"></div>
            <div id="notes-delete-modal-actions">
                <button id="notes-delete-cancel" type="button">Cancel</button>
                <button id="notes-delete-confirm" type="button">Delete</button>
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
    delete: document.getElementById('notes-delete'),
    tabEditor: document.getElementById('notes-tab-editor'),
    tabViewer: document.getElementById('notes-tab-viewer'),
    editorWrap: document.getElementById('notes-editor-wrap'),
    previewWrap: document.getElementById('notes-preview-wrap'),
    modal: document.getElementById('notes-modal'),
    modalInput: document.getElementById('notes-modal-input'),
    modalCancel: document.getElementById('notes-modal-cancel'),
    modalCreate: document.getElementById('notes-modal-create'),
    deleteModal: document.getElementById('notes-delete-modal'),
    deleteModalBody: document.getElementById('notes-delete-modal-body'),
    deleteCancel: document.getElementById('notes-delete-cancel'),
    deleteConfirm: document.getElementById('notes-delete-confirm')
};

const state = {
    files: [],
    currentFile: '',
    dirty: false,
    renderTimer: null,
    autosaveTimer: null,
    viewMode: 'viewer',
    renamingFile: null,
    deletingFile: null
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

    const rxWailsUrl = /^(wails:\/\/wails\/|http:\/\/localhost:[0-9]+\/|wails:\/\/wails.localhost:[0-9]+\/)/;

    elements.preview.querySelectorAll('img').forEach((img) => {
        if (img.src.match(rxWailsUrl)) {
            const path = img.src.replace(rxWailsUrl, '');
            GetImage(path).then((image) => {
                if (image.match(/^error: /)) {
                    console.log(image);
                } else {
                    img.src = image;
                }
            });
        }
    });

    let rxBookmark = /^(wails:\/\/wails\/|http:\/\/localhost:[0-9]+\/|wails:\/\/wails.localhost:[0-9]+\/)#/;

    elements.preview.querySelectorAll('a').forEach(a => {
        if (!a.href.match(rxWailsUrl)) {
            a.addEventListener('click', (e) => {
                e.preventDefault();
                BrowserOpenURL(a.href);
            });
        }

        if (!a.href.match(rxBookmark)) {
            /*let id = a.href.replace(rxBookmark, '');
            console.log(id);
            //a.href = "#"+id;
            a.addEventListener("click", () => {
                document.getElementById(id).scrollIntoView();
            });*/
        }
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

function scheduleAutoSave() {
    if (state.autosaveTimer) {
        clearTimeout(state.autosaveTimer);
    }
    state.autosaveTimer = setTimeout(() => {
        state.autosaveTimer = null;
        saveFile();
    }, 1000);
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
        item.addEventListener('dblclick', (e) => {
            e.preventDefault();
            openRenamePrompt(file);
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

function openDeletePrompt(file) {
    state.deletingFile = file;
    const fileName = file.split('/').pop();
    elements.deleteModalBody.textContent = `Are you sure you want to delete "${fileName}"?`;
    elements.deleteModal.dataset.open = 'true';
    elements.deleteModal.setAttribute('aria-hidden', 'false');
    setTimeout(() => {
        elements.deleteConfirm.focus();
    }, 0);
}

function closeDeletePrompt() {
    elements.deleteModal.dataset.open = 'false';
    elements.deleteModal.setAttribute('aria-hidden', 'true');
    state.deletingFile = null;
}

async function confirmDelete() {
    if (!state.deletingFile) {
        setStatus('Select a note to delete.', true);
        return;
    }

    const deleteFn = getWailsFunction('DeleteFile');
    if (!deleteFn) {
        setStatus('DeleteFile is not available.', true);
        return;
    }

    const fileToDelete = state.deletingFile;
    const fileName = fileToDelete.split('/').pop();

    try {
        await deleteFn(fileToDelete);
        if (state.currentFile === fileToDelete) {
            state.currentFile = '';
            elements.editor.value = '';
            renderMarkdown();
            setDirty(false);
        }
        closeDeletePrompt();
        await refreshFiles();
        setStatus(`Deleted ${fileName}.`, false);
    } catch (err) {
        setStatus(`Failed to delete ${fileName}.`, true);
        console.error(err);
    }
}

function openNewFilePrompt() {
    state.renamingFile = null;
    elements.modal.dataset.open = 'true';
    elements.modal.setAttribute('aria-hidden', 'false');
    elements.modalInput.value = '';
    elements.modal.querySelector('#notes-modal-title').textContent = 'New note name';
    elements.modalCreate.textContent = 'Create';
    setTimeout(() => {
        elements.modalInput.focus();
    }, 0);
}

function openRenamePrompt(file) {
    state.renamingFile = file;
    const fileName = file.split('/').pop().replace(/\.md$/, '');
    elements.modal.dataset.open = 'true';
    elements.modal.setAttribute('aria-hidden', 'false');
    elements.modalInput.value = fileName;
    elements.modal.querySelector('#notes-modal-title').textContent = 'Rename note';
    elements.modalCreate.textContent = 'Rename';
    setTimeout(() => {
        elements.modalInput.focus();
        elements.modalInput.select();
    }, 0);
}

function closeNewFilePrompt() {
    elements.modal.dataset.open = 'false';
    elements.modal.setAttribute('aria-hidden', 'true');
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
    let fileName = normalizeNoteName(elements.modalInput.value);
    if (fileName === '') {
        setStatus('File name cannot be empty.', true);
        return;
    }

    // Handle rename operation
    if (state.renamingFile) {
        const newPath = state.renamingFile.split('/').slice(0, -1).join('/') + '/' + fileName;
        const renameFn = getWailsFunction('RenameFile');
        if (!renameFn) {
            setStatus('RenameFile is not available.', true);
            return;
        }

        try {
            await renameFn(state.renamingFile, newPath);
            await refreshFiles();
            if (state.currentFile === state.renamingFile) {
                await loadFile(newPath);
            }
            closeNewFilePrompt();
            setStatus(`Renamed to ${newPath}.`, false);
        } catch (err) {
            setStatus(`Failed to rename file.`, true);
            console.error(err);
        }
        return;
    }

    // Handle new file creation
    fileName = "$NOTES/" + fileName;

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

    const notesFileSize = result.fontSize * 2;
    const notesStatusFontSize = result.fontSize - 2;
    const notesTitleFontSize = result.fontSize + 4;

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

        ::-webkit-scrollbar {
            width: 5px;
            height: 5px;
            background-color: var(--bg);
        }

        ::-webkit-scrollbar-track {
            background-color: var(--bg);
        }

        ::-webkit-scrollbar-thumb {
            background-color: var(--fg);
            border-radius: 4px;
        }

        ::-webkit-scrollbar-thumb:hover {
            background-color: var(--accent);
        }

        #notes-app {
            display: grid;
            grid-template-columns: 20% 1fr;
            height: 100vh;
            overflow: hidden;
            color: var(--fg);
            background: var(--bg);
        }

        #notes-sidebar {
            display: flex;
            flex-direction: column;
            /* border-right: 2px solid var(--fg); */
            padding: 16px;
            gap: 12px;
            min-height: 0;
            overflow: hidden;
        }

        #notes-sidebar-header {
            display: flex;
            flex-direction: column;
            gap: 12px;
        }

        #notes-title {
            font-size: ${notesTitleFontSize}px;
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

        #notes-delete-modal {
            position: fixed;
            inset: 0;
            display: none;
            align-items: center;
            justify-content: center;
            background: rgba(0, 0, 0, 0.45);
            z-index: 999;
        }

        #notes-delete-modal[data-open="true"] {
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

        #notes-delete-modal-card {
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

        #notes-delete-modal-title {
            color: var(--accent);
            font-size: ${result.fontSize}px;
        }

        #notes-delete-modal-body {
            opacity: 0.9;
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

        #notes-new:hover {
            border-color: var(--green) !important;
            color: var(--green) !important;
        }
        
        #notes-modal-create:hover {
            border-color: var(--green) !important;
            color: var(--green) !important;
        }

        #notes-modal-actions {
            display: flex;
            gap: 10px;
            justify-content: flex-end;
        }

        #notes-delete-modal-actions {
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

        #notes-delete-modal-actions button {
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

        #notes-delete-modal-actions button:hover {
            border-color: var(--accent);
            color: var(--accent);
        }

        #notes-delete-confirm {
            border-color: var(--error);
            color: var(--error);
        }

        #notes-delete-confirm:hover {
            border-color: var(--error) !important;
            color: var(--error) !important;
        }

        #notes-list {
            display: flex;
            flex-direction: column;
            gap: 6px;
            overflow-y: auto;
            overflow-x: hidden;
            flex: 1;
        }

        .notes-file {
            min-height: ${notesFileSize}px;
            text-align: left;
            border-radius: 0;
            border: 2px solid transparent;
            background: transparent;
            color: var(--fg);
            padding: 6px 8px;
            cursor: pointer;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
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
            font-size: ${notesStatusFontSize}px;
            opacity: 0.8;
            align-self: center;
            margin-left: auto;
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
            align-items: center;
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

        #notes-tabs button:hover {
            border-color: var(--fg);
        }

        #notes-new {
            margin-left: auto;
        }

        #notes-new:hover {
            border-color: var(--fg);
            color: var(--fg);
        }

        #notes-delete {
            color: var(--error);
        }

        #notes-delete:hover {
            border-color: var(--error) !important;
            color: var(--error);
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
window.refreshFiles = refreshFiles;

if (elements.editor) {
    elements.editor.addEventListener('input', () => {
        setDirty(true);
        scheduleRender();
        scheduleAutoSave();
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

elements.delete.addEventListener('click', () => {
    if (!state.currentFile) {
        setStatus('Select a note to delete.', true);
        return;
    }
    openDeletePrompt(state.currentFile);
});

elements.deleteCancel.addEventListener('click', () => {
    closeDeletePrompt();
});

elements.deleteConfirm.addEventListener('click', () => {
    confirmDelete();
});

document.addEventListener('keydown', (event) => {
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 's') {
        event.preventDefault();
        saveFile();
    }

    if (event.key === 'F2' && state.currentFile && elements.modal.dataset.open === 'false') {
        event.preventDefault();
        openRenamePrompt(state.currentFile);
    }

    if (event.key === 'Tab') {
        event.preventDefault();
        SendIpc("focus", {});
    }

    if (event.key === 'Escape' && elements.modal.dataset.open === 'true') {
        event.preventDefault();
        closeNewFilePrompt();
    } else if (event.key === 'Escape' && elements.deleteModal.dataset.open === 'true') {
        event.preventDefault();
        closeDeletePrompt();
    } else if (event.key === 'Escape') {
        event.preventDefault();
        SendIpc('focus', {})
        WindowHide();
    }
});

elements.modalInput.addEventListener('keydown', (event) => {
    if (event.key === 'Enter') {
        event.preventDefault();
        createNewFile();
    }
});

setViewMode('viewer');
