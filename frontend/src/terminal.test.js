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
        accent: { Red: 120, Green: 180, Blue: 240 },
        yellow: { Red: 220, Green: 180, Blue: 40 },
        selection: { Red: 80, Green: 120, Blue: 200 },
        green: { Red: 70, Green: 180, Blue: 110 },
        red: { Red: 220, Green: 50, Blue: 60 },
        searchResult: { Red: 64, Green: 64, Blue: 255 },
        whiteBright: { Red: 255, Green: 255, Blue: 255 },
    },
}));
const terminalGetTabsMock = vi.fn(async () => []);
const terminalGetPanesMock = vi.fn(async () => []);
const terminalClosePaneMock = vi.fn(async () => {});
const terminalClonePaneMock = vi.fn(async () => {});
const terminalConfirmKillPaneMock = vi.fn(async () => {});
const terminalPaneZoomMock = vi.fn(async () => {});
const terminalCancelPaneAutoCloseMock = vi.fn(async () => {});
const terminalRequestRedrawMock = vi.fn(async () => {});
const terminalMouseLeaveMock = vi.fn(async () => {});
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

vi.mock('./wails-runtime.js', () => ({
    EventsOn: eventsOnMock,
}));

vi.mock('./wails-bindings.js', () => ({
    GetWindowStyle: getWindowStyleMock,
    GetCustomRegexp: vi.fn(async () => []),
    SendIpc: sendIpcMock,
    TerminalCopyImageDataURL: terminalCopyImageDataURLMock,
    TerminalGetTabs: terminalGetTabsMock,
    TerminalGetPanes: terminalGetPanesMock,
    TerminalClosePane: terminalClosePaneMock,
    TerminalClonePane: terminalClonePaneMock,
    TerminalConfirmKillPane: terminalConfirmKillPaneMock,
    TerminalPaneZoom: terminalPaneZoomMock,
    TerminalCancelPaneAutoClose: terminalCancelPaneAutoCloseMock,
    TerminalMouseLeave: terminalMouseLeaveMock,
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
        getGlyphOffset: () => ({ x: 0, y: 2 }),
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
        rect: vi.fn(),
        clip: vi.fn(),
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
        terminalMouseLeaveMock.mockClear();
        terminalGetPanesMock.mockReset().mockResolvedValue([]);
        terminalClosePaneMock.mockReset().mockResolvedValue(undefined);
        terminalClonePaneMock.mockReset().mockResolvedValue(undefined);
        terminalConfirmKillPaneMock.mockReset().mockResolvedValue(undefined);
        terminalPaneZoomMock.mockReset().mockResolvedValue(undefined);
        terminalCancelPaneAutoCloseMock.mockReset().mockResolvedValue(undefined);
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

    it('renders hyperlink tooltip icon rows and literal file values', async () => {
        await import('./terminal.js');
        await flushPromises();
        const rows = [
            ['Scheme', 'f5fc', 'file'],
            ['Path', 'f07b', '/project/docs'],
            ['File', 'f15b', '&lt;filename&gt;-**literal**-`code`.md'],
        ];
        eventHandlers.get('tooltipShow')({ text: rows.map(([label, icon, value]) =>
            `<div class="shared-tooltip-row"><span class="shared-tooltip-icon" role="img" aria-label="${label}">&#x${icon};</span> <code>${value}</code></div>`,
        ).join('\n') });
        await flushPromises();

        const tooltip = document.getElementById('ttyphoon-shared-tooltip');
        expect(tooltip.style.display).toBe('block');
        expect(Array.from(tooltip.querySelectorAll('.shared-tooltip-icon')).map((icon) => icon.textContent)).toEqual([
            '\uf5fc', '\uf07b', '\uf15b',
        ]);
        expect(Array.from(tooltip.querySelectorAll('code')).map((value) => value.textContent)).toEqual([
            'file', '/project/docs', '<filename>-**literal**-`code`.md',
        ]);
        expect(tooltip.querySelector('filename, strong, ul')).toBeNull();
        expect(document.getElementById('ttyphoon-shared-tooltip-style').textContent).toContain('.shared-tooltip-icon');
    });

    it('clears backend hover when the pointer leaves the terminal for Notes', async () => {
        await import('./terminal.js');
        await flushPromises();
        const notes = document.createElement('div');
        notes.id = 'notes-pane';
        document.body.appendChild(notes);
        document.getElementById('ttyphoon-terminal').dispatchEvent(new MouseEvent('mouseleave', {
            relatedTarget: notes,
        }));
        await flushPromises();
        expect(terminalMouseLeaveMock).toHaveBeenCalledOnce();
    });

    it('shows every pane close control but reveals running panes only on hover', async () => {
        await import('./terminal.js');
        await flushPromises();
        await flushPromises();

        const panesChanged = eventHandlers.get('terminalPanes');
        panesChanged([
            { id: '%1', name: 'Running shell', x: 0, y: 0, width: 40, height: 24, canClose: true, finished: false, zoomed: false },
            { id: '%2', name: 'Completed command', x: 41, y: 5, width: 39, height: 19, canClose: true, finished: true, zoomed: true },
        ]);
        let buttons = document.querySelectorAll('.terminal-pane-close');
        let zoomButtons = document.querySelectorAll('.terminal-pane-zoom');
        const cloneButtons = document.querySelectorAll('.terminal-pane-clone');
        expect(buttons).toHaveLength(2);
        expect(zoomButtons).toHaveLength(2);
        expect(cloneButtons).toHaveLength(2);
        expect(cloneButtons[0].textContent).toBe(String.fromCodePoint(0xf24d));
        expect(cloneButtons[0].getAttribute('aria-label')).toBe('Clone pane');
        expect(cloneButtons[0].dataset.hovered).toBe('false');
        expect(cloneButtons[0].nextElementSibling).toBe(zoomButtons[0]);
        expect(buttons[0].dataset.paneId).toBe('%1');
        expect(buttons[0].dataset.hovered).toBe('false');
        expect(buttons[0].getAttribute('aria-label')).toBe('Kill pane');
        expect(buttons[0].classList.contains('terminal-pane-action')).toBe(true);
        expect(buttons[1].dataset.paneId).toBe('%2');
        expect(buttons[1].style.left).toBe('778px');
        expect(buttons[1].style.top).toBe('102px');
        expect(buttons[1].getAttribute('aria-label')).toBe('Close completed pane');
        expect(zoomButtons[0].dataset.hovered).toBe('false');
        expect(zoomButtons[0].classList.contains('terminal-pane-action')).toBe(true);
        expect(zoomButtons[1].dataset.finished).toBe('true');
        expect(zoomButtons[1].getAttribute('aria-label')).toBe('Restore pane');
        expect(zoomButtons[1].nextElementSibling).toBe(buttons[1]);
        expect(document.getElementById('terminal-theme').textContent).toContain(
            '.terminal-pane-action[data-finished="false"][data-hovered="false"]',
        );

        const canvas = document.getElementById('ttyphoon-terminal');
        canvas.width = 800;
        canvas.height = 600;
        canvas.getBoundingClientRect = () => ({ left: 0, top: 0, width: 800, height: 600 });
        document.getElementById('terminal-viewport').dispatchEvent(new MouseEvent('pointermove', {
            bubbles: true, clientX: 100, clientY: 100,
        }));
        expect(buttons[0].dataset.hovered).toBe('true');
        expect(zoomButtons[0].dataset.hovered).toBe('true');
        expect(cloneButtons[0].dataset.hovered).toBe('true');
        cloneButtons[0].click();
        await flushPromises();
        expect(terminalClonePaneMock).toHaveBeenCalledWith();
        expect(terminalConfirmKillPaneMock).not.toHaveBeenCalled();
        const fullsizeToggle = vi.fn();
        window.addEventListener('ttyphoon-terminal-fullsize-toggle', fullsizeToggle);
        zoomButtons[0].click();
        await flushPromises();
        expect(terminalPaneZoomMock).toHaveBeenCalledWith('%1');
        expect(fullsizeToggle).toHaveBeenCalledOnce();
        window.removeEventListener('ttyphoon-terminal-fullsize-toggle', fullsizeToggle);

        buttons[0].click();
        await flushPromises();
        expect(terminalConfirmKillPaneMock).toHaveBeenCalledWith('%1', 'Running shell');
        expect(terminalClosePaneMock).not.toHaveBeenCalled();

        panesChanged([[{ id: '%2', name: 'Completed command', x: 0, y: 0, width: 60, height: 24, canClose: true, finished: true }]]);
        buttons = document.querySelectorAll('.terminal-pane-close');
        expect(buttons).toHaveLength(1);
        expect(buttons[0].style.left).toBe('578px');
        expect(buttons[0].style.top).toBe('2px');
        buttons[0].click();
        await flushPromises();
        expect(terminalClosePaneMock).toHaveBeenCalledWith('%2');
        expect(document.querySelector('.terminal-pane-close')).toBeNull();
        expect(document.getElementById('terminal-pane').hasAttribute('data-completed-panes')).toBe(false);

        panesChanged([]);
        expect(document.querySelector('.terminal-pane-close')).toBeNull();
    });

    it('exits terminal fullsize when the active tmux window changes, but not on same-window refresh', async () => {
        await import('./terminal.js');
        await flushPromises();
        const tabsChanged = eventHandlers.get('terminalTabs');
        const panesChanged = eventHandlers.get('terminalPanes');
        const terminalPane = document.getElementById('terminal-pane');
        const fullsizeToggle = vi.fn();
        window.addEventListener('ttyphoon-terminal-fullsize-toggle', fullsizeToggle);

        tabsChanged({ tabs: [{ id: '@1', active: true }] });
        panesChanged([{ id: '%7', x: 0, y: 0, width: 80, height: 24, canClose: true, zoomed: true }]);
        terminalPane.dataset.fullsize = 'true';
        tabsChanged({ tabs: [{ id: '@1', active: true }] });
        expect(fullsizeToggle).not.toHaveBeenCalled();
        expect(terminalPaneZoomMock).not.toHaveBeenCalled();

        tabsChanged({ tabs: [{ id: '@2', active: true }, { id: '@1', active: false }] });
        expect(fullsizeToggle).toHaveBeenCalledOnce();
        expect(terminalPaneZoomMock).toHaveBeenCalledWith('%7');
        window.removeEventListener('ttyphoon-terminal-fullsize-toggle', fullsizeToggle);
    });

    it('shows the success countdown without resetting it on layout changes and lets it be cancelled', async () => {
        await import('./terminal.js');
        await flushPromises();
        await flushPromises();
        const panesChanged = eventHandlers.get('terminalPanes');
        vi.useFakeTimers();
        try {
            const pane = { id: '%2', x: 0, y: 0, width: 80, height: 24, canClose: true, finished: true, autoCloseAt: Date.now() + 30000 };
            panesChanged([pane]);
            let timer = document.querySelector('.terminal-pane-timer');
            expect(Array.from(document.getElementById('terminal-pane-controls').children, (element) =>
                element.classList.contains('terminal-pane-clone') ? 'clone'
                    : element.classList.contains('terminal-pane-zoom') ? 'zoom'
                        : element.classList.contains('terminal-pane-timer') ? 'timer' : 'close')).toEqual(['clone', 'zoom', 'timer', 'close']);
            expect(timer.getAttribute('aria-label')).toContain('30s');
            expect(timer.firstElementChild.style.getPropertyValue('--progress-deg')).toBe('0deg');
            await vi.advanceTimersByTimeAsync(15000);
            expect(timer.firstElementChild.style.getPropertyValue('--progress-deg')).toBe('180deg');
            panesChanged([{ ...pane, width: 60 }]);
            timer = document.querySelector('.terminal-pane-timer');
            expect(timer.getAttribute('aria-label')).toContain('15s');
            timer.click();
            await vi.advanceTimersByTimeAsync(0);
            expect(terminalCancelPaneAutoCloseMock).toHaveBeenCalledWith('%2');
            expect(document.querySelector('.terminal-pane-timer')).toBeNull();
            expect(document.querySelector('.terminal-pane-close')).not.toBeNull();
            await vi.advanceTimersByTimeAsync(31000);
            expect(terminalClosePaneMock).not.toHaveBeenCalled();
            document.querySelector('.terminal-pane-close').click();
            await vi.advanceTimersByTimeAsync(0);
            expect(terminalClosePaneMock).toHaveBeenCalledWith('%2');
        } finally {
            panesChanged([]);
            vi.useRealTimers();
        }
    });

    it('uses the configured auto-close duration for countdown progress', async () => {
        await import('./terminal.js');
        await flushPromises();
        await flushPromises();
        vi.useFakeTimers();
        try {
            const panesChanged = eventHandlers.get('terminalPanes');
            panesChanged([{
                id: '%5', x: 0, y: 0, width: 80, height: 24,
                canClose: true, finished: true,
                autoCloseAt: Date.now() + 12000,
                autoCloseDuration: 12000,
            }]);
            const timer = document.querySelector('.terminal-pane-timer');
            expect(timer.getAttribute('aria-label')).toContain('12s');
            await vi.advanceTimersByTimeAsync(6000);
            expect(timer.firstElementChild.style.getPropertyValue('--progress-deg')).toBe('180deg');
            expect(timer.getAttribute('aria-label')).toContain('6s');
        } finally {
            eventHandlers.get('terminalPanes')([]);
            vi.useRealTimers();
        }
    });

    it('loads retained pane controls from the initial snapshot', async () => {
        terminalGetPanesMock.mockResolvedValue([
            { id: '%3', x: 0, y: 0, width: 80, height: 24, canClose: true, finished: true, zoomed: false },
        ]);
        await import('./terminal.js');
        await flushPromises();
        await flushPromises();
        expect(document.querySelector('.terminal-pane-close')?.dataset.paneId).toBe('%3');
    });

    it.each([
        [1, 'rgb(22, 52, 46)'],
        [2, 'rgb(52, 26, 36)'],
    ])('tints completed pane backgrounds at 20 percent without borders for result %s', async (result, colour) => {
        await import('./terminal.js');
        await flushPromises();
        await flushPromises();
        const redraw = eventHandlers.get('terminalRedraw');
        latestOffscreenCtx.strokeRect.mockClear();
        redraw([[2, 0, 0, 40, 24, result]]);
        expect(latestOffscreenCtx.fillStyle).toBe(colour);
        expect(latestOffscreenCtx.strokeRect).not.toHaveBeenCalled();
        redraw([[1, 1, 1, 1, 'X', 0, 0xAABBCC, 0]]);
        expect(latestOffscreenCtx.fillStyle).toBe('rgb(170, 187, 204)');
        redraw([[2, 41, 0, 39, 24, 0]]);
        expect(latestOffscreenCtx.fillStyle).toBe('rgb(10, 20, 30)');
    });

    it('splits a stacked-pane divider background at the half-cell pixel', async () => {
        await import('./terminal.js');
        await flushPromises();
        await flushPromises();
        const redraw = eventHandlers.get('terminalRedraw');
        latestOffscreenCtx.fillRect.mockClear();

        redraw([
            [2, 0, -1, 80, 24, 1 << 3],
            [2, 0, 23, 80, 25, 1 << 2],
        ]);

        const frameFills = latestOffscreenCtx.fillRect.mock.calls.filter(([, , width]) => width > 700);
        expect(frameFills[0][1]).toBe(0);
        expect(frameFills[0][3]).toBe(470);
        expect(frameFills[1][1]).toBe(470);
    });

    it('keeps the background continuous across two dividers in a three-pane stack', async () => {
        await import('./terminal.js');
        await flushPromises();
        await flushPromises();
        const redraw = eventHandlers.get('terminalRedraw');
        latestOffscreenCtx.fillRect.mockClear();

        redraw([
            [2, 0, -1, 80, 8, 1 << 3],
            [2, 0, 7, 80, 8, (1 << 2) | (1 << 3)],
            [2, 0, 15, 80, 8, 1 << 2],
        ]);

        const frameFills = latestOffscreenCtx.fillRect.mock.calls.filter(([, , width]) => width > 700);
        expect(frameFills.map(([, y, , height]) => [y, height])).toEqual([
            [0, 150], [150, 160], [310, 152],
        ]);
    });

    it('splits the inactive-pane hint at the same divider midpoint', async () => {
        await import('./terminal.js');
        await flushPromises();
        await flushPromises();
        const redraw = eventHandlers.get('terminalRedraw');
        latestOffscreenCtx.fillRect.mockClear();
        window.terminalFocusedState = true;

        redraw([
            [8, 0, 23, 81, 1, 51, 1],
            [8, 0, 23, 81, 1, 51, 2],
        ]);

        expect(latestOffscreenCtx.fillRect).toHaveBeenNthCalledWith(1, 0, 460, 810, 10);
        expect(latestOffscreenCtx.fillRect).toHaveBeenNthCalledWith(2, 0, 470, 810, 10);
    });

    it('vertically centres glyphs within adjusted cells', async () => {
        await import('./terminal.js');
        await flushPromises();
        await flushPromises();

        const redraw = eventHandlers.get('terminalRedraw');
        latestOffscreenCtx.fillText.mockClear();

        redraw([
            [1, 2, 3, 1, 'X', 0, 0xFFFFFF, 0x000000],
        ]);

        expect(latestOffscreenCtx.fillText).toHaveBeenCalledWith('X', 20, 62);
    });

    it('clips glyphs to their allocated cells before drawing', async () => {
        await import('./terminal.js');
        await flushPromises();
        await flushPromises();

        const redraw = eventHandlers.get('terminalRedraw');
        latestOffscreenCtx.rect.mockClear();
        latestOffscreenCtx.clip.mockClear();
        latestOffscreenCtx.fillText.mockClear();

        redraw([
            [1, 2, 3, 2, 'XX', 0, 0xFFFFFF, 0x000000],
        ]);

        expect(latestOffscreenCtx.rect).toHaveBeenCalledWith(20, 60, 20, 20);
        expect(latestOffscreenCtx.clip).toHaveBeenCalledTimes(1);
        expect(latestOffscreenCtx.rect.mock.invocationCallOrder[0])
            .toBeLessThan(latestOffscreenCtx.fillText.mock.invocationCallOrder[0]);
    });

    it('applies redraw batches that arrive while a canvas paint is pending', async () => {
        await import('./terminal.js');
        await flushPromises();
        await flushPromises();

        const frameCallbacks = [];
        window.requestAnimationFrame = (callback) => {
            frameCallbacks.push(callback);
            return frameCallbacks.length;
        };
        const redraw = eventHandlers.get('terminalRedraw');
        latestOffscreenCtx.fillRect.mockClear();

        redraw([[2, 0, 0, 20, 10]]);
        redraw([[2, 20, 0, 20, 10]]);

        expect(latestOffscreenCtx.fillRect).toHaveBeenCalledTimes(2);
        expect(latestOffscreenCtx.fillRect).toHaveBeenNthCalledWith(1, 0, 0, 202, 202);
        expect(latestOffscreenCtx.fillRect).toHaveBeenNthCalledWith(2, 200, 0, 202, 202);

        frameCallbacks[0](0);
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
        // flags: folded only = 1 << 5 = 32
        redraw([
            [5, 1, 4, 3, 20, 0x778899, 32],
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
