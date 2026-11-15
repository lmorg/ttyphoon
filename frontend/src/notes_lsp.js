// Notes LSP and typos integration for the main editor document.
// notes.js injects its shared state and helpers via initNotesLsp() so this
// module does not import the side-effecting notes.js entry point.
import {
    NotesLspApplyCodeAction, NotesLspChangeDocument, NotesLspCloseDocument, NotesLspCodeActions,
    NotesLspCodeLens, NotesLspCompletion, NotesLspDefinition, NotesLspDocumentSymbols,
    NotesLspExecuteCodeLens, NotesLspFormat, NotesLspFormatRange, NotesLspHover,
    NotesLspInlayHints, NotesLspOpenDocument, NotesLspPrepareRename, NotesLspReferences,
    NotesLspRename, NotesLspSemanticTokens, NotesLspSemanticTokensDelta, NotesLspSignatureHelp,
    NotesLspWorkspaceSymbols,
    NotesTyposChangeDocument, NotesTyposCloseDocument, NotesTyposOpenDocument,
    ResolveNotesLspLanguage,
} from './wails-bindings.js';
import { EventsOn } from './wails-runtime.js';
import { showLocalMenu } from './popup_menu';
import { marked } from 'marked';
import { processMarkdownContainer } from './markdown-utils.js';

let
    CONTEXT_ICON_CODE, CONTEXT_ICON_FIND, elements, jupyterSpellCheckHandles, mainDocument,
    notesMutationAdapter, state, addHistoryPosition, applyMonacoLineIndentationForSelection,
    applyMonacoTabIndent, currentEditorToken, escapeHtml, fileUriToPath,
    getMainEditorSelectionRange, getMainEditorValue, isMonacoActive, isSvgFile, jumpEditorToOffset,
    loadFile, normalizePathForMatch, notifyTerminal, openTextPrompt, replaceMainDocumentText,
    resolveNotesFileFromAbsolutePath, runProjectFindSearch, setMainEditorSelectionRange,
    setMainEditorValue, setProjectReferencesResults, setToolsPanelCollapsed, setToolsTab,
    shouldMonacoTabIndent, showNotesLocalMenu, updateFindOptionButtons, getMonacoMainEditor,
    getNotesSpellCheckHandle, getActiveNotesDocument, isNotesDocumentOpen, getFocusedEditorHost;

export function initNotesLsp(deps) {
    ({
        CONTEXT_ICON_CODE, CONTEXT_ICON_FIND, elements, jupyterSpellCheckHandles, mainDocument,
        notesMutationAdapter, state, addHistoryPosition, applyMonacoLineIndentationForSelection,
        applyMonacoTabIndent, currentEditorToken, escapeHtml, fileUriToPath,
        getMainEditorSelectionRange, getMainEditorValue, isMonacoActive, isSvgFile,
        jumpEditorToOffset, loadFile, normalizePathForMatch, notifyTerminal, openTextPrompt,
        replaceMainDocumentText, resolveNotesFileFromAbsolutePath, runProjectFindSearch,
        setMainEditorSelectionRange, setMainEditorValue, setProjectReferencesResults,
        setToolsPanelCollapsed, setToolsTab, shouldMonacoTabIndent, showNotesLocalMenu,
        updateFindOptionButtons, getMonacoMainEditor, getNotesSpellCheckHandle,
        getActiveNotesDocument, isNotesDocumentOpen, getFocusedEditorHost,
    } = deps);

    EventsOn('notesLspDiagnostics', payload => {
        const data = Array.isArray(payload?.[0]) ? payload[0] : payload;
        if (!data || typeof data.uri !== 'string') return;

        lspDiagnosticsStore.set(data.uri, Array.isArray(data.diagnostics) ? data.diagnostics : []);
        if (data.uri === state.currentFileUri) {
            // While the user is actively typing, avoid repainting diagnostics from an
            // older editor snapshot; wait for a short idle window.
            if (state.lspChangeTimer || (Date.now() - state.lastEditorInputAt) < LSP_DIAGNOSTIC_RENDER_IDLE_MS) {
                return;
            }
            renderLspEditorDecorations();
        }
    });

    EventsOn('notesTyposDiagnostics', payload => {
        const data = Array.isArray(payload?.[0]) ? payload[0] : payload;
        if (!data || typeof data.uri !== 'string') return;
        typosDiagnosticsStore.set(data.uri, Array.isArray(data.diagnostics) ? data.diagnostics : []);
        routeTyposDiagnostics(data.uri, Array.isArray(data.diagnostics) ? data.diagnostics : []);
    });
}

export const LSP_CHANGE_DEBOUNCE_MS = 200;

// Captures the document shown when an LSP request starts. The returned check
// tells whether it is still shown when the response arrives, so a response is
// never painted or applied over another document (ADR 0100 decision 5).
function captureShownDocument() {
    const doc = getActiveNotesDocument();
    return () => doc === getActiveNotesDocument();
}

const LSP_DIAGNOSTIC_RENDER_IDLE_MS = 220;

const LSP_HOVER_DEBOUNCE_MS = 250;

// Monaco model URI → the LSP document it shows ({ filePath, fileUri }).
// Providers are registered per language, so they also run for other models
// of that language, such as the Git diff editor's; only registered models
// resolve to a document.
const lspModelDocuments = new Map();

function lspDocumentForModel(model) {
    const uri = model?.uri ? String(model.uri.toString()) : '';
    return (uri && lspModelDocuments.get(uri)) || null;
}

export function isCurrentFileLspEligible() {
    if (isSvgFile(state.currentFile)) {
        return false;
    }
    return state.currentFileType === 'code' || state.currentFileType === 'json' || state.currentFileType === 'markdown' || state.currentFileType === 'html';
}

// Returns the active Jupyter code block LSP context, or null when the main editor is active.
function getActiveLspTarget() {
    if (!state.lspActiveBlockId || !state.lspActiveBlockEditor) {
        return null;
    }
    const block = state.jupyterCodeBlocks[state.lspActiveBlockId];
    if (!block || !block.lspMode || !block.lspFilePath) {
        return null;
    }
    return { filePath: block.lspFilePath, editor: state.lspActiveBlockEditor };
}

function clearLspChangeTimer() {
    if (state.lspChangeTimer) {
        clearTimeout(state.lspChangeTimer);
        state.lspChangeTimer = null;
    }
}

function clearLspHoverTimer() {
    if (state.lspHoverTimer) {
        clearTimeout(state.lspHoverTimer);
        state.lspHoverTimer = null;
    }
}

export function hideLspHoverTooltip() {
    if (lspHoverTooltipEl) {
        lspHoverTooltipEl.style.display = 'none';
        lspHoverTooltipEl.innerHTML = '';
    }
}

export function hideLspCompletion() {
    state.lspCompletionVisible = false;
    state.lspCompletionItems = [];
    state.lspCompletionIndex = 0;
    if (lspCompletionEl) {
        lspCompletionEl.style.display = 'none';
        lspCompletionEl.innerHTML = '';
    }
}

export function closeOpenLspTooltips() {
    let closedAny = false;

    if (lspTooltipEl && lspTooltipEl.style.display !== 'none' && lspTooltipEl.style.display !== '') {
        lspTooltipEl.style.display = 'none';
        closedAny = true;
    }

    if (lspHoverTooltipEl && lspHoverTooltipEl.style.display !== 'none' && lspHoverTooltipEl.style.display !== '') {
        hideLspHoverTooltip();
        closedAny = true;
    }

    if (closedAny) {
        return true;
    }

    if (lspCompletionEl && lspCompletionEl.style.display !== 'none' && lspCompletionEl.style.display !== '') {
        hideLspCompletion();
        closedAny = true;
    }

    return closedAny;
}

