import { describe, it, expect, beforeEach } from 'vitest';
import { attachVimMode } from './vim-mode.js';

// ─── helpers ─────────────────────────────────────────────────────────────────

/** Create a fresh textarea, attach vim-mode, and return { ta, vim }. */
function setup(initialValue = '', initialPos = 0) {
    document.body.innerHTML = '<textarea id="ta"></textarea>';
    const ta = document.getElementById('ta');
    ta.value = initialValue;
    ta.selectionStart = ta.selectionEnd = initialPos;
    const vim = attachVimMode(ta);
    return { ta, vim };
}

/** Fire a keydown event on the textarea. Returns the event object. */
function key(ta, keyName, opts = {}) {
    const ev = new KeyboardEvent('keydown', {
        key: keyName,
        bubbles: true,
        cancelable: true,
        ...opts,
    });
    ta.dispatchEvent(ev);
    return ev;
}

/** Enter NORMAL mode by pressing Escape. */
function toNormal(ta, vim) {
    key(ta, 'Escape');
    expect(vim.getMode()).toBe('normal');
}

// ─── mode transitions ─────────────────────────────────────────────────────────

describe('vim-mode — mode transitions', () => {
    it('starts in insert mode', () => {
        const { vim } = setup('hello');
        expect(vim.getMode()).toBe('insert');
    });

    it('Escape → normal', () => {
        const { ta, vim } = setup('hello');
        key(ta, 'Escape');
        expect(vim.getMode()).toBe('normal');
    });

    it('Escape in normal clears pending state but stays normal', () => {
        const { ta, vim } = setup('hello');
        toNormal(ta, vim);
        key(ta, 'd');            // pending operator
        key(ta, 'Escape');       // cancel
        expect(vim.getMode()).toBe('normal');
        // d+Escape should NOT delete anything
        expect(ta.value).toBe('hello');
    });

    it('i returns to insert from normal', () => {
        const { ta, vim } = setup('hello');
        toNormal(ta, vim);
        key(ta, 'i');
        expect(vim.getMode()).toBe('insert');
    });

    it('a returns to insert from normal', () => {
        const { ta, vim } = setup('hello');
        toNormal(ta, vim);
        key(ta, 'a');
        expect(vim.getMode()).toBe('insert');
    });

    it('I returns to insert and moves to first non-blank', () => {
        const { ta, vim } = setup('  hello', 6);
        toNormal(ta, vim);
        key(ta, 'I');
        expect(vim.getMode()).toBe('insert');
        expect(ta.selectionStart).toBe(2); // first non-blank
    });

    it('A returns to insert and moves to end of line', () => {
        const { ta, vim } = setup('hello\nworld', 0);
        toNormal(ta, vim);
        key(ta, 'A');
        expect(vim.getMode()).toBe('insert');
        expect(ta.selectionStart).toBe(5); // end of 'hello'
    });

    it('o opens a line below and enters insert', () => {
        const { ta, vim } = setup('hello\nworld', 0);
        toNormal(ta, vim);
        key(ta, 'o');
        expect(vim.getMode()).toBe('insert');
        expect(ta.value).toBe('hello\n\nworld');
        expect(ta.selectionStart).toBe(6);
    });

    it('O opens a line above and enters insert', () => {
        const { ta, vim } = setup('hello\nworld', 6);
        toNormal(ta, vim);
        key(ta, 'O');
        expect(vim.getMode()).toBe('insert');
        expect(ta.value).toBe('hello\n\nworld');
        expect(ta.selectionStart).toBe(6);
    });

    it('R enters replace mode', () => {
        const { ta, vim } = setup('hello');
        toNormal(ta, vim);
        key(ta, 'R');
        expect(vim.getMode()).toBe('replace');
    });

    it('Escape exits replace mode → normal', () => {
        const { ta, vim } = setup('hello');
        toNormal(ta, vim);
        key(ta, 'R');
        key(ta, 'Escape');
        expect(vim.getMode()).toBe('normal');
    });

    it('r enters replace-once mode', () => {
        const { ta, vim } = setup('hello');
        toNormal(ta, vim);
        key(ta, 'r');
        expect(vim.getMode()).toBe('replace-once');
    });

    it('Escape exits replace-once → normal', () => {
        const { ta, vim } = setup('hello');
        toNormal(ta, vim);
        key(ta, 'r');
        key(ta, 'Escape');
        expect(vim.getMode()).toBe('normal');
    });

    it('Ctrl+key does not change mode in insert', () => {
        const { ta, vim } = setup('hello');
        key(ta, 's', { ctrlKey: true });
        expect(vim.getMode()).toBe('insert');
    });

    it('Ctrl+key does not change mode in normal', () => {
        const { ta, vim } = setup('hello');
        toNormal(ta, vim);
        key(ta, 's', { ctrlKey: true });
        expect(vim.getMode()).toBe('normal');
        expect(ta.value).toBe('hello');
    });
});

