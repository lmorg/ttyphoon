import { describe, it, expect } from 'vitest';
import {
    evaluateTableFormula,
    isTableFormula,
    parseTableFunctionCall,
    resolveTableFunctionArg,
    resolveTableFunctionArgs,
    resolveTableFunctionArgsAsync,
} from './table-expressions.js';

describe('isTableFormula', () => {
    it('returns true for strings starting with =', () => {
        expect(isTableFormula('=R[0]C[1] + R[1]C[0]')).toBe(true);
        expect(isTableFormula('  =test')).toBe(true);
    });

    it('returns false for non-formula strings', () => {
        expect(isTableFormula('hello')).toBe(false);
        expect(isTableFormula('42')).toBe(false);
        expect(isTableFormula('')).toBe(false);
        expect(isTableFormula(null)).toBe(false);
        expect(isTableFormula(undefined)).toBe(false);
    });
});

describe('evaluateTableFormula', () => {
    const table = [
        ['Name', 'A', 'B', 'C'],
        ['Row1', '10', '20', '30'],
        ['Row2', '5', '15', '25'],
        ['Row3', '2', '4', '6'],
    ];

    it('returns non-formula strings as-is', () => {
        expect(evaluateTableFormula('hello', 0, 0, table)).toBe('hello');
    });

    it('evaluates simple arithmetic', () => {
        expect(evaluateTableFormula('=1 + 2', 0, 0, table)).toBe(3);
        expect(evaluateTableFormula('=10 * 3', 0, 0, table)).toBe(30);
        expect(evaluateTableFormula('=10 / 2', 0, 0, table)).toBe(5);
        expect(evaluateTableFormula('=10 - 3', 0, 0, table)).toBe(7);
    });

    it('evaluates power operator', () => {
        expect(evaluateTableFormula('=2 ^ 3', 0, 0, table)).toBe(8);
        expect(evaluateTableFormula('=3 ^ 2', 0, 0, table)).toBe(9);
    });

    it('evaluates subexpressions with parentheses', () => {
        expect(evaluateTableFormula('=(2 + 3) * 4', 0, 0, table)).toBe(20);
        expect(evaluateTableFormula('=2 * (3 + 4)', 0, 0, table)).toBe(14);
    });

    it('resolves relative cell references with R[]C[] syntax', () => {
        // From cell (1,1) which is "10", R[0]C[1] = row+0,col+1 = (1,2) = "20"
        expect(evaluateTableFormula('=R[0]C[1]', 1, 1, table)).toBe(20);
        // From cell (1,1), R[1]C[0] = row+1,col+0 = (2,1) = "5"
        expect(evaluateTableFormula('=R[1]C[0]', 1, 1, table)).toBe(5);
    });

    it('resolves negative relative references', () => {
        // From cell (2,2) which is "15", R[0]C[-1] = row+0,col-1 = (2,1) = "5"
        expect(evaluateTableFormula('=R[0]C[-1]', 2, 2, table)).toBe(5);
        // From cell (2,2), R[-1]C[0] = row-1,col+0 = (1,2) = "20"
        expect(evaluateTableFormula('=R[-1]C[0]', 2, 2, table)).toBe(20);
    });

    it('resolves absolute cell references with RnCn syntax', () => {
        // R2C2 = absolute row 2, col 2 (1-based) = (1,1) = "10"
        expect(evaluateTableFormula('=R2C2', 0, 0, table)).toBe(10);
        // R1C1 = (0,0) = "Name" (non-numeric) = 0
        expect(evaluateTableFormula('=R1C1 + 5', 3, 3, table)).toBe(5);
        // R3C3 = (2,2) = "15"
        expect(evaluateTableFormula('=R3C3', 0, 0, table)).toBe(15);
    });

    it('resolves absolute cell references with A1 syntax', () => {
        // B2 = (1,1) = "10"
        expect(evaluateTableFormula('=B2', 0, 0, table)).toBe(10);
        // A1 = (0,0) = "Name" (non-numeric) = 0
        expect(evaluateTableFormula('=A1 + 5', 3, 3, table)).toBe(5);
        // C3 = (2,2) = "15"
        expect(evaluateTableFormula('=C3', 0, 0, table)).toBe(15);
    });

    it('combines cell references with operators', () => {
        // From cell (1,1)="10": R[0]C[1]="20" + R[0]C[2]="30" = 50
        expect(evaluateTableFormula('=R[0]C[1] + R[0]C[2]', 1, 1, table)).toBe(50);
        // From cell (1,1)="10": R[0]C[1]="20" * R[1]C[0]="5" = 100
        expect(evaluateTableFormula('=R[0]C[1] * R[1]C[0]', 1, 1, table)).toBe(100);
    });

    it('returns 0 for out-of-bounds references', () => {
        // From cell (0,0), R[0]C[-1] is out of bounds
        expect(evaluateTableFormula('=R[0]C[-1] + 5', 0, 0, table)).toBe(5);
        // From cell (3,3), R[0]C[1] is out of bounds
        expect(evaluateTableFormula('=R[0]C[1] + 7', 3, 3, table)).toBe(7);
    });

    it('detects circular references', () => {
        const circTable = [
            ['=R[0]C[1]', '=R[0]C[-1]'],
        ];
        expect(evaluateTableFormula('=R[0]C[1]', 0, 0, circTable)).toBe('#ERR');
    });

    it('evaluates chained formulas (formula referencing another formula)', () => {
        const chainTable = [
            ['10', '20', '=R[0]C[-1] + R[0]C[-2]'],  // col2 = 20 + 10 = 30
            ['5',  '=R[0]C[-1] * 2', '=R[0]C[-1]'],  // col1 = 5*2=10, col2 = 10
        ];
        // From (0,2): R[0]C[-1]=col1="20", R[0]C[-2]=col0="10" => 30
        expect(evaluateTableFormula('=R[0]C[-1] + R[0]C[-2]', 0, 2, chainTable)).toBe(30);
        // From (1,2): R[0]C[-1]=col1="=R[0]C[-1] * 2" which from (1,1) => col0="5" => 5*2=10
        expect(evaluateTableFormula('=R[0]C[-1]', 1, 2, chainTable)).toBe(10);
    });

    it('mixes absolute and relative references', () => {
        // From cell (1,1)="10": R2C3 (absolute row2,col3 = (1,2)) = "20", R[1]C[0] (relative) = (2,1) = "5"
        expect(evaluateTableFormula('=R2C3 + R[1]C[0]', 1, 1, table)).toBe(25);
    });

    it('supports formulas mixing A1 and R1C1 styles', () => {
        // B2 = 10, R3C3 = 15
        expect(evaluateTableFormula('=B2 + R3C3', 0, 0, table)).toBe(25);
        // C2 = 20, R[1]C[0] from (1,1) = (2,1) = 5
        expect(evaluateTableFormula('=C2 + R[1]C[0]', 1, 1, table)).toBe(25);
    });

    it('evaluates equivalent A1 and R1C1 formulas to the same value', () => {
        const r1c1 = evaluateTableFormula('=R2C2 + R2C3 + R3C2', 0, 0, table);
        const a1 = evaluateTableFormula('=B2 + C2 + B3', 0, 0, table);
        expect(r1c1).toBe(35);
        expect(a1).toBe(35);
        expect(a1).toBe(r1c1);
    });

    it('returns #ERR for invalid characters', () => {
        expect(evaluateTableFormula('=alert(1)', 0, 0, table)).toBe('#ERR');
        expect(evaluateTableFormula('=foo', 0, 0, table)).toBe('#ERR');
    });

    it('handles non-numeric cell values as 0', () => {
        // From cell (1,1)="10": R[0]C[-1] = col0,row1 = "Row1" (non-numeric) = 0
        expect(evaluateTableFormula('=R[0]C[-1] + 5', 1, 1, table)).toBe(5);
    });
});

