const DEFAULT_MAX_GROUPS = 500;

function cloneTransaction(tx) {
    return {
        id: tx.id,
        filePath: tx.filePath || '',
        source: tx.source || '',
        label: tx.label || '',
        beforeText: String(tx.beforeText ?? ''),
        beforeSelectionStart: Number(tx.beforeSelectionStart) || 0,
        beforeSelectionEnd: Number(tx.beforeSelectionEnd) || 0,
        afterText: String(tx.afterText ?? ''),
        afterSelectionStart: Number(tx.afterSelectionStart) || 0,
        afterSelectionEnd: Number(tx.afterSelectionEnd) || 0,
        timestamp: Number(tx.timestamp) || Date.now(),
        groupId: tx.groupId,
    };
}

function createGroup(id, label = '', source = '') {
    return {
        id,
        label: String(label || ''),
        source: String(source || ''),
        createdAt: Date.now(),
        transactions: [],
    };
}

export function createEditorUndoManager(options = {}) {
    const maxGroups = Number.isFinite(options.maxGroups)
        ? Math.max(1, Number(options.maxGroups))
        : DEFAULT_MAX_GROUPS;

    let undoGroups = [];
    let redoGroups = [];
    let activeGroup = null;
    let nextTxId = 1;
    let nextGroupId = 1;

    function trimUndoGroups() {
        if (undoGroups.length <= maxGroups) {
            return;
        }
        undoGroups = undoGroups.slice(undoGroups.length - maxGroups);
    }

    function ensureActiveGroupAttached() {
        if (!activeGroup) {
            return;
        }
        if (undoGroups.length === 0 || undoGroups[undoGroups.length - 1] !== activeGroup) {
            undoGroups.push(activeGroup);
            trimUndoGroups();
        }
    }

    function beginGroup(label = '', source = '') {
        if (activeGroup) {
            return activeGroup.id;
        }

        activeGroup = createGroup(nextGroupId++, label, source);
        return activeGroup.id;
    }

    function commitGroup(groupId) {
        if (!activeGroup) {
            return;
        }
        if (groupId != null && activeGroup.id !== groupId) {
            return;
        }
        if (activeGroup.transactions.length === 0) {
            undoGroups = undoGroups.filter((group) => group !== activeGroup);
        }
        activeGroup = null;
    }

    function record(rawTx) {
        if (!rawTx) {
            return null;
        }

        const tx = cloneTransaction({
            ...rawTx,
            id: rawTx.id ?? nextTxId++,
            timestamp: rawTx.timestamp ?? Date.now(),
        });

        let targetGroup = null;
        if (activeGroup) {
            targetGroup = activeGroup;
            if (!targetGroup.label) {
                targetGroup.label = tx.label || '';
            }
            if (!targetGroup.source) {
                targetGroup.source = tx.source || '';
            }
            tx.groupId = targetGroup.id;
            ensureActiveGroupAttached();
        } else {
            targetGroup = createGroup(nextGroupId++, tx.label, tx.source);
            tx.groupId = targetGroup.id;
            undoGroups.push(targetGroup);
            trimUndoGroups();
        }

        targetGroup.transactions.push(tx);
        // New edits invalidate redo history.
        redoGroups = [];

        return tx;
    }

    function canUndo() {
        return undoGroups.length > 0;
    }

    function canRedo() {
        return redoGroups.length > 0;
    }

    function undo(applyTransaction) {
        if (!canUndo()) {
            return false;
        }

        const group = undoGroups.pop();
        const txs = group.transactions || [];
        for (let i = txs.length - 1; i >= 0; i -= 1) {
            applyTransaction?.(txs[i], 'undo');
        }
        redoGroups.push(group);

        if (activeGroup === group) {
            activeGroup = null;
        }

        return true;
    }

    function redo(applyTransaction) {
        if (!canRedo()) {
            return false;
        }

        const group = redoGroups.pop();
        const txs = group.transactions || [];
        for (let i = 0; i < txs.length; i += 1) {
            applyTransaction?.(txs[i], 'redo');
        }
        undoGroups.push(group);
        trimUndoGroups();

        return true;
    }

    function clearForDocument(filePath) {
        const doc = String(filePath || '');
        if (!doc) {
            return;
        }

        undoGroups = undoGroups
            .map((group) => ({
                ...group,
                transactions: (group.transactions || []).filter((tx) => String(tx.filePath || '') !== doc),
            }))
            .filter((group) => group.transactions.length > 0);

        redoGroups = redoGroups
            .map((group) => ({
                ...group,
                transactions: (group.transactions || []).filter((tx) => String(tx.filePath || '') !== doc),
            }))
            .filter((group) => group.transactions.length > 0);

        if (activeGroup) {
            activeGroup.transactions = (activeGroup.transactions || []).filter((tx) => String(tx.filePath || '') !== doc);
            if (activeGroup.transactions.length === 0) {
                activeGroup = null;
            }
        }
    }

    function snapshot() {
        return {
            undoGroups: undoGroups.map((group) => ({
                ...group,
                transactions: group.transactions.map((tx) => cloneTransaction(tx)),
            })),
            redoGroups: redoGroups.map((group) => ({
                ...group,
                transactions: group.transactions.map((tx) => cloneTransaction(tx)),
            })),
            activeGroupId: activeGroup ? activeGroup.id : null,
            nextTxId,
            nextGroupId,
        };
    }

    function restore(state) {
        const src = state || {};
        undoGroups = Array.isArray(src.undoGroups)
            ? src.undoGroups.map((group) => ({
                ...group,
                transactions: Array.isArray(group.transactions) ? group.transactions.map((tx) => cloneTransaction(tx)) : [],
            }))
            : [];
        redoGroups = Array.isArray(src.redoGroups)
            ? src.redoGroups.map((group) => ({
                ...group,
                transactions: Array.isArray(group.transactions) ? group.transactions.map((tx) => cloneTransaction(tx)) : [],
            }))
            : [];

        const activeId = src.activeGroupId;
        activeGroup = undoGroups.find((group) => group.id === activeId) || null;

        nextTxId = Number(src.nextTxId) > 0 ? Number(src.nextTxId) : 1;
        nextGroupId = Number(src.nextGroupId) > 0 ? Number(src.nextGroupId) : 1;
        trimUndoGroups();
    }

    function reset() {
        undoGroups = [];
        redoGroups = [];
        activeGroup = null;
        nextTxId = 1;
        nextGroupId = 1;
    }

    return {
        beginGroup,
        commitGroup,
        record,
        undo,
        redo,
        canUndo,
        canRedo,
        clearForDocument,
        snapshot,
        restore,
        reset,
        // Debug helpers
        getUndoDepth: () => undoGroups.length,
        getRedoDepth: () => redoGroups.length,
        getActiveGroupId: () => (activeGroup ? activeGroup.id : null),
    };
}
