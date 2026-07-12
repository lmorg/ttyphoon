import { marked } from "marked";
import { configureMarked, processMarkdownContainer } from './markdown-utils.js';

let tooltipEl = null;
let mouseTrackingBound = false;
let lastMouseX = 0;
let lastMouseY = 0;
let tooltipRenderSeq = 0;

configureMarked();

function ensureTooltipStyle() {
    if (document.getElementById('ttyphoon-shared-tooltip-style')) {
        return;
    }

    const style = document.createElement('style');
    style.id = 'ttyphoon-shared-tooltip-style';
    style.textContent = `
        #ttyphoon-shared-tooltip {
            position: fixed;
            z-index: 9003;
            pointer-events: none;
            max-width: 60vw;
            padding: 6px 10px;
            border-radius: 8px;
            border: 1px solid var(--terminal-menu-border, rgba(127,127,127,0.35));
            background: color-mix(in srgb, var(--terminal-menu-bg, var(--bg)) 75%, transparent);
            -webkit-backdrop-filter: blur(3px);
            backdrop-filter: blur(3px);
            color: var(--terminal-menu-fg, var(--fg));
            font-family: var(--terminal-menu-font, var(--font-family, monospace));
            font-size: var(--terminal-menu-font-size, 12px);
            box-shadow: 0 12px 30px rgba(0,0,0,0.45);
            white-space: normal;
            overflow-wrap: anywhere;
            opacity: 0.92;
            animation: tty-menu-appear 0.12s ease-out;
            display: none;
        }

        #ttyphoon-shared-tooltip > :first-child {
            margin-top: 0;
        }

        #ttyphoon-shared-tooltip > :last-child {
            margin-bottom: 0;
        }

        #ttyphoon-shared-tooltip :where(p, ul, ol, pre, blockquote, table) {
            margin-top: 0.4em;
            margin-bottom: 0.4em;
        }

        #ttyphoon-shared-tooltip li > p {
            margin-top: 0;
            margin-bottom: 0;
        }

        #ttyphoon-shared-tooltip :where(code, pre) {
            font-family: var(--terminal-menu-font, var(--font-family, monospace));
        }
    `;
    document.head.appendChild(style);
}

function ensureTooltipElement() {
    ensureTooltipStyle();

    if (tooltipEl && document.body.contains(tooltipEl)) {
        return tooltipEl;
    }

    tooltipEl = document.getElementById('ttyphoon-shared-tooltip');
    if (tooltipEl) {
        return tooltipEl;
    }

    tooltipEl = document.createElement('div');
    tooltipEl.id = 'ttyphoon-shared-tooltip';
    tooltipEl.className = 'markdown-body';
    document.body.appendChild(tooltipEl);
    return tooltipEl;
}

function positionTooltip(x = lastMouseX, y = lastMouseY) {
    const el = ensureTooltipElement();
    if (el.style.display !== 'block') {
        return;
    }

    const pointerX = Number.isFinite(Number(x)) ? Number(x) : lastMouseX;
    const pointerY = Number.isFinite(Number(y)) ? Number(y) : lastMouseY;
    const nextX = Math.min(pointerX + 14, window.innerWidth - el.offsetWidth - 8);
    const nextY = Math.min(pointerY + 14, window.innerHeight - el.offsetHeight - 8);
    el.style.left = `${Math.max(8, nextX)}px`;
    el.style.top = `${Math.max(8, nextY)}px`;
}

export function updateSharedTooltipPointer(x, y) {
    if (Number.isFinite(Number(x))) {
        lastMouseX = Number(x);
    }
    if (Number.isFinite(Number(y))) {
        lastMouseY = Number(y);
    }
    positionTooltip(lastMouseX, lastMouseY);
}

export async function showSharedTooltip(text) {
    const value = String(text || '').trim();
    const el = ensureTooltipElement();
    if (!value) {
        closeSharedTooltip();
        return;
    }

    const renderSeq = tooltipRenderSeq + 1;
    tooltipRenderSeq = renderSeq;

    el.innerHTML = marked.parse(value);
    await processMarkdownContainer(el);
    if (tooltipRenderSeq !== renderSeq) {
        return;
    }

    el.style.display = 'block';
    positionTooltip(lastMouseX, lastMouseY);
}

export function closeSharedTooltip() {
    tooltipRenderSeq += 1;
    const el = ensureTooltipElement();
    el.style.display = 'none';
    el.innerHTML = '';
}

export function bindSharedTooltipMouseTracking() {
    ensureTooltipElement();
    if (mouseTrackingBound) {
        return;
    }

    mouseTrackingBound = true;
    window.addEventListener('mousemove', (event) => {
        updateSharedTooltipPointer(event.clientX, event.clientY);
    });
}

export function bindSharedTooltipEvents(eventsOn) {
    bindSharedTooltipMouseTracking();
    eventsOn('tooltipShow', (payload) => {
        const p = Array.isArray(payload?.[0]) ? payload[0] : payload;
        void showSharedTooltip(p?.text);
    });
    eventsOn('tooltipClose', () => {
        closeSharedTooltip();
    });
}