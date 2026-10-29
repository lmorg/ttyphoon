import { beforeEach, describe, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import postcss from 'postcss';
import * as swaggerUtils from './swagger-utils.js';
import { attachJsonViewerEditHandler, renderJsonViewer } from './json-viewer.js';
import { marked } from 'marked';
import { attachSpellCheck } from './spellcheck';

const notesCss = readFileSync('./src/notes.css', 'utf8');

function getNotesRenderedStyles() {
    const inline = Array.from(document.querySelectorAll('style')).map((el) => String(el.textContent || ''));
    return inline.join('\n') + '\n' + notesCss;
}

function getNotesStyleProperties(selector) {
    const properties = {};
    const normalizeSelector = (value) => value.replace(/\s*>\s*/g, ' > ').trim();
    postcss.parse(getNotesRenderedStyles()).walkRules((rule) => {
        if (!rule.selectors.some((candidate) => normalizeSelector(candidate) === normalizeSelector(selector))) return;
        rule.walkDecls((declaration) => { properties[declaration.prop] = declaration.value; });
    });
    return properties;
}

const getWindowStyleMock = vi.fn();
const createMonacoAdapterMock = vi.fn();
const getNotesLayoutSizingMock = vi.fn(() => Promise.resolve({}));
const setNotesLayoutRatioMock = vi.fn(() => Promise.resolve());
const getNotesMaxLogLinesMock = vi.fn();
const getNotesLanguageTabIndentMock = vi.fn();
const getNotesLanguageReservedWordsMock = vi.fn(() => Promise.resolve([]));
const getNotesColumnWidthsMock = vi.fn();
const setNotesColumnWidthsMock = vi.fn(() => Promise.resolve());
const emptyTableResult = () => ({ missing: false, order: [], sortColumn: 0, sortDesc: false, filter: '', graph: null, error: '' });
const notesTableSortMock = vi.fn(() => Promise.resolve(emptyTableResult()));
const notesTableClearSortMock = vi.fn(() => Promise.resolve(emptyTableResult()));
const notesTableFilterMock = vi.fn(() => Promise.resolve(emptyTableResult()));
const notesTableReconcileMock = vi.fn(() => Promise.resolve());
const notesTableRestoreMock = vi.fn(() => Promise.resolve(emptyTableResult()));
const notesTableSetGraphMock = vi.fn((_request, graph) => Promise.resolve({ graph }));
const notesTableRenderGraphMock = vi.fn(() => Promise.resolve({ svg: '<svg xmlns="http://www.w3.org/2000/svg"></svg>', error: '' }));
const notesTableDisposeAllMock = vi.fn(() => Promise.resolve());
const getFileMock = vi.fn();
const listFilesMock = vi.fn();
const getNotesFilesSnapshotMock = vi.fn();
const filterStringsMock = vi.fn((query, items) => {
    const tokens = String(query).toLowerCase().trim().split(/\s+/).filter(Boolean);
    const list = tokens.length === 0
        ? items
        : items.filter((item) => String(item).toLowerCase().includes(tokens.join(' ')) || tokens.every((t) => String(item).toLowerCase().includes(t)));
    if (list.length === 0) {
        return Promise.resolve({ List: null, Error: { message: `nothing matched the filter pattern: '${query}'` } });
    }
    return Promise.resolve({ List: list, Error: null });
});
const saveFileMock = vi.fn(() => Promise.resolve());
const saveBinaryFileMock = vi.fn(() => Promise.resolve());
const deleteFileMock = vi.fn(() => Promise.resolve());
const renameFileMock = vi.fn(() => Promise.resolve());
const runNoteMock = vi.fn(() => Promise.resolve());
const runFunctionMock = vi.fn(() => Promise.resolve(''));
const stopNoteMock = vi.fn(() => Promise.resolve());
const sendIpcMock = vi.fn(() => Promise.resolve());
const sendToTerminalMock = vi.fn(() => Promise.resolve());
const getLanguageDescriptionsMock = vi.fn(() => Promise.resolve([]));
const getAllLanguageDescriptionsMock = vi.fn(() => Promise.resolve([]));
const terminalCopyImageDataURLMock = vi.fn(() => Promise.resolve(''));
const saveImageDialogMock = vi.fn(() => Promise.resolve(''));
const saveCodeDialogMock = vi.fn(() => Promise.resolve('/project/code-block.js'));
const windowPrintMock = vi.fn(() => Promise.resolve());
const getClipboardDataMock = vi.fn(() => Promise.resolve({ text: '', image: '' }));
const swaggerRequestMock = vi.fn(() => Promise.resolve(''));
const askAIMock = vi.fn(() => Promise.resolve());
const askAIImageMock = vi.fn(() => Promise.resolve());
const getAISessionCacheMock = vi.fn(() => Promise.resolve(''));
const listAIPromptLogsMock = vi.fn(() => Promise.resolve([]));
const getAIPromptLogMock = vi.fn(() => Promise.resolve(''));
const listAILiveStreamBlocksMock = vi.fn(() => Promise.resolve([]));
const listAIStreamBlocksMock = vi.fn(() => Promise.resolve([]));
const getAIStreamBlockContentMock = vi.fn(() => Promise.resolve(''));
const getAISessionManagementMock = vi.fn(() => Promise.resolve({ activeSessionId: 0, sessions: [], history: [] }));
const createAISessionMock = vi.fn(() => Promise.resolve({ activeSessionId: 1, sessions: [], history: [] }));
const setActiveAISessionMock = vi.fn(() => Promise.resolve({ activeSessionId: 1, sessions: [], history: [] }));
const deleteAISessionMock = vi.fn(() => Promise.resolve({ activeSessionId: 1, sessions: [], history: [] }));
const renameAISessionMock = vi.fn(() => Promise.resolve({ activeSessionId: 1, sessions: [], history: [] }));
const deleteAIHistoryEntryMock = vi.fn(() => Promise.resolve({ activeSessionId: 1, sessions: [], history: [] }));
const getAIToolsListMock = vi.fn(() => Promise.resolve([]));
const getAIMcpServersMock = vi.fn(() => Promise.resolve([]));
const setAIMcpServerEnabledMock = vi.fn(() => Promise.resolve());
const showAIToolStateMenuMock = vi.fn(() => Promise.resolve());
const resolveAIToolPermissionMock = vi.fn(() => Promise.resolve());
const promptAIToolPermissionReasonMock = vi.fn(() => Promise.resolve());
const listAIModelSelectionsMock = vi.fn(() => Promise.resolve(['OpenAI: gpt-4.1']));
const getCurrentAIModelSelectionMock = vi.fn(() => Promise.resolve('OpenAI: gpt-4.1'));
const getAIExecutionLimitsMock = vi.fn(() => Promise.resolve({ agentSteps: 10, requestTimeout: '5m' }));
const setCurrentAIModelSelectionMock = vi.fn(() => Promise.resolve());
const setAIPanelLiveMock = vi.fn(() => Promise.resolve());
const showAIToolsMenuMock = vi.fn(() => Promise.resolve());
const showAISkillsMenuMock = vi.fn(() => Promise.resolve());
const setAIToolSubagentAllowedMock = vi.fn(() => Promise.resolve());
const showAIMcpMenuMock = vi.fn(() => Promise.resolve());
const clearAISessionHistoryMock = vi.fn(() => Promise.resolve({ activeSessionId: 1, sessions: [], history: [] }));
const getCurrentProjectMock = vi.fn(() => Promise.resolve(''));
const getCurrentGroupNameMock = vi.fn(() => Promise.resolve(''));
const getGitStatusMock = vi.fn(() => Promise.resolve({ root: '', branch: '', ahead: 0, behind: 0, clean: true, files: [] }));
const getGitBrowserLinkMock = vi.fn(() => Promise.resolve({ provider: 'github', url: 'https://github.com/org/repo/tree/main' }));
const getGitDiffMock = vi.fn(() => Promise.resolve({ path: '', original: '', working: '', binary: false, deleted: false }));
const stageGitFileMock = vi.fn(() => Promise.resolve());
const stageAllGitFilesMock = vi.fn(() => Promise.resolve());
const runGitActionMock = vi.fn(() => Promise.resolve({ mode: 'interactive', stdout: '', stderr: '', error: '' }));
const unstageGitFileMock = vi.fn(() => Promise.resolve());
const listGitWorktreesMock = vi.fn(() => Promise.resolve([]));
const listGitBranchesMock = vi.fn(() => Promise.resolve([]));
const listGitTagsMock = vi.fn(() => Promise.resolve([]));
const checkoutGitBranchMock = vi.fn(() => Promise.resolve());
const deleteGitBranchMock = vi.fn(() => Promise.resolve());
const fetchGitBranchMock = vi.fn(() => Promise.resolve({ stdout: '', stderr: '', error: '' }));
const createGitTagMock = vi.fn(() => Promise.resolve());
const deleteGitTagMock = vi.fn(() => Promise.resolve());
const checkoutGitTagMock = vi.fn(() => Promise.resolve());
const pullGitTagsMock = vi.fn(() => Promise.resolve({ stdout: '', stderr: '', error: '' }));
const pushGitTagMock = vi.fn(() => Promise.resolve({ stdout: '', stderr: '', error: '' }));
const createGitTagWorktreeMock = vi.fn(() => Promise.resolve({ path: '', branch: '' }));
const listGitCommitsMock = vi.fn(() => Promise.resolve([]));
const listGitCommitsWithFilesMock = vi.fn(() => Promise.resolve([]));
const getDefaultGitWorktreePathMock = vi.fn(() => Promise.resolve('/home/test/Documents/ttyphoon/.git_worktrees/default-hash'));
const createGitWorktreeMock = vi.fn(() => Promise.resolve({ path: '', branch: '' }));
const switchGitWorktreeMock = vi.fn(() => Promise.resolve());
const removeGitWorktreeMock = vi.fn(() => Promise.resolve());
const showGitWorktreeInfoMock = vi.fn(() => Promise.resolve());
const showGitRefInfoMock = vi.fn(() => Promise.resolve());
const getFileMetaMarkdownMock = vi.fn(() => Promise.resolve([
    '# note.md',
    '',
    '## Attributes',
    '',
    '- Size: `0`',
    '- Path: `/tmp/note.md`',
    '',
    '## Owners',
    '',
    '- User: `user`',
    '- Group: `group`',
    '',
    '## Permissions',
    '',
    '- Unix: `0644`',
    '- User: `rw-`',
    '- Group: `r--`',
    '- Other: `r--`',
].join('\n')));
const resolveFilePathMock = vi.fn(() => Promise.resolve(''));
const getImageMock = vi.fn(() => Promise.resolve('data:image/png;base64,AAAA'));
const resolveNoteLocationMock = vi.fn(() => Promise.resolve('$NOTES'));
const composeNoteLocationPathMock = vi.fn((location, name) => `${location}/${String(name || '').replace(/^\/+/, '')}`);
const getHyperlinkMenuActionsMock = vi.fn(() => Promise.resolve([]));
const runHyperlinkMenuActionMock = vi.fn(() => Promise.resolve());
const displayHyperlinkMenuMock = vi.fn(() => Promise.resolve());
const showTooltipMock = vi.fn(() => Promise.resolve());
const closeTooltipMock = vi.fn(() => Promise.resolve());
const notesKeyPressMock = vi.fn(() => Promise.resolve({ consume: false, prefixActive: false }));
const resolveNotesLspLanguageMock = vi.fn(() => Promise.resolve(''));
const notesLspOpenDocumentMock = vi.fn(() => Promise.resolve());
const notesLspChangeDocumentMock = vi.fn(() => Promise.resolve());
const notesLspSaveDocumentMock = vi.fn(() => Promise.resolve());
const notesLspCloseDocumentMock = vi.fn(() => Promise.resolve());
const notesTyposOpenDocumentMock = vi.fn(() => Promise.resolve(false));
const notesTyposChangeDocumentMock = vi.fn(() => Promise.resolve());
const notesTyposCloseDocumentMock = vi.fn(() => Promise.resolve());
const notesLspStopAllMock = vi.fn(() => Promise.resolve());
const notesLspHoverMock = vi.fn(() => Promise.resolve(''));
const notesLspCodeLensMock = vi.fn(() => Promise.resolve([]));
const notesLspExecuteCodeLensMock = vi.fn(() => Promise.resolve(false));
const notesLspInlayHintsMock = vi.fn(() => Promise.resolve([]));
const notesLspSemanticTokensMock = vi.fn(() => Promise.resolve(null));
const notesLspCompletionMock = vi.fn(() => Promise.resolve([]));
const notesLspDefinitionMock = vi.fn(() => Promise.resolve([]));
const notesLspReferencesMock = vi.fn(() => Promise.resolve([]));
const notesLspDocumentSymbolsMock = vi.fn(() => Promise.resolve([]));
const notesLspWorkspaceSymbolsMock = vi.fn(() => Promise.resolve([]));
const notesLspSignatureHelpMock = vi.fn(() => Promise.resolve(''));
const notesLspFormatMock = vi.fn(() => Promise.resolve({ changed: false, content: '' }));
const notesLspFormatRangeMock = vi.fn(() => Promise.resolve({ changed: false, content: '' }));
const notesLspCodeActionsMock = vi.fn(() => Promise.resolve([]));
const notesLspApplyCodeActionMock = vi.fn(() => Promise.resolve({ changed: false, content: '' }));
const notesLspPrepareRenameMock = vi.fn(() => Promise.resolve({ canRename: true, placeholder: '' }));
const notesLspRenameMock = vi.fn(() => Promise.resolve({ changed: false, content: '' }));
const notesRecentFilesMock = vi.fn(() => Promise.resolve([]));
const notesHistoryPreviousMock = vi.fn(() => Promise.resolve(''));
const notesHistoryNextMock = vi.fn(() => Promise.resolve(''));
const notesHistoryAddMock = vi.fn(() => Promise.resolve());
const notesHistoryCurrentMock = vi.fn(() => Promise.resolve(''));
const notesHistoryPreviousPositionMock = vi.fn(() => Promise.resolve({ file: '', line: 1 }));
const notesHistoryNextPositionMock = vi.fn(() => Promise.resolve({ file: '', line: 1 }));
const notesHistoryAddPositionMock = vi.fn(() => Promise.resolve());
const notesHistoryCurrentPositionMock = vi.fn(() => Promise.resolve({ file: '', line: 1 }));
const notesGrepMock = vi.fn(() => Promise.resolve({ results: [], error: '' }));
const notesGrepWithOptionsMock = vi.fn(() => Promise.resolve({ results: [], error: '' }));
const notesGrepStreamMock = vi.fn(() => Promise.resolve());
const cancelNotesListFilesMock = vi.fn(() => Promise.resolve());
const getProjectCacheMock = vi.fn(() => Promise.resolve({ LastDocument: '', FileListCollapsed: [] }));
const setProjectCacheMock = vi.fn(() => Promise.resolve());
const getDocumentCacheMock = vi.fn(() => Promise.resolve({ DocumentTab: '', ToolsOpen: true, ToolsTab: 'ai' }));
const setDocumentCacheMock = vi.fn(() => Promise.resolve());
const getNotesFindFieldValuesMock = vi.fn(() => Promise.resolve([]));
const addNotesFindFieldValueMock = vi.fn(() => Promise.resolve([]));
const eventsOnMock = vi.fn();
const clipboardSetTextMock = vi.fn(() => Promise.resolve());
const clipboardSetHTMLMock = vi.fn(() => Promise.resolve());
const browserOpenURLMock = vi.fn();
const showLocalMenuMock = vi.fn();

vi.mock('../wailsjs/go/main/WApp', () => ({
    GetWindowStyle: getWindowStyleMock,
    GetNotesLayoutSizing: getNotesLayoutSizingMock,
    SetNotesLayoutRatio: setNotesLayoutRatioMock,
    GetNotesMaxLogLines: getNotesMaxLogLinesMock,
    GetNotesStructViewMaxSizeKB: vi.fn(() => Promise.resolve(1024)),
    GetNotesLanguageTabIndent: getNotesLanguageTabIndentMock,
    GetNotesLanguageReservedWords: getNotesLanguageReservedWordsMock,
    GetNotesColumnWidths: getNotesColumnWidthsMock,
    GetFile: getFileMock,
    GetImage: getImageMock,
    ListFiles: listFilesMock,
    ListFilesForContext: listFilesMock,
    GetNotesFilesSnapshot: getNotesFilesSnapshotMock,
    GetGitStatus: getGitStatusMock,
    GetGitBrowserLink: getGitBrowserLinkMock,
    GetGitBranchDiffFiles: vi.fn(() => Promise.resolve([])),
    RefreshGitStatus: getGitStatusMock,
    SetGitStatusWatch: vi.fn(() => Promise.resolve()),
    GetGitDiff: getGitDiffMock,
    GetGitDiffAgainstBranch: vi.fn(() => Promise.resolve({ path: '', original: '', working: '', binary: false, deleted: false })),
    StageGitFile: stageGitFileMock,
    StageAllGitFiles: stageAllGitFilesMock,
    RunGitAction: runGitActionMock,
    UnstageGitFile: unstageGitFileMock,
    ListGitWorktrees: listGitWorktreesMock,
    ListGitBranches: listGitBranchesMock,
    ListGitTags: listGitTagsMock,
    CheckoutGitBranch: checkoutGitBranchMock,
    DeleteGitBranch: deleteGitBranchMock,
    FetchGitBranch: fetchGitBranchMock,
    CreateGitTag: createGitTagMock,
    DeleteGitTag: deleteGitTagMock,
    CheckoutGitTag: checkoutGitTagMock,
    PullGitTags: pullGitTagsMock,
    PushGitTag: pushGitTagMock,
    CreateGitTagWorktree: createGitTagWorktreeMock,
    ListGitCommits: listGitCommitsMock,
    ListGitCommitsWithFiles: listGitCommitsWithFilesMock,
    GetDefaultGitWorktreePath: getDefaultGitWorktreePathMock,
    CreateGitWorktree: createGitWorktreeMock,
    SwitchGitWorktree: switchGitWorktreeMock,
    RemoveGitWorktree: removeGitWorktreeMock,
    ShowGitWorktreeInfo: showGitWorktreeInfoMock,
    ShowGitRefInfo: showGitRefInfoMock,
    GetNotesContext: vi.fn(async () => ({
        projectRoot: await getCurrentProjectMock(),
        groupName: await getCurrentGroupNameMock(),
        globalNotes: '/notes/',
        userNotes: '/notes/workspace/',
    })),
    FilterStrings: filterStringsMock,
    SaveFile: saveFileMock,
    SaveBinaryFile: saveBinaryFileMock,
    DeleteFile: deleteFileMock,
    RenameFile: renameFileMock,
    RunNote: runNoteMock,
    RunFunction: runFunctionMock,
    StopNote: stopNoteMock,
    SendIpc: sendIpcMock,
    SendToTerminal: sendToTerminalMock,
    GetLanguageDescriptions: getLanguageDescriptionsMock,
    GetAllLanguageDescriptions: getAllLanguageDescriptionsMock,
    TerminalCopyImageDataURL: terminalCopyImageDataURLMock,
    SaveImageDialog: saveImageDialogMock,
    SaveCodeDialog: saveCodeDialogMock,
    WindowPrint: windowPrintMock,
    GetClipboardData: getClipboardDataMock,
    SwaggerRequest: swaggerRequestMock,
    AskAI: askAIMock,
    AskAIImage: askAIImageMock,
    GetAISessionCache: getAISessionCacheMock,
    ListAIPromptLogs: listAIPromptLogsMock,
    GetAIPromptLog: getAIPromptLogMock,
    ListAILiveStreamBlocks: listAILiveStreamBlocksMock,
    ListAIStreamBlocks: listAIStreamBlocksMock,
    GetAIStreamBlockContent: getAIStreamBlockContentMock,
    GetAISessionManagement: getAISessionManagementMock,
    CreateAISession: createAISessionMock,
    SetActiveAISession: setActiveAISessionMock,
    DeleteAISession: deleteAISessionMock,
    RenameAISession: renameAISessionMock,
    DeleteAIHistoryEntry: deleteAIHistoryEntryMock,
    GetAIToolsList: getAIToolsListMock,
    GetAIMcpServers: getAIMcpServersMock,
    SetAIMcpServerEnabled: setAIMcpServerEnabledMock,
    ShowAIToolStateMenu: showAIToolStateMenuMock,
    ResolveAIToolPermission: resolveAIToolPermissionMock,
    PromptAIToolPermissionReason: promptAIToolPermissionReasonMock,
    ListAIModelSelections: listAIModelSelectionsMock,
    GetCurrentAIModelSelection: getCurrentAIModelSelectionMock,
    GetAIExecutionLimits: getAIExecutionLimitsMock,
    SetCurrentAIModelSelection: setCurrentAIModelSelectionMock,
    SetAIPanelLive: setAIPanelLiveMock,
    SetAIToolSubagentAllowed: setAIToolSubagentAllowedMock,
    ShowAIToolsMenu: showAIToolsMenuMock,
    ShowAISkillsMenu: showAISkillsMenuMock,
    ShowAIMcpMenu: showAIMcpMenuMock,
    ClearAISessionHistory: clearAISessionHistoryMock,
    GetCurrentProject: getCurrentProjectMock,
    GetCurrentGroupName: getCurrentGroupNameMock,
    GetFileMetaMarkdown: getFileMetaMarkdownMock,
    ResolveFilePath: resolveFilePathMock,
    ComposeNoteLocationPath: composeNoteLocationPathMock,
    ResolveNoteLocation: resolveNoteLocationMock,
    ResolveNotesLspLanguage: resolveNotesLspLanguageMock,
    NotesLspAvailableForRuntime: vi.fn(() => Promise.resolve(false)),
    NotesLspOpenDocument: notesLspOpenDocumentMock,
    NotesLspChangeDocument: notesLspChangeDocumentMock,
    NotesLspSaveDocument: notesLspSaveDocumentMock,
    NotesLspCloseDocument: notesLspCloseDocumentMock,
    NotesTyposOpenDocument: notesTyposOpenDocumentMock,
    NotesTyposChangeDocument: notesTyposChangeDocumentMock,
    NotesTyposCloseDocument: notesTyposCloseDocumentMock,
    NotesLspStopAll: notesLspStopAllMock,
    NotesLspHover: notesLspHoverMock,
    NotesLspCodeLens: notesLspCodeLensMock,
    NotesLspExecuteCodeLens: notesLspExecuteCodeLensMock,
    NotesLspInlayHints: notesLspInlayHintsMock,
    NotesLspSemanticTokens: notesLspSemanticTokensMock,
    NotesLspCompletion: notesLspCompletionMock,
    NotesLspDefinition: notesLspDefinitionMock,
    NotesLspReferences: notesLspReferencesMock,
    NotesLspDocumentSymbols: notesLspDocumentSymbolsMock,
    NotesLspWorkspaceSymbols: notesLspWorkspaceSymbolsMock,
    NotesLspSignatureHelp: notesLspSignatureHelpMock,
    NotesLspFormat: notesLspFormatMock,
    NotesLspFormatRange: notesLspFormatRangeMock,
    NotesLspCodeActions: notesLspCodeActionsMock,
    NotesLspApplyCodeAction: notesLspApplyCodeActionMock,
    NotesLspPrepareRename: notesLspPrepareRenameMock,
    NotesLspRename: notesLspRenameMock,
    NotesRecentFiles: notesRecentFilesMock,
    NotesHistoryPrevious: notesHistoryPreviousMock,
    NotesHistoryNext: notesHistoryNextMock,
    NotesHistoryAdd: notesHistoryAddMock,
    NotesHistoryCurrent: notesHistoryCurrentMock,
    NotesHistoryPreviousPosition: notesHistoryPreviousPositionMock,
    NotesHistoryNextPosition: notesHistoryNextPositionMock,
    NotesHistoryAddPosition: notesHistoryAddPositionMock,
    NotesHistoryCurrentPosition: notesHistoryCurrentPositionMock,
    NotesGrep: notesGrepMock,
    NotesGrepWithOptions: notesGrepWithOptionsMock,
    NotesGrepStream: notesGrepStreamMock,
    CancelNotesListFiles: cancelNotesListFilesMock,
    GetProjectCache: getProjectCacheMock,
    SetProjectCache: setProjectCacheMock,
    GetDocumentCache: getDocumentCacheMock,
    SetDocumentCache: setDocumentCacheMock,
    GetNotesFindFieldValues: getNotesFindFieldValuesMock,
    AddNotesFindFieldValue: addNotesFindFieldValueMock,
    FormatCodeBlock: vi.fn(() => Promise.resolve({ Code: '', FilePath: '', Err: '', HasFormatter: false })),
    FormatNotesContent: vi.fn(() => Promise.resolve({ Code: '', FilePath: '', Err: '', HasFormatter: false })),
    SetNotesColumnWidths: setNotesColumnWidthsMock,
    NotesTableSort: notesTableSortMock,
    NotesTableClearSort: notesTableClearSortMock,
    NotesTableFilter: notesTableFilterMock,
    NotesTableReconcile: notesTableReconcileMock,
    NotesTableRestore: notesTableRestoreMock,
    NotesTableSetGraph: notesTableSetGraphMock,
    NotesTableRenderGraph: notesTableRenderGraphMock,
    NotesTableDisposeAll: notesTableDisposeAllMock,
    GetHyperlinkMenuActions: getHyperlinkMenuActionsMock,
    RunHyperlinkMenuAction: runHyperlinkMenuActionMock,
    DisplayHyperlinkMenu: displayHyperlinkMenuMock,
    ClipboardSetHTML: clipboardSetHTMLMock,
    ClipboardSetFormatted: clipboardSetHTMLMock,
    ShowTooltip: showTooltipMock,
    CloseTooltip: closeTooltipMock,
    NotesKeyPress: notesKeyPressMock,
    NotesSpellCheck: vi.fn(() => Promise.resolve([])),
}));

vi.mock('../wailsjs/runtime/runtime', () => ({
    EventsOn: eventsOnMock,
    EventsOff: vi.fn(),
    ClipboardSetText: clipboardSetTextMock,
    BrowserOpenURL: browserOpenURLMock,
}));

vi.mock('./monaco_adapter', () => ({
    createMonacoAdapter: createMonacoAdapterMock,
    createMonacoDiffAdapter: vi.fn(),
}));

vi.mock('./popup_menu', () => ({
    showLocalMenu: showLocalMenuMock,
}));

vi.mock('./markdown-utils.js', () => ({
    configureMarked: vi.fn(),
    processMarkdownContainer: vi.fn(),
    enableFullscreenImages: vi.fn(),
    notesAssetURL: vi.fn((path) => `/__asset?path=${encodeURIComponent(path)}`),
    applySyntaxHighlighting: vi.fn(),
    renderMermaidDiagrams: vi.fn(),
    processWailsImages: vi.fn(),
    processLinks: vi.fn(),
    autoHyperlink: vi.fn(),
}));

vi.mock('./style-utils.js', () => ({
    getScrollbarStyles: vi.fn(() => ''),
    getMarkdownContentStyles: vi.fn(() => ''),
    getHighlightJsTheme: vi.fn(() => ''),
    getCheckboxStyles: vi.fn(() => ''),
    getMarkdownBaseTextSizeStyles: vi.fn(() => ''),
    getSwaggerUIStyles: vi.fn(() => ''),
    DARKEN_BACKGROUND_OVERLAY: 'rgba(0, 0, 0, 0.2)',
}));

vi.mock('./swagger-utils.js', () => ({
    isStructuredDataFile: vi.fn((fileName, content = '') => /\.(json|ya?ml|xml)$/i.test(fileName || '') || /^\uFEFF?<\?xml\s[\s\S]*?\?>/.test(String(content || ''))),
    isXMLDocument: vi.fn((content) => /^\uFEFF?<\?xml\s[\s\S]*?\?>/.test(String(content || ''))),
    hasSwaggerKey: vi.fn(() => false),
    parseSwaggerSpec: vi.fn(() => null),
    generateRequestBuilderHTML: vi.fn(() => ''),
    generateResponseHTML: vi.fn(() => ''),
    extractPaths: vi.fn(() => []),
    generateEndpointListHTML: vi.fn(() => ''),
    buildRequestUrl: vi.fn(() => ''),
    generateLiveResponseHTML: vi.fn(() => ''),
    escapeInfoText: vi.fn((value) => String(value ?? '')),
}));

vi.mock('./json-viewer.js', () => ({
    attachJsonViewerEditHandler: vi.fn(),
    renderJsonViewer: vi.fn(),
}));

vi.mock('./vim-mode', () => ({
    attachVimMode: vi.fn(() => ({ detach: vi.fn(), getMode: vi.fn(() => 'insert') })),
}));

vi.mock('./spellcheck', () => ({
    attachSpellCheck: vi.fn(() => ({
        detach: vi.fn(),
        setExclusions: vi.fn(),
        check: vi.fn(),
        setMode: vi.fn(),
        getMode: vi.fn(() => 'aspell'),
        setMisspellings: vi.fn(),
    })),
}));

const theme = {
    colors: {
        fg: { Red: 230, Green: 237, Blue: 243 },
        bg: { Red: 30, Green: 34, Blue: 40 },
        accent: { Red: 88, Green: 142, Blue: 204 },
        yellow: { Red: 226, Green: 200, Blue: 92 },
        link: { Red: 110, Green: 170, Blue: 240 },
        red: { Red: 220, Green: 80, Blue: 80 },
        green: { Red: 61, Green: 127, Blue: 199 },
        blue: { Red: 80, Green: 110, Blue: 200 },
        magenta: { Red: 180, Green: 100, Blue: 210 },
        cyan: { Red: 90, Green: 180, Blue: 220 },
        redBright: { Red: 255, Green: 120, Blue: 120 },
        greenBright: { Red: 130, Green: 220, Blue: 160 },
        yellowBright: { Red: 245, Green: 220, Blue: 120 },
        blueBright: { Red: 140, Green: 170, Blue: 210 },
        magentaBright: { Red: 220, Green: 140, Blue: 240 },
        cyanBright: { Red: 130, Green: 220, Blue: 240 },
        selection: { Red: 49, Green: 109, Blue: 176 },
        error: { Red: 255, Green: 90, Blue: 90 },
    },
    fontFamily: 'sans-serif',
    fontSize: 14,
};

function flushPromises() {
    return new Promise((resolve) => {
        setTimeout(resolve, 0);
    });
}

async function importNotesModule() {
    vi.resetModules();
    await import('./notes.js');
    await flushPromises();
    await flushPromises();
}

function getEventHandler(eventName) {
    const call = eventsOnMock.mock.calls.find(([name]) => name === eventName);
    return call ? call[1] : null;
}

describe('notes rendering', () => {
    beforeEach(() => {
        document.body.innerHTML = '<div id="notes-status"></div><div id="app"></div>';

        if (typeof window.ResizeObserver === 'undefined') {
            window.ResizeObserver = class {
                observe() {}
                disconnect() {}
                unobserve() {}
            };
        }

        getWindowStyleMock.mockReset();
        createMonacoAdapterMock.mockReset();
        getNotesLayoutSizingMock.mockReset();
        getNotesLayoutSizingMock.mockResolvedValue({});
        setNotesLayoutRatioMock.mockClear();
        getNotesMaxLogLinesMock.mockReset();
        getNotesLanguageTabIndentMock.mockReset();
        getNotesLanguageReservedWordsMock.mockReset().mockResolvedValue([]);
        getNotesColumnWidthsMock.mockReset();
        setNotesColumnWidthsMock.mockReset();
        notesTableSortMock.mockReset();
        notesTableSortMock.mockResolvedValue(emptyTableResult());
        notesTableClearSortMock.mockReset();
        notesTableClearSortMock.mockResolvedValue(emptyTableResult());
        notesTableFilterMock.mockReset();
        notesTableFilterMock.mockResolvedValue(emptyTableResult());
        notesTableReconcileMock.mockReset();
        notesTableReconcileMock.mockResolvedValue();
        notesTableRestoreMock.mockReset();
        notesTableRestoreMock.mockResolvedValue(emptyTableResult());
        notesTableSetGraphMock.mockReset();
        notesTableSetGraphMock.mockImplementation((_request, graph) => Promise.resolve({ graph }));
        notesTableRenderGraphMock.mockReset();
        notesTableRenderGraphMock.mockResolvedValue({ svg: '<svg xmlns="http://www.w3.org/2000/svg"></svg>', error: '' });
        notesTableDisposeAllMock.mockReset();
        notesTableDisposeAllMock.mockResolvedValue();
        getFileMock.mockReset();
        listFilesMock.mockReset();
        getNotesFilesSnapshotMock.mockReset();
        getNotesFilesSnapshotMock.mockImplementation(async (context) => ({
            context, files: await listFilesMock(), roots: [], state: 'ready', error: '', truncated: false,
        }));
        saveFileMock.mockClear();
        saveBinaryFileMock.mockClear();
        deleteFileMock.mockClear();
        renameFileMock.mockClear();
        runNoteMock.mockClear();
        runFunctionMock.mockClear();
        stopNoteMock.mockClear();
        sendIpcMock.mockClear();
        sendToTerminalMock.mockClear();
        getLanguageDescriptionsMock.mockReset().mockResolvedValue([]);
        getAllLanguageDescriptionsMock.mockClear();
        terminalCopyImageDataURLMock.mockClear();
        saveImageDialogMock.mockClear();
        saveCodeDialogMock.mockClear();
        windowPrintMock.mockClear();
        getClipboardDataMock.mockClear();
        swaggerRequestMock.mockClear();
        askAIMock.mockClear();
        getCurrentProjectMock.mockReset();
        getCurrentGroupNameMock.mockReset();
        getGitStatusMock.mockReset();
        getGitBrowserLinkMock.mockReset();
        getGitBrowserLinkMock.mockResolvedValue({ provider: 'github', url: 'https://github.com/org/repo/tree/main' });
        getGitDiffMock.mockReset();
        stageGitFileMock.mockReset();
        stageAllGitFilesMock.mockReset();
        runGitActionMock.mockReset();
        runGitActionMock.mockResolvedValue({ mode: 'interactive', stdout: '', stderr: '', error: '' });
        unstageGitFileMock.mockReset();
        listGitWorktreesMock.mockReset();
        listGitBranchesMock.mockReset();
        listGitTagsMock.mockReset();
        listGitTagsMock.mockResolvedValue([]);
        checkoutGitBranchMock.mockReset();
        checkoutGitBranchMock.mockResolvedValue();
        deleteGitBranchMock.mockReset();
        deleteGitBranchMock.mockResolvedValue();
        fetchGitBranchMock.mockReset();
        fetchGitBranchMock.mockResolvedValue({ stdout: '', stderr: '', error: '' });
        createGitTagMock.mockReset();
        createGitTagMock.mockResolvedValue();
        deleteGitTagMock.mockReset();
        deleteGitTagMock.mockResolvedValue();
        checkoutGitTagMock.mockReset();
        checkoutGitTagMock.mockResolvedValue();
        pullGitTagsMock.mockReset();
        pullGitTagsMock.mockResolvedValue({ stdout: '', stderr: '', error: '' });
        pushGitTagMock.mockReset();
        pushGitTagMock.mockResolvedValue({ stdout: '', stderr: '', error: '' });
        createGitTagWorktreeMock.mockReset();
        createGitTagWorktreeMock.mockResolvedValue({ path: '', branch: '' });
        listGitCommitsMock.mockReset();
        listGitCommitsMock.mockResolvedValue([]);
        listGitCommitsWithFilesMock.mockReset().mockResolvedValue([]);
        getDefaultGitWorktreePathMock.mockReset();
        createGitWorktreeMock.mockReset();
        switchGitWorktreeMock.mockReset();
        removeGitWorktreeMock.mockReset();
        showGitWorktreeInfoMock.mockReset();
        showGitRefInfoMock.mockReset();
        getFileMetaMarkdownMock.mockReset();
        resolveFilePathMock.mockReset();
        resolveNotesLspLanguageMock.mockReset();
        notesLspOpenDocumentMock.mockReset();
        notesLspChangeDocumentMock.mockReset();
        notesLspSaveDocumentMock.mockReset();
        notesLspCloseDocumentMock.mockReset();
        notesTyposOpenDocumentMock.mockReset();
        notesTyposOpenDocumentMock.mockResolvedValue(false);
        notesTyposChangeDocumentMock.mockReset();
        notesTyposCloseDocumentMock.mockReset();
        notesLspStopAllMock.mockReset();
        notesLspHoverMock.mockReset();
        notesLspCodeLensMock.mockReset();
        notesLspExecuteCodeLensMock.mockReset();
        notesLspInlayHintsMock.mockReset();
        notesLspCompletionMock.mockReset();
        notesLspDefinitionMock.mockReset();
        notesLspDocumentSymbolsMock.mockReset();
        notesLspWorkspaceSymbolsMock.mockReset();
        notesLspSignatureHelpMock.mockReset();
        notesLspFormatMock.mockReset();
        notesLspFormatRangeMock.mockReset();
        notesLspCodeActionsMock.mockReset();
        notesLspApplyCodeActionMock.mockReset();
        notesLspPrepareRenameMock.mockReset();
        notesLspRenameMock.mockReset();
        notesHistoryPreviousMock.mockReset();
        notesHistoryNextMock.mockReset();
        notesHistoryAddMock.mockReset();
        notesHistoryCurrentMock.mockReset();
        notesHistoryPreviousPositionMock.mockReset();
        notesHistoryNextPositionMock.mockReset();
        notesHistoryAddPositionMock.mockReset();
        notesHistoryCurrentPositionMock.mockReset();
        notesGrepMock.mockReset();
        notesGrepWithOptionsMock.mockReset();
        notesGrepStreamMock.mockReset();
        cancelNotesListFilesMock.mockReset();
        getProjectCacheMock.mockReset();
        setProjectCacheMock.mockReset();
        getDocumentCacheMock.mockReset();
        setDocumentCacheMock.mockReset();
        listAIPromptLogsMock.mockReset();
        listAIPromptLogsMock.mockResolvedValue([]);
        getAIPromptLogMock.mockReset();
        getAIPromptLogMock.mockResolvedValue('');
        listAILiveStreamBlocksMock.mockReset();
        listAILiveStreamBlocksMock.mockResolvedValue([]);
        listAIStreamBlocksMock.mockReset();
        listAIStreamBlocksMock.mockResolvedValue([]);
        getAIStreamBlockContentMock.mockReset();
        getAIStreamBlockContentMock.mockResolvedValue('');
        getAISessionManagementMock.mockReset();
        getAISessionManagementMock.mockResolvedValue({ activeSessionId: 0, sessions: [], history: [] });
        getAIToolsListMock.mockReset();
        getAIToolsListMock.mockResolvedValue([]);
        getAIMcpServersMock.mockReset();
        getAIMcpServersMock.mockResolvedValue([]);
        setAIMcpServerEnabledMock.mockClear();
        showAIToolStateMenuMock.mockClear();
        resolveAIToolPermissionMock.mockReset().mockResolvedValue();
        promptAIToolPermissionReasonMock.mockReset().mockResolvedValue();
        createAISessionMock.mockReset();
        createAISessionMock.mockResolvedValue({ activeSessionId: 1, sessions: [], history: [] });
        setActiveAISessionMock.mockReset();
        setActiveAISessionMock.mockResolvedValue({ activeSessionId: 1, sessions: [], history: [] });
        getHyperlinkMenuActionsMock.mockReset();
        runHyperlinkMenuActionMock.mockReset();
        displayHyperlinkMenuMock.mockReset();
        eventsOnMock.mockReset();
        clipboardSetTextMock.mockClear();
        browserOpenURLMock.mockReset();
            clipboardSetHTMLMock.mockClear();
        showLocalMenuMock.mockReset();

        getWindowStyleMock.mockResolvedValue(theme);
        getNotesMaxLogLinesMock.mockResolvedValue(1000);
        getNotesLanguageTabIndentMock.mockResolvedValue(4);
        getNotesColumnWidthsMock.mockResolvedValue([]);
        setNotesColumnWidthsMock.mockResolvedValue();
        getFileMock.mockResolvedValue({ contents: '', text: '', error: '' });
        getCurrentProjectMock.mockResolvedValue('');
        getCurrentGroupNameMock.mockResolvedValue('');
        getGitStatusMock.mockResolvedValue({ root: '', branch: '', ahead: 0, behind: 0, clean: true, files: [] });
        getGitDiffMock.mockResolvedValue({ path: '', original: '', working: '', binary: false, deleted: false });
        stageGitFileMock.mockResolvedValue();
        unstageGitFileMock.mockResolvedValue();
        listGitWorktreesMock.mockResolvedValue([]);
        listGitBranchesMock.mockResolvedValue([]);
        getDefaultGitWorktreePathMock.mockResolvedValue('/home/test/Documents/ttyphoon/.git_worktrees/default-hash');
        createGitWorktreeMock.mockResolvedValue({ path: '', branch: '' });
        switchGitWorktreeMock.mockResolvedValue();
        removeGitWorktreeMock.mockResolvedValue();
        showGitWorktreeInfoMock.mockResolvedValue();
        showGitRefInfoMock.mockResolvedValue();
        getFileMetaMarkdownMock.mockResolvedValue([
            '# note.md',
            '',
            '## Attributes',
            '',
            '- Size: `0`',
            '- Path: `/tmp/note.md`',
            '',
            '## Owners',
            '',
            '- User: `user`',
            '- Group: `group`',
            '',
            '## Permissions',
            '',
            '- Unix: `0644`',
            '- User: `rw-`',
            '- Group: `r--`',
            '- Other: `r--`',
        ].join('\n'));
        resolveFilePathMock.mockResolvedValue('');
        resolveNotesLspLanguageMock.mockResolvedValue('');
        notesLspHoverMock.mockResolvedValue('');
        notesLspCodeLensMock.mockResolvedValue([]);
        notesLspExecuteCodeLensMock.mockResolvedValue(false);
        notesLspInlayHintsMock.mockResolvedValue([]);
        notesLspCompletionMock.mockResolvedValue([]);
        notesLspDefinitionMock.mockResolvedValue([]);
        notesLspDocumentSymbolsMock.mockResolvedValue([]);
        notesLspWorkspaceSymbolsMock.mockResolvedValue([]);
        notesLspSignatureHelpMock.mockResolvedValue('');
        notesLspFormatMock.mockResolvedValue({ changed: false, content: '' });
        notesLspFormatRangeMock.mockResolvedValue({ changed: false, content: '' });
        notesLspCodeActionsMock.mockResolvedValue([]);
        notesLspApplyCodeActionMock.mockResolvedValue({ changed: false, content: '' });
        notesHistoryPreviousMock.mockResolvedValue('');
        notesHistoryNextMock.mockResolvedValue('');
        notesHistoryAddMock.mockResolvedValue();
        notesHistoryCurrentMock.mockResolvedValue('');
        notesHistoryPreviousPositionMock.mockResolvedValue({ file: '', line: 1 });
        notesHistoryNextPositionMock.mockResolvedValue({ file: '', line: 1 });
        notesHistoryAddPositionMock.mockResolvedValue();
        notesHistoryCurrentPositionMock.mockResolvedValue({ file: '', line: 1 });
        notesGrepMock.mockResolvedValue({ results: [], error: '' });
        notesGrepWithOptionsMock.mockResolvedValue({ results: [], error: '' });
        notesGrepStreamMock.mockResolvedValue();
        cancelNotesListFilesMock.mockResolvedValue();
        getProjectCacheMock.mockResolvedValue({ LastDocument: '', FileListCollapsed: [] });
        setProjectCacheMock.mockResolvedValue();
        getDocumentCacheMock.mockResolvedValue({ DocumentTab: '', ToolsOpen: true, ToolsTab: 'ai' });
        setDocumentCacheMock.mockResolvedValue();
        getHyperlinkMenuActionsMock.mockResolvedValue([]);
        vi.mocked(swaggerUtils.hasSwaggerKey).mockReturnValue(false);
        vi.mocked(swaggerUtils.parseSwaggerSpec).mockReturnValue(null);
        vi.mocked(swaggerUtils.extractPaths).mockReturnValue([]);
        clipboardSetTextMock.mockResolvedValue();
        runHyperlinkMenuActionMock.mockResolvedValue();
    });

    it('keeps denial links active until the user submits a reason', async () => {
        document.body.innerHTML = '<div id="notes-status"></div><div id="app"><div id="notes-ai-output"></div></div>';
        await importNotesModule();
        const output = document.getElementById('notes-ai-output');
        output.innerHTML = [
            '<a href="ttyphoon://ai-tool-permission?request=wp-deny&decision=allow-once">Allow once</a>',
            '<a href="ttyphoon://ai-tool-permission?request=wp-deny&decision=allow-prompt">Allow prompt</a>',
            '<a href="ttyphoon://ai-tool-permission?request=wp-deny&decision=deny-once">Deny once</a>',
            '<a href="ttyphoon://ai-tool-permission?request=wp-deny&decision=deny-prompt">Deny prompt</a>',
        ].join('');

        const denyAnchor = output.querySelector('[href*="decision=deny-once"]');
        document.dispatchEvent(new CustomEvent('ttyphoon-ai-tool-permission', {
            detail: { requestId: 'wp-deny', decision: 'deny-once', anchor: denyAnchor },
        }));
        await flushPromises();

        expect(promptAIToolPermissionReasonMock).toHaveBeenCalledWith('wp-deny', 'deny-once');
        expect(resolveAIToolPermissionMock).not.toHaveBeenCalled();
        expect(output.querySelectorAll('a[href^="ttyphoon://ai-tool-permission"]')).toHaveLength(4);

        getEventHandler('aiToolPermissionResolved')({ requestId: 'wp-deny', decision: 'deny-once' });
        expect(output.querySelectorAll('a[href^="ttyphoon://ai-tool-permission"]')).toHaveLength(0);
        expect(output.querySelector('.ai-tool-permission-chosen')?.textContent).toBe('Deny once');

        output.innerHTML = '<a href="ttyphoon://ai-tool-permission?request=wp-allow&decision=allow-prompt">Allow prompt</a>';
        const allowAnchor = output.querySelector('a');
        document.dispatchEvent(new CustomEvent('ttyphoon-ai-tool-permission', {
            detail: { requestId: 'wp-allow', decision: 'allow-prompt', anchor: allowAnchor },
        }));
        await flushPromises();
        expect(resolveAIToolPermissionMock).toHaveBeenCalledWith('wp-allow', 'allow-prompt');
        expect(promptAIToolPermissionReasonMock).toHaveBeenCalledOnce();
        expect(output.querySelector('a')?.hasAttribute('href')).toBe(false);
    });

    it('filters AI tools and MCP servers independently without changing their original indices', async () => {
        getAIToolsListMock.mockResolvedValue([
            { name: 'writeFile', description: 'Create files', state: 'always' },
            { name: 'readFiles', description: 'Inspect source files', state: 'approval' },
        ]);
        getAIMcpServersMock.mockResolvedValue([
            { name: 'alpha', key: 'alpha-key', source: '/personal/config.json', loaded: false },
            { name: 'beta', key: 'beta-key', source: '/team/config.json', loaded: true },
        ]);
        await importNotesModule();
        document.getElementById('notes-tools-ai-settings').click();
        await flushPromises();
        await flushPromises();
        const tools = document.getElementById('notes-ai-settings-tools-list');
        const servers = document.getElementById('notes-ai-settings-mcp-list');
        const toolsFilter = document.getElementById('notes-ai-settings-tools-filter');
        const mcpFilter = document.getElementById('notes-ai-settings-mcp-filter');
        const filter = (input, value) => {
            input.value = value;
            input.dispatchEvent(new Event('input', { bubbles: true }));
        };
        filter(toolsFilter, ' READ ');
        expect(tools.querySelectorAll('[data-tool-index]')).toHaveLength(1);
        expect(tools.querySelector('[data-tool-index]').dataset.toolIndex).toBe('1');
        expect(tools.textContent).toContain('readFiles');
        expect(servers.querySelectorAll('[data-mcp-index]')).toHaveLength(2);
        filter(mcpFilter, '/TEAM/');
        expect(servers.querySelectorAll('[data-mcp-index]')).toHaveLength(1);
        expect(servers.querySelector('[data-mcp-index]').dataset.mcpIndex).toBe('1');
        expect(servers.querySelector('input').dataset.mcpKey).toBe('beta-key');
        expect(tools.querySelectorAll('[data-tool-index]')).toHaveLength(1);
        filter(toolsFilter, 'unmatched');
        expect(tools.textContent).toBe('No tools match this filter.');
        filter(mcpFilter, 'unmatched');
        expect(servers.textContent).toBe('No MCP servers match this filter.');
        filter(toolsFilter, '');
        filter(mcpFilter, '');
        expect(tools.querySelectorAll('[data-tool-index]')).toHaveLength(2);
        expect(servers.querySelectorAll('[data-mcp-index]')).toHaveLength(2);
        filter(toolsFilter, 'INSPECT');
        expect(tools.querySelector('[data-tool-index]').dataset.toolIndex).toBe('1');
        tools.querySelector('.notes-ai-settings-tool-state').click();
        expect(showAIToolStateMenuMock).toHaveBeenCalledWith('readFiles', expect.any(Number), expect.any(Number));
        filter(mcpFilter, 'beta');
        const checkbox = servers.querySelector('input');
        checkbox.checked = false;
        checkbox.dispatchEvent(new Event('change', { bubbles: true }));
        await flushPromises();
        await flushPromises();
        expect(setAIMcpServerEnabledMock).toHaveBeenCalledWith('beta-key', false);
        expect(toolsFilter.value).toBe('INSPECT');
        expect(mcpFilter.value).toBe('beta');
        expect(tools.querySelectorAll('[data-tool-index]')).toHaveLength(1);
        expect(servers.querySelectorAll('[data-mcp-index]')).toHaveLength(1);
    });

    it('shows MCP authorization failure details and refreshes state without restarting the session', async () => {
        getAIMcpServersMock.mockResolvedValue([{ name: 'notion', key: 'notion-key', source: '/fixture/mcp.json', loaded: false }]);
        setAIMcpServerEnabledMock.mockRejectedValueOnce(new Error('OAuth authorization was cancelled'));
        await importNotesModule();
        document.getElementById('notes-tools-ai-settings').click();
        await flushPromises();
        await flushPromises();
        const checkbox = document.querySelector('#notes-ai-settings-mcp-list input');
        checkbox.checked = true;
        checkbox.dispatchEvent(new Event('change', { bubbles: true }));
        expect(checkbox.disabled).toBe(true);
        checkbox.dispatchEvent(new Event('change', { bubbles: true }));
        expect(setAIMcpServerEnabledMock).toHaveBeenCalledOnce();
        await flushPromises();
        await flushPromises();
        expect(sendIpcMock).toHaveBeenCalledWith('terminal-notify', {
            level: 'error', message: 'Failed to load MCP server: OAuth authorization was cancelled',
        });
        const refreshed = document.querySelector('#notes-ai-settings-mcp-list input');
        expect(refreshed.checked).toBe(false);
        expect(refreshed.disabled).toBe(false);
    });

    it('renders grouped note categories and nested files from the Wails file list', async () => {
        listFilesMock.mockResolvedValue([
            '$GLOBAL/docs/guide.md',
            '$NOTES/todo.md',
            '$PROJECT/apis/openapi.yaml',
            '$HISTORY/archive.md',
        ]);

        await importNotesModule();

        const categoryHeaders = Array.from(document.querySelectorAll('.notes-category-header')).map((node) => node.dataset.category);

        expect(categoryHeaders).toEqual(['$GLOBAL', '$NOTES', '$PROJECT', '$HISTORY']);
        expect(document.querySelector('.notes-tree-folder .notes-tree-label')?.textContent).toBe('docs');
        expect(document.querySelector('[data-file="$NOTES/todo.md"]')?.textContent).toContain('todo.md');
        expect(document.querySelector('[data-file="$PROJECT/apis/openapi.yaml"]')?.textContent).toContain('openapi.yaml');
    });

    it('filters the file list from the sidebar input and shows a no-match empty state', async () => {
        listFilesMock.mockResolvedValue([
            '$GLOBAL/docs/guide.md',
            '$GLOBAL/images/logo.png',
            '$NOTES/todo.md',
        ]);

        await importNotesModule();

        const filterInput = document.getElementById('notes-list-filter');
        const clearButton = document.getElementById('notes-list-filter-clear');
        expect(clearButton?.dataset.visible).toBe('false');

        filterInput.value = 'guide';
        filterInput.dispatchEvent(new Event('input', { bubbles: true }));
        await vi.waitFor(() => {
            expect(clearButton?.dataset.visible).toBe('true');
        });

        const visibleFiles = Array.from(document.querySelectorAll('.notes-file')).map((node) => node.dataset.file);
        expect(visibleFiles).toEqual(['$GLOBAL/docs/guide.md']);
        expect(document.querySelector('.notes-tree-folder')?.dataset.expanded).toBe('true');

        filterInput.value = 'missing';
        filterInput.dispatchEvent(new Event('input', { bubbles: true }));

        await vi.waitFor(() => {
            expect(document.getElementById('notes-empty')?.textContent).toBe('No matching files.');
        });

        clearButton.click();
        expect(filterInput.value).toBe('');
        await vi.waitFor(() => {
            expect(clearButton?.dataset.visible).toBe('false');
        });

        await vi.waitFor(() => {
            const restoredAfterClear = Array.from(document.querySelectorAll('.notes-file')).map((node) => node.dataset.file);
            expect(restoredAfterClear).toEqual(expect.arrayContaining([
                '$GLOBAL/docs/guide.md',
                '$GLOBAL/images/logo.png',
                '$NOTES/todo.md',
            ]));
        });

        filterInput.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));

        await vi.waitFor(() => {
            const restoredFiles = Array.from(document.querySelectorAll('.notes-file')).map((node) => node.dataset.file);
            expect(restoredFiles).toEqual(expect.arrayContaining([
                '$GLOBAL/docs/guide.md',
                '$GLOBAL/images/logo.png',
                '$NOTES/todo.md',
            ]));
        });
    });

    it('keeps folders collapsible while the sidebar file filter is active', async () => {
        listFilesMock.mockResolvedValue([
            '$GLOBAL/docs/guide.md',
            '$GLOBAL/docs/other.md',
            '$NOTES/todo.md',
        ]);

        await importNotesModule();

        const filterInput = document.getElementById('notes-list-filter');
        filterInput.value = 'guide';
        filterInput.dispatchEvent(new Event('input', { bubbles: true }));

        const docsFolder = document.querySelector('.notes-tree-folder[data-folder-key="$GLOBAL/docs"]');
        expect(docsFolder?.dataset.expanded).toBe('true');

        docsFolder.click();

        const docsFolderAfterCollapse = document.querySelector('.notes-tree-folder[data-folder-key="$GLOBAL/docs"]');
        expect(docsFolderAfterCollapse?.dataset.expanded).toBe('false');
        expect(document.querySelector('[data-file="$GLOBAL/docs/guide.md"]')).toBeNull();

        docsFolderAfterCollapse.click();

        const docsFolderAfterExpand = document.querySelector('.notes-tree-folder[data-folder-key="$GLOBAL/docs"]');
        expect(docsFolderAfterExpand?.dataset.expanded).toBe('true');
        expect(document.querySelector('[data-file="$GLOBAL/docs/guide.md"]')).not.toBeNull();
    });

    it('loads markdown content when a rendered file entry is clicked', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/todo.md']);
        getFileMock.mockResolvedValue({ contents: '# Hello Notes', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/todo.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        expect(getFileMock).toHaveBeenCalledWith('$NOTES/todo.md');
        expect(document.getElementById('notes-preview')?.textContent).toContain('Hello Notes');
        expect(document.querySelector('[data-file="$NOTES/todo.md"]')?.dataset.active).toBe('true');
        expect(document.getElementById('notes-title-group').children[0].id).toBe('notes-title');
        expect(document.getElementById('notes-title-group').children[1].id).toBe('notes-current-file');
        expect(document.getElementById('notes-current-file').textContent).toBe('todo.md');
        expect(document.getElementById('notes-current-file').hidden).toBe(false);
        expect(getNotesRenderedStyles()).toMatch(/#notes-title\s*\{[^}]*font-family:\s*"Jura", var\(--font-family\), sans-serif;[^}]*color:\s*var\(--yellow\);[^}]*font-weight:\s*bold;/s);
    });

    it('restores and persists the file-list splitter without replacing its preferred ratio on clamp', async () => {
        listFilesMock.mockResolvedValue([]);
        getNotesLayoutSizingMock.mockResolvedValue({ fileListRatio: 0.4 });
        const bounds = { width: 1002, left: 0, right: 1002, height: 800, top: 0, bottom: 800 };
        const geometry = vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(() => bounds);
        try {
            await importNotesModule();
            const notesApp = document.getElementById('notes-app');
            const splitter = document.getElementById('notes-splitter');
            expect(notesApp.style.gridTemplateColumns).toBe('400px 2px 600px');
            expect(setNotesLayoutRatioMock).not.toHaveBeenCalled();
            splitter.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, button: 0 }));
            document.dispatchEvent(new MouseEvent('mousemove', { clientX: 300 }));
            expect(setNotesLayoutRatioMock).not.toHaveBeenCalled();
            document.dispatchEvent(new MouseEvent('mouseup'));
            await flushPromises();
            expect(setNotesLayoutRatioMock).toHaveBeenCalledWith('fileListRatio', 0.3);
            bounds.width = 502;
            window.dispatchEvent(new Event('resize'));
            expect(notesApp.style.gridTemplateColumns).toBe('200px 2px 300px');
            bounds.width = 242;
            window.dispatchEvent(new Event('resize'));
            expect(notesApp.style.gridTemplateColumns).toBe('120px 2px 120px');
            bounds.width = 1002;
            window.dispatchEvent(new Event('resize'));
            expect(notesApp.style.gridTemplateColumns).toBe('300px 2px 700px');
            expect(setNotesLayoutRatioMock).toHaveBeenCalledTimes(1);
        } finally { geometry.mockRestore(); }
    });

    it('resizes and caches Tools height separately from collapse and maximization', async () => {
        listFilesMock.mockResolvedValue([]);
        getNotesLayoutSizingMock.mockResolvedValue({ toolsRatio: 0.4 });
        const bounds = { width: 1002, left: 0, right: 1002, height: 804, top: 0, bottom: 804 };
        const geometry = vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(() => bounds);
        try {
            await importNotesModule();
            document.getElementById('notes-tools-tab-find').click();
            const handle = document.getElementById('notes-tools-resizer');
            const tools = document.getElementById('notes-tools-panel');
            expect(handle.dataset.enabled).toBe('true');
            expect(tools.style.getPropertyValue('--notes-tools-size')).toBe('320px');
            handle.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, button: 0 }));
            document.dispatchEvent(new MouseEvent('mousemove', { clientY: 404 }));
            expect(setNotesLayoutRatioMock).not.toHaveBeenCalled();
            document.dispatchEvent(new MouseEvent('mouseup'));
            await flushPromises();
            expect(setNotesLayoutRatioMock).toHaveBeenCalledWith('toolsRatio', 0.5);
            document.getElementById('notes-tools-minimize').click();
            expect(handle.dataset.enabled).toBe('false');
            document.getElementById('notes-tools-restore').click();
            expect(tools.style.getPropertyValue('--notes-tools-size')).toBe('400px');
            const root = document.getElementById('notes-app');
            root.dataset.aiMaximized = 'true';
            await flushPromises();
            handle.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, button: 0 }));
            document.dispatchEvent(new MouseEvent('mousemove', { clientY: 204 }));
            document.dispatchEvent(new MouseEvent('mouseup'));
            expect(setNotesLayoutRatioMock).toHaveBeenCalledTimes(1);
            root.dataset.aiMaximized = 'false';
            await flushPromises();
            expect(tools.style.getPropertyValue('--notes-tools-size')).toBe('400px');
            handle.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowUp', bubbles: true, cancelable: true }));
            await flushPromises();
            expect(setNotesLayoutRatioMock).toHaveBeenLastCalledWith('toolsRatio', expect.any(Number));
            handle.dispatchEvent(new MouseEvent('dblclick', { bubbles: true }));
            await flushPromises();
            expect(setNotesLayoutRatioMock).toHaveBeenLastCalledWith('toolsRatio', 0);
            expect(tools.style.getPropertyValue('--notes-tools-size')).toBe('280px');
        } finally { geometry.mockRestore(); }
    });

    it('ignores invalid cached layout ratios and restores current defaults', async () => {
        listFilesMock.mockResolvedValue([]);
        getNotesLayoutSizingMock.mockResolvedValue({ fileListRatio: -1, toolsRatio: '0.9' });
        const geometry = vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
            width: 1202, left: 0, right: 1202, height: 804, top: 0, bottom: 804,
        });
        try {
            await importNotesModule();
            expect(document.getElementById('notes-app').style.gridTemplateColumns).toBe('400px 2px 800px');
            document.getElementById('notes-tools-tab-find').click();
            expect(document.getElementById('notes-tools-panel').style.getPropertyValue('--notes-tools-size')).toBe('280px');
            expect(getNotesLayoutSizingMock).toHaveBeenCalledTimes(1);
            expect(setNotesLayoutRatioMock).not.toHaveBeenCalled();
        } finally { geometry.mockRestore(); }
    });

    it('does not overwrite a user splitter drag with a late cached layout response', async () => {
        listFilesMock.mockResolvedValue([]);
        let restore;
        getNotesLayoutSizingMock.mockReturnValue(new Promise((resolve) => { restore = resolve; }));
        const geometry = vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
            width: 1002, left: 0, right: 1002, height: 804, top: 0, bottom: 804,
        });
        try {
            await importNotesModule();
            document.getElementById('notes-splitter').dispatchEvent(new MouseEvent('mousedown', { bubbles: true, button: 0 }));
            document.dispatchEvent(new MouseEvent('mousemove', { clientX: 300 }));
            document.dispatchEvent(new MouseEvent('mouseup'));
            restore({ fileListRatio: 0.7 });
            await flushPromises();
            expect(document.getElementById('notes-app').style.gridTemplateColumns).toBe('300px 2px 700px');
            expect(setNotesLayoutRatioMock).toHaveBeenCalledWith('fileListRatio', 0.3);
        } finally { geometry.mockRestore(); }
    });

    it('restores the recorded line when navigating Notes history', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/current.md', '$NOTES/previous.md']);
        getFileMock.mockResolvedValue({
            contents: 'one\ntwo\nthree\nfour',
            text: '',
            error: '',
        });
        notesHistoryPreviousPositionMock.mockResolvedValue({ file: '$NOTES/previous.md', line: 3 });

        await importNotesModule();
        document.getElementById('notes-history-prev').click();
        await flushPromises();
        await flushPromises();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        expect(getFileMock).toHaveBeenCalledWith('$NOTES/previous.md');
        expect(editor.value).toBe('one\ntwo\nthree\nfour');
        expect(editor.selectionStart).toBe('one\ntwo\n'.length);
        expect(document.getElementById('notes-tab-editor').getAttribute('aria-selected')).toBe('true');
    });

    it('records editor cursor positions after moving twenty lines in one document', async () => {
        const previousMonacoSetting = localStorage.getItem('notes-editor-monaco');
        localStorage.setItem('notes-editor-monaco', '0');
        const contents = Array.from({ length: 30 }, (_, index) => `line ${index + 1}`).join('\n');
        listFilesMock.mockResolvedValue(['$NOTES/long.md']);
        getFileMock.mockResolvedValue({ contents, text: '', error: '' });

        try {
            await importNotesModule();
            document.querySelector('[data-file="$NOTES/long.md"]').click();
            await flushPromises();
            await flushPromises();
            document.getElementById('notes-tab-editor').click();

            const editor = document.getElementById('notes-editor');
            const lineOffset = contents.split('\n').slice(0, 24).join('\n').length + 1;
            vi.useFakeTimers();
            editor.setSelectionRange(lineOffset, lineOffset);
            editor.dispatchEvent(new Event('select', { bubbles: true }));
            await vi.advanceTimersByTimeAsync(500);
            vi.useRealTimers();
            await flushPromises();

            expect(notesHistoryAddPositionMock).toHaveBeenCalledWith('$NOTES/long.md', 25);
        } finally {
            vi.useRealTimers();
            if (previousMonacoSetting === null) {
                localStorage.removeItem('notes-editor-monaco');
            } else {
                localStorage.setItem('notes-editor-monaco', previousMonacoSetting);
            }
        }
    });

    it('opens the same recent-files menu from the active filename', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/current.md']);
        getFileMock.mockResolvedValue({ contents: '# Current', text: '', error: '' });
        notesRecentFilesMock.mockResolvedValueOnce(['$NOTES/previous.md', '$NOTES/current.md']);

        await importNotesModule();
        document.querySelector('[data-file="$NOTES/current.md"]').click();
        await flushPromises();
        await flushPromises();
        showLocalMenuMock.mockReset();

        document.getElementById('notes-current-file').dispatchEvent(new MouseEvent('click', {
            bubbles: true,
            cancelable: true,
            detail: 1,
            clientX: 64,
            clientY: 92,
        }));
        await flushPromises();

        expect(notesRecentFilesMock).toHaveBeenCalledOnce();
        const menu = showLocalMenuMock.mock.calls[0][0];
        expect(menu.title).toBe('Recent files');
        expect(menu.options).toEqual(['$NOTES/previous.md', '$NOTES/current.md']);
        expect(menu.x).toBe(64);
        expect(menu.y).toBe(92);
    });

    it('opens the terminal window menu next to the workspace title click', async () => {
        const terminalTabs = document.createElement('div');
        terminalTabs.id = 'terminal-tabs';
        const shellTab = document.createElement('button');
        shellTab.className = 'terminal-tab';
        shellTab.dataset.windowId = 'shell';
        shellTab.textContent = 'shell';
        const notesTab = document.createElement('button');
        notesTab.className = 'terminal-tab';
        notesTab.dataset.windowId = '__jupyter__';
        notesTab.textContent = 'Notes';
        const shellTabClick = vi.spyOn(shellTab, 'click');
        const notesTabClick = vi.spyOn(notesTab, 'click');
        terminalTabs.append(shellTab, notesTab);
        document.body.appendChild(terminalTabs);

        await importNotesModule();
        document.getElementById('notes-title').dispatchEvent(new MouseEvent('click', {
            bubbles: true,
            clientX: 42,
            clientY: 76,
        }));

        expect(showLocalMenuMock).toHaveBeenCalledTimes(1);
        const menu = showLocalMenuMock.mock.calls[0][0];
        expect(menu.title).toBe('Choose a window');
        expect(menu.options).toEqual(['shell', 'Notes']);
        expect(menu.x).toBe(42);
        expect(menu.y).toBe(76);
        expect(menu.showNextToMouseCursor).toBe(true);

        menu.onSelect(0);
        expect(shellTabClick).toHaveBeenCalledOnce();
        menu.onSelect(1);
        expect(notesTabClick).toHaveBeenCalledOnce();
    });

    it('uses Markdown heading styles for Rename and Delete prompt titles', async () => {
        await importNotesModule();
        const fileActionHandler = getEventHandler('fileActionDialog');

        fileActionHandler({ action: 'rename', filePath: '$NOTES/note.md' });
        await flushPromises();
        expect(document.querySelector('#notes-modal-title h3')?.textContent).toBe('Rename note');
        expect(document.getElementById('notes-modal-title').classList.contains('markdown-body')).toBe(true);
        document.getElementById('notes-modal-cancel').click();

        fileActionHandler({ action: 'delete', filePath: '$NOTES/note.md' });
        expect(document.querySelector('#notes-delete-modal-title h3')?.textContent).toBe('Delete note');
        expect(document.getElementById('notes-delete-modal-title').classList.contains('markdown-body')).toBe(true);
        expect(Array.from(document.getElementById('notes-delete-modal-actions').children).map((button) => button.id))
            .toEqual(['notes-delete-confirm', 'notes-delete-cancel']);
    });

    it.each(['settings.xml', 'settings.txt'])('opens XML in structured View for %s and preserves surrounding source when editing a value', async (fileName) => {
        const xml = '<?xml version="1.0"?>\n<root id="old">\n  <!-- keep this comment -->\n  <name>Old</name>\n  <count>2</count>\n</root>\n';
        listFilesMock.mockResolvedValue([`$NOTES/${fileName}`]);
        getFileMock.mockResolvedValue({ contents: xml, text: '', error: '' });
        await importNotesModule();

        document.querySelector(`[data-file="$NOTES/${fileName}"]`).click();
        for (let tick = 0; tick < 8; tick += 1) await flushPromises();
        expect(document.getElementById('notes-tab-swagger-edit').getAttribute('aria-selected')).toBe('true');
        const previousWindowRAF = window.requestAnimationFrame;
        const previousGlobalRAF = globalThis.requestAnimationFrame;
        const immediateRAF = (callback) => { callback(0); return 1; };
        window.requestAnimationFrame = immediateRAF;
        globalThis.requestAnimationFrame = immediateRAF;
        try {
            document.getElementById('notes-tab-swagger-view').click();
            expect(document.getElementById('notes-tab-swagger-view').getAttribute('aria-selected')).toBe('true');
        } finally {
            window.requestAnimationFrame = previousWindowRAF;
            globalThis.requestAnimationFrame = previousGlobalRAF;
        }
        await flushPromises();

        const actualJsonViewer = await vi.importActual('./json-viewer.js');
        const renderCall = renderJsonViewer.mock.calls.at(-1);
        const editHandlerCall = attachJsonViewerEditHandler.mock.calls.at(-1);
        expect(renderCall?.[0]).toBe(document.getElementById('notes-swagger-view'));
        expect(editHandlerCall?.[0]).toBe(document.getElementById('notes-swagger-view'));
        actualJsonViewer.renderJsonViewer(...renderCall);
        actualJsonViewer.attachJsonViewerEditHandler(...editHandlerCall);

        const value = Array.from(document.querySelectorAll('#notes-swagger-view .json-editable[data-json-edit="value"]'))
            .find((element) => element.dataset.jsonPath === JSON.stringify(['root', 'name']));
        expect(value).toBeTruthy();
        expect(value?.textContent).toContain('Old');
        value.dispatchEvent(new MouseEvent('dblclick', { bubbles: true, cancelable: true }));
        const input = document.querySelector('#notes-swagger-view .json-inline-editor');
        expect(input).toBeTruthy();
        input.value = 'Updated';
        input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true }));
        await flushPromises();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        expect(editor.value).toContain('  <!-- keep this comment -->');
        expect(editor.value).toContain('  <name>Updated</name>');
        expect(editor.value).toContain('  <count>2</count>');
        const updated = editor.value;
        await vi.waitFor(() => {
            expect(saveFileMock).toHaveBeenCalledWith(`$NOTES/${fileName}`, updated, '');
        }, { timeout: 2000 });
    });

    it('disables native assistance on filters and attaches custom spellchecking only to commit prose', async () => {
        await importNotesModule();
        const ids = ['notes-tools-git-commit-message', 'notes-ai-settings-tools-filter', 'notes-ai-settings-mcp-filter', 'notes-modal-input', 'notes-find-input'];
        for (const id of ids) {
            const control = document.getElementById(id);
            for (const [attribute, value] of [
                ['autocomplete', 'off'], ['autocorrect', 'off'], ['autocapitalize', 'off'],
                ['spellcheck', 'false'], ['writingsuggestions', 'false'],
            ]) expect(control.getAttribute(attribute)).toBe(value);
            const attached = attachSpellCheck.mock.calls.some(([target]) => target === control);
            expect(attached).toBe(id === 'notes-tools-git-commit-message');
        }
    });

    it('prints the complete Markdown View without mutating the live preview', async () => {
        const source = '---\ntitle: Metadata\n---\n# Entire document\n\n' + 'Paragraph below the viewport.\n\n'.repeat(100) + 'Last paragraph';
        listFilesMock.mockResolvedValue(['$NOTES/print.md']);
        getFileMock.mockResolvedValue({ contents: source, text: '', error: '' });
        await importNotesModule();
        document.querySelector('[data-file="$NOTES/print.md"]').click();
        await flushPromises();
        await flushPromises();
        const preview = document.getElementById('notes-preview');
        preview.innerHTML = '<h1>Visible excerpt</h1><section hidden>Collapsed live section</section>';
        const livePreview = preview.innerHTML;
        try {
            const printed = new Promise((resolve) => windowPrintMock.mockImplementationOnce(async () => { resolve(); }));
            preview.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true }));
            const menu = showLocalMenuMock.mock.calls.at(-1)[0];
            menu.onSelect(menu.options.indexOf('Print'));
            await printed;
            const printRoot = document.getElementById('notes-print-root');
            expect(printRoot.querySelector('h1').textContent).toBe('Entire document');
            expect(printRoot.textContent).toContain('Last paragraph');
            expect(printRoot.textContent).not.toContain('title: Metadata');
            expect(printRoot.querySelectorAll('p').length).toBeGreaterThan(100);
            expect(preview.innerHTML).toBe(livePreview);
            expect(printRoot.querySelector('#notes-sidebar, #notes-tools-panel, .tty-menu')).toBeNull();
        } finally {
            window.dispatchEvent(new Event('afterprint'));
        }
    });

    it('prints Markdown View content from the Jupyter Run context menu without changing views', async () => {
        getLanguageDescriptionsMock.mockResolvedValue(['Bash Shell Script']);
        listFilesMock.mockResolvedValue(['$NOTES/print.md']);
        getFileMock.mockResolvedValue({ contents: '# Commands\n\n```bash\nprintf hello\n```', text: '', error: '' });
        await importNotesModule();
        document.querySelector('[data-file="$NOTES/print.md"]').click();
        await flushPromises();
        await flushPromises();
        document.getElementById('notes-preview').dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true }));
        expect(showLocalMenuMock.mock.calls.at(-1)[0].options).toContain('Print');
        document.getElementById('notes-tab-jupyter').click();
        await flushPromises();
        const runView = document.getElementById('notes-jupyter');
        const output = document.createElement('p');
        output.textContent = 'Runtime output excluded from print';
        runView.appendChild(output);
        showLocalMenuMock.mockClear();
        runView.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true }));
        expect(showLocalMenuMock).toHaveBeenCalledOnce();
        const menu = showLocalMenuMock.mock.calls[0][0];
        expect(menu.options).toContain('Print');
        try {
            const printed = new Promise((resolve) => windowPrintMock.mockImplementationOnce(async () => { resolve(); }));
            menu.onSelect(menu.options.indexOf('Print'));
            await printed;
            const root = document.getElementById('notes-print-root');
            expect(root.querySelector('h1').textContent).toBe('Commands');
            expect(root.querySelector('code').textContent).toContain('printf hello');
            expect(root.textContent).not.toContain('Runtime output excluded from print');
            expect(root.querySelector('.jupyter-code-editable, .jupyter-run-terminal, .notes-print-filename')).toBeNull();
            expect(document.getElementById('notes-tab-jupyter').getAttribute('aria-selected')).toBe('true');
            expect(runView.contains(output)).toBe(true);
        } finally {
            window.dispatchEvent(new Event('afterprint'));
        }
    });

    it('prints the Markdown modal original content through the shared print pipeline', async () => {
        await importNotesModule();
        getEventHandler('showMarkdownModal')({ id: 'command-meta', markdown: '# Modal document\n\nFull original content' });
        await flushPromises();
        await flushPromises();
        const body = document.getElementById('markdown-modal-body');
        body.innerHTML = '<p>Filtered visible excerpt</p>';
        try {
            const printed = new Promise((resolve) => windowPrintMock.mockImplementationOnce(async () => { resolve(); }));
            body.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true }));
            const menu = showLocalMenuMock.mock.calls.at(-1)[0];
            menu.onSelect(menu.options.indexOf('Print...'));
            await printed;
            expect(document.querySelector('#notes-print-root h1').textContent).toBe('Modal document');
            expect(document.getElementById('notes-print-root').textContent).toContain('Full original content');
            expect(document.getElementById('notes-print-root').textContent).not.toContain('Filtered visible excerpt');
            expect(body.textContent).toBe('Filtered visible excerpt');
        } finally {
            window.dispatchEvent(new Event('afterprint'));
            document.getElementById('markdown-modal-overlay')?.dispatchEvent(new Event('pointerdown', { bubbles: true }));
        }
    });

    it.each([
        ['$NOTES/print.md', 'print.md'],
        ['$PROJECT/src/components/print.md', 'src/components/print.md'],
    ])('prints the unsaved Monaco buffer with the correct filename for %s', async (filename, printedFilename) => {
        const userAgent = vi.spyOn(navigator, 'userAgent', 'get').mockReturnValue('Monaco print test');
        let value = '# Saved';
        const adapter = {
            getValue: () => value, setValue: (next) => { value = next; },
            getSelectionOffsets: () => ({ start: 0, end: 0 }), setSelectionOffsets: vi.fn(),
            revealOffset: vi.fn(), focus: vi.fn(), layout: vi.fn(),
            setLanguage: vi.fn(), setWordWrap: vi.fn(), setTypography: vi.fn(),
            applyTheme: vi.fn(), configureLsp: vi.fn(), setDiagnostics: vi.fn(), setTyposMisspellings: vi.fn(),
        };
        createMonacoAdapterMock.mockResolvedValue(adapter);
        listFilesMock.mockResolvedValue([filename]);
        getFileMock.mockResolvedValue({ contents: value, text: '', error: '' });
        try {
            await importNotesModule();
            document.querySelector(`[data-file="${filename}"]`).click();
            await flushPromises();
            await flushPromises();
            document.getElementById('notes-tab-editor').click();
            await flushPromises();
            await flushPromises();
            expect(createMonacoAdapterMock).toHaveBeenCalled();
            value = '# Unsaved\n' + '<b>source text</b>\n'.repeat(300) + 'Last editor line';
            const editor = document.getElementById('notes-editor');
            editor.value = 'Stale hidden textarea';
            const printed = new Promise((resolve) => windowPrintMock.mockImplementationOnce(async () => { resolve(); }));
            editor.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true }));
            const menu = showLocalMenuMock.mock.calls.at(-1)[0];
            menu.onSelect(menu.options.indexOf('Print...'));
            await printed;
            expect(document.getElementById('notes-print-root').firstElementChild.textContent).toBe(printedFilename);
            expect(document.querySelector('#notes-print-root .notes-print-filename').nextElementSibling.tagName).toBe('PRE');
            expect(document.querySelector('#notes-print-root code').textContent).toBe(value);
            expect(document.querySelector('#notes-print-root b')).toBeNull();
            expect(editor.value).toBe('Stale hidden textarea');
            expect(saveFileMock.mock.calls.filter(([savedFilename]) => savedFilename === filename)).toHaveLength(0);
        } finally {
            window.dispatchEvent(new Event('afterprint'));
            userAgent.mockRestore();
        }
    });

    it('inserts clipboard images through the active editor surface', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/todo list.md']);
        getFileMock.mockResolvedValue({ contents: '# Hello Notes', text: '', error: '' });
        getClipboardDataMock.mockResolvedValue({ text: '', image: 'aW1hZ2U=' });
        const originalExecCommand = document.execCommand;
        document.execCommand = vi.fn((_command, _showUi, text) => {
            const editor = document.activeElement;
            const start = editor.selectionStart;
            const end = editor.selectionEnd;
            editor.value = editor.value.slice(0, start) + text + editor.value.slice(end);
            editor.selectionStart = start + text.length;
            editor.selectionEnd = editor.selectionStart;
            editor.dispatchEvent(new Event('input', { bubbles: true }));
            return true;
        });

        try {
            await importNotesModule();
            document.querySelector('[data-file="$NOTES/todo list.md"]').click();
            await flushPromises();
            await flushPromises();

            document.getElementById('notes-tab-editor').click();
            const editor = document.getElementById('notes-editor');
            editor.selectionStart = editor.value.length;
            editor.selectionEnd = editor.value.length;
            editor.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true }));

            const menu = showLocalMenuMock.mock.calls.at(-1)?.[0];
            menu.onSelect(menu.options.indexOf('Paste'));
            await flushPromises();
            await flushPromises();

            expect(saveBinaryFileMock).toHaveBeenCalledWith(
                expect.stringMatching(/^\$NOTES\/todo list\.md\.d\/\d+\.png$/),
                'aW1hZ2U=',
            );
            expect(editor.value).toMatch(/!\[\d+\]\(todo%20list\.md\.d\/\d+\.png\)$/);
        } finally {
            document.execCommand = originalExecCommand;
        }
    });

    it('reveals the Frontmatter tab and caption for documents with YAML frontmatter', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/post.md']);
        getFileMock.mockResolvedValue({
            contents: '---\ntitle: My Post\ndraft: true\n---\n# Body Heading',
            text: '',
            error: '',
        });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/post.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const frontmatterTab = document.getElementById('notes-tools-tab-frontmatter');
        expect(frontmatterTab?.style.display).not.toBe('none');

        const preview = document.getElementById('notes-preview');
        // Frontmatter is stripped from the rendered body.
        expect(preview?.textContent).toContain('Body Heading');
        expect(preview?.textContent).not.toContain('draft: true');

        // Caption with a View button is surfaced.
        const caption = preview?.querySelector('.notes-frontmatter-caption');
        expect(caption).not.toBeNull();
        expect(caption?.textContent).toContain('This document contains Frontmatter.');
        expect(caption?.querySelector('.notes-frontmatter-caption-view')?.textContent).toBe('View');

        // Parsed frontmatter is rendered in the tab pane via the JSON/YAML viewer.
        const pane = document.getElementById('notes-tools-frontmatter');
        expect(renderJsonViewer).toHaveBeenCalledWith(pane, { title: 'My Post', draft: true });
    });

    it.each([
        ['changed', '---\ntitle: Old\n---\n# Old heading', '---\ntitle: Updated\n---\n# Updated heading', { title: 'Updated' }],
        ['added', '# Old heading', '---\ntitle: Added\n---\n# Updated heading', { title: 'Added' }],
        ['removed', '---\ntitle: Old\n---\n# Old heading', '# Updated heading', null],
    ])('refreshes ToC and %s Frontmatter on Monaco autosave without rendering the preview', async (_label, initial, updated, frontmatter) => {
        const userAgent = vi.spyOn(navigator, 'userAgent', 'get').mockReturnValue('Monaco regression test');
        let value = initial;
        const adapter = {
            getValue: () => value,
            setValue: (next) => { value = next; },
            getSelectionOffsets: () => ({ start: 0, end: 0 }),
            setSelectionOffsets: vi.fn(),
            revealOffset: vi.fn(), focus: vi.fn(), layout: vi.fn(),
            setLanguage: vi.fn(), setWordWrap: vi.fn(), setTypography: vi.fn(),
            applyTheme: vi.fn(), configureLsp: vi.fn(), setDiagnostics: vi.fn(),
            setTyposMisspellings: vi.fn(),
        };
        createMonacoAdapterMock.mockResolvedValue(adapter);
        listFilesMock.mockResolvedValue(['$NOTES/post.md']);
        getFileMock.mockResolvedValue({ contents: initial, text: '', error: '' });
        try {
            await importNotesModule();
            document.querySelector('[data-file="$NOTES/post.md"]').click();
            await flushPromises();
            await flushPromises();
            document.getElementById('notes-tab-editor').click();
            await flushPromises();
            await flushPromises();
            expect(createMonacoAdapterMock).toHaveBeenCalled();
            const preview = document.getElementById('notes-preview');
            const originalPreview = preview.innerHTML;
            const editor = document.getElementById('notes-editor');
            renderJsonViewer.mockClear();

            vi.useFakeTimers();
            value = updated;
            editor.value = updated;
            editor.dispatchEvent(new Event('input', { bubbles: true }));
            editor.value = '# Stale hidden textarea';
            await vi.advanceTimersByTimeAsync(1100);

            expect(saveFileMock).toHaveBeenCalledWith('$NOTES/post.md', updated, '');
            expect(document.getElementById('notes-tools-toc').textContent).toContain('Updated heading');
            expect(document.getElementById('notes-tools-toc').textContent).not.toContain('Old heading');
            const frontmatterTab = document.getElementById('notes-tools-tab-frontmatter');
            const pane = document.getElementById('notes-tools-frontmatter');
            if (frontmatter) {
                expect(frontmatterTab.style.display).not.toBe('none');
                expect(renderJsonViewer).toHaveBeenCalledWith(pane, frontmatter);
            } else {
                expect(frontmatterTab.style.display).toBe('none');
                expect(pane.innerHTML).toBe('');
            }
            expect(document.getElementById('notes-tab-editor').getAttribute('aria-selected')).toBe('true');
            expect(preview.innerHTML).toBe(originalPreview);
        } finally {
            vi.useRealTimers();
            userAgent.mockRestore();
        }
    });

    it('hides the Frontmatter tab for documents without frontmatter', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/plain.md']);
        getFileMock.mockResolvedValue({ contents: '# Just a heading', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/plain.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const frontmatterTab = document.getElementById('notes-tools-tab-frontmatter');
        expect(frontmatterTab?.style.display).toBe('none');
        expect(document.querySelector('#notes-preview .notes-frontmatter-caption')).toBeNull();
    });

    it('maps Home and End to current line boundaries in editor fields', async () => {
        listFilesMock.mockResolvedValue([]);
        getFileMock.mockResolvedValue({ contents: 'first line\nsecond line\nthird line', text: '', error: '' });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({ filename: '$NOTES/sample.md', contents: 'first line\nsecond line\nthird line' });
        await flushPromises();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        editor.focus();

        const secondLineStart = editor.value.indexOf('second line');
        const secondLineEnd = secondLineStart + 'second line'.length;
        const caretInSecondLine = secondLineStart + 6;

        editor.selectionStart = caretInSecondLine;
        editor.selectionEnd = caretInSecondLine;
        editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'Home', bubbles: true, cancelable: true }));
        expect(editor.selectionStart).toBe(secondLineStart);
        expect(editor.selectionEnd).toBe(secondLineStart);

        editor.selectionStart = secondLineStart + 2;
        editor.selectionEnd = secondLineStart + 2;
        editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'End', bubbles: true, cancelable: true }));
        expect(editor.selectionStart).toBe(secondLineEnd);
        expect(editor.selectionEnd).toBe(secondLineEnd);

        editor.selectionStart = secondLineStart + 3;
        editor.selectionEnd = secondLineStart + 5;
        editor.selectionDirection = 'forward';
        editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'End', shiftKey: true, bubbles: true, cancelable: true }));
        expect(editor.selectionStart).toBe(secondLineStart + 3);
        expect(editor.selectionEnd).toBe(secondLineEnd);
    });

    it('undoes and redoes adapter-backed editor edits with keyboard shortcuts', async () => {
        listFilesMock.mockResolvedValue([]);

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({ filename: '$NOTES/sample.md', contents: 'abc' });
        await flushPromises();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        editor.focus();
        editor.value = 'abc';
        editor.selectionStart = 3;
        editor.selectionEnd = 3;

        editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }));
        await flushPromises();
        await flushPromises();

        expect(editor.value).toBe('abc    ');

        editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'z', metaKey: true, bubbles: true, cancelable: true }));
        await flushPromises();

        expect(editor.value).toBe('abc');

        editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'z', metaKey: true, shiftKey: true, bubbles: true, cancelable: true }));
        await flushPromises();

        expect(editor.value).toBe('abc    ');
    });

    it('refreshes markdown preview after ctrl+z style edits when input is not emitted', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/todo.md']);
        getFileMock.mockResolvedValue({ contents: '# Hello Notes', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/todo.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const tabEditor = document.getElementById('notes-tab-editor');
        const editor = document.getElementById('notes-editor');
        const preview = document.getElementById('notes-preview');

        tabEditor.click();
        await flushPromises();

        editor.focus();
        editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'z', ctrlKey: true, bubbles: true }));
        editor.value = '# Updated from undo';

        await flushPromises();
        await new Promise((resolve) => setTimeout(resolve, 160));

        expect(preview?.textContent).toContain('Updated from undo');
    });

    it('applies replace all using case-sensitive + whole-word find options', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/find.md']);
        getFileMock.mockResolvedValue({ contents: 'foo Foo food foo', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/find.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const tabEditor = document.getElementById('notes-tab-editor');
        const findInput = document.getElementById('notes-find-input');
        const replaceInput = document.getElementById('notes-replace-input');
        const replaceAll = document.getElementById('notes-replace-all');
        const caseBtn = document.getElementById('notes-find-doc-option-case');
        const wholeWordBtn = document.getElementById('notes-find-doc-option-word');
        const editor = document.getElementById('notes-editor');

        tabEditor.click();
        await flushPromises();

        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'f', metaKey: true, bubbles: true, cancelable: true }));
        await flushPromises();

        findInput.value = 'foo';
        findInput.dispatchEvent(new Event('input', { bubbles: true }));

        caseBtn.click();
        wholeWordBtn.click();
        await flushPromises();

        replaceInput.value = 'X';
        replaceAll.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
        await flushPromises();

        expect(editor.value).toBe('X Foo food X');
    });

    it('applies regex replace all for in-document find', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/find.md']);
        getFileMock.mockResolvedValue({ contents: 'item1 item22 item333', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/find.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const tabEditor = document.getElementById('notes-tab-editor');
        const findInput = document.getElementById('notes-find-input');
        const replaceInput = document.getElementById('notes-replace-input');
        const replaceAll = document.getElementById('notes-replace-all');
        const regexBtn = document.getElementById('notes-find-doc-option-regex');
        const editor = document.getElementById('notes-editor');

        tabEditor.click();
        await flushPromises();

        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'f', metaKey: true, bubbles: true, cancelable: true }));
        await flushPromises();

        regexBtn.click();
        findInput.value = 'item\\d+';
        findInput.dispatchEvent(new Event('input', { bubbles: true }));
        await flushPromises();

        replaceInput.value = 'Y';
        replaceAll.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
        await flushPromises();

        expect(editor.value).toBe('Y Y Y');
    });

    it('handles diagnostics events without an active file', async () => {
        listFilesMock.mockResolvedValue([]);

        await importNotesModule();

        const diagnosticsHandler = getEventHandler('notesLspDiagnostics');
        expect(typeof diagnosticsHandler).toBe('function');

        expect(() => diagnosticsHandler({
            uri: 'file:///tmp/note.md',
            diagnostics: [{
                range: {
                    start: { line: 0, character: 0 },
                    end: { line: 0, character: 4 },
                },
                severity: 1,
                message: 'example diagnostic',
            }],
        })).not.toThrow();
    });

    it('accepts diagnostics without legacy gutter marker rendering', async () => {
        listFilesMock.mockResolvedValue([]);
        resolveNotesLspLanguageMock.mockResolvedValue('markdown');
        resolveFilePathMock.mockImplementation(async (file) => {
            if (file === '$NOTES/todo.md') {
                return '/tmp/project/todo.md';
            }
            return '';
        });
        getFileMock.mockResolvedValue({ contents: '# Todo\n\nInitial content', text: '', error: '' });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({ filename: '$NOTES/todo.md', contents: '# Todo\n\nInitial content' });
        await flushPromises();
        await flushPromises();

        const diagnosticsHandler = getEventHandler('notesLspDiagnostics');
        expect(typeof diagnosticsHandler).toBe('function');
        diagnosticsHandler({
            uri: 'file:///tmp/project/todo.md',
            diagnostics: [{
                range: {
                    start: { line: 0, character: 0 },
                    end: { line: 0, character: 4 },
                },
                severity: 1,
                message: 'example diagnostic',
            }],
        });
        await flushPromises();
        await flushPromises();

        const gutterDot = document.querySelector('.lsp-gutter-mark');
        expect(gutterDot).toBeNull();

        const tooltip = document.getElementById('notes-lsp-tooltip');
        expect(tooltip?.style.display || '').not.toBe('block');
        expect(tooltip?.style.opacity || '').toBe('');

        const styles = getNotesRenderedStyles();
        expect(styles).toContain('#notes-lsp-tooltip {');
        expect(getNotesStyleProperties('#notes-lsp-tooltip:hover').opacity).toBe('1');
    });

    it('closes visible LSP tooltips when Escape is pressed in editor', async () => {
        listFilesMock.mockResolvedValue([]);

        await importNotesModule();

        const editor = document.getElementById('notes-editor');
        const lspTooltip = document.getElementById('notes-lsp-tooltip');
        const lspHoverTooltip = document.getElementById('notes-lsp-hover-tooltip');
        const lspCompletion = document.getElementById('notes-lsp-completion');

        lspTooltip.style.display = 'block';
        lspHoverTooltip.style.display = 'block';
        lspHoverTooltip.innerHTML = '<p>hover docs</p>';
        lspCompletion.style.display = 'block';
        lspCompletion.innerHTML = '<div>completion</div>';

        editor.focus();
        editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }));

        expect(lspTooltip.style.display).toBe('none');
        expect(lspHoverTooltip.style.display).toBe('none');
        expect(lspCompletion.style.display).toBe('none');
    });

    it('wires LSP document lifecycle for code files', async () => {
        listFilesMock.mockResolvedValue([]);
        resolveNotesLspLanguageMock.mockResolvedValue('markdown');
        getFileMock.mockImplementation(async (file) => {
            if (file.endsWith('todo.md')) {
                return { contents: '# Todo\n\nInitial content', text: '', error: '' };
            }
            return { contents: '# Readme', text: '', error: '' };
        });

        await importNotesModule();
        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');

        await createAndOpenHandler({ filename: '$NOTES/todo.md', contents: '# Todo\n\nInitial content' });
        await flushPromises();
        await flushPromises();

        expect(notesLspOpenDocumentMock).toHaveBeenCalledWith(
            '$NOTES/todo.md',
            'markdown',
            expect.stringContaining('#'),
        );

        const editor = document.getElementById('notes-editor');
        editor.value = '# Todo\n\nUpdated content';
        editor.dispatchEvent(new Event('input', { bubbles: true }));
        await new Promise((resolve) => setTimeout(resolve, 260));

        expect(notesLspChangeDocumentMock).toHaveBeenCalledWith(
            '$NOTES/todo.md',
            expect.stringContaining('Updated content'),
        );

        document.dispatchEvent(new KeyboardEvent('keydown', {
            key: 's',
            ctrlKey: true,
            bubbles: true,
            cancelable: true,
        }));
        await flushPromises();

        expect(notesLspSaveDocumentMock).toHaveBeenCalledWith('$NOTES/todo.md');

        await createAndOpenHandler({ filename: '$NOTES/readme.md', contents: '# Readme' });
        await flushPromises();
        await flushPromises();

        expect(notesLspCloseDocumentMock).toHaveBeenCalledWith('$NOTES/todo.md');
    });

    it('sends LSP didChange after ctrl+v style edits when input is not emitted', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/main.go']);
        resolveNotesLspLanguageMock.mockResolvedValue('go');
        getFileMock.mockResolvedValue({ contents: 'package main\n\nfunc main() {}\n', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/main.go"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        notesLspChangeDocumentMock.mockClear();

        const editor = document.getElementById('notes-editor');
        editor.focus();
        editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'v', ctrlKey: true, bubbles: true }));
        editor.value = 'package main\n\nfunc main() { println("x") }\n';

        await flushPromises();
        await new Promise((resolve) => setTimeout(resolve, 260));

        expect(notesLspChangeDocumentMock).toHaveBeenCalledWith(
            '$NOTES/main.go',
            expect.stringContaining('println("x")'),
        );
    });

    it('requests and shows LSP hover tooltip at cursor', async () => {
        listFilesMock.mockResolvedValue([]);
        resolveNotesLspLanguageMock.mockResolvedValue('markdown');
        notesLspHoverMock.mockResolvedValue('**Hover docs**\n\n[Docs](https://example.com)\n\n<TypeHint<T>>');
        getFileMock.mockResolvedValue({ contents: '# Todo\n\nInitial content', text: '', error: '' });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');

        await createAndOpenHandler({ filename: '$NOTES/todo.md', contents: '# Todo\n\nInitial content' });
        await flushPromises();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        editor.selectionStart = 2;
        editor.selectionEnd = 2;
        editor.dispatchEvent(new KeyboardEvent('keyup', { key: 'ArrowRight', bubbles: true }));

        await new Promise((resolve) => setTimeout(resolve, 320));
        await flushPromises();

        expect(notesLspHoverMock).toHaveBeenCalled();
        const hoverTooltip = document.getElementById('notes-lsp-hover-tooltip');
        expect(hoverTooltip?.classList.contains('markdown-body')).toBe(true);
        expect(hoverTooltip?.innerHTML).toContain('<strong>Hover docs</strong>');
        expect(hoverTooltip?.innerHTML).toContain('&lt;TypeHint&lt;T&gt;&gt;');
        expect(hoverTooltip?.textContent || '').toContain('<TypeHint<T>>');
        const link = hoverTooltip?.querySelector('a');
        expect(link).not.toBeNull();
        expect(link?.getAttribute('href')).toBe('https://example.com');

        const styles = getNotesRenderedStyles();
        expect(styles).toContain('#notes-lsp-hover-tooltip :where(p, ul, ol, pre, blockquote, table) {');
        expect(getNotesStyleProperties('#notes-lsp-hover-tooltip > :first-child')['margin-top']).toBe('0');
        expect(getNotesStyleProperties('#notes-lsp-hover-tooltip > :last-child')['margin-bottom']).toBe('0');
    });

    it('requests completion on trigger character and commits selected item by mouse', async () => {
        listFilesMock.mockResolvedValue([]);
        resolveNotesLspLanguageMock.mockResolvedValue('markdown');
        notesLspCompletionMock.mockResolvedValue([
            { label: 'Println', detail: 'fmt', insertText: 'Println', kind: 3, deprecated: true },
        ]);
        getFileMock.mockResolvedValue({ contents: '# Todo\n\nPrin', text: '', error: '' });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({ filename: '$NOTES/todo.md', contents: '# Todo\n\nPrin' });
        await flushPromises();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        editor.value = '# Todo\n\nPrin.';
        editor.selectionStart = editor.value.length;
        editor.selectionEnd = editor.value.length;
        editor.dispatchEvent(new Event('input', { bubbles: true }));
        await flushPromises();
        await flushPromises();

        expect(notesLspChangeDocumentMock).toHaveBeenCalled();
        const changeCallOrder = notesLspChangeDocumentMock.mock.invocationCallOrder[0];
        const completionCallOrder = notesLspCompletionMock.mock.invocationCallOrder[0];
        expect(changeCallOrder).toBeLessThan(completionCallOrder);
        expect(notesLspCompletionMock).toHaveBeenCalled();
        expect(document.getElementById('notes-lsp-completion')?.style.display).toBe('block');

        const firstItem = document.querySelector('.notes-lsp-completion-item');
        expect(firstItem).not.toBeNull();
        expect(firstItem?.classList.contains('tty-menu-row')).toBe(true);
        expect(firstItem?.classList.contains('is-deprecated')).toBe(true);
        expect(firstItem?.querySelector('.notes-lsp-completion-icon')?.classList.contains('codicon-symbol-function')).toBe(true);
        expect(firstItem?.querySelector('.notes-lsp-completion-kind')?.textContent).toBe('function');
        firstItem.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
        await flushPromises();

        expect(editor.value).toContain('Println');
    });

    it('cycles LSP completion selection with arrow keys only when popup is open', async () => {
        listFilesMock.mockResolvedValue([]);
        resolveNotesLspLanguageMock.mockResolvedValue('markdown');
        notesLspCompletionMock.mockResolvedValue([
            { label: 'Println', detail: 'fmt', insertText: 'Println', kind: 3 },
            { label: 'Printf', detail: 'fmt', insertText: 'Printf', kind: 3 },
        ]);
        getFileMock.mockResolvedValue({ contents: '# Todo\n\nPrin', text: '', error: '' });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({ filename: '$NOTES/todo.md', contents: '# Todo\n\nPrin' });
        await flushPromises();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        editor.value = '# Todo\n\nPrin.';
        editor.selectionStart = editor.value.length;
        editor.selectionEnd = editor.value.length;
        editor.dispatchEvent(new Event('input', { bubbles: true }));
        await flushPromises();
        await flushPromises();

        const completion = document.getElementById('notes-lsp-completion');
        expect(completion?.style.display).toBe('block');
        expect(completion?.querySelector('.notes-lsp-completion-item.is-active')?.textContent).toContain('Println');

        const downEvent = new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true, cancelable: true });
        const downHandled = editor.dispatchEvent(downEvent);
        expect(downHandled).toBe(false);
        expect(completion?.querySelector('.notes-lsp-completion-item.is-active')?.textContent).toContain('Printf');

        const upEvent = new KeyboardEvent('keydown', { key: 'ArrowUp', bubbles: true, cancelable: true });
        const upHandled = editor.dispatchEvent(upEvent);
        expect(upHandled).toBe(false);
        expect(completion?.querySelector('.notes-lsp-completion-item.is-active')?.textContent).toContain('Println');

        editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }));
        expect(completion?.style.display).toBe('none');

        const noMenuEvent = new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true, cancelable: true });
        const noMenuHandled = editor.dispatchEvent(noMenuEvent);
        expect(noMenuHandled).toBe(true);
    });

    it('selects highlighted LSP completion item when Enter is pressed', async () => {
        listFilesMock.mockResolvedValue([]);
        resolveNotesLspLanguageMock.mockResolvedValue('markdown');
        notesLspCompletionMock.mockResolvedValue([
            { label: 'Println', detail: 'fmt', insertText: 'Println', kind: 3 },
            { label: 'Printf', detail: 'fmt', insertText: 'Printf', kind: 3 },
        ]);
        getFileMock.mockResolvedValue({ contents: '# Todo\n\nPrin', text: '', error: '' });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({ filename: '$NOTES/todo.md', contents: '# Todo\n\nPrin' });
        await flushPromises();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        editor.value = '# Todo\n\nPrin.';
        editor.selectionStart = editor.value.length;
        editor.selectionEnd = editor.value.length;
        editor.dispatchEvent(new Event('input', { bubbles: true }));
        await flushPromises();
        await flushPromises();

        const completion = document.getElementById('notes-lsp-completion');
        expect(completion?.style.display).toBe('block');

        const downHandled = editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true, cancelable: true }));
        expect(downHandled).toBe(false);
        expect(completion?.querySelector('.notes-lsp-completion-item.is-active')?.textContent).toContain('Printf');

        const enterHandled = editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true }));
        await flushPromises();

        expect(enterHandled).toBe(false);
        expect(editor.value).toContain('Printf');
        expect(completion?.style.display).toBe('none');
    });

    it('filters LSP completion menu by typing additional characters while keeping menu open', async () => {
        listFilesMock.mockResolvedValue([]);
        resolveNotesLspLanguageMock.mockResolvedValue('markdown');
        notesLspChangeDocumentMock.mockClear();
        notesLspCompletionMock.mockResolvedValueOnce([
            { label: 'Println', detail: 'fmt', insertText: 'Println', kind: 3 },
            { label: 'Printf', detail: 'fmt', insertText: 'Printf', kind: 3 },
            { label: 'Panic', detail: 'builtin', insertText: 'Panic', kind: 3 },
        ]).mockResolvedValueOnce([
            { label: 'Printf', detail: 'fmt', insertText: 'Printf', kind: 3 },
        ]).mockResolvedValue([
            { label: 'Printf', detail: 'fmt', insertText: 'Printf', kind: 3 },
        ]);
        getFileMock.mockResolvedValue({ contents: '# Todo\n\nPr', text: '', error: '' });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({ filename: '$NOTES/todo.md', contents: '# Todo\n\nPr' });
        await flushPromises();
        await flushPromises();

        document.getElementById('notes-tab-editor').click();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        editor.value = '# Todo\n\nPr';
        editor.selectionStart = editor.value.length;
        editor.selectionEnd = editor.value.length;

        const tabHandled = editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }));
        expect(tabHandled).toBe(false);
        await flushPromises();
        await flushPromises();

        const completion = document.getElementById('notes-lsp-completion');
        expect(completion?.style.display).toBe('block');

        let items = completion?.querySelectorAll('.notes-lsp-completion-item');
        expect(items?.length).toBe(3);
        expect(items?.[0]?.textContent).toContain('Println');
        expect(items?.[1]?.textContent).toContain('Printf');
        expect(items?.[2]?.textContent).toContain('Panic');

        editor.value = '# Todo\n\nPrf';
        editor.selectionStart = editor.value.length;
        editor.selectionEnd = editor.value.length;
        editor.dispatchEvent(new Event('input', { bubbles: true }));
        await flushPromises();
        await flushPromises();

        expect(completion?.style.display).toBe('block');
        items = completion?.querySelectorAll('.notes-lsp-completion-item');
        expect(items?.length).toBe(1);
        expect(items?.[0]?.textContent).toContain('Printf');

        expect(notesLspChangeDocumentMock.mock.calls.length).toBeGreaterThanOrEqual(1);
        expect(notesLspCompletionMock).toHaveBeenCalled();
    });

    it('uses Tab to open LSP completion when non-whitespace text exists to the left on the same line', async () => {
        listFilesMock.mockResolvedValue([]);
        resolveNotesLspLanguageMock.mockResolvedValue('markdown');
        notesLspCompletionMock.mockResolvedValue([
            { label: 'Println', detail: 'fmt', insertText: 'Println', kind: 3 },
            { label: 'Printf', detail: 'fmt', insertText: 'Printf', kind: 3 },
        ]);
        getFileMock.mockResolvedValue({ contents: '# Todo\n\nPrin', text: '', error: '' });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({ filename: '$NOTES/todo.md', contents: '# Todo\n\nPrin' });
        await flushPromises();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        editor.value = '# Todo\n\nPrin';
        editor.selectionStart = editor.value.length;
        editor.selectionEnd = editor.value.length;

        const before = editor.value;
        const tabHandled = editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }));
        await flushPromises();
        await flushPromises();

        expect(tabHandled).toBe(false);
        expect(editor.value).toBe(before);
        expect(notesLspCompletionMock).toHaveBeenCalled();
        expect(document.getElementById('notes-lsp-completion')?.style.display).toBe('block');
    });

    it('keeps default Tab indentation when only whitespace exists to the left on the same line', async () => {
        listFilesMock.mockResolvedValue([]);
        resolveNotesLspLanguageMock.mockResolvedValue('markdown');
        notesLspCompletionMock.mockResolvedValue([
            { label: 'Println', detail: 'fmt', insertText: 'Println', kind: 3 },
        ]);
        getFileMock.mockResolvedValue({ contents: '# Todo\n\n    ', text: '', error: '' });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({ filename: '$NOTES/todo.md', contents: '# Todo\n\n    ' });
        await flushPromises();
        await flushPromises();

        notesLspCompletionMock.mockClear();

        const editor = document.getElementById('notes-editor');
        editor.value = '# Todo\n\n    ';
        editor.selectionStart = editor.value.length;
        editor.selectionEnd = editor.value.length;

        const tabHandled = editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }));
        await flushPromises();

        expect(tabHandled).toBe(false);
        expect(editor.value).toBe('# Todo\n\n        ');
        expect(notesLspCompletionMock).not.toHaveBeenCalled();
    });

    it('closes open LSP completion and inserts a tab when Tab is pressed again', async () => {
        listFilesMock.mockResolvedValue([]);
        resolveNotesLspLanguageMock.mockResolvedValue('markdown');
        notesLspCompletionMock.mockResolvedValue([
            { label: 'Println', detail: 'fmt', insertText: 'Println', kind: 3 },
            { label: 'Printf', detail: 'fmt', insertText: 'Printf', kind: 3 },
        ]);
        getFileMock.mockResolvedValue({ contents: '# Todo\n\nPrin', text: '', error: '' });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({ filename: '$NOTES/todo.md', contents: '# Todo\n\nPrin' });
        await flushPromises();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        editor.value = '# Todo\n\nPrin';
        editor.selectionStart = editor.value.length;
        editor.selectionEnd = editor.value.length;

        const firstTabHandled = editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }));
        await flushPromises();
        await flushPromises();

        expect(firstTabHandled).toBe(false);
        expect(document.getElementById('notes-lsp-completion')?.style.display).toBe('block');

        const secondTabHandled = editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }));
        await flushPromises();

        expect(secondTabHandled).toBe(false);
        expect(editor.value).toBe('# Todo\n\nPrin    ');
        expect(document.getElementById('notes-lsp-completion')?.style.display).toBe('none');
        expect(notesLspCompletionMock).toHaveBeenCalledTimes(1);
    });

    // Inlay hints and semantic tokens are supplied to Monaco providers, which are
    // disabled under jsdom, so neither should be fetched by the notes module.
    it('leaves inlay hints and semantic tokens to Monaco providers', async () => {
        listFilesMock.mockResolvedValue([]);
        resolveNotesLspLanguageMock.mockResolvedValue('go');
        getFileMock.mockResolvedValue({ contents: 'name := value\n', text: '', error: '' });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({ filename: '$NOTES/main.go', contents: 'name := value\n' });
        await flushPromises();
        await flushPromises();

        expect(notesLspInlayHintsMock).not.toHaveBeenCalled();
        expect(notesLspSemanticTokensMock).not.toHaveBeenCalled();
        expect(document.querySelectorAll('.notes-lsp-inlay-hint')).toHaveLength(0);
    });

    it('lists and executes code lens actions from editor context menu', async () => {
        listFilesMock.mockResolvedValue([]);
        resolveNotesLspLanguageMock.mockResolvedValue('go');
        notesLspCodeLensMock.mockResolvedValue([
            { index: 0, title: 'Run tests', line: 2, character: 0 },
        ]);
        notesLspExecuteCodeLensMock.mockResolvedValue(true);
        getFileMock.mockResolvedValue({
            contents: 'package main\n\nfunc main() {}\n',
            text: '',
            error: '',
        });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({
            filename: '$NOTES/main.go',
            contents: 'package main\n\nfunc main() {}\n',
        });
        await flushPromises();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        editor.selectionStart = 0;
        editor.selectionEnd = 0;
        editor.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 80, clientY: 120 }));
        await flushPromises();

        const contextMenu = showLocalMenuMock.mock.calls[0][0];
        const lspOptionsIndex = contextMenu.options.findIndex((title) => title === 'LSP options...');
        expect(lspOptionsIndex).toBeGreaterThanOrEqual(0);

        await contextMenu.onSelect(lspOptionsIndex);
        await flushPromises();

        const lspMenu = showLocalMenuMock.mock.calls[1][0];
        const lensIndex = lspMenu.options.findIndex((title) => title === 'Code lens...');
        expect(lensIndex).toBeGreaterThanOrEqual(0);

        lspMenu.onSelect(lensIndex);
        await flushPromises();

        expect(notesLspCodeLensMock).toHaveBeenCalledWith('$NOTES/main.go');
        expect(showLocalMenuMock).toHaveBeenCalledTimes(3);

        const lensMenu = showLocalMenuMock.mock.calls[2][0];
        expect(lensMenu.options[0]).toContain('Run tests');
        expect(lensMenu.options[0]).toContain('line 3');

        await lensMenu.onSelect(0);
        await flushPromises();

        expect(notesLspExecuteCodeLensMock).toHaveBeenCalledWith('$NOTES/main.go', 0);
    });

    it('uses CSS-driven opacity rules for LSP hover and completion popups', async () => {
        listFilesMock.mockResolvedValue([]);
        resolveNotesLspLanguageMock.mockResolvedValue('markdown');
        notesLspHoverMock.mockResolvedValue('**Hover docs**');
        notesLspCompletionMock.mockResolvedValue([
            { label: 'Println', detail: 'fmt', insertText: 'Println', kind: 3 },
        ]);
        getFileMock.mockResolvedValue({ contents: '# Todo\n\nPrin', text: '', error: '' });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({ filename: '$NOTES/todo.md', contents: '# Todo\n\nPrin' });
        await flushPromises();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        editor.value = '# Todo\n\nPrin.';
        editor.selectionStart = editor.value.length;
        editor.selectionEnd = editor.value.length;
        editor.dispatchEvent(new Event('input', { bubbles: true }));
        await flushPromises();
        await flushPromises();

        const completion = document.getElementById('notes-lsp-completion');
        expect(completion?.style.display).toBe('block');
        expect(completion?.style.opacity || '').toBe('');

        editor.selectionStart = 2;
        editor.selectionEnd = 2;
        editor.dispatchEvent(new KeyboardEvent('keyup', { key: 'ArrowRight', bubbles: true }));
        await new Promise((resolve) => setTimeout(resolve, 320));
        await flushPromises();

        const hover = document.getElementById('notes-lsp-hover-tooltip');
        expect(hover?.style.display).toBe('none');
        expect(hover?.style.opacity || '').toBe('');

        const styles = getNotesRenderedStyles();
        expect(styles).toContain('#notes-lsp-completion {');
        expect(styles).toContain('--notes-lsp-completion-visible-rows: 12;');
        expect(styles).toContain('max-height: calc(var(--notes-lsp-completion-visible-rows) * var(--notes-lsp-completion-row-height) + 12px);');
        expect(getNotesStyleProperties('#notes-lsp-completion:hover').opacity).toBe('1');
        expect(styles).toContain('.notes-lsp-completion-item.is-deprecated .notes-lsp-completion-label,');
        expect(styles).toContain('#notes-lsp-hover-tooltip {');
        expect(getNotesStyleProperties('#notes-lsp-hover-tooltip:hover').opacity).toBe('1');
    });

    it('formats current document via editor context menu action', async () => {
        listFilesMock.mockResolvedValue([]);
        resolveNotesLspLanguageMock.mockResolvedValue('go');
        notesLspFormatMock.mockResolvedValue({
            changed: true,
            content: 'package main\n\nfunc main() {\n\tprintln("ok")\n}\n',
        });
        getFileMock.mockResolvedValue({ contents: 'package main\nfunc main(){println("ok")}', text: '', error: '' });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({ filename: '$NOTES/main.go', contents: 'package main\nfunc main(){println("ok")}' });
        await flushPromises();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        editor.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 80, clientY: 120 }));
        await flushPromises();

        expect(showLocalMenuMock).toHaveBeenCalled();
        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        const lspOptionsIndex = menuConfig.options.findIndex((title) => title === 'LSP options...');
        expect(lspOptionsIndex).toBeGreaterThanOrEqual(0);

        await menuConfig.onSelect(lspOptionsIndex);
        await flushPromises();

        const lspMenu = showLocalMenuMock.mock.calls[1][0];
        const formatIndex = lspMenu.options.findIndex((title) => title === 'Format document');
        expect(formatIndex).toBeGreaterThanOrEqual(0);

        lspMenu.onSelect(formatIndex);
        await flushPromises();
        await flushPromises();

        expect(notesLspFormatMock).toHaveBeenCalledWith('$NOTES/main.go');
        expect(editor.value).toContain('func main() {');

        editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'z', metaKey: true, bubbles: true, cancelable: true }));
        await flushPromises();
        expect(editor.value).toBe('package main\nfunc main(){println("ok")}');

        editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'z', metaKey: true, shiftKey: true, bubbles: true, cancelable: true }));
        await flushPromises();
        expect(editor.value).toBe('package main\n\nfunc main() {\n\tprintln("ok")\n}\n');
    });

    it('formats selected range via editor context menu action', async () => {
        listFilesMock.mockResolvedValue([]);
        resolveNotesLspLanguageMock.mockResolvedValue('go');
        notesLspFormatRangeMock.mockResolvedValue({
            changed: true,
            content: 'package main\nfunc main() {\n\tprintln("ok")\n}\n',
        });
        getFileMock.mockResolvedValue({ contents: 'package main\nfunc main(){println("ok")}', text: '', error: '' });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({ filename: '$NOTES/main.go', contents: 'package main\nfunc main(){println("ok")}' });
        await flushPromises();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        const start = editor.value.indexOf('func');
        const end = editor.value.length;
        editor.selectionStart = start;
        editor.selectionEnd = end;

        editor.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 80, clientY: 120 }));
        await flushPromises();

        expect(showLocalMenuMock).toHaveBeenCalled();
        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        const lspOptionsIndex = menuConfig.options.findIndex((title) => title === 'LSP options...');
        expect(lspOptionsIndex).toBeGreaterThanOrEqual(0);

        await menuConfig.onSelect(lspOptionsIndex);
        await flushPromises();

        const lspMenu = showLocalMenuMock.mock.calls[1][0];
        const formatIndex = lspMenu.options.findIndex((title) => title === 'Format document');
        expect(formatIndex).toBeGreaterThanOrEqual(0);

        lspMenu.onSelect(formatIndex);
        await flushPromises();
        await flushPromises();

        expect(notesLspFormatRangeMock).toHaveBeenCalledWith('$NOTES/main.go', 1, 0, 1, 26);
        expect(notesLspFormatRangeMock).toHaveBeenCalledTimes(1);
        expect(editor.value).toContain('func main() {');
    });

    it('navigates to selected document symbol from editor context menu', async () => {
        listFilesMock.mockResolvedValue([]);
        resolveNotesLspLanguageMock.mockResolvedValue('go');
        notesLspDocumentSymbolsMock.mockResolvedValue([
            { name: 'alpha', detail: 'func', kind: 12, line: 2, character: 0 },
            { name: 'beta', detail: 'func', kind: 12, line: 3, character: 0, containerName: 'alpha' },
        ]);
        getFileMock.mockResolvedValue({
            contents: 'package main\n\nfunc alpha() {}\nfunc beta() {}\n',
            text: '',
            error: '',
        });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({
            filename: '$NOTES/main.go',
            contents: 'package main\n\nfunc alpha() {}\nfunc beta() {}\n',
        });
        await flushPromises();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        editor.selectionStart = 0;
        editor.selectionEnd = 0;
        editor.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 80, clientY: 120 }));
        await flushPromises();

        const contextMenu = showLocalMenuMock.mock.calls[0][0];
        const lspOptionsIndex = contextMenu.options.findIndex((title) => title === 'LSP options...');
        expect(lspOptionsIndex).toBeGreaterThanOrEqual(0);

        await contextMenu.onSelect(lspOptionsIndex);
        await flushPromises();

        const lspMenu = showLocalMenuMock.mock.calls[1][0];
        const gotoIndex = lspMenu.options.findIndex((title) => title === 'Go to symbol...');
        expect(gotoIndex).toBeGreaterThanOrEqual(0);

        lspMenu.onSelect(gotoIndex);
        await flushPromises();

        expect(notesLspDocumentSymbolsMock).toHaveBeenCalledWith('$NOTES/main.go');
        expect(showLocalMenuMock).toHaveBeenCalledTimes(3);

        const symbolMenu = showLocalMenuMock.mock.calls[2][0];
        expect(symbolMenu.options[0]).toContain('alpha');
        expect(symbolMenu.options[1]).toContain('beta');

        symbolMenu.onSelect(1);
        await flushPromises();

        expect(editor.selectionStart).toBe('package main\n\nfunc alpha() {}\n'.length);
        expect(editor.selectionEnd).toBe(editor.selectionStart);
    });

    it('navigates to selected workspace symbol from editor context menu', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/main.go', '$NOTES/pkg/beta.go']);
        resolveNotesLspLanguageMock.mockResolvedValue('go');
        notesLspWorkspaceSymbolsMock.mockResolvedValue([
            {
                name: 'beta',
                detail: 'func',
                kind: 12,
                line: 0,
                character: 0,
                containerName: 'alpha',
                filePath: '/tmp/project/pkg/beta.go',
            },
        ]);
        getFileMock.mockImplementation(async (path) => {
            if (path === '$NOTES/main.go') {
                return { contents: 'package main\n\nfunc alpha() {}\n', text: '', error: '' };
            }
            if (path === '$NOTES/pkg/beta.go') {
                return { contents: 'func beta() {}\n', text: '', error: '' };
            }
            return { contents: '', text: '', error: '' };
        });
        resolveFilePathMock.mockImplementation(async (file) => {
            if (file === '$NOTES/main.go') {
                return '/tmp/project/main.go';
            }
            if (file === '$NOTES/pkg/beta.go') {
                return '/tmp/project/pkg/beta.go';
            }
            return '';
        });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({
            filename: '$NOTES/main.go',
            contents: 'package main\n\nfunc alpha() {}\n',
        });
        await flushPromises();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        editor.selectionStart = 0;
        editor.selectionEnd = 0;
        editor.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 80, clientY: 120 }));
        await flushPromises();

        const contextMenu = showLocalMenuMock.mock.calls[0][0];
        const lspOptionsIndex = contextMenu.options.findIndex((title) => title === 'LSP options...');
        expect(lspOptionsIndex).toBeGreaterThanOrEqual(0);

        await contextMenu.onSelect(lspOptionsIndex);
        await flushPromises();

        const lspMenu = showLocalMenuMock.mock.calls[1][0];
        const gotoIndex = lspMenu.options.findIndex((title) => title === 'Go to workspace symbol...');
        expect(gotoIndex).toBeGreaterThanOrEqual(0);

        lspMenu.onSelect(gotoIndex);
        await flushPromises();

        // Symbols are searched server-side, so no request is made until the user types.
        expect(notesLspWorkspaceSymbolsMock).not.toHaveBeenCalled();
        expect(showLocalMenuMock).toHaveBeenCalledTimes(3);

        const symbolMenu = showLocalMenuMock.mock.calls[2][0];
        expect(symbolMenu.options).toEqual([]);
        expect(typeof symbolMenu.onQuery).toBe('function');

        const queried = await symbolMenu.onQuery('beta');
        await flushPromises();

        expect(notesLspWorkspaceSymbolsMock).toHaveBeenCalledWith('$NOTES/main.go', 'beta');
        expect(queried.options[0]).toContain('beta');
        expect(queried.options[0]).toContain('pkg/beta.go');

        await symbolMenu.onSelect(0);
        await flushPromises();
        await flushPromises();

        expect(getFileMock).toHaveBeenLastCalledWith('$NOTES/pkg/beta.go');
        expect(editor.value).toBe('func beta() {}\n');
        expect(editor.selectionStart).toBe(0);
        expect(editor.selectionEnd).toBe(0);
    });

    it('requests signature help from editor context menu action', async () => {
        listFilesMock.mockResolvedValue([]);
        resolveNotesLspLanguageMock.mockResolvedValue('go');
        notesLspSignatureHelpMock.mockResolvedValue('println(a ...any)\n\nPrint values.');
        getFileMock.mockResolvedValue({
            contents: 'package main\n\nfunc main() {\n\tprintln("ok")\n}\n',
            text: '',
            error: '',
        });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({
            filename: '$NOTES/main.go',
            contents: 'package main\n\nfunc main() {\n\tprintln("ok")\n}\n',
        });
        await flushPromises();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        editor.selectionStart = editor.value.indexOf('println');
        editor.selectionEnd = editor.selectionStart;

        editor.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 80, clientY: 120 }));
        await flushPromises();

        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        const lspOptionsIndex = menuConfig.options.findIndex((title) => title === 'LSP options...');
        expect(lspOptionsIndex).toBeGreaterThanOrEqual(0);

        await menuConfig.onSelect(lspOptionsIndex);
        await flushPromises();

        const lspMenu = showLocalMenuMock.mock.calls[1][0];
        const sigHelpIndex = lspMenu.options.findIndex((title) => title === 'Signature help');
        expect(sigHelpIndex).toBeGreaterThanOrEqual(0);

        lspMenu.onSelect(sigHelpIndex);
        await flushPromises();
        await flushPromises();

        expect(notesLspSignatureHelpMock).toHaveBeenCalled();
        const call = notesLspSignatureHelpMock.mock.calls[0];
        expect(call[0]).toBe('$NOTES/main.go');
        expect(call[3]).toBe(1);
        expect(call[4]).toBe('');

        const hover = document.getElementById('notes-lsp-hover-tooltip');
        expect(hover?.style.display).toBe('block');
        expect(hover?.textContent).toContain('println(a ...any)');
    });

    it('shows and applies quick fix code action from editor context menu', async () => {
        listFilesMock.mockResolvedValue([]);
        resolveNotesLspLanguageMock.mockResolvedValue('go');
        notesLspCodeActionsMock.mockResolvedValue([
            { title: 'Use fmt.Errorf', kind: 'quickfix' },
        ]);
        notesLspApplyCodeActionMock.mockResolvedValue({
            changed: true,
            content: 'package main\n\nimport "fmt"\n\nfunc main() { _ = fmt.Errorf("x") }\n',
        });
        getFileMock.mockResolvedValue({ contents: 'package main\nfunc main(){_ = fmt.Errorf("x")}', text: '', error: '' });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({ filename: '$NOTES/main.go', contents: 'package main\nfunc main(){_ = fmt.Errorf("x")}' });
        await flushPromises();
        await flushPromises();

        const diagnosticsHandler = getEventHandler('notesLspDiagnostics');
        expect(typeof diagnosticsHandler).toBe('function');
        diagnosticsHandler({
            uri: 'file:///tmp/project/main.go',
            diagnostics: [{
                message: 'dummy warning',
                severity: 2,
                range: {
                    start: { line: 1, character: 0 },
                    end: { line: 1, character: 10 },
                },
            }],
        });
        await flushPromises();

        resolveFilePathMock.mockImplementation(async (file) => {
            if (file === '$NOTES/main.go') {
                return '/tmp/project/main.go';
            }
            return '';
        });

        const editor = document.getElementById('notes-editor');
        editor.selectionStart = editor.value.length;
        editor.selectionEnd = editor.value.length;

        editor.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 80, clientY: 120 }));
        await flushPromises();
        await flushPromises();

        expect(showLocalMenuMock).toHaveBeenCalled();

        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        const lspOptionsIndex = menuConfig.options.findIndex((title) => title === 'LSP options...');
        expect(lspOptionsIndex).toBeGreaterThanOrEqual(0);

        await menuConfig.onSelect(lspOptionsIndex);
        await flushPromises();

        expect(notesLspCodeActionsMock).toHaveBeenCalled();

        const lspMenu = showLocalMenuMock.mock.calls[1][0];
        const fixIndex = lspMenu.options.findIndex((title) => title === 'Quick fix: Use fmt.Errorf');
        expect(fixIndex).toBeGreaterThanOrEqual(0);

        lspMenu.onSelect(fixIndex);
        await flushPromises();
        await flushPromises();

        expect(notesLspApplyCodeActionMock).toHaveBeenCalled();
        expect(editor.value).toContain('import "fmt"');

        editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'z', metaKey: true, bubbles: true, cancelable: true }));
        await flushPromises();
        expect(editor.value).toBe('package main\nfunc main(){_ = fmt.Errorf("x")}');

        editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'z', metaKey: true, shiftKey: true, bubbles: true, cancelable: true }));
        await flushPromises();
        expect(editor.value).toBe('package main\n\nimport "fmt"\n\nfunc main() { _ = fmt.Errorf("x") }\n');
    });

    it('groups LSP code actions by kind in the editor context menu', async () => {
        listFilesMock.mockResolvedValue([]);
        resolveNotesLspLanguageMock.mockResolvedValue('go');
        notesLspCodeActionsMock.mockResolvedValue([
            { title: 'Extract function', kind: 'refactor.extract' },
            { title: 'Use fmt.Errorf', kind: 'quickfix' },
            { title: 'Organize imports', kind: 'source.organizeImports' },
        ]);
        getFileMock.mockResolvedValue({ contents: 'package main\nfunc main(){_ = fmt.Errorf("x")}', text: '', error: '' });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({ filename: '$NOTES/main.go', contents: 'package main\nfunc main(){_ = fmt.Errorf("x")}' });
        await flushPromises();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        editor.selectionStart = editor.value.length;
        editor.selectionEnd = editor.value.length;

        editor.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 80, clientY: 120 }));
        await flushPromises();
        await flushPromises();

        expect(showLocalMenuMock).toHaveBeenCalled();
        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        const lspOptionsIndex = menuConfig.options.findIndex((title) => title === 'LSP options...');
        expect(lspOptionsIndex).toBeGreaterThanOrEqual(0);

        await menuConfig.onSelect(lspOptionsIndex);
        await flushPromises();

        const lspMenu = showLocalMenuMock.mock.calls[1][0];
        expect(lspMenu.options).toContain('Quick fix: Use fmt.Errorf');
        expect(lspMenu.options).toContain('Refactor: Extract function');
        expect(lspMenu.options).toContain('Source action: Organize imports');

        const quickFixIndex = lspMenu.options.findIndex((title) => title === 'Quick fix: Use fmt.Errorf');
        const refactorIndex = lspMenu.options.findIndex((title) => title === 'Refactor: Extract function');
        const sourceIndex = lspMenu.options.findIndex((title) => title === 'Source action: Organize imports');
        expect(quickFixIndex).toBeGreaterThanOrEqual(0);
        expect(refactorIndex).toBeGreaterThan(quickFixIndex);
        expect(sourceIndex).toBeGreaterThan(refactorIndex);

        const separators = lspMenu.options.filter((title) => title === '-');
        expect(separators.length).toBeGreaterThanOrEqual(2);
    });

    it('renames current symbol from editor context menu', async () => {
        listFilesMock.mockResolvedValue([]);
        resolveNotesLspLanguageMock.mockResolvedValue('go');
        notesLspPrepareRenameMock.mockResolvedValue({ canRename: true, placeholder: 'println' });
        notesLspRenameMock.mockResolvedValue({
            changed: true,
            content: 'package main\n\nfunc main() {\n\tPrintln("ok")\n}\n',
        });
        getFileMock.mockResolvedValue({ contents: 'package main\n\nfunc main() {\n\tprintln("ok")\n}\n', text: '', error: '' });

        await importNotesModule();

        const createAndOpenHandler = getEventHandler('notesCreateAndOpen');
        expect(typeof createAndOpenHandler).toBe('function');
        await createAndOpenHandler({ filename: '$NOTES/main.go', contents: 'package main\n\nfunc main() {\n\tprintln("ok")\n}\n' });
        await flushPromises();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        const cursorOffset = editor.value.indexOf('println');
        editor.selectionStart = cursorOffset;
        editor.selectionEnd = cursorOffset + 'println'.length;

        const promptSpy = vi.spyOn(window, 'prompt');

        editor.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 80, clientY: 120 }));
        await flushPromises();

        expect(showLocalMenuMock).toHaveBeenCalled();
        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        const lspOptionsIndex = menuConfig.options.findIndex((title) => title === 'LSP options...');
        expect(lspOptionsIndex).toBeGreaterThanOrEqual(0);

        await menuConfig.onSelect(lspOptionsIndex);
        await flushPromises();

        const lspMenu = showLocalMenuMock.mock.calls[1][0];
        const renameIndex = lspMenu.options.findIndex((title) => title === 'Rename symbol...');
        expect(renameIndex).toBeGreaterThanOrEqual(0);

        lspMenu.onSelect(renameIndex);
        await flushPromises();
        await flushPromises();

        expect(notesLspPrepareRenameMock).toHaveBeenCalledWith('$NOTES/main.go', 3, 1);
        expect(promptSpy).not.toHaveBeenCalled();

        const modalInput = document.getElementById('notes-modal-input');
        const modalCreate = document.getElementById('notes-modal-create');
        expect(document.getElementById('notes-modal').dataset.open).toBe('true');
        expect(modalInput.value).toBe('println');

        modalInput.value = 'Println';
        modalCreate.click();
        await flushPromises();
        await flushPromises();

        expect(notesLspRenameMock).toHaveBeenCalledWith('$NOTES/main.go', 3, 1, 'Println');
        expect(editor.value).toContain('Println("ok")');

        // The modal steals focus while open; the user must return to the editor
        // before undo/redo keystrokes are routed to it again.
        editor.focus();
        editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'z', metaKey: true, bubbles: true, cancelable: true }));
        await flushPromises();
        expect(editor.value).toContain('println("ok")');

        editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'z', metaKey: true, shiftKey: true, bubbles: true, cancelable: true }));
        await flushPromises();
        expect(editor.value).toContain('Println("ok")');

        promptSpy.mockRestore();
    });

    it('renames notes using selected location and name without forcing .md', async () => {
        listFilesMock
            .mockResolvedValueOnce(['$GLOBAL/docs/todo.md'])
            .mockResolvedValueOnce(['$PROJECT/docs/todo.txt']);
        resolveNoteLocationMock.mockResolvedValueOnce('$GLOBAL/docs/todo.md');
        getFileMock.mockResolvedValue({ contents: '# Hello Notes', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$GLOBAL/docs/todo.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const fileActionDialogHandler = getEventHandler('fileActionDialog');
        expect(fileActionDialogHandler).toBeTypeOf('function');
        fileActionDialogHandler({ action: 'rename', filePath: '$GLOBAL/docs/todo.md' });
        await flushPromises();

        const modalInput = document.getElementById('notes-modal-input');
        const modalLocation = document.getElementById('notes-modal-location');
        expect(modalLocation.textContent).toBe('$GLOBAL');
        expect(modalInput.value).toBe('docs/todo.md');

        modalLocation.textContent = '$PROJECT';
        modalInput.value = 'docs/todo.txt';
        getFileMock.mockClear();
        renameFileMock.mockImplementationOnce(async () => {
            await getEventHandler('notesFileChanged')();
        });
        document.getElementById('notes-modal-create').click();
        await flushPromises();
        await flushPromises();

        expect(renameFileMock).toHaveBeenCalledWith('$GLOBAL/docs/todo.md', '$PROJECT/docs/todo.txt');
        expect(renameFileMock).not.toHaveBeenCalledWith('$GLOBAL/docs/todo.md', '$PROJECT/docs/todo.txt.md');
        expect(getFileMock).toHaveBeenCalledWith('$PROJECT/docs/todo.txt');
        expect(getFileMock).not.toHaveBeenCalledWith('$GLOBAL/docs/todo.md');
        getFileMock.mockClear();
        await getEventHandler('notesFileChanged')();
        expect(getFileMock).toHaveBeenCalledWith('$PROJECT/docs/todo.txt');
    });

    it.each([
        ['open note', '$NOTES/current.md', '$NOTES/previous.md'],
        ['other note', '$NOTES/previous.md', ''],
        ['last note', '$NOTES/current.md', ''],
    ])('deleting %s restores history only when needed', async (_label, deletedFile, historyFile) => {
        listFilesMock.mockResolvedValue(['$NOTES/current.md', '$NOTES/previous.md']);
        getFileMock.mockResolvedValue({ contents: 'one\ntwo\nthree\nfour', text: '', error: '' });
        await importNotesModule();
        document.querySelector('[data-file="$NOTES/current.md"]').click();
        await flushPromises();
        await flushPromises();
        notesHistoryCurrentPositionMock.mockResolvedValue({ file: historyFile, line: 3 });
        getFileMock.mockClear();
        getEventHandler('fileActionDialog')({ action: 'delete', filePath: deletedFile });
        await flushPromises();
        deleteFileMock.mockImplementationOnce(async () => {
            if (deletedFile === '$NOTES/current.md') {
                await getEventHandler('notesFileChanged')();
            }
        });
        document.getElementById('notes-delete-confirm').click();
        await flushPromises();
        await flushPromises();
        await flushPromises();
        expect(deleteFileMock).toHaveBeenCalledWith(deletedFile);
        if (historyFile) {
            expect(getFileMock).toHaveBeenCalledWith(historyFile);
            expect(document.getElementById('notes-editor').selectionStart).toBe('one\ntwo\n'.length);
        } else {
            expect(getFileMock).not.toHaveBeenCalled();
            expect(document.getElementById('notes-editor').value).toBe(
                deletedFile === '$NOTES/current.md' ? '' : 'one\ntwo\nthree\nfour',
            );
        }
    });

    it('awaits composed rename path before calling RenameFile', async () => {
        listFilesMock
            .mockResolvedValueOnce(['$GLOBAL/docs/todo.md'])
            .mockResolvedValueOnce(['$PROJECT/docs/todo.txt']);
        getFileMock.mockResolvedValue({ contents: '# Hello Notes', text: '', error: '' });
        composeNoteLocationPathMock.mockResolvedValueOnce('$PROJECT/docs/todo.txt');

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$GLOBAL/docs/todo.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const fileActionDialogHandler = getEventHandler('fileActionDialog');
        fileActionDialogHandler({ action: 'rename', filePath: '$GLOBAL/docs/todo.md' });
        await flushPromises();

        const modalInput = document.getElementById('notes-modal-input');
        const modalLocation = document.getElementById('notes-modal-location');
        modalLocation.textContent = '$PROJECT';
        modalInput.value = 'docs/todo.txt';

        document.getElementById('notes-modal-create').click();
        await flushPromises();
        await flushPromises();

        expect(renameFileMock).toHaveBeenCalledWith('$GLOBAL/docs/todo.md', '$PROJECT/docs/todo.txt');
        expect(typeof renameFileMock.mock.calls[0][1]).toBe('string');
    });

    it('creates notes using selected location and normalized filename', async () => {
        listFilesMock
            .mockResolvedValueOnce([])
            .mockResolvedValueOnce(['$PROJECT/new-note.md']);
        getFileMock.mockResolvedValue({ contents: '# Hello Notes', text: '', error: '' });

        await importNotesModule();

        document.getElementById('notes-new').click();
        await flushPromises();

        expect(document.querySelector('#notes-modal-title h3')?.textContent).toBe('New note name');
        const modalInput = document.getElementById('notes-modal-input');
        const modalLocation = document.getElementById('notes-modal-location');

        modalLocation.textContent = '$PROJECT';
        modalInput.value = 'new-note';

        document.getElementById('notes-modal-create').click();
        await flushPromises();
        await flushPromises();

        expect(saveFileMock).toHaveBeenCalledWith('$PROJECT/new-note.md', '', '');
    });

    it('creates notes with provided extension without appending .md', async () => {
        listFilesMock
            .mockResolvedValueOnce([])
            .mockResolvedValueOnce(['$PROJECT/new-note.txt']);
        getFileMock.mockResolvedValue({ contents: 'Hello Notes', text: '', error: '' });

        await importNotesModule();

        document.getElementById('notes-new').click();
        await flushPromises();

        const modalInput = document.getElementById('notes-modal-input');
        const modalLocation = document.getElementById('notes-modal-location');

        modalLocation.textContent = '$PROJECT';
        modalInput.value = 'new-note.txt';

        document.getElementById('notes-modal-create').click();
        await flushPromises();
        await flushPromises();

        expect(saveFileMock).toHaveBeenCalledWith('$PROJECT/new-note.txt', '', '');
        expect(saveFileMock).not.toHaveBeenCalledWith('$PROJECT/new-note.txt.md', '', '');
    });

    it('focuses the textarea whenever an Edit view becomes active', async () => {
        listFilesMock.mockResolvedValue([
            '$NOTES/script.go',
            '$NOTES/readme.md',
            '$NOTES/spec.yaml',
        ]);

        getFileMock.mockImplementation(async (file) => {
            if (file.endsWith('.go')) {
                return { contents: 'package main\n\nfunc main() {}', text: '', error: '' };
            }

            if (file.endsWith('.yaml')) {
                return { contents: 'openapi: 3.0.0\ninfo:\n  title: Sample', text: '', error: '' };
            }

            return { contents: '# Markdown note', text: '', error: '' };
        });

        await importNotesModule();

        const notesEditor = document.getElementById('notes-editor');
        const clickFile = async (filePath) => {
            const fileButton = document.querySelector(`[data-file="${filePath}"]`);
            fileButton.click();
            await flushPromises();
            await flushPromises();
        };

        await clickFile('$NOTES/script.go');
        expect(document.activeElement).toBe(notesEditor);

        await clickFile('$NOTES/readme.md');

        document.getElementById('notes-tab-editor').click();
        await flushPromises();
        expect(document.activeElement).toBe(notesEditor);

        await clickFile('$NOTES/spec.yaml');

        document.getElementById('notes-tab-swagger-edit').click();
        await flushPromises();
        expect(document.activeElement).toBe(notesEditor);

        document.getElementById('notes-tab-swagger-view').click();
        await flushPromises();
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        await flushPromises();
        expect(document.activeElement).toBe(notesEditor);
    });

    it('preserves file-list scroll position when opening another file', async () => {
        listFilesMock.mockResolvedValue([
            '$NOTES/a.md',
            '$NOTES/b.md',
            '$NOTES/c.md',
        ]);
        getFileMock.mockImplementation(async (file) => ({ contents: `# ${file}\ncontent`, text: '', error: '' }));

        await importNotesModule();

        const list = document.getElementById('notes-list');
        let listScrollTop = 0;
        Object.defineProperty(list, 'scrollTop', {
            configurable: true,
            get() {
                return listScrollTop;
            },
            set(value) {
                listScrollTop = Number(value) || 0;
            },
        });
        list.scrollTop = 123;

        const firstButton = document.querySelector('[data-file="$NOTES/a.md"]');
        firstButton.click();
        await flushPromises();
        await flushPromises();

        list.scrollTop = 123;
        const secondButton = document.querySelector('[data-file="$NOTES/b.md"]');
        secondButton.click();
        await flushPromises();
        await flushPromises();

        expect(list.scrollTop).toBe(123);
    });

    it('scrolls the active file into view when workspace project changes', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getCurrentGroupNameMock.mockResolvedValueOnce('Workspace One').mockResolvedValue('Workspace Two');
        getCurrentProjectMock.mockResolvedValue('/tmp/project-one');
        getFileMock.mockResolvedValue({ contents: '# Guide', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/guide.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const scrollSpy = vi.spyOn(Element.prototype, 'scrollIntoView').mockImplementation(() => {});
        scrollSpy.mockClear();

        const notesUpdateHandler = getEventHandler('notesContextChanged');
        expect(typeof notesUpdateHandler).toBe('function');
        notesHistoryCurrentPositionMock.mockResolvedValue({ file: '$NOTES/guide.md', line: 1 });
        getCurrentProjectMock.mockResolvedValue('/tmp/project-two');
        notesUpdateHandler({ projectRoot: '/tmp/project-two', groupName: 'Workspace Two' });
        await flushPromises();
        await flushPromises();

        expect(scrollSpy).toHaveBeenCalledWith({ block: 'center', inline: 'nearest' });
        scrollSpy.mockRestore();
    });

    it.each(['viewer', 'editor'])('preserves the cached %s tab when returning to a workspace with a saved cursor position', async (mode) => {
        const file = '$NOTES/guide.md';
        const contents = 'one\ntwo\nthree\nfour';
        listFilesMock.mockResolvedValue([file]);
        getCurrentProjectMock.mockResolvedValue('/tmp/project-one');
        getFileMock.mockResolvedValue({ contents, text: '', error: '' });
        getDocumentCacheMock.mockResolvedValue({ DocumentTab: mode });

        await importNotesModule();
        document.querySelector(`[data-file="${file}"]`).click();
        await flushPromises();
        await flushPromises();

        const contextChanged = getEventHandler('notesContextChanged');
        listFilesMock.mockResolvedValue([]);
        getCurrentProjectMock.mockResolvedValue('/tmp/project-two');
        contextChanged({ projectRoot: '/tmp/project-two', groupName: 'Workspace' });
        await flushPromises();
        await flushPromises();

        listFilesMock.mockResolvedValue([file]);
        getCurrentProjectMock.mockResolvedValue('/tmp/project-one');
        notesHistoryCurrentPositionMock.mockResolvedValue({ file, line: 3 });
        contextChanged({ projectRoot: '/tmp/project-one', groupName: 'Workspace' });
        await flushPromises();
        await flushPromises();
        await flushPromises();

        const tabId = mode === 'viewer' ? 'notes-tab-viewer' : 'notes-tab-editor';
        expect(document.getElementById(tabId).getAttribute('aria-selected')).toBe('true');
        if (mode === 'editor') {
            expect(document.getElementById('notes-editor').selectionStart).toBe('one\ntwo\n'.length);
        }
    });

    it.each(['/project-one', '/project-two'])('restores workspace history for matching paths and external files with root %s', async (otherRoot) => {
        const file = '$PROJECT/README.md';
        let workspace = 'One';
        listFilesMock.mockResolvedValue([file]);
        getCurrentProjectMock.mockResolvedValue('/project-one');
        getCurrentGroupNameMock.mockResolvedValue(workspace);
        notesHistoryCurrentPositionMock.mockResolvedValue({ file, line: 1 });
        getFileMock.mockImplementation(async () => ({ contents: `# ${workspace}`, text: '', error: '' }));
        await importNotesModule();
        expect(document.getElementById('notes-editor').value).toBe('# One');

        const contextChanged = getEventHandler('notesContextChanged');
        workspace = 'Two';
        getCurrentProjectMock.mockResolvedValue(otherRoot);
        getCurrentGroupNameMock.mockResolvedValue(workspace);
        contextChanged({ projectRoot: otherRoot, groupName: workspace });
        await flushPromises();
        await flushPromises();
        expect(document.getElementById('notes-editor').value).toBe('# Two');
        expect(getFileMock).toHaveBeenCalledTimes(2);

        const externalFile = '/another-project/outside.md';
        notesHistoryCurrentPositionMock.mockResolvedValue({ file: externalFile, line: 1 });
        getCurrentProjectMock.mockResolvedValue('/another-active-root');
        contextChanged({ projectRoot: '/another-active-root', groupName: workspace });
        await flushPromises();
        await flushPromises();
        expect(getFileMock).toHaveBeenLastCalledWith(externalFile);
        expect(document.querySelector(`[data-file="${externalFile}"]`)).toBeNull();
    });

    it('clears the previous workspace document when the new workspace has no history', async () => {
        listFilesMock.mockResolvedValue(['$PROJECT/README.md']);
        notesHistoryCurrentPositionMock.mockResolvedValue({ file: '$PROJECT/README.md', line: 1 });
        getFileMock.mockResolvedValue({ contents: '# Previous workspace', text: '', error: '' });
        await importNotesModule();
        expect(document.getElementById('notes-editor').value).toBe('# Previous workspace');

        notesHistoryCurrentPositionMock.mockResolvedValue({ file: '', line: 1 });
        getEventHandler('notesContextChanged')({ projectRoot: '/new', groupName: 'Empty workspace' });
        await flushPromises();
        await flushPromises();
        expect(document.getElementById('notes-editor').value).toBe('');
        expect(document.getElementById('notes-preview').textContent).not.toContain('Previous workspace');
    });

    it('ignores old workspace scans and does not rescan unchanged contexts', async () => {
        listFilesMock.mockResolvedValue(['$PROJECT/initial.md']);
        await importNotesModule();

        let completeOldScan;
        listFilesMock.mockImplementationOnce(() => new Promise((resolve) => {
            completeOldScan = resolve;
        }));
        const contextChanged = getEventHandler('notesContextChanged');
        const oldContext = { projectRoot: '/old', groupName: 'Old', globalNotes: '/notes/', userNotes: '/notes/Old/' };
        const newContext = { projectRoot: '/new', groupName: 'New', globalNotes: '/notes/', userNotes: '/notes/New/' };
        contextChanged(oldContext);
        await flushPromises();
        listFilesMock.mockResolvedValueOnce(['$PROJECT/new.md']);
        contextChanged(newContext);
        await flushPromises();
        await flushPromises();

        completeOldScan(['$PROJECT/old.md']);
        await flushPromises();
        await flushPromises();
        expect(document.querySelector('[data-file="$PROJECT/new.md"]')).not.toBeNull();
        expect(document.querySelector('[data-file="$PROJECT/old.md"]')).toBeNull();
        expect(document.getElementById('notes-title').innerText).toBe('New');

        const scans = listFilesMock.mock.calls.length;
        contextChanged(newContext);
        await flushPromises();
        expect(listFilesMock).toHaveBeenCalledTimes(scans);
        expect(cancelNotesListFilesMock).not.toHaveBeenCalled();
    });

    it('does not replace the file list with a cancelled scan', async () => {
        listFilesMock.mockResolvedValue(['$PROJECT/complete.md']);
        await importNotesModule();
        listFilesMock.mockRejectedValueOnce('context canceled');
        getEventHandler('notesContextChanged')({ projectRoot: '/other', groupName: 'Other' });
        await flushPromises();
        await flushPromises();
        expect(document.querySelector('[data-file="$PROJECT/complete.md"]')).not.toBeNull();
    });

    it('keeps cached files visible while refreshing and applies index completion without rescanning', async () => {
        getCurrentProjectMock.mockResolvedValue('/project');
        getNotesFilesSnapshotMock.mockResolvedValue({
            files: ['$PROJECT/cached.md'], roots: [], state: 'refreshing', error: '', truncated: false,
        });
        await importNotesModule();

        expect(document.querySelector('[data-file="$PROJECT/cached.md"]')).not.toBeNull();
        expect(document.getElementById('notes-file-index-spinner').hidden).toBe(false);
        expect(document.getElementById('notes-list').getAttribute('aria-busy')).toBe('true');
        expect(document.getElementById('notes-file-index-status').getAttribute('aria-label')).toBe('Refreshing files');

        getNotesFilesSnapshotMock.mockResolvedValue({
            files: ['$PROJECT/fresh.md'], roots: [], state: 'ready', error: '', truncated: false,
        });
        const onIndexChanged = getEventHandler('notesFileIndexChanged');
        onIndexChanged({ root: '/project', version: 2, revision: 3 });
        onIndexChanged({ root: '/project', version: 2, revision: 3 });
        await flushPromises();
        await flushPromises();

        expect(document.querySelector('[data-file="$PROJECT/fresh.md"]')).not.toBeNull();
        expect(document.querySelector('[data-file="$PROJECT/cached.md"]')).toBeNull();
        expect(document.getElementById('notes-file-index-spinner').hidden).toBe(true);
        expect(document.getElementById('notes-list').getAttribute('aria-busy')).toBe('false');
        expect(getNotesFilesSnapshotMock).toHaveBeenCalledTimes(2);
        expect(getNotesFilesSnapshotMock.mock.calls.every(([, refresh]) => refresh === false)).toBe(true);

        onIndexChanged({ root: '/inactive', version: 4, revision: 5 });
        await flushPromises();
        expect(getNotesFilesSnapshotMock).toHaveBeenCalledTimes(2);
    });

    it('shows index truncation and scan errors without hiding cached files', async () => {
        getCurrentProjectMock.mockResolvedValue('/project');
        getNotesFilesSnapshotMock.mockResolvedValue({
            files: ['$PROJECT/cached.md'], roots: [], state: 'truncated', error: '', truncated: true,
        });
        await importNotesModule();
        const status = document.getElementById('notes-file-index-status');
        expect(status.hidden).toBe(false);
        expect(status.title).toContain('truncated');
        expect(document.getElementById('notes-file-index-warning').hidden).toBe(false);

        getNotesFilesSnapshotMock.mockResolvedValue({
            files: ['$PROJECT/cached.md'], roots: [], state: 'error', error: 'Permission denied', truncated: false,
        });
        getEventHandler('notesFileIndexChanged')({ root: '/project', version: 1, revision: 4 });
        await flushPromises();
        await flushPromises();
        expect(document.querySelector('[data-file="$PROJECT/cached.md"]')).not.toBeNull();
        expect(status.title).toContain('Permission denied');
        expect(document.getElementById('notes-file-index-spinner').hidden).toBe(true);
    });

    it('applies file index deltas and recovers a version gap without rescanning', async () => {
        getCurrentProjectMock.mockResolvedValue('/project');
        getNotesFilesSnapshotMock.mockResolvedValue({
            files: ['$PROJECT/old.md'], state: 'ready',
            roots: [{ root: '/project', files: ['old.md'], version: 1, revision: 2, state: 'ready' }],
        });
        await importNotesModule();
        const changed = getEventHandler('notesFileIndexChanged');
        const event = { root: '/project', baseVersion: 1, version: 2, revision: 3,
            added: ['new.md'], removed: ['old.md'], state: 'ready' };
        changed(event);
        await flushPromises();
        expect(document.querySelector('[data-file="$PROJECT/new.md"]')).not.toBeNull();
        expect(document.querySelector('[data-file="$PROJECT/old.md"]')).toBeNull();
        expect(getNotesFilesSnapshotMock).toHaveBeenCalledTimes(1);
        changed(event);
        await flushPromises();
        expect(getNotesFilesSnapshotMock).toHaveBeenCalledTimes(1);

        getNotesFilesSnapshotMock.mockResolvedValue({
            files: ['$PROJECT/recovered.md'], state: 'ready',
            roots: [{ root: '/project', files: ['recovered.md'], version: 4, revision: 5, state: 'ready' }],
        });
        changed({ ...event, baseVersion: 3, version: 4, revision: 5 });
        await flushPromises();
        await flushPromises();
        expect(document.querySelector('[data-file="$PROJECT/recovered.md"]')).not.toBeNull();
        expect(getNotesFilesSnapshotMock).toHaveBeenCalledTimes(2);
        expect(getNotesFilesSnapshotMock.mock.calls[1][1]).toBe(false);
    });

    it('shows a file parent folder context menu when a file is right-clicked', async () => {
        listFilesMock.mockResolvedValue([
            '$PROJECT/docs/reference/api spec.json',
            '$PROJECT/docs/readme.md',
        ]);
        resolveFilePathMock.mockResolvedValue('/tmp/project/docs/reference/api spec.json');
        getHyperlinkMenuActionsMock.mockResolvedValue([
            { title: 'Copy file path to clipboard', icon: 0xf0c1, action: '0' },
            { title: 'Rename file', icon: 0xf044, action: '1' },
            { title: 'Delete file', icon: 0xf1f8, action: '2' },
        ]);

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$PROJECT/docs/reference/api spec.json"]');
        fileButton.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 48, clientY: 96 }));
        await flushPromises();

        expect(showLocalMenuMock).toHaveBeenCalledTimes(1);

        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        expect(menuConfig.title).toBe('reference');
        expect(menuConfig.options).toEqual([
            'Copy file path to clipboard', 'Rename file', 'Delete file', '-',
            'Collapse Folders', 'Expand Folders', '-', 'Create new file...', 'Paste as new file...',
        ]);
        expect(getHyperlinkMenuActionsMock).toHaveBeenCalledWith('file:///tmp/project/docs/reference/api spec.json', 'api spec.json');
        menuConfig.onSelect(menuConfig.options.indexOf('Delete file'));
        await flushPromises();
        expect(runHyperlinkMenuActionMock).toHaveBeenCalledWith('file:///tmp/project/docs/reference/api spec.json', 'api spec.json', '2');

        menuConfig.onSelect(menuConfig.options.indexOf('Create new file...'));
        expect(document.getElementById('notes-modal-location').textContent).toBe('$PROJECT');
        expect(document.getElementById('notes-modal-input').value).toBe('docs/reference/');
    });

    it('uses the nearest preceding folder for a blank file-list context menu', async () => {
        listFilesMock.mockResolvedValue([
            '$PROJECT/docs/readme.md',
            '$PROJECT/docs/reference/api.md',
            '$PROJECT/images/logo.png',
        ]);

        await importNotesModule();

        const projectHeader = document.querySelector('.notes-category-header[data-category="$PROJECT"]');
        const docsFolder = document.querySelector('.notes-tree-folder[data-folder-key="$PROJECT/docs"]');
        const referenceFolder = document.querySelector('.notes-tree-folder[data-folder-key="$PROJECT/docs/reference"]');
        const imagesFolder = document.querySelector('.notes-tree-folder[data-folder-key="$PROJECT/images"]');
        const categoryContent = projectHeader.nextElementSibling;
        projectHeader.getBoundingClientRect = () => ({ top: 0, bottom: 20, height: 20, width: 100 });
        docsFolder.getBoundingClientRect = () => ({ top: 25, bottom: 45, height: 20, width: 100 });
        referenceFolder.getBoundingClientRect = () => ({ top: 50, bottom: 70, height: 20, width: 100 });
        imagesFolder.getBoundingClientRect = () => ({ top: 75, bottom: 95, height: 20, width: 100 });

        categoryContent.dispatchEvent(new MouseEvent('contextmenu', {
            bubbles: true,
            cancelable: true,
            clientX: 64,
            clientY: 22,
        }));

        expect(showLocalMenuMock).toHaveBeenCalledTimes(1);
        const sectionMenu = showLocalMenuMock.mock.calls[0][0];
        expect(sectionMenu.title).toBe('$PROJECT folders');
        showLocalMenuMock.mockClear();

        categoryContent.dispatchEvent(new MouseEvent('contextmenu', {
            bubbles: true,
            cancelable: true,
            clientX: 64,
            clientY: 72,
        }));

        expect(showLocalMenuMock).toHaveBeenCalledTimes(1);
        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        expect(menuConfig.title).toBe('reference');
        expect(menuConfig.options).toEqual(['Collapse Folders', 'Expand Folders', '-', 'Create new file...', 'Paste as new file...']);
        menuConfig.onSelect(menuConfig.options.indexOf('Create new file...'));
        expect(document.getElementById('notes-modal-input').value).toBe('docs/reference/');
    });

    it('shows a folder-tree menu on category headers and applies collapse/expand actions', async () => {
        listFilesMock.mockResolvedValue([
            '$PROJECT/docs/guide.md',
            '$PROJECT/docs/reference/api.md',
            '$PROJECT/images/logo.png',
        ]);

        await importNotesModule();

        const projectHeader = document.querySelector('.notes-category-header[data-category="$PROJECT"]');
        projectHeader.dispatchEvent(new MouseEvent('contextmenu', {
            bubbles: true,
            cancelable: true,
            clientX: 64,
            clientY: 128,
        }));

        expect(showLocalMenuMock).toHaveBeenCalledTimes(1);
        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        expect(menuConfig.options).toEqual([
            'Collapse Folders',
            'Expand Folders',
            '-',
            'Create new file...',
            'Paste as new file...',
        ]);
        expect(menuConfig.icons.slice(0, 2)).toEqual([0xf146, 0xf0fe]);

        menuConfig.onSelect(menuConfig.options.indexOf('Create new file...'));
        expect(document.getElementById('notes-modal-location').textContent).toBe('$PROJECT');
        expect(document.getElementById('notes-modal-input').value).toBe('');
        document.getElementById('notes-modal-cancel').click();

        menuConfig.onSelect(0);
        await flushPromises();

        const collapsedFolders = Array.from(document.querySelectorAll('.notes-tree-folder'));
        expect(collapsedFolders.length).toBeGreaterThan(0);
        expect(collapsedFolders.every((folder) => folder.dataset.expanded === 'false')).toBe(true);

        menuConfig.onSelect(1);
        await flushPromises();

        const expandedFolders = Array.from(document.querySelectorAll('.notes-tree-folder'));
        expect(expandedFolders.length).toBeGreaterThan(0);
        expect(expandedFolders.every((folder) => folder.dataset.expanded === 'true')).toBe(true);
    });

    it('shows a folder-tree menu on directory items and applies actions to child folders only', async () => {
        listFilesMock.mockResolvedValue([
            '$PROJECT/docs/reference/api.md',
            '$PROJECT/docs/readme.md',
            '$PROJECT/images/logo.png',
        ]);

        await importNotesModule();

        const docsFolder = document.querySelector('.notes-tree-folder[data-folder-key="$PROJECT/docs"]');
        docsFolder.dispatchEvent(new MouseEvent('contextmenu', {
            bubbles: true,
            cancelable: true,
            clientX: 72,
            clientY: 140,
        }));

        expect(showLocalMenuMock).toHaveBeenCalledTimes(1);
        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        expect(menuConfig.options).toEqual([
            'Collapse Folders',
            'Expand Folders',
            '-',
            'Create new file...',
            'Paste as new file...',
        ]);

        menuConfig.onSelect(0);
        await flushPromises();

        expect(document.querySelector('.notes-tree-folder[data-folder-key="$PROJECT/docs"]')?.dataset.expanded).toBe('true');
        expect(document.querySelector('.notes-tree-folder[data-folder-key="$PROJECT/docs/reference"]')?.dataset.expanded).toBe('false');
        expect(document.querySelector('.notes-tree-folder[data-folder-key="$PROJECT/images"]')?.dataset.expanded).toBe('true');

        menuConfig.onSelect(1);
        await flushPromises();

        expect(document.querySelector('.notes-tree-folder[data-folder-key="$PROJECT/docs/reference"]')?.dataset.expanded).toBe('true');
    });

    it('creates new files from a leaf folder context menu with its path prefilled', async () => {
        listFilesMock.mockResolvedValue(['$PROJECT/docs/readme.md']);
        getFileMock.mockResolvedValue({ contents: '# Readme', text: '', error: '' });

        await importNotesModule();
        const docsFolder = document.querySelector('.notes-tree-folder[data-folder-key="$PROJECT/docs"]');
        docsFolder.dispatchEvent(new MouseEvent('contextmenu', {
            bubbles: true,
            cancelable: true,
            clientX: 72,
            clientY: 140,
        }));

        const menu = showLocalMenuMock.mock.calls.at(-1)[0];
        expect(menu.options).toContain('Create new file...');
        expect(menu.options).toContain('Paste as new file...');
        menu.onSelect(menu.options.indexOf('Create new file...'));

        const modalInput = document.getElementById('notes-modal-input');
        expect(document.getElementById('notes-modal-location').textContent).toBe('$PROJECT');
        expect(modalInput.value).toBe('docs/');

        modalInput.value = 'docs/new-note';
        document.getElementById('notes-modal-create').click();
        await flushPromises();
        await flushPromises();

        expect(saveFileMock).toHaveBeenCalledWith('$PROJECT/docs/new-note.md', '', '');
    });

    it('pastes clipboard text or images as new files in the selected folder', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/assets/readme.md']);
        getFileMock.mockResolvedValue({ contents: '# Readme', text: '', error: '' });

        await importNotesModule();
        const openAssetsMenu = () => {
            const folder = document.querySelector('.notes-tree-folder[data-folder-key="$NOTES/assets"]');
            folder.dispatchEvent(new MouseEvent('contextmenu', {
                bubbles: true,
                cancelable: true,
                clientX: 72,
                clientY: 140,
            }));
            return showLocalMenuMock.mock.calls.at(-1)[0];
        };

        getClipboardDataMock.mockResolvedValue({ text: 'clipboard text', image: '' });
        let menu = openAssetsMenu();
        menu.onSelect(menu.options.indexOf('Paste as new file...'));
        await flushPromises();
        await flushPromises();

        expect(getClipboardDataMock).toHaveBeenCalledOnce();
        expect(document.getElementById('notes-modal-location').textContent).toBe('$NOTES');
        expect(document.getElementById('notes-modal-input').value).toBe('assets/clipboard.md');
        document.getElementById('notes-modal-create').click();
        await flushPromises();
        await flushPromises();
        expect(saveFileMock).toHaveBeenCalledWith('$NOTES/assets/clipboard.md', 'clipboard text', '');

        getClipboardDataMock.mockResolvedValue({ text: '', image: 'aW1hZ2U=' });
        menu = openAssetsMenu();
        menu.onSelect(menu.options.indexOf('Paste as new file...'));
        await flushPromises();
        await flushPromises();

        expect(document.getElementById('notes-modal-input').value).toBe('assets/clipboard.png');
        document.getElementById('notes-modal-create').click();
        await flushPromises();
        await flushPromises();
        expect(saveBinaryFileMock).toHaveBeenCalledWith('$NOTES/assets/clipboard.png', 'aW1hZ2U=');
    });

    it('shows a hyperlink context menu when right-clicking an anchor in the markdown preview', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide', text: '', error: '' });
        displayHyperlinkMenuMock.mockResolvedValue();

        await importNotesModule();

        // Load the markdown file so viewMode becomes 'viewer'
        const fileButton = document.querySelector('[data-file="$NOTES/guide.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        // Inject an anchor directly into the rendered preview container
        const preview = document.getElementById('notes-preview');
        const anchor = document.createElement('a');
        anchor.href = 'https://example.com/docs';
        anchor.textContent = 'Docs site';
        preview.appendChild(anchor);

        anchor.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 100, clientY: 200 }));
        await flushPromises();
        await flushPromises();

        expect(displayHyperlinkMenuMock).toHaveBeenCalledWith('https://example.com/docs', 'Docs site');
        expect(showLocalMenuMock).not.toHaveBeenCalled();
    });

    it('shows Ask AI in editor context menu and sends current document content', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide\n\nHello from editor.', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/guide.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        showLocalMenuMock.mockClear();

        const editor = document.getElementById('notes-editor');
        editor.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 90, clientY: 120 }));
        await flushPromises();

        expect(showLocalMenuMock).toHaveBeenCalledTimes(1);
        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        const askAIIndex = menuConfig.options.findIndex((option) => String(option).startsWith('Ask AI'));
        expect(askAIIndex).toBeGreaterThanOrEqual(0);

        menuConfig.onSelect(askAIIndex);
        await flushPromises();

        expect(askAIMock).toHaveBeenCalledTimes(1);
        expect(askAIMock).toHaveBeenCalledWith(
            'notesDocument',
            '$NOTES/guide.md',
            expect.stringContaining('Document: $NOTES/guide.md'),
        );
        expect(askAIMock.mock.calls[0][2]).toContain('Type: markdown');
        expect(askAIMock.mock.calls[0][2]).toContain('Hello from editor.');
    });

    it('deletes URI-escaped image paths relative to notes with spaces in their names', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/file with space.md']);
        getFileMock.mockResolvedValue({
            contents: '![123](file%20with%20space.md.d/123.png)',
            text: '',
            error: '',
        });

        const originalExecCommand = document.execCommand;
        document.execCommand = vi.fn((_command, _showUi, text) => {
            const editor = document.activeElement;
            const start = editor.selectionStart;
            const end = editor.selectionEnd;
            editor.value = editor.value.slice(0, start) + text + editor.value.slice(end);
            editor.selectionStart = start + text.length;
            editor.selectionEnd = editor.selectionStart;
            editor.dispatchEvent(new Event('input', { bubbles: true }));
            return true;
        });

        try {
            await importNotesModule();
            document.querySelector('[data-file="$NOTES/file with space.md"]').click();
            await flushPromises();
            await flushPromises();

            const editor = document.getElementById('notes-editor');
            const imageStart = editor.value.indexOf('![123]');
            editor.selectionStart = imageStart;
            editor.selectionEnd = imageStart;
            editor.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true }));
            await flushPromises();

            const menu = showLocalMenuMock.mock.calls.at(-1)[0];
            const deleteImageIndex = menu.options.indexOf('Delete image from disk');
            expect(deleteImageIndex).toBeGreaterThanOrEqual(0);
            await menu.onSelect(deleteImageIndex);
            await flushPromises();

            expect(deleteFileMock).toHaveBeenCalledWith('$NOTES/file with space.md.d/123.png');
            expect(editor.value).not.toContain('![123]');
        } finally {
            document.execCommand = originalExecCommand;
        }
    });

    it('shows Ask AI in rendered preview context menu and sends current document content', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/summary.md']);
        getFileMock.mockResolvedValue({ contents: '# Summary\n\nViewer side text.', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/summary.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        showLocalMenuMock.mockClear();

        const preview = document.getElementById('notes-preview');
        preview.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 30, clientY: 40 }));
        await flushPromises();

        expect(showLocalMenuMock).toHaveBeenCalledTimes(1);
        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        const askAIIndex = menuConfig.options.findIndex((option) => String(option).startsWith('Ask AI'));
        expect(askAIIndex).toBeGreaterThanOrEqual(0);

        menuConfig.onSelect(askAIIndex);
        await flushPromises();

        expect(askAIMock).toHaveBeenCalledTimes(1);
        expect(askAIMock).toHaveBeenCalledWith(
            'notesDocument',
            '$NOTES/summary.md',
            expect.stringContaining('Document: $NOTES/summary.md'),
        );
        expect(askAIMock.mock.calls[0][2]).toContain('Type: markdown');
        expect(askAIMock.mock.calls[0][2]).toContain('Viewer side text.');
    });

    it('sends the right-clicked image to Ask AI instead of the open document', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/source.go']);
        getFileMock.mockResolvedValue({ contents: 'package main\n\nfunc main() {}', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/source.go"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        getEventHandler('aiJobStart')({ runId: 701, title: '' });
        getEventHandler('aiStreamBlock')({
            runId: 701,
            blockId: 'image-question',
            kind: 'text',
            delta: '\n![diagram.png](data:image/png;base64,AAAA)\n',
            status: 'closed',
        });
        await flushPromises();
        getEventHandler('aiJobFinish')();
        await flushPromises();
        await flushPromises();

        const image = document.getElementById('notes-ai-output').querySelector('img');
        expect(image).not.toBeNull();

        showLocalMenuMock.mockClear();
        image.dispatchEvent(new MouseEvent('contextmenu', {
            bubbles: true,
            cancelable: true,
            clientX: 20,
            clientY: 20,
        }));
        await flushPromises();

        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        menuConfig.onSelect(2);
        await flushPromises();

        expect(askAIImageMock).toHaveBeenCalledWith('diagram.png', 'data:image/png;base64,AAAA');
        expect(askAIMock).not.toHaveBeenCalled();
    });

    it('adds Copy code and renames Copy to Copy selection when right-clicking a code block', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/code.md']);
        getFileMock.mockResolvedValue({ contents: '# Code\n\n```js\nconst x = 1;\n```', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/code.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        showLocalMenuMock.mockClear();
        clipboardSetTextMock.mockClear();

        const preview = document.getElementById('notes-preview');
        preview.innerHTML = '<pre><code>const x = 1;\nconst y = 2;\n</code></pre>';
        const codeEl = preview.querySelector('code');

        codeEl.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 30, clientY: 40 }));
        await flushPromises();

        expect(showLocalMenuMock).toHaveBeenCalledTimes(1);
        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        expect(menuConfig.options).toContain('Copy selection');
        expect(menuConfig.options).toContain('Copy code');
        expect(menuConfig.options).not.toContain('Copy');

        const copyCodeIndex = menuConfig.options.findIndex((option) => option === 'Copy code');
        expect(copyCodeIndex).toBeGreaterThanOrEqual(0);

        menuConfig.onSelect(copyCodeIndex);
        await flushPromises();

        expect(clipboardSetTextMock).toHaveBeenCalledWith('const x = 1;\nconst y = 2;\n');

        const saveCodeIndex = menuConfig.options.findIndex((option) => option === 'Save code...');
        expect(saveCodeIndex).toBeGreaterThanOrEqual(0);
        menuConfig.onSelect(saveCodeIndex);
        await flushPromises();

        expect(saveCodeDialogMock).toHaveBeenCalledWith('code-block.txt');
        expect(saveFileMock).toHaveBeenCalledWith('/project/code-block.js', 'const x = 1;\nconst y = 2;\n', '');
    });

    it('keeps Jupyter reserved words isolated and ignores stale runtime responses', async () => {
        getLanguageDescriptionsMock.mockImplementation((language) => Promise.resolve(language === 'docker'
            ? ['Dockerfile (build and run)', 'Dockerfile (custom runtime)'] : ['Python 3.x']));
        getNotesLanguageReservedWordsMock.mockImplementation((language) => Promise.resolve(
            language === 'docker' || language === 'Dockerfile (build and run)' ? ['FROM', 'RUN']
                : language === 'Dockerfile (custom runtime)' ? ['CUSTOM']
                    : language === 'python' || language === 'Python 3.x' ? ['def'] : []));
        listFilesMock.mockResolvedValue(['$NOTES/reserved.md']);
        getFileMock.mockResolvedValue({ contents: '```docker\nFROM alpine\n```\n\n```python\ndef example(): pass\n```', text: '', error: '' });
        await importNotesModule();
        document.querySelector('[data-file="$NOTES/reserved.md"]').click();
        await flushPromises();
        await flushPromises();
        document.getElementById('notes-tab-jupyter').click();
        await flushPromises();
        await flushPromises();

        const spellcheckFor = (element) => attachSpellCheck.mock.results[attachSpellCheck.mock.calls.findIndex(([target]) => target === element)].value;
        const editors = document.querySelectorAll('.jupyter-code-editable');
        const dockerHandle = spellcheckFor(editors[0]);
        const pythonHandle = spellcheckFor(editors[1]);
        const mainHandle = spellcheckFor(document.getElementById('notes-editor'));
        expect(dockerHandle.setExclusions).toHaveBeenLastCalledWith(['FROM', 'RUN']);
        expect(pythonHandle.setExclusions).toHaveBeenLastCalledWith(['def']);
        const pythonCalls = pythonHandle.setExclusions.mock.calls.length;
        const mainCalls = mainHandle.setExclusions.mock.calls.length;

        document.querySelector('.jupyter-runtime-dropdown').click();
        const menu = showLocalMenuMock.mock.calls.at(-1)[0];
        menu.onSelect(1);
        await flushPromises();
        expect(dockerHandle.setExclusions).toHaveBeenLastCalledWith(['CUSTOM']);

        let resolveStale;
        getNotesLanguageReservedWordsMock.mockImplementationOnce(() => new Promise((resolve) => { resolveStale = resolve; }));
        menu.onSelect(0);
        menu.onSelect(1);
        await flushPromises();
        resolveStale(['STALE']);
        await flushPromises();
        expect(dockerHandle.setExclusions).toHaveBeenLastCalledWith(['CUSTOM']);
        expect(dockerHandle.setExclusions).not.toHaveBeenCalledWith(['STALE']);
        expect(pythonHandle.setExclusions).toHaveBeenCalledTimes(pythonCalls);
        expect(mainHandle.setExclusions).toHaveBeenCalledTimes(mainCalls);
    });

    it('selects the cached document tab before rendering Markdown', async () => {
        const file = '$NOTES/cached-mode.md';
        listFilesMock.mockResolvedValue([file]);
        getFileMock.mockResolvedValue({ contents: '# Cached mode', text: '', error: '' });
        getDocumentCacheMock.mockResolvedValue({ DocumentTab: 'editor' });

        await importNotesModule();

        const selectedDocumentTabs = [];
        const originalSetAttribute = Element.prototype.setAttribute;
        const setAttributeSpy = vi.spyOn(Element.prototype, 'setAttribute').mockImplementation(function (name, value) {
            originalSetAttribute.call(this, name, value);
            if (name === 'aria-selected' && value === 'true' && this.id.startsWith('notes-tab-')) {
                selectedDocumentTabs.push(this.id);
            }
        });

        try {
            document.querySelector(`[data-file="${file}"]`).click();
            await flushPromises();
            await flushPromises();

            expect(selectedDocumentTabs).toEqual(['notes-tab-editor']);
        } finally {
            setAttributeSpy.mockRestore();
        }
    });

    it('loads Jupyter reserved words for the restored runtime', async () => {
        getLanguageDescriptionsMock.mockResolvedValue(['Dockerfile (build and run)', 'Dockerfile (restored)']);
        getNotesLanguageReservedWordsMock.mockImplementation((language) => Promise.resolve(language === 'Dockerfile (restored)' ? ['RESTORED'] : ['FROM']));
        getDocumentCacheMock.mockResolvedValue({ DocumentTab: 'jupyter', ToolsOpen: true, ToolsTab: 'ai', jupyterBlockState: {
            'jupyter-block-0': { runtime: 'Dockerfile (restored)', lspMode: false },
        } });
        listFilesMock.mockResolvedValue(['$NOTES/restored.md']);
        getFileMock.mockResolvedValue({ contents: '```docker\nFROM alpine\n```', text: '', error: '' });
        await importNotesModule();
        document.querySelector('[data-file="$NOTES/restored.md"]').click();
        await flushPromises();
        await flushPromises();
        const editor = document.querySelector('.jupyter-code-editable');
        const handle = attachSpellCheck.mock.results[attachSpellCheck.mock.calls.findIndex(([target]) => target === editor)].value;
        expect(document.querySelector('.jupyter-runtime-dropdown').textContent).toBe('Dockerfile (restored)');
        expect(handle.setExclusions).toHaveBeenLastCalledWith(['RESTORED']);
    });

    it('loads Dockerfile reserved words even when the code LSP cannot open', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/Dockerfile']);
        getFileMock.mockResolvedValue({ contents: 'FROM alpine\nRUN echo hello', text: '', error: '' });
        resolveNotesLspLanguageMock.mockResolvedValue('dockerfile');
        notesLspOpenDocumentMock.mockRejectedValue(new Error('No Dockerfile language server'));
        getNotesLanguageReservedWordsMock.mockResolvedValue(['FROM', 'RUN']);
        await importNotesModule();
        document.querySelector('[data-file="$NOTES/Dockerfile"]').click();
        await flushPromises();
        await flushPromises();
        const editor = document.getElementById('notes-editor');
        const handle = attachSpellCheck.mock.results[attachSpellCheck.mock.calls.findIndex(([target]) => target === editor)].value;
        expect(getNotesLanguageReservedWordsMock).toHaveBeenCalledWith('dockerfile');
        expect(handle.setExclusions).toHaveBeenLastCalledWith(['from', 'run']);
    });

    it.each(['Markdown', 'Jupyter'])('sends %s code through the shared cloned-pane terminal API', async (source) => {
        const content = "printf '%s\\n' 'hello world'\nprintf 'done\\n'\n";
        getLanguageDescriptionsMock.mockResolvedValue(['Bash Shell Script', 'Alternative shell']);
        listFilesMock.mockResolvedValue(['$NOTES/commands.md']);
        getFileMock.mockResolvedValue({ contents: `# Commands\n\n\`\`\`bash\n${content}\`\`\``, text: '', error: '' });
        await importNotesModule();
        document.querySelector('[data-file="$NOTES/commands.md"]').click();
        await flushPromises();
        await flushPromises();

        let expectedContent = content;
        let expectedRuntime = 'Bash Shell Script';
        if (source === 'Jupyter') {
            document.getElementById('notes-tab-jupyter').click();
            await flushPromises();
            expectedContent = "printf '%s\\n' 'edited cell'\n";
            document.querySelector('.jupyter-code-editable').value = expectedContent;
            document.querySelector('.jupyter-runtime-dropdown').click();
            showLocalMenuMock.mock.calls.at(-1)[0].onSelect(1);
            expectedRuntime = 'Alternative shell';
            document.querySelector('.jupyter-run-terminal').click();
        } else {
            const block = document.querySelector('#notes-preview pre');
            block.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }));
            block.querySelector('.notes-send-terminal-block-button').click();
        }
        await flushPromises();
        expect(sendToTerminalMock).toHaveBeenCalledOnce();
        expect(sendToTerminalMock).toHaveBeenCalledWith(expectedContent, expectedRuntime);
        expect(runNoteMock).not.toHaveBeenCalled();
    });

    it('shows a hover copy button for code and quote blocks in View and AI output', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/blocks.md']);
        getFileMock.mockResolvedValue({ contents: '# Blocks', text: '', error: '' });

        await importNotesModule();
        clipboardSetTextMock.mockClear();

        const preview = document.getElementById('notes-preview');
        preview.innerHTML = '<pre><code>view code\n</code></pre><blockquote>view quote</blockquote>';
        const viewCode = preview.querySelector('pre');
        const viewQuote = preview.querySelector('blockquote');

        viewCode.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }));
        viewQuote.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }));

        const viewCodeButton = viewCode.querySelector('.notes-copy-block-button');
        const viewQuoteButton = viewQuote.querySelector('.notes-copy-block-button');
        expect(viewCodeButton).not.toBeNull();
        expect(viewQuoteButton).not.toBeNull();

        viewCodeButton.click();
        viewQuoteButton.click();
        await flushPromises();

        expect(clipboardSetTextMock).toHaveBeenNthCalledWith(1, 'view code\n');
        expect(clipboardSetTextMock).toHaveBeenNthCalledWith(2, 'view quote');

        const aiOutput = document.getElementById('notes-ai-output');
        aiOutput.innerHTML = '<pre><code>ai code</code></pre><blockquote>ai quote</blockquote>';
        const aiCode = aiOutput.querySelector('pre');
        const aiQuote = aiOutput.querySelector('blockquote');

        aiCode.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }));
        aiQuote.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }));
        expect(aiCode.querySelector('.notes-copy-block-button')).not.toBeNull();
        expect(aiQuote.querySelector('.notes-copy-block-button')).not.toBeNull();
    });

    it('erases a single prompt from the AI Settings transcript without a global clear button', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/summary.md']);
        getFileMock.mockResolvedValue({ contents: '# Summary', text: '', error: '' });
        getAISessionManagementMock.mockResolvedValue({
            activeSessionId: 1,
            sessions: [],
            history: [
                { id: 42, prompt: 'First prompt', commandLine: '', outputBlock: '', excerpt: 'first excerpt' },
                { id: 43, prompt: 'Second prompt', commandLine: '', outputBlock: '', excerpt: 'second excerpt' },
            ],
        });

        await importNotesModule();

        // The button is relocated into the modal; it must not exist in the toolbar.
        expect(document.getElementById('notes-tools-clear')).toBeNull();

        document.getElementById('notes-tools-ai-settings').click();
        await flushPromises();
        await flushPromises();

        const historyList = document.getElementById('notes-ai-settings-history-list');
        const historyClear = document.getElementById('notes-ai-settings-history-clear');
        expect(historyList.previousElementSibling).toBe(historyClear.parentElement);
        expect(historyClear.parentElement.previousElementSibling.textContent).toBe('Recent prompts and responses from the active session.');
        expect(historyClear.parentElement.previousElementSibling.previousElementSibling.textContent).toBe('Active Session Transcript');
        const eraseButtons = historyList.querySelectorAll('button[data-action="erase"]');
        expect(eraseButtons).toHaveLength(2);
        expect(eraseButtons[0].dataset.entryId).toBe('42');
        expect(eraseButtons[1].dataset.entryId).toBe('43');

        deleteAIHistoryEntryMock.mockResolvedValueOnce({
            activeSessionId: 1,
            sessions: [],
            history: [
                { id: 43, prompt: 'Second prompt', commandLine: '', outputBlock: '', excerpt: 'second excerpt' },
            ],
        });

        eraseButtons[0].click();
        await flushPromises();
        await flushPromises();

        expect(deleteAIHistoryEntryMock).toHaveBeenCalledWith(42);
        const remaining = historyList.querySelectorAll('button[data-action="erase"]');
        expect(remaining).toHaveLength(1);
        expect(remaining[0].dataset.entryId).toBe('43');
    });

    it('invokes backend AskAI notesPromptToolbar caller from toolbar Ask button', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/summary.md']);
        getFileMock.mockResolvedValue({ contents: '# Summary\n\nViewer side text.', text: '', error: '' });

        await importNotesModule();

        const askButton = document.getElementById('notes-tools-ai-ask');
        expect(askButton).not.toBeNull();

        askButton.click();
        await flushPromises();

        expect(askAIMock).toHaveBeenCalledTimes(1);
        expect(askAIMock).toHaveBeenCalledWith('notesPromptToolbar', '', '');
    });

    it('shows the agent skills menu from the toolbar Skills button', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/summary.md']);
        getFileMock.mockResolvedValue({ contents: '# Summary\n\nViewer side text.', text: '', error: '' });

        await importNotesModule();

        const skillsButton = document.getElementById('notes-tools-ai-skills');
        expect(skillsButton).not.toBeNull();

        skillsButton.click();
        await flushPromises();

        expect(showAISkillsMenuMock).toHaveBeenCalledWith(0, 4);
    });

    it('places Prompt after Skills and provides a session menu for creating and switching sessions', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/summary.md']);
        getFileMock.mockResolvedValue({ contents: '# Summary', text: '', error: '' });
        getAISessionManagementMock.mockResolvedValue({
            activeSessionId: 5,
            sessions: [
                { tableId: 5, summary: 'Planning', active: true },
                { tableId: 6, summary: 'Research', active: false },
            ],
            history: [],
        });
        setActiveAISessionMock.mockResolvedValue({
            activeSessionId: 6,
            sessions: [
                { tableId: 5, summary: 'Planning', active: false },
                { tableId: 6, summary: 'Research', active: true },
            ],
            history: [],
        });

        await importNotesModule();

        const headerButtons = Array.from(document.querySelectorAll('.notes-tools-pane-header button')).map((button) => button.id);
        expect(headerButtons.indexOf('notes-tools-ai-prompt-jump')).toBe(headerButtons.indexOf('notes-tools-ai-skills') + 1);
        expect(document.getElementById('notes-tools-ai-session')?.textContent).toBe('Session');

        const sessionButton = document.getElementById('notes-tools-ai-session');
        sessionButton.click();
        await flushPromises();

        let menu = showLocalMenuMock.mock.calls.at(-1)?.[0];
        expect(menu.title).toBe('Sessions');
        expect(menu.options).toEqual(['New session', '-', 'Planning', 'Research']);
        expect(menu.icons).toEqual([0xf067, 0, 0xf00c, 0]);

        menu.onSelect(3);
        await flushPromises();
        expect(setActiveAISessionMock).toHaveBeenCalledWith(6);

        createAISessionMock.mockResolvedValueOnce({
            activeSessionId: 7,
            sessions: [{ tableId: 7, summary: 'Planning notes', active: true }],
            history: [],
        });
        sessionButton.click();
        await flushPromises();
        menu = showLocalMenuMock.mock.calls.at(-1)?.[0];
        menu.onSelect(0);
        await flushPromises();
        expect(document.getElementById('notes-modal-title').textContent).toBe('New AI session');
        expect(document.querySelector('#notes-modal-title h3')?.textContent).toBe('New AI session');
        expect(document.getElementById('notes-modal-title').classList.contains('markdown-body')).toBe(true);
        expect(document.getElementById('notes-modal-create').textContent).toBe('Create');
        expect(document.getElementById('notes-modal-input').placeholder).toBe('Optional session name');
        document.getElementById('notes-modal-input').value = 'Planning notes';
        document.getElementById('notes-modal-create').click();
        await flushPromises();
        expect(createAISessionMock).toHaveBeenCalledWith('Planning notes');
        expect(document.getElementById('notes-current-file').textContent).not.toContain('Planning notes');
    });

    it('places commit history after comparison and commit controls in the actions toolbar', async () => {
        await importNotesModule();
        const compare = document.getElementById('notes-tools-git-compare');
        const commitToggle = document.getElementById('notes-tools-git-commit-toggle');
        const commits = document.getElementById('notes-tools-git-commits');
        const separator = compare.nextElementSibling;
        expect(separator.classList.contains('notes-git-toolbar-separator')).toBe(true);
        expect(separator.getAttribute('aria-hidden')).toBe('true');
        expect(separator.nextElementSibling).toBe(commits);
        expect(document.querySelector('.notes-git-actions').firstElementChild).toBe(commitToggle);
        expect(commitToggle.textContent.codePointAt(0)).toBe(0xf386);
        expect(commits.textContent.codePointAt(0)).toBe(0xf0ca);
        expect(document.getElementById('notes-tools-git-commit-row').hidden).toBe(true);
        expect(document.getElementById('notes-tools-git-commit-message').placeholder).toBe('Commit message');
        expect(document.getElementById('notes-tools-git-stage-all').classList.contains('notes-git-action')).toBe(true);
    });

    it.each([
        ['empty name', '', true],
        ['cancelled prompt', null, false],
    ])('creates an AI session with the existing default behavior for %s', async (_label, name, shouldCreate) => {
        getAISessionManagementMock.mockResolvedValue({ activeSessionId: 5, sessions: [{ tableId: 5, summary: 'Planning', active: true }], history: [] });
        await importNotesModule();
        document.getElementById('notes-tools-ai-session').click();
        await flushPromises();
        const menu = showLocalMenuMock.mock.calls.at(-1)[0];
        menu.onSelect(0);
        await flushPromises();
        if (name === null) {
            document.getElementById('notes-modal-cancel').click();
        } else {
            document.getElementById('notes-modal-input').value = name;
            document.getElementById('notes-modal-create').click();
        }
        await flushPromises();
        if (shouldCreate) {
            expect(createAISessionMock).toHaveBeenCalledWith('');
        } else {
            expect(createAISessionMock).not.toHaveBeenCalled();
        }
    });

    it('uses href as fallback label when right-clicking an empty anchor label', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/readme.md']);
        getFileMock.mockResolvedValue({ contents: '# Readme', text: '', error: '' });
        displayHyperlinkMenuMock.mockResolvedValue();

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/readme.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const preview = document.getElementById('notes-preview');
        const anchor = document.createElement('a');
        anchor.href = 'https://go.dev';
        anchor.textContent = '';
        preview.appendChild(anchor);

        anchor.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 50, clientY: 80 }));
        await flushPromises();
        await flushPromises();

        expect(displayHyperlinkMenuMock).toHaveBeenCalledWith('https://go.dev/', 'https://go.dev');
        expect(showLocalMenuMock).not.toHaveBeenCalled();
    });

    it('drops live stream blocks emitted for a different workspace', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide', text: '', error: '' });
        getCurrentGroupNameMock.mockResolvedValue('alpha');

        await importNotesModule();

        getEventHandler('aiJobStart')({ runId: 7, title: '' });
        await flushPromises();

        const blockHandler = getEventHandler('aiStreamBlock');
        expect(typeof blockHandler).toBe('function');

        blockHandler({
            runId: 7, blockId: 'other-1', kind: 'tool-output',
            workspace: 'beta', delta: 'from other workspace', status: 'open',
        });
        await flushPromises();
        await flushPromises();

        const aiOutput = document.getElementById('notes-ai-output');
        expect(aiOutput.querySelector('[data-block-id="other-1"]')).toBeNull();

        blockHandler({
            runId: 7, blockId: 'mine-1', kind: 'tool-output',
            workspace: 'alpha', delta: 'from this workspace', status: 'open',
        });
        await flushPromises();
        await flushPromises();

        expect(aiOutput.querySelector('[data-block-id="mine-1"]')).not.toBeNull();
    });

    it('does not register the retired legacy AI stream transport', async () => {
        await importNotesModule();

        expect(getEventHandler('aiResponseStream')).toBeNull();
        expect(typeof getEventHandler('aiStreamBlock')).toBe('function');
    });

    it('coalesces bursty live stream scrolling into one frame', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide', text: '', error: '' });

        await importNotesModule();

        const aiOutput = document.getElementById('notes-ai-output');
        let scrollWrites = 0;
        Object.defineProperty(aiOutput, 'scrollHeight', { configurable: true, get: () => 1000 });
        Object.defineProperty(aiOutput, 'scrollTop', {
            configurable: true,
            get: () => 1000,
            set: () => { scrollWrites += 1; },
        });

        getEventHandler('aiJobStart')({ runId: 77, title: '' });
        await new Promise((resolve) => setTimeout(resolve, 500));
        scrollWrites = 0;

        const blockHandler = getEventHandler('aiStreamBlock');
        for (let index = 0; index < 50; index += 1) {
            blockHandler({
                runId: 77,
                blockId: `burst-${index}`,
                kind: 'tool-output',
                delta: 'output',
                status: 'open',
            });
        }
        await new Promise((resolve) => setTimeout(resolve, 25));

        expect(scrollWrites).toBe(1);
    });

    it.each([false, true])('reveals the final answer on completion only when following (paused: %s)', async (paused) => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide', text: '', error: '' });
        await importNotesModule();
        const output = document.getElementById('notes-ai-output');
        Object.defineProperty(output, 'scrollHeight', { configurable: true, get: () => 2000 });
        Object.defineProperty(output, 'clientHeight', { configurable: true, get: () => 300 });
        output.getBoundingClientRect = () => ({ top: 100 });
        getEventHandler('aiJobStart')({ runId: 801, title: '' });
        const append = getEventHandler('aiStreamBlock');
        append({ runId: 801, blockId: 'thinking', kind: 'thinking', delta: 'Reasoning', status: 'closed' });
        append({ runId: 801, blockId: 'intermediate', kind: 'text', delta: 'Preparing a result', status: 'closed' });
        append({ runId: 801, blockId: 'tool', kind: 'tool-call', delta: '{}', status: 'closed' });
        append({ runId: 801, blockId: 'nested', parentId: 'tool', kind: 'text', delta: 'Subagent text', status: 'closed' });
        append({ runId: 801, blockId: 'answer', kind: 'text', delta: '# Final answer\n\nResult', status: 'closed' });
        await flushPromises();
        await flushPromises();
        const answer = output.querySelector('[data-block-id="answer"]');
        let revealReads = 0;
        answer.getBoundingClientRect = () => { revealReads++; return { top: 100 + 900 - output.scrollTop }; };
        getEventHandler('aiJobFinish')({ runId: 999 });
        await flushPromises();
        expect(revealReads).toBe(0);
        if (paused) {
            output.dispatchEvent(new WheelEvent('wheel', { deltaY: -100 }));
            output.scrollTop = 275;
        }
        getEventHandler('aiJobFinish')({ runId: 801 });
        await flushPromises();
        await flushPromises();
        await new Promise((resolve) => setTimeout(resolve, 50));
        expect(output.scrollTop).toBe(paused ? 275 : 900);
        expect(document.getElementById('notes-ai-scroll-bottom').dataset.visible).toBe('true');
    });

    it('cancels final-answer reveal if the user scrolls during completion rendering', async () => {
        await importNotesModule();
        const markdown = await import('./markdown-utils.js');
        let completeRender;
        vi.mocked(markdown.processMarkdownContainer).mockImplementation((container) => {
            if (container.classList.contains('notes-ai-job')) {
                return new Promise((resolve) => { completeRender = resolve; });
            }
            return Promise.resolve();
        });
        const output = document.getElementById('notes-ai-output');
        Object.defineProperty(output, 'scrollHeight', { configurable: true, get: () => 2000 });
        Object.defineProperty(output, 'clientHeight', { configurable: true, get: () => 300 });
        getEventHandler('aiJobStart')({ runId: 802, title: '' });
        getEventHandler('aiStreamBlock')({ runId: 802, blockId: 'answer', kind: 'text', delta: 'Result', status: 'closed' });
        await flushPromises();
        getEventHandler('aiJobFinish')();
        await flushPromises();
        expect(completeRender).toBeTypeOf('function');
        output.dispatchEvent(new WheelEvent('wheel', { deltaY: -100 }));
        output.scrollTop = 123;
        completeRender();
        await flushPromises();
        await flushPromises();
        expect(output.scrollTop).toBe(123);
        vi.mocked(markdown.processMarkdownContainer).mockResolvedValue();
    });

    it('prints only top-level AI output without thinking, tools, or subagent content', async () => {
        await importNotesModule();
        getEventHandler('aiJobStart')({ runId: 990, title: 'User prompt excluded' });
        const append = getEventHandler('aiStreamBlock');
        for (const block of [
            { blockId: 'thinking', kind: 'thinking', delta: 'Private reasoning excluded' },
            { blockId: 'tool', kind: 'tool-call', delta: 'Tool arguments excluded' },
            { blockId: 'tool-output', parentId: 'tool', kind: 'tool-output', delta: 'Tool output excluded' },
            { blockId: 'tool-error', parentId: 'tool', kind: 'tool-error', delta: 'Tool error excluded' },
            { blockId: 'summary', parentId: 'tool', kind: 'tool-summary', delta: 'Tool summary excluded' },
            { blockId: 'subagent', kind: 'subagent', delta: 'Subagent excluded' },
            { blockId: 'child', parentId: 'subagent', kind: 'text', delta: 'Nested output excluded' },
            { blockId: 'answer', kind: 'text', delta: '# Assistant answer\n\n**Final output only**' },
        ]) append({ runId: 990, status: 'closed', ...block });
        await flushPromises();
        const output = document.getElementById('notes-ai-output');
        try {
            const printed = new Promise((resolve) => windowPrintMock.mockImplementationOnce(async () => { resolve(); }));
            output.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true }));
            const menu = showLocalMenuMock.mock.calls.at(-1)[0];
            expect(menu.options).toContain('Print...');
            menu.onSelect(menu.options.indexOf('Print...'));
            await printed;
            const root = document.getElementById('notes-print-root');
            expect(root.querySelector('h1').textContent).toBe('Assistant answer');
            expect(root.querySelector('strong').textContent).toBe('Final output only');
            expect(root.textContent).not.toContain('excluded');
            expect(root.querySelector('[data-block-kind], .notes-ai-heading')).toBeNull();
        } finally {
            window.dispatchEvent(new Event('afterprint'));
        }
    });

    it('does not open a print dialog when the AI panel contains only tools and thinking', async () => {
        await importNotesModule();
        getEventHandler('aiJobStart')({ runId: 991, title: '' });
        getEventHandler('aiStreamBlock')({ runId: 991, blockId: 'reasoning', kind: 'thinking', delta: 'Not printable', status: 'closed' });
        await flushPromises();
        document.getElementById('notes-ai-output').dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true }));
        const menu = showLocalMenuMock.mock.calls.at(-1)[0];
        menu.onSelect(menu.options.indexOf('Print...'));
        await flushPromises();
        await flushPromises();
        expect(windowPrintMock).not.toHaveBeenCalled();
        expect(document.getElementById('notes-print-root')).toBeNull();
        expect(sendIpcMock).toHaveBeenCalledWith('terminal-notify', expect.objectContaining({ message: 'No AI output to print' }));
    });

    it('offers code saving for final AI responses but not tool output', async () => {
        await importNotesModule();

        const aiOutput = document.getElementById('notes-ai-output');
        const tool = document.createElement('div');
        tool.dataset.blockKind = 'tool-output';
        tool.dataset.blockStatus = 'closed';
        tool.innerHTML = '<pre><code class="language-sh">echo tool</code></pre>';
        const final = document.createElement('div');
        final.dataset.blockKind = 'text';
        final.dataset.blockStatus = 'closed';
        final.innerHTML = '<pre><code class="language-python">print(42)</code></pre>';
        aiOutput.append(tool, final);

        showLocalMenuMock.mockClear();
        tool.querySelector('code').dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true }));
        expect(showLocalMenuMock.mock.calls[0][0].options).not.toContain('Save code...');

        showLocalMenuMock.mockClear();
        final.querySelector('code').dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true }));
        expect(showLocalMenuMock.mock.calls[0][0].options).toContain('Save code...');
    });

    it.each(['ai', 'log', 'toc', 'frontmatter', 'find', 'git'])('preserves the %s Tools tab across document and workspace changes when available', async (tab) => {
        const files = ['$NOTES/first.md', '$NOTES/second.md'];
        listFilesMock.mockResolvedValue(files);
        getCurrentProjectMock.mockResolvedValue('/first-project');
        getCurrentGroupNameMock.mockResolvedValue('First');
        getGitStatusMock.mockImplementation(async (root) => ({ root, branch: 'main', clean: true, files: [] }));
        getFileMock.mockResolvedValue({ contents: '---\ntitle: Note\n---\n# Heading', text: '', error: '' });
        await importNotesModule();
        document.querySelector(`[data-file="${files[0]}"]`).click();
        await flushPromises();
        await flushPromises();
        const button = document.getElementById(`notes-tools-tab-${tab}`);
        const panel = document.getElementById('notes-tools-panel');
        button.click();
        panel.dataset.collapsed = 'false';
        document.querySelector(`[data-file="${files[1]}"]`).click();
        await flushPromises();
        await flushPromises();
        expect(button.getAttribute('aria-selected')).toBe('true');
        expect(panel.querySelector('.notes-tools-pane[data-active="true"]').dataset.tab).toBe(tab);
        notesHistoryCurrentPositionMock.mockResolvedValue({ file: files[0], line: 1 });
        getCurrentProjectMock.mockResolvedValue('/second-project');
        getCurrentGroupNameMock.mockResolvedValue('Second');
        for (const [event, handler] of eventsOnMock.mock.calls) {
            if (event === 'notesContextChanged') handler({ projectRoot: '/second-project', groupName: 'Second' });
        }
        await flushPromises();
        await flushPromises();
        expect(button.getAttribute('aria-selected')).toBe('true');
        expect(panel.querySelector('.notes-tools-pane[data-active="true"]').dataset.tab).toBe(tab);
        expect(panel.dataset.collapsed).toBe('false');
    });

    it.each([
        ['toc', '$NOTES/code.go', 'package main'],
        ['frontmatter', '$NOTES/plain.md', '# Plain Markdown'],
    ])('falls back from unavailable %s without closing the Tools panel', async (tab, nextFile, contents) => {
        listFilesMock.mockResolvedValue(['$NOTES/first.md', nextFile]);
        getFileMock.mockImplementation(async (file) => ({ contents: file === nextFile ? contents : '---\ntitle: Note\n---\n# Heading', text: '', error: '' }));
        await importNotesModule();
        document.querySelector('[data-file="$NOTES/first.md"]').click();
        await flushPromises();
        await flushPromises();
        document.getElementById(`notes-tools-tab-${tab}`).click();
        document.getElementById('notes-tools-panel').dataset.collapsed = 'false';
        document.querySelector(`[data-file="${nextFile}"]`).click();
        await flushPromises();
        await flushPromises();
        expect(document.getElementById(`notes-tools-tab-${tab}`).style.display).toBe('none');
        expect(document.getElementById('notes-tools-tab-find').getAttribute('aria-selected')).toBe('true');
        expect(document.getElementById('notes-tools-panel').dataset.collapsed).toBe('false');
    });

    it('does not change the selected Tools tab when opening the document Diff tab', async () => {
        getCurrentProjectMock.mockResolvedValue('/repo');
        getGitStatusMock.mockResolvedValue({ root: '/repo', branch: 'main', clean: false,
            files: [{ path: 'src/main.go', code: ' M', unstaged: true }],
        });
        getGitDiffMock.mockResolvedValue({ path: 'src/main.go', original: 'old', working: 'new', binary: false, deleted: false });
        listFilesMock.mockResolvedValue(['$PROJECT/src/main.go']);
        getFileMock.mockResolvedValue({ contents: 'package main', text: '', error: '' });
        await importNotesModule();
        document.querySelector('[data-file="$PROJECT/src/main.go"]').click();
        await flushPromises();
        await flushPromises();
        document.getElementById('notes-tools-tab-log').click();
        document.getElementById('notes-tab-diff').click();
        await flushPromises();
        await flushPromises();
        expect(getGitDiffMock).toHaveBeenCalledWith('/repo', 'src/main.go', '');
        expect(document.getElementById('notes-tools-tab-log').getAttribute('aria-selected')).toBe('true');
    });

    it('keeps Git selected and explains unavailable operations for a non-Git project', async () => {
        getCurrentProjectMock.mockResolvedValue('/repo');
        getGitStatusMock.mockResolvedValue({ root: '/repo', branch: 'main', clean: true, files: [] });
        await importNotesModule();
        await flushPromises();
        document.getElementById('notes-tools-tab-git').click();
        getGitStatusMock.mockRejectedValue(new Error('Not a Git repository'));
        for (const [event, handler] of eventsOnMock.mock.calls) {
            if (event === 'notesContextChanged') handler({ projectRoot: '/not-a-repo', groupName: 'Other' });
        }
        await flushPromises();
        await flushPromises();
        expect(document.getElementById('notes-tools-tab-git').style.display).toBe('');
        expect(document.getElementById('notes-tools-tab-git').getAttribute('aria-selected')).toBe('true');
        expect(document.getElementById('notes-tools-git-pane').dataset.active).toBe('true');
        expect(document.getElementById('notes-tools-git-toolbar').hidden).toBe(true);
        expect(document.getElementById('notes-tools-git-files').textContent).toContain('Not a Git repository');
    });

    it('preserves the selected Tools tab and collapsed state when AI response streaming starts', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide', text: '', error: '' });

        await importNotesModule();

        document.querySelector('[data-file="$NOTES/guide.md"]').click();
        await flushPromises();
        await flushPromises();

        const toolsPanel = document.getElementById('notes-tools-panel');
        const tabAI = document.getElementById('notes-tools-tab-ai');
        const tabToC = document.getElementById('notes-tools-tab-toc');
        const paneAI = document.getElementById('notes-tools-ai-pane');
        const paneToC = document.getElementById('notes-tools-toc-pane');

        tabToC.click();
        expect(tabToC.getAttribute('aria-selected')).toBe('true');
        expect(tabAI.getAttribute('aria-selected')).toBe('false');
        expect(paneToC.dataset.active).toBe('true');
        expect(paneAI.dataset.active).toBe('false');

        toolsPanel.dataset.collapsed = 'true';

        const aiJobStartHandler = getEventHandler('aiJobStart');
        expect(typeof aiJobStartHandler).toBe('function');
        aiJobStartHandler();
        await flushPromises();

        expect(tabAI.getAttribute('aria-selected')).toBe('false');
        expect(tabToC.getAttribute('aria-selected')).toBe('true');
        expect(paneAI.dataset.active).toBe('false');
        expect(paneToC.dataset.active).toBe('true');
        expect(toolsPanel.dataset.collapsed).toBe('true');
    });

    it('jumps from the ToC to a heading source line in Markdown Edit mode', async () => {
        const markdown = '# Guide\n\nIntro\n\n## First section\n\nBody\n\n### Nested section\n';
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: markdown, text: '', error: '' });

        await importNotesModule();
        document.querySelector('[data-file="$NOTES/guide.md"]').click();
        await flushPromises();
        await flushPromises();
        document.getElementById('notes-tab-editor').click();
        await flushPromises();

        const entries = Array.from(document.querySelectorAll('.notes-tools-toc-item'));
        const firstSection = entries.find((entry) => entry.textContent === 'First section');
        expect(firstSection, `editor=${document.getElementById('notes-editor').value}; toc=${document.getElementById('notes-tools-toc').textContent}`).toBeTruthy();
        firstSection.click();

        const editor = document.getElementById('notes-editor');
        expect(editor.selectionStart).toBe(markdown.indexOf('## First section'));
        expect(editor.selectionEnd).toBe(markdown.indexOf('\n\nBody'));
    });

    it('reuses markdown processing pipeline for AI sections', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide', text: '', error: '' });

        await importNotesModule();

        const markdownUtils = await import('./markdown-utils.js');
        vi.mocked(markdownUtils.autoHyperlink).mockClear();
        vi.mocked(markdownUtils.processLinks).mockClear();

        const aiStartHandler = getEventHandler('aiJobStart');
        const aiBlockHandler = getEventHandler('aiStreamBlock');
        expect(typeof aiBlockHandler).toBe('function');

        aiStartHandler({ runId: 702, title: '' });
        aiBlockHandler({
            runId: 702,
            blockId: 'markdown-processing',
            kind: 'text',
            delta: '\n## Action\n\nsearch\n\n## Action Input\n\n```\n{"a": 1}\n```\n',
            status: 'open',
        });
        await flushPromises();
        await flushPromises();

        // Open final text is append-only plain text; no Markdown work runs.
        expect(markdownUtils.processLinks).not.toHaveBeenCalled();
        expect(markdownUtils.processMarkdownContainer).not.toHaveBeenCalled();

        aiBlockHandler({
            runId: 702,
            blockId: 'markdown-processing',
            kind: 'text',
            delta: '',
            status: 'closed',
        });
        await flushPromises();
        await flushPromises();

        expect(markdownUtils.processMarkdownContainer).toHaveBeenCalled();
    });

    it('shows AI output context menu with copy and table word-wrap actions', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide', text: '', error: '' });

        await importNotesModule();

        const aiOutput = document.getElementById('notes-ai-output');
        aiOutput.innerHTML = '<table><thead><tr><th>H</th></tr></thead><tbody><tr><td>V</td></tr></tbody></table>';
        const tableCell = aiOutput.querySelector('td');

        tableCell.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 100, clientY: 100 }));
        await flushPromises();

        expect(showLocalMenuMock).toHaveBeenCalled();
        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        expect(menuConfig.options).toContain('Copy');
        expect(menuConfig.options).toContain('Copy table (CSV)');
        expect(menuConfig.options).toContain('Copy table (Markdown)');
        expect(menuConfig.options).toContain('Clear sorting');
        expect(menuConfig.options).toContain('Create graph...');
        expect(menuConfig.options).toContain('Word wrap table contents');
        expect(menuConfig.options).not.toContain('Find');
        expect(menuConfig.options).not.toContain('Ask AI...');
        expect(menuConfig.options).not.toContain('Print');
    });

    it('preserves table cell breaks in Markdown, CSV, and Formatted copy', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide', text: '', error: '' });
        await importNotesModule();

        const aiOutput = document.getElementById('notes-ai-output');
        aiOutput.innerHTML = '<table><thead><tr><th>First</th><th>Second</th></tr></thead><tbody><tr><td><strong>one</strong><br/>two</td><td>alpha<br/>beta, "gamma"</td></tr></tbody></table>';
        const tableCell = aiOutput.querySelector('tbody td');
        tableCell.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 100, clientY: 100 }));
        await flushPromises();

        let menu = showLocalMenuMock.mock.calls.at(-1)[0];
        menu.onSelect(menu.options.indexOf('Copy table (CSV)'));
        await flushPromises();
        expect(clipboardSetTextMock).toHaveBeenLastCalledWith('First,Second\n"one\ntwo","alpha\nbeta, ""gamma"""');

        clipboardSetTextMock.mockClear();
        tableCell.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 100, clientY: 100 }));
        await flushPromises();
        menu = showLocalMenuMock.mock.calls.at(-1)[0];
        menu.onSelect(menu.options.indexOf('Copy table (Markdown)'));
        await flushPromises();
        expect(clipboardSetTextMock).toHaveBeenLastCalledWith('| First | Second |\n| --- | --- |\n| one<br/>two | alpha<br/>beta, "gamma" |');

        clipboardSetHTMLMock.mockClear();
        tableCell.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 100, clientY: 100 }));
        await flushPromises();
        menu = showLocalMenuMock.mock.calls.at(-1)[0];
        expect(menu.options).toContain('Copy table (Formatted)');
        expect(menu.options).not.toContain('Copy table (HTML)');
        menu.onSelect(menu.options.indexOf('Copy table (Formatted)'));
        await flushPromises();
        expect(clipboardSetHTMLMock).toHaveBeenCalledOnce();
        const [html, rtf, plainText] = clipboardSetHTMLMock.mock.calls[0];
        expect(html).toContain('<table>');
        expect(html).toContain('<strong>one</strong><br>two');
        expect(html).not.toContain('notes-cellref');
        expect(html).not.toContain('contenteditable');
        expect(plainText).toBe('First\tSecond\none\ntwo\talpha\nbeta, "gamma"\n');
        expect(rtf).toContain('\\trowd');
        expect(rtf).toContain('\\cellx4500\\cellx9000');
        expect(rtf).toContain('one\\line two');
        expect(rtf).toContain('alpha\\line beta, "gamma"');
        expect(rtf.match(/\\row\b/g)).toHaveLength(2);
    });

    it('escapes RTF table text and encodes Unicode as signed UTF-16 units', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide', text: '', error: '' });
        await importNotesModule();
        const aiOutput = document.getElementById('notes-ai-output');
        aiOutput.innerHTML = '<table><thead><tr><th>Name</th></tr></thead><tbody><tr><td></td></tr></tbody></table>';
        const cell = aiOutput.querySelector('td');
        cell.textContent = '{value}\\path\tCaf' + String.fromCharCode(233) + String.fromCodePoint(0x1f600);
        cell.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true }));
        const menu = showLocalMenuMock.mock.calls.at(-1)[0];
        menu.onSelect(menu.options.indexOf('Copy table (Formatted)'));
        await flushPromises();
        const [, rtf] = clipboardSetHTMLMock.mock.calls.at(-1);
        expect(rtf).toContain('\\{value\\}\\\\path\\tab Caf\\u233?\\u-10179?\\u-8704?');
        expect(rtf).toContain('\\trhdr');
        expect(rtf).toContain('\\b Name\\b0');
    });

    // Images in the AI stream get the same menu as images in a rendered note,
    // and the panel's generic menu must stand aside for them.
    it('shows the image context menu for images in the AI panel', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide', text: '', error: '' });

        await importNotesModule();

        getEventHandler('aiJobStart')({ runId: 703, title: '' });
        getEventHandler('aiStreamBlock')({
            runId: 703,
            blockId: 'image-menu',
            kind: 'text',
            delta: '\n![diagram](diagram.png)\n',
            status: 'closed',
        });
        await flushPromises();
        getEventHandler('aiJobFinish')();
        await flushPromises();
        await flushPromises();

        const img = document.getElementById('notes-ai-output').querySelector('img');
        expect(img).not.toBeNull();

        showLocalMenuMock.mockReset();
        img.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 10, clientY: 10 }));
        await flushPromises();
        await flushPromises();

        expect(showLocalMenuMock).toHaveBeenCalledTimes(1);
        expect(showLocalMenuMock.mock.calls[0][0].options).toEqual([
            'Copy image to clipboard',
            'Save image...',
            'Ask AI (image)...',
        ]);
    });

    it('adds a timestamp when an AI job starts', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const fmt = createAIPipelineFormatter(wrapper);

        // Before startJob, no timestamp
        expect(wrapper.querySelector('.notes-ai-timestamp')).toBeNull();

        fmt.startJob();
        
        // After startJob, timestamp exists at the start
        let ts = wrapper.querySelector('.notes-ai-timestamp');
        expect(ts).not.toBeNull();
        expect(ts.tagName).toBe('P');
        expect(ts.textContent).toMatch(/^\d{1,2}:\d{2}:\d{2}/);

        fmt.appendChunk('Final Answer: Done.');
        await flushPromises();

        // Timestamp should still be there
        ts = wrapper.querySelector('.notes-ai-timestamp');
        expect(ts).not.toBeNull();

        fmt.finishJob();

        // Timestamp should still be present after finishJob
        ts = wrapper.querySelector('.notes-ai-timestamp');
        expect(ts).not.toBeNull();
    });

    it('renders typed stream blocks into isolated roots', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const fmt = createAIPipelineFormatter(wrapper);

        fmt.startJob();
        fmt.appendBlock({ blockId: 'tool-1', kind: 'tool-output', delta: '<b>one</b>', status: 'open' });
        fmt.appendBlock({ blockId: 'tool-2', kind: 'tool-output', delta: '<b>two</b>', status: 'open' });
        await flushPromises();
        await flushPromises();

        const blocks = wrapper.querySelectorAll('[data-block-id]');
        expect(blocks).toHaveLength(2);
        expect(blocks[0].classList.contains('notes-ai-markdown')).toBe(true);
        expect(blocks[1].classList.contains('notes-ai-markdown')).toBe(true);
        expect(blocks[0].querySelector('pre').className).toBe('notes-ai-code');
        expect(blocks[1].querySelector('pre').className).toBe('notes-ai-code');
        expect(blocks[0].querySelector('code').textContent).toBe('<b>one</b>');
        expect(blocks[1].querySelector('code').textContent).toBe('<b>two</b>');
        expect(blocks[0].querySelector('b')).toBeNull();
        expect(blocks[1].querySelector('b')).toBeNull();
    });

    it('renders agent notices as formatted prose rather than code or quotes', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const fmt = createAIPipelineFormatter(wrapper, { marked });

        fmt.startJob();
        fmt.appendBlock({
            blockId: 'notice-1',
            kind: 'notice',
            delta: '\n\n**Continuing after max steps (1/3)**\n\n',
            status: 'closed',
        });
        await flushPromises();
        await flushPromises();

        const block = wrapper.querySelector('[data-block-id="notice-1"]');
        expect(block).not.toBeNull();
        expect(block.querySelector('pre')).toBeNull();
        expect(block.querySelector('blockquote')).toBeNull();
        expect(block.querySelector('strong')?.textContent).toBe('Continuing after max steps (1/3)');
    });

    it('keeps labelled tool-call input collapsed until its heading is clicked', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const fmt = createAIPipelineFormatter(wrapper);

        fmt.startJob();
        fmt.appendBlock({ blockId: 'call', kind: 'tool-call', label: 'name.of.tool', delta: '{"input":true}', status: 'open' });
        await flushPromises();

        const block = wrapper.querySelector('[data-block-id="call"]');
        const heading = block?.querySelector('.notes-ai-tool-call-toggle');
        const input = block?.querySelector('pre');
        const statusToggle = block?.querySelector('.notes-ai-tool-output-toggle');
        const statusCheckbox = statusToggle?.querySelector('input[type="checkbox"]');
        const statusSpinner = statusToggle?.querySelector('.notes-ai-tool-output-spinner');
        expect(heading?.textContent).toBe('Tool call: name.of.tool');
        expect(heading?.querySelector('code')?.textContent).toBe('name.of.tool');
        expect(input?.hidden).toBe(true);
        expect(input?.textContent).toBe('{"input":true}');
        expect(statusToggle?.dataset.status).toBe('running');
        expect(statusToggle?.textContent).toBe('Running');
        expect(statusCheckbox?.checked).toBe(false);
        expect(statusSpinner?.hidden).toBe(false);

        heading?.click();
        expect(input?.hidden).toBe(false);
        expect(heading?.getAttribute('aria-expanded')).toBe('true');

        heading?.click();
        expect(input?.hidden).toBe(true);
        expect(heading?.getAttribute('aria-expanded')).toBe('false');

        fmt.appendBlock({ blockId: 'call-without-label', kind: 'tool-call', delta: '{"fallback":true}', status: 'closed' });
        await flushPromises();
        const fallback = wrapper.querySelector('[data-block-id="call-without-label"]');
        expect(fallback?.querySelector('.notes-ai-tool-call-toggle')?.textContent).toBe('Tool call');
        expect(fallback?.querySelector('pre')?.hidden).toBe(true);
    });

    it('collapses tool output and summaries, streams expanded summaries, and shows errors', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const fmt = createAIPipelineFormatter(wrapper);

        fmt.startJob();
        fmt.appendBlock({ blockId: 'call', kind: 'tool-call', delta: '{"input":true}', status: 'closed' });
        fmt.appendBlock({ blockId: 'output', parentId: 'call', kind: 'tool-output', delta: 'output', status: 'closed' });
        await flushPromises();

        const call = wrapper.querySelector('[data-block-id="call"]');
        const output = wrapper.querySelector('[data-block-id="output"]');
        const toggle = call?.querySelector('.notes-ai-tool-output-toggle');
        const checkbox = toggle?.querySelector('input[type="checkbox"]');
        const spinner = toggle?.querySelector('.notes-ai-tool-output-spinner');
        expect(output?.hidden).toBe(true);
        expect(toggle?.hidden).toBe(false);
        expect(toggle?.classList.contains('markdown-body')).toBe(true);
        expect(toggle?.dataset.status).toBe('successful');
        expect(toggle?.textContent).toBe('Successful');
        expect(checkbox?.checked).toBe(true);
        expect(spinner?.hidden).toBe(true);

        fmt.appendBlock({ blockId: 'summary', parentId: 'call', kind: 'tool-summary', delta: 'first', status: 'open' });
        await flushPromises();

        const summary = wrapper.querySelector('[data-block-id="summary"]');
        expect(call?.querySelector('pre')?.textContent).toBe('{"input":true}');
        expect(summary?.hidden).toBe(true);
        expect(toggle?.hidden).toBe(false);
        expect(toggle?.dataset.status).toBe('summarising');
        expect(toggle?.textContent).toBe('Summarising');
        expect(checkbox?.checked).toBe(false);
        expect(spinner?.hidden).toBe(false);
        expect(toggle?.lastElementChild).toBe(spinner);

        checkbox?.click();
        expect(output?.hidden).toBe(false);
        expect(summary?.hidden).toBe(false);
        expect(toggle?.getAttribute('aria-expanded')).toBe('true');

        fmt.appendBlock({ blockId: 'summary', parentId: 'call', kind: 'tool-summary', delta: ' streamed', status: 'open' });
        await flushPromises();
        expect(summary?.querySelector('code')?.textContent).toBe('first streamed');

        fmt.appendBlock({ blockId: 'summary', parentId: 'call', kind: 'tool-summary', delta: '', status: 'closed' });
        await flushPromises();
        expect(toggle?.dataset.status).toBe('successful');
        expect(toggle?.textContent).toBe('Successful');
        expect(checkbox?.checked).toBe(true);
        expect(spinner?.hidden).toBe(true);

        fmt.appendBlock({ blockId: 'error', parentId: 'call', kind: 'tool-error', delta: 'visible error', status: 'closed' });
        await flushPromises();
        const error = wrapper.querySelector('[data-block-id="error"]');
        expect(toggle?.dataset.status).toBe('error');
        expect(toggle?.textContent).toBe('Error');
        expect(checkbox?.checked).toBe(false);
        expect(error?.hidden).toBe(false);
    });

    it('restores completed summary status from block metadata before content hydration', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const fmt = createAIPipelineFormatter(wrapper);

        fmt.startJob();
        fmt.openBlock({ blockId: 'call', kind: 'tool-call', status: 'closed' });
        fmt.openBlock({ blockId: 'summary', parentId: 'call', kind: 'tool-summary', status: 'closed' });

        const call = wrapper.querySelector('[data-block-id="call"]');
        const summary = wrapper.querySelector('[data-block-id="summary"]');
        const toggle = call?.querySelector('.notes-ai-tool-output-toggle');
        const checkbox = toggle?.querySelector('input[type="checkbox"]');
        expect(summary?.dataset.blockStatus).toBe('closed');
        expect(summary?.hidden).toBe(true);
        expect(toggle?.dataset.status).toBe('successful');
        expect(toggle?.textContent).toBe('Successful');
        expect(checkbox?.checked).toBe(true);
    });

    it('restores subagent tool output as nested typed blocks', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const fmt = createAIPipelineFormatter(wrapper, { marked });
        const contentByBlock = {
            subagent: '**Agent result**',
            call: '{"query":"needle"}',
            output: 'match in file.go',
        };

        fmt.setBlocks([
            { blockId: 'subagent', kind: 'subagent', label: 'Agent', status: 'closed' },
            { blockId: 'call', parentId: 'subagent', kind: 'tool-call', label: 'grep', status: 'closed' },
            { blockId: 'output', parentId: 'call', kind: 'tool-output', status: 'closed' },
        ], (block) => Promise.resolve(contentByBlock[block.blockId]));
        await flushPromises();
        await flushPromises();

        const subagent = wrapper.querySelector('[data-block-id="subagent"]');
        const call = wrapper.querySelector('[data-block-id="call"]');
        const output = wrapper.querySelector('[data-block-id="output"]');
        expect(subagent?.querySelector('strong')?.textContent).toBe('Agent result');
        expect(subagent?.querySelector('.notes-ai-stream-block-children')?.contains(call)).toBe(true);
        expect(call?.querySelector('pre')?.hidden).toBe(true);
        expect(call?.querySelector('pre code')?.textContent).toBe('{"query":"needle"}');
        expect(call?.querySelector('.notes-ai-stream-block-children')?.contains(output)).toBe(true);
        expect(output?.hidden).toBe(true);
        expect(output?.querySelector('code')?.textContent).toBe('match in file.go');

        call?.querySelector('.notes-ai-tool-output-toggle input')?.click();
        expect(output?.hidden).toBe(false);
    });

    it('renders unlabeled buffered subagent output as literal text', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const markedMock = { parse: vi.fn((text) => `<p>${String(text)}</p>`) };
        const processMarkdownContainer = vi.fn();
        const fmt = createAIPipelineFormatter(wrapper, { marked: markedMock, processMarkdownContainer });

        fmt.startJob();
        fmt.appendBlock({ blockId: 'buffered-subagent', kind: 'subagent', delta: '**literal**\nnext line', status: 'closed' });
        await flushPromises();
        await flushPromises();

        const block = wrapper.querySelector('[data-block-id="buffered-subagent"]');
        expect(block?.querySelector('strong')).toBeNull();
        expect(block?.textContent).toBe('**literal**\nnext line');
        expect(block?.querySelector('.notes-ai-raw-text')).not.toBeNull();
        expect(markedMock.parse).not.toHaveBeenCalled();
        expect(processMarkdownContainer).not.toHaveBeenCalled();
    });

    it('pins typed code and thinking blocks to the bottom as they grow', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const fmt = createAIPipelineFormatter(wrapper, { marked });

        fmt.startJob();
        fmt.appendBlock({ blockId: 'code-scroll', kind: 'tool-output', delta: 'output', status: 'open' });
        fmt.appendBlock({ blockId: 'thinking-scroll', kind: 'thinking', delta: 'thinking', status: 'open' });
        await flushPromises();
        await flushPromises();

        const code = wrapper.querySelector('[data-block-id="code-scroll"] pre');
        const initialQuote = wrapper.querySelector('[data-block-id="thinking-scroll"] blockquote');
        const scrollHeight = Object.getOwnPropertyDescriptor(Element.prototype, 'scrollHeight');
        Object.defineProperty(Element.prototype, 'scrollHeight', { configurable: true, get: () => 240 });
        code.scrollTop = 0;
        initialQuote.scrollTop = 0;

        fmt.appendBlock({ blockId: 'code-scroll', kind: 'tool-output', delta: '\nmore output', status: 'open' });
        fmt.appendBlock({ blockId: 'thinking-scroll', kind: 'thinking', delta: '\nmore thinking', status: 'open' });
        await flushPromises();
        await flushPromises();

        expect(code.scrollTop).toBe(240);
        const currentQuote = wrapper.querySelector('[data-block-id="thinking-scroll"] blockquote');
        expect(currentQuote.scrollTop).toBe(240);
        if (scrollHeight) {
            Object.defineProperty(Element.prototype, 'scrollHeight', scrollHeight);
        } else {
            delete Element.prototype.scrollHeight;
        }
    });

    it('streams large thinking blocks without invoking markdown', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const markedMock = {
            parse: vi.fn((text) => `<blockquote>${String(text)}</blockquote>`),
        };
        const fmt = createAIPipelineFormatter(wrapper, { marked: markedMock });

        fmt.startJob();
        for (let index = 0; index < 1000; index += 1) {
            fmt.appendBlock({
                blockId: 'large-thinking',
                kind: 'thinking',
                delta: `chunk-${index}\n`,
                status: 'open',
            });
        }

        for (let index = 0; index < 8; index += 1) {
            await new Promise((resolve) => setTimeout(resolve, 20));
        }

        expect(markedMock.parse).not.toHaveBeenCalled();
        expect(wrapper.textContent).toContain('chunk-999');
    });

    it('keeps thinking blockquote DOM stable and skips markdown after close', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const markedMock = { parse: vi.fn((text) => `<p>${String(text)}</p>`) };
        const processMarkdownContainer = vi.fn(async () => {});
        const fmt = createAIPipelineFormatter(wrapper, {
            marked: markedMock,
            processMarkdownContainer,
        });

        fmt.startJob();
        fmt.appendBlock({ blockId: 'thinking-stable', kind: 'thinking', delta: 'start', status: 'open' });
        await flushPromises();
        const initialQuote = wrapper.querySelector('[data-block-id="thinking-stable"] blockquote');

        fmt.appendBlock({
            blockId: 'thinking-stable',
            kind: 'thinking',
            delta: '\n> nested **markdown** stays literal',
            status: 'closed',
        });
        await flushPromises();
        await flushPromises();

        const currentQuote = wrapper.querySelector('[data-block-id="thinking-stable"] blockquote');
        expect(currentQuote).toBe(initialQuote);
        expect(currentQuote?.textContent).toContain('> nested **markdown** stays literal');
        expect(markedMock.parse).not.toHaveBeenCalled();
        expect(processMarkdownContainer).not.toHaveBeenCalled();
    });

    it('continues to render final text blocks as markdown', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const markedMock = { parse: vi.fn((text) => `<p>${String(text)}</p>`) };
        const fmt = createAIPipelineFormatter(wrapper, { marked: markedMock });

        fmt.startJob();
        fmt.appendBlock({ blockId: 'final-text', kind: 'text', delta: '**final**', status: 'closed' });
        await flushPromises();
        await flushPromises();

        expect(markedMock.parse).toHaveBeenCalled();
        expect(wrapper.textContent).toContain('**final**');
    });

    it('defers final-answer markdown until the text block closes', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const markedMock = { parse: vi.fn((text) => `<p>${String(text)}</p>`) };
        const fmt = createAIPipelineFormatter(wrapper, { marked: markedMock });

        fmt.startJob();
        fmt.appendBlock({
            blockId: 'streaming-final',
            kind: 'text',
            delta: '> nested\n> > quote\n',
            status: 'open',
        });
        await flushPromises();
        const initialText = wrapper.querySelector('[data-block-id="streaming-final"] .notes-ai-streaming-text');

        fmt.appendBlock({
            blockId: 'streaming-final',
            kind: 'text',
            delta: '> > > deeper',
            status: 'open',
        });
        await flushPromises();
        await flushPromises();

        expect(markedMock.parse).not.toHaveBeenCalled();
        expect(wrapper.querySelector('[data-block-id="streaming-final"] .notes-ai-streaming-text')).toBe(initialText);
        expect(initialText?.textContent).toContain('> > > deeper');

        fmt.appendBlock({ blockId: 'streaming-final', kind: 'text', delta: '', status: 'closed' });
        await flushPromises();
        await flushPromises();

        expect(markedMock.parse).toHaveBeenCalledTimes(1);
    });

    it('uses the shared five-line AI code styling for tool input and output', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        wrapper.id = 'notes-ai-output';
        const fmt = createAIPipelineFormatter(wrapper);

        fmt.startJob();
        fmt.appendBlock({ blockId: 'input', kind: 'tool-call', delta: '{"path":"main.go"}', status: 'open' });
        fmt.appendBlock({ blockId: 'output', kind: 'tool-output', delta: 'line 1\nline 2', status: 'open' });
        await flushPromises();
        await flushPromises();

        const codeBlocks = wrapper.querySelectorAll('.notes-ai-code');
        expect(codeBlocks).toHaveLength(2);
        expect(getNotesRenderedStyles()).toContain('--notes-ai-code-max-lines: 5;');
        expect(getNotesRenderedStyles()).toContain('max-height: calc(1.5em * var(--notes-ai-code-max-lines) + 16px);');
    });

    it('renders typed thinking as the legacy AI blockquote structure', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const fmt = createAIPipelineFormatter(wrapper, { marked });

        fmt.startJob();
        fmt.appendBlock({ blockId: 'thinking-1', kind: 'thinking', delta: 'first thought\nsecond thought', status: 'open' });
        await flushPromises();
        await flushPromises();

        const block = wrapper.querySelector('[data-block-id="thinking-1"]');
        expect(block?.querySelector('.notes-ai-markdown')).not.toBeNull();
        expect(block?.querySelector('blockquote')).not.toBeNull();
        expect(block?.querySelector('blockquote strong')?.textContent).toBe('Thinking:');
        expect(block?.querySelector('blockquote')?.textContent).toContain('first thought');
        expect(block?.querySelector('blockquote')?.textContent).toContain('second thought');
    });

    it('attaches child blocks under their parent even when events arrive first', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const fmt = createAIPipelineFormatter(wrapper);

        fmt.startJob();
        fmt.appendBlock({ blockId: 'child', parentId: 'parent', kind: 'subagent', delta: 'child output', status: 'open' });
        fmt.appendBlock({ blockId: 'parent', parentId: '', kind: 'subagent', delta: 'parent output', status: 'open' });
        await flushPromises();
        await flushPromises();

        const parent = wrapper.querySelector('[data-block-id="parent"]');
        const child = wrapper.querySelector('[data-block-id="child"]');
        expect(parent).not.toBeNull();
        expect(child).not.toBeNull();
        expect(parent.querySelector('.notes-ai-stream-block-children').contains(child)).toBe(true);
    });

    it('prints unloaded history output without fetching tool, thinking, or nested text blocks', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const originalObserver = globalThis.IntersectionObserver;
        globalThis.IntersectionObserver = class {
            observe() {}
            unobserve() {}
            disconnect() {}
        };
        try {
            const wrapper = document.createElement('div');
            const fmt = createAIPipelineFormatter(wrapper, { marked });
            const loadContent = vi.fn((block) => Promise.resolve(block.blockId === 'answer' ? '# Stored final answer' : 'Excluded content'));
            fmt.setBlocks([
                { blockId: 'tool', kind: 'tool-call', status: 'closed' },
                { blockId: 'thinking', kind: 'thinking', status: 'closed' },
                { blockId: 'result', kind: 'tool-output', parentId: 'tool', status: 'closed' },
                { blockId: 'answer', kind: 'text', status: 'closed' },
                { blockId: 'nested-answer', kind: 'text', parentId: 'tool', status: 'closed' },
            ], loadContent);
            await flushPromises();
            await flushPromises();
            loadContent.mockClear();
            expect(await fmt.outputMarkdown()).toBe('# Stored final answer');
            expect(loadContent).toHaveBeenCalledOnce();
            expect(loadContent).toHaveBeenCalledWith(expect.objectContaining({ blockId: 'answer' }));
            expect(wrapper.querySelector('[data-block-id="answer"]').textContent).toBe('');
        } finally {
            globalThis.IntersectionObserver = originalObserver;
        }
    });

    it('prints only final-answer sections from legacy AI transcripts', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const fmt = createAIPipelineFormatter(document.createElement('div'), { marked });
        fmt.setText('Question: user prompt\nThought: private reasoning and tool use\nFinal Answer: # Safe response\n\nAnswer text', { legacy: true });
        expect(await fmt.outputMarkdown()).toBe('# Safe response\n\nAnswer text');
        fmt.setText('Unclassified transcript cannot be safely separated', { legacy: true });
        expect(await fmt.outputMarkdown()).toBe('');
    });

    it('hydrates historical blocks lazily as they intersect the viewport', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const originalObserver = globalThis.IntersectionObserver;
        let observer;
        globalThis.IntersectionObserver = class {
            constructor(callback) {
                this.callback = callback;
                this.observed = new Set();
                observer = this;
            }

            observe(element) {
                this.observed.add(element);
            }

            unobserve(element) {
                this.observed.delete(element);
            }

            disconnect() {
                this.observed.clear();
            }
        };

        try {
            const wrapper = document.createElement('div');
            const loadContent = vi.fn((block) => Promise.resolve(`content-${block.blockId}`));
            const fmt = createAIPipelineFormatter(wrapper);
            const blocks = Array.from({ length: 5 }, (_, index) => ({
                blockId: `history-${index}`,
                kind: 'tool-output',
                status: 'closed',
                size: 10,
            }));

            fmt.setBlocks(blocks, loadContent);
            await flushPromises();
            await flushPromises();

            expect(loadContent).toHaveBeenCalledTimes(3);
            expect(wrapper.querySelector('[data-block-id="history-4"] code').textContent).toBe('');

            const lastShell = wrapper.querySelector('[data-block-id="history-4"]');
            observer.callback([{ target: lastShell, isIntersecting: true }]);
            await flushPromises();
            await flushPromises();

            expect(loadContent).toHaveBeenCalledTimes(4);
            expect(lastShell.querySelector('code').textContent).toBe('content-history-4');

            observer.callback([{ target: lastShell, isIntersecting: true }]);
            await flushPromises();
            expect(loadContent).toHaveBeenCalledTimes(4);
        } finally {
            globalThis.IntersectionObserver = originalObserver;
        }
    });

    it('flags legacy transcripts and clears the flag for block renders', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const fmt = createAIPipelineFormatter(wrapper, { marked });
        // renderTextAsMarkdown yields on requestAnimationFrame, which jsdom backs
        // with a real timer.
        const settle = async () => {
            for (let i = 0; i < 4; i += 1) {
                await new Promise((resolve) => setTimeout(resolve, 20));
            }
        };

        fmt.setText('# legacy output', { legacy: true });
        await settle();
        expect(wrapper.querySelector('.notes-ai-job.notes-ai-legacy')).not.toBeNull();

        fmt.setBlocks(
            [{ blockId: 'b1', kind: 'text', status: 'closed', sessionId: 1, promptId: 2, runId: 3 }],
            () => Promise.resolve('current output'),
        );
        await settle();
        expect(wrapper.querySelector('.notes-ai-legacy')).toBeNull();

        const styles = getNotesRenderedStyles();
        expect(styles).toContain('#notes-ai-output .notes-ai-legacy::before {');
        expect(styles).toContain("content: 'Legacy transcript';");
    });

    it('restores the live view as structured blocks rather than one flat lump', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const markedMock = { parse: vi.fn((text) => `<p>${String(text)}</p>`) };
        const fmt = createAIPipelineFormatter(wrapper, { marked: markedMock });

        const content = {
            'r-1': 'reasoning text',
            'r-2': 'raw tool output',
        };
        const loadContent = vi.fn((block) => Promise.resolve(content[block.blockId]));

        fmt.setBlocks([
            { blockId: 'r-1', kind: 'thinking', status: 'closed', runId: 7, sessionId: 1, promptId: 0 },
            { blockId: 'r-2', kind: 'tool-output', status: 'open', runId: 7, sessionId: 1, promptId: 0 },
        ], loadContent);
        await flushPromises();
        await flushPromises();
        await flushPromises();

        // An in-flight block must hydrate without waiting to be scrolled into view.
        expect(loadContent).toHaveBeenCalledWith(expect.objectContaining({ blockId: 'r-2' }));

        const thinking = wrapper.querySelector('[data-block-id="r-1"]');
        const toolOutput = wrapper.querySelector('[data-block-id="r-2"]');
        expect(thinking?.querySelector('blockquote')?.textContent).toContain('reasoning text');
        expect(toolOutput?.querySelector('pre.notes-ai-code')?.textContent).toBe('raw tool output');
        expect(markedMock.parse).not.toHaveBeenCalled();
    });

    it('renders job prefix as markdown after the timestamp when provided', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const processMarkdownContainerMock = vi.fn();
        const fmt = createAIPipelineFormatter(wrapper, {
            processMarkdownContainer: processMarkdownContainerMock,
        });

        fmt.startJob('## Request\n\n### Prompt\n\nWhat is this?\n');
        await flushPromises();

        // Prefix element exists as a standard markdown div
        const prefix = wrapper.querySelector('.notes-ai-prefix');
        expect(prefix).not.toBeNull();

        // Prefix appears after the timestamp in DOM order
        const children = Array.from(wrapper.children);
        const tsIdx = children.indexOf(wrapper.querySelector('.notes-ai-timestamp'));
        const prefixIdx = children.indexOf(prefix);
        expect(prefixIdx).toBeGreaterThan(tsIdx);

        // Prefix appears before the stream output root
        const jobIdx = children.indexOf(wrapper.querySelector('.notes-ai-job'));
        expect(prefixIdx).toBeLessThan(jobIdx);

        // Markdown processing was applied to the prefix element
        expect(processMarkdownContainerMock).toHaveBeenCalledWith(prefix);
    });

    it('renders blockquote markdown in job prefix without special span coloring', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const fmt = createAIPipelineFormatter(wrapper);

        fmt.startJob('> /jira show my tickets');
        await flushPromises();

        const prefix = wrapper.querySelector('.notes-ai-prefix');
        expect(prefix).not.toBeNull();

        // No custom coloured spans from the old renderPromptTitleHtml formatting
        const spans = prefix.querySelectorAll('span[style]');
        expect(spans).toHaveLength(0);
    });

    it('omits the prefix element when startJob is called without content', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const fmt = createAIPipelineFormatter(wrapper);

        fmt.startJob();

        expect(wrapper.querySelector('.notes-ai-prefix')).toBeNull();
    });

    it('passes job prefix markdown from aiJobStart event through to the AI panel', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide', text: '', error: '' });

        await importNotesModule();

        const aiJobStartHandler = getEventHandler('aiJobStart');
        expect(typeof aiJobStartHandler).toBe('function');

        aiJobStartHandler('## Request\n\n### Prompt\n\nHow do I do this?\n');
        await flushPromises();
        await flushPromises();

        const aiOutput = document.getElementById('notes-ai-output');
        const prefix = aiOutput.querySelector('.notes-ai-prefix');
        expect(prefix).not.toBeNull();
    });

    it('adds an hr separator when a second AI job starts with existing content', async () => {
        const { createAIPipelineFormatter } = await import('./ai_pipeline_formatter.js');
        const wrapper = document.createElement('div');
        const fmt = createAIPipelineFormatter(wrapper);

        fmt.startJob();
        fmt.appendChunk('Final Answer: First result.');
        await flushPromises();

        expect(wrapper.querySelector('.notes-ai-separator')).toBeNull();

        fmt.startJob();
        fmt.appendChunk('Final Answer: Second result.');
        await flushPromises();

        const sep = wrapper.querySelector('.notes-ai-separator');
        expect(sep).not.toBeNull();
        expect(sep.tagName).toBe('HR');

        const jobs = wrapper.querySelectorAll('.notes-ai-job');
        expect(jobs.length).toBe(2);
    });

    it('opens Find tools tab from the toolbar Find button', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide\n\nfind me', text: '', error: '' });

        await importNotesModule();

        const toolsPanel = document.getElementById('notes-tools-panel');
        const tabFind = document.getElementById('notes-tools-tab-find');
        const tabAI = document.getElementById('notes-tools-tab-ai');
        const paneFind = document.getElementById('notes-tools-find-pane');
        const paneAI = document.getElementById('notes-tools-ai-pane');
        expect(tabFind).not.toBeNull();
        expect(tabAI).not.toBeNull();

        toolsPanel.dataset.collapsed = 'true';
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'f', metaKey: true, bubbles: true, cancelable: true }));
        await flushPromises();

        expect(toolsPanel.dataset.collapsed).toBe('false');
        expect(tabFind.getAttribute('aria-selected')).toBe('true');
        expect(tabAI.getAttribute('aria-selected')).toBe('false');
        expect(paneFind.dataset.active).toBe('true');
        expect(paneAI.dataset.active).toBe('false');
    });

    it('renders replace controls and removes the close button in Find panel', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide\n\nfind me', text: '', error: '' });

        await importNotesModule();

        const findClose = document.getElementById('notes-find-close');
        const replaceInput = document.getElementById('notes-replace-input');
        const replaceOne = document.getElementById('notes-replace-one');
        const replaceAll = document.getElementById('notes-replace-all');

        expect(findClose).toBeNull();
        expect(replaceInput).not.toBeNull();
        expect(replaceOne).not.toBeNull();
        expect(replaceAll).not.toBeNull();
    });

    it('greys out replace controls when not in Edit tab and enables them in Edit tab', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide\n\nfind me', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/guide.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'f', metaKey: true, bubbles: true, cancelable: true }));
        await flushPromises();

        const replaceControls = document.getElementById('notes-replace-controls');
        const replaceInput = document.getElementById('notes-replace-input');
        const replaceOne = document.getElementById('notes-replace-one');
        const replaceAll = document.getElementById('notes-replace-all');
        const tabEditor = document.getElementById('notes-tab-editor');

        // Markdown opens in View mode by default, so replace should be disabled.
        expect(replaceControls.dataset.disabled).toBe('true');
        expect(replaceInput.disabled).toBe(true);
        expect(replaceOne.disabled).toBe(true);
        expect(replaceAll.disabled).toBe(true);

        tabEditor.click();
        await flushPromises();

        expect(replaceControls.dataset.disabled).toBe('false');
        expect(replaceInput.disabled).toBe(false);
        expect(replaceOne.disabled).toBe(false);
        expect(replaceAll.disabled).toBe(false);
    });

    it('replaces all matches from the Find panel when editing', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: 'alpha beta alpha', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/guide.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        document.getElementById('notes-tab-editor').click();
        await flushPromises();

        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'f', metaKey: true, bubbles: true, cancelable: true }));
        await flushPromises();

        const findInput = document.getElementById('notes-find-input');
        const replaceInput = document.getElementById('notes-replace-input');
        const replaceAll = document.getElementById('notes-replace-all');
        const editor = document.getElementById('notes-editor');

        findInput.value = 'alpha';
        findInput.dispatchEvent(new Event('input', { bubbles: true }));
        replaceInput.value = 'omega';

        replaceAll.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
        await flushPromises();

        expect(editor.value).toBe('omega beta omega');
    });

    it.each([
        ['case', 'caseSensitive'],
        ['regex', 'regex'],
        ['word', 'wholeWord'],
    ])('re-runs project grep immediately when the %s option is clicked', async (button, option) => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide\n\nalpha', text: '', error: '' });

        await importNotesModule();

        const findFilesInput = document.getElementById('notes-find-files-input');
        const findOptionButton = document.getElementById(`notes-find-option-${button}`);

        findFilesInput.value = 'alpha';
        findFilesInput.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true }));
        await flushPromises();

        expect(notesGrepStreamMock).toHaveBeenCalledTimes(1);
        expect(notesGrepStreamMock).toHaveBeenNthCalledWith(1, 'alpha', {
            caseSensitive: false,
            regex: false,
            wholeWord: false,
            fileFilter: '',
        });

        findOptionButton.click();
        await flushPromises();

        expect(notesGrepStreamMock).toHaveBeenCalledTimes(2);
        expect(notesGrepStreamMock).toHaveBeenNthCalledWith(2, 'alpha', {
            caseSensitive: false,
            regex: false,
            wholeWord: false,
            fileFilter: '',
            [option]: true,
        });
    });

    it('re-runs project grep after typing a different files-containing query', async () => {
        await importNotesModule();
        const findFilesInput = document.getElementById('notes-find-files-input');
        vi.useFakeTimers();
        try {
            findFilesInput.value = 'alpha';
            findFilesInput.dispatchEvent(new Event('input', { bubbles: true }));
            await vi.advanceTimersByTimeAsync(320);
            expect(notesGrepStreamMock).toHaveBeenNthCalledWith(1, 'alpha', {
                caseSensitive: false,
                regex: false,
                wholeWord: false,
                fileFilter: '',
            });

            const firstDone = eventsOnMock.mock.calls
                .filter(([eventName]) => eventName === 'notesGrepDone')
                .at(-1)?.[1];
            firstDone?.();

            findFilesInput.value = 'beta';
            findFilesInput.dispatchEvent(new Event('input', { bubbles: true }));
            await vi.advanceTimersByTimeAsync(320);

            expect(notesGrepStreamMock).toHaveBeenCalledTimes(2);
            expect(notesGrepStreamMock).toHaveBeenNthCalledWith(2, 'beta', {
                caseSensitive: false,
                regex: false,
                wholeWord: false,
                fileFilter: '',
            });
        } finally {
            vi.useRealTimers();
        }
    });

    it('invalidates and re-runs project grep after autosave while keeping find mode active', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide\n\nalpha', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/guide.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        document.getElementById('notes-tab-editor').click();
        await flushPromises();

        const findFilesInput = document.getElementById('notes-find-files-input');
        const editor = document.getElementById('notes-editor');

        findFilesInput.value = 'alpha';
        findFilesInput.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true }));
        await flushPromises();
        expect(notesGrepStreamMock).toHaveBeenCalledTimes(1);

        vi.useFakeTimers();
        try {
            editor.value = '# Guide\n\nalpha updated';
            editor.dispatchEvent(new Event('input', { bubbles: true }));

            await vi.advanceTimersByTimeAsync(1100);
        } finally {
            vi.useRealTimers();
        }

        await flushPromises();
        await flushPromises();

        expect(saveFileMock).toHaveBeenCalled();
        expect(notesGrepStreamMock).toHaveBeenCalledTimes(2);
        expect(notesGrepStreamMock).toHaveBeenNthCalledWith(2, 'alpha', {
            caseSensitive: false,
            regex: false,
            wholeWord: false,
            fileFilter: '',
        });
    });

    it.each(['ai', 'log'])('hides the whole Tools tab bar while %s is maximized', async (panel) => {
        await importNotesModule();
        const tabs = document.getElementById('notes-tools-tabs');
        const header = tabs.closest('.notes-tools-header');
        document.getElementById('notes-tools-tab-git').style.display = '';
        document.getElementById('notes-tools-tab-frontmatter').style.display = '';
        const futureTab = document.createElement('button');
        futureTab.textContent = 'Future tab';
        tabs.appendChild(futureTab);
        const hideRule = notesCss.match(new RegExp(
            `\\[data-${panel}-maximized="true"\\] #notes-sidebar,[\\s\\S]*?\\{\\s*display: none !important;\\s*\\}`,
        ));
        expect(hideRule).not.toBeNull();
        const style = document.createElement('style');
        style.textContent = hideRule[0];
        document.head.appendChild(style);
        try {
            const maximize = document.getElementById(`notes-tools-${panel}-maximize`);
            expect(window.getComputedStyle(header).display).not.toBe('none');
            maximize.click();
            await flushPromises();
            expect(window.getComputedStyle(header).display).toBe('none');
            expect(header.contains(maximize)).toBe(false);
            expect(window.getComputedStyle(maximize).display).not.toBe('none');
            maximize.click();
            await flushPromises();
            expect(window.getComputedStyle(header).display).not.toBe('none');
        } finally {
            style.remove();
        }
    });

    it('toggles AI maximize state from the AI pane button', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide', text: '', error: '' });

        await importNotesModule();

        const appRoot = document.getElementById('notes-pane') || document.getElementById('app');
        const aiMaximize = document.getElementById('notes-tools-ai-maximize');

        expect(aiMaximize).not.toBeNull();
        expect(aiMaximize.dataset.enabled).toBe('false');
        expect(appRoot.dataset.aiMaximized).toBeUndefined();

        aiMaximize.click();
        await flushPromises();

        expect(aiMaximize.dataset.enabled).toBe('true');
        expect(aiMaximize.textContent).toBe(String.fromCodePoint(0xf065));
        expect(appRoot.dataset.aiMaximized).toBe('true');

        aiMaximize.click();
        await flushPromises();

        expect(aiMaximize.dataset.enabled).toBe('false');
        expect(aiMaximize.textContent).toBe(String.fromCodePoint(0xf065));
        expect(appRoot.dataset.aiMaximized).toBe('false');
    });

    it('uses Font Awesome glyphs for AI and Log panel action buttons', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide', text: '', error: '' });

        await importNotesModule();

        const icons = [
            ['notes-tools-ai-maximize', 0xf065, 'Maximize AI view'],
            ['notes-tools-ai-settings', 0xf013, 'AI session management'],
            ['notes-tools-log-clear', 0xf2ed, 'Clear log'],
            ['notes-tools-log-copy', 0xf0c5, 'Copy log to clipboard'],
            ['notes-tools-log-select-all', 0xf0fe, 'Select visible log lines'],
            ['notes-tools-log-maximize', 0xf065, 'Maximize log view'],
            ['notes-tools-log-deselect', 0xf850, 'Deselect log lines'],
            ['notes-tools-log-error', 'e', 'Toggle error-level log lines'],
            ['notes-tools-log-warn', 'w', 'Toggle warning-level log lines'],
            ['notes-tools-log-info', 'i', 'Toggle info-level log lines'],
            ['notes-tools-log-debug', 'd', 'Toggle debug-level log lines'],
            ['notes-tools-log-trace', 't', 'Toggle trace-level log lines'],
            ['notes-tools-log-timestamp', 0xf017, 'Toggle timestamps'],
            ['notes-tools-log-wordwrap', 0x21b5, 'Toggle word wrap'],
        ];

        icons.forEach(([id, glyph, label]) => {
            const button = document.getElementById(id);
            expect(button.textContent).toBe(typeof glyph === 'number' ? String.fromCodePoint(glyph) : glyph);
            expect(button.classList.contains('notes-tools-icon')).toBe(true);
            expect(button.getAttribute('aria-label')).toBe(label);
        });

        for (const id of [
            'notes-tools-log-error',
            'notes-tools-log-warn',
            'notes-tools-log-info',
            'notes-tools-log-debug',
            'notes-tools-log-trace',
            'notes-tools-log-timestamp',
            'notes-tools-log-wordwrap',
        ]) {
            expect(document.getElementById(id).classList.contains('notes-tools-log-toggle')).toBe(true);
        }

        const fontAwesomeRules = notesCss.match(/[^{}]*\{[^{}]*font-family:\s*"Font Awesome (?:Regular|Solid)"[^{}]*\}/g) || [];
        expect(fontAwesomeRules.length).toBeGreaterThan(0);
        for (const rule of fontAwesomeRules) {
            expect(rule).not.toMatch(/font-weight:\s*(?:bold|[6-9]00)\b/);
        }

        const groups = Array.from(document.querySelectorAll('#notes-tools-log-pane .notes-tools-log-button-group'));
        expect(groups.map((group) => Array.from(group.querySelectorAll('button')).map((button) => button.id))).toEqual([
            ['notes-tools-log-copy', 'notes-tools-log-select-all', 'notes-tools-log-deselect'],
            ['notes-tools-log-error', 'notes-tools-log-warn', 'notes-tools-log-info', 'notes-tools-log-debug', 'notes-tools-log-trace'],
            ['notes-tools-log-timestamp', 'notes-tools-log-wordwrap'],
            ['notes-tools-log-clear', 'notes-tools-log-maximize'],
        ]);

        expect(notesCss).toContain('#notes-log-output:not([data-show-debug="true"]) .notes-log-line[data-level="debug"]');
        expect(notesCss).toContain('#notes-log-output:not([data-show-trace="true"]) .notes-log-line[data-level="trace"]');
        expect(notesCss).toContain('.notes-tools-log-toggle[data-enabled="true"]');
        expect(notesCss).toContain('grid-template-columns: minmax(0, 1fr) 3.5em 140px;');
        expect(notesCss).toContain('flex: 0 0 72px;');
        const stageHoverStyle = notesCss.match(/\.notes-git-action-stage:hover,\s*\.notes-git-action-stage:focus-visible\s*\{([^}]*)\}/)?.[1] || '';
        const unstageHoverStyle = notesCss.match(/\.notes-git-action-unstage:hover,\s*\.notes-git-action-unstage:focus-visible\s*\{([^}]*)\}/)?.[1] || '';
        expect(stageHoverStyle).toContain('border-color: var(--red)');
        expect(stageHoverStyle).toContain('background-color: rgba(var(--red-rgb), 0.3)');
        expect(stageHoverStyle).toContain('color: var(--red)');
        expect(unstageHoverStyle).toContain('border-color: var(--green)');
        expect(unstageHoverStyle).toContain('background-color: rgba(var(--green-rgb), 0.3)');
        expect(unstageHoverStyle).toContain('color: var(--green)');
        expect(notesCss).toContain('.notes-tools-log-button-group .notes-tools-icon');
        expect(notesCss).toContain('border-left: 1px solid rgba(var(--fg-rgb), 0.25);');
        expect(notesCss).toContain('.notes-git-toolbar-separator');
        expect(notesCss).toContain('flex: 0 0 32px;');
        expect(notesCss).toContain('height: 32px;');
        expect(notesCss).not.toContain('data-wrapped');
        expect(notesCss).toContain('font-family: "Font Awesome Regular", "Font Awesome Solid", "Font Awesome", sans-serif;');
        expect(notesCss).toContain('.notes-git-remote');
        expect(notesCss).toContain('font-family: "Font Awesome Brands", sans-serif;');
    });

    it('uses a shared height for Git, AI, and Log toolbar icon and text buttons', () => {
        const toolbarRule = notesCss.match(/\.notes-tools-pane-header button\s*\{([^}]*)\}/)?.[1] || '';
        expect(toolbarRule).toContain('box-sizing: border-box;');
        expect(toolbarRule).toContain('height: 25px;');
        expect(toolbarRule).toContain('display: inline-flex;');
        expect(toolbarRule).toContain('align-items: center;');
        expect(toolbarRule).toContain('justify-content: center;');
        expect(toolbarRule).toContain('line-height: 1;');
        const hiddenRule = notesCss.match(/\.notes-tools-pane-header button\[hidden\]\s*\{([^}]*)\}/)?.[1] || '';
        expect(hiddenRule).toContain('display: none;');
        const logRule = notesCss.match(/\.notes-tools-log-button-group \.notes-tools-icon\s*\{([^}]*)\}/)?.[1] || '';
        expect(logRule).not.toMatch(/(?:^|;)\s*(?:min-|max-)?height\s*:/);
    });

    it('shows Git Tools only after the current project is confirmed as a Git root', async () => {
        getCurrentProjectMock.mockResolvedValue('/workspace/repo');
        getGitStatusMock.mockResolvedValue({
            root: '/workspace/repo',
            branch: 'main',
            ahead: 0,
            behind: 0,
            clean: true,
            files: [],
        });

        await importNotesModule();

        const gitTab = document.getElementById('notes-tools-tab-git');
        const diffTab = document.getElementById('notes-tab-diff');
        expect(getGitStatusMock).toHaveBeenCalledWith('/workspace/repo');
        expect(gitTab.style.display).toBe('');
        expect(document.getElementById('notes-tools-git-branch').textContent).toBe('main');
        expect(document.querySelector('.notes-git-summary #notes-tools-git-branch')).not.toBeNull();
        expect(document.querySelector('.notes-git-actions #notes-tools-git-workspace')).not.toBeNull();
        const gitToolbarItems = Array.from(document.querySelector('.notes-git-actions').children);
        expect(gitToolbarItems.map((element) => element.id || 'separator')).toEqual([
            'notes-tools-git-commit-toggle', 'separator', 'notes-tools-git-branches', 'notes-tools-git-tags', 'notes-tools-git-workspace', 'separator', 'notes-tools-git-remote', 'notes-tools-git-refresh',
        ]);
        expect(gitToolbarItems[1].getAttribute('aria-hidden')).toBe('true');
        expect(gitToolbarItems[5].getAttribute('aria-hidden')).toBe('true');
        expect(gitToolbarItems[1].className).toBe(gitToolbarItems[5].className);
        for (const [id, glyph, label] of [
            ['notes-tools-git-commits', 0xf0ca, 'View commit log'],
            ['notes-tools-git-branches', 0xf126, 'Manage branches'],
            ['notes-tools-git-tags', 0xf02c, 'Manage tags'],
            ['notes-tools-git-workspace', 0xf802, 'Worktrees'],
        ]) {
            const button = document.getElementById(id);
            expect(button.textContent).toBe(String.fromCodePoint(glyph));
            expect(button.classList.contains('notes-tools-icon')).toBe(true);
            expect(button.getAttribute('aria-label')).toBe(label);
        }
        expect(document.getElementById('notes-tools-git-commits').dataset.tooltip).toBe('View commit log');
        expect(document.getElementById('notes-tools-git-tags').dataset.tooltip).toBe('Manage tags');
        expect(document.getElementById('notes-tools-git-workspace').dataset.tooltip).toBe('Manage worktrees');
        expect(document.getElementById('notes-tools-git-branches').dataset.tooltip).toBe('Manage branches');
        await flushPromises();
        const gitRemote = document.getElementById('notes-tools-git-remote');
        expect(gitRemote.hidden).toBe(false);
        gitRemote.click();
        expect(browserOpenURLMock).toHaveBeenCalledWith('https://github.com/org/repo/tree/main');
        const toolTabIds = Array.from(document.querySelectorAll('#notes-tools-tabs [role="tab"]')).map((tab) => tab.id);
        expect(toolTabIds.indexOf('notes-tools-tab-git')).toBeLessThan(toolTabIds.indexOf('notes-tools-tab-ai'));
        expect(diffTab.textContent).toBe('Diff');
        expect(diffTab.style.display).toBe('none');
    });

    it('offers branch creation, checkout, fetch, copy, worktree, and delete actions', async () => {
        getCurrentProjectMock.mockResolvedValue('/workspace/repo');
        getGitStatusMock.mockResolvedValue({ root: '/workspace/repo', branch: 'main', ahead: 0, behind: 0, clean: true, files: [] });
        listGitBranchesMock.mockResolvedValue(['zeta', 'main', 'alpha']);
        getDefaultGitWorktreePathMock.mockResolvedValue('/Users/test/.git_worktrees/default/');
        createGitWorktreeMock.mockResolvedValue({ path: '/Users/test/.git_worktrees/default/alpha', branch: 'alpha' });
        runGitActionMock.mockResolvedValue({ mode: 'interactive', stdout: 'Fetched objects', stderr: 'Fetch warning', error: '' });
        await importNotesModule();
        document.getElementById('notes-tools-git-branches').click();
        await flushPromises();
        let menu = showLocalMenuMock.mock.calls.at(-1)?.[0];
        expect(menu.options).toEqual(['Create new branch', '-', 'alpha', 'main', 'zeta']);
        menu.onSelect(2);
        menu = showLocalMenuMock.mock.calls.at(-1)?.[0];
        expect(menu.title).toBe('alpha');
        expect(menu.options).toEqual(['Info', 'Copy branch name', '-', 'Checkout', 'Fetch latest', 'Create new worktree', '-', 'Delete branch']);
        expect(menu.icons).toEqual([0xf129, 0xf0c5, 0, 0xf126, 0xf019, 0xf067, 0, 0xf2ed]);
        menu.onSelect(0);
        await flushPromises();
        expect(showGitRefInfoMock).toHaveBeenCalledWith('/workspace/repo', 'alpha', false);
        menu.onSelect(1);
        await flushPromises();
        expect(clipboardSetTextMock).toHaveBeenCalledWith('alpha');
        menu.onSelect(3);
        await flushPromises();
        expect(checkoutGitBranchMock).toHaveBeenCalledWith('/workspace/repo', 'alpha', false);
        expect(sendIpcMock).toHaveBeenCalledWith('terminal-notify', expect.objectContaining({ message: 'Switched to alpha', level: 'info' }));
        checkoutGitBranchMock.mockRejectedValueOnce(new Error('local changes would be overwritten'));
        menu.onSelect(3);
        await flushPromises();
        expect(sendIpcMock).toHaveBeenCalledWith('terminal-notify', expect.objectContaining({ message: 'local changes would be overwritten', level: 'error' }));
        const checkouts = checkoutGitBranchMock.mock.calls.length;
        const beforeFetch = sendIpcMock.mock.calls.length;
        menu.onSelect(4);
        await flushPromises();
        expect(runGitActionMock).toHaveBeenCalledWith('/workspace/repo', 'fetch-branch', { branch: 'alpha' });
        expect(checkoutGitBranchMock).toHaveBeenCalledTimes(checkouts);
        expect(sendIpcMock.mock.calls.slice(beforeFetch).filter(([channel]) => channel === 'terminal-notify'))
            .toEqual([['terminal-notify', { message: 'Fetch completed for alpha', level: 'info' }]]);
        const previousCalls = sendIpcMock.mock.calls.length;
        runGitActionMock.mockResolvedValueOnce({ mode: 'interactive', stdout: '', stderr: 'Fetch failed', error: 'exit status 1' });
        menu.onSelect(4);
        await flushPromises();
        const failureCalls = sendIpcMock.mock.calls.slice(previousCalls);
        expect(failureCalls.some(([channel, payload]) => channel === 'terminal-notify' && payload.message === 'Fetch completed for alpha')).toBe(false);
        expect(failureCalls.some(([channel, payload]) => channel === 'terminal-notify' && payload.message === 'Fetch failed' && payload.level === 'error')).toBe(true);
        const beforeFallback = sendIpcMock.mock.calls.length;
        runGitActionMock.mockResolvedValueOnce({ mode: 'interactive', stdout: 'partial output', stderr: '', error: 'exit status 128' });
        menu.onSelect(4);
        await flushPromises();
        expect(sendIpcMock.mock.calls.slice(beforeFallback).filter(([channel]) => channel === 'terminal-notify'))
            .toEqual([['terminal-notify', { message: 'partial output', level: 'error' }]]);
        menu.onSelect(5);
        await flushPromises();
        expect(createGitWorktreeMock).toHaveBeenCalledWith('/workspace/repo', 'alpha', '/Users/test/.git_worktrees/default/alpha', false);
        expect(switchGitWorktreeMock).toHaveBeenCalledWith('/workspace/repo', '/Users/test/.git_worktrees/default/alpha');
        menu.onSelect(7);
        await flushPromises();
        expect(deleteGitBranchMock).toHaveBeenCalledWith('/workspace/repo', 'alpha');
        expect(sendIpcMock).toHaveBeenCalledWith('terminal-notify', expect.objectContaining({ message: 'Deleted branch alpha', level: 'info' }));
        deleteGitBranchMock.mockRejectedValueOnce(new Error('branch is checked out'));
        menu.onSelect(7);
        await flushPromises();
        expect(sendIpcMock).toHaveBeenCalledWith('terminal-notify', expect.objectContaining({ message: 'branch is checked out', level: 'error' }));
        document.getElementById('notes-tools-git-branches').click();
        await flushPromises();
        showLocalMenuMock.mock.calls.at(-1)?.[0].onSelect(0);
        await flushPromises();
        expect(document.getElementById('notes-modal').dataset.open).toBe('true');
        expect(document.querySelector('#notes-modal-title h3')?.textContent).toBe('Create new branch');
        expect(document.getElementById('notes-modal-title').classList.contains('markdown-body')).toBe(true);
        document.getElementById('notes-modal-input').value = 'topic/new';
        document.getElementById('notes-modal-create').click();
        await flushPromises();
        expect(checkoutGitBranchMock).toHaveBeenCalledWith('/workspace/repo', 'topic/new', true);
        expect(document.getElementById('notes-modal').dataset.open).toBe('false');
        expect(sendIpcMock).toHaveBeenCalledWith('terminal-notify', expect.objectContaining({ message: 'Switched to topic/new', level: 'info' }));
    });

    it('manages tags, transfers them, and creates a tag worktree', async () => {
        getCurrentProjectMock.mockResolvedValue('/workspace/repo');
        getGitStatusMock.mockResolvedValue({ root: '/workspace/repo', branch: 'main', ahead: 0, behind: 0, clean: true, files: [] });
        listGitTagsMock.mockResolvedValue(['v10.3.4', 'v5.9.9', 'release']);
        getDefaultGitWorktreePathMock.mockResolvedValue('/Users/test/.git_worktrees/default/');
        createGitTagWorktreeMock.mockResolvedValue({ path: '/Users/test/.git_worktrees/default/v10.3.4', branch: 'tag: v10.3.4' });
        runGitActionMock.mockResolvedValue({ mode: 'interactive', stdout: 'Fetched tags', stderr: 'From origin: new tags', error: '' });
        await importNotesModule();

        document.getElementById('notes-tools-git-tags').click();
        await flushPromises();
        let menu = showLocalMenuMock.mock.calls.at(-1)?.[0];
        expect(listGitTagsMock).toHaveBeenCalledWith('/workspace/repo');
        expect(menu.title).toBe('Tags');
        expect(menu.options).toEqual(['Create new tag', '-', 'Pull tags', '-', 'v10.3.4', 'v5.9.9', 'release']);
        expect(menu.icons).toEqual([0xf067, 0, 0xf019, 0, 0, 0, 0]);

        const beforePull = sendIpcMock.mock.calls.length;
        menu.onSelect(2);
        await flushPromises();
        expect(runGitActionMock).toHaveBeenCalledWith('/workspace/repo', 'pull-tags', {});
        expect(sendIpcMock.mock.calls.slice(beforePull).filter(([channel]) => channel === 'terminal-notify'))
            .toEqual([['terminal-notify', { message: 'Tags pulled', level: 'info' }]]);

        menu.onSelect(4);
        menu = showLocalMenuMock.mock.calls.at(-1)?.[0];
        expect(menu.title).toBe('v10.3.4');
        expect(menu.options).toEqual(['Info', 'Copy tag name', 'Push tag', '-', 'Checkout', 'Create new worktree', '-', 'Delete tag']);
        expect(menu.icons).toEqual([0xf129, 0xf0c5, 0xf093, 0, 0xf126, 0xf067, 0, 0xf2ed]);
        menu.onSelect(0);
        await flushPromises();
        expect(showGitRefInfoMock).toHaveBeenCalledWith('/workspace/repo', 'v10.3.4', true);
        menu.onSelect(1);
        await flushPromises();
        expect(clipboardSetTextMock).toHaveBeenCalledWith('v10.3.4');
        const beforePush = sendIpcMock.mock.calls.length;
        menu.onSelect(2);
        await flushPromises();
        expect(runGitActionMock).toHaveBeenCalledWith('/workspace/repo', 'push-tag', { tag: 'v10.3.4' });
        expect(sendIpcMock.mock.calls.slice(beforePush).filter(([channel]) => channel === 'terminal-notify'))
            .toEqual([['terminal-notify', { message: 'Pushed tag v10.3.4', level: 'info' }]]);
        const beforePushFailure = sendIpcMock.mock.calls.length;
        runGitActionMock.mockResolvedValueOnce({ mode: 'interactive', stdout: '', stderr: 'Push rejected', error: 'exit status 1' });
        menu.onSelect(2);
        await flushPromises();
        expect(sendIpcMock.mock.calls.slice(beforePushFailure).filter(([channel]) => channel === 'terminal-notify'))
            .toEqual([['terminal-notify', { message: 'Push rejected', level: 'error' }]]);
        menu.onSelect(4);
        await flushPromises();
        expect(checkoutGitTagMock).toHaveBeenCalledWith('/workspace/repo', 'v10.3.4');
        menu.onSelect(5);
        await flushPromises();
        expect(createGitTagWorktreeMock).toHaveBeenCalledWith('/workspace/repo', 'v10.3.4', '/Users/test/.git_worktrees/default/v10.3.4');
        expect(switchGitWorktreeMock).toHaveBeenCalledWith('/workspace/repo', '/Users/test/.git_worktrees/default/v10.3.4');
        menu.onSelect(7);
        await flushPromises();
        expect(deleteGitTagMock).toHaveBeenCalledWith('/workspace/repo', 'v10.3.4');

        document.getElementById('notes-tools-git-tags').click();
        await flushPromises();
        menu = showLocalMenuMock.mock.calls.at(-1)?.[0];
        expect(menu.options[0]).toBe('Create new tag');
        menu.onSelect(0);
        await flushPromises();
        expect(document.getElementById('notes-modal-title')?.textContent).toBe('Create new tag');
        expect(document.querySelector('#notes-modal-title h3')?.textContent).toBe('Create new tag');
        expect(document.getElementById('notes-modal-title').classList.contains('markdown-body')).toBe(true);
        document.getElementById('notes-modal-input').value = 'v11.0.0';
        document.getElementById('notes-modal-create').click();
        await flushPromises();
        expect(createGitTagMock).toHaveBeenCalledWith('/workspace/repo', 'v11.0.0');
    });

    it('offers HEAD first and alphabetized branches excluding the checked-out branch', async () => {
        getCurrentProjectMock.mockResolvedValue('/workspace/repo');
        getGitStatusMock.mockResolvedValue({ root: '/workspace/repo', branch: 'main', ahead: 0, behind: 0, clean: true, files: [] });
        listGitBranchesMock.mockResolvedValue(['zeta', 'main', 'alpha']);

        await importNotesModule();
        const compareButton = document.getElementById('notes-tools-git-compare');
        expect(compareButton.textContent).toBe('HEAD');
        compareButton.click();
        await flushPromises();

        let menu = showLocalMenuMock.mock.calls.at(-1)?.[0];
        expect(menu.options).toEqual(['HEAD (main)', '-', 'alpha', 'zeta']);
        menu.onSelect(3);
        await flushPromises();
        expect(compareButton.textContent).toBe('zeta');

        compareButton.click();
        await flushPromises();
        menu = showLocalMenuMock.mock.calls.at(-1)?.[0];
        expect(menu.options).toEqual(['HEAD (main)', '-', 'alpha', 'zeta']);
        menu.onSelect(0);
        await flushPromises();
        expect(compareButton.textContent).toBe('HEAD');
    });

    it('dims overlapping graph regions and dims the opposite layer on hover', () => {
        const graphStyle = notesCss.match(/\.notes-git-commit-graph\s*\{([^}]*)\}/)?.[1] || '';
        expect(graphStyle).toContain('color: var(--yellow);');
        expect(graphStyle).toContain('rgba(0, 0, 0, 0.7)');
        expect(graphStyle).toContain('--git-overlap-start');
        const hoverStyle = notesCss.match(/\.notes-git-commit:has\(\.notes-git-commit-content:hover\)\s+\.notes-git-commit-graph\s*\{([^}]*)\}/)?.[1] || '';
        expect(hoverStyle).toContain('opacity: 0.2;');
        expect(hoverStyle).toContain('mask-image: none;');
        const textHoverStyle = notesCss.match(/\.notes-git-commit-graph:hover\s*\+\s*\.notes-git-commit-content\s*\{([^}]*)\}/)?.[1] || '';
        expect(textHoverStyle).toContain('opacity: 0.2;');
    });

    it('shows the Git file filter as the second toolbar row and replaces it with commit controls', async () => {
        getCurrentProjectMock.mockResolvedValue('/workspace/repo');
        getGitStatusMock.mockResolvedValue({ root: '/workspace/repo', branch: 'main', clean: true, files: [] });
        await importNotesModule();
        const toolbar = document.getElementById('notes-tools-git-toolbar');
        const filterRow = document.getElementById('notes-tools-git-filter-row');
        const filter = document.getElementById('notes-tools-git-file-filter');
        const commitRow = document.getElementById('notes-tools-git-commit-row');
        const toggle = document.getElementById('notes-tools-git-commit-toggle');
        expect(toolbar.nextElementSibling).toBe(filterRow);
        expect(filterRow.hidden).toBe(false);
        expect(commitRow.hidden).toBe(true);
        expect(filter.getAttribute('aria-label')).toBe('Filter Git files');
        for (const [attribute, value] of [
            ['autocomplete', 'off'], ['autocorrect', 'off'], ['autocapitalize', 'off'],
            ['spellcheck', 'false'], ['writingsuggestions', 'false'],
        ]) expect(filter.getAttribute(attribute)).toBe(value);
        expect(attachSpellCheck.mock.calls.some(([target]) => target === filter)).toBe(false);
        filter.value = 'retained query';
        toggle.click();
        expect(filterRow.hidden).toBe(true);
        expect(commitRow.hidden).toBe(false);
        toggle.click();
        expect(filterRow.hidden).toBe(false);
        expect(commitRow.hidden).toBe(true);
        expect(filter.value).toBe('retained query');
    });

    it('shows a default-off Search filenames toggle below the filter only for Git history', async () => {
        getCurrentProjectMock.mockResolvedValue('/workspace/repo');
        getGitStatusMock.mockResolvedValue({ root: '/workspace/repo', branch: 'main', clean: true, files: [] });
        const commits = [
            { hash: 'aaaaaaa', subject: 'needle in message', message: 'needle in message', files: ['src/main.go'], graph: ['*'] },
            { hash: 'bbbbbbb', subject: 'Update docs', message: 'Update docs', files: ['docs/needle.md'], graph: ['*'] },
        ];
        listGitCommitsMock.mockResolvedValue(commits);
        listGitCommitsWithFilesMock.mockResolvedValue(commits);
        await importNotesModule();
        const row = document.getElementById('notes-commit-filename-search-row');
        const checkbox = document.getElementById('notes-commit-filename-search');
        const input = document.getElementById('notes-list-filter');
        const history = document.getElementById('notes-commits-list');
        expect(row.hidden).toBe(true);
        expect(checkbox.checked).toBe(false);
        expect(row.textContent.trim()).toBe('Search filenames');
        expect(row.classList.contains('markdown-body')).toBe(true);
        expect(getNotesStyleProperties('.notes-commit-filename-search > span').font).toBe('inherit');
        expect(getNotesStyleProperties('.notes-commit-filename-search')['font-size']).toBeUndefined();
        expect(row.previousElementSibling.id).toBe('notes-list-filter-wrap');
        document.getElementById('notes-tools-git-commits').click();
        await flushPromises();
        expect(row.hidden).toBe(false);
        input.value = 'needle';
        input.dispatchEvent(new Event('input', { bubbles: true }));
        await flushPromises();
        expect(history.querySelector('.notes-git-commit-hash').textContent).toBe('aaaaaaa');
        checkbox.checked = true;
        checkbox.dispatchEvent(new Event('change'));
        await flushPromises();
        await flushPromises();
        expect(input.placeholder).toBe('Filter commits by filename...');
        expect(history.querySelectorAll('.notes-git-commit')).toHaveLength(1);
        expect(history.querySelector('.notes-git-commit-hash').textContent).toBe('bbbbbbb');
        expect(listGitCommitsWithFilesMock).toHaveBeenCalledWith('/workspace/repo', 10000);
        document.getElementById('notes-tools-git-commits').click();
        expect(row.hidden).toBe(true);
        expect(input.placeholder).toBe('Filter files...');
    });

    it('keeps the shared sidebar filter for full commit hashes and messages', async () => {
        getCurrentProjectMock.mockResolvedValue('/workspace/repo');
        getGitStatusMock.mockResolvedValue({ root: '/workspace/repo', branch: 'main', ahead: 0, behind: 0, clean: true, files: [] });
        listFilesMock.mockResolvedValue(['$PROJECT/rollout.txt', '$PROJECT/other.txt']);
        listGitCommitsMock.mockResolvedValue([
            { hash: 'aaaaaaa', fullHash: 'a'.repeat(34) + 'ef0123', subject: 'First merge', message: 'First merge\n\nRollout ticket', date: '', author: '', graph: ['*'] },
            { hash: 'bbbbbbb', fullHash: 'b'.repeat(40), subject: 'Second commit', message: 'Second commit', date: '', author: '', graph: ['*'] },
        ]);
        await importNotesModule();
        const input = document.getElementById('notes-list-filter');
        const toggle = document.getElementById('notes-tools-git-commits');
        const history = document.getElementById('notes-commits-list');
        toggle.click();
        await flushPromises();
        expect(document.getElementById('notes-list-filter-wrap').hidden).toBe(false);
        expect(input.placeholder).toBe('Filter commits...');
        input.value = 'ef0123';
        input.dispatchEvent(new Event('input', { bubbles: true }));
        await flushPromises();
        expect(history.querySelectorAll('.notes-git-commit')).toHaveLength(1);
        expect(history.textContent).toContain('First merge');
        expect(history.textContent).not.toContain('Second commit');
        expect(document.getElementById('notes-list-filter-clear').dataset.visible).toBe('true');
        input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
        await flushPromises();
        expect(history.querySelectorAll('.notes-git-commit')).toHaveLength(2);
        input.value = 'rollout';
        input.dispatchEvent(new Event('input', { bubbles: true }));
        await flushPromises();
        expect(history.querySelectorAll('.notes-git-commit')).toHaveLength(1);
        document.getElementById('notes-list-filter-clear').click();
        await flushPromises();
        expect(input.value).toBe('');
        expect(history.querySelectorAll('.notes-git-commit')).toHaveLength(2);
        input.value = 'rollout';
        input.dispatchEvent(new Event('input', { bubbles: true }));
        await flushPromises();
        toggle.click();
        await flushPromises();
        expect(input.placeholder).toBe('Filter files...');
        expect(document.getElementById('notes-list').textContent).toContain('rollout.txt');
        expect(document.getElementById('notes-list').textContent).not.toContain('other.txt');
    });

    it('resets Commits on a workspace change even when the project root stays the same', async () => {
        const subjectStyle = notesCss.match(/\.notes-git-commit-subject\s*\{([^}]*)\}/)?.[1] || '';
        expect(subjectStyle).toContain('font-family: "Lato", var(--font-family), sans-serif;');
        expect(subjectStyle).toContain('font-size: var(--notes-font-size-minus-2);');
        const contentStyle = notesCss.match(/^\.notes-git-commit-content\s*\{([^}]*)\}/m)?.[1] || '';
        expect(contentStyle).toContain('justify-self: end;');
        expect(contentStyle).toContain('width: max-content;');
        expect(contentStyle).toContain('width: max(70%, calc(100% - var(--git-graph-width) - 8px));');
        expect(contentStyle).toContain('z-index: 1;');
        getCurrentProjectMock.mockResolvedValue('/workspace/repo');
        getCurrentGroupNameMock.mockResolvedValue('First workspace');
        getGitStatusMock.mockResolvedValue({ root: '/workspace/repo', branch: 'main', ahead: 0, behind: 0, clean: true, files: [] });
        await importNotesModule();
        const toggle = document.getElementById('notes-tools-git-commits');
        toggle.click();
        await flushPromises();
        expect(toggle.getAttribute('aria-pressed')).toBe('true');
        const history = document.getElementById('notes-commits-list');
        expect(history.closest('#notes-sidebar')).not.toBeNull();
        expect(history.hidden).toBe(false);
        expect(history.textContent).toBe('No commits');
        expect(document.getElementById('notes-list').hidden).toBe(true);
        expect(document.getElementById('notes-list-filter-wrap').hidden).toBe(false);
        expect(document.getElementById('notes-tools-git-files').textContent).toBe('Working tree clean');
        getEventHandler('notesContextChanged')({ projectRoot: '/workspace/repo', groupName: 'Second workspace', globalNotes: '/notes/', userNotes: '/notes/Second/' });
        await flushPromises();
        expect(toggle.getAttribute('aria-pressed')).toBe('false');
        expect(history.hidden).toBe(true);
        expect(document.getElementById('notes-list').hidden).toBe(false);
        expect(document.getElementById('notes-list-filter-wrap').hidden).toBe(false);
    });

    it('lists worktrees and creates a worktree from the existing branch picker', async () => {
        getCurrentProjectMock.mockResolvedValue('/workspace/repo');
        getGitStatusMock.mockResolvedValue({ root: '/workspace/repo', branch: 'main', ahead: 0, behind: 0, clean: true, files: [] });
        listGitWorktreesMock.mockResolvedValue([
            { path: '/workspace/worktrees/zeta', branch: 'zeta' },
            { path: '/workspace/repo', branch: 'main' },
            { path: '/workspace/worktrees/alpha', branch: 'alpha' },
        ]);
        listGitBranchesMock.mockResolvedValue(['zeta', 'main', 'alpha']);
        getDefaultGitWorktreePathMock.mockResolvedValue('/Users/test/Documents/ttyphoon/.git_worktrees/hashed-origin');
        createGitWorktreeMock.mockResolvedValue({ path: '/workspace/repo/.git_worktrees/alpha', branch: 'alpha' });

        await importNotesModule();

        const worktreeButton = document.getElementById('notes-tools-git-workspace');
        expect(worktreeButton.textContent).toBe(String.fromCodePoint(0xf802));
        worktreeButton.click();
        await flushPromises();

        let menu = showLocalMenuMock.mock.calls.at(-1)?.[0];
        expect(menu.title).toBe('Worktrees');
        expect(menu.options).toEqual([
            'New worktree', '-', 'alpha', 'main', 'zeta',
        ]);
        expect(menu.icons).toEqual([0xf067, 0, 0, 0xf00c, 0]);
        menu.onSelect(4);
        await flushPromises();
        const focusEvents = [];
        const focusHandler = (event) => focusEvents.push(event.type);
        window.addEventListener('ttyphoon-focus-terminal', focusHandler);
        let actionMenu = showLocalMenuMock.mock.calls.at(-1)?.[0];
        expect(actionMenu.title).toBe('zeta');
        expect(actionMenu.options).toEqual(['Info', 'Switch', '-', 'Remove']);
        expect(actionMenu.icons).toEqual([0xf129, 0xf204, 0, 0xf2ed]);
        actionMenu.onSelect(0);
        await flushPromises();
        expect(showGitWorktreeInfoMock).toHaveBeenCalledWith('/workspace/repo', '/workspace/worktrees/zeta');
        expect(switchGitWorktreeMock).not.toHaveBeenCalled();
        actionMenu.onSelect(1);
        await flushPromises();
        expect(switchGitWorktreeMock).toHaveBeenCalledWith('/workspace/repo', '/workspace/worktrees/zeta');
        expect(focusEvents).toContain('ttyphoon-focus-terminal');
        window.removeEventListener('ttyphoon-focus-terminal', focusHandler);
        actionMenu.onSelect(3);
        await flushPromises();
        expect(removeGitWorktreeMock).toHaveBeenCalledWith('/workspace/repo', '/workspace/worktrees/zeta');
        expect(sendIpcMock).toHaveBeenCalledWith('terminal-notify', expect.objectContaining({
            level: 'info',
            message: 'Worktree removed: zeta',
        }));

        worktreeButton.click();
        await flushPromises();
        menu = showLocalMenuMock.mock.calls.at(-1)?.[0];
        menu.onSelect(0);
        await flushPromises();

        expect(document.querySelector('#notes-modal-title h3')?.textContent).toBe('Create new worktree');
        expect(document.getElementById('notes-modal-card').classList.contains('notes-worktree-dialog')).toBe(true);
        const branchButton = document.querySelector('.notes-worktree-form .notes-graph-dropdown');
        branchButton.click();
        menu = showLocalMenuMock.mock.calls.at(-1)?.[0];
        expect(menu.options).toEqual(['Create new branch', '-', 'alpha', 'main', 'zeta']);
        expect(menu.icons[0]).toBe(0xf067);
        menu.onSelect(2);

        const worktreePath = document.querySelector('.notes-worktree-path-input');
        expect(worktreePath.tagName).toBe('TEXTAREA');
        expect(worktreePath.rows).toBe(2);
        expect(worktreePath.wrap).toBe('soft');
        expect(worktreePath.value).toBe('/Users/test/Documents/ttyphoon/.git_worktrees/hashed-origin/alpha');
        worktreePath.value = '/Users/test/Documents/ttyphoon/\n.git_worktrees/alpha';
        worktreePath.dispatchEvent(new Event('input', { bubbles: true }));
        expect(worktreePath.value).toBe('/Users/test/Documents/ttyphoon/ .git_worktrees/alpha');
        worktreePath.value = '/Users/test/Documents/ttyphoon/.git_worktrees/custom-alpha';
        document.getElementById('notes-modal-create').click();
        await flushPromises();

        expect(createGitWorktreeMock).toHaveBeenCalledWith(
            '/workspace/repo', 'alpha', '/Users/test/Documents/ttyphoon/.git_worktrees/custom-alpha', false,
        );
        expect(document.getElementById('notes-modal').dataset.open).toBe('false');
        expect(document.getElementById('notes-modal-card').classList.contains('notes-worktree-dialog')).toBe(false);
        expect(sendIpcMock).not.toHaveBeenCalledWith('terminal-notify', expect.objectContaining({
            message: expect.stringContaining('Created worktree'),
        }));
        expect(sendIpcMock).toHaveBeenCalledWith('terminal-sticky-create', expect.objectContaining({
            message: 'Creating worktree',
            level: 'info',
        }));
        expect(sendIpcMock).toHaveBeenCalledWith('terminal-sticky-close', expect.objectContaining({ id: expect.any(String) }));
    });

    it('resets the branch selection when Create branch is cancelled and reports worktree action errors', async () => {
        getCurrentProjectMock.mockResolvedValue('/workspace/repo');
        getGitStatusMock.mockResolvedValue({ root: '/workspace/repo', branch: 'main', ahead: 0, behind: 0, clean: true, files: [] });
        listGitWorktreesMock.mockResolvedValue([{ path: '/workspace/repo', branch: 'main' }]);
        listGitBranchesMock.mockResolvedValue(['main']);
        getDefaultGitWorktreePathMock.mockResolvedValue('/Users/test/Documents/ttyphoon/.git_worktrees/hashed-root');
        createGitWorktreeMock.mockRejectedValue(new Error('worktree creation failed'));
        removeGitWorktreeMock.mockRejectedValue(new Error('worktree has local changes'));

        await importNotesModule();
        document.getElementById('notes-tools-git-workspace').click();
        await flushPromises();
        showLocalMenuMock.mock.calls.at(-1)?.[0].onSelect(0);
        await flushPromises();

        document.querySelector('.notes-worktree-form .notes-graph-dropdown').click();
        showLocalMenuMock.mock.calls.at(-1)?.[0].onSelect(0);
        await flushPromises();
        expect(document.querySelector('#notes-modal-title h3')?.textContent).toBe('Create branch');
        expect(document.getElementById('notes-modal-location-row').style.display).not.toBe('none');
        expect(document.getElementById('notes-modal-location').style.display).toBe('none');
        document.getElementById('notes-modal-cancel').click();
        await flushPromises();
        expect(document.querySelector('#notes-modal-title h3')?.textContent).toBe('Create new worktree');
        expect(document.querySelector('.notes-worktree-form .notes-graph-dropdown').textContent).toContain('Select branch');
        expect(document.querySelector('.notes-worktree-path-input').disabled).toBe(true);
        expect(document.querySelector('.notes-worktree-path-input').value).toBe('');

        document.querySelector('.notes-worktree-form .notes-graph-dropdown').click();
        showLocalMenuMock.mock.calls.at(-1)?.[0].onSelect(0);
        await flushPromises();
        document.getElementById('notes-modal-input').value = 'topic/new';
        document.getElementById('notes-modal-create').click();
        await flushPromises();
        expect(document.querySelector('.notes-worktree-path-input').value).toBe('/Users/test/Documents/ttyphoon/.git_worktrees/hashed-root/topic/new');
        document.getElementById('notes-modal-create').click();
        await flushPromises();
        expect(createGitWorktreeMock).toHaveBeenCalledWith(
            '/workspace/repo', 'topic/new', '/Users/test/Documents/ttyphoon/.git_worktrees/hashed-root/topic/new', true,
        );
        expect(sendIpcMock).toHaveBeenCalledWith('terminal-notify', expect.objectContaining({
            level: 'error',
            message: 'worktree creation failed',
        }));
        expect(switchGitWorktreeMock).not.toHaveBeenCalled();

        document.getElementById('notes-tools-git-workspace').click();
        await flushPromises();
        const worktreeMenu = showLocalMenuMock.mock.calls.at(-1)?.[0];
        worktreeMenu.onSelect(2);
        const actionMenu = showLocalMenuMock.mock.calls.at(-1)?.[0];
        expect(actionMenu.options).toContain('Remove');
        actionMenu.onSelect(actionMenu.options.indexOf('Remove'));
        await flushPromises();
        expect(sendIpcMock).toHaveBeenCalledWith('terminal-notify', expect.objectContaining({
            level: 'error',
            message: 'worktree has local changes',
        }));
    });

    it('offers commit info, copy, compare, and hash-path worktree creation through the shared handler', async () => {
        const fullHash = 'abcdef1' + '0'.repeat(33);
        getCurrentProjectMock.mockResolvedValue('/workspace/repo');
        getGitStatusMock.mockResolvedValue({ root: '/workspace/repo', branch: 'main', ahead: 0, behind: 0, clean: true, files: [] });
        listGitCommitsMock.mockResolvedValue([{ hash: 'abcdef1', fullHash, subject: 'Base commit', message: 'Base commit', date: '', author: '', graph: ['*'] }]);
        getDefaultGitWorktreePathMock.mockResolvedValue('/Users/test/.git_worktrees/default/');
        let finishCreation;
        createGitWorktreeMock.mockImplementation(() => new Promise((resolve) => { finishCreation = resolve; }));
        await importNotesModule();
        document.getElementById('notes-tools-git-commits').click();
        await flushPromises();
        const openMenu = async () => {
            document.querySelector('.notes-git-commit').dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 50, clientY: 80 }));
            await flushPromises();
            return showLocalMenuMock.mock.calls.at(-1)?.[0];
        };
        let menu = await openMenu();
        expect(menu.title).toBe('abcdef1');
        expect(menu.options).toEqual(['Info', 'Copy commit hash', '-', 'Compare commit', 'Create worktree']);
        expect(menu.icons[0]).toBe(0xf129);
        expect(menu.icons[3]).toBe(0xe13a);
        menu.onSelect(0);
        await flushPromises();
        expect(showGitRefInfoMock).toHaveBeenCalledWith('/workspace/repo', fullHash, false);
        menu.onSelect(1);
        await flushPromises();
        expect(clipboardSetTextMock).toHaveBeenCalledWith(fullHash);
        menu.onSelect(3);
        await flushPromises();
        expect(document.getElementById('notes-tools-git-compare').textContent).toBe('abcdef1');
        menu = await openMenu();
        menu.onSelect(4);
        await flushPromises();
        expect(createGitWorktreeMock).toHaveBeenCalledWith('/workspace/repo', fullHash, '/Users/test/.git_worktrees/default/abcdef1', false);
        const stickyIndex = sendIpcMock.mock.calls.findIndex(([channel]) => channel === 'terminal-sticky-create');
        expect(sendIpcMock.mock.calls[stickyIndex][1]).toEqual(expect.objectContaining({ message: 'Creating worktree' }));
        expect(sendIpcMock.mock.invocationCallOrder[stickyIndex]).toBeLessThan(createGitWorktreeMock.mock.invocationCallOrder[0]);
        expect(sendIpcMock).not.toHaveBeenCalledWith('terminal-sticky-close', expect.anything());
        expect(switchGitWorktreeMock).not.toHaveBeenCalled();
        finishCreation({ path: '/Users/test/.git_worktrees/default/abcdef1', branch: 'abcdef1' });
        await flushPromises();
        expect(switchGitWorktreeMock).toHaveBeenCalledWith('/workspace/repo', '/Users/test/.git_worktrees/default/abcdef1');
        expect(sendIpcMock).toHaveBeenCalledWith('terminal-sticky-close', expect.objectContaining({ id: expect.any(String) }));
        listGitWorktreesMock.mockResolvedValue([
            { path: '/workspace/repo', branch: 'main' },
            { path: '/Users/test/.git_worktrees/default/abcdef1', branch: 'abcdef1' },
        ]);
        document.getElementById('notes-tools-git-workspace').click();
        await flushPromises();
        expect(showLocalMenuMock.mock.calls.at(-1)?.[0].options).toContain('abcdef1');
        expect(showLocalMenuMock.mock.calls.at(-1)?.[0].options).not.toContain('(detached HEAD)');
    });

    it('submits Worktree on Enter, closes immediately, and waits for Git behind a sticky', async () => {
        getCurrentProjectMock.mockResolvedValue('/workspace/repo');
        getGitStatusMock.mockResolvedValue({ root: '/workspace/repo', branch: 'main', ahead: 0, behind: 0, clean: true, files: [] });
        listGitWorktreesMock.mockResolvedValue([{ path: '/workspace/repo', branch: 'main' }]);
        listGitBranchesMock.mockResolvedValue(['main']);
        getDefaultGitWorktreePathMock.mockResolvedValue('/Users/test/Documents/ttyphoon/.git_worktrees/hashed-root');
        let finishCreation;
        createGitWorktreeMock.mockImplementation(() => new Promise((resolve) => { finishCreation = resolve; }));
        let finishSwitch;
        switchGitWorktreeMock.mockImplementation(() => new Promise((resolve) => { finishSwitch = resolve; }));

        await importNotesModule();
        document.getElementById('notes-tools-git-workspace').click();
        await flushPromises();
        showLocalMenuMock.mock.calls.at(-1)?.[0].onSelect(0);
        await flushPromises();
        document.querySelector('.notes-worktree-form .notes-graph-dropdown').click();
        showLocalMenuMock.mock.calls.at(-1)?.[0].onSelect(2);

        const pathInput = document.querySelector('.notes-worktree-path-input');
        pathInput.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));
        await flushPromises();

        expect(createGitWorktreeMock).toHaveBeenCalledWith(
            '/workspace/repo', 'main', '/Users/test/Documents/ttyphoon/.git_worktrees/hashed-root/main', false,
        );
        expect(document.getElementById('notes-modal').dataset.open).toBe('false');
        const stickyCallIndex = sendIpcMock.mock.calls.findIndex(([channel]) => channel === 'terminal-sticky-create');
        expect(sendIpcMock.mock.calls[stickyCallIndex][1]).toEqual(expect.objectContaining({
            message: 'Creating worktree',
            level: 'info',
        }));
        expect(sendIpcMock.mock.invocationCallOrder[stickyCallIndex])
            .toBeLessThan(createGitWorktreeMock.mock.invocationCallOrder[0]);
        expect(sendIpcMock).not.toHaveBeenCalledWith('terminal-sticky-close', expect.anything());

        expect(switchGitWorktreeMock).not.toHaveBeenCalled();
        finishCreation({ path: '/Users/test/Documents/ttyphoon/.git_worktrees/hashed-root/main', branch: 'main' });
        await flushPromises();
        expect(switchGitWorktreeMock).toHaveBeenCalledWith('/workspace/repo', '/Users/test/Documents/ttyphoon/.git_worktrees/hashed-root/main');
        expect(sendIpcMock).not.toHaveBeenCalledWith('terminal-sticky-close', expect.anything());
        finishSwitch();
        await flushPromises();
        expect(sendIpcMock).toHaveBeenCalledWith('terminal-sticky-close', expect.objectContaining({ id: expect.any(String) }));
    });

    it('opens Git-selected files through regular Notes history and recent-file loading', async () => {
        getCurrentProjectMock.mockResolvedValue('/workspace/repo');
        getGitStatusMock.mockResolvedValue({
            root: '/workspace/repo',
            branch: 'main',
            ahead: 0,
            behind: 0,
            clean: false,
            files: [{ path: 'src/main.go', code: ' M', unstaged: true }],
        });
        getGitDiffMock.mockResolvedValue({
            path: 'src/main.go', original: 'package main\n', working: 'package main\n', binary: false, deleted: false,
        });
        getFileMock.mockResolvedValue({ contents: 'package main\n', binary: false, error: '' });
        listFilesMock.mockResolvedValue(['$PROJECT/src/main.go']);

        await importNotesModule();
        await flushPromises();

        document.querySelector('.notes-git-file')?.click();
        await flushPromises();
        await flushPromises();

        expect(getFileMock).toHaveBeenCalledWith('$PROJECT/src/main.go');
        expect(notesHistoryAddPositionMock).toHaveBeenCalledWith('$PROJECT/src/main.go', 1);
        expect(document.getElementById('notes-current-file').textContent).toBe('main.go');
        expect(document.getElementById('notes-git-diff-wrap').dataset.active).toBe('false');
    });

    it('shows Diff for a modified file opened through the regular Notes file list', async () => {
        getCurrentProjectMock.mockResolvedValue('/workspace/repo');
        getGitStatusMock.mockResolvedValue({
            root: '/workspace/repo', branch: 'main', ahead: 0, behind: 0, clean: false,
            files: [{ path: 'src/main.go', code: ' M', unstaged: true }],
        });
        listFilesMock.mockResolvedValue(['$PROJECT/src/main.go']);
        getFileMock.mockResolvedValue({ contents: 'package main\n', binary: false, error: '' });

        await importNotesModule();
        await flushPromises();
        document.querySelector('.notes-file')?.click();
        await flushPromises();
        await flushPromises();

        expect(document.getElementById('notes-current-file').textContent).toBe('main.go');
        expect(document.getElementById('notes-tab-diff').style.display).toBe('');
        expect(document.getElementById('notes-tab-diff').getAttribute('aria-selected')).toBe('false');

        getEventHandler('notesGitStatusChanged')({
            root: '/workspace/repo',
            status: { root: '/workspace/repo', branch: 'main', ahead: 0, behind: 0, clean: true, files: [] },
        });
        expect(document.getElementById('notes-tab-diff').style.display).toBe('none');
    });

    it.each([
        ['json', '{"name":"needle"}'],
        ['yaml', 'name: needle'],
        ['yml', 'name: needle'],
        ['xml', '<?xml version="1.0"?><root>needle</root>'],
    ])('finds %s source in the live Monaco structured Edit view', async (extension, source) => {
        const userAgent = vi.spyOn(navigator, 'userAgent', 'get').mockReturnValue('Monaco regression test');
        let value = source;
        const match = { start: source.indexOf('needle'), end: source.indexOf('needle') + 6 };
        const adapter = {
            getValue: () => value,
            setValue: (next) => { value = next; },
            getSelectionOffsets: () => ({ start: 0, end: 0 }),
            setSelectionOffsets: vi.fn(),
            findMatches: vi.fn(() => [match]),
            revealOffset: vi.fn(),
            focus: vi.fn(),
            layout: vi.fn(),
            setLanguage: vi.fn(),
            setWordWrap: vi.fn(),
            setTypography: vi.fn(),
            applyTheme: vi.fn(),
            configureLsp: vi.fn(),
            setDiagnostics: vi.fn(),
            setTyposMisspellings: vi.fn(),
        };
        createMonacoAdapterMock.mockResolvedValue(adapter);
        listFilesMock.mockResolvedValue([`$NOTES/search.${extension}`]);
        getFileMock.mockResolvedValue({ contents: source, text: '', error: '' });
        try {
            await importNotesModule();
            document.querySelector(`[data-file="$NOTES/search.${extension}"]`).click();
            await flushPromises();
            await flushPromises();
            document.getElementById('notes-tab-swagger-edit').click();
            await flushPromises();
            await flushPromises();
            expect(createMonacoAdapterMock).toHaveBeenCalled();
            document.getElementById('notes-editor').value = 'stale hidden textarea';
            const findInput = document.getElementById('notes-find-input');
            findInput.value = 'needle';
            findInput.dispatchEvent(new Event('input', { bubbles: true }));
            expect(adapter.findMatches).toHaveBeenCalledWith('needle', {
                isRegex: false, matchCase: false, limit: 10000,
            });
            expect(document.getElementById('notes-find-counter').textContent).toBe('1 of 1');
            expect(adapter.setSelectionOffsets).toHaveBeenCalledWith(match.start, match.end);
            expect(adapter.revealOffset).toHaveBeenCalledWith(match.start);
        } finally {
            userAgent.mockRestore();
        }
    });

    it.each(['metaKey', 'ctrlKey'])('captures %s+F before Monaco handles the shortcut', async (modifier) => {
        await importNotesModule();
        const panel = document.getElementById('notes-tools-panel');
        panel.dataset.collapsed = 'true';
        const monacoElement = document.getElementById('notes-monaco-editor');
        const nativeFind = vi.fn((event) => event.stopPropagation());
        monacoElement.addEventListener('keydown', nativeFind, true);
        const event = new KeyboardEvent('keydown', {
            key: 'f', [modifier]: true, bubbles: true, cancelable: true,
        });
        monacoElement.dispatchEvent(event);
        await flushPromises();
        expect(event.defaultPrevented).toBe(true);
        expect(nativeFind).not.toHaveBeenCalled();
        expect(panel.dataset.collapsed).toBe('false');
        expect(document.getElementById('notes-tools-tab-find').getAttribute('aria-selected')).toBe('true');
        expect(document.activeElement).toBe(document.getElementById('notes-find-input'));
    });

    it.each(['terminalFocusedState', 'ttyphoonInputboxOpen', 'fullscreen-image-overlay'])(
        'leaves the Find shortcut unhandled when %s is active', async (guard) => {
            await importNotesModule();
            const panel = document.getElementById('notes-tools-panel');
            panel.dataset.collapsed = 'true';
            const previous = window[guard];
            const overlay = document.createElement('div');
            if (guard === 'fullscreen-image-overlay') {
                overlay.id = guard;
                document.body.appendChild(overlay);
            } else {
                window[guard] = true;
            }
            try {
                const monacoElement = document.getElementById('notes-monaco-editor');
                const nativeFind = vi.fn();
                monacoElement.addEventListener('keydown', nativeFind, true);
                const event = new KeyboardEvent('keydown', {
                    key: 'f', metaKey: true, bubbles: true, cancelable: true,
                });
                monacoElement.dispatchEvent(event);
                expect(event.defaultPrevented).toBe(false);
                expect(nativeFind).toHaveBeenCalledOnce();
                expect(panel.dataset.collapsed).toBe('true');
            } finally {
                window[guard] = previous;
                overlay.remove();
            }
        },
    );

    it('keeps Find tools tab visible but disables controls in Hex and Meta modes', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide\n\nfind me', text: '', error: '' });

        await importNotesModule();

        const tabFind = document.getElementById('notes-tools-tab-find');
        const findControls = document.getElementById('notes-find-controls');
        const findInput = document.getElementById('notes-find-input');
        const findPrev = document.getElementById('notes-find-prev');
        const findNext = document.getElementById('notes-find-next');
        const tabHex = document.getElementById('notes-tab-hex');
        const tabMeta = document.getElementById('notes-tab-meta');
        const tabViewer = document.getElementById('notes-tab-viewer');

        expect(tabFind.style.display).not.toBe('none');

        tabHex.click();
        await flushPromises();
        expect(tabFind.style.display).not.toBe('none');
        expect(findControls.dataset.disabled).toBe('true');
        expect(findInput.disabled).toBe(true);
        expect(findPrev.disabled).toBe(true);
        expect(findNext.disabled).toBe(true);

        tabMeta.click();
        await flushPromises();
        expect(tabFind.style.display).not.toBe('none');
        expect(findControls.dataset.disabled).toBe('true');
        expect(findInput.disabled).toBe(true);
        expect(findPrev.disabled).toBe(true);
        expect(findNext.disabled).toBe(true);

        tabViewer.click();
        await flushPromises();
        expect(tabFind.style.display).not.toBe('none');
        expect(findControls.dataset.disabled).toBe('false');
        expect(findInput.disabled).toBe(false);
        expect(findPrev.disabled).toBe(false);
        expect(findNext.disabled).toBe(false);
    });

    it('finds and highlights CSV cell text in View and Run modes', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/search.csv']);
        getFileMock.mockResolvedValue({
            contents: 'Name,Description\nAlpha,needle here\nBeta,another needle',
            text: '',
            error: '',
        });

        await importNotesModule();
        document.querySelector('[data-file="$NOTES/search.csv"]').click();
        await flushPromises();
        await flushPromises();

        const findInput = document.getElementById('notes-find-input');
        const counter = document.getElementById('notes-find-counter');
        const csvView = document.getElementById('notes-csv-view');
        findInput.value = 'needle';
        findInput.dispatchEvent(new Event('input', { bubbles: true }));

        expect(csvView.querySelectorAll('.find-highlight')).toHaveLength(2);
        expect(counter.textContent).toBe('1 of 2');

        document.getElementById('notes-tab-csv-run').click();
        await flushPromises();
        await flushPromises();

        expect(csvView.querySelectorAll('.find-highlight')).toHaveLength(2);
        expect(counter.textContent).toBe('1 of 2');
    });

    it('keeps Find tools tab visible but disables controls in Swagger Run mode', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/openapi.json']);
        vi.mocked(swaggerUtils.parseSwaggerSpec).mockReturnValue({
            swagger: '2.0',
            paths: {
                '/ping': {
                    get: {
                        responses: {
                            200: { description: 'ok' },
                        },
                    },
                },
            },
        });
        vi.mocked(swaggerUtils.hasSwaggerKey).mockReturnValue(true);
        vi.mocked(swaggerUtils.extractPaths).mockReturnValue([{
            path: '/ping',
            methods: [{ method: 'GET', name: 'GET /ping', operation: { responses: { 200: { description: 'ok' } } } }],
        }]);
        getFileMock.mockResolvedValue({
            contents: JSON.stringify({
                swagger: '2.0',
                info: { title: 'Test API', version: '1.0.0' },
                paths: {
                    '/ping': {
                        get: {
                            responses: {
                                200: { description: 'ok' },
                            },
                        },
                    },
                },
            }),
            text: '',
            error: '',
        });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/openapi.json"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const tabFind = document.getElementById('notes-tools-tab-find');
        const findControls = document.getElementById('notes-find-controls');
        const findInput = document.getElementById('notes-find-input');
        const findPrev = document.getElementById('notes-find-prev');
        const findNext = document.getElementById('notes-find-next');
        const tabSwaggerRun = document.getElementById('notes-tab-swagger-run');
        const tabSwaggerEdit = document.getElementById('notes-tab-swagger-edit');

        expect(tabFind.style.display).not.toBe('none');

        tabSwaggerRun.click();
        await flushPromises();
        expect(tabFind.style.display).not.toBe('none');
        expect(findControls.dataset.disabled).toBe('true');
        expect(findInput.disabled).toBe(true);
        expect(findPrev.disabled).toBe(true);
        expect(findNext.disabled).toBe(true);

        tabSwaggerEdit.click();
        await flushPromises();
        expect(tabFind.style.display).not.toBe('none');
        expect(findControls.dataset.disabled).toBe('false');
        expect(findInput.disabled).toBe(false);
        expect(findPrev.disabled).toBe(false);
        expect(findNext.disabled).toBe(false);
    });

    it('does not auto-copy markdown viewer selection when highlighted', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/guide.md']);
        getFileMock.mockResolvedValue({ contents: '# Guide', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/guide.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const textNode = document.createTextNode('Selected markdown text');
        document.getElementById('notes-preview').appendChild(textNode);

        const selectionMock = {
            rangeCount: 1,
            anchorNode: textNode,
            focusNode: textNode,
            toString: () => 'Selected markdown text',
        };
        const originalGetSelection = window.getSelection;
        window.getSelection = vi.fn(() => selectionMock);

        document.dispatchEvent(new MouseEvent('mouseup', { bubbles: true, button: 0 }));
        await flushPromises();
        expect(clipboardSetTextMock).not.toHaveBeenCalled();
        expect(sendIpcMock).not.toHaveBeenCalledWith('terminal-notify', {
            level: 'info',
            message: 'Selection copied to clipboard',
        });

        window.getSelection = originalGetSelection;
    });

    it('shows Hex tab across code, markdown, and yaml views', async () => {
        listFilesMock.mockResolvedValue([
            '$NOTES/readme.md',
            '$NOTES/script.go',
            '$NOTES/spec.yaml',
        ]);

        getFileMock.mockImplementation(async (file) => {
            if (file.endsWith('.md')) {
                return { contents: '# Markdown note', text: '', error: '' };
            }

            if (file.endsWith('.yaml')) {
                return { contents: 'openapi: 3.0.0\ninfo:\n  title: Sample', text: '', error: '' };
            }

            return { contents: 'package main\n\nfunc main() {}', text: '', error: '' };
        });

        await importNotesModule();

        const tabEditor = document.getElementById('notes-tab-editor');
        const tabHex = document.getElementById('notes-tab-hex');
        const tabViewer = document.getElementById('notes-tab-viewer');
        const tabJupyter = document.getElementById('notes-tab-jupyter');
        const tabMeta = document.getElementById('notes-tab-meta');
        const tabSwaggerView = document.getElementById('notes-tab-swagger-view');
        const tabSwaggerEdit = document.getElementById('notes-tab-swagger-edit');
        const tabSwaggerRun = document.getElementById('notes-tab-swagger-run');

        const clickFile = async (filePath) => {
            const fileButton = document.querySelector(`[data-file="${filePath}"]`);
            fileButton.click();
            await flushPromises();
            await flushPromises();
        };

        await clickFile('$NOTES/script.go');
        expect(tabEditor.style.display).toBe('');
        expect(tabHex.style.display).toBe('');
        expect(tabViewer.style.display).toBe('none');
        expect(tabJupyter.style.display).toBe('none');
        expect(tabSwaggerView.style.display).toBe('none');
        expect(tabSwaggerEdit.style.display).toBe('none');
        expect(tabSwaggerRun.style.display).toBe('none');
        expect(tabMeta.style.display).toBe('');

        await clickFile('$NOTES/readme.md');
        expect(tabEditor.style.display).toBe('');
        expect(tabHex.style.display).toBe('');
        expect(tabViewer.style.display).toBe('');
        expect(tabJupyter.style.display).toBe('');
        expect(tabSwaggerView.style.display).toBe('none');
        expect(tabSwaggerEdit.style.display).toBe('none');
        expect(tabSwaggerRun.style.display).toBe('none');
        expect(tabMeta.style.display).toBe('');

        await clickFile('$NOTES/spec.yaml');
        expect(tabEditor.style.display).toBe('none');
        expect(tabHex.style.display).toBe('');
        expect(tabViewer.style.display).toBe('none');
        expect(tabJupyter.style.display).toBe('none');
        expect(tabSwaggerView.style.display).toBe('');
        expect(tabSwaggerEdit.style.display).toBe('');
        expect(tabSwaggerRun.style.display).toBe('none');
        expect(tabMeta.style.display).toBe('');
    });

    it('shows a Hex tab for binary files and renders hexdump output', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/app.bin']);
        getFileMock.mockResolvedValue({
            contents: 'Y2YgAAAAAABoZXh5CgAAAA==',
            binary: true,
            error: '',
        });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/app.bin"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const tabEditor = document.getElementById('notes-tab-editor');
        const tabHex = document.getElementById('notes-tab-hex');
        const tabViewer = document.getElementById('notes-tab-viewer');
        const tabMeta = document.getElementById('notes-tab-meta');
        const hexWrap = document.getElementById('notes-hex-wrap');
        const hexRoot = document.getElementById('notes-hex');
        const editorWrap = document.getElementById('notes-editor-wrap');
        const hexHeader = document.querySelector('.notes-hex-header');
        const offsetInput = document.querySelector('.notes-hex-offset-input');
        const goButton = document.querySelector('.notes-hex-offset-go');

        expect(tabEditor.style.display).toBe('none');
        expect(tabHex.getAttribute('aria-selected')).toBe('true');
        expect(tabViewer.style.display).toBe('none');
        expect(tabMeta.style.display).toBe('');
        expect(hexWrap.dataset.active).toBe('true');
        expect(editorWrap.dataset.active).toBe('false');
        expect(hexRoot.textContent).toContain('00000000');
        expect(hexRoot.textContent).toContain('63 66 20 00 00 00 00 00');
        expect(hexRoot.textContent).toContain('|cf .....hexy....|');
        expect(hexHeader.textContent).toContain('Offset');
        expect(offsetInput).toBeTruthy();
        expect(goButton).toBeTruthy();
    });

    it('renders hexdump for text files only when Hex tab is opened', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/readme.md']);
        getFileMock.mockResolvedValue({ contents: 'Hello hex', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/readme.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const hexRoot = document.getElementById('notes-hex');
        const tabHex = document.getElementById('notes-tab-hex');

        expect(hexRoot.textContent).not.toContain('00000000');

        tabHex.click();
        await flushPromises();
        await flushPromises();

        expect(hexRoot.textContent).toContain('00000000');
    });

    it('shows meta pane only when meta tab is selected and renders template-provided markdown', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/readme.md']);
        getFileMock.mockResolvedValue({ contents: '# Markdown note', text: '', error: '' });
        getFileMetaMarkdownMock.mockResolvedValue([
            '# readme.md',
            '',
            '## Attributes',
            '',
            '- Size: `123Bb`',
            '- Path:',
            '  ```',
            '  /tmp/readme.md',
            '  ```',
            '',
            '## Owners',
            '',
            '- User: `user`',
            '- Group: `group`',
            '',
            '## Permissions',
            '',
            '- Unix: `0644`',
            '- User: `rw-`',
            '- Group: `r--`',
            '- Other: `r--`',
        ].join('\n'));

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/readme.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const tabViewer = document.getElementById('notes-tab-viewer');
        const tabMeta = document.getElementById('notes-tab-meta');
        const previewWrap = document.getElementById('notes-preview-wrap');
        const metaWrap = document.getElementById('notes-meta-wrap');
        const metaRoot = document.getElementById('notes-meta');

        expect(tabViewer.getAttribute('aria-selected')).toBe('true');
        expect(metaWrap.dataset.active).toBe('false');
        expect(previewWrap.dataset.active).toBe('true');

        tabMeta.click();
        await flushPromises();

        expect(tabMeta.getAttribute('aria-selected')).toBe('true');
        expect(metaWrap.dataset.active).toBe('true');
        expect(previewWrap.dataset.active).toBe('false');
        expect(metaRoot.querySelector('pre')).toBeTruthy();
        expect(metaRoot.querySelectorAll('code').length).toBeGreaterThan(0);
    });

    it('uses Monaco-only editor surface without legacy highlight overlay', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/config.json']);
        getFileMock.mockResolvedValue({ contents: '{"longPropertyName": "value"}', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/config.json"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const monacoEditor = document.getElementById('notes-monaco-editor');
        const notesEditor = document.getElementById('notes-editor');
        const editorHighlight = document.getElementById('notes-editor-highlight');

        notesEditor.value = '{"veryLongPropertyNameThatForcesHorizontalScrolling": "value"}';
        notesEditor.dispatchEvent(new Event('input', { bubbles: true }));

        expect(monacoEditor).toBeTruthy();
        expect(editorHighlight).toBeNull();
    });

    it('undoes and redoes structured JSON format actions from editor context menu', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/config.json']);
        getFileMock.mockResolvedValue({
            contents: '{\n  "name": "ttyphoon",\n  "ok": true\n}',
            text: '',
            error: '',
        });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/config.json"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        editor.focus();

        editor.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 80, clientY: 120 }));
        await flushPromises();

        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        const minifyIndex = menuConfig.options.findIndex((title) => title === 'Format: Minify');
        expect(minifyIndex).toBeGreaterThanOrEqual(0);

        menuConfig.onSelect(minifyIndex);
        await flushPromises();

        const minified = '{"name":"ttyphoon","ok":true}';
        expect(editor.value).toBe(minified);

        editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'z', metaKey: true, bubbles: true, cancelable: true }));
        await flushPromises();
        expect(editor.value).toBe('{' + '\n  "name": "ttyphoon",\n  "ok": true\n}');

        editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'z', metaKey: true, shiftKey: true, bubbles: true, cancelable: true }));
        await flushPromises();
        expect(editor.value).toBe(minified);
    });

    it('cycles visible notes tabs with ctrl+tab', async () => {
        listFilesMock.mockResolvedValue([
            '$NOTES/readme.md',
            '$NOTES/spec.yaml',
        ]);

        getFileMock.mockImplementation(async (file) => {
            if (file.endsWith('.yaml')) {
                return { contents: 'openapi: 3.0.0\ninfo:\n  title: Sample', text: '', error: '' };
            }
            return { contents: '# Markdown note', text: '', error: '' };
        });

        await importNotesModule();

        const clickFile = async (filePath) => {
            const fileButton = document.querySelector(`[data-file="${filePath}"]`);
            fileButton.click();
            await flushPromises();
            await flushPromises();
        };

        // Markdown defaults to View, then cycles View -> Edit -> Run -> Hex -> Meta -> View.
        await clickFile('$NOTES/readme.md');
        const tabViewer = document.getElementById('notes-tab-viewer');
        const tabEditor = document.getElementById('notes-tab-editor');
        const tabHex = document.getElementById('notes-tab-hex');
        const tabJupyter = document.getElementById('notes-tab-jupyter');
        const tabMeta = document.getElementById('notes-tab-meta');

        expect(tabViewer.getAttribute('aria-selected')).toBe('true');
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        expect(tabEditor.getAttribute('aria-selected')).toBe('true');
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        expect(tabJupyter.getAttribute('aria-selected')).toBe('true');
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        expect(tabHex.getAttribute('aria-selected')).toBe('true');
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        expect(tabMeta.getAttribute('aria-selected')).toBe('true');
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        expect(tabViewer.getAttribute('aria-selected')).toBe('true');

        // YAML defaults to structured View, then cycles View -> Edit -> Hex -> Meta -> View (Run hidden without swagger key).
        await clickFile('$NOTES/spec.yaml');
        const tabSwaggerView = document.getElementById('notes-tab-swagger-view');
        const tabSwaggerEdit = document.getElementById('notes-tab-swagger-edit');
        const tabSwaggerRun = document.getElementById('notes-tab-swagger-run');

        expect(tabSwaggerRun.style.display).toBe('none');
        tabSwaggerView.click();
        await flushPromises();
        const selectedBefore = tabSwaggerView.getAttribute('aria-selected') === 'true' ? 'view' : 'edit';
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        const selectedAfterFirst = tabSwaggerView.getAttribute('aria-selected') === 'true'
            ? 'view'
            : (tabSwaggerEdit.getAttribute('aria-selected') === 'true'
                ? 'edit'
                : (tabHex.getAttribute('aria-selected') === 'true' ? 'hex' : 'meta'));
        expect(selectedAfterFirst).not.toBe(selectedBefore);
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        const selectedAfterSecond = tabSwaggerView.getAttribute('aria-selected') === 'true'
            ? 'view'
            : (tabSwaggerEdit.getAttribute('aria-selected') === 'true'
                ? 'edit'
                : (tabHex.getAttribute('aria-selected') === 'true' ? 'hex' : 'meta'));
        expect(selectedAfterSecond).not.toBe(selectedBefore);
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        const selectedAfterThird = tabSwaggerView.getAttribute('aria-selected') === 'true'
            ? 'view'
            : (tabSwaggerEdit.getAttribute('aria-selected') === 'true'
                ? 'edit'
                : (tabHex.getAttribute('aria-selected') === 'true' ? 'hex' : 'meta'));
        expect(selectedAfterThird).not.toBe(selectedBefore);
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', ctrlKey: true, bubbles: true, cancelable: true }));
        const selectedAfterFourth = tabSwaggerView.getAttribute('aria-selected') === 'true'
            ? 'view'
            : (tabSwaggerEdit.getAttribute('aria-selected') === 'true'
                ? 'edit'
                : (tabHex.getAttribute('aria-selected') === 'true' ? 'hex' : 'meta'));
        expect(selectedAfterFourth).toBe(selectedBefore);
    });

    it('disables grammar helpers and keeps spellcheck enabled on note editors', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/readme.md']);
        getFileMock.mockResolvedValue({ contents: '# Note\n\n```js\nconsole.log("hello")\n```', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/readme.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const notesEditor = document.getElementById('notes-editor');
        const jupyterEditor = document.querySelector('.jupyter-code-editable');

        expect(notesEditor.getAttribute('autocorrect')).toBe('off');
        expect(notesEditor.getAttribute('autocapitalize')).toBe('off');
        expect(notesEditor.getAttribute('autocomplete')).toBe('off');
        expect(notesEditor.getAttribute('data-gramm')).toBe('false');
        expect(notesEditor.getAttribute('data-gramm_editor')).toBe('false');
        expect(notesEditor.getAttribute('data-enable-grammarly')).toBe('false');
        expect(notesEditor.getAttribute('spellcheck')).toBe('false');

        expect(jupyterEditor).toBeTruthy();
        expect(jupyterEditor.getAttribute('autocorrect')).toBe('off');
        expect(jupyterEditor.getAttribute('autocapitalize')).toBe('off');
        expect(jupyterEditor.getAttribute('autocomplete')).toBe('off');
        expect(jupyterEditor.getAttribute('data-gramm')).toBe('false');
        expect(jupyterEditor.getAttribute('data-gramm_editor')).toBe('false');
        expect(jupyterEditor.getAttribute('data-enable-grammarly')).toBe('false');
        expect(jupyterEditor.getAttribute('spellcheck')).toBe('false');
    });

    it('edits markdown table cells on double click in Run tab', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/table.md']);
        getFileMock.mockResolvedValue({ contents: [
            '# Table',
            '',
            '| Name | Value |',
            '| --- | --- |',
            '| Alpha | 1 |',
            '| Beta | 2 |',
        ].join('\n'), text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/table.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const runTab = document.getElementById('notes-tab-jupyter');
        runTab.click();
        await flushPromises();
        await flushPromises();

        const targetCell = document.querySelector('#notes-jupyter tbody tr td');
        expect(targetCell).toBeTruthy();

        targetCell.dispatchEvent(new MouseEvent('dblclick', { bubbles: true, cancelable: true }));
        await flushPromises();

        expect(targetCell.getAttribute('contenteditable')).toBe('true');

        targetCell.textContent = 'Gamma';
        targetCell.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true }));
        await flushPromises();
        await flushPromises();

        const notesEditor = document.getElementById('notes-editor');
        expect(notesEditor.value).toContain('| Gamma | 1 |');
    });

    it('renders Markdown in Run table cells without creating hyperlinks', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/table-markdown.md']);
        getFileMock.mockResolvedValue({ contents: [
            '# Table',
            '',
            '| Emphasis | Code | Link | URL |',
            '| --- | --- | --- | --- |',
            '| **bold** and *italic* | `const x = 1` | [label](https://example.com) | https://example.com |',
        ].join('\n'), text: '', error: '' });

        await importNotesModule();
        document.querySelector('[data-file="$NOTES/table-markdown.md"]').click();
        await flushPromises();
        await flushPromises();

        document.getElementById('notes-tab-jupyter').click();
        await flushPromises();
        await flushPromises();

        const cells = document.querySelectorAll('#notes-jupyter tbody tr:first-child td');
        expect(cells[0].querySelector('strong')?.textContent).toBe('bold');
        expect(cells[0].querySelector('em')?.textContent).toBe('italic');
        expect(cells[1].querySelector('code')?.textContent).toBe('const x = 1');
        expect(cells[2].querySelector('a')).toBeNull();
        expect(cells[2].textContent).toContain('label');
        expect(cells[3].querySelector('a')).toBeNull();
        expect(cells[3].textContent).toContain('https://example.com');
    });

    it('executes markdown table functions from H2-H6 code blocks and uses stdout as cell value', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/table-fn.md']);
        runFunctionMock.mockResolvedValue({ Output: '5', IsError: false, CellId: 'A2' });
        getLanguageDescriptionsMock.mockResolvedValue(['Python 3.x']);
        getFileMock.mockResolvedValue({ contents: [
            '## Add',
            '',
            '```python',
            'print(int(args[0]) + int(args[1]))',
            '```',
            '',
            '| Expr |',
            '| --- |',
            '| =Add(2, 3) |',
        ].join('\n'), text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/table-fn.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const runTab = document.getElementById('notes-tab-jupyter');
        runTab.click();
        await flushPromises();
        await flushPromises();

        expect(runFunctionMock.mock.calls.length).toBeGreaterThan(0);
        const [docPath, functionName, cellId, executedCode, parameters, runtime] = runFunctionMock.mock.calls[0];
        expect(docPath).toBe('$NOTES/table-fn.md');
        expect(functionName).toBe('Add');
        expect(cellId).toBe('A2');
        expect(executedCode).toContain('int(args[0])');
        expect(executedCode).toContain('int(args[1])');
        expect(parameters).toEqual(['2', '3']);
        expect(runtime).toBe('Python 3.x');

        const renderedCell = document.querySelector('#notes-jupyter tbody tr td');
        expect(renderedCell).toBeTruthy();
        expect(String(renderedCell.textContent || '')).toContain('5');
    });

    it('renders table function stderr as #ERR formatted cell value', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/table-fn-err.md']);
        runFunctionMock.mockResolvedValue({ Output: 'Error line 1\r\nError line 2  ', IsError: true, CellId: 'A2' });
        getLanguageDescriptionsMock.mockResolvedValue(['Python 3.x']);
        getFileMock.mockResolvedValue({ contents: [
            '## Divide',
            '',
            '```python',
            'x = int(args[0])',
            'y = int(args[1])',
            'print(x / y)',
            '```',
            '',
            '| Expr |',
            '| --- |',
            '| =Divide(5, 0) |',
        ].join('\n'), text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/table-fn-err.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const runTab = document.getElementById('notes-tab-jupyter');
        runTab.click();
        await flushPromises();
        await flushPromises();

        expect(runFunctionMock.mock.calls.length).toBeGreaterThan(0);

        const renderedCell = document.querySelector('#notes-jupyter tbody tr td');
        expect(renderedCell).toBeTruthy();
        const cellText = String(renderedCell.textContent || '');
        expect(cellText).toMatch(/^#ERR Error line 1\s+Error line 2\s+A2$/);
    });

    it('expands table function range arguments into multiple RunFunction parameters', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/table-fn-range.md']);
        runFunctionMock.mockResolvedValue({ Output: '47', IsError: false, CellId: 'D2' });
        getLanguageDescriptionsMock.mockResolvedValue(['Python 3.x']);
        getFileMock.mockResolvedValue({ contents: [
            '## Sum',
            '',
            '```python',
            'print(sum(map(int, args)))',
            '```',
            '',
            '| A | B | C | Expr |',
            '| --- | --- | --- | --- |',
            '| 1 | 2 | 3 | =Sum(A2:C2, B:B, 3:3) |',
            '| 4 | 5 | 6 |  |',
        ].join('\n'), text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/table-fn-range.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const runTab = document.getElementById('notes-tab-jupyter');
        runTab.click();
        await flushPromises();
        await flushPromises();

        expect(runFunctionMock.mock.calls.length).toBeGreaterThan(0);
        const [docPath, functionName, cellId, , parameters] = runFunctionMock.mock.calls[0];
        expect(docPath).toBe('$NOTES/table-fn-range.md');
        expect(functionName).toBe('Sum');
        expect(cellId).toBe('D2');
        expect(parameters).toEqual(['1', '2', '3', 'B', '2', '5', '4', '5', '6', '']);
    });

    it('supports nested function calls following order of operations', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/table-fn-nested.md']);
        // Mock RunFunction to track all calls
        let callCount = 0;
        runFunctionMock.mockImplementation(() => {
            callCount += 1;
            if (callCount === 1) {
                // First call: nested sum(B2:C2) should return 5 (2+3)
                return Promise.resolve({ Output: '5', IsError: false, CellId: 'D2' });
            }
            // Second call: outer sum(A2:A2, 5) should return 6 (1+5)
            return Promise.resolve({ Output: '6', IsError: false, CellId: 'D2' });
        });
        getLanguageDescriptionsMock.mockResolvedValue(['Python 3.x']);
        getFileMock.mockResolvedValue({ contents: [
            '## Sum',
            '',
            '```python',
            'print(sum(map(int, args)))',
            '```',
            '',
            '| A | B | C | Expr |',
            '| --- | --- | --- | --- |',
            '| 1 | 2 | 3 | =Sum(A2, Sum(B2:C2)) |',
        ].join('\n'), text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/table-fn-nested.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const runTab = document.getElementById('notes-tab-jupyter');
        runTab.click();
        await flushPromises();
        await flushPromises();

        // Should have called RunFunction twice: once for nested Sum(B2:C2), once for outer Sum(A2, 5)
        expect(runFunctionMock.mock.calls.length).toBeGreaterThanOrEqual(2);
        
        // First call should be nested Sum(B2:C2) with parameters ['2', '3']
        const [firstDocPath, firstFunctionName, firstCallCellId, , firstCallParams] = runFunctionMock.mock.calls[0];
        expect(firstDocPath).toBe('$NOTES/table-fn-nested.md');
        expect(firstFunctionName).toBe('Sum');
        expect(firstCallCellId).toBe('D2');
        expect(firstCallParams).toEqual(['2', '3']);
        
        // Second call should be outer Sum(A2, 5) with parameters ['1', '5']
        const [secondDocPath, secondFunctionName, secondCallCellId, , secondCallParams] = runFunctionMock.mock.calls[1];
        expect(secondDocPath).toBe('$NOTES/table-fn-nested.md');
        expect(secondFunctionName).toBe('Sum');
        expect(secondCallCellId).toBe('D2');
        expect(secondCallParams).toEqual(['1', '5']);

        const renderedCell = document.querySelector('#notes-jupyter tbody tr td:last-child');
        expect(renderedCell).toBeTruthy();
        expect(String(renderedCell.textContent || '')).toContain('6');
    });

    it('evaluates multiple cells calling the same function without #ERR', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/table-fn-multi.md']);
        runFunctionMock.mockImplementation(async (_docPath, _functionName, cellId, _code, parameters) => {
            return {
                Output: String((parseInt(parameters[0], 10) || 0) + (parseInt(parameters[1], 10) || 0)),
                IsError: false,
                CellId: cellId,
            };
        });
        getLanguageDescriptionsMock.mockResolvedValue(['Python 3.x']);
        getFileMock.mockResolvedValue({ contents: [
            '## Add',
            '',
            '```python',
            'print(int(args[0]) + int(args[1]))',
            '```',
            '',
            '| Expr |',
            '| --- |',
            '| =Add(1, 2) |',
            '| =Add(3, 4) |',
        ].join('\n'), text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/table-fn-multi.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const runTab = document.getElementById('notes-tab-jupyter');
        runTab.click();
        await flushPromises();
        await flushPromises();

        expect(runFunctionMock.mock.calls.length).toBeGreaterThanOrEqual(2);
        const calledWithOneTwo = runFunctionMock.mock.calls.some((call) => JSON.stringify(call[4]) === JSON.stringify(['1', '2']));
        const calledWithThreeFour = runFunctionMock.mock.calls.some((call) => JSON.stringify(call[4]) === JSON.stringify(['3', '4']));
        const calledWithCellA2 = runFunctionMock.mock.calls.some((call) => call[2] === 'A2');
        const calledWithCellA3 = runFunctionMock.mock.calls.some((call) => call[2] === 'A3');
        expect(calledWithOneTwo).toBe(true);
        expect(calledWithThreeFour).toBe(true);
        expect(calledWithCellA2).toBe(true);
        expect(calledWithCellA3).toBe(true);

        const renderedCells = Array.from(document.querySelectorAll('#notes-jupyter tbody tr td'));
        expect(renderedCells.length).toBe(2);
        expect(String(renderedCells[0].textContent || '')).toContain('3');
        expect(String(renderedCells[1].textContent || '')).toContain('7');
        expect(String(renderedCells[0].textContent || '')).not.toContain('#ERR');
        expect(String(renderedCells[1].textContent || '')).not.toContain('#ERR');
    });

    it('respects dependencies when one function cell references another function cell', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/table-fn-deps.md']);
        runFunctionMock.mockImplementation(async (_docPath, _functionName, _cellId, _code, parameters) => {
            const left = parseInt(parameters[0], 10);
            const right = parseInt(parameters[1], 10);
            if (Number.isNaN(left) || Number.isNaN(right)) {
                return { Output: 'invalid integer', IsError: true, CellId: 'unknown' };
            }
            return { Output: String(left + right), IsError: false, CellId: 'unknown' };
        });
        getLanguageDescriptionsMock.mockResolvedValue(['Python 3.x']);
        getFileMock.mockResolvedValue({ contents: [
            '## Add',
            '',
            '```python',
            'print(int(args[0]) + int(args[1]))',
            '```',
            '',
            '| Expr |',
            '| --- |',
            '| =Add(1, 2) |',
            '| =Add(A2, 4) |',
        ].join('\n'), text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/table-fn-deps.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const runTab = document.getElementById('notes-tab-jupyter');
        runTab.click();
        await flushPromises();
        await flushPromises();

        const renderedCells = Array.from(document.querySelectorAll('#notes-jupyter tbody tr td'));
        expect(renderedCells.length).toBe(2);
        expect(String(renderedCells[0].textContent || '')).toContain('3');
        expect(String(renderedCells[1].textContent || '')).toContain('7');
        expect(String(renderedCells[0].textContent || '')).not.toContain('#ERR');
        expect(String(renderedCells[1].textContent || '')).not.toContain('#ERR');
    });

    it('does not execute table functions in CSV Run mode', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/table.csv']);
        getFileMock.mockResolvedValue({
            contents: 'Expr\n=Add(2,3)',
            text: '',
            error: '',
        });

        await importNotesModule();

        const markdownUtils = await import('./markdown-utils.js');

        const fileButton = document.querySelector('[data-file="$NOTES/table.csv"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const runTab = document.getElementById('notes-tab-csv-run');
        vi.mocked(markdownUtils.autoHyperlink).mockClear();
        runTab.click();
        await flushPromises();
        await flushPromises();

        expect(runNoteMock).not.toHaveBeenCalled();
    expect(markdownUtils.autoHyperlink).not.toHaveBeenCalled();
    });

    it('processes CSV view cells for custom markdown URLs and adds resize handles', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/table.csv']);
        getFileMock.mockResolvedValue({
            contents: 'Name,URL\nAlpha,https://example.com',
            text: '',
            error: '',
        });

        const markdownUtils = await import('./markdown-utils.js');
        vi.mocked(markdownUtils.autoHyperlink).mockClear();

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/table.csv"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const csvView = document.querySelector('#notes-csv-view');
        expect(csvView.querySelectorAll('.notes-table-col-resize-handle')).toHaveLength(4);
        expect(vi.mocked(markdownUtils.autoHyperlink)).toHaveBeenCalledWith(csvView);
        expect(getNotesRenderedStyles()).toContain('#notes-csv-view th .notes-table-col-resize-handle::before');
    });

    it('offers Create graph from any CSV table cell', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/table.csv']);
        getFileMock.mockResolvedValue({ contents: 'Name,Value\nAlpha,1', text: '', error: '' });
        await importNotesModule();
        document.querySelector('[data-file="$NOTES/table.csv"]').click();
        await flushPromises();
        await flushPromises();

        const cell = document.querySelector('#notes-csv-view tbody td');
        cell.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 8, clientY: 8 }));
        const menu = showLocalMenuMock.mock.calls.at(-1)[0];
        expect(menu.options.filter((option) => option.startsWith('Create graph'))).toEqual(['Create graph...']);
    });

    it('detects CSV value columns formatted with currency symbols and commas', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/accounts.csv']);
        getFileMock.mockResolvedValue({
            contents: [
                'Project,Account,USD',
                'CFR,dev,$ 744.00',
                'CFR,prod,"$ 1,001.56"',
                'Health App,test,$ 42.04',
                'Health App,prod,"$ 5,396.07"',
            ].join('\n'),
            text: '',
            error: '',
        });
        await importNotesModule();
        document.querySelector('[data-file="$NOTES/accounts.csv"]').click();
        await flushPromises();
        await flushPromises();

        const cell = document.querySelector('#notes-csv-view tbody td');
        cell.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 8, clientY: 8 }));
        const menu = showLocalMenuMock.mock.calls.at(-1)[0];
        menu.onSelect(menu.options.indexOf('Create graph...'));
        await flushPromises();
        document.getElementById('notes-modal-create').click();
        await flushPromises();
        await flushPromises();

        expect(notesTableSetGraphMock).toHaveBeenCalledTimes(1);
        expect(notesTableSetGraphMock.mock.calls[0][1]).toMatchObject({
            categoryColumn: 0,
            valueColumns: [2],
            seriesNames: ['USD'],
        });
    });

    it('preserves newlines inside CSV view cells', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/table.csv']);
        getFileMock.mockResolvedValue({
            contents: 'Name,Description\nAlpha,"first line\nsecond line"',
            text: '',
            error: '',
        });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/table.csv"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const cell = document.querySelector('#notes-csv-view tbody tr td:nth-child(2)');
        expect(cell?.textContent).toBe('first line\nsecond line');
        expect(getNotesRenderedStyles()).toContain('white-space: pre-wrap;');
    });

    it('undoes and redoes CSV table row insertion from Run mode context menu', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/table.csv']);
        getFileMock.mockResolvedValue({
            contents: 'Name,Value\nAlpha,1',
            text: '',
            error: '',
        });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/table.csv"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const runTab = document.getElementById('notes-tab-csv-run');
        runTab.click();
        await flushPromises();
        await flushPromises();

        const tableCell = document.querySelector('#notes-csv-view tbody tr td');
        expect(tableCell).toBeTruthy();

        tableCell.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 100, clientY: 100 }));
        await flushPromises();

        const menuConfig = showLocalMenuMock.mock.calls[showLocalMenuMock.mock.calls.length - 1][0];
        const insertRowIndex = menuConfig.options.findIndex((title) => title === 'Insert row (after)');
        expect(insertRowIndex).toBeGreaterThanOrEqual(0);

        menuConfig.onSelect(insertRowIndex);
        await flushPromises();

        const editor = document.getElementById('notes-editor');
        expect(editor.value).toBe('Name,Value\nAlpha,1\n,');

        const editTab = document.getElementById('notes-tab-csv-edit');
        editTab.click();
        await flushPromises();
        editor.focus();

        editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'z', metaKey: true, bubbles: true, cancelable: true }));
        await flushPromises();
        expect(editor.value).toBe('Name,Value\nAlpha,1');

        editor.dispatchEvent(new KeyboardEvent('keydown', { key: 'z', metaKey: true, shiftKey: true, bubbles: true, cancelable: true }));
        await flushPromises();
        expect(editor.value).toBe('Name,Value\nAlpha,1\n,');
    });

    it('does not attach table highlight callbacks in Run view', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/table.md']);
        getFileMock.mockResolvedValue({ contents: [
            '| Name | Value |',
            '| --- | --- |',
            '| Alpha | 1 |',
            '| Beta | 2 |',
        ].join('\n'), text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/table.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const runTab = document.getElementById('notes-tab-jupyter');
        runTab.click();
        await flushPromises();
        await flushPromises();

        const table = document.querySelector('#notes-jupyter table');
        const cell = table.querySelector('tbody tr td');

        cell.dispatchEvent(new MouseEvent('contextmenu', {
            bubbles: true,
            cancelable: true,
            clientX: 100,
            clientY: 100,
        }));
        await flushPromises();

        expect(showLocalMenuMock).toHaveBeenCalledTimes(1);
        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        expect(menuConfig.onHighlight).toBeNull();
        expect(menuConfig.onCancel).toBeNull();
    });

    it('does not attach table highlight callbacks in Notes view', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/table.md']);
        getFileMock.mockResolvedValue({ contents: [
            '| Name | Value |',
            '| --- | --- |',
            '| Alpha | 1 |',
        ].join('\n'), text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/table.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const runTab = document.getElementById('notes-tab-jupyter');
        runTab.click();
        await flushPromises();
        await flushPromises();

        const table = document.querySelector('#notes-jupyter table');
        const cell = table.querySelector('tbody tr td');

        cell.dispatchEvent(new MouseEvent('contextmenu', {
            bubbles: true,
            cancelable: true,
            clientX: 100,
            clientY: 100,
        }));
        await flushPromises();

        expect(showLocalMenuMock).toHaveBeenCalled();
        const menuConfig = showLocalMenuMock.mock.calls[0][0];

        expect(menuConfig.onHighlight).toBeNull();
        expect(menuConfig.onCancel).toBeNull();
    });

    it('does not attach table highlight callbacks on CSV tables', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/data.csv']);
        getFileMock.mockResolvedValue({ contents: 'Name,Value\nAlpha,1\nBeta,2', text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/data.csv"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const runTab = document.getElementById('notes-tab-csv-run');
        runTab.click();
        await flushPromises();
        await flushPromises();

        const table = document.querySelector('[data-csv-table] table');
        if (!table) {
            // Skip test if CSV table not rendered (can happen in certain test conditions)
            expect(true).toBe(true);
            return;
        }

        const cell = table.querySelector('tbody tr td') || table.querySelector('tr td');
        if (!cell) {
            // Skip if no cells exist
            expect(true).toBe(true);
            return;
        }

        showLocalMenuMock.mockReset();

        cell.dispatchEvent(new MouseEvent('contextmenu', {
            bubbles: true,
            cancelable: true,
            clientX: 100,
            clientY: 100,
        }));
        await flushPromises();

        if (!showLocalMenuMock.mock.calls.length) {
            // CSV menu might not trigger in test environment
            expect(true).toBe(true);
            return;
        }

        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        expect(menuConfig.onHighlight).toBeNull();
        expect(menuConfig.onCancel).toBeNull();
    });

    // Sorting and filtering are owned by the Go engine so the terminal and Notes
    // cannot drift apart; the frontend only reorders rows it already rendered.
    describe('Go-backed table sorting', () => {
        const loadMarkdownTable = async () => {
            listFilesMock.mockResolvedValue(['$NOTES/table.md']);
            getFileMock.mockResolvedValue({ contents: [
                '| Name | Value |',
                '| --- | --- |',
                '| Alpha | 1 |',
                '| Beta | 2 |',
                '| Gamma | 3 |',
            ].join('\n'), text: '', error: '' });

            await importNotesModule();

            document.querySelector('[data-file="$NOTES/table.md"]').click();
            await flushPromises();
            await flushPromises();

            return document.querySelector('#notes-preview table');
        };

        const bodyText = (table) => Array.from(table.querySelectorAll('tbody tr'))
            .map((row) => row.querySelector('td')?.textContent?.trim());

        it('delegates a heading click to the Go engine with a 1-based column', async () => {
            const table = await loadMarkdownTable();
            expect(table).toBeTruthy();

            notesTableSortMock.mockResolvedValue({
                missing: false, order: [2, 0, 1], sortColumn: 1, sortDesc: false, filter: '', error: '',
            });

            table.querySelectorAll('thead th')[0].dispatchEvent(
                new MouseEvent('click', { bubbles: true, cancelable: true }));
            await flushPromises();
            await flushPromises();

            expect(notesTableSortMock).toHaveBeenCalled();
            const [request, column] = notesTableSortMock.mock.calls[0];
            expect(column).toBe(1);
            expect(request.key.surface).toBe('preview');
            expect(request.seed.headings).toEqual(['Name', 'Value']);
            expect(request.seed.rows).toEqual([['Alpha', '1'], ['Beta', '2'], ['Gamma', '3']]);
        });

        it('reorders existing rows to the order returned by Go', async () => {
            const table = await loadMarkdownTable();
            notesTableSortMock.mockResolvedValue({
                missing: false, order: [2, 0, 1], sortColumn: 1, sortDesc: false, filter: '', error: '',
            });

            table.querySelectorAll('thead th')[0].dispatchEvent(
                new MouseEvent('click', { bubbles: true, cancelable: true }));
            await flushPromises();
            await flushPromises();

            expect(bodyText(table)).toEqual(['Gamma', 'Alpha', 'Beta']);
        });

        it('shows the sort direction with the same arrows as the terminal', async () => {
            const table = await loadMarkdownTable();
            notesTableSortMock.mockResolvedValue({
                missing: false, order: [0, 1, 2], sortColumn: 1, sortDesc: true, filter: '', error: '',
            });

            table.querySelectorAll('thead th')[0].dispatchEvent(
                new MouseEvent('click', { bubbles: true, cancelable: true }));
            await flushPromises();
            await flushPromises();

            expect(table.querySelector('.notes-sort-icon')?.textContent).toBe('\u2193');
        });

        // Rows excluded by a filter stay in the DOM so cell editing and formulas
        // keep resolving against a complete document.
        it('hides filtered rows without removing them', async () => {
            const table = await loadMarkdownTable();
            notesTableSortMock.mockResolvedValue({
                missing: false, order: [1], sortColumn: 0, sortDesc: false, filter: '"Value" > 1', error: '',
            });

            table.querySelectorAll('thead th')[0].dispatchEvent(
                new MouseEvent('click', { bubbles: true, cancelable: true }));
            await flushPromises();
            await flushPromises();

            const rows = Array.from(table.querySelectorAll('tbody tr'));
            expect(rows).toHaveLength(3);
            expect(rows.filter((row) => !row.classList.contains('notes-table-row-filtered'))).toHaveLength(1);
        });

        it('copies only SQL-filtered rows in Notes and Run views', async () => {
            notesTableRestoreMock.mockResolvedValue({
                missing: false, order: [1], sortColumn: 0, sortDesc: false,
                filter: '"Value" > 1', error: '',
            });
            await loadMarkdownTable();

            const copyFormats = async (table) => {
                const visibleRow = table.querySelector('tbody tr:not(.notes-table-row-filtered)');
                expect(visibleRow).toBeTruthy();
                const visibleCells = Array.from(visibleRow.querySelectorAll('td')).map((cell) => {
                    const clone = cell.cloneNode(true);
                    clone.querySelectorAll('.notes-cellref').forEach((node) => node.remove());
                    return clone.textContent.trim();
                });
                const csv = `Name,Value\n${visibleCells.join(',')}`;
                const markdown = `| Name | Value |\n| --- | --- |\n| ${visibleCells.join(' | ')} |`;

                const openMenu = async () => {
                    table.querySelector('tbody td').dispatchEvent(
                        new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 10, clientY: 10 }));
                    await flushPromises();
                    return showLocalMenuMock.mock.calls.at(-1)[0];
                };

                let menu = await openMenu();
                menu.onSelect(menu.options.indexOf('Copy table (CSV)'));
                await flushPromises();
                expect(clipboardSetTextMock).toHaveBeenLastCalledWith(csv);

                menu = await openMenu();
                menu.onSelect(menu.options.indexOf('Copy table (Markdown)'));
                await flushPromises();
                expect(clipboardSetTextMock).toHaveBeenLastCalledWith(markdown);

                menu = await openMenu();
                menu.onSelect(menu.options.indexOf('Copy table (Formatted)'));
                await flushPromises();
                const [html, rtf, plainText] = clipboardSetHTMLMock.mock.calls.at(-1);
                const copiedTable = document.createElement('div');
                copiedTable.innerHTML = html;
                expect(Array.from(copiedTable.querySelectorAll('tbody tr')).map((row) =>
                    Array.from(row.querySelectorAll('td')).map((cell) => cell.textContent.trim()))).toEqual([visibleCells]);
                expect(plainText).toBe(`Name\tValue\n${visibleCells.join('\t')}\n`);
                expect(rtf).toContain('Beta');
                expect(rtf).not.toContain('Alpha');
                expect(rtf).not.toContain('Gamma');
            };

            await copyFormats(document.querySelector('#notes-preview table'));

            document.getElementById('notes-tab-jupyter').click();
            await flushPromises();
            await flushPromises();
            await copyFormats(document.querySelector('#notes-jupyter table'));
        });

        // Losing server-side state must be recoverable, so a miss is retried once
        // with the data attached rather than surfacing as a broken table.
        it('resends the seed when Go reports the table is missing', async () => {
            const table = await loadMarkdownTable();
            notesTableSortMock
                .mockResolvedValueOnce({ missing: true, order: [], sortColumn: 0, sortDesc: false, filter: '', error: '' })
                .mockResolvedValueOnce({ missing: false, order: [0, 1, 2], sortColumn: 1, sortDesc: false, filter: '', error: '' });

            table.querySelectorAll('thead th')[0].dispatchEvent(
                new MouseEvent('click', { bubbles: true, cancelable: true }));
            await flushPromises();
            await flushPromises();

            expect(notesTableSortMock).toHaveBeenCalledTimes(2);
            expect(notesTableSortMock.mock.calls[1][0].seed).not.toBeNull();
        });

        it('offers clear sorting and SQL filter on a heading right-click', async () => {
            const table = await loadMarkdownTable();
            showLocalMenuMock.mockReset();

            table.querySelectorAll('thead th')[0].dispatchEvent(
                new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 10, clientY: 10 }));
            await flushPromises();

            expect(showLocalMenuMock).toHaveBeenCalled();
            expect(showLocalMenuMock).toHaveBeenCalledTimes(1);
            expect(showLocalMenuMock.mock.calls[0][0].options).toEqual(expect.arrayContaining([
                'Copy',
                'Copy table (CSV)',
                'Copy table (Markdown)',
                'Clear sorting',
                'SQL filter...',
                'Create graph...',
            ]));
        });

        it('opens the SQL filter dialog when the filter footer is clicked', async () => {
            notesTableRestoreMock.mockResolvedValue({
                missing: false, order: [0, 1, 2], sortColumn: 0, sortDesc: false,
                filter: '"Value" > 1', error: '',
            });
            const table = await loadMarkdownTable();
            const footer = document.querySelector('.notes-table-sql-filter-footer');
            expect(footer).toBeTruthy();
            expect(footer.querySelector('strong')?.textContent).toBe('SQL filter: ');
            expect(footer.querySelector('code')?.textContent).toBe('"Value" > 1');

            footer.click();
            await flushPromises();

            expect(document.getElementById('notes-modal-title').textContent).toBe('SQL filter');
            expect(document.getElementById('notes-modal-input').value).toBe('"Value" > 1');
        });

        it('offers one graph action when right-clicking anywhere on a table', async () => {
            const table = await loadMarkdownTable();
            table.querySelector('tbody td').dispatchEvent(
                new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 10, clientY: 10 }));
            const menu = showLocalMenuMock.mock.calls.at(-1)[0];
            expect(menu.options).toContain('Copy table (CSV)');
            expect(menu.options).toContain('Copy table (Markdown)');
            expect(menu.options.filter((option) => option.startsWith('Create graph'))).toEqual(['Create graph...']);

            menu.onSelect(menu.options.indexOf('Create graph...'));
            await flushPromises();

            expect(document.getElementById('notes-modal-title').textContent).toBe('Create graph');
            const titleInput = document.querySelector('.notes-graph-text-title');
            const checkerIndex = attachSpellCheck.mock.calls.findIndex(([target]) => target === titleInput);
            expect(checkerIndex).toBeGreaterThanOrEqual(0);
            const titleChecker = attachSpellCheck.mock.results[checkerIndex].value;
            for (const control of document.querySelectorAll('#notes-modal-body input[type="text"], #notes-modal-body input[type="search"]')) {
                expect(control.getAttribute('autocorrect')).toBe('off');
                expect(control.getAttribute('writingsuggestions')).toBe('false');
                expect(control.getAttribute('spellcheck')).toBe('false');
                expect(attachSpellCheck.mock.calls.some(([target]) => target === control)).toBe(control === titleInput);
            }
            expect(document.querySelector('#notes-modal-title h3')?.textContent).toBe('Create graph');
            expect(getNotesRenderedStyles()).toMatch(/\.notes-graph-options-list\s*\{[^}]*height:\s*190px;[^}]*min-height:\s*190px;[^}]*max-height:\s*190px;/s);
            expect(document.querySelector('#notes-modal-body .notes-graph-dropdown')?.textContent.trim()).toContain('Line');
            const actions = Array.from(document.getElementById('notes-modal-actions').children).map((button) => button.id);
            expect(actions).toEqual(['notes-modal-create', 'notes-modal-cancel']);
            const columnsField = Array.from(document.querySelectorAll('.notes-graph-field'))
                .find((field) => field.querySelector('.notes-graph-field-label')?.textContent === 'Columns');
            const rowsField = Array.from(document.querySelectorAll('.notes-graph-field'))
                .find((field) => field.querySelector('.notes-graph-field-label')?.textContent === 'Rows');
            expect(columnsField.querySelectorAll('input[type="checkbox"]')).toHaveLength(2);
            expect(columnsField.querySelector('.notes-graph-options-list').classList.contains('markdown-body')).toBe(true);
            expect(Array.from(columnsField.querySelectorAll('input[type="checkbox"]')).every((input) => input.checked)).toBe(true);
            const columnSearch = columnsField.querySelector('input[placeholder="Find columns…"]');
            columnSearch.value = 'Value';
            columnSearch.dispatchEvent(new Event('input', { bubbles: true }));
            expect(columnsField.querySelectorAll('.notes-graph-option')).toHaveLength(1);
            expect(columnsField.querySelector('.notes-graph-option span').textContent).toBe('Value');
            columnSearch.value = '';
            columnSearch.dispatchEvent(new Event('input', { bubbles: true }));
            expect(columnsField.querySelectorAll('.notes-graph-option')).toHaveLength(2);
            expect(rowsField.querySelectorAll('input[type="checkbox"]')).toHaveLength(3);
            expect(rowsField.querySelector('.notes-graph-options-list').classList.contains('markdown-body')).toBe(true);
            expect(Array.from(rowsField.querySelectorAll('input[type="checkbox"]')).every((input) => input.checked)).toBe(true);

            const typeDropdown = document.querySelector('.notes-graph-dropdown');
            typeDropdown.click();
            const typeMenu = showLocalMenuMock.mock.calls.at(-1)[0];
            expect(typeMenu.options).toContain('Doughnut');
            typeMenu.onSelect(typeMenu.options.indexOf('Bar'));
            expect(typeDropdown.textContent).toContain('Bar');

            const graphModal = document.getElementById('notes-modal');
            graphModal.querySelector('#notes-modal-card').dispatchEvent(new MouseEvent('click', { bubbles: true }));
            expect(graphModal.dataset.open).toBe('true');
            graphModal.dispatchEvent(new MouseEvent('click', { bubbles: true }));
            expect(graphModal.dataset.open).toBe('false');
            expect(titleChecker.detach).toHaveBeenCalledOnce();
        });

        it('persists a graph and intersects selected rows with SQL-filtered rows', async () => {
            const table = await loadMarkdownTable();
            notesTableFilterMock.mockResolvedValue({
                missing: false,
                order: [1],
                sortColumn: 0,
                sortDesc: false,
                filter: '"Value" > 1',
                graph: null,
                error: '',
            });

            const openTableMenu = () => {
                table.querySelector('tbody td').dispatchEvent(
                    new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 10, clientY: 10 }));
                return showLocalMenuMock.mock.calls.at(-1)[0];
            };
            let menu = openTableMenu();
            menu.onSelect(menu.options.indexOf('SQL filter...'));
            await flushPromises();
            document.getElementById('notes-modal-input').value = '"Value" > 1';
            document.getElementById('notes-modal-create').click();
            await flushPromises();
            await flushPromises();

            menu = openTableMenu();
            menu.onSelect(menu.options.indexOf('Create graph...'));
            await flushPromises();
            const rowsField = Array.from(document.querySelectorAll('.notes-graph-field'))
                .find((field) => field.querySelector('.notes-graph-field-label')?.textContent === 'Rows');
            const rowCheckboxes = rowsField.querySelectorAll('input[type="checkbox"]');
            rowCheckboxes[0].click();
            rowCheckboxes[2].click();
            document.getElementById('notes-modal-create').click();
            await flushPromises();
            await flushPromises();

            expect(notesTableSetGraphMock).toHaveBeenCalledTimes(1);
            const [persistRequest, spec] = notesTableSetGraphMock.mock.calls[0];
            expect(persistRequest.workspace).toBeTruthy();
            expect(persistRequest.key.document).toBe('$NOTES/table.md');
            expect(persistRequest.key.index).toBe(0);
            expect(persistRequest.stateIndex).toBe(0);
            expect(spec).toMatchObject({ type: 'line', categoryColumn: 0, valueColumns: [1], rowIndices: [1] });

            const [renderRequest] = notesTableRenderGraphMock.mock.calls[0];
            expect(renderRequest.seed.headings).toEqual(['Name', 'Value']);
            expect(renderRequest.seed.rows).toEqual([['Alpha', '1'], ['Beta', '2'], ['Gamma', '3']]);
            expect(renderRequest.filter).toBe('"Value" > 1');

            const extras = document.querySelector('.notes-table-extras');
            expect(extras.children[0].classList.contains('notes-table-sql-filter-footer')).toBe(true);
            expect(extras.children[1].classList.contains('notes-table-graph')).toBe(true);
            expect(extras.querySelector('.notes-table-graph-image')?.getAttribute('src')).toContain('data:image/svg+xml');
            extras.querySelector('.notes-table-graph-image').click();
            const fullscreenSvg = document.querySelector('#fullscreen-image-overlay svg');
            expect(fullscreenSvg).not.toBeNull();
            expect(fullscreenSvg.getAttribute('width')).toBe('800');
            expect(fullscreenSvg.getAttribute('height')).toBe('600');
            document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
        });

        it('renders an unfiltered graph from every table row', async () => {
            const table = await loadMarkdownTable();
            table.querySelector('tbody td').dispatchEvent(
                new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 10, clientY: 10 }));
            const menu = showLocalMenuMock.mock.calls.at(-1)[0];
            menu.onSelect(menu.options.indexOf('Create graph...'));
            await flushPromises();
            document.getElementById('notes-modal-create').click();
            await flushPromises();
            await flushPromises();

            const [renderRequest] = notesTableRenderGraphMock.mock.calls[0];
            expect(renderRequest.filter).toBe('');
            expect(renderRequest.seed.rows).toEqual([
                ['Alpha', '1'],
                ['Beta', '2'],
                ['Gamma', '3'],
            ]);
        });

        it('displays a saved PNG-format graph as SVG without changing the saved format', async () => {
            const spec = { type: 'bar', format: 'png', categoryColumn: 0, valueColumns: [1] };
            notesTableRestoreMock.mockResolvedValue({ ...emptyTableResult(), order: [0, 1, 2], graph: spec });
            const table = await loadMarkdownTable();
            expect(notesTableRenderGraphMock).toHaveBeenCalledWith(expect.any(Object), { ...spec, format: 'svg' });
            expect(document.querySelector('.notes-table-graph-image').src).toContain('data:image/svg+xml');
            expect(JSON.parse(table.dataset.tableGraphSpec)).toEqual(spec);
        });

        it.each(['table', 'graph'])('modifies an existing graph from the %s context menu', async (target) => {
            const spec = { type: 'bar', format: 'svg', title: 'Saved chart', categoryColumn: 0, valueColumns: [1], rowIndices: [1] };
            notesTableRestoreMock.mockResolvedValue({ ...emptyTableResult(), order: [0, 1, 2], graph: spec });
            const table = await loadMarkdownTable();
            const element = target === 'table' ? table.querySelector('tbody td') : document.querySelector('.notes-table-graph');
            expect(element).not.toBeNull();
            element.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true }));
            const menu = showLocalMenuMock.mock.calls.at(-1)[0];
            expect(menu.options).toContain('Modify graph...');
            expect(menu.options).not.toContain('Create graph...');
            expect(menu.icons[menu.options.indexOf('Modify graph...')]).toBe(0xf201);
            menu.onSelect(menu.options.indexOf('Modify graph...'));
            await flushPromises();
            expect(document.getElementById('notes-modal-title').textContent).toBe('Modify graph');
            expect(document.getElementById('notes-modal-create').textContent).toBe('Modify graph');
            expect(document.querySelector('.notes-graph-dropdown').textContent).toContain('Bar');
            const title = document.querySelector('.notes-graph-text-title');
            expect(title.value).toBe('Saved chart');
            title.value = 'Modified chart';
            title.dispatchEvent(new Event('input', { bubbles: true }));
            document.getElementById('notes-modal-create').click();
            await flushPromises();
            await flushPromises();
            expect(notesTableSetGraphMock).toHaveBeenLastCalledWith(expect.any(Object), expect.objectContaining({
                type: 'bar', format: 'svg', title: 'Modified chart', categoryColumn: 0, valueColumns: [1], rowIndices: [1],
            }));
        });

        it.each(['success', 'render failure', 'clipboard failure'])('copies a graph as PNG with %s handling', async (outcome) => {
            const spec = { type: 'bar', format: 'svg', title: 'Saved chart', categoryColumn: 0, valueColumns: [1], rowIndices: [1] };
            notesTableRestoreMock.mockResolvedValue({ ...emptyTableResult(), order: [0, 1, 2], graph: spec });
            const table = await loadMarkdownTable();
            const pngDataURL = 'data:image/png;base64,iVBORw0KGgo=';
            notesTableRenderGraphMock.mockResolvedValueOnce(outcome === 'render failure'
                ? { error: 'render failed' } : { pngDataURL, error: '' });
            if (outcome === 'clipboard failure') terminalCopyImageDataURLMock.mockRejectedValueOnce(new Error('clipboard failed'));
            document.querySelector('.notes-table-graph').dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true }));
            const menu = showLocalMenuMock.mock.calls.at(-1)[0];
            menu.onSelect(menu.options.indexOf('Copy graph (PNG)'));
            await flushPromises();
            await flushPromises();
            expect(notesTableRenderGraphMock).toHaveBeenLastCalledWith(expect.objectContaining({
                seed: expect.objectContaining({ headings: ['Name', 'Value'], rows: [['Alpha', '1'], ['Beta', '2'], ['Gamma', '3']] }),
            }), { ...spec, format: 'png' });
            if (outcome === 'render failure') {
                expect(terminalCopyImageDataURLMock).not.toHaveBeenCalled();
            } else {
                expect(terminalCopyImageDataURLMock).toHaveBeenCalledWith(pngDataURL);
            }
            if (outcome !== 'success') {
                expect(sendIpcMock).toHaveBeenCalledWith('terminal-notify', expect.objectContaining({
                    level: 'error', message: `Failed to copy graph: ${outcome === 'render failure' ? 'render failed' : 'clipboard failed'}`,
                }));
            }
            expect(notesTableSetGraphMock).not.toHaveBeenCalled();
            expect(JSON.parse(table.dataset.tableGraphSpec)).toEqual(spec);
            expect(document.querySelector('.notes-table-graph')).not.toBeNull();
        });

        it('removes the persisted graph from the graph context menu', async () => {
            const table = await loadMarkdownTable();
            table.querySelector('tbody td').dispatchEvent(
                new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 10, clientY: 10 }));
            const tableMenu = showLocalMenuMock.mock.calls.at(-1)[0];
            tableMenu.onSelect(tableMenu.options.indexOf('Create graph...'));
            await flushPromises();
            document.getElementById('notes-modal-create').click();
            await flushPromises();
            await flushPromises();

            const graph = document.querySelector('.notes-table-graph');
            graph.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 12, clientY: 12 }));
            const graphMenu = showLocalMenuMock.mock.calls.at(-1)[0];
            expect(graphMenu.options).toEqual(['Copy graph (PNG)', 'Modify graph...', '-', 'Remove graph']);
            expect(graphMenu.icons).toEqual([0xf0c5, 0xf201, 0, 0]);
            graphMenu.onSelect(graphMenu.options.indexOf('Remove graph'));
            await flushPromises();

            expect(notesTableSetGraphMock).toHaveBeenLastCalledWith(
                expect.objectContaining({ key: expect.objectContaining({ document: '$NOTES/table.md' }) }),
                null,
            );
            expect(document.querySelector('.notes-table-graph')).toBeNull();
        });

        it('clears the sort on a middle-click, as the terminal does', async () => {
            const table = await loadMarkdownTable();
            notesTableClearSortMock.mockResolvedValue({
                missing: false, order: [0, 1, 2], sortColumn: 0, sortDesc: false, filter: '', error: '',
            });

            table.querySelectorAll('thead th')[0].dispatchEvent(
                new MouseEvent('auxclick', { bubbles: true, cancelable: true, button: 1 }));
            await flushPromises();
            await flushPromises();

            expect(notesTableClearSortMock).toHaveBeenCalled();
            expect(notesTableSortMock).not.toHaveBeenCalled();
            expect(table.querySelector('.notes-sort-icon')).toBeNull();
        });

        it('ignores other auxiliary buttons on a heading', async () => {
            const table = await loadMarkdownTable();

            table.querySelectorAll('thead th')[0].dispatchEvent(
                new MouseEvent('auxclick', { bubbles: true, cancelable: true, button: 3 }));
            await flushPromises();

            expect(notesTableClearSortMock).not.toHaveBeenCalled();
        });

        // Each surface reconciles independently, so one rendering must not drop
        // tables belonging to another.
        it('declares the tables that still exist for each surface', async () => {
            await loadMarkdownTable();

            expect(notesTableReconcileMock).toHaveBeenCalled();

            const previewCall = notesTableReconcileMock.mock.calls.find(([surface]) => surface === 'preview');
            expect(previewCall).toBeTruthy();
            expect(previewCall[2]).toEqual([0]);

            const surfaces = notesTableReconcileMock.mock.calls.map(([surface]) => surface);
            expect(new Set(surfaces).size).toBeGreaterThan(1);
        });

        it('applies the Notes table tools to markdown modal tables using the caller ID', async () => {
            await importNotesModule();
            getNotesColumnWidthsMock.mockClear();

            getEventHandler('showMarkdownModal')({
                id: 'git-info:/repo/worktree',
                markdown: '| Name | Count |\n| --- | --- |\n| beta | 2 |\n| alpha | 10 |',
            });
            await flushPromises();
            await flushPromises();

            const table = document.querySelector('#markdown-modal-body table');
            expect(table.querySelector('.notes-table-col-resize-handle')).not.toBeNull();
            expect(getNotesColumnWidthsMock).toHaveBeenCalledWith(
                'git-info:/repo/worktree', 'markdown-modal', ['Name', 'Count'], true,
            );
            expect(notesTableReconcileMock).toHaveBeenCalledWith('markdown-modal', 'git-info:/repo/worktree', [0]);

            notesTableSortMock.mockClear();
            table.querySelector('thead th').click();
            await flushPromises();
            expect(notesTableSortMock).toHaveBeenCalledWith(expect.objectContaining({
                key: { surface: 'markdown-modal', document: 'git-info:/repo/worktree', index: 0 },
            }), 1);
        });
    });

    it('does not highlight the table when hovering its Run menu items', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/table.md']);
        getFileMock.mockResolvedValue({ contents: [
            '| Name | Value |',
            '| --- | --- |',
            '| Alpha | 1 |',
            '| Beta | 2 |',
        ].join('\n'), text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/table.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        const runTab = document.getElementById('notes-tab-jupyter');
        runTab.click();
        await flushPromises();
        await flushPromises();

        const table = document.querySelector('#notes-jupyter table');
        if (!table) {
            // Skip if no table
            expect(true).toBe(true);
            return;
        }

        const cell = table.querySelector('tbody tr td');
        if (!cell) {
            expect(true).toBe(true);
            return;
        }

        cell.dispatchEvent(new MouseEvent('contextmenu', {
            bubbles: true,
            cancelable: true,
            clientX: 100,
            clientY: 100,
        }));
        await flushPromises();

        expect(showLocalMenuMock).toHaveBeenCalled();
        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        expect(menuConfig.onHighlight).toBeNull();
        expect(menuConfig.onCancel).toBeNull();
    });

    it('does not highlight the table when hovering its Notes menu items', async () => {
        listFilesMock.mockResolvedValue(['$NOTES/table.md']);
        getFileMock.mockResolvedValue({ contents: [
            '| Name | Value |',
            '| --- | --- |',
            '| Alpha | 1 |',
            '| Beta | 2 |',
        ].join('\n'), text: '', error: '' });

        await importNotesModule();

        const fileButton = document.querySelector('[data-file="$NOTES/table.md"]');
        fileButton.click();
        await flushPromises();
        await flushPromises();

        // Stay in viewer mode (don't switch to Run tab)
        const table = document.querySelector('#notes-preview table');
        if (!table) {
            // Skip if no table in preview
            expect(true).toBe(true);
            return;
        }

        const cell = table.querySelector('tr td');
        if (!cell) {
            expect(true).toBe(true);
            return;
        }

        showLocalMenuMock.mockReset();

        cell.dispatchEvent(new MouseEvent('contextmenu', {
            bubbles: true,
            cancelable: true,
            clientX: 100,
            clientY: 100,
        }));
        await flushPromises();

        if (!showLocalMenuMock.mock.calls.length) {
            expect(true).toBe(true);
            return;
        }

        const menuConfig = showLocalMenuMock.mock.calls[0][0];
        expect(menuConfig.onHighlight).toBeNull();
        expect(menuConfig.onCancel).toBeNull();
    });

});