// ─── replace mode ─────────────────────────────────────────────────────────────

describe('vim-mode — replace mode (R)', () => {
    it('overwrites characters in sequence', () => {
        const { ta, vim } = setup('hello world', 0);
        toNormal(ta, vim);
        key(ta, 'R');
        key(ta, 'H');
        key(ta, 'E');
        key(ta, 'L');
        expect(ta.value).toBe('HELlo world');
        expect(vim.getMode()).toBe('replace');
    });

    it('inserts at end-of-line when replacing', () => {
        const { ta, vim } = setup('hi', 2); // cursor after last char
        toNormal(ta, vim);
        // back up to position 1 (last valid NORMAL pos)
        expect(ta.selectionStart).toBe(1);
        key(ta, 'R');
        key(ta, 'X'); // replaces 'i'
        key(ta, 'Y'); // at EOL — should insert
        expect(ta.value).toBe('hXY');
    });

    it('Backspace moves cursor back in replace mode', () => {
        const { ta, vim } = setup('hello', 2);
        toNormal(ta, vim);
        key(ta, 'R');
        key(ta, 'Backspace');
        expect(ta.selectionStart).toBe(1);
    });
});

// ─── replace-once mode (r) ────────────────────────────────────────────────────

describe('vim-mode — replace-once (r)', () => {
    it('replaces single char and returns to normal', () => {
        const { ta, vim } = setup('hello', 0);
        toNormal(ta, vim);
        key(ta, 'r');
        key(ta, 'H');
        expect(ta.value).toBe('Hello');
        expect(ta.selectionStart).toBe(0);
        expect(vim.getMode()).toBe('normal');
    });
});

// ─── cursor motions ───────────────────────────────────────────────────────────

describe('vim-mode — h/l left/right', () => {
    it('l moves right', () => {
        const { ta, vim } = setup('hello', 0);
        toNormal(ta, vim);
        key(ta, 'l');
        expect(ta.selectionStart).toBe(1);
    });

    it('h moves left', () => {
        const { ta, vim } = setup('hello', 2);
        toNormal(ta, vim);
        key(ta, 'h');
        expect(ta.selectionStart).toBe(1);
    });

    it('l does not move past last char', () => {
        const { ta, vim } = setup('hi', 0);
        toNormal(ta, vim);
        key(ta, 'l'); key(ta, 'l'); key(ta, 'l');
        expect(ta.selectionStart).toBe(1); // 'i' is the last char
    });

    it('h does not move before col 0', () => {
        const { ta, vim } = setup('hi', 0);
        toNormal(ta, vim);
        key(ta, 'h'); key(ta, 'h');
        expect(ta.selectionStart).toBe(0);
    });

    it('ArrowLeft / ArrowRight work same as h/l', () => {
        const { ta, vim } = setup('hello', 0);
        toNormal(ta, vim);
        key(ta, 'ArrowRight');
        expect(ta.selectionStart).toBe(1);
        key(ta, 'ArrowLeft');
        expect(ta.selectionStart).toBe(0);
    });

    it('count prefix: 3l moves 3 right', () => {
        const { ta, vim } = setup('hello world', 0);
        toNormal(ta, vim);
        key(ta, '3'); key(ta, 'l');
        expect(ta.selectionStart).toBe(3);
    });
});

