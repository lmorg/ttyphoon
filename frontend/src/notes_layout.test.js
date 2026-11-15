import { describe, expect, it } from 'vitest';
import {
    closeLayoutView,
    createLayout,
    equaliseLayoutDivider,
    layoutViewIds,
    neighbourLayoutView,
    parseLayout,
    resizeLayoutDivider,
    serialiseLayout,
    splitLayoutView,
} from './notes_layout';

const view = (id) => ({ type: 'view', id });

describe('notes layout tree', () => {
    it('adds a sibling to a split of the same direction and nests a split of the other direction', () => {
        let tree = splitLayoutView(createLayout('a'), 'a', 'b', 'row');
        tree = splitLayoutView(tree, 'b', 'c', 'row');
        expect(tree).toEqual({ type: 'split', direction: 'row', children: [view('a'), view('b'), view('c')], sizes: [0.5, 0.25, 0.25] });

        const before = tree;
        tree = splitLayoutView(tree, 'b', 'd', 'column', { before: true });
        expect(before.children[1]).toEqual(view('b'));
        expect(tree.children[1]).toEqual({ type: 'split', direction: 'column', children: [view('d'), view('b')], sizes: [0.5, 0.5] });
        expect(tree.sizes).toEqual([0.5, 0.25, 0.25]);
        expect(layoutViewIds(tree)).toEqual(['a', 'd', 'b', 'c']);

        expect(() => splitLayoutView(tree, 'a', 'c', 'row')).toThrow();
        expect(() => splitLayoutView(tree, 'x', 'y', 'row')).toThrow();
        expect(() => splitLayoutView(tree, 'a', 'y', 'diagonal')).toThrow();
    });

    it('gives a closed view space to its neighbours and collapses splits left with one child', () => {
        let tree = splitLayoutView(createLayout('a'), 'a', 'b', 'row');
        tree = splitLayoutView(tree, 'b', 'c', 'column');
        tree = splitLayoutView(tree, 'c', 'd', 'row');
        // row[a, column[b, row[c, d]]]: closing b merges row[c, d] into the root row.
        tree = closeLayoutView(tree, 'b');
        expect(tree).toEqual({ type: 'split', direction: 'row', children: [view('a'), view('c'), view('d')], sizes: [0.5, 0.25, 0.25] });

        tree = closeLayoutView(tree, 'a');
        expect(tree.sizes).toEqual([0.5, 0.5]);
        tree = closeLayoutView(tree, 'c');
        expect(tree).toEqual(view('d'));
        expect(closeLayoutView(tree, 'd')).toEqual(view('d'));
        expect(closeLayoutView(tree, 'missing')).toBe(tree);
    });

    it('resizes and equalises dividers without shrinking a view below the minimum', () => {
        let tree = splitLayoutView(createLayout('a'), 'a', 'b', 'column');
        tree = splitLayoutView(tree, 'b', 'c', 'row');
        // column[a, row[b, c]]: the row is at path [1].
        tree = resizeLayoutDivider(tree, [1], 0, 0.9, { min: 0.1 });
        expect(tree.children[1].sizes).toEqual([0.9, expect.closeTo(0.1)]);
        tree = resizeLayoutDivider(tree, [], 0, -0.2);
        expect(tree.sizes[0]).toBeCloseTo(0.3);
        expect(resizeLayoutDivider(tree, [], 1, 0.1)).toEqual(tree);

        tree = equaliseLayoutDivider(tree, [1], 0);
        expect(tree.children[1].sizes).toEqual([0.5, 0.5]);
    });

    it('round-trips through JSON and repairs saved layouts', () => {
        let tree = splitLayoutView(createLayout('a'), 'a', 'b', 'row');
        tree = splitLayoutView(tree, 'b', 'c', 'column');
        const saved = JSON.parse(JSON.stringify(serialiseLayout(tree)));
        expect(parseLayout(saved)).toEqual(tree);

        expect(parseLayout({
            type: 'split',
            direction: 'row',
            sizes: [1, 3, 'x'],
            children: [view('a'), view('a'), { type: 'split', direction: 'row', children: [view('b'), { type: 'pane' }] }],
        })).toEqual({ type: 'split', direction: 'row', children: [view('a'), view('b')], sizes: [0.5, 0.5] });
        expect(parseLayout({ type: 'split', direction: 'sideways', children: [view('a')] })).toBeNull();
        expect(parseLayout(null)).toBeNull();

        const withTabs = { type: 'view', id: 'a', tabs: ['$NOTES/a.md'], extra: true };
        expect(parseLayout(withTabs, { parseView: ({ id, tabs }) => ({ type: 'view', id, tabs }) }))
            .toEqual({ type: 'view', id: 'a', tabs: ['$NOTES/a.md'] });
        expect(parseLayout(withTabs, { parseView: () => null })).toBeNull();
    });

    it('finds the neighbouring view on each side across nested splits', () => {
        // a | (b over (c | d))
        let tree = createLayout('a');
        tree = splitLayoutView(tree, 'a', 'b', 'row');
        tree = splitLayoutView(tree, 'b', 'c', 'column');
        tree = splitLayoutView(tree, 'c', 'd', 'row');

        expect(neighbourLayoutView(tree, 'a', 'right')).toBe('b');
        expect(neighbourLayoutView(tree, 'a', 'left')).toBeNull();
        expect(neighbourLayoutView(tree, 'b', 'down')).toBe('c');
        expect(neighbourLayoutView(tree, 'd', 'up')).toBe('b');
        expect(neighbourLayoutView(tree, 'd', 'left')).toBe('c');
        expect(neighbourLayoutView(tree, 'c', 'left')).toBe('a');
        expect(neighbourLayoutView(tree, 'd', 'right')).toBeNull();
        expect(neighbourLayoutView(tree, 'missing', 'up')).toBeNull();
    });
});
