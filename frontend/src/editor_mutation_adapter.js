function clampSelection(value, start, end) {
    const len = String(value || '').length;
    const safeStart = Math.max(0, Math.min(Number(start) || 0, len));
    const safeEnd = Math.max(safeStart, Math.min(Number(end) || safeStart, len));
    return { safeStart, safeEnd, len };
}

function captureState(textarea) {
    return {
        text: String(textarea?.value || ''),
        selectionStart: Number(textarea?.selectionStart) || 0,
        selectionEnd: Number(textarea?.selectionEnd) || 0,
    };
}

function emitInput(textarea) {
    textarea.dispatchEvent(new Event('input', { bubbles: true }));
}

function buildTransaction(meta, before, after, groupId) {
    return {
        filePath: String(meta.filePath || ''),
        source: String(meta.source || ''),
        label: String(meta.label || ''),
        beforeText: before.text,
        beforeSelectionStart: before.selectionStart,
        beforeSelectionEnd: before.selectionEnd,
        afterText: after.text,
        afterSelectionStart: after.selectionStart,
        afterSelectionEnd: after.selectionEnd,
        groupId,
    };
}

export function createEditorMutationAdapter({ manager, getFilePath } = {}) {
    function resolveFilePath(textarea, explicitFilePath) {
        if (String(explicitFilePath || '').trim() !== '') {
            return String(explicitFilePath);
        }
        if (typeof getFilePath === 'function') {
            return String(getFilePath(textarea) || '');
        }
        return String(textarea?.dataset?.filePath || '');
    }

    function applySnapshot(textarea, tx, direction = 'undo', emit = true) {
        if (!textarea || !tx) {
            return;
        }

        const toAfter = direction === 'redo';
        const nextText = toAfter ? tx.afterText : tx.beforeText;
        const nextStart = toAfter ? tx.afterSelectionStart : tx.beforeSelectionStart;
        const nextEnd = toAfter ? tx.afterSelectionEnd : tx.beforeSelectionEnd;

        textarea.value = String(nextText || '');
        const { safeStart, safeEnd } = clampSelection(textarea.value, nextStart, nextEnd);
        try {
            textarea.setSelectionRange(safeStart, safeEnd);
        } catch {
            // Detached nodes / jsdom edge cases.
        }

        if (emit) {
            emitInput(textarea);
        }
    }

    function replaceRange(textarea, options = {}) {
        if (!textarea) {
            return null;
        }

        const before = captureState(textarea);
        const {
            start = before.selectionStart,
            end = before.selectionEnd,
            text = '',
            cursor,
            emit = true,
            groupId,
            filePath,
            source = 'programmatic-edit',
            label = 'Replace range',
        } = options;

        const { safeStart, safeEnd } = clampSelection(before.text, start, end);
        textarea.setRangeText(String(text || ''), safeStart, safeEnd, 'preserve');

        const defaultCursor = safeStart + String(text || '').length;
        const { safeStart: nextCursor } = clampSelection(textarea.value, cursor ?? defaultCursor, cursor ?? defaultCursor);
        try {
            textarea.setSelectionRange(nextCursor, nextCursor);
        } catch {
            // Detached nodes / jsdom edge cases.
        }

        if (emit) {
            emitInput(textarea);
        }

        const after = captureState(textarea);
        return manager?.record?.(buildTransaction({
            filePath: resolveFilePath(textarea, filePath),
            source,
            label,
        }, before, after, groupId)) || null;
    }

    function replaceDocumentText(textarea, options = {}) {
        if (!textarea) {
            return null;
        }

        const before = captureState(textarea);
        const {
            text = '',
            selectionStart = 0,
            selectionEnd = selectionStart,
            emit = true,
            groupId,
            filePath,
            source = 'programmatic-edit',
            label = 'Replace document text',
        } = options;

        textarea.value = String(text || '');
        const { safeStart, safeEnd } = clampSelection(textarea.value, selectionStart, selectionEnd);
        try {
            textarea.setSelectionRange(safeStart, safeEnd);
        } catch {
            // Detached nodes / jsdom edge cases.
        }

        if (emit) {
            emitInput(textarea);
        }

        const after = captureState(textarea);
        return manager?.record?.(buildTransaction({
            filePath: resolveFilePath(textarea, filePath),
            source,
            label,
        }, before, after, groupId)) || null;
    }

    function insertText(textarea, options = {}) {
        if (!textarea) {
            return null;
        }

        const before = captureState(textarea);
        return replaceRange(textarea, {
            ...options,
            start: before.selectionStart,
            end: before.selectionEnd,
            cursor: options.cursor,
            label: options.label || 'Insert text',
            source: options.source || 'insert-text',
        });
    }

    function deleteRange(textarea, options = {}) {
        if (!textarea) {
            return null;
        }

        return replaceRange(textarea, {
            ...options,
            text: '',
            label: options.label || 'Delete range',
            source: options.source || 'delete-range',
        });
    }

    return {
        replaceRange,
        replaceDocumentText,
        insertText,
        deleteRange,
        applySnapshot,
    };
}
