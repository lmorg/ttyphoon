export function createFontController(offCtx) {
    let cellWidth = 10;
    let cellHeight = 20;
    let fontSize = 15;
    let fontFamily = '"Fira Code", monospace';
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
        const measuredHeight = Math.ceil((metrics.fontBoundingBoxAscent || fontSize) + (metrics.fontBoundingBoxDescent || fontSize * 0.2));

        cellWidth = Math.max(1, measuredWidth + adjustWidth);
        cellHeight = Math.max(1, measuredHeight + adjustHeight);
    }

    async function loadGlyphSizeFromGo(windowStyle) {
        if (glyphSizeCached) {
            return;
        }

        // Load the configured font with OpenType ligature features enabled
        // (liga = standard ligatures, calt = contextual alternates).
        // Using the FontFace API lets us attach feature settings that canvas
        // will honour when calling fillText — unlike CSS font-feature-settings
        // which canvas does not observe.  If the font is already loaded by a
        // prior @font-face rule the browser deduplicates and this is a no-op.
        try {
            const face = new FontFace(fontFamily, `local("${fontFamily}")`, {
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
