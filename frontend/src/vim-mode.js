/**
 * vim-mode.js — Vim-style keyboard navigation for <textarea> elements.
 *
 * Usage:
 *   import { attachVimMode } from './vim-mode.js';
 *   const vim = attachVimMode(textareaElement);   // returns a detach() fn
 *   vim.detach();                                  // remove all listeners
 *
 * Modes
 *   INSERT  — default; text flows normally.  Esc → NORMAL.
 *   NORMAL  — navigation / operator mode.   i/a/I/A/o/O → INSERT.
 *   REPLACE — R was pressed; every char overtypes.  Esc → NORMAL.
 *   REPLACE_ONCE — r was pressed; next char overtypes then returns to NORMAL.
 *
 * Supported in NORMAL mode:
 *   Motions (standalone or after operator):
 *     h j k l   ← ↓ ↑ →  (character / line)
 *     w b e     word-forward / word-back / end-of-word
 *     0 ^ $     start-of-line (col 0) / first non-blank / end-of-line
 *     gg G      start-of-file / end-of-file
 *     %         jump to matching bracket
 *
 *   Operators (take a motion or count):
 *     d{motion} — delete
 *     c{motion} — change (delete + enter INSERT)
 *     y{motion} — yank (copies to internal buffer; does NOT touch clipboard)
 *     p P       — paste internal buffer after / before cursor
 *     dd        — delete line
 *     cc        — change line (delete line body + INSERT)
 *     yy        — yank line
 *     D         — delete to end of line
 *     C         — change to end of line
 *     Y         — yank to end of line
 *     x X       — delete char under / before cursor
 *
 *   Enter INSERT:
 *     i I a A o O
 *
 *   Replace:
 *     r{char}   — replace single char, stay in NORMAL
 *     R         — enter REPLACE mode (overtype until Esc)
 *
 *   Number prefixes: any leading digits before a motion/operator multiply it.
 *     e.g. 3w  3dd  5j
 *
 * Indicator:
 *   A small floating badge is injected immediately after the textarea (or
 *   inside the nearest positioned ancestor if needed).  It shows the current
 *   mode name while not in INSERT.  You can style it via the CSS class
 *   `vim-mode-indicator`.
 */

// ─── constants ───────────────────────────────────────────────────────────────

const MODE_INSERT       = 'insert';
const MODE_NORMAL       = 'normal';
const MODE_REPLACE      = 'replace';
const MODE_REPLACE_ONCE = 'replace-once';

const MODE_LABELS = {
    [MODE_INSERT]:       '',
    [MODE_NORMAL]:       '-- VIM KEYS --',
    [MODE_REPLACE]:      '-- REPLACE --',
    [MODE_REPLACE_ONCE]: '-- REPLACE (r) --',
};

// ─── indicator DOM ───────────────────────────────────────────────────────────

function createIndicator() {
    const el = document.createElement('div');
    el.className = 'vim-mode-indicator';
    el.setAttribute('aria-live', 'polite');
    el.setAttribute('aria-atomic', 'true');
    el.style.cssText = [
        'position:fixed',
        'padding:2px 8px',
        'border-radius:3px',
        'font-size:0.75em',
        'font-family:monospace',
        'font-weight:bold',
        'pointer-events:none',
        'user-select:none',
        'z-index:10000',
        'opacity:0',
        'transition:opacity 0.1s',
        'background:var(--bg,#1e2228)',
        'color:var(--accent,#588acc)',
        'border:1px solid var(--accent,#588acc)',
        'white-space:nowrap',
    ].join(';');
    document.body.appendChild(el);
    return el;
}

/**
 * Measure the pixel position of the caret inside a textarea.
 * Returns { x, y, lineHeight } in viewport coordinates.
 */
