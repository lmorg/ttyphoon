import {
    GetWindowStyle, GetMarkdown,
    ListFiles, SaveFile, SaveBinaryFile, DeleteFile, RenameFile,
    RunNote, StopNote, SendIpc, SendToTerminal,
    GetLanguageDescriptions, GetAllLanguageDescriptions, TerminalCopyImageDataURL,
    SaveImageDialog, WindowPrint, GetClipboardData, SwaggerRequest
} from '../wailsjs/go/main/WApp';
import { EventsOn, ClipboardSetText } from '../wailsjs/runtime/runtime';

import { showLocalMenu } from './popup_menu';

import { marked } from "marked";
import hljs from "highlight.js/lib/common";

import { configureMarked, processMarkdownContainer } from './markdown-utils.js';
import { getScrollbarStyles, getMarkdownContentStyles, getHighlightJsTheme, getCheckboxStyles, getMarkdownBaseTextSizeStyles, getSwaggerUIStyles } from './style-utils.js';
import { 
    isJsonFile, hasSwaggerKey, parseSwaggerSpec, generateRequestBuilderHTML, generateResponseHTML,
    extractPaths, generateEndpointListHTML, buildRequestUrl, generateLiveResponseHTML, escapeInfoText
} from './swagger-utils.js';
import { renderJsonViewer } from './json-viewer.js';

const CONTEXT_ICON_COPY = 0xf0c5;
const CONTEXT_ICON_PASTE = 0xf0ea;
const CONTEXT_ICON_FIND = 0xf002;
const CONTEXT_ICON_PRINT = 0xf02f;
const CONTEXT_ICON_CHECKBOX = 0xf14a;
const CONTEXT_ICON_CODE = 0xf121;
const CONTEXT_ICON_DELETE = 0xf2ed;

const app = document.getElementById('notes-pane') || document.getElementById('app') || (() => {
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
                <button id="notes-tab-viewer" type="button" class="tab" role="tab" aria-selected="true">View</button>
                <button id="notes-tab-editor" type="button" class="tab" role="tab" aria-selected="false">Edit</button>
                <button id="notes-tab-jupyter" type="button" class="tab" role="tab" aria-selected="false">Run</button>
                <button id="notes-tab-swagger-view" type="button" class="tab" role="tab" aria-selected="false" style="display: none;" data-swagger="true">View</button>
                <button id="notes-tab-swagger-edit" type="button" class="tab" role="tab" aria-selected="false" style="display: none;" data-swagger="true">Edit</button>
                <button id="notes-tab-swagger-run" type="button" class="tab" role="tab" aria-selected="false" style="display: none;" data-swagger="true">Run</button>
                <div id="notes-toolbar" class="notes-toolbar">
                    <button id="notes-new" type="button" class="notes-toolbar-btn" title="New" aria-label="New note">&#xe494;</button>
                    <button id="notes-rename" type="button" class="notes-toolbar-btn" title="Rename" aria-label="Rename current note">&#xf044;</button>
                    <button id="notes-delete" type="button" class="notes-toolbar-btn" title="Delete" aria-label="Delete current note">&#xf2ed;</button>
                    <button id="notes-find" type="button" class="notes-toolbar-btn" title="Find" aria-label="Find">&#xf002;</button>
                </div>
            </div>
            <div id="notes-panel">
                <div id="notes-editor-wrap" role="tabpanel">
                    <textarea id="notes-editor" spellcheck="false"></textarea>
                </div>
                <div id="notes-preview-wrap" class="markdown-body" role="tabpanel">
                    <div id="notes-preview"></div>
                </div>
                <div id="notes-jupyter-wrap" class="markdown-body" role="tabpanel">
                    <div id="notes-jupyter"></div>
                </div>
                <div id="notes-swagger-edit-wrap" role="tabpanel" style="display: none;">
                    <textarea id="notes-swagger-editor" spellcheck="false"></textarea>
                </div>
                <div id="notes-swagger-view-wrap" role="tabpanel" style="display: none;">
                    <div id="notes-swagger-view" class="json-viewer"></div>
                </div>
                <div id="notes-swagger-run-wrap" class="swagger-ui" role="tabpanel" style="display: none;">
                    <div id="notes-swagger-layout" class="swagger-layout">
                        <div id="notes-swagger-info" class="swagger-info markdown-body"></div>
                        <aside id="notes-swagger-endpoints" class="swagger-endpoints-pane"></aside>
                        <section id="notes-swagger-main" class="swagger-main-pane">
                            <div id="notes-swagger-request-builder"></div>
                            <div id="notes-swagger-response"></div>
                        </section>
                    </div>
                </div>
                <div id="notes-ai-panel" class="notes-ai-panel" data-collapsed="true">
                    <div class="notes-ai-header">
                        <button id="notes-ai-toggle" type="button" class="notes-ai-toggle" title="Toggle AI panel">AI ▾</button>
                        <button id="notes-ai-clear" type="button" class="notes-ai-clear" title="Clear AI output">Clear</button>
                    </div>
                    <div id="notes-ai-output" class="notes-ai-output"></div>
                </div>
                <button id="notes-ai-restore" type="button" class="notes-ai-restore" title="Show AI panel">AI</button>
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
    <div id="notes-find-bar" data-open="false" aria-hidden="true">
        <input id="notes-find-input" type="text" placeholder="Find..." autocomplete="off" />
        <span id="notes-find-counter"></span>
        <button id="notes-find-prev" type="button" title="Previous match">↑</button>
        <button id="notes-find-next" type="button" title="Next match">↓</button>
        <button id="notes-find-close" type="button" title="Close find">✕</button>
    </div>
`;

const elements = {
    title: document.getElementById('notes-title'),
    list: document.getElementById('notes-list'),
    editor: document.getElementById('notes-editor'),
    preview: document.getElementById('notes-preview'),
    jupyter: document.getElementById('notes-jupyter'),
    status: document.getElementById('notes-status'),
    newFile: document.getElementById('notes-new'),
    rename: document.getElementById('notes-rename'),
    delete: document.getElementById('notes-delete'),
    find: document.getElementById('notes-find'),
    tabEditor: document.getElementById('notes-tab-editor'),
    tabViewer: document.getElementById('notes-tab-viewer'),
    tabJupyter: document.getElementById('notes-tab-jupyter'),
    tabSwaggerView: document.getElementById('notes-tab-swagger-view'),
    tabSwaggerEdit: document.getElementById('notes-tab-swagger-edit'),
    tabSwaggerRun: document.getElementById('notes-tab-swagger-run'),
    editorWrap: document.getElementById('notes-editor-wrap'),
    previewWrap: document.getElementById('notes-preview-wrap'),
    jupyterWrap: document.getElementById('notes-jupyter-wrap'),
    swaggerViewWrap: document.getElementById('notes-swagger-view-wrap'),
    swaggerEditWrap: document.getElementById('notes-swagger-edit-wrap'),
    swaggerRunWrap: document.getElementById('notes-swagger-run-wrap'),
    swaggerView: document.getElementById('notes-swagger-view'),
    swaggerEndpoints: document.getElementById('notes-swagger-endpoints'),
    swaggerEditor: document.getElementById('notes-swagger-editor'),
    swaggerRequestBuilder: document.getElementById('notes-swagger-request-builder'),
    swaggerResponse: document.getElementById('notes-swagger-response'),
    modal: document.getElementById('notes-modal'),
    modalInput: document.getElementById('notes-modal-input'),
    modalCancel: document.getElementById('notes-modal-cancel'),
    modalCreate: document.getElementById('notes-modal-create'),
    deleteModal: document.getElementById('notes-delete-modal'),
    deleteModalBody: document.getElementById('notes-delete-modal-body'),
    deleteCancel: document.getElementById('notes-delete-cancel'),
    deleteConfirm: document.getElementById('notes-delete-confirm'),
    findBar: document.getElementById('notes-find-bar'),
    findInput: document.getElementById('notes-find-input'),
    findCounter: document.getElementById('notes-find-counter'),
    findPrev: document.getElementById('notes-find-prev'),
    findNext: document.getElementById('notes-find-next'),
    findClose: document.getElementById('notes-find-close'),
    aiPanel: document.getElementById('notes-ai-panel'),
    aiToggle: document.getElementById('notes-ai-toggle'),
    aiClear: document.getElementById('notes-ai-clear'),
    aiOutput: document.getElementById('notes-ai-output'),
    aiRestore: document.getElementById('notes-ai-restore')
};

const state = {
    files: [],
    currentFile: '',
    currentFileType: 'markdown',  // 'markdown' or 'json'
    dirty: false,
    renderTimer: null,
    autosaveTimer: null,
    viewMode: 'viewer',
    renamingFile: null,
    deletingFile: null,
    findMatches: [],
    findCurrentIndex: -1,
    findQuery: '',
    expandedCategories: {
        '$GLOBAL': true,
        '$NOTES': true,
        '$PROJ': true,
        '$HISTORY': false,
    },
    jupyterCodeBlocks: {},
    jupyterBlockCounter: 0,
    swaggerSpec: null,
    swaggerRunAvailable: false,
    swaggerSelectedEndpoint: null,
    swaggerEndpointFilter: ''
};

configureMarked();

function setStatus(message, isError) {
    elements.status.textContent = message || '';
    elements.status.dataset.state = isError ? 'error' : 'ok';
}

function notifyTerminal(message, level = 'info') {
    if (!message) {
        return;
    }

    SendIpc('terminal-notify', {
        level,
        message,
    }).catch(() => {});
}

function openStickyProgress(id, message) {
    SendIpc('terminal-sticky-create', {
        id: String(id),
        message,
        level: 'info',
    }).catch(() => {});
}

function updateStickyProgress(id, message) {
    SendIpc('terminal-sticky-update', {
        id: String(id),
        message,
    }).catch(() => {});
}

function closeStickyProgress(id, finalMessage, level = 'info') {
    SendIpc('terminal-sticky-close', {
        id: String(id),
    }).catch(() => {});
    if (finalMessage) {
        notifyTerminal(finalMessage, level);
    }
}

function yieldToUI() {
    return new Promise((resolve) => {
        setTimeout(resolve, 0);
    });
}

function renderMarkdown() {
    const markdown = elements.editor.value || '';
    elements.preview.innerHTML = marked.parse(markdown);

    // Apply common markdown processing
    processMarkdownContainer(elements.preview);

    // Enable context menus on images
    enableImageContextMenus(elements.preview);

    // Keep checkboxes readonly in viewer mode
    setupInteractiveCheckboxes(elements.preview, false);

    // Re-apply find highlights if find bar is open and in viewer mode
    if (elements.findBar.dataset.open === 'true' && state.findQuery && state.viewMode === 'viewer') {
        setTimeout(() => {
            performFind();
        }, 0);
    }
}

function setupInteractiveCheckboxes(container, isEditable) {
    const checkboxes = container.querySelectorAll('input[type="checkbox"]');
    
    checkboxes.forEach((checkbox, index) => {
        if (!isEditable) {
            checkbox.setAttribute('disabled', 'disabled');
            return;
        }

        checkbox.removeAttribute('disabled');
        checkbox.addEventListener('change', (e) => {
            toggleCheckboxInMarkdown(index, e.target.checked);
        });
    });
}

function toggleCheckboxInMarkdown(checkboxIndex, isChecked) {
    const lines = elements.editor.value.split('\n');
    let currentCheckboxIndex = 0;
    let modified = false;

    for (let i = 0; i < lines.length; i++) {
        const checkboxMatch = lines[i].match(/^(\s*[-*+]?\s*)\[( |x|X)\](.*)$/);
        if (!checkboxMatch) {
            continue;
        }

        if (currentCheckboxIndex === checkboxIndex) {
            const newState = isChecked ? 'x' : ' ';
            lines[i] = `${checkboxMatch[1]}[${newState}]${checkboxMatch[3]}`;
            modified = true;
            break;
        }
        currentCheckboxIndex++;
    }

    if (modified) {
        elements.editor.value = lines.join('\n');
        saveFile();
        // Keep viewer in sync when changes are made from jupyter mode
        if (state.viewMode === 'jupyter') {
            renderMarkdown();
        }
        // Don't re-render jupyter here to avoid resetting checkbox focus
    }
}

function updateMarkdownCodeBlock(blockIndex, newContent) {
    const markdown = elements.editor.value;
    const rxCodeBlock = /```[^\n]*\n[\s\S]*?\n```/g;
    let match;
    let index = 0;
    let lastIndex = 0;
    let updated = false;
    let result = '';

    while ((match = rxCodeBlock.exec(markdown)) !== null) {
        if (index === blockIndex) {
            const block = match[0];
            const headerEnd = block.indexOf('\n');
            const footerStart = block.lastIndexOf('\n```');
            if (headerEnd === -1 || footerStart === -1) {
                return false;
            }

            const header = block.slice(0, headerEnd + 1);
            const footer = block.slice(footerStart);
            const trimmedContent = newContent.replace(/[\r\n]+$/, '');
            const updatedBlock = header + trimmedContent + footer;

            result += markdown.slice(lastIndex, match.index) + updatedBlock;
            lastIndex = match.index + match[0].length;
            updated = true;
            break;
        }
        index++;
    }

    if (!updated) {
        return false;
    }

    result += markdown.slice(lastIndex);
    elements.editor.value = result;
    return true;
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

