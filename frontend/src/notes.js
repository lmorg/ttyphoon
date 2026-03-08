import { GetWindowStyle, GetMarkdown, GetParameters, GetImage, SendIpc, GetCustomRegexp, WindowShow, WindowHide } from '../wailsjs/go/main/WApp';
import { EventsOn, BrowserOpenURL } from '../wailsjs/runtime/runtime';

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
                <button id="notes-tab-viewer" type="button" role="tab" aria-selected="true">View</button>
                <button id="notes-tab-editor" type="button" role="tab" aria-selected="false">Edit</button>
                <button id="notes-tab-jupyter" type="button" role="tab" aria-selected="false">Run</button>
                <button id="notes-new" type="button">New</button>
                <button id="notes-rename" type="button" title="Rename current note">Rename</button>
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
                <div id="notes-jupyter-wrap" class="markdown-body" role="tabpanel">
                    <div id="notes-jupyter"></div>
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
    tabEditor: document.getElementById('notes-tab-editor'),
    tabViewer: document.getElementById('notes-tab-viewer'),
    tabJupyter: document.getElementById('notes-tab-jupyter'),
    editorWrap: document.getElementById('notes-editor-wrap'),
    previewWrap: document.getElementById('notes-preview-wrap'),
    jupyterWrap: document.getElementById('notes-jupyter-wrap'),
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
    findClose: document.getElementById('notes-find-close')
};

const state = {
    files: [],
    currentFile: '',
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
    jupyterBlockCounter: 0
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

    // Make checkboxes interactive
    setupInteractiveCheckboxes();

    // Apply custom regex hyperlinks
    autoHyperlink();

    // Re-apply find highlights if find bar is open and in viewer mode
    if (elements.findBar.dataset.open === 'true' && state.findQuery && state.viewMode === 'viewer') {
        setTimeout(() => {
            performFind();
        }, 0);
    }
}

function setupInteractiveCheckboxes() {
    const checkboxes = elements.preview.querySelectorAll('input[type="checkbox"]');
    
    checkboxes.forEach((checkbox, index) => {
        // Remove disabled attribute to make clickable
        checkbox.removeAttribute('disabled');
        
        checkbox.addEventListener('change', (e) => {
            toggleCheckboxInMarkdown(index, e.target.checked);
        });
    });
}

async function autoHyperlink() {
    const customRegexps = await GetCustomRegexp?.() || [];
    
    if (!customRegexps || customRegexps.length === 0) {
        return;
    }

    for (const custom of customRegexps) {
        if (!custom.pattern || !custom.link) {
            continue;
        }

        try {
            const regex = new RegExp(custom.pattern, 'g');
            
            // Walk through all text nodes in the preview
            const walker = document.createTreeWalker(
                elements.preview,
                NodeFilter.SHOW_TEXT,
                null,
                false
            );

            const nodesToProcess = [];
            let node;
            while ((node = walker.nextNode())) {
                // Skip if inside an <a> tag
                let parent = node.parentNode;
                let insideLink = false;
                while (parent) {
                    if (parent.tagName === 'A') {
                        insideLink = true;
                        break;
                    }
                    parent = parent.parentNode;
                }
                
                if (!insideLink && regex.test(node.textContent)) {
                    regex.lastIndex = 0; // Reset regex state
                    nodesToProcess.push(node);
                }
            }

            // Process matches and create hyperlinks
            nodesToProcess.forEach((textNode) => {
                const text = textNode.textContent;
                const parts = [];
                let lastIndex = 0;
                let match;
                
                regex.lastIndex = 0; // Reset for this text node
                while ((match = regex.exec(text)) !== null) {
                    // Add text before match
                    if (match.index > lastIndex) {
                        parts.push(document.createTextNode(text.substring(lastIndex, match.index)));
                    }

                    // Create hyperlink
                    const matchedText = match[0];
                    const link = matchedText.replace(new RegExp(custom.pattern), custom.link);
                    const a = document.createElement('a');
                    a.href = link;
                    a.textContent = matchedText;
                    a.addEventListener('click', (e) => {
                        e.preventDefault();
                        BrowserOpenURL(a.href);
                    });
                    parts.push(a);

                    lastIndex = regex.lastIndex;
                }

                // Add remaining text
                if (lastIndex < text.length) {
                    parts.push(document.createTextNode(text.substring(lastIndex)));
                }

                // Replace original text node with parts
                if (parts.length > 0) {
                    const parent = textNode.parentNode;
                    parts.forEach((part) => {
                        parent.insertBefore(part, textNode);
                    });
                    parent.removeChild(textNode);
                }
            });
        } catch (err) {
            console.error('Error processing custom regex:', custom.pattern, err);
        }
    }
}

