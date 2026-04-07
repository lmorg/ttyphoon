import { afterEach, describe, expect, it, vi } from 'vitest';

import {
    buildRequestUrl,
    extractHeaders,
    generateRequestBuilderHTML,
    hasSwaggerKey,
    parseSwaggerSpec,
} from './swagger-utils';

const openApiSpec = {
    openapi: '3.0.3',
    servers: [{ url: 'https://api.example.com/v1' }],
    paths: {
        '/pets/{id}': {
            parameters: [
                {
                    name: 'traceId',
                    in: 'header',
                    required: true,
                    schema: { type: 'string', example: 'trace-1' },
                },
            ],
            post: {
                summary: 'Create pet',
                parameters: [
                    {
                        name: 'id',
                        in: 'path',
                        required: true,
                        schema: { type: 'string' },
                    },
                    {
                        name: 'expand',
                        in: 'query',
                        schema: { type: 'string', example: 'owner' },
                    },
                ],
                requestBody: {
                    content: {
                        'application/xml': { schema: { type: 'string' } },
                        'application/json': {
                            schema: {
                                type: 'object',
                                properties: {
                                    name: { type: 'string' },
                                },
                            },
                        },
                    },
                },
                responses: {
                    200: {
                        description: 'ok',
                        content: {
                            'application/json': { schema: { type: 'object' } },
                            'application/xml': { schema: { type: 'string' } },
                        },
                    },
                },
            },
        },
    },
};

describe('swagger-utils', () => {
    afterEach(() => {
        vi.restoreAllMocks();
    });

    it('parses structured specs from json and yaml and detects swagger keys', () => {
        vi.spyOn(console, 'error').mockImplementation(() => {});

        const jsonSpec = parseSwaggerSpec('{"openapi":"3.0.0","paths":{}}');
        const yamlSpec = parseSwaggerSpec('swagger: "2.0"\npaths: {}\n');

        expect(hasSwaggerKey(jsonSpec)).toBe(true);
        expect(hasSwaggerKey(yamlSpec)).toBe(true);
        expect(parseSwaggerSpec('not: [valid')).toBe(null);
    });

    it('extracts content-type, accept, and explicit header parameters', () => {
        const pathItem = openApiSpec.paths['/pets/{id}'];
        const headers = extractHeaders(pathItem.post, pathItem, openApiSpec);

        expect(headers).toEqual(
            expect.arrayContaining([
                expect.objectContaining({
                    name: 'Content-Type',
                    value: 'application/json',
                    options: ['application/xml', 'application/json'],
                }),
                expect.objectContaining({
                    name: 'Accept',
                    value: 'application/json',
                    options: ['application/json', 'application/xml'],
                }),
                expect.objectContaining({
                    name: 'traceId',
                    value: 'trace-1',
                    required: true,
                }),
            ]),
        );
    });

    it('falls back to swagger 2 consumes and produces when requestBody content is absent', () => {
        const swagger2Spec = {
            swagger: '2.0',
            host: 'example.com',
            basePath: '/api',
            schemes: ['https'],
            consumes: ['application/xml', 'application/json'],
            produces: ['text/plain', 'application/json'],
            paths: {
                '/status': {
                    post: {
                        responses: {
                            200: { description: 'ok' },
                        },
                    },
                },
            },
        };

        const headers = extractHeaders(swagger2Spec.paths['/status'].post, swagger2Spec.paths['/status'], swagger2Spec);

        expect(headers).toEqual(
            expect.arrayContaining([
                expect.objectContaining({
                    name: 'Content-Type',
                    value: 'application/json',
                    options: ['application/xml', 'application/json'],
                }),
                expect.objectContaining({
                    name: 'Accept',
                    value: 'application/json',
                    options: ['text/plain', 'application/json'],
                }),
            ]),
        );
    });

    it('renders popup-trigger header controls for multi-option headers', () => {
        const html = generateRequestBuilderHTML(openApiSpec, { path: '/pets/{id}', method: 'POST' });

        expect(html).toContain('class="swagger-method-selector"');
        expect(html).toContain('class="swagger-header-dropdown"');
        expect(html).toContain('data-header-name="Content-Type"');
        expect(html).toContain('data-header-name="Accept"');
        expect(html).toContain('&#xf141;');
    });

    it('builds request urls with substituted path and query parameters', () => {
        const url = buildRequestUrl(openApiSpec, { path: '/pets/{id}', method: 'POST' }, {
            id: 'pet/123',
            expand: 'owner',
        });

        expect(url).toBe('https://api.example.com/v1/pets/pet%2F123?expand=owner');
    });
});