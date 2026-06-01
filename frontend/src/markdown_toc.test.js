import { describe, expect, it } from 'vitest';

import { updateMarkdownTableOfContentsText } from './markdown_toc';

describe('markdown_toc', () => {
    it('updates an existing ToC block even when it appears later in the document', () => {
        const source = [
            '# Project',
            '',
            'Intro paragraph.',
            '',
            'More notes here.',
            '',
            '## Alpha',
            'Text',
            '',
            '## Beta',
            'More text',
            '',
            '- [Old Heading](#old-heading)',
            '- [Stale](#stale)',
            '',
            'Closing text.',
        ].join('\n');

        const result = updateMarkdownTableOfContentsText(source);

        expect(result.updated).toBe(true);
        expect(result.text).toContain('- [Alpha](#alpha)');
        expect(result.text).toContain('- [Beta](#beta)');
        expect(result.text).not.toContain('- [Old Heading](#old-heading)');
        expect(result.text).not.toContain('- [Stale](#stale)');

        const tocStarts = result.text.match(/- \[Alpha\]\(#alpha\)/g) || [];
        expect(tocStarts).toHaveLength(1);
    });

    it('inserts a ToC after front matter title area when none exists', () => {
        const source = [
            '# Project',
            '',
            '## Alpha',
            '',
            '### Beta',
        ].join('\n');

        const result = updateMarkdownTableOfContentsText(source);

        expect(result.updated).toBe(true);
        expect(result.text).toContain('- [Alpha](#alpha)');
        expect(result.text).toContain('  - [Beta](#beta)');
    });

    it('reports no update when no headings are present', () => {
        const source = 'Plain text only\n\nNo headings';

        const result = updateMarkdownTableOfContentsText(source);

        expect(result.updated).toBe(false);
        expect(result.reason).toBe('no-headings');
        expect(result.text).toBe(source);
    });
});
