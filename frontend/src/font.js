export function createFontController(offCtx) {
    let cellWidth = 10;
    let cellHeight = 20;
    let fontSize = 15;
    let fontFamily = '';

    try {
        fontFamily = getComputedStyle(document.documentElement).getPropertyValue('--terminal-menu-font').trim()
            || getComputedStyle(document.documentElement).getPropertyValue('--font-family').trim()
            || getComputedStyle(document.body).fontFamily;
    } catch {
        fontFamily = '';
    }
    let glyphSizeCached = false;

    function applyConfiguredFontFromWindowStyle(windowStyle) {
        const parsed = parseInt(windowStyle?.fontSize, 10);
        if (!Number.isNaN(parsed) && parsed > 0) {
            fontSize = parsed;
        }

        if (windowStyle?.fontFamily) {
            fontFamily = windowStyle.fontFamily;
        }

        if (offCtx) {
            offCtx.font = `${fontSize}px ${fontFamily}`;
        }
    }

    function configureFontMetricsFallback(windowStyle) {
        if (!offCtx) {
            return;
        }

        applyConfiguredFontFromWindowStyle(windowStyle);

        offCtx.font = `${fontSize}px ${fontFamily}`;
        const metrics = offCtx.measureText('M');
        const adjustWidth = Number.isFinite(windowStyle?.adjustCellWidth) ? windowStyle.adjustCellWidth : 0;
        const adjustHeight = Number.isFinite(windowStyle?.adjustCellHeight) ? windowStyle.adjustCellHeight : 0;

        const measuredWidth = Math.ceil(metrics.width || fontSize * 0.6);

        // Use emHeightAscent + emHeightDescent (the em-square) for cell height.
        // fontBoundingBoxAscent/Descent are font-level maximums that span every
        // glyph in the font (e.g. tall accented capitals) and can be 2× the
        // configured fontSize for code fonts like Fira Code, causing "double
        // height" rows. The em-square equals approximately fontSize regardless
        // of which glyphs the font contains and matches what FreeType/SDL uses.
        const emAscent = Number.isFinite(metrics.emHeightAscent) && metrics.emHeightAscent > 0
            ? metrics.emHeightAscent : fontSize * 0.8;
        const emDescent = Number.isFinite(metrics.emHeightDescent) && metrics.emHeightDescent > 0
            ? metrics.emHeightDescent : fontSize * 0.2;
        const measuredHeight = Math.ceil(emAscent + emDescent);

        cellWidth = Math.max(1, measuredWidth + adjustWidth);
        cellHeight = Math.max(1, measuredHeight + adjustHeight);
    }

    async function loadGlyphSizeFromGo(windowStyle) {
        if (glyphSizeCached) {
            return;
        }

        // Measure immediately so getCellSize() never returns the hardcoded
        // defaults while the custom font is still loading asynchronously.
        configureFontMetricsFallback(windowStyle);

        // Load the configured font with OpenType ligature features enabled
        // (liga = standard ligatures, calt = contextual alternates).
        // Using the FontFace API lets us attach feature settings that canvas
        // will honour when calling fillText — unlike CSS font-feature-settings
        // which canvas does not observe.  If the font is already loaded by a
        // prior @font-face rule the browser deduplicates and this is a no-op.
        // FontFace() requires a bare family name, not a CSS font-family list.
        // Strip surrounding quotes and any fallback families so that e.g.
        // '"Fira Code", monospace' becomes 'Fira Code'.
        const primaryFamily = fontFamily.split(',')[0].trim().replace(/^["']|["']$/g, '');

        try {
            const face = new FontFace(primaryFamily, `local("${primaryFamily}")`, {
                featureSettings: '"liga" 1, "calt" 1',
            });
            await face.load();
            document.fonts.add(face);
        } catch {
            // If the FontFace API fails (e.g. font not installed locally),
            // fall back to waiting for the CSS-declared face to be ready.
            try {
                await document.fonts.load(`${fontSize}px ${fontFamily}`);
            } catch {
                // non-fatal — proceed with whatever font is available
            }
        }

        // Re-measure now that the actual font is loaded for accurate metrics.
        configureFontMetricsFallback(windowStyle);
        glyphSizeCached = true;
    }

    function applyCellStyle(cmd) {
        const fontParts = [];
        if (cmd.italic) {
            fontParts.push('italic');
        }
        if (cmd.bold) {
            fontParts.push('bold');
        }
        fontParts.push(`${fontSize}px`);
        fontParts.push(fontFamily);
        offCtx.font = fontParts.join(' ');
        offCtx.textBaseline = 'top';
    }

    function getCellSize() {
        return { cellWidth, cellHeight };
    }

    return {
        applyConfiguredFontFromWindowStyle,
        loadGlyphSizeFromGo,
        applyCellStyle,
        getCellSize,
    };
}
