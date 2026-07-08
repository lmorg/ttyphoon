const TEXT_INPUT_TYPES = new Set([
    '',
    'text',
    'search',
    'url',
    'tel',
    'password',
    'email',
]);

function isEligibleTextInput(target) {
    if (!(target instanceof HTMLInputElement)) {
        return false;
    }

    const inputType = String(target.type || '').toLowerCase();
    if (!TEXT_INPUT_TYPES.has(inputType)) {
        return false;
    }

    return true;
}

function isEligibleTarget(target) {
    if (!(target instanceof Element)) {
        return false;
    }

    // Monaco manages its own Home/End navigation via its hidden internal
    // <textarea class="inputarea">. Intercepting it here steals the first
    // keypress (moving the caret inside the hidden textarea) and forces a
    // double-tap, so let Monaco handle these keys natively.
    if (target.closest('.monaco-editor')) {
        return false;
    }

    if (target instanceof HTMLTextAreaElement || isEligibleTextInput(target)) {
        return !target.disabled && !target.readOnly;
    }

    return false;
}

function getCaretPosition(target) {
    const direction = String(target.selectionDirection || 'none').toLowerCase();
    if (direction === 'backward') {
        return target.selectionStart;
    }

    return target.selectionEnd;
}

function getLineBoundary(value, caretPos, key) {
    const safeCaret = Math.max(0, Math.min(caretPos, value.length));
    const lineStart = value.lastIndexOf('\n', Math.max(0, safeCaret - 1)) + 1;
    const lineEndIndex = value.indexOf('\n', safeCaret);
    const lineEnd = lineEndIndex === -1 ? value.length : lineEndIndex;

    return key === 'Home' ? lineStart : lineEnd;
}

function setSelection(target, nextFocus, withShift) {
    if (!withShift) {
        target.setSelectionRange(nextFocus, nextFocus, 'none');
        return;
    }

    const direction = String(target.selectionDirection || 'none').toLowerCase();
    const anchor = direction === 'backward' ? target.selectionEnd : target.selectionStart;
    const start = Math.min(anchor, nextFocus);
    const end = Math.max(anchor, nextFocus);
    const nextDirection = nextFocus < anchor ? 'backward' : 'forward';

    target.setSelectionRange(start, end, nextDirection);
}

function shouldHandleKey(event) {
    if (!event || event.defaultPrevented || event.isComposing) {
        return false;
    }

    if (event.ctrlKey || event.metaKey || event.altKey) {
        return false;
    }

    return event.key === 'Home' || event.key === 'End';
}

export function initLineNavigationKeys(root = document) {
    if (!root || root.__ttyphoonLineNavInit === true) {
        return;
    }

    root.__ttyphoonLineNavInit = true;

    root.addEventListener('keydown', (event) => {
        if (!shouldHandleKey(event) || !isEligibleTarget(event.target)) {
            return;
        }

        const target = event.target;
        const currentCaret = getCaretPosition(target);
        const nextFocus = getLineBoundary(target.value || '', currentCaret, event.key);

        if (nextFocus === currentCaret && !event.shiftKey) {
            return;
        }

        event.preventDefault();
        event.stopPropagation();
        setSelection(target, nextFocus, event.shiftKey);
    }, true);
}
