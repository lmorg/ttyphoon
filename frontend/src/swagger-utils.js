/**
 * Swagger/OpenAPI utilities for parsing and rendering
 * Handles spec parsing and Postman-like UI generation
 */

/**
 * Detect if a filename is a Swagger/OpenAPI file
 * @param {string} filename - File path
 * @returns {boolean} True if file matches swagger/openapi pattern
 */
export function isSwaggerFile(filename) {
    if (!filename) return false;
    const lower = filename.toLowerCase();
    return (lower.includes('swagger') || lower.includes('openapi')) && lower.endsWith('.json');
}

/**
 * Parse Swagger/OpenAPI JSON spec
 * @param {string} jsonContent - Raw JSON string
 * @returns {Object|null} Parsed spec or null if invalid
 */
export function parseSwaggerSpec(jsonContent) {
    try {
        return JSON.parse(jsonContent);
    } catch (err) {
        console.error('Failed to parse Swagger spec:', err);
        return null;
    }
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
export function extractParameters(operation) {
    if (!operation || !operation.parameters) return [];
    
    return operation.parameters.map(param => ({
        name: param.name,
        in: param.in,
        required: param.required || false,
        description: param.description || '',
        schema: param.schema || param.type || 'string',
        example: param.example || ''
    }));
}

/**
 * Extract request body schema from operation
 * @param {Object} operation - Operation object from spec
 * @returns {Object} Body schema and content type
 */
export function extractRequestBody(operation) {
    if (!operation || !operation.requestBody) {
        return null;
    }
    
    const requestBody = operation.requestBody;
    const content = requestBody.content || {};
    
    // Prefer application/json
    const jsonContent = content['application/json'];
    if (!jsonContent) {
        return null;
    }
    
    return {
        required: requestBody.required || false,
        schema: jsonContent.schema || {},
        example: jsonContent.example || generateSchemaExample(jsonContent.schema)
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
export function extractHeaders(operation) {
    const headers = [];
    
    // Add Content-Type if request body exists
    if (operation.requestBody) {
        headers.push({
            name: 'Content-Type',
            value: 'application/json'
        });
    }
    
    // Add specific headers from parameters marked as 'header'
    if (operation.parameters) {
        const headerParams = operation.parameters.filter(p => p.in === 'header');
        headerParams.forEach(param => {
            headers.push({
                name: param.name,
                value: param.example || `{${param.name}}`,
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
export function generateSchemaExample(schema) {
    if (!schema) return '{}';
    
    const example = {};
    
    if (schema.properties) {
        for (const [key, prop] of Object.entries(schema.properties)) {
            if (prop.example !== undefined) {
                example[key] = prop.example;
            } else if (prop.type === 'string') {
                example[key] = `"${key}"`;
            } else if (prop.type === 'number' || prop.type === 'integer') {
                example[key] = 0;
            } else if (prop.type === 'boolean') {
                example[key] = true;
            } else if (prop.type === 'array') {
                example[key] = [];
            } else {
                example[key] = null;
            }
        }
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
    
    const parameters = extractParameters(operation);
    const requestBody = extractRequestBody(operation);
    const headers = extractHeaders(operation);
    
    let html = `
        <div class="swagger-request-builder">
            <div class="swagger-method-url-bar">
                <select class="swagger-method-selector" disabled>
                    <option selected>${selectedEndpoint.method}</option>
                </select>
                <input type="text" class="swagger-url-input" value="${selectedEndpoint.path}" readonly />
                <button class="swagger-send-btn">Send</button>
            </div>
            
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
            html += `
                <div class="swagger-header-item">
                    <span class="swagger-header-name">${escapeHtml(header.name)}</span>
                    <span class="swagger-header-value">${escapeHtml(header.value)}</span>
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
        html += `
            <table class="swagger-params-table">
                <thead>
                    <tr>
                        <th>Name</th>
                        <th>Type</th>
                        <th>In</th>
                        <th>Required</th>
                        <th>Description</th>
                    </tr>
                </thead>
                <tbody>
        `;

        parameters.forEach(param => {
            const schemaType = typeof param.schema === 'string' ? param.schema : param.schema.type || 'string';
            html += `
                <tr>
                    <td><code>${escapeHtml(param.name)}</code></td>
                    <td><code>${escapeHtml(schemaType)}</code></td>
                    <td>${escapeHtml(param.in)}</td>
                    <td>${param.required ? '✓' : ''}</td>
                    <td>${escapeHtml(param.description)}</td>
                </tr>
            `;
        });

        html += `
                </tbody>
            </table>
        `;
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
                <pre class="swagger-response-body"><code>${escapeHtml(generateSchemaExample(successResponse.schema))}</code></pre>
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

/**
 * Build the full request URL from spec + selected endpoint.
 * @param {Object} spec - Parsed Swagger spec
 * @param {Object} endpoint - {path, method}
 * @returns {string} Full URL
 */
export function buildRequestUrl(spec, endpoint) {
    if (!spec || !endpoint) return '';
    const base = getBaseUrl(spec).replace(/\/$/, '');
    const path = (endpoint.path || '').replace(/^\/?/, '/');
    return base + path;
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
