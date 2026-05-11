// table-expressions.js
// Formula parser and evaluator for Excel-like table formulas with relative and absolute cell references.
// Relative: R[1]C[-1] (offsets in square brackets). Absolute: R1C1 (1-based indices, no brackets).
// Supports operators: +, -, *, /, ^, (, )

/**
 * Convert 0-based row/column indexes to Excel-style A1 references.
 * @param {number} row
 * @param {number} col
 * @returns {string}
 */
export function getCellReference(row, col) {
    let colRef = '';
    let c = col;
    do {
        colRef = String.fromCharCode(65 + (c % 26)) + colRef;
        c = Math.floor(c / 26) - 1;
    } while (c >= 0);
    return colRef + (row + 1);
}

function parseA1ColumnReference(colLetters) {
    let colIdx = 0;
    const normalized = String(colLetters || '').toUpperCase();
    for (let i = 0; i < normalized.length; i += 1) {
        colIdx *= 26;
        colIdx += normalized.charCodeAt(i) - 65 + 1;
    }
    return colIdx - 1;
}

function getTableCellValue(rows, rowIdx, colIdx) {
    if (rowIdx < 0 || rowIdx >= rows.length || colIdx < 0 || colIdx >= (rows[0]?.length || 0)) {
        return '';
    }

    const cell = rows[rowIdx][colIdx];
    return cell == null ? '' : String(cell);
}

/**
 * Parse a function style formula, eg =MyFn(1, A2, "x")
 * @param {string} formula
 * @returns {{ fnName: string, args: string[] } | null}
 */
export function parseTableFunctionCall(formula) {
    if (!isTableFormula(formula)) {
        return null;
    }

    const match = String(formula).trim().match(/^=\s*([A-Za-z_][A-Za-z0-9_]*)\s*\((.*)\)\s*$/s);
    if (!match) {
        return null;
    }

    const fnName = match[1];
    const argsRaw = match[2] || '';

    const args = [];
    let current = '';
    let inQuote = false;

    for (let i = 0; i < argsRaw.length; i += 1) {
        const ch = argsRaw[i];

        if (inQuote) {
            current += ch;
            if (ch === '"' && argsRaw[i - 1] !== '\\') {
                inQuote = false;
            }
            continue;
        }

        if (ch === '"') {
            inQuote = true;
            current += ch;
            continue;
        }

        if (ch === ',') {
            args.push(current.trim());
            current = '';
            continue;
        }

        if (/\s/.test(ch)) {
            continue;
        }

        current += ch;
    }

    if (current.trim().length > 0 || argsRaw.trim().length > 0) {
        args.push(current.trim());
    }

    return { fnName, args };
}

/**
 * Resolve a function argument to a scalar string value.
 * Supports quoted literals and formula/reference expressions.
 * @param {string} arg
 * @param {number} row
 * @param {number} col
 * @param {string[][]} rows
 * @returns {string}
 */
export function resolveTableFunctionArg(arg, row, col, rows) {
    const source = String(arg || '').trim();
    if (!source) {
        return '';
    }

    const doubleQuoted = source.match(/^"(.*)"$/s);
    if (doubleQuoted) {
        return doubleQuoted[1];
    }

    const r1c1Pattern = /^R(\[(-?\d+)\]|(\d+))C(\[(-?\d+)\]|(\d+))$/i;
    const r1c1Match = source.match(r1c1Pattern);
    if (r1c1Match) {
        const targetRow = r1c1Match[2] !== undefined ? row + parseInt(r1c1Match[2], 10) : parseInt(r1c1Match[3], 10) - 1;
        const targetCol = r1c1Match[5] !== undefined ? col + parseInt(r1c1Match[5], 10) : parseInt(r1c1Match[6], 10) - 1;
        if (targetRow < 0 || targetRow >= rows.length || targetCol < 0 || targetCol >= (rows[0]?.length || 0)) {
            return '';
        }
        const cell = rows[targetRow][targetCol];
        return cell == null ? '' : String(cell);
    }

    const a1Pattern = /^([A-Z]+)(\d+)$/i;
    const a1Match = source.match(a1Pattern);
    if (a1Match) {
        const colLetters = a1Match[1].toUpperCase();
        const rowIdx = parseInt(a1Match[2], 10) - 1;

        const colIdx = parseA1ColumnReference(colLetters);

        if (rowIdx < 0 || rowIdx >= rows.length || colIdx < 0 || colIdx >= (rows[0]?.length || 0)) {
            return '';
        }

        const cell = rows[rowIdx][colIdx];
        return cell == null ? '' : String(cell);
    }

    if (/^-?\d+(?:\.\d+)?$/.test(source)) {
        return source;
    }

    const evaluated = evaluateTableFormula(`=${source}`, row, col, rows);
    if (evaluated === '#ERR') {
        return source;
    }

    return String(evaluated);
}

