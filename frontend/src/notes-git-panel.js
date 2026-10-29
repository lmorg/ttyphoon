function statusCode(file) {
    return String(file?.code || '??');
}

function statusTooltip(code) {
    if (code === '??') {
        return '??: untracked file; Git has not added this path to the index';
    }
    const meaning = {
        ' ': 'no change',
        A: 'added',
        M: 'modified',
        D: 'deleted',
        R: 'renamed',
        C: 'copied',
        U: 'unmerged conflict',
        T: 'file type changed',
        '!': 'ignored',
        '?': 'untracked',
    };
    const [index, worktree] = String(code || '  ').padEnd(2, ' ');
    return `${code}: index (${index}) ${meaning[index] || 'changed'}; working tree (${worktree}) ${meaning[worktree] || 'changed'}`;
}

function rootKey(root) {
    return String(root || '').replaceAll('\\', '/').replace(/\/+$/, '');
}

const gitRemoteIcons = {
    github: [0xf09b, 'GitHub'],
    azuredevops: [0xf3ca, 'Azure DevOps'],
    gitlab: [0xf296, 'GitLab'],
    bitbucket: [0xf171, 'Bitbucket'],
};

export function initNotesGitPanel(elements, api, options = {}) {
    if (!elements?.gitTab || !elements?.gitPane || !elements?.gitToolbar || !elements?.gitFileList || !elements?.gitRemote || !elements?.gitCompare
        || !elements?.gitCommitToggle || !elements?.gitCommitRow || !elements?.gitCommitMessage
        || !elements?.gitCommitSubmit || !elements?.gitStageAll) {
        return null;
    }

    let projectRoot = '';
    let requestGeneration = 0;
    let currentStatus = null;
    let compareBranch = '';
    let compareLabel = '';
    let commitsActive = false;
    let historyRequest = 0;
    let historyLimit = 100;
    let historyLayoutObserver = null;
    let historyFilter = '';
    let filenameSearch = false;
    let historySnapshotIncludesFiles = false;
    let historySnapshot = null;
    let historySnapshotLimit = 0;
    let historyLoad = null;
    let watchActive = false;
    let remoteRequest = 0;
    let fileFilterRequest = 0;
    const statusCache = new Map();

    const syncStatusWatch = () => {
        if (typeof api.setStatusWatch !== 'function') {
            return;
        }
        Promise.resolve(api.setStatusWatch(projectRoot, watchActive && Boolean(currentStatus))).catch(() => {});
    };

    const setAvailable = (available) => {
        elements.gitTab.style.display = '';
        elements.gitToolbar.hidden = !available;
        if (elements.gitFilterRow) elements.gitFilterRow.hidden = !available || !elements.gitCommitRow.hidden;
        elements.gitPane.dataset.available = String(available);
        options.onAvailabilityChange?.(available);
        if (!available) {
            fileFilterRequest++;
            elements.gitStageAll.disabled = true;
            elements.gitCommitRow.hidden = true;
            elements.gitCommitToggle.setAttribute('aria-pressed', 'false');
            historyRequest++;
            setCommitsActive(false);
        }
    };

    const showMessage = (message, kind = 'info', target = elements.gitFileList) => {
        target.replaceChildren();
        const item = document.createElement('div');
        item.className = `notes-git-message notes-git-message-${kind}`;
        item.textContent = message;
        target.appendChild(item);
    };

    const hideRemoteLink = () => {
        elements.gitRemote.hidden = true;
        delete elements.gitRemote.dataset.url;
    };

    const updateRemoteLink = async (status, root) => {
        const request = ++remoteRequest;
        if (!status || typeof api.getBrowserLink !== 'function' || typeof api.openRemote !== 'function') {
            hideRemoteLink();
            return;
        }
        try {
            const link = await api.getBrowserLink(root, status.branch || '');
            if (request !== remoteRequest || root !== projectRoot) {
                return;
            }
            const remote = gitRemoteIcons[link?.provider];
            if (!remote || !link?.url) {
                hideRemoteLink();
                return;
            }
            const [glyph, name] = remote;
            elements.gitRemote.textContent = String.fromCodePoint(glyph);
            elements.gitRemote.dataset.url = link.url;
            elements.gitRemote.dataset.tooltip = `Open ${name} repository`;
            elements.gitRemote.setAttribute('aria-label', `Open ${name} repository`);
            elements.gitRemote.hidden = false;
        } catch {
            if (request === remoteRequest) {
                hideRemoteLink();
            }
        }
    };

    const applyFileFilter = async () => {
        const request = ++fileFilterRequest;
        const status = currentStatus;
        const root = projectRoot;
        const query = String(elements.gitFileFilter?.value || '').trim();
        const rows = [...elements.gitFileList.querySelectorAll('.notes-git-file-row')];
        const result = query ? await api.filterStrings(query, rows.map((row) => row.dataset.path)) : null;
        if (request !== fileFilterRequest || status !== currentStatus || root !== projectRoot) return;
        elements.gitFileList.querySelector('.notes-git-filter-empty')?.remove();
        const matches = result ? new Set(result.Error ? [] : result.List || []) : null;
        for (const row of rows) row.hidden = matches !== null && !matches.has(row.dataset.path);
        if (query && rows.length && rows.every((row) => row.hidden)) {
            const message = document.createElement('div');
            message.className = 'notes-git-message notes-git-filter-empty';
            message.textContent = 'No matching files';
            elements.gitFileList.appendChild(message);
        }
    };

    elements.gitFileFilter?.addEventListener('input', () => {
        void applyFileFilter().catch((error) => options.notify?.(String(error?.message || error), 'error'));
    });
    elements.gitFileFilter?.addEventListener('keydown', (event) => {
        if (event.key === 'Escape') {
            event.preventDefault();
            event.stopPropagation();
            elements.gitFileFilter.value = '';
            void applyFileFilter();
        }
    });

    const renderStatus = (status, cachedStatus = status) => {
        fileFilterRequest++;
        currentStatus = status;
        syncStatusWatch();
        void updateRemoteLink(status, projectRoot);
        elements.gitCompare.textContent = compareLabel || compareBranch || 'HEAD';
        statusCache.set(rootKey(projectRoot), { status: cachedStatus });
        elements.gitWorkspace.textContent = String.fromCodePoint(0xf802);
        elements.gitWorkspace.dataset.tooltip = 'Manage worktrees';
        elements.gitBranch.textContent = status.branch || 'Detached HEAD';
        elements.gitAheadBehind.textContent = [
            status.ahead > 0 ? `↑${status.ahead}` : '',
            status.behind > 0 ? `↓${status.behind}` : '',
        ].filter(Boolean).join(' ');
        options.onStatusChange?.(status, projectRoot);
        const stageableFiles = cachedStatus?.files || status.files || [];
        elements.gitStageAll.disabled = !stageableFiles.some((file) => file.unstaged && !file.untracked);

        if (commitsActive) {
            void renderCommits(true);
        }
        elements.gitFileList.replaceChildren();

        if (status.clean || !status.files?.length) {
            showMessage('Working tree clean');
            return;
        }

        const fragment = document.createDocumentFragment();
        const files = [...status.files].sort((left, right) => {
            const leftEqual = left.compareOnly && left.code === '=';
            const rightEqual = right.compareOnly && right.code === '=';
            if (leftEqual !== rightEqual) return leftEqual ? 1 : -1;
            const leftGroup = left.untracked ? 0 : left.unstaged ? 1 : 2;
            const rightGroup = right.untracked ? 0 : right.unstaged ? 1 : 2;
            return leftGroup - rightGroup || left.path.localeCompare(right.path);
        });
        for (const file of files) {
            const row = document.createElement('div');
            row.className = 'notes-git-file-row';
            row.dataset.path = file.path;
            row.setAttribute('role', 'listitem');

            const open = document.createElement('button');
            open.type = 'button';
            open.className = 'notes-git-file';
            open.dataset.path = file.path;
            open.dataset.tooltip = file.originalPath
                ? `Open diff: ${file.originalPath} → ${file.path}`
                : `Open diff: ${file.path}`;
            open.setAttribute('aria-label', open.dataset.tooltip);
            open.textContent = file.path;

            const code = document.createElement('span');
            code.className = 'notes-git-status-code';
            const comparisonEqual = file.compareOnly && file.code === '=';
            code.textContent = file.compareOnly ? (comparisonEqual ? '=' : '≠') : statusCode(file);
            code.dataset.statusCode = statusCode(file);
            code.dataset.tooltip = file.compareOnly
                ? `${comparisonEqual ? 'Matches' : 'Differs from'} ${compareBranch}`
                : statusTooltip(code.dataset.statusCode);
            code.setAttribute('aria-label', `Git status ${code.dataset.tooltip}`);

            const actions = document.createElement('span');
            actions.className = 'notes-git-file-actions';
            if (file.staged) {
                actions.appendChild(createAction('Staged', 'unstage', file.path, 'Click to unstage'));
            }
            if (file.unstaged || file.untracked) {
                const label = file.untracked ? 'Untracked' : 'Unstaged';
                const stageButton = createAction(label, 'stage', file.path, 'Click to stage');
                if (file.untracked) {
                    stageButton.classList.add('notes-git-action-untracked');
                }
                actions.appendChild(stageButton);
            }

            row.append(open, code, actions);
            fragment.appendChild(row);
        }
        elements.gitFileList.appendChild(fragment);
        void applyFileFilter().catch((error) => options.notify?.(String(error?.message || error), 'error'));
    };

    const loadCommits = (root, limit, force) => {
        const includeFiles = filenameSearch && Boolean(historyFilter.trim());
        if (!force && historySnapshot && historySnapshotLimit >= limit && (!includeFiles || historySnapshotIncludesFiles)) {
            return Promise.resolve(historySnapshot.slice(0, limit));
        }
        if (!force && historyLoad?.root === root && historyLoad.limit >= limit && (!includeFiles || historyLoad.includeFiles)) {
            return historyLoad.promise.then((commits) => commits.slice(0, limit));
        }
        const load = { root, limit, includeFiles };
        const list = includeFiles ? api.listCommitsWithFiles : api.listCommits;
        load.promise = list(root, limit).then((commits) => {
            if (historyLoad === load && root === projectRoot) {
                historySnapshot = commits;
                historySnapshotLimit = limit;
                historySnapshotIncludesFiles = includeFiles;
            }
            return commits;
        }).finally(() => {
            if (historyLoad === load) historyLoad = null;
        });
        historyLoad = load;
        return load.promise;
    };

    const renderCommits = async (force = false) => {
        const request = ++historyRequest;
        const root = projectRoot;
        try {
            const query = historyFilter.trim();
            const searchFilenames = filenameSearch;
            const limit = query ? 10000 : historyLimit;
            const commits = await loadCommits(root, limit, force);
            if (request !== historyRequest || !commitsActive || root !== projectRoot) {
                return;
            }
            historyLayoutObserver?.disconnect();
            historyLayoutObserver = null;
            if (!commits.length) {
                showMessage('No commits', 'info', elements.gitCommitList);
                return;
            }
            const searchText = (commit) => [commit.fullHash || commit.hash, commit.message || commit.subject, ...(commit.tags || [])].join('\n');
            let matches = null;
            if (query) {
                const searchItems = searchFilenames
                    ? [...new Set(commits.flatMap((commit) => commit.files || []))]
                    : commits.map(searchText);
                const result = await api.filterStrings(query, searchItems);
                if (request !== historyRequest || !commitsActive || root !== projectRoot) return;
                matches = new Set(result?.Error ? [] : result?.List || []);
                if (!matches.size) {
                    showMessage('No matching commits', 'info', elements.gitCommitList);
                    return;
                }
            }
            const width = Math.max(2, ...commits.flatMap((commit) => commit.graph.map((line) => line.length))) * 8;
            const fragment = document.createDocumentFragment();
            const historyRows = [];
            for (const [commitIndex, commit] of commits.entries()) {
                if (matches && !(searchFilenames
                    ? (commit.files || []).some((path) => matches.has(path))
                    : matches.has(searchText(commit)))) continue;
                const row = document.createElement('div');
                row.className = 'notes-find-files-item notes-git-commit';
                row.classList.toggle('notes-git-commit-filtered', Boolean(query));
                row.setAttribute('role', 'listitem');
                row.tabIndex = 0;
                row.setAttribute('aria-label', `Compare against commit ${commit.hash}: ${commit.subject}`);
                const selectCommit = () => {
                    void setCompareBranch(commit.fullHash || commit.hash, commit.hash);
                };
                row.addEventListener('click', selectCommit);
                row.addEventListener('contextmenu', (event) => {
                    event.preventDefault();
                    event.stopPropagation();
                    options.showCommitMenu?.({
                        root, commit, x: event.clientX, y: event.clientY, compare: selectCommit,
                    });
                });
                row.addEventListener('keydown', (event) => {
                    if (event.key === 'Enter' || event.key === ' ') {
                        event.preventDefault();
                        selectCommit();
                    }
                });
                row.style.setProperty('--git-graph-width', `${width}px`);
                row.appendChild(commitGraph(commit.graph, width, commits[commitIndex - 1]?.graph.at(-1), commits[commitIndex + 1]?.graph[0]));
                const content = document.createElement('div');
                content.className = 'notes-git-commit-content';
                const header = document.createElement('div');
                header.className = 'notes-git-commit-header';
                const hash = document.createElement('span');
                hash.className = 'notes-git-commit-hash';
                hash.textContent = commit.hash;
                header.appendChild(hash);
                if (commit.tags?.length) {
                    const tags = document.createElement('span');
                    tags.className = 'notes-git-commit-tags';
                    tags.textContent = commit.tags.join(', ');
                    header.appendChild(tags);
                }
                content.appendChild(header);
                for (const [className, text] of [
                    ['notes-git-commit-subject', commit.subject],
                    ['notes-git-commit-meta', commit.date],
                    ['notes-git-commit-meta', `(${commit.author})`],
                ]) {
                    const line = document.createElement('div');
                    line.className = className;
                    line.textContent = text;
                    content.appendChild(line);
                }
                row.appendChild(content);
                fragment.appendChild(row);
                historyRows.push(row);
            }
            if (!query && commits.length === historyLimit && historyLimit < 10000) {
                const more = document.createElement('button');
                more.type = 'button';
                more.className = 'notes-tools-clear notes-git-commits-more';
                more.textContent = 'Load more';
                more.addEventListener('click', () => {
                    historyLimit += 100;
                    void renderCommits();
                });
                fragment.appendChild(more);
            }
            elements.gitCommitList.replaceChildren(fragment);
            const updateOverlap = (row) => {
                const graph = row.querySelector('.notes-git-commit-graph');
                const content = row.querySelector('.notes-git-commit-content');
                const bounds = graph.getBoundingClientRect();
                if (bounds.width > 0) {
                    row.style.setProperty('--git-overlap-start', `${Math.max(0, content.getBoundingClientRect().left - bounds.left)}px`);
                }
            };
            for (const row of historyRows) updateOverlap(row);
            if (!query && typeof ResizeObserver === 'function') {
                historyLayoutObserver = new ResizeObserver((entries) => {
                    for (const entry of entries) updateOverlap(entry.target);
                });
                for (const row of historyRows) historyLayoutObserver.observe(row);
            }
        } catch (error) {
            if (request === historyRequest && commitsActive && root === projectRoot) {
                showMessage(String(error?.message || error), 'error', elements.gitCommitList);
            }
        }
    };

    const setCommitsActive = (active) => {
        const next = active === true;
        if (next === commitsActive) {
            return;
        }
        commitsActive = next;
        historyLayoutObserver?.disconnect();
        historyLayoutObserver = null;
        historyRequest++;
        historyLimit = 100;
        elements.gitCommits?.setAttribute('aria-pressed', String(commitsActive));
        if (elements.gitFilenameSearchRow) elements.gitFilenameSearchRow.hidden = !commitsActive;
        options.onCommitsChange?.(commitsActive);
        if (commitsActive) {
            historyFilter = options.getCommitFilter?.() ?? historyFilter;
            void renderCommits(true);
        } else if (currentStatus) {
            renderStatus(currentStatus, statusCache.get(rootKey(projectRoot))?.status || currentStatus);
        }
    };

    elements.gitCommits?.addEventListener('click', () => setCommitsActive(!commitsActive));

    elements.gitFilenameSearch?.addEventListener('change', () => {
        filenameSearch = elements.gitFilenameSearch.checked;
        if (commitsActive) {
            if (elements.listFilter) elements.listFilter.placeholder = filenameSearch ? 'Filter commits by filename...' : 'Filter commits...';
            void renderCommits();
        }
    });

    const refresh = async (force = true) => {
        const generation = ++requestGeneration;
        if (!projectRoot) {
            currentStatus = null;
            showMessage('Git operations are unavailable: No project root is selected.');
            setAvailable(false);
            return false;
        }

        elements.gitRefresh.disabled = true;
        try {
            const status = force && api.refreshStatus
                ? await api.refreshStatus(projectRoot)
                : await api.getStatus(projectRoot);
            if (generation !== requestGeneration) {
                return false;
            }
            let displayStatus = status;
            if (compareBranch) {
                const branch = compareBranch;
                let files;
                try {
                    if (typeof api.getBranchDiffFiles !== 'function') {
                        throw new Error('Branch comparison is unavailable');
                    }
                    files = await api.getBranchDiffFiles(projectRoot, branch);
                } catch (error) {
                    if (generation !== requestGeneration || branch !== compareBranch) {
                        return false;
                    }
                    showMessage(`Could not compare with ${branch}: ${error?.message || error}`, 'error');
                    setAvailable(true);
                    return false;
                }
                if (generation !== requestGeneration || branch !== compareBranch) {
                    return false;
                }
                displayStatus = { ...status, files, clean: files.length === 0 };
            }
            renderStatus(displayStatus, status);
            setAvailable(true);
            return true;
        } catch (error) {
            if (generation !== requestGeneration) {
                return false;
            }
            currentStatus = null;
            void updateRemoteLink(null, projectRoot);
            statusCache.set(rootKey(projectRoot), { error });
            showMessage(String(error?.message || error || 'Git is unavailable'), 'error');
            setAvailable(false);
            options.onStatusChange?.(null, projectRoot);
            return false;
        } finally {
            if (generation === requestGeneration) {
                elements.gitRefresh.disabled = false;
            }
        }
    };

    const setProjectRoot = async (root) => {
        const nextRoot = String(root || '').trim();
        if (nextRoot === projectRoot && currentStatus) {
            return true;
        }
        elements.gitCommitRow.hidden = true;
        elements.gitCommitToggle.setAttribute('aria-pressed', 'false');
        elements.gitCommitMessage.value = '';
        projectRoot = nextRoot;
        fileFilterRequest++;
        if (elements.gitFileFilter) elements.gitFileFilter.value = '';
        filenameSearch = false;
        if (elements.gitFilenameSearch) elements.gitFilenameSearch.checked = false;
        if (elements.gitFilenameSearchRow) elements.gitFilenameSearchRow.hidden = true;
        historySnapshot = null;
        historySnapshotLimit = 0;
        historyLoad = null;
        historyLayoutObserver?.disconnect();
        historyLayoutObserver = null;
        commitsActive = false;
        historyRequest++;
        historyLimit = 100;
        elements.gitCommits?.setAttribute('aria-pressed', 'false');
        options.onCommitsChange?.(false);
        compareBranch = '';
        compareLabel = '';
        elements.gitCompare.textContent = 'HEAD';
        currentStatus = null;
        setAvailable(false);
        syncStatusWatch();
        void updateRemoteLink(null, nextRoot);
        elements.gitFileList.replaceChildren();
        showMessage('Checking Git availability…');
        const cached = statusCache.get(rootKey(projectRoot));
        if (cached?.status) {
            renderStatus(cached.status);
            setAvailable(true);
        } else if (cached?.error) {
            currentStatus = null;
            showMessage(String(cached.error?.message || cached.error), 'error');
            setAvailable(false);
        }
        return refresh(false);
    };

    elements.gitWorkspace.addEventListener('click', () => {
        if (currentStatus) {
            options.showWorkspaceMenu?.(currentStatus.root || projectRoot, elements.gitWorkspace);
        }
    });

    const setCompareBranch = async (branch, label = branch) => {
        const nextBranch = String(branch || '').trim();
        if (nextBranch === compareBranch) {
            return true;
        }
        compareBranch = nextBranch;
        compareLabel = String(label || '').trim();
        elements.gitCompare.textContent = compareLabel || compareBranch || 'HEAD';
        return refresh(true);
    };

    elements.gitCompare.addEventListener('click', () => {
        if (currentStatus) {
            options.showCompareMenu?.(
                currentStatus.root || projectRoot,
                currentStatus.branch || '',
                compareBranch,
                elements.gitCompare,
                setCompareBranch,
            );
        }
    });

    elements.gitRefresh.addEventListener('click', () => {
        void refresh();
    });

    const setCommitRowVisible = (visible) => {
        elements.gitCommitRow.hidden = !visible;
        if (elements.gitFilterRow) elements.gitFilterRow.hidden = visible || elements.gitToolbar.hidden;
        elements.gitCommitToggle.setAttribute('aria-pressed', String(visible));
        if (visible) {
            elements.gitCommitMessage.focus();
        }
    };

    elements.gitCommitToggle.setAttribute('aria-pressed', 'false');
    elements.gitCommitRow.hidden = true;
    elements.gitCommitToggle.addEventListener('click', () => {
        setCommitRowVisible(elements.gitCommitRow.hidden);
    });

    const submitCommit = async () => {
        const message = String(elements.gitCommitMessage.value || '').trim();
        if (!message) {
            options.notify?.('Commit message is required', 'warn');
            elements.gitCommitMessage.focus();
            return;
        }
        if (!projectRoot || typeof api.runGitAction !== 'function') {
            return;
        }
        elements.gitCommitSubmit.disabled = true;
        try {
            const result = await api.runGitAction(projectRoot, 'commit', { message });
            const { mode, ...output } = result || {};
            if (output.error) {
                options.notify?.('Commit failed. See terminal for output', 'error');
                return;
            }
            elements.gitCommitMessage.value = '';
            elements.gitCommitMessage.dispatchEvent(new Event('input', { bubbles: true }));
            if (options.notifyTransfer) {
                options.notifyTransfer(output, 'Commit created');
            } else {
                options.notify?.('Commit created', 'info');
            }
            await refresh(true);
        } catch (error) {
            options.notify?.('Commit failed. See terminal for output', 'error');
        } finally {
            elements.gitCommitSubmit.disabled = false;
        }
    };

    elements.gitCommitSubmit.addEventListener('click', () => void submitCommit());
    elements.gitCommitMessage.addEventListener('keydown', (event) => {
        if (event.key === 'Enter' && !event.isComposing) {
            event.preventDefault();
            void submitCommit();
        }
    });

    elements.gitStageAll.addEventListener('click', async () => {
        if (!projectRoot || elements.gitStageAll.disabled || typeof api.stageAll !== 'function') {
            return;
        }
        elements.gitStageAll.disabled = true;
        try {
            await api.stageAll(projectRoot);
            options.notify?.('All tracked changes staged', 'info');
            await refresh(true);
        } catch (error) {
            options.notify?.(String(error?.message || error), 'error');
            elements.gitStageAll.disabled = false;
        }
    });

    elements.gitRemote.addEventListener('click', () => {
        const url = elements.gitRemote.dataset.url;
        if (url) {
            void api.openRemote(url);
        }
    });

    elements.gitPane.addEventListener('mouseover', (event) => {
        const target = event.target.closest('[data-tooltip]');
        if (target && elements.gitPane.contains(target)) {
            options.showTooltip?.(target.dataset.tooltip);
        }
    });

    elements.gitPane.addEventListener('mouseout', (event) => {
        const target = event.target.closest('[data-tooltip]');
        if (target && elements.gitPane.contains(target) && !target.contains(event.relatedTarget)) {
            options.hideTooltip?.();
        }
    });

    const openFileDiff = async (file) => {
        if (!file) {
            return false;
        }
        const root = projectRoot;
        const generation = requestGeneration;
        try {
            const diff = compareBranch
                ? await api.getDiffAgainstBranch(root, file.path, '', compareBranch)
                : await api.getDiff(root, file.path, file.originalPath || '');
            if (generation !== requestGeneration || root !== projectRoot) {
                return false;
            }
            await options.onOpenDiff?.({ root, file, diff });
            return true;
        } catch (error) {
            options.notify?.(String(error?.message || error || `Could not read ${file.path}`), 'error');
            return false;
        }
    };

    elements.gitFileList.addEventListener('click', async (event) => {
        const action = event.target.closest('[data-git-action]');
        if (action && elements.gitFileList.contains(action)) {
            event.stopPropagation();
            const path = action.dataset.path;
            const operationName = action.dataset.gitAction;
            const operation = operationName === 'stage' ? api.stageFile : api.unstageFile;
            action.disabled = true;
            try {
                if (options.beforeFileAction
                    && !await options.beforeFileAction({ root: projectRoot, path, action: operationName })) {
                    return;
                }
                await operation(projectRoot, path);
                await refresh();
            } catch (error) {
                options.notify?.(String(error?.message || error || 'Git operation failed'), 'error');
            } finally {
                action.disabled = false;
            }
            return;
        }

        const fileButton = event.target.closest('.notes-git-file');
        if (!fileButton || !elements.gitFileList.contains(fileButton)) {
            return;
        }
        const file = currentStatus?.files?.find((item) => item.path === fileButton.dataset.path);
        if (!file) {
            return;
        }
        fileButton.disabled = true;
        try {
            await openFileDiff(file);
        } finally {
            fileButton.disabled = false;
        }
    });

    if (typeof api.eventsOn === 'function') {
        api.eventsOn('notesContextChanged', (context) => {
            void setProjectRoot(context?.projectRoot || '');
        });
        api.eventsOn('notesGitStatusChanged', (event) => {
            const root = rootKey(event?.root);
            if (!root) {
                return;
            }
            if (event?.error) {
                statusCache.set(root, { error: event.error });
                if (root === rootKey(projectRoot)) {
                    currentStatus = null;
                    void updateRemoteLink(null, projectRoot);
                    showMessage(String(event.error), 'error');
                    setAvailable(false);
                    options.onStatusChange?.(null, projectRoot);
                }
                return;
            }
            if (!event?.status) {
                return;
            }
            statusCache.set(root, { status: event.status });
            if (root === rootKey(projectRoot)) {
                if (compareBranch) {
                    void refresh(false);
                } else {
                    renderStatus(event.status);
                    setAvailable(true);
                }
            }
        });
    }

    void api.getCurrentProject().then(setProjectRoot).catch(() => setProjectRoot(''));

    return {
        refresh,
        setProjectRoot,
        setCompareBranch,
        setCommitsActive,
        isCommitsActive: () => commitsActive,
        setCommitFilter(query) {
            historyFilter = String(query || '');
            if (commitsActive) return renderCommits();
        },
        getCompareBranch: () => compareBranch,
        setWatchActive(active) {
            watchActive = active === true;
            syncStatusWatch();
        },
        getCurrentStatus: () => currentStatus,
        getCurrentRoot: () => projectRoot,
        getCachedStatus(root = projectRoot) {
            return statusCache.get(rootKey(root))?.status || null;
        },
        getFileStatus(filePath, root = projectRoot) {
            const status = statusCache.get(rootKey(root))?.status;
            return status?.files?.find((file) => file.path === filePath) || null;
        },
        openFileDiff(filePath, root = projectRoot) {
            const status = statusCache.get(rootKey(root))?.status;
            const file = status?.files?.find((item) => item.path === filePath);
            return openFileDiff(file);
        },
    };
}