describe('vim-mode — j/k up/down', () => {
    it('j moves down one line', () => {
        const { ta, vim } = setup('hello\nworld', 0);
        toNormal(ta, vim);
        key(ta, 'j');
        expect(ta.selectionStart).toBe(6); // 'w' of 'world'
    });

    it('k moves up one line', () => {
        const { ta, vim } = setup('hello\nworld', 6);
        toNormal(ta, vim);
        key(ta, 'k');
        expect(ta.selectionStart).toBe(0);
    });

    it('j on last line stays on last line', () => {
        const { ta, vim } = setup('hello', 0);
        toNormal(ta, vim);
        key(ta, 'j');
        expect(ta.selectionStart).toBe(0);
    });

    it('k on first line stays on first line', () => {
        const { ta, vim } = setup('hello', 2);
        toNormal(ta, vim);
        key(ta, 'k');
        expect(ta.selectionStart).toBe(2);
    });

    it('j preserves column when moving to shorter line', () => {
        // 'hello\nhi\nworld'
        // cursor at col 4 of 'hello', j moves to end of 'hi' (col 1, max for normal)
        const { ta, vim } = setup('hello\nhi\nworld', 4);
        toNormal(ta, vim);
        key(ta, 'j');
        expect(ta.selectionStart).toBe(7); // 'i' of 'hi'
    });

    it('ArrowDown / ArrowUp work same as j/k', () => {
        const { ta, vim } = setup('hello\nworld', 0);
        toNormal(ta, vim);
        key(ta, 'ArrowDown');
        expect(ta.selectionStart).toBe(6);
        key(ta, 'ArrowUp');
        expect(ta.selectionStart).toBe(0);
    });

    it('count prefix: 2j moves 2 lines down', () => {
        const { ta, vim } = setup('a\nb\nc\nd', 0);
        toNormal(ta, vim);
        key(ta, '2'); key(ta, 'j');
        expect(ta.selectionStart).toBe(4); // 'c'
    });
});

describe('vim-mode — word motions w/b/e', () => {
    it('w moves to start of next word', () => {
        const { ta, vim } = setup('hello world', 0);
        toNormal(ta, vim);
        key(ta, 'w');
        expect(ta.selectionStart).toBe(6); // 'w' of 'world'
    });

    it('b moves to start of previous word', () => {
        const { ta, vim } = setup('hello world', 6);
        toNormal(ta, vim);
        key(ta, 'b');
        expect(ta.selectionStart).toBe(0);
    });

    it('e moves to end of current/next word', () => {
        const { ta, vim } = setup('hello world', 0);
        toNormal(ta, vim);
        key(ta, 'e');
        expect(ta.selectionStart).toBe(4); // last 'o' of 'hello'
    });

    it('count prefix: 2w skips two words', () => {
        const { ta, vim } = setup('one two three', 0);
        toNormal(ta, vim);
        key(ta, '2'); key(ta, 'w');
        expect(ta.selectionStart).toBe(8); // 't' of 'three'
    });
});

