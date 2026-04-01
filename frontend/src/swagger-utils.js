/**
 * Swagger/OpenAPI utilities for parsing and rendering
 * Handles spec parsing and Postman-like UI generation
 */

import YAML from 'yaml';

/**
 * Detect if a filename is a JSON or YAML file
 * @param {string} filename - File path
 * @returns {boolean} True if file ends with .json, .yml, or .yaml
 */
export function isStructuredDataFile(filename) {
    if (!filename) return false;
    const lower = filename.toLowerCase();
    return lower.endsWith('.json') || lower.endsWith('.yml') || lower.endsWith('.yaml');
}

/**
 * Check whether a parsed JSON/YAML document has a top-level Swagger/OpenAPI key.
 * @param {Object|null} spec - Parsed JSON object
 * @returns {boolean} True if the top-level swagger or openapi key exists
 */
export function hasSwaggerKey(spec) {
    return !!(
        spec &&
        typeof spec === 'object' &&
        !Array.isArray(spec) &&
        (Object.prototype.hasOwnProperty.call(spec, 'swagger') ||
            Object.prototype.hasOwnProperty.call(spec, 'openapi'))
    );
}

/**
 * Parse Swagger/OpenAPI JSON spec
 * @param {string} jsonContent - Raw JSON string
 * @returns {Object|null} Parsed spec or null if invalid
 */
export function parseSwaggerSpec(jsonContent) {
    try {
        return JSON.parse(jsonContent);
    } catch (_) {
        try {
            return YAML.parse(jsonContent);
        } catch (err) {
            console.error('Failed to parse structured spec:', err);
            return null;
        }
    }
}

function decodeJsonPointerToken(token) {
    return token.replace(/~1/g, '/').replace(/~0/g, '~');
}

function resolveSpecRef(spec, ref) {
    if (!spec || typeof ref !== 'string' || !ref.startsWith('#/')) {
        return null;
    }

    return ref
        .slice(2)
        .split('/')
        .map(decodeJsonPointerToken)
        .reduce((acc, key) => (acc && typeof acc === 'object' ? acc[key] : undefined), spec) || null;
}

function resolveMaybeRef(spec, item) {
    if (!item || typeof item !== 'object') {
        return item;
    }

    if (typeof item.$ref === 'string') {
        return resolveSpecRef(spec, item.$ref) || item;
    }

    return item;
}

function getOperationParameters(operation, pathItem, spec) {
    const pathParams = Array.isArray(pathItem?.parameters) ? pathItem.parameters : [];
    const opParams = Array.isArray(operation?.parameters) ? operation.parameters : [];
    return [...pathParams, ...opParams].map((param) => resolveMaybeRef(spec, param)).filter(Boolean);
}

function extractSchemaPrimitiveHint(schema) {
    const resolved = resolveMaybeRef(null, schema);
    if (typeof resolved === 'string') {
        return resolved;
    }
    return resolved?.type || 'string';
}