function graphConnections(upper, lower) {
    if (!upper || !lower) {
        return [];
    }
    const connections = [];
    const isLane = (character) => Boolean(character && '|*/\\.'.includes(character));
    for (const [column, character] of [...upper].entries()) {
        if (character !== '/' && character !== '\\') {
            continue;
        }
        const direction = character === '\\' ? 1 : -1;
        const opposite = character === '\\' ? '/' : '\\';
        const continuing = [column + direction, column + 2 * direction]
            .find((position) => position >= 0 && lower[position] === character);
        const target = lower[column] === opposite
            ? column
            : continuing ?? [column + direction, column + 2 * direction, column]
                .find((position) => position >= 0 && isLane(lower[position]));
        if (target !== undefined) {
            connections.push({ from: column, to: target });
        }
    }
    for (const [column, character] of [...upper].entries()) {
        if (!character || !'|*.'.includes(character)) {
            continue;
        }
        const direct = Boolean(lower[column] && '|*.'.includes(lower[column]));
        if (direct) {
            connections.push({ from: column, to: column });
        }
        if (!direct || character === '*' || character === '.') {
            if (lower[column + 1] === '\\' && !connections.some((edge) => edge.to === column + 1)) {
                connections.push({ from: column, to: column + 1 });
            }
        }
        if (lower[column - 1] === '/' && !connections.some((edge) => edge.to === column - 1)) {
            connections.push({ from: column, to: column - 1 });
        }
    }
    return connections;
}