describe('parseTableFunctionCall', () => {
    it('parses numbers, quoted strings, and cell references', () => {
        const parsed = parseTableFunctionCall('=MyFn(2, "hello world", A1, R[1]C[-1], -3.5)');
        expect(parsed).toEqual({
            fnName: 'MyFn',
            args: ['2', '"hello world"', 'A1', 'R[1]C[-1]', '-3.5'],
        });
    });

    it('ignores spaces outside quotes', () => {
        const parsed = parseTableFunctionCall('=Trim(  1  ,  "a b c"  ,  B2  )');
        expect(parsed).toEqual({
            fnName: 'Trim',
            args: ['1', '"a b c"', 'B2'],
        });
    });
});

describe('resolveTableFunctionArg', () => {
    const rows = [
        ['Head', 'A', 'B'],
        ['R1', '10', 'hello'],
        ['R2', '5', '20'],
    ];

    it('resolves A1-style references to cell values', () => {
        expect(resolveTableFunctionArg('B2', 1, 1, rows)).toBe('10');
        expect(resolveTableFunctionArg('C2', 1, 1, rows)).toBe('hello');
    });

    it('resolves R1C1-style references to cell values', () => {
        expect(resolveTableFunctionArg('R2C2', 0, 0, rows)).toBe('10');
        expect(resolveTableFunctionArg('R[1]C[1]', 0, 0, rows)).toBe('10');
    });

    it('preserves quoted strings and numeric literals', () => {
        expect(resolveTableFunctionArg('"hello world"', 0, 0, rows)).toBe('hello world');
        expect(resolveTableFunctionArg('-12.5', 0, 0, rows)).toBe('-12.5');
    });
});

