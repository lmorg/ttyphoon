import { describe, it, expect } from 'vitest';

import { getStructuredEditor, jsonEditor, yamlEditor } from './structured-editors.js';

describe('getStructuredEditor', () => {
    it('resolves the JSON editor for .json files', () => {
        expect(getStructuredEditor('config.json')).toBe(jsonEditor);
        expect(getStructuredEditor('API.JSON')).toBe(jsonEditor);
    });

    it('resolves the YAML editor for .yaml and .yml files', () => {
        expect(getStructuredEditor('config.yaml')).toBe(yamlEditor);
        expect(getStructuredEditor('config.yml')).toBe(yamlEditor);
        expect(getStructuredEditor('CONFIG.YML')).toBe(yamlEditor);
    });

    it('returns null for unknown formats', () => {
        expect(getStructuredEditor('notes.txt')).toBeNull();
        expect(getStructuredEditor('')).toBeNull();
        expect(getStructuredEditor(undefined)).toBeNull();
    });
});

describe('jsonEditor.setValue', () => {
    it('replaces only the targeted value, preserving formatting', () => {
        const source = '{\n  "name": "old",\n  "count": 1\n}';
        const result = jsonEditor.setValue(source, ['name'], 'new');
        expect(result).toBe('{\n  "name": "new",\n  "count": 1\n}');
    });

    it('preserves sibling whitespace and key order on nested edits', () => {
        const source = '{\n  "a": {\n    "b": 1,\n    "c": 2\n  }\n}';
        const result = jsonEditor.setValue(source, ['a', 'c'], 99);
        expect(result).toBe('{\n  "a": {\n    "b": 1,\n    "c": 99\n  }\n}');
    });

    it('edits array elements by index', () => {
        const source = '{\n  "list": [\n    "x",\n    "y"\n  ]\n}';
        const result = jsonEditor.setValue(source, ['list', 1], 'z');
        expect(result).toBe('{\n  "list": [\n    "x",\n    "z"\n  ]\n}');
    });

    it('supports boolean and null values', () => {
        const source = '{\n  "flag": false,\n  "value": 1\n}';
        expect(jsonEditor.setValue(source, ['flag'], true)).toContain('"flag": true');
        expect(jsonEditor.setValue(source, ['value'], null)).toContain('"value": null');
    });
});

describe('jsonEditor.renameKey', () => {
    it('renames a key, leaving the value and surrounding bytes intact', () => {
        const source = '{\n  "old": "value",\n  "keep": 1\n}';
        const result = jsonEditor.renameKey(source, ['old'], 'fresh');
        expect(result).toBe('{\n  "old": "value",\n  "keep": 1\n}'.replace('"old"', '"fresh"'));
        expect(result).toBe('{\n  "fresh": "value",\n  "keep": 1\n}');
    });

    it('renames a nested key without disturbing siblings', () => {
        const source = '{\n  "parent": {\n    "child": true\n  }\n}';
        const result = jsonEditor.renameKey(source, ['parent', 'child'], 'kid');
        expect(result).toBe('{\n  "parent": {\n    "kid": true\n  }\n}');
    });

    it('throws when the target is not an object property', () => {
        const source = '{\n  "list": [1, 2]\n}';
        expect(() => jsonEditor.renameKey(source, ['list', 0], 'nope')).toThrow();
    });
});

describe('yamlEditor.setValue', () => {
    it('replaces a value while preserving comments', () => {
        const source = '# top comment\nname: old # inline\ncount: 1\n';
        const result = yamlEditor.setValue(source, ['name'], 'new');
        expect(result).toContain('# top comment');
        expect(result).toContain('# inline');
        expect(result).toContain('name: new');
        expect(result).toContain('count: 1');
    });

    it('preserves comments on nested edits', () => {
        const source = 'parent:\n  # keep me\n  child: 1\n  other: 2\n';
        const result = yamlEditor.setValue(source, ['parent', 'child'], 42);
        expect(result).toContain('# keep me');
        expect(result).toContain('child: 42');
        expect(result).toContain('other: 2');
    });
});