describe('vim-mode — WORD motions W/B/E', () => {
    // W treats any run of non-whitespace as one WORD, regardless of punctuation.
    it('W moves to start of next WORD', () => {
        const { ta, vim } = setup('foo.bar baz', 0);
        toNormal(ta, vim);
        key(ta, 'W');
        // w would stop at the dot; W skips the whole 'foo.bar' token
        expect(ta.selectionStart).toBe(8); // 'b' of 'baz'
    });

    it('W moves across multiple whitespace chars', () => {
        const { ta, vim } = setup('hello   world', 0);
        toNormal(ta, vim);
        key(ta, 'W');
        expect(ta.selectionStart).toBe(8); // 'w' of 'world'
    });

    it('W respects count prefix', () => {
        const { ta, vim } = setup('one two three', 0);
        toNormal(ta, vim);
        key(ta, '2'); key(ta, 'W');
        expect(ta.selectionStart).toBe(8); // 't' of 'three'
    });

    it('B moves to start of previous WORD', () => {
        const { ta, vim } = setup('foo.bar baz', 8);
        toNormal(ta, vim);
        key(ta, 'B');
        // B skips back past 'baz' AND the whole 'foo.bar' WORD
        expect(ta.selectionStart).toBe(0);
    });

    it('B from middle of WORD goes to WORD start', () => {
        const { ta, vim } = setup('hello world', 8); // inside 'world'
        toNormal(ta, vim);
        key(ta, 'B');
        expect(ta.selectionStart).toBe(6); // 'w' of 'world'
    });

    it('B respects count prefix', () => {
        const { ta, vim } = setup('one two three', 8);
        toNormal(ta, vim);
        key(ta, '2'); key(ta, 'B');
        expect(ta.selectionStart).toBe(0);
    });

    it('E moves to end of current WORD', () => {
        const { ta, vim } = setup('foo.bar baz', 0);
        toNormal(ta, vim);
        key(ta, 'E');
        // e would stop at 'foo'; E goes to end of 'foo.bar'
        expect(ta.selectionStart).toBe(6); // 'r' of 'foo.bar'
    });

    it('E respects count prefix', () => {
        const { ta, vim } = setup('one two three', 0);
        toNormal(ta, vim);
        key(ta, '2'); key(ta, 'E');
        expect(ta.selectionStart).toBe(6); // 'o' of 'two'
    });

    it('dW deletes a whole WORD token', () => {
        const { ta, vim } = setup('foo.bar baz', 0);
        toNormal(ta, vim);
        key(ta, 'd'); key(ta, 'W');
        expect(ta.value).toBe('baz');
        expect(vim.getMode()).toBe('normal');
    });
});

describe('vim-mode — line motions 0 ^ $ gg G', () => {
    it('0 moves to column 0', () => {
        const { ta, vim } = setup('  hello', 4);
        toNormal(ta, vim);
        key(ta, '0');
        expect(ta.selectionStart).toBe(0);
    });

    it('^ moves to first non-blank', () => {
        const { ta, vim } = setup('  hello', 4);
        toNormal(ta, vim);
        key(ta, '^');
        expect(ta.selectionStart).toBe(2);
    });

    it('$ moves to last char of line', () => {
        const { ta, vim } = setup('hello\nworld', 0);
        toNormal(ta, vim);
        key(ta, '$');
        expect(ta.selectionStart).toBe(4); // last 'o' of 'hello'
    });

    it('End works same as $', () => {
        const { ta, vim } = setup('hello\nworld', 0);
        toNormal(ta, vim);
        key(ta, 'End');
        expect(ta.selectionStart).toBe(4);
    });

    it('g moves to first non-blank of file', () => {
        const { ta, vim } = setup('  hello\nworld', 8);
        toNormal(ta, vim);
        key(ta, 'g');
        expect(ta.selectionStart).toBe(2); // first non-blank
    });

    it('G moves to last line', () => {
        const { ta, vim } = setup('hello\nworld', 0);
        toNormal(ta, vim);
        key(ta, 'G');
        expect(ta.selectionStart).toBe(6); // 'w' of 'world'
    });
});

describe('vim-mode — % bracket matching', () => {
    it('% jumps from ( to )', () => {
        const { ta, vim } = setup('fn(arg)', 2); // cursor on '('
        toNormal(ta, vim);
        key(ta, '%');
        expect(ta.selectionStart).toBe(6); // ')'
    });

    it('% jumps from ) to (', () => {
        const { ta, vim } = setup('fn(arg)', 6); // cursor on ')'
        toNormal(ta, vim);
        key(ta, '%');
        expect(ta.selectionStart).toBe(2); // '('
    });

    it('% stays put when no bracket at cursor', () => {
        const { ta, vim } = setup('hello', 0);
        toNormal(ta, vim);
        key(ta, '%');
        expect(ta.selectionStart).toBe(0);
    });
});

