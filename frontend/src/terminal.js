import { GetWindowStyle } from '../wailsjs/go/main/WApp';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { wireKeyboardEvents, wireMouseEvents } from './events';
import { createFontController } from './font';
import { drawGauge } from './gauge';
import { drawBlockChrome } from './block_chrome';
import { initTerminalPopupMenu } from './popup_menu';

document.querySelector('#app').innerHTML = `
    <canvas id="ttyphoon-terminal"></canvas>
`;

const canvas = document.getElementById('ttyphoon-terminal');
const ctx = canvas.getContext('2d');
const offscreen = document.createElement('canvas');
//const offscreen = document.getElementById('ttyphoon-terminal-buf');
const offCtx = offscreen.getContext('2d');
const font = createFontController(offCtx);
let windowStyle;
let rafPending = false;

function fitCanvasToWindow() {
    canvas.width = window.innerWidth;
    canvas.height = window.innerHeight;
    offscreen.width = canvas.width;
    offscreen.height = canvas.height;
}

function drawCell(cmd) {
    if (!offCtx) {
        return;
    }

    const { cellWidth, cellHeight } = font.getCellSize();

    const xCell = Number.isFinite(cmd.x) ? cmd.x : 0;
    const yCell = Number.isFinite(cmd.y) ? cmd.y : 0;
    const widthCells = Number.isFinite(cmd.width) && cmd.width > 0 ? cmd.width : 1;

    const x = xCell * cellWidth;
    const y = yCell * cellHeight;
    const width = widthCells * cellWidth;

    if (cmd.bg) {
        offCtx.fillStyle = `rgb(${cmd.bg.Red}, ${cmd.bg.Green}, ${cmd.bg.Blue})`;
        offCtx.fillRect(x, y, width, cellHeight);
    }

    font.applyCellStyle(cmd);

    if (cmd.fg) {
        offCtx.fillStyle = `rgb(${cmd.fg.Red}, ${cmd.fg.Green}, ${cmd.fg.Blue})`;
    } else {
        offCtx.fillStyle = '#ffffff';
    }

    if (cmd.char) {
        offCtx.fillText(cmd.char, x, y);
    }

    if (cmd.underline) {
        const lineY = y + cellHeight - 2;
        offCtx.fillRect(x, lineY, width, 1);
    }

    if (cmd.strike) {
        const lineY = y + Math.floor(cellHeight / 2);
        offCtx.fillRect(x, lineY, width, 1);
    }
}

function drawFrame() {
    if (!offCtx) {
        return;
    }

    const bg = windowStyle?.colors?.bg;
    if (bg) {
        offCtx.fillStyle = `rgb(${bg.Red}, ${bg.Green}, ${bg.Blue})`;
        offCtx.fillRect(0, 0, offscreen.width, offscreen.height);
    } else {
        offCtx.clearRect(0, 0, offscreen.width, offscreen.height);
    }
}

EventsOn("terminalRedraw", ops => {
    if (rafPending) {
        return;
    }
   rafPending = true;

    const drawOps = Array.isArray(ops?.[0]) ? ops[0] : ops;

    if (!Array.isArray(drawOps) || drawOps.length === 0) {
        return;
    }

    for (const cmd of drawOps) {
        if (cmd.op === 'frame') {
            drawFrame();
            continue;
        }
        if (cmd.op === 'cell') {
            drawCell(cmd);
            continue;
        }
        if (cmd.op === 'gauge_h' || cmd.op === 'gauge_v') {
            drawGauge(offCtx, font.getCellSize, cmd);
            continue;
        }
        if (cmd.op === 'block_chrome') {
            drawBlockChrome(offCtx, font.getCellSize, cmd);
        }
    }

    requestAnimationFrame(() => {
        ctx.drawImage(offscreen, 0, 0);
        rafPending = false;
    });
});

GetWindowStyle().then((result) => {
    windowStyle = result;
    document.body.style.margin = '0';
    document.body.style.overflow = 'hidden';
    document.body.style.backgroundColor = `rgb(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue})`;
    font.applyConfiguredFontFromWindowStyle(windowStyle);
    fitCanvasToWindow();
    font.loadGlyphSizeFromGo(windowStyle).then(() => {
        //drawFrame();
        wireKeyboardEvents(canvas);
        wireMouseEvents(canvas, font.getCellSize);
        initTerminalPopupMenu(canvas);
        canvas.focus();
        window['go']['main']['WApp']['TerminalRequestRedraw']().catch(() => {});
    });
});

window.addEventListener('resize', () => {
    fitCanvasToWindow();
    //drawFrame();
});
