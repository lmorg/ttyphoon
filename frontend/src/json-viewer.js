/**
 * Generic JSON viewer with expandable/collapsible nodes.
 * Can be reused to render any JSON string or object.
 */

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

function buildNode(value, key, depth, path = [], parentType = null) {
    const indent = depth * 18;
    const isEditableKey = key !== null && parentType === 'object';
    const keyPrefix = key !== null
        ? `${isEditableKey
            ? `<span class="json-key json-editable" data-json-edit="key" data-json-path="${encodePath(path)}">"${escapeHtml(key)}"</span>`
            : `<span class="json-key">"${escapeHtml(key)}"</span>`
        }<span class="json-colon">: </span>`
        : '';

    if (Array.isArray(value)) {
        const children = value.map((item, index) => buildNode(item, String(index), depth + 1, [...path, index], 'array')).join('');
        return `
            <div class="json-node" data-node-type="array" data-expanded="true">
                <div class="json-row" style="padding-left: ${indent}px;">
                    <button type="button" class="json-toggle" aria-label="Collapse node" aria-expanded="true"></button>
                    ${keyPrefix}<span class="json-brace">[</span><span class="json-meta">${value.length} item${value.length === 1 ? '' : 's'}</span><span class="json-brace">]</span>
                </div>
                <div class="json-children">${children}</div>
            </div>
        `;
    }

    if (value && typeof value === 'object') {
        const entries = Object.entries(value);
        const children = entries.map(([childKey, childValue]) => buildNode(childValue, childKey, depth + 1, [...path, childKey], 'object')).join('');
        return `
            <div class="json-node" data-node-type="object" data-expanded="true">
                <div class="json-row" style="padding-left: ${indent}px;">
                    <button type="button" class="json-toggle" aria-label="Collapse node" aria-expanded="true"></button>
                    ${keyPrefix}<span class="json-brace">{</span><span class="json-meta">${entries.length} propert${entries.length === 1 ? 'y' : 'ies'}</span><span class="json-brace">}</span>
                </div>
                <div class="json-children">${children}</div>
            </div>
        `;
    }

    return `
        <div class="json-node json-node-leaf" data-node-type="leaf">
            <div class="json-row" style="padding-left: ${indent}px;">
                <span class="json-toggle-placeholder"></span>
                ${keyPrefix}${formatPrimitive(value, path)}
            </div>
        </div>
    `;
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
        const rootHtml = buildNode(parsed, null, 0, []);

        container.innerHTML = `<div class="json-viewer-root">${rootHtml}</div>`;

        const toggles = container.querySelectorAll('.json-toggle');
        toggles.forEach((toggle) => {
            toggle.addEventListener('click', () => {
                const node = toggle.closest('.json-node');
                if (!node) {
                    return;
                }

                const isExpanded = node.getAttribute('data-expanded') !== 'false';
                const nextExpanded = !isExpanded;
                node.setAttribute('data-expanded', nextExpanded ? 'true' : 'false');
                toggle.setAttribute('aria-expanded', nextExpanded ? 'true' : 'false');
                toggle.setAttribute('aria-label', nextExpanded ? 'Collapse node' : 'Expand node');
            });
        });

        const editableElements = container.querySelectorAll('.json-editable');
        editableElements.forEach((editable) => {
            editable.addEventListener('dblclick', (event) => {
                event.preventDefault();
                event.stopPropagation();

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
            });
        });
    } catch (err) {
        container.innerHTML = `
            <div class="json-viewer-error">
                Invalid JSON: ${escapeHtml(err.message || 'Unable to parse JSON')}
            </div>
        `;
    }
}

export function attachJsonViewerEditHandler(container, onEditCommit) {
    if (!container) {
        return;
    }

    container.__jsonViewerOnEditCommit = onEditCommit;
}