function emitCurrentFileName() {
    const fileName = state.currentFile ? state.currentFile.split('/').pop() : '';
    app.dataset.currentFileName = fileName;
    window.dispatchEvent(new CustomEvent('notes-current-file', {
        detail: { fileName }
    }));
}

function setViewMode(mode) {
    // Determine the mode based on current file type
    if (state.currentFileType === 'json') {
        if (mode === 'swagger-view' || mode === 'swagger-edit' || (mode === 'swagger-run' && state.swaggerRunAvailable)) {
            state.viewMode = mode;
        } else {
            state.viewMode = 'swagger-view';
        }
    } else {
        state.viewMode = mode === 'viewer' ? 'viewer' : (mode === 'jupyter' ? 'jupyter' : 'editor');
    }
    
    // Share active notes mode with ttyphoon.js so cross-pane focus behavior can follow mode intent.
    app.dataset.viewMode = state.viewMode;
    
    // Markdown tabs
    const isEditor = state.viewMode === 'editor';
    const isJupyter = state.viewMode === 'jupyter';
    const isViewer = state.viewMode === 'viewer';
    
    elements.tabEditor.setAttribute('aria-selected', isEditor ? 'true' : 'false');
    elements.tabViewer.setAttribute('aria-selected', isViewer ? 'true' : 'false');
    elements.tabJupyter.setAttribute('aria-selected', isJupyter ? 'true' : 'false');
    
    elements.editorWrap.dataset.active = isEditor ? 'true' : 'false';
    elements.previewWrap.dataset.active = isViewer ? 'true' : 'false';
    elements.jupyterWrap.dataset.active = isJupyter ? 'true' : 'false';
    
    // Swagger tabs
    const isSwaggerView = state.viewMode === 'swagger-view';
    const isSwaggerEdit = state.viewMode === 'swagger-edit';
    const isSwaggerRun = state.viewMode === 'swagger-run';
    
    elements.tabSwaggerView.setAttribute('aria-selected', isSwaggerView ? 'true' : 'false');
    elements.tabSwaggerEdit.setAttribute('aria-selected', isSwaggerEdit ? 'true' : 'false');
    elements.tabSwaggerRun.setAttribute('aria-selected', isSwaggerRun ? 'true' : 'false');
    
    elements.swaggerViewWrap.dataset.active = isSwaggerView ? 'true' : 'false';
    elements.swaggerEditWrap.dataset.active = isSwaggerEdit ? 'true' : 'false';
    elements.swaggerRunWrap.dataset.active = isSwaggerRun ? 'true' : 'false';
    
    // Re-perform find if find bar is open
    if (elements.findBar.dataset.open === 'true' && state.findQuery) {
        performFind();
    }
}

function renderJupyterView() {
    // Reset jupyter state for the new render
    state.jupyterCodeBlocks = {};
    state.jupyterBlockCounter = 0;
    
    const markdown = elements.editor.value || '';
    elements.jupyter.innerHTML = marked.parse(markdown);
    
    // Apply common markdown processing
    processMarkdownContainer(elements.jupyter);

    // Enable context menus on images
    enableImageContextMenus(elements.jupyter);
    
    // Enable checkbox editing and save behavior in jupyter mode
    setupInteractiveCheckboxes(elements.jupyter, true);
    convertToJupyterCodeBlocks();
    
    // Re-apply find highlights if find bar is open and in jupyter mode
    if (elements.findBar.dataset.open === 'true' && state.findQuery && state.viewMode === 'jupyter') {
        setTimeout(() => {
            performFind();
        }, 0);
    }
}

function convertToJupyterCodeBlocks() {
    const codeBlocks = elements.jupyter.querySelectorAll('pre');
    
    codeBlocks.forEach((pre) => {
        const code = pre.querySelector('code');
        if (!code) return;
        
        const langClass = Array.from(code.classList).find(cls => cls.startsWith('language-'));
        const language = langClass ? langClass.replace('language-', '') : '';
        const blockId = `jupyter-block-${state.jupyterBlockCounter++}`;
        const content = code.textContent;
        
        state.jupyterCodeBlocks[blockId] = {
            language,
            runtime: language,
            originalContent: content,
            currentContent: content
        };
        
        const wrapper = document.createElement('div');
        wrapper.className = 'jupyter-code-block';
        wrapper.dataset.blockId = blockId;
        
        const toolbar = document.createElement('div');
        toolbar.className = 'jupyter-toolbar';
        
        const runNotesBtn = document.createElement('button');
        runNotesBtn.type = 'button';
        runNotesBtn.className = 'jupyter-btn jupyter-run-notes';
        runNotesBtn.textContent = 'Run';
        runNotesBtn.addEventListener('click', () => runCodeBlockInNotes(blockId));
        
        const stopNotesBtn = document.createElement('button');
        stopNotesBtn.type = 'button';
        stopNotesBtn.className = 'jupyter-btn jupyter-stop-notes';
        stopNotesBtn.textContent = 'Stop';
        stopNotesBtn.style.display = 'none'; // Initially hidden
        stopNotesBtn.addEventListener('click', () => stopCodeBlockInNotes(blockId));
        
        const runTerminalBtn = document.createElement('button');
        runTerminalBtn.type = 'button';
        runTerminalBtn.className = 'jupyter-btn jupyter-run-terminal';
        runTerminalBtn.textContent = 'Send to terminal';
        runTerminalBtn.addEventListener('click', () => runCodeBlockInTerminal(blockId));
        
        const runtimeDropdown = document.createElement('select');
        runtimeDropdown.className = 'jupyter-runtime-dropdown';
        runtimeDropdown.title = 'Select runtime';
        
        // Populate dropdown immediately
        (async () => {
            try {
                const hasLanguage = Boolean(language);
                let descriptions = [];
                let defaultSelection = '';

                if (hasLanguage) {
                    const matches = await GetLanguageDescriptions(language);
                    if (matches && matches.length > 0) {
                        // Markdown language exists in YAML: only show those options
                        descriptions = matches;
                        defaultSelection = matches[0];
                    } else {
                        // Markdown language not in YAML: show all options, default to markdown language
                        descriptions = await GetAllLanguageDescriptions();
                        descriptions.sort((a, b) => a.localeCompare(b));
                        defaultSelection = language;
                    }
                } else {
                    // No markdown language: autodetect using highlight.js
                    let detectedLanguage = '';
                    if (content) {
                        try {
                            const result = hljs.highlightAuto(content);
                            if (result && result.language) {
                                detectedLanguage = result.language;
                            }
                        } catch (err) {
                            console.warn('Highlight.js autodetection failed:', err);
                        }
                    }

                    descriptions = await GetAllLanguageDescriptions();
                    descriptions.sort((a, b) => a.localeCompare(b));

                    if (detectedLanguage) {
                        const detectedMatches = await GetLanguageDescriptions(detectedLanguage);
                        defaultSelection = detectedMatches && detectedMatches.length > 0
                            ? detectedMatches[0]
                            : 'language unknown';
                    } else {
                        defaultSelection = 'language unknown';
                    }
                }

                // Populate dropdown
                runtimeDropdown.innerHTML = '';

                // If we have a custom default that's not in the list, add it first
                if (defaultSelection && !descriptions.includes(defaultSelection)) {
                    const option = document.createElement('option');
                    option.value = defaultSelection;
                    option.textContent = defaultSelection;
                    runtimeDropdown.appendChild(option);
                }

                // Add all available descriptions
                descriptions.forEach((desc) => {
                    const option = document.createElement('option');
                    option.value = desc;
                    option.textContent = desc;
                    if (desc === defaultSelection) {
                        option.selected = true;
                    }
                    runtimeDropdown.appendChild(option);
                });

                // Set runtime state
                state.jupyterCodeBlocks[blockId].runtime = defaultSelection
                    || (descriptions.length > 0 ? descriptions[0] : language || 'language unknown');

            } catch (err) {
                console.error('Error fetching language descriptions:', err);
                const option = document.createElement('option');
                option.value = language || 'language unknown';
                option.textContent = language || 'language unknown';
                runtimeDropdown.appendChild(option);
                state.jupyterCodeBlocks[blockId].runtime = language || 'language unknown';
            }
        })();
        
        runtimeDropdown.addEventListener('change', () => {
            state.jupyterCodeBlocks[blockId].runtime = runtimeDropdown.value;
        });
        
        toolbar.appendChild(runNotesBtn);
        toolbar.appendChild(stopNotesBtn);
        toolbar.appendChild(runTerminalBtn);
        toolbar.appendChild(runtimeDropdown);
        
        const editableCode = document.createElement('textarea');
        editableCode.className = 'jupyter-code-editable';
        editableCode.dataset.language = language;
        editableCode.value = content;
        editableCode.spellcheck = false;

        const codeEditor = document.createElement('div');
        codeEditor.className = 'jupyter-code-editor';

        const lineNumbers = document.createElement('div');
        lineNumbers.className = 'jupyter-line-numbers';

        const renderLineNumbers = () => {
            const lineCount = Math.max(1, editableCode.value.split('\n').length);
            lineNumbers.textContent = Array.from({ length: lineCount }, (_, i) => i + 1).join('\n');
        };
        
        // Auto-resize textarea to fit content
        const autoResize = () => {
            editableCode.style.height = 'auto';
            editableCode.style.height = editableCode.scrollHeight + 'px';
        };
        editableCode.addEventListener('input', () => {
            autoResize();
            renderLineNumbers();
            const blockState = state.jupyterCodeBlocks[blockId];
            if (!blockState) {
                return;
            }
            blockState.currentContent = editableCode.value;

            const blockIndex = parseInt(blockId.replace('jupyter-block-', ''), 10);
            if (Number.isNaN(blockIndex)) {
                return;
            }

            const updated = updateMarkdownCodeBlock(blockIndex, blockState.currentContent);
            if (!updated) {
                return;
            }

            setDirty(true);
            scheduleRender();
            scheduleAutoSave();
        });
        editableCode.addEventListener('scroll', () => {
            lineNumbers.scrollTop = editableCode.scrollTop;
        });
        // Set initial height
        setTimeout(() => {
            autoResize();
            renderLineNumbers();
        }, 0);
        
        const outputWrapper = document.createElement('div');
        outputWrapper.className = 'jupyter-output-wrapper';
        outputWrapper.style.display = 'none'; // Initially hidden
        
        const outputToggle = document.createElement('button');
        outputToggle.type = 'button';
        outputToggle.className = 'jupyter-output-toggle';
        outputToggle.textContent = 'Output ▾';
        outputToggle.dataset.collapsed = 'false';
        
        const outputBlock = document.createElement('pre');
        outputBlock.className = 'jupyter-output';
        outputBlock.textContent = '';
        outputBlock.style.display = 'block';
        
        outputToggle.addEventListener('click', () => {
            const isCollapsed = outputBlock.style.display === 'none';
            outputBlock.style.display = isCollapsed ? 'block' : 'none';
            outputToggle.textContent = isCollapsed ? 'Output ▾' : 'Output ▸';
            outputToggle.dataset.collapsed = isCollapsed ? 'false' : 'true';
        });
        
        outputWrapper.appendChild(outputToggle);
        outputWrapper.appendChild(outputBlock);
        
        pre.replaceWith(wrapper);
        wrapper.appendChild(toolbar);
        codeEditor.appendChild(lineNumbers);
        codeEditor.appendChild(editableCode);
        wrapper.appendChild(codeEditor);
        wrapper.appendChild(outputWrapper);
    });
}