describe('resolveTableFunctionArgs', () => {
    const rows = [
        ['Name', 'A', 'B', 'C'],
        ['Row1', '10', '20', '30'],
        ['Row2', '5', '15', '25'],
        ['Row3', '2', '4', '6'],
    ];

    it('expands A1 ranges into row-major parameter lists', () => {
        expect(resolveTableFunctionArgs('B2:C3', 0, 0, rows)).toEqual(['10', '20', '5', '15']);
    });

    it('expands entire columns into parameter lists', () => {
        expect(resolveTableFunctionArgs('B:B', 0, 0, rows)).toEqual(['A', '10', '5', '2']);
    });

    it('expands entire rows into parameter lists', () => {
        expect(resolveTableFunctionArgs('2:3', 0, 0, rows)).toEqual(['Row1', '10', '20', '30', 'Row2', '5', '15', '25']);
    });
});

describe('resolveTableFunctionArgsAsync', () => {
    const rows = [
        ['Name', 'A', 'B', 'C'],
        ['Row1', '10', '20', '30'],
        ['Row2', '5', '15', '25'],
        ['Row3', '2', '4', '6'],
    ];

    it('resolves scalar values without executor', async () => {
        const mockExecutor = async () => '#ERR';
        const result = await resolveTableFunctionArgsAsync(mockExecutor, 'A1', 0, 0, rows);
        expect(result).toEqual(['Name']);
    });

    it('detects and executes nested function calls', async () => {
        let executorCalled = false;
        const mockExecutor = async (fnName, fnArgs) => {
            executorCalled = true;
            expect(fnName).toBe('sum');
            expect(fnArgs).toEqual(['B2:B4']);
            return '25'; // 10 + 5 + 2 + 8 = 25 (or whatever the nested function returns)
        };

        const result = await resolveTableFunctionArgsAsync(mockExecutor, 'sum(B2:B4)', 0, 0, rows);
        expect(result).toEqual(['25']);
        expect(executorCalled).toBe(true);
    });

    it('expands ranges when executor is not needed', async () => {
        const mockExecutor = async () => '#ERR';
        const result = await resolveTableFunctionArgsAsync(mockExecutor, 'A2:B3', 0, 0, rows);
        expect(result).toEqual(['Row1', '10', 'Row2', '5']);
    });
});

