function normalizeMarkdownHeadingForAnchor(text) {
    return String(text || '')
        .trim()
        .toLowerCase()
        .replace(/[\u2018\u2019]/g, '')
        .replace(/[^a-z0-9\s-]/g, '')
        .replace(/\s+/g, '-')
        .replace(/-+/g, '-')
        .replace(/^-|-$/g, '');
}

function parseMarkdownHeadingsForToc(markdown) {
    const lines = String(markdown || '').split('\n');
    const headings = [];
    let inFence = false;

    for (const line of lines) {
        const trimmed = String(line || '').trim();

        if (/^(```|~~~)/.test(trimmed)) {
            inFence = !inFence;
            continue;
        }

        if (inFence) {
            continue;
        }

        const mdMatch = trimmed.match(/^(#{2,6})\s+(.*?)\s*#*\s*$/);
        if (mdMatch) {
            headings.push({
                level: mdMatch[1].length,
                text: mdMatch[2].trim(),
            });
            continue;
        }

        const htmlMatch = trimmed.match(/^<h([2-6])\b[^>]*>(.*?)<\/h\1>$/i);
        if (htmlMatch) {
            const inner = htmlMatch[2]
                .replace(/<[^>]+>/g, '')
                .trim();
            if (inner) {
                headings.push({
                    level: Number.parseInt(htmlMatch[1], 10),
                    text: inner,
                });
            }
        }
    }

    return headings;
}

function isTocEntryLine(line) {
    return /^\s*-\s+\[[^\]]+\]\(#[^)]+\)\s*$/.test(String(line || ''));
}

function findTocInsertionIndex(lines) {
    let i = 0;

    while (i < lines.length && String(lines[i] || '').trim() === '') {
        i += 1;
    }

    while (i < lines.length) {
        const line = String(lines[i] || '').trim();

        if (line === '') {
            i += 1;
            continue;
        }

        if (/^#\s+/.test(line)) {
            i += 1;
            continue;
        }

        if (/^<h1\b/i.test(line)) {
            i += 1;
            continue;
        }

        break;
    }

    return i;
}

function findExistingTocRange(lines, startIndex) {
    let i = Math.max(0, startIndex);

    while (i < lines.length) {
        if (!isTocEntryLine(lines[i])) {
            i += 1;
            continue;
        }

        const firstEntry = i;
        let entryCount = 0;
        let end = firstEntry;

        while (end < lines.length) {
            const trimmed = String(lines[end] || '').trim();
            if (trimmed === '') {
                end += 1;
                continue;
            }

            if (isTocEntryLine(lines[end])) {
                entryCount += 1;
                end += 1;
                continue;
            }

            break;
        }

        while (end > firstEntry && String(lines[end - 1] || '').trim() === '') {
            end -= 1;
        }

        if (entryCount >= 1) {
            return { start: firstEntry, end };
        }

        i = end + 1;
    }

    return null;
}

function buildTocLinesFromHeadings(headings) {
    if (!Array.isArray(headings) || headings.length === 0) {
        return [];
    }

    const baseLevel = headings[0].level;
    const slugCounts = new Map();

    return headings.map((heading) => {
        const indentDepth = Math.max(0, heading.level - baseLevel);
        const indent = '  '.repeat(indentDepth);
        const baseSlug = normalizeMarkdownHeadingForAnchor(heading.text);
        if (!baseSlug) {
            return null;
        }

        const count = slugCounts.get(baseSlug) || 0;
        slugCounts.set(baseSlug, count + 1);
        const slug = count === 0 ? baseSlug : `${baseSlug}-${count}`;

        return `${indent}- [${heading.text}](#${slug})`;
    }).filter(Boolean);
}

export function updateMarkdownTableOfContentsText(source) {
    const markdown = String(source || '');
    const headings = parseMarkdownHeadingsForToc(markdown);
    if (headings.length === 0) {
        return {
            updated: false,
            reason: 'no-headings',
            text: markdown,
        };
    }

    const tocLines = buildTocLinesFromHeadings(headings);
    if (tocLines.length === 0) {
        return {
            updated: false,
            reason: 'no-headings',
            text: markdown,
        };
    }

    const lines = markdown.split('\n');
    const insertionIndex = findTocInsertionIndex(lines);
    const existingRange = findExistingTocRange(lines, insertionIndex);

    if (existingRange) {
        lines.splice(existingRange.start, existingRange.end - existingRange.start, ...tocLines);
    } else {
        const before = lines.slice(0, insertionIndex);
        const after = lines.slice(insertionIndex);
        const nextLines = [];

        nextLines.push(...before);
        if (nextLines.length > 0 && String(nextLines[nextLines.length - 1] || '').trim() !== '') {
            nextLines.push('');
        }

        nextLines.push(...tocLines);
        nextLines.push('');

        if (after.length > 0 && String(after[0] || '').trim() !== '') {
            nextLines.push('');
        }

        nextLines.push(...after);
        lines.splice(0, lines.length, ...nextLines);
    }

    return {
        updated: true,
        reason: 'updated',
        text: lines.join('\n').replace(/\n{3,}/g, '\n\n'),
    };
}