/**
 * Resolve a function argument to zero-or-more scalar string values (async, with nested function support).
 * Ranges expand to multiple parameters in row-major order.
 * Nested function calls are evaluated recursively via the provided executor function.
 * @param {Function} functionExecutor - Async function(fnName, fnArgs, row, col) that executes a function
 * @param {string} arg
 * @param {number} row
 * @param {number} col
 * @param {string[][]} rows
 * @returns {Promise<string[]>}
 */
export async function resolveTableFunctionArgsAsync(functionExecutor, arg, row, col, rows) {
    const source = String(arg || '').trim();
    if (!source) {
        return [''];
    }

    // Check if this is a nested function call: functionName(...)
    const isFunctionCall = /^[A-Za-z_][A-Za-z0-9_]*\s*\(/.test(source);
    if (isFunctionCall) {
        const parsed = parseTableFunctionCall(`=${source}`);
        if (parsed) {
            // Recursively resolve nested function
            const result = await functionExecutor(parsed.fnName, parsed.args, row, col);
            return [String(result)];
        }
    }

    // Check for A1:B5 range pattern
    const a1RangeMatch = source.match(/^([A-Z]+)(\d+):([A-Z]+)(\d+)$/i);
    if (a1RangeMatch) {
        const startCol = parseA1ColumnReference(a1RangeMatch[1]);
        const startRow = parseInt(a1RangeMatch[2], 10) - 1;
        const endCol = parseA1ColumnReference(a1RangeMatch[3]);
        const endRow = parseInt(a1RangeMatch[4], 10) - 1;
        const rowStart = Math.min(startRow, endRow);
        const rowEnd = Math.max(startRow, endRow);
        const colStart = Math.min(startCol, endCol);
        const colEnd = Math.max(startCol, endCol);
        const values = [];

        for (let rowIdx = rowStart; rowIdx <= rowEnd; rowIdx += 1) {
            for (let colIdx = colStart; colIdx <= colEnd; colIdx += 1) {
                values.push(getTableCellValue(rows, rowIdx, colIdx));
            }
        }

        return values;
    }

    // Check for A:B column range pattern
    const wholeColumnMatch = source.match(/^([A-Z]+):([A-Z]+)$/i);
    if (wholeColumnMatch) {
        const startCol = parseA1ColumnReference(wholeColumnMatch[1]);
        const endCol = parseA1ColumnReference(wholeColumnMatch[2]);
        const colStart = Math.min(startCol, endCol);
        const colEnd = Math.max(startCol, endCol);
        const values = [];

        for (let rowIdx = 0; rowIdx < rows.length; rowIdx += 1) {
            for (let colIdx = colStart; colIdx <= colEnd; colIdx += 1) {
                values.push(getTableCellValue(rows, rowIdx, colIdx));
            }
        }

        return values;
    }

    // Check for 1:2 row range pattern
    const wholeRowMatch = source.match(/^(\d+):(\d+)$/);
    if (wholeRowMatch) {
        const startRow = parseInt(wholeRowMatch[1], 10) - 1;
        const endRow = parseInt(wholeRowMatch[2], 10) - 1;
        const rowStart = Math.min(startRow, endRow);
        const rowEnd = Math.max(startRow, endRow);
        const values = [];
        const colCount = rows[0]?.length || 0;

        for (let rowIdx = rowStart; rowIdx <= rowEnd; rowIdx += 1) {
            for (let colIdx = 0; colIdx < colCount; colIdx += 1) {
                values.push(getTableCellValue(rows, rowIdx, colIdx));
            }
        }

        return values;
    }

    // Default: single scalar value
    return [resolveTableFunctionArg(source, row, col, rows)];
}

/**
 * Resolve a function argument to zero-or-more scalar string values.
 * Ranges expand to multiple parameters in row-major order.
 * @param {string} arg
 * @param {number} row
 * @param {number} col
 * @param {string[][]} rows
 * @returns {string[]}
 */
export function resolveTableFunctionArgs(arg, row, col, rows) {
    const source = String(arg || '').trim();
    if (!source) {
        return [''];
    }

    const a1RangeMatch = source.match(/^([A-Z]+)(\d+):([A-Z]+)(\d+)$/i);
    if (a1RangeMatch) {
        const startCol = parseA1ColumnReference(a1RangeMatch[1]);
        const startRow = parseInt(a1RangeMatch[2], 10) - 1;
        const endCol = parseA1ColumnReference(a1RangeMatch[3]);
        const endRow = parseInt(a1RangeMatch[4], 10) - 1;
        const rowStart = Math.min(startRow, endRow);
        const rowEnd = Math.max(startRow, endRow);
        const colStart = Math.min(startCol, endCol);
        const colEnd = Math.max(startCol, endCol);
        const values = [];

        for (let rowIdx = rowStart; rowIdx <= rowEnd; rowIdx += 1) {
            for (let colIdx = colStart; colIdx <= colEnd; colIdx += 1) {
                values.push(getTableCellValue(rows, rowIdx, colIdx));
            }
        }

        return values;
    }

    const wholeColumnMatch = source.match(/^([A-Z]+):([A-Z]+)$/i);
    if (wholeColumnMatch) {
        const startCol = parseA1ColumnReference(wholeColumnMatch[1]);
        const endCol = parseA1ColumnReference(wholeColumnMatch[2]);
        const colStart = Math.min(startCol, endCol);
        const colEnd = Math.max(startCol, endCol);
        const values = [];

        for (let rowIdx = 0; rowIdx < rows.length; rowIdx += 1) {
            for (let colIdx = colStart; colIdx <= colEnd; colIdx += 1) {
                values.push(getTableCellValue(rows, rowIdx, colIdx));
            }
        }

        return values;
    }

    const wholeRowMatch = source.match(/^(\d+):(\d+)$/);
    if (wholeRowMatch) {
        const startRow = parseInt(wholeRowMatch[1], 10) - 1;
        const endRow = parseInt(wholeRowMatch[2], 10) - 1;
        const rowStart = Math.min(startRow, endRow);
        const rowEnd = Math.max(startRow, endRow);
        const values = [];
        const colCount = rows[0]?.length || 0;

        for (let rowIdx = rowStart; rowIdx <= rowEnd; rowIdx += 1) {
            for (let colIdx = 0; colIdx < colCount; colIdx += 1) {
                values.push(getTableCellValue(rows, rowIdx, colIdx));
            }
        }

        return values;
    }

    return [resolveTableFunctionArg(source, row, col, rows)];
}

/**
 * Inject resolved args into code templates.
 * Supported tokens: {{args}}, {{arg1}}, {{arg2}}, ...
 * @param {string} code
 * @param {string[]} args
 * @returns {string}
 */
export function injectFunctionArgsIntoCode(code, args) {
    let rendered = String(code || '');
    rendered = rendered.replace(/\{\{\s*args\s*\}\}/gi, JSON.stringify(args));

    args.forEach((arg, index) => {
        const n = index + 1;
        const escaped = String(arg ?? '')
            .replace(/\\/g, '\\\\')
            .replace(/"/g, '\\"');
        rendered = rendered.replace(new RegExp(`\\{\\{\\s*arg${n}\\s*\\}\\}`, 'gi'), escaped);
    });

    return rendered;
}

/**
 * Parse and evaluate a formula string for a table cell.
 * @param {string} formula - The formula string (must start with '=')
 * @param {number} row - The row index of the current cell (0-based)
 * @param {number} col - The column index of the current cell (0-based)
 * @param {string[][]} table - The table data as a 2D array of strings
 * @param {Set<string>} [visited] - Used for cycle detection
 * @returns {number|string} - The evaluated result, or error string
 */
export function evaluateTableFormula(formula, row, col, table, visited = new Set()) {
    if (!formula.startsWith('=')) return formula;
    try {
        // 1. Replace R1C1 and R[1]C[1] style references
        const refPattern = /R(\[(-?\d+)\]|(\d+))C(\[(-?\d+)\]|(\d+))/gi;
        let expr = formula.slice(1).replace(refPattern, (match, _rFull, rRel, rAbs, _cFull, cRel, cAbs) => {
            const targetRow = rRel !== undefined ? row + parseInt(rRel, 10) : parseInt(rAbs, 10) - 1;
            const targetCol = cRel !== undefined ? col + parseInt(cRel, 10) : parseInt(cAbs, 10) - 1;
            if (
                targetRow < 0 || targetRow >= table.length ||
                targetCol < 0 || targetCol >= (table[0]?.length || 0)
            ) {
                return '0'; // Out of bounds = 0
            }
            const refKey = `${targetRow},${targetCol}`;
            if (visited.has(refKey)) {
                throw new Error('Circular reference');
            }
            visited.add(refKey);
            const refValue = table[targetRow][targetCol] || '';
            let val;
            if (typeof refValue === 'string' && refValue.trim().startsWith('=')) {
                val = evaluateTableFormula(refValue, targetRow, targetCol, table, visited);
                if (val === '#ERR' || val === 'NaN') {
                    throw new Error('Propagated error');
                }
            } else {
                val = parseFloat(refValue);
            }
            visited.delete(refKey);
            return isNaN(val) ? '0' : val;
        });

        // 2. Replace A1-style references (e.g., A1, B2, AA10)
        // Only match if not part of a longer word, and not inside quotes
        // Use negative lookbehind for word boundary or start, and negative lookahead for word boundary
        const a1Pattern = /\b([A-Z]+)(\d+)\b/g;
        expr = expr.replace(a1Pattern, (match, colLetters, rowNum) => {
            // Convert colLetters (A, B, ..., Z, AA, AB, ...) to 0-based index
            let colIdx = 0;
            for (let i = 0; i < colLetters.length; i++) {
                colIdx *= 26;
                colIdx += colLetters.charCodeAt(i) - 65 + 1;
            }
            colIdx -= 1;
            const rowIdx = parseInt(rowNum, 10) - 1;
            if (
                rowIdx < 0 || rowIdx >= table.length ||
                colIdx < 0 || colIdx >= (table[0]?.length || 0)
            ) {
                return '0';
            }
            const refKey = `${rowIdx},${colIdx}`;
            if (visited.has(refKey)) {
                throw new Error('Circular reference');
            }
            visited.add(refKey);
            const refValue = table[rowIdx][colIdx] || '';
            let val;
            if (typeof refValue === 'string' && refValue.trim().startsWith('=')) {
                val = evaluateTableFormula(refValue, rowIdx, colIdx, table, visited);
                if (val === '#ERR' || val === 'NaN') {
                    throw new Error('Propagated error');
                }
            } else {
                val = parseFloat(refValue);
            }
            visited.delete(refKey);
            return isNaN(val) ? '0' : val;
        });

        // Only allow safe math operators
        if (/[^0-9+\-*/^(). ]/.test(expr)) {
            throw new Error('Invalid characters in formula');
        }
        // Replace ^ with ** for JS eval
        const jsExpr = expr.replace(/\^/g, '**');
        // eslint-disable-next-line no-new-func
        const result = Function(`"use strict";return (${jsExpr})`)();
        if (typeof result === 'number' && isFinite(result)) {
            return result;
        }
        return 'NaN';
    } catch (err) {
        return '#ERR';
    }
}

/**
 * Utility: Check if a string is a formula (starts with '=')
 */
export function isTableFormula(str) {
    return typeof str === 'string' && str.trim().startsWith('=');
}
