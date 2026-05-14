import { beforeEach, describe, expect, it, vi } from 'vitest';

const eventHandlers = new Map();

const eventsOnMock = vi.fn((eventName, handler) => {
    eventHandlers.set(eventName, handler);
});

const getWindowStyleMock = vi.fn(async () => ({
    statusBar: true,
    fontSize: 14,
    fontFamily: 'monospace',
    colors: {
        bg: { Red: 10, Green: 20, Blue: 30 },
        fg: { Red: 220, Green: 220, Blue: 220 },
        yellow: { Red: 220, Green: 180, Blue: 40 },
        selection: { Red: 80, Green: 120, Blue: 200 },
        green: { Red: 70, Green: 180, Blue: 110 },
        searchResult: { Red: 64, Green: 64, Blue: 255 },
        whiteBright: { Red: 255, Green: 255, Blue: 255 },
    },
}));
const terminalGetTabsMock = vi.fn(async () => []);
const terminalRequestRedrawMock = vi.fn(async () => {});
const terminalSetGlyphSizeMock = vi.fn(async () => {});
const terminalResizeMock = vi.fn(async () => {});
const sendIpcMock = vi.fn(async () => {});
const terminalCopyImageDataURLMock = vi.fn(async () => {});
const terminalSelectWindowMock = vi.fn(async () => {});

const wireKeyboardEventsMock = vi.fn();
const wireMouseEventsMock = vi.fn();
const drawGaugeMock = vi.fn();
const drawBlockChromeMock = vi.fn();
const initTerminalPopupMenuMock = vi.fn();
const initInputBoxMock = vi.fn();
const showFullscreenImageOverlayMock = vi.fn();
let latestTerminalCtx = null;
let latestOffscreenCtx = null;

const fontApplyCellStyleMock = vi.fn((cmd) => {
    // Keep this tiny and deterministic for assertions.
    fontApplyCellStyleMock.lastCmd = cmd;
});

vi.mock('../wailsjs/runtime/runtime', () => ({
    EventsOn: eventsOnMock,
}));

vi.mock('../wailsjs/go/main/WApp', () => ({
    GetWindowStyle: getWindowStyleMock,
    SendIpc: sendIpcMock,
    TerminalCopyImageDataURL: terminalCopyImageDataURLMock,
    TerminalGetTabs: terminalGetTabsMock,
    TerminalRequestRedraw: terminalRequestRedrawMock,
    TerminalResize: terminalResizeMock,
    TerminalSelectWindow: terminalSelectWindowMock,
    TerminalSetGlyphSize: terminalSetGlyphSizeMock,
}));

vi.mock('./events', () => ({
    wireKeyboardEvents: wireKeyboardEventsMock,
    wireMouseEvents: wireMouseEventsMock,
}));

vi.mock('./font', () => ({
    createFontController: () => ({
        applyConfiguredFontFromWindowStyle: vi.fn(),
        loadGlyphSizeFromGo: vi.fn(async () => {}),
        applyCellStyle: fontApplyCellStyleMock,
        getCellSize: () => ({ cellWidth: 10, cellHeight: 20 }),
    }),
}));

vi.mock('./gauge', () => ({
    drawGauge: drawGaugeMock,
}));

vi.mock('./block_chrome', () => ({
    drawBlockChrome: drawBlockChromeMock,
}));

vi.mock('./popup_menu', () => ({
    initTerminalPopupMenu: initTerminalPopupMenuMock,
}));

vi.mock('./inputbox', () => ({
    initInputBox: initInputBoxMock,
}));

vi.mock('./fullscreen-image-overlay', () => ({
    showFullscreenImageOverlay: showFullscreenImageOverlayMock,
}));

vi.mock('./style-utils', () => ({
    DARKEN_BACKGROUND_OVERLAY: 'rgba(0,0,0,0.2)',
}));

vi.mock('./assets/sound/ding.mp3', () => ({
    default: 'ding.mp3',
}));

function makeContext2d() {
    return {
        canvas: { width: 800, height: 600 },
        font: '',
        textBaseline: 'top',
        fillStyle: '',
        strokeStyle: '',
        lineWidth: 0,
        globalAlpha: 1,
        shadowColor: 'transparent',
        shadowBlur: 0,
        __strokeStyles: [],
        save: vi.fn(),
        restore: vi.fn(),
        beginPath: vi.fn(),
        moveTo: vi.fn(),
        lineTo: vi.fn(),
        stroke: vi.fn(function trackStroke() {
            this.__strokeStyles.push(this.strokeStyle);
        }),
        strokeRect: vi.fn(),
        fillRect: vi.fn(),
        clearRect: vi.fn(),
        fillText: vi.fn(),
        strokeText: vi.fn(),
        drawImage: vi.fn(),
        setLineDash: vi.fn(),
        measureText: vi.fn(() => ({ width: 10, emHeightAscent: 12, emHeightDescent: 4 })),
    };
}

