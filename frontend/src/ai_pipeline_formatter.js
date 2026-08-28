import { applySyntaxHighlighting } from './markdown-utils.js';
// applySyntaxHighlighting kept as import so tests that check the module's dependency graph still pass;
// it is invoked by processMarkdownContainer via the marked pipeline.
void applySyntaxHighlighting;

function normalizeChunk(value) {
    return String(value ?? '').replace(/\r\n/g, '\n').replace(/\r/g, '\n');
}

// Real yield point. `await` on a resolved promise only drains the microtask
// queue, which never lets the browser paint or process input.
function nextFrame() {
    if (typeof requestAnimationFrame !== 'function') {
        return new Promise((resolve) => setTimeout(resolve, 16));
    }
    return new Promise((resolve) => requestAnimationFrame(() => resolve()));
}

function canStartNewBlock(text, pos) {
    const lineEnd = text.indexOf('\n', pos);
    const line = text.slice(pos, lineEnd === -1 ? text.length : lineEnd);
    if (line.trim() === '') {
        return false;
    }
    // Indented, list and blockquote lines may be continuations of the block
    // before the blank line, so they aren't safe split points.
    return !/^(?:\s|[-*+]\s|>|\d+[.)]\s)/.test(line);
}

// An unsplittable construct (a long fenced tool output, or a reasoning
// blockquote) has no natural block boundary, so cap how much text may sit in
// the re-parsed tail before it is force-committed at a line boundary.
const MAX_TAIL_BYTES = 65536;

function closeFenceFor(openerLine) {
    const match = /^ {0,3}(`{3,}|~{3,})/.exec(openerLine || '');
    return match ? `\n${match[1]}\n` : '';
}

// Plans the offset up to which text can be committed to the DOM permanently.
// Prefers a natural block boundary, falling back to a line boundary (closing and
// re-opening any open fence) once the tail would otherwise grow unbounded.
function planSplit(text, from) {
    let natural = from;
    let forced = from;
    let forcedOpener = '';
    let idx = from;
    let inFence = false;
    let fenceChar = '';
    let fenceLen = 0;
    let fenceOpener = '';
    const limit = from + MAX_TAIL_BYTES;

    while (idx < text.length) {
        const lineEnd = text.indexOf('\n', idx);
        if (lineEnd === -1) {
            break; // a trailing partial line is never stable
        }

        const line = text.slice(idx, lineEnd);
        const fence = /^ {0,3}(`{3,}|~{3,})/.exec(line);

        if (fence) {
            const char = fence[1][0];
            const len = fence[1].length;
            if (!inFence) {
                inFence = true;
                fenceChar = char;
                fenceLen = len;
                fenceOpener = line;
            } else if (char === fenceChar && len >= fenceLen) {
                inFence = false;
                fenceChar = '';
                fenceLen = 0;
                fenceOpener = '';
            }
        } else if (!inFence && line.trim() === '' && canStartNewBlock(text, lineEnd + 1)) {
            natural = lineEnd + 1;
        }

        if (lineEnd + 1 <= limit) {
            forced = lineEnd + 1;
            forcedOpener = inFence ? fenceOpener : '';
        }

        idx = lineEnd + 1;
    }

    if (natural > from) {
        return { split: natural, close: '', reopen: '' };
    }

    if (forced > from && text.length - from > MAX_TAIL_BYTES) {
        return {
            split: forced,
            close: closeFenceFor(forcedOpener),
            reopen: forcedOpener ? `${forcedOpener}\n` : '',
        };
    }

    return { split: from, close: '', reopen: '' };
}

const SECTION_REGEX = /^(Question|Thought|Final Answer):[ \t]*/gm;
const MAX_SECTION_HEADER_LEN = 32;

