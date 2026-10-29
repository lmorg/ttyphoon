import { describe, expect, it, vi } from 'vitest';

import { initNotesGitPanel } from './notes-git-panel.js';

function createElements() {
    const gitTab = document.createElement('button');
    gitTab.style.display = 'none';
    const gitPane = document.createElement('div');
    const gitToolbar = document.createElement('div');
    gitToolbar.hidden = true;
    const gitWorkspace = document.createElement('button');
    const gitCompare = document.createElement('button');
    const gitCommits = document.createElement('button');
    const gitCommitToggle = document.createElement('button');
    const gitCommitRow = document.createElement('div');
    const gitFilterRow = document.createElement('div');
    gitFilterRow.hidden = true;
    const gitFileFilter = document.createElement('input');
    gitFileFilter.type = 'search';
    gitFilterRow.appendChild(gitFileFilter);
    const gitCommitMessage = document.createElement('input');
    const gitCommitSubmit = document.createElement('button');
    const gitPush = document.createElement('button');
    const gitStageAll = document.createElement('button');
    const gitRemote = document.createElement('button');
    const gitBranch = document.createElement('span');
    const gitAheadBehind = document.createElement('span');
    const gitRefresh = document.createElement('button');
    const gitFileList = document.createElement('div');
    const gitCommitList = document.createElement('div');
    const gitFilenameSearchRow = document.createElement('label');
    gitFilenameSearchRow.hidden = true;
    const gitFilenameSearch = document.createElement('input');
    gitFilenameSearch.type = 'checkbox';
    gitFilenameSearchRow.appendChild(gitFilenameSearch);
    gitToolbar.append(gitWorkspace, gitCompare, gitCommits, gitCommitToggle, gitPush, gitRemote, gitBranch, gitAheadBehind, gitRefresh);
    gitCommitRow.append(gitCommitMessage, gitCommitSubmit, gitStageAll);
    gitPane.append(gitToolbar, gitFilterRow, gitCommitRow, gitFileList);
    return {
        gitTab,
        gitPane,
        gitToolbar,
        gitWorkspace,
        gitCompare,
        gitCommits,
        gitCommitToggle,
        gitCommitRow,
        gitFilterRow,
        gitFileFilter,
        gitCommitMessage,
        gitCommitSubmit,
        gitPush,
        gitStageAll,
        gitRemote,
        gitBranch,
        gitAheadBehind,
        gitRefresh,
        gitFileList,
        gitCommitList,
        gitFilenameSearchRow,
        gitFilenameSearch,
    };
}

async function flushPromises() {
    await new Promise((resolve) => setTimeout(resolve, 0));
    await new Promise((resolve) => setTimeout(resolve, 0));
}

function createApi(overrides = {}) {
    return {
        getCurrentProject: vi.fn(() => Promise.resolve('/work/repo')),
        filterStrings: vi.fn((query, items) => Promise.resolve({
            List: items.filter((item) => item.toLowerCase().includes(query.toLowerCase())),
        })),
        listCommits: vi.fn(() => Promise.resolve([
            { hash: 'abcdef1', subject: 'Merge feature', date: '2026-10-12 12:30:00', author: 'Test User', graph: ['*   ', '|\\  '] },
            { hash: 'abcdef2', subject: 'Feature commit', date: '2026-10-12 12:00:00', author: 'Test User', graph: ['| * ', '|/  '] },
        ])),
        listCommitsWithFiles: vi.fn(() => Promise.resolve([])),
        getBrowserLink: vi.fn(() => Promise.resolve({
            provider: 'github',
            url: 'https://github.com/org/repo/tree/main',
        })),
        getBranchDiffFiles: vi.fn(() => Promise.resolve([
            { path: 'src/main.go', code: ' M', staged: false, unstaged: true, compareOnly: true },
        ])),
        openRemote: vi.fn(() => Promise.resolve()),
        getStatus: vi.fn(() => Promise.resolve({
            root: '/work/repo',
            branch: 'main',
            ahead: 2,
            behind: 1,
            clean: false,
            files: [
                { path: 'src/main.go', code: ' M', staged: false, unstaged: true },
                { path: 'README.md', code: 'A ', staged: true, unstaged: false },
                { path: 'new-file.txt', code: '??', staged: false, unstaged: false, untracked: true },
            ],
        })),
        refreshStatus: vi.fn(() => Promise.resolve({
            root: '/work/repo', branch: 'main', ahead: 2, behind: 1, clean: false,
            files: [
                { path: 'src/main.go', code: ' M', staged: false, unstaged: true },
                { path: 'README.md', code: 'A ', staged: true, unstaged: false },
                { path: 'new-file.txt', code: '??', staged: false, unstaged: false, untracked: true },
            ],
        })),
        getDiff: vi.fn(() => Promise.resolve({ path: 'src/main.go', original: 'old', working: 'new' })),
        getDiffAgainstBranch: vi.fn(() => Promise.resolve({ path: 'src/main.go', original: 'branch', working: 'new' })),
        stageFile: vi.fn(() => Promise.resolve()),
        stageAll: vi.fn(() => Promise.resolve()),
        unstageFile: vi.fn(() => Promise.resolve()),
        runGitAction: vi.fn(() => Promise.resolve({ mode: 'interactive', stdout: 'commit output', stderr: '', error: '' })),
        confirmGitPush: vi.fn(() => Promise.resolve({ cancelled: 'true' })),
        eventsOn: vi.fn(),
        ...overrides,
    };
}