async function runCodeBlockInNotes(blockId) {
    const block = state.jupyterCodeBlocks[blockId];
    if (!block) return;
    
    const editableElement = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-code-editable`);
    if (editableElement) {
        block.currentContent = editableElement.value;
    }
    
    // Toggle Run/Stop buttons
    const runBtn = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-run-notes`);
    const stopBtn = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-stop-notes`);
    if (runBtn) runBtn.style.display = 'none';
    if (stopBtn) stopBtn.style.display = 'inline-block';
    
    // Show the output wrapper when running
    const outputWrapper = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-output-wrapper`);
    if (outputWrapper) {
        outputWrapper.style.display = 'block';
    }
    
    // Clear previous output
    const outputBlock = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-output`);
    if (outputBlock) {
        outputBlock.textContent = '';
    }
    
    try {
        await RunNote(blockId, block.currentContent, block.runtime);
    } catch (err) {
        console.error('Error running code:', err);
        const outputBlock = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-output`);
        if (outputBlock) {
            outputBlock.textContent = `Error: ${err.message}`;
        }
        // Reset buttons on error
        if (runBtn) runBtn.style.display = 'inline-block';
        if (stopBtn) stopBtn.style.display = 'none';
    }
}

async function stopCodeBlockInNotes(blockId) {
    try {
        await StopNote(blockId);
    } catch (err) {
        console.error('Error stopping code:', err);
    }
    
    // Toggle buttons back
    const runBtn = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-run-notes`);
    const stopBtn = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-stop-notes`);
    if (runBtn) runBtn.style.display = 'inline-block';
    if (stopBtn) stopBtn.style.display = 'none';
}