function lspCompletionKindMeta(kind) {
    switch (Number(kind || 0)) {
    case 2:
        return { iconClass: 'codicon codicon-symbol-method', badge: 'method', title: 'Method' };
    case 3:
        return { iconClass: 'codicon codicon-symbol-function', badge: 'function', title: 'Function' };
    case 4:
        return { iconClass: 'codicon codicon-symbol-constructor', badge: 'constructor', title: 'Constructor' };
    case 5:
        return { iconClass: 'codicon codicon-symbol-field', badge: 'field', title: 'Field' };
    case 6:
        return { iconClass: 'codicon codicon-symbol-variable', badge: 'variable', title: 'Variable' };
    case 7:
        return { iconClass: 'codicon codicon-symbol-class', badge: 'class', title: 'Class' };
    case 8:
        return { iconClass: 'codicon codicon-symbol-interface', badge: 'interface', title: 'Interface' };
    case 9:
        return { iconClass: 'codicon codicon-symbol-module', badge: 'module', title: 'Module' };
    case 10:
        return { iconClass: 'codicon codicon-symbol-property', badge: 'property', title: 'Property' };
    case 11:
        return { iconClass: 'codicon codicon-symbol-unit', badge: 'unit', title: 'Unit' };
    case 12:
        return { iconClass: 'codicon codicon-symbol-value', badge: 'value', title: 'Value' };
    case 13:
        return { iconClass: 'codicon codicon-symbol-enum', badge: 'enum', title: 'Enum' };
    case 14:
        return { iconClass: 'codicon codicon-symbol-keyword', badge: 'keyword', title: 'Keyword' };
    case 15:
        return { iconClass: 'codicon codicon-symbol-snippet', badge: 'snippet', title: 'Snippet' };
    case 17:
        return { iconClass: 'codicon codicon-symbol-file', badge: 'file', title: 'File' };
    case 18:
        return { iconClass: 'codicon codicon-symbol-reference', badge: 'reference', title: 'Reference' };
    case 19:
        return { iconClass: 'codicon codicon-folder', badge: 'folder', title: 'Folder' };
    case 21:
        return { iconClass: 'codicon codicon-symbol-constant', badge: 'constant', title: 'Constant' };
    case 22:
        return { iconClass: 'codicon codicon-symbol-struct', badge: 'struct', title: 'Struct' };
    case 24:
        return { iconClass: 'codicon codicon-symbol-event', badge: 'event', title: 'Event' };
    case 25:
        return { iconClass: 'codicon codicon-symbol-operator', badge: 'operator', title: 'Operator' };
    case 26:
        return { iconClass: 'codicon codicon-symbol-type-parameter', badge: 'type', title: 'Type parameter' };
    default:
        return { iconClass: 'codicon codicon-symbol-misc', badge: '', title: 'Symbol' };
    }
}

function replaceCurrentIdentifierWithCompletion(text) {
    const target = getActiveLspTarget();

    if (!target && isMonacoActive()) {
        const source = getMonacoMainEditor().getValue() || '';
        const selection = getMonacoMainEditor().getSelectionOffsets();
        const cursor = selection.start || 0;
        const left = source.slice(0, cursor);
        const match = left.match(/[A-Za-z0-9_-]+$/);
        const start = match ? cursor - match[0].length : cursor;
        const insertText = String(text || '');

        hideLspCompletion();
        getMonacoMainEditor().replaceRange(start, cursor, insertText, 'lsp-completion');
        const next = start + insertText.length;
        getMonacoMainEditor().setSelectionOffsets(next, next);
        getMonacoMainEditor().focus();
        return;
    }

    const editor = target?.editor;
    if (!editor) {
        return;
    }

    const source = editor.value || '';
    const cursor = editor.selectionStart || 0;
    const left = source.slice(0, cursor);
    const match = left.match(/[A-Za-z0-9_-]+$/);
    const start = match ? cursor - match[0].length : cursor;
    // Hide completion BEFORE dispatching input event so the input handler doesn't try to filter.
    hideLspCompletion();
    notesMutationAdapter.replaceRange(editor, {
        start,
        end: cursor,
        text,
        source: 'lsp-completion',
        label: 'Apply completion item',
        emit: true,
    });
}

export function commitActiveLspCompletion() {
    if (!state.lspCompletionVisible || state.lspCompletionItems.length === 0) {
        return false;
    }

    const index = Math.max(0, Math.min(state.lspCompletionIndex, state.lspCompletionItems.length - 1));
    const item = state.lspCompletionItems[index];
    if (!item) {
        return false;
    }

    replaceCurrentIdentifierWithCompletion(item.insertText || item.label);
    hideLspCompletion();
    return true;
}

function renderLspCompletionPopup() {
    if (!lspCompletionEl) {
        return;
    }

    if (!state.lspCompletionVisible || state.lspCompletionItems.length === 0) {
        hideLspCompletion();
        return;
    }

    const activeIndex = Math.max(0, Math.min(state.lspCompletionIndex, state.lspCompletionItems.length - 1));
    state.lspCompletionIndex = activeIndex;

    const fragment = document.createDocumentFragment();
    state.lspCompletionItems.forEach((item, idx) => {
        const row = document.createElement('div');
        row.className = 'tty-menu-row notes-lsp-completion-item';
        row.dataset.active = idx === activeIndex ? 'true' : 'false';
        row.dataset.deprecated = item.deprecated === true ? 'true' : 'false';
        row.classList.toggle('is-active', idx === activeIndex);
        row.classList.toggle('is-deprecated', item.deprecated === true);

        const kind = lspCompletionKindMeta(item.kind);

        const icon = document.createElement('span');
        icon.className = `tty-menu-row-icon notes-lsp-completion-icon ${kind.iconClass}`;
        icon.title = kind.title;
        icon.setAttribute('aria-label', kind.title);
        row.appendChild(icon);

        const label = document.createElement('span');
        label.className = 'tty-menu-row-label notes-lsp-completion-label';
        label.textContent = String(item.label || '');
        row.appendChild(label);

        if (item.detail) {
            const detail = document.createElement('span');
            detail.className = 'notes-lsp-completion-detail';
            detail.textContent = String(item.detail);
            row.appendChild(detail);
        }

        if (kind.badge) {
            const badge = document.createElement('span');
            badge.className = 'notes-lsp-completion-kind';
            badge.textContent = kind.badge;
            badge.title = kind.title;
            row.appendChild(badge);
        }

        row.addEventListener('mousedown', (event) => {
            event.preventDefault();
            state.lspCompletionIndex = idx;
            commitActiveLspCompletion();
        });

        fragment.appendChild(row);
    });

    lspCompletionEl.innerHTML = '';
    lspCompletionEl.appendChild(fragment);
    lspCompletionEl.style.display = 'block';

    const anchor = (isMonacoActive() && !state.lspActiveBlockId)
        ? getMonacoCompletionMenuAnchor()
        : getLspAnchorViewportPoint();
    const rawX = anchor.x;
    const rawY = anchor.y;
    const x = Math.min(rawX + 14, window.innerWidth - lspCompletionEl.offsetWidth - 8);
    const y = Math.min(rawY + 18, window.innerHeight - lspCompletionEl.offsetHeight - 8);
    lspCompletionEl.style.left = `${Math.max(8, x)}px`;
    lspCompletionEl.style.top = `${Math.max(8, y)}px`;
}

export function moveLspCompletionSelection(delta) {
    if (!state.lspCompletionVisible || state.lspCompletionItems.length === 0) {
        return false;
    }

    const count = state.lspCompletionItems.length;
    const current = Math.max(0, Math.min(state.lspCompletionIndex, count - 1));
    const next = (current + delta + count) % count;
    state.lspCompletionIndex = next;
    renderLspCompletionPopup();

    if (lspCompletionEl) {
        const activeRow = lspCompletionEl.querySelector('.notes-lsp-completion-item.is-active');
        if (activeRow && typeof activeRow.scrollIntoView === 'function') {
            activeRow.scrollIntoView({ block: 'nearest' });
        }
    }

    return true;
}

export async function requestLspCompletion(triggerKind = 1, triggerChar = '') {
    const blockTarget = getActiveLspTarget();
    if (!blockTarget && (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible())) {
        hideLspCompletion();
        return;
    }

    const completionFile = blockTarget ? blockTarget.filePath : state.currentFile;
    const completionSource = blockTarget ? (blockTarget.editor.value || '') : getMainEditorValue();
    const completionSelection = blockTarget
        ? { start: blockTarget.editor.selectionStart || 0 }
        : getMainEditorSelectionRange();
    const pos = offsetToLspPosition(completionSource, completionSelection.start || 0);
    const isShown = captureShownDocument();
    try {
        const items = await NotesLspCompletion(completionFile, pos.line, pos.character, triggerKind, triggerChar);
        if (!isShown()) {
            return;
        }
        const list = Array.isArray(items) ? items : [];
        if (list.length === 0) {
            hideLspCompletion();
            return;
        }

        state.lspCompletionItems = list;
        state.lspCompletionIndex = 0;
        state.lspCompletionVisible = true;
        renderLspCompletionPopup();
    } catch {
        hideLspCompletion();
    }
}