function getCaretCoords(textarea) {
    const cs = getComputedStyle(textarea);
    // When the textarea doesn't wrap (pre / nowrap), the mirror must also not
    // wrap — otherwise it creates phantom visual lines and the vertical offset drifts.
    const isWrapping = !['nowrap', 'pre'].includes(cs.whiteSpace);

    const mirror = document.createElement('div');
    const styleProps = [
        'position:fixed',
        'top:-9999px',
        'left:-9999px',
        'visibility:hidden',
        isWrapping ? 'white-space:pre-wrap' : 'white-space:pre',
        `font:${cs.font}`,
        `font-size:${cs.fontSize}`,
        `font-family:${cs.fontFamily}`,
        `font-weight:${cs.fontWeight}`,
        `line-height:${cs.lineHeight}`,
        `letter-spacing:${cs.letterSpacing}`,
        `padding:${cs.padding}`,
        `border:${cs.border}`,
        `box-sizing:${cs.boxSizing}`,
        `tab-size:${cs.tabSize}`,
    ];
    if (isWrapping) {
        // Constrain width so the mirror wraps exactly like the textarea does.
        styleProps.push(
            `width:${textarea.clientWidth}px`,
            'word-wrap:break-word',
            'overflow-wrap:break-word',
        );
    }
    mirror.style.cssText = styleProps.join(';');

    const pos = textarea.selectionStart || 0;
    const before = textarea.value.slice(0, pos);

    const textNode = document.createTextNode(before);
    mirror.appendChild(textNode);

    const caret = document.createElement('span');
    caret.textContent = '\u200b'; // zero-width space — marks the caret position
    mirror.appendChild(caret);

    document.body.appendChild(mirror);

    const taRect   = textarea.getBoundingClientRect();
    const spanRect = caret.getBoundingClientRect();

    // Compute scroll-adjusted position relative to the textarea viewport
    const x = taRect.left + (spanRect.left - mirror.getBoundingClientRect().left) - textarea.scrollLeft;
    const y = taRect.top  + (spanRect.top  - mirror.getBoundingClientRect().top)  - textarea.scrollTop;

    const lineHeight = parseFloat(cs.lineHeight) || parseFloat(cs.fontSize) * 1.2;

    document.body.removeChild(mirror);

    return { x, y, lineHeight };
}

function positionIndicator(el, textarea) {
    if (el.style.opacity === '0') return; // hidden — no need to measure
    const { x, y, lineHeight } = getCaretCoords(textarea);
    const taRect = textarea.getBoundingClientRect();

    // Clamp so the badge stays within the textarea bounds
    const badgeW = el.offsetWidth  || 120;
    const badgeH = el.offsetHeight || 20;

    const rawLeft = x;
    const rawTop  = y + lineHeight + 2; // just below the current line

    const left = Math.min(Math.max(rawLeft, taRect.left), taRect.right  - badgeW);
    const top  = rawTop + badgeH > taRect.bottom
        ? y - badgeH - 2   // flip above caret if it would overflow below
        : rawTop;

    el.style.left = `${Math.round(left)}px`;
    el.style.top  = `${Math.round(top)}px`;
}

function updateIndicator(el, textarea, mode) {
    const label = MODE_LABELS[mode] ?? '';
    el.textContent = label;
    el.style.opacity = label ? '1' : '0';
    if (label) positionIndicator(el, textarea);
}

// ─── text helpers ─────────────────────────────────────────────────────────────

function getLines(text) {
    return text.split('\n');
}

/** Return {line, col} from a flat offset. */
function offsetToLineCol(text, offset) {
    const before = text.slice(0, offset);
    const line = (before.match(/\n/g) || []).length;
    const col  = offset - (before.lastIndexOf('\n') + 1);
    return { line, col };
}

/** Return flat offset from {line, col}. */
function lineColToOffset(text, line, col) {
    const lines = getLines(text);
    let off = 0;
    for (let i = 0; i < line && i < lines.length; i++) {
        off += lines[i].length + 1; // +1 for \n
    }
    const safeCol = Math.min(col, (lines[line] || '').length);
    return off + safeCol;
}

