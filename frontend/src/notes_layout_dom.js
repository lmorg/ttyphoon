// Renders a Notes layout tree (notes_layout.js) into the DOM. View leaves are
// the views' own root elements, supplied by `getViewRoot`; split branches are
// `.notes-layout-split` flex containers. Elements already in the right place
// are not touched, because moving a view's root reloads its iframes and resets
// Monaco's layout.

const SPLIT_CLASS = 'notes-layout-split';

export function renderNotesLayout(host, tree, getViewRoot) {
    if (!host) {
        return;
    }
    if (!tree) {
        host.replaceChildren();
        return;
    }
    placeNode(host, 0, tree, '', getViewRoot);
    removeChildrenFrom(host, 1);
}

function placeNode(parent, index, node, flex, getViewRoot) {
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
            placeNode(el, childIndex, child, `${node.sizes[childIndex]} 1 0px`, getViewRoot);
        });
        removeChildrenFrom(el, node.children.length);
    }
}

function removeChildrenFrom(parent, index) {
    while (parent.children.length > index) {
        parent.lastElementChild.remove();
    }
}
