// Open Notes documents (ADR 0100 decision 4). A NotesDocument holds the state
// that belongs to a file rather than to the view showing it, and is shared by
// every view of that file.

import { createDocumentBuffer } from './notes_document_buffer';

const PROJECT_PREFIX = '$PROJECT';

// Backend calls take Notes paths such as `$NOTES/a.md`. Those are unique except
// for `$PROJECT/...`, which means a different file in each project, so the key
// replaces that prefix with the project root.
export function notesDocumentKey(path, project = '') {
    const notesPath = String(path || '');
    const root = String(project || '').replace(/\/+$/, '');
    if (root && (notesPath === PROJECT_PREFIX || notesPath.startsWith(`${PROJECT_PREFIX}/`))) {
        return root + notesPath.slice(PROJECT_PREFIX.length);
    }
    return notesPath;
}

export function createNotesDocument({ path, project = '', uri = '', fileType = 'markdown', buffer } = {}) {
    return {
        key: notesDocumentKey(path, project),
        path: String(path || ''),
        project: String(project || ''),
        uri: String(uri || ''),
        fileType,
        buffer: buffer || createDocumentBuffer(),
        dirty: false,
        autosaveTimer: null,
        lspOpenFile: '',
        lspChangeTimer: null,
        typosOpenFile: '',
        typosChangeTimer: null,
        jupyterCodeBlocks: {},
        jupyterBlockCounter: 0,
        // Block IDs with a run in progress on the backend.
        runningJupyterBlocks: new Set(),
        // Find in the document (ADR 0100 question 10).
        findQuery: '',
        findMatches: [],
        findCurrentIndex: -1,
        findDocLastExecutedSignature: '',
    };
}

const DOCUMENT_TIMERS = ['autosaveTimer', 'lspChangeTimer', 'typosChangeTimer'];

// onClose runs after a document leaves the registry, to release what it holds
// elsewhere (LSP registrations, kernels). Its result is returned by close().
export function createNotesDocumentRegistry({ onClose } = {}) {
    const documents = new Map();

    function get(path, project = '') {
        return documents.get(notesDocumentKey(path, project)) || null;
    }

    // Opening a document that is already open returns the existing record, so
    // every view of a file shares one document. Fields only apply to new ones.
    function open(path, fields = {}) {
        const existing = get(path, fields.project);
        if (existing) {
            return existing;
        }
        const doc = createNotesDocument({ ...fields, path });
        if (!doc.key) {
            throw new Error('NotesDocument needs a path');
        }
        documents.set(doc.key, doc);
        return doc;
    }

    function close(docOrPath, project = '') {
        const doc = typeof docOrPath === 'object' && docOrPath
            ? docOrPath
            : get(docOrPath, project);
        if (!doc || documents.get(doc.key) !== doc) {
            return undefined;
        }
        documents.delete(doc.key);
        for (const timer of DOCUMENT_TIMERS) {
            if (doc[timer]) {
                clearTimeout(doc[timer]);
                doc[timer] = null;
            }
        }
        return onClose ? onClose(doc) : undefined;
    }

    return {
        open,
        get,
        close,
        has: (path, project = '') => Boolean(get(path, project)),
        list: () => [...documents.values()],
        get size() {
            return documents.size;
        },
    };
}