/** Clamp col to valid range for the given line (NORMAL mode keeps off EOL). */
function clampCol(lines, line, col, normalMode) {
    const len = (lines[line] || '').length;
    const max = normalMode ? Math.max(0, len - 1) : len;
    return Math.max(0, Math.min(col, max));
}

// ─── motion functions ─────────────────────────────────────────────────────────
// Each returns the new flat offset after applying `count` iterations.

function motionH(text, from, count) {
    const { line, col } = offsetToLineCol(text, from);
    const lines = getLines(text);
    const newCol = Math.max(0, col - count);
    return lineColToOffset(text, line, newCol);
}

function motionL(text, from, count) {
    const { line, col } = offsetToLineCol(text, from);
    const lines = getLines(text);
    const lineLen = (lines[line] || '').length;
    const newCol = Math.min(lineLen - 1, col + count); // stay on last char in normal
    return lineColToOffset(text, line, Math.max(0, newCol));
}

function motionJ(text, from, count, wantCol) {
    const { line, col } = offsetToLineCol(text, from);
    const lines = getLines(text);
    const newLine = Math.min(lines.length - 1, line + count);
    const newCol  = clampCol(lines, newLine, wantCol ?? col, true);
    return lineColToOffset(text, newLine, newCol);
}

function motionK(text, from, count, wantCol) {
    const { line, col } = offsetToLineCol(text, from);
    const lines = getLines(text);
    const newLine = Math.max(0, line - count);
    const newCol  = clampCol(lines, newLine, wantCol ?? col, true);
    return lineColToOffset(text, newLine, newCol);
}

function motionW(text, from, count) {
    let pos = from;
    for (let i = 0; i < count; i++) {
        // skip current word chars
        while (pos < text.length && /\S/.test(text[pos]) && text[pos] !== '\n') pos++;
        // skip whitespace (including newlines)
        while (pos < text.length && /\s/.test(text[pos])) pos++;
    }
    return Math.min(pos, text.length - 1);
}

/** W — WORD forward: skip any non-whitespace, then skip whitespace. */
function motionWW(text, from, count) {
    let pos = from;
    for (let i = 0; i < count; i++) {
        while (pos < text.length && !/\s/.test(text[pos])) pos++;
        while (pos < text.length &&  /\s/.test(text[pos])) pos++;
    }
    return Math.min(pos, text.length - 1);
}

function motionB(text, from, count) {
    let pos = from;
    for (let i = 0; i < count; i++) {
        if (pos > 0) pos--;
        // skip whitespace backwards
        while (pos > 0 && /\s/.test(text[pos])) pos--;
        // skip word chars backwards
        while (pos > 0 && /\S/.test(text[pos - 1]) && text[pos - 1] !== '\n') pos--;
    }
    return Math.max(0, pos);
}

/** B — WORD back: skip whitespace backwards, then skip any non-whitespace. */
function motionBW(text, from, count) {
    let pos = from;
    for (let i = 0; i < count; i++) {
        if (pos > 0) pos--;
        while (pos > 0 &&  /\s/.test(text[pos])) pos--;
        while (pos > 0 && !/\s/.test(text[pos - 1])) pos--;
    }
    return Math.max(0, pos);
}

function motionE(text, from, count) {
    let pos = from;
    for (let i = 0; i < count; i++) {
        if (pos < text.length - 1) pos++;
        // skip whitespace
        while (pos < text.length - 1 && /\s/.test(text[pos])) pos++;
        // advance to end of word
        while (pos < text.length - 1 && /\S/.test(text[pos + 1]) && text[pos + 1] !== '\n') pos++;
    }
    return pos;
}

/** E — WORD end forward: skip whitespace, then skip any non-whitespace. */
function motionEW(text, from, count) {
    let pos = from;
    for (let i = 0; i < count; i++) {
        if (pos < text.length - 1) pos++;
        while (pos < text.length - 1 &&  /\s/.test(text[pos])) pos++;
        while (pos < text.length - 1 && !/\s/.test(text[pos + 1])) pos++;
    }
    return pos;
}