function flushPromises() {
    return new Promise((resolve) => setTimeout(resolve, 0));
}

describe('terminal compact redraw decoder', () => {
    beforeEach(() => {
        vi.resetModules();
        eventHandlers.clear();
        eventsOnMock.mockClear();
        fontApplyCellStyleMock.mockClear();

        document.body.innerHTML = '<div id="terminal-pane"></div>';

        const ctx = makeContext2d();
        const offCtx = makeContext2d();
        latestTerminalCtx = ctx;
        latestOffscreenCtx = offCtx;

        vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(function getContext(type) {
            if (type !== '2d') {
                return null;
            }
            return this.id === 'ttyphoon-terminal' ? ctx : offCtx;
        });

        window.requestAnimationFrame = (cb) => {
            cb(0);
            return 1;
        };
        window.cancelAnimationFrame = vi.fn();

        class MockImage {
            constructor() {
                this.complete = true;
                this.naturalWidth = 200;
                this.naturalHeight = 100;
                this.onload = null;
                this._src = '';
            }

            set src(value) {
                this._src = value;
                if (typeof this.onload === 'function') {
                    this.onload();
                }
            }

            get src() {
                return this._src;
            }
        }

        window.Image = MockImage;
    });

    it('decodes compact cell flags and packed colours from terminalRedraw', async () => {
        await import('./terminal.js');
        await flushPromises();
        await flushPromises();

        const redraw = eventHandlers.get('terminalRedraw');
        expect(typeof redraw).toBe('function');

        // [op=1(cell), x, y, width, char, flags, fg24, bg24]
        // raw SGR flags: bold(2) + italic(4) + strike(8) + searchResult(256) + underlineStyle1(1<<10=1024) = 1294
        redraw([
            [1, 3, 2, 1, 'X', 1294, 0x112233, 0x445566],
        ]);

        expect(fontApplyCellStyleMock).toHaveBeenCalledTimes(1);
        expect(fontApplyCellStyleMock).toHaveBeenCalledWith(expect.objectContaining({
            op: 'cell',
            x: 3,
            y: 2,
            width: 1,
            char: 'X',
            bold: true,
            italic: true,
            underlineStyle: 1,
            strike: true,
            searchResult: true,
            fg: { Red: 0x11, Green: 0x22, Blue: 0x33 },
            bg: { Red: 0x44, Green: 0x55, Blue: 0x66 },
        }));
    });

    it('decodes compact block chrome folded flag and colour from terminalRedraw', async () => {
        await import('./terminal.js');
        await flushPromises();
        await flushPromises();

        const redraw = eventHandlers.get('terminalRedraw');
        expect(typeof redraw).toBe('function');

        drawBlockChromeMock.mockClear();

        // [op=5(block_chrome), x, y, height, endX, fg24, flags]
        // flags: folded only = 1 << 7 = 128
        redraw([
            [5, 1, 4, 3, 20, 0x778899, 128],
        ]);

        expect(drawBlockChromeMock).toHaveBeenCalledTimes(1);
        expect(drawBlockChromeMock).toHaveBeenCalledWith(
            expect.any(Object),
            expect.any(Function),
            expect.objectContaining({
                op: 'block_chrome',
                x: 1,
                y: 4,
                height: 3,
                endX: 20,
                folded: true,
                fg: { Red: 0x77, Green: 0x88, Blue: 0x99 },
            }),
        );
    });

    it('decodes compact image scale values from terminalRedraw fixed-point fields', async () => {
        await import('./terminal.js');
        await flushPromises();
        await flushPromises();

        const imageCachePut = eventHandlers.get('terminalImageCachePut');
        const redraw = eventHandlers.get('terminalRedraw');
        expect(typeof imageCachePut).toBe('function');
        expect(typeof redraw).toBe('function');

        latestOffscreenCtx.drawImage.mockClear();

        imageCachePut({ id: 42, data: 'data:image/png;base64,AAAA' });

        // [op=9(image), x, y, width, height, imageId, srcWidth, srcHeight, sx1000, sy1000]
        redraw([
            [9, 2, 3, 4, 2, 42, 1, 1, 500, 250],
        ]);

        expect(latestOffscreenCtx.drawImage).toHaveBeenCalledTimes(1);

        const call = latestOffscreenCtx.drawImage.mock.calls[0];
        expect(call[1]).toBe(0);
        expect(call[2]).toBe(0);
        expect(call[3]).toBe(100); // naturalWidth(200) * 0.5
        expect(call[4]).toBe(25);  // naturalHeight(100) * 0.25
        expect(call[5]).toBe(20);  // x=2 * cellWidth=10
        expect(call[6]).toBe(60);  // y=3 * cellHeight=20
        expect(call[7]).toBe(40);  // width=4 * cellWidth=10
        expect(call[8]).toBe(40);  // height=2 * cellHeight=20
    });

    it('decodes compact table boundaries and draws expected vertical lines', async () => {
        await import('./terminal.js');
        await flushPromises();
        await flushPromises();

        const redraw = eventHandlers.get('terminalRedraw');
        expect(typeof redraw).toBe('function');

        latestOffscreenCtx.moveTo.mockClear();
        latestOffscreenCtx.lineTo.mockClear();

        // [op=10(table), x, y, height, width, fg24, boundaries[]]
        redraw([
            [10, 1, 2, 3, 8, 0xAABBCC, [2, 5, 8]],
        ]);

        // cellWidth=10, cellHeight=20 => x=10, y=40, h=100
        expect(latestOffscreenCtx.moveTo).toHaveBeenCalledWith(10, 40);
        expect(latestOffscreenCtx.lineTo).toHaveBeenCalledWith(10, 100);

        expect(latestOffscreenCtx.moveTo).toHaveBeenCalledWith(30, 40);
        expect(latestOffscreenCtx.lineTo).toHaveBeenCalledWith(30, 100);

        expect(latestOffscreenCtx.moveTo).toHaveBeenCalledWith(60, 40);
        expect(latestOffscreenCtx.lineTo).toHaveBeenCalledWith(60, 100);

        expect(latestOffscreenCtx.moveTo).toHaveBeenCalledWith(90, 40);
        expect(latestOffscreenCtx.lineTo).toHaveBeenCalledWith(90, 100);
    });

    it('renders dashed underline using cell text colour', async () => {
        await import('./terminal.js');
        await flushPromises();
        await flushPromises();

        const redraw = eventHandlers.get('terminalRedraw');
        expect(typeof redraw).toBe('function');

        latestOffscreenCtx.__strokeStyles = [];
        latestOffscreenCtx.stroke.mockClear();

        // [op=1(cell), x, y, width, char, flags, fg24, bg24]
        // raw SGR flags: dashed underline style 5 => 5 << 10 = 5120
        redraw([
            [1, 0, 0, 1, 'X', 5120, 0x112233, 0x000000],
        ]);

        expect(latestOffscreenCtx.stroke).toHaveBeenCalled();
        expect(latestOffscreenCtx.__strokeStyles).toContain('rgb(17, 34, 51)');
    });

    it('decodes compact underline colour from terminalRedraw', async () => {
        await import('./terminal.js');
        await flushPromises();
        await flushPromises();

        const redraw = eventHandlers.get('terminalRedraw');
        expect(typeof redraw).toBe('function');

        // raw SGR flags: underlineStyle1 => 1 << 10 = 1024
        redraw([
            [1, 0, 0, 1, 'X', 1024, 0x112233, 0x445566, 0xAABBCC],
        ]);

        expect(fontApplyCellStyleMock).toHaveBeenCalledWith(expect.objectContaining({
            ulc: { Red: 0xAA, Green: 0xBB, Blue: 0xCC },
        }));
    });

    it('renders underline with explicit underline colour when provided', async () => {
        await import('./terminal.js');
        await flushPromises();
        await flushPromises();

        const redraw = eventHandlers.get('terminalRedraw');
        expect(typeof redraw).toBe('function');

        latestOffscreenCtx.__strokeStyles = [];
        latestOffscreenCtx.stroke.mockClear();

        // [op=1(cell), x, y, width, char, flags, fg24, bg24, ulc24]
        // raw SGR flags: dashed underline style 5 => 5 << 10 = 5120
        redraw([
            [1, 0, 0, 1, 'X', 5120, 0x112233, 0x000000, 0x8899AA],
        ]);

        expect(latestOffscreenCtx.stroke).toHaveBeenCalled();
        expect(latestOffscreenCtx.__strokeStyles).toContain('rgb(136, 153, 170)');
    });
});
