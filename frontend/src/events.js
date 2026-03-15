import {
    TerminalTextInput,
    TerminalKeyPress,
    TerminalMouseButton,
    TerminalMouseMotion,
    TerminalMouseWheel,
} from '../wailsjs/go/main/WApp';

function mouseButtonToGo(button) {
    switch (button) {
    case 0:
        return 1;
    case 1:
        return 2;
    case 2:
        return 3;
    case 3:
        return 4;
    case 4:
        return 5;
    default:
        return 1;
    }
}

function eventToCell(canvas, event, getCellSize) {
    const { cellWidth, cellHeight } = getCellSize();
    const rect = canvas.getBoundingClientRect();
    const x = Math.floor((event.clientX - rect.left) / cellWidth);
    const y = Math.floor((event.clientY - rect.top) / cellHeight);
    return { x, y };
}

function isEditableTarget(target) {
    if (!target || !(target instanceof HTMLElement)) {
        return false;
    }

    if (target.isContentEditable) {
        return true;
    }

    const tag = target.tagName;
    return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT';
}

export function wireKeyboardEvents(canvas) {
    canvas.tabIndex = 0;
    canvas.style.outline = 'none';

    canvas.addEventListener('mousedown', () => {
        canvas.focus();
    });

    window.addEventListener('keydown', (event) => {
        if (isEditableTarget(event.target) || event.isComposing) {
            return;
        }

        const isTextInput = event.key &&
            event.key.length === 1 &&
            !event.ctrlKey &&
            !event.altKey &&
            !event.metaKey;

        if (isTextInput) {
            event.preventDefault();
            TerminalTextInput(event.key).catch(() => {});
            return;
        }

        event.preventDefault();
        TerminalKeyPress(
            event.key,
            event.ctrlKey,
            event.altKey,
            event.shiftKey,
            event.metaKey,
        ).catch(() => {});
    });
}

export function wireMouseEvents(canvas, getCellSize) {
    let lastMouseCell = { x: 0, y: 0 };

    canvas.addEventListener('contextmenu', (event) => {
        event.preventDefault();
    });

    canvas.addEventListener('mousedown', (event) => {
        const pos = eventToCell(canvas, event, getCellSize);
        lastMouseCell = pos;
        TerminalMouseButton(
            pos.x,
            pos.y,
            mouseButtonToGo(event.button),
            event.detail || 1,
            true,
        ).catch(() => {});
    });

    canvas.addEventListener('mouseup', (event) => {
        const pos = eventToCell(canvas, event, getCellSize);
        lastMouseCell = pos;
        TerminalMouseButton(
            pos.x,
            pos.y,
            mouseButtonToGo(event.button),
            event.detail || 1,
            false,
        ).catch(() => {});
    });

    canvas.addEventListener('mousemove', (event) => {
        const pos = eventToCell(canvas, event, getCellSize);
        const relX = pos.x - lastMouseCell.x;
        const relY = pos.y - lastMouseCell.y;
        lastMouseCell = pos;
        TerminalMouseMotion(
            pos.x,
            pos.y,
            relX,
            relY,
            event.buttons,
        ).catch(() => {});
    });

    canvas.addEventListener('wheel', (event) => {
        event.preventDefault();
        const pos = eventToCell(canvas, event, getCellSize);
        const moveX = Math.sign(event.deltaX);
        const moveY = -Math.sign(event.deltaY);
        TerminalMouseWheel(
            pos.x,
            pos.y,
            moveX,
            moveY,
        ).catch(() => {});
    }, { passive: false });
}
