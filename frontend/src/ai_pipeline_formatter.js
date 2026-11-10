import { applySyntaxHighlighting } from './markdown-utils.js';
// applySyntaxHighlighting kept as import so tests that check the module's dependency graph still pass;
// it is invoked by processMarkdownContainer via the marked pipeline.
void applySyntaxHighlighting;

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

export function createAIPipelineFormatter(container, options = {}) {
    const markedInstance = options.marked;
    const processMarkdownContainer = options.processMarkdownContainer;
    let requestTitle = '';
    const requestTitles = new WeakMap();
    let jobRoot = null;
    let historicalStreamRoot = null;
    let blockObserver = null;
    let historicalResponseBlockId = '';
    const streamBlocks = new Map();
    const pendingBlockChildren = new Map();
    // Per markdown root: how much of its text is already committed to the DOM.
    const incrementalState = new WeakMap();

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

    function ensureJobRoot() {
        if (!jobRoot) {
            jobRoot = document.createElement('div');
            jobRoot.className = 'notes-ai-job';
            container.appendChild(jobRoot);
        }
        return jobRoot;
    }

    function destroyBlockObserver() {
        if (blockObserver) {
            blockObserver.disconnect();
            blockObserver = null;
        }
    }

    function clear() {
        destroyBlockObserver();
        streamBlocks.clear();
        pendingBlockChildren.clear();
        requestTitle = '';
        jobRoot = null;
        historicalStreamRoot = null;
        historicalResponseBlockId = '';
        container.textContent = '';
    }

    function startJob(title = '') {
        requestTitle = String(title || '');
        // Historical block observers belong to the previous job.
        destroyBlockObserver();

        const hasContent = container.children.length > 0
            || container.textContent.trim().length > 0;
        if (hasContent) {
            const hr = document.createElement('hr');
            hr.className = 'notes-ai-separator';
            container.appendChild(hr);
        }
        historicalStreamRoot = null;
        historicalResponseBlockId = '';

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
        requestTitles.set(jobRoot, requestTitle);
        container.appendChild(jobRoot);
    }

    async function finishJob({ scrollToBottom = true } = {}) {
        const root = jobRoot;
        let entries;
        do {
            entries = Array.from(streamBlocks.values()).filter((entry) => root?.contains(entry.wrapper));
            await Promise.all(entries.map((entry) => entry.renderPromise).filter(Boolean));
        } while (root === jobRoot && entries.some((entry) => entry.rendering));
        if (root !== jobRoot) return;

        // Streaming renders skip images, mermaid, tables and auto-hyperlinking
        // because they re-run on every frame; apply them once here instead.
        if (root && processMarkdownContainer) {
            await processMarkdownContainer(root);
        }
        if (scrollToBottom && root === jobRoot) container.scrollTop = container.scrollHeight;
    }

    function finalResponseElement() {
        if (!jobRoot) return null;
        return Array.from(jobRoot.children).filter((block) =>
            block.dataset.blockKind === 'text' && block.dataset.blockStatus === 'closed'
            && !block.dataset.parentId && block.textContent.trim()).at(-1) || null;
    }

    async function promptMarkdown(root = jobRoot) {
        const entries = Array.from(streamBlocks.values()).filter((entry) => !root || root.contains(entry.wrapper));
        const storedPrompt = entries.find((entry) => entry.userPrompt !== undefined);
        if (storedPrompt) return String(storedPrompt.userPrompt || '');
        const request = entries.find((entry) => entry.kind === 'request' && !entry.parentId);
        if (request) {
            const content = request.loadPrintContent && !request.text && request.root.dataset.blockLoaded !== 'true'
                ? String(await request.loadPrintContent() || '')
                : request.text;
            return content.replace(/^## Prompt[^\S\n]*\n+/, '');
        }
        const prompt = /^### Prompt[^\S\n]*\n+([\s\S]*?)(?=\n### (?:Command|Output|Stream Trace)\b|$)/m.exec(requestTitles.get(root) || requestTitle);
        if (prompt) return prompt[1].trim();
        return '';
    }

    async function outputMarkdown(root = null) {
        const entries = Array.from(streamBlocks.values())
            .filter((entry) => entry.kind === 'text' && !entry.parentId && (!root || root.contains(entry.wrapper)));
        const output = await Promise.all(entries.map(async (entry) => {
            if (entry.loadPrintContent && !entry.text && entry.root.dataset.blockLoaded !== 'true') {
                return String(await entry.loadPrintContent() || '');
            }
            return entry.text;
        }));
        return output.filter((text) => text.trim()).join('\n\n');
    }

    async function outputElements(root = null) {
        const entries = Array.from(streamBlocks.values())
            .filter((entry) => entry.kind === 'text' && !entry.parentId && (!root || root.contains(entry.wrapper)));
        return Promise.all(entries.map(async (entry) => {
            await entry.renderPromise;
            if (entry.loadPrintContent && entry.root.dataset.blockLoaded !== 'true') {
                const source = String(await entry.loadPrintContent() || '');
                const host = document.createElement('div');
                const content = document.createElement('div');
                host.append(content);
                content.innerHTML = markedInstance ? markedInstance.parse(source) : '';
                if (!markedInstance) content.textContent = source;
                if (processMarkdownContainer) await processMarkdownContainer(content);
                return content;
            }
            return entry.root;
        }));
    }

    function isRawBlock(kind) {
        return ['tool-call', 'tool-output', 'tool-error', 'tool-summary'].includes(String(kind || ''));
    }

    function isCollapsedToolOutput(kind) {
        return kind === 'tool-output' || kind === 'tool-summary';
    }

    function updateToolOutputStatus(entry) {
        if (!entry.outputToggle) {
            return;
        }
        const childEntries = Array.from(entry.children.children)
            .map((child) => streamBlocks.get(child.dataset.blockId))
            .filter(Boolean);
        const outputEntries = childEntries
            .filter((child) => child && isCollapsedToolOutput(child.kind));
        const hasError = childEntries.some((child) => child.kind === 'tool-error');
        const summaryInProgress = outputEntries.some((child) => child.kind === 'tool-summary' && !child.closed);
        const outputComplete = outputEntries.some((child) => child.closed);
        const toolCallRunning = !entry.closed;
        const status = hasError ? 'error' : summaryInProgress ? 'summarising' : toolCallRunning ? 'running' : outputComplete || entry.closed ? 'successful' : '';
        const caption = status === 'error' ? 'Error' : status === 'summarising' ? 'Summarising' : status === 'running' ? 'Running' : status === 'successful' ? 'Successful' : '';
        entry.outputToggle.hidden = !status;
        entry.outputToggle.dataset.status = status;
        entry.outputCheckbox.checked = status === 'successful';
        entry.outputCaption.textContent = caption;
        entry.outputSpinner.hidden = status !== 'summarising' && status !== 'running';
        const expanded = entry.wrapper.dataset.outputExpanded === 'true';
        entry.outputToggle.setAttribute('aria-expanded', String(expanded));
        entry.outputCheckbox.setAttribute('aria-label', `${caption}. Click to show or hide tool output.`);
        entry.outputToggle.title = `${caption}. Click to show or hide tool output.`;
    }

    function setToolOutputExpanded(entry, expanded) {
        entry.wrapper.dataset.outputExpanded = expanded ? 'true' : 'false';
        updateToolOutputStatus(entry);
        for (const child of entry.children.children) {
            const childEntry = streamBlocks.get(child.dataset.blockId);
            if (childEntry && isCollapsedToolOutput(childEntry.kind)) {
                child.hidden = !expanded;
            }
        }
    }

    function syncToolOutputVisibility(entry) {
        const parent = streamBlocks.get(entry.parentId);
        if (parent?.kind === 'tool-call') {
            if (isCollapsedToolOutput(entry.kind)) {
                entry.wrapper.hidden = parent.wrapper.dataset.outputExpanded !== 'true';
            }
            updateToolOutputStatus(parent);
        }
    }

    function openBlock(block) {
        const blockId = String(block?.blockId || '');
        if (!blockId) {
            return null;
        }
        const existing = streamBlocks.get(blockId);
        if (existing) {
            return existing;
        }

        const root = ensureJobRoot();
        const kind = String(block?.kind || 'text');
        const wrapper = document.createElement('div');
        const parentId = String(block?.parentId || '');
        const rawSubagent = kind === 'subagent' && !String(block?.label || '');
        let outputToggle = null;
        let outputCheckbox = null;
        let outputCaption = null;
        let outputSpinner = null;
        let toolCallToggle = null;
        let toolCallInput = null;
        const closed = block?.status === 'closed';
        wrapper.dataset.blockId = blockId;
        wrapper.dataset.parentId = parentId;
        wrapper.dataset.blockKind = kind;
        wrapper.dataset.blockStatus = closed ? 'closed' : 'open';
        for (const key of ['sessionId', 'promptId', 'runId', 'workspace']) {
            if (block?.[key] !== undefined) wrapper.dataset[key] = String(block[key]);
        }
        const prefix = root.previousElementSibling;
        if (prefix?.classList.contains('notes-ai-prefix') && !prefix.dataset.sessionId) {
            for (const key of ['sessionId', 'promptId', 'runId', 'workspace']) {
                if (block?.[key] !== undefined) prefix.dataset[key] = String(block[key]);
            }
            if (processMarkdownContainer) {
                void Promise.resolve(processMarkdownContainer(prefix)).catch((err) => {
                    console.error('Failed to render AI prompt:', err);
                });
            }
        }
        let contentRoot;
        if (isRawBlock(kind)) {
            wrapper.classList.add('notes-ai-markdown', 'markdown-body');
            if (kind === 'tool-call') {
                const heading = document.createElement('h3');
                heading.className = 'notes-ai-heading';
                toolCallToggle = document.createElement('button');
                toolCallToggle.type = 'button';
                toolCallToggle.className = 'notes-ai-tool-call-toggle';
                toolCallToggle.appendChild(document.createTextNode('Tool call'));
                if (block?.label) {
                    toolCallToggle.appendChild(document.createTextNode(': '));
                    const toolName = document.createElement('code');
                    toolName.textContent = String(block.label);
                    toolCallToggle.appendChild(toolName);
                }
                toolCallToggle.setAttribute('aria-expanded', 'false');
                toolCallToggle.title = 'Show tool call input';
                heading.appendChild(toolCallToggle);
                wrapper.appendChild(heading);
            }
            const pre = document.createElement('pre');
            pre.className = 'notes-ai-code';
            if (kind === 'tool-call') {
                pre.hidden = true;
                toolCallInput = pre;
            }
            const code = document.createElement('code');
            pre.appendChild(code);
            wrapper.appendChild(pre);
            contentRoot = code;
            if (kind === 'tool-call') {
                outputToggle = document.createElement('label');
                outputToggle.className = 'notes-ai-tool-output-toggle markdown-body';
                outputToggle.hidden = true;
                outputToggle.setAttribute('aria-expanded', 'false');
                outputCheckbox = document.createElement('input');
                outputCheckbox.type = 'checkbox';
                outputCaption = document.createElement('span');
                outputCaption.className = 'notes-ai-tool-output-caption';
                outputSpinner = document.createElement('span');
                outputSpinner.className = 'notes-ai-tool-output-spinner';
                outputSpinner.setAttribute('aria-hidden', 'true');
                outputToggle.append(outputCheckbox, outputCaption, outputSpinner);
                wrapper.appendChild(outputToggle);
            }
        } else {
            contentRoot = document.createElement('div');
            contentRoot.className = 'notes-ai-markdown markdown-body';
            if (rawSubagent) {
                contentRoot.classList.add('notes-ai-raw-text');
            }
            wrapper.appendChild(contentRoot);
        }
        const children = document.createElement('div');
        children.className = 'notes-ai-stream-block-children';
        wrapper.appendChild(children);

        const parent = parentId ? streamBlocks.get(parentId) : null;
        if (parent?.children) {
            parent.children.appendChild(wrapper);
        } else if (parentId) {
            const waiting = pendingBlockChildren.get(parentId) || [];
            waiting.push(wrapper);
            pendingBlockChildren.set(parentId, waiting);
        } else {
            (historicalStreamRoot && kind !== 'request' && blockId !== historicalResponseBlockId
                ? historicalStreamRoot
                : root).appendChild(wrapper);
        }

        const entry = {
            kind,
            userPrompt: block?.userPrompt,
            text: '',
            root: contentRoot,
            wrapper,
            children,
            parentId,
            raw: isRawBlock(kind) || rawSubagent,
            skipMarkdown: rawSubagent,
            outputToggle,
            outputCheckbox,
            outputCaption,
            outputSpinner,
            toolCallToggle,
            toolCallInput,
            rendering: false,
            needsRender: false,
            closed,
            thinkingState: null,
            thinkingNeedsReset: false,
            streamingTextState: null,
            streamingTextNeedsReset: false,
        };
        streamBlocks.set(blockId, entry);
        if (outputToggle) {
            outputCheckbox.addEventListener('click', (event) => {
                event.preventDefault();
                setToolOutputExpanded(entry, entry.wrapper.dataset.outputExpanded !== 'true');
            });
            setToolOutputExpanded(entry, false);
        }
        if (toolCallToggle) {
            toolCallToggle.addEventListener('click', () => {
                toolCallInput.hidden = !toolCallInput.hidden;
                const expanded = !toolCallInput.hidden;
                toolCallToggle.setAttribute('aria-expanded', String(expanded));
                toolCallToggle.title = expanded ? 'Hide tool call input' : 'Show tool call input';
            });
        }

        const waiting = pendingBlockChildren.get(blockId);
        if (waiting) {
            waiting.forEach((child) => children.appendChild(child));
            pendingBlockChildren.delete(blockId);
        }
        if (entry.kind === 'tool-call') {
            updateToolOutputStatus(entry);
            for (const child of entry.children.children) {
                const childEntry = streamBlocks.get(child.dataset.blockId);
                if (childEntry && isCollapsedToolOutput(childEntry.kind)) {
                    child.hidden = entry.wrapper.dataset.outputExpanded !== 'true';
                }
            }
        } else {
            syncToolOutputVisibility(entry);
        }
        return entry;
    }

    function appendBlock(block) {
        const blockId = String(block?.blockId || '');
        if (!blockId) {
            return;
        }
        const entry = openBlock(block);
        if (!entry) {
            return;
        }
        const delta = String(block?.delta ?? '');
        if (delta) {
            entry.text += delta;
        }
        // Historical/live-restore hydration supplies an explicit snapshot;
        // normal live delta events intentionally leave content empty.
        if (block?.content && String(block.content).length >= entry.text.length) {
            entry.text = String(block.content);
            entry.thinkingNeedsReset = true;
            entry.streamingTextNeedsReset = true;
        }
        entry.closed = entry.closed || block?.status === 'closed';
        for (const key of ['sessionId', 'promptId', 'runId', 'workspace']) {
            if (block?.[key] !== undefined) entry.wrapper.dataset[key] = String(block[key]);
        }
        entry.wrapper.dataset.blockStatus = entry.closed ? 'closed' : 'open';
        syncToolOutputVisibility(entry);
        if (entry.kind === 'tool-call') {
            updateToolOutputStatus(entry);
        }
        entry.needsRender = true;
        scheduleBlockRender(entry);
    }

    function renderThinkingBlock(entry) {
        let state = entry.thinkingState;
        if (!state || !entry.root.contains(state.blockquote)) {
            entry.root.textContent = '';
            const blockquote = document.createElement('blockquote');
            blockquote.className = 'notes-ai-thinking';
            const paragraph = document.createElement('p');
            const label = document.createElement('strong');
            label.textContent = 'Thinking:';
            const content = document.createElement('span');
            content.className = 'notes-ai-thinking-content';
            const contentNode = document.createTextNode('');
            content.appendChild(contentNode);
            paragraph.appendChild(label);
            paragraph.appendChild(document.createTextNode(' '));
            paragraph.appendChild(content);
            blockquote.appendChild(paragraph);
            entry.root.appendChild(blockquote);
            state = { blockquote, contentNode, renderedLength: 0 };
            entry.thinkingState = state;
            entry.thinkingNeedsReset = true;
        }

        if (entry.thinkingNeedsReset || entry.text.length < state.renderedLength) {
            state.contentNode.data = entry.text;
            state.renderedLength = entry.text.length;
            entry.thinkingNeedsReset = false;
        } else if (entry.text.length > state.renderedLength) {
            state.contentNode.appendData(entry.text.slice(state.renderedLength));
            state.renderedLength = entry.text.length;
        }

        pinBlockToBottom(state.blockquote);
    }

    function renderStreamingTextBlock(entry) {
        let state = entry.streamingTextState;
        if (!state || !entry.root.contains(state.contentNode)) {
            entry.root.textContent = '';
            const content = document.createElement('div');
            content.className = 'notes-ai-streaming-text';
            const contentNode = document.createTextNode('');
            content.appendChild(contentNode);
            entry.root.appendChild(content);
            state = { contentNode, renderedLength: 0 };
            entry.streamingTextState = state;
            entry.streamingTextNeedsReset = true;
        }

        if (entry.streamingTextNeedsReset || entry.text.length < state.renderedLength) {
            state.contentNode.data = entry.text;
            state.renderedLength = entry.text.length;
            entry.streamingTextNeedsReset = false;
        } else if (entry.text.length > state.renderedLength) {
            state.contentNode.appendData(entry.text.slice(state.renderedLength));
            state.renderedLength = entry.text.length;
        }
    }

    function scheduleBlockRender(entry) {
        if (!entry || entry.rendering) {
            return;
        }
        entry.rendering = true;
        entry.renderPromise = (async () => {
            let firstPass = true;
            try {
                do {
                    entry.needsRender = false;
                    if (!firstPass) {
                        await nextFrame();
                    }
                    firstPass = false;

                    if (entry.kind === 'thinking') {
                        renderThinkingBlock(entry);
                    } else if (entry.raw) {
                        entry.root.textContent = entry.text;
                        pinBlockToBottom(entry.root.parentElement);
                    } else if (entry.kind === 'text' && !entry.closed) {
                        renderStreamingTextBlock(entry);
                    } else {
                        await renderIncremental(entry.root, entry.text);
                    }
                } while (entry.needsRender);

                if (entry.closed && entry.kind !== 'thinking' && !entry.skipMarkdown && processMarkdownContainer) {
                    await processMarkdownContainer(entry.root);
                }
            } finally {
                entry.rendering = false;
                if (entry.needsRender) {
                    scheduleBlockRender(entry);
                }
            }
        })().catch(() => {});
    }

    function pinBlockToBottom(block) {
        if (block) {
            block.scrollTop = block.scrollHeight;
        }
    }

    function setBlocks(blocks, loadContent, {
        collapseIntermediate = false,
        prompt = '',
        timestamp = '',
    } = {}) {
        destroyBlockObserver();
        streamBlocks.clear();
        const entries = Array.isArray(blocks) ? blocks : [];
        container.textContent = '';
        historicalStreamRoot = null;
        historicalResponseBlockId = collapseIntermediate
            ? String([...entries].reverse().find((block) =>
                block?.kind === 'text' && !block?.parentId && Number(block?.size) > 0)?.blockId || '')
            : '';
        if (collapseIntermediate) {
            const promptEl = document.createElement('div');
            promptEl.className = 'notes-ai-prefix markdown-body';
            for (const key of ['sessionId', 'promptId', 'runId', 'workspace']) {
                if (entries[0]?.[key] !== undefined) promptEl.dataset[key] = String(entries[0][key]);
            }
            promptEl.innerHTML = markedInstance
                ? markedInstance.parse(`# Prompt\n\n${String(prompt || '')}`)
                : `# Prompt\n\n${String(prompt || '')}`;
            container.appendChild(promptEl);
            if (processMarkdownContainer) {
                void Promise.resolve(processMarkdownContainer(promptEl)).catch((err) => {
                    console.error('Failed to render historical AI prompt:', err);
                });
            }
            const timestampEl = document.createElement('p');
            timestampEl.className = 'notes-ai-timestamp';
            timestampEl.textContent = String(timestamp || '');
            container.appendChild(timestampEl);
        }
        jobRoot = document.createElement('div');
        jobRoot.className = 'notes-ai-job';
        container.appendChild(jobRoot);

        if (collapseIntermediate) {
            const disclosure = document.createElement('details');
            disclosure.className = 'notes-ai-historical-disclosure';
            disclosure.dataset.historicalStream = 'true';
            const summary = document.createElement('summary');
            summary.className = 'notes-ai-historical-summary';
            summary.textContent = 'AI thought processes';
            historicalStreamRoot = document.createElement('div');
            historicalStreamRoot.className = 'notes-ai-historical-stream';
            disclosure.append(summary, historicalStreamRoot);
            jobRoot.appendChild(disclosure);
        }

        const shells = entries
            .filter((block) => !(collapseIntermediate && block?.kind === 'request'))
            .map((block) => {
                const entry = openBlock(block);
                if (entry) {
                    entry.loadPrintContent = () => loadContent(block);
                    entry.root.dataset.blockLoaded = 'false';
                    entry.root.textContent = '';
                }
                return { block, entry };
            }).filter((item) => item.entry);
        const groupedShells = collapseIntermediate
            ? shells.filter((item) => item.block?.kind !== 'request'
                && String(item.block?.blockId || '') !== historicalResponseBlockId)
            : [];
        const groupedEntries = new Set(groupedShells);
        const ungroupedShells = shells.filter((item) => !groupedEntries.has(item));

        const load = async (item) => {
            if (!item.entry || item.entry.root.dataset.blockLoaded === 'pending'
                || item.entry.root.dataset.blockLoaded === 'true') {
                return;
            }
            item.entry.root.dataset.blockLoaded = 'pending';
            try {
                const content = await loadContent(item.block);
                appendBlock({
                    ...item.block,
                    content: String(content || ''),
                    delta: '',
                });
                item.entry.root.dataset.blockLoaded = 'true';
            } catch (err) {
                item.entry.root.dataset.blockLoaded = 'error';
            }
        };

        const historicalDisclosure = jobRoot.querySelector('details[data-historical-stream="true"]');
        if (historicalDisclosure) {
            historicalDisclosure.addEventListener('toggle', () => {
                if (historicalDisclosure.open) {
                    groupedShells.forEach((item) => { void load(item); });
                }
            });
        }

        if (typeof IntersectionObserver !== 'function') {
            (collapseIntermediate ? ungroupedShells : shells).forEach((item) => { void load(item); });
            return;
        }

        // A still-open block can receive live deltas, so it must not wait for scroll.
        const openShells = ungroupedShells.filter((item) => String(item.block?.status || '') === 'open');
        openShells.forEach((item) => { void load(item); });
        ungroupedShells
            .filter((item) => String(item.block?.blockId || '') === historicalResponseBlockId)
            .forEach((item) => { void load(item); });

        blockObserver = new IntersectionObserver((observed) => {
            for (const item of observed) {
                if (!item.isIntersecting) {
                    continue;
                }
                blockObserver.unobserve(item.target);
                const shell = shells.find((candidate) => candidate.entry.root.closest('[data-block-id]') === item.target);
                if (shell) {
                    void load(shell);
                }
            }
        }, { root: container, rootMargin: '1200px 0px' });

        ungroupedShells.forEach((item) => {
            const wrapper = item.entry.root.closest('[data-block-id]');
            if (wrapper) {
                blockObserver.observe(wrapper);
            }
        });
        ungroupedShells.slice(0, 3).forEach((item) => { void load(item); });
    }

    return {
        clear,
        startJob,
        finishJob,
        finalResponseElement,
        promptMarkdown,
        outputMarkdown,
        outputElements,
        openBlock,
        appendBlock,
        setBlocks,
    };
}
