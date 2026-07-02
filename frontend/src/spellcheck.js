/**
 * spellcheck.js — textarea spell-check overlay using aspell via the Go backend.
 *
 * Renders terminal-style red wavy underlines on a <canvas> positioned over the
 * textarea. Word positions are measured with the mirror-div + Range API technique
 * so alignment is accurate for both word-wrapped and non-word-wrapped editors.
 *
 * Right-click context menus are handled entirely by the existing editor handlers.
 *
 * Usage:
 *   import { attachSpellCheck } from './spellcheck';
 *   const sc = attachSpellCheck(textarea);
 *   sc.setExclusions(['async', 'typeof']); // e.g. LSP keywords
 *   sc.detach();
 *
 * Two data sources are supported:
 *   - 'aspell'   (default): the overlay calls the aspell backend itself.
 *   - 'external'          : the overlay renders misspellings pushed in via
 *                           setMisspellings() — used for typos-lsp diagnostics,
 *                           which reuse this same red wavy-underline chrome.
 */

import { NotesSpellCheck } from '../wailsjs/go/main/WApp';
import { showLocalMenu } from './popup_menu';

const DEBOUNCE_MS = 800;

// Wavy underline parameters.
const WAVE_AMPLITUDE = 1.5;
const WAVE_FREQUENCY = 0.4;

// ─── Public API ──────────────────────────────────────────────────────────────

/**
 * Attach spell-check to a textarea element.
 *
 * @param {HTMLTextAreaElement} textarea
 * @param {{ exclusions?: string[] }} [options]
 * @returns {{ detach(): void, setExclusions(words: string[]): void, check(): void }}
 */
