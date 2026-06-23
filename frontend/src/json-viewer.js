/**
 * Generic JSON viewer with expandable/collapsible nodes.
 * Can be reused to render any JSON string or object.
 *
 * Rendering is lazy: at load only a bounded shallow slice of the tree is
 * materialised (see INITIAL_EXPANSION_BUDGET). Deeper container nodes are
 * rendered collapsed and their children are streamed into the DOM on first
 * expand. Small documents fall entirely within the budget and therefore
 * render fully expanded, preserving the previous behaviour.
 */

// Number of container nodes (objects/arrays) that may be eagerly expanded
// during the initial render. Once exhausted, remaining containers render
// collapsed and lazy. Tuned so typical documents render fully expanded while
// very large documents stay responsive.
const INITIAL_EXPANSION_BUDGET = 500;

// Maximum number of direct children a single container renders into the DOM at
// once. The expansion budget above only bounds nesting depth; without this a
// single huge array/object (counted as one expansion) would still emit all of
// its children, producing a pathologically large DOM that froze the UI when it
// was later torn down on file switch. Excess children are revealed on demand
// via a "show more" affordance.
const CHILD_PAGE_SIZE = 200;

function escapeHtml(value) {
    return String(value)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

function encodePath(path) {
    return escapeHtml(JSON.stringify(path));
}

function encodeScalar(value) {
    return escapeHtml(JSON.stringify(value));
}

function formatPrimitive(value, path) {
    const pathAttr = encodePath(path);
    const valueAttr = encodeScalar(value);

    if (value === null) {
        return `<span class="json-value json-value-null json-editable" data-json-edit="value" data-json-path="${pathAttr}" data-json-value="${valueAttr}">null</span>`;
    }

    if (typeof value === 'string') {
        return `<span class="json-value json-value-string json-editable" data-json-edit="value" data-json-path="${pathAttr}" data-json-value="${valueAttr}">"${escapeHtml(value)}"</span>`;
    }

    if (typeof value === 'number') {
        return `<span class="json-value json-value-number json-editable" data-json-edit="value" data-json-path="${pathAttr}" data-json-value="${valueAttr}">${value}</span>`;
    }

    if (typeof value === 'boolean') {
        return `<span class="json-value json-value-boolean json-editable" data-json-edit="value" data-json-path="${pathAttr}" data-json-value="${valueAttr}">${value}</span>`;
    }

    return `<span class="json-value json-value-string json-editable" data-json-edit="value" data-json-path="${pathAttr}" data-json-value="${valueAttr}">"${escapeHtml(String(value))}"</span>`;
}

function buildKeyPrefix(key, path, parentType) {
    if (key === null) {
        return '';
    }

    const isEditableKey = parentType === 'object';
    const keyHtml = isEditableKey
        ? `<span class="json-key json-editable" data-json-edit="key" data-json-path="${encodePath(path)}">"${escapeHtml(key)}"</span>`
        : `<span class="json-key">"${escapeHtml(key)}"</span>`;

    return `${keyHtml}<span class="json-colon">: </span>`;
}

function containerEntries(value, type) {
    if (type === 'array') {
        // [key (string), childValue, pathSegment (number)]
        return value.map((item, index) => [String(index), item, index]);
    }

    // [key (string), childValue, pathSegment (string)]
    return Object.entries(value).map(([childKey, childValue]) => [childKey, childValue, childKey]);
}

function makeNodeId(ctx) {
    ctx.idSeq.n += 1;
    return `jn${ctx.idSeq.n}`;
}

function renderShowMore(moreId, remaining, childDepth) {
    const indent = childDepth * 18;
    const label = remaining === 1 ? 'Show 1 more item' : `Show ${remaining} more items`;
    return `
        <div class="json-show-more" data-more-id="${moreId}" style="padding-left: ${indent}px;">
            <button type="button" class="json-show-more-btn">${label}</button>
        </div>
    `;
}

/**
 * Render up to CHILD_PAGE_SIZE of the supplied child entries. If more remain,
 * the surplus is registered against the context and a "show more" placeholder
 * is appended so the next page can be streamed in on demand. This keeps the
 * DOM bounded in breadth as well as depth.
 */
function buildChildrenHtml(entries, childDepth, path, type, ctx) {
    const total = entries.length;
    const pageEnd = Math.min(total, CHILD_PAGE_SIZE);

    let html = '';
    for (let i = 0; i < pageEnd; i += 1) {
        const [childKey, childValue, pathSeg] = entries[i];
        html += buildNode(childValue, childKey, childDepth, [...path, pathSeg], type, ctx);
    }

    if (total > pageEnd) {
        const moreId = makeNodeId(ctx);
        ctx.moreMap.set(moreId, { entries, start: pageEnd, childDepth, path, type });
        html += renderShowMore(moreId, total - pageEnd, childDepth);
    }

    return html;
}

/**
 * Reveal the next page of children for a "show more" placeholder. Newly
 * rendered containers are collapsed/lazy (zero budget) so revealing a page
 * never cascades into another large render.
 */
function expandShowMore(container, moreEl) {
    const moreId = moreEl.getAttribute('data-more-id');
    if (!moreId) {
        return;
    }

    const ctx = container.__jsonViewerCtx;
    const descriptor = ctx && ctx.moreMap.get(moreId);
    if (!descriptor) {
        return;
    }

    const { entries, start, childDepth, path, type } = descriptor;
    ctx.moreMap.delete(moreId);

    const childCtx = { lazyMap: ctx.lazyMap, moreMap: ctx.moreMap, idSeq: ctx.idSeq, budget: { expansions: 0 } };
    const pageEnd = Math.min(entries.length, start + CHILD_PAGE_SIZE);

    let html = '';
    for (let i = start; i < pageEnd; i += 1) {
        const [childKey, childValue, pathSeg] = entries[i];
        html += buildNode(childValue, childKey, childDepth, [...path, pathSeg], type, childCtx);
    }

    if (entries.length > pageEnd) {
        const nextId = makeNodeId(ctx);
        ctx.moreMap.set(nextId, { entries, start: pageEnd, childDepth, path, type });
        html += renderShowMore(nextId, entries.length - pageEnd, childDepth);
    }

    moreEl.insertAdjacentHTML('beforebegin', html);
    moreEl.remove();
}

function buildContainerNode(value, key, depth, path, parentType, ctx, type) {
    const indent = depth * 18;
    const keyPrefix = buildKeyPrefix(key, path, parentType);
    const isArray = type === 'array';
    const entries = containerEntries(value, type);
    const count = entries.length;
    const meta = isArray
        ? `${count} item${count === 1 ? '' : 's'}`
        : `${count} propert${count === 1 ? 'y' : 'ies'}`;
    const openBrace = isArray ? '[' : '{';
    const closeBrace = isArray ? ']' : '}';

    let expanded = true;
    let childrenHtml = '';
    let lazyAttr = '';

    if (count === 0) {
        // Nothing to expand; render as an expanded but empty node.
        expanded = true;
    } else if (ctx.budget.expansions > 0) {
        // Eagerly expand while we still have budget.
        ctx.budget.expansions -= 1;
        childrenHtml = buildChildrenHtml(entries, depth + 1, path, type, ctx);
    } else {
        // Budget exhausted: defer children until the node is first expanded.
        expanded = false;
        const nodeId = makeNodeId(ctx);
        ctx.lazyMap.set(nodeId, { value, depth, path, type });
        lazyAttr = ` data-lazy="true" data-node-id="${nodeId}"`;
    }

    const expandedAttr = expanded ? 'true' : 'false';
    return `
        <div class="json-node" data-node-type="${type}" data-json-path="${encodePath(path)}" data-expanded="${expandedAttr}"${lazyAttr}>
            <div class="json-row" style="padding-left: ${indent}px;">
                <button type="button" class="json-toggle" aria-label="${expanded ? 'Collapse node' : 'Expand node'}" aria-expanded="${expandedAttr}"></button>
                ${keyPrefix}<span class="json-brace">${openBrace}</span><span class="json-meta">${meta}</span><span class="json-brace">${closeBrace}</span>
            </div>
            <div class="json-children">${childrenHtml}</div>
        </div>
    `;
}

function buildNode(value, key, depth, path = [], parentType = null, ctx) {
    if (Array.isArray(value)) {
        return buildContainerNode(value, key, depth, path, parentType, ctx, 'array');
    }

    if (value && typeof value === 'object') {
        return buildContainerNode(value, key, depth, path, parentType, ctx, 'object');
    }

    const indent = depth * 18;
    const keyPrefix = buildKeyPrefix(key, path, parentType);
    return `
        <div class="json-node json-node-leaf" data-node-type="leaf" data-json-path="${encodePath(path)}">
            <div class="json-row" style="padding-left: ${indent}px;">
                <span class="json-toggle-placeholder"></span>
                ${keyPrefix}${formatPrimitive(value, path)}
            </div>
        </div>
    `;
}

/**
 * Build and inject the direct children of a lazily-rendered container node on
 * first expand. Children are themselves rendered collapsed/lazy (zero budget)
 * so each expand only materialises one additional level — streaming the tree
 * in on demand rather than all at once.
 */
function expandLazyNode(container, node) {
    const nodeId = node.getAttribute('data-node-id');
    if (!nodeId) {
        return;
    }

    const ctx = container.__jsonViewerCtx;
    const descriptor = ctx && ctx.lazyMap.get(nodeId);
    if (!descriptor) {
        return;
    }

    const { value, depth, path, type } = descriptor;
    const entries = containerEntries(value, type);
    const childCtx = { lazyMap: ctx.lazyMap, moreMap: ctx.moreMap, idSeq: ctx.idSeq, budget: { expansions: 0 } };
    const childrenHtml = buildChildrenHtml(entries, depth + 1, path, type, childCtx);

    const childrenContainer = node.querySelector(':scope > .json-children');
    if (childrenContainer) {
        childrenContainer.innerHTML = childrenHtml;
    }

    node.removeAttribute('data-lazy');
    node.removeAttribute('data-node-id');
    ctx.lazyMap.delete(nodeId);
}

function isJsonContainerNode(node) {
    if (!(node instanceof Element)) {
        return false;
    }
    const type = node.getAttribute('data-node-type');
    return type === 'object' || type === 'array';
}

function setJsonNodeExpandedState(node, expanded) {
    node.setAttribute('data-expanded', expanded ? 'true' : 'false');
    const toggle = node.querySelector(':scope > .json-row > .json-toggle');
    if (toggle) {
        toggle.setAttribute('aria-expanded', expanded ? 'true' : 'false');
        toggle.setAttribute('aria-label', expanded ? 'Collapse node' : 'Expand node');
    }
}

/**
 * Recursively expand a container node and every descendant container,
 * materialising lazily-rendered children on the way down. Invoked by the
 * "Expand all" context-menu action.
 */
export function expandJsonViewerSubtree(container, node) {
    if (!container || !isJsonContainerNode(node)) {
        return;
    }

    const stack = [node];
    while (stack.length > 0) {
        const current = stack.pop();
        if (!isJsonContainerNode(current)) {
            continue;
        }

        if (current.getAttribute('data-lazy') === 'true') {
            expandLazyNode(container, current);
        }
        setJsonNodeExpandedState(current, true);

        const childrenWrap = current.querySelector(':scope > .json-children');
        if (childrenWrap) {
            childrenWrap.querySelectorAll(':scope > .json-node').forEach((child) => stack.push(child));
        }
    }
}

/**
 * Recursively collapse a container node and every materialised descendant
 * container. Invoked by the "Collapse all" context-menu action.
 */
export function collapseJsonViewerSubtree(node) {
    if (!isJsonContainerNode(node)) {
        return;
    }

    setJsonNodeExpandedState(node, false);
    node.querySelectorAll('.json-node[data-node-type="object"], .json-node[data-node-type="array"]').forEach((child) => {
        setJsonNodeExpandedState(child, false);
    });
}

function parseInput(input) {
    if (typeof input === 'string') {
        return JSON.parse(input);
    }

    return input;
}

export function renderJsonViewer(container, input) {
    if (!container) {
        return;
    }

    try {
        const parsed = parseInput(input);
        const ctx = {
            lazyMap: new Map(),
            moreMap: new Map(),
            idSeq: { n: 0 },
            budget: { expansions: INITIAL_EXPANSION_BUDGET },
        };
        const rootHtml = buildNode(parsed, null, 0, [], null, ctx);

        const newRoot = document.createElement('div');
        newRoot.className = 'json-viewer-root';
        newRoot.innerHTML = rootHtml;

        // Swap content with replaceChildren rather than innerHTML. The previous
        // tree is detached as a single node in one cheap operation; because the
        // rendered DOM is already bounded in size (see INITIAL_EXPANSION_BUDGET
        // and CHILD_PAGE_SIZE) the orphaned subtree is reclaimed by GC without a
        // blocking teardown and without any per-node disposal work.
        container.__jsonViewerCtx = ctx;
        container.replaceChildren(newRoot);

        attachJsonViewerDelegation(container);
    } catch (err) {
        container.innerHTML = `
            <div class="json-viewer-error">
                Invalid JSON: ${escapeHtml(err.message || 'Unable to parse JSON')}
            </div>
        `;
    }
}

/**
 * Attach delegated listeners to the viewer container once. Using delegation
 * (a single click/dblclick handler) avoids binding listeners to every toggle
 * and editable node, which previously scaled linearly with document size.
 */
function attachJsonViewerDelegation(container) {
    if (container.__jsonViewerDelegated) {
        return;
    }
    container.__jsonViewerDelegated = true;

    container.addEventListener('click', (event) => {
        const showMoreBtn = event.target instanceof Element ? event.target.closest('.json-show-more-btn') : null;
        if (showMoreBtn && container.contains(showMoreBtn)) {
            const moreEl = showMoreBtn.closest('.json-show-more');
            if (moreEl) {
                expandShowMore(container, moreEl);
            }
            return;
        }

        const toggle = event.target instanceof Element ? event.target.closest('.json-toggle') : null;
        if (!toggle || !container.contains(toggle)) {
            return;
        }

        const node = toggle.closest('.json-node');
        if (!node) {
            return;
        }

        if (node.getAttribute('data-lazy') === 'true') {
            expandLazyNode(container, node);
        }

        const isExpanded = node.getAttribute('data-expanded') !== 'false';
        const nextExpanded = !isExpanded;
        node.setAttribute('data-expanded', nextExpanded ? 'true' : 'false');
        toggle.setAttribute('aria-expanded', nextExpanded ? 'true' : 'false');
        toggle.setAttribute('aria-label', nextExpanded ? 'Collapse node' : 'Expand node');
    });

    container.addEventListener('dblclick', (event) => {
        const editable = event.target instanceof Element ? event.target.closest('.json-editable') : null;
        if (!editable || !container.contains(editable)) {
            return;
        }

        event.preventDefault();
        event.stopPropagation();
        startInlineEdit(container, editable);
    });
}

function startInlineEdit(container, editable) {
    if (container.querySelector('.json-inline-editor')) {
        return;
    }

    const editType = editable.getAttribute('data-json-edit');
    const pathAttr = editable.getAttribute('data-json-path');
    if (!editType || !pathAttr) {
        return;
    }

    let path;
    try {
        path = JSON.parse(pathAttr);
    } catch {
        return;
    }

    const rawValueAttr = editable.getAttribute('data-json-value');
    let initialValue = '';
    if (editType === 'key') {
        initialValue = String(path[path.length - 1] ?? '');
    } else if (rawValueAttr) {
        try {
            const parsedValue = JSON.parse(rawValueAttr);
            initialValue = typeof parsedValue === 'string' ? parsedValue : String(parsedValue);
        } catch {
            initialValue = editable.textContent || '';
        }
    }

    const input = document.createElement('input');
    input.type = 'text';
    input.className = 'json-inline-editor';
    input.value = initialValue;
    input.spellcheck = false;
    input.setAttribute('autocomplete', 'off');
    input.setAttribute('autocorrect', 'off');
    input.setAttribute('autocapitalize', 'off');
    input.setAttribute('data-json-inline-editor', 'true');
    input.setAttribute('aria-label', editType === 'key' ? 'Edit property name' : 'Edit value');

    const originalText = editable.innerHTML;
    let finished = false;

    const cleanup = () => {
        if (finished) {
            return;
        }
        finished = true;
        editable.innerHTML = originalText;
        editable.classList.remove('json-editing');
    };

    const cancel = () => {
        cleanup();
    };

    const commit = async () => {
        if (finished) {
            return;
        }

        finished = true;
        editable.classList.remove('json-editing');

        if (typeof container.__jsonViewerOnEditCommit === 'function') {
            await container.__jsonViewerOnEditCommit({
                editType,
                path,
                text: input.value,
            });
        }
    };

    editable.classList.add('json-editing');
    editable.textContent = '';
    editable.appendChild(input);

    const width = Math.max(72, editable.getBoundingClientRect().width + 24, (input.value.length + 1) * 10);
    input.style.width = `${width}px`;
    input.focus();
    input.select();

    input.addEventListener('keydown', async (keyEvent) => {
        if (keyEvent.key === 'Enter') {
            keyEvent.preventDefault();
            keyEvent.stopPropagation();
            await commit();
        } else if (keyEvent.key === 'Escape') {
            keyEvent.preventDefault();
            keyEvent.stopPropagation();
            cancel();
        }
    });

    input.addEventListener('blur', () => {
        if (!finished) {
            cancel();
        }
    });
}

export function attachJsonViewerEditHandler(container, onEditCommit) {
    if (!container) {
        return;
    }

    container.__jsonViewerOnEditCommit = onEditCommit;
}

/**
 * Begin inline editing of the key whose path matches `path`. Used after a new
 * key is inserted via the context menu so the user can immediately type its
 * name. Returns true when a matching key node was found and focused.
 */
export function startJsonViewerKeyEdit(container, path) {
    if (!container) {
        return false;
    }

    const target = JSON.stringify(path);
    const editables = container.querySelectorAll('.json-editable[data-json-edit="key"]');
    for (const editable of editables) {
        if (editable.getAttribute('data-json-path') === target) {
            startInlineEdit(container, editable);
            return true;
        }
    }
    return false;
}
