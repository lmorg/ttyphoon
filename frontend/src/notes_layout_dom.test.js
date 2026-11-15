import { describe, expect, it } from 'vitest';
import { closeLayoutView, createLayout, splitLayoutView } from './notes_layout';
import { bindNotesLayoutDividers, renderNotesLayout } from './notes_layout_dom';

function setup(ids) {
    const host = document.createElement('div');
    document.body.appendChild(host);
    const roots = new Map(ids.map((id) => {
        const el = document.createElement('section');
        el.className = 'notes-view';
        el.dataset.viewId = id;
        return [id, el];
    }));
    return { host, roots, getRoot: (id) => roots.get(id) };
}

function describeDom(el) {
    if (el.classList.contains('notes-layout-split')) {
        const children = [...el.children].filter((child) => !child.classList.contains('notes-layout-divider'));
        return { [el.dataset.direction]: children.map(describeDom), flex: el.style.flex };
    }
    return el.dataset.viewId;
}

describe('renderNotesLayout', () => {
    it('renders a single view directly into the host', () => {
        const { host, roots, getRoot } = setup(['a']);
        renderNotesLayout(host, createLayout('a'), getRoot);
        expect([...host.children]).toEqual([roots.get('a')]);
        expect(roots.get('a').style.flex).toBe('');
    });

    it('nests splits, sizes children, and leaves unchanged views in place', () => {
        const { host, roots, getRoot } = setup(['a', 'b', 'c']);
        let tree = splitLayoutView(createLayout('a'), 'a', 'b', 'row');
        tree = splitLayoutView(tree, 'b', 'c', 'column');
        renderNotesLayout(host, tree, getRoot);

        const root = host.firstElementChild;
        expect(host.children).toHaveLength(1);
        expect(describeDom(root).row[0]).toBe('a');
        expect(describeDom(root.children[2])).toEqual({ column: ['b', 'c'], flex: '0.5 1 0px' });
        expect(roots.get('a').style.flex).toBe('0.5 1 0px');

        const mutations = [];
        const observer = new MutationObserver((records) => mutations.push(...records));
        observer.observe(host, { childList: true, subtree: true });
        renderNotesLayout(host, tree, getRoot);
        observer.takeRecords().forEach((record) => mutations.push(record));
        expect(mutations.filter((record) => record.type === 'childList')).toEqual([]);

        renderNotesLayout(host, closeLayoutView(tree, 'a'), getRoot);
        observer.disconnect();
        expect(host.firstElementChild).toBe(root);
        expect(describeDom(root)).toEqual({ column: ['b', 'c'], flex: '' });
        expect(roots.get('a').isConnected).toBe(false);

        renderNotesLayout(host, null, getRoot);
        expect(host.children).toHaveLength(0);
    });

    it('drags a divider within the minimum size and equalises it on double-click', () => {
        const { host, roots, getRoot } = setup(['a', 'b', 'c']);
        let tree = splitLayoutView(createLayout('a'), 'a', 'b', 'row');
        tree = splitLayoutView(tree, 'b', 'c', 'column');
        const setLayout = (next) => {
            tree = next;
            renderNotesLayout(host, tree, getRoot);
        };
        setLayout(tree);
        bindNotesLayoutDividers(host, { getLayout: () => tree, setLayout });
        roots.get('a').getBoundingClientRect = () => ({ width: 500, height: 400 });
        host.firstElementChild.children[2].getBoundingClientRect = () => ({ width: 500, height: 400 });

        const [rowDivider, columnDivider] = host.querySelectorAll('.notes-layout-divider');
        expect([rowDivider.dataset.path, rowDivider.dataset.index]).toEqual(['', '0']);
        expect([columnDivider.dataset.path, columnDivider.dataset.index]).toEqual(['1', '0']);
        const pointer = (type, target, clientX) => target.dispatchEvent(
            new MouseEvent(type, { bubbles: true, cancelable: true, button: 0, clientX }),
        );

        pointer('pointerdown', rowDivider, 500);
        pointer('pointermove', rowDivider, 600);
        expect(tree.sizes).toEqual([0.6, 0.4]);
        expect(roots.get('a').style.flex).toBe('0.6 1 0px');
        // Each move is measured from where the drag began, and stops at 120px.
        pointer('pointermove', rowDivider, 1000);
        expect(tree.sizes).toEqual([0.88, 0.12]);
        pointer('pointerup', rowDivider, 1000);
        pointer('pointermove', rowDivider, 0);
        expect(tree.sizes).toEqual([0.88, 0.12]);

        rowDivider.dispatchEvent(new MouseEvent('dblclick', { bubbles: true }));
        expect(tree.sizes).toEqual([0.5, 0.5]);
        expect(host.querySelectorAll('.notes-layout-divider')[0]).toBe(rowDivider);
    });
});