async function runCodeBlockInTerminal(blockId) {
    const block = state.jupyterCodeBlocks[blockId];
    if (!block) return;
    
    const editableElement = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-code-editable`);
    if (editableElement) {
        block.currentContent = editableElement.value;
    }
    
        try {
            await SendToTerminal(block.currentContent);
        } catch (err) {
            console.error('Error sending to terminal:', err);
        }
}

async function refreshFiles() {
    try {
        const files = await ListFiles();
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

    // Group files by category
    const categories = {
        '$GLOBAL': [],
        '$NOTES': [],
        '$PROJ': []
    };

    state.files.forEach((file) => {
        if (file.startsWith('$GLOBAL/')) {
            categories['$GLOBAL'].push(file);
        } else if (file.startsWith('$NOTES/')) {
            categories['$NOTES'].push(file);
        } else if (file.startsWith('$PROJ/')) {
            categories['$PROJ'].push(file);
        }
    });

    // Render each category
    Object.keys(categories).forEach((category) => {
        const files = categories[category];
        if (files.length === 0) {
            return;
        }

        // Create category header
        const categoryHeader = document.createElement('div');
        categoryHeader.className = 'notes-category-header';
        categoryHeader.dataset.category = category;
        categoryHeader.dataset.expanded = state.expandedCategories[category] ? 'true' : 'false';
        
        const arrow = document.createElement('span');
        arrow.className = 'notes-category-arrow';
        arrow.textContent = state.expandedCategories[category] ? '▼' : '▶';
        
        const label = document.createElement('span');
        label.textContent = category;
        
        categoryHeader.appendChild(arrow);
        categoryHeader.appendChild(label);
        
        categoryHeader.addEventListener('click', () => {
            toggleCategory(category);
        });
        
        elements.list.appendChild(categoryHeader);

        // Create category content container
        const categoryContent = document.createElement('div');
        categoryContent.className = 'notes-category-content';
        categoryContent.dataset.expanded = state.expandedCategories[category] ? 'true' : 'false';

        files.forEach((file) => {
            const item = document.createElement('button');
            item.type = 'button';
            item.className = 'notes-file';
            
            // Display only the filename without the category prefix
            const displayName = file.replace(/^\$[A-Z]+\//, '');
            item.textContent = displayName;
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
            
            categoryContent.appendChild(item);
        });

        elements.list.appendChild(categoryContent);
    });
}

function toggleCategory(category) {
    state.expandedCategories[category] = !state.expandedCategories[category];
    renderFileList();
}

/**
 * Show/hide tabs based on file type
 */
function updateTabVisibility(fileType) {
    const isJson = fileType === 'json';
    
    // Hide/show markdown tabs
    elements.tabViewer.style.display = isJson ? 'none' : '';
    elements.tabEditor.style.display = isJson ? 'none' : '';
    elements.tabJupyter.style.display = isJson ? 'none' : '';
    
    // Hide/show JSON tabs
    elements.tabSwaggerView.style.display = isJson ? '' : 'none';
    elements.tabSwaggerEdit.style.display = isJson ? '' : 'none';
    elements.tabSwaggerRun.style.display = isJson && state.swaggerRunAvailable ? '' : 'none';
}

function renderSwaggerJsonView() {
    if (!elements.swaggerView || !elements.swaggerEditor) {
        return;
    }

    renderJsonViewer(elements.swaggerView, elements.swaggerEditor.value || '{}');
}


function safeSwaggerInfoUrl(value) {
    if (typeof value !== 'string') {
        return '';
    }

    const trimmed = value.trim();
    return /^https?:\/\//i.test(trimmed) ? trimmed : '';
}

function renderSwaggerInfoMetaValue(label, value) {
    if (!value) {
        return '';
    }

    return `
        <div class="swagger-info-meta-item">
            <span class="swagger-info-meta-label">${label}</span>
            <span class="swagger-info-meta-value">${value}</span>
        </div>
    `;
}

function renderSwaggerInfoMetadata(info) {
    if (!info || typeof info !== 'object') {
        return '';
    }

    const items = [];

    if (typeof info.summary === 'string' && info.summary.trim()) {
        items.push(renderSwaggerInfoMetaValue('Summary', escapeInfoText(info.summary.trim())));
    }

    if (typeof info.version === 'string' && info.version.trim()) {
        items.push(renderSwaggerInfoMetaValue('Version', escapeInfoText(info.version.trim())));
    }

    const termsUrl = safeSwaggerInfoUrl(info.termsOfService);
    if (termsUrl) {
        items.push(renderSwaggerInfoMetaValue(
            'Terms',
            `<a href="${escapeInfoText(termsUrl)}" target="_blank" rel="noopener noreferrer">${escapeInfoText(termsUrl)}</a>`
        ));
    }

    if (info.contact && typeof info.contact === 'object') {
        const contactParts = [];
        if (typeof info.contact.name === 'string' && info.contact.name.trim()) {
            contactParts.push(escapeInfoText(info.contact.name.trim()));
        }

        const contactUrl = safeSwaggerInfoUrl(info.contact.url);
        if (contactUrl) {
            contactParts.push(`<a href="${escapeInfoText(contactUrl)}" target="_blank" rel="noopener noreferrer">${escapeInfoText(contactUrl)}</a>`);
        }

        if (typeof info.contact.email === 'string' && info.contact.email.trim()) {
            const email = info.contact.email.trim();
            contactParts.push(`<a href="mailto:${encodeURIComponent(email)}">${escapeInfoText(email)}</a>`);
        }

        if (contactParts.length > 0) {
            items.push(renderSwaggerInfoMetaValue('Contact', contactParts.join(' · ')));
        }
    }

    if (info.license && typeof info.license === 'object') {
        const licenseName = typeof info.license.name === 'string' && info.license.name.trim()
            ? info.license.name.trim()
            : '';
        const licenseUrl = safeSwaggerInfoUrl(info.license.url);

        if (licenseName || licenseUrl) {
            const licenseValue = licenseUrl
                ? `<a href="${escapeInfoText(licenseUrl)}" target="_blank" rel="noopener noreferrer">${escapeInfoText(licenseName || licenseUrl)}</a>`
                : escapeInfoText(licenseName);
            items.push(renderSwaggerInfoMetaValue('License', licenseValue));
        }
    }

    if (items.length === 0) {
        return '';
    }

    return `<div class="swagger-info-meta">${items.join('')}</div>`;
}

function updateSwaggerLayoutMode() {
    if (!elements.swaggerRunWrap) {
        return;
    }

    const width = elements.swaggerRunWrap.getBoundingClientRect().width;
    if (width <= 0) {
        return;
    }

    const compact = width <= 900;
    elements.swaggerRunWrap.setAttribute('data-layout', compact ? 'compact' : 'wide');
}

/**
 * Render the Swagger/OpenAPI UI in the Run tab
 */
function renderSwaggerUI() {
    if (!state.swaggerSpec || !elements.swaggerEndpoints || !elements.swaggerRequestBuilder || !elements.swaggerResponse) {
        return;
    }

    const swaggerInfoEl = document.getElementById('notes-swagger-info');
    if (swaggerInfoEl) {
        const info = state.swaggerSpec.info || {};
        const title = typeof info.title === 'string' && info.title.trim() ? info.title.trim() : '';
        const description = typeof info.description === 'string' && info.description.trim() ? info.description.trim() : '';
        const metadata = renderSwaggerInfoMetadata(info);
        if (title || description || metadata) {
            swaggerInfoEl.innerHTML =
                (title ? `<h1 class="swagger-info-title">${escapeInfoText(title)}</h1>` : '') +
                (description ? `<div class="swagger-info-description markdown-body">${marked.parse(description)}</div>` : '') +
                metadata;
            processMarkdownContainer(swaggerInfoEl);
            swaggerInfoEl.style.display = '';
        } else {
            swaggerInfoEl.innerHTML = '';
            swaggerInfoEl.style.display = 'none';
        }
    }

    const currentFilterInput = elements.swaggerEndpoints.querySelector('#notes-swagger-endpoint-filter');
    const restoreFilterFocus = document.activeElement === currentFilterInput;
    const filterSelectionStart = restoreFilterFocus ? currentFilterInput.selectionStart : null;
    const filterSelectionEnd = restoreFilterFocus ? currentFilterInput.selectionEnd : null;
    
    // If no endpoint selected, select the first one
    if (!state.swaggerSelectedEndpoint) {
        const paths = extractPaths(state.swaggerSpec);
        if (paths.length > 0 && paths[0].methods.length > 0) {
            state.swaggerSelectedEndpoint = {
                path: paths[0].path,
                method: paths[0].methods[0].method
            };
        }
    }

    const endpointListHtml = generateEndpointListHTML(
        state.swaggerSpec,
        state.swaggerSelectedEndpoint,
        state.swaggerEndpointFilter
    );

    elements.swaggerEndpoints.innerHTML = `
        <div class="swagger-endpoints-header">Operations</div>
        <input
            id="notes-swagger-endpoint-filter"
            class="swagger-endpoint-filter"
            type="text"
            placeholder="Filter operations..."
            autocomplete="off"
            value="${state.swaggerEndpointFilter.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/\"/g, '&quot;')}"
        />
        ${endpointListHtml}
    `;
    
    // Render request builder and response
    elements.swaggerRequestBuilder.innerHTML = generateRequestBuilderHTML(state.swaggerSpec, state.swaggerSelectedEndpoint);
    elements.swaggerResponse.innerHTML = generateResponseHTML(state.swaggerSpec, state.swaggerSelectedEndpoint);

    // Render parameter descriptions using the same markdown pipeline as preview/info.
    elements.swaggerRequestBuilder.querySelectorAll('.swagger-param-description[data-markdown]').forEach((descEl) => {
        const markdown = descEl.getAttribute('data-markdown') || '';
        descEl.innerHTML = marked.parse(markdown);
        processMarkdownContainer(descEl);
    });
    
    // Add tab switching logic for nested tabs
    setupSwaggerTabSwitching();
    setupSwaggerEndpointSelection();
    setupSwaggerSendButton();

    if (restoreFilterFocus) {
        const nextFilterInput = elements.swaggerEndpoints.querySelector('#notes-swagger-endpoint-filter');
        if (nextFilterInput) {
            nextFilterInput.focus();
            const start = typeof filterSelectionStart === 'number' ? filterSelectionStart : nextFilterInput.value.length;
            const end = typeof filterSelectionEnd === 'number' ? filterSelectionEnd : start;
            nextFilterInput.setSelectionRange(start, end);
        }
    }
}

function setupSwaggerEndpointSelection() {
    const filterInput = elements.swaggerEndpoints.querySelector('#notes-swagger-endpoint-filter');
    if (filterInput) {
        filterInput.addEventListener('input', (event) => {
            state.swaggerEndpointFilter = event.target.value || '';
            renderSwaggerUI();
        });
    }

    const endpointButtons = elements.swaggerEndpoints.querySelectorAll('.swagger-endpoint-item');
    endpointButtons.forEach((button) => {
        button.addEventListener('click', () => {
            const path = button.getAttribute('data-path') || '';
            const method = button.getAttribute('data-method') || '';
            if (!path || !method) {
                return;
            }

            state.swaggerSelectedEndpoint = { path, method };
            renderSwaggerUI();
        });
    });
}

/**
 * Wire up the Send button to execute the current endpoint via the Go backend.
 */
function setupSwaggerSendButton() {
    const sendBtn = elements.swaggerRequestBuilder.querySelector('.swagger-send-btn');
    if (!sendBtn) {
        return;
    }

    sendBtn.addEventListener('click', () => {
        sendSwaggerRequest();
    });
}

async function sendSwaggerRequest() {
    if (!state.swaggerSpec || !state.swaggerSelectedEndpoint) {
        return;
    }

    const sendBtn = elements.swaggerRequestBuilder.querySelector('.swagger-send-btn');
    if (sendBtn) {
        sendBtn.disabled = true;
        sendBtn.dataset.sending = 'true';
        sendBtn.textContent = 'Sending…';
    }

    // Collect headers from the displayed header items
    // Values may be <input>, <select> (interactive) or <span> (static)
    const headers = {};
    elements.swaggerRequestBuilder.querySelectorAll('.swagger-header-item').forEach((item) => {
        const name = item.querySelector('.swagger-header-name')?.textContent?.trim();
        const valueEl = item.querySelector('.swagger-header-value');
        if (!name || !valueEl) return;
        const value = ('value' in valueEl ? valueEl.value : valueEl.textContent)?.trim();
        if (name && value) {
            headers[name] = value;
        }
    });

    // Collect body from the editable textarea
    const bodyTextarea = elements.swaggerRequestBuilder.querySelector('.swagger-body-editor');
    const body = bodyTextarea ? bodyTextarea.value : '';

    // Collect parameter values from the form inputs
    const parameters = {};
    elements.swaggerRequestBuilder.querySelectorAll('.swagger-param-input').forEach((input) => {
        const paramName = input.dataset.paramName;
        const paramIn = input.dataset.paramIn;
        const value = input.value?.trim();
        if (paramName && value) {
            parameters[paramName] = value;
        }
    });

    const url = buildRequestUrl(state.swaggerSpec, state.swaggerSelectedEndpoint, parameters);

    try {
        const response = await SwaggerRequest({
            method: state.swaggerSelectedEndpoint.method,
            url,
            headers,
            body,
        });

        elements.swaggerResponse.innerHTML = generateLiveResponseHTML(response);
        setupSwaggerResponseTabs();
    } catch (err) {
        elements.swaggerResponse.innerHTML = generateLiveResponseHTML({
            error: String(err?.message || err),
        });
    } finally {
        if (sendBtn) {
            sendBtn.disabled = false;
            sendBtn.dataset.sending = 'false';
            sendBtn.textContent = 'Send';
        }
    }
}

function setupSwaggerResponseTabs() {
    const responseTabs = elements.swaggerResponse.querySelectorAll('.swagger-response-tab');
    const responsePanels = elements.swaggerResponse.querySelectorAll('.swagger-response-panel');

    responseTabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const panelName = tab.getAttribute('data-tab');
            responsePanels.forEach(panel => panel.classList.remove('swagger-response-panel-active'));
            const selectedPanel = elements.swaggerResponse.querySelector(`.swagger-response-panel[data-panel="${panelName}"]`);
            if (selectedPanel) selectedPanel.classList.add('swagger-response-panel-active');
            responseTabs.forEach(t => t.setAttribute('aria-selected', 'false'));
            tab.setAttribute('aria-selected', 'true');
        });
    });
}

/**
 * Setup event listeners for nested tabs in swagger UI
 */
function setupSwaggerTabSwitching() {
    // Request tabs
    const requestTabs = elements.swaggerRequestBuilder.querySelectorAll('.swagger-request-tab');
    const requestPanels = elements.swaggerRequestBuilder.querySelectorAll('.swagger-request-panel');
    
    requestTabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const panelName = tab.getAttribute('data-tab');
            
            // Hide all panels
            requestPanels.forEach(panel => {
                panel.classList.remove('swagger-request-panel-active');
                panel.setAttribute('data-panel', panel.getAttribute('data-panel'));
            });
            
            // Show selected panel
            const selectedPanel = elements.swaggerRequestBuilder.querySelector(`.swagger-request-panel[data-panel="${panelName}"]`);
            if (selectedPanel) {
                selectedPanel.classList.add('swagger-request-panel-active');
            }
            
            // Update tab selection
            requestTabs.forEach(t => t.setAttribute('aria-selected', 'false'));
            tab.setAttribute('aria-selected', 'true');
        });
    });
    
    // Response tabs
    const responseTabs = elements.swaggerResponse.querySelectorAll('.swagger-response-tab');
    const responsePanels = elements.swaggerResponse.querySelectorAll('.swagger-response-panel');
    
    responseTabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const panelName = tab.getAttribute('data-tab');
            
            // Hide all panels
            responsePanels.forEach(panel => {
                panel.classList.remove('swagger-response-panel-active');
            });
            
            // Show selected panel
            const selectedPanel = elements.swaggerResponse.querySelector(`.swagger-response-panel[data-panel="${panelName}"]`);
            if (selectedPanel) {
                selectedPanel.classList.add('swagger-response-panel-active');
            }
            
            // Update tab selection
            responseTabs.forEach(t => t.setAttribute('aria-selected', 'false'));
            tab.setAttribute('aria-selected', 'true');
        });
    });
}

async function loadFile(file) {
    if (!file) {
        return;
    }

    try {
        const loadingJson = isJsonFile(file);
        const stickyId = loadingJson ? Date.now() : null;
        const fileName = file ? file.split('/').pop() : 'json file';

        if (loadingJson) {
            openStickyProgress(stickyId, `Loading ${fileName}… reading file`);
        }

        const doc = await GetMarkdown(file);

        state.currentFile = file;
        emitCurrentFileName();
        
        // Detect file type
        if (loadingJson) {
            state.currentFileType = 'json';
            updateStickyProgress(stickyId, `Loading ${fileName}… parsing json`);
            await yieldToUI();
            state.swaggerSpec = parseSwaggerSpec(doc);
            state.swaggerRunAvailable = hasSwaggerKey(state.swaggerSpec);

            if (!state.swaggerSpec) {
                closeStickyProgress(stickyId, `Failed to parse ${fileName}`, 'warn');
            }

            state.swaggerSelectedEndpoint = null;
            state.swaggerEndpointFilter = '';
            
            // Update UI for JSON / swagger-capable JSON
            updateTabVisibility('json');
            
            // Set editor content
            elements.swaggerEditor.value = doc || '';

            // Render JSON tree view
            updateStickyProgress(stickyId, `Loading ${fileName}… rendering viewer`);
            await yieldToUI();
            renderSwaggerJsonView();
            
            // Render swagger UI only for JSON documents with a top-level swagger key
            if (state.swaggerRunAvailable) {
                updateStickyProgress(stickyId, `Loading ${fileName}… rendering run view`);
                await yieldToUI();
                renderSwaggerUI();
            } else {
                elements.swaggerResponse.innerHTML = '';
                elements.swaggerRequestBuilder.innerHTML = '';
                elements.swaggerEndpoints.innerHTML = '';
            }
            
            // Set default view mode to JSON viewer
            setViewMode('swagger-view');
            closeStickyProgress(stickyId);
        } else {
            state.currentFileType = 'markdown';
            state.swaggerSpec = null;
            state.swaggerRunAvailable = false;
            
            // Update UI for markdown
            updateTabVisibility('markdown');
            
            // Set editor content
            elements.editor.value = doc || '';
            
            // Render markdown views
            renderMarkdown();
            renderJupyterView();
            
            // Set default view mode to viewer
            setViewMode('viewer');
        }
        
        setDirty(false);
        renderFileList();
        
        // Close find bar when loading a new file
        if (elements.findBar.dataset.open === 'true') {
            closeFindBar();
        }
    } catch (err) {
        if (stickyId) {
            closeStickyProgress(stickyId, `Failed to load ${file.split('/').pop()}`, 'error');
        }
        setStatus(`Failed to load ${file}.`, true);
        console.error(err);
    }
}

async function saveFile() {
    if (!state.currentFile) {
        setStatus('Select a note before saving.', true);
        return;
    }

    try {
        const content = state.currentFileType === 'json' 
            ? elements.swaggerEditor.value 
            : elements.editor.value;
        
        await SaveFile(state.currentFile, content);
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

    const fileToDelete = state.deletingFile;
    const fileName = fileToDelete.split('/').pop();

    try {
        await DeleteFile(fileToDelete);
        if (state.currentFile === fileToDelete) {
            state.currentFile = '';
            emitCurrentFileName();
            elements.editor.value = '';
            elements.swaggerEditor.value = '';
            elements.swaggerView.innerHTML = '';
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

function openFindBar() {
    elements.findBar.dataset.open = 'true';
    elements.findBar.setAttribute('aria-hidden', 'false');
    setTimeout(() => {
        elements.findInput.focus();
        elements.findInput.select();
    }, 0);
}

function closeFindBar() {
    elements.findBar.dataset.open = 'false';
    elements.findBar.setAttribute('aria-hidden', 'true');
    clearHighlights();
    state.findMatches = [];
    state.findCurrentIndex = -1;
    state.findQuery = '';
    elements.findCounter.textContent = '';
}

function getActiveFindContainer() {
    return state.viewMode === 'jupyter' ? elements.jupyter : elements.preview;
}

function clearHighlights() {
    // Clear highlights in both rendered panes
    [elements.preview, elements.jupyter].forEach((container) => {
        const highlights = container.querySelectorAll('.find-highlight');
        highlights.forEach((el) => {
            const parent = el.parentNode;
            parent.replaceChild(document.createTextNode(el.textContent), el);
            parent.normalize();
        });
    });

    // Clear editor selection
    if (state.viewMode === 'editor') {
        elements.editor.setSelectionRange(0, 0);
    }
}

function performFind() {
    const query = elements.findInput.value;
    if (!query) {
        closeFindBar();
        return;
    }

    state.findQuery = query;
    clearHighlights();
    state.findMatches = [];
    state.findCurrentIndex = -1;

    if (state.viewMode === 'editor') {
        findInEditor();
    } else {
        findInRenderedPane();
    }

    if (state.findMatches.length > 0) {
        state.findCurrentIndex = 0;
        highlightCurrentMatch();
    }

    updateFindCounter();
}

function findInEditor() {
    const text = elements.editor.value.toLowerCase();
    const query = state.findQuery.toLowerCase();
    let index = 0;

    while ((index = text.indexOf(query, index)) !== -1) {
        state.findMatches.push({
            start: index,
            end: index + query.length
        });
        index += query.length;
    }
}

function findInRenderedPane() {
    const query = state.findQuery;
    const container = getActiveFindContainer();
    const walker = document.createTreeWalker(
        container,
        NodeFilter.SHOW_TEXT,
        null,
        false
    );

    const nodesToProcess = [];
    let node;
    while ((node = walker.nextNode())) {
        if (node.textContent.toLowerCase().includes(query.toLowerCase())) {
            nodesToProcess.push(node);
        }
    }

    nodesToProcess.forEach((textNode) => {
        const text = textNode.textContent;
        const lowerText = text.toLowerCase();
        const lowerQuery = query.toLowerCase();
        const parts = [];
        let lastIndex = 0;
        let index;

        while ((index = lowerText.indexOf(lowerQuery, lastIndex)) !== -1) {
            if (index > lastIndex) {
                parts.push(document.createTextNode(text.substring(lastIndex, index)));
            }

            const highlight = document.createElement('span');
            highlight.className = 'find-highlight';
            highlight.textContent = text.substring(index, index + query.length);
            parts.push(highlight);
            state.findMatches.push(highlight);

            lastIndex = index + query.length;
        }

        if (lastIndex < text.length) {
            parts.push(document.createTextNode(text.substring(lastIndex)));
        }

        const parent = textNode.parentNode;
        parts.forEach((part) => {
            parent.insertBefore(part, textNode);
        });
        parent.removeChild(textNode);
    });
}

function highlightCurrentMatch() {
    if (state.findMatches.length === 0 || state.findCurrentIndex === -1) {
        return;
    }

    if (state.viewMode === 'editor') {
        const match = state.findMatches[state.findCurrentIndex];
        elements.editor.focus();
        elements.editor.setSelectionRange(match.start, match.end);
        
        // Scroll to the selection
        const lineHeight = parseInt(getComputedStyle(elements.editor).lineHeight);
        const textBeforeMatch = elements.editor.value.substring(0, match.start);
        const lineNumber = (textBeforeMatch.match(/\n/g) || []).length;
        elements.editor.scrollTop = lineNumber * lineHeight - elements.editor.clientHeight / 2;
    } else {
        // Clear previous active highlight
        const prevActive = getActiveFindContainer().querySelector('.find-highlight-active');
        if (prevActive) {
            prevActive.classList.remove('find-highlight-active');
        }

        // Highlight current match
        const currentMatch = state.findMatches[state.findCurrentIndex];
        currentMatch.classList.add('find-highlight-active');
        currentMatch.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
}

function nextMatch() {
    if (state.findMatches.length === 0) {
        return;
    }

    state.findCurrentIndex = (state.findCurrentIndex + 1) % state.findMatches.length;
    highlightCurrentMatch();
    updateFindCounter();
}

function prevMatch() {
    if (state.findMatches.length === 0) {
        return;
    }

    state.findCurrentIndex = (state.findCurrentIndex - 1 + state.findMatches.length) % state.findMatches.length;
    highlightCurrentMatch();
    updateFindCounter();
}

function updateFindCounter() {
    if (state.findMatches.length === 0) {
        elements.findCounter.textContent = 'No matches';
    } else {
        elements.findCounter.textContent = `${state.findCurrentIndex + 1} of ${state.findMatches.length}`;
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

function normalizeNotePath(rawName) {
    const fileName = normalizeNoteName(rawName);
    if (fileName === '') {
        return '';
    }

    if (fileName.startsWith('$') || fileName.startsWith('/')) {
        return fileName;
    }

    return `$NOTES/${fileName}`;
}

function deriveImageExtension(mimeType) {
    if (!mimeType) {
        return 'png';
    }

    const subtype = mimeType.split('/')[1] || '';
    const normalized = subtype.toLowerCase().split('+')[0];
    if (normalized === 'jpeg') {
        return 'jpg';
    }

    if (/^[a-z0-9]+$/.test(normalized)) {
        return normalized;
    }

    return 'png';
}

function buildImagePaths(notePath, epoch, extension) {
    const slash = notePath.lastIndexOf('/');
    const dir = slash === -1 ? '' : notePath.slice(0, slash + 1);
    const file = slash === -1 ? notePath : notePath.slice(slash + 1);
    const stem = file.replace(/\.[^/.]+$/, '');

    const imageFileName = `${stem}.${epoch}.${extension}`;
    return {
        imagePath: `${dir}${imageFileName}`,
        imageFileName,
    };
}

function getMarkdownImageAtCursor(markdown, cursor) {
    if (!markdown || !Number.isFinite(cursor)) {
        return null;
    }

    const imageRegex = /!\[[^\]]*\]\(([^)]+)\)/g;
    let match;

    while ((match = imageRegex.exec(markdown)) !== null) {
        const start = match.index;
        const end = start + match[0].length;
        if (cursor < start || cursor > end) {
            continue;
        }

        const rawTarget = (match[1] || '').trim();
        if (rawTarget === '') {
            return null;
        }

        let imagePath = rawTarget;
        if (rawTarget.startsWith('<') && rawTarget.endsWith('>')) {
            imagePath = rawTarget.slice(1, -1).trim();
        } else {
            const splitAt = rawTarget.search(/\s/);
            if (splitAt !== -1) {
                imagePath = rawTarget.slice(0, splitAt).trim();
            }
        }

        return {
            markdown: match[0],
            markdownStart: start,
            markdownEnd: end,
            imagePath,
        };
    }

    return null;
}

function isRelativeMarkdownImagePath(imagePath) {
    if (!imagePath) {
        return false;
    }

    if (imagePath.startsWith('/') || imagePath.startsWith('$') || imagePath.startsWith('//')) {
        return false;
    }

    // Exclude schemes like http:, https:, data:, file:, etc.
    if (/^[a-z][a-z0-9+.-]*:/i.test(imagePath)) {
        return false;
    }

    return true;
}

function resolveRelativeAssetPath(notePath, relativePath) {
    const slash = notePath.lastIndexOf('/');
    const dir = slash === -1 ? '' : notePath.slice(0, slash + 1);
    return `${dir}${relativePath}`;
}

function enableImageContextMenus(container) {
    const images = container.querySelectorAll('img');
    images.forEach((img) => {
        img.addEventListener('contextmenu', async (e) => {
            e.preventDefault();
            
            const src = img.src;
            if (!src) return;
            
            // Use the original filename from the data attribute if available
            let filename = img.dataset.originalFilename || 'Image';
            
            // For relative image paths (from note markdown images), convert to dataURL
            let dataURLToCopy = src;
            if (src.startsWith('file://') || (!src.startsWith('data:') && !src.startsWith('http'))) {
                // It's a file path, we need to fetch and convert to dataURL
                try {
                    const response = await fetch(src);
                    const blob = await response.blob();
                    dataURLToCopy = await new Promise((resolve) => {
                        const reader = new FileReader();
                        reader.onload = () => resolve(reader.result);
                        reader.readAsDataURL(blob);
                    });
                } catch (err) {
                    console.error('Failed to load image for clipboard:', err);
                    return;
                }
            }
            
            showLocalMenu({
                title: filename,
                options: ['Copy image to clipboard', 'Save image...'],
                x: e.clientX,
                y: e.clientY,
                icons: [0xf0c5, 0xf0c7],
                onSelect: (index) => {
                    if (index === 0) {
                        TerminalCopyImageDataURL(dataURLToCopy).catch(() => {
                            setStatus('Failed to copy image to clipboard.', true);
                        });
                    } else if (index === 1) {
                        saveImageToFile(filename, dataURLToCopy);
                    }
                },
            });
        });
    });
}

function copyTextToClipboard(text) {
    if (!text) {
        return;
    }

    ClipboardSetText(text).catch(() => {});
}

function getEditorSelectionText() {
    const start = elements.editor.selectionStart;
    const end = elements.editor.selectionEnd;
    return elements.editor.value.slice(start, end);
}

function getRenderedSelectionText(container) {
    const selection = window.getSelection();
    if (!selection || selection.rangeCount === 0 || selection.isCollapsed) {
        return '';
    }

    const anchorNode = selection.anchorNode;
    const focusNode = selection.focusNode;
    const selectionInContainer =
        (anchorNode && container.contains(anchorNode)) ||
        (focusNode && container.contains(focusNode));

    if (!selectionInContainer) {
        return '';
    }

    return selection.toString();
}

function createCopyMenuItem(getText, title = 'Copy') {
    return {
        title,
        icon: CONTEXT_ICON_COPY,
        onSelect: () => {
            copyTextToClipboard(getText());
        },
    };
}

function createFindMenuItem(title = 'Find text...') {
    return {
        title,
        icon: CONTEXT_ICON_FIND,
        onSelect: () => {
            openFindBar();
        },
    };
}

function createPrintMenuItem(title = 'Print...') {
    return {
        title,
        icon: CONTEXT_ICON_PRINT,
        onSelect: () => {
            WindowPrint();
        },
    };
}

function showNotesLocalMenu(menuItems, x, y, title = 'Select an action') {
    showLocalMenu({
        title,
        options: menuItems.map((item) => item.title),
        icons: menuItems.map((item) => item.icon),
        x,
        y,
        onSelect: (index) => {
            const item = menuItems[index];
            if (item && typeof item.onSelect === 'function') {
                item.onSelect();
            }
        },
    });
}

function initRenderedNotesContextMenu(container, viewMode) {
    container.addEventListener('contextmenu', (e) => {
        if (state.viewMode !== viewMode) {
            return;
        }

        if (e.target instanceof Element && e.target.closest('img')) {
            return;
        }

        e.preventDefault();

        showNotesLocalMenu([
            createCopyMenuItem(() => getRenderedSelectionText(container), 'Copy'),
            { title: '-' },
            createFindMenuItem('Find'),
            createPrintMenuItem('Print'),
        ], e.clientX, e.clientY);
    });
}

async function createNewFile() {
    let fileName = normalizeNoteName(elements.modalInput.value);
    if (fileName === '') {
        setStatus('File name cannot be empty.', true);
        return;
    }

    // Handle rename operation
    if (state.renamingFile) {
        try {
            await RenameFile(state.renamingFile, fileName);
            await refreshFiles();
            if (state.currentFile === state.renamingFile) {
                await loadFile(fileName);
            }
            closeNewFilePrompt();
            setStatus(`Renamed to ${fileName}.`, false);
        } catch (err) {
            setStatus(`Failed to rename file.`, true);
            console.error(err);
        }
        return;
    }

    // Handle new file creation

    const exists = state.files.some((file) => file === fileName);
    if (exists) {
        closeNewFilePrompt();
        await loadFile(fileName);
        setStatus(`${fileName} already exists.`, false);
        return;
    }

    try {
        await SaveFile(fileName, '');
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

async function createAndOpenFile(filename, contents) {
    const fileName = normalizeNotePath(filename);
    if (fileName === '') {
        setStatus('File name cannot be empty.', true);
        return;
    }

    try {
        await SaveFile(fileName, contents || '');
        await refreshFiles();
        await loadFile(fileName);
        //setViewMode('editor');
        setViewMode('viewer');
        setStatus(`Created ${fileName}.`, false);
    } catch (err) {
        setStatus(`Failed to create ${fileName}.`, true);
        console.error(err);
    }
}

async function saveImageToFile(filename, dataURL) {
    try {
        // Open save dialog via Wails runtime API (through Go binding)
        const savedPath = await SaveImageDialog(filename);
        
        if (!savedPath) {
            return; // User cancelled
        }
        
        // Extract base64 data from dataURL
        const base64Data = dataURL.split(',')[1];
        if (!base64Data) {
            setStatus('Failed to extract image data.', true);
            return;
        }
        
        // Save the file
        await SaveBinaryFile(savedPath, base64Data);
        setStatus(`Image saved to ${savedPath}.`, false);
    } catch (err) {
        setStatus(`Failed to save image: ${err.message || err}`, true);
        console.error('Error saving image:', err);
    }
}

EventsOn("notesCreateAndOpen", params => {
    createAndOpenFile(params.filename, params.contents);
});

EventsOn("notesUpdate", group => {
    elements.title.innerText = group;
    refreshFiles();
});

EventsOn("noteRun", (data) => {
    const { blockId, output, isError } = data;
    const outputBlock = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-output`);
    if (!outputBlock) return;

    const text = String(output ?? '');
    const isErr = String(isError) === 'true';

    if (outputBlock.childNodes.length > 0 && text.length > 0 && text[0] !== '\n' && text[0] !== '\r') {
        outputBlock.appendChild(document.createTextNode('\n'));
    }

    const span = document.createElement('span');
    span.className = isErr ? 'jupyter-output-line-error' : 'jupyter-output-line';
    span.textContent = text;
    outputBlock.appendChild(span);
});

