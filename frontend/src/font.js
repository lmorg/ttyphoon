export function createFontController(offCtx) {
    let cellWidth = 10;
    let cellHeight = 20;
    let fontSize = 18;
    let fontFamily = 'monospace';
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
        cellWidth = Math.ceil(metrics.width || fontSize * 0.6);
        cellHeight = Math.ceil((metrics.fontBoundingBoxAscent || fontSize) + (metrics.fontBoundingBoxDescent || fontSize * 0.2));
    }

    async function loadGlyphSizeFromGo(windowStyle) {
        if (glyphSizeCached) {
            return;
        }

        try {
            const glyph = await window['go']['main']['WApp']['GetTerminalGlyphSize']();
            if (glyph && glyph.X > 0 && glyph.Y > 0) {
                cellWidth = glyph.X;
                cellHeight = glyph.Y;
                glyphSizeCached = true;
                return;
            }
        } catch {
            // fallback below
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
