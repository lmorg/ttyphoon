import { describe, expect, it, vi } from 'vitest';

import { createEditorUndoManager } from './editor_undo_manager.js';

describe('editor undo manager', () => {
    it('records singleton transactions and supports undo/redo', () => {
        const manager = createEditorUndoManager();
        const apply = vi.fn();

        manager.record({
            filePath: '$NOTES/test.md',
            source: 'test',
            label: 'A',
            beforeText: 'a',
            beforeSelectionStart: 1,
            beforeSelectionEnd: 1,
            afterText: 'ab',
            afterSelectionStart: 2,
            afterSelectionEnd: 2,
        });

        expect(manager.getUndoDepth()).toBe(1);
        expect(manager.canUndo()).toBe(true);

        expect(manager.undo(apply)).toBe(true);
        expect(apply).toHaveBeenCalledTimes(1);
        expect(apply.mock.calls[0][1]).toBe('undo');
        expect(manager.getUndoDepth()).toBe(0);
        expect(manager.getRedoDepth()).toBe(1);

        expect(manager.redo(apply)).toBe(true);
        expect(apply).toHaveBeenCalledTimes(2);
        expect(apply.mock.calls[1][1]).toBe('redo');
        expect(manager.getUndoDepth()).toBe(1);
        expect(manager.getRedoDepth()).toBe(0);
    });

    it('groups transactions between beginGroup and commitGroup', () => {
        const manager = createEditorUndoManager();
        const apply = vi.fn();

        const groupId = manager.beginGroup('Typing', 'typing');
        manager.record({
            filePath: '$NOTES/test.md',
            source: 'typing',
            label: 'insert-1',
            beforeText: 'a',
            beforeSelectionStart: 1,
            beforeSelectionEnd: 1,
            afterText: 'ab',
            afterSelectionStart: 2,
            afterSelectionEnd: 2,
        });
        manager.record({
            filePath: '$NOTES/test.md',
            source: 'typing',
            label: 'insert-2',
            beforeText: 'ab',
            beforeSelectionStart: 2,
            beforeSelectionEnd: 2,
            afterText: 'abc',
            afterSelectionStart: 3,
            afterSelectionEnd: 3,
        });
        manager.commitGroup(groupId);

        expect(manager.getUndoDepth()).toBe(1);
        expect(manager.undo(apply)).toBe(true);

        // Undo applies grouped transactions in reverse order.
        expect(apply).toHaveBeenCalledTimes(2);
        expect(apply.mock.calls[0][0].label).toBe('insert-2');
        expect(apply.mock.calls[1][0].label).toBe('insert-1');
    });

    it('clears redo stack when a new transaction is recorded after undo', () => {
        const manager = createEditorUndoManager();

        manager.record({
            filePath: '$NOTES/test.md',
            source: 'test',
            label: 'A',
            beforeText: 'a',
            beforeSelectionStart: 1,
            beforeSelectionEnd: 1,
            afterText: 'ab',
            afterSelectionStart: 2,
            afterSelectionEnd: 2,
        });

        manager.undo(() => {});
        expect(manager.getRedoDepth()).toBe(1);

        manager.record({
            filePath: '$NOTES/test.md',
            source: 'test',
            label: 'B',
            beforeText: 'ab',
            beforeSelectionStart: 2,
            beforeSelectionEnd: 2,
            afterText: 'abc',
            afterSelectionStart: 3,
            afterSelectionEnd: 3,
        });

        expect(manager.getRedoDepth()).toBe(0);
        expect(manager.getUndoDepth()).toBe(1);
    });
});
