import { applySyntaxHighlighting } from './markdown-utils.js';

function normalizeChunk(value) {
    return String(value ?? '').replace(/\r\n/g, '\n').replace(/\r/g, '\n');
}

function escapeHtml(value) {
    return String(value ?? '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}

const SECTION_REGEX = /^(Question|Thought|Final Answer|Action|Action Input|Action Output):[ \t]*/gm;

const ACTION_INPUT_HEADING_REGEX = /Action Input:[ \t]*/g;
const INLINE_HEADING_REGEX = /[ \t]*(Question|Thought|Final Answer|Action|Action Input|Action Output):[ \t]*/y;

function findJsonBoundary(text, startIndex) {
    let i = startIndex;
    while (i < text.length && /\s/.test(text[i])) {
        i += 1;
    }

    if (i >= text.length) {
        return -1;
    }

    const opening = text[i];
    if (opening !== '{' && opening !== '[') {
        return -1;
    }

    let depth = 0;
    let inString = false;
    let escaped = false;

    for (let j = i; j < text.length; j += 1) {
        const ch = text[j];

        if (inString) {
            if (escaped) {
                escaped = false;
            } else if (ch === '\\') {
                escaped = true;
            } else if (ch === '"') {
                inString = false;
            }
            continue;
        }

        if (ch === '"') {
            inString = true;
            continue;
        }

        if (ch === '{' || ch === '[') {
            depth += 1;
            continue;
        }

        if (ch === '}' || ch === ']') {
            depth -= 1;
            if (depth === 0) {
                return j + 1;
            }
            if (depth < 0) {
                return -1;
            }
        }
    }

    return -1;
}

function splitInlineHeadingsAfterActionInputJson(text) {
    let source = String(text || '');
    let scanFrom = 0;

    while (scanFrom < source.length) {
        ACTION_INPUT_HEADING_REGEX.lastIndex = scanFrom;
        const headingMatch = ACTION_INPUT_HEADING_REGEX.exec(source);

        if (!headingMatch) {
            break;
        }

        const contentStart = headingMatch.index + headingMatch[0].length;
        const jsonBoundary = findJsonBoundary(source, contentStart);
        if (jsonBoundary === -1) {
            scanFrom = contentStart;
            continue;
        }

        let cursor = jsonBoundary;
        while (cursor < source.length && (source[cursor] === ' ' || source[cursor] === '\t')) {
            cursor += 1;
        }

        INLINE_HEADING_REGEX.lastIndex = cursor;
        const inlineHeadingMatch = INLINE_HEADING_REGEX.exec(source);

        if (inlineHeadingMatch && source[jsonBoundary - 1] !== '\n') {
            source = `${source.slice(0, jsonBoundary)}\n${source.slice(cursor)}`;
            scanFrom = jsonBoundary + 1;
            continue;
        }

        scanFrom = jsonBoundary;
    }

    return source;
}

function normalizeInlineHeadingBoundaries(text) {
    return splitInlineHeadingsAfterActionInputJson(text);
}

function sectionKindFromLabel(label) {
    const lower = String(label || '').toLowerCase();
    if (lower === 'action' || lower === 'action input' || lower === 'action output') {
        return 'code';
    }
    return 'markdown';
}

function parseSections(source) {
    const text = normalizeInlineHeadingBoundaries(source);
    const matches = Array.from(text.matchAll(SECTION_REGEX));

    if (matches.length === 0) {
        return [];
    }

    const sections = [];
    for (let i = 0; i < matches.length; i += 1) {
        const match = matches[i];
        const label = match[1] || '';
        const sectionStart = match.index ?? 0;
        const contentStart = sectionStart + match[0].length;
        const sectionEnd = i + 1 < matches.length ? (matches[i + 1].index ?? text.length) : text.length;

        const rawContent = text.slice(contentStart, sectionEnd);
        const content = rawContent.startsWith('\n') ? rawContent.slice(1) : rawContent;

        sections.push({
            label,
            kind: sectionKindFromLabel(label),
            content,
        });
    }

    const expanded = [];
    for (let i = 0; i < sections.length; i += 1) {
        const section = sections[i];
        if (section.label !== 'Action Input') {
            expanded.push(section);
            continue;
        }

        const split = splitActionInputContent(section.content);
        expanded.push({
            ...section,
            content: split.actionInput,
        });

        // Only synthesise a trailing Final Answer if the next real section is
        // NOT already a Final Answer — otherwise the same text would appear twice.
        const nextSection = sections[i + 1];
        const nextIsFinalAnswer = nextSection && nextSection.label === 'Final Answer';
        if (split.trailingFinalAnswer && !nextIsFinalAnswer) {
            expanded.push({
                label: 'Final Answer',
                kind: 'markdown',
                content: split.trailingFinalAnswer,
            });
        }
    }

    return expanded;
}

function normalizeActionInputContent(text) {
    const value = String(text || '');
    if (value.endsWith('}') && !value.endsWith('}\n')) {
        return `${value}\n`;
    }
    return value;
}

function splitActionInputContent(content) {
    const text = String(content || '');
    const boundary = findJsonBoundary(text, 0);
    if (boundary === -1) {
        return {
            actionInput: text,
            trailingFinalAnswer: '',
        };
    }

    const actionInput = text.slice(0, boundary);
    const trailingFinalAnswer = text.slice(boundary).trimStart();

    return {
        actionInput,
        trailingFinalAnswer,
    };
}

function renderPromptTitleHtml(title) {
    const source = String(title || '');
    const prompt = source.replace(/^\s*>\s?/, '');
    const commandMatch = prompt.match(/^(\/[\-_.a-zA-Z0-9]+ )(.*)$/s);

    if (!commandMatch) {
        return `<blockquote><p><span style="color: var(--fg);">${escapeHtml(prompt)}</span></p></blockquote>`;
    }

    return `<blockquote><p><span style="color: var(--yellow);">${escapeHtml(commandMatch[1])}</span><span style="color: var(--fg);">${escapeHtml(commandMatch[2])}</span></p></blockquote>`;
}

export function createAIPipelineFormatter(container, options = {}) {
    const markedInstance = options.marked;
    const processMarkdownContainer = options.processMarkdownContainer;
    const processCodeContainer = options.processCodeContainer || processMarkdownContainer;

    let streamText = '';
    let renderVersion = 0;
    let jobRoot = null;
    let isRendering = false;
    let needsRender = false;

    function ensureJobRoot() {
        if (!jobRoot) {
            jobRoot = document.createElement('div');
            jobRoot.className = 'notes-ai-job';
            container.appendChild(jobRoot);
        }
        return jobRoot;
    }

    function isNearBottom(element, threshold = 8) {
        if (!element) {
            return true;
        }

        const scrollHeight = Number(element.scrollHeight) || 0;
        const clientHeight = Number(element.clientHeight) || 0;
        const scrollTop = Number(element.scrollTop) || 0;

        if (scrollHeight <= 0 || clientHeight <= 0) {
            return true;
        }

        return scrollTop + clientHeight >= scrollHeight - threshold;
    }

    function clear() {
        streamText = '';
        jobRoot = null;
        isRendering = false;
        needsRender = false;
        container.textContent = '';
    }

    function startJob(title = '') {
        const hasContent = container.children.length > 0
            || container.textContent.trim().length > 0;
        if (hasContent) {
            const hr = document.createElement('hr');
            hr.className = 'notes-ai-separator';
            container.appendChild(hr);
        }
        streamText = '';
        renderVersion += 1;
        isRendering = false;
        needsRender = false;

        // Timestamp sits directly in container (before jobRoot) so the render
        // loop, which owns jobRoot's contents, cannot accidentally wipe it.
        const ts = document.createElement('p');
        ts.className = 'notes-ai-timestamp';
        const now = new Date();
        ts.textContent = now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
        container.appendChild(ts);

        if (title) {
            const titleEl = document.createElement('div');
            titleEl.className = 'notes-ai-title markdown-body';
            titleEl.textContent = title;
            container.appendChild(titleEl);
            void (async () => {
                titleEl.innerHTML = renderPromptTitleHtml(title);
                if (processMarkdownContainer) {
                    await processMarkdownContainer(titleEl);
                }
            })();
        }

        jobRoot = document.createElement('div');
        jobRoot.className = 'notes-ai-job';
        container.appendChild(jobRoot);
    }

    function finishJob() {
        container.scrollTop = container.scrollHeight;
    }

    function buildSectionShell(label) {
        const sectionEl = document.createElement('section');
        sectionEl.className = 'notes-ai-section';
        sectionEl.dataset.label = label;
        const heading = document.createElement('h2');
        heading.className = 'notes-ai-heading';
        heading.textContent = label;
        sectionEl.appendChild(heading);
        return sectionEl;
    }

    function buildCodeSection(label, content) {
        const sectionEl = buildSectionShell(label);
        const pre = document.createElement('pre');
        pre.className = 'notes-ai-code';
        const code = document.createElement('code');
        if (label === 'Action Input' || label === 'Action Output') {
            code.className = 'language-json';
        }
        code.textContent = content;
        pre.appendChild(code);
        sectionEl.appendChild(pre);
        return sectionEl;
    }

    async function patchMarkdownSection(sectionEl, content, version) {
        let markdownRoot = sectionEl.querySelector('.notes-ai-markdown');
        if (!markdownRoot) {
            markdownRoot = document.createElement('div');
            markdownRoot.className = 'notes-ai-markdown markdown-body';
            sectionEl.appendChild(markdownRoot);
        }
        markdownRoot.innerHTML = markedInstance ? markedInstance.parse(content || '') : content;
        if (processMarkdownContainer) {
            await processMarkdownContainer(markdownRoot);
        }
        return version === renderVersion;
    }

    async function patchCodeSection(sectionEl, version) {
        if (processCodeContainer) {
            await processCodeContainer(sectionEl);
        }
        await applySyntaxHighlighting(sectionEl);
        return version === renderVersion;
    }

    async function renderCurrentStream(version) {
        const root = ensureJobRoot();
        const sections = parseSections(streamText);

        // No structured sections: render the whole stream as markdown.
        if (sections.length === 0) {
            let markdownRoot = root.querySelector('.notes-ai-markdown');
            if (!markdownRoot || root.childElementCount !== 1 || root.firstElementChild !== markdownRoot) {
                root.textContent = '';
                markdownRoot = document.createElement('div');
                markdownRoot.className = 'notes-ai-markdown markdown-body';
                root.appendChild(markdownRoot);
            }

            if (!streamText) {
                markdownRoot.innerHTML = '';
                return;
            }

            markdownRoot.innerHTML = markedInstance ? markedInstance.parse(streamText) : streamText;
            if (processMarkdownContainer) {
                await processMarkdownContainer(markdownRoot);
                if (version !== renderVersion) {
                    return;
                }
            }

            return;
        }

        // Remove any stale leading raw text node if sections appeared.
        if (root.firstChild && root.firstChild.nodeType === Node.TEXT_NODE) {
            root.removeChild(root.firstChild);
        }

        // Remove fallback markdown root when transitioning into structured sections.
        const topLevelMarkdown = Array.from(root.children).filter((el) => el.classList.contains('notes-ai-markdown'));
        for (const el of topLevelMarkdown) {
            root.removeChild(el);
        }

        // Build a key for each section so duplicates (eg two Actions) are distinct.
        const labelCounts = new Map();
        const sectionKeys = sections.map((s) => {
            const n = (labelCounts.get(s.label) || 0) + 1;
            labelCounts.set(s.label, n);
            return `${s.label}:${n}`;
        });

        const existingEls = Array.from(root.querySelectorAll('.notes-ai-section'));

        // ── Trim excess elements that no longer correspond to a section ──────────
        for (let i = sections.length; i < existingEls.length; i += 1) {
            root.removeChild(existingEls[i]);
        }

        // ── Patch / append each section ──────────────────────────────────────────
        const codeCounts = new Map();

        for (let i = 0; i < sections.length; i += 1) {
            const section = sections[i];
            const key = sectionKeys[i];
            let el = existingEls[i];

            if (section.kind === 'code') {
                const n = (codeCounts.get(section.label) || 0) + 1;
                codeCounts.set(section.label, n);

                const content = (section.label === 'Action Input' || section.label === 'Action Output')
                    ? normalizeActionInputContent(section.content)
                    : String(section.content || '');

                if (el && el.dataset.label === section.label && el.dataset.key === key) {
                    // Same slot, same section type — only patch changed text.
                    const code = el.querySelector('code');
                    if (code && code.textContent !== content) {
                        const pre = el.querySelector('.notes-ai-code');
                        const sticky = isNearBottom(pre);
                        code.textContent = content;
                        const ok = await patchCodeSection(el, version);
                        if (!ok) {
                            return;
                        }
                        if (sticky) {
                            pre.scrollTop = pre.scrollHeight;
                        }
                    }
                } else {
                    // New section or label mismatch — build fresh.
                    const newEl = buildCodeSection(section.label, content);
                    newEl.dataset.key = key;
                    if (el) {
                        root.replaceChild(newEl, el);
                        existingEls[i] = newEl;
                    } else {
                        root.appendChild(newEl);
                        existingEls.push(newEl);
                    }
                    const ok = await patchCodeSection(newEl, version);
                    if (!ok) {
                        return;
                    }
                    // New code block always starts scrolled to bottom.
                    const pre = newEl.querySelector('.notes-ai-code');
                    if (pre) {
                        pre.scrollTop = pre.scrollHeight;
                    }
                }
            } else {
                const rawContent = section.content || '';

                if (el && el.dataset.label === section.label && el.dataset.key === key) {
                    // Same slot — only re-render if content changed.
                    if (el.dataset.rawContent !== rawContent) {
                        el.dataset.rawContent = rawContent;
                        const ok = await patchMarkdownSection(el, rawContent, version);
                        if (!ok) {
                            return;
                        }
                    }
                } else {
                    // New section or label mismatch — build fresh.
                    const newEl = buildSectionShell(section.label);
                    newEl.dataset.key = key;
                    newEl.dataset.rawContent = rawContent;
                    if (el) {
                        root.replaceChild(newEl, el);
                        existingEls[i] = newEl;
                    } else {
                        root.appendChild(newEl);
                        existingEls.push(newEl);
                    }
                    const ok = await patchMarkdownSection(newEl, rawContent, version);
                    if (!ok) {
                        return;
                    }
                }
            }
        }

        container.scrollTop = container.scrollHeight;
    }

    function appendChunk(chunk) {
        const text = normalizeChunk(chunk);
        if (!text) {
            return;
        }

        streamText += text;
        renderVersion += 1;

        if (isRendering) {
            needsRender = true;
            return;
        }

        void (async () => {
            isRendering = true;
            try {
                do {
                    needsRender = false;
                    await renderCurrentStream(renderVersion);
                } while (needsRender);
            } finally {
                isRendering = false;
            }
        })();
    }

    function setText(text) {
        streamText = String(text || '');
        renderVersion += 1;

        if (isRendering) {
            needsRender = true;
            return;
        }

        void (async () => {
            isRendering = true;
            try {
                do {
                    needsRender = false;
                    await renderCurrentStream(renderVersion);
                } while (needsRender);
            } finally {
                isRendering = false;
            }
        })();
    }

    return {
        clear,
        startJob,
        finishJob,
        appendChunk,
        setText,
    };
}