function motionLineStart(text, from) {
    const { line } = offsetToLineCol(text, from);
    return lineColToOffset(text, line, 0);
}

function motionLineFirstNonBlank(text, from) {
    const { line } = offsetToLineCol(text, from);
    const lines = getLines(text);
    const lineText = lines[line] || '';
    const col = lineText.search(/\S/);
    return lineColToOffset(text, line, col >= 0 ? col : 0);
}

function motionLineEnd(text, from) {
    const { line } = offsetToLineCol(text, from);
    const lines = getLines(text);
    const lineLen = (lines[line] || '').length;
    return lineColToOffset(text, line, Math.max(0, lineLen - 1));
}

function motionFileStart(text) {
    // first non-blank of first line
    const firstNonBlank = text.search(/\S/);
    if (firstNonBlank < 0 || text[firstNonBlank] === '\n') return 0;
    return firstNonBlank;
}

function motionFileEnd(text) {
    const lines = getLines(text);
    const lastLine = lines.length - 1;
    const lineText = lines[lastLine] || '';
    const col = lineText.search(/\S/);
    return lineColToOffset(text, lastLine, col >= 0 ? col : 0);
}

function motionMatchBracket(text, from) {
    const open  = '([{';
    const close = ')]}';
    const ch = text[from];
    const openIdx = open.indexOf(ch);
    const closeIdx = close.indexOf(ch);

    if (openIdx >= 0) {
        // scan forward
        let depth = 1;
        for (let i = from + 1; i < text.length; i++) {
            if (text[i] === open[openIdx]) depth++;
            else if (text[i] === close[openIdx]) { depth--; if (depth === 0) return i; }
        }
    } else if (closeIdx >= 0) {
        // scan backward
        let depth = 1;
        for (let i = from - 1; i >= 0; i--) {
            if (text[i] === close[closeIdx]) depth++;
            else if (text[i] === open[closeIdx]) { depth--; if (depth === 0) return i; }
        }
    }
    return from; // no match
}

// ─── range helpers (for operators) ───────────────────────────────────────────

/** Return the inclusive text range [start, end) for deleting to a motion. */
function rangeForMotion(text, from, motionFn) {
    const to = motionFn(text, from);
    return from <= to ? [from, to] : [to, from + 1];
}

/** Expand a range to cover full lines (for dd, yy, etc.) */
function fullLineRange(text, from) {
    const lines = getLines(text);
    const { line } = offsetToLineCol(text, from);
    const lineStart = lineColToOffset(text, line, 0);
    const isLastLine = line >= lines.length - 1;
    if (isLastLine) {
        // delete the newline before this line too (if any)
        const start = line > 0 ? lineStart - 1 : lineStart;
        return [start, text.length];
    }
    return [lineStart, lineStart + (lines[line] || '').length + 1];
}

/** Range from cursor to end of line (D). */
function rangeToLineEnd(text, from) {
    const { line } = offsetToLineCol(text, from);
    const lines = getLines(text);
    const lineEnd = lineColToOffset(text, line, (lines[line] || '').length);
    return [from, lineEnd];
}

/**
 * Compute the range covering `n` whole lines starting at `from`.
 * Mirrors the deletion logic in fullLineRange but for n lines.
 */
function multiLineRange(text, from, n) {
    const allLines = getLines(text);
    const { line: startLine } = offsetToLineCol(text, from);
    const endLine = Math.min(startLine + n - 1, allLines.length - 1);
    const lineStart = lineColToOffset(text, startLine, 0);
    const isEndLastLine = endLine >= allLines.length - 1;
    if (isEndLastLine) {
        const start = startLine > 0 ? lineStart - 1 : lineStart;
        return [start, text.length];
    }
    // End = start of the line immediately after the last deleted line.
    const afterEnd = lineColToOffset(text, endLine + 1, 0);
    return [lineStart, afterEnd];
}

// ─── textarea mutation helpers ────────────────────────────────────────────────