function getMonacoCompletionMenuAnchor() {
    const caretPoint = getMonacoMainEditor()?.getCursorViewportPoint?.();
    if (caretPoint && Number.isFinite(caretPoint.x) && Number.isFinite(caretPoint.y)) {
        return caretPoint;
    }

    const rect = getFocusedEditorHost()?.getBoundingClientRect?.();
    if (rect) {
        return {
            x: rect.left + 24,
            y: rect.top + 24,
        };
    }

    return getLspAnchorViewportPoint();
}

export function handleMonacoCompletionRequest(payload = {}) {
    if (!isMonacoActive() || state.lspActiveBlockId) {
        return false;
    }

    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
        return false;
    }

    const source = String(payload.source || '');
    const key = String(payload.key || '');
    const ctrlKey = payload.ctrlKey === true;
    const metaKey = payload.metaKey === true;
    const altKey = payload.altKey === true;
    const shiftKey = payload.shiftKey === true;

    if (source === 'keydown' && !ctrlKey && !metaKey && !altKey && key === 'Tab') {
        const selection = getMonacoMainEditor()?.getSelectionOffsets?.();
        const start = Number(selection?.start) || 0;
        const end = Number(selection?.end) || start;
        if (end > start) {
            void applyMonacoLineIndentationForSelection(shiftKey);
            return true;
        }
    }

    if (source === 'keydown' && state.lspCompletionVisible) {
        if (key === 'Escape') {
            hideLspCompletion();
            return true;
        }

        if (!ctrlKey && !metaKey && !altKey && key === 'ArrowDown') {
            return moveLspCompletionSelection(1);
        }

        if (!ctrlKey && !metaKey && !altKey && key === 'ArrowUp') {
            return moveLspCompletionSelection(-1);
        }

        if (!ctrlKey && !metaKey && !altKey && (key === 'Enter' || key === 'Tab')) {
            return commitActiveLspCompletion();
        }

        if (!ctrlKey && !metaKey && !altKey && (key === 'Backspace' || key === 'Delete')) {
            requestAnimationFrame(() => {
                void requestLspCompletionAfterSync(getMainEditorValue(), '', 1);
            });
        }
    }

    if (source === 'type' && (key === '.' || key === ':' || key === '>')) {
        void requestLspCompletionAfterSync(getMainEditorValue(), key, 2);
        return true;
    }

    if (source === 'type' && state.lspCompletionVisible) {
        if (/^[-_A-Za-z0-9]$/.test(key)) {
            return false;
        }

        if (!/^\s$/.test(key)) {
            hideLspCompletion();
        }
    }

    if (source === 'keydown') {
        const isTab = key === 'Tab' && !ctrlKey && !metaKey && !altKey;
        const isCtrlSpace = (key === ' ' || key === 'Space' || key === 'Spacebar')
            && (ctrlKey || metaKey)
            && !altKey;

        if (isTab && shouldMonacoTabIndent()) {
            void applyMonacoTabIndent();
            return true;
        }

        if (isTab || isCtrlSpace) {
            void requestLspCompletionAfterSync(getMainEditorValue(), '', 1);
            return true;
        }
    }

    return false;
}

export async function requestLspCompletionAfterSync(content, triggerChar = '', triggerKind = 2) {
    const blockTarget = getActiveLspTarget();
    if (!blockTarget && (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible())) {
        return;
    }

    const syncFile = blockTarget ? blockTarget.filePath : state.currentFile;
    const isShown = captureShownDocument();
    try {
        await NotesLspChangeDocument(syncFile, String(content || ''));
    } catch {
        // Fall through to completion request even when fast sync fails.
    }

    if (isShown()) {
        await requestLspCompletion(triggerKind, triggerChar);
    }
}

function lspPositionToEditorOffset(content, line, character) {
    const source = String(content || '');
    const lines = source.split('\n');
    const safeLine = Math.max(0, Math.min(Number(line) || 0, Math.max(lines.length - 1, 0)));
    const lineContent = lines[safeLine] || '';
    const safeCharacter = Math.max(0, Math.min(Number(character) || 0, lineContent.length));

    let offset = 0;
    for (let i = 0; i < safeLine; i += 1) {
        offset += (lines[i] || '').length + 1;
    }

    return offset + safeCharacter;
}

