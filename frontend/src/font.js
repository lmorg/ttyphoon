import { GetTerminalGlyphSize } from '../wailsjs/go/main/WApp';

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

        // Wait for the configured web font to be ready before measuring or
        // rendering.  Without this, canvas silently falls back to monospace
        // because @font-face fonts load asynchronously and canvas does not
        // participate in the CSS font loading lifecycle.
        try {
            await document.fonts.load(`${fontSize}px ${fontFamily}`);
        } catch {
            // non-fatal — proceed with whatever font is available
        }

        try {
            const glyph = await GetTerminalGlyphSize();
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