EventsOn("noteComplete", (data) => {
    const { blockId } = data;
    // Toggle buttons back to Run
    const runBtn = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-run-notes`);
    const stopBtn = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-stop-notes`);
    if (runBtn) runBtn.style.display = 'inline-block';
    if (stopBtn) stopBtn.style.display = 'none';
});

// AI Panel Event Handlers
function setAIPanelCollapsed(collapsed) {
    const isCollapsed = collapsed === true;
    elements.aiPanel.dataset.collapsed = isCollapsed ? 'true' : 'false';
    elements.aiToggle.textContent = isCollapsed ? 'AI ▲' : 'AI ▼';
    if (elements.aiRestore) {
        elements.aiRestore.style.display = isCollapsed ? 'inline-flex' : 'none';
    }
    localStorage.setItem('notes-ai-panel-collapsed', String(isCollapsed));
}

function toggleAIPanel() {
    const isCollapsed = elements.aiPanel.dataset.collapsed === 'true';
    setAIPanelCollapsed(!isCollapsed);
}

function clearAIOutput() {
    elements.aiOutput.textContent = '';
}

function appendAIText(text) {
    if (elements.aiOutput.textContent === 'No AI response yet') {
        elements.aiOutput.textContent = '';
    }
    elements.aiOutput.appendChild(document.createTextNode(text));
    elements.aiOutput.scrollTop = elements.aiOutput.scrollHeight;
}

