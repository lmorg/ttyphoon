// Notes layout tree (ADR 0100 decision 2). Leaves are views, `{ type: 'view',
// id }`; branches are splits, `{ type: 'split', direction, children, sizes }`.
// `direction` is 'row' (side by side) or 'column' (stacked), as in CSS flex,
// and `sizes` are the children's shares of the split, summing to 1.
//
// Every function returns a new tree and leaves its argument unchanged. Trees
// are kept normalised: no split has fewer than two children, and no split
// directly contains a split of the same direction.

export const LAYOUT_ROW = 'row';
export const LAYOUT_COLUMN = 'column';

const DEFAULT_MIN_SIZE = 0.05;

export function createLayout(viewId) {
    return { type: 'view', id: String(viewId) };
}

export function layoutViewIds(node) {
    if (!node) {
        return [];
    }
    if (node.type === 'view') {
        return [node.id];
    }
    return node.children.flatMap(layoutViewIds);
}

export function findLayoutView(node, viewId) {
    if (!node) {
        return null;
    }
    if (node.type === 'view') {
        return node.id === viewId ? node : null;
    }
    for (const child of node.children) {
        const found = findLayoutView(child, viewId);
        if (found) {
            return found;
        }
    }
    return null;
}

// Each view's rectangle in the unit square, from the split sizes.
export function layoutViewRects(node, rect = { x: 0, y: 0, width: 1, height: 1 }) {
    if (!node) {
        return [];
    }
    if (node.type === 'view') {
        return [{ id: node.id, ...rect }];
    }
    let offset = 0;
    return node.children.flatMap((child, index) => {
        const share = node.sizes[index];
        const childRect = node.direction === LAYOUT_ROW
            ? { x: rect.x + offset * rect.width, y: rect.y, width: share * rect.width, height: rect.height }
            : { x: rect.x, y: rect.y + offset * rect.height, width: rect.width, height: share * rect.height };
        offset += share;
        return layoutViewRects(child, childRect);
    });
}

const EDGE_EPSILON = 1e-6;

// The view next to `viewId` in `side` ('left', 'right', 'up' or 'down'): of
// the views touching that edge, the one sharing most of it, then the one
// nearest its start. Null at the edge of the layout.
export function neighbourLayoutView(tree, viewId, side) {
    const rects = layoutViewRects(tree);
    const from = rects.find((rect) => rect.id === viewId);
    if (!from) {
        return null;
    }
    const horizontal = side === 'left' || side === 'right';
    const edge = {
        left: from.x,
        right: from.x + from.width,
        up: from.y,
        down: from.y + from.height,
    }[side];
    if (edge === undefined) {
        return null;
    }
    const start = horizontal ? from.y : from.x;
    const end = start + (horizontal ? from.height : from.width);
    let best = null;
    for (const rect of rects) {
        const near = {
            left: rect.x + rect.width,
            right: rect.x,
            up: rect.y + rect.height,
            down: rect.y,
        }[side];
        if (rect.id === viewId || Math.abs(near - edge) > EDGE_EPSILON) {
            continue;
        }
        const otherStart = horizontal ? rect.y : rect.x;
        const otherEnd = otherStart + (horizontal ? rect.height : rect.width);
        const overlap = Math.min(end, otherEnd) - Math.max(start, otherStart);
        if (overlap <= EDGE_EPSILON) {
            continue;
        }
        const distance = Math.abs(otherStart - start);
        if (!best || overlap > best.overlap + EDGE_EPSILON
            || (Math.abs(overlap - best.overlap) <= EDGE_EPSILON && distance < best.distance)) {
            best = { id: rect.id, overlap, distance };
        }
    }
    return best?.id ?? null;
}

// Splits `viewId` in `direction`, putting the new view after it (right or
// below), or before it when `before` is set. The new view takes half of the
// split view's space.
export function splitLayoutView(tree, viewId, newView, direction, { before = false } = {}) {
    if (!isDirection(direction)) {
        throw new Error(`unknown layout direction: ${direction}`);
    }
    const leaf = typeof newView === 'object' && newView ? { ...newView, type: 'view', id: String(newView.id) } : createLayout(newView);
    if (!findLayoutView(tree, viewId)) {
        throw new Error(`no view ${viewId} in layout`);
    }
    if (findLayoutView(tree, leaf.id)) {
        throw new Error(`view ${leaf.id} is already in layout`);
    }
    const pair = (node) => (before ? [leaf, node] : [node, leaf]);

    const visit = (node) => {
        if (node.type === 'view') {
            return node.id === viewId ? { type: 'split', direction, children: pair(node), sizes: [0.5, 0.5] } : node;
        }
        const index = node.children.findIndex((child) => child.type === 'view' && child.id === viewId);
        if (index >= 0 && node.direction === direction) {
            const half = node.sizes[index] / 2;
            return {
                ...node,
                children: [...node.children.slice(0, index), ...pair(node.children[index]), ...node.children.slice(index + 1)],
                sizes: [...node.sizes.slice(0, index), half, half, ...node.sizes.slice(index + 1)],
            };
        }
        return { ...node, children: node.children.map(visit) };
    };
    return normaliseLayout(visit(tree));
}