function applyDelete(textarea, start, end) {
    const v = textarea.value;
    const newValue = v.slice(0, start) + v.slice(end);
    textarea.value = newValue;
    textarea.selectionStart = textarea.selectionEnd = start;
    textarea.dispatchEvent(new Event('input', { bubbles: true }));
    return newValue;
}

function applyInsert(textarea, pos, text) {
    const v = textarea.value;
    const newValue = v.slice(0, pos) + text + v.slice(pos);
    textarea.value = newValue;
    const newPos = pos + text.length;
    textarea.selectionStart = textarea.selectionEnd = newPos;
    textarea.dispatchEvent(new Event('input', { bubbles: true }));
    return newValue;
}

function applyReplace(textarea, pos, ch) {
    const v = textarea.value;
    if (pos >= v.length) return;
    const newValue = v.slice(0, pos) + ch + v.slice(pos + 1);
    textarea.value = newValue;
    textarea.selectionStart = textarea.selectionEnd = pos;
    textarea.dispatchEvent(new Event('input', { bubbles: true }));
}

function getCurrentPos(textarea) {
    return textarea.selectionStart || 0;
}

function setPos(textarea, pos) {
    const clamped = Math.max(0, Math.min(pos, textarea.value.length));
    textarea.selectionStart = textarea.selectionEnd = clamped;
}

// ─── main ─────────────────────────────────────────────────────────────────────

/**
 * Attach vim-mode to a textarea.
 * @param {HTMLTextAreaElement} textarea
 * @returns {{ detach: () => void }}
 */
