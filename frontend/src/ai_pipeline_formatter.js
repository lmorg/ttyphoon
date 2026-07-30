
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

const SECTION_REGEX = /^(Question|Thought|Final Answer):[ 	]*/gm;

function parseSections(source) {
    const text = String(source || '');
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
            kind: 'markdown',
            content,
        });
    }

    return sections;
}

export function createAIPipelineFormatter(container, options = {}) {
    const markedInstance = options.marked;
    const processMarkdownContainer = options.processMarkdownContainer;

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

    function startJob(content = '') {
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

        if (content) {
            const prefixEl = document.createElement('div');
            prefixEl.className = 'notes-ai-prefix markdown-body';
            container.appendChild(prefixEl);
            void (async () => {
                prefixEl.innerHTML = markedInstance ? markedInstance.parse(String(content)) : String(content);
                if (processMarkdownContainer) {
                    await processMarkdownContainer(prefixEl);
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
        for (let i = 0; i < sections.length; i += 1) {
            const section = sections[i];
            const key = sectionKeys[i];
            let el = existingEls[i];

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