// ─── delete operators ─────────────────────────────────────────────────────────

describe('vim-mode — x / X', () => {
    it('x deletes char under cursor', () => {
        const { ta, vim } = setup('hello', 1);
        toNormal(ta, vim);
        key(ta, 'x');
        expect(ta.value).toBe('hllo');
        expect(ta.selectionStart).toBe(1);
        expect(vim.getMode()).toBe('normal');
    });

    it('X deletes char before cursor', () => {
        const { ta, vim } = setup('hello', 2);
        toNormal(ta, vim);
        key(ta, 'X');
        expect(ta.value).toBe('hllo');
        expect(ta.selectionStart).toBe(1);
    });

    it('count prefix: 3x deletes three chars', () => {
        const { ta, vim } = setup('hello world', 0);
        toNormal(ta, vim);
        key(ta, '3'); key(ta, 'x');
        expect(ta.value).toBe('lo world');
    });

    it('x does not delete the newline at end of line', () => {
        // 'h\nworld': 'h' is the only char on line 0. After x deletes 'h',
        // line 0 is empty and the cursor sits on the newline — x must not delete it.
        const { ta, vim } = setup('h\nworld', 0);
        toNormal(ta, vim);
        key(ta, 'x'); // deletes 'h'
        expect(ta.value).toBe('\nworld');
        key(ta, 'x'); // cursor is on '\n' (col 0, lineLen 0) — must be a no-op
        expect(ta.value).toBe('\nworld');
    });
});

describe('vim-mode — D (delete to EOL)', () => {
    it('D deletes from cursor to end of line', () => {
        const { ta, vim } = setup('hello world\nnext', 5);
        toNormal(ta, vim);
        key(ta, 'D');
        expect(ta.value).toBe('hello\nnext');
        expect(vim.getMode()).toBe('normal');
    });
});

describe('vim-mode — dd (delete line)', () => {
    it('dd deletes current middle line', () => {
        const { ta, vim } = setup('first\nsecond\nthird', 6);
        toNormal(ta, vim);
        key(ta, 'd'); key(ta, 'd');
        expect(ta.value).toBe('first\nthird');
        expect(vim.getMode()).toBe('normal');
    });

    it('dd deletes last line and removes preceding newline', () => {
        const { ta, vim } = setup('first\nsecond', 6);
        toNormal(ta, vim);
        key(ta, 'd'); key(ta, 'd');
        expect(ta.value).toBe('first');
    });

    it('dd on only line clears content', () => {
        const { ta, vim } = setup('hello', 0);
        toNormal(ta, vim);
        key(ta, 'd'); key(ta, 'd');
        expect(ta.value).toBe('');
    });
});

describe('vim-mode — dw (delete word)', () => {
    it('dw deletes from cursor to start of next word', () => {
        const { ta, vim } = setup('hello world', 0);
        toNormal(ta, vim);
        key(ta, 'd'); key(ta, 'w');
        expect(ta.value).toBe('world');
        expect(vim.getMode()).toBe('normal');
    });
});

// ─── change operator ──────────────────────────────────────────────────────────

describe('vim-mode — c (change)', () => {
    it('cw deletes word and enters insert', () => {
        const { ta, vim } = setup('hello world', 0);
        toNormal(ta, vim);
        key(ta, 'c'); key(ta, 'w');
        expect(ta.value).toBe('world');
        expect(vim.getMode()).toBe('insert');
    });

    it('cc deletes line body and enters insert', () => {
        const { ta, vim } = setup('first\nsecond\nthird', 6);
        toNormal(ta, vim);
        key(ta, 'c'); key(ta, 'c');
        expect(ta.value).toBe('first\nthird');
        expect(vim.getMode()).toBe('insert');
    });

    it('C deletes to EOL and enters insert', () => {
        const { ta, vim } = setup('hello world', 5);
        toNormal(ta, vim);
        key(ta, 'C');
        expect(ta.value).toBe('hello');
        expect(vim.getMode()).toBe('insert');
    });
});