function buildSchemaExampleValue(schema, spec, visitedRefs = new Set()) {
    if (!schema || typeof schema !== 'object') {
        return undefined;
    }

    if (schema.$ref && typeof schema.$ref === 'string') {
        if (visitedRefs.has(schema.$ref)) {
            return undefined;
        }
        visitedRefs.add(schema.$ref);
        const resolved = resolveSpecRef(spec, schema.$ref);
        return buildSchemaExampleValue(resolved, spec, visitedRefs);
    }

    if (schema.example !== undefined) {
        return schema.example;
    }

    if (schema.default !== undefined) {
        return schema.default;
    }

    if (Array.isArray(schema.enum) && schema.enum.length > 0) {
        return schema.enum[0];
    }

    if (Array.isArray(schema.oneOf) && schema.oneOf.length > 0) {
        return buildSchemaExampleValue(schema.oneOf[0], spec, visitedRefs);
    }

    if (Array.isArray(schema.anyOf) && schema.anyOf.length > 0) {
        return buildSchemaExampleValue(schema.anyOf[0], spec, visitedRefs);
    }

    if (Array.isArray(schema.allOf) && schema.allOf.length > 0) {
        const merged = {};
        let hasValues = false;
        schema.allOf.forEach((part) => {
            const partValue = buildSchemaExampleValue(part, spec, visitedRefs);
            if (partValue && typeof partValue === 'object' && !Array.isArray(partValue)) {
                Object.assign(merged, partValue);
                hasValues = true;
            }
        });
        if (hasValues) {
            return merged;
        }
    }

    const type = schema.type;

    if (type === 'object' || schema.properties) {
        const obj = {};
        let hasValues = false;
        const properties = schema.properties || {};
        for (const [key, propSchema] of Object.entries(properties)) {
            const value = buildSchemaExampleValue(propSchema, spec, visitedRefs);
            if (value !== undefined) {
                obj[key] = value;
                hasValues = true;
            }
        }

        if (!hasValues && Array.isArray(schema.required)) {
            schema.required.forEach((key) => {
                if (!(key in obj)) {
                    obj[key] = '';
                }
            });
            hasValues = schema.required.length > 0;
        }

        return hasValues ? obj : {};
    }

    if (type === 'array') {
        const itemExample = buildSchemaExampleValue(schema.items || {}, spec, visitedRefs);
        return itemExample === undefined ? [] : [itemExample];
    }

    if (type === 'number' || type === 'integer') {
        return 0;
    }

    if (type === 'boolean') {
        return false;
    }

    if (type === 'string') {
        return '';
    }

    return undefined;
}

function getParameterExampleValue(param, spec) {
    if (!param || typeof param !== 'object') {
        return '';
    }

    if (param.example !== undefined) {
        return String(param.example);
    }

    if (param.default !== undefined) {
        return String(param.default);
    }

    const schema = resolveMaybeRef(spec, param.schema || null);
    if (schema && schema.example !== undefined) {
        return String(schema.example);
    }

    if (schema && schema.default !== undefined) {
        return String(schema.default);
    }

    if (schema && Array.isArray(schema.enum) && schema.enum.length > 0) {
        return String(schema.enum[0]);
    }

    return '';
}

/**
 * Extract all paths and methods from spec
 * @param {Object} spec - Parsed Swagger spec
 * @returns {Array} Array of {path, methods}
 */
export function extractPaths(spec) {
    if (!spec || !spec.paths) return [];
    
    const paths = [];
    const httpMethods = ['get', 'post', 'put', 'delete', 'patch', 'head', 'options'];
    
    for (const [path, pathItem] of Object.entries(spec.paths)) {
        const methods = [];
        for (const method of httpMethods) {
            if (pathItem[method]) {
                methods.push({
                    method: method.toUpperCase(),
                    name: pathItem[method].summary || `${method.toUpperCase()} ${path}`,
                    operation: pathItem[method]
                });
            }
        }
        if (methods.length > 0) {
            paths.push({ path, methods });
        }
    }
    
    return paths;
}

/**
 * Extract parameters for a specific operation
 * @param {Object} operation - Operation object from spec
 * @returns {Array} Array of parameters
 */
export function extractParameters(operation, pathItem = null, spec = null) {
    const params = getOperationParameters(operation, pathItem, spec);
    if (params.length === 0) return [];

    return params.map(param => {
        const schema = resolveMaybeRef(spec, param.schema || null) || param.type || 'string';
        return {
            name: param.name,
            in: param.in,
            required: param.required || false,
            description: param.description || '',
            schema,
            example: getParameterExampleValue(param, spec)
        };
    });
}

/**
 * Extract request body schema from operation
 * @param {Object} operation - Operation object from spec
 * @returns {Object} Body schema and content type
 */
export function extractRequestBody(operation, spec = null) {
    if (!operation || !operation.requestBody) {
        return null;
    }

    const requestBody = resolveMaybeRef(spec, operation.requestBody);
    const content = requestBody.content || {};

    // Prefer application/json
    const jsonContent = content['application/json'] || Object.entries(content).find(([key]) => key.includes('json'))?.[1];
    if (!jsonContent) {
        return null;
    }

    const schema = resolveMaybeRef(spec, jsonContent.schema || {});
    const explicitExample = jsonContent.example !== undefined
        ? jsonContent.example
        : (jsonContent.examples && typeof jsonContent.examples === 'object'
            ? Object.values(jsonContent.examples).find((item) => item?.value !== undefined)?.value
            : undefined);

    const bodyExample = explicitExample !== undefined
        ? explicitExample
        : buildSchemaExampleValue(schema, spec);

    return {
        required: requestBody.required || false,
        schema,
        example: typeof bodyExample === 'string' ? bodyExample : JSON.stringify(bodyExample ?? {}, null, 2)
    };
}

