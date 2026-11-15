import viewTemplate from './notes_view.html?raw';

let nextViewId = 1;

// Creates a Notes editor view (mode tabs, editor host and preview/structured
// view hosts) from one template. Parts are looked up within the view's root by
// their data-part name, never by global ID, so several views can coexist
// (ADR 0100 decision 3).
export function createNotesView({ host, before = null } = {}) {
    const template = document.createElement('template');
    template.innerHTML = viewTemplate.trim();
    const root = template.content.firstElementChild;
    const id = `notes-view-${nextViewId++}`;
    root.dataset.viewId = id;

    const parts = { root };
    root.querySelectorAll('[data-part]').forEach((el) => {
        parts[el.dataset.part] = el;
    });

    if (host) {
        host.insertBefore(root, before);
    }

    return {
        id,
        root,
        parts,
        contains(node) {
            return Boolean(node) && root.contains(node);
        },
        destroy() {
            root.remove();
        },
    };
}
