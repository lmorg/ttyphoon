// A string-backed stand-in for createMonacoAdapter, for jsdom tests where
// Monaco cannot run. It implements the adapter API that notes.js uses and
// fires onChange/onSelectionChange the way the real adapter does, so tests
// exercise the same Monaco code paths as the app.

function clampOffset(value, length) {
    return Math.max(0, Math.min(Number(value) || 0, length));
}

let fakeModelCount = 0;

function createFakeModel(value, language, onChange) {
    return {
        value: String(value || ''),
        selection: { start: 0, end: 0, active: 0 },
        language: String(language || 'plaintext'),
        // Like Monaco: replaceRange (executeEdits) is undoable, setValue resets history.
        undoStack: [],
        uri: `inmemory://model/${++fakeModelCount}`,
        onChange,
        // Like the real adapter: editors showing the model, and the one that
        // showed it last, which selection and edits go through.
        shownBy: new Set(),
        activeEditor: null,
        selectionVersion: 0,
    };
}

export function createFakeMonacoAdapter(container, options = {}) {
    const placeholder = createFakeModel(options.value, options.language);
    let current = placeholder;
    let disposed = false;
    // Each editor has its own selection in each model it shows.
    const selections = new Map([[placeholder, placeholder.selection]]);
    const selectionVersions = new Map();
    // The shown model's fields, as the single-model fake used to hold them.
    const field = (name) => ({ get: () => current[name], set: (next) => { current[name] = next; } });
    const local = Object.defineProperties({}, {
        value: field('value'),
        selection: {
            get: () => selections.get(current) || { start: 0, end: 0, active: 0 },
            set: (next) => { selections.set(current, next); },
        },
        language: field('language'),
        undoStack: field('undoStack'),
    });
    const releaseCurrent = () => {
        if (current === placeholder) {
            return;
        }
        current.shownBy.delete(adapter);
        if (current.activeEditor === adapter) {
            current.selection = { ...local.selection };
            current.selectionVersion += 1;
            current.activeEditor = null;
        }
        selectionVersions.set(current, current.selectionVersion);
    };

    const emitModelChange = (target) => {
        target.onChange?.();
        if (target === current && typeof options.onChange === 'function') {
            options.onChange(target.value);
        }
    };
    const emitChange = () => emitModelChange(current);

    const emitSelection = () => {
        if (typeof options.onSelectionChange === 'function') {
            options.onSelectionChange(local.selection.start, local.selection.end, local.selection.active);
        }
    };

    const select = (start, end, active = end) => {
        const safeStart = clampOffset(start, local.value.length);
        const safeEnd = Math.max(safeStart, clampOffset(end ?? safeStart, local.value.length));
        local.selection = { start: safeStart, end: safeEnd, active: clampOffset(active, local.value.length) };
        emitSelection();
    };

    const adapter = {
        options,
        container,
        // Stands in for the Monaco model that LSP provider callbacks receive.
        get model() {
            const shown = current;
            return { uri: { toString: () => shown.uri }, getValue: () => shown.value };
        },
        lspCallbacks: null,
        diagnostics: [],
        typosMisspellings: [],
        wordWrap: false,
        typography: null,
        theme: null,
        focusCount: 0,

        createDocumentModel({ value = '', language = 'plaintext', onChange } = {}) {
            const target = createFakeModel(value, language, onChange);
            const active = () => target.activeEditor;
            return {
                fakeModel: target,
                getValue: () => target.value,
                setValue(next) {
                    if (active()) {
                        active().setValue(next);
                        return;
                    }
                    const text = String(next || '');
                    if (text !== target.value) {
                        target.value = text;
                        target.undoStack = [];
                        target.selection = { start: 0, end: 0, active: 0 };
                        target.selectionVersion += 1;
                        emitModelChange(target);
                    }
                },
                getSelectionOffsets: () => (active()
                    ? active().getSelectionOffsets()
                    : { start: target.selection.start, end: target.selection.end }),
                setSelectionOffsets(start, end) {
                    if (active()) {
                        active().setSelectionOffsets(start, end);
                        return;
                    }
                    const safeStart = clampOffset(start, target.value.length);
                    const safeEnd = Math.max(safeStart, clampOffset(end ?? safeStart, target.value.length));
                    target.selection = { start: safeStart, end: safeEnd, active: safeEnd };
                    target.selectionVersion += 1;
                },
                replaceRange(start, end, text) {
                    if (active()) {
                        active().replaceRange(start, end, text);
                        return;
                    }
                    const safeStart = clampOffset(start, target.value.length);
                    const safeEnd = Math.max(safeStart, clampOffset(end ?? safeStart, target.value.length));
                    target.undoStack.push(target.value);
                    target.value = target.value.slice(0, safeStart) + String(text || '') + target.value.slice(safeEnd);
                    emitModelChange(target);
                },
                setLanguage(next) {
                    target.language = String(next || 'plaintext');
                },
                dispose() {
                    target.disposed = true;
                    for (const editor of [...target.shownBy]) {
                        editor.showModel(null);
                    }
                },
            };
        },
        showModel(handle) {
            const next = handle?.fakeModel || placeholder;
            if (next === current) {
                if (handle) {
                    next.activeEditor = adapter;
                }
                return;
            }
            releaseCurrent();
            current = next;
            if (handle) {
                next.shownBy.add(adapter);
                next.activeEditor = adapter;
                if (selectionVersions.get(next) !== next.selectionVersion) {
                    selections.set(next, { ...next.selection });
                }
            }
        },
        getValue: () => local.value,
        setValue(next) {
            const text = String(next || '');
            if (text === local.value) {
                return;
            }
            local.value = text;
            local.undoStack = [];
            local.selection = { start: 0, end: 0, active: 0 };
            emitChange();
        },
        getLanguage: () => local.language,
        setLanguage(next) {
            local.language = String(next || 'plaintext');
        },
        setDiagnostics(list) {
            adapter.diagnostics = Array.isArray(list) ? list : [];
        },
        setTyposMisspellings(list) {
            adapter.typosMisspellings = Array.isArray(list) ? list : [];
        },
        refreshInlayHints() {},
        getModelUri: () => current.uri,
        configureLsp(callbacks = {}) {
            adapter.lspCallbacks = callbacks;
        },
        getSelectionOffsets: () => ({ start: local.selection.start, end: local.selection.end }),
        setSelectionOffsets(start, end) {
            select(start, end);
        },
        getSelectionText: () => local.value.slice(local.selection.start, local.selection.end),
        replaceRange(start, end, text) {
            const safeStart = clampOffset(start, local.value.length);
            const safeEnd = Math.max(safeStart, clampOffset(end ?? safeStart, local.value.length));
            local.undoStack.push(local.value);
            local.value = local.value.slice(0, safeStart) + String(text || '') + local.value.slice(safeEnd);
            emitChange();
        },
        undo() {
            if (local.undoStack.length === 0) {
                return false;
            }
            local.value = local.undoStack.pop();
            emitChange();
            return true;
        },
        findMatches(query, findOptions = {}) {
            const search = String(query || '');
            if (!search) {
                return [];
            }
            const source = findOptions.isRegex === true
                ? search
                : search.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
            let pattern;
            try {
                pattern = new RegExp(source, findOptions.matchCase === true ? 'g' : 'gi');
            } catch {
                return [];
            }
            const limit = Number(findOptions.limit) || 10000;
            const matches = [];
            for (const match of local.value.matchAll(pattern)) {
                if (match[0].length === 0) {
                    continue;
                }
                matches.push({ start: match.index, end: match.index + match[0].length });
                if (matches.length >= limit) {
                    break;
                }
            }
            return matches;
        },
        saveViewState() {
            return { selection: { ...local.selection } };
        },
        restoreViewState(viewState) {
            if (viewState?.selection) {
                local.selection = { ...viewState.selection };
            }
        },
        revealOffset() {},
        getCursorViewportPoint: () => ({ x: 0, y: 0 }),
        focus() {
            adapter.focusCount += 1;
        },
        openCommandPalette() {},
        layout() {},
        setWordWrap(enabled) {
            adapter.wordWrap = Boolean(enabled);
        },
        setTypography(next) {
            adapter.typography = next;
        },
        applyTheme(colors) {
            adapter.theme = colors;
        },
        dispose() {
            releaseCurrent();
            disposed = true;
        },
        isDisposed: () => disposed,

        // Test helpers: simulate what a user does inside Monaco.
        typeText(text) {
            const insert = String(text || '');
            const start = local.selection.start;
            adapter.replaceRange(local.selection.start, local.selection.end, insert);
            const caret = start + insert.length;
            select(caret, caret);
            // Monaco's onDidType, which the adapter forwards as a 'type' request.
            options.onCompletionRequest?.({ source: 'type', key: insert.slice(-1) });
        },
        moveCursor(offset) {
            select(offset, offset);
        },
        pressEscape() {
            return typeof options.onEscape === 'function' ? Boolean(options.onEscape()) : false;
        },
        requestCompletion(payload) {
            return options.onCompletionRequest?.(payload);
        },
        contextMenu(event = new MouseEvent('contextmenu', { bubbles: true, cancelable: true })) {
            options.onContextMenu?.(event);
            return event;
        },
    };

    return adapter;
}

// Stand-in for createMonacoDiffAdapter: the modified side behaves like the
// main editor fake, and the original side is kept for assertions.
export function createFakeMonacoDiffAdapter(container, options = {}) {
    const adapter = createFakeMonacoAdapter(container, options);
    adapter.diff = null;
    adapter.setDiff = (diff) => {
        adapter.diff = diff;
    };
    adapter.getModifiedValue = adapter.getValue;
    adapter.setModifiedValue = adapter.setValue;
    return adapter;
}