/**
 * Extract response schemas from operation
 * @param {Object} operation - Operation object from spec
 * @returns {Array} Array of {status, schema, headers}
 */
export function extractResponses(operation) {
    if (!operation || !operation.responses) return [];
    
    return Object.entries(operation.responses).map(([status, response]) => ({
        status,
        description: response.description || '',
        schema: response.content?.['application/json']?.schema || null,
        headers: response.headers || {}
    }));
}

/**
 * Extract headers from operation
 * @param {Object} operation - Operation object from spec
 * @returns {Array} Array of {name, value}
 */
export function extractHeaders(operation, pathItem = null, spec = null) {
    const headers = [];

    // Add Content-Type derived from the spec's requestBody.content MIME keys (OpenAPI 3.0)
    if (operation.requestBody) {
        const requestBody = resolveMaybeRef(spec, operation.requestBody);
        const contentTypes = Object.keys(requestBody.content || {});

        const preferred = contentTypes.includes('application/json')
            ? 'application/json'
            : (contentTypes[0] || 'application/json');

        headers.push({
            name: 'Content-Type',
            value: preferred,
            options: contentTypes.length > 1 ? contentTypes : null
        });
    } else if (operation.consumes || (spec && spec.consumes)) {
        // Swagger 2.0 style: operation-level consumes or global spec-level consumes
        const consumes = operation.consumes || spec.consumes || [];
        const preferred = consumes.includes('application/json')
            ? 'application/json'
            : (consumes[0] || 'application/json');

        headers.push({
            name: 'Content-Type',
            value: preferred,
            options: consumes.length > 1 ? consumes : null
        });
    }

    // Add Accept header from response content types
    if (operation.responses) {
        const acceptTypes = new Set();

        Object.entries(operation.responses).forEach(([status, response]) => {
            const res = resolveMaybeRef(spec, response);
            // OpenAPI 3.0: response.content
            if (res.content) {
                Object.keys(res.content).forEach(mimeType => acceptTypes.add(mimeType));
            }
        });

        // Swagger 2.0: check operation.produces or global spec.produces
        if (acceptTypes.size === 0 && (operation.produces || (spec && spec.produces))) {
            const produces = operation.produces || spec.produces || [];
            produces.forEach(mimeType => acceptTypes.add(mimeType));
        }

        if (acceptTypes.size > 0) {
            const acceptArray = Array.from(acceptTypes);
            const preferred = acceptArray.includes('application/json')
                ? 'application/json'
                : acceptArray[0];

            headers.push({
                name: 'Accept',
                value: preferred,
                options: acceptArray.length > 1 ? acceptArray : null
            });
        }
    }
    
    // Add specific headers from parameters marked as 'header'
    const mergedParams = getOperationParameters(operation, pathItem, spec);
    if (mergedParams.length > 0) {
        const headerParams = mergedParams.filter(p => p.in === 'header');
        headerParams.forEach(param => {
            headers.push({
                name: param.name,
                value: getParameterExampleValue(param, spec) || `{${param.name}}`,
                required: param.required || false
            });
        });
    }
    
    return headers;
}

/**
 * Generate a simple example from a JSON schema
 * @param {Object} schema - JSON Schema object
 * @returns {string} JSON string of example
 */
export function generateSchemaExample(schema, spec = null) {
    if (!schema) return '{}';

    const example = buildSchemaExampleValue(schema, spec);
    if (example === undefined) {
        return '{}';
    }

    if (typeof example === 'string') {
        return example;
    }

    return JSON.stringify(example, null, 2);
}

/**
 * Generate HTML for request builder UI
 * @param {Object} spec - Parsed Swagger spec
 * @param {Object} selectedEndpoint - {path, method}
 * @returns {string} HTML for request builder
 */
