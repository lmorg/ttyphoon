// The text of the open Notes document (ADR 0100). Monaco is created lazily, so
// until a view is attached the buffer holds the text and selection itself.
// Once a view (the Monaco adapter) is attached, reads and edits go to the view,
// so edits made through the buffer land on Monaco's undo stack.
//
// Edits (replaceRange/replaceAll) notify change listeners. setValue is a reset,
// such as loading a file: it is not undoable and does not notify.

export function normaliseLineEndings(text) {
    return String(text || '').replace(/\r\n?/g, '\n');
}

function clamp(value, length) {
    return Math.max(0, Math.min(Number(value) || 0, length));
}

export function createDocumentBuffer(initialText = '') {
    let text = normaliseLineEndings(initialText);
    let selection = { start: 0, end: 0 };
    let view = null;
    let suppressViewChanges = 0;
    const listeners = new Set();

    function emit(source) {
        for (const listener of [...listeners]) {
            listener({ source });
        }
    }

    function getValue() {
        return view ? String(view.getValue() || '') : text;
    }

    function getSelection() {
        if (view) {
            const offsets = view.getSelectionOffsets();
            return { start: Number(offsets?.start) || 0, end: Number(offsets?.end) || 0 };
        }
        return { ...selection };
    }

    function setSelection(start, end = start) {
        if (view) {
            view.setSelectionOffsets(start, end);
            return;
        }
        const safeStart = clamp(start, text.length);
        const safeEnd = Math.max(safeStart, clamp(end, text.length));
        selection = { start: safeStart, end: safeEnd };
    }

    function setValue(nextText) {
        text = normaliseLineEndings(nextText);
        selection = { start: 0, end: 0 };
        if (view) {
            suppressViewChanges += 1;
            try {
                view.setValue(text);
            } finally {
                suppressViewChanges -= 1;
            }
        }
    }

    // By default the caret moves to the end of the inserted text, as when
    // typing. preserveSelection keeps the selection where it was, mapped
    // through the edit, for edits made from another view.
    function replaceRange(start, end, insert, { source = 'notes-document-edit', cursor, preserveSelection = false } = {}) {
        const current = getValue();
        const safeStart = clamp(start, current.length);
        const safeEnd = Math.max(safeStart, clamp(end, current.length));
        const insertText = normaliseLineEndings(insert);
        let nextSelection;
        if (preserveSelection) {
            const delta = insertText.length - (safeEnd - safeStart);
            const map = (offset) => {
                if (offset <= safeStart) {
                    return offset;
                }
                return offset >= safeEnd ? offset + delta : safeStart;
            };
            const previous = getSelection();
            nextSelection = { start: map(previous.start), end: map(previous.end) };
        } else {
            const nextCursor = cursor ?? safeStart + insertText.length;
            nextSelection = { start: nextCursor, end: nextCursor };
        }

        if (view) {
            // The view's change event reaches listeners via handleViewChange.
            view.replaceRange(safeStart, safeEnd, insertText, source);
            setSelection(nextSelection.start, nextSelection.end);
            return;
        }

        text = current.slice(0, safeStart) + insertText + current.slice(safeEnd);
        setSelection(nextSelection.start, nextSelection.end);
        emit(source);
    }

    // Replace the whole document as the smallest range edit, so undo restores
    // only what changed and an unchanged document is not marked as edited.
    function replaceAll(nextText, { source = 'notes-document-edit', selectionStart, selectionEnd } = {}) {
        const current = getValue();
        const next = normaliseLineEndings(nextText);
        if (current !== next) {
            let prefix = 0;
            const maxPrefix = Math.min(current.length, next.length);
            while (prefix < maxPrefix && current[prefix] === next[prefix]) {
                prefix += 1;
            }
            let suffix = 0;
            const maxSuffix = Math.min(current.length, next.length) - prefix;
            while (suffix < maxSuffix && current[current.length - 1 - suffix] === next[next.length - 1 - suffix]) {
                suffix += 1;
            }
            replaceRange(prefix, current.length - suffix, next.slice(prefix, next.length - suffix), { source });
        }
        if (selectionStart !== undefined) {
            setSelection(selectionStart, selectionEnd ?? selectionStart);
        }
    }

    function attachView(nextView) {
        const pending = { value: text, selection: { ...selection } };
        view = nextView;
        if (String(view.getValue() || '') !== pending.value) {
            suppressViewChanges += 1;
            try {
                view.setValue(pending.value);
            } finally {
                suppressViewChanges -= 1;
            }
        }
        view.setSelectionOffsets(pending.selection.start, pending.selection.end);
    }

    // Called by the attached view's own change event (typing, undo, edits).
    function handleViewChange(source = 'view') {
        if (suppressViewChanges > 0) {
            return;
        }
        emit(source);
    }

    function onChange(listener) {
        listeners.add(listener);
        return () => listeners.delete(listener);
    }

    return {
        getValue,
        setValue,
        getSelection,
        setSelection,
        replaceRange,
        replaceAll,
        attachView,
        handleViewChange,
        hasView: () => view !== null,
        onChange,
    };
}