describe('Notes Git panel', () => {
    it.each(['read', 'open', 'comparison'])('notifies diff %s errors without replacing the Git file list', async (failure) => {
        const elements = createElements();
        const notify = vi.fn();
        const api = createApi();
        const onOpenDiff = vi.fn(() => Promise.resolve());
        if (failure === 'read') api.getDiff.mockRejectedValueOnce(new Error('Diff failed'));
        else if (failure === 'comparison') api.getDiffAgainstBranch.mockRejectedValueOnce(new Error('Diff failed'));
        else onOpenDiff.mockRejectedValueOnce(new Error('Diff failed'));
        const panel = initNotesGitPanel(elements, api, { notify, onOpenDiff });
        await flushPromises();
        if (failure === 'comparison') await panel.setCompareBranch('feature');
        const button = elements.gitFileList.querySelector('.notes-git-file');
        const originalList = elements.gitFileList.innerHTML;
        button.click();
        await flushPromises();
        expect(notify).toHaveBeenCalledWith('Diff failed', 'error');
        expect(elements.gitFileList.innerHTML).toBe(originalList);
        expect(button.disabled).toBe(false);
    });

    it.each(['stage', 'unstage', 'stageAll'])('notifies %s errors without replacing the Git file list', async (action) => {
        const elements = createElements();
        const notify = vi.fn();
        const api = createApi();
        const method = { stage: 'stageFile', unstage: 'unstageFile', stageAll: 'stageAll' }[action];
        api[method].mockRejectedValueOnce(new Error('Git index is locked'));
        initNotesGitPanel(elements, api, { notify });
        await flushPromises();
        const button = action === 'stageAll' ? elements.gitStageAll : elements.gitFileList.querySelector(`[data-git-action="${action}"]`);
        const originalList = elements.gitFileList.innerHTML;
        button.click();
        await flushPromises();
        expect(notify).toHaveBeenCalledWith('Git index is locked', 'error');
        expect(elements.gitFileList.innerHTML).toBe(originalList);
        expect(button.disabled).toBe(false);
    });

    it('sorts untracked, unstaged, then staged files alphabetically within each group', async () => {
        const elements = createElements();
        const files = Object.freeze([
            { path: 'z-staged.txt', staged: true },
            { path: 'z-work.txt', unstaged: true },
            { path: 'z-new.txt', untracked: true },
            { path: 'a-staged.txt', staged: true },
            { path: 'm-mixed.txt', staged: true, unstaged: true },
            { path: 'a-new.txt', untracked: true },
            { path: 'a-work.txt', unstaged: true },
        ]);
        const api = createApi({
            getStatus: vi.fn(async () => ({ root: '/work/repo', branch: 'main', clean: false, files })),
        });
        initNotesGitPanel(elements, api);
        await flushPromises();

        expect(Array.from(elements.gitFileList.querySelectorAll('.notes-git-file-row'), (row) => row.dataset.path)).toEqual([
            'a-new.txt', 'z-new.txt',
            'a-work.txt', 'm-mixed.txt', 'z-work.txt',
            'a-staged.txt', 'z-staged.txt',
        ]);
    });

    it('toggles the commit row, focuses its message, and stages tracked files only', async () => {
        const elements = createElements();
        const api = createApi();
        const notify = vi.fn();
        const panel = initNotesGitPanel(elements, api, { notify });
        await flushPromises();
        const focus = vi.spyOn(elements.gitCommitMessage, 'focus');
        expect(elements.gitStageAll.disabled).toBe(false);
        expect(elements.gitCommitRow.hidden).toBe(true);
        elements.gitCommitToggle.click();
        expect(elements.gitCommitRow.hidden).toBe(false);
        expect(elements.gitCommitToggle.getAttribute('aria-pressed')).toBe('true');
        expect(focus).toHaveBeenCalledOnce();
        elements.gitCommitToggle.click();
        expect(elements.gitCommitRow.hidden).toBe(true);
        expect(elements.gitCommitToggle.getAttribute('aria-pressed')).toBe('false');

        api.refreshStatus.mockClear();
        elements.gitStageAll.click();
        await flushPromises();
        expect(api.stageAll).toHaveBeenCalledWith('/work/repo');
        expect(api.refreshStatus).toHaveBeenCalledWith('/work/repo');
        expect(notify).toHaveBeenCalledWith('All tracked changes staged', 'info');
        expect(panel.getCurrentRoot()).toBe('/work/repo');
    });

    it('validates the commit request and follows the Git package execution mode', async () => {
        const elements = createElements();
        const api = createApi();
        const notify = vi.fn();
        const notifyTransfer = vi.fn();
        initNotesGitPanel(elements, api, { notify, notifyTransfer });
        await flushPromises();
        const focus = vi.spyOn(elements.gitCommitMessage, 'focus');
        elements.gitCommitSubmit.click();
        await flushPromises();
        expect(api.runGitAction).not.toHaveBeenCalled();
        expect(elements.gitCommitRow.hidden).toBe(false);
        expect(focus).toHaveBeenCalledOnce();

        elements.gitCommitSubmit.click();
        await flushPromises();
        expect(notify).toHaveBeenCalledWith('Commit message is required', 'warn');

        elements.gitCommitMessage.value = '  Add feature  ';
        api.refreshStatus.mockClear();
        elements.gitCommitSubmit.click();
        await flushPromises();
        expect(api.runGitAction).toHaveBeenCalledWith('/work/repo', 'commit', { message: 'Add feature' });
        expect(notifyTransfer).toHaveBeenCalledWith(
            { stdout: 'commit output', stderr: '', error: '' }, 'Commit created',
        );
        expect(elements.gitCommitMessage.value).toBe('');
        expect(api.refreshStatus).toHaveBeenCalledWith('/work/repo');

        api.refreshStatus.mockClear();
        api.runGitAction.mockResolvedValueOnce({ mode: 'interactive', stdout: 'hook output', stderr: '', error: 'git action failed with exit code 1' });
        elements.gitCommitMessage.value = 'Fix hook';
        elements.gitCommitSubmit.click();
        await flushPromises();
        expect(notify).toHaveBeenLastCalledWith('Commit failed. See terminal for output', 'error');
        expect(notifyTransfer).toHaveBeenCalledTimes(1);
        expect(elements.gitCommitMessage.value).toBe('Fix hook');
        expect(api.refreshStatus).not.toHaveBeenCalled();

        api.runGitAction.mockRejectedValueOnce(new Error('bridge failure'));
        elements.gitCommitMessage.value = 'Bridge failure';
        elements.gitCommitSubmit.click();
        await flushPromises();
        expect(notify).toHaveBeenLastCalledWith('Commit failed. See terminal for output', 'error');

        api.runGitAction.mockResolvedValueOnce({ mode: 'go', stdout: 'fetch completed', stderr: '', error: '' });
        elements.gitCommitMessage.value = 'Run Go action';
        elements.gitCommitSubmit.click();
        await flushPromises();
        expect(notifyTransfer).toHaveBeenCalledWith(
            { stdout: 'fetch completed', stderr: '', error: '' }, 'Commit created',
        );
        expect(api.refreshStatus).toHaveBeenCalledWith('/work/repo');
    });

    it('keeps Push idle when confirmation is cancelled', async () => {
        const elements = createElements();
        const api = createApi();
        const panel = initNotesGitPanel(elements, api);
        await flushPromises();
        elements.gitCommitToggle.click();

        elements.gitPush.click();
        await flushPromises();

        expect(api.confirmGitPush).toHaveBeenCalledWith('/work/repo');
        expect(api.runGitAction).not.toHaveBeenCalled();
        expect(panel.getCurrentRoot()).toBe('/work/repo');
    });

    it('reports a confirmed push result and refreshes Git status', async () => {
        const elements = createElements();
        const api = createApi({ confirmGitPush: vi.fn(() => Promise.resolve({
            mode: 'interactive', stdout: 'pushed output', stderr: '', error: '',
        })) });
        const notify = vi.fn();
        const notifyTransfer = vi.fn();
        initNotesGitPanel(elements, api, { notify, notifyTransfer });
        await flushPromises();
        api.refreshStatus.mockClear();
        elements.gitCommitToggle.click();
        elements.gitPush.click();
        await flushPromises();
        expect(notifyTransfer).toHaveBeenCalledWith(
            { stdout: 'pushed output', stderr: '', error: '' }, 'Push completed',
        );
        expect(api.refreshStatus).toHaveBeenCalledWith('/work/repo');
        expect(elements.gitPush.disabled).toBe(false);
    });

    it('opens a commit context menu without changing comparison until requested', async () => {
        const elements = createElements();
        const api = createApi();
        const showCommitMenu = vi.fn();
        const panel = initNotesGitPanel(elements, api, { showCommitMenu });
        await flushPromises();
        panel.setCommitsActive(true);
        await flushPromises();
        const event = new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 25, clientY: 40 });
        elements.gitCommitList.querySelector('.notes-git-commit').dispatchEvent(event);
        expect(event.defaultPrevented).toBe(true);
        expect(panel.getCompareBranch()).toBe('');
        expect(showCommitMenu).toHaveBeenCalledWith(expect.objectContaining({ root: '/work/repo', x: 25, y: 40, commit: expect.objectContaining({ hash: 'abcdef1' }) }));
        showCommitMenu.mock.calls[0][0].compare();
        await flushPromises();
        expect(panel.getCompareBranch()).toBe('abcdef1');
    });

    it('selects a commit through the same comparison refresh and diff path as the branch menu', async () => {
        const elements = createElements();
        const fullHash = 'abcdef1' + '0'.repeat(33);
        const api = createApi({
            refreshStatus: vi.fn(() => Promise.resolve({ root: '/work/repo', branch: 'main', files: [], clean: true })),
            listCommits: vi.fn(() => Promise.resolve([
                { hash: 'abcdef1', fullHash, subject: 'Base commit', date: '', author: '', graph: ['*'] },
            ])),
        });
        const onStatusChange = vi.fn();
        const panel = initNotesGitPanel(elements, api, { onStatusChange });
        await flushPromises();
        panel.setCommitsActive(true);
        await flushPromises();
        elements.gitCommitList.querySelector('.notes-git-commit').click();
        await flushPromises();
        expect(elements.gitCompare.textContent).toBe('abcdef1');
        expect(panel.getCompareBranch()).toBe(fullHash);
        expect(api.refreshStatus).toHaveBeenCalledWith('/work/repo');
        expect(api.getBranchDiffFiles).toHaveBeenCalledWith('/work/repo', fullHash);
        expect(onStatusChange).toHaveBeenLastCalledWith(expect.objectContaining({ files: [expect.objectContaining({ path: 'src/main.go' })] }), '/work/repo');
        elements.gitFileList.querySelector('.notes-git-file').click();
        await flushPromises();
        expect(api.getDiffAgainstBranch).toHaveBeenCalledWith('/work/repo', 'src/main.go', '', fullHash);
        await panel.setCompareBranch('');
        expect(elements.gitCompare.textContent).toBe('HEAD');
        for (const key of ['Enter', ' ']) {
            api.refreshStatus.mockClear();
            elements.gitCommitList.querySelector('.notes-git-commit').dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true }));
            await flushPromises();
            expect(elements.gitCompare.textContent).toBe('abcdef1');
            expect(api.refreshStatus).toHaveBeenCalledOnce();
            await panel.setCompareBranch('');
            await flushPromises();
        }
    });

    it('renders tags after hashes and filters hashes, full messages, and tags without reloading Git per query', async () => {
        const elements = createElements();
        const commits = [
            { hash: 'aaaaaaa', fullHash: 'a'.repeat(34) + 'ef0123', subject: 'First merge', message: 'First merge\n\nRollout ticket', tags: ['v1.2.3', 'release/current'], date: '', author: 'Test User', graph: ['*'] },
            { hash: 'bbbbbbb', fullHash: 'b'.repeat(40), subject: 'Second commit', message: 'Second commit', date: '', author: 'Test User', graph: ['*'] },
        ];
        const api = createApi({ listCommits: vi.fn(() => Promise.resolve(commits)) });
        const panel = initNotesGitPanel(elements, api);
        await flushPromises();
        panel.setCommitsActive(true);
        await flushPromises();
        const header = elements.gitCommitList.querySelector('.notes-git-commit-header');
        expect(header.firstElementChild.className).toBe('notes-git-commit-hash');
        expect(header.lastElementChild.className).toBe('notes-git-commit-tags');
        expect(header.lastElementChild.textContent).toBe('v1.2.3, release/current');
        expect(elements.gitCommitList.querySelectorAll('.notes-git-commit-tags')).toHaveLength(1);
        await panel.setCommitFilter('ef0123');
        expect(api.listCommits).toHaveBeenLastCalledWith('/work/repo', 10000);
        expect(elements.gitCommitList.querySelectorAll('.notes-git-commit')).toHaveLength(1);
        expect(elements.gitCommitList.querySelector('.notes-git-commit-hash').textContent).toBe('aaaaaaa');
        expect(elements.gitCommitList.querySelector('.notes-git-commit').classList.contains('notes-git-commit-filtered')).toBe(true);
        expect(api.filterStrings).toHaveBeenCalledWith('ef0123', commits.map((commit) => [commit.fullHash, commit.message, ...(commit.tags || [])].join('\n')));
        await panel.setCommitFilter('ROLLOUT');
        expect(elements.gitCommitList.querySelector('.notes-git-commit-subject').textContent).toBe('First merge');
        await panel.setCommitFilter('V1.2.3');
        expect(elements.gitCommitList.querySelectorAll('.notes-git-commit')).toHaveLength(1);
        expect(elements.gitCommitList.querySelector('.notes-git-commit-hash').textContent).toBe('aaaaaaa');
        await panel.setCommitFilter('release/current');
        expect(elements.gitCommitList.querySelectorAll('.notes-git-commit')).toHaveLength(1);
        expect(api.listCommits).toHaveBeenCalledTimes(2);
        await panel.setCommitFilter('Test User');
        expect(elements.gitCommitList.textContent).toBe('No matching commits');
        await panel.setCommitFilter('');
        expect(elements.gitCommitList.querySelectorAll('.notes-git-commit')).toHaveLength(2);
        expect(elements.gitCommitList.querySelector('.notes-git-commit-filtered')).toBeNull();
        expect(elements.gitCommitList.querySelector('.notes-git-commit-graph')).not.toBeNull();
        expect(api.listCommits).toHaveBeenCalledTimes(2);
    });

    it('defaults filename search off and filters only matching changed paths when enabled', async () => {
        const elements = createElements();
        const commits = [
            { hash: 'aaaaaaa', message: 'needle in message', subject: 'needle in message', tags: ['needle-tag'], files: ['src/api.go'], graph: ['*'] },
            { hash: 'bbbbbbb', message: 'Update docs', subject: 'Update docs', files: ['docs/needle.md', 'docs/guide.md'], graph: ['*'] },
        ];
        const api = createApi({
            listCommits: vi.fn(async () => commits), listCommitsWithFiles: vi.fn(async () => commits),
        });
        const panel = initNotesGitPanel(elements, api);
        await flushPromises();
        expect(elements.gitFilenameSearch.checked).toBe(false);
        expect(elements.gitFilenameSearchRow.hidden).toBe(true);
        panel.setCommitsActive(true);
        await flushPromises();
        expect(elements.gitFilenameSearchRow.hidden).toBe(false);
        elements.gitFilenameSearch.checked = true;
        elements.gitFilenameSearch.dispatchEvent(new Event('change'));
        await flushPromises();
        expect(api.listCommitsWithFiles).not.toHaveBeenCalled();
        elements.gitFilenameSearch.checked = false;
        elements.gitFilenameSearch.dispatchEvent(new Event('change'));
        await flushPromises();
        await panel.setCommitFilter('needle');
        expect(elements.gitCommitList.querySelector('.notes-git-commit-hash').textContent).toBe('aaaaaaa');
        expect(api.listCommitsWithFiles).not.toHaveBeenCalled();
        elements.gitFilenameSearch.checked = true;
        elements.gitFilenameSearch.dispatchEvent(new Event('change'));
        await flushPromises();
        expect(elements.gitCommitList.querySelectorAll('.notes-git-commit')).toHaveLength(1);
        expect(elements.gitCommitList.querySelector('.notes-git-commit-hash').textContent).toBe('bbbbbbb');
        expect(api.filterStrings).toHaveBeenLastCalledWith('needle', ['src/api.go', 'docs/needle.md', 'docs/guide.md']);
        expect(api.listCommitsWithFiles).toHaveBeenCalledOnce();
        await panel.setCommitFilter('api');
        expect(elements.gitCommitList.querySelector('.notes-git-commit-hash').textContent).toBe('aaaaaaa');
        expect(api.listCommitsWithFiles).toHaveBeenCalledOnce();
        await panel.setCommitFilter('');
        expect(elements.gitCommitList.querySelectorAll('.notes-git-commit')).toHaveLength(2);
        panel.setCommitsActive(false);
        expect(elements.gitFilenameSearchRow.hidden).toBe(true);
        panel.setProjectRoot('/other/repo');
        expect(elements.gitFilenameSearch.checked).toBe(false);
    });

    it('ignores a delayed filename-search result after the toggle is disabled', async () => {
        const elements = createElements();
        const commits = [
            { hash: 'aaaaaaa', message: 'needle', subject: 'needle', files: ['other.go'], graph: ['*'] },
            { hash: 'bbbbbbb', message: 'Other', subject: 'Other', files: ['needle.go'], graph: ['*'] },
        ];
        let finish;
        const api = createApi({
            listCommits: vi.fn(async () => commits),
            listCommitsWithFiles: vi.fn(() => new Promise((resolve) => { finish = resolve; })),
        });
        const panel = initNotesGitPanel(elements, api);
        await flushPromises();
        panel.setCommitsActive(true);
        await flushPromises();
        await panel.setCommitFilter('needle');
        elements.gitFilenameSearch.checked = true;
        elements.gitFilenameSearch.dispatchEvent(new Event('change'));
        elements.gitFilenameSearch.checked = false;
        elements.gitFilenameSearch.dispatchEvent(new Event('change'));
        await flushPromises();
        finish(commits);
        await flushPromises();
        expect(elements.gitCommitList.querySelectorAll('.notes-git-commit')).toHaveLength(1);
        expect(elements.gitCommitList.querySelector('.notes-git-commit-hash').textContent).toBe('aaaaaaa');
    });

    it('uses the current shared query when re-enabling Commits after the filter was cleared', async () => {
        const elements = createElements();
        let query = 'Merge';
        const panel = initNotesGitPanel(elements, createApi(), { getCommitFilter: () => query });
        await flushPromises();
        panel.setCommitsActive(true);
        await flushPromises();
        expect(elements.gitCommitList.querySelectorAll('.notes-git-commit')).toHaveLength(1);
        panel.setCommitsActive(false);
        query = '';
        panel.setCommitsActive(true);
        await flushPromises();
        expect(elements.gitCommitList.querySelectorAll('.notes-git-commit')).toHaveLength(2);
    });

    it('ignores a delayed filter response after a newer commit query', async () => {
        const elements = createElements();
        const api = createApi();
        let resolveFilter;
        const panel = initNotesGitPanel(elements, api);
        await flushPromises();
        panel.setCommitsActive(true);
        await flushPromises();
        api.filterStrings.mockImplementationOnce(() => new Promise((resolve) => { resolveFilter = resolve; }));
        const oldQuery = panel.setCommitFilter('Merge');
        await flushPromises();
        await panel.setCommitFilter('Feature commit');
        resolveFilter({ List: ['abcdef1\nMerge feature'] });
        await oldQuery;
        expect(elements.gitCommitList.querySelectorAll('.notes-git-commit')).toHaveLength(1);
        expect(elements.gitCommitList.querySelector('.notes-git-commit-subject').textContent).toBe('Feature commit');
    });

    it('continues a56202f0 collapsing slashes past adjacent vertical lanes', async () => {
        const elements = createElements();
        const api = createApi({ listCommits: vi.fn(() => Promise.resolve([
            {
                hash: 'a56202f0', subject: 'Merge', date: '', author: '',
                graph: String.raw`| | * | | | | | | | |
| | |\ \ \ \ \ \ \ \ \
| | |/ / / / / / / / /
| |/| | | | | | | | |
`.trimEnd().split('\n'),
            },
            { hash: 'f22599f3', subject: 'Merge', date: '', author: '', graph: ['| * | | | | | | | | |'] },
        ])) });
        const panel = initNotesGitPanel(elements, api);
        await flushPromises();
        panel.setCommitsActive(true);
        await flushPromises();
        const path = elements.gitCommitList.querySelector('svg path').getAttribute('d');
        expect(path).toContain('M 44 45 L 28 63');
        expect(path).toContain('M 36 45 L 36 63');
        expect(path).not.toContain('M 44 45 L 36 63');
        expect(path).not.toContain('M 36 45 L 28 63');
    });

    it('updates overlap masking on resize and releases its observer on toggle-off', async () => {
        let resized;
        const disconnect = vi.fn();
        vi.stubGlobal('ResizeObserver', class {
            constructor(callback) { resized = callback; }
            observe() {}
            disconnect() { disconnect(); }
        });
        try {
            const elements = createElements();
            const panel = initNotesGitPanel(elements, createApi());
            await flushPromises();
            panel.setCommitsActive(true);
            await flushPromises();
            const row = elements.gitCommitList.querySelector('.notes-git-commit');
            const graph = row.querySelector('.notes-git-commit-graph');
            const content = row.querySelector('.notes-git-commit-content');
            vi.spyOn(graph, 'getBoundingClientRect').mockReturnValue({ width: 600, left: 10 });
            const bounds = vi.spyOn(content, 'getBoundingClientRect').mockReturnValue({ left: 130 });
            resized([{ target: row }]);
            expect(row.style.getPropertyValue('--git-overlap-start')).toBe('120px');
            bounds.mockReturnValue({ left: 82 });
            resized([{ target: row }]);
            expect(row.style.getPropertyValue('--git-overlap-start')).toBe('72px');
            panel.setCommitsActive(false);
            expect(disconnect).toHaveBeenCalledOnce();
        } finally {
            vi.unstubAllGlobals();
        }
    });

    it.each([
        ['a56202f0 direction reversal', String.raw`| | * | | | | | | | |
| | |\ \ \ \ \ \ \ \ \
| | |/ / / / / / / / /
| |/| | | | | | | | |
| * | | | | | | | | |
`],
        ['eebc4be8 underscore bridge', String.raw`* | | | | | | | | | | | | | |
| |_|_|_|_|_|_|_|_|_|_|_|_|/
|/| | | | | | | | | | | | |
* | | | | | | | | | | | | |
`],
        ['9a1dddf5 nested collapse', String.raw`| | | | * | | | | | | | | | | |
| | | |/ / / / / / / / / / / /
| | |/| | | | | | | | | | | |
| | * | | | | | | | | | | | |
`],
        ['f601effe overlapping collapse', String.raw`| | | | | | | | | | | | * | | | | | | | | | | | | | | | | | |
| | | | | | | | | | | |/| | | | | | | | | | | | | | | | | | |
| | | | | | | | | | |/|/ / / / / / / / / / / / / / / / / / /
| | | | | | | | | | | * | | | | | | | | | / / / / / / / / /
`],
    ])('preserves all incoming diagonal lanes in %s', async (_name, raw) => {
        const graph = raw.trimEnd().split('\n');
        const elements = createElements();
        const api = createApi({ listCommits: vi.fn(() => Promise.resolve([
            { hash: 'fixture', subject: 'Merge', date: '', author: '', graph },
        ])) });
        const panel = initNotesGitPanel(elements, api);
        await flushPromises();
        panel.setCommitsActive(true);
        await flushPromises();
        const path = elements.gitCommitList.querySelector('svg path').getAttribute('d');
        const segments = [...path.matchAll(/M ([\d.-]+) ([\d.-]+) L ([\d.-]+) ([\d.-]+)/g)]
            .map((match) => match.slice(1).map(Number));
        const step = 72 / graph.length;
        for (let row = 1; row < graph.length; row++) {
            for (const [column, character] of [...graph[row]].entries()) {
                if (!'/\\'.includes(character) || '_-'.includes(graph[row][column - 1] || ' ')) continue;
                const matches = segments.filter((segment) => Math.abs(segment[1] - (row - 0.5) * step) < 0.001
                    && Math.abs(segment[3] - (row + 0.5) * step) < 0.001 && segment[2] === column * 8 + 4);
                expect(matches, `missing input for ${character} at row ${row}, column ${column}`).not.toHaveLength(0);
                if (graph[row - 1][column] === '\\' && character === '/') {
                    expect(matches.some((segment) => segment[0] === column * 8 + 4), `shifted reversal at row ${row}, column ${column}`).toBe(true);
                }
            }
        }
    });

    it('connects octopus merge elbows and collapsing lanes to their actual neighbours', async () => {
        const elements = createElements();
        const graphs = [
            ['*-. ', '|\\ \\'],
            ['| | * ', '| |/ '],
            ['| * ', '|/ '],
            ['* '],
        ];
        const api = createApi({ listCommits: vi.fn(() => Promise.resolve(graphs.map((graph, index) => ({
            hash: String(index), subject: 'Merge', date: '', author: '', graph,
        })))) });
        const panel = initNotesGitPanel(elements, api);
        await flushPromises();
        panel.setCommitsActive(true);
        await flushPromises();
        const paths = [...elements.gitCommitList.querySelectorAll('svg path')].map((path) => path.getAttribute('d'));
        expect(paths[0]).toContain('M 20 18 L 28 54');
        expect(paths[0]).toContain('M 4 18 H 20');
        expect(paths[1]).toContain('M 36 18 L 28 54');
        expect(paths[1]).toContain('M 28 54 L 24 72');
        expect(paths[2]).toContain('M 24 0 L 20 18');
        expect(paths[2]).toContain('M 20 18 L 12 54');
        expect(paths[3]).toContain('M 8 0 L 4 36');
        expect(paths[3]).not.toContain('72');
    });

    it('joins expanding diagonal lanes across consecutive commit rows', async () => {
        const elements = createElements();
        const graphs = [
            ['| | * |   ', '| | |\\ \\'],
            ['| | * \\ \\   ', '| | |\\ \\ \\'],
            ['| | * \\ \\ \\   ', '| | |\\ \\ \\ \\'],
            ['| * | \\ \\ \\ \\   ', '| |\\ \\ \\ \\ \\ \\'],
        ];
        const api = createApi({ listCommits: vi.fn(() => Promise.resolve(graphs.map((graph, index) => ({
            hash: String(index), subject: 'Merge', date: '', author: '', graph,
        })))) });
        const panel = initNotesGitPanel(elements, api);
        await flushPromises();
        panel.setCommitsActive(true);
        await flushPromises();
        const paths = [...elements.gitCommitList.querySelectorAll('svg path')].map((path) => path.getAttribute('d'));
        const endpoints = (path, top) => {
            const positions = [];
            for (const match of path.matchAll(/M ([\d.-]+) ([\d.-]+) (?:L ([\d.-]+) ([\d.-]+)|V ([\d.-]+)|H ([\d.-]+))/g)) {
                const startX = Number(match[1]);
                const startY = Number(match[2]);
                const endX = match[3] !== undefined ? Number(match[3]) : match[6] !== undefined ? Number(match[6]) : startX;
                const endY = match[4] !== undefined ? Number(match[4]) : match[5] !== undefined ? Number(match[5]) : startY;
                if (top && startY === 0) positions.push(startX);
                if (!top && endY === 72) positions.push(endX);
            }
            return [...new Set(positions)].sort((left, right) => left - right);
        };
        for (let index = 0; index < paths.length - 1; index++) {
            expect(endpoints(paths[index], false)).toEqual(endpoints(paths[index + 1], true));
        }
        expect(endpoints(paths[0], false)).toEqual([4, 20, 36, 48, 64]);
    });

    it('discards delayed commit responses after switching back to files', async () => {
        const elements = createElements();
        let resolveHistory;
        const api = createApi({ listCommits: vi.fn(() => new Promise((resolve) => { resolveHistory = resolve; })) });
        const panel = initNotesGitPanel(elements, api);
        await flushPromises();
        panel.setCommitsActive(true);
        panel.setCommitsActive(false);
        resolveHistory([{ hash: 'late', subject: 'Old history', date: '', author: '', graph: ['*'] }]);
        await flushPromises();
        expect(elements.gitCommitList.querySelector('.notes-git-commit')).toBeNull();
        expect(elements.gitFileList.querySelector('.notes-git-file')).not.toBeNull();
    });

    it('toggles a graph-backed commit list and restores files on workspace reset', async () => {
        const elements = createElements();
        const api = createApi();
        const panel = initNotesGitPanel(elements, api);
        await flushPromises();
        elements.gitCommits.click();
        await flushPromises();
        expect(elements.gitCommits.getAttribute('aria-pressed')).toBe('true');
        expect(api.listCommits).toHaveBeenCalledWith('/work/repo', 100);
        expect(elements.gitCommitList.querySelectorAll('.notes-git-commit')).toHaveLength(2);
        expect(elements.gitCommitList.querySelector('.notes-git-commit-hash').textContent).toBe('abcdef1');
        expect(elements.gitCommitList.querySelector('.notes-git-commit-subject').textContent).toBe('Merge feature');
        expect(Array.from(elements.gitCommitList.querySelector('.notes-git-commit').querySelectorAll('.notes-git-commit-meta'), (line) => line.textContent)).toEqual([
            '2026-10-12 12:30:00', '(Test User)',
        ]);
        expect(elements.gitCommitList.querySelector('svg path').getAttribute('d')).toContain('L');
        expect(elements.gitCommitList.querySelector('.notes-git-commit').style.getPropertyValue('--git-graph-width')).toBe('32px');
        expect(elements.gitFileList.querySelector('.notes-git-commit')).toBeNull();
        expect(elements.gitFileList.querySelector('.notes-git-file')).not.toBeNull();
        panel.setCommitsActive(false);
        expect(elements.gitCommits.getAttribute('aria-pressed')).toBe('false');
        expect(elements.gitFileList.querySelector('.notes-git-file')).not.toBeNull();
    });

    it('shows repository status and exposes workspace, stage, and diff actions', async () => {
        const elements = createElements();
        const api = createApi();
        const showWorkspaceMenu = vi.fn();
        const showTooltip = vi.fn();
        const hideTooltip = vi.fn();
        const onOpenDiff = vi.fn();
        const beforeFileAction = vi.fn(() => Promise.resolve(true));
        const panel = initNotesGitPanel(elements, api, {
            showWorkspaceMenu, showTooltip, hideTooltip, onOpenDiff, beforeFileAction,
        });
        await flushPromises();

        expect(elements.gitTab.style.display).toBe('');
        expect(elements.gitWorkspace.textContent).toBe(String.fromCodePoint(0xf802));
        expect(elements.gitBranch.textContent).toBe('main');
        expect(elements.gitCompare.textContent).toBe('HEAD');
        expect(elements.gitAheadBehind.textContent).toBe('↑2 ↓1');
        expect(elements.gitRemote.hidden).toBe(false);
        expect(elements.gitRemote.textContent).toBe(String.fromCodePoint(0xf09b));
        expect(elements.gitRemote.getAttribute('aria-label')).toBe('Open GitHub repository');
        elements.gitRemote.click();
        expect(api.openRemote).toHaveBeenCalledWith('https://github.com/org/repo/tree/main');
        expect(elements.gitFileList.textContent).toContain('src/main.go');
        expect(elements.gitFileList.textContent).toContain('README.md');
        const statusCode = elements.gitFileList.querySelector('[data-path="src/main.go"] .notes-git-status-code');
        statusCode.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }));
        expect(showTooltip).toHaveBeenCalledWith(expect.stringContaining('working tree (M) modified'));
        statusCode.dispatchEvent(new MouseEvent('mouseout', { bubbles: true, relatedTarget: document.body }));
        expect(hideTooltip).toHaveBeenCalled();

        elements.gitWorkspace.click();
        expect(showWorkspaceMenu).toHaveBeenCalledWith('/work/repo', elements.gitWorkspace);

        const stageButton = elements.gitFileList.querySelector('[data-git-action="stage"][data-path="src/main.go"]');
        const unstageButton = elements.gitFileList.querySelector('[data-git-action="unstage"]');
        expect(stageButton.classList.contains('notes-git-action-stage')).toBe(true);
        expect(unstageButton.classList.contains('notes-git-action-unstage')).toBe(true);
        expect(stageButton.textContent).toBe('Unstaged');
        expect(stageButton.dataset.tooltip).toBe('Click to stage');
        expect(stageButton.hasAttribute('title')).toBe(false);
        expect(unstageButton.textContent).toBe('Staged');
        expect(unstageButton.dataset.tooltip).toBe('Click to unstage');
        expect(unstageButton.hasAttribute('title')).toBe(false);
        const untrackedButton = elements.gitFileList.querySelector('[data-git-action="stage"][data-path="new-file.txt"]');
        expect(untrackedButton.textContent).toBe('Untracked');
        expect(untrackedButton.classList.contains('notes-git-action-stage')).toBe(true);
        expect(untrackedButton.classList.contains('notes-git-action-untracked')).toBe(true);
        expect(untrackedButton.dataset.tooltip).toBe('Click to stage');
        expect(untrackedButton.getAttribute('aria-label')).toBe('Click to stage');
        stageButton.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }));
        expect(showTooltip).toHaveBeenCalledWith('Click to stage');
        stageButton.click();
        await flushPromises();
        expect(beforeFileAction).toHaveBeenCalledWith({
            root: '/work/repo',
            path: 'src/main.go',
            action: 'stage',
        });
        expect(api.stageFile).toHaveBeenCalledWith('/work/repo', 'src/main.go');

        const fileButton = elements.gitFileList.querySelector('.notes-git-file[data-path="src/main.go"]');
        fileButton.click();
        await flushPromises();
        expect(api.getDiff).toHaveBeenCalledWith('/work/repo', 'src/main.go', '');
        expect(onOpenDiff).toHaveBeenCalledWith(expect.objectContaining({
            root: '/work/repo',
            file: expect.objectContaining({ path: 'src/main.go' }),
            diff: expect.objectContaining({ original: 'old', working: 'new' }),
        }));
        expect(panel.getCurrentRoot()).toBe('/work/repo');
    });

    it('filters Git files and swaps the second row with commit controls without losing the query', async () => {
        const elements = createElements();
        const api = createApi();
        const panel = initNotesGitPanel(elements, api);
        await flushPromises();
        expect(elements.gitFilterRow.hidden).toBe(false);
        expect(elements.gitCommitRow.hidden).toBe(true);
        const visiblePaths = () => [...elements.gitFileList.querySelectorAll('.notes-git-file-row')]
            .filter((row) => !row.hidden).map((row) => row.dataset.path);
        elements.gitFileFilter.value = 'main.go';
        elements.gitFileFilter.dispatchEvent(new Event('input'));
        await flushPromises();
        expect(visiblePaths()).toEqual(['src/main.go']);
        expect(api.filterStrings).toHaveBeenLastCalledWith('main.go', expect.arrayContaining(['src/main.go']));
        elements.gitCommitToggle.click();
        expect(elements.gitCommitRow.hidden).toBe(false);
        expect(elements.gitFilterRow.hidden).toBe(true);
        elements.gitCommitToggle.click();
        expect(elements.gitCommitRow.hidden).toBe(true);
        expect(elements.gitFilterRow.hidden).toBe(false);
        expect(elements.gitFileFilter.value).toBe('main.go');
        await panel.refresh();
        await flushPromises();
        expect(visiblePaths()).toEqual(['src/main.go']);
        elements.gitFileFilter.value = 'missing-file';
        elements.gitFileFilter.dispatchEvent(new Event('input'));
        await flushPromises();
        expect(visiblePaths()).toEqual([]);
        expect(elements.gitFileList.textContent).toContain('No matching files');
        elements.gitFileFilter.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
        expect(elements.gitFileFilter.value).toBe('');
        expect(visiblePaths()).toHaveLength(3);
        expect(elements.gitFileList.querySelector('.notes-git-filter-empty')).toBeNull();
        elements.gitFileFilter.value = 'main';
        await panel.setProjectRoot('/other/repo');
        expect(elements.gitFileFilter.value).toBe('');
    });

    it('ignores a delayed file-filter response after the filter is cleared', async () => {
        const elements = createElements();
        let complete;
        initNotesGitPanel(elements, createApi({
            filterStrings: vi.fn(() => new Promise((resolve) => { complete = resolve; })),
        }));
        await flushPromises();
        elements.gitFileFilter.value = 'old query';
        elements.gitFileFilter.dispatchEvent(new Event('input'));
        elements.gitFileFilter.value = '';
        elements.gitFileFilter.dispatchEvent(new Event('input'));
        complete({ List: [] });
        await flushPromises();
        expect([...elements.gitFileList.querySelectorAll('.notes-git-file-row')].every((row) => !row.hidden)).toBe(true);
        expect(elements.gitFileList.querySelector('.notes-git-filter-empty')).toBeNull();
    });

    it('switches comparison branches without changing the status branch and opens branch-based diffs', async () => {
        const elements = createElements();
        const api = createApi();
        const showCompareMenu = vi.fn();
        const onOpenDiff = vi.fn();
        const panel = initNotesGitPanel(elements, api, { showCompareMenu, onOpenDiff });
        await flushPromises();

        elements.gitCompare.click();
        expect(showCompareMenu).toHaveBeenCalledWith(
            '/work/repo', 'main', '', elements.gitCompare, expect.any(Function),
        );
        await panel.setCompareBranch('release/2');
        await flushPromises();

        expect(elements.gitCompare.textContent).toBe('release/2');
        expect(elements.gitBranch.textContent).toBe('main');
        expect(api.getBranchDiffFiles).toHaveBeenCalledWith('/work/repo', 'release/2');
        expect(elements.gitFileList.querySelector('.notes-git-status-code').textContent).toBe('≠');
        expect(panel.getCachedStatus().files).toHaveLength(3);

        elements.gitFileList.querySelector('.notes-git-file').click();
        await flushPromises();
        expect(api.getDiffAgainstBranch).toHaveBeenCalledWith('/work/repo', 'src/main.go', '', 'release/2');
        expect(api.getDiff).not.toHaveBeenCalled();
        expect(onOpenDiff).toHaveBeenCalledWith(expect.objectContaining({
            diff: expect.objectContaining({ original: 'branch' }),
        }));
    });

    it('appends equal commit files after differing files and opens their comparison without staging controls', async () => {
        const elements = createElements();
        const api = createApi({
            getBranchDiffFiles: vi.fn(async () => [
                { path: 'a-equal.go', code: '=', compareOnly: true },
                { path: 'z-different.go', code: ' M', compareOnly: true, unstaged: true },
                { path: 'b-equal.go', code: '=', compareOnly: true },
            ]),
        });
        const panel = initNotesGitPanel(elements, api);
        await flushPromises();
        await panel.setCompareBranch('abcdef123456');
        const rows = [...elements.gitFileList.querySelectorAll('.notes-git-file-row')];
        expect(rows.map((row) => row.dataset.path)).toEqual(['z-different.go', 'a-equal.go', 'b-equal.go']);
        expect(rows.map((row) => row.querySelector('.notes-git-status-code').textContent)).toEqual(['≠', '=', '=']);
        expect(rows[1].querySelector('.notes-git-status-code').dataset.tooltip).toBe('Matches abcdef123456');
        expect(rows[1].querySelectorAll('.notes-git-action')).toHaveLength(0);
        rows[1].querySelector('.notes-git-file').click();
        await flushPromises();
        expect(api.getDiffAgainstBranch).toHaveBeenCalledWith('/work/repo', 'a-equal.go', '', 'abcdef123456');
        elements.gitFileFilter.value = 'a-equal.go';
        elements.gitFileFilter.dispatchEvent(new Event('input'));
        await flushPromises();
        expect([...elements.gitFileList.querySelectorAll('.notes-git-file-row')]
            .filter((row) => !row.hidden).map((row) => row.dataset.path)).toEqual(['a-equal.go']);
        elements.gitFileFilter.value = '';
        elements.gitFileFilter.dispatchEvent(new Event('input'));
        await panel.setCompareBranch('');
        expect([...elements.gitFileList.querySelectorAll('.notes-git-status-code')].some((code) => code.textContent === '=')).toBe(false);
    });

    it('keeps the Git panel available when the selected comparison branch disappears', async () => {
        const elements = createElements();
        const api = createApi({
            getBranchDiffFiles: vi.fn(() => Promise.reject(new Error('comparison branch does not exist'))),
        });
        const panel = initNotesGitPanel(elements, api);
        await flushPromises();

        await panel.setCompareBranch('deleted-branch');

        expect(elements.gitTab.style.display).toBe('');
        expect(elements.gitFileList.textContent).toContain('Could not compare with deleted-branch');
    });

    it.each(['project root is not a Git working tree', 'git executable was not found'])('keeps Git selected for %s and recovers for a valid root', async (reason) => {
        const elements = createElements();
        const api = createApi({
            getStatus: vi.fn(() => Promise.reject(new Error(reason))),
        });

        elements.gitTab.setAttribute('aria-selected', 'true');
        elements.gitPane.dataset.active = 'true';
        const panel = initNotesGitPanel(elements, api);
        await flushPromises();

        expect(elements.gitTab.style.display).toBe('');
        expect(elements.gitTab.getAttribute('aria-selected')).toBe('true');
        expect(elements.gitPane.dataset.active).toBe('true');
        expect(elements.gitToolbar.hidden).toBe(true);
        expect(elements.gitCommitRow.hidden).toBe(true);
        expect(elements.gitFileList.textContent).toContain(reason);
        api.getStatus.mockResolvedValue({ root: '/valid/repo', branch: 'main', clean: true, files: [] });
        await panel.setProjectRoot('/valid/repo');
        expect(elements.gitToolbar.hidden).toBe(false);
        expect(elements.gitTab.getAttribute('aria-selected')).toBe('true');
        expect(elements.gitFileList.textContent).toContain('Working tree clean');
    });

    it('hides all operation controls on watcher failure without deselecting Git', async () => {
        const elements = createElements();
        const api = createApi();
        initNotesGitPanel(elements, api);
        await flushPromises();
        elements.gitTab.setAttribute('aria-selected', 'true');
        elements.gitPane.dataset.active = 'true';
        elements.gitCommitToggle.click();
        expect(elements.gitToolbar.hidden).toBe(false);
        expect(elements.gitCommitRow.hidden).toBe(false);
        const changed = api.eventsOn.mock.calls.find(([name]) => name === 'notesGitStatusChanged')[1];
        changed({ root: '/work/repo', error: 'Repository no longer exists' });
        expect(elements.gitToolbar.hidden).toBe(true);
        expect(elements.gitCommitRow.hidden).toBe(true);
        expect(elements.gitTab.getAttribute('aria-selected')).toBe('true');
        expect(elements.gitPane.dataset.active).toBe('true');
        expect(elements.gitFileList.textContent).toContain('Repository no longer exists');
    });

    it('shows the reason when no project root is available without hiding Git', async () => {
        const elements = createElements();
        const api = createApi({ getCurrentProject: vi.fn(() => Promise.resolve('')) });
        initNotesGitPanel(elements, api);
        await flushPromises();
        expect(elements.gitTab.style.display).toBe('');
        expect(elements.gitToolbar.hidden).toBe(true);
        expect(elements.gitFileList.textContent).toContain('No project root');
        expect(api.getStatus).not.toHaveBeenCalled();
    });

    it('shows provider-specific brand glyphs and hides unsupported remotes', async () => {
        const elements = createElements();
        const api = createApi();
        const panel = initNotesGitPanel(elements, api);
        await flushPromises();

        for (const [provider, glyph] of [
            ['azuredevops', 0xf3ca],
            ['gitlab', 0xf296],
            ['bitbucket', 0xf171],
        ]) {
            api.getBrowserLink.mockResolvedValueOnce({ provider, url: `https://${provider}.example/repo` });
            await panel.refresh(false);
            await flushPromises();
            expect(elements.gitRemote.hidden).toBe(false);
            expect(elements.gitRemote.textContent).toBe(String.fromCodePoint(glyph));
        }

        api.getBrowserLink.mockResolvedValueOnce({ provider: '', url: '' });
        await panel.refresh(false);
        await flushPromises();
        expect(elements.gitRemote.hidden).toBe(true);
    });

    it('rehydrates its root cache from watcher-published Git status', async () => {
        const elements = createElements();
        const api = createApi();
        const eventHandlers = new Map();
        api.eventsOn.mockImplementation((name, handler) => eventHandlers.set(name, handler));
        const panel = initNotesGitPanel(elements, api);
        await flushPromises();

        eventHandlers.get('notesGitStatusChanged')({
            root: '/work/repo',
            status: {
                root: '/work/repo',
                branch: 'updated',
                ahead: 0,
                behind: 0,
                clean: false,
                files: [{ path: 'other.txt', code: '??', untracked: true }],
            },
        });

        expect(elements.gitBranch.textContent).toBe('updated');
        expect(panel.getCachedStatus('/work/repo').files[0].path).toBe('other.txt');
    });

    it('starts the metadata watch only while the Git tab is active', async () => {
        const elements = createElements();
        const api = createApi({ setStatusWatch: vi.fn(() => Promise.resolve()) });
        const panel = initNotesGitPanel(elements, api);
        await flushPromises();

        panel.setWatchActive(true);
        expect(api.setStatusWatch).toHaveBeenLastCalledWith('/work/repo', true);

        panel.setWatchActive(false);
        expect(api.setStatusWatch).toHaveBeenLastCalledWith('/work/repo', false);
    });
});
