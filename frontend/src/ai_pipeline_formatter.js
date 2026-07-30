
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
    const lazyChunkSize = Math.max(8, Number(options.lazyChunkSize) || 24);

    let streamText = '';
    let renderVersion = 0;
    let jobRoot = null;
    let isRendering = false;
    let needsRender = false;
    let lazyChunkObserver = null;

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
        destroyLazyObserver();
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
        // setText is used for one-shot full-document renders (eg session restore).
        // The content is already complete, so render it as a single markdown
        // block rather than running it through parseSections — which would split
        // a large historical log into N sections and call processMarkdownContainer
        // N times (one full hljs + link-scan pass per section), locking the UI.
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
                    await renderTextAsMarkdown(renderVersion);
                } while (needsRender);
            } finally {
                isRendering = false;
            }
        })();
    }

    function destroyLazyObserver() {
        if (lazyChunkObserver) {
            lazyChunkObserver.disconnect();
            lazyChunkObserver = null;
        }
    }

    function buildLazyChunks(markdownRoot) {
        const children = Array.from(markdownRoot.children);
        if (children.length === 0) {
            return [];
        }

        const fragments = [];
        for (let i = 0; i < children.length; i += lazyChunkSize) {
            fragments.push(children.slice(i, i + lazyChunkSize));
        }

        markdownRoot.textContent = '';

        const chunks = [];
        for (const nodes of fragments) {
            const chunk = document.createElement('section');
            chunk.className = 'notes-ai-lazy-chunk';
            chunk.dataset.lazyProcessed = '';

            const content = document.createElement('div');
            content.className = 'notes-ai-lazy-chunk-content';
            for (const node of nodes) {
                content.appendChild(node);
            }

            const spinner = document.createElement('div');
            spinner.className = 'notes-ai-lazy-spinner notes-ai-lazy-spinner-chunk';

            chunk.appendChild(content);
            chunk.appendChild(spinner);
            markdownRoot.appendChild(chunk);
            chunks.push(chunk);
        }

        return chunks;
    }

    function primeLazyChunks(chunks, version) {
        if (!Array.isArray(chunks) || chunks.length === 0) {
            return;
        }

        const viewportTop = Number(container.scrollTop) || 0;
        const viewportBottom = viewportTop + (Number(container.clientHeight) || 0);
        const prewarmTop = Math.max(0, viewportTop - ((Number(container.clientHeight) || 0) * 2));
        const prewarmBottom = viewportBottom + ((Number(container.clientHeight) || 0) * 2);

        let matched = 0;
        for (const chunk of chunks) {
            const chunkTop = chunk.offsetTop;
            const chunkBottom = chunkTop + chunk.offsetHeight;
            if (chunkBottom < prewarmTop || chunkTop > prewarmBottom) {
                continue;
            }

            matched += 1;
            void processLazyChunk(chunk, version);
        }

        if (matched > 0) {
            return;
        }

        const fromBottom = isNearBottom(container);
        const seed = fromBottom ? chunks.slice(-3) : chunks.slice(0, 3);
        for (const chunk of seed) {
            void processLazyChunk(chunk, version);
        }
    }

    async function renderTextAsMarkdown(version) {
        destroyLazyObserver();

        const root = ensureJobRoot();
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

        // Show a loading spinner while marked.parse() runs (synchronous and
        // can block for large documents).  We yield to the browser after
        // inserting the spinner so it actually paints before we block.
        markdownRoot.innerHTML = '';
        const loadingSpinner = document.createElement('div');
        loadingSpinner.className = 'notes-ai-lazy-spinner notes-ai-lazy-spinner-page';
        markdownRoot.appendChild(loadingSpinner);

        // Yield so the browser paints the spinner before the blocking parse
        await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));
        if (version !== renderVersion) return;

        // Parse markdown into HTML and insert into DOM.
        markdownRoot.innerHTML = markedInstance ? markedInstance.parse(streamText) : streamText;

        // Lazy-process elements as they scroll into view using IntersectionObserver.
        // This avoids running expensive operations (syntax highlighting, mermaid,
        // autoHyperlink, table setup, image processing) on the entire document at
        // once, which would lock the UI for large session logs.
        if (!processMarkdownContainer) {
            return;
        }

        const scrollRoot = container; // the scroll parent (#notes-ai-output)

        const chunks = buildLazyChunks(markdownRoot);

        lazyChunkObserver = new IntersectionObserver((entries) => {
            for (const entry of entries) {
                if (!entry.isIntersecting) {
                    continue;
                }

                const chunk = entry.target;
                lazyChunkObserver.unobserve(chunk);
                void processLazyChunk(chunk, version);
            }
        }, {
            root: scrollRoot,
            rootMargin: '1200px 0px',
        });

        for (const chunk of chunks) {
            lazyChunkObserver.observe(chunk);
        }

        primeLazyChunks(chunks, version);
    }

    async function processLazyChunk(chunk, version) {
        if (!chunk || chunk.dataset.lazyProcessed === 'done' || chunk.dataset.lazyProcessed === 'pending') {
            return;
        }

        chunk.dataset.lazyProcessed = 'pending';

        const chunkContent = chunk.querySelector('.notes-ai-lazy-chunk-content');
        if (!chunkContent) {
            chunk.dataset.lazyProcessed = 'done';
            return;
        }

        try {
            await processMarkdownContainer(chunkContent);
            if (version !== renderVersion) {
                return;
            }

            const codeBlocks = chunkContent.querySelectorAll('pre');
            for (const block of codeBlocks) {
                block.scrollTop = block.scrollHeight;
            }
        } finally {
            chunk.dataset.lazyProcessed = 'done';
        }
    }

    return {
        clear,
        startJob,
        finishJob,
        appendChunk,
        setText,
    };
}