export function generateRequestBuilderHTML(spec, selectedEndpoint) {
    if (!selectedEndpoint || !spec || !spec.paths) {
        return `
            <div class="swagger-request-builder">
                <div class="swagger-empty-state">
                    <p>No endpoint selected. Select an endpoint to view request details.</p>
                </div>
            </div>
        `;
    }
    
    const pathItem = spec.paths[selectedEndpoint.path];
    if (!pathItem) {
        return `
            <div class="swagger-request-builder">
                <div class="swagger-empty-state">
                    <p>Endpoint not found in specification.</p>
                </div>
            </div>
        `;
    }
    
    const operation = pathItem[selectedEndpoint.method.toLowerCase()];
    if (!operation) {
        return `
            <div class="swagger-request-builder">
                <div class="swagger-empty-state">
                    <p>Operation not found for ${selectedEndpoint.method} ${selectedEndpoint.path}.</p>
                </div>
            </div>
        `;
    }
    
    const parameters = extractParameters(operation, pathItem, spec);
    const requestBody = extractRequestBody(operation, spec);
    const headers = extractHeaders(operation, pathItem, spec);
    const endpointTitle = operation.summary || operation.description || `${selectedEndpoint.method} ${selectedEndpoint.path}`;
    
    let html = `
        <div class="swagger-request-builder">
            <div class="swagger-endpoint-sticky">
                <div class="swagger-endpoint-heading markdown-body">
                    <h2 class="swagger-endpoint-title">${escapeHtml(endpointTitle)}</h2>
                </div>
                <div class="swagger-method-url-bar">
                    <button type="button" class="swagger-method-selector" title="Select method">${selectedEndpoint.method}</button>
                    <input type="text" class="swagger-url-input" value="${selectedEndpoint.path}" readonly />
                    <button class="swagger-send-btn">Send</button>
                </div>
                <div class="markdown-body"><h3>Request</h3></div>
                <div class="swagger-request-tabs" role="tablist">
                    <button class="swagger-request-tab" role="tab" data-tab="headers" aria-selected="true">
                        Headers
                    </button>
                    <button class="swagger-request-tab" role="tab" data-tab="body" aria-selected="false">
                        Body
                    </button>
                    <button class="swagger-request-tab" role="tab" data-tab="params" aria-selected="false">
                        Parameters
                    </button>
                </div>
            </div>`;
    
    // Headers Panel
    html += `
        <div class="swagger-request-panel swagger-request-panel-active" data-panel="headers" role="tabpanel">
            <div class="swagger-headers-list">
    `;
    
    if (headers.length === 0) {
        html += `<p class="swagger-empty-field">No headers</p>`;
    } else {
        headers.forEach(header => {
            let valueHtml;
            if (header.options) {
                // Multiple MIME types — render editable input with popup trigger button
                valueHtml = `
                    <div class="swagger-header-value-wrap">
                        <input type="text" class="swagger-header-value swagger-header-input" data-header-name="${escapeHtml(header.name)}" value="${escapeHtml(header.value)}" />
                        <button type="button" class="swagger-header-dropdown" data-header-name="${escapeHtml(header.name)}" data-header-options="${escapeHtml(JSON.stringify(header.options))}" title="Select value" aria-label="Select ${escapeHtml(header.name)} value">&#xf150;</button>
                    </div>
                `;
            } else {
                // Editable single-value input
                valueHtml = `<input type="text" class="swagger-header-value swagger-header-input" data-header-name="${escapeHtml(header.name)}" value="${escapeHtml(header.value)}" />`;
            }
            html += `
                <div class="swagger-header-item">
                    <span class="swagger-header-name">${escapeHtml(header.name)}</span>
                    ${valueHtml}
                </div>
            `;
        });
    }
    
    html += `</div></div>`;
    
    // Body Panel
    html += `
        <div class="swagger-request-panel" data-panel="body" role="tabpanel">
    `;

    if (requestBody) {
        html += `<textarea class="swagger-body-editor">${escapeHtml(requestBody.example)}</textarea>`;
    } else {
        html += `<p class="swagger-empty-field">No request body for this operation</p>`;
    }

    html += `
        </div>
    `;
    
    // Parameters Panel
    html += `
        <div class="swagger-request-panel" data-panel="params" role="tabpanel">
    `;

    if (parameters.length > 0) {
        html += `<div class="swagger-params-form">`;

        parameters.forEach(param => {
            const schemaType = extractSchemaPrimitiveHint(param.schema);
            const required = param.required ? ' *' : '';
            const requiredAttr = param.required ? ' required' : '';

            html += `
                <div class="swagger-param-item">
                    <label class="swagger-param-label">
                        <span class="swagger-param-name">${escapeHtml(param.name)}${required}</span>
                        <span class="swagger-param-meta">${escapeHtml(param.in)} • ${escapeHtml(schemaType)}</span>
                    </label>
                    <input
                        type="text"
                        class="swagger-param-input"
                        data-param-name="${escapeHtml(param.name)}"
                        data-param-in="${escapeHtml(param.in)}"
                        data-param-type="${escapeHtml(schemaType)}"
                        placeholder="${param.example ? 'e.g., ' + escapeHtml(param.example) : 'Enter value'}"
                        value="${param.example ? escapeHtml(param.example) : ''}"
                        ${requiredAttr}
                    />
                    ${param.description
                        ? `<div class="swagger-param-description markdown-body" data-markdown="${escapeHtml(param.description)}"></div>`
                        : ''}
                </div>
            `;
        });

        html += `</div>`;
    } else {
        html += `<p class="swagger-empty-field">No parameters for this operation</p>`;
    }

    html += `
        </div>
    `;
    
    html += `</div>`;
    
    return html;
}