function commitGraph(lines, width, previousLine, nextLine) {
    const namespace = 'http://www.w3.org/2000/svg';
    const graph = document.createElementNS(namespace, 'svg');
    graph.classList.add('notes-git-commit-graph');
    graph.setAttribute('viewBox', `0 0 ${width} 72`);
    graph.setAttribute('preserveAspectRatio', 'none');
    graph.setAttribute('aria-hidden', 'true');
    const step = 72 / Math.max(1, lines.length);
    const paths = [];
    const laneX = (column) => column * 8 + 4;
    for (const edge of graphConnections(previousLine, lines[0])) {
        const boundary = (laneX(edge.from) + laneX(edge.to)) / 2;
        paths.push(`M ${boundary} 0 L ${laneX(edge.to)} ${step / 2}`);
    }
    for (const [rowIndex, line] of lines.entries()) {
        const middle = (rowIndex + 0.5) * step;
        const following = lines[rowIndex + 1] ?? nextLine;
        for (const edge of graphConnections(line, following)) {
            const destinationX = rowIndex + 1 < lines.length
                ? laneX(edge.to)
                : (laneX(edge.from) + laneX(edge.to)) / 2;
            const destinationY = rowIndex + 1 < lines.length ? middle + step : 72;
            paths.push(`M ${laneX(edge.from)} ${middle} L ${destinationX} ${destinationY}`);
        }
        for (const [column, character] of [...line].entries()) {
            if (character === '*') {
                const node = document.createElementNS(namespace, 'circle');
                node.setAttribute('cx', String(laneX(column)));
                node.setAttribute('cy', String(middle));
                node.setAttribute('r', '3');
                graph.appendChild(node);
            }
            if ((character === '-' || character === '_') && line[column - 1] && line[column + 1]) {
                paths.push(`M ${laneX(column - 1)} ${middle} H ${laneX(column + 1)}`);
            }
        }
    }
    const path = document.createElementNS(namespace, 'path');
    path.setAttribute('d', paths.join(' '));
    path.setAttribute('fill', 'none');
    path.setAttribute('stroke', 'currentColor');
    path.setAttribute('stroke-width', '1');
    path.setAttribute('vector-effect', 'non-scaling-stroke');
    graph.prepend(path);
    return graph;
}

function createAction(label, action, path, tooltip) {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = `notes-git-action notes-git-action-${action}`;
    button.dataset.gitAction = action;
    button.dataset.path = path;
    button.textContent = label;
    button.dataset.tooltip = tooltip;
    button.setAttribute('aria-label', tooltip);
    return button;
}
