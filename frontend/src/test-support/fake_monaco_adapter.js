// A string-backed stand-in for createMonacoAdapter, for jsdom tests where
// Monaco cannot run. It implements the adapter API that notes.js uses and
// fires onChange/onSelectionChange the way the real adapter does, so tests
// exercise the same Monaco code paths as the app.

function clampOffset(value, length) {
    return Math.max(0, Math.min(Number(value) || 0, length));
}

export function createFakeMonacoAdapter(container, options = {}) {
    let value = String(options.value || '');
    let selection = { start: 0, end: 0, active: 0 };
    let language = String(options.language || 'plaintext');
    let disposed = false;

    const emitChange = () => {
        if (typeof options.onChange === 'function') {
            options.onChange(value);
        }
    };

    const emitSelection = () => {
        if (typeof options.onSelectionChange === 'function') {
            options.onSelectionChange(selection.start, selection.end, selection.active);
        }
    };

    const select = (start, end, active = end) => {
        const safeStart = clampOffset(start, value.length);
        const safeEnd = Math.max(safeStart, clampOffset(end ?? safeStart, value.length));
        selection = { start: safeStart, end: safeEnd, active: clampOffset(active, value.length) };
        emitSelection();
    };

    const adapter = {
        options,
        container,
        lspCallbacks: null,
        diagnostics: [],
        typosMisspellings: [],
        wordWrap: false,
        typography: null,
        theme: null,
        focusCount: 0,

        getValue: () => value,
        setValue(next) {
            const text = String(next || '');
            if (text === value) {
                return;
            }
            value = text;
            selection = { start: 0, end: 0, active: 0 };
            emitChange();
        },
        getLanguage: () => language,
        setLanguage(next) {
            language = String(next || 'plaintext');
        },
        setDiagnostics(list) {
            adapter.diagnostics = Array.isArray(list) ? list : [];
        },
        setTyposMisspellings(list) {
            adapter.typosMisspellings = Array.isArray(list) ? list : [];
        },
        refreshInlayHints() {},
        configureLsp(callbacks = {}) {
            adapter.lspCallbacks = callbacks;
        },
        getSelectionOffsets: () => ({ start: selection.start, end: selection.end }),
        setSelectionOffsets(start, end) {
            select(start, end);
        },
        getSelectionText: () => value.slice(selection.start, selection.end),
        replaceRange(start, end, text) {
            const safeStart = clampOffset(start, value.length);
            const safeEnd = Math.max(safeStart, clampOffset(end ?? safeStart, value.length));
            value = value.slice(0, safeStart) + String(text || '') + value.slice(safeEnd);
            emitChange();
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
            for (const match of value.matchAll(pattern)) {
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
            disposed = true;
        },
        isDisposed: () => disposed,

        // Test helpers: simulate what a user does inside Monaco.
        typeText(text) {
            const insert = String(text || '');
            const start = selection.start;
            adapter.replaceRange(selection.start, selection.end, insert);
            const caret = start + insert.length;
            select(caret, caret);
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
