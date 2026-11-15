// Renders a Notes layout tree (notes_layout.js) into the DOM. View leaves are
// the views' own root elements, supplied by `getViewRoot`; split branches are
// `.notes-layout-split` flex containers, with a `.notes-layout-divider`
// between each pair of children. Elements already in the right place are not
// touched, because moving a view's root reloads its iframes and resets
// Monaco's layout.

import { equaliseLayoutDivider, resizeLayoutDivider } from './notes_layout.js';

const SPLIT_CLASS = 'notes-layout-split';
const DIVIDER_CLASS = 'notes-layout-divider';
// The smallest a view can be dragged, in pixels (ADR 0100 question 20).
const MIN_VIEW_PIXELS = 120;

export function renderNotesLayout(host, tree, getViewRoot) {
    if (!host) {
        return;
    }
    if (!tree) {
        host.replaceChildren();
        return;
    }
    placeNode(host, 0, tree, '', getViewRoot, []);
    removeChildrenFrom(host, 1);
}

function placeNode(parent, index, node, flex, getViewRoot, path) {
    let el;
    if (node.type === 'view') {
        el = getViewRoot(node.id);
        if (!el) {
            throw new Error(`no root element for view ${node.id}`);
        }
    } else {
        const existing = parent.children[index];
        el = existing?.classList.contains(SPLIT_CLASS) ? existing : document.createElement('div');
        el.classList.add(SPLIT_CLASS);
        if (el.dataset.direction !== node.direction) {
            el.dataset.direction = node.direction;
        }
    }
    if (parent.children[index] !== el) {
        parent.insertBefore(el, parent.children[index] || null);
    }
    if (el.style.flex !== flex) {
        el.style.flex = flex;
    }
    if (node.type === 'split') {
        node.children.forEach((child, childIndex) => {
            if (childIndex > 0) {
                placeDivider(el, childIndex * 2 - 1, path, childIndex - 1);
            }
            placeNode(el, childIndex * 2, child, `${node.sizes[childIndex]} 1 0px`, getViewRoot, [...path, childIndex]);
        });
        removeChildrenFrom(el, node.children.length * 2 - 1);
    }
}

function placeDivider(parent, index, path, dividerIndex) {
    const existing = parent.children[index];
    const el = existing?.classList.contains(DIVIDER_CLASS) ? existing : document.createElement('div');
    if (el !== existing) {
        el.className = DIVIDER_CLASS;
        el.setAttribute('role', 'separator');
        parent.insertBefore(el, existing || null);
    }
    const pathText = path.join('.');
    if (el.dataset.path !== pathText) {
        el.dataset.path = pathText;
    }
    if (el.dataset.index !== String(dividerIndex)) {
        el.dataset.index = String(dividerIndex);
    }
}

function dividerTarget(el) {
    return {
        path: el.dataset.path ? el.dataset.path.split('.').map(Number) : [],
        index: Number(el.dataset.index),
    };
}

// Dragging a divider resizes the two children either side of it, and
// double-clicking it gives them equal shares (ADR 0100 question 20). Each move
// resizes from the layout as the drag began, so clamping doesn't drift.
export function bindNotesLayoutDividers(host, { getLayout, setLayout }) {
    let drag = null;

    host.addEventListener('pointerdown', (event) => {
        const divider = event.target instanceof Element ? event.target.closest(`.${DIVIDER_CLASS}`) : null;
        if (!divider || event.button !== 0) {
            return;
        }
        event.preventDefault();
        const split = divider.parentElement;
        const horizontal = split.dataset.direction !== 'column';
        const span = [...split.children]
            .filter((child) => !child.classList.contains(DIVIDER_CLASS))
            .reduce((total, child) => {
                const rect = child.getBoundingClientRect();
                return total + (horizontal ? rect.width : rect.height);
            }, 0);
        drag = {
            divider,
            horizontal,
            span,
            start: horizontal ? event.clientX : event.clientY,
            layout: getLayout(),
            ...dividerTarget(divider),
        };
        divider.setPointerCapture?.(event.pointerId);
        divider.dataset.dragging = 'true';
    });

    host.addEventListener('pointermove', (event) => {
        if (!drag || drag.span <= 0) {
            return;
        }
        const delta = ((drag.horizontal ? event.clientX : event.clientY) - drag.start) / drag.span;
        const min = Math.min(0.5, MIN_VIEW_PIXELS / drag.span);
        setLayout(resizeLayoutDivider(drag.layout, drag.path, drag.index, delta, { min }));
    });

    const endDrag = () => {
        if (drag) {
            delete drag.divider.dataset.dragging;
            drag = null;
        }
    };
    host.addEventListener('pointerup', endDrag);
    host.addEventListener('pointercancel', endDrag);

    host.addEventListener('dblclick', (event) => {
        const divider = event.target instanceof Element ? event.target.closest(`.${DIVIDER_CLASS}`) : null;
        if (divider) {
            const { path, index } = dividerTarget(divider);
            setLayout(equaliseLayoutDivider(getLayout(), path, index));
        }
    });
}

function removeChildrenFrom(parent, index) {
    while (parent.children.length > index) {
        parent.lastElementChild.remove();
    }
}