export function attachSpellCheck(textarea, options = {}) {
    let exclusionSet = new Set((options.exclusions || []).map(w => String(w).toLowerCase()));
    // Data source: 'aspell' fetches from the backend; 'external' renders
    // misspellings pushed in via setMisspellings() (e.g. typos-lsp diagnostics).
    let mode = options.mode === 'external' ? 'external' : 'aspell';
    /** Raw results from the backend, unfiltered. */
    let rawMisspellings = [];
    /** Filtered misspellings after exclusions applied. */
    let currentMisspellings = [];
    /**
     * Cached word position data from the last measureWordRects call.
     * Each entry: { wordRect, mirrorRect }.
     */
    let cachedWordData = [];
    let lastCheckedText = null;
    let checkTimer = null;
    let destroyed = false;

    // ── Canvas overlay ────────────────────────────────────────────────────

    const canvas = document.createElement('canvas');
    canvas.style.cssText = 'position:absolute;top:0;left:0;pointer-events:none;z-index:2000';
    canvas.setAttribute('aria-hidden', 'true');

    // Insert the canvas as a sibling of the textarea inside its own parent so
    // it inherits visibility automatically — when any ancestor gets display:none
    // (e.g. a tab panel becomes inactive) the canvas disappears with it, with
    // no JS bookkeeping required.
    const canvasParent = textarea.parentNode;
    let savedParentPosition = null;
    if (getComputedStyle(canvasParent).position === 'static') {
        savedParentPosition = canvasParent.style.position;
        canvasParent.style.position = 'relative';
    }
    canvasParent.insertBefore(canvas, textarea.nextSibling);

    // ── Mirror div helpers ────────────────────────────────────────────────

    /**
     * Build the cssText for a mirror div that matches the textarea layout.
     * When isWrapping is false (white-space:pre / nowrap), no width constraint
     * is applied so long lines don't wrap, mirroring the textarea exactly.
     */
    function buildMirrorCss(cs, isWrapping) {
        const parts = [
            // position:absolute inside canvasParent keeps the mirror in the
            // same coordinate space as the canvas — avoids any position:fixed
            // viewport quirks in Chromium-based WebViews (e.g. Wails).
            'position:absolute',
            'top:0',
            'left:0',
            'visibility:hidden',
            'pointer-events:none',
            'z-index:-1',
            isWrapping ? 'white-space:pre-wrap' : 'white-space:pre',
            // Use individual properties rather than shorthands — computed-style
            // shorthands like `padding` and `border` may return "" on Chromium
            // when sides differ, causing the mirror to have wrong dimensions.
            `font-size:${cs.fontSize}`,
            `font-family:${cs.fontFamily}`,
            `font-weight:${cs.fontWeight}`,
            `font-style:${cs.fontStyle}`,
            `font-variant:${cs.fontVariant}`,
            `line-height:${cs.lineHeight}`,
            `letter-spacing:${cs.letterSpacing}`,
            `word-spacing:${cs.wordSpacing}`,
            `padding-top:${cs.paddingTop}`,
            `padding-right:${cs.paddingRight}`,
            `padding-bottom:${cs.paddingBottom}`,
            `padding-left:${cs.paddingLeft}`,
            `border-top-width:${cs.borderTopWidth}`,
            `border-right-width:${cs.borderRightWidth}`,
            `border-bottom-width:${cs.borderBottomWidth}`,
            `border-left-width:${cs.borderLeftWidth}`,
            'border-style:solid',
            `box-sizing:${cs.boxSizing}`,
            `tab-size:${cs.tabSize}`,
            // word-break and overflow-wrap control where lines break; if the
            // mirror breaks at different columns than the textarea all Y
            // positions on subsequent lines will be wrong.
            `word-break:${cs.wordBreak}`,
            `overflow-wrap:${cs.overflowWrap}`,
            // Ligature and font-feature settings affect per-character widths.
            // Copy them explicitly so the mirror renders identically.
            `font-feature-settings:${cs.fontFeatureSettings}`,
            `font-variant-ligatures:${cs.fontVariantLigatures}`,
        ];
        if (isWrapping) {
            parts.push(`width:${textarea.clientWidth}px`);
        }
        return parts.join(';');
    }

    /**
     * Measure the bounding rect of each misspelled word using the Range API on
     * a clean text node inside a mirror div. Returns an array of objects each
     * containing { wordRect, mirrorRect } in viewport coordinates.
     *
     * Using Range.getBoundingClientRect() on a plain text node (rather than
     * inserting <span> elements) avoids side-effects on kerning, ligatures,
     * and line-break decisions.
     */
    function measureWordRects(misspellings, text) {
        if (!misspellings.length) return [];

        const cs = getComputedStyle(textarea);
        const isWrapping = !['nowrap', 'pre'].includes(cs.whiteSpace);

        const mirror = document.createElement('div');
        mirror.style.cssText = buildMirrorCss(cs, isWrapping);

        const textNode = document.createTextNode(text);
        mirror.appendChild(textNode);
        // Insert into canvasParent so the mirror shares the same coordinate
        // space as the canvas.
        canvasParent.insertBefore(mirror, canvas);

        const mirrorRect = mirror.getBoundingClientRect();

        const result = misspellings
            .filter(m => m.wordStart >= 0 && m.wordStart + m.wordLength <= text.length)
            .map(m => {
            const range = document.createRange();
            range.setStart(textNode, m.wordStart);
            range.setEnd(textNode, m.wordStart + m.wordLength);
            return {
                wordRect:    range.getBoundingClientRect(),
                mirrorRect,
                misspelling: m,
            };
        });

        canvasParent.removeChild(mirror);
        return result;
    }

    // ── Drawing ───────────────────────────────────────────────────────────

    /** Draw a wavy underline from canvas x1 to x2 at baseline y. */
    function drawWavy(ctx, x1, x2, baseY) {
        if (x2 < x1) return;
        ctx.beginPath();
        for (let x = x1; x <= x2; x++) {
            const y = baseY + Math.sin((x - x1) * WAVE_FREQUENCY) * WAVE_AMPLITUDE;
            if (x === x1) ctx.moveTo(x, y);
            else          ctx.lineTo(x, y);
        }
        ctx.stroke();
    }

    /**
     * Position the canvas and paint wavy underlines for all cached word
     * positions. Uses textarea.scrollLeft/scrollTop at call time so scroll
     * is always current.
     */
    function drawCanvas() {
        if (destroyed) return;

        // Track the textarea's position within the parent. Using offsetTop/Left
        // rather than getBoundingClientRect keeps coordinates in the parent's
        // layout space, which is what position:absolute needs.
        canvas.style.top  = `${textarea.offsetTop}px`;
        canvas.style.left = `${textarea.offsetLeft}px`;

        const w = Math.max(1, Math.round(textarea.offsetWidth));
        const h = Math.max(1, Math.round(textarea.offsetHeight));
        // Resizing the canvas clears it automatically, so only resize when
        // dimensions actually change.
        if (canvas.width !== w || canvas.height !== h) {
            canvas.width  = w;
            canvas.height = h;
        }

        const ctx = canvas.getContext('2d');
        if (!ctx) return;
        ctx.clearRect(0, 0, w, h);

        if (!cachedWordData.length) return;

        // Resolve red colour from the terminal theme CSS variable.
        const docStyle  = getComputedStyle(document.documentElement);
        const redColour = docStyle.getPropertyValue('--red').trim() || '#e05555';

        ctx.save();
        // Clip to textarea viewport bounds.
        ctx.beginPath();
        ctx.rect(0, 0, w, h);
        ctx.clip();

        ctx.strokeStyle = redColour;
        ctx.globalAlpha = 1;
        ctx.lineWidth   = 1.5;
        ctx.lineJoin    = 'round';
        ctx.lineCap     = 'round';

        const sl = textarea.scrollLeft;
        const st = textarea.scrollTop;

        for (const { wordRect, mirrorRect } of cachedWordData) {
            // Mirror-relative positions: (rect.left - mirror.left) gives the
            // horizontal offset within the mirror's text layout. Subtracting
            // scrollLeft/scrollTop converts to canvas coords (canvas origin =
            // textarea top-left in the viewport).
            const x1    = Math.round((wordRect.left  - mirrorRect.left) - sl);
            const x2    = Math.round((wordRect.right - mirrorRect.left) - sl);
            const top   = Math.round((wordRect.top   - mirrorRect.top)  - st);
            const lineH = wordRect.height || 16;
            // Place underline just above the bottom of the line box.
            const baseY = top + lineH - 1;

            drawWavy(ctx, x1, x2, baseY);
        }

        ctx.restore();
    }

    // ── Suggestions popup ─────────────────────────────────────────────────

    /**
     * Apply a spelling suggestion: replace the misspelled word in the textarea,
     * restore cursor position to just after the replacement, and re-trigger the
     * input event so the spell-checker schedules a fresh check.
     */
    function applySuggestion(misspelling, suggestion) {
        const text = textarea.value;
        const { wordStart, wordLength } = misspelling;
        textarea.value =
            text.slice(0, wordStart) + suggestion + text.slice(wordStart + wordLength);
        textarea.selectionStart = textarea.selectionEnd = wordStart + suggestion.length;
        textarea.dispatchEvent(new Event('input', { bubbles: true }));
    }

    /**
     * Show the shared popup menu near the clicked word.
     * Uses showLocalMenu so the suggestions list matches the rest of the UI.
     */
    function showSuggestionsPopup(anchorX, anchorY, misspelling) {
        if (!misspelling.suggestions || !misspelling.suggestions.length) {
            showLocalMenu({
                title: misspelling.misspeltWord,
                options: ['No spelling suggestions'],
                icons: [],
                x: anchorX,
                y: anchorY,
                showNextToMouseCursor: true,
            });
            return;
        }

        showLocalMenu({
            title: misspelling.misspeltWord,
            options: misspelling.suggestions,
            icons:   misspelling.suggestions.map(() => 0xf040),
            x: anchorX,
            y: anchorY,
            showNextToMouseCursor: true,
            onSelect: (index) => applySuggestion(misspelling, misspelling.suggestions[index]),
        });
    }

    // ── Spell-check runner ────────────────────────────────────────────────

    /**
     * Apply a raw misspelling list (from aspell or pushed externally): filter by
     * exclusions, measure word rects, and repaint the overlay.
     */
    function applyMisspellings(raw, text) {
        rawMisspellings     = Array.isArray(raw) ? raw : [];
        currentMisspellings = rawMisspellings.filter(
            m => m && m.misspeltWord && !exclusionSet.has(String(m.misspeltWord).toLowerCase()),
        );
        cachedWordData      = measureWordRects(currentMisspellings, text);
        drawCanvas();
    }

    async function runCheck() {
        if (destroyed) return;
        // In external mode the overlay is fed by setMisspellings(); never call aspell.
        if (mode === 'external') return;
        const text = textarea.value;
        if (text === lastCheckedText) return;
        lastCheckedText = text;

        let raw = [];
        try {
            raw = (await NotesSpellCheck(text)) || [];
        } catch {
            // aspell not available or backend error — degrade silently.
        }

        if (destroyed) return;

        applyMisspellings(raw, text);
    }

    function scheduleCheck() {
        clearTimeout(checkTimer);
        checkTimer = setTimeout(() => runCheck(), DEBOUNCE_MS);
    }

    // ── Event listeners ───────────────────────────────────────────────────

    function onInput() {
        scheduleCheck();
    }

    /** On textarea scroll: redraw with cached positions — no re-measurement needed. */
    function onScroll() {
        drawCanvas();
    }

    /**
     * On click: if the click lands inside a wavy-underlined word, show the
     * suggestions popup. Uses the same coordinate transform as drawCanvas so
     * hit-testing is always consistent with what is rendered.
     */
    function onTextareaClick(event) {
        if (!cachedWordData.length) return;

        const canvasRect = canvas.getBoundingClientRect();
        const canvasX = event.clientX - canvasRect.left;
        const canvasY = event.clientY - canvasRect.top;

        const sl = textarea.scrollLeft;
        const st = textarea.scrollTop;

        for (const { wordRect, mirrorRect, misspelling } of cachedWordData) {
            const x1    = Math.round((wordRect.left  - mirrorRect.left) - sl);
            const x2    = Math.round((wordRect.right - mirrorRect.left) - sl);
            const top   = Math.round((wordRect.top   - mirrorRect.top)  - st);
            const lineH = wordRect.height || 16;

            if (canvasX >= x1 && canvasX <= x2 && canvasY >= top && canvasY <= top + lineH) {
                // Position popup just below the word in viewport coords.
                const popupX = canvasRect.left + x1;
                const popupY = canvasRect.top  + top + lineH + 2;
                showSuggestionsPopup(popupX, popupY, misspelling);
                return;
            }
        }
    }

    let rafId = null;
    /**
     * On geometry changes (window resize / ancestor scroll) re-measure word
     * positions because textarea width (and thus line-wrapping) may have changed.
     */
    function onGeometryChange() {
        if (rafId !== null) return;
        rafId = requestAnimationFrame(() => {
            rafId = null;
            if (!destroyed) {
                cachedWordData = measureWordRects(currentMisspellings, textarea.value);
                drawCanvas();
            }
        });
    }

    textarea.addEventListener('input',  onInput);
    textarea.addEventListener('scroll', onScroll);
    textarea.addEventListener('click',  onTextareaClick);
    window.addEventListener('resize',   onGeometryChange);
    window.addEventListener('scroll',   onGeometryChange, true);

    // Initial position + first check.
    drawCanvas();
    scheduleCheck();

    // ── Public handle ─────────────────────────────────────────────────────

    return {
        detach() {
            if (destroyed) return;
            destroyed = true;
            clearTimeout(checkTimer);
            if (rafId !== null) cancelAnimationFrame(rafId);
            textarea.removeEventListener('input',  onInput);
            textarea.removeEventListener('scroll', onScroll);
            textarea.removeEventListener('click',  onTextareaClick);
            window.removeEventListener('resize',   onGeometryChange);
            window.removeEventListener('scroll',   onGeometryChange, true);
            if (canvas.parentNode) canvas.parentNode.removeChild(canvas);
            if (savedParentPosition !== null) {
                canvasParent.style.position = savedParentPosition;
            }
        },

        /**
         * Update the exclusion list. Re-filters the last backend result without
         * calling aspell again, then redraws.
         * @param {string[]} words
         */
        setExclusions(words) {
            exclusionSet        = new Set((words || []).map(w => String(w).toLowerCase()));
            currentMisspellings = rawMisspellings.filter(
                m => !exclusionSet.has(m.misspeltWord.toLowerCase()),
            );
            cachedWordData = measureWordRects(currentMisspellings, textarea.value);
            drawCanvas();
        },

        /** Trigger an immediate spell-check, bypassing the debounce. */
        check() {
            lastCheckedText = null;
            runCheck();
        },

        /**
         * Switch the data source. 'external' suppresses aspell so the overlay is
         * driven solely by setMisspellings() (typos-lsp); 'aspell' restores the
         * built-in aspell checker. Switching always clears the current underlines.
         * @param {'aspell'|'external'} nextMode
         */
        setMode(nextMode) {
            const next = nextMode === 'external' ? 'external' : 'aspell';
            if (next === mode) return;
            mode = next;
            applyMisspellings([], textarea.value);
            if (mode === 'external') {
                clearTimeout(checkTimer);
            } else {
                lastCheckedText = null;
                scheduleCheck();
            }
        },

        /** Current data source mode. */
        getMode() {
            return mode;
        },

        /**
         * Replace the rendered misspellings (external mode). Each item must be
         * { misspeltWord, wordStart, wordLength, suggestions? } with absolute
         * character offsets into the textarea value.
         * @param {Array<{misspeltWord:string,wordStart:number,wordLength:number,suggestions?:string[]}>} list
         */
        setMisspellings(list) {
            if (mode !== 'external') return;
            applyMisspellings(list, textarea.value);
        },
    };
}

