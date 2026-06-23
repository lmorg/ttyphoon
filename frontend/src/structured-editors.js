/**
 * Surgical, format-preserving editors for structured documents.
 *
 * The JSON/YAML "View" mode lets the user edit individual keys and values in a
 * tree. Re-serialising the whole document on every edit (parse -> mutate object
 * -> stringify) reflows the entire file and, for YAML, discards all comments —
 * producing noisy git diffs and data loss.
 *
 * A StructuredEditor instead patches only the bytes belonging to the targeted
 * node, leaving comments, formatting, key ordering, and every untouched node
 * byte-for-byte intact.
 *
 * Each format is a self-contained editor registered in `editors` below. Adding
 * a new format (TOML, XML, ...) is a matter of implementing this shape and
 * appending it to the registry:
 *
 * @typedef {Object} StructuredEditor
 * @property {string} id
 *   Stable identifier, useful for tests and diagnostics.
 * @property {(fileName: string) => boolean} match
 *   Returns true when this editor handles the given file name.
 * @property {(source: string, path: Array<string|number>, value: any) => string} setValue
 *   Returns new source text with the scalar at `path` replaced by `value`.
 * @property {(source: string, path: Array<string|number>, nextKey: string) => string} renameKey
 *   Returns new source text with the object key addressed by `path` renamed to
 *   `nextKey`, preserving the associated value (and its comments/formatting).
 * @property {(source: string, parentPath: Array<string|number>, key: string, value: any) => string} addKey
 *   Returns new source text with a new `key: value` property appended to the
 *   object at `parentPath`.
 * @property {(source: string, arrayPath: Array<string|number>, value: any) => string} addItem
 *   Returns new source text with `value` appended to the array at `arrayPath`.
 * @property {(source: string, path: Array<string|number>) => string} deleteNode
 *   Returns new source text with the property or array element at `path`
 *   removed.
 */

import YAML from 'yaml';
import { applyEdits, findNodeAtLocation, modify, parseTree } from 'jsonc-parser';

const JSON_FORMATTING = { tabSize: 2, insertSpaces: true, eol: '\n' };

/** @type {StructuredEditor} */
const jsonEditor = {
    id: 'json',

    match(fileName) {
        return /\.json$/i.test(fileName || '');
    },

    setValue(source, path, value) {
        const edits = modify(source, path, value, { formattingOptions: JSON_FORMATTING });
        return applyEdits(source, edits);
    },

    renameKey(source, path, nextKey) {
        const tree = parseTree(source);
        if (!tree) {
            throw new Error('Unable to parse JSON document.');
        }

        const valueNode = findNodeAtLocation(tree, path);
        const propertyNode = valueNode ? valueNode.parent : null;
        if (!propertyNode || propertyNode.type !== 'property' || !propertyNode.children) {
            throw new Error('Only object properties can be renamed.');
        }

        const keyNode = propertyNode.children[0];
        if (!keyNode) {
            throw new Error('Unable to locate property to rename.');
        }

        // Replace only the key token, leaving the value and all surrounding
        // bytes untouched.
        const before = source.slice(0, keyNode.offset);
        const after = source.slice(keyNode.offset + keyNode.length);
        return `${before}${JSON.stringify(String(nextKey))}${after}`;
    },

    addKey(source, parentPath, key, value) {
        const edits = modify(source, [...parentPath, String(key)], value, { formattingOptions: JSON_FORMATTING });
        return applyEdits(source, edits);
    },

    addItem(source, arrayPath, value) {
        const tree = parseTree(source);
        const arrayNode = tree ? findNodeAtLocation(tree, arrayPath) : null;
        const length = arrayNode && arrayNode.type === 'array' && arrayNode.children
            ? arrayNode.children.length
            : 0;
        // isArrayInsertion makes jsonc-parser splice a new element in rather
        // than overwrite the element currently at `length`.
        const edits = modify(source, [...arrayPath, length], value, {
            formattingOptions: JSON_FORMATTING,
            isArrayInsertion: true,
        });
        return applyEdits(source, edits);
    },

    deleteNode(source, path) {
        // Passing `undefined` as the value removes the targeted property or
        // array element.
        const edits = modify(source, path, undefined, { formattingOptions: JSON_FORMATTING });
        return applyEdits(source, edits);
    },
};

function parseYamlDocument(source) {
    const doc = YAML.parseDocument(source);
    if (doc.errors && doc.errors.length > 0) {
        throw new Error(doc.errors[0].message || 'Unable to parse YAML document.');
    }
    return doc;
}

/** @type {StructuredEditor} */
const yamlEditor = {
    id: 'yaml',

    match(fileName) {
        return /\.ya?ml$/i.test(fileName || '');
    },

    setValue(source, path, value) {
        const doc = parseYamlDocument(source);
        // setIn replaces only the targeted node; comments and formatting on
        // every other node are retained by the document model.
        doc.setIn(path, value);
        return doc.toString();
    },

    renameKey(source, path, nextKey) {
        const doc = parseYamlDocument(source);
        const parentPath = path.slice(0, -1);
        const currentKey = path[path.length - 1];

        const parent = parentPath.length === 0 ? doc.contents : doc.getIn(parentPath, true);
        if (!parent || !YAML.isMap(parent)) {
            throw new Error('Only object properties can be renamed.');
        }

        const pair = parent.items.find((item) => {
            const key = YAML.isScalar(item.key) ? item.key.value : item.key;
            return String(key) === String(currentKey);
        });
        if (!pair) {
            throw new Error('Unable to locate property to rename.');
        }

        // Mutating the existing key scalar's value (rather than replacing the
        // pair) keeps the value node and any attached comments in place.
        if (YAML.isScalar(pair.key)) {
            pair.key.value = nextKey;
        } else {
            pair.key = doc.createNode(nextKey);
        }

        return doc.toString();
    },

    addKey(source, parentPath, key, value) {
        const doc = parseYamlDocument(source);
        // setIn appends a new pair to the targeted map without disturbing the
        // ordering, comments, or formatting of existing entries.
        doc.setIn([...parentPath, String(key)], value);
        return doc.toString();
    },

    addItem(source, arrayPath, value) {
        const doc = parseYamlDocument(source);
        // addIn appends to the sequence at arrayPath (or the document root when
        // arrayPath is empty).
        doc.addIn(arrayPath, value);
        return doc.toString();
    },

    deleteNode(source, path) {
        const doc = parseYamlDocument(source);
        doc.deleteIn(path);
        return doc.toString();
    },
};

const editors = [jsonEditor, yamlEditor];

/**
 * Resolves the surgical editor for a file, or null when no format matches.
 * @param {string} fileName
 * @returns {StructuredEditor|null}
 */
export function getStructuredEditor(fileName) {
    return editors.find((editor) => editor.match(fileName)) || null;
}

export { jsonEditor, yamlEditor };