// typos-lsp encodes the misspelt word and its correction(s) in the diagnostic
// message using backticks, e.g. "`teh` should be `the`" or, for multiple
// corrections, "`recieve` should be `receive`, `relieve`". The first quoted
// token is the misspelt word; the remainder are suggestions.
function parseTyposDiagnosticMessage(message) {
    const matches = String(message || '').match(/`([^`]+)`/g);
    if (!matches || matches.length === 0) {
        return { word: '', suggestions: [] };
    }
    const unquote = (s) => s.slice(1, -1);
    return {
        word: unquote(matches[0]),
        suggestions: matches.slice(1).map(unquote),
    };
}

// Convert typos-lsp diagnostics into the misspelling shape consumed by the
// spellcheck canvas overlay (absolute character offsets into `text`).
function typosDiagnosticsToMisspellings(diagnostics, text) {
    const source = String(text || '');
    const list = [];
    for (const diag of Array.isArray(diagnostics) ? diagnostics : []) {
        if (!diag || !diag.range || !diag.range.start || !diag.range.end) {
            continue;
        }
        const start = lspPositionToEditorOffset(source, diag.range.start.line, diag.range.start.character);
        const end = lspPositionToEditorOffset(source, diag.range.end.line, diag.range.end.character);
        if (end <= start) {
            continue;
        }
        const parsed = parseTyposDiagnosticMessage(diag.message);
        list.push({
            misspeltWord: parsed.word || source.slice(start, end),
            wordStart: start,
            wordLength: end - start,
            suggestions: parsed.suggestions,
        });
    }
    return list;
}

function getLspSpellCheckExclusionSet() {
    return new Set([
        ...lspSpellCheckExclusions.symbols,
        ...lspSpellCheckExclusions.tokens,
        ...lspSpellCheckExclusions.keywords,
    ].map((entry) => String(entry || '').toLowerCase()));
}

// Route an incoming typos-lsp diagnostics payload to the spellcheck overlay of
// whichever editor owns the document (the main editor or a Jupyter code block).
function routeTyposDiagnostics(uri, diagnostics) {
    const targetPath = normalizePathForMatch(fileUriToPath(uri));
    if (!targetPath) {
        return;
    }

    // Main editor.
    if (state.typosOpenFile && getNotesSpellCheckHandle()
        && normalizePathForMatch(fileUriToPath(state.currentFileUri)) === targetPath) {
        getNotesSpellCheckHandle().setMisspellings(
            typosDiagnosticsToMisspellings(diagnostics, getMainEditorValue()));
        return;
    }

    // Jupyter code blocks.
    for (const blockId of Object.keys(state.jupyterCodeBlocks || {})) {
        const block = state.jupyterCodeBlocks[blockId];
        if (!block || !block.typosFilePath) {
            continue;
        }
        if (normalizePathForMatch(block.typosFilePath) !== targetPath) {
            continue;
        }
        const handle = jupyterSpellCheckHandles[blockId];
        if (!handle) {
            return;
        }
        const editable = elements.jupyter?.querySelector(`[data-block-id="${blockId}"] .jupyter-code-editable`);
        const text = editable ? (editable.value || '') : (block.currentContent || '');
        handle.setMisspellings(typosDiagnosticsToMisspellings(diagnostics, text));
        return;
    }
}

async function requestLspInlayHints() {
    if (isMonacoActive()) {
        getMonacoMainEditor().refreshInlayHints();
    }
}

async function requestLspDefinition() {
    if (!state.currentFile || !isCurrentFileLspEligible()) {
        return;
    }

    const selection = getMainEditorSelectionRange();
    const pos = offsetToLspPosition(getMainEditorValue(), selection.start || 0);

    const isShown = captureShownDocument();
    try {
        const locations = await NotesLspDefinition(state.currentFile, pos.line, pos.character);
        if (!isShown()) {
            return;
        }
        const target = Array.isArray(locations) && locations.length > 0 ? locations[0] : null;
        if (!target) {
            notifyTerminal('Definition not found', 'info');
            return;
        }

        const targetAbsPath = normalizePathForMatch(target.filePath || fileUriToPath(target.uri));
        if (!targetAbsPath) {
            notifyTerminal('Definition target is not a local file', 'warn');
            return;
        }

        const targetNotesFile = await resolveNotesFileFromAbsolutePath(targetAbsPath);
        if (!targetNotesFile) {
            notifyTerminal('Definition target is outside indexed Notes files', 'warn');
            return;
        }

        addHistoryPosition(targetNotesFile, Math.max(0, Number(target.line) || 0) + 1);
        await loadFile(targetNotesFile);

        const targetLine = Math.max(0, Number(target.line) || 0);
        const targetChar = Math.max(0, Number(target.character) || 0);
        const offset = lspPositionToEditorOffset(getMainEditorValue(), targetLine, targetChar);
        setMainEditorSelectionRange(offset, offset);
        if (isMonacoActive()) {
            getMonacoMainEditor().focus();
            getMonacoMainEditor().revealOffset(offset);
        }

        state.lspHoverLastKey = '';
        scheduleLspHover();
    } catch {
        notifyTerminal('Failed to resolve definition', 'error');
    }
}

export async function formatCurrentLspDocument(options = {}) {
    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
        return;
    }

    const preferSelection = options.preferSelection !== false;
    const notifyOnError = options.notifyOnError !== false;
    const file = state.currentFile;
    const isShown = captureShownDocument();

    try {
        const selection = getMainEditorSelectionRange();
        const selectionStart = Number(selection.start) || 0;
        const selectionEnd = Number(selection.end) || selectionStart;
        const currentText = getMainEditorValue();

        let result;
        if (preferSelection && selectionEnd > selectionStart) {
            const startPos = offsetToLspPosition(currentText, selectionStart);
            const endPos = offsetToLspPosition(currentText, selectionEnd);

            result = await NotesLspFormatRange(
                file,
                startPos.line,
                startPos.character,
                endPos.line,
                endPos.character,
            );
        } else {
            result = await NotesLspFormat(file);
        }

        if (!isShown() || !result || !result.changed) {
            return;
        }

        const nextContent = String(result.content || '');
        if (nextContent === String(currentText || '')) {
            return;
        }

        const prevContent = String(currentText || '');
        const prevLen = prevContent.length;
        const nextLen = nextContent.length;
        const ratio = prevLen > 0 ? nextLen / prevLen : 1;
        const nextSelectionStart = Math.min(Math.round(selectionStart * ratio), nextLen);
        const nextSelectionEnd = Math.min(Math.round(selectionEnd * ratio), nextLen);

        if (isMonacoActive()) {
            setMainEditorValue(nextContent);
            setMainEditorSelectionRange(nextSelectionStart, nextSelectionEnd);
        } else {
            replaceMainDocumentText(nextContent, { source: preferSelection && selectionEnd > selectionStart ? 'lsp-format-range' : 'lsp-format', selectionStart: nextSelectionStart, selectionEnd: nextSelectionEnd });
        }
    } catch {
        if (notifyOnError) {
            notifyTerminal('Failed to format document', 'error');
        }
    }
}

function symbolDisplayLabel(item) {
    const name = String(item?.name || '').trim();
    const detail = String(item?.detail || '').trim();
    const container = String(item?.containerName || '').trim();

    let label = name;
    if (detail) {
        label += ` - ${detail}`;
    }
    if (container) {
        label += ` (${container})`;
    }

    return label;
}

function workspaceSymbolDisplayLabel(item) {
    const base = symbolDisplayLabel(item);
    const filePath = normalizePathForMatch(item?.filePath || fileUriToPath(item?.uri || ''));
    if (!filePath) {
        return base;
    }

    const shortPath = filePath.split('/').slice(-2).join('/');
    return `${base} - ${shortPath}`;
}

function codeLensDisplayLabel(item) {
    const title = String(item?.title || '').trim() || 'Code lens';
    const line = Math.max(0, Number(item?.line) || 0);
    return `${title} (line ${line + 1})`;
}

export async function goToCurrentLspSymbol() {
    if (!state.currentFile || !isCurrentFileLspEligible()) {
        notifyTerminal('No Language Server Protocol (LSP) has been defined for this file type', 'warn');
        return;
    }

    if (state.lspOpenFile !== state.currentFile) {
        await openCurrentLspDocument(getMainEditorValue());
    }

    if (state.lspOpenFile !== state.currentFile) {
        notifyTerminal('Language server is not active for the current file', 'warn');
        return;
    }

    let symbols = [];
    const isShown = captureShownDocument();
    try {
        symbols = await NotesLspDocumentSymbols(state.currentFile);
    } catch {
        notifyTerminal('Failed to fetch document symbols', 'error');
        return;
    }
    if (!isShown()) {
        return;
    }

    const entries = Array.isArray(symbols)
        ? symbols.filter((item) => item && String(item.name || '').trim() !== '')
        : [];

    if (entries.length === 0) {
        notifyTerminal('No symbols found', 'info');
        return;
    }

    const options = entries.map((item) => symbolDisplayLabel(item));
    const icons = options.map(() => CONTEXT_ICON_CODE);
    showLocalMenu({
        title: 'Go to symbol',
        options,
        icons,
        x: window.innerWidth / 2,
        y: window.innerHeight / 2,
        showNextToMouseCursor: true,
        onSelect: (index) => {
            const picked = entries[index];
            if (!picked) {
                return;
            }

            const line = Math.max(0, Number(picked.line) || 0);
            const character = Math.max(0, Number(picked.character) || 0);
            const offset = lspPositionToEditorOffset(getMainEditorValue(), line, character);

            jumpEditorToOffset(offset);

            state.lspHoverLastKey = '';
            scheduleLspHover();
        },
    });
}

export async function goToWorkspaceLspSymbol() {
    if (!state.currentFile || !isCurrentFileLspEligible()) {
        notifyTerminal('No Language Server Protocol (LSP) has been defined for this file type', 'warn');
        return;
    }

    if (state.lspOpenFile !== state.currentFile) {
        await openCurrentLspDocument(getMainEditorValue());
    }

    if (state.lspOpenFile !== state.currentFile) {
        notifyTerminal('Language server is not active for the current file', 'warn');
        return;
    }

    // Servers match symbols against the query and return nothing for an empty
    // one (and cap results), so search server-side on each keystroke.
    let entries = [];

    const queryWorkspaceSymbols = async (rawQuery) => {
        const query = String(rawQuery || '').trim();
        entries = [];
        if (!query) {
            return { options: [], icons: [] };
        }

        let symbols = [];
        try {
            symbols = await NotesLspWorkspaceSymbols(state.currentFile, query);
        } catch {
            notifyTerminal('Failed to fetch workspace symbols', 'error');
            return { options: [], icons: [] };
        }

        entries = Array.isArray(symbols)
            ? symbols.filter((item) => item && String(item.name || '').trim() !== '')
            : [];

        const options = entries.map((item) => workspaceSymbolDisplayLabel(item));
        return { options, icons: options.map(() => CONTEXT_ICON_CODE) };
    };

    showLocalMenu({
        title: 'Go to workspace symbol',
        options: [],
        icons: [],
        x: window.innerWidth / 2,
        y: window.innerHeight / 2,
        showSearch: true,
        hideItemsUntilQuery: true,
        onQuery: queryWorkspaceSymbols,
        showNextToMouseCursor: true,
        onSelect: async (index) => {
            try {
                const picked = entries[index];
                if (!picked) {
                    return;
                }

                const targetAbsPath = normalizePathForMatch(picked.filePath || fileUriToPath(picked.uri));
                if (!targetAbsPath) {
                    notifyTerminal('Workspace symbol target is not a local file', 'warn');
                    return;
                }

                const targetNotesFile = await resolveNotesFileFromAbsolutePath(targetAbsPath);
                if (!targetNotesFile) {
                    notifyTerminal('Workspace symbol target is outside indexed Notes files', 'warn');
                    return;
                }

                addHistoryPosition(targetNotesFile, Math.max(0, Number(picked.line) || 0) + 1);
                await loadFile(targetNotesFile);

                const line = Math.max(0, Number(picked.line) || 0);
                const character = Math.max(0, Number(picked.character) || 0);
                const offset = lspPositionToEditorOffset(getMainEditorValue(), line, character);
                jumpEditorToOffset(offset);

                state.lspHoverLastKey = '';
                scheduleLspHover();
            } catch (err) {
                console.error('Workspace symbol navigation failed:', err);
                notifyTerminal('Failed to navigate to workspace symbol', 'error');
            }
        },
    });
}

async function runCurrentLspCodeLens() {
    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
        return;
    }

    let lenses = [];
    const file = state.currentFile;
    const isShown = captureShownDocument();
    try {
        lenses = await NotesLspCodeLens(file);
    } catch {
        notifyTerminal('Failed to fetch code lens actions', 'error');
        return;
    }
    if (!isShown()) {
        return;
    }

    const entries = Array.isArray(lenses)
        ? lenses.filter((item) => item && Number.isFinite(Number(item.index)))
        : [];
    if (entries.length === 0) {
        notifyTerminal('No code lens actions found', 'info');
        return;
    }

    const options = entries.map((item) => codeLensDisplayLabel(item));
    const icons = options.map(() => CONTEXT_ICON_CODE);
    showLocalMenu({
        title: 'Code lens',
        options,
        icons,
        x: window.innerWidth / 2,
        y: window.innerHeight / 2,
        showNextToMouseCursor: true,
        onSelect: async (index) => {
            const picked = entries[index];
            if (!picked) {
                return;
            }

            try {
                const applied = await NotesLspExecuteCodeLens(file, Number(picked.index) || 0);
                if (!applied) {
                    notifyTerminal('Code lens had no executable command', 'info');
                }
            } catch {
                notifyTerminal('Failed to run code lens', 'error');
            }
        },
    });
}

function isPositionWithinRange(line, character, range) {
    if (!range || !range.start || !range.end) {
        return false;
    }

    const startLine = Number(range.start.line) || 0;
    const startChar = Number(range.start.character) || 0;
    const endLine = Number(range.end.line) || startLine;
    const endChar = Number(range.end.character) || startChar;

    if (line < startLine || line > endLine) {
        return false;
    }
    if (line === startLine && character < startChar) {
        return false;
    }
    if (line === endLine && character > endChar) {
        return false;
    }

    return true;
}

function lspDiagnosticsAtPosition(line, character, uri = state.currentFileUri) {
    if (!uri) {
        return [];
    }

    const diagnostics = lspDiagnosticsStore.get(uri) || [];
    return diagnostics.filter((diag) => isPositionWithinRange(line, character, diag.range));
}

function formatLspDiagnosticsForHover(diagnostics) {
    const list = Array.isArray(diagnostics) ? diagnostics : [];
    if (list.length === 0) {
        return '';
    }

    const severityLabel = (value) => {
        switch (Number(value) || 0) {
        case 1: return 'Error';
        case 2: return 'Warning';
        case 3: return 'Info';
        case 4: return 'Hint';
        default: return 'Info';
        }
    };

    const lines = ['Diagnostics:'];
    for (const diag of list) {
        const message = String(diag?.message || '').trim();
        if (!message) {
            continue;
        }
        const severity = severityLabel(diag?.severity);
        const source = String(diag?.source || '').trim();
        const prefix = source ? `${severity} (${source})` : severity;
        lines.push(`- ${prefix}: ${message}`);
    }

    return lines.length > 1 ? lines.join('\n') : '';
}

async function getLspCodeActionsForCursor() {
    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
        return { line: 0, character: 0, diagnostics: [], actions: [] };
    }

    const selection = getMainEditorSelectionRange();
    const pos = offsetToLspPosition(getMainEditorValue(), selection.start || 0);
    const diagnostics = lspDiagnosticsAtPosition(pos.line, pos.character);

    const isShown = captureShownDocument();
    try {
        const actions = await NotesLspCodeActions(state.currentFile, pos.line, pos.character, diagnostics);
        if (!isShown()) {
            return { line: 0, character: 0, diagnostics: [], actions: [] };
        }
        return {
            line: pos.line,
            character: pos.character,
            diagnostics,
            actions: Array.isArray(actions) ? actions : [],
        };
    } catch {
        return {
            line: pos.line,
            character: pos.character,
            diagnostics,
            actions: [],
        };
    }
}

function lspCodeActionMenuSection(kind) {
    const value = String(kind || '').trim().toLowerCase();
    if (value === 'quickfix' || value.startsWith('quickfix.')) {
        return { key: 'quickfix', label: 'Quick fix', rank: 0 };
    }
    if (value === 'refactor' || value.startsWith('refactor.')) {
        return { key: 'refactor', label: 'Refactor', rank: 1 };
    }
    if (value === 'source' || value.startsWith('source.')) {
        return { key: 'source', label: 'Source action', rank: 2 };
    }

    return { key: 'other', label: 'Code action', rank: 3 };
}

function buildLspCodeActionMenuItems(actions, line, character, diagnostics) {
    const grouped = new Map();

    (Array.isArray(actions) ? actions : []).forEach((action, index) => {
        const title = String(action?.title || '').trim();
        if (!title) {
            return;
        }

        const section = lspCodeActionMenuSection(action?.kind);
        const entry = {
            title: `${section.label}: ${title}`,
            icon: CONTEXT_ICON_CODE,
            rank: section.rank,
            onSelect: () => {
                void applyLspCodeActionFromCursor(index, line, character, diagnostics);
            },
        };

        if (!grouped.has(section.key)) {
            grouped.set(section.key, []);
        }
        grouped.get(section.key).push(entry);
    });

    const sections = Array.from(grouped.values())
        .filter((items) => items.length > 0)
        .sort((left, right) => (left[0]?.rank || 0) - (right[0]?.rank || 0));

    const menuItems = [];
    sections.forEach((items, index) => {
        if (index > 0) {
            menuItems.push({ title: '-' });
        }
        menuItems.push(...items);
    });

    return menuItems;
}

export async function showEditorLspOptionsMenu(x, y) {
    const menuItems = [];
    if (isCurrentFileLspEligible()) {
        menuItems.push(
        {
            title: 'Format document',
            icon: CONTEXT_ICON_CODE,
            onSelect: () => {
                void formatCurrentLspDocument();
            },
        },
        {
            title: 'Go to symbol...',
            icon: CONTEXT_ICON_CODE,
            onSelect: () => {
                void goToCurrentLspSymbol();
            },
        },
        {
            title: 'Go to workspace symbol...',
            icon: CONTEXT_ICON_CODE,
            onSelect: () => {
                void goToWorkspaceLspSymbol();
            },
        },
        {
            title: 'Signature help',
            icon: CONTEXT_ICON_CODE,
            onSelect: () => {
                void requestLspSignatureHelpFromCursor(1, '');
            },
        },
        {
            title: 'Code lens...',
            icon: CONTEXT_ICON_CODE,
            onSelect: () => {
                void runCurrentLspCodeLens();
            },
        },
        {
            title: 'Rename symbol...',
            icon: CONTEXT_ICON_CODE,
            onSelect: () => {
                void renameCurrentLspSymbol();
            },
        },
        );
    }

    menuItems.push({
        title: 'Find references',
        icon: CONTEXT_ICON_FIND,
        onSelect: () => {
            void findReferencesFromEditor();
        },
    });

    if (!isCurrentFileLspEligible()) {
        showNotesLocalMenu(menuItems, x, y, 'Find references');
        return;
    }

    const codeActionData = await getLspCodeActionsForCursor();
    if (codeActionData.actions.length > 0) {
        menuItems.push({ title: '-' });
        menuItems.push(
            ...buildLspCodeActionMenuItems(
                codeActionData.actions,
                codeActionData.line,
                codeActionData.character,
                codeActionData.diagnostics,
            ),
        );
    }

    showNotesLocalMenu(menuItems, x, y, 'LSP options');
}

function formatLspReferencePath(filePath) {
    const path = String(filePath || '').replace(/\\/g, '/');
    const root = String(state.currentProjectRoot || '').replace(/\\/g, '/').replace(/\/+$/, '');
    if (root && (path === root || path.startsWith(`${root}/`))) {
        return `$LSP/${path.slice(root.length).replace(/^\/+/, '')}`;
    }
    return `$LSP/${path.split('/').pop() || path}`;
}

export async function findReferencesFromEditor() {
    const token = currentEditorToken();
    if (!token || !state.currentFile) {
        notifyTerminal('Place the cursor on a symbol to find references', 'info');
        return;
    }

    const pos = offsetToLspPosition(getMainEditorValue(), token.start);
    if (isCurrentFileLspEligible()) {
        try {
            const locations = await NotesLspReferences(state.currentFile, pos.line, pos.character);
            if (Array.isArray(locations) && locations.length > 0) {
                setProjectReferencesResults(token.symbol, 'LSP', locations.map((item) => ({
                    fileName: String(item?.filePath || item?.uri || '').split('/').pop() || token.symbol,
                    path: String(item?.filePath || item?.uri || ''),
                    displayPath: formatLspReferencePath(item?.filePath || item?.uri || ''),
                    line: (Number(item?.line) || 0) + 1,
                    source: 'LSP',
                    context: Array.isArray(item?.context) ? item.context.map((line) => String(line)) : [],
                })));
                return;
            }
        } catch (err) {
            console.warn('LSP references unavailable; using text search fallback', err);
        }
    }

    state.findFilesMode = 'references';
    state.findFilesSource = 'Text search fallback';
    state.findFilesReferenceSymbol = token.symbol;
    elements.findFilesInput.value = token.symbol;
    state.findOptions.wholeWord = true;
    updateFindOptionButtons();
    setToolsPanelCollapsed(false);
    setToolsTab('find');
    runProjectFindSearch(token.symbol);
}

async function applyLspCodeActionFromCursor(index, line, character, diagnostics) {
    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
        return;
    }

    const isShown = captureShownDocument();
    try {
        const result = await NotesLspApplyCodeAction(state.currentFile, line, character, index, diagnostics || []);
        if (!isShown() || !result || !result.changed) {
            return;
        }

        const nextContent = String(result.content || '');
        const currentText = getMainEditorValue();
        if (nextContent === String(currentText || '')) {
            return;
        }

        const selection = getMainEditorSelectionRange();
        const selectionStart = Number(selection.start) || 0;
        const selectionEnd = Number(selection.end) || selectionStart;
        const nextLen = nextContent.length;
        if (isMonacoActive()) {
            setMainEditorValue(nextContent);
            setMainEditorSelectionRange(Math.min(selectionStart, nextLen), Math.min(selectionEnd, nextLen));
        } else {
            replaceMainDocumentText(nextContent, { source: 'lsp-code-action', selectionStart: Math.min(selectionStart, nextLen), selectionEnd: Math.min(selectionEnd, nextLen) });
        }
    } catch {
        notifyTerminal('Failed to apply code action', 'error');
    }
}

async function renameCurrentLspSymbol() {
    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
        return;
    }

    const selection = getMainEditorSelectionRange();
    const selectionStart = Number(selection.start) || 0;
    const selectionEnd = Number(selection.end) || selectionStart;
    const currentSelection = selectionEnd > selectionStart
        ? String(getMainEditorValue() || '').slice(selectionStart, selectionEnd)
        : '';

    const pos = offsetToLspPosition(getMainEditorValue(), selectionStart);

    let prepare;
    const file = state.currentFile;
    const isShown = captureShownDocument();
    try {
        prepare = await NotesLspPrepareRename(file, pos.line, pos.character);
    } catch {
        notifyTerminal('Failed to prepare symbol rename', 'error');
        return;
    }
    if (!isShown()) {
        return;
    }

    if (!prepare || !prepare.canRename) {
        notifyTerminal('Rename not available at this cursor position', 'info');
        return;
    }

    const suggested = String(prepare.placeholder || currentSelection || '').trim();
    const nextName = await openTextPrompt({ title: 'Rename symbol', value: suggested, confirmLabel: 'Rename' });
    if (nextName === null || !isShown()) {
        return;
    }

    const trimmedName = String(nextName).trim();
    if (!trimmedName) {
        notifyTerminal('Rename cancelled: new name is empty', 'warn');
        return;
    }

    try {
        const result = await NotesLspRename(file, pos.line, pos.character, trimmedName);
        if (!isShown()) {
            return;
        }
        if (!result || !result.changed) {
            notifyTerminal('No rename edits applied', 'info');
            return;
        }

        const nextContent = String(result.content || '');
        if (nextContent === String(getMainEditorValue() || '')) {
            return;
        }

        const nextLen = nextContent.length;
        if (isMonacoActive()) {
            setMainEditorValue(nextContent);
            setMainEditorSelectionRange(Math.min(selectionStart, nextLen), Math.min(selectionEnd, nextLen));
        } else {
            replaceMainDocumentText(nextContent, { source: 'lsp-rename', selectionStart: Math.min(selectionStart, nextLen), selectionEnd: Math.min(selectionEnd, nextLen) });
        }
    } catch {
        notifyTerminal('Failed to rename symbol', 'error');
    }
}

function offsetToLspPosition(content, offset) {
    const source = String(content || '');
    const clampedOffset = Math.max(0, Math.min(Number(offset) || 0, source.length));
    const before = source.slice(0, clampedOffset);
    const lines = before.split('\n');
    return {
        line: Math.max(0, lines.length - 1),
        character: lines[lines.length - 1]?.length || 0,
    };
}

function getLspAnchorViewportPoint() {
    if (Number.isFinite(state.lspHoverMouseX) && Number.isFinite(state.lspHoverMouseY)
        && (state.lspHoverMouseX !== 0 || state.lspHoverMouseY !== 0)) {
        return {
            x: state.lspHoverMouseX,
            y: state.lspHoverMouseY,
        };
    }

    const fallbackRect = getFocusedEditorHost()?.getBoundingClientRect();
    if (fallbackRect) {
        return {
            x: fallbackRect.left + 20,
            y: fallbackRect.top + 20,
        };
    }

    return { x: 20, y: 20 };
}

export function scheduleLspHover() {
    if (state.lspCompletionVisible) {
        clearLspHoverTimer();
        hideLspHoverTooltip();
        return;
    }

    const blockTarget = getActiveLspTarget();
    if (!blockTarget && (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible())) {
        hideLspHoverTooltip();
        return;
    }

    const hoverFilePath = blockTarget ? blockTarget.filePath : state.currentFile;
    const readHoverPosition = () => (blockTarget
        ? offsetToLspPosition(blockTarget.editor.value || '', blockTarget.editor.selectionStart || 0)
        : offsetToLspPosition(mainDocument.getValue(), mainDocument.getSelection().start));

    clearLspHoverTimer();
    state.lspHoverTimer = setTimeout(async () => {
        state.lspHoverTimer = null;

        const pos = readHoverPosition();
        const key = `${hoverFilePath}:${pos.line}:${pos.character}`;
        if (key === state.lspHoverLastKey) {
            return;
        }
        state.lspHoverLastKey = key;
        const isShown = captureShownDocument();

        try {
            const hoverText = await NotesLspHover(hoverFilePath, pos.line, pos.character);
            if (!isShown()) {
                return;
            }
            const diagnosticsText = (!blockTarget && hoverFilePath === state.currentFile)
                ? formatLspDiagnosticsForHover(lspDiagnosticsAtPosition(pos.line, pos.character))
                : '';

            const sections = [];
            if (hoverText) {
                sections.push(String(hoverText));
            }
            if (diagnosticsText) {
                sections.push(diagnosticsText);
            }

            if (sections.length === 0) {
                hideLspHoverTooltip();
                return;
            }

            await renderLspHoverTooltipText(sections.join('\n\n'));
            lspHoverTooltipEl.style.display = 'block';

            const anchor = getLspAnchorViewportPoint();
            const rawX = anchor.x;
            const rawY = anchor.y;
            const x = Math.min(rawX + 14, window.innerWidth - lspHoverTooltipEl.offsetWidth - 8);
            const y = Math.min(rawY + 14, window.innerHeight - lspHoverTooltipEl.offsetHeight - 8);
            lspHoverTooltipEl.style.left = `${Math.max(8, x)}px`;
            lspHoverTooltipEl.style.top = `${Math.max(8, y)}px`;
        } catch {
            hideLspHoverTooltip();
        }
    }, LSP_HOVER_DEBOUNCE_MS);
}

async function renderLspHoverTooltipText(text) {
    lspHoverTooltipEl.innerHTML = marked.parse(escapeHtml(text));
    await processMarkdownContainer(lspHoverTooltipEl);
}

async function requestLspSignatureHelpFromCursor(triggerKind = 1, triggerChar = '') {
    if (state.lspCompletionVisible) {
        hideLspHoverTooltip();
        return;
    }

    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
        return;
    }

    const pos = offsetToLspPosition(mainDocument.getValue(), mainDocument.getSelection().start);

    const isShown = captureShownDocument();
    try {
        const text = await NotesLspSignatureHelp(state.currentFile, pos.line, pos.character, triggerKind, triggerChar);
        if (!isShown()) {
            return;
        }
        if (!text) {
            notifyTerminal('No signature help available', 'info');
            return;
        }

        await renderLspHoverTooltipText(text);
        lspHoverTooltipEl.style.display = 'block';

        const anchor = getLspAnchorViewportPoint();
        const rawX = anchor.x;
        const rawY = anchor.y;
        const x = Math.min(rawX + 14, window.innerWidth - lspHoverTooltipEl.offsetWidth - 8);
        const y = Math.min(rawY + 14, window.innerHeight - lspHoverTooltipEl.offsetHeight - 8);
        lspHoverTooltipEl.style.left = `${Math.max(8, x)}px`;
        lspHoverTooltipEl.style.top = `${Math.max(8, y)}px`;
    } catch {
        notifyTerminal('Failed to fetch signature help', 'error');
    }
}

// When a NotesDocument closes, release its LSP and typos registrations. The
// returned promises settle when the backend has closed them.
export function releaseLspDocument(doc) {
    const closes = [];
    if (doc.lspOpenFile) {
        for (const [uri, entry] of lspModelDocuments) {
            if (entry.filePath === doc.lspOpenFile) {
                lspModelDocuments.delete(uri);
            }
        }
        closes.push(NotesLspCloseDocument(doc.lspOpenFile));
        doc.lspOpenFile = '';
    }
    if (doc.typosOpenFile) {
        closes.push(NotesTyposCloseDocument(doc.typosOpenFile));
        doc.typosOpenFile = '';
    }
    return closes;
}

// typos-lsp only checks documents a view shows (ADR 0100 question 13), so a
// document moving to a background tab releases its typos registration. It is
// opened again when the document is next shown.
export function releaseTyposDocument(doc) {
    clearTimeout(doc.typosChangeTimer);
    doc.typosChangeTimer = null;
    const file = doc.typosOpenFile;
    doc.typosOpenFile = '';
    if (!file) {
        return Promise.resolve();
    }
    return Promise.resolve(NotesTyposCloseDocument(file)).catch((err) => {
        console.error('notes typos close failed:', err);
    });
}

// When the editor stops showing a document, clear the LSP and spellcheck UI
// that belonged to it. keepTypos leaves its underlines for another view
// still showing it.
export function resetLspViewState({ keepTypos = false } = {}) {
    clearLspHoverTimer();
    hideLspHoverTooltip();
    hideLspCompletion();
    state.lspHoverLastKey = '';
    getNotesSpellCheckHandle()?.setMode('aspell');
    if (isMonacoActive() && !keepTypos) {
        getMonacoMainEditor().setTyposMisspellings([]);
    }
}

export async function closeOpenLspDocument() {
    for (const [uri, doc] of lspModelDocuments) {
        if (doc.filePath === state.lspOpenFile) {
            lspModelDocuments.delete(uri);
        }
    }
    clearLspChangeTimer();
    clearLspHoverTimer();
    hideLspHoverTooltip();
    hideLspCompletion();

    const openFile = state.lspOpenFile;
    if (!openFile) {
        return;
    }

    try {
        await NotesLspCloseDocument(openFile);
    } catch (err) {
        console.error('notes lsp close failed:', err);
    } finally {
        state.lspOpenFile = '';
        state.lspHoverLastKey = '';
    }
}

export async function openCurrentLspDocument(content) {
    // The backend reference-counts opens, so release the document this editor
    // holds before taking a new reference, even when reloading the same file.
    // lspOpenFile is set even if no server started, so closing first (rather
    // than reopening and then closing) keeps a failed start retryable.
    await closeOpenLspDocument();
    const doc = getActiveNotesDocument();
    const file = doc.path;
    if (!file || !isCurrentFileLspEligible()) {
        return;
    }

    try {
        const languageID = await ResolveNotesLspLanguage(file);
        if (!languageID || doc !== getActiveNotesDocument()) {
            return;
        }

        await NotesLspOpenDocument(file, languageID, String(content || ''));
        // A document holds one reference. Release this one if the document
        // closed meanwhile or another open already registered it.
        if (!isNotesDocumentOpen(doc) || doc.lspOpenFile) {
            await NotesLspCloseDocument(file).catch((err) => console.error('notes lsp close failed:', err));
            return;
        }
        doc.lspOpenFile = file;
        if (doc !== getActiveNotesDocument()) {
            return;
        }

        if (isMonacoActive()) {
            const editor = getMonacoMainEditor();
            lspModelDocuments.set(editor.getModelUri(), { filePath: state.currentFile, fileUri: state.currentFileUri });
            editor.configureLsp({
                signature: async ({ model, line, character }) => {
                    const doc = lspDocumentForModel(model);
                    if (!doc) {
                        return '';
                    }
                    return await NotesLspSignatureHelp(doc.filePath, line, character, 1, '');
                },
                semanticTokens: async ({ model, previousResultId } = {}) => {
                    const doc = lspDocumentForModel(model);
                    if (!doc) {
                        return null;
                    }
                    try {
                        await NotesLspChangeDocument(doc.filePath, model.getValue());
                    } catch {
                        // Semantic tokens can still use the last synced state.
                    }
                    if (previousResultId) {
                        return await NotesLspSemanticTokensDelta(doc.filePath, previousResultId);
                    }
                    return await NotesLspSemanticTokens(doc.filePath);
                },
                inlayHints: async ({ model }) => {
                    const doc = lspDocumentForModel(model);
                    if (!doc) {
                        return [];
                    }
                    const hints = await NotesLspInlayHints(doc.filePath);
                    return Array.isArray(hints)
                        ? hints.filter((item) => item && String(item.label || '') !== '')
                        : [];
                },
                formatDocument: async ({ model }) => {
                    const doc = lspDocumentForModel(model);
                    if (!doc) {
                        return null;
                    }
                    return await NotesLspFormat(doc.filePath);
                },
                formatRange: async ({ model, start, end }) => {
                    const doc = lspDocumentForModel(model);
                    if (!doc) {
                        return null;
                    }
                    return await NotesLspFormatRange(
                        doc.filePath,
                        Number(start?.line) || 0,
                        Number(start?.character) || 0,
                        Number(end?.line) || 0,
                        Number(end?.character) || 0,
                    );
                },
                definition: async ({ model, line, character }) => {
                    const doc = lspDocumentForModel(model);
                    if (!doc) {
                        return [];
                    }
                    const locations = await NotesLspDefinition(doc.filePath, line, character);
                    return Array.isArray(locations) ? locations : [];
                },
                prepareRename: async ({ model, line, character }) => {
                    const doc = lspDocumentForModel(model);
                    if (!doc) {
                        return { canRename: false };
                    }
                    return await NotesLspPrepareRename(doc.filePath, line, character);
                },
                rename: async ({ model, line, character, newName }) => {
                    const doc = lspDocumentForModel(model);
                    if (!doc) {
                        return null;
                    }
                    return await NotesLspRename(doc.filePath, line, character, String(newName || ''));
                },
                codeActions: async ({ model, line, character }) => {
                    const doc = lspDocumentForModel(model);
                    if (!doc) {
                        return [];
                    }
                    const diagnostics = lspDiagnosticsAtPosition(line, character, doc.fileUri);
                    const actions = await NotesLspCodeActions(doc.filePath, line, character, diagnostics);
                    return Array.isArray(actions) ? actions : [];
                },
                applyCodeAction: async (action) => {
                    const doc = lspDocumentForModel(action?.model);
                    const actionIndex = Number(action?.index);
                    if (!doc || !Number.isFinite(actionIndex)) {
                        return null;
                    }
                    const line = Number(action?.line) || 0;
                    const character = Number(action?.character) || 0;
                    const diagnostics = lspDiagnosticsAtPosition(line, character, doc.fileUri);
                    return await NotesLspApplyCodeAction(doc.filePath, line, character, actionIndex, diagnostics);
                },
            });
        }

        await requestLspInlayHints();
        void updateSpellCheckExclusionsFromDocSymbols();
    } catch (err) {
        console.error('notes lsp open failed:', err);
    }
}

/**
 * Fetch document symbols from the LSP server and use their names as
 * spell-check exclusions. Document symbols are available immediately after
 * didOpen, making them
 * a reliable exclusion source on file open.
 */
async function updateSpellCheckExclusionsFromDocSymbols() {
    const fileAtCall = state.currentFile;
    if (!getNotesSpellCheckHandle() || !fileAtCall || state.lspOpenFile !== fileAtCall || !isCurrentFileLspEligible()) {
        return;
    }

    function extractNames(symbols) {
        const names = [];
        function walk(syms) {
            if (!Array.isArray(syms)) return;
            for (const s of syms) {
                if (s && s.name) names.push(String(s.name));
                if (s && s.children) walk(s.children);
            }
        }
        walk(symbols);
        return names;
    }

    // First attempt — may return empty if gopls hasn't analysed the file yet.
    try {
        const symbols = await NotesLspDocumentSymbols(fileAtCall);
        if (state.currentFile !== fileAtCall) return;
        const names = extractNames(symbols);
        if (names.length > 0) {
            lspSpellCheckExclusions.symbols = names;
            applyLspSpellCheckExclusions();
            return;
        }
    } catch {
        // fall through to retry
    }

    // Retry after a short delay to give gopls time to finish analysing.
    await new Promise(resolve => setTimeout(resolve, 2000));
    if (state.currentFile !== fileAtCall || state.lspOpenFile !== fileAtCall) return;

    try {
        const symbols = await NotesLspDocumentSymbols(fileAtCall);
        if (state.currentFile !== fileAtCall) return;
        const names = extractNames(symbols);
        if (names.length > 0) {
            lspSpellCheckExclusions.symbols = names;
            applyLspSpellCheckExclusions();
        }
    } catch {
        // LSP not available or request failed — degrade silently.
    }
}

export function scheduleLspDidChange() {
    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
        return;
    }

    clearLspChangeTimer();
    state.lspChangeTimer = setTimeout(async () => {
        state.lspChangeTimer = null;
        try {
            await NotesLspChangeDocument(state.currentFile, mainDocument.getValue());
            await requestLspInlayHints();
        } catch (err) {
            console.error('notes lsp change failed:', err);
        }
    }, LSP_CHANGE_DEBOUNCE_MS);
}

export async function openCurrentTyposDocument(content) {
    if (!getNotesSpellCheckHandle()) {
        return;
    }
    if (!state.currentFile || !isCurrentFileLspEligible()) {
        // Non-eligible files keep aspell.
        await closeCurrentTyposDocument();
        return;
    }

    // The backend reference-counts opens. typosOpenFile is only set after a
    // successful open, so reopening first and then releasing the old reference
    // turns a same-file reload into a didChange. A document holds one
    // reference, so the old one is only released if it is still registered,
    // and the new one is released if the document closed meanwhile or
    // another open registered it first.
    const doc = getActiveNotesDocument();
    const file = doc.path;
    let previousFile = doc.typosOpenFile;
    const releasePrevious = async () => {
        const previous = previousFile;
        previousFile = '';
        if (previous && doc.typosOpenFile === previous) {
            doc.typosOpenFile = '';
            await NotesTyposCloseDocument(previous).catch((err) => console.error('notes typos close failed:', err));
        }
    };
    const showAspell = () => {
        if (doc !== getActiveNotesDocument() || doc.typosOpenFile) {
            return;
        }
        getNotesSpellCheckHandle()?.setMode('aspell');
        if (isMonacoActive()) {
            getMonacoMainEditor().setTyposMisspellings([]);
        }
    };
    try {
        const languageID = await ResolveNotesLspLanguage(file);
        const ok = await NotesTyposOpenDocument(file, languageID || '', String(content || ''));
        await releasePrevious();
        if (!ok) {
            showAspell();
            return;
        }
        if (!isNotesDocumentOpen(doc) || doc.typosOpenFile || doc !== getActiveNotesDocument()) {
            await NotesTyposCloseDocument(file).catch((err) => console.error('notes typos close failed:', err));
            return;
        }
        doc.typosOpenFile = file;
        getNotesSpellCheckHandle()?.setMode('external');
        const diagnostics = doc.uri ? (typosDiagnosticsStore.get(doc.uri) || []) : [];
        getNotesSpellCheckHandle()?.setMisspellings(typosDiagnosticsToMisspellings(diagnostics, getMainEditorValue()));
    } catch (err) {
        await releasePrevious();
        showAspell();
        console.error('notes typos open failed:', err);
    }
}

export async function closeCurrentTyposDocument() {
    if (state.typosChangeTimer) {
        clearTimeout(state.typosChangeTimer);
        state.typosChangeTimer = null;
    }
    const openFile = state.typosOpenFile;
    state.typosOpenFile = '';
    getNotesSpellCheckHandle()?.setMode('aspell');
    if (isMonacoActive()) {
        getMonacoMainEditor().setTyposMisspellings([]);
    }
    if (!openFile) {
        return;
    }
    try {
        await NotesTyposCloseDocument(openFile);
    } catch (err) {
        console.error('notes typos close failed:', err);
    }
}

// Sends a document's debounced LSP and typos changes now, so a document that
// moves to a background tab leaves the servers with its latest text.
export function flushDocumentChanges(doc) {
    const pending = [
        ['lspChangeTimer', doc.lspOpenFile, NotesLspChangeDocument],
        ['typosChangeTimer', doc.typosOpenFile, NotesTyposChangeDocument],
    ];
    for (const [timer, file, send] of pending) {
        if (!doc[timer]) {
            continue;
        }
        clearTimeout(doc[timer]);
        doc[timer] = null;
        if (file) {
            Promise.resolve(send(file, doc.buffer.getValue())).catch((err) => {
                console.error('notes document change failed:', err);
            });
        }
    }
}

export function scheduleTyposDidChange() {
    if (!state.typosOpenFile || state.typosOpenFile !== state.currentFile) {
        return;
    }
    if (state.typosChangeTimer) {
        clearTimeout(state.typosChangeTimer);
    }
    state.typosChangeTimer = setTimeout(async () => {
        state.typosChangeTimer = null;
        try {
            await NotesTyposChangeDocument(state.currentFile, mainDocument.getValue());
        } catch (err) {
            console.error('notes typos change failed:', err);
        }
    }, LSP_CHANGE_DEBOUNCE_MS);
}

// Keyed by file URI → array of Diagnostic objects from the server.
export const lspDiagnosticsStore = new Map();

const typosDiagnosticsStore = new Map();

// Tooltip element — created once and re-positioned on hover.
export const lspTooltipEl = (() => {
    const el = document.createElement('div');
    el.id = 'notes-lsp-tooltip';
    document.body.appendChild(el);
    return el;
})();

export const lspHoverTooltipEl = (() => {
    const el = document.createElement('div');
    el.id = 'notes-lsp-hover-tooltip';
    el.className = 'markdown-body';
    el.addEventListener('mousedown', (e) => {
        if (e.button === 0) e.preventDefault(); // left click: keep editor focus
    });
    el.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') hideLspHoverTooltip();
    });
    document.body.appendChild(el);
    return el;
})();

const lspCompletionEl = (() => {
    const el = document.createElement('div');
    el.id = 'notes-lsp-completion';
    document.body.appendChild(el);
    return el;
})();

const LSP_SEVERITY_CLASS = ['', 'error', 'warning', 'info', 'hint'];

export function clearVisibleLspDiagnostics(options = {}) {
    const preserveCompletion = options && options.preserveCompletion === true;
    if (isMonacoActive()) {
        getMonacoMainEditor().setDiagnostics([]);
    }
    if (lspTooltipEl) {
        lspTooltipEl.style.display = 'none';
    }
    hideLspHoverTooltip();
    if (!preserveCompletion) {
        hideLspCompletion();
    }
}

export function clearCurrentFileLspDiagnosticsCache() {
    if (!state.currentFileUri) {
        return;
    }
    lspDiagnosticsStore.delete(state.currentFileUri);
}

function renderLspDiagnostics() {
    if (!isMonacoActive()) {
        return;
    }

    // Collect diagnostics for the currently open file.
    const currentUri = state.currentFileUri || null;

    const diags = currentUri ? (lspDiagnosticsStore.get(currentUri) || []) : [];
    getMonacoMainEditor().setDiagnostics(diags);
}

export function renderLspEditorDecorations() {
    renderLspDiagnostics();
}

// Three independent exclusion sources — always merged before being applied.
// Separating them prevents one source from clobbering another when they
// arrive at different times.
export const lspSpellCheckExclusions = { symbols: [], tokens: [], keywords: [] };

export function applyLspSpellCheckExclusions() {
    const merged = Array.from(getLspSpellCheckExclusionSet());
    getNotesSpellCheckHandle()?.setExclusions(merged);
}