// Removes `viewId`. Its space goes to its neighbours in proportion to their
// sizes. The only view in a layout is never removed.
export function closeLayoutView(tree, viewId) {
    if (!tree || tree.type === 'view' || !findLayoutView(tree, viewId)) {
        return tree;
    }
    const visit = (node) => {
        if (node.type === 'view') {
            return node.id === viewId ? null : node;
        }
        const children = [];
        const sizes = [];
        node.children.forEach((child, index) => {
            const next = visit(child);
            if (next) {
                children.push(next);
                sizes.push(node.sizes[index]);
            }
        });
        return children.length ? { ...node, children, sizes } : null;
    };
    return normaliseLayout(visit(tree));
}

// Moves the divider between children `index` and `index + 1` of the split at
// `path` (child indices from the root) by `delta`, a share of that split.
// Neither child shrinks below `min`.
export function resizeLayoutDivider(tree, path, index, delta, { min = DEFAULT_MIN_SIZE } = {}) {
    return updateSplitAt(tree, path, (split) => {
        if (index < 0 || index + 1 >= split.children.length) {
            return split;
        }
        const first = split.sizes[index];
        const second = split.sizes[index + 1];
        const total = first + second;
        const floor = Math.min(min, total / 2);
        const nextFirst = Math.min(total - floor, Math.max(floor, first + Number(delta || 0)));
        const sizes = [...split.sizes];
        sizes[index] = nextFirst;
        sizes[index + 1] = total - nextFirst;
        return { ...split, sizes };
    });
}

// Gives children `index` and `index + 1` of the split at `path` equal shares
// of the space they occupy together.
export function equaliseLayoutDivider(tree, path, index) {
    return updateSplitAt(tree, path, (split) => {
        if (index < 0 || index + 1 >= split.children.length) {
            return split;
        }
        const half = (split.sizes[index] + split.sizes[index + 1]) / 2;
        const sizes = [...split.sizes];
        sizes[index] = half;
        sizes[index + 1] = half;
        return { ...split, sizes };
    });
}

export function serialiseLayout(tree) {
    return JSON.parse(JSON.stringify(tree));
}

// Rebuilds a layout from saved data, returning null when nothing usable is
// left. `parseView` turns a saved leaf into a view leaf, or null to drop it;
// by default leaves keep only their id. Duplicate view ids, unknown nodes and
// invalid sizes are dropped or repaired.
export function parseLayout(value, { parseView = (leaf) => createLayout(leaf.id) } = {}) {
    const seen = new Set();
    const visit = (node) => {
        if (!node || typeof node !== 'object') {
            return null;
        }
        if (node.type === 'view') {
            if (typeof node.id !== 'string' || !node.id || seen.has(node.id)) {
                return null;
            }
            const leaf = parseView(node);
            if (!leaf || leaf.type !== 'view' || leaf.id !== node.id) {
                return null;
            }
            seen.add(node.id);
            return leaf;
        }
        if (node.type !== 'split' || !isDirection(node.direction) || !Array.isArray(node.children)) {
            return null;
        }
        const savedSizes = Array.isArray(node.sizes) && node.sizes.length === node.children.length ? node.sizes : [];
        const children = [];
        const sizes = [];
        node.children.forEach((child, index) => {
            const next = visit(child);
            if (next) {
                children.push(next);
                const size = Number(savedSizes[index]);
                sizes.push(Number.isFinite(size) && size > 0 ? size : NaN);
            }
        });
        if (!children.length) {
            return null;
        }
        const repaired = sizes.some(Number.isNaN) ? children.map(() => 1) : sizes;
        return { type: 'split', direction: node.direction, children, sizes: repaired };
    };
    return normaliseLayout(visit(value));
}

export function normaliseLayout(node) {
    if (!node || node.type === 'view') {
        return node || null;
    }
    const children = [];
    const sizes = [];
    node.children.forEach((child, index) => {
        const next = normaliseLayout(child);
        if (!next) {
            return;
        }
        const size = node.sizes[index];
        if (next.type === 'split' && next.direction === node.direction) {
            next.children.forEach((grandchild, inner) => {
                children.push(grandchild);
                sizes.push(size * next.sizes[inner]);
            });
            return;
        }
        children.push(next);
        sizes.push(size);
    });
    if (!children.length) {
        return null;
    }
    if (children.length === 1) {
        return children[0];
    }
    const total = sizes.reduce((sum, size) => sum + size, 0);
    return {
        ...node,
        children,
        sizes: total > 0 ? sizes.map((size) => size / total) : children.map(() => 1 / children.length),
    };
}

function updateSplitAt(tree, path, update) {
    const visit = (node, depth) => {
        if (!node || node.type !== 'split') {
            return node;
        }
        if (depth === path.length) {
            return update(node);
        }
        const index = path[depth];
        if (index < 0 || index >= node.children.length) {
            return node;
        }
        const children = [...node.children];
        children[index] = visit(children[index], depth + 1);
        return { ...node, children };
    };
    return visit(tree, 0);
}

function isDirection(direction) {
    return direction === LAYOUT_ROW || direction === LAYOUT_COLUMN;
}