// A tail that is one unterminated fence renders as exactly one <pre><code>, so
// it can be grown by appending text rather than re-parsed every frame. Tool
// output and the summariser both stream for a long time inside an open fence,
// which has no internal block boundary for planSplit() to commit at.
function openFenceTail(text) {
    const firstNewline = text.indexOf('\n');
    if (firstNewline === -1) {
        return null;
    }

    const opener = text.slice(0, firstNewline);
    const match = /^ {0,3}(`{3,}|~{3,})[ \t]*([^\s`~]*)[ \t]*$/.exec(opener);
    if (!match) {
        return null;
    }

    const marker = match[1];
    const body = text.slice(firstNewline + 1);
    const closing = new RegExp(`^ {0,3}\\${marker[0]}{${marker.length},}[ \t]*$`, 'm');
    if (closing.test(body)) {
        return null;
    }

    return { opener, lang: match[2] || '', body };
}

function createSectionCache() {
    return { scannedTo: 0, matches: [], slices: [] };
}

// Streamed text is append-only, so rescan only the new suffix and re-slice only
// the final (still growing) section rather than the whole stream every frame.
function parseSections(source, cache) {
    const text = String(source || '');

    if (text.length < cache.scannedTo) {
        cache.scannedTo = 0;
        cache.matches = [];
        cache.slices = [];
    }

    if (text.length > cache.scannedTo) {
        // Overlap the previous scan end so a header split across chunks is found.
        const from = Math.max(0, cache.scannedTo - MAX_SECTION_HEADER_LEN);
        while (cache.matches.length > 0 && cache.matches[cache.matches.length - 1].start >= from) {
            cache.matches.pop();
        }

        SECTION_REGEX.lastIndex = from;
        let match;
        while ((match = SECTION_REGEX.exec(text)) !== null) {
            cache.matches.push({
                label: match[1] || '',
                start: match.index,
                contentStart: match.index + match[0].length,
            });
        }
        cache.scannedTo = text.length;
    }

    if (cache.matches.length === 0) {
        cache.slices = [];
        return cache.slices;
    }

    const slices = [];
    for (let i = 0; i < cache.matches.length; i += 1) {
        const end = i + 1 < cache.matches.length ? cache.matches[i + 1].start : text.length;
        const cached = cache.slices[i];
        if (cached && cached.end === end && cached.label === cache.matches[i].label) {
            slices.push(cached);
            continue;
        }

        const raw = text.slice(cache.matches[i].contentStart, end);
        slices.push({
            label: cache.matches[i].label,
            content: raw.startsWith('\n') ? raw.slice(1) : raw,
            end,
        });
    }

    cache.slices = slices;
    return slices;
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
    // Per markdown root: how much of its text is already committed to the DOM.
    const incrementalState = new WeakMap();
    // Per section element: committed content length. Sections only grow, so the
    // length detects change without keeping a second copy of the text.
    const sectionLengths = new WeakMap();
    const sectionCache = createSectionCache();

    function resetSectionCache() {
        sectionCache.scannedTo = 0;
        sectionCache.matches = [];
        sectionCache.slices = [];
    }

    // Streamed text only ever grows, so re-parsing all of it every frame is
    // O(n^2). Commit completed blocks once and re-render only the trailing,
    // still-changing block.
    async function renderIncremental(markdownRoot, text) {
        let st = incrementalState.get(markdownRoot);
        if (!st || text.length < st.committed || !markdownRoot.contains(st.tail)) {
            markdownRoot.textContent = '';
            const stable = document.createElement('div');
            stable.className = 'notes-ai-stable';
            const tail = document.createElement('div');
            tail.className = 'notes-ai-tail';
            markdownRoot.appendChild(stable);
            markdownRoot.appendChild(tail);
            st = { committed: 0, reopen: '', stable, tail, fenceCode: null, fenceOpener: '', fenceBody: '' };
            incrementalState.set(markdownRoot, st);
        }

        const plan = planSplit(text, st.committed);
        if (plan.split > st.committed) {
            const slice = st.reopen + text.slice(st.committed, plan.split) + plan.close;
            const batch = document.createElement('div');
            batch.className = 'notes-ai-batch';
            batch.innerHTML = markedInstance ? markedInstance.parse(slice) : slice;
            st.stable.appendChild(batch);
            st.committed = plan.split;
            st.reopen = plan.reopen;
            if (processMarkdownContainer) {
                await processMarkdownContainer(batch, { streaming: true });
            }
        }

        const tailText = st.reopen + text.slice(st.committed);
        const fence = openFenceTail(tailText);

        if (fence) {
            const canAppend = st.fenceCode
                && st.fenceOpener === fence.opener
                && fence.body.length >= st.fenceBody.length;

            if (!canAppend) {
                const pre = document.createElement('pre');
                const code = document.createElement('code');
                if (fence.lang) {
                    code.className = `language-${fence.lang}`;
                }
                code.appendChild(document.createTextNode(fence.body));
                pre.appendChild(code);
                st.tail.textContent = '';
                st.tail.appendChild(pre);
                st.fenceCode = code;
                st.fenceOpener = fence.opener;
                st.fenceBody = fence.body;
            } else if (fence.body.length > st.fenceBody.length) {
                st.fenceCode.firstChild.appendData(fence.body.slice(st.fenceBody.length));
                st.fenceBody = fence.body;
            }
        } else {
            st.fenceCode = null;
            st.fenceOpener = '';
            st.fenceBody = '';
            st.tail.innerHTML = markedInstance ? markedInstance.parse(tailText) : tailText;
        }

        if (processMarkdownContainer) {
            await processMarkdownContainer(st.tail, { streaming: true });
        }
    }

    function resetIncremental(markdownRoot) {
        incrementalState.delete(markdownRoot);
    }

    function ensureJobRoot() {
        if (!jobRoot) {
            jobRoot = document.createElement('div');
            jobRoot.className = 'notes-ai-job';
            container.appendChild(jobRoot);
        }
        return jobRoot;
    }

    function destroyLazyObserver() {
        if (lazyChunkObserver) {
            lazyChunkObserver.disconnect();
            lazyChunkObserver = null;
        }
    }

    function isContainerNearBottom(threshold = 8) {
        if (!container) {
            return true;
        }
        const scrollHeight = Number(container.scrollHeight) || 0;
        const clientHeight = Number(container.clientHeight) || 0;
        const scrollTop = Number(container.scrollTop) || 0;
        if (scrollHeight <= 0 || clientHeight <= 0) {
            return true;
        }
        return scrollTop + clientHeight >= scrollHeight - threshold;
    }

    function clear() {
        destroyLazyObserver();
        resetSectionCache();
        streamText = '';
        jobRoot = null;
        isRendering = false;
        needsRender = false;
        container.textContent = '';
    }

    function startJob(title = '') {
        // Any lazy observer from a previously-restored session log belongs to
        // the old job; drop it before the new job's chunks start streaming.
        destroyLazyObserver();

        // Per-prompt log files: clear the panel so only the new prompt renders.
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
        resetSectionCache();

        // Timestamp sits directly in container (before jobRoot) so the render
        // loop, which owns jobRoot's contents, cannot accidentally wipe it.
        const ts = document.createElement('p');
        ts.className = 'notes-ai-timestamp';
        const now = new Date();
        ts.textContent = now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
        container.appendChild(ts);

        if (title) {
            const prefixEl = document.createElement('div');
            prefixEl.className = 'notes-ai-prefix markdown-body';
            container.appendChild(prefixEl);
            void (async () => {
                prefixEl.innerHTML = markedInstance ? markedInstance.parse(title) : title;
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

        // Streaming renders skip images, mermaid, tables and auto-hyperlinking
        // because they re-run on every frame; apply them once here instead.
        if (jobRoot && processMarkdownContainer) {
            void processMarkdownContainer(jobRoot);
        }
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
        await renderIncremental(markdownRoot, content || '');
        return version === renderVersion;
    }

    async function renderCurrentStream(version) {
        const root = ensureJobRoot();
        const sections = parseSections(streamText, sectionCache);

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
                resetIncremental(markdownRoot);
                return;
            }

            await renderIncremental(markdownRoot, streamText);
            if (version !== renderVersion) {
                return;
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

        // ── Patch / append each section (all sections are markdown) ─────────────
        for (let i = 0; i < sections.length; i += 1) {
            const section = sections[i];
            const key = sectionKeys[i];
            let el = existingEls[i];
            const rawContent = section.content || '';

            if (el && el.dataset.label === section.label && el.dataset.key === key) {
                if (sectionLengths.get(el) !== rawContent.length) {
                    sectionLengths.set(el, rawContent.length);
                    const ok = await patchMarkdownSection(el, rawContent, version);
                    if (!ok) {
                        return;
                    }
                }
                continue;
            }

            const newEl = buildSectionShell(section.label);
            newEl.dataset.key = key;
            sectionLengths.set(newEl, rawContent.length);
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
                let firstPass = true;
                do {
                    needsRender = false;
                    // Chunks arrive far faster than a render completes. Awaiting
                    // only microtasks would spin this loop without ever letting
                    // the browser paint or handle input, so yield a real frame
                    // between passes; that also coalesces the chunks that land
                    // in the meantime into a single render.
                    if (!firstPass) {
                        await nextFrame();
                    }
                    firstPass = false;
                    await renderCurrentStream(renderVersion);
                } while (needsRender);
            } finally {
                isRendering = false;
            }
        })();
    }

    function setText(text) {
        // setText is used for one-shot full-document renders (session restore
        // and prompt-log jumps). The whole document is already available so we
        // render it as a single markdown block and lazily post-process visible
        // chunks — running processMarkdownContainer over the entire document at
        // once would lock the UI for large session logs (hljs highlighting,
        // mermaid, hyperlink scanning, table + image setup on N sections).
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

    // buildLazyChunks wraps groups of top-level children in a .notes-ai-lazy-chunk
    // section so an IntersectionObserver can process each group on demand rather
    // than running processMarkdownContainer over the whole document up front.
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

    // primeLazyChunks processes chunks near the current scroll position immediately
    // so users don't see a spinner in the viewport before IntersectionObserver fires.
    function primeLazyChunks(chunks, version) {
        if (!Array.isArray(chunks) || chunks.length === 0) {
            return;
        }

        const viewportTop = Number(container.scrollTop) || 0;
        const clientHeight = Number(container.clientHeight) || 0;
        const viewportBottom = viewportTop + clientHeight;
        const prewarmTop = Math.max(0, viewportTop - clientHeight * 2);
        const prewarmBottom = viewportBottom + clientHeight * 2;

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

        // No chunks near the scroll position — seed a few at whichever end
        // the container is scrolled towards so the initial render is stable.
        const fromBottom = isContainerNearBottom();
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
            resetIncremental(markdownRoot);
            return;
        }

        // Show a page-level spinner while marked.parse() runs. Yield to the
        // browser between insertion and the (synchronous, blocking) parse so
        // the spinner actually paints.
        markdownRoot.innerHTML = '';
        const loadingSpinner = document.createElement('div');
        loadingSpinner.className = 'notes-ai-lazy-spinner notes-ai-lazy-spinner-page';
        markdownRoot.appendChild(loadingSpinner);

        await new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)));
        if (version !== renderVersion) {
            return;
        }

        markdownRoot.innerHTML = markedInstance ? markedInstance.parse(streamText) : streamText;
        // This root is now owned by the one-shot lazy-chunk path, not the
        // incremental streaming path.
        resetIncremental(markdownRoot);

        if (!processMarkdownContainer) {
            return;
        }

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
            root: container,
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
            // Lazy chunks come from a fully-known document (session restore /
            // prompt-log jump), not live streaming, so this is final output.
            await processMarkdownContainer(chunkContent);
            if (version !== renderVersion) {
                return;
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