// ─── yank and paste ───────────────────────────────────────────────────────────

describe('vim-mode — yank and paste', () => {
    it('yw yanks a word and p pastes it after cursor', () => {
        const { ta, vim } = setup('hello world', 0);
        toNormal(ta, vim);
        key(ta, 'y'); key(ta, 'w'); // yank 'hello '
        // cursor stays at 0; paste after cursor
        key(ta, 'p');
        expect(ta.value).toContain('hello');
        expect(vim.getMode()).toBe('normal');
    });

    it('yy yanks full line and p pastes it below', () => {
        const { ta, vim } = setup('first\nsecond', 0);
        toNormal(ta, vim);
        key(ta, 'y'); key(ta, 'y'); // yank 'first\n'
        key(ta, 'p');
        expect(ta.value).toBe('first\nfirst\nsecond');
    });

    it('Y is alias for yy', () => {
        const { ta, vim } = setup('hello\nworld', 0);
        toNormal(ta, vim);
        key(ta, 'Y');
        key(ta, 'p');
        expect(ta.value).toBe('hello\nhello\nworld');
    });

    it('P pastes before cursor', () => {
        const { ta, vim } = setup('hello world', 6); // cursor at 'w'
        toNormal(ta, vim);
        key(ta, 'y'); key(ta, 'w'); // yank 'world'
        // move back to start of 'hello'
        key(ta, '0');
        key(ta, 'P');
        // 'world' pasted before 'hello'
        expect(ta.value).toContain('world');
        expect(vim.getMode()).toBe('normal');
    });

    it('x then p re-inserts the deleted char', () => {
        const { ta, vim } = setup('abc', 0);
        toNormal(ta, vim);
        key(ta, 'x'); // delete 'a', yankBuf = 'a'
        key(ta, 'p'); // paste 'a' after 'b'
        expect(ta.value).toBe('bac');
    });
});

// ─── indicator ───────────────────────────────────────────────────────────────

describe('vim-mode — indicator element', () => {
    it('indicator is appended to document.body', () => {
        setup('hello');
        const indicator = document.querySelector('.vim-mode-indicator');
        expect(indicator).not.toBeNull();
        expect(document.body.contains(indicator)).toBe(true);
    });

    it('indicator is hidden in insert mode', () => {
        setup('hello');
        const indicator = document.querySelector('.vim-mode-indicator');
        expect(indicator.style.opacity).toBe('0');
    });

    it('indicator is visible in normal mode', () => {
        const { ta } = setup('hello');
        key(ta, 'Escape');
        const indicator = document.querySelector('.vim-mode-indicator');
        expect(indicator.style.opacity).toBe('1');
        expect(indicator.textContent).toBe('-- VIM KEYS --');
    });

    it('indicator shows REPLACE in replace mode', () => {
        const { ta } = setup('hello');
        key(ta, 'Escape');
        key(ta, 'R');
        const indicator = document.querySelector('.vim-mode-indicator');
        expect(indicator.textContent).toBe('-- REPLACE --');
    });

    it('indicator shows REPLACE (r) in replace-once mode', () => {
        const { ta } = setup('hello');
        key(ta, 'Escape');
        key(ta, 'r');
        const indicator = document.querySelector('.vim-mode-indicator');
        expect(indicator.textContent).toBe('-- REPLACE (once) --');
    });

    it('indicator is hidden again after returning to insert', () => {
        const { ta } = setup('hello');
        key(ta, 'Escape');
        key(ta, 'i');
        const indicator = document.querySelector('.vim-mode-indicator');
        expect(indicator.style.opacity).toBe('0');
    });

    it('detach removes the indicator from the DOM', () => {
        const { vim } = setup('hello');
        const indicator = document.querySelector('.vim-mode-indicator');
        expect(indicator).not.toBeNull();
        vim.detach();
        expect(document.querySelector('.vim-mode-indicator')).toBeNull();
    });
});