export function attachVimMode(textarea) {
    let mode    = MODE_INSERT;
    let countBuf = '';       // digits typed before a key (e.g. "3" before "w")
    let operator = null;     // pending operator: 'd' | 'c' | 'y' | null
    let wantCol  = null;     // remembered column for j/k
    let yankBuf  = '';       // internal yank register

    const indicator = createIndicator();

    function setMode(m) {
        mode     = m;
        operator = null;
        countBuf = '';
        updateIndicator(indicator, textarea, m);
    }

    function getCount() {
        const n = parseInt(countBuf, 10);
        return (Number.isFinite(n) && n > 0) ? n : 1;
    }

    // ── apply an operator to a range ─────────────────────────────────────────
    function applyOperator(op, start, end, insertAfter) {
        if (op === 'y') {
            yankBuf = textarea.value.slice(start, end);
            setPos(textarea, start);
            setMode(MODE_NORMAL);
            return;
        }
        yankBuf = textarea.value.slice(start, end);
        applyDelete(textarea, start, end);

        if (op === 'd') {
            // clamp to valid NORMAL position
            const v = textarea.value;
            const { line } = offsetToLineCol(v, start);
            const lines = getLines(v);
            const clamped = lineColToOffset(v, line, clampCol(lines, line, start, true));
            setPos(textarea, clamped);
            setMode(MODE_NORMAL);
        } else if (op === 'c') {
            setMode(MODE_INSERT);
            if (typeof insertAfter === 'number') setPos(textarea, insertAfter);
            else setPos(textarea, start);
        }
    }

    // ── handle a motion key, optionally with a pending operator ──────────────
    function handleMotion(motionResult, isLinewise, insertPos) {
        const count  = getCount();
        const op     = operator;
        const from   = getCurrentPos(textarea);
        const text   = textarea.value;

        let newPos = motionResult;

        if (op) {
            const [start, end] = from <= newPos
                ? [from, newPos]
                : [newPos, from + 1];
            applyOperator(op, start, end, insertPos ?? start);
        } else {
            if (isLinewise) {
                wantCol = null;
            }
            setPos(textarea, newPos);
            setMode(MODE_NORMAL);
        }
    }

    // ── normal-mode key dispatcher ────────────────────────────────────────────
    function handleNormalKey(event) {
        const key  = event.key;
        const text = textarea.value;
        const from = getCurrentPos(textarea);
        const count = getCount();

        // Collect count digits (but not when we already have an operator
        // waiting — digit '0' at start of line has special meaning only
        // when countBuf is empty and operator is null).
        if (/^[1-9]$/.test(key) || (key === '0' && countBuf !== '')) {
            if (!operator || countBuf !== '') {
                // accumulate digit into count buffer
                countBuf += key;
                event.preventDefault();
                event.stopPropagation();
                return;
            }
        }

        // ── Enter INSERT mode ─────────────────────────────────────────────
        if (key === 'i') {
            setMode(MODE_INSERT);
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'I') {
            setPos(textarea, motionLineFirstNonBlank(text, from));
            setMode(MODE_INSERT);
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'a') {
            // insert after current char
            const lineLen = (getLines(text)[offsetToLineCol(text, from).line] || '').length;
            const col = offsetToLineCol(text, from).col;
            if (col < lineLen) setPos(textarea, from + 1);
            setMode(MODE_INSERT);
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'A') {
            const { line } = offsetToLineCol(text, from);
            const lines = getLines(text);
            setPos(textarea, lineColToOffset(text, line, (lines[line] || '').length));
            setMode(MODE_INSERT);
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'o') {
            const { line } = offsetToLineCol(text, from);
            const lines = getLines(text);
            const lineEnd = lineColToOffset(text, line, (lines[line] || '').length);
            applyInsert(textarea, lineEnd, '\n');
            setMode(MODE_INSERT);
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'O') {
            const { line } = offsetToLineCol(text, from);
            const lineStart = lineColToOffset(text, line, 0);
            applyInsert(textarea, lineStart, '\n');
            setPos(textarea, lineStart);
            setMode(MODE_INSERT);
            event.preventDefault();
            event.stopPropagation();
            return;
        }

        // ── Replace mode ──────────────────────────────────────────────────
        if (key === 'R') {
            setMode(MODE_REPLACE);
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'r') {
            setMode(MODE_REPLACE_ONCE);
            event.preventDefault();
            event.stopPropagation();
            return;
        }

        // ── Operators ─────────────────────────────────────────────────────
        if (key === 'd' || key === 'c' || key === 'y') {
            if (operator === key) {
                // doubled operator: dd / cc / yy — respects count prefix
                const [start, end] = multiLineRange(text, from, count);
                applyOperator(key, start, end, start);
                countBuf = '';
                event.preventDefault();
                event.stopPropagation();
                return;
            }
            operator = key;
            event.preventDefault();
            event.stopPropagation();
            return;
        }

        if (key === 'D') {
            const [start, end] = rangeToLineEnd(text, from);
            applyOperator('d', start, end);
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'C') {
            const [start, end] = rangeToLineEnd(text, from);
            applyOperator('c', start, end, start);
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'Y') {
            const [start, end] = fullLineRange(text, from);
            applyOperator('y', start, end);
            event.preventDefault();
            event.stopPropagation();
            return;
        }

        // ── x / X ────────────────────────────────────────────────────────
        if (key === 'x') {
            for (let i = 0; i < count && from < textarea.value.length; i++) {
                const pos = getCurrentPos(textarea);
                const v = textarea.value;
                const { line } = offsetToLineCol(v, pos);
                const lineLen = (getLines(v)[line] || '').length;
                const col = offsetToLineCol(v, pos).col;
                // Don't delete the newline
                if (col < lineLen) {
                    yankBuf = v[pos];
                    applyDelete(textarea, pos, pos + 1);
                }
            }
            // Clamp cursor to valid NORMAL position
            const v = textarea.value;
            const p = getCurrentPos(textarea);
            const { line } = offsetToLineCol(v, p);
            const lines = getLines(v);
            setPos(textarea, lineColToOffset(v, line, clampCol(lines, line, p, true)));
            setMode(MODE_NORMAL);
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'X') {
            for (let i = 0; i < count; i++) {
                const pos = getCurrentPos(textarea);
                if (pos > 0 && textarea.value[pos - 1] !== '\n') {
                    yankBuf = textarea.value[pos - 1];
                    applyDelete(textarea, pos - 1, pos);
                }
            }
            setMode(MODE_NORMAL);
            event.preventDefault();
            event.stopPropagation();
            return;
        }

        // ── Paste ─────────────────────────────────────────────────────────
        if (key === 'p') {
            if (yankBuf) {
                const isLine = yankBuf.endsWith('\n');
                let insertPos = from;
                if (isLine) {
                    const { line } = offsetToLineCol(text, from);
                    const lines = getLines(text);
                    // Insert after the newline that terminates the current line.
                    insertPos = Math.min(
                        lineColToOffset(text, line, (lines[line] || '').length) + 1,
                        text.length,
                    );
                } else {
                    // paste after cursor
                    insertPos = from + 1 <= text.length ? from + 1 : from;
                }
                applyInsert(textarea, insertPos, yankBuf);
                setPos(textarea, insertPos);
            }
            setMode(MODE_NORMAL);
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'P') {
            if (yankBuf) {
                const isLine = yankBuf.endsWith('\n');
                let insertPos = from;
                if (isLine) {
                    const { line } = offsetToLineCol(text, from);
                    insertPos = lineColToOffset(text, line, 0);
                }
                applyInsert(textarea, insertPos, yankBuf);
                setPos(textarea, insertPos);
            }
            setMode(MODE_NORMAL);
            event.preventDefault();
            event.stopPropagation();
            return;
        }

        // ── Motions ───────────────────────────────────────────────────────
        if (key === 'h' || key === 'ArrowLeft') {
            const newPos = motionH(text, from, count);
            wantCol = offsetToLineCol(textarea.value, newPos).col;
            handleMotion(newPos, false);
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'l' || key === 'ArrowRight') {
            const newPos = motionL(text, from, count);
            wantCol = offsetToLineCol(textarea.value, newPos).col;
            handleMotion(newPos, false);
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'j' || key === 'ArrowDown') {
            const col = wantCol ?? offsetToLineCol(text, from).col;
            const newPos = motionJ(text, from, count, col);
            wantCol = col;
            handleMotion(newPos, true);
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'k' || key === 'ArrowUp') {
            const col = wantCol ?? offsetToLineCol(text, from).col;
            const newPos = motionK(text, from, count, col);
            wantCol = col;
            handleMotion(newPos, true);
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'w') {
            handleMotion(motionW(text, from, count), false);
            wantCol = null;
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'W') {
            handleMotion(motionWW(text, from, count), false);
            wantCol = null;
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'b') {
            handleMotion(motionB(text, from, count), false);
            wantCol = null;
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'B') {
            handleMotion(motionBW(text, from, count), false);
            wantCol = null;
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'e') {
            // For operators, include the end char
            const raw = motionE(text, from, count);
            handleMotion(operator ? raw + 1 : raw, false);
            wantCol = null;
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'E') {
            const raw = motionEW(text, from, count);
            handleMotion(operator ? raw + 1 : raw, false);
            wantCol = null;
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === '0') {
            // countBuf is empty here (digit '0' only reaches here if countBuf was '')
            handleMotion(motionLineStart(text, from), false);
            wantCol = 0;
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === '^') {
            handleMotion(motionLineFirstNonBlank(text, from), false);
            wantCol = null;
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === '$' || key === 'End') {
            handleMotion(motionLineEnd(text, from), false);
            wantCol = Infinity;
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'g') {
            // 'gg' — first non-blank of file.  We detect by checking if operator or countBuf
            // encoded "g" was already buffered; simplest: just treat standalone g as go-to-top.
            handleMotion(motionFileStart(text), false);
            wantCol = null;
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === 'G') {
            const newPos = count === 1 && countBuf === ''
                ? motionFileEnd(text)
                : (() => {
                    const lines = getLines(text);
                    const targetLine = Math.min(count - 1, lines.length - 1);
                    const col = (lines[targetLine] || '').search(/\S/);
                    return lineColToOffset(text, targetLine, col >= 0 ? col : 0);
                })();
            handleMotion(newPos, false);
            wantCol = null;
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        if (key === '%') {
            handleMotion(motionMatchBracket(text, from), false);
            wantCol = null;
            event.preventDefault();
            event.stopPropagation();
            return;
        }

        // ── Undo (u) — delegate to browser/host (Ctrl+Z equivalent) ─────
        if (key === 'u') {
            document.execCommand('undo');
            setMode(MODE_NORMAL);
            event.preventDefault();
            event.stopPropagation();
            return;
        }

        // Unknown key — clear pending state to avoid stuck operators
        if (key.length === 1 || key === 'Backspace' || key === 'Delete') {
            operator = null;
            countBuf = '';
        }
    }

    // ── main keydown handler ──────────────────────────────────────────────────
    function onKeyDown(event) {
        // Let Ctrl/Meta shortcuts pass through in all modes (save, copy, etc.)
        if (event.ctrlKey || event.metaKey) {
            return;
        }

        if (mode === MODE_INSERT) {
            if (event.key === 'Escape') {
                setMode(MODE_NORMAL);
                // Clamp cursor to valid NORMAL position (must not sit on EOL).
                const v = textarea.value;
                const p = getCurrentPos(textarea);
                const { line, col } = offsetToLineCol(v, p);
                const lines = getLines(v);
                const clamped = lineColToOffset(v, line, clampCol(lines, line, col, true));
                setPos(textarea, clamped);
                event.preventDefault();
                event.stopPropagation();
            }
            return;
        }

        if (mode === MODE_REPLACE_ONCE) {
            if (event.key === 'Escape') {
                setMode(MODE_NORMAL);
                event.preventDefault();
                event.stopPropagation();
                return;
            }
            if (event.key.length === 1) {
                applyReplace(textarea, getCurrentPos(textarea), event.key);
                setMode(MODE_NORMAL);
                event.preventDefault();
                event.stopPropagation();
            }
            return;
        }

        if (mode === MODE_REPLACE) {
            if (event.key === 'Escape') {
                setMode(MODE_NORMAL);
                event.preventDefault();
                event.stopPropagation();
                return;
            }
            if (event.key === 'Backspace') {
                // Move cursor back one position (replace mode backspace)
                const p = getCurrentPos(textarea);
                if (p > 0) setPos(textarea, p - 1);
                event.preventDefault();
                event.stopPropagation();
                return;
            }
            if (event.key.length === 1) {
                const p = getCurrentPos(textarea);
                if (p < textarea.value.length && textarea.value[p] !== '\n') {
                    applyReplace(textarea, p, event.key);
                    setPos(textarea, p + 1);
                } else {
                    // At EOL: insert instead of replace
                    applyInsert(textarea, p, event.key);
                }
                event.preventDefault();
                event.stopPropagation();
            }
            return;
        }

        // MODE_NORMAL
        if (event.key === 'Escape') {
            // Already in normal; just clear pending state
            operator = null;
            countBuf = '';
            event.preventDefault();
            event.stopPropagation();
            return;
        }

        handleNormalKey(event);
    }

    // Reposition the badge whenever the cursor moves or the editor scrolls.
    function onScroll() { positionIndicator(indicator, textarea); }
    function onKeyUp()  { positionIndicator(indicator, textarea); }
    function onMouseUp(){ positionIndicator(indicator, textarea); }

    textarea.addEventListener('keydown', onKeyDown);
    textarea.addEventListener('keyup',   onKeyUp);
    textarea.addEventListener('mouseup', onMouseUp);
    textarea.addEventListener('scroll',  onScroll);

    return {
        detach() {
            textarea.removeEventListener('keydown', onKeyDown);
            textarea.removeEventListener('keyup',   onKeyUp);
            textarea.removeEventListener('mouseup', onMouseUp);
            textarea.removeEventListener('scroll',  onScroll);
            indicator.remove();
        },
        getMode() {
            return mode;
        },
    };
}