/**
 * Generate HTML for response display
 * @param {Object} spec - Parsed Swagger spec
 * @param {Object} selectedEndpoint - {path, method}
 * @returns {string} HTML for response section
 */
export function generateResponseHTML(spec, selectedEndpoint) {
    if (!selectedEndpoint || !spec || !spec.paths) {
        return `
            <div class="swagger-response-section">
                <div class="swagger-response-header">
                    <span class="swagger-response-meta">No endpoint selected</span>
                </div>
            </div>
        `;
    }
    
    const pathItem = spec.paths[selectedEndpoint.path];
    if (!pathItem) {
        return `
            <div class="swagger-response-section">
                <div class="swagger-response-header">
                    <span class="swagger-response-meta">Endpoint not found</span>
                </div>
            </div>
        `;
    }
    
    const operation = pathItem[selectedEndpoint.method.toLowerCase()];
    if (!operation) {
        return `
            <div class="swagger-response-section">
                <div class="swagger-response-header">
                    <span class="swagger-response-meta">Operation not found</span>
                </div>
            </div>
        `;
    }
    
    const responses = extractResponses(operation);
    const successResponse = responses.find(r => r.status === '200') || responses[0];
    
    let html = `
        <div class="swagger-response-section">
            <div class="markdown-body">
                <h3>Example Response</h3>
            </div>
            <div class="swagger-response-header">
    `;
    
    if (successResponse) {
        const statusClass = `swagger-status-${successResponse.status.charAt(0)}xx`;
        html += `<span class="swagger-status-badge ${statusClass}">${successResponse.status}</span>`;
        html += `<span class="swagger-response-meta">${escapeHtml(successResponse.description || 'No response data')}</span>`;
    } else {
        html += `<span class="swagger-response-meta">No response data yet</span>`;
    }
    
    html += `</div>`;
    
    // Response Tabs
    if (successResponse && successResponse.schema) {
        html += `
            <div class="swagger-response-tabs" role="tablist">
                <button class="swagger-response-tab swagger-response-tab-active" role="tab" data-tab="body" aria-selected="true">
                    Body
                </button>
                <button class="swagger-response-tab" role="tab" data-tab="headers" aria-selected="false">
                    Headers
                </button>
            </div>
            
            <div class="swagger-response-panel swagger-response-panel-active" data-panel="body" role="tabpanel">
                <pre class="swagger-response-body"><code>${escapeHtml(generateSchemaExample(successResponse.schema, spec))}</code></pre>
            </div>
            
            <div class="swagger-response-panel" data-panel="headers" role="tabpanel">
                <div class="swagger-headers-list">
        `;
        
        if (Object.keys(successResponse.headers).length === 0) {
            html += `<p class="swagger-empty-field">No headers defined</p>`;
        } else {
            for (const [headerName, headerDef] of Object.entries(successResponse.headers)) {
                html += `
                    <div class="swagger-header-item">
                        <span class="swagger-header-name">${escapeHtml(headerName)}</span>
                        <span class="swagger-header-value">${escapeHtml(headerDef.description || '')}</span>
                    </div>
                `;
            }
        }
        
        html += `</div></div>`;
    } else {
        html += `<div class="swagger-empty-state"><p>No response schema defined</p></div>`;
    }
    
    html += `</div>`;
    
    return html;
}

