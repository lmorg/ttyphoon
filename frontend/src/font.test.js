import { describe, expect, it } from 'vitest';

import { createFontController } from './font.js';

function createMockContext() {
    return {
        font: '',
        textBaseline: '',
        measureText: () => ({
            width: 9,
            emHeightAscent: 12,
            emHeightDescent: 3,
        }),
    };
}

describe('createFontController', () => {
    it.each([
        [0, 0],
        [3, 2],
        [4, 2],
        [-3, -1],
    ])('centres glyphs for a cell-height adjustment of %i', async (adjustCellHeight, expectedOffsetY) => {
        const ctx = createMockContext();
        const controller = createFontController(ctx);
        const windowStyle = {
            fontFamily: 'monospace',
            fontSize: 15,
            adjustCellWidth: 0,
            adjustCellHeight,
        };

        await controller.loadGlyphSizeFromGo(windowStyle);

        expect(controller.getGlyphOffset()).toEqual({ x: 0, y: expectedOffsetY });
    });

    it('invalidates cached glyph metrics when the configured font changes', async () => {
        const ctx = createMockContext();
        const controller = createFontController(ctx);

        controller.applyConfiguredFontFromWindowStyle({
            fontFamily: '"Fira Code", monospace',
            fontSize: 15,
            adjustCellWidth: 0,
            adjustCellHeight: 0,
        });

        await controller.loadGlyphSizeFromGo({
            fontFamily: '"Fira Code", monospace',
            fontSize: 15,
            adjustCellWidth: 0,
            adjustCellHeight: 0,
        });

        ctx.measureText = () => ({
            width: 11,
            emHeightAscent: 13,
            emHeightDescent: 4,
        });

        controller.applyConfiguredFontFromWindowStyle({
            fontFamily: '"Hack", monospace',
            fontSize: 16,
            adjustCellWidth: 0,
            adjustCellHeight: 0,
        });

        await controller.loadGlyphSizeFromGo({
            fontFamily: '"Hack", monospace',
            fontSize: 16,
            adjustCellWidth: 0,
            adjustCellHeight: 0,
        });

        expect(controller.getCellSize()).toEqual({
            cellWidth: 11,
            cellHeight: 17,
        });
        expect(ctx.font).toBe('16px "Hack", monospace');
    });
});