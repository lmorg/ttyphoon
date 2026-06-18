/**
 * spellcheck.js — textarea spell-check overlay using aspell via the Go backend.
 *
 * Renders terminal-style red wavy underlines on a <canvas> positioned over the
 * textarea. Word positions are measured with the mirror-div technique (the same
 * approach used by vim-mode for caret tracking) so alignment is accurate for
 * both word-wrapped and non-word-wrapped editors.
 *
 * Right-click context menus are handled entirely by the existing editor handlers.
 *
 * Usage:
 *   import { attachSpellCheck } from './spellcheck';
 *   const sc = attachSpellCheck(textarea);
 *   sc.setExclusions(['async', 'typeof']); // e.g. LSP keywords
 *   sc.detach();
 */

import { NotesSpellCheck } from '../wailsjs/go/main/WApp';

const DEBOUNCE_MS = 800;

// Wavy underline parameters — matched to the terminal renderer style.
const WAVE_AMPLITUDE = 2.5;
const WAVE_FREQUENCY = 0.35;

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
    /** Raw results from the backend, unfiltered. */
    let rawMisspellings = [];
    /** Filtered misspellings after exclusions applied. */
    let currentMisspellings = [];
    /**
     * Cached word position data from the last measureWordRects call.
     * Contains { wordStart, wordLength, misspeltWord, startRect, endRect, mirrorRect }.
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
    // Ensure the parent forms a positioning context for position:absolute.
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
        const base = [
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
            base.push(`width:${textarea.clientWidth}px`);
        }
        return base.join(';');
    }

    /**
     * Measure the bounding rect of each misspelled word using the Range API on
     * a clean text node inside a mirror div. Returns an array of objects each
     * containing: { wordStart, wordLength, misspeltWord, suggestions, wordRect, mirrorRect }
     * where the rects are in *viewport* coordinates.
     *
     * Using Range.getBoundingClientRect() on a plain text node (rather than
     * inserting <span> elements between text nodes) avoids any inline-element
     * side-effects on kerning, ligatures, and line-break decisions.
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
        // space as the canvas — delta (wordRect - mirrorRect) is then purely
        // layout-relative and unaffected by any WebView viewport quirks.
        canvasParent.insertBefore(mirror, canvas);

        const mirrorRect = mirror.getBoundingClientRect();

        const result = misspellings.map(m => {
            const range = document.createRange();
            range.setStart(textNode, m.wordStart);
            range.setEnd(textNode, m.wordStart + m.wordLength);
            return {
                wordStart:    m.wordStart,
                wordLength:   m.wordLength,
                misspeltWord: m.misspeltWord,
                suggestions:  m.suggestions,
                wordRect:     range.getBoundingClientRect(),
                mirrorRect,
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
            else ctx.lineTo(x, y);
        }
        ctx.stroke();
    }

    /**
     * Position the canvas and paint wavy underlines for all cached word positions.
     * Uses textarea.scrollLeft/scrollTop at call time so scroll is always current.
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
        // Resizing the canvas clears it, so only resize when dimensions change.
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
        const redColour = docStyle.getPropertyValue('--terminal-red').trim() || '#e05555';

        ctx.save();
        // Clip to textarea viewport bounds.
        ctx.beginPath();
        ctx.rect(0, 0, w, h);
        ctx.clip();

        ctx.strokeStyle = redColour;
        ctx.lineWidth   = 1.5;
        ctx.lineJoin    = 'round';
        ctx.lineCap     = 'round';

        const sl = textarea.scrollLeft;
        const st = textarea.scrollTop;

        for (const wd of cachedWordData) {
            const { wordRect, mirrorRect } = wd;

            // Mirror-relative positions: (rect.left - mirror.left) gives the
            // horizontal offset within the mirror's text layout. Subtracting
            // scrollLeft/scrollTop converts to canvas coords (canvas origin =
            // textarea top-left in the viewport).
            const wordCanvasX = Math.round((wordRect.left  - mirrorRect.left) - sl);
            const wordCanvasY = Math.round((wordRect.top   - mirrorRect.top)  - st);
            const wordWidth   = Math.round(wordRect.right  - wordRect.left);

            // Use the range's measured height as line height.
            const lineH = wordRect.height || 16;
            // Place underline 2 px above the bottom of the line box.
            const baseY = wordCanvasY + lineH - 2;

            drawWavy(ctx, wordCanvasX, wordCanvasX + wordWidth, baseY);
        }

        ctx.restore();
    }

    // ── Spell-check runner ────────────────────────────────────────────────

    async function runCheck() {
        if (destroyed) return;
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

        rawMisspellings     = raw;
        currentMisspellings = raw.filter(m => !exclusionSet.has(m.misspeltWord.toLowerCase()));
        cachedWordData      = measureWordRects(currentMisspellings, text);
        drawCanvas();
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
    };
}