/**
 * Generate HTML for endpoint list/navigator
 * @param {Object} spec - Parsed Swagger spec
 * @param {Function} onSelect - Callback when endpoint is selected
 * @returns {string} HTML for endpoint navigator
 */
export function generateEndpointListHTML(spec, selectedEndpoint, filterQuery = '') {
    if (!spec) {
        return `<div class="swagger-empty-state"><p>No Swagger spec loaded</p></div>`;
    }
    
    const paths = extractPaths(spec);
    
    if (paths.length === 0) {
        return `<div class="swagger-empty-state"><p>No endpoints found in spec</p></div>`;
    }
    
    const query = (filterQuery || '').trim().toLowerCase();

    let html = `<div class="swagger-endpoints-list">`;
    let visibleCount = 0;
    
    paths.forEach(({ path, methods }) => {
        methods.forEach(({ method, name }) => {
            if (query) {
                const haystack = `${method} ${path} ${name}`.toLowerCase();
                if (!haystack.includes(query)) {
                    return;
                }
            }

            const isSelected = selectedEndpoint && 
                             selectedEndpoint.path === path && 
                             selectedEndpoint.method === method;
            const methodClass = `swagger-method-${method.toLowerCase()}`;

            visibleCount++;
            
            html += `
                <button class="swagger-endpoint-item ${isSelected ? 'swagger-endpoint-selected' : ''}" 
                        data-path="${escapeHtml(path)}" 
                        data-method="${method}">
                    <span class="swagger-method-badge ${methodClass}">${method}</span>
                    <span class="swagger-endpoint-path">${escapeHtml(path)}</span>
                    <span class="swagger-endpoint-summary">${escapeHtml(name)}</span>
                </button>
            `;
        });
    });

    if (visibleCount === 0) {
        html += `<div class="swagger-empty-state"><p>No operations match your filter</p></div>`;
    }
    
    html += `</div>`;
    
    return html;
}

/**
 * Escape HTML special characters
 * @param {string} str - String to escape
 * @returns {string} Escaped string
 */
