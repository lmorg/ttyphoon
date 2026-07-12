import { describe, expect, it, vi } from 'vitest';

import { createEditorUndoManager } from './editor_undo_manager.js';
import { createEditorMutationAdapter } from './editor_mutation_adapter.js';

describe('editor mutation adapter', () => {
    it('replaceRange mutates textarea, emits input, and records transaction', () => {
        const manager = createEditorUndoManager();
        const adapter = createEditorMutationAdapter({ manager, getFilePath: () => '$NOTES/test.md' });

        const textarea = document.createElement('textarea');
        textarea.value = 'hello world';
        textarea.selectionStart = 6;
        textarea.selectionEnd = 11;

        const inputSpy = vi.fn();
        textarea.addEventListener('input', inputSpy);

        const tx = adapter.replaceRange(textarea, {
            text: 'ttyphoon',
            source: 'unit-test',
            label: 'replace word',
        });

        expect(textarea.value).toBe('hello ttyphoon');
        expect(textarea.selectionStart).toBe(14);
        expect(textarea.selectionEnd).toBe(14);
        expect(inputSpy).toHaveBeenCalledTimes(1);

        expect(tx).toBeTruthy();
        expect(tx.filePath).toBe('$NOTES/test.md');
        expect(tx.beforeText).toBe('hello world');
        expect(tx.afterText).toBe('hello ttyphoon');
    });

    it('replaceDocumentText records undo/redo snapshots and applySnapshot restores content', () => {
        const manager = createEditorUndoManager();
        const adapter = createEditorMutationAdapter({ manager, getFilePath: () => '$NOTES/test.md' });

        const textarea = document.createElement('textarea');
        textarea.value = 'abc';
        textarea.selectionStart = 3;
        textarea.selectionEnd = 3;

        adapter.replaceDocumentText(textarea, {
            text: 'xyz',
            selectionStart: 1,
            selectionEnd: 1,
            source: 'unit-test',
            label: 'replace document',
        });

        expect(textarea.value).toBe('xyz');
        expect(textarea.selectionStart).toBe(1);

        manager.undo((tx, direction) => {
            adapter.applySnapshot(textarea, tx, direction, true);
        });
        expect(textarea.value).toBe('abc');
        expect(textarea.selectionStart).toBe(3);

        manager.redo((tx, direction) => {
            adapter.applySnapshot(textarea, tx, direction, true);
        });
        expect(textarea.value).toBe('xyz');
        expect(textarea.selectionStart).toBe(1);
    });
});