// Event listener for streaming AI responses
EventsOn("aiResponseStream", (chunk) => {
    const text = String(chunk ?? '');
    if (text) {
        appendAIText(text);
        // Auto-expand AI panel when response starts
        if (elements.aiPanel.dataset.collapsed === 'true') {
            toggleAIPanel();
        }
    }
});

// Setup AI panel listeners
if (elements.aiToggle) {
    elements.aiToggle.addEventListener('click', toggleAIPanel);
}
if (elements.aiClear) {
    elements.aiClear.addEventListener('click', clearAIOutput);
}
if (elements.aiRestore) {
    elements.aiRestore.addEventListener('click', () => setAIPanelCollapsed(false));
}

// Always start minimized on application launch.
setAIPanelCollapsed(true);

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
            --red: rgb(${result.colors.red.Red}, ${result.colors.red.Green}, ${result.colors.red.Blue});
            --green: rgb(${result.colors.green.Red}, ${result.colors.green.Green}, ${result.colors.green.Blue});
            --yellow: rgb(${result.colors.yellow.Red}, ${result.colors.yellow.Green}, ${result.colors.yellow.Blue});
            --blue: rgb(${result.colors.blue.Red}, ${result.colors.blue.Green}, ${result.colors.blue.Blue});
            --magenta: rgb(${result.colors.magenta.Red}, ${result.colors.magenta.Green}, ${result.colors.magenta.Blue});
            --cyan: rgb(${result.colors.cyan.Red}, ${result.colors.cyan.Green}, ${result.colors.cyan.Blue});
            --red-bright: rgb(${result.colors.redBright.Red}, ${result.colors.redBright.Green}, ${result.colors.redBright.Blue});
            --green-bright: rgb(${result.colors.greenBright.Red}, ${result.colors.greenBright.Green}, ${result.colors.greenBright.Blue});
            --yellow-bright: rgb(${result.colors.yellowBright.Red}, ${result.colors.yellowBright.Green}, ${result.colors.yellowBright.Blue});
            --blue-bright: rgb(${result.colors.blueBright.Red}, ${result.colors.blueBright.Green}, ${result.colors.blueBright.Blue});
            --magenta-bright: rgb(${result.colors.magentaBright.Red}, ${result.colors.magentaBright.Green}, ${result.colors.magentaBright.Blue});
            --cyan-bright: rgb(${result.colors.cyanBright.Red}, ${result.colors.cyanBright.Green}, ${result.colors.cyanBright.Blue});
            --selection: rgb(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue});
            --error: rgb(${result.colors.error.Red}, ${result.colors.error.Green}, ${result.colors.error.Blue});
            --font-family: ${result.fontFamily};
        }

        * {
            box-sizing: border-box;
            font-family: var(--font-family);
        }

        body {
            margin: 0 !important;
            padding: 0 !important;
        }

        ::selection {
            background-color: var(--selection);
        }

        ${getScrollbarStyles(result.colors)}

        #notes-app {
            display: grid;
            grid-template-columns: 25% 1fr;
            height: 100%;
            overflow: hidden;
            color: var(--fg);
            background: var(--bg);
        }

        #notes-sidebar {
            display: flex;
            flex-direction: column;
            padding-left: 15px;
            padding-top: 10px;
            padding-right: 0px;
            padding-bottom: 5px;
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
            border-color: var(--selection);
            color: var(--selection);
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
            border-radius: 5px;
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
            border-radius: 5px;
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

        .notes-toolbar {
            display: flex;
            gap: 4px;
            margin-left: auto;
            align-items: center;
        }

        .notes-toolbar-btn {
            border: none;
            background: transparent;
            color: var(--fg);
            font-size: 16px;
            cursor: pointer;
            padding: 6px 8px;
            display: flex;
            align-items: center;
            justify-content: center;
            border-radius: 4px;
            font-family: "Font Awesome Solid", "Font Awesome", sans-serif;
            font-weight: 900;
            transition: color 0.2s, background-color 0.2s;
            border-width: 1px !important;
        }

        /*.notes-toolbar-btn:hover {
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.3);
            color: var(--fg);
        }*/

        #notes-new:hover {
            color: var(--green) !important;
        }

        #notes-rename:hover, #notes-find:hover {
            color: var(--yellow) !important;
            border-radius: 5px;
            border-color: var(--yellow) !important;
            background-color: rgba(${result.colors.yellow.Red}, ${result.colors.yellow.Green}, ${result.colors.yellow.Blue}, 0.3);
        }

        #notes-delete:hover {
            color: var(--red) !important;
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
            border-radius: 5px;
            border: 2px solid var(--fg);
            background: transparent;
            color: var(--fg);
            padding: 6px 10px;
            cursor: pointer;
        }

        #notes-delete-modal-actions button {
            border-radius: 5px;
            border: 2px solid var(--fg);
            background: transparent;
            color: var(--fg);
            padding: 6px 10px;
            cursor: pointer;
        }

        #notes-modal-actions button:hover {
            border-color: var(--selection);
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.2);
            transition: all 0.2s ease;
        }

        #notes-delete-modal-actions button:hover {
            border-color: var(--selection);
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.2);
            transition: all 0.2s ease;
        }

        #notes-delete-confirm {
            border-color: var(--error);
            color: var(--error);
        }

        #notes-delete-confirm:hover {
            border-color: var(--error) !important;
            color: var(--error) !important;
            background-color: rgba(${result.colors.error.Red}, ${result.colors.error.Green}, ${result.colors.error.Blue}, 0.2);
            transition: all 0.2s ease;
        }

        #notes-list {
            display: flex;
            flex-direction: column;
            gap: 6px;
            overflow-y: auto;
            overflow-x: hidden;
            flex: 1;
        }

        .notes-category-header {
            display: flex;
            align-items: center;
            gap: 6px;
            padding: 6px 8px;
            cursor: pointer;
            color: var(--accent);
            font-weight: bold;
            border: 2px solid transparent;
            user-select: none;
        }

        .notes-category-header:hover {
            border-color: var(--selection);
        }

        .notes-category-arrow {
            font-size: ${result.fontSize - 2}px;
            width: 12px;
            display: inline-block;
        }

        .notes-category-content {
            display: flex;
            flex-direction: column;
            gap: 4px;
            padding-left: 18px;
        }

        .notes-category-content[data-expanded="false"] {
            display: none;
        }

        .notes-file {
            min-height: ${notesFileSize}px;
            text-align: left;
            border-radius: 5px;
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
            border-color: var(--fg);
            color: var(--fg);
        }

        .notes-file:hover {
            border-color: var(--selection);
        }

        #notes-empty {
            opacity: 0.7;
        }

        #notes-status {
            font-size: ${notesStatusFontSize}px;
            opacity: 0.8;
            color: var(--fg);
        }

        #notes-status[data-state="error"] {
            color: var(--error);
        }

        #notes-main {
            display: flex;
            flex-direction: column;
            gap: 12px;
            padding: 2px 4px;
            height: 100%;
            min-height: 0;
        }

        #notes-tabs {
            display: flex;
            gap: 8px;
            padding: 1px 0px 0 8px;
            border-bottom: 2px solid var(--fg);
            align-items: center;
            box-sizing: border-box;
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
            border-color: var(--fg);
            border-bottom: 5px;
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.2);
            border-color: var(--fg) !important;
        }

        .tab {
            border-top-left-radius: 5px !important;
            border-top-right-radius: 5px !important;
            border: 2px solid !important;
            border-bottom: 0 !important;
            border-color: rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.2) !important;
        }

        .tab:hover {
            border: 2px solid !important;
            border-bottom: 0 !important;
            border-color: var(--fg) !important;
        }

        #notes-tabs button:hover {
            border-color: var(--selection);
        }

        #notes-new:hover {
            border-color: var(--green) !important;
            color: var(--green) !important;
            background-color: rgba(${result.colors.green.Red}, ${result.colors.green.Green}, ${result.colors.green.Blue}, 0.2);
            border-radius: 5px;
        }

        #notes-delete {
            color: var(--error);
        }

        #notes-delete:hover {
            border-color: var(--error) !important;
            color: var(--error);
            background-color: rgba(${result.colors.error.Red}, ${result.colors.error.Green}, ${result.colors.error.Blue}, 0.2);
            border-radius: 5px;
        }

        #notes-panel {
            position: relative;
            flex: 1;
            min-height: 0;
            display: flex;
            flex-direction: column;
        }

        #notes-editor-wrap,
        #notes-preview-wrap,
        #notes-jupyter-wrap {
            flex: 1;
            display: none;
            min-height: 0;
            border-bottom: 1px solid rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.2);
        }

        #notes-swagger-edit-wrap,
        #notes-swagger-view-wrap,
        #notes-swagger-run-wrap {
            flex: 1;
            display: none;
            min-height: 0;
            overflow: hidden;
        }

        #notes-editor-wrap[data-active="true"],
        #notes-preview-wrap[data-active="true"],
        #notes-jupyter-wrap[data-active="true"] {
            display: block;
        }

        #notes-swagger-view-wrap[data-active="true"],
        #notes-swagger-edit-wrap[data-active="true"],
        #notes-swagger-run-wrap[data-active="true"] {
            display: flex !important;
        }

        .notes-ai-panel {
            display: flex;
            flex-direction: column;
            border-top: 2px solid var(--fg);
            transition: all 0.3s ease;
            overflow: hidden;
        }

        .notes-ai-panel[data-collapsed="false"] {
            flex: 0 1 35%;
            overflow-y: auto;
        }

        .notes-ai-panel[data-collapsed="true"] {
            flex: 0 0 0;
            min-height: 0;
            border-top: 0;
            opacity: 0;
            pointer-events: none;
        }

        .notes-ai-restore {
            display: none;
            position: absolute;
            right: 12px;
            bottom: 12px;
            z-index: 2;
            border-radius: 999px;
            border: 1px solid rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.4);
            background: rgba(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue}, 0.9);
            color: var(--fg);
            padding: 6px 12px;
            cursor: pointer;
            font-size: ${result.fontSize - 2}px;
            align-items: center;
            justify-content: center;
        }

        .notes-ai-restore:hover {
            border-color: var(--fg);
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 1);
        }

        .notes-ai-header {
            display: flex;
            gap: 8px;
            align-items: center;
            padding: 8px 12px;
            background: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.1);
            border-bottom: 1px solid rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.2);
            flex-shrink: 0;
        }

        .notes-ai-header button {
            border-radius: 3px;
            border: 1px solid rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.3);
            background: transparent;
            color: var(--fg);
            padding: 4px 10px;
            cursor: pointer;
            font-size: ${result.fontSize - 2}px;
            transition: all 0.2s ease;
        }

        .notes-ai-header button:hover {
            border-color: var(--fg);
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.2);
        }

        #notes-ai-clear:hover {
            color: var(--error);
            border-color: var(--error);
        }

        #notes-ai-output {
            flex: 1;
            padding: 12px;
            font-size: ${result.fontSize}px;
            line-height: 1.5;
            overflow-x: hidden;
            overflow-y: auto;
            white-space: pre-wrap;
            word-wrap: break-word;
            overflow-wrap: anywhere;
            word-break: break-word;
            font-family: monospace;
            color: var(--fg);
            background-color: rgba(0, 0, 0, 0.2);
        }

        #notes-ai-output:empty::before {
            content: "No AI response yet";
            opacity: 0.5;
            font-style: italic;
        }

        #notes-editor {
            width: 100%;
            height: 100%;
            resize: none;
            border-radius: 0;
            border: 2px solid var(--bg);
            background: transparent;
            color: var(--fg);
            padding: 10px;
            font-size: ${result.fontSize}px;
            line-height: 1.4;
        }

        #notes-editor:focus {
            outline: none;
            box-shadow: none;
            border: 2px solid var(--selection);
        }

        #notes-editor:not(:focus) {
            background-color: rgba(0, 0, 0, 0.2);
        }

        #notes-preview-wrap,
        #notes-jupyter-wrap {
            overflow-y: auto;
            padding-left: 16px;
        }

        ${getMarkdownBaseTextSizeStyles('#notes-preview', result.fontSize)}

        ${getMarkdownBaseTextSizeStyles('#notes-jupyter', result.fontSize)}

        ${getMarkdownBaseTextSizeStyles('#notes-swagger-info', result.fontSize)}

        ${getMarkdownBaseTextSizeStyles('#notes-swagger-request-builder .swagger-param-description', result.fontSize)}

        ${getMarkdownContentStyles(result.colors, result.fontSize, 'markdown-body')}

        ${getCheckboxStyles(result.colors, result.fontSize, 'markdown-body')}

        ${getHighlightJsTheme(result.colors, true)}

        #notes-preview img,
        #notes-jupyter img {
            max-width: 100%;
            height: auto;
        }

        #notes-find-bar {
            border-radius: 5px;
            position: absolute;
            top: 16px;
            right: 16px;
            display: none;
            align-items: center;
            gap: 8px;
            padding: 8px 12px;
            background: var(--bg);
            border: 2px solid var(--fg);
            z-index: 100;
        }

        #notes-find-bar[data-open="true"] {
            display: flex;
        }

        #notes-find-input {
            border-radius: 0;
            border: 2px solid var(--fg);
            background: transparent;
            color: var(--fg);
            padding: 4px 8px;
            font-size: ${result.fontSize}px;
            outline: none;
            min-width: 200px;
        }

        #notes-find-counter {
            font-size: ${result.fontSize - 2}px;
            opacity: 0.8;
            white-space: nowrap;
        }

        #notes-find-bar button {
            border-radius: 5px;
            border: 2px solid var(--fg);
            background: transparent;
            color: var(--fg);
            padding: 4px 8px;
            cursor: pointer;
            font-size: ${result.fontSize}px;
        }

        #notes-find-bar button:hover {
            border-color: var(--accent);
            color: var(--accent);
            transition: all 0.2s ease;
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.3);
        }

        .find-highlight {
            background-color: var(--accent);
            color: var(--bg);
        }

        .find-highlight-active {
            background-color: var(--blue);
            color: var(--bg);
        }

        /* Jupyter UI Styles */

        #notes-jupyter-wrap pre {
            border-left: 0;
            padding-left: 10px;
            /*white-space: pre-wrap;
            word-wrap: break-word;*/
        }

        .jupyter-code-block {
            margin: 16px 0;
            border: 2px solid var(--fg);
            border-radius: 5px;
        }

        .jupyter-toolbar {
            display: flex;
            gap: 8px;
            padding: 0px;
            padding-left: 8px;
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.2);
            border-bottom: 2px solid var(--fg);
            align-items: center;
        }

        .jupyter-btn {
            padding: 5px 12px;
            margin-top: 8px;
            margin-bottom: 8px;
            background-color: transparent;
            border: 1px solid var(--fg);
            color: var(--fg);
            cursor: pointer;
            font-size: ${result.fontSize - 2}px;
            border-radius: 5px;
            transition: all 0.2s ease;
            align-items: center;
            vertical-align: middle;
        }
     
        .jupyter-btn:hover {
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.3);
            border-color: var(--accent);
            color: var(--accent);
        }

        .jupyter-btn:active {
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.5);
        }

        .jupyter-stop-notes {
            border-color: var(--red);
            color: var(--red);
        }

        .jupyter-stop-notes:hover {
            background-color: rgba(${result.colors.red.Red}, ${result.colors.red.Green}, ${result.colors.red.Blue}, 0.3);
            border-color: var(--red);
            color: var(--red);
        }

        .jupyter-stop-notes:active {
            background-color: rgba(${result.colors.red.Red}, ${result.colors.red.Green}, ${result.colors.red.Blue}, 0.5);
        }

        .jupyter-runtime-dropdown {
            margin: 8px;
            padding: 5px 24px 5px 12px;
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0);
            border: none;
            color: var(--accent);
            font-size: ${result.fontSize - 2}px;
            opacity: 0.8;
            cursor: pointer;
            outline: none;
            -webkit-appearance: none;
            -moz-appearance: none;
            appearance: none;
            text-align: right;
            align-items: right;
            vertical-align: middle;
        }

        .jupyter-runtime-dropdown:hover {
            opacity: 1;
            color: var(--fg);
        }

        .jupyter-runtime-dropdown:focus {
            opacity: 1;
            color: var(--fg);
        }

        .jupyter-runtime-dropdown option {
            background-color: var(--bg);
            color: var(--fg);
            padding: 4px 8px;
        }

        .jupyter-code-editor {
            display: flex;
            align-items: stretch;
            background-color: var(--bg);
        }

        .jupyter-line-numbers {
            min-width: 42px;
            margin: 0;
            padding: 12px 8px 12px 10px;
            border-right: 1px solid rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.2);
            color: var(--fg);
            opacity: 0.45;
            font-family: monospace;
            font-size: ${result.fontSize}px;
            line-height: 1.5;
            text-align: right;
            white-space: pre;
            user-select: none;
            pointer-events: none;
            overflow: hidden;
        }

        .jupyter-code-editable {
            flex: 1;
            width: auto;
            margin: 0;
            padding: 12px;
            background-color: var(--bg);
            border: none;
            color: var(--fg);
            font-family: monospace;
            font-size: ${result.fontSize}px;
            line-height: 1.5;
            overflow-x: auto;
            overflow-y: hidden;
            outline: none;
            resize: none;
            box-sizing: border-box;
            white-space: pre;
        }

        .jupyter-code-editable:focus {
            outline: none;
        }

        .jupyter-code-editable:not(:focus) {
            background-color: rgba(0, 0, 0, 0.2);
        }

        .jupyter-output-wrapper {
            border-top: 2px solid var(--fg);
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.1);
        }

        .jupyter-output-toggle {
            width: 100%;
            padding: 8px 12px;
            background-color: transparent;
            border: none;
            border-bottom: 1px solid var(--fg);
            color: var(--fg);
            cursor: pointer;
            font-size: ${result.fontSize - 2}px;
            text-align: left;
            transition: all 0.2s ease;
        }

        .jupyter-output-toggle:hover {
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.2);
            color: var(--accent);
        }

        .jupyter-output {
            margin: 0;
            padding: 12px;
            background-color: var(--bg);
            color: var(--fg);
            font-family: monospace;
            font-size: ${result.fontSize - 2}px;
            line-height: 1.4;
            overflow-x: auto;
            white-space: pre-wrap;
            word-wrap: break-word;
            border: none;
        }

        .jupyter-output-line {
            color: var(--green);
        }

        .jupyter-output-line-error {
            color: var(--error);
        }

        /* Swagger Editor */
        #notes-swagger-editor {
            width: 100%;
            height: 100%;
            padding: 12px;
            border: none;
            background-color: var(--bg);
            color: rgb(${result.colors.green.Red}, ${result.colors.green.Green}, ${result.colors.green.Blue});
            font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Courier New', monospace;
            font-size: ${result.fontSize}px;
            resize: none;
            overflow: auto;
        }

        #notes-swagger-editor:not(:focus) {
            background-color: rgba(0, 0, 0, 0.2);
        }

        #notes-swagger-view-wrap {
            display: flex;
            flex-direction: column;
            height: 100%;
            overflow: auto;
            padding: 10px;
        }

        #notes-swagger-view {
            overflow: auto;
            padding-right: 8px;
            font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Courier New', monospace;
            font-size: ${result.fontSize}px;
            line-height: 1.45;
        }

        .json-viewer-error {
            color: var(--error);
            border: 1px solid rgba(${result.colors.error.Red}, ${result.colors.error.Green}, ${result.colors.error.Blue}, 0.4);
            background-color: rgba(${result.colors.error.Red}, ${result.colors.error.Green}, ${result.colors.error.Blue}, 0.12);
            border-radius: 4px;
            padding: 10px;
            white-space: pre-wrap;
        }

        .json-node {
            color: var(--fg);
        }

        .json-node[data-expanded="false"] > .json-children {
            display: none;
        }

        .json-row {
            display: flex;
            align-items: baseline;
            gap: 6px;
            min-height: 22px;
            white-space: nowrap;
        }

        .json-toggle,
        .json-toggle-placeholder {
            width: 16px;
            min-width: 16px;
            height: 16px;
            display: inline-flex;
            align-items: center;
            justify-content: center;
        }

        .json-toggle {
            border: none;
            background: transparent;
            color: var(--green);
            padding: 0;
            margin: 0;
            cursor: pointer;
        }

        .json-node[data-expanded="false"] > .json-row > .json-toggle {
            color: var(--red);
        }

        .json-toggle:hover {
            filter: brightness(1.15);
        }

        .json-toggle::before {
            content: "\\f146";
            font-family: "Font Awesome Solid", "Font Awesome", sans-serif;
            font-weight: 900;
            font-size: 12px;
            line-height: 1;
        }

        .json-node[data-expanded="false"] > .json-row > .json-toggle::before {
            content: "\\f0fe";
        }

        .json-key {
            color: var(--accent);
        }

        .json-colon,
        .json-brace {
            color: rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.85);
        }

        .json-meta {
            color: rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.55);
            margin-left: 6px;
            font-style: italic;
            font-size: ${Math.max(result.fontSize - 2, 10)}px;
        }

        .json-value-string {
            color: var(--green);
        }

        .json-value-number {
            color: var(--cyan);
        }

        .json-value-boolean {
            color: var(--yellow);
        }

        .json-value-null {
            color: var(--magenta);
        }

        #notes-swagger-edit-wrap {
            display: flex;
            flex-direction: column;
            height: 100%;
            overflow: hidden;
        }

        #notes-swagger-run-wrap {
            display: flex;
            flex-direction: column;
            height: 100%;
            overflow: hidden;
            padding: 12px;
        }

        ${getSwaggerUIStyles(result.colors, result.fontSize)}

    `;

    document.head.appendChild(style);
}

GetWindowStyle().then((result) => {
    applyWindowStyle(result);
});

EventsOn('terminalStyleUpdate', payload => {
    const result = Array.isArray(payload?.[0]) ? payload[0] : payload;
    if (result && result.colors) {
        applyWindowStyle(result);
    }
});

refreshFiles();
window.refreshFiles = refreshFiles;

function insertEditorText(text) {
    if (!text) {
        return;
    }

    elements.editor.focus();
    document.execCommand('insertText', false, text);
}

async function savePastedImageDataUrl(dataUrl, mimeType) {
    if (!state.currentFile) {
        setStatus('Select a note before pasting an image.', true);
        return;
    }

    const comma = dataUrl.indexOf(',');
    if (comma <= 0 || comma >= dataUrl.length - 1) {
        setStatus('Clipboard image format is invalid.', true);
        return;
    }

    const base64Payload = dataUrl.slice(comma + 1);
    const epoch = Math.floor(Date.now() / 1000);
    const ext = deriveImageExtension(mimeType || 'image/png');
    const paths = buildImagePaths(state.currentFile, epoch, ext);

    try {
        await SaveBinaryFile(paths.imagePath, base64Payload);

        const alt = String(epoch);
        const markdownImage = `![${alt}](${paths.imageFileName})`;
        const start = elements.editor.selectionStart;
        const end = elements.editor.selectionEnd;
        const value = elements.editor.value;

        elements.editor.value = value.slice(0, start) + markdownImage + value.slice(end);
        elements.editor.selectionStart = start + markdownImage.length;
        elements.editor.selectionEnd = start + markdownImage.length;

        setDirty(true);
        scheduleRender();
        scheduleAutoSave();
        setStatus(`Saved image ${paths.imageFileName}.`, false);
    } catch (err) {
        setStatus('Failed to save pasted image.', true);
        console.error(err);
    }
}

function handleEditorImagePaste(event) {
    if (state.viewMode !== 'editor') {
        return;
    }

    const items = event.clipboardData && event.clipboardData.items;
    if (!items) {
        return;
    }

    for (const item of items) {
        if (!item.type.startsWith('image/')) {
            continue;
        }

        event.preventDefault();

        const file = item.getAsFile();
        if (!file) {
            return;
        }

        const reader = new FileReader();
        reader.onload = async (e) => {
            const dataUrl = String(e.target.result || '');
            await savePastedImageDataUrl(dataUrl, file.type);
        };
        reader.readAsDataURL(file);

        // Only handle the first image item
        return;
    }
}

function decodeClipboardPayload(payload) {
    if (!payload || typeof payload !== 'object') {
        return { text: '', image: '' };
    }

    return {
        text: String(payload.text || ''),
        image: String(payload.image || ''),
    };
}

async function pasteFromGoClipboard() {
    try {
        const payload = await GetClipboardData();
        const { text, image } = decodeClipboardPayload(payload);

        if (image !== '') {
            const dataUrl = `data:image/png;base64,${image}`;
            await savePastedImageDataUrl(dataUrl, 'image/png');
            return;
        }

        if (text !== '') {
            insertEditorText(text);
        }
    } catch (err) {
        setStatus('Failed to paste from clipboard.', true);
        console.error(err);
    }
}

if (elements.editor) {
    elements.editor.addEventListener('input', () => {
        setDirty(true);
        scheduleRender();
        scheduleAutoSave();
    });

    elements.editor.addEventListener('paste', (event) => {
        handleEditorImagePaste(event);
    });
}

if (elements.swaggerEditor) {
    elements.swaggerEditor.addEventListener('input', () => {
        setDirty(true);
        scheduleAutoSave();
        // Revalidate JSON, refresh the JSON view, and only expose Run for docs with a swagger key.
        state.swaggerSpec = parseSwaggerSpec(elements.swaggerEditor.value);
        state.swaggerRunAvailable = hasSwaggerKey(state.swaggerSpec);
        updateTabVisibility('json');
        renderSwaggerJsonView();

        if (!state.swaggerRunAvailable && state.viewMode === 'swagger-run') {
            setViewMode('swagger-view');
            return;
        }

        if (state.swaggerRunAvailable && state.viewMode === 'swagger-run') {
            renderSwaggerUI();
        }
    });
}

let _editorSelectionBeforeContextMenu = null;

elements.editor.addEventListener('mousedown', (e) => {
    if (e.button === 2) {
        _editorSelectionBeforeContextMenu = {
            start: elements.editor.selectionStart,
            end: elements.editor.selectionEnd,
        };
    }
});

elements.editor.addEventListener('contextmenu', (e) => {
    // Restore selection that WebKit changed on right-click
    if (_editorSelectionBeforeContextMenu !== null) {
        elements.editor.selectionStart = _editorSelectionBeforeContextMenu.start;
        elements.editor.selectionEnd = _editorSelectionBeforeContextMenu.end;
        _editorSelectionBeforeContextMenu = null;
    }
    e.preventDefault();

    const menuItems = [
        createCopyMenuItem(() => getEditorSelectionText(), 'Copy'),
        {
            title: 'Paste',
            icon: CONTEXT_ICON_PASTE,
            onSelect: async () => {
                await pasteFromGoClipboard();
            },
        },
        { title: '-' },
        createFindMenuItem('Find text...'),
        createPrintMenuItem('Print...'),
        { title: '-' },
        {
            title: 'Insert checkbox',
            icon: CONTEXT_ICON_CHECKBOX,
            onSelect: () => {
                const lineStart = elements.editor.value.lastIndexOf('\n', elements.editor.selectionStart - 1) + 1;
                elements.editor.focus();
                elements.editor.selectionStart = lineStart;
                elements.editor.selectionEnd = lineStart;
                document.execCommand('insertText', false, '- [ ] ');
            },
        },
        {
            title: 'Insert code block',
            icon: CONTEXT_ICON_CODE,
            onSelect: () => {
                const selStart = elements.editor.selectionStart;
                const selected = elements.editor.value.slice(selStart, elements.editor.selectionEnd);
                elements.editor.focus();
                document.execCommand('insertText', false, '```\n' + selected + '\n```');
                // Move cursor to after the opening ``` so the user can type a language
                elements.editor.selectionStart = selStart + 3;
                elements.editor.selectionEnd = selStart + 3;
            },
        },
    ];

    const imageAtCursor = getMarkdownImageAtCursor(elements.editor.value, elements.editor.selectionStart);
    if (state.currentFile && imageAtCursor && isRelativeMarkdownImagePath(imageAtCursor.imagePath)) {
        menuItems.push(
        { title: '-' },
        {
            title: 'Delete image from disk',
            icon: CONTEXT_ICON_DELETE,
            onSelect: async () => {
                const imageDiskPath = resolveRelativeAssetPath(state.currentFile, imageAtCursor.imagePath);

                try {
                    await DeleteFile(imageDiskPath);

                    elements.editor.focus();
                    elements.editor.selectionStart = imageAtCursor.markdownStart;
                    elements.editor.selectionEnd = imageAtCursor.markdownEnd;
                    document.execCommand('insertText', false, '');
                    notifyTerminal(`Deleted image ${imageAtCursor.imagePath}.`, 'info');
                } catch (err) {
                    notifyTerminal(`Failed to delete image ${imageAtCursor.imagePath}.`, 'error');
                    console.error(err);
                }
            },
        });
    }

    showNotesLocalMenu(menuItems, e.clientX, e.clientY);
});

