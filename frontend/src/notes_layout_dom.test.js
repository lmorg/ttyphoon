import { describe, expect, it } from 'vitest';
import { closeLayoutView, createLayout, splitLayoutView } from './notes_layout';
import { renderNotesLayout } from './notes_layout_dom';

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
        return { [el.dataset.direction]: [...el.children].map(describeDom), flex: el.style.flex };
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
        expect(describeDom(root.children[1])).toEqual({ column: ['b', 'c'], flex: '0.5 1 0px' });
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
});
