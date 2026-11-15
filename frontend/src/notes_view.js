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

    // What this view last showed for each document, keyed by document key:
    // view mode and editor view state (cursor and scroll). A view keeps its
    // own, so two views of one document can differ (ADR 0100 question 15).
    const documentStates = new Map();

    return {
        id,
        root,
        parts,
        // The view's Monaco adapter, created lazily into parts.monacoEditor.
        editor: null,
        editorInit: null,
        // Key of the document whose content the view's editor currently holds.
        shownDocumentKey: '',
        // Keys of the documents open as tabs in this view, in tab order
        // (ADR 0100 question 16).
        tabs: [],
        // Adds a tab after afterKey's tab, or at the end.
        addTab(key, afterKey = '') {
            if (!key || this.tabs.includes(key)) {
                return;
            }
            const index = afterKey ? this.tabs.indexOf(afterKey) : -1;
            if (index < 0) {
                this.tabs.push(key);
            } else {
                this.tabs.splice(index + 1, 0, key);
            }
        },
        // Returns the removed tab's index, or -1.
        removeTab(key) {
            const index = this.tabs.indexOf(key);
            if (index >= 0) {
                this.tabs.splice(index, 1);
            }
            return index;
        },
        // A document moved to a new key, as when its file is renamed.
        renameDocument(oldKey, newKey) {
            if (!oldKey || !newKey || oldKey === newKey) {
                return;
            }
            if (this.tabs.includes(oldKey)) {
                this.removeTab(newKey);
                this.tabs[this.tabs.indexOf(oldKey)] = newKey;
            }
            if (documentStates.has(oldKey)) {
                documentStates.set(newKey, documentStates.get(oldKey));
                documentStates.delete(oldKey);
            }
            if (this.shownDocumentKey === oldKey) {
                this.shownDocumentKey = newKey;
            }
        },
        contains(node) {
            return Boolean(node) && root.contains(node);
        },
        rememberDocument(key, data) {
            if (key) {
                documentStates.set(key, { ...documentStates.get(key), ...data });
            }
        },
        recallDocument(key) {
            return documentStates.get(key) || null;
        },
        destroy() {
            this.editor?.dispose?.();
            this.editor = null;
            documentStates.clear();
            this.tabs = [];
            root.remove();
        },
    };
}
