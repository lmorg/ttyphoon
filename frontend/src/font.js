export function createFontController(offCtx) {
    let cellWidth = 10;
    let cellHeight = 20;
    let fontSize = 15;
    let fontFamily = '';
    let adjustCellWidth = 0;
    let adjustCellHeight = 0;
    let glyphOffsetY = 0;

    try {
        fontFamily = getComputedStyle(document.documentElement).getPropertyValue('--terminal-menu-font').trim()
            || getComputedStyle(document.documentElement).getPropertyValue('--font-family').trim()
            || getComputedStyle(document.body).fontFamily;
    } catch {
        fontFamily = '';
    }
    let glyphSizeCached = false;

    function applyConfiguredFontFromWindowStyle(windowStyle) {
        let fontChanged = false;
        const parsed = parseInt(windowStyle?.fontSize, 10);
        if (!Number.isNaN(parsed) && parsed > 0 && parsed !== fontSize) {
            fontSize = parsed;
            fontChanged = true;
        }

        if (windowStyle?.fontFamily && windowStyle.fontFamily !== fontFamily) {
            fontFamily = windowStyle.fontFamily;
            fontChanged = true;
        }

        const nextAdjustWidth = Number.isFinite(windowStyle?.adjustCellWidth) ? windowStyle.adjustCellWidth : 0;
        if (nextAdjustWidth !== adjustCellWidth) {
            adjustCellWidth = nextAdjustWidth;
            fontChanged = true;
        }

        const nextAdjustHeight = Number.isFinite(windowStyle?.adjustCellHeight) ? windowStyle.adjustCellHeight : 0;
        if (nextAdjustHeight !== adjustCellHeight) {
            adjustCellHeight = nextAdjustHeight;
            fontChanged = true;
        }

        if (fontChanged) {
            glyphSizeCached = false;
        }

        if (offCtx) {
            offCtx.font = `${fontSize}px ${fontFamily}`;
        }

        return fontChanged;
    }

    function configureFontMetricsFallback(windowStyle) {
        if (!offCtx) {
            return;
        }

        applyConfiguredFontFromWindowStyle(windowStyle);

        offCtx.font = `${fontSize}px ${fontFamily}`;
        const metrics = offCtx.measureText('M');

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

        cellWidth = Math.max(1, measuredWidth + adjustCellWidth);
        cellHeight = Math.max(1, measuredHeight + adjustCellHeight);
        glyphOffsetY = Math.ceil((cellHeight - measuredHeight) / 2);
    }

    async function loadGlyphSizeFromGo(windowStyle) {
        if (glyphSizeCached) {
            return;
        }

        // Measure immediately so getCellSize() never returns the hardcoded
        // defaults while the custom font is still loading asynchronously.
        configureFontMetricsFallback(windowStyle);

        // Resolve the configured family through CSS so a bundled @font-face is
        // preferred over a same-named font installed on the host.
        try {
            await document.fonts.load(`${fontSize}px ${fontFamily}`, 'M');
        } catch {
            // Non-fatal: canvas will use the configured fallback family.
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

    function getGlyphOffset() {
        return { x: 0, y: glyphOffsetY };
    }

    return {
        applyConfiguredFontFromWindowStyle,
        loadGlyphSizeFromGo,
        applyCellStyle,
        getCellSize,
        getGlyphOffset,
    };
}