// ─── event prevention ────────────────────────────────────────────────────────

describe('vim-mode — event prevention', () => {
    it('Escape in normal mode is prevented', () => {
        const { ta, vim } = setup('hello');
        toNormal(ta, vim);
        const ev = key(ta, 'Escape');
        expect(ev.defaultPrevented).toBe(true);
    });

    it('motion keys in normal mode are prevented', () => {
        const { ta, vim } = setup('hello');
        toNormal(ta, vim);
        const ev = key(ta, 'h');
        expect(ev.defaultPrevented).toBe(true);
    });

    it('Escape in insert mode is prevented', () => {
        const { ta } = setup('hello');
        const ev = key(ta, 'Escape');
        expect(ev.defaultPrevented).toBe(true);
    });

    it('regular keys in insert mode are NOT prevented', () => {
        const { ta } = setup('hello');
        const ev = key(ta, 'a');
        expect(ev.defaultPrevented).toBe(false);
    });

    it('Ctrl+key is never prevented in any mode', () => {
        const { ta, vim } = setup('hello');
        // insert
        let ev = key(ta, 's', { ctrlKey: true });
        expect(ev.defaultPrevented).toBe(false);
        // normal
        toNormal(ta, vim);
        ev = key(ta, 's', { ctrlKey: true });
        expect(ev.defaultPrevented).toBe(false);
    });
});

// ─── detach ───────────────────────────────────────────────────────────────────

describe('vim-mode — detach', () => {
    it('after detach, Escape no longer changes mode', () => {
        const { ta, vim } = setup('hello');
        vim.detach();
        const ev = key(ta, 'Escape');
        expect(ev.defaultPrevented).toBe(false);
        // getMode still returns last known mode
        expect(vim.getMode()).toBe('insert');
    });

    it('after detach, normal-mode keys do not affect content', () => {
        const { ta, vim } = setup('hello');
        toNormal(ta, vim);
        vim.detach();
        // 'x' would delete a char in normal; after detach it should be a no-op
        key(ta, 'x');
        expect(ta.value).toBe('hello');
    });
});

// ─── count prefix ─────────────────────────────────────────────────────────────

describe('vim-mode — count prefix', () => {
    it('counts up to two digits', () => {
        const { ta, vim } = setup('a b c d e f g h i j k l', 0);
        toNormal(ta, vim);
        key(ta, '1'); key(ta, '0'); key(ta, 'l');
        // 10 chars right (each char is separated by space: a=0,space=1,b=2,...l=20)
        expect(ta.selectionStart).toBe(10);
    });

    it('count resets after a motion', () => {
        const { ta, vim } = setup('hello world', 0);
        toNormal(ta, vim);
        key(ta, '3'); key(ta, 'l'); // move 3 right
        key(ta, 'l');               // plain l — should move 1
        expect(ta.selectionStart).toBe(4);
    });

    it('2dd deletes two lines', () => {
        const { ta, vim } = setup('a\nb\nc\nd', 0);
        toNormal(ta, vim);
        key(ta, '2'); key(ta, 'd'); key(ta, 'd');
        expect(ta.value).toBe('c\nd');
    });
});

// ─── input events ─────────────────────────────────────────────────────────────

describe('vim-mode — input events fired on mutations', () => {
    it('x fires an input event', () => {
        const { ta, vim } = setup('hello', 0);
        toNormal(ta, vim);
        let fired = 0;
        ta.addEventListener('input', () => fired++);
        key(ta, 'x');
        expect(fired).toBeGreaterThan(0);
    });

    it('r{char} fires an input event', () => {
        const { ta, vim } = setup('hello', 0);
        toNormal(ta, vim);
        let fired = 0;
        ta.addEventListener('input', () => fired++);
        key(ta, 'r');
        key(ta, 'H');
        expect(fired).toBeGreaterThan(0);
    });
});