function escapeHtml(str) {
    if (typeof str !== 'string') return '';
    return str
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

export function escapeInfoText(str) {
    return escapeHtml(str);
}

/**
 * Build the full request URL from spec + selected endpoint + parameters.
 * @param {Object} spec - Parsed Swagger spec
 * @param {Object} endpoint - {path, method}
 * @param {Object} parameters - {paramName: value} for all parameters
 * @returns {string} Full URL
 */
export function buildRequestUrl(spec, endpoint, parameters = {}) {
    if (!spec || !endpoint) return '';
    
    const base = getBaseUrl(spec).replace(/\/$/, '');
    let path = (endpoint.path || '').replace(/^\/?/, '/');
    
    // Substitute path parameters (e.g., {id} → value)
    Object.entries(parameters).forEach(([name, value]) => {
        if (value !== undefined && value !== null && value !== '') {
            path = path.replace(`{${name}}`, encodeURIComponent(String(value)));
        }
    });
    
    // Collect query parameters
    const queryParams = new URLSearchParams();
    const operation = getOperationFromEndpoint(spec, endpoint);
    const pathItem = spec.paths?.[endpoint.path] || null;
    const operationParams = extractParameters(operation, pathItem, spec);
    
    operationParams.forEach(param => {
        if (param.in === 'query' && parameters[param.name]) {
            const value = parameters[param.name];
            if (value !== undefined && value !== null && value !== '') {
                queryParams.set(param.name, value);
            }
        }
    });
    
    const queryString = queryParams.toString();
    const fullPath = path + (queryString ? `?${queryString}` : '');
    
    return base + fullPath;
}

/**
 * Helper to get operation object from spec and endpoint
 * @private
 */
function getOperationFromEndpoint(spec, endpoint) {
    if (!spec || !spec.paths || !endpoint) return {};
    const pathItem = spec.paths[endpoint.path];
    if (!pathItem) return {};
    return pathItem[endpoint.method.toLowerCase()] || {};
}

/**
 * Generate HTML to display a live (actual) HTTP response.
 * @param {Object} response - {statusCode, status, headers, body, error}
 * @returns {string} HTML
 */
export function generateLiveResponseHTML(response) {
    if (!response) {
        return `<div class="swagger-response-section"><div class="swagger-empty-state"><p>No response</p></div></div>`;
    }

    if (response.error) {
        return `
            <div class="swagger-response-section">
                <div class="swagger-response-header">
                    <span class="swagger-status-badge swagger-status-error">Error</span>
                    <span class="swagger-response-meta">${escapeHtml(response.error)}</span>
                </div>
            </div>
        `;
    }

    const statusCode = response.statusCode || 0;
    const statusCategory = statusCode >= 500 ? '5xx' : statusCode >= 400 ? '4xx' : statusCode >= 300 ? '3xx' : statusCode >= 200 ? '2xx' : '1xx';
    const statusClass = `swagger-status-${statusCategory}`;

    let bodyDisplay = response.body || '';
    try {
        bodyDisplay = JSON.stringify(JSON.parse(response.body), null, 2);
    } catch (_) { /* not JSON — show raw */ }

    const responseHeaders = response.headers || {};

    return `
        <div class="swagger-response-section swagger-live-response">
            <div class="markdown-body">
                <h3>Response</h3>
            </div>
            <div class="swagger-response-header">
                <span class="swagger-status-badge ${statusClass}">${escapeHtml(String(response.status || statusCode))}</span>
                <span class="swagger-response-meta swagger-live-badge">Live</span>
            </div>
            <div class="swagger-response-tabs" role="tablist">
                <button class="swagger-response-tab swagger-response-tab-active" role="tab" data-tab="body" aria-selected="true">Body</button>
                <button class="swagger-response-tab" role="tab" data-tab="headers" aria-selected="false">Headers</button>
            </div>
            <div class="swagger-response-panel swagger-response-panel-active" data-panel="body" role="tabpanel">
                <pre class="swagger-response-body"><code>${escapeHtml(bodyDisplay)}</code></pre>
            </div>
            <div class="swagger-response-panel" data-panel="headers" role="tabpanel">
                <div class="swagger-headers-list">
                    ${Object.keys(responseHeaders).length === 0
                        ? '<p class="swagger-empty-field">No headers</p>'
                        : Object.entries(responseHeaders).map(([k, v]) =>
                            `<div class="swagger-header-item"><span class="swagger-header-name">${escapeHtml(k)}</span><span class="swagger-header-value">${escapeHtml(v)}</span></div>`
                          ).join('')
                    }
                </div>
            </div>
        </div>
    `;
}

/**
 * Get the base URL from swagger spec
 * @param {Object} spec - Parsed Swagger spec
 * @returns {string} Base URL
 */
export function getBaseUrl(spec) {
    if (!spec) return '';
    
    // OpenAPI 3.0+
    if (spec.servers && spec.servers.length > 0) {
        return spec.servers[0].url || '';
    }
    
    // Swagger 2.0
    if (spec.host) {
        const scheme = (spec.schemes && spec.schemes[0]) || 'https';
        const basePath = spec.basePath || '';
        return `${scheme}://${spec.host}${basePath}`;
    }
    
    return '';
}

/**
 * Get API info from spec
 * @param {Object} spec - Parsed Swagger spec
 * @returns {Object} {title, description, version}
 */
export function getApiInfo(spec) {
    if (!spec || !spec.info) {
        return { title: '', description: '', version: '' };
    }
    
    return {
        title: spec.info.title || '',
        description: spec.info.description || '',
        version: spec.info.version || ''
    };
}