function toggleCheckboxInMarkdown(checkboxIndex, isChecked) {
    const lines = elements.editor.value.split('\n');
    let currentCheckboxIndex = 0;
    let modified = false;

    for (let i = 0; i < lines.length; i++) {
        const line = lines[i];
        // Match markdown task list items: - [ ] or - [x] or - [X]
        const checkboxMatch = line.match(/^(\s*[-*+]\s+)\[([ xX])\](.*)$/);
        
        if (checkboxMatch) {
            if (currentCheckboxIndex === checkboxIndex) {
                // Toggle the checkbox
                const newState = isChecked ? 'x' : ' ';
                lines[i] = `${checkboxMatch[1]}[${newState}]${checkboxMatch[3]}`;
                modified = true;
                break;
            }
            currentCheckboxIndex++;
        }
    }

    if (modified) {
        elements.editor.value = lines.join('\n');
        saveFile();
        // Don't call renderMarkdown() here as it would reset checkbox focus
        // The file will be saved and the change is already reflected in the checkbox state
    }
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
    state.viewMode = mode === 'viewer' ? 'viewer' : (mode === 'jupyter' ? 'jupyter' : 'editor');
    const isEditor = state.viewMode === 'editor';
    const isJupyter = state.viewMode === 'jupyter';
    const isViewer = state.viewMode === 'viewer';
    
    elements.tabEditor.setAttribute('aria-selected', isEditor ? 'true' : 'false');
    elements.tabViewer.setAttribute('aria-selected', isViewer ? 'true' : 'false');
    elements.tabJupyter.setAttribute('aria-selected', isJupyter ? 'true' : 'false');
    
    elements.editorWrap.dataset.active = isEditor ? 'true' : 'false';
    elements.previewWrap.dataset.active = isViewer ? 'true' : 'false';
    elements.jupyterWrap.dataset.active = isJupyter ? 'true' : 'false';
    
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
    
    // Apply syntax highlighting to code blocks before conversion
    elements.jupyter.querySelectorAll('pre code').forEach((block) => {
        hljs.highlightElement(block);
    });
    
    // Handle images with Wails URLs
    const rxWailsUrl = /^(wails:\/\/wails\/|http:\/\/localhost:[0-9]+\/|wails:\/\/wails.localhost:[0-9]+\/)/;
    
    elements.jupyter.querySelectorAll('img').forEach((img) => {
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
    
    // Handle external links
    let rxBookmark = /^(wails:\/\/wails\/|http:\/\/localhost:[0-9]+\/|wails:\/\/wails.localhost:[0-9]+\/)#/;
    
    elements.jupyter.querySelectorAll('a').forEach(a => {
        if (!a.href.match(rxWailsUrl)) {
            a.addEventListener('click', (e) => {
                e.preventDefault();
                BrowserOpenURL(a.href);
            });
        }
    });
    
    setupInteractiveCheckboxes();
    autoHyperlink();
    convertToJupyterCodeBlocks();
}

function convertToJupyterCodeBlocks() {
    const codeBlocks = elements.jupyter.querySelectorAll('pre');
    
    codeBlocks.forEach((pre) => {
        const code = pre.querySelector('code');
        if (!code) return;
        
        const langClass = Array.from(code.classList).find(cls => cls.startsWith('language-'));
        if (!langClass) return;
        
        const language = langClass.replace('language-', '');
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
            const getDescriptionsFn = window.go?.main?.WApp?.GetLanguageDescriptions;
            if (getDescriptionsFn) {
                try {
                    const descriptions = await getDescriptionsFn(language);
                    runtimeDropdown.innerHTML = '';
                    if (descriptions && descriptions.length > 0) {
                        descriptions.forEach((desc) => {
                            const option = document.createElement('option');
                            option.value = desc;
                            option.textContent = desc;
                            runtimeDropdown.appendChild(option);
                        });
                        // Set runtime to the first description in the list
                        state.jupyterCodeBlocks[blockId].runtime = descriptions[0];
                    } else {
                        const option = document.createElement('option');
                        option.value = language;
                        option.textContent = language;
                        runtimeDropdown.appendChild(option);
                        state.jupyterCodeBlocks[blockId].runtime = language;
                    }
                } catch (err) {
                    console.error('Error fetching language descriptions:', err);
                    const option = document.createElement('option');
                    option.value = language;
                    option.textContent = language;
                    runtimeDropdown.appendChild(option);
                    state.jupyterCodeBlocks[blockId].runtime = language;
                }
            }
        })();
        
        runtimeDropdown.addEventListener('change', () => {
            state.jupyterCodeBlocks[blockId].runtime = runtimeDropdown.value;
        });
        
        toolbar.appendChild(runNotesBtn);
        toolbar.appendChild(runTerminalBtn);
        toolbar.appendChild(runtimeDropdown);
        
        const editableCode = document.createElement('textarea');
        editableCode.className = 'jupyter-code-editable';
        editableCode.dataset.language = language;
        editableCode.value = content;
        editableCode.spellcheck = false;
        
        // Auto-resize textarea to fit content
        const autoResize = () => {
            editableCode.style.height = 'auto';
            editableCode.style.height = editableCode.scrollHeight + 'px';
        };
        editableCode.addEventListener('input', autoResize);
        // Set initial height
        setTimeout(autoResize, 0);
        
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
        wrapper.appendChild(editableCode);
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
    
    const runNoteFn = window.go?.main?.WApp?.RunNote;
    if (runNoteFn) {
        try {
            await runNoteFn(blockId, block.currentContent, block.runtime);
        } catch (err) {
            console.error('Error running code:', err);
            const outputBlock = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-output`);
            if (outputBlock) {
                outputBlock.textContent = `Error: ${err.message}`;
            }
        }
    }
}

async function runCodeBlockInTerminal(blockId) {
    const block = state.jupyterCodeBlocks[blockId];
    if (!block) return;
    
    const editableElement = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-code-editable`);
    if (editableElement) {
        block.currentContent = editableElement.value;
    }
    
    const sendIpcFn = SendIpc;
    if (sendIpcFn) {
        try {
            await sendIpcFn('noteRunTerminal', {
                blockId: blockId,
                code: block.currentContent,
                language: block.language
            });
        } catch (err) {
            console.error('Error sending to terminal:', err);
        }
    }
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

async function loadFile(file) {
    if (!file) {
        return;
    }

    try {
        const doc = await GetMarkdown(file);
        state.currentFile = file;
        elements.editor.value = doc || '';
        renderMarkdown();
        renderJupyterView();
        setDirty(false);
        renderFileList();
        
        // Close find bar when loading a new file
        if (elements.findBar.dataset.open === 'true') {
            closeFindBar();
        }
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

function clearHighlights() {
    // Clear highlights in viewer
    const highlights = elements.preview.querySelectorAll('.find-highlight');
    highlights.forEach((el) => {
        const parent = el.parentNode;
        parent.replaceChild(document.createTextNode(el.textContent), el);
        parent.normalize();
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
        findInViewer();
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

function findInViewer() {
    const query = state.findQuery;
    const walker = document.createTreeWalker(
        elements.preview,
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
        const prevActive = elements.preview.querySelector('.find-highlight-active');
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

async function createNewFile() {
    let fileName = normalizeNoteName(elements.modalInput.value);
    if (fileName === '') {
        setStatus('File name cannot be empty.', true);
        return;
    }

    // Handle rename operation
    if (state.renamingFile) {
        const renameFn = getWailsFunction('RenameFile');
        if (!renameFn) {
            setStatus('RenameFile is not available.', true);
            return;
        }

        try {
            await renameFn(state.renamingFile, fileName);
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

async function createAndOpenFile(filename, contents) {
    const fileName = normalizeNotePath(filename);
    if (fileName === '') {
        setStatus('File name cannot be empty.', true);
        return;
    }

    const saveFn = getWailsFunction('SaveFile');
    if (!saveFn) {
        setStatus('SaveFile is not available.', true);
        return;
    }

    try {
        await saveFn(fileName, contents || '');
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

window.createAndOpenFile = createAndOpenFile;

EventsOn("notesCreateAndOpen", params => {
    createAndOpenFile(params.filename, params.contents);
    WindowShow();
});

EventsOn("updateTitle", newTitle => {
    elements.title.innerText = "Notes: " + newTitle;
});

EventsOn("noteRun", (data) => {
    const { blockId, output } = data;
    const outputBlock = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-output`);
    if (outputBlock) {
        const currentText = outputBlock.textContent;
        outputBlock.textContent = currentText ? `${currentText}\n${output}` : output;
    }
});

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
            opacity: 0.2;
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
            grid-template-columns: 25% 1fr;
            height: 100vh;
            overflow: hidden;
            color: var(--fg);
            background: var(--bg);
        }

        #notes-sidebar {
            display: flex;
            flex-direction: column;
            /*border-right: 2px solid var(--fg); */
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

        #notes-rename:hover {
            border-color: var(--yellow) !important;
            color: var(--yellow) !important;
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
            border-color: var(--selection);
            /*color: var(--selection);*/
        }

        #notes-delete-modal-actions button:hover {
            border-color: var(--selection);
            /*color: var(--selection);*/
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
            /*padding-bottom: 2px;*/
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
            border-color: var(--fg);
            /*color: var(--bg);
            background: var(--fg);*/
            border-bottom: 5px;
        }

        #notes-tabs button:hover {
            border-color: var(--selection);
        }

        #notes-new:hover {
            border-color: var(--green) !important;
            color: var(--green) !important;
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
        #notes-preview-wrap,
        #notes-jupyter-wrap {
            position: absolute;
            inset: 0;
            display: none;
            min-height: 0;
        }

        #notes-editor-wrap[data-active="true"],
        #notes-preview-wrap[data-active="true"],
        #notes-jupyter-wrap[data-active="true"] {
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

        #notes-preview {
            font-size: ${result.fontSize}px;
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
            border-left: 2px solid var(--green);
            margin: 0;
            padding: 10px 10px 10px 20px;
            overflow-x: auto;
        }

        .markdown-body blockquote {
            border: 0;
            border-left: 2px solid var(--magenta);
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

        .markdown-body input[type="checkbox"] {
            appearance: none;
            -webkit-appearance: none;
            -moz-appearance: none;
            cursor: pointer;
            margin-right: 6px;
            width: ${result.fontSize}px;
            height: ${result.fontSize}px;
            border: 2px solid var(--red);
            background: transparent;
            position: relative;
            vertical-align: middle;
            flex-shrink: 0;
        }

        .markdown-body input[type="checkbox"]:hover {
            border-color: var(--accent);
        }

        .markdown-body input[type="checkbox"]:checked:hover {
            border-color: var(--accent);
            background: var(--accent);
        }

        .markdown-body input[type="checkbox"]:checked {
            background: var(--green);
            border-color: var(--green);
        }

        .markdown-body input[type="checkbox"]:checked::after {
            content: '✓';
            position: absolute;
            color: var(--bg);
            font-size: ${result.fontSize}px;
            font-weight: bold;
            left: 50%;
            top: 50%;
            transform: translate(-50%, -50%);
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

        #notes-find-bar {
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
            border-radius: 0;
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
        #notes-jupyter-wrap {
            overflow-y: auto;
            padding: 16px;
        }

        #notes-jupyter-wrap pre {
            border-left: 0;
            padding-left: 10px;
        }

        .jupyter-code-block {
            margin: 16px 0;
            border: 2px solid var(--fg);
            border-radius: 4px;
            overflow: hidden;
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
            border-radius: 2px;
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

        .jupyter-code-editable {
            width: 100%;
            margin: 0;
            padding: 12px;
            background-color: var(--bg);
            border: none;
            color: var(--fg);
            font-family: monospace;
            font-size: ${result.fontSize}px;
            line-height: 1.5;
            overflow: hidden;
            white-space: pre;
            outline: none;
            resize: none;
            box-sizing: border-box;
        }

        .jupyter-code-editable:focus {
            outline: none;
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

    `;

    document.head.appendChild(style);
}

GetWindowStyle().then((result) => {
    applyWindowStyle(result);
});

GetParameters().then((params) => {
    if (params.filename != '' && params.content != '') {
        setTimeout(function() {
            window.createAndOpenFile(params.filename, params.content);
        }, 1);
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

elements.tabJupyter.addEventListener('click', () => {
    setViewMode('jupyter');
    renderJupyterView();
});

elements.newFile.addEventListener('click', () => {
    openNewFilePrompt();
});

elements.rename.addEventListener('click', () => {
    if (!state.currentFile) {
        setStatus('Select a note to rename.', true);
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
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 's') {
        event.preventDefault();
        saveFile();
    }

    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'f') {
        event.preventDefault();
        openFindBar();
    }

    /*if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'e') {
        event.preventDefault();
        setViewMode('editor');
    }

    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'v') {
        event.preventDefault();
        setViewMode('viewer');
    }*/

    if (event.key === 'F2' && state.currentFile && elements.modal.dataset.open === 'false') {
        event.preventDefault();
        openRenamePrompt(state.currentFile);
    }

    if (event.key === 'Tab') {
        event.preventDefault();
        SendIpc("focus", {});
    }

    if (event.key === 'Escape' && elements.findBar.dataset.open === 'true') {
        event.preventDefault();
        closeFindBar();
    } else if (event.key === 'Escape' && elements.modal.dataset.open === 'true') {
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