describe('yamlEditor.renameKey', () => {
    it('renames a key while keeping its value and comments', () => {
        const source = '# doc\nold: value # note\nkeep: 1\n';
        const result = yamlEditor.renameKey(source, ['old'], 'fresh');
        expect(result).toContain('fresh: value');
        expect(result).toContain('# note');
        expect(result).toContain('# doc');
        expect(result).toContain('keep: 1');
        expect(result).not.toMatch(/^old:/m);
    });

    it('renames a nested key without disturbing siblings', () => {
        const source = 'parent:\n  child: true\n  sibling: false\n';
        const result = yamlEditor.renameKey(source, ['parent', 'child'], 'kid');
        expect(result).toContain('kid: true');
        expect(result).toContain('sibling: false');
    });

    it('throws when the parent is not a mapping', () => {
        const source = 'list:\n  - a\n  - b\n';
        expect(() => yamlEditor.renameKey(source, ['list', 0], 'nope')).toThrow();
    });
});

describe('jsonEditor.addKey / addItem / deleteNode', () => {
    it('adds a new key to the root object', () => {
        const source = '{\n  "name": "x"\n}';
        const result = jsonEditor.addKey(source, [], 'extra', '');
        expect(JSON.parse(result)).toEqual({ name: 'x', extra: '' });
        expect(result).toContain('"name": "x"');
    });

    it('adds a new key to a nested object without disturbing siblings', () => {
        const source = '{\n  "parent": {\n    "child": 1\n  }\n}';
        const result = jsonEditor.addKey(source, ['parent'], 'newKey', '');
        expect(JSON.parse(result)).toEqual({ parent: { child: 1, newKey: '' } });
    });

    it('appends an item to an array', () => {
        const source = '{\n  "list": [\n    "a",\n    "b"\n  ]\n}';
        const result = jsonEditor.addItem(source, ['list'], 'c');
        expect(JSON.parse(result)).toEqual({ list: ['a', 'b', 'c'] });
    });

    it('deletes an object property', () => {
        const source = '{\n  "keep": 1,\n  "drop": 2\n}';
        const result = jsonEditor.deleteNode(source, ['drop']);
        expect(JSON.parse(result)).toEqual({ keep: 1 });
    });

    it('deletes an array element by index', () => {
        const source = '{\n  "list": [\n    "a",\n    "b",\n    "c"\n  ]\n}';
        const result = jsonEditor.deleteNode(source, ['list', 1]);
        expect(JSON.parse(result)).toEqual({ list: ['a', 'c'] });
    });
});

describe('yamlEditor.addKey / addItem / deleteNode', () => {
    it('adds a new key while preserving existing comments', () => {
        const source = '# doc\nname: x # note\n';
        const result = yamlEditor.addKey(source, [], 'extra', '');
        expect(result).toContain('# doc');
        expect(result).toContain('# note');
        expect(result).toMatch(/extra:/);
    });

    it('adds a key to a nested mapping', () => {
        const source = 'parent:\n  child: 1\n';
        const result = yamlEditor.addKey(source, ['parent'], 'newKey', '');
        expect(result).toContain('child: 1');
        expect(result).toMatch(/newKey:/);
    });

    it('appends an item to a sequence preserving siblings', () => {
        const source = 'list:\n  - a\n  - b\n';
        const result = yamlEditor.addItem(source, ['list'], 'c');
        expect(result).toContain('- a');
        expect(result).toContain('- b');
        expect(result).toContain('- c');
    });

    it('deletes a key while keeping unrelated comments', () => {
        const source = '# doc\nkeep: 1\ndrop: 2\n';
        const result = yamlEditor.deleteNode(source, ['drop']);
        expect(result).toContain('# doc');
        expect(result).toContain('keep: 1');
        expect(result).not.toMatch(/^drop:/m);
    });

    it('deletes a sequence element by index', () => {
        const source = 'list:\n  - a\n  - b\n  - c\n';
        const result = yamlEditor.deleteNode(source, ['list', 1]);
        expect(result).toContain('- a');
        expect(result).toContain('- c');
        expect(result).not.toMatch(/- b/);
    });
});