initRenderedNotesContextMenu(elements.preview, 'viewer');
initRenderedNotesContextMenu(elements.jupyter, 'jupyter');

elements.tabEditor.addEventListener('click', () => {
    setViewMode('editor');
});

elements.tabViewer.addEventListener('click', () => {
    setViewMode('viewer');
});

elements.tabJupyter.addEventListener('click', () => {
    setViewMode('jupyter');
    renderJupyterView();
});

elements.tabSwaggerView.addEventListener('click', () => {
    setViewMode('swagger-view');
    renderSwaggerJsonView();
});

elements.tabSwaggerEdit.addEventListener('click', () => {
    setViewMode('swagger-edit');
});

elements.tabSwaggerRun.addEventListener('click', () => {
    setViewMode('swagger-run');
    updateSwaggerLayoutMode();
    renderSwaggerUI();
});

elements.newFile.addEventListener('click', () => {
    openNewFilePrompt();
});

elements.rename.addEventListener('click', () => {
    if (!state.currentFile) {
        notifyTerminal('Select a note to rename.', 'warn');
        return;
    }
    openRenamePrompt(state.currentFile);
});

elements.modalCancel.addEventListener('click', () => {
    closeNewFilePrompt();
});

elements.modalCreate.addEventListener('click', () => {
    createNewFile();
});

elements.delete.addEventListener('click', () => {
    if (!state.currentFile) {
        notifyTerminal('Select a note to delete.', 'warn');
        return;
    }
    openDeletePrompt(state.currentFile);
});

elements.find.addEventListener('click', () => {
    openFindBar();
});

elements.deleteCancel.addEventListener('click', () => {
    closeDeletePrompt();
});

elements.deleteConfirm.addEventListener('click', () => {
    confirmDelete();
});

elements.findInput.addEventListener('input', () => {
    performFind();
});

elements.findNext.addEventListener('click', () => {
    nextMatch();
});

elements.findPrev.addEventListener('click', () => {
    prevMatch();
});

elements.findClose.addEventListener('click', () => {
    closeFindBar();
});

document.addEventListener('keydown', (event) => {
    // Block keyboard shortcuts if fullscreen image overlay is open
    if (document.getElementById('fullscreen-image-overlay')) {
        return;
    }

    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 's') {
        event.preventDefault();
        saveFile();
    }

    /*if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'f') {
        event.preventDefault();
        openFindBar();
    }*/

    /*if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'e') {
        event.preventDefault();
        setViewMode('editor');
    }

    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'v') {
        event.preventDefault();
        setViewMode('viewer');
    }*/

    /*if (event.key === 'F2' && state.currentFile && elements.modal.dataset.open === 'false') {
        event.preventDefault();
        openRenamePrompt(state.currentFile);
    }*/

    if (event.key === 'Escape' && elements.findBar.dataset.open === 'true') {
        event.preventDefault();
        closeFindBar();
    } else if (event.key === 'Escape' && elements.modal.dataset.open === 'true') {
        event.preventDefault();
        closeNewFilePrompt();
    } else if (event.key === 'Escape' && elements.deleteModal.dataset.open === 'true') {
        event.preventDefault();
        closeDeletePrompt();
    }
});

elements.modalInput.addEventListener('keydown', (event) => {
    if (event.key === 'Enter') {
        event.preventDefault();
        createNewFile();
    }
});

elements.findInput.addEventListener('keydown', (event) => {
    if (event.key === 'Enter') {
        event.preventDefault();
        if (event.shiftKey) {
            prevMatch();
        } else {
            nextMatch();
        }
    }
});

setViewMode('viewer');

if (typeof ResizeObserver !== 'undefined' && elements.swaggerRunWrap) {
    const swaggerPaneResizeObserver = new ResizeObserver(() => {
        updateSwaggerLayoutMode();
    });
    swaggerPaneResizeObserver.observe(elements.swaggerRunWrap);
} else {
    window.addEventListener('resize', () => {
        updateSwaggerLayoutMode();
    });
}
