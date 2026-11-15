import {
    GetWindowStyle, GetNotesMaxLogLines, GetNotesColumnWidths, SetNotesColumnWidths, GetFile,
    GetGitStatus, GetGitBrowserLink, GetGitBranchDiffFiles, GetGitDiff, GetGitDiffAgainstBranch,
    RefreshGitStatus, SetGitStatusWatch, StageGitFile, StageAllGitFiles, UnstageGitFile, RunGitAction, ConfirmGitPush,
        ListGitWorktrees, ListGitBranches, ListGitTags, ListGitCommits, ListGitCommitsWithFiles, GetDefaultGitWorktreePath, CreateGitWorktree, CreateGitTagWorktree,
    CheckoutGitBranch, DeleteGitBranch,
        CreateGitTag, DeleteGitTag, CheckoutGitTag,
    SwitchGitWorktree, RemoveGitWorktree, ShowGitWorktreeInfo, ShowGitRefInfo,
    NotesTableSort, NotesTableClearSort, NotesTableFilter, NotesTableReconcile, NotesTableRestore, NotesTableSetGraph, NotesTableRenderGraph, NotesTableDisposeAll,
    GetNotesStructViewMaxSizeKB,
    GetNotesContext, GetNotesFilesSnapshot, SaveFile, SaveBinaryFile, DeleteFile, RenameFile,
    RunNote, RunFunction, StopNote, SendIpc, SendToTerminal, WatchNotesFiles,
    GetNotesOpenTabs, SetNotesOpenTabs,
    GetLanguageDescriptions, GetAllLanguageDescriptions, TerminalCopyImageDataURL,
    ResolveFilePath, GetHyperlinkMenuActions, RunHyperlinkMenuAction,
    DisplayHyperlinkMenu, GetHyperlinkTooltip,
    SaveImageDialog, SaveCodeDialog, WindowPrint, GetClipboardData, CopyFormattedSelection, CopyFormattedTable, SwaggerRequest, NotesKeyPress,
    ShowCommandPalette, GetCurrentProject, GetCurrentGroupName, GetFileMetaMarkdown, AskAI, AskAIImage,
    ShowAISkillsMenu,
    GetAISessionManagement, CreateAISession, SetActiveAISession, DeleteAISession, RenameAISession, DeleteAIHistoryEntry,
    ListAIModelSelections, GetCurrentAIModelSelection, GetAIExecutionLimits, SetCurrentAIModelSelection, SetAIPanelLive,
    ListAIPromptLogs,
    ListAIStreamBlocks, GetAIStreamBlockContent, GetAISessionPrompt, ListAILiveStreamBlocks,
    GetAIToolsList, SetAIToolSubagentAllowed, SetAIToolState, ShowAIToolStateMenu, ShowAIToolSubagentMenu, ResolveAIToolPermission, PromptAIToolPermissionReason, GetAIMcpServers, SetAIMcpServerEnabled, ClearAISessionHistory, ClearAILog,
    ResolveNotesLspLanguage, NotesLspAvailableForRuntime, NotesRecentFiles, ResolveNoteLocation, ComposeNoteLocationPath,
    NotesHistoryPreviousPosition, NotesHistoryNextPosition, NotesHistoryAddPosition, NotesHistoryCurrentPosition, NotesGrepStream,
    GetProjectCache, SetProjectCache,
    GetNotesFindFieldValues, AddNotesFindFieldValue,
    GetDocumentCache, SetDocumentCache, FormatCodeBlock, FormatNotesContent, CompleteSyntax,
    NotesLspOpenDocument, NotesLspSaveDocument,
    NotesLspCloseDocument, NotesLspStopAll, NotesTyposOpenDocument, NotesTyposChangeDocument, NotesTyposCloseDocument,
    NotesLspFormat, GetNotesLanguageTabIndent, GetNotesLanguageReservedWords,
    FilterStrings,
    ResolveAIUserQuestion,
} from './wails-bindings.js';
import { EventsOn, EventsOff, ClipboardSetText, BrowserOpenURL } from './wails-runtime.js';

import {
    showLocalMenu,
} from './popup_menu';
import { initNotesLogPanel } from './notes-log-panel';
import { discardPrintDocument, printDocument } from './notes-print.js';
import { initNotesAIPanel } from './notes-ai-panel';
import { bindSharedTooltipMouseTracking, closeSharedTooltip, showSharedTooltip, updateSharedTooltipPointer } from './shared_tooltip';
import { createHyperlinkTooltip } from './hyperlink-tooltip';
import { captureRenderedElement, captureRenderedElements, createRenderedSelectionCopier } from './copy-formatted';
import { attachVimMode } from './vim-mode';
import { attachSpellCheck, createSpellCheckEngine } from './spellcheck';
import { createNotesDocument, createNotesDocumentRegistry, notesDocumentKey } from './notes_document';
import { normaliseLineEndings } from './notes_document_buffer.js';
import {
    LSP_CHANGE_DEBOUNCE_MS,
    applyLspSpellCheckExclusions,
    clearCurrentFileLspDiagnosticsCache,
    clearVisibleLspDiagnostics,
    flushDocumentChanges,
    closeCurrentTyposDocument,
    closeOpenLspDocument,
    closeOpenLspTooltips,
    commitActiveLspCompletion,
    findReferencesFromEditor,
    formatCurrentLspDocument,
    goToCurrentLspSymbol,
    goToWorkspaceLspSymbol,
    handleMonacoCompletionRequest,
    hideLspCompletion,
    hideLspHoverTooltip,
    isCurrentFileLspEligible,
    lspDiagnosticsStore,
    lspHoverTooltipEl,
    lspSpellCheckExclusions,
    lspTooltipEl,
    moveLspCompletionSelection,
    openCurrentLspDocument,
    openCurrentTyposDocument,
    releaseLspDocument,
    releaseTyposDocument,
    renderLspEditorDecorations,
    requestLspCompletion,
    requestLspCompletionAfterSync,
    resetLspViewState,
    scheduleLspDidChange,
    scheduleLspHover,
    scheduleTyposDidChange,
    showEditorLspOptionsMenu,
    initNotesLsp,
} from './notes_lsp.js';
import { disableNativeTextAssistance, disableNativeTextAssistanceIn } from './text-input.js';
import { createEditorUndoManager } from './editor_undo_manager';
import { createEditorMutationAdapter } from './editor_mutation_adapter';
import { createMonacoAdapter, createMonacoDiffAdapter } from './monaco_adapter';
import { initNotesGitPanel } from './notes-git-panel.js';
import { initMarkdownModal } from './markdown-modal';
import './notes.css';
import notesTemplate from './notes.html?raw';
import { createNotesView } from './notes_view.js';
import { closeLayoutView, createLayout, layoutViewIds, neighbourLayoutView, splitLayoutView } from './notes_layout.js';
import { bindNotesLayoutDividers, renderNotesLayout } from './notes_layout_dom.js';

import { marked } from "marked";
import hljs from "highlight.js/lib/common";
import YAML from 'yaml';

const FIND_FILES_SEARCH_DEBOUNCE_MS = 320;
const FIND_FILES_RENDER_PAGE_SIZE = 50;
const FIND_FILES_VIRTUAL_ROW_HEIGHT = 74;

// Maximum file size (in KB) for the JSON/YAML structured View mode. Files above
// this are shown a "file too large" message instead of the rendered tree.
// Overwritten on startup by GetNotesStructViewMaxSizeKB().
let notesStructViewMaxSizeKB = 1024;

// Import additional syntax highlighting languages (not in common bundle)
import lang1c from "highlight.js/lib/languages/1c";
import abnf from "highlight.js/lib/languages/abnf";
import accesslog from "highlight.js/lib/languages/accesslog";
import actionscript from "highlight.js/lib/languages/actionscript";
import ada from "highlight.js/lib/languages/ada";
import angelscript from "highlight.js/lib/languages/angelscript";
import apache from "highlight.js/lib/languages/apache";
import applescript from "highlight.js/lib/languages/applescript";
import arcade from "highlight.js/lib/languages/arcade";
import arduino from "highlight.js/lib/languages/arduino";
import armasm from "highlight.js/lib/languages/armasm";
import asciidoc from "highlight.js/lib/languages/asciidoc";
import aspectj from "highlight.js/lib/languages/aspectj";
import autohotkey from "highlight.js/lib/languages/autohotkey";
import autoit from "highlight.js/lib/languages/autoit";
import avrasm from "highlight.js/lib/languages/avrasm";
import awk from "highlight.js/lib/languages/awk";
import axapta from "highlight.js/lib/languages/axapta";
import basic from "highlight.js/lib/languages/basic";
import bnf from "highlight.js/lib/languages/bnf";
import brainfuck from "highlight.js/lib/languages/brainfuck";
import cal from "highlight.js/lib/languages/cal";
import capnproto from "highlight.js/lib/languages/capnproto";
import ceylon from "highlight.js/lib/languages/ceylon";
import clean from "highlight.js/lib/languages/clean";
import clojure from "highlight.js/lib/languages/clojure";
import clojureRepl from "highlight.js/lib/languages/clojure-repl";
import cmake from "highlight.js/lib/languages/cmake";
import coffeescript from "highlight.js/lib/languages/coffeescript";
import coq from "highlight.js/lib/languages/coq";
import cos from "highlight.js/lib/languages/cos";
import crmsh from "highlight.js/lib/languages/crmsh";
import crystal from "highlight.js/lib/languages/crystal";
import csp from "highlight.js/lib/languages/csp";
import d from "highlight.js/lib/languages/d";
import dart from "highlight.js/lib/languages/dart";
import delphi from "highlight.js/lib/languages/delphi";
import django from "highlight.js/lib/languages/django";
import dns from "highlight.js/lib/languages/dns";
import dockerfile from "highlight.js/lib/languages/dockerfile";
import dos from "highlight.js/lib/languages/dos";
import dsconfig from "highlight.js/lib/languages/dsconfig";
import dts from "highlight.js/lib/languages/dts";
import dust from "highlight.js/lib/languages/dust";
import ebnf from "highlight.js/lib/languages/ebnf";
import elixir from "highlight.js/lib/languages/elixir";
import elm from "highlight.js/lib/languages/elm";
import erb from "highlight.js/lib/languages/erb";
import erlang from "highlight.js/lib/languages/erlang";
import erlangRepl from "highlight.js/lib/languages/erlang-repl";
import excel from "highlight.js/lib/languages/excel";
import fix from "highlight.js/lib/languages/fix";
import flix from "highlight.js/lib/languages/flix";
import fortran from "highlight.js/lib/languages/fortran";
import fsharp from "highlight.js/lib/languages/fsharp";
import gams from "highlight.js/lib/languages/gams";
import gauss from "highlight.js/lib/languages/gauss";
import gcode from "highlight.js/lib/languages/gcode";
import gherkin from "highlight.js/lib/languages/gherkin";
import glsl from "highlight.js/lib/languages/glsl";
import gml from "highlight.js/lib/languages/gml";
import golo from "highlight.js/lib/languages/golo";
import gradle from "highlight.js/lib/languages/gradle";
import groovy from "highlight.js/lib/languages/groovy";
import haml from "highlight.js/lib/languages/haml";
import handlebars from "highlight.js/lib/languages/handlebars";
import haskell from "highlight.js/lib/languages/haskell";
import haxe from "highlight.js/lib/languages/haxe";
import hsp from "highlight.js/lib/languages/hsp";
import http from "highlight.js/lib/languages/http";
import hy from "highlight.js/lib/languages/hy";
import inform7 from "highlight.js/lib/languages/inform7";
import irpf90 from "highlight.js/lib/languages/irpf90";
import isbl from "highlight.js/lib/languages/isbl";
import jbossCli from "highlight.js/lib/languages/jboss-cli";
import julia from "highlight.js/lib/languages/julia";
import juliaRepl from "highlight.js/lib/languages/julia-repl";
import lasso from "highlight.js/lib/languages/lasso";
import latex from "highlight.js/lib/languages/latex";
import ldif from "highlight.js/lib/languages/ldif";
import leaf from "highlight.js/lib/languages/leaf";
import lisp from "highlight.js/lib/languages/lisp";
import livecodeserver from "highlight.js/lib/languages/livecodeserver";
import livescript from "highlight.js/lib/languages/livescript";
import llvm from "highlight.js/lib/languages/llvm";
import lsl from "highlight.js/lib/languages/lsl";
import mathematica from "highlight.js/lib/languages/mathematica";
import matlab from "highlight.js/lib/languages/matlab";
import maxima from "highlight.js/lib/languages/maxima";
import mel from "highlight.js/lib/languages/mel";
import mercury from "highlight.js/lib/languages/mercury";
import mipsasm from "highlight.js/lib/languages/mipsasm";
import mizar from "highlight.js/lib/languages/mizar";
import mojolicious from "highlight.js/lib/languages/mojolicious";
import monkey from "highlight.js/lib/languages/monkey";
import moonscript from "highlight.js/lib/languages/moonscript";
import n1ql from "highlight.js/lib/languages/n1ql";
import nestedtext from "highlight.js/lib/languages/nestedtext";
import nginx from "highlight.js/lib/languages/nginx";
import nim from "highlight.js/lib/languages/nim";
import nix from "highlight.js/lib/languages/nix";
import nodeRepl from "highlight.js/lib/languages/node-repl";
import nsis from "highlight.js/lib/languages/nsis";
import ocaml from "highlight.js/lib/languages/ocaml";
import openscad from "highlight.js/lib/languages/openscad";
import oxygene from "highlight.js/lib/languages/oxygene";
import parser3 from "highlight.js/lib/languages/parser3";
import pf from "highlight.js/lib/languages/pf";
import pgsql from "highlight.js/lib/languages/pgsql";
import pony from "highlight.js/lib/languages/pony";
import powershell from "highlight.js/lib/languages/powershell";
import processing from "highlight.js/lib/languages/processing";
import profile from "highlight.js/lib/languages/profile";
import prolog from "highlight.js/lib/languages/prolog";
import properties from "highlight.js/lib/languages/properties";
import protobuf from "highlight.js/lib/languages/protobuf";
import puppet from "highlight.js/lib/languages/puppet";
import purebasic from "highlight.js/lib/languages/purebasic";
import q from "highlight.js/lib/languages/q";
import qml from "highlight.js/lib/languages/qml";
import reasonml from "highlight.js/lib/languages/reasonml";
import rib from "highlight.js/lib/languages/rib";
import roboconf from "highlight.js/lib/languages/roboconf";
import routeros from "highlight.js/lib/languages/routeros";
import rsl from "highlight.js/lib/languages/rsl";
import ruleslanguage from "highlight.js/lib/languages/ruleslanguage";
import sas from "highlight.js/lib/languages/sas";
import scala from "highlight.js/lib/languages/scala";
import scheme from "highlight.js/lib/languages/scheme";
import scilab from "highlight.js/lib/languages/scilab";
import smali from "highlight.js/lib/languages/smali";
import smalltalk from "highlight.js/lib/languages/smalltalk";
import sml from "highlight.js/lib/languages/sml";
import sqf from "highlight.js/lib/languages/sqf";
import stan from "highlight.js/lib/languages/stan";
import stata from "highlight.js/lib/languages/stata";
import step21 from "highlight.js/lib/languages/step21";
import stylus from "highlight.js/lib/languages/stylus";
import subunit from "highlight.js/lib/languages/subunit";
import taggerscript from "highlight.js/lib/languages/taggerscript";
import tap from "highlight.js/lib/languages/tap";
import tcl from "highlight.js/lib/languages/tcl";
import thrift from "highlight.js/lib/languages/thrift";
import murex from "highlight-js-murex";
import terraform from "highlight-js-terraform";
import tp from "highlight.js/lib/languages/tp";
import twig from "highlight.js/lib/languages/twig";
import vala from "highlight.js/lib/languages/vala";
import vbscript from "highlight.js/lib/languages/vbscript";
import vbscriptHtml from "highlight.js/lib/languages/vbscript-html";
import verilog from "highlight.js/lib/languages/verilog";
import vhdl from "highlight.js/lib/languages/vhdl";
import vim from "highlight.js/lib/languages/vim";
import wren from "highlight.js/lib/languages/wren";
import x86asm from "highlight.js/lib/languages/x86asm";
import xl from "highlight.js/lib/languages/xl";
import xquery from "highlight.js/lib/languages/xquery";
import zephir from "highlight.js/lib/languages/zephir";

// Register all languages with highlight.js
hljs.registerLanguage('1c', lang1c);
hljs.registerLanguage('abnf', abnf);
hljs.registerLanguage('accesslog', accesslog);
hljs.registerLanguage('actionscript', actionscript);
hljs.registerLanguage('ada', ada);
hljs.registerLanguage('angelscript', angelscript);
hljs.registerLanguage('apache', apache);
hljs.registerLanguage('applescript', applescript);
hljs.registerLanguage('arcade', arcade);
hljs.registerLanguage('arduino', arduino);
hljs.registerLanguage('armasm', armasm);
hljs.registerLanguage('asciidoc', asciidoc);
hljs.registerLanguage('aspectj', aspectj);
hljs.registerLanguage('autohotkey', autohotkey);
hljs.registerLanguage('autoit', autoit);
hljs.registerLanguage('avrasm', avrasm);
hljs.registerLanguage('awk', awk);
hljs.registerLanguage('axapta', axapta);
hljs.registerLanguage('basic', basic);
hljs.registerLanguage('bnf', bnf);
hljs.registerLanguage('brainfuck', brainfuck);
hljs.registerLanguage('cal', cal);
hljs.registerLanguage('capnproto', capnproto);
hljs.registerLanguage('ceylon', ceylon);
hljs.registerLanguage('clean', clean);
hljs.registerLanguage('clojure', clojure);
hljs.registerLanguage('clojure-repl', clojureRepl);
hljs.registerLanguage('cmake', cmake);
hljs.registerLanguage('coffeescript', coffeescript);
hljs.registerLanguage('coq', coq);
hljs.registerLanguage('cos', cos);
hljs.registerLanguage('crmsh', crmsh);
hljs.registerLanguage('crystal', crystal);
hljs.registerLanguage('csp', csp);
hljs.registerLanguage('d', d);
hljs.registerLanguage('dart', dart);
hljs.registerLanguage('delphi', delphi);
hljs.registerLanguage('django', django);
hljs.registerLanguage('dns', dns);
hljs.registerLanguage('dockerfile', dockerfile);
hljs.registerLanguage('dos', dos);
hljs.registerLanguage('dsconfig', dsconfig);
hljs.registerLanguage('dts', dts);
hljs.registerLanguage('dust', dust);
hljs.registerLanguage('ebnf', ebnf);
hljs.registerLanguage('elixir', elixir);
hljs.registerLanguage('elm', elm);
hljs.registerLanguage('erb', erb);
hljs.registerLanguage('erlang', erlang);
hljs.registerLanguage('erlang-repl', erlangRepl);
hljs.registerLanguage('excel', excel);
hljs.registerLanguage('fix', fix);
hljs.registerLanguage('flix', flix);
hljs.registerLanguage('fortran', fortran);
hljs.registerLanguage('fsharp', fsharp);
hljs.registerLanguage('gams', gams);
hljs.registerLanguage('gauss', gauss);
hljs.registerLanguage('gcode', gcode);
hljs.registerLanguage('gherkin', gherkin);
hljs.registerLanguage('glsl', glsl);
hljs.registerLanguage('gml', gml);
hljs.registerLanguage('golo', golo);
hljs.registerLanguage('gradle', gradle);
hljs.registerLanguage('groovy', groovy);
hljs.registerLanguage('haml', haml);
hljs.registerLanguage('handlebars', handlebars);
hljs.registerLanguage('haskell', haskell);
hljs.registerLanguage('haxe', haxe);
hljs.registerLanguage('hsp', hsp);
hljs.registerLanguage('http', http);
hljs.registerLanguage('hy', hy);
hljs.registerLanguage('inform7', inform7);
hljs.registerLanguage('irpf90', irpf90);
hljs.registerLanguage('isbl', isbl);
hljs.registerLanguage('jboss-cli', jbossCli);
hljs.registerLanguage('julia', julia);
hljs.registerLanguage('julia-repl', juliaRepl);
hljs.registerLanguage('lasso', lasso);
hljs.registerLanguage('latex', latex);
hljs.registerLanguage('ldif', ldif);
hljs.registerLanguage('leaf', leaf);
hljs.registerLanguage('lisp', lisp);
hljs.registerLanguage('livecodeserver', livecodeserver);
hljs.registerLanguage('livescript', livescript);
hljs.registerLanguage('llvm', llvm);
hljs.registerLanguage('lsl', lsl);
hljs.registerLanguage('mathematica', mathematica);
hljs.registerLanguage('matlab', matlab);
hljs.registerLanguage('maxima', maxima);
hljs.registerLanguage('mel', mel);
hljs.registerLanguage('mercury', mercury);
hljs.registerLanguage('mipsasm', mipsasm);
hljs.registerLanguage('mizar', mizar);
hljs.registerLanguage('mojolicious', mojolicious);
hljs.registerLanguage('monkey', monkey);
hljs.registerLanguage('moonscript', moonscript);
hljs.registerLanguage('n1ql', n1ql);
hljs.registerLanguage('nestedtext', nestedtext);
hljs.registerLanguage('nginx', nginx);
hljs.registerLanguage('nim', nim);
hljs.registerLanguage('nix', nix);
hljs.registerLanguage('node-repl', nodeRepl);
hljs.registerLanguage('nsis', nsis);
hljs.registerLanguage('ocaml', ocaml);
hljs.registerLanguage('openscad', openscad);
hljs.registerLanguage('oxygene', oxygene);
hljs.registerLanguage('parser3', parser3);
hljs.registerLanguage('pf', pf);
hljs.registerLanguage('pgsql', pgsql);
hljs.registerLanguage('pony', pony);
hljs.registerLanguage('powershell', powershell);
hljs.registerLanguage('processing', processing);
hljs.registerLanguage('profile', profile);
hljs.registerLanguage('prolog', prolog);
hljs.registerLanguage('properties', properties);
hljs.registerLanguage('protobuf', protobuf);
hljs.registerLanguage('puppet', puppet);
hljs.registerLanguage('purebasic', purebasic);
hljs.registerLanguage('q', q);
hljs.registerLanguage('qml', qml);
hljs.registerLanguage('reasonml', reasonml);
hljs.registerLanguage('rib', rib);
hljs.registerLanguage('roboconf', roboconf);
hljs.registerLanguage('routeros', routeros);
hljs.registerLanguage('rsl', rsl);
hljs.registerLanguage('ruleslanguage', ruleslanguage);
hljs.registerLanguage('sas', sas);
hljs.registerLanguage('scala', scala);
hljs.registerLanguage('scheme', scheme);
hljs.registerLanguage('scilab', scilab);
hljs.registerLanguage('smali', smali);
hljs.registerLanguage('smalltalk', smalltalk);
hljs.registerLanguage('sml', sml);
hljs.registerLanguage('sqf', sqf);
hljs.registerLanguage('stan', stan);
hljs.registerLanguage('stata', stata);
hljs.registerLanguage('step21', step21);
hljs.registerLanguage('stylus', stylus);
hljs.registerLanguage('subunit', subunit);
hljs.registerLanguage('taggerscript', taggerscript);
hljs.registerLanguage('tap', tap);
hljs.registerLanguage('tcl', tcl);
hljs.registerLanguage('thrift', thrift);
hljs.registerLanguage('murex', murex);
hljs.registerLanguage('terraform', terraform);
hljs.registerLanguage('tp', tp);
hljs.registerLanguage('twig', twig);
hljs.registerLanguage('vala', vala);
hljs.registerLanguage('vbscript', vbscript);
hljs.registerLanguage('vbscript-html', vbscriptHtml);
hljs.registerLanguage('verilog', verilog);
hljs.registerLanguage('vhdl', vhdl);
hljs.registerLanguage('vim', vim);
hljs.registerLanguage('wren', wren);
hljs.registerLanguage('x86asm', x86asm);
hljs.registerLanguage('xl', xl);
hljs.registerLanguage('xquery', xquery);
hljs.registerLanguage('zephir', zephir);

import { autoHyperlink, configureMarked, processMarkdownContainer, enableFullscreenImages, processLinks, notesAssetURL } from './markdown-utils.js';
import { showFullscreenImageOverlay } from './fullscreen-image-overlay.js';
import { getScrollbarStyles, getMarkdownContentStyles, getHighlightJsTheme, getCheckboxStyles, getMarkdownBaseTextSizeStyles, getSwaggerUIStyles, DARKEN_BACKGROUND_OVERLAY } from './style-utils.js';
import { 
    isStructuredDataFile, hasSwaggerKey, parseSwaggerSpec, generateRequestBuilderHTML, generateResponseHTML,
    extractPaths, generateEndpointListHTML, buildRequestUrl, generateLiveResponseHTML, escapeInfoText
} from './swagger-utils.js';
import { attachJsonViewerEditHandler, collapseJsonViewerSubtree, expandJsonViewerSubtree, renderJsonViewer, startJsonViewerKeyEdit } from './json-viewer.js';
import { getStructuredEditor, yamlEditor } from './structured-editors.js';
import { getHexDumpStyles, renderHexDump } from './hex-viewer.js';
import { updateMarkdownTableOfContentsText } from './markdown_toc.js';
import {
    evaluateTableFormula,
    isTableFormula,
    getCellReference,
    parseTableFunctionCall,
    resolveTableFunctionArg,
    resolveTableFunctionArgs,
    resolveTableFunctionArgsAsync,
} from './table-expressions.js';
import { createAIPipelineFormatter } from './ai_pipeline_formatter.js';
import { initLineNavigationKeys } from './line-navigation.js';
import { restoreLayoutRatio, saveLayoutRatio } from './layout-sizing.js';

const CONTEXT_ICON_COPY = 0xf0c5;
const CONTEXT_ICON_PASTE = 0xf0ea;
const CONTEXT_ICON_FIND = 0xf002;
const CONTEXT_ICON_PRINT = 0xf02f;
const CONTEXT_ICON_CHECKBOX = 0xf14a;
const CONTEXT_ICON_TICK = 0xf00c;
const CONTEXT_ICON_CODE = 0xf121;
const CONTEXT_ICON_TABLE = 0xf0ce;
const CONTEXT_ICON_EDIT = 0xf044;
const CONTEXT_ICON_DELETE = 0xf2ed;
const CONTEXT_ICON_ASK_AI = 0xf544;
const CONTEXT_ICON_EXPAND_ALL = 0xf0fe;
const CONTEXT_ICON_COLLAPSE_ALL = 0xf146;
const CONTEXT_ICON_ADD = 0xf067;
const CONTEXT_ICON_SEND_TO_TERMINAL = 0xf120;

// Inject cell reference CSS if not present
function ensureCellRefStyle() {
    return;
}

const IS_WINDOWS = typeof navigator !== 'undefined' && (
    /Windows/i.test(navigator.userAgent || '') ||
    /Win/i.test(navigator.platform || '')
);
const PRIMARY_PATH_SEPARATOR = IS_WINDOWS ? '\\' : '/';
const FALLBACK_PATH_SEPARATOR = IS_WINDOWS ? '/' : '\\';

const app = document.getElementById('notes-pane') || document.getElementById('app') || (() => {
    const root = document.createElement('div');
    root.id = 'app';
    document.body.appendChild(root);
    return root;
})();

document.title = 'Notes';

initLineNavigationKeys(document);

app.innerHTML = notesTemplate;

// Every open view, by ID, and the layout tree that arranges them in the editor
// area (ADR 0100 decision 2).
const notesViews = new Map();
const firstNotesView = createNotesView();
notesViews.set(firstNotesView.id, firstNotesView);
let notesLayout = createLayout(firstNotesView.id);
let notesFocusedView = firstNotesView;
firstNotesView.root.dataset.focused = 'true';

function renderNotesViewsLayout() {
    renderNotesLayout(document.getElementById('notes-views'), notesLayout, (id) => notesViews.get(id)?.root);
}
renderNotesViewsLayout();
bindNotesLayoutDividers(document.getElementById('notes-views'), {
    getLayout: () => notesLayout,
    setLayout: (next) => {
        notesLayout = next;
        renderNotesViewsLayout();
    },
});

// The view that global chrome (status bar, Tools panel, LSP overlays, the
// toolbar) follows. Its shown document is the active NotesDocument, and
// rendering goes to its parts. Other views keep what they last showed; their
// editors still show live edits, as views of one document share its model.
function focusedView() {
    return notesFocusedView;
}

function notesViewList() {
    return layoutViewIds(notesLayout).map((id) => notesViews.get(id)).filter(Boolean);
}

// The view whose tabs are remembered per workspace, until the whole layout is
// (ADR 0100 Stage 8).
function primaryNotesView() {
    return notesViewList()[0] || notesFocusedView;
}

const notesToolbar = document.getElementById('notes-toolbar');
firstNotesView.parts.toolbarSlot.replaceWith(notesToolbar);

const elements = {
    appRoot: app,
    title: document.getElementById('notes-title'),
    currentFileTitle: document.getElementById('notes-current-file'),
    list: document.getElementById('notes-list'),
    gitCommitList: document.getElementById('notes-commits-list'),
    gitFilterRow: document.getElementById('notes-tools-git-filter-row'),
    gitFileFilter: document.getElementById('notes-tools-git-file-filter'),
    gitFilenameSearchRow: document.getElementById('notes-commit-filename-search-row'),
    gitFilenameSearch: document.getElementById('notes-commit-filename-search'),
    listFilter: document.getElementById('notes-list-filter'),
    listFilterClear: document.getElementById('notes-list-filter-clear'),
    get editorShell() { return focusedView().parts.editorShell; },
    get monacoEditor() { return focusedView().parts.monacoEditor; },
    get preview() { return focusedView().parts.preview; },
    get htmlViewWrap() { return focusedView().parts.htmlViewWrap; },
    get htmlViewFrame() { return focusedView().parts.htmlViewFrame; },
    get jupyter() { return focusedView().parts.jupyter; },
    status: document.getElementById('notes-status'),
    newFile: document.getElementById('notes-new'),
    historyPrev: document.getElementById('notes-history-prev'),
    historyNext: document.getElementById('notes-history-next'),
    get tabEditor() { return focusedView().parts.tabEditor; },
    get tabHex() { return focusedView().parts.tabHex; },
    get tabViewer() { return focusedView().parts.tabViewer; },
    get tabJupyter() { return focusedView().parts.tabJupyter; },
    get tabSwaggerView() { return focusedView().parts.tabSwaggerView; },
    get tabSwaggerEdit() { return focusedView().parts.tabSwaggerEdit; },
    get tabSwaggerRun() { return focusedView().parts.tabSwaggerRun; },
    get tabImageView() { return focusedView().parts.tabImageView; },
    get tabMeta() { return focusedView().parts.tabMeta; },
    get tabs() { return focusedView().parts.tabs; },
    get tabDiff() { return focusedView().parts.tabDiff; },
    get tabCsvView() { return focusedView().parts.tabCsvView; },
    get tabCsvEdit() { return focusedView().parts.tabCsvEdit; },
    get tabCsvRun() { return focusedView().parts.tabCsvRun; },
    get editorWrap() { return focusedView().parts.editorWrap; },
    get gitDiffWrap() { return focusedView().parts.gitDiffWrap; },
    get gitDiffEditor() { return focusedView().parts.gitDiffEditor; },
    get gitDiffMessage() { return focusedView().parts.gitDiffMessage; },
    get hexWrap() { return focusedView().parts.hexWrap; },
    get hex() { return focusedView().parts.hex; },
    get previewWrap() { return focusedView().parts.previewWrap; },
    get jupyterWrap() { return focusedView().parts.jupyterWrap; },
    get imageViewWrap() { return focusedView().parts.imageViewWrap; },
    get imageViewImg() { return focusedView().parts.imageViewImg; },
    get metaWrap() { return focusedView().parts.metaWrap; },
    get meta() { return focusedView().parts.meta; },
    get csvViewWrap() { return focusedView().parts.csvViewWrap; },
    get csvView() { return focusedView().parts.csvView; },
    get swaggerViewWrap() { return focusedView().parts.swaggerViewWrap; },
    get swaggerRunWrap() { return focusedView().parts.swaggerRunWrap; },
    get swaggerView() { return focusedView().parts.swaggerView; },
    get swaggerEndpoints() { return focusedView().parts.swaggerEndpoints; },
    get swaggerRequestBuilder() { return focusedView().parts.swaggerRequestBuilder; },
    get swaggerResponse() { return focusedView().parts.swaggerResponse; },
    modal: document.getElementById('notes-modal'),
    modalBody: document.getElementById('notes-modal-body'),
    modalLocationRow: document.getElementById('notes-modal-location-row'),
    modalLocation: document.getElementById('notes-modal-location'),
    modalInput: document.getElementById('notes-modal-input'),
    modalCancel: document.getElementById('notes-modal-cancel'),
    modalCreate: document.getElementById('notes-modal-create'),
    deleteModal: document.getElementById('notes-delete-modal'),
    deleteModalTitle: document.getElementById('notes-delete-modal-title'),
    deleteModalBody: document.getElementById('notes-delete-modal-body'),
    deleteCancel: document.getElementById('notes-delete-cancel'),
    deleteConfirm: document.getElementById('notes-delete-confirm'),
    findInput: document.getElementById('notes-find-input'),
    findInputClear: document.getElementById('notes-find-input-clear'),
    findControls: document.getElementById('notes-find-controls'),
    findCounter: document.getElementById('notes-find-counter'),
    findPrev: document.getElementById('notes-find-prev'),
    findNext: document.getElementById('notes-find-next'),
    findDocOptionCase: document.getElementById('notes-find-doc-option-case'),
    findDocOptionRegex: document.getElementById('notes-find-doc-option-regex'),
    findDocOptionWord: document.getElementById('notes-find-doc-option-word'),
    findFilesInput: document.getElementById('notes-find-files-input'),
    findReferences: document.getElementById('notes-find-references'),
    findFilesClear: document.getElementById('notes-find-files-clear'),
    findFilesResults: document.getElementById('notes-find-files-results'),
    findFilesHeading: document.getElementById('notes-find-files-heading'),
    findOptionCase: document.getElementById('notes-find-option-case'),
    findOptionRegex: document.getElementById('notes-find-option-regex'),
    findOptionWord: document.getElementById('notes-find-option-word'),
    replaceControls: document.getElementById('notes-replace-controls'),
    replaceInput: document.getElementById('notes-replace-input'),
    replaceInputClear: document.getElementById('notes-replace-input-clear'),
    replaceOne: document.getElementById('notes-replace-one'),
    replaceAll: document.getElementById('notes-replace-all'),
    toolsTabs: document.getElementById('notes-tools-tabs'),
    toolsTabFind: document.getElementById('notes-tools-tab-find'),
    toolsTabAI: document.getElementById('notes-tools-tab-ai'),
    toolsTabToC: document.getElementById('notes-tools-tab-toc'),
    toolsTabFrontmatter: document.getElementById('notes-tools-tab-frontmatter'),
    toolsTabLog: document.getElementById('notes-tools-tab-log'),
    toolsTabGit: document.getElementById('notes-tools-tab-git'),
    gitTab: document.getElementById('notes-tools-tab-git'),
    gitPane: document.getElementById('notes-tools-git-pane'),
    gitToolbar: document.getElementById('notes-tools-git-toolbar'),
    gitWorkspace: document.getElementById('notes-tools-git-workspace'),
    gitRemote: document.getElementById('notes-tools-git-remote'),
    gitBranch: document.getElementById('notes-tools-git-branch'),
    gitAheadBehind: document.getElementById('notes-tools-git-ahead-behind'),
    gitCompare: document.getElementById('notes-tools-git-compare'),
    gitCommits: document.getElementById('notes-tools-git-commits'),
    gitCommitToggle: document.getElementById('notes-tools-git-commit-toggle'),
    gitCommitRow: document.getElementById('notes-tools-git-commit-row'),
    gitCommitMessage: document.getElementById('notes-tools-git-commit-message'),
    gitCommitSubmit: document.getElementById('notes-tools-git-commit-submit'),
    gitPush: document.getElementById('notes-tools-git-push'),
    gitStageAll: document.getElementById('notes-tools-git-stage-all'),
    gitBranches: document.getElementById('notes-tools-git-branches'),
    gitTags: document.getElementById('notes-tools-git-tags'),
    gitRefresh: document.getElementById('notes-tools-git-refresh'),
    gitFileList: document.getElementById('notes-tools-git-files'),
    toolsFindPane: document.getElementById('notes-tools-find-pane'),
    toolsAIPane: document.getElementById('notes-tools-ai-pane'),
    toolsToCPane: document.getElementById('notes-tools-toc-pane'),
    toolsFrontmatterPane: document.getElementById('notes-tools-frontmatter-pane'),
    toolsLogPane: document.getElementById('notes-tools-log-pane'),
    toolsToC: document.getElementById('notes-tools-toc'),
    toolsFrontmatter: document.getElementById('notes-tools-frontmatter'),
    toolsPanel: document.getElementById('notes-tools-panel'),
    toolsMinimize: document.getElementById('notes-tools-minimize'),
    toolsAIPromptJump: document.getElementById('notes-tools-ai-prompt-jump'),
    aiSettingsModelPicker: document.getElementById('notes-ai-settings-model-picker'),
    aiSettingsExecutionLimits: document.getElementById('notes-ai-settings-execution-limits'),
    toolsAIMaximize: document.getElementById('notes-tools-ai-maximize'),
    toolsAIAsk: document.getElementById('notes-tools-ai-ask'),
    toolsAISkills: document.getElementById('notes-tools-ai-skills'),
    toolsAISettings: document.getElementById('notes-tools-ai-settings'),
    toolsAISession: document.getElementById('notes-tools-ai-session'),
    aiOutput: document.getElementById('notes-ai-output'),
    aiScrollBottom: document.getElementById('notes-ai-scroll-bottom'),
    aiSettingsModal: document.getElementById('notes-ai-settings-modal'),
    aiSettingsSessionsList: document.getElementById('notes-ai-settings-sessions-list'),
    aiSettingsSessionNew: document.getElementById('notes-ai-settings-session-new'),
    aiSettingsHistoryList: document.getElementById('notes-ai-settings-history-list'),
    aiSettingsToolsList: document.getElementById('notes-ai-settings-tools-list'),
    aiSettingsToolsFilter: document.getElementById('notes-ai-settings-tools-filter'),
    aiSettingsMcpList: document.getElementById('notes-ai-settings-mcp-list'),
    aiSettingsMcpFilter: document.getElementById('notes-ai-settings-mcp-filter'),
    aiToolMetaModal: document.getElementById('notes-ai-tool-meta-modal'),
    aiToolMetaCard: document.getElementById('notes-ai-tool-meta-card'),
    aiToolMetaContent: document.getElementById('notes-ai-tool-meta-content'),
    aiSettingsHistoryClear: document.getElementById('notes-ai-settings-history-clear'),
    logOutput: document.getElementById('notes-log-output'),
    toolsLogMaximize: document.getElementById('notes-tools-log-maximize'),
    toolsLogTimestamp: document.getElementById('notes-tools-log-timestamp'),
    toolsLogDebug: document.getElementById('notes-tools-log-debug'),
    toolsLogTrace: document.getElementById('notes-tools-log-trace'),
    toolsLogError: document.getElementById('notes-tools-log-error'),
    toolsLogWarn: document.getElementById('notes-tools-log-warn'),
    toolsLogInfo: document.getElementById('notes-tools-log-info'),
    toolsLogWordwrap: document.getElementById('notes-tools-log-wordwrap'),
    toolsLogCopy: document.getElementById('notes-tools-log-copy'),
    toolsLogSelectAll: document.getElementById('notes-tools-log-select-all'),
    toolsLogDeselect: document.getElementById('notes-tools-log-deselect'),
    toolsLogClear: document.getElementById('notes-tools-log-clear'),
    logScrollBottom: document.getElementById('notes-log-scroll-bottom'),
    toolsRestore: document.getElementById('notes-tools-restore')
};

const state = {
    files: [],
    mutatingFile: '',
    // A folder being renamed; changes to files under it are not external.
    mutatingFolder: '',
    suspendDocumentCacheSave: false,
    viewTainted: false,
    renderTimer: null,
    viewMode: 'viewer',
    renamingFile: null,
    renamingFolder: false,
    historyPositionTimer: null,
    pendingNewFile: null,
    textPromptHandler: null,
    graphPromptHandler: null,
    graphPromptSubmit: null,
    graphPromptSpellCheck: null,
    gitWorktreePrompt: null,
    gitBranchPromptHandler: null,
    deletingFile: null,
    deleteConfirmAction: null,
    findDocOptions: {
        caseSensitive: false,
        regex: false,
        wholeWord: false,
    },
    findFilesQuery: '',
    findFilesResults: [],
    findFilesMode: 'grep',
    findFilesSource: '',
    findFilesReferenceSymbol: '',
    findFilesBusy: false,
    findFilesLastExecutedSignature: '',
    findFilesError: '',
    findFilesTimer: null,
    findFilesSeq: 0,
    findFilesSelectedKey: '',
    findFilesStreamHandlers: null,
    findFilesRenderQueued: false,
    findFilesVirtualStart: 0,
    findFilesScrollRenderQueued: false,
    findOptions: {
        caseSensitive: false,
        regex: false,
        wholeWord: false,
    },
    fileFilterQuery: '',
    filteredFiles: null,
    expandedCategories: {
        '$GLOBAL': true,
        '$NOTES': true,
        '$PROJECT': true,
        '$HISTORY': false,
    },
    expandedFolders: {},
    currentProjectRoot: '',      // current project root, used to index FileListCollapsed map
    lspActiveBlockId: null,
    lspActiveBlockEditor: null,
    swaggerSpec: null,
    swaggerRunAvailable: false,
    swaggerViewTooLarge: false,
    // True when the JSON/YAML View tab DOM already reflects the current file contents.
    // Reset to false on file load, on any editor input that changes the source,
    // and on structured-viewer edits.
    swaggerViewCurrent: false,
    swaggerSelectedEndpoint: null,
    swaggerEndpointFilter: '',
    frontmatter: null,  // parsed markdown frontmatter object (null when absent)
    editorLanguage: '',
    fileMetaMarkdown: '',
    hexSourceType: '',
    hexSourceValue: '',
    hexSourceFile: '',
    hexSourceOptions: null,
    hexRenderedFile: '',
    hexLoadingPromise: null,
    markdownWrapMode: false,  // New: track word wrap mode for markdown files
    markdownTableWordWrapMode: true,  // track table word wrap mode for View/Run modes
    aiModelSelections: [],
    aiCurrentModelSelection: '',
    // Whether the panel is following live output or showing a historical prompt.
    aiPanelLive: true,
    aiRunActive: false,
    aiSelectedPrompt: null,
    // AI logs are loaded lazily: on workspace switch we mark them pending and
    // only actually fetch/render once the AI tab is selected, so the (possibly
    // large) log never blocks file/workspace switching.
    aiSessionCachePending: false,
    aiSessionCachePendingWorkspace: '',
    aiSessionManagement: {
        activeSessionId: 0,
        sessions: [],
        history: [],
    },
    aiToolsList: [],
    aiMcpServersList: [],
    aiPromptJumpTargets: [],
    aiStickToBottom: true,
    currentWorkspaceName: '',
    currentWorkspaceKey: '',
    lspHoverTimer: null,
    lspHoverLastKey: '',
    lspHoverMouseX: 0,
    lspHoverMouseY: 0,
    lspCompletionItems: [],
    lspCompletionIndex: 0,
    lspCompletionVisible: false,
    lastEditorInputAt: 0,
};

const AI_BOTTOM_THRESHOLD_PX = 24;
// How long the bottom-chase keeps running after the last request for it.
const AI_BOTTOM_CHASE_MS = 400;
const NOTE_LOCATIONS = ['$GLOBAL', '$NOTES', '$PROJECT'];

// Each NotesDocument has its own buffer and Monaco model (ADR 0100 decision
// 2). mainDocument is the active document's buffer; its change listeners hear
// only edits to the active document.
const activeDocumentChangeListeners = new Set();
const mainDocument = {
    getValue: () => activeNotesDocument.buffer.getValue(),
    getSelection: () => activeNotesDocument.buffer.getSelection(),
    setSelection: (start, end) => activeNotesDocument.buffer.setSelection(start, end),
    setValue: (text) => activeNotesDocument.buffer.setValue(text),
    replaceRange: (start, end, insert, options) => activeNotesDocument.buffer.replaceRange(start, end, insert, options),
    replaceAll: (text, options) => activeNotesDocument.buffer.replaceAll(text, options),
    onChange(listener) {
        activeDocumentChangeListeners.add(listener);
        return () => activeDocumentChangeListeners.delete(listener);
    },
};

function newNotesDocument(fields = {}) {
    const doc = createNotesDocument(fields);
    doc.buffer.onChange((event) => {
        if (doc !== activeNotesDocument) {
            return;
        }
        for (const listener of [...activeDocumentChangeListeners]) {
            listener(event);
        }
    });
    return doc;
}

// Open documents (ADR 0100). A document stays open while a view has a tab for
// it. With no file open, the active document is a detached placeholder so its
// fields always have a home.
const notesDocuments = createNotesDocumentRegistry({
    create: newNotesDocument,
    onClose: releaseNotesDocument,
});
let activeNotesDocument = newNotesDocument();

// Shows a document in a view's editor, creating the document's model the first
// time. The buffer attaches to the model, so its edits are Monaco edits on the
// document's own undo stack whether or not the document is shown.
function showDocumentInView(view, doc) {
    if (!view.editor) {
        return;
    }
    if (!doc.editorModel) {
        doc.editorModel = view.editor.createDocumentModel({
            value: doc.buffer.getValue(),
            language: state.editorLanguage || 'plaintext',
            onChange: () => doc.buffer.handleViewChange('monaco'),
        });
        doc.buffer.attachView(doc.editorModel);
    }
    view.editor.showModel(doc.editorModel);
}

// Placeholders are not in the registry, so they are released here.
function retireNotesDocument(doc) {
    if (isNotesDocumentOpen(doc)) {
        notesDocuments.close(doc);
    } else {
        doc.editorModel?.dispose();
        doc.editorModel = null;
    }
}

function findNotesDocument(key) {
    return notesDocuments.list().find((doc) => doc.key === key) || null;
}

function isNotesDocumentInTab(doc) {
    return Boolean(doc?.key) && notesViewList().some((view) => view.tabs.includes(doc.key));
}

// Activating an open document opens or focuses its tab, next to the tab that
// was active. The previous document stays open if it still has a tab.
function setActiveNotesDocument(next) {
    const previous = activeNotesDocument;
    activeNotesDocument = next;
    const view = focusedView();
    if (isNotesDocumentOpen(next)) {
        view.addTab(next.key, previous.key);
    }
    showDocumentInView(view, next);
    if (!isNotesDocumentInTab(previous)) {
        retireNotesDocument(previous);
    } else if (previous !== next) {
        void releaseTyposDocument(previous);
    }
    renderNotesTabs(view);
}

// The backend watches the files of open documents for changes made outside
// Notes. Changes to the open set are sent once per task, numbered so the
// backend can ignore an older set that arrives late.
let notesWatchGeneration = 0;
let notesWatchSyncQueued = false;

function syncWatchedNotesFiles() {
    if (notesWatchSyncQueued) {
        return;
    }
    notesWatchSyncQueued = true;
    queueMicrotask(() => {
        notesWatchSyncQueued = false;
        const files = notesDocuments.list().map(({ key, path, project }) => ({ key, path, project }));
        Promise.resolve(WatchNotesFiles(++notesWatchGeneration, files)).catch((err) => {
            console.error('Failed to watch open notes:', err);
        });
    });
}

// Open tabs are remembered per workspace (ADR 0100 question 8). Saving waits
// until the workspace's tabs have been restored, so the empty strip at
// startup or during a workspace change never overwrites them.
let notesTabsWorkspace = null;
let notesTabsSaveQueued = false;
let notesTabsLastSaved = '';

function saveNotesOpenTabs() {
    if (notesTabsSaveQueued) {
        return;
    }
    notesTabsSaveQueued = true;
    queueMicrotask(() => {
        notesTabsSaveQueued = false;
        const workspace = notesTabsWorkspace;
        if (workspace === null || workspace !== state.currentWorkspaceName) {
            return;
        }
        const docs = primaryNotesView().tabs.map(findNotesDocument).filter(Boolean);
        const tabs = docs.map(({ path, project }) => ({ path, project }));
        const active = Math.max(0, docs.indexOf(activeNotesDocument));
        const saved = JSON.stringify([workspace, tabs, active]);
        if (saved === notesTabsLastSaved) {
            return;
        }
        notesTabsLastSaved = saved;
        Promise.resolve(SetNotesOpenTabs(workspace, tabs, active)).catch((err) => {
            notesTabsLastSaved = '';
            console.error('Failed to save open notes tabs:', err);
        });
    });
}

// Reopens a workspace's remembered tabs. Background documents are read when
// their tab is first shown; the focused one loads now unless loadActive is
// false. Returns whether there were tabs to restore.
async function restoreNotesOpenTabs(saved, { loadActive = true, isCurrent = () => true } = {}) {
    const files = Array.isArray(saved?.files) ? saved.files.filter(Boolean) : [];
    if (files.length === 0) {
        return false;
    }
    const project = await GetCurrentProject();
    if (!isCurrent()) {
        return true;
    }
    const view = focusedView();
    for (const path of files) {
        view.addTab(notesDocuments.open(path, { project }).key);
    }
    syncWatchedNotesFiles();
    renderNotesTabs(view);
    const active = files[saved.active] || files[0];
    if (loadActive && active) {
        await loadFile(active);
    }
    return true;
}

function activateNotesDocument(path, { project = '' } = {}) {
    const next = notesDocuments.open(path, { project });
    syncWatchedNotesFiles();
    if (next !== activeNotesDocument) {
        setActiveNotesDocument(next);
        return next;
    }
    // The focused view may be newly showing the active document, as on
    // focusing another view of it.
    const view = focusedView();
    showDocumentInView(view, next);
    if (!view.tabs.includes(next.key)) {
        view.addTab(next.key);
        renderNotesTabs(view);
    }
    return next;
}

// Block IDs are only unique within a document, so each run gets a run ID that
// is unique across open documents. The backend echoes it in noteRun and
// noteComplete, and it maps back to the document and block that started it.
const jupyterRuns = new Map();
let jupyterRunCounter = 0;

function startJupyterRun(doc, blockId) {
    const runId = `run-${++jupyterRunCounter}`;
    doc.runningJupyterBlocks.set(blockId, runId);
    jupyterRuns.set(runId, { doc, blockId });
    return runId;
}

function appendJupyterOutput(outputBlock, text, isError) {
    if (outputBlock.childNodes.length > 0 && text.length > 0 && text[0] !== '\n' && text[0] !== '\r') {
        outputBlock.appendChild(document.createTextNode('\n'));
    }
    const span = document.createElement('span');
    span.className = isError ? 'jupyter-output-line-error' : 'jupyter-output-line';
    span.textContent = text;
    outputBlock.appendChild(span);
}

// Shows a rendered block's recorded output and whether it is still running.
function restoreJupyterBlockRun(blockId, { runButton, stopButton, outputWrapper, outputBlock }) {
    const doc = activeNotesDocument;
    const output = doc.jupyterOutputs.get(blockId);
    const running = doc.runningJupyterBlocks.has(blockId);
    if (!output && !running) {
        return;
    }
    outputWrapper.style.display = 'block';
    for (const line of output || []) {
        appendJupyterOutput(outputBlock, line.text, line.isError);
    }
    if (running) {
        runButton.style.display = 'none';
        stopButton.style.display = 'inline-block';
    }
}

function finishJupyterRun(runId) {
    const run = jupyterRuns.get(runId);
    if (!run) return null;
    jupyterRuns.delete(runId);
    if (run.doc.runningJupyterBlocks.get(run.blockId) === runId) {
        run.doc.runningJupyterBlocks.delete(run.blockId);
    }
    return run;
}

// A closed document stops its block runs and releases its own and its
// blocks' LSP and typos documents.
function releaseNotesDocument(doc) {
    syncWatchedNotesFiles();
    for (const runId of doc.runningJupyterBlocks.values()) {
        jupyterRuns.delete(runId);
        Promise.resolve(StopNote(runId)).catch((err) => {
            console.error('Error stopping code block:', err);
        });
    }
    doc.runningJupyterBlocks.clear();
    doc.jupyterOutputs.clear();
    const blocks = doc.jupyterCodeBlocks;
    doc.jupyterCodeBlocks = {};
    doc.editorModel?.dispose();
    doc.editorModel = null;
    return Promise.all([
        settleLspDocumentCloses(releaseLspDocument(doc)),
        releaseJupyterBlockLspDocuments(blocks),
    ]);
}

function isNotesDocumentOpen(doc) {
    return Boolean(doc) && notesDocuments.get(doc.path, doc.project) === doc;
}

function closeActiveNotesDocument() {
    focusedView().removeTab(activeNotesDocument.key);
    setActiveNotesDocument(newNotesDocument());
}

// The document a view shows: the active one in the focused view, and in any
// other view the one it last showed.
function notesViewDocument(view) {
    if (view === focusedView()) {
        return activeNotesDocument;
    }
    return findNotesDocument(view.shownDocumentKey);
}

function renderAllNotesTabs() {
    notesViewList().forEach(renderNotesTabs);
}

function renderNotesTabs(view) {
    const strip = view.parts.docTabs;
    const docs = view.tabs.map(findNotesDocument).filter(Boolean);
    const shown = notesViewDocument(view);
    strip.hidden = docs.length === 0;
    strip.replaceChildren(...docs.map((doc) => {
        const name = getPathFileName(doc.path);
        const tab = document.createElement('div');
        tab.className = 'notes-doc-tab';
        tab.setAttribute('role', 'tab');
        tab.setAttribute('aria-selected', String(doc === shown));
        tab.dataset.docKey = doc.key;
        tab.dataset.dirty = String(doc.dirty === true);
        tab.dataset.conflicted = String(doc.conflicted === true);
        tab.dataset.deleted = String(doc.deleted === true);
        tab.title = doc.path;
        const label = document.createElement('span');
        label.className = 'notes-doc-tab-name';
        label.textContent = name;
        const close = document.createElement('button');
        close.type = 'button';
        close.className = 'notes-doc-tab-close';
        close.setAttribute('aria-label', `Close ${name}`);
        close.textContent = '×';
        tab.append(label, close);
        return tab;
    }));
    renderNotesNotice(view);
    if (view === primaryNotesView()) {
        saveNotesOpenTabs();
    }
}

const NOTES_DOCUMENT_NOTICES = {
    deleted: {
        message: (name) => `${name} was deleted while it had unsaved edits.`,
        keep: 'Recreate file',
        discard: 'Discard edits',
    },
    conflicted: {
        message: (name) => `${name} changed on disk while it had unsaved edits.`,
        keep: 'Keep my edits',
        discard: 'Load disk version',
    },
};

function notesDocumentNotice(doc) {
    if (doc.deleted) {
        return 'deleted';
    }
    return doc.conflicted ? 'conflicted' : '';
}

// Explains why the view's document is not being saved, and offers the
// choices.
function renderNotesNotice(view) {
    const doc = notesViewDocument(view);
    const notice = doc ? notesDocumentNotice(doc) : '';
    const text = NOTES_DOCUMENT_NOTICES[notice];
    view.parts.docNotice.hidden = !text;
    view.parts.docNotice.dataset.notice = notice;
    view.parts.docNoticeMessage.textContent = text ? text.message(getPathFileName(doc.path)) : '';
    view.parts.docNoticeKeep.textContent = text?.keep || '';
    view.parts.docNoticeDiscard.textContent = text?.discard || '';
}

// A conflicted or deleted document keeps its unsaved edits until the user
// chooses what to do with them, so nothing saves it automatically.
function savesNotesDocumentAutomatically(doc) {
    return !doc.conflicted && !doc.deleted;
}

// The file of an open document was deleted. A clean document's tab closes;
// one with unsaved edits stays open, marked deleted (ADR 0100 question 12).
async function handleNotesDocumentDeleted(doc) {
    if (!isNotesDocumentOpen(doc)) {
        return;
    }
    if (doc.dirty) {
        clearTimeout(doc.autosaveTimer);
        doc.autosaveTimer = null;
        doc.conflicted = false;
        doc.deleted = true;
        renderAllNotesTabs();
        if (doc === activeNotesDocument) {
            setStatus(`${doc.path} was deleted: recreate it or discard your edits`, true);
        }
        return;
    }
    if (isNotesDocumentInTab(doc)) {
        for (const view of notesViewList().filter((candidate) => candidate.tabs.includes(doc.key))) {
            await closeNotesTab(view, doc.key, { discard: true });
        }
    } else if (doc !== activeNotesDocument) {
        notesDocuments.close(doc);
    }
}

// The file of a document with unsaved edits changed on disk. Autosave stops
// until the user chooses which version to keep.
function markNotesDocumentConflicted(doc) {
    clearTimeout(doc.autosaveTimer);
    doc.autosaveTimer = null;
    doc.conflicted = true;
    renderAllNotesTabs();
    if (doc === activeNotesDocument) {
        setStatus(`${doc.path} changed on disk: keep your edits or load the disk version`, true);
    }
}

// Saves the document's edits over the version on disk, or recreates its
// deleted file.
async function keepNotesDocumentEdits(doc = activeNotesDocument) {
    doc.conflicted = false;
    renderAllNotesTabs();
    await saveFile(doc);
}

async function loadNotesDocumentFromDisk(doc = activeNotesDocument) {
    if (doc !== activeNotesDocument) {
        return;
    }
    clearTimeout(doc.autosaveTimer);
    doc.autosaveTimer = null;
    doc.conflicted = false;
    setDirty(false);
    renderAllNotesTabs();
    await reloadActiveNotesDocument();
}

function isNotesPathMutating(path) {
    return path === state.mutatingFile
        || Boolean(state.mutatingFolder && path.startsWith(`${state.mutatingFolder}${PRIMARY_PATH_SEPARATOR}`));
}

// A file changed on disk. A clean document reloads: the active one now, a
// background one when it is next shown, as showing a tab reads the file. A
// document with unsaved edits is conflicted unless the disk text matches.
async function handleNotesDocumentChangedOnDisk(doc) {
    if (!doc?.path || isNotesPathMutating(doc.path)) {
        return;
    }
    if (!doc.dirty) {
        if (doc === activeNotesDocument) {
            await reloadActiveNotesDocument();
        }
        return;
    }
    if (doc !== activeNotesDocument) {
        markNotesDocumentConflicted(doc);
        return;
    }
    let result = null;
    try {
        result = await GetFile(doc.path);
    } catch (err) {
        console.error(err);
    }
    if (!isNotesDocumentOpen(doc) && doc !== activeNotesDocument) {
        return;
    }
    const diskText = result && !result.error && !(result.binary ?? result.text)
        ? normaliseLineEndings(result.contents)
        : null;
    if (diskText !== null && diskText === doc.buffer.getValue()) {
        doc.conflicted = false;
        if (doc === activeNotesDocument) {
            setDirty(false);
        } else {
            doc.dirty = false;
        }
        renderAllNotesTabs();
        return;
    }
    markNotesDocumentConflicted(doc);
}

function markActiveNotesTabDirty() {
    for (const view of notesViewList()) {
        const tab = [...view.parts.docTabs.children].find((el) => el.dataset.docKey === activeNotesDocument.key);
        if (tab) {
            tab.dataset.dirty = String(state.dirty === true);
        }
    }
}

async function activateNotesTab(view, key) {
    if (view !== focusedView()) {
        await focusNotesView(view, key);
        return;
    }
    const doc = findNotesDocument(key);
    if (doc && doc !== activeNotesDocument) {
        await loadFile(doc.path);
    }
}

// Closing a tab saves the document's unsaved edits (keeping the tab if that
// fails) and closes the document unless another tab shows it. Closing the
// active tab activates its neighbour. The edits of a deleted document, or of
// any document with discard, are dropped.
async function closeNotesTab(view, key, { discard = false } = {}) {
    const doc = findNotesDocument(key);
    if (doc?.dirty && !discard && !doc.deleted) {
        clearTimeout(doc.autosaveTimer);
        doc.autosaveTimer = null;
        if (!await saveFile(doc)) {
            return;
        }
    }
    const index = view.removeTab(key);
    if (index < 0) {
        return;
    }
    if (doc && !discard && !doc.deleted) {
        rememberClosedNotesTab(doc);
    }
    if (doc && doc !== activeNotesDocument && !isNotesDocumentInTab(doc)) {
        notesDocuments.close(doc);
    }
    renderNotesTabs(view);
    if (await closeEmptyNotesView(view)) {
        return;
    }
    const neighbour = findNotesDocument(view.tabs[Math.min(index, view.tabs.length - 1)]);
    if (view !== focusedView()) {
        // An unfocused view shows its next tab's document in its editor; the
        // rest of the view is rendered when it is next focused.
        if (view.shownDocumentKey === key) {
            view.shownDocumentKey = neighbour?.key || '';
            if (neighbour) {
                showDocumentInView(view, neighbour);
            }
            renderNotesTabs(view);
        }
        return;
    }
    if (doc !== activeNotesDocument) {
        return;
    }
    if (neighbour) {
        await loadFile(neighbour.path);
    } else {
        await showNoNotesDocument();
    }
}

const MAX_CLOSED_NOTES_TABS = 20;
const closedNotesTabs = [];

function rememberClosedNotesTab(doc) {
    const index = closedNotesTabs.findIndex((entry) => entry.key === doc.key);
    if (index >= 0) {
        closedNotesTabs.splice(index, 1);
    }
    closedNotesTabs.push({ key: doc.key, path: doc.path });
    if (closedNotesTabs.length > MAX_CLOSED_NOTES_TABS) {
        closedNotesTabs.shift();
    }
}

// Reopens the most recently closed tab whose document is not open again
// already, as Cmd+Shift+T does in a browser.
async function reopenClosedNotesTab() {
    while (closedNotesTabs.length > 0) {
        const entry = closedNotesTabs.pop();
        if (!focusedView().tabs.includes(entry.key)) {
            await loadFile(entry.path);
            return;
        }
    }
}

async function cycleNotesDocumentTabs(step) {
    const view = focusedView();
    const tabs = view.tabs;
    if (tabs.length < 2) {
        return;
    }
    const index = tabs.indexOf(activeNotesDocument.key);
    const next = index < 0 ? 0 : (index + step + tabs.length) % tabs.length;
    await activateNotesTab(view, tabs[next]);
}

async function closeActiveNotesTab() {
    if (focusedView().tabs.includes(activeNotesDocument.key)) {
        await closeNotesTab(focusedView(), activeNotesDocument.key);
    }
}

const NOTES_DOCUMENT_TAB_COMMANDS = {
    next: () => cycleNotesDocumentTabs(1),
    previous: () => cycleNotesDocumentTabs(-1),
    close: closeActiveNotesTab,
    reopen: reopenClosedNotesTab,
    'split-right': () => splitNotesView('row'),
    'split-down': () => splitNotesView('column'),
    'focus-left': () => focusNeighbourNotesView('left'),
    'focus-right': () => focusNeighbourNotesView('right'),
    'focus-up': () => focusNeighbourNotesView('up'),
    'focus-down': () => focusNeighbourNotesView('down'),
};

const NOTES_VIEW_FOCUS_KEYS = {
    ArrowLeft: 'left',
    ArrowRight: 'right',
    ArrowUp: 'up',
    ArrowDown: 'down',
};

// Cmd+Option+arrows move focus to the neighbouring view (ADR 0100 question 18).
async function focusNeighbourNotesView(side) {
    const view = notesViews.get(neighbourLayoutView(notesLayout, focusedView().id, side));
    if (view) {
        await focusNotesView(view);
        if (state.viewMode === 'editor') {
            view.editor?.focus();
        }
    }
}

// Cmd+Shift+] and Cmd+Shift+[ step through the document tabs, Cmd+W closes
// the active one, Cmd+Shift+T reopens the last closed (ADR 0100 question 14),
// Cmd+\ splits the view to the right (question 17) and Cmd+Option+arrows move
// between views (question 18). Ctrl+Tab still cycles the Notes mode tabs.
function notesDocumentTabCommandForKey(event) {
    if (event.metaKey && event.altKey && !event.ctrlKey && !event.shiftKey) {
        const side = NOTES_VIEW_FOCUS_KEYS[event.key];
        // Left to Monaco (add cursor above/below) when there is no view there.
        return side && neighbourLayoutView(notesLayout, focusedView().id, side) ? `focus-${side}` : '';
    }
    if (!event.metaKey || event.ctrlKey || event.altKey) {
        return '';
    }
    const key = event.key.toLowerCase();
    if (event.shiftKey && (event.code === 'BracketRight' || key === ']' || key === '}')) {
        return 'next';
    }
    if (event.shiftKey && (event.code === 'BracketLeft' || key === '[' || key === '{')) {
        return 'previous';
    }
    if (!event.shiftKey && key === 'w') {
        return 'close';
    }
    if (event.shiftKey && key === 't') {
        return 'reopen';
    }
    if (!event.shiftKey && (event.code === 'Backslash' || key === '\\')) {
        return 'split-right';
    }
    return '';
}

// Closes every tab and every view but the focused one, leaving the active
// document open but untabbed until another document replaces it, as on
// switching workspace.
async function closeBackgroundNotesTabs() {
    const keys = new Set(notesViewList().flatMap((view) => view.tabs));
    for (const view of notesViewList()) {
        if (view !== focusedView()) {
            removeNotesView(view);
        }
    }
    const docs = [...keys].map(findNotesDocument).filter((doc) => doc && doc !== activeNotesDocument);
    focusedView().tabs = [];
    for (const doc of docs) {
        if (doc.dirty && savesNotesDocumentAutomatically(doc)) {
            await saveFile(doc);
        }
        notesDocuments.close(doc);
    }
    renderNotesTabs(focusedView());
}

// Moves focus, and with it the toolbar and global chrome, to a view, then
// loads its document (or the given tab) into it. Unsaved edits are kept, as
// another view may have been editing the same document.
async function focusNotesView(view, key = view.shownDocumentKey) {
    if (!notesViews.has(view.id)) {
        return;
    }
    const previous = focusedView();
    if (view !== previous) {
        rememberShownDocumentState();
        notesToolbar.replaceWith(previous.parts.toolbarSlot);
        view.parts.toolbarSlot.replaceWith(notesToolbar);
        delete previous.root.dataset.focused;
        view.root.dataset.focused = 'true';
        notesFocusedView = view;
        // The view's DOM shows whatever it last rendered, so its document is
        // loaded afresh below rather than the outgoing view's state recorded
        // against it.
        view.shownDocumentKey = '';
        renderNotesTabs(previous);
    }
    const doc = findNotesDocument(key);
    if (doc) {
        await loadFile(doc.path, { keepUnsavedEdits: true });
    } else {
        await showNoNotesDocument();
        renderNotesTabs(view);
    }
}

// Removes a view whose last tab has closed, unless it is the only view. A
// focused view first hands focus to a neighbour, which releases the closed
// document if no other tab has it.
async function closeEmptyNotesView(view) {
    if (view.tabs.length > 0 || notesViews.size < 2 || !notesViews.has(view.id)) {
        return false;
    }
    if (view === focusedView()) {
        const nextId = ['left', 'up', 'right', 'down']
            .map((side) => neighbourLayoutView(notesLayout, view.id, side))
            .find(Boolean) || layoutViewIds(notesLayout).find((id) => id !== view.id);
        await focusNotesView(notesViews.get(nextId));
    }
    removeNotesView(view);
    return true;
}

// Removes an unfocused view from the layout. Its documents stay open; callers
// close those no other view tabs.
function removeNotesView(view) {
    if (view === focusedView() || !notesViews.delete(view.id)) {
        return;
    }
    notesLayout = closeLayoutView(notesLayout, view.id);
    view.destroy();
    renderNotesViewsLayout();
}

// Splits the focused view, side by side (row) or stacked (column), opening its
// document in the new view, which takes focus (ADR 0100 question 17).
async function splitNotesView(direction) {
    const source = focusedView();
    const doc = state.currentFile ? activeNotesDocument : null;
    rememberShownDocumentState();
    const view = createNotesView();
    notesViews.set(view.id, view);
    bindNotesView(view);
    notesLayout = splitLayoutView(notesLayout, source.id, view.id, direction);
    renderNotesViewsLayout();
    if (doc) {
        view.addTab(doc.key);
        const remembered = source.recallDocument(doc.key);
        if (remembered) {
            view.rememberDocument(doc.key, remembered);
        }
    }
    renderNotesTabs(view);
    await focusNotesView(view, doc?.key || '');
}

// The first click on an unfocused view only focuses it, so it can't act on the
// focused view's document by mistake. Its tab strip handles its own clicks.
function bindNotesViewFocus(view) {
    let swallowing = false;
    const swallow = (event) => {
        if (swallowing) {
            event.preventDefault();
            event.stopPropagation();
        }
    };
    view.root.addEventListener('pointerdown', (event) => {
        swallowing = false;
        if (view === focusedView() || view.parts.docTabs.contains(event.target)) {
            return;
        }
        swallowing = true;
        swallow(event);
        void focusNotesView(view);
    }, true);
    view.root.addEventListener('mousedown', swallow, true);
    view.root.addEventListener('mouseup', swallow, true);
    view.root.addEventListener('click', (event) => {
        swallow(event);
        swallowing = false;
    }, true);
    view.root.addEventListener('focusin', (event) => {
        if (view !== focusedView() && !swallowing && !view.parts.docTabs.contains(event.target)) {
            void focusNotesView(view);
        }
    });
    view.parts.docTabs.addEventListener('contextmenu', (event) => {
        const tab = event.target.closest('.notes-doc-tab');
        if (!tab) {
            return;
        }
        event.preventDefault();
        const key = tab.dataset.docKey;
        const splitWith = (direction) => async () => {
            if (view !== focusedView() || activeNotesDocument?.key !== key) {
                await activateNotesTab(view, key);
            }
            await splitNotesView(direction);
        };
        showNotesLocalMenu([
            { title: 'Split right', onSelect: () => void splitWith('row')() },
            { title: 'Split down', onSelect: () => void splitWith('column')() },
            { title: '-' },
            { title: 'Close tab', onSelect: () => void closeNotesTab(view, key) },
        ], event.clientX, event.clientY, 'Document tab');
    });
}

function bindNotesViewTabs(view) {
    view.parts.docTabs.addEventListener('click', (event) => {
        const tab = event.target.closest('.notes-doc-tab');
        if (!tab) {
            return;
        }
        if (event.target.closest('.notes-doc-tab-close')) {
            void closeNotesTab(view, tab.dataset.docKey);
        } else {
            void activateNotesTab(view, tab.dataset.docKey);
        }
    });

    view.parts.docTabs.addEventListener('auxclick', (event) => {
        const tab = event.target.closest('.notes-doc-tab');
        if (tab && event.button === 1) {
            event.preventDefault();
            void closeNotesTab(view, tab.dataset.docKey);
        }
    });

    view.parts.docNoticeKeep.addEventListener('click', () => {
        void keepNotesDocumentEdits();
    });

    view.parts.docNoticeDiscard.addEventListener('click', () => {
        if (activeNotesDocument.deleted) {
            void closeNotesTab(view, activeNotesDocument.key, { discard: true });
        } else {
            void loadNotesDocumentFromDisk();
        }
    });
}

// The current-file fields read the active document. currentFile and
// currentFileProject identify it, so they change only by switching document.
Object.defineProperties(state, {
    currentFile: { enumerable: true, get: () => activeNotesDocument.path },
    // The project when the file was opened, so saves after a project switch
    // still go to the right place.
    currentFileProject: { enumerable: true, get: () => activeNotesDocument.project },
    ...Object.fromEntries([
        ['currentFileUri', 'uri'],
        // 'markdown' | 'json' | 'html' | 'code' | 'image' | 'csv' | 'binary'
        ['currentFileType', 'fileType'],
        ['dirty', 'dirty'],
        ['autosaveTimer', 'autosaveTimer'],
        ['lspOpenFile', 'lspOpenFile'],
        ['lspChangeTimer', 'lspChangeTimer'],
        ['typosOpenFile', 'typosOpenFile'],
        ['typosChangeTimer', 'typosChangeTimer'],
        ['jupyterCodeBlocks', 'jupyterCodeBlocks'],
        ['jupyterBlockCounter', 'jupyterBlockCounter'],
        ['findQuery', 'findQuery'],
        ['findMatches', 'findMatches'],
        ['findCurrentIndex', 'findCurrentIndex'],
        ['findDocLastExecutedSignature', 'findDocLastExecutedSignature'],
    ].map(([name, field]) => [name, {
        enumerable: true,
        get: () => activeNotesDocument[field],
        set: (value) => { activeNotesDocument[field] = value; },
    }])),
});
let latestWindowStyle = null;
let aiPromptJumpRefreshTimer = null;
let aiPromptJumpRefreshDeadline = 0;
const AI_PROMPT_JUMP_REFRESH_MAX_DELAY_MS = 500;
let aiPromptJumpObserver = null;
let aiBottomScrollRetryTimers = [];
let aiBottomChaseHandle = 0;
let aiBottomChaseUntil = 0;
let aiStreamScrollHandle = 0;
let aiScrollButtonHandle = 0;

function nowMs() {
    return typeof performance !== 'undefined' && typeof performance.now === 'function'
        ? performance.now()
        : Date.now();
}

// Coalesces the button refresh; it measures the container, which forces layout.
function scheduleAIScrollButtonUpdate() {
    if (aiScrollButtonHandle) {
        return;
    }
    aiScrollButtonHandle = requestAnimationFrame(() => {
        aiScrollButtonHandle = 0;
        updateAIScrollBottomButton();
    });
}

// Monaco is created lazily the first time the editor becomes visible. Until
// then mainDocument holds the text (e.g. for a file opened in preview).
function isMonacoActive() {
    return focusedView().editor !== null;
}

function isGitDiffActive() {
    return Boolean(focusedView().root.dataset.gitDiff === 'true' && activeGitDiff && notesGitDiffAdapter);
}

function getMainEditorValue() {
    if (isGitDiffActive()) {
        return notesGitDiffAdapter.getModifiedValue();
    }
    return mainDocument.getValue();
}

// Resets the document (e.g. on load); not undoable and not a change. Documents
// are edited with LF line endings because Run view source offsets assume LF.
function setMainEditorValue(value) {
    mainDocument.setValue(value);
    if (isGitDiffActive()) {
        notesGitDiffAdapter.setModifiedValue(mainDocument.getValue());
    }
}

// An undoable whole-document edit that keeps the selection unless one is given.
function replaceMainDocumentText(text, { source = 'notes-document-edit', selectionStart, selectionEnd } = {}) {
    const selection = mainDocument.getSelection();
    mainDocument.replaceAll(text, {
        source,
        selectionStart: selectionStart ?? selection.start,
        selectionEnd: selectionEnd ?? selectionStart ?? selection.end,
    });
}

function getMainEditorSelectionRange() {
    if (isGitDiffActive()) {
        return notesGitDiffAdapter.getSelectionOffsets();
    }
    return mainDocument.getSelection();
}

function lineNumberAtOffset(text, offset) {
    const boundedOffset = Math.max(0, Math.min(String(text || '').length, Number(offset) || 0));
    return String(text || '').slice(0, boundedOffset).split('\n').length;
}

function currentEditorLineForFile(file) {
    if (!file || file !== state.currentFile) {
        return 1;
    }
    return lineNumberAtOffset(getMainEditorValue(), getMainEditorSelectionRange().end);
}

function scheduleHistoryPosition(file, offset) {
    if (!file || file !== state.currentFile || !['editor', 'swagger-edit', 'csv-edit'].includes(state.viewMode)) {
        return;
    }

    const line = lineNumberAtOffset(getMainEditorValue(), offset);
    clearTimeout(state.historyPositionTimer);
    state.historyPositionTimer = setTimeout(() => {
        state.historyPositionTimer = null;
        if (file !== state.currentFile || !['editor', 'swagger-edit', 'csv-edit'].includes(state.viewMode)) {
            return;
        }
        NotesHistoryAddPosition(file, line).catch((err) => {
            console.error('Failed to save Notes history position:', err);
        });
    }, 500);
}

function addHistoryPosition(file, line = 1) {
    NotesHistoryAddPosition(file, line).catch((err) => {
        console.error('Failed to save Notes history position:', err);
    });
}

function scheduleCurrentHistoryPosition() {
    scheduleHistoryPosition(state.currentFile, getMainEditorSelectionRange().end);
}

function setMainEditorSelectionRange(start, end) {
    if (isGitDiffActive()) {
        notesGitDiffAdapter.setSelectionOffsets(start, end);
        return;
    }
    mainDocument.setSelection(start, end);
}

function getMainEditorSelectionText() {
    if (isGitDiffActive()) {
        return notesGitDiffAdapter.getSelectionText();
    }
    const range = getMainEditorSelectionRange();
    return getMainEditorValue().slice(range.start, range.end);
}

function replaceMainEditorRange(start, end, text) {
    if (isGitDiffActive()) {
        notesGitDiffAdapter.replaceRange(start, end, text, 'notes-git-diff-context-menu');
        const nextCursor = Number(start) + String(text || '').length;
        notesGitDiffAdapter.setSelectionOffsets(nextCursor, nextCursor);
        return;
    }
    mainDocument.replaceRange(start, end, text, { source: 'notes-main-editor-replace' });
}

function insertTextInMainEditor(text) {
    const { start, end } = getMainEditorSelectionRange();
    replaceMainEditorRange(start, end, text);
}

function insertTextAtMainEditorLineStart(text) {
    const { start } = getMainEditorSelectionRange();
    const value = getMainEditorValue();
    const lineStart = value.lastIndexOf('\n', Math.max(0, start - 1)) + 1;
    replaceMainEditorRange(lineStart, lineStart, text);
}

function getMonacoTypographyOptions() {
    const computed = elements.monacoEditor ? getComputedStyle(elements.monacoEditor) : null;
    const parsedFontSize = Number.parseFloat(computed?.fontSize || '');
    const parsedLineHeight = Number.parseFloat(computed?.lineHeight || '');
    const fallbackFontSize = Number(latestWindowStyle?.fontSize) || 13;
    const safeFontSize = Number.isFinite(parsedFontSize) && parsedFontSize > 0 ? parsedFontSize : fallbackFontSize;

    return {
        fontFamily: String(computed?.fontFamily || latestWindowStyle?.fontFamily || '').trim(),
        fontSize: safeFontSize,
        lineHeight: Number.isFinite(parsedLineHeight) && parsedLineHeight > 0
            ? parsedLineHeight
            : Math.round(safeFontSize * 1.4),
    };
}

async function ensureMonacoMainEditor(view = focusedView()) {
    if (!view.parts.monacoEditor || view.editor) {
        return;
    }

    // view.editor is only assigned after an await, so concurrent callers must
    // share one creation or Monaco is created twice on the same container.
    if (!view.editorInit) {
        view.editorInit = createMonacoMainEditor(view).finally(() => {
            view.editorInit = null;
        });
    }

    try {
        await view.editorInit;
    } catch (err) {
        // A failed boot leaves view.editor null and silently degrades every
        // Monaco-dependent path, so surface it rather than reject unhandled.
        console.error('Monaco editor creation failed:', err);
    }
}

// Each view has its own editor. Its callbacks act on the active document, so
// they are ignored while another view is focused.
async function createMonacoMainEditor(view) {
    const typography = getMonacoTypographyOptions();
    const focused = () => view === focusedView();

    const editor = await createMonacoAdapter(view.parts.monacoEditor, {
        value: mainDocument.getValue(),
        language: state.editorLanguage || 'plaintext',
        highlightJs: hljs,
        fontSize: typography.fontSize,
        fontFamily: typography.fontFamily,
        lineHeight: typography.lineHeight,
        themeColors: latestWindowStyle?.colors || null,
        onSelectionChange: (start, end, activeOffset) => {
            if (!focused()) {
                return;
            }
            scheduleHistoryPosition(state.currentFile, activeOffset);
            if (isCurrentFileLspEligible() && state.lspOpenFile === state.currentFile) {
                state.lspHoverLastKey = '';
                scheduleLspHover();
            }
        },
        onPointerMove: (x, y) => {
            if (!focused()) {
                return;
            }
            state.lspHoverMouseX = Number(x) || 0;
            state.lspHoverMouseY = Number(y) || 0;
        },
        onBlur: () => {
            hideLspHoverTooltip();
        },
        onPaste: (event) => {
            if (focused()) {
                handleEditorImagePaste(event);
            }
        },
        onPrimaryPaste: () => {
            if (!focused() || state.viewMode !== 'editor' || state.currentFileType !== 'markdown') {
                return false;
            }
            void pasteFromGoClipboard(null, true);
            return true;
        },
        onContextMenu: (event) => {
            if (focused()) {
                openMainEditorContextMenu(event);
            }
        },
        onCompletionRequest: (payload) => {
            return focused() ? handleMonacoCompletionRequest(payload) : undefined;
        },
        onEscape: () => (focused() ? closeOpenLspTooltips() : false),
    });
    if (!notesViews.has(view.id)) {
        editor.dispose();
        return;
    }
    view.editor = editor;

    editor.setWordWrap(Boolean(state.markdownWrapMode));
    if (latestWindowStyle?.colors) {
        editor.applyTheme(latestWindowStyle.colors);
    }

    // Monaco boot can race against file loading, so the buffer hands over
    // whatever text it holds now.
    const doc = notesViewDocument(view);
    if (doc) {
        showDocumentInView(view, doc);
    }

    editor.setLanguage(state.editorLanguage || 'plaintext');
    editor.setTypography(getMonacoTypographyOptions());
    const spellcheckMisspellings = notesSpellCheckHandle?.getMisspellings?.();
    if (focused() && Array.isArray(spellcheckMisspellings)) {
        editor.setTyposMisspellings(spellcheckMisspellings);
    }
    editor.layout();

    requestAnimationFrame(() => {
        if (view.editor === editor) {
            editor.layout();
        }
    });
}

// Force Monaco to re-measure against its real, visible box across the next two
// animation frames. A display:none -> visible transition (tab switch, pane
// maximize/restore) doesn't give Monaco a correct size in the same frame, so a
// single synchronous layout() can latch a stale/zero viewport (blank editor or
// content offset with a top shadow). Two deferred measured layouts fix that.
function scheduleMonacoRelayout(view = focusedView()) {
    const editor = view.editor;
    if (!editor) {
        return;
    }
    requestAnimationFrame(() => {
        if (view.editor !== editor) {
            return;
        }
        editor.layout();
        requestAnimationFrame(() => {
            if (view.editor === editor) {
                editor.layout();
            }
        });
    });
}

// Lazily create Monaco (only ever while its container is visible/sized) and push
// the current document, language, wrap mode and theme before laying it out. This
// is the single entry point used whenever the editor tab becomes visible.
async function ensureMonacoVisibleAndLaidOut() {
    await ensureMonacoMainEditor();
    if (!isMonacoActive()) {
        return;
    }

    focusedView().editor.setLanguage(state.editorLanguage || 'plaintext');
    focusedView().editor.setWordWrap(Boolean(state.markdownWrapMode));
    focusedView().editor.setTypography(getMonacoTypographyOptions());
    const spellcheckMisspellings = notesSpellCheckHandle?.getMisspellings?.();
    if (Array.isArray(spellcheckMisspellings)) {
        focusedView().editor.setTyposMisspellings(spellcheckMisspellings);
    }
    if (latestWindowStyle?.colors) {
        focusedView().editor.applyTheme(latestWindowStyle.colors);
    }

    focusedView().editor.layout();
    scheduleMonacoRelayout();
}

configureMarked();

// Belt-and-suspenders relayout: any size change to a view's editor box (pane
// maximize/restore, splitter drag, window resize, splitting a view) triggers a
// measured Monaco relayout. This complements Monaco's own automaticLayout,
// which can latch a stale viewport across display:none -> visible transitions.
function observeNotesViewEditorSize(view) {
    if (typeof ResizeObserver === 'undefined' || !view.parts.monacoEditor?.parentElement) {
        return;
    }
    const monacoResizeObserver = new ResizeObserver(() => {
        scheduleMonacoRelayout(view);
    });
    monacoResizeObserver.observe(view.parts.monacoEditor.parentElement);
    view.disposers.push(() => monacoResizeObserver.disconnect());
}

function escapeEditorHtml(text) {
    return String(text || '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;');
}

function encodeTextToBase64(text) {
    const source = String(text || '');
    if (!source) {
        return '';
    }

    const bytes = new TextEncoder().encode(source);
    let binary = '';
    const chunkSize = 0x8000;

    for (let i = 0; i < bytes.length; i += chunkSize) {
        const chunk = bytes.subarray(i, i + chunkSize);
        binary += String.fromCharCode(...chunk);
    }

    return btoa(binary);
}

function clearHexSource() {
    state.hexSourceType = '';
    state.hexSourceValue = '';
    state.hexSourceFile = '';
    state.hexSourceOptions = null;
    state.hexRenderedFile = '';
    state.hexLoadingPromise = null;
}

function setHexSource(file, sourceType, sourceValue, options = {}) {
    state.hexSourceFile = file;
    state.hexSourceType = sourceType;
    state.hexSourceValue = sourceValue || '';
    state.hexSourceOptions = {
        fontSize: options.fontSize,
        adjustCellHeight: options.adjustCellHeight,
    };
    state.hexRenderedFile = '';
    state.hexLoadingPromise = null;
}

async function ensureHexDumpForCurrentFile() {
    const targetFile = state.currentFile;
    if (!targetFile) {
        return false;
    }

    if (state.hexRenderedFile === targetFile && state.hexSourceFile === targetFile) {
        return true;
    }

    if (state.hexSourceFile !== targetFile) {
        if (state.currentFileType !== 'image') {
            return false;
        }

        if (!state.hexLoadingPromise) {
            state.hexLoadingPromise = GetFile(targetFile)
                .then((result) => {
                    if (!result || result.error) {
                        notifyTerminal(result && result.error ? result.error : 'Failed to load hex data', 'warn');
                        return false;
                    }

                    if (state.currentFile !== targetFile) {
                        return false;
                    }

                    const sourceType = result.binary ? 'base64' : 'text';
                    setHexSource(targetFile, sourceType, result.contents || '', {
                        fontSize: result.fontSize,
                        adjustCellHeight: result.adjustCellHeight,
                    });
                    return true;
                })
                .catch((err) => {
                    notifyTerminal(String(err && err.message ? err.message : err), 'warn');
                    return false;
                })
                .finally(() => {
                    state.hexLoadingPromise = null;
                });
        }

        const loaded = await state.hexLoadingPromise;
        if (!loaded) {
            return false;
        }
    }

    if (state.hexSourceFile !== targetFile || !state.hexSourceType) {
        return false;
    }

    const base64Data = state.hexSourceType === 'base64'
        ? state.hexSourceValue
        : encodeTextToBase64(state.hexSourceValue);

    renderHexDump(elements.hex, base64Data, state.hexSourceOptions || {});
    state.hexRenderedFile = targetFile;
    return true;
}

function inferEditorLanguage(file, content) {
    const fileName = String(file || '').toLowerCase();
    const extension = fileName.includes('.') ? fileName.split('.').pop() : '';

    //const extensionMap = { foo: 'bar' }
    const extensionMap = {
        go: 'go',
        js: 'javascript',
        mjs: 'javascript',
        cjs: 'javascript',
        ts: 'typescript',
        jsx: 'javascript',
        tsx: 'typescript',
        py: 'python',
        rs: 'rust',
        c: 'c',
        h: 'c',
        cc: 'cpp',
        cpp: 'cpp',
        hpp: 'cpp',
        cs: 'csharp',
        fs: 'fsharp',
        fsx: 'fsharp',
        vb: 'vb',
        vbs: 'vb',
        java: 'java',
        jsh: 'java',
        kt: 'kotlin',
        kts: 'kotlin',
        swift: 'swift',
        php: 'php',
        rb: 'ruby',
        pl: 'perl',
        raku: 'perl',
        clj: 'clojure',
        exs: 'elixir',
        escript: 'elixir',
        jl: 'julia',
        scala: 'scala',
        sc: 'scala',
        lua: 'lua',
        r: 'r',
        st: 'st',
        dart: 'dart',
        tcl: 'tcl',
        pas: 'pascal',
        sh: 'bash',
        bash: 'bash',
        zsh: 'bash',
        fish: 'bash',
        nu: 'bash',
        awk: 'bash',
        ps1: 'powershell',
        json: 'json',
        yaml: 'yaml',
        yml: 'yaml',
        toml: 'toml',
        ini: 'ini',
        sql: 'sql',
        tf: 'terraform',
        tfvars: 'terraform',
        hcl: 'terraform',
        mx: 'murex',
        md: 'markdown',
        markdown: 'markdown',
        html: 'html',
        htm: 'html',
        xhtml: 'html',
        xht: 'html',
        shtml: 'html',
        xml: 'xml',
        plist: 'xml',
        manifest: 'xml',
        m: 'objective-c',
        css: 'css',
        scss: 'scss',
        sass: 'scss',
        less: 'less',
        dockerfile: 'dockerfile',
        makefile: 'makefile',
        scm: 'scheme',
        rkt: 'scheme',
        lisp: 'scheme',
    };

    if (extension && extensionMap[extension]) {
        return extensionMap[extension];
    }

    if (fileName.endsWith('/dockerfile') || fileName.endsWith('dockerfile')) {
        return 'dockerfile';
    }

    if (fileName.endsWith('/makefile') || fileName.endsWith('makefile')) {
        return 'makefile';
    }

    const markdownFenceMatch = String(content || '').match(/^```\s*([a-z0-9_+-]+)/im);
    if (markdownFenceMatch && hljs.getLanguage(markdownFenceMatch[1])) {
        return markdownFenceMatch[1];
    }

    return 'plaintext';
}

function renderEditorDecorations() {
    if (!isMonacoActive()) {
        return;
    }
    renderLspEditorDecorations();
    focusedView().editor.layout();
}

function refreshEditorLanguage(file, content) {
    state.editorLanguage = inferEditorLanguage(file, content);
    if (isMonacoActive()) {
        focusedView().editor.setLanguage(state.editorLanguage || 'plaintext');
    }
    renderEditorDecorations();
}

// Get indentation string (spaces) for the current language
async function getIndentationString() {
    const language = state.editorLanguage || '';
    const spaceCount = await GetNotesLanguageTabIndent(language);
    return ' '.repeat(Math.max(1, spaceCount || 4));
}

function usesCodeEditorDecorations() {
    return state.currentFileType === 'code' || state.currentFileType === 'json' || state.currentFileType === 'markdown' || state.currentFileType === 'html';
}

const SYNTAX_COMPLETION_TRIGGER_KEYS = new Set(['(', '[', '{', '"', "'", '`', '<', '>']);
const syntaxCompletionRequestSeq = new WeakMap();

function nextSyntaxCompletionRequestSeq(textarea) {
    const next = (syntaxCompletionRequestSeq.get(textarea) || 0) + 1;
    syntaxCompletionRequestSeq.set(textarea, next);
    return next;
}

function isLatestSyntaxCompletionRequest(textarea, seq) {
    return (syntaxCompletionRequestSeq.get(textarea) || 0) === seq;
}

function syntaxCompletionTriggerFromKey(key) {
    return key === 'Enter' ? '\n' : key;
}

function isSyntaxCompletionKeyEvent(event) {
    if (!event || event.isComposing || event.ctrlKey || event.metaKey || event.altKey) {
        return false;
    }

    if (event.key === 'Enter') {
        return true;
    }

    return SYNTAX_COMPLETION_TRIGGER_KEYS.has(String(event.key || ''));
}

// Phase 1 foundation: central undo manager + mutation adapter for editor range edits.
const notesUndoManager = createEditorUndoManager();
const notesMutationAdapter = createEditorMutationAdapter({
    manager: notesUndoManager,
    getFilePath: () => state.currentFile || '',
});

function applyTextareaEdit(textarea, start, end, text, cursor) {
    if (!textarea) {
        return;
    }

    notesMutationAdapter.replaceRange(textarea, {
        start,
        end,
        text,
        cursor,
        source: 'syntax-completion',
        label: 'Apply textarea edit',
        emit: true,
    });
}

function maybeHandleSyntaxCompletionKey(event, textarea, options = {}) {
    if (!isSyntaxCompletionKeyEvent(event) || !textarea) {
        return false;
    }

    const trigger = syntaxCompletionTriggerFromKey(event.key);
    const source = textarea.value || '';
    const selectionStart = textarea.selectionStart || 0;
    const selectionEnd = textarea.selectionEnd || 0;
    const cursor = selectionStart;
    const requestSeq = nextSyntaxCompletionRequestSeq(textarea);

    event.preventDefault();
    event.stopPropagation();

    const applyFallback = () => {
        if (!isLatestSyntaxCompletionRequest(textarea, requestSeq)) {
            return;
        }
        applyTextareaEdit(textarea, selectionStart, selectionEnd, trigger, selectionStart + trigger.length);
    };

    void CompleteSyntax(
        options.docPath || '',
        options.languageHint || '',
        source,
        cursor,
        selectionStart,
        selectionEnd,
        trigger,
    ).then((result) => {
        if (!isLatestSyntaxCompletionRequest(textarea, requestSeq)) {
            return;
        }

        if (!result || result.error) {
            applyFallback();
            return;
        }

        if (!result.applied) {
            applyFallback();
            return;
        }

        applyTextareaEdit(textarea, result.start, result.end, result.text, result.cursor);
    }).catch(() => {
        applyFallback();
    });

    return true;
}

function isMarkdownNotesFile(fileName) {
    return /\.(md|markdown)$/i.test(String(fileName || ''));
}

function isHtmlViewFile(fileName) {
    return /\.(html?|xhtml|xht|shtml)$/i.test(String(fileName || ''));
}

function isImageFile(fileName) {
    return /\.(png|jpe?g|gif|webp|svg|bmp|ico|tiff?)$/i.test(String(fileName || ''));
}

function isSvgFile(fileName) {
    return /\.svg$/i.test(String(fileName || ''));
}

function isCsvFile(fileName) {
    return /\.csv$/i.test(String(fileName || ''));
}

/**
 * Parse CSV text into a 2D array of strings.
 * Handles quoted fields (including embedded commas and newlines).
 */
function parseCsv(text) {
    const rows = [];
    let row = [];
    let field = '';
    let inQuotes = false;
    const n = text.length;

    for (let i = 0; i < n; i++) {
        const ch = text[i];
        if (inQuotes) {
            if (ch === '"') {
                // Peek ahead: escaped quote?
                if (i + 1 < n && text[i + 1] === '"') {
                    field += '"';
                    i++;
                } else {
                    inQuotes = false;
                }
            } else {
                field += ch;
            }
        } else {
            if (ch === '"') {
                inQuotes = true;
            } else if (ch === ',') {
                row.push(field);
                field = '';
            } else if (ch === '\r') {
                // skip
            } else if (ch === '\n') {
                row.push(field);
                field = '';
                rows.push(row);
                row = [];
            } else {
                field += ch;
            }
        }
    }
    // trailing field/row
    if (field !== '' || row.length > 0) {
        row.push(field);
        rows.push(row);
    }
    // Drop a trailing empty row (common with files ending in \n)
    if (rows.length > 0 && rows[rows.length - 1].every(f => f === '')) {
        rows.pop();
    }
    return rows;
}

function escapeCsvField(value) {
    const text = String(value ?? '');
    if (text.includes('"') || text.includes(',') || text.includes('\n') || text.includes('\r')) {
        return `"${text.replace(/"/g, '""')}"`;
    }
    return text;
}

function serializeCsvRows(rows) {
    return (rows || [])
        .map((row) => (row || []).map((field) => escapeCsvField(field)).join(','))
        .join('\n');
}

function escapeHtml(str) {
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}

function updateCsvCell(rowIndex, columnIndex, value) {
    const rows = parseCsv(mainDocument.getValue());
    while (rows.length <= rowIndex) {
        rows.push([]);
    }

    const row = rows[rowIndex];
    while (row.length <= columnIndex) {
        row.push('');
    }

    row[columnIndex] = String(value ?? '').trim();

    replaceMainDocumentText(serializeCsvRows(rows), { source: 'csv-table-edit' });
    setDirty(true);
    renderCsvView(mainDocument.getValue(), { interactive: state.viewMode === 'csv-run' });
    scheduleAutoSave();
    saveFile();

    return true;
}

function insertCsvRowAfter(rowIndex) {
    const rows = parseCsv(mainDocument.getValue());
    const colCount = rows.length > 0 ? Math.max(...rows.map(r => r.length)) : 1;
    const newRow = Array(colCount).fill('');
    rows.splice(rowIndex + 1, 0, newRow);
    replaceMainDocumentText(serializeCsvRows(rows), { source: 'csv-table-edit' });
    setDirty(true);
    renderCsvView(mainDocument.getValue(), { interactive: state.viewMode === 'csv-run' });
    scheduleAutoSave();
    saveFile();
}

function insertCsvColumnAfter(columnIndex) {
    const rows = parseCsv(mainDocument.getValue());
    rows.forEach(row => {
        while (row.length <= columnIndex) row.push('');
        row.splice(columnIndex + 1, 0, '');
    });
    replaceMainDocumentText(serializeCsvRows(rows), { source: 'csv-table-edit' });
    setDirty(true);
    renderCsvView(mainDocument.getValue(), { interactive: state.viewMode === 'csv-run' });
    scheduleAutoSave();
    saveFile();
}

function deleteCsvRow(rowIndex) {
    const rows = parseCsv(mainDocument.getValue());
    if (rowIndex < 0 || rowIndex >= rows.length) {
        return;
    }

    // Keep at least one row so table structure remains valid.
    if (rows.length <= 1) {
        return;
    }

    rows.splice(rowIndex, 1);
    replaceMainDocumentText(serializeCsvRows(rows), { source: 'csv-table-edit' });
    setDirty(true);
    renderCsvView(mainDocument.getValue(), { interactive: state.viewMode === 'csv-run' });
    scheduleAutoSave();
    saveFile();
}

function deleteCsvColumn(columnIndex) {
    const rows = parseCsv(mainDocument.getValue());
    if (rows.length === 0) {
        return;
    }

    const maxCols = Math.max(...rows.map(r => r.length), 0);
    if (maxCols <= 1 || columnIndex < 0 || columnIndex >= maxCols) {
        return;
    }

    rows.forEach(row => {
        while (row.length < maxCols) {
            row.push('');
        }
        row.splice(columnIndex, 1);
    });

    replaceMainDocumentText(serializeCsvRows(rows), { source: 'csv-table-edit' });
    setDirty(true);
    renderCsvView(mainDocument.getValue(), { interactive: state.viewMode === 'csv-run' });
    scheduleAutoSave();
    saveFile();
}

function setupInteractiveTableCells(container, isEditable, resolveCommit, afterCommit) {
    if (!container || !isEditable) {
        return;
    }

    const tables = Array.from(container.querySelectorAll('table'));
    if (tables.length === 0) {
        return;
    }

    tables.forEach((table, tableIndex) => {
        const commitCell = typeof resolveCommit === 'function'
            ? resolveCommit(table, tableIndex)
            : null;
        if (typeof commitCell !== 'function') {
            return;
        }

        const attachEditor = (cell, sourceRowIndex, columnIndex) => {
            cell.addEventListener('dblclick', (event) => {
                event.preventDefault();
                event.stopPropagation();

                if (cell.dataset.tableEditing === 'true') {
                    return;
                }

                // Find the .notes-table-cell-wrap and .notes-cellref inside this cell
                const wrap = cell.querySelector('.notes-table-cell-wrap');
                const cellRef = wrap ? wrap.querySelector('.notes-cellref') : null;
                const contentSpan = wrap ? wrap.querySelector('span:first-child') : null;
                const sortIcon = cell.querySelector('.notes-sort-icon');

                // Save the cellref and sort icon elements to restore later
                let cellRefNode = null;
                let sortIconNode = null;
                if (cellRef) {
                    cellRefNode = cellRef;
                    cellRef.remove();
                }
                if (sortIcon) {
                    sortIconNode = sortIcon;
                    sortIcon.remove();
                }

                const displayValue = contentSpan ? contentSpan.textContent : String(cell.textContent || '').trim();
                // If the cell has a formula, show the raw formula for editing
                const rawFormula = cell.dataset.formula || '';
                const editValue = rawFormula || displayValue;

                cell.dataset.tableEditing = 'true';
                cell.setAttribute('contenteditable', 'true');
                disableNativeTextAssistance(cell);
                cell.style.outline = 'none';
                cell.style.boxShadow = 'inset 0 0 0 1px var(--accent)';

                // Show formula text for editing
                cell.textContent = rawFormula || displayValue;

                const selection = window.getSelection ? window.getSelection() : null;
                const range = document.createRange ? document.createRange() : null;
                if (selection && range) {
                    range.selectNodeContents(cell);
                    range.collapse(false);
                    if (typeof selection.removeAllRanges === 'function') {
                        selection.removeAllRanges();
                    }
                    if (typeof selection.addRange === 'function') {
                        selection.addRange(range);
                    }
                }
                cell.focus();

                const finish = (commit) => {
                    if (cell.dataset.tableEditing !== 'true') {
                        return;
                    }

                    cell.dataset.tableEditing = 'false';
                    cell.removeAttribute('contenteditable');
                    cell.removeAttribute('spellcheck');
                    cell.style.outline = '';
                    cell.style.boxShadow = '';

                    const nextValue = commit ? String(cell.textContent || '').trim() : editValue;
                    // Restore the cell's HTML structure with cellref after editing
                    if (!commit) {
                        // Restore display (evaluated) value on cancel
                        if (wrap && contentSpan) {
                            contentSpan.textContent = displayValue;
                        }
                    } else {
                        if (wrap && contentSpan) {
                            contentSpan.textContent = nextValue;
                        }
                    }
                    // Restore the cellref and sort icon nodes if they were present
                    if (wrap && cellRefNode) {
                        wrap.appendChild(cellRefNode);
                    }
                    cell.innerHTML = wrap ? wrap.outerHTML : cell.innerHTML;
                    if (sortIconNode) {
                        cell.prepend(sortIconNode);
                    }

                    cell.removeEventListener('keydown', onKeyDown);
                    cell.removeEventListener('blur', onBlur);

                    if (commit) {
                        commitCell(sourceRowIndex, columnIndex, nextValue);
                        if (typeof afterCommit === 'function') {
                            afterCommit();
                        }
                    }
                };

                const onKeyDown = (keyEvent) => {
                    if (keyEvent.key === 'Enter') {
                        keyEvent.preventDefault();
                        finish(true);
                    } else if (keyEvent.key === 'Escape') {
                        keyEvent.preventDefault();
                        finish(false);
                    }
                };

                const onBlur = () => {
                    finish(true);
                };

                cell.addEventListener('keydown', onKeyDown);
                cell.addEventListener('blur', onBlur);
            });
        };

        const headerRow = table.querySelector('thead tr');
        if (headerRow) {
            const headerCells = Array.from(headerRow.querySelectorAll('th'));
            headerCells.forEach((cell, columnIndex) => {
                attachEditor(cell, 0, columnIndex);
            });
        }

        const bodyRows = Array.from(table.querySelectorAll('tbody tr'));
        bodyRows.forEach((row, rowIndex) => {
            const cells = Array.from(row.querySelectorAll('td'));
            cells.forEach((cell, columnIndex) => {
                attachEditor(cell, rowIndex + 1, columnIndex);
            });
        });
    });
}

function renderCsvView(content, options = {}) {
    const interactive = Boolean(options.interactive);
    const renderCellRef = (ref) => interactive ? `<span class="notes-cellref">${ref}</span>` : '';
    const rows = parseCsv(content || '');
    if (rows.length === 0) {
        elements.csvView.innerHTML = '<p class="notes-csv-empty">Empty file</p>';
        return;
    }

    // Evaluate formulas for display (but not for editing)
    const [headerRow, ...dataRows] = rows;
    // Build a table of display values (header untouched)
    const displayRows = [headerRow, ...dataRows.map((row, rIdx) =>
        headerRow.map((_, cIdx) => {
            const val = row[cIdx] ?? '';
            if (isTableFormula(val)) {
                // rIdx+1 because dataRows skips header
                return evaluateTableFormula(val, rIdx + 1, cIdx, rows);
            }
            return val;
        })
    )];

    ensureCellRefStyle();
    const thead = headerRow.map((h, cIdx) => {
        const ref = getCellReference(0, cIdx);
        return `<th><span class="notes-table-cell-wrap"><span>${escapeHtml(h)}</span>${renderCellRef(ref)}</span></th>`;
    }).join('');
    const tbody = dataRows.map((r, rIdx) => {
        const cells = headerRow.map((_, cIdx) => {
            const origVal = r[cIdx] ?? '';
            const displayVal = displayRows[rIdx + 1][cIdx];
            const ref = getCellReference(rIdx + 1, cIdx);
            const formulaAttr = (interactive && isTableFormula(origVal))
                ? ` data-formula="${escapeHtml(origVal)}"`
                : '';
            return `<td${formulaAttr}><span class="notes-table-cell-wrap"><span>${escapeHtml(displayVal)}</span>${renderCellRef(ref)}</span></td>`;
        }).join('');
        return `<tr>${cells}</tr>`;
    }).join('');

    elements.csvView.innerHTML = `<table><thead><tr>${thead}</tr></thead><tbody>${tbody}</tbody></table>`;

    if (interactive) {
        setupInteractiveTableCells(
            elements.csvView,
            true,
            () => (sourceRowIndex, columnIndex, value) => updateCsvCell(sourceRowIndex, columnIndex, value),
        );
    }

    // Enable column sorting (available in both view and run mode)
    setupNotesTableQueries(elements.csvView, 'csv', state.currentFile || '');
    void setupTableColumnResizing(elements.csvView, false, state.currentFile);
    if (!interactive) {
        void autoHyperlink(elements.csvView);
    }

    if (elements.toolsTabFind?.getAttribute('aria-selected') === 'true' && state.findQuery
        && (state.viewMode === 'csv-view' || state.viewMode === 'csv-run')) {
        setTimeout(() => performFind(), 0);
    }
}

function setCodeEditorMode(enabled) {
    if (!elements.editorShell) {
        return;
    }

    elements.editorShell.dataset.codeView = enabled ? 'true' : 'false';

    if (!enabled) {
        delete elements.editorShell.dataset.fileType;
    }
}

function setStatus(message, isError) {
    elements.status.textContent = message || '';
    elements.status.dataset.state = isError ? 'error' : 'ok';
}

function getPathParts(path) {
    if (!path) {
        return [];
    }

    const source = String(path).includes(PRIMARY_PATH_SEPARATOR)
        ? String(path)
        : String(path).replaceAll(FALLBACK_PATH_SEPARATOR, PRIMARY_PATH_SEPARATOR);

    return source.split(PRIMARY_PATH_SEPARATOR).filter(Boolean);
}

function getPathFileName(path) {
    const parts = getPathParts(path);
    return parts.length === 0 ? '' : parts[parts.length - 1];
}

function splitCategoryPath(file) {
    const match = String(file || '').match(/^(\$[A-Z]+)(?:[\\/](.*))?$/);
    if (!match) {
        return {
            category: '',
            relativePath: String(file || ''),
        };
    }

    return {
        category: match[1],
        relativePath: match[2] || '',
    };
}

function sortTreeNodes(nodes) {
    nodes.sort((left, right) => {
        if (left.type !== right.type) {
            return left.type === 'folder' ? -1 : 1;
        }

        return left.name.localeCompare(right.name, undefined, { numeric: true, sensitivity: 'base' });
    });

    nodes.forEach((node) => {
        if (node.type === 'folder') {
            sortTreeNodes(node.children);
        }
    });
}

function buildFileTree(files) {
    const root = [];

    files.forEach((file) => {
        const { relativePath } = splitCategoryPath(file);
        const segments = getPathParts(relativePath);
        let level = root;

        segments.forEach((segment, index) => {
            const isLeaf = index === segments.length - 1;
            let node = level.find((entry) => entry.name === segment && entry.type === (isLeaf ? 'file' : 'folder'));

            if (!node) {
                node = isLeaf
                    ? { type: 'file', name: segment, file }
                    : { type: 'folder', name: segment, path: segments.slice(0, index + 1).join(PRIMARY_PATH_SEPARATOR), children: [] };
                level.push(node);
            }

            if (!isLeaf) {
                level = node.children;
            }
        });
    });

    sortTreeNodes(root);
    return root;
}

function createTreeIndent(depth, continueAtLevels = []) {
    const indent = document.createElement('span');
    indent.className = 'notes-tree-indent';
    indent.setAttribute('aria-hidden', 'true');

    for (let ancestorDepth = 1; ancestorDepth < depth; ancestorDepth += 1) {
        const segment = document.createElement('span');
        segment.className = 'notes-tree-branch';
        
        const shouldContinue = continueAtLevels[ancestorDepth] === true;
        segment.classList.add(shouldContinue ? 'notes-tree-branch-continue' : 'notes-tree-branch-empty');

        indent.appendChild(segment);
    }

    return indent;
}

function renderTreeNodeItem(container, category, node, depth, continueAtLevels, isLast) {
    // Create the indent column - shows ancestor continuation lines
    const indentForItem = createTreeIndent(depth, continueAtLevels);

    // Add the current level's connector (elbow or end)
    if (depth > 0) {
        const lastSegment = document.createElement('span');
        lastSegment.className = 'notes-tree-branch';
        lastSegment.classList.add(isLast ? 'notes-tree-branch-end' : 'notes-tree-branch-elbow');
        indentForItem.appendChild(lastSegment);
    }

    const label = document.createElement('span');
    label.className = 'notes-tree-label';
    label.textContent = node.name;

    if (node.type === 'folder') {
        const folder = document.createElement('button');
        folder.type = 'button';
        folder.className = 'notes-tree-folder';
        folder.appendChild(indentForItem);
        folder.appendChild(label);

        const folderKey = `${category}${PRIMARY_PATH_SEPARATOR}${node.path}`;
        const expanded = state.expandedFolders[folderKey] !== false;
        folder.dataset.folderKey = folderKey;
        folder.dataset.expanded = expanded ? 'true' : 'false';
        folder.setAttribute('aria-expanded', expanded ? 'true' : 'false');
        container.appendChild(folder);

        // Render children if expanded
        if (expanded && Array.isArray(node.children) && node.children.length > 0) {
            const newContinueAtLevels = [...continueAtLevels];
            newContinueAtLevels[depth] = !isLast; // Pass true to children if this node has siblings after it
            renderTreeNodesList(container, category, node.children, depth + 1, newContinueAtLevels);
        }
    } else {
        const item = document.createElement('button');
        item.type = 'button';
        item.className = 'notes-file notes-tree-file';
        item.dataset.file = node.file;
        const projectRoot = state.currentProjectRoot
            ? `${state.currentProjectRoot.replace(/[\\/]+$/, '')}/`
            : '(none)';
        const { relativePath } = splitCategoryPath(node.file);
        const directory = `/${getPathParts(relativePath).slice(0, -1).join('/')}`;
        const tooltipCode = (value) => `<code>${escapeHtml(value).replace(
            /[\\`*_[\]{}()#+.!|~-]/g,
            (character) => `&#${character.charCodeAt(0)};`,
        )}</code>`;
        item.dataset.tooltip = [
            ['Project', 'f015', projectRoot],
            ['Path', 'f07b', directory],
            ['File', 'f15b', node.name],
        ].map(([label, icon, value]) => `<div class="notes-file-tooltip-row"><span class="notes-file-tooltip-icon" role="img" aria-label="${label}">&#x${icon};</span> ${tooltipCode(value)}</div>`).join('\n');
        item.appendChild(indentForItem);
        item.appendChild(label);

        if (node.file === state.currentFile) {
            item.dataset.active = 'true';
        }

        container.appendChild(item);
    }
}

function renderTreeNodesList(container, category, nodes, depth = 0, continueAtLevels = []) {
    nodes.forEach((node, index) => {
        const isLast = index === nodes.length - 1;
        renderTreeNodeItem(container, category, node, depth, continueAtLevels, isLast);
    });
}

function notifyTerminal(message, level = 'info') {
    if (!message) {
        return;
    }

    SendIpc('terminal-notify', {
        level,
        message,
    }).catch(() => {});
}

function openStickyProgress(id, message) {
    return SendIpc('terminal-sticky-create', {
        id: String(id),
        message,
        level: 'info',
    }).catch(() => {});
}

function updateStickyProgress(id, message) {
    SendIpc('terminal-sticky-update', {
        id: String(id),
        message,
    }).catch(() => {});
}

function closeStickyProgress(id, finalMessage, level = 'info') {
    SendIpc('terminal-sticky-close', {
        id: String(id),
    }).catch(() => {});
    if (finalMessage) {
        notifyTerminal(finalMessage, level);
    }
}

function yieldToUI() {
    return new Promise((resolve) => {
        setTimeout(resolve, 0);
    });
}

// Matches a leading YAML frontmatter block: the document must begin with a line
// containing only "---", followed by the YAML body, terminated by another line
// containing only "---". An optional BOM at the very start is tolerated.
const FRONTMATTER_RX = /^\uFEFF?---[ \t]*\r?\n([\s\S]*?)\r?\n---[ \t]*(?:\r?\n|$)/;

function parseFrontmatter(markdown) {
    const text = String(markdown ?? '');
    const match = text.match(FRONTMATTER_RX);
    if (!match) {
        return { present: false, data: null, body: text };
    }

    let data;
    try {
        data = YAML.parse(match[1]);
    } catch {
        // Malformed YAML frontmatter is left in the document body untouched.
        return { present: false, data: null, body: text };
    }

    // Only object/array frontmatter is meaningful to render in the tree viewer.
    if (data === null || typeof data !== 'object') {
        return { present: false, data: null, body: text };
    }

    return { present: true, data, body: text.slice(match[0].length) };
}

// Parses the current document's frontmatter, updates state, refreshes the
// Frontmatter tab, and returns the markdown body with the frontmatter stripped.
function applyDocumentFrontmatter() {
    const raw = getMainEditorValue();
    if (state.currentFileType !== 'markdown') {
        state.frontmatter = null;
        syncFrontmatterTab();
        return raw;
    }

    const fm = parseFrontmatter(raw);
    state.frontmatter = fm.present ? fm.data : null;
    syncFrontmatterTab();
    return fm.body;
}

// Shows or hides the Frontmatter tab based on the current document and renders
// the parsed frontmatter into its pane using the JSON/YAML tree viewer.
function syncFrontmatterTab() {
    const present = state.currentFileType === 'markdown' && state.frontmatter != null;

    if (elements.toolsTabFrontmatter) {
        elements.toolsTabFrontmatter.style.display = present ? '' : 'none';
    }

    if (present) {
        attachJsonViewerEditHandler(elements.toolsFrontmatter, commitFrontmatterEdit);
        renderJsonViewer(elements.toolsFrontmatter, state.frontmatter);
        return;
    }

    if (elements.toolsFrontmatter) {
        elements.toolsFrontmatter.innerHTML = '';
    }

    // Fall back only when the selected tab is unavailable in this document.
    if (elements.toolsTabFrontmatter?.getAttribute('aria-selected') === 'true') {
        setToolsTab('find');
    }
}

// Prepends a caption to a rendered markdown container indicating that the
// document contains frontmatter, with a button to reveal it in the Tools panel.
function insertFrontmatterCaption(container) {
    if (!container || state.currentFileType !== 'markdown' || state.frontmatter == null) {
        return;
    }

    const caption = document.createElement('div');
    caption.className = 'notes-frontmatter-caption';

    const label = document.createElement('span');
    label.className = 'notes-frontmatter-caption-text';
    label.textContent = 'This document contains Frontmatter. ';

    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'notes-frontmatter-caption-view';
    button.textContent = 'View';
    button.addEventListener('click', () => {
        setToolsPanelCollapsed(false);
        setToolsTab('frontmatter');
    });

    caption.append(label, button);
    container.prepend(caption);
}

let markdownRenderSeq = 0;

// A render stops after an await once a newer render has started or its
// document is no longer shown, so it never post-processes another render's DOM.
async function renderMarkdown() {
    const renderSeq = ++markdownRenderSeq;
    const doc = activeNotesDocument;
    const isCurrent = () => renderSeq === markdownRenderSeq && doc === activeNotesDocument;
    const markdown = applyDocumentFrontmatter();
    elements.preview.innerHTML = marked.parse(markdown);

    // Apply common markdown processing
    await processMarkdownContainer(elements.preview);
    if (!isCurrent()) {
        return;
    }

    // Enable context menus on images
    enableImageContextMenus(elements.preview);

    // Keep checkboxes readonly in viewer mode
    setupInteractiveCheckboxes(elements.preview, false);

    // Enable collapsible H1-H6 sections
    setupCollapsibleHeadings(elements.preview);

    // Keep table horizontal scrolling local to each table rather than the whole section.
    wrapTablesForHorizontalScroll(elements.preview);

    // Enable column sorting on all tables
    setupNotesTableQueries(elements.preview, 'preview', doc.path);

    // Apply the word-wrap CSS class and enable resizable table columns with persisted widths.
    elements.preview.classList.toggle('notes-table-wordwrap-on', state.markdownTableWordWrapMode);
    await setupTableColumnResizing(elements.preview, state.markdownTableWordWrapMode, doc.path);
    if (!isCurrent()) {
        return;
    }

    // Surface a caption when the document carries frontmatter.
    insertFrontmatterCaption(elements.preview);

    refreshToolsToC();

    // Re-apply find highlights when Find tab is active in viewer mode.
    if (elements.toolsTabFind?.getAttribute('aria-selected') === 'true' && state.findQuery && state.viewMode === 'viewer') {
        setTimeout(() => {
            performFind();
        }, 0);
    }
}

function renderHtmlView() {
    if (!elements.htmlViewFrame) {
        return;
    }

    // Sandboxed iframe without allow-scripts keeps inline/external JS disabled.
    elements.htmlViewFrame.srcdoc = mainDocument.getValue();
}

function getToCHeadingKey(level, text) {
    return `${String(level)}::${String(text || '').trim().toLowerCase()}`;
}

function markdownInlineText(value) {
    const holder = document.createElement('div');
    holder.innerHTML = marked.parseInline(String(value || ''));
    return String(holder.textContent || '').trim();
}

function getMarkdownHeadingSourceDescriptors(markdown) {
    const lines = String(markdown || '').split('\n');
    const headings = [];
    let start = 0;
    if ((lines[0] || '').trim() === '---') {
        const end = lines.findIndex((line, index) => index > 0 && /^(---|\.\.\.)\s*$/.test(line.trim()));
        if (end >= 0) start = end + 1;
    }

    let fence = null;
    for (let index = start; index < lines.length; index += 1) {
        const line = lines[index];
        const trimmed = line.trim();
        const fenceMatch = trimmed.match(/^(`{3,}|~{3,})/);
        if (fenceMatch) {
            const marker = fenceMatch[1];
            if (!fence) fence = { char: marker[0], length: marker.length };
            else if (marker[0] === fence.char && marker.length >= fence.length) fence = null;
            continue;
        }
        if (fence) continue;

        const setext = index + 1 < lines.length ? lines[index + 1].match(/^ {0,3}(=+|-+)\s*$/) : null;
        if (trimmed && setext) {
            headings.push({ level: setext[1][0] === '=' ? 1 : 2, text: markdownInlineText(trimmed), sourceLine: index + 1 });
            index += 1;
            continue;
        }

        const atx = line.match(/^ {0,3}(#{1,6})(?:\s+|$)(.*?)\s*#*\s*$/);
        if (atx) {
            const text = markdownInlineText(atx[2]);
            if (text) headings.push({ level: atx[1].length, text, sourceLine: index + 1 });
            continue;
        }

        const html = line.match(/^ {0,3}<h([1-6])\b[^>]*>(.*?)<\/h\1>/i);
        if (html) {
            const text = markdownInlineText(html[2].replace(/<[^>]*>/g, ''));
            if (text) headings.push({ level: Number(html[1]), text, sourceLine: index + 1 });
        }
    }
    return headings;
}

function getHeadingDescriptors(root, sourceMarkdown = null) {
    const headings = Array.from(root?.querySelectorAll?.('h1, h2, h3, h4, h5, h6') || []);
    const seen = new Map();
    const descriptors = [];
    const sourceHeadings = sourceMarkdown === null ? [] : getMarkdownHeadingSourceDescriptors(sourceMarkdown);
    let sourceCursor = 0;

    headings.forEach((heading) => {
        const level = Number.parseInt(String(heading.tagName).slice(1), 10) || 1;
        const text = String(heading.textContent || '').trim();
        if (!text) {
            return;
        }

        const key = getToCHeadingKey(level, text);
        const occurrence = seen.get(key) || 0;
        seen.set(key, occurrence + 1);

        let sourceMatch = -1;
        for (let index = sourceCursor; index < sourceHeadings.length; index += 1) {
            const source = sourceHeadings[index];
            if (source.level === level && source.text.trim().toLowerCase() === text.toLowerCase()) {
                sourceMatch = index;
                break;
            }
        }
        if (sourceMatch < 0 && descriptors.length < sourceHeadings.length) {
            sourceMatch = descriptors.length;
        }
        const sourceLine = sourceMatch >= 0 ? sourceHeadings[sourceMatch]?.sourceLine : undefined;
        if (sourceMatch >= sourceCursor) sourceCursor = sourceMatch + 1;

        descriptors.push({
            heading,
            level,
            text,
            occurrence,
            anchor: String(heading.id || ''),
            sourceLine,
        });
    });

    return descriptors;
}

function setToolsToCActiveItem(activeEntry) {
    const tocItems = elements.toolsToC?.querySelectorAll('.notes-tools-toc-item') || [];
    let activeButton = null;

    for (const item of tocItems) {
        const isActive = Boolean(activeEntry)
            && String(item.dataset.level) === String(activeEntry.level)
            && String(item.dataset.text || '').trim().toLowerCase() === String(activeEntry.text || '').trim().toLowerCase()
            && String(item.dataset.occurrence) === String(activeEntry.occurrence);

        item.dataset.active = isActive ? 'true' : 'false';
        if (isActive) {
            activeButton = item;
        }
    }

    if (activeButton) {
        activeButton.scrollIntoView({ block: 'nearest' });
    }
}

function clearToolsToCHighlight(resetScroll = false) {
    setToolsToCActiveItem(null);
    if (resetScroll && elements.toolsToC) {
        elements.toolsToC.scrollTop = 0;
    }
}

function getToolsToCModeContext() {
    if (state.viewMode === 'jupyter') {
        return {
            scrollContainer: elements.jupyterWrap,
            headingRoot: elements.jupyter,
        };
    }

    if (state.viewMode === 'viewer') {
        return {
            scrollContainer: elements.previewWrap,
            headingRoot: elements.preview,
        };
    }

    return null;
}

function syncToolsToCHighlightForMode() {
    if (!elements.toolsToC || state.currentFileType !== 'markdown') {
        clearToolsToCHighlight(true);
        return;
    }

    if (state.viewMode !== 'viewer' && state.viewMode !== 'jupyter') {
        clearToolsToCHighlight(true);
        return;
    }

    const modeContext = getToolsToCModeContext();
    if (!modeContext?.scrollContainer || !modeContext?.headingRoot) {
        clearToolsToCHighlight(false);
        return;
    }

    const descriptors = getHeadingDescriptors(modeContext.headingRoot);
    if (!descriptors.length) {
        clearToolsToCHighlight(false);
        return;
    }

    const containerRect = modeContext.scrollContainer.getBoundingClientRect();
    const threshold = containerRect.top + 28;

    let activeEntry = null;
    for (const descriptor of descriptors) {
        const rect = descriptor.heading.getBoundingClientRect();
        if (rect.top <= threshold) {
            activeEntry = descriptor;
        } else {
            break;
        }
    }

    if (!activeEntry) {
        activeEntry = descriptors[0];
    }

    setToolsToCActiveItem(activeEntry);
}

function refreshToolsToC() {
    if (!elements.toolsToC) {
        return;
    }

    if (state.currentFileType !== 'markdown') {
        elements.toolsToC.innerHTML = '';
        return;
    }

    const markdown = getMainEditorValue();
    const descriptors = state.viewMode === 'editor'
        ? getMarkdownHeadingSourceDescriptors(markdown).map((entry, index, all) => ({
            ...entry,
            heading: null,
            anchor: '',
            occurrence: all.slice(0, index).filter((candidate) => candidate.level === entry.level && candidate.text.toLowerCase() === entry.text.toLowerCase()).length,
        }))
        : getHeadingDescriptors(elements.preview, markdown);
    if (!descriptors.length) {
        elements.toolsToC.innerHTML = '<div class="notes-tools-toc-empty">No headings found</div>';
        clearToolsToCHighlight(false);
        return;
    }

    const list = document.createElement('div');
    list.className = 'notes-tools-toc-list';

    descriptors.forEach((entry) => {
        const item = document.createElement('button');
        item.type = 'button';
        item.className = 'notes-tools-toc-item';
        item.dataset.level = String(entry.level);
        item.dataset.text = entry.text;
        item.dataset.occurrence = String(entry.occurrence);
        item.dataset.anchor = String(entry.anchor || '');
        if (Number.isFinite(entry.sourceLine)) item.dataset.sourceLine = String(entry.sourceLine);
        item.style.paddingLeft = `${Math.max(0, entry.level - 1) * 14 + 10}px`;
        item.textContent = entry.text;
        list.appendChild(item);
    });

    if (!list.childElementCount) {
        elements.toolsToC.innerHTML = '<div class="notes-tools-toc-empty">No headings found</div>';
        return;
    }

    elements.toolsToC.replaceChildren(list);
    syncToolsToCHighlightForMode();
}

function scrollToToolsToCHeading(entryButton) {
    if (state.viewMode === 'editor') {
        const sourceLine = Number.parseInt(String(entryButton?.dataset?.sourceLine || ''), 10);
        if (Number.isFinite(sourceLine) && sourceLine > 0) {
            jumpEditorToLine(sourceLine);
        }
        return;
    }

    const modeContext = getToolsToCModeContext();
    if (!modeContext?.headingRoot) {
        return;
    }

    const headingId = String(entryButton?.dataset?.anchor || '');
    if (headingId) {
        const byId = modeContext.headingRoot.querySelector(`#${CSS.escape(headingId)}`);
        if (byId) {
            byId.scrollIntoView({ behavior: 'smooth', block: 'start' });
            return;
        }
    }

    const level = Number.parseInt(String(entryButton?.dataset?.level || '1'), 10) || 1;
    const text = String(entryButton?.dataset?.text || '').trim().toLowerCase();
    const occurrence = Number.parseInt(String(entryButton?.dataset?.occurrence || '0'), 10) || 0;

    const candidates = Array.from(modeContext.headingRoot.querySelectorAll(`h${level}`))
        .filter((heading) => String(heading.textContent || '').trim().toLowerCase() === text);

    const target = candidates[occurrence] || candidates[0];
    if (target) {
        target.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
}

function updateToolsTabVisibility(fileType) {
    const showToC = fileType === 'markdown';
    if (elements.toolsTabToC) {
        elements.toolsTabToC.style.display = showToC ? '' : 'none';
    }

    // Frontmatter is markdown-only. Whether it is actually present depends on the
    // document content and is finalised by syncFrontmatterTab() during render;
    // here we only force the tab hidden for non-markdown files.
    if (!showToC) {
        state.frontmatter = null;
        if (elements.toolsTabFrontmatter) {
            elements.toolsTabFrontmatter.style.display = 'none';
        }
    }

    // Preserve selection unless its document-specific tab is no longer available.
    const tocActive = elements.toolsTabToC?.getAttribute('aria-selected') === 'true';
    const frontmatterActive = elements.toolsTabFrontmatter?.getAttribute('aria-selected') === 'true';
    if (!showToC && (tocActive || frontmatterActive)) {
        setToolsTab('find');
    }

    if (showToC) {
        refreshToolsToC();
    } else {
        clearToolsToCHighlight(true);
    }

    updateFindAvailability();
}

function renderMetaView() {
    const markdown = state.fileMetaMarkdown || '# Unknown file';
    elements.meta.innerHTML = marked.parse(markdown);
    processMarkdownContainer(elements.meta);
    wrapTablesForHorizontalScroll(elements.meta);
}

async function refreshFileMetaMarkdown(file) {
    if (!file) {
        state.fileMetaMarkdown = '';
        renderMetaView();
        return;
    }

    try {
        state.fileMetaMarkdown = await GetFileMetaMarkdown(file);
    } catch (err) {
        state.fileMetaMarkdown = '';
        console.error(err);
    }

    renderMetaView();
}

function setupInteractiveCheckboxes(container, isEditable) {
    const checkboxes = container.querySelectorAll('input[type="checkbox"]');
    
    checkboxes.forEach((checkbox, index) => {
        if (!isEditable) {
            checkbox.setAttribute('disabled', 'disabled');
            return;
        }

        checkbox.removeAttribute('disabled');
        checkbox.addEventListener('change', (e) => {
            toggleCheckboxInMarkdown(index, e.target.checked);
        });
    });
}

function getCollapsibleSectionWrapper(heading) {
    const wrapper = heading?.nextElementSibling;
    return wrapper?.classList?.contains('collapsible-section') ? wrapper : null;
}

function collapseCollapsibleHeading(heading) {
    const wrapper = getCollapsibleSectionWrapper(heading);
    if (!heading || !wrapper) {
        return;
    }

    if (heading.dataset.collapsed === 'true') {
        return;
    }

    const height = wrapper.scrollHeight;
    wrapper.style.transition = 'none';
    wrapper.style.maxHeight = `${height}px`;
    wrapper.offsetHeight;
    wrapper.style.transition = 'max-height 0.3s ease, opacity 0.3s ease';
    requestAnimationFrame(() => {
        wrapper.style.maxHeight = '0px';
        wrapper.style.opacity = '0';
    });
    heading.dataset.collapsed = 'true';
    heading.style.fontStyle = 'italic';
}

function expandCollapsibleHeading(heading) {
    const wrapper = getCollapsibleSectionWrapper(heading);
    if (!heading || !wrapper) {
        return;
    }

    if (heading.dataset.collapsed !== 'true') {
        wrapper.style.maxHeight = '100000px';
        wrapper.style.opacity = '1';
        heading.style.fontStyle = '';
        return;
    }

    wrapper.style.maxHeight = `${wrapper.scrollHeight}px`;
    wrapper.style.opacity = '1';
    wrapper.addEventListener('transitionend', () => {
        if (heading.dataset.collapsed !== 'true') {
            wrapper.style.maxHeight = '100000px';
        }
    }, { once: true });
    heading.dataset.collapsed = 'false';
    heading.style.fontStyle = '';
}

function setCollapsibleHeadingState(heading, collapsed) {
    if (collapsed) {
        collapseCollapsibleHeading(heading);
    } else {
        expandCollapsibleHeading(heading);
    }
}

function getRenderedSectionHeading(container, target, clientY) {
    const isAIOutput = container === elements.aiOutput;
    if (!isAIOutput && (state.currentFileType !== 'markdown' || (container !== elements.preview && container !== elements.jupyter))) {
        return null;
    }
    if (isAIOutput && target instanceof Element) {
        const markdownRoot = target.closest('.notes-ai-markdown, .notes-ai-prefix');
        if (markdownRoot && container.contains(markdownRoot)) {
            container = markdownRoot;
        } else if (target.closest('[data-block-kind]')) {
            return null;
        }
    }
    const selector = 'h1, h2, h3, h4, h5, h6';
    const clickedHeading = target instanceof Element ? target.closest(selector) : null;
    if (clickedHeading && container.contains(clickedHeading)) {
        return getCollapsibleSectionWrapper(clickedHeading) ? clickedHeading : null;
    }

    let precedingHeading = null;
    for (const heading of container.querySelectorAll(selector)) {
        const rect = heading.getBoundingClientRect();
        const precedesClick = rect.height > 0
            ? rect.top <= clientY
            : target instanceof Element && Boolean(heading.compareDocumentPosition(target) & Node.DOCUMENT_POSITION_FOLLOWING);
        if (precedesClick) {
            precedingHeading = heading;
        }
    }
    return getCollapsibleSectionWrapper(precedingHeading) ? precedingHeading : null;
}

function createCollapseSectionMenuItem(heading) {
    const collapsed = heading.dataset.collapsed === 'true';
    return {
        title: collapsed ? 'Expand section' : 'Collapse section',
        icon: collapsed ? CONTEXT_ICON_EXPAND_ALL : CONTEXT_ICON_COLLAPSE_ALL,
        onSelect: () => setCollapsibleHeadingState(heading, !collapsed),
    };
}

function setupCollapsibleHeadings(container) {
    const headings = container.querySelectorAll('h1, h2, h3, h4, h5, h6');
    const headingList = Array.from(headings);

    headingList.forEach((heading) => {
        if (getCollapsibleSectionWrapper(heading)) return;
        const level = parseInt(heading.tagName[1], 10);

        // Collect the sibling elements that belong to this heading's section.
        // Processed in document order so inner headings (h2, h3…) are already
        // children of an outer wrapper when we reach them — nextElementSibling
        // still returns the correct in-section siblings.
        const sectionEls = [];
        let sibling = heading.nextElementSibling;
        while (sibling) {
            const sibTag = sibling.tagName.toUpperCase();
            if (/^H[1-6]$/.test(sibTag) && parseInt(sibTag[1], 10) <= level) break;
            sectionEls.push(sibling);
            sibling = sibling.nextElementSibling;
        }

        if (sectionEls.length === 0) return;

        // Wrap section content in a div so we can animate it as a unit.
        const wrapper = document.createElement('div');
        wrapper.classList.add('collapsible-section');
        wrapper.style.overflowY = 'hidden';
        wrapper.style.transition = 'max-height 0.3s ease, opacity 0.3s ease';
        wrapper.style.maxHeight = '100000px';
        wrapper.style.opacity = '1';
        heading.insertAdjacentElement('afterend', wrapper);
        sectionEls.forEach((el) => wrapper.appendChild(el));
    });
}

function wrapTablesForHorizontalScroll(container) {
    if (!container) {
        return;
    }

    const tables = container.querySelectorAll('table');
    tables.forEach((table) => {
        if (!(table instanceof HTMLElement)) {
            return;
        }

        const parent = table.parentElement;
        if (!parent || parent.classList.contains('notes-table-scroll-wrap')) {
            return;
        }

        const wrapper = document.createElement('div');
        wrapper.className = 'notes-table-scroll-wrap';
        table.before(wrapper);
        wrapper.appendChild(table);
    });
}

function applyNotesTableWordWrapMode(container) {
    if (!container) {
        return;
    }
    if (state.markdownTableWordWrapMode) {
        container.classList.add('notes-table-wordwrap-on');
    } else {
        container.classList.remove('notes-table-wordwrap-on');
    }

    const filename = container === elements.aiOutput ? '' : state.currentFile;
    void setupTableColumnResizing(container, state.markdownTableWordWrapMode, filename);
}

function parseMarkdownTableRow(line) {
    const source = String(line ?? '').trim();
    const hasLeadingPipe = source.startsWith('|');
    const hasTrailingPipe = source.endsWith('|');
    const body = source.replace(/^\|/, '').replace(/\|$/, '');

    const cells = [];
    let current = '';
    let escaped = false;

    for (let i = 0; i < body.length; i += 1) {
        const char = body[i];
        if (escaped) {
            current += char;
            escaped = false;
            continue;
        }

        if (char === '\\') {
            escaped = true;
            current += char;
            continue;
        }

        if (char === '|') {
            cells.push(current.trim());
            current = '';
            continue;
        }

        current += char;
    }

    cells.push(current.trim());

    return { cells, hasLeadingPipe, hasTrailingPipe };
}

function serializeMarkdownTableRow(cells, hasLeadingPipe = true, hasTrailingPipe = true) {
    const escapedCells = cells.map((cell) => String(cell ?? '')
        .replace(/\n/g, ' ')
        .replace(/\|/g, '\\|')
        .trim());

    const core = ` ${escapedCells.join(' | ')} `;
    if (hasLeadingPipe && hasTrailingPipe) {
        return `|${core}|`;
    }
    if (hasLeadingPipe) {
        return `|${core}`;
    }
    if (hasTrailingPipe) {
        return `${core}|`;
    }
    return core;
}

function isMarkdownTableSeparatorLine(line) {
    return /^\s*\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|?\s*$/.test(String(line ?? ''));
}

function findMarkdownTableBlocks(markdown) {
    const lines = String(markdown ?? '').split('\n');
    const blocks = [];

    for (let i = 0; i < lines.length - 1; i += 1) {
        const headerLine = lines[i];
        const separatorLine = lines[i + 1];

        if (!String(headerLine).includes('|') || !isMarkdownTableSeparatorLine(separatorLine)) {
            continue;
        }

        const rowLineIndexes = [i];
        let j = i + 2;
        while (j < lines.length) {
            const rowLine = lines[j];
            if (!String(rowLine).includes('|') || String(rowLine).trim() === '') {
                break;
            }
            rowLineIndexes.push(j);
            j += 1;
        }

        blocks.push({
            rowLineIndexes,
            separatorLineIndex: i + 1,
        });

        i = j - 1;
    }

    return blocks;
}

function commitMarkdownTableEdit(lines, label) {
    replaceMainDocumentText(lines.join('\n'), { source: 'markdown-table-edit' });
    state.viewTainted = true;
    void saveFile();
}

function updateMarkdownTableCell(block, sourceRowIndex, columnIndex, value) {
    if (!block || !Array.isArray(block.rowLineIndexes)) {
        return false;
    }

    const lineIndex = block.rowLineIndexes[sourceRowIndex];
    if (!Number.isInteger(lineIndex)) {
        return false;
    }

    const lines = getMainEditorValue().split('\n');
    if (lineIndex < 0 || lineIndex >= lines.length) {
        return false;
    }

    const parsed = parseMarkdownTableRow(lines[lineIndex]);
    while (parsed.cells.length <= columnIndex) {
        parsed.cells.push('');
    }

    parsed.cells[columnIndex] = String(value ?? '').trim();
    lines[lineIndex] = serializeMarkdownTableRow(parsed.cells, parsed.hasLeadingPipe, parsed.hasTrailingPipe);
    commitMarkdownTableEdit(lines, 'Edit markdown table cell');

    return true;
}

function insertMarkdownRowAfter(block, sourceRowIndex) {
    if (!block || !Array.isArray(block.rowLineIndexes)) return;

    const lines = getMainEditorValue().split('\n');
    // Determine column count from the header row
    const headerParsed = parseMarkdownTableRow(lines[block.rowLineIndexes[0]]);
    const colCount = headerParsed.cells.length;
    const newRow = serializeMarkdownTableRow(
        Array(colCount).fill(''),
        headerParsed.hasLeadingPipe,
        headerParsed.hasTrailingPipe,
    );

    // Find the line to insert after
    const lastRowIdx = block.rowLineIndexes[sourceRowIndex];
    // If inserting after header (sourceRowIndex === 0), insert after separator
    const insertAfterLine = sourceRowIndex === 0 ? block.separatorLineIndex : lastRowIdx;
    lines.splice(insertAfterLine + 1, 0, newRow);
    commitMarkdownTableEdit(lines, 'Insert markdown table row');
    renderJupyterView();
}

function insertMarkdownColumnAfter(block, columnIndex) {
    if (!block || !Array.isArray(block.rowLineIndexes)) return;

    const lines = getMainEditorValue().split('\n');

    // Insert into each data/header row
    block.rowLineIndexes.forEach(lineIdx => {
        const parsed = parseMarkdownTableRow(lines[lineIdx]);
        while (parsed.cells.length <= columnIndex) parsed.cells.push('');
        parsed.cells.splice(columnIndex + 1, 0, '');
        lines[lineIdx] = serializeMarkdownTableRow(parsed.cells, parsed.hasLeadingPipe, parsed.hasTrailingPipe);
    });

    // Insert into separator row
    const sepParsed = parseMarkdownTableRow(lines[block.separatorLineIndex]);
    while (sepParsed.cells.length <= columnIndex) sepParsed.cells.push('---');
    sepParsed.cells.splice(columnIndex + 1, 0, '---');
    lines[block.separatorLineIndex] = serializeMarkdownTableRow(sepParsed.cells, sepParsed.hasLeadingPipe, sepParsed.hasTrailingPipe);

    commitMarkdownTableEdit(lines, 'Insert markdown table column');
    renderJupyterView();
}

function deleteMarkdownRow(block, sourceRowIndex) {
    if (!block || !Array.isArray(block.rowLineIndexes)) {
        return;
    }

    // Keep at least one row in the table body (header + one data row minimum).
    if (sourceRowIndex <= 0 || block.rowLineIndexes.length <= 2) {
        return;
    }

    const lines = getMainEditorValue().split('\n');
    const lineIndex = block.rowLineIndexes[sourceRowIndex];
    if (!Number.isInteger(lineIndex) || lineIndex < 0 || lineIndex >= lines.length) {
        return;
    }

    lines.splice(lineIndex, 1);
    commitMarkdownTableEdit(lines, 'Delete markdown table row');
    renderJupyterView();
}

function deleteMarkdownColumn(block, columnIndex) {
    if (!block || !Array.isArray(block.rowLineIndexes)) {
        return;
    }

    const lines = getMainEditorValue().split('\n');
    const headerParsed = parseMarkdownTableRow(lines[block.rowLineIndexes[0]]);
    if (columnIndex < 0 || columnIndex >= headerParsed.cells.length || headerParsed.cells.length <= 1) {
        return;
    }

    block.rowLineIndexes.forEach(lineIdx => {
        const parsed = parseMarkdownTableRow(lines[lineIdx]);
        if (columnIndex < parsed.cells.length) {
            parsed.cells.splice(columnIndex, 1);
        }
        lines[lineIdx] = serializeMarkdownTableRow(parsed.cells, parsed.hasLeadingPipe, parsed.hasTrailingPipe);
    });

    const sepParsed = parseMarkdownTableRow(lines[block.separatorLineIndex]);
    if (columnIndex < sepParsed.cells.length) {
        sepParsed.cells.splice(columnIndex, 1);
    }
    lines[block.separatorLineIndex] = serializeMarkdownTableRow(sepParsed.cells, sepParsed.hasLeadingPipe, sepParsed.hasTrailingPipe);

    commitMarkdownTableEdit(lines, 'Delete markdown table column');
    renderJupyterView();
}

function findMarkdownFunctionCodeBlock(container, functionName) {
    if (!container || !functionName) {
        return null;
    }

    const headings = Array.from(container.querySelectorAll('h1, h2, h3, h4, h5, h6'));
    const targetHeading = headings.find((h) => String(h.textContent || '').trim().toLowerCase() === String(functionName).trim().toLowerCase());
    if (!targetHeading) {
        return null;
    }

    let sibling = targetHeading.nextElementSibling;
    while (sibling) {
        if (/^H[1-6]$/.test(sibling.tagName)) {
            break;
        }

        // Check for a .jupyter-code-block — either directly or nested inside a
        // collapsible-section wrapper created by setupCollapsibleHeadings.
        const jupBlock = (sibling.classList?.contains('jupyter-code-block') ? sibling : null)
            ?? sibling.querySelector?.('.jupyter-code-block');
        if (jupBlock) {
            const blockId = jupBlock.dataset ? jupBlock.dataset.blockId : '';
            const blockState = blockId ? state.jupyterCodeBlocks?.[blockId] : null;
            if (blockState) {
                return {
                    code: String(blockState.currentContent ?? blockState.originalContent ?? ''),
                    language: String(blockState.language || ''),
                };
            }
        }

        const codeNode = sibling.tagName === 'PRE'
            ? sibling.querySelector('code')
            : sibling.querySelector('pre code');
        if (codeNode) {
            const langClass = Array.from(codeNode.classList || []).find(cls => cls.startsWith('language-'));
            const language = langClass ? langClass.replace('language-', '') : '';
            return {
                code: String(codeNode.textContent || ''),
                language,
            };
        }

        sibling = sibling.nextElementSibling;
    }

    return null;
}

async function resolveRuntimeForFunctionLanguage(language) {
    const lang = String(language || '').trim();
    if (!lang) {
        return 'language unknown';
    }

    try {
        const matches = await GetLanguageDescriptions(lang);
        if (Array.isArray(matches) && matches.length > 0) {
            return matches[0];
        }
    } catch (err) {
        console.warn('Unable to resolve function runtime:', err);
    }

    return lang;
}

function parseA1ColumnToIndex(columnLetters) {
    let colIdx = 0;
    const normalized = String(columnLetters || '').toUpperCase();
    for (let i = 0; i < normalized.length; i += 1) {
        colIdx *= 26;
        colIdx += normalized.charCodeAt(i) - 65 + 1;
    }
    return colIdx - 1;
}

function buildRefFromRowCol(row, col) {
    return getCellReference(row, col);
}

function parseCoordinateReference(ref, row, col) {
    const source = String(ref || '').trim();
    if (!source) {
        return null;
    }

    const r1c1Pattern = /^R(\[(-?\d+)\]|(\d+))C(\[(-?\d+)\]|(\d+))$/i;
    const r1c1Match = source.match(r1c1Pattern);
    if (r1c1Match) {
        const targetRow = r1c1Match[2] !== undefined ? row + parseInt(r1c1Match[2], 10) : parseInt(r1c1Match[3], 10) - 1;
        const targetCol = r1c1Match[5] !== undefined ? col + parseInt(r1c1Match[5], 10) : parseInt(r1c1Match[6], 10) - 1;
        return { row: targetRow, col: targetCol };
    }

    const a1Match = source.match(/^([A-Z]+)(\d+)$/i);
    if (a1Match) {
        return {
            row: parseInt(a1Match[2], 10) - 1,
            col: parseA1ColumnToIndex(a1Match[1]),
        };
    }

    return null;
}

function getFormulaDependencies(formula, row, col, rowCount, colCount) {
    const refs = new Set();
    const source = String(formula || '');
    if (!isTableFormula(source)) {
        return refs;
    }

    const fnCall = parseTableFunctionCall(source);
    const candidates = fnCall ? fnCall.args : [source.slice(1)];

    for (const candidate of candidates) {
        const arg = String(candidate || '').trim();
        if (!arg) {
            continue;
        }

        const a1Range = arg.match(/^([A-Z]+)(\d+):([A-Z]+)(\d+)$/i);
        if (a1Range) {
            const startCol = parseA1ColumnToIndex(a1Range[1]);
            const startRow = parseInt(a1Range[2], 10) - 1;
            const endCol = parseA1ColumnToIndex(a1Range[3]);
            const endRow = parseInt(a1Range[4], 10) - 1;
            const rowStart = Math.max(0, Math.min(startRow, endRow));
            const rowEnd = Math.min(rowCount - 1, Math.max(startRow, endRow));
            const colStart = Math.max(0, Math.min(startCol, endCol));
            const colEnd = Math.min(colCount - 1, Math.max(startCol, endCol));

            for (let rowIdx = rowStart; rowIdx <= rowEnd; rowIdx += 1) {
                for (let colIdx = colStart; colIdx <= colEnd; colIdx += 1) {
                    refs.add(buildRefFromRowCol(rowIdx, colIdx));
                }
            }
            continue;
        }

        const wholeColumnRange = arg.match(/^([A-Z]+):([A-Z]+)$/i);
        if (wholeColumnRange) {
            const startCol = parseA1ColumnToIndex(wholeColumnRange[1]);
            const endCol = parseA1ColumnToIndex(wholeColumnRange[2]);
            const colStart = Math.max(0, Math.min(startCol, endCol));
            const colEnd = Math.min(colCount - 1, Math.max(startCol, endCol));
            for (let rowIdx = 0; rowIdx < rowCount; rowIdx += 1) {
                for (let colIdx = colStart; colIdx <= colEnd; colIdx += 1) {
                    refs.add(buildRefFromRowCol(rowIdx, colIdx));
                }
            }
            continue;
        }

        const wholeRowRange = arg.match(/^(\d+):(\d+)$/);
        if (wholeRowRange) {
            const startRow = parseInt(wholeRowRange[1], 10) - 1;
            const endRow = parseInt(wholeRowRange[2], 10) - 1;
            const rowStart = Math.max(0, Math.min(startRow, endRow));
            const rowEnd = Math.min(rowCount - 1, Math.max(startRow, endRow));
            for (let rowIdx = rowStart; rowIdx <= rowEnd; rowIdx += 1) {
                for (let colIdx = 0; colIdx < colCount; colIdx += 1) {
                    refs.add(buildRefFromRowCol(rowIdx, colIdx));
                }
            }
            continue;
        }

        const directCoordinate = parseCoordinateReference(arg, row, col);
        if (directCoordinate) {
            if (
                directCoordinate.row >= 0 && directCoordinate.row < rowCount &&
                directCoordinate.col >= 0 && directCoordinate.col < colCount
            ) {
                refs.add(buildRefFromRowCol(directCoordinate.row, directCoordinate.col));
            }
            continue;
        }

        const embeddedA1Refs = arg.match(/\b([A-Z]+)(\d+)\b/g) || [];
        for (const token of embeddedA1Refs) {
            const parsed = parseCoordinateReference(token, row, col);
            if (parsed && parsed.row >= 0 && parsed.row < rowCount && parsed.col >= 0 && parsed.col < colCount) {
                refs.add(buildRefFromRowCol(parsed.row, parsed.col));
            }
        }

        const embeddedR1C1Refs = arg.match(/R(\[[-+]?\d+\]|\d+)C(\[[-+]?\d+\]|\d+)/gi) || [];
        for (const token of embeddedR1C1Refs) {
            const parsed = parseCoordinateReference(token, row, col);
            if (parsed && parsed.row >= 0 && parsed.row < rowCount && parsed.col >= 0 && parsed.col < colCount) {
                refs.add(buildRefFromRowCol(parsed.row, parsed.col));
            }
        }
    }

    return refs;
}

async function runMarkdownTableFunction(container, fnName, fnArgs, row, col, rows, cellId, file = state.currentFile) {
    const block = findMarkdownFunctionCodeBlock(container, fnName);
    if (!block) {
        return '#ERR';
    }

    const currentCellId = String(cellId || getCellReference(row, col));

    // Create a function executor for nested function calls
    const functionExecutor = async (nestedFnName, nestedFnArgs, nestedRow, nestedCol) => {
        return runMarkdownTableFunction(container, nestedFnName, nestedFnArgs, nestedRow, nestedCol, rows, currentCellId, file);
    };

    // Resolve arguments with support for nested function calls
    const resolvedArgArrays = await Promise.all(
        fnArgs.map((arg) => resolveTableFunctionArgsAsync(functionExecutor, arg, row, col, rows))
    );
    const resolvedArgs = resolvedArgArrays.flatMap((arr) => arr);

    const runtime = await resolveRuntimeForFunctionLanguage(block.language);

    try {
        const result = await RunFunction(file, fnName, currentCellId, block.code, resolvedArgs, runtime);

        // Backward compatibility: support both structured and string responses.
        const isStructured = result && typeof result === 'object';
        const isError = isStructured
            ? Boolean(result.IsError ?? result.isError)
            : false;
        const output = isStructured
            ? String(result.Output ?? result.output ?? '')
            : String(result ?? '');

        if (isError) {
            return `#ERR ${output}`;
        }

        return output;
    } catch (err) {
        const stderr = String(err ?? '');
        return `#ERR ${stderr}`;
    }
}

function renderMarkdownTableCell(value) {
    const renderer = new marked.Renderer();
    renderer.link = function ({ tokens }) {
        return this.parser.parseInline(tokens);
    };
    renderer.html = ({ text }) => escapeHtml(text);
    return marked.parseInline(String(value ?? ''), { renderer });
}

function unwrapTableCellLinks(cell) {
    cell.querySelectorAll('a').forEach((link) => {
        link.replaceWith(...Array.from(link.childNodes));
    });
}

async function evaluateTableFormulasInPlace(container) {
    const file = state.currentFile;
    const isJsdom = typeof navigator !== 'undefined' && /jsdom/i.test(String(navigator.userAgent || ''));

    const flushUiPaint = () => new Promise((resolve) => {
        if (typeof requestAnimationFrame === 'function') {
            requestAnimationFrame(() => resolve());
            return;
        }
        Promise.resolve().then(resolve);
    });

    const tables = Array.from(container.querySelectorAll('table'));
    for (const table of tables) {
        // Build 2D array from DOM
        const rows = [];
        const headerRow = table.querySelector('thead tr');
        if (headerRow) {
            rows.push(Array.from(headerRow.querySelectorAll('th, td')).map(c => String(c.textContent || '').trim()));
        }
        Array.from(table.querySelectorAll('tbody tr')).forEach(tr => {
            rows.push(Array.from(tr.querySelectorAll('td, th')).map(c => String(c.textContent || '').trim()));
        });

        ensureCellRefStyle();

        // Collect all table cells first, then evaluate formulas in parallel.
        const tableCells = [];

        if (headerRow) {
            const headerCells = Array.from(headerRow.querySelectorAll('th, td'));
            for (const [colIdx, cell] of headerCells.entries()) {
                tableCells.push({
                    cell,
                    row: 0,
                    col: colIdx,
                    val: rows[0][colIdx],
                    ref: getCellReference(0, colIdx),
                });
            }
        }

        const bodyRowOffset = headerRow ? 1 : 0;
        const bodyRows = Array.from(table.querySelectorAll('tbody tr'));
        for (const [rIdx, tr] of bodyRows.entries()) {
            const cells = Array.from(tr.querySelectorAll('td, th'));
            for (const [colIdx, cell] of cells.entries()) {
                const row = bodyRowOffset + rIdx;
                tableCells.push({
                    cell,
                    row,
                    col: colIdx,
                    val: rows[row][colIdx],
                    ref: getCellReference(row, colIdx),
                });
            }
        }

        // Render non-formula cells immediately so UI has baseline content.
        for (const { cell, val, ref } of tableCells) {
            if (!isTableFormula(val)) {
                unwrapTableCellLinks(cell);
                cell.innerHTML = `<span class="notes-table-cell-wrap"><span>${cell.innerHTML}</span><span class="notes-cellref">${ref}</span></span>`;
            }
        }

        const formulaTasks = tableCells.filter(({ val }) => isTableFormula(val));

        // Render formula placeholders so users see immediate progress while async calls run.
        for (const { cell, val, ref } of formulaTasks) {
            cell.dataset.formula = val;
            cell.innerHTML = `<span class="notes-table-cell-wrap"><span>...</span><span class="notes-cellref">${ref}</span></span>`;
        }

        const formulaTaskMap = new Map(formulaTasks.map((task) => [task.ref, task]));
        const dependencyRefsMap = new Map();

        const rowCount = rows.length;
        const colCount = rows[0]?.length || 0;

        for (const task of formulaTasks) {
            const rawDeps = getFormulaDependencies(task.val, task.row, task.col, rowCount, colCount);
            const filtered = new Set(Array.from(rawDeps).filter((depRef) => depRef !== task.ref && formulaTaskMap.has(depRef)));
            dependencyRefsMap.set(task.ref, filtered);
        }

        const indegree = new Map();
        const dependents = new Map();
        for (const task of formulaTasks) {
            const deps = dependencyRefsMap.get(task.ref) || new Set();
            indegree.set(task.ref, deps.size);
            for (const depRef of deps) {
                if (!dependents.has(depRef)) {
                    dependents.set(depRef, new Set());
                }
                dependents.get(depRef).add(task.ref);
            }
        }

        const computedRefs = new Set();
        const ready = formulaTasks
            .map((task) => task.ref)
            .filter((ref) => (indegree.get(ref) || 0) === 0);

        while (ready.length > 0) {
            const batchRefs = ready.splice(0, ready.length);
            const batchTasks = batchRefs.map((ref) => formulaTaskMap.get(ref)).filter(Boolean);

            await Promise.allSettled(batchTasks.map(async (task) => {
                const { cell, row, col, val, ref } = task;
                const fnCall = parseTableFunctionCall(val);
                const content = fnCall
                    ? await runMarkdownTableFunction(container, fnCall.fnName, fnCall.args, row, col, rows, ref, file)
                    : evaluateTableFormula(val, row, col, rows);

                rows[row][col] = String(content ?? '');
                cell.dataset.formula = val;
                cell.innerHTML = `<span class="notes-table-cell-wrap"><span>${renderMarkdownTableCell(content)}</span><span class="notes-cellref">${ref}</span></span>`;
                computedRefs.add(ref);
            }));

            // Yield so the browser can paint completed cells before the next dependency batch.
            if (!isJsdom) {
                await flushUiPaint();
            }

            for (const ref of batchRefs) {
                const nextRefs = dependents.get(ref);
                if (!nextRefs) {
                    continue;
                }

                for (const dependentRef of nextRefs) {
                    const nextIndegree = (indegree.get(dependentRef) || 0) - 1;
                    indegree.set(dependentRef, nextIndegree);
                    if (nextIndegree === 0) {
                        ready.push(dependentRef);
                    }
                }
            }
        }

        // Circular or unresolved dependencies fallback: evaluate remaining formulas best-effort.
        const remaining = formulaTasks.filter((task) => !computedRefs.has(task.ref));
        for (const task of remaining) {
            const { cell, row, col, val, ref } = task;
            const fnCall = parseTableFunctionCall(val);
            const content = fnCall
                ? await runMarkdownTableFunction(container, fnCall.fnName, fnCall.args, row, col, rows, ref, file)
                : evaluateTableFormula(val, row, col, rows);
            rows[row][col] = String(content ?? '');
            cell.dataset.formula = val;
            cell.innerHTML = `<span class="notes-table-cell-wrap"><span>${renderMarkdownTableCell(content)}</span><span class="notes-cellref">${ref}</span></span>`;

            if (!isJsdom) {
                await flushUiPaint();
            }
        }
    }
}

function setupInteractiveMarkdownTables(container, isEditable) {
    const blocks = findMarkdownTableBlocks(getMainEditorValue());

    setupInteractiveTableCells(
        container,
        isEditable,
        (_table, tableIndex) => {
            const block = blocks[tableIndex];
            if (!block) {
                return null;
            }

            return (sourceRowIndex, columnIndex, value) => {
                updateMarkdownTableCell(block, sourceRowIndex, columnIndex, value);
            };
        },
        () => {
            if (state.viewMode === 'jupyter') {
                renderJupyterView();
            }
        },
    );
}

function getTableCellTextContent(cell) {
    if (!cell) {
        return '';
    }

    const clone = cell.cloneNode(true);
    if (!(clone instanceof HTMLElement)) {
        return String(cell.textContent || '').trim();
    }

    clone.querySelectorAll('.notes-sort-icon, .notes-table-col-resize-handle, .notes-cellref').forEach((el) => el.remove());
    return String(clone.textContent || '').trim();
}

function getTableHeadingValues(table) {
    if (!table) {
        return [];
    }

    const headerCells = Array.from(table.querySelectorAll('thead tr:first-child th'));
    if (headerCells.length > 0) {
        return headerCells.map((cell) => getTableCellTextContent(cell));
    }

    const firstRowCells = Array.from(table.querySelectorAll('tr:first-child th, tr:first-child td'));
    return firstRowCells.map((cell) => getTableCellTextContent(cell));
}

function ensureTableColGroup(table, columnCount) {
    let colgroup = table.querySelector('colgroup.notes-table-colgroup');
    if (!colgroup) {
        colgroup = document.createElement('colgroup');
        colgroup.className = 'notes-table-colgroup';
        table.prepend(colgroup);
    }

    while (colgroup.children.length < columnCount) {
        colgroup.appendChild(document.createElement('col'));
    }
    while (colgroup.children.length > columnCount) {
        colgroup.lastElementChild.remove();
    }

    return colgroup;
}

function applyTableColumnWidths(table, widths) {
    if (!table || !Array.isArray(widths) || widths.length === 0) {
        return;
    }

    const headerCells = Array.from(table.querySelectorAll('thead tr:first-child th'));
    if (headerCells.length === 0) {
        return;
    }

    const colgroup = ensureTableColGroup(table, headerCells.length);
    const cols = Array.from(colgroup.querySelectorAll('col'));

    cols.forEach((col, idx) => {
        const width = Number(widths[idx]);
        if (!Number.isFinite(width) || width <= 0) {
            return;
        }
        col.style.width = `${Math.max(48, Math.round(width))}px`;
    });

    table.style.tableLayout = 'fixed';
}

function collectTableColumnWidths(table) {
    if (!table) {
        return [];
    }

    const headerCells = Array.from(table.querySelectorAll('thead tr:first-child th'));
    if (headerCells.length === 0) {
        return [];
    }

    const colgroup = ensureTableColGroup(table, headerCells.length);
    const cols = Array.from(colgroup.querySelectorAll('col'));

    return cols.map((col, idx) => {
        const fromStyle = Number.parseFloat(col.style.width || '');
        if (Number.isFinite(fromStyle) && fromStyle > 0) {
            return Math.max(48, Math.round(fromStyle));
        }

        const measured = headerCells[idx]?.getBoundingClientRect?.().width || 0;
        return Math.max(48, Math.round(measured));
    });
}

function getAITableIdentity(table) {
    if (!elements.aiOutput?.contains(table)) return null;
    const markdown = table.closest('.notes-ai-markdown, .notes-ai-prefix');
    const block = table.closest('[data-block-id]') || markdown;
    const sessionId = Number(block?.dataset.sessionId) || 0;
    const promptId = Number(block?.dataset.promptId) || 0;
    const runId = Number(block?.dataset.runId) || 0;
    if (!sessionId || (!promptId && !runId)) return null;
    const workspace = block.dataset.workspace || state.currentWorkspaceName || state.currentWorkspaceKey || '';
    const index = Array.from((markdown || block).querySelectorAll('table')).indexOf(table);
    return {
        workspace,
        document: JSON.stringify(['ai-table', workspace, sessionId,
            promptId || `run:${runId}`, markdown?.classList.contains('notes-ai-prefix') ? 'request' : block.dataset.blockId, index]),
    };
}

async function finalizeAITableIdentities(payload) {
    if (!(Number(payload?.promptId) > 0)) return;
    const blocks = Array.from(elements.aiOutput.querySelectorAll('[data-run-id]'))
        .filter((block) => Number(block.dataset.runId) === Number(payload.runId));
    const tables = blocks.flatMap((block) => Array.from(block.querySelectorAll('table[data-table-surface="ai"]')))
        .filter((table, index, all) => all.indexOf(table) === index);
    for (const block of blocks) {
        block.dataset.promptId = String(payload.promptId);
    }
    for (const table of tables) {
        const identity = getAITableIdentity(table);
        if (!identity || identity.document === table.dataset.tableDocument) continue;
        table.dataset.tableDocument = identity.document;
        try {
            const request = notesTableRequest(table, true);
            const spec = JSON.parse(table.dataset.tableGraphSpec || 'null');
            if (spec) {
                const result = await NotesTableSetGraph(request, spec);
                if (result?.error) throw new Error(result.error);
            }
            if (table.dataset.tableFilter || Number(table.dataset.sortColumn)) {
                const result = await NotesTableFilter(request, request.filter);
                if (result?.error) throw new Error(result.error);
            }
            if (table.querySelector('col[style*="width"]')) {
                const headings = Array.from(table.querySelectorAll('thead th')).map(getTableCellTextContent);
                await SetNotesColumnWidths(identity.document, 'ai-panel', headings,
                    state.markdownTableWordWrapMode, collectTableColumnWidths(table));
            }
        } catch (err) {
            console.error('Failed to preserve AI table settings after completion:', err);
            notifyTerminal('Failed to preserve AI table settings after completion', 'error');
        }
    }
}

async function setupTableColumnResizing(container, wrapped, filename = state.currentFile, viewNameOverride = '') {
    if (!container || filename === null) {
        return;
    }

    const viewName = viewNameOverride || (container === elements.preview
        ? 'view'
        : container === elements.jupyter
            ? 'run'
            : container === elements.csvView
                ? 'csv'
                : container === elements.aiOutput
                    ? 'ai-panel'
                    : String(state.viewMode || 'view'));

    const tables = Array.from(container.querySelectorAll('table'));
    for (const table of tables) {
        if (!(table instanceof HTMLTableElement)) {
            continue;
        }

        const headerCells = Array.from(table.querySelectorAll('thead tr:first-child th'));
        if (headerCells.length === 0) {
            continue;
        }

        const headings = getTableHeadingValues(table);
        if (headings.length === 0) {
            continue;
        }

        const aiIdentity = getAITableIdentity(table);
        if (elements.aiOutput?.contains(table) && !aiIdentity) continue;
        const widthFilename = aiIdentity?.document ?? filename;
        const widthViewName = aiIdentity ? 'ai-panel' : viewName;
        const storedWidths = await GetNotesColumnWidths(widthFilename, widthViewName, headings, wrapped).catch(() => []);
        if (Array.isArray(storedWidths) && storedWidths.length > 0) {
            applyTableColumnWidths(table, storedWidths);
        }

        const colgroup = ensureTableColGroup(table, headerCells.length);
        const cols = Array.from(colgroup.querySelectorAll('col'));

        const startColumnResize = (columnIndex, event, handle) => {
            if (event.button !== 0) {
                return;
            }

            event.preventDefault();
            event.stopPropagation();

            table.dataset.resizeDragActive = 'true';
            handle?.classList.add('is-dragging');

            const column = cols[columnIndex];
            if (!column) {
                return;
            }

            const widthAnchor = headerCells[columnIndex];
            const currentWidth = Number.parseFloat(column.style.width || '') || widthAnchor.getBoundingClientRect().width;
            const startX = event.clientX;
            table.style.tableLayout = 'fixed';

            const onMouseMove = (moveEvent) => {
                const deltaX = moveEvent.clientX - startX;
                const nextWidth = Math.max(48, Math.round(currentWidth + deltaX));
                column.style.width = `${nextWidth}px`;
            };

            const onMouseUp = () => {
                window.removeEventListener('mousemove', onMouseMove);
                window.removeEventListener('mouseup', onMouseUp);
                handle?.classList.remove('is-dragging');

                setTimeout(() => {
                    delete table.dataset.resizeDragActive;
                }, 0);

                const widths = collectTableColumnWidths(table);
                if (widths.length === 0) {
                    return;
                }

                SetNotesColumnWidths(getAITableIdentity(table)?.document ?? widthFilename, widthViewName, headings, wrapped, widths).catch((err) => {
                    console.error('Failed to save table column widths:', err);
                });
            };

            window.addEventListener('mousemove', onMouseMove);
            window.addEventListener('mouseup', onMouseUp);
        };

        const rows = Array.from(table.querySelectorAll('tr'));
        rows.forEach((row) => {
            const cells = Array.from(row.querySelectorAll('th, td'));
            cells.forEach((cell, columnIndex) => {
                if (columnIndex >= headerCells.length) {
                    return;
                }

                let handle = cell.querySelector('.notes-table-col-resize-handle');
                if (!handle) {
                    handle = document.createElement('span');
                    handle.className = 'notes-table-col-resize-handle';
                    handle.setAttribute('aria-hidden', 'true');
                    cell.appendChild(handle);
                }

                if (handle.dataset.bound === 'true') {
                    return;
                }
                handle.dataset.bound = 'true';
                handle.addEventListener('mousedown', (event) => startColumnResize(columnIndex, event, handle));
            });
        });
    }
}

// Sorting and filtering are owned by the Go engine (tablecore), the same one the
// terminal table widget uses, so both surfaces behave identically. Go returns an
// order of source row indices; nothing here rewrites a cell, which is what keeps
// formulas, cell editing, resize handles and word wrap intact.
// See adr/0028-go-backed-notes-table-sorting.md.
const NOTES_TABLE_EPHEMERAL_CELLS = 2000;
const NOTES_TABLE_SORT_ASC = '\u2191';
const NOTES_TABLE_SORT_DESC = '\u2193';

// Source order, which stops matching DOM order as soon as a sort is applied.
function notesTableSourceRows(table) {
    return Array.from(table.querySelectorAll('tbody tr'))
        .sort((a, b) => (Number(a.dataset.sourceRow) || 0) - (Number(b.dataset.sourceRow) || 0));
}

function notesTableSeed(table) {
    return {
        headings: Array.from(table.querySelectorAll('thead th')).map((th) => getTableCellTextContent(th)),
        rows: notesTableSourceRows(table).map((row) =>
            Array.from(row.querySelectorAll('td, th')).map((cell) => getTableCellTextContent(cell))),
    };
}

function notesTableIsSmall(table) {
    const columns = table.querySelectorAll('thead th').length;
    const rows = table.querySelectorAll('tbody tr').length;
    return columns * (rows + 1) <= NOTES_TABLE_EPHEMERAL_CELLS;
}

function notesTableRequest(table, includeSeed) {
    return {
        key: {
            surface: table.dataset.tableSurface || '',
            document: table.dataset.tableDocument || '',
            index: Number(table.dataset.tableIndex) || 0,
        },
        workspace: table.dataset.tableSurface === 'ai'
            ? getAITableIdentity(table)?.workspace || state.currentWorkspaceKey || ''
            : state.currentWorkspaceKey || '',
        stateIndex: Number(table.dataset.tableStateIndex) || 0,
        seed: includeSeed ? notesTableSeed(table) : null,
        sortColumn: Number(table.dataset.sortColumn) || 0,
        sortDesc: table.dataset.sortDesc === 'true',
        filter: table.dataset.tableFilter || '',
    };
}

function renderNotesTableFilterFooter(table) {
    const extras = notesTableExtras(table, true);
    if (!extras) {
        return;
    }
    extras.querySelectorAll(':scope > .notes-table-sql-filter-footer').forEach((footer) => footer.remove());
    const filter = String(table.dataset.tableFilter || '').trim();
    if (!filter) {
        return;
    }

    const footer = document.createElement('button');
    footer.type = 'button';
    footer.className = 'notes-table-sql-filter-footer';
    footer.title = 'Edit SQL filter';
    const label = document.createElement('strong');
    label.textContent = 'SQL filter: ';
    const value = document.createElement('code');
    value.textContent = filter;
    footer.append(label, value);
    footer.addEventListener('click', () => {
        void promptNotesTableFilter(table);
    });
    const graph = extras.querySelector(':scope > .notes-table-graph');
    extras.insertBefore(footer, graph || null);
}

function notesTableExtras(table, create = false) {
    if (!table?.parentElement) {
        return null;
    }
    const scrollWrap = table.closest('.notes-table-scroll-wrap');
    const parent = scrollWrap?.parentElement || table.parentElement;
    const tableIndex = String(table.dataset.tableIndex || '0');
    let extras = Array.from(parent.children).find((child) =>
        child.classList?.contains('notes-table-extras') && child.dataset.tableIndex === tableIndex);
    if (!extras && create) {
        extras = document.createElement('div');
        extras.className = 'notes-table-extras';
        extras.dataset.tableIndex = tableIndex;
        const anchor = scrollWrap || table;
        anchor.insertAdjacentElement('afterend', extras);
    }
    return extras || null;
}

function notesTableNumericColumns(table) {
    const seed = notesTableSeed(table);
    return seed.headings.flatMap((_, column) => {
        const values = seed.rows.map((row) => String(row[column] || '').trim()).filter(Boolean);
        const isNumeric = (value) => {
            const normalized = value.replace(/[\p{Sc},]/gu, '').trim();
            return normalized !== '' && Number.isFinite(Number(normalized));
        };
        if (values.length === 0 || !values.every(isNumeric)) {
            return [];
        }
        return [column];
    });
}

const notesGraphTypes = [
    { value: 'line', label: 'Line' },
    { value: 'area', label: 'Area' },
    { value: 'bar', label: 'Bar' },
    { value: 'horizontal-bar', label: 'Horizontal bar' },
    { value: 'scatter', label: 'Scatter' },
    { value: 'pie', label: 'Pie' },
    { value: 'doughnut', label: 'Doughnut' },
];

const notesGraphRowPageSize = 100;

function notesGraphField(labelText) {
    const field = document.createElement('div');
    field.className = 'notes-graph-field';
    const label = document.createElement('div');
    label.className = 'notes-graph-field-label';
    label.textContent = labelText;
    field.appendChild(label);
    return { field, label };
}

function setNotesModalTitle(text, markdownHeading = true) {
    const title = elements.modal.querySelector('#notes-modal-title');
    title.classList.toggle('markdown-body', markdownHeading);
    title.classList.toggle('notes-graph-title', markdownHeading);
    if (!markdownHeading) {
        title.textContent = text;
        return;
    }

    setMarkdownModalHeading(title, text);
}

function setMarkdownModalHeading(container, text) {
    container.classList.add('markdown-body');
    const heading = document.createElement('h3');
    heading.textContent = text;
    container.replaceChildren(heading);
}

function openNotesTableGraphPrompt(table) {
    const headings = Array.from(table.querySelectorAll('thead th')).map((cell) => getTableCellTextContent(cell));
    if (headings.length < 2) {
        notifyTerminal('A graph needs a category column and at least one value column.', 'warn');
        return;
    }

    let existing = null;
    try {
        existing = JSON.parse(table.dataset.tableGraphSpec || 'null');
    } catch {
        existing = null;
    }
    const selectedColumns = existing
        ? new Set([...(existing.categoryColumns?.length ? existing.categoryColumns : [existing.categoryColumn]), ...(existing.valueColumns || [])])
        : new Set(headings.map((_, index) => index));
    const tableRows = notesTableSourceRows(table);
    const savedRows = Array.isArray(existing?.rowIndices) ? existing.rowIndices : [];
    const prompt = {
        table,
        headings,
        rows: tableRows,
        type: existing?.type || 'line',
        title: existing?.title || '',
        selectedColumns,
        columnQuery: '',
        selectedRows: savedRows.length > 0 ? new Set(savedRows) : null,
        rowQuery: '',
        rowLimit: notesGraphRowPageSize,
        rowList: null,
        rowCount: null,
    };

    state.graphPromptHandler = (spec) => {
        if (spec !== undefined) {
            void setNotesTableGraph(table, spec);
        }
    };
    state.graphPromptSubmit = null;
    state.renamingFile = null;
    state.textPromptHandler = null;
    elements.modal.dataset.open = 'true';
    elements.modal.setAttribute('aria-hidden', 'false');
    elements.modalLocationRow.style.display = 'none';
    elements.modalBody.style.display = 'flex';
    elements.modalInput.value = '';
    const graphActionLabel = existing ? 'Modify' : 'Create';
    setNotesModalTitle(graphActionLabel, true);
    elements.modalCreate.textContent = graphActionLabel;
    elements.modalBody.replaceChildren();

    if (existing) {
        const removeButton = document.createElement('button');
        removeButton.type = 'button';
        removeButton.id = 'notes-graph-remove';
        removeButton.textContent = 'Remove';
        removeButton.addEventListener('click', () => finishNotesTableGraphPrompt(null));
        elements.modalCreate.before(removeButton);
    }

    const form = document.createElement('div');
    form.className = 'notes-graph-form';
    elements.modalBody.appendChild(form);

    const typeField = notesGraphField('Chart type');
    const typeButton = document.createElement('button');
    typeButton.type = 'button';
    typeButton.className = 'notes-graph-dropdown';
    const updateTypeButton = () => {
        typeButton.textContent = `${notesGraphTypes.find((item) => item.value === prompt.type)?.label || 'Line'}  \u2304`;
    };
    updateTypeButton();
    typeButton.addEventListener('click', () => {
        const rect = typeButton.getBoundingClientRect();
        showNotesLocalMenu(notesGraphTypes.map((item) => ({
            title: item.label,
            icon: prompt.type === item.value ? CONTEXT_ICON_TICK : 0,
            onSelect: () => {
                prompt.type = item.value;
                updateTypeButton();
            },
        })), rect.left, rect.bottom, 'Chart type');
    });
    typeField.field.appendChild(typeButton);
    form.appendChild(typeField.field);

    const titleField = notesGraphField('Title');
    const titleInput = document.createElement('input');
    titleInput.type = 'text';
    titleInput.className = 'notes-graph-text-title';
    titleInput.value = prompt.title;
    titleInput.placeholder = 'Optional chart title';
    titleInput.addEventListener('input', () => { prompt.title = titleInput.value; });
    titleField.field.appendChild(titleInput);
    form.appendChild(titleField.field);

    const columnsField = notesGraphField('Columns');
    const columnsHelp = document.createElement('div');
    columnsHelp.className = 'notes-graph-help';
    columnsHelp.textContent = 'Selected non-numeric columns labels the data, joined with commas; numeric columns become series.';
    columnsField.field.appendChild(columnsHelp);
    const columnActions = document.createElement('div');
    columnActions.className = 'notes-graph-list-actions';
    const selectAllColumns = document.createElement('button');
    selectAllColumns.type = 'button';
    selectAllColumns.className = 'notes-graph-small-button';
    selectAllColumns.textContent = 'Select all';
    selectAllColumns.addEventListener('click', () => {
        prompt.selectedColumns = new Set(headings.map((_, index) => index));
        renderColumns();
    });
    columnActions.appendChild(selectAllColumns);
    columnsField.field.appendChild(columnActions);
    const columnSearch = document.createElement('input');
    columnSearch.type = 'search';
    columnSearch.className = 'notes-graph-text';
    columnSearch.placeholder = 'Find columns…';
    columnSearch.addEventListener('input', () => {
        prompt.columnQuery = columnSearch.value.trim().toLowerCase();
        renderColumns();
    });
    columnsField.field.appendChild(columnSearch);
    const columnsList = document.createElement('div');
    columnsList.className = 'notes-graph-options-list markdown-body';
    columnsField.field.appendChild(columnsList);
    form.appendChild(columnsField.field);

    const renderColumns = () => {
        columnsList.replaceChildren();
        headings.forEach((heading, index) => {
            const columnName = heading || `Column ${index + 1}`;
            if (prompt.columnQuery && !`${index + 1} ${columnName}`.toLowerCase().includes(prompt.columnQuery)) {
                return;
            }
            const option = document.createElement('label');
            option.className = 'notes-graph-option';
            const checkbox = document.createElement('input');
            checkbox.type = 'checkbox';
            checkbox.checked = prompt.selectedColumns.has(index);
            checkbox.addEventListener('change', () => {
                if (checkbox.checked) prompt.selectedColumns.add(index);
                else prompt.selectedColumns.delete(index);
            });
            const text = document.createElement('span');
            text.textContent = columnName;
            option.append(checkbox, text);
            columnsList.appendChild(option);
        });
    };
    renderColumns();

    const rowsField = notesGraphField('Rows');
    const rowsHelp = document.createElement('div');
    rowsHelp.className = 'notes-graph-help';
    rowsHelp.textContent = 'All rows are selected by default. The SQL filter, when active, is applied too.';
    rowsField.field.appendChild(rowsHelp);
    const rowActions = document.createElement('div');
    rowActions.className = 'notes-graph-list-actions';
    const selectAllRows = document.createElement('button');
    selectAllRows.type = 'button';
    selectAllRows.className = 'notes-graph-small-button';
    selectAllRows.textContent = 'Select all';
    selectAllRows.addEventListener('click', () => {
        prompt.selectedRows = null;
        prompt.rowLimit = notesGraphRowPageSize;
        renderRows();
    });
    prompt.rowCount = document.createElement('span');
    prompt.rowCount.className = 'notes-graph-count';
    rowActions.append(selectAllRows, prompt.rowCount);
    rowsField.field.appendChild(rowActions);

    const rowSearch = document.createElement('input');
    rowSearch.type = 'search';
    rowSearch.className = 'notes-graph-text';
    rowSearch.placeholder = 'Find rows…';
    rowSearch.addEventListener('input', () => {
        prompt.rowQuery = rowSearch.value.trim().toLowerCase();
        prompt.rowLimit = notesGraphRowPageSize;
        renderRows();
    });
    rowsField.field.appendChild(rowSearch);
    prompt.rowList = document.createElement('div');
    prompt.rowList.className = 'notes-graph-options-list markdown-body';
    rowsField.field.appendChild(prompt.rowList);
    form.appendChild(rowsField.field);

    function renderRows() {
        const matches = prompt.rows.filter((row, index) => {
            if (!prompt.rowQuery) return true;
            const label = `${index + 1} ${Array.from(row.querySelectorAll('td, th')).map((cell) => getTableCellTextContent(cell)).join(' ')}`;
            return label.toLowerCase().includes(prompt.rowQuery);
        });
        const selectedCount = prompt.selectedRows === null
            ? prompt.rows.length
            : prompt.selectedRows.size;
        prompt.rowCount.textContent = `${selectedCount} of ${prompt.rows.length} selected`;
        prompt.rowList.replaceChildren();
        matches.slice(0, prompt.rowLimit).forEach((row) => {
            const index = Number(row.dataset.sourceRow) || 0;
            const label = document.createElement('label');
            label.className = 'notes-graph-option';
            const checkbox = document.createElement('input');
            checkbox.type = 'checkbox';
            checkbox.checked = prompt.selectedRows === null || prompt.selectedRows.has(index);
            checkbox.addEventListener('change', () => {
                if (prompt.selectedRows === null) {
                    prompt.selectedRows = new Set(prompt.rows.map((item) => Number(item.dataset.sourceRow) || 0));
                }
                if (checkbox.checked) prompt.selectedRows.add(index);
                else prompt.selectedRows.delete(index);
                if (prompt.selectedRows.size === prompt.rows.length) prompt.selectedRows = null;
                updateRowCount();
            });
            const cells = Array.from(row.querySelectorAll('td, th')).map((cell) => getTableCellTextContent(cell));
            const text = document.createElement('span');
            text.textContent = `${index + 1}. ${cells.join(' | ')}`;
            label.append(checkbox, text);
            prompt.rowList.appendChild(label);
        });
        if (matches.length > prompt.rowLimit) {
            const more = document.createElement('button');
            more.type = 'button';
            more.className = 'notes-graph-small-button';
            more.textContent = `Show ${Math.min(notesGraphRowPageSize, matches.length - prompt.rowLimit)} more`;
            more.addEventListener('click', () => {
                prompt.rowLimit += notesGraphRowPageSize;
                renderRows();
            });
            prompt.rowList.appendChild(more);
        }
    }

    function updateRowCount() {
        const selectedCount = prompt.selectedRows === null
            ? prompt.rows.length
            : prompt.selectedRows.size;
        prompt.rowCount.textContent = `${selectedCount} of ${prompt.rows.length} selected`;
    }
    renderRows();

    state.graphPromptSubmit = () => {
        const columns = [...prompt.selectedColumns].sort((a, b) => a - b);
        if (columns.length < 2) {
            notifyTerminal('Select a category column and at least one value column.', 'warn');
            return;
        }
        const numericColumns = new Set(notesTableNumericColumns(table));
        let categoryColumns = columns.filter((column) => !numericColumns.has(column));
        if (categoryColumns.length === 0) categoryColumns = [columns[0]];
        const categoryColumn = categoryColumns[0];
        let valueColumns = columns.filter((column) => !categoryColumns.includes(column) && numericColumns.has(column));
        if (prompt.type === 'pie' || prompt.type === 'doughnut') valueColumns = valueColumns.slice(0, 1);
        if (valueColumns.length === 0) {
            notifyTerminal('Select at least one numeric value column.', 'warn');
            return;
        }
        if (prompt.selectedRows !== null && prompt.selectedRows.size === 0) {
            notifyTerminal('Select at least one row.', 'warn');
            return;
        }
        const allSourceRows = prompt.rows.length;
        const rowIndices = prompt.selectedRows === null || prompt.selectedRows.size === allSourceRows
            ? []
            : [...prompt.selectedRows].sort((a, b) => a - b);
        finishNotesTableGraphPrompt({
            type: prompt.type,
            format: 'svg',
            title: prompt.title.trim(),
            categoryColumn,
            categoryColumns,
            valueColumns,
            rowIndices,
            seriesNames: valueColumns.map((column) => headings[column]),
            categoryAxisTitle: categoryColumns.map((column) => headings[column]).join(', '),
        });
    };

    disableNativeTextAssistanceIn(form);
    state.graphPromptSpellCheck = attachSpellCheck(titleInput);
    elements.modalCreate.textContent = graphActionLabel;
    setTimeout(() => typeButton.focus(), 0);
}

function finishNotesTableGraphPrompt(spec) {
    if (!state.graphPromptHandler) return;
    const handler = state.graphPromptHandler;
    state.graphPromptHandler = null;
    state.graphPromptSubmit = null;
    state.graphPromptSpellCheck?.detach();
    state.graphPromptSpellCheck = null;
    elements.modalLocationRow.style.display = '';
    elements.modalBody.style.display = 'none';
    elements.modalBody.replaceChildren();
    document.getElementById('notes-graph-remove')?.remove();
    elements.modalInput.placeholder = 'example-note';
    setNotesModalTitle('New note name');
    elements.modal.dataset.open = 'false';
    elements.modal.setAttribute('aria-hidden', 'true');
    handler(spec);
}

async function setNotesTableGraph(table, spec) {
    const request = notesTableRequest(table, true);
    const result = await NotesTableSetGraph(request, spec);
    if (result?.error) {
        notifyTerminal(String(result.error), 'error');
        return;
    }
    table.dataset.tableGraphSpec = spec ? JSON.stringify(spec) : '';
    await renderNotesTableGraph(table);
}

async function copyNotesTableGraphPNG(table) {
    try {
        const spec = JSON.parse(table.dataset.tableGraphSpec || 'null');
        if (!spec) {
            throw new Error('No graph is available to copy.');
        }
        const result = await NotesTableRenderGraph(notesTableRequest(table, true), { ...spec, format: 'png' });
        if (result?.error || !result?.pngDataURL) {
            throw new Error(result?.error || 'Graph rendering returned no PNG.');
        }
        await TerminalCopyImageDataURL(result.pngDataURL);
    } catch (err) {
        notifyTerminal(`Failed to copy graph: ${err instanceof Error ? err.message : err}`, 'error');
    }
}

async function renderNotesTableGraph(table) {
    const extras = notesTableExtras(table, false);
    if (!extras) {
        return;
    }
    let spec;
    try {
        spec = JSON.parse(table.dataset.tableGraphSpec || 'null');
    } catch {
        spec = null;
    }
    let graph = extras.querySelector(':scope > .notes-table-graph');
    if (!spec) {
        graph?.remove();
        return;
    }
    if (!graph) {
        graph = document.createElement('div');
        graph.className = 'notes-table-graph';
        extras.appendChild(graph);
        graph.addEventListener('contextmenu', (event) => {
            const container = graph.closest('[data-copy-rendered], .markdown-body');
            if (container && hasRenderedSelection(container)) return;
            event.preventDefault();
            event.stopPropagation();
            showNotesLocalMenu([{
                title: 'Copy graph (PNG)',
                icon: CONTEXT_ICON_COPY,
                onSelect: () => void copyNotesTableGraphPNG(table),
            }, createTableGraphMenuItem(table), { title: '-', icon: 0 }, {
                title: 'Remove graph',
                icon: 0,
                onSelect: () => void setNotesTableGraph(table, null),
            }], event.clientX, event.clientY, 'Graph');
        });
    }
    const renderID = String(Number(graph.dataset.renderID || '0') + 1);
    graph.dataset.renderID = renderID;
    graph.setAttribute('aria-busy', 'true');
    const result = await NotesTableRenderGraph(notesTableRequest(table, true), { ...spec, format: 'svg' });
    if (!graph.isConnected || graph.dataset.renderID !== renderID) {
        return;
    }
    graph.setAttribute('aria-busy', 'false');
    if (result?.error) {
        graph.textContent = String(result.error);
        return;
    }
    if (!result?.svg) {
        graph.replaceChildren();
        return;
    }
    const image = document.createElement('img');
    image.className = 'notes-table-graph-image';
    image.alt = String(spec.title || 'Table graph');
    image.src = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(result.svg)}`;
    image.dataset.fullscreenBound = 'true';
    image.style.cursor = 'zoom-in';
    image.addEventListener('click', (event) => {
        event.preventDefault();
        const document = new DOMParser().parseFromString(result.svg, 'image/svg+xml');
        const svg = document.documentElement;
        if (svg.localName !== 'svg') {
            return;
        }
        const viewBox = svg.viewBox?.baseVal;
        const dimensions = String(svg.getAttribute('viewBox') || '').trim().split(/[\s,]+/).map(Number);
        const width = viewBox?.width || dimensions[2] || Number.parseFloat(svg.getAttribute('width')) || 800;
        const height = viewBox?.height || dimensions[3] || Number.parseFloat(svg.getAttribute('height')) || 600;
        showFullscreenImageOverlay({
            svgElement: svg.cloneNode(true),
            sourceWidth: Math.round(width),
            sourceHeight: Math.round(height),
        });
    });
    graph.replaceChildren(image);
}

async function notesTableInvoke(table, call) {
    // Small tables are rebuilt per query on the Go side, so send the data up
    // front rather than paying a round trip to be told it was needed.
    let result = await call(notesTableRequest(table, notesTableIsSmall(table)));
    if (result && result.missing) {
        result = await call(notesTableRequest(table, true));
    }
    return result;
}

function applyNotesTableResult(table, result) {
    if (!result) {
        return;
    }
    if (result.error) {
        notifyTerminal(String(result.error), 'error');
        return;
    }

    table.dataset.sortColumn = String(Number(result.sortColumn) || 0);
    table.dataset.sortDesc = result.sortDesc ? 'true' : 'false';
    table.dataset.tableFilter = String(result.filter || '');
    if (Object.prototype.hasOwnProperty.call(result, 'graph')) {
        table.dataset.tableGraphSpec = result.graph ? JSON.stringify(result.graph) : '';
    }

    applyNotesTableOrder(table, Array.isArray(result.order) ? result.order : []);
    renderNotesTableSortIndicator(table);
    renderNotesTableFilterFooter(table);
    void renderNotesTableGraph(table);
}

function applyNotesTableOrder(table, order) {
    const tbody = table.querySelector('tbody');
    if (!tbody) {
        return;
    }

    const bySource = new Map();
    for (const row of tbody.querySelectorAll('tr')) {
        bySource.set(Number(row.dataset.sourceRow) || 0, row);
    }

    const visible = new Set(order);
    const fragment = document.createDocumentFragment();

    for (const sourceIndex of order) {
        const row = bySource.get(sourceIndex);
        if (!row) continue;
        row.classList.remove('notes-table-row-filtered');
        fragment.appendChild(row);
    }

    // Filtered-out rows stay in the DOM, hidden, so editing and formulas keep
    // resolving against a complete document.
    for (const sourceIndex of Array.from(bySource.keys()).sort((a, b) => a - b)) {
        if (visible.has(sourceIndex)) continue;
        const row = bySource.get(sourceIndex);
        row.classList.add('notes-table-row-filtered');
        fragment.appendChild(row);
    }

    // A single insertion: appending row by row risks a layout pass per row.
    tbody.appendChild(fragment);
}

function renderNotesTableSortIndicator(table) {
    table.querySelectorAll('.notes-sort-icon').forEach((icon) => icon.remove());

    const column = Number(table.dataset.sortColumn) || 0;
    if (column < 1) {
        return;
    }

    const th = table.querySelectorAll('thead th')[column - 1];
    if (!th) {
        return;
    }

    const icon = document.createElement('span');
    icon.className = 'notes-sort-icon';
    icon.textContent = table.dataset.sortDesc === 'true' ? NOTES_TABLE_SORT_DESC : NOTES_TABLE_SORT_ASC;
    th.prepend(icon);
}

async function sortNotesTableColumn(table, column) {
    applyNotesTableResult(table, await notesTableInvoke(table, (req) => NotesTableSort(req, column)));
}

async function clearNotesTableSort(table) {
    applyNotesTableResult(table, await notesTableInvoke(table, (req) => NotesTableClearSort(req)));
}

async function promptNotesTableFilter(table) {
    const where = await openTextPrompt({
        title: 'SQL filter',
        value: table.dataset.tableFilter || '',
        confirmLabel: 'Apply',
        placeholder: '"Column" > 10 — leave empty to reset',
    });
    if (where === null) {
        return;
    }

    applyNotesTableResult(table, await notesTableInvoke(table, (req) => NotesTableFilter(req, where)));
}

function createTableFilterMenuItem(table) {
    return {
        title: 'SQL filter...',
        icon: 0xf0b0,
        onSelect: () => {
            void promptNotesTableFilter(table);
        },
    };
}

function createTableGraphMenuItem(table) {
    return {
        title: table.dataset.tableGraphSpec ? 'Modify graph...' : 'Create graph...',
        icon: 0xf201,
        onSelect: () => openNotesTableGraphPrompt(table),
    };
}

function createClearTableSortMenuItem(table) {
    return {
        title: 'Clear sorting',
        icon: 0,
        onSelect: () => {
            void clearNotesTableSort(table);
        },
    };
}

function setupNotesTableQueries(container, surface, documentPath = '') {
    if (!container) return;

    const indices = [];
    const headingOccurrences = new Map();

    Array.from(container.querySelectorAll('table')).forEach((table, tableIndex) => {
        const tbody = table.querySelector('tbody');
        const headerRow = table.querySelector('thead tr');
        if (!tbody || !headerRow) return;

        const aiIdentity = surface === 'ai' ? getAITableIdentity(table) : null;
        if (surface === 'ai' && !aiIdentity) return;
        indices.push(tableIndex);

        table.dataset.tableSurface = surface;
        table.dataset.tableDocument = aiIdentity?.document ?? documentPath;
        table.dataset.tableIndex = String(aiIdentity ? 0 : tableIndex);
        const headingIdentity = JSON.stringify(getTableHeadingValues(table).map((heading) => heading.trim()));
        const stateIndex = headingOccurrences.get(headingIdentity) || 0;
        headingOccurrences.set(headingIdentity, stateIndex + 1);
        table.dataset.tableStateIndex = String(aiIdentity ? 0 : stateIndex);
        table.dataset.sortColumn = table.dataset.sortColumn || '0';
        table.dataset.sortDesc = table.dataset.sortDesc || 'false';
        table.dataset.tableFilter = table.dataset.tableFilter || '';

        Array.from(tbody.querySelectorAll('tr')).forEach((row, i) => {
            row.dataset.sourceRow = String(i);
        });

        void restoreNotesTableState(table);

        if (table.dataset.tableQueriesBound === 'true') return;
        table.dataset.tableQueriesBound = 'true';

        Array.from(headerRow.querySelectorAll('th')).forEach((th, colIndex) => {
            th.addEventListener('click', (e) => {
                if (table.dataset.resizeDragActive === 'true') {
                    e.preventDefault();
                    e.stopPropagation();
                    return;
                }

                e.preventDefault();
                e.stopPropagation();
                void sortNotesTableColumn(table, colIndex + 1);
            });

            // Middle-click clears the sort, as in the terminal. Browsers deliver
            // it as auxclick, and mousedown has to be suppressed or WebKit starts
            // autoscroll instead.
            th.addEventListener('mousedown', (e) => {
                if (e.button === 1) {
                    e.preventDefault();
                }
            });

            th.addEventListener('auxclick', (e) => {
                if (e.button !== 1 || table.dataset.resizeDragActive === 'true') {
                    return;
                }

                e.preventDefault();
                e.stopPropagation();
                void clearNotesTableSort(table);
            });
        });

    });

    // Declaring what still exists lets the backend drop everything else for this
    // surface, so a missed cleanup self-heals rather than accumulating.
    try {
        if (typeof NotesTableReconcile === 'function') {
            const documents = surface === 'ai'
                ? Array.from(container.querySelectorAll('table')).map((table) => table.dataset.tableDocument).filter(Boolean)
                : [documentPath];
            for (const document of new Set(documents)) {
                void Promise.resolve(NotesTableReconcile(surface, document, surface === 'ai' ? [0] : indices)).catch(() => {});
            }
        }
    } catch (err) {
        console.error('Failed to reconcile Notes tables:', err);
    }
}

async function restoreNotesTableState(table) {
    try {
        const result = await NotesTableRestore(notesTableRequest(table, true));
        applyNotesTableResult(table, result);
    } catch (err) {
        console.error('Failed to restore Notes table state:', err);
    }
}

function toggleCheckboxInMarkdown(checkboxIndex, isChecked) {
    const lines = mainDocument.getValue().split('\n');
    let currentCheckboxIndex = 0;
    let modified = false;

    for (let i = 0; i < lines.length; i++) {
        const checkboxMatch = lines[i].match(/^(\s*[-*+]?\s*)\[( |x|X)\](.*)$/);
        if (!checkboxMatch) {
            continue;
        }

        if (currentCheckboxIndex === checkboxIndex) {
            const newState = isChecked ? 'x' : ' ';
            lines[i] = `${checkboxMatch[1]}[${newState}]${checkboxMatch[3]}`;
            modified = true;
            break;
        }
        currentCheckboxIndex++;
    }

    if (modified) {
        replaceMainDocumentText(lines.join('\n'), { source: 'markdown-checkbox' });
        saveFile();
        // Keep viewer in sync when changes are made from jupyter mode
        if (state.viewMode === 'jupyter') {
            renderMarkdown();
        }
        // Don't re-render jupyter here to avoid resetting checkbox focus
    }
}

function updateMarkdownTableOfContents() {
    if (state.currentFileType !== 'markdown') {
        return;
    }

    const source = mainDocument.getValue();
    const result = updateMarkdownTableOfContentsText(source);
    if (!result.updated) {
        notifyTerminal('No headings found to generate a table of contents.', 'info');
        return;
    }

    replaceMainDocumentText(result.text, { source: 'markdown-toc', selectionStart: 0, selectionEnd: 0 });
    if (usesCodeEditorDecorations()) {
        refreshEditorLanguage(state.currentFile, mainDocument.getValue());
    }
    hideLspHoverTooltip();
    hideLspCompletion();
    if (isMonacoActive()) {
        focusedView().editor.revealOffset(0);
    }
    setDirty(true);
    scheduleRender();
    scheduleAutoSave();
    saveFile();
    notifyTerminal('Table of contents updated.', 'info');
}

function updateMarkdownCodeBlock(blockIndex, newContent) {
    const markdown = mainDocument.getValue();
    const rxCodeBlock = /```[^\n]*\n[\s\S]*?\n```/g;
    let match;
    let index = 0;
    let lastIndex = 0;
    let updated = false;
    let result = '';

    while ((match = rxCodeBlock.exec(markdown)) !== null) {
        if (index === blockIndex) {
            const block = match[0];
            const headerEnd = block.indexOf('\n');
            const footerStart = block.lastIndexOf('\n```');
            if (headerEnd === -1 || footerStart === -1) {
                return false;
            }

            const header = block.slice(0, headerEnd + 1);
            const footer = block.slice(footerStart);
            const trimmedContent = newContent.replace(/[\r\n]+$/, '');
            const updatedBlock = header + trimmedContent + footer;

            result += markdown.slice(lastIndex, match.index) + updatedBlock;
            lastIndex = match.index + match[0].length;
            updated = true;
            break;
        }
        index++;
    }

    if (!updated) {
        return false;
    }

    result += markdown.slice(lastIndex);
    replaceMainDocumentText(result, { source: 'markdown-code-block' });
    return true;
}

function scheduleRender() {
    if (state.renderTimer) {
        clearTimeout(state.renderTimer);
    }
    state.renderTimer = setTimeout(() => {
        state.renderTimer = null;
        if (state.currentFileType === 'html') {
            renderHtmlView();
            return;
        }
        renderMarkdown();
    }, 120);
}

function scheduleAutoSave() {
    const doc = activeNotesDocument;
    if (doc.autosaveTimer) {
        clearTimeout(doc.autosaveTimer);
    }
    if (!savesNotesDocumentAutomatically(doc)) {
        return;
    }
    doc.autosaveTimer = setTimeout(() => {
        doc.autosaveTimer = null;
        if (doc === activeNotesDocument && savesNotesDocumentAutomatically(doc)) {
            saveFile();
        }
    }, 1000);
}

function buildLineIndentationEdit(source, start, end, indentation, outdent = false) {
    const text = String(source || '');
    const indent = String(indentation || '');
    const indentWidth = Math.max(1, indent.length);
    const safeStart = Math.max(0, Math.min(Number(start) || 0, text.length));
    const safeEnd = Math.max(safeStart, Math.min(Number(end) || safeStart, text.length));

    if (safeEnd <= safeStart) {
        return null;
    }

    const lineStart = text.lastIndexOf('\n', Math.max(0, safeStart - 1)) + 1;
    const inclusiveEnd = (safeEnd > safeStart && text[safeEnd - 1] === '\n')
        ? Math.max(safeStart, safeEnd - 1)
        : safeEnd;
    const lineEndIdx = text.indexOf('\n', inclusiveEnd);
    const lineEnd = lineEndIdx === -1 ? text.length : lineEndIdx;

    const selectedBlock = text.slice(lineStart, lineEnd);
    const lines = selectedBlock.split('\n');
    const transformed = lines.map((line) => {
        if (!outdent) {
            return indent + line;
        }

        if (line.startsWith('\t')) {
            return line.slice(1);
        }

        let remove = 0;
        while (remove < indentWidth && line[remove] === ' ') {
            remove += 1;
        }
        return line.slice(remove);
    });

    const replacement = transformed.join('\n');
    return {
        start: lineStart,
        end: lineEnd,
        text: replacement,
        selectionStart: lineStart,
        selectionEnd: lineStart + replacement.length,
    };
}

function shouldMonacoTabIndent() {
    const selection = focusedView().editor?.getSelectionOffsets?.();
    const cursor = Number(selection?.start) || 0;
    const source = getMainEditorValue();
    const lineStart = source.lastIndexOf('\n', Math.max(0, cursor - 1)) + 1;
    const leftOfCaret = source.slice(lineStart, cursor);
    return /^\s*$/.test(leftOfCaret);
}

async function applyMonacoTabIndent() {
    if (!isMonacoActive()) {
        return;
    }

    const selection = focusedView().editor.getSelectionOffsets();
    const start = Number(selection?.start) || 0;
    const end = Number(selection?.end) || start;
    const indentation = await getIndentationString();

    hideLspCompletion();
    focusedView().editor.replaceRange(start, end, indentation, 'notes-monaco-tab-indent');
    const next = start + indentation.length;
    focusedView().editor.setSelectionOffsets(next, next);
    focusedView().editor.focus();
}

async function applyMonacoLineIndentationForSelection(outdent = false) {
    if (!isMonacoActive()) {
        return false;
    }

    const selection = focusedView().editor.getSelectionOffsets();
    const start = Number(selection?.start) || 0;
    const end = Number(selection?.end) || start;
    if (end <= start) {
        return false;
    }

    const indentation = await getIndentationString();
    const edit = buildLineIndentationEdit(getMainEditorValue(), start, end, indentation, outdent);
    if (!edit) {
        return false;
    }

    hideLspCompletion();
    focusedView().editor.replaceRange(edit.start, edit.end, edit.text, outdent ? 'notes-monaco-shift-tab-outdent' : 'notes-monaco-tab-indent-lines');
    focusedView().editor.setSelectionOffsets(edit.selectionStart, edit.selectionEnd);
    focusedView().editor.focus();
    return true;
}

function fileUriToPath(uri) {
    const source = String(uri || '').trim();
    if (!source) {
        return '';
    }

    try {
        const parsed = new URL(source);
        if (parsed.protocol !== 'file:') {
            return '';
        }

        let path = decodeURIComponent(parsed.pathname || '');
        if (/^\/[A-Za-z]:\//.test(path)) {
            path = path.slice(1);
        }
        return path.replace(/\\/g, '/');
    } catch {
        return '';
    }
}

function normalizePathForMatch(path) {
    const source = String(path || '').trim().replace(/\\/g, '/');
    if (!source) {
        return '';
    }
    return source.endsWith('/') ? source.slice(0, -1) : source;
}

async function resolveNotesFileFromAbsolutePath(absPath) {
    const normalizedTarget = normalizePathForMatch(absPath);
    if (!normalizedTarget) {
        return '';
    }

    if (state.currentFile) {
        try {
            const currentResolved = normalizePathForMatch(await ResolveFilePath(state.currentFile));
            if (currentResolved === normalizedTarget) {
                return state.currentFile;
            }
        } catch {
            // Ignore and continue with file list lookup.
        }
    }

    for (const file of state.files) {
        try {
            const resolved = normalizePathForMatch(await ResolveFilePath(file));
            if (resolved === normalizedTarget) {
                return file;
            }
        } catch {
            // Keep scanning; one bad entry should not stop lookup.
        }
    }

    return '';
}

function currentEditorToken() {
    const content = getMainEditorValue();
    const selection = getMainEditorSelectionRange();
    const cursor = Math.max(0, Number(selection?.start) || 0);
    const left = content.slice(0, cursor);
    const right = content.slice(cursor);
    const leftMatch = left.match(/[A-Za-z_][A-Za-z0-9_]*/g);
    const rightMatch = right.match(/^[A-Za-z0-9_]*/);
    const leftToken = leftMatch ? leftMatch[leftMatch.length - 1] : '';
    const rightToken = rightMatch ? rightMatch[0] : '';
    const symbol = `${leftToken}${rightToken}`;
    const start = cursor - leftToken.length;
    return symbol ? { symbol, start } : null;
}

function setProjectReferencesResults(symbol, source, results) {
    state.findFilesMode = 'references';
    state.findFilesSource = source;
    state.findFilesReferenceSymbol = symbol;
    state.findFilesQuery = symbol;
    state.findFilesResults = results;
    state.findFilesBusy = false;
    state.findFilesError = '';
    state.findFilesSelectedKey = '';
    resetProjectFindPaging();
    elements.findFilesInput.value = symbol;
    updateFindFilesClearButtonVisibility();
    setToolsPanelCollapsed(false);
    setToolsTab('find');
    renderProjectFindResults();
    renderFileList();
}

// ── typos-lsp spellchecking (main editor) ──────────────────────────────────
//
// When typos-lsp is available, eligible project files are spellchecked by it
// instead of aspell, rendering with the same red wavy-underline chrome. aspell
// remains the fallback (and is always used for form fields like the input box).

// Replace a textarea's value with formatted text, preserving the caret/selection
// proportionally. Returns false when nothing changed.
function applyFormattedText(textarea, formatted, options = {}) {
    if (!textarea || typeof formatted !== 'string' || formatted.length === 0) {
        return false;
    }
    if (formatted === textarea.value) {
        return false;
    }
    const prevStart = textarea.selectionStart || 0;
    const prevEnd = textarea.selectionEnd || 0;
    const prevLen = textarea.value.length;
    const newLen = formatted.length;
    const ratio = prevLen > 0 ? newLen / prevLen : 1;
    const newStart = Math.min(Math.round(prevStart * ratio), newLen);
    const newEnd = Math.min(Math.round(prevEnd * ratio), newLen);

    notesMutationAdapter.replaceDocumentText(textarea, {
        text: formatted,
        selectionStart: newStart,
        selectionEnd: newEnd,
        source: options.source || 'format',
        label: options.label || 'Apply formatted text',
        filePath: options.filePath,
        emit: false,
    });
    return true;
}

// Keeps the caret at the same proportional position in the formatted text.
function applyFormattedMainDocument(formatted, source) {
    const current = mainDocument.getValue();
    if (typeof formatted !== 'string' || formatted.length === 0 || formatted === current) {
        return;
    }
    const { start, end } = mainDocument.getSelection();
    const ratio = current.length > 0 ? formatted.length / current.length : 1;
    replaceMainDocumentText(formatted, {
        source,
        selectionStart: Math.min(Math.round(start * ratio), formatted.length),
        selectionEnd: Math.min(Math.round(end * ratio), formatted.length),
    });
}

// Unified formatter for the main editor: try the configured format command
// (e.g. goimports) first, then fall back to LSP formatting. Invoked explicitly
// by cmd/ctrl+s "save and format" and editor context menu actions.
async function formatMainEditor({ notifyOnError = false } = {}) {
    if (!state.currentFile) {
        return;
    }
    if (state.viewMode !== 'editor' && state.viewMode !== 'swagger-edit') {
        return;
    }
    if (!isCurrentFileLspEligible()) {
        return;
    }

    const content = mainDocument.getValue();

    // 1. Configured format command (e.g. goimports / ruff).
    try {
        const result = await FormatNotesContent(state.currentFile, content, state.editorLanguage || '');
        if (result && result.HasFormatter) {
            const err = typeof result.Err === 'string' ? result.Err : '';
            if (err) {
                if (notifyOnError) {
                    notifyTerminal(err, 'error');
                }
                return;
            }
            const formatted = typeof result.Code === 'string' ? result.Code : '';
            applyFormattedMainDocument(formatted, 'format-command');
            return;
        }
    } catch (err) {
        console.error('notes format command failed:', err);
        // fall through to LSP formatting
    }

    // 2. Fall back to LSP formatting when no format command is configured.
    if (state.lspOpenFile === state.currentFile) {
        await formatCurrentLspDocument({ preferSelection: false, notifyOnError });
    }
}

// Unified formatter for a Jupyter code block: try the configured format command
// (e.g. goimports) first, then fall back to LSP formatting. Invoked explicitly
// by cmd/ctrl+s "save and format" and LSP-mode toggle flows.
async function formatJupyterBlock(blockId, { notifyOnError = false } = {}) {
    const block = state.jupyterCodeBlocks[blockId];
    if (!block) {
        return;
    }
    const editableCode = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-code-editable`);
    if (!editableCode) {
        return;
    }

    const content = editableCode.value;

    // 1. Configured format command (snippet-aware, e.g. goimports / ruff).
    try {
        const result = await FormatCodeBlock(state.currentFile, blockId, content, block.runtime);
        if (result && result.HasFormatter) {
            const goErr = typeof result.Err === 'string' ? result.Err : '';
            if (goErr) {
                if (notifyOnError) {
                    notifyTerminal(goErr, 'error');
                }
                return;
            }
            const formattedCode = typeof result.Code === 'string' ? result.Code : '';
            if (applyFormattedText(editableCode, formattedCode, {
                source: 'jupyter-format-command',
                label: 'Format Jupyter block (command)',
                filePath: state.currentFile ? `${state.currentFile}#${blockId}` : blockId,
            })) {
                block.currentContent = editableCode.value;
                editableCode.dispatchEvent(new Event('input', { bubbles: true }));
            }
            return;
        }
    } catch (err) {
        console.error('jupyter block format command failed:', err);
        // fall through to LSP formatting
    }

    // 2. Fall back to LSP formatting when no format command is configured.
    if (block.lspMode && block.lspFilePath) {
        try {
            const result = await NotesLspFormat(block.lspFilePath);
            if (result && result.changed) {
                const nextContent = String(result.content || '');
                if (applyFormattedText(editableCode, nextContent, {
                    source: 'jupyter-format-lsp',
                    label: 'Format Jupyter block (LSP)',
                    filePath: state.currentFile ? `${state.currentFile}#${blockId}` : blockId,
                })) {
                    block.currentContent = editableCode.value;
                    editableCode.dispatchEvent(new Event('input', { bubbles: true }));
                }
            }
        } catch (err) {
            if (notifyOnError) {
                notifyTerminal('Failed to format block', 'error');
            }
            console.error('jupyter block LSP format failed:', err);
        }
    }
}

// Debounced sync of a Jupyter code block's content to the typos-lsp server so
// its spellcheck diagnostics stay current.
function scheduleTyposBlockChange(blockId, editableCode) {
    const block = state.jupyterCodeBlocks[blockId];
    if (!block || !block.typosFilePath) {
        return;
    }
    if (block.typosChangeTimer) {
        clearTimeout(block.typosChangeTimer);
    }
    block.typosChangeTimer = setTimeout(async () => {
        block.typosChangeTimer = null;
        try {
            await NotesTyposChangeDocument(block.typosFilePath, editableCode.value || '');
        } catch (err) {
            console.error('block typos change failed:', err);
        }
    }, LSP_CHANGE_DEBOUNCE_MS);
}

function setDirty(isDirty) {
    state.dirty = isDirty;
    markActiveNotesTabDirty();
    const label = state.currentFile ? state.currentFile : 'No file selected';
    elements.status.textContent = isDirty ? `${label} (unsaved)` : label;
}

function focusActiveEditorForViewMode() {
    // Keep terminal ownership on app startup; ttyphoon.js will hand off when Notes is explicitly focused.
    if (window.terminalFocusedState === true) {
        return;
    }

    const shouldFocusEditor =
        state.viewMode === 'editor' ||
        state.viewMode === 'swagger-edit' ||
        state.viewMode === 'csv-edit';

    if (!shouldFocusEditor) {
        return;
    }

    setTimeout(() => {
        if (window.terminalFocusedState === true) {
            return;
        }

        const stillShouldFocus =
            state.viewMode === 'editor' ||
            state.viewMode === 'swagger-edit' ||
            state.viewMode === 'csv-edit';

        if (!stillShouldFocus) {
            return;
        }

        if (isGitDiffActive()) {
            notesGitDiffAdapter.focus();
            return;
        }

        if (isMonacoActive()) {
            focusedView().editor.focus();
        }
    }, 0);
}

function emitCurrentFileName() {
    const fileName = state.currentFile ? getPathFileName(state.currentFile) : '';
    app.dataset.currentFileName = fileName;
    elements.currentFileTitle.textContent = fileName;
    elements.currentFileTitle.hidden = !fileName;
    window.dispatchEvent(new CustomEvent('notes-current-file', {
        detail: { fileName }
    }));
    updateGitDiffTabForCurrentFile();
}

function currentProjectRelativeGitPath() {
    const currentFile = String(state.currentFile || '').replaceAll('\\', '/');
    if (currentFile.startsWith('$PROJECT/')) {
        return currentFile.slice('$PROJECT/'.length);
    }
    const root = String(state.currentFileProject || state.currentProjectRoot || '')
        .replaceAll('\\', '/').replace(/\/+$/, '');
    return root && currentFile.startsWith(`${root}/`)
        ? currentFile.slice(root.length + 1)
        : '';
}

function updateGitDiffTabForCurrentFile(status, projectRoot) {
    if (!elements.tabDiff) {
        return;
    }
    if (activeGitDiff) {
        elements.tabDiff.style.display = '';
        return;
    }

    const root = state.currentFileProject || state.currentProjectRoot || projectRoot || notesGitPanel?.getCurrentRoot() || '';
    const normalizeRoot = (value) => String(value || '').replaceAll('\\', '/').replace(/\/+$/, '');
    if (projectRoot && root && normalizeRoot(projectRoot) !== normalizeRoot(root)) {
        return;
    }
    const relativePath = currentProjectRelativeGitPath();
    const snapshot = status || notesGitPanel?.getCachedStatus(root);
    const hasDiff = Boolean(relativePath && snapshot?.files?.some((file) =>
        String(file.path || '').replaceAll('\\', '/') === relativePath));
    elements.tabDiff.style.display = hasDiff ? '' : 'none';
    if (!hasDiff && elements.tabDiff.getAttribute('aria-selected') === 'true') {
        setNotesGitDiffTabActive(false);
        elements.gitDiffWrap.dataset.active = 'false';
        setViewMode(state.viewMode);
    }
}

function setViewMode(mode) {
    if (activeGitDiff && focusedView().root.dataset.gitDiff === 'true') {
        void closeNotesGitDiff();
    }
    // Determine the mode based on current file type.
    if (mode === 'meta') {
        state.viewMode = 'meta';
    } else if (mode === 'hex') {
        state.viewMode = 'hex';
    } else if (state.currentFileType === 'json') {
        if (mode === 'swagger-view' || mode === 'swagger-edit' || (mode === 'swagger-run' && state.swaggerRunAvailable)) {
            state.viewMode = mode;
        } else {
            state.viewMode = 'swagger-view';
        }
    } else if (state.currentFileType === 'html') {
        state.viewMode = mode === 'viewer' ? 'html-view' : 'editor';
    } else if (state.currentFileType === 'code') {
        state.viewMode = isSvgFile(state.currentFile) && mode === 'image-view' ? 'image-view' : 'editor';
    } else if (state.currentFileType === 'binary') {
        state.viewMode = 'hex';
    } else if (state.currentFileType === 'image') {
        state.viewMode = 'image-view';
    } else if (state.currentFileType === 'csv') {
        if (mode === 'csv-view' || mode === 'csv-edit' || mode === 'csv-run') {
            state.viewMode = mode;
        } else {
            state.viewMode = 'csv-view';
        }
    } else {
        state.viewMode = mode === 'viewer' ? 'viewer' : (mode === 'jupyter' ? 'jupyter' : 'editor');
    }
    
    // Share active notes mode with ttyphoon.js so cross-pane focus behavior can follow mode intent.
    app.dataset.viewMode = state.viewMode;
    
    // Standard tabs
    const isEditor = state.viewMode === 'editor';
    const isHex = state.viewMode === 'hex';
    const isJupyter = state.viewMode === 'jupyter';
    const isViewer = state.viewMode === 'viewer';
    const isHtmlView = state.viewMode === 'html-view';
    const isMeta = state.viewMode === 'meta';
    
    elements.tabEditor.setAttribute('aria-selected', isEditor ? 'true' : 'false');
    elements.tabHex.setAttribute('aria-selected', isHex ? 'true' : 'false');
    elements.tabViewer.setAttribute('aria-selected', (isViewer || isHtmlView) ? 'true' : 'false');
    elements.tabJupyter.setAttribute('aria-selected', isJupyter ? 'true' : 'false');
    elements.tabMeta.setAttribute('aria-selected', isMeta ? 'true' : 'false');
    elements.tabDiff.setAttribute('aria-selected', 'false');
    
    const isStructuredEdit = state.currentFileType === 'json' && state.viewMode === 'swagger-edit';
    elements.editorWrap.dataset.active = (isEditor || isStructuredEdit) ? 'true' : 'false';
    elements.hexWrap.dataset.active = isHex ? 'true' : 'false';
    elements.previewWrap.dataset.active = isViewer ? 'true' : 'false';
    elements.htmlViewWrap.dataset.active = isHtmlView ? 'true' : 'false';
    elements.jupyterWrap.dataset.active = isJupyter ? 'true' : 'false';
    elements.metaWrap.dataset.active = isMeta ? 'true' : 'false';
    elements.gitDiffWrap.dataset.active = 'false';
    
    // Swagger tabs
    const isSwaggerView = state.viewMode === 'swagger-view';
    const isSwaggerEdit = state.viewMode === 'swagger-edit';
    const isSwaggerRun = state.viewMode === 'swagger-run';
    
    elements.tabSwaggerView.setAttribute('aria-selected', isSwaggerView ? 'true' : 'false');
    elements.tabSwaggerEdit.setAttribute('aria-selected', isSwaggerEdit ? 'true' : 'false');
    elements.tabSwaggerRun.setAttribute('aria-selected', isSwaggerRun ? 'true' : 'false');
    
    elements.swaggerViewWrap.dataset.active = isSwaggerView ? 'true' : 'false';
    elements.swaggerRunWrap.dataset.active = isSwaggerRun ? 'true' : 'false';

    // Image view tab
    const isImageView = state.viewMode === 'image-view';
    elements.tabImageView.setAttribute('aria-selected', isImageView ? 'true' : 'false');
    elements.imageViewWrap.dataset.active = isImageView ? 'true' : 'false';

    // CSV tabs
    const isCsvView = state.viewMode === 'csv-view';
    const isCsvEdit = state.viewMode === 'csv-edit';
    const isCsvRun = state.viewMode === 'csv-run';
    elements.tabCsvView.setAttribute('aria-selected', isCsvView ? 'true' : 'false');
    elements.tabCsvEdit.setAttribute('aria-selected', isCsvEdit ? 'true' : 'false');
    elements.tabCsvRun.setAttribute('aria-selected', isCsvRun ? 'true' : 'false');
    elements.csvViewWrap.dataset.active = (isCsvView || isCsvRun) ? 'true' : 'false';
    // csv-edit reuses the main editor wrap
    if (state.currentFileType === 'csv') {
        elements.editorWrap.dataset.active = isCsvEdit ? 'true' : 'false';
        if (isCsvView || isCsvRun) {
            renderCsvView(mainDocument.getValue(), { interactive: isCsvRun });
        }
    }

    // The main editor wrap is shared by code/markdown/json edit and csv edit.
    // Whenever it becomes visible, (lazily) create Monaco and lay it out against
    // its now-visible box.
    const editorWrapActive = elements.editorWrap.dataset.active === 'true';
    if (editorWrapActive) {
        void ensureMonacoVisibleAndLaidOut();
    } else if ((isEditor && usesCodeEditorDecorations()) || isStructuredEdit) {
        renderEditorDecorations();
    }

    if (isMeta) {
        renderMetaView();
    }

    if (isHtmlView) {
        renderHtmlView();
    }

    updateFindAvailability();
    if (state.currentFileType === 'markdown') {
        refreshToolsToC();
    }
    syncToolsToCHighlightForMode();

    if (isHex) {
        void ensureHexDumpForCurrentFile();
    }
    
    // Re-perform find if Find tab is currently active.
    if (elements.toolsTabFind?.getAttribute('aria-selected') === 'true' && state.findQuery) {
        performFind();
    }

    saveDocumentCache();
    focusActiveEditorForViewMode();
}

function setEditorWrapMode(enabled) {
    const wrapEnabled = enabled === true;
    state.markdownWrapMode = wrapEnabled;
    elements.editorShell.dataset.wrapMode = wrapEnabled ? 'true' : 'false';
    if (isMonacoActive()) {
        focusedView().editor.setWordWrap(wrapEnabled);
    }
    if (isGitDiffActive()) {
        notesGitDiffAdapter.setWordWrap(wrapEnabled);
    }
}

function toggleMarkdownWrapMode() {
    const isCodeLike = state.currentFileType === 'code' || state.currentFileType === 'markdown' || state.currentFileType === 'json' || state.currentFileType === 'html';
    const isStructuredEdit = state.currentFileType === 'json' && state.viewMode === 'swagger-edit';
    if (!isCodeLike || (state.viewMode !== 'editor' && !isStructuredEdit)) {
        return;
    }

    setEditorWrapMode(!state.markdownWrapMode);
    saveDocumentCache();
    
    // Re-render decorations to adjust layout
    renderEditorDecorations();
}

function collectMarkdownRunSourceBlocks(tokens, source, sourceOffset, blocks) {
    let searchFrom = 0;
    for (const token of tokens || []) {
        const raw = String(token.raw || '');
        const tokenStart = raw ? source.indexOf(raw, searchFrom) : -1;
        if (tokenStart < 0) {
            if (token.type === 'heading' || token.type === 'paragraph' || token.type === 'list') {
                blocks.push(null);
            }
            continue;
        }

        const absoluteStart = sourceOffset + tokenStart;
        if (token.type === 'heading' || token.type === 'paragraph' || token.type === 'list') {
            const trailingLineBreaks = raw.match(/(?:\r\n|\n|\r)+$/)?.[0].length || 0;
            const end = absoluteStart + raw.length - trailingLineBreaks;
            blocks.push(end > absoluteStart ? {
                type: token.type,
                start: absoluteStart,
                end,
            } : null);
        }

        if (token.type === 'html') {
            const template = document.createElement('template');
            template.innerHTML = raw;
            const selector = 'h1, h2, h3, h4, h5, h6, p, ul, ol';
            for (const element of template.content.querySelectorAll(selector)) {
                if (!element.parentElement?.closest(selector)) {
                    blocks.push(null);
                }
            }
        } else if (Array.isArray(token.tokens)) {
            collectMarkdownRunSourceBlocks(token.tokens, raw, absoluteStart, blocks);
        }

        searchFrom = tokenStart + raw.length;
    }
}

function markMarkdownRunSourceBlocks(container, markdown, sourceOffset, source) {
    container.__runMarkdownSource = source;
    const blocks = [];
    collectMarkdownRunSourceBlocks(marked.lexer(markdown), markdown, 0, blocks);

    const selector = 'h1, h2, h3, h4, h5, h6, p, ul, ol';
    const renderedBlocks = Array.from(container.querySelectorAll(selector))
        .filter((element) => !element.parentElement.closest('ul, ol'));
    for (let index = 0; index < Math.min(blocks.length, renderedBlocks.length); index += 1) {
        const block = blocks[index];
        const element = renderedBlocks[index];
        if (!block) {
            continue;
        }
        const matchesType = block.type === 'heading'
            ? /^H[1-6]$/.test(element.tagName)
            : block.type === 'paragraph'
                ? element.tagName === 'P'
                : element.tagName === 'UL' || element.tagName === 'OL';
        if (!matchesType) {
            continue;
        }

        const start = sourceOffset + block.start;
        const end = sourceOffset + block.end;
        element.dataset.runMarkdownSourceStart = String(start);
        element.dataset.runMarkdownSourceEnd = String(end);
        element.dataset.runMarkdownSourceText = source.slice(start, end);
    }
}

// The backend reference-counts LSP and typos opens, so block documents must be
// released before their blocks are discarded; restored blocks reopen them.
async function releaseJupyterBlockLspDocuments(blocks) {
    const closes = [];
    for (const block of Object.values(blocks || {})) {
        if (block?.typosChangeTimer) {
            clearTimeout(block.typosChangeTimer);
            block.typosChangeTimer = null;
        }
        if (block?.lspFilePath) {
            closes.push(NotesLspCloseDocument(block.lspFilePath));
        }
        if (block?.typosFilePath) {
            closes.push(NotesTyposCloseDocument(block.typosFilePath));
        }
    }
    await settleLspDocumentCloses(closes);
}

async function settleLspDocumentCloses(closes) {
    const results = await Promise.allSettled(closes);
    for (const result of results) {
        if (result.status === 'rejected') {
            console.error('Error closing LSP document:', result.reason);
        }
    }
}

let jupyterRenderSeq = 0;

async function renderJupyterView() {
    const renderSeq = ++jupyterRenderSeq;
    const doc = activeNotesDocument;
    const isCurrent = () => renderSeq === jupyterRenderSeq && doc === activeNotesDocument;

    // Reset jupyter state for the new render
    const previousBlocks = state.jupyterCodeBlocks;
    state.jupyterCodeBlocks = {};
    void releaseJupyterBlockLspDocuments(previousBlocks);
    state.jupyterBlockCounter = 0;
    
    const source = getMainEditorValue();
    const markdown = applyDocumentFrontmatter();
    elements.jupyter.innerHTML = marked.parse(markdown);
    if (state.currentFileType === 'markdown') {
        markMarkdownRunSourceBlocks(elements.jupyter, markdown, source.length - markdown.length, source);
    }
    
    // Apply common markdown processing
    await processMarkdownContainer(elements.jupyter);
    if (!isCurrent()) {
        return;
    }

    // Enable context menus on images
    enableImageContextMenus(elements.jupyter);
    
    // Enable checkbox editing and save behavior in jupyter mode
    setupInteractiveCheckboxes(elements.jupyter, true);

    // Enable collapsible H1-H6 sections
    setupCollapsibleHeadings(elements.jupyter);

    // Surface a caption when the document carries frontmatter.
    insertFrontmatterCaption(elements.jupyter);

    // Render code blocks immediately so they are not blocked by table evaluation.
    convertToJupyterCodeBlocks();

    evaluateTableFormulasInPlace(elements.jupyter)
        .catch((err) => {
            console.warn('Table formula evaluation failed:', err);
        })
        .finally(() => {
            if (!isCurrent()) {
                return;
            }
            setupInteractiveMarkdownTables(elements.jupyter, true);

            // Keep table horizontal scrolling local to each table rather than the whole section.
            wrapTablesForHorizontalScroll(elements.jupyter);

            // Enable column sorting on all tables
            setupNotesTableQueries(elements.jupyter, 'jupyter', doc.path);

            elements.jupyter.classList.toggle('notes-table-wordwrap-on', state.markdownTableWordWrapMode);
            void setupTableColumnResizing(elements.jupyter, state.markdownTableWordWrapMode, doc.path);

            // Re-apply find highlights when Find tab is active in jupyter mode.
            if (elements.toolsTabFind?.getAttribute('aria-selected') === 'true' && state.findQuery && state.viewMode === 'jupyter') {
                setTimeout(() => {
                    performFind();
                }, 0);
            }
        });
}

async function beginMarkdownRunBlockEdit(container, block) {
    const start = Number.parseInt(block.dataset.runMarkdownSourceStart || '', 10);
    const end = Number.parseInt(block.dataset.runMarkdownSourceEnd || '', 10);
    const source = container.__runMarkdownSource;
    const sourceText = block.dataset.runMarkdownSourceText;
    if (!Number.isInteger(start) || !Number.isInteger(end) || end < start || typeof source !== 'string' || source.slice(start, end) !== sourceText || getMainEditorValue() !== source) {
        const file = state.currentFile;
        await renderJupyterView();
        if (state.currentFile !== file || state.viewMode !== 'jupyter' || getMainEditorValue() !== container.__runMarkdownSource) {
            return;
        }
        const candidates = Array.from(container.querySelectorAll('[data-run-markdown-source-start]'))
            .filter((element) => element.tagName === block.tagName && element.dataset.runMarkdownSourceText === sourceText);
        if (candidates.length === 1) {
            await beginMarkdownRunBlockEdit(container, candidates[0]);
        }
        return;
    }

    const wasDirty = state.dirty;
    const wasViewTainted = state.viewTainted;
    const editor = document.createElement('div');
    editor.className = 'notes-run-markdown-inline-edit';

    const textarea = document.createElement('textarea');
    textarea.className = 'notes-run-markdown-inline-edit-textarea';
    textarea.setAttribute('aria-label', 'Edit Markdown source');
    textarea.value = sourceText;
    disableNativeTextAssistance(textarea);
    const autoResize = () => {
        textarea.rows = textarea.value.split('\n').length;
        textarea.style.height = 'auto';
        const styles = getComputedStyle(textarea);
        const borderHeight = (Number.parseFloat(styles.borderTopWidth) || 0)
            + (Number.parseFloat(styles.borderBottomWidth) || 0);
        if (textarea.scrollHeight > 0) {
            textarea.style.height = `${textarea.scrollHeight + borderHeight}px`;
        }
    };
    textarea.addEventListener('input', autoResize);

    const actions = document.createElement('div');
    actions.className = 'notes-run-markdown-inline-edit-actions';
    const saveButton = document.createElement('button');
    saveButton.type = 'button';
    saveButton.textContent = 'Save';
    const cancelButton = document.createElement('button');
    cancelButton.type = 'button';
    cancelButton.textContent = 'Cancel';
    actions.append(saveButton, cancelButton);
    editor.append(textarea, actions);
    block.replaceWith(editor);
    autoResize();
    const spellCheck = attachSpellCheck(textarea);
    const removalObserver = new MutationObserver(() => {
        if (!container.contains(editor)) {
            detachSpellCheck();
        }
    });
    const detachSpellCheck = () => {
        removalObserver.disconnect();
        spellCheck.detach();
    };
    removalObserver.observe(container, { childList: true, subtree: true });

    let appliedText = sourceText;
    let sourceApplied = false;
    let saving = false;

    const cancel = () => {
        if (saving) {
            return;
        }
        if (sourceApplied) {
            // Restore through an edit, not a reset, so LSP and spellcheck see
            // the original text again; then undo the edit's side effects.
            replaceMainDocumentText(source, { source: 'notes-run-markdown-cancel' });
            if (state.autosaveTimer) {
                clearTimeout(state.autosaveTimer);
                state.autosaveTimer = null;
            }
            setDirty(wasDirty);
            state.viewTainted = wasViewTainted;
        }
        detachSpellCheck();
        editor.replaceWith(block);
    };

    const save = async () => {
        if (saving) {
            return;
        }
        saving = true;
        saveButton.disabled = true;
        cancelButton.disabled = true;

        const currentSource = getMainEditorValue();
        const expectedSource = sourceApplied
            ? source.slice(0, start) + appliedText + source.slice(end)
            : source;
        if (currentSource !== expectedSource) {
            notifyTerminal('Markdown changed while this block was being edited. Cancel and reopen Run.', 'warn');
            saving = false;
            saveButton.disabled = false;
            cancelButton.disabled = false;
            return;
        }

        const nextText = textarea.value;
        if (nextText !== appliedText || !sourceApplied) {
            const rangeEnd = sourceApplied ? start + appliedText.length : end;
            mainDocument.replaceRange(start, rangeEnd, nextText, {
                source: 'notes-run-markdown-edit',
                preserveSelection: true,
            });
            appliedText = nextText;
            sourceApplied = true;
            state.viewTainted = true;
            setDirty(true);
        }

        if (state.autosaveTimer) {
            clearTimeout(state.autosaveTimer);
            state.autosaveTimer = null;
        }
        const saved = await saveFile();
        if (!saved) {
            saving = false;
            saveButton.disabled = false;
            cancelButton.disabled = false;
            return;
        }

        state.viewTainted = false;
        detachSpellCheck();
        await renderJupyterView();
    };

    saveButton.addEventListener('click', () => { void save(); });
    cancelButton.addEventListener('click', cancel);
    textarea.addEventListener('keydown', (event) => {
        if (event.key === 'Escape') {
            event.preventDefault();
            cancel();
        } else if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
            event.preventDefault();
            void save();
        }
    });

    textarea.focus();
}

function convertToJupyterCodeBlocks() {
    const codeBlocks = elements.jupyter.querySelectorAll('pre');
    
    codeBlocks.forEach((pre) => {
        const code = pre.querySelector('code');
        if (!code) return;
        
        const langClass = Array.from(code.classList).find(cls => cls.startsWith('language-'));
        const language = langClass ? langClass.replace('language-', '') : '';
        const blockId = `jupyter-block-${state.jupyterBlockCounter++}`;
        const content = code.textContent;
        
        state.jupyterCodeBlocks[blockId] = {
            language,
            runtime: language,
            originalContent: content,
            currentContent: content,
            lspMode: false,
            lspFilePath: ''
        };
        
        const wrapper = document.createElement('div');
        wrapper.className = 'jupyter-code-block';
        wrapper.dataset.blockId = blockId;
        wrapper.addEventListener('dblclick', (event) => {
            event.stopPropagation();
        });
        
        const toolbar = document.createElement('div');
        toolbar.className = 'jupyter-toolbar';
        
        const runNotesBtn = document.createElement('button');
        runNotesBtn.type = 'button';
        runNotesBtn.className = 'jupyter-btn jupyter-run-notes';
        runNotesBtn.textContent = 'Run';
        runNotesBtn.addEventListener('click', () => runCodeBlockInNotes(blockId));
        
        const stopNotesBtn = document.createElement('button');
        stopNotesBtn.type = 'button';
        stopNotesBtn.className = 'jupyter-btn jupyter-stop-notes';
        stopNotesBtn.textContent = 'Stop';
        stopNotesBtn.style.display = 'none'; // Initially hidden
        stopNotesBtn.addEventListener('click', () => stopCodeBlockInNotes(blockId));
        
        const copyBtn = document.createElement('button');
        copyBtn.type = 'button';
        copyBtn.className = 'jupyter-btn jupyter-copy-notes';
        copyBtn.textContent = 'Copy';
        copyBtn.title = 'Copy code to clipboard';
        copyBtn.addEventListener('click', async () => {
            try {
                await ClipboardSetText(editableCode.value || '');
                copyBtn.dataset.copied = 'true';
                //notifyTerminal('Code copied to clipboard', 'info');
                
                // Remove animation state after animation completes (0.48s)
                setTimeout(() => {
                    copyBtn.dataset.copied = 'false';
                }, 480);
            } catch (err) {
                console.error('Error copying to clipboard:', err);
                notifyTerminal('Failed to copy to clipboard', 'error');
            }
        });
        
        const runTerminalBtn = document.createElement('button');
        runTerminalBtn.type = 'button';
        runTerminalBtn.className = 'jupyter-btn jupyter-run-terminal';
        runTerminalBtn.textContent = 'Send to terminal';
        runTerminalBtn.addEventListener('click', () => runCodeBlockInTerminal(blockId));
        
        const lspBtn = document.createElement('button');
        lspBtn.type = 'button';
        lspBtn.className = 'jupyter-btn jupyter-lsp-notes';
        lspBtn.textContent = 'LSP';
        lspBtn.dataset.lspEnabled = 'false';
        lspBtn.addEventListener('click', () => toggleLspModeForBlock(blockId));
        
        const runtimeLink = document.createElement('button');
        runtimeLink.type = 'button';
        runtimeLink.className = 'jupyter-runtime-dropdown';
        runtimeLink.title = 'Select runtime';
        runtimeLink.textContent = language || 'language unknown';

        let runtimeOptions = [];

        // Load runtime options immediately
        (async () => {
            try {
                const hasLanguage = Boolean(language);
                let descriptions = [];
                let defaultSelection = '';

                if (hasLanguage) {
                    const matches = await GetLanguageDescriptions(language);
                    if (matches && matches.length > 0) {
                        // Markdown language exists in YAML: only show those options
                        descriptions = matches;
                        defaultSelection = matches[0];
                    } else {
                        // Markdown language not in YAML: show all options, default to markdown language
                        descriptions = await GetAllLanguageDescriptions();
                        descriptions.sort((a, b) => a.localeCompare(b));
                        defaultSelection = language;
                    }
                } else {
                    // No markdown language: autodetect using highlight.js
                    let detectedLanguage = '';
                    if (content) {
                        try {
                            const result = hljs.highlightAuto(content);
                            if (result && result.language) {
                                detectedLanguage = result.language;
                            }
                        } catch (err) {
                            console.warn('Highlight.js autodetection failed:', err);
                        }
                    }

                    descriptions = await GetAllLanguageDescriptions();
                    descriptions.sort((a, b) => a.localeCompare(b));

                    if (detectedLanguage) {
                        const detectedMatches = await GetLanguageDescriptions(detectedLanguage);
                        defaultSelection = detectedMatches && detectedMatches.length > 0
                            ? detectedMatches[0]
                            : 'language unknown';
                    } else {
                        defaultSelection = 'language unknown';
                    }
                }

                // Build ordered options list (prepend custom default if not already present)
                runtimeOptions = [];
                if (defaultSelection && !descriptions.includes(defaultSelection)) {
                    runtimeOptions.push(defaultSelection);
                }
                runtimeOptions.push(...descriptions);

                // Set runtime state and update button label
                const resolved = defaultSelection
                    || (descriptions.length > 0 ? descriptions[0] : language || 'language unknown');
                state.jupyterCodeBlocks[blockId].runtime = resolved;
                runtimeLink.textContent = resolved;

            } catch (err) {
                console.error('Error fetching language descriptions:', err);
                const fallback = language || 'language unknown';
                runtimeOptions = [fallback];
                state.jupyterCodeBlocks[blockId].runtime = fallback;
                runtimeLink.textContent = fallback;
            }
            
            // Restore LSP mode and other state from document cache
            await restoreJupyterBlockState(blockId);
            void refreshJupyterReservedWords(blockId);
        })();

        runtimeLink.addEventListener('click', () => {
            const rect = runtimeLink.getBoundingClientRect();
            showNotesLocalMenu(
                runtimeOptions.map((desc) => ({
                    title: desc,
                    icon: desc === state.jupyterCodeBlocks[blockId].runtime ? 0xf00c : 0,
                    onSelect: () => {
                        state.jupyterCodeBlocks[blockId].runtime = desc;
                        runtimeLink.textContent = desc;
                        void refreshJupyterReservedWords(blockId);
                    },
                })),
                rect.left,
                rect.bottom,
                'Select runtime',
            );
        });
        
        toolbar.appendChild(runNotesBtn);
        toolbar.appendChild(stopNotesBtn);
        toolbar.appendChild(lspBtn);
        toolbar.appendChild(copyBtn);
        toolbar.appendChild(runTerminalBtn);
        toolbar.appendChild(runtimeLink);
        
        const editableCode = document.createElement('textarea');
        editableCode.className = 'jupyter-code-editable';
        editableCode.dataset.language = language;
        editableCode.value = content;
        editableCode.setAttribute('autocorrect', 'off');
        editableCode.setAttribute('autocapitalize', 'off');
        editableCode.setAttribute('autocomplete', 'off');
        editableCode.setAttribute('spellcheck', 'false');
        editableCode.setAttribute('data-gramm', 'false');
        editableCode.setAttribute('data-gramm_editor', 'false');
        editableCode.setAttribute('data-enable-grammarly', 'false');

        const codeEditor = document.createElement('div');
        codeEditor.className = 'jupyter-code-editor';

        const lineNumbers = document.createElement('div');
        lineNumbers.className = 'jupyter-line-numbers';
        const lineNumbersInner = document.createElement('div');
        lineNumbersInner.className = 'jupyter-line-numbers-inner';
        lineNumbers.appendChild(lineNumbersInner);

        // Syntax highlight layer — sits behind the textarea
        const highlightPre = document.createElement('pre');
        highlightPre.className = 'jupyter-highlight';
        highlightPre.setAttribute('aria-hidden', 'true');
        const highlightCode = document.createElement('code');
        highlightCode.className = `hljs language-${language || 'plaintext'}`;
        highlightPre.appendChild(highlightCode);

        const renderHighlight = () => {
            const code = editableCode.value;
            const lang = state.jupyterCodeBlocks[blockId]?.language || language;
            try {
                if (lang && hljs.getLanguage(lang)) {
                    highlightCode.innerHTML = hljs.highlight(code, { language: lang, ignoreIllegals: true }).value;
                } else {
                    highlightCode.innerHTML = hljs.highlightAuto(code).value;
                }
            } catch {
                highlightCode.textContent = code;
            }
        };

        // Wrapper that positions the highlight layer and textarea together
        const codeArea = document.createElement('div');
        codeArea.className = 'jupyter-code-area';

        const renderLineNumbers = () => {
            const lineCount = Math.max(1, editableCode.value.split('\n').length);
            lineNumbersInner.textContent = Array.from({ length: lineCount }, (_, i) => String(i + 1)).join('\n');
        };
        
        // Auto-resize textarea to fit content
        const autoResize = () => {
            editableCode.style.height = 'auto';
            const maxHeight = parseFloat(getComputedStyle(editableCode).maxHeight || '0');
            const targetHeight = Number.isFinite(maxHeight) && maxHeight > 0
                ? Math.min(editableCode.scrollHeight, maxHeight)
                : editableCode.scrollHeight;
            editableCode.style.height = `${targetHeight}px`;
        };

        const syncHighlightViewport = () => {
            const contentWidth = Math.max(editableCode.scrollWidth, editableCode.clientWidth);
            const contentHeight = Math.max(editableCode.scrollHeight, editableCode.clientHeight);
            highlightPre.style.minWidth = `${contentWidth}px`;
            highlightPre.style.minHeight = `${contentHeight}px`;
            highlightPre.style.transform = `translate(${-editableCode.scrollLeft}px, ${-editableCode.scrollTop}px)`;
            lineNumbersInner.style.minHeight = `${contentHeight}px`;
            lineNumbersInner.style.minWidth = `${lineNumbers.clientWidth}px`;
            lineNumbersInner.style.transform = `translateY(${-editableCode.scrollTop}px)`;
        };

        editableCode.addEventListener('input', () => {
            autoResize();
            renderLineNumbers();
            renderHighlight();
            syncHighlightViewport();
            const blockState = state.jupyterCodeBlocks[blockId];
            if (!blockState) {
                return;
            }
            blockState.currentContent = editableCode.value;

            if (blockState.typosFilePath) {
                scheduleTyposBlockChange(blockId, editableCode);
            }

            if (blockState.lspMode && blockState.lspFilePath) {
                // Trigger completion on LSP trigger characters.
                if (state.lspActiveBlockId === blockId) {
                    const cursor = editableCode.selectionStart || 0;
                    const value = editableCode.value || '';
                    const prevChar = cursor > 0 ? value[cursor - 1] : '';
                    state.lspHoverLastKey = '';
                    hideLspHoverTooltip();
                    if (state.lspCompletionVisible) {
                        void requestLspCompletionAfterSync(value, '', 1);
                    } else if (prevChar === '.' || prevChar === ':' || prevChar === '>') {
                        void requestLspCompletionAfterSync(value, prevChar);
                    } else if (!/[A-Za-z0-9_-]/.test(prevChar)) {
                        hideLspCompletion();
                    }
                }
            }

            const blockIndex = parseInt(blockId.replace('jupyter-block-', ''), 10);
            if (Number.isNaN(blockIndex)) {
                return;
            }

            const updated = updateMarkdownCodeBlock(blockIndex, blockState.currentContent);
            if (!updated) {
                return;
            }

            setDirty(true);
            scheduleRender();
            scheduleAutoSave();
        });
        editableCode.addEventListener('focus', () => {
            const block = state.jupyterCodeBlocks[blockId];
            if (block && block.lspMode && block.lspFilePath) {
                state.lspActiveBlockId = blockId;
                state.lspActiveBlockEditor = editableCode;
            }
        });

        editableCode.addEventListener('blur', () => {
            if (state.lspActiveBlockId === blockId) {
                state.lspActiveBlockId = null;
                state.lspActiveBlockEditor = null;
                hideLspHoverTooltip();
                hideLspCompletion();
                state.lspHoverLastKey = '';
            }
        });

        editableCode.addEventListener('mousemove', (event) => {
            state.lspHoverMouseX = event.clientX;
            state.lspHoverMouseY = event.clientY;
        });

        editableCode.addEventListener('mouseup', () => {
            if (state.lspActiveBlockId === blockId) {
                state.lspHoverLastKey = '';
                scheduleLspHover();
            }
        });

        editableCode.addEventListener('keyup', () => {
            if (state.lspActiveBlockId === blockId) {
                scheduleLspHover();
            }
        });

        editableCode.addEventListener('keydown', async (event) => {
            if (state.lspActiveBlockId === blockId && !event.ctrlKey && !event.metaKey && !event.altKey) {
                if (event.key === 'Escape' && closeOpenLspTooltips()) {
                    event.preventDefault();
                    event.stopPropagation();
                    return;
                }
                if (event.key === 'Enter' && commitActiveLspCompletion()) {
                    event.preventDefault();
                    event.stopPropagation();
                    return;
                }
                if (event.key === 'ArrowDown' && moveLspCompletionSelection(1)) {
                    event.preventDefault();
                    event.stopPropagation();
                    return;
                }
                if (event.key === 'ArrowUp' && moveLspCompletionSelection(-1)) {
                    event.preventDefault();
                    event.stopPropagation();
                    return;
                }
            }

            const blockState = state.jupyterCodeBlocks[blockId] || {};
            if (maybeHandleSyntaxCompletionKey(event, editableCode, {
                docPath: state.currentFile || '',
                languageHint: blockState.runtime || blockState.language || editableCode.dataset.language || '',
            })) {
                return;
            }

            if (event.key !== 'Tab' || event.ctrlKey || event.metaKey || event.altKey) {
                return;
            }

            const hasSelection = (editableCode.selectionEnd || 0) > (editableCode.selectionStart || 0);
            if (hasSelection) {
                event.preventDefault();
                event.stopPropagation();
                const indentation = await getIndentationString();
                const edit = buildLineIndentationEdit(
                    editableCode.value || '',
                    editableCode.selectionStart || 0,
                    editableCode.selectionEnd || 0,
                    indentation,
                    event.shiftKey === true,
                );

                if (!edit) {
                    return;
                }

                notesMutationAdapter.replaceDocumentText(editableCode, {
                    text: editableCode.value.slice(0, edit.start) + edit.text + editableCode.value.slice(edit.end),
                    selectionStart: edit.selectionStart,
                    selectionEnd: edit.selectionEnd,
                    source: event.shiftKey === true ? 'jupyter-shift-tab-outdent' : 'jupyter-tab-indent-lines',
                    label: event.shiftKey === true ? 'Jupyter shift+tab outdent selection' : 'Jupyter tab indent selection',
                    filePath: state.currentFile ? `${state.currentFile}#${blockId}` : blockId,
                    emit: true,
                });
                return;
            }

            if (state.lspActiveBlockId === blockId && state.lspCompletionVisible) {
                hideLspCompletion();
                event.preventDefault();
                event.stopPropagation();
                const start = editableCode.selectionStart;
                const end = editableCode.selectionEnd;
                const indentation = await getIndentationString();
                notesMutationAdapter.replaceRange(editableCode, {
                    start,
                    end,
                    text: indentation,
                    source: 'jupyter-tab-indent',
                    label: 'Jupyter tab indentation (completion open)',
                    filePath: state.currentFile ? `${state.currentFile}#${blockId}` : blockId,
                    emit: true,
                });
                return;
            }

            if (state.lspActiveBlockId === blockId) {
                const source = editableCode.value || '';
                const cursor = editableCode.selectionStart || 0;
                const lineStart = source.lastIndexOf('\n', Math.max(0, cursor - 1)) + 1;
                const leftOfCaret = source.slice(lineStart, cursor);
                if (!/^\s*$/.test(leftOfCaret)) {
                    event.preventDefault();
                    event.stopPropagation();
                    hideLspHoverTooltip();
                    hideLspCompletion();
                    void requestLspCompletion();
                    return;
                }
            }

            // Insert tab character and keep focus in the code editor
            event.preventDefault();
            event.stopPropagation();

            const start = editableCode.selectionStart;
            const end = editableCode.selectionEnd;
            const indentation = await getIndentationString();
            notesMutationAdapter.replaceRange(editableCode, {
                start,
                end,
                text: indentation,
                source: 'jupyter-tab-indent',
                label: 'Jupyter tab indentation',
                filePath: state.currentFile ? `${state.currentFile}#${blockId}` : blockId,
                emit: true,
            });
        });
        editableCode.addEventListener('scroll', () => {
            syncHighlightViewport();
            if (state.lspActiveBlockId === blockId) {
                hideLspHoverTooltip();
                hideLspCompletion();
            }
        });
        // Set initial height and highlight
        setTimeout(() => {
            autoResize();
            renderLineNumbers();
            renderHighlight();
            syncHighlightViewport();
        }, 0);
        
        const outputWrapper = document.createElement('div');
        outputWrapper.className = 'jupyter-output-wrapper';
        outputWrapper.style.display = 'none'; // Initially hidden
        
        const outputToggle = document.createElement('button');
        outputToggle.type = 'button';
        outputToggle.className = 'jupyter-output-toggle';
        outputToggle.textContent = 'Output ▾';
        outputToggle.dataset.collapsed = 'false';
        
        const outputBlock = document.createElement('pre');
        outputBlock.className = 'jupyter-output';
        outputBlock.textContent = '';
        outputBlock.style.display = 'block';
        
        outputToggle.addEventListener('click', () => {
            const isCollapsed = outputBlock.style.display === 'none';
            outputBlock.style.display = isCollapsed ? 'block' : 'none';
            outputToggle.textContent = isCollapsed ? 'Output ▾' : 'Output ▸';
            outputToggle.dataset.collapsed = isCollapsed ? 'false' : 'true';
        });
        
        outputWrapper.appendChild(outputToggle);
        outputWrapper.appendChild(outputBlock);
        
        pre.replaceWith(wrapper);
        wrapper.appendChild(toolbar);
        codeArea.appendChild(highlightPre);
        codeArea.appendChild(editableCode);
        codeEditor.appendChild(lineNumbers);
        codeEditor.appendChild(codeArea);
        wrapper.appendChild(codeEditor);
        wrapper.appendChild(outputWrapper);
        restoreJupyterBlockRun(blockId, { runButton: runNotesBtn, stopButton: stopNotesBtn, outputWrapper, outputBlock });

        attachVimMode(editableCode, {
            mutationAdapter: notesMutationAdapter,
            undoManager: notesUndoManager,
            filePathResolver: () => (state.currentFile ? `${state.currentFile}#${blockId}` : blockId),
        });
        // Detach any stale spellcheck overlay from a previous render of this
        // block before attaching a fresh one, then restore typos mode if the
        // block currently has LSP-mode spellchecking active.
        if (jupyterSpellCheckHandles[blockId]) {
            try { jupyterSpellCheckHandles[blockId].detach(); } catch { /* already gone */ }
        }
        const blockSpellCheck = attachSpellCheck(editableCode);
        jupyterSpellCheckHandles[blockId] = blockSpellCheck;
        void refreshJupyterReservedWords(blockId);
        const blockForSpell = state.jupyterCodeBlocks[blockId];
        if (blockForSpell && blockForSpell.typosFilePath) {
            blockSpellCheck.setMode('external');
        }
    });
}

async function runCodeBlockInNotes(blockId) {
    const block = state.jupyterCodeBlocks[blockId];
    if (!block) return;
    
    const editableElement = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-code-editable`);
    if (editableElement) {
        block.currentContent = editableElement.value;
    }
    
    // Toggle Run/Stop buttons
    const runBtn = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-run-notes`);
    const stopBtn = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-stop-notes`);
    if (runBtn) runBtn.style.display = 'none';
    if (stopBtn) stopBtn.style.display = 'inline-block';
    
    // Show the output wrapper when running
    const outputWrapper = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-output-wrapper`);
    if (outputWrapper) {
        outputWrapper.style.display = 'block';
    }
    
    // Clear previous output
    const outputBlock = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-output`);
    if (outputBlock) {
        outputBlock.textContent = '';
    }
    
    const doc = activeNotesDocument;
    doc.jupyterOutputs.set(blockId, []);
    const runId = startJupyterRun(doc, blockId);
    try {
        await RunNote(doc.path, blockId, runId, block.currentContent, block.runtime);
    } catch (err) {
        finishJupyterRun(runId);
        console.error('Error running code:', err);
        const outputBlock = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-output`);
        if (outputBlock) {
            outputBlock.textContent = `Error: ${err.message}`;
        }
        // Reset buttons on error
        if (runBtn) runBtn.style.display = 'inline-block';
        if (stopBtn) stopBtn.style.display = 'none';
    }
}

function scrollJupyterOutputToBottom(outputBlock) {
    if (!outputBlock) {
        return;
    }

    outputBlock.scrollTop = outputBlock.scrollHeight;
}

async function stopCodeBlockInNotes(blockId) {
    const runId = activeNotesDocument.runningJupyterBlocks.get(blockId);
    if (runId) {
        try {
            await StopNote(runId);
        } catch (err) {
            console.error('Error stopping code:', err);
        }
    }
    
    // Toggle buttons back
    const runBtn = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-run-notes`);
    const stopBtn = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-stop-notes`);
    if (runBtn) runBtn.style.display = 'inline-block';
    if (stopBtn) stopBtn.style.display = 'none';
}

async function runCodeBlockInTerminal(blockId) {
    const block = state.jupyterCodeBlocks[blockId];
    if (!block) return;
    
    const editableElement = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-code-editable`);
    if (editableElement) {
        block.currentContent = editableElement.value;
    }
    
    try {
        await SendToTerminal(block.currentContent, block.runtime);
        window.dispatchEvent(new CustomEvent('ttyphoon-focus-terminal'));
    } catch (err) {
        console.error('Error sending to terminal:', err);
        notifyTerminal(`Failed to send to terminal: ${err}`, 'error');
    }
}

async function toggleLspModeForBlock(blockId) {
    const block = state.jupyterCodeBlocks[blockId];
    if (!block) return;

    const lspBtn = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-lsp-notes`);
    const editableElement = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-code-editable`);

    if (!lspBtn || !editableElement) return;

    // Toggle LSP mode
    block.lspMode = !block.lspMode;
    lspBtn.dataset.lspEnabled = String(block.lspMode);

    // If enabling LSP mode, format the code
    if (block.lspMode) {
        block.currentContent = editableElement.value;
        
        try {
            const result = await FormatCodeBlock(state.currentFile, blockId, block.currentContent, block.runtime);
            const goErr = typeof result?.Err === 'string' ? result.Err : '';

            if (goErr) {
                notifyTerminal(goErr, 'error');
                // Disable LSP mode on error
                block.lspMode = false;
                lspBtn.dataset.lspEnabled = 'false';
                return;
            }

            const formattedCode = typeof result?.Code === 'string' ? result.Code : '';
            const formattedFilePath = typeof result?.FilePath === 'string' ? result.FilePath : '';

            if (applyFormattedText(editableElement, formattedCode, {
                source: 'jupyter-lsp-toggle-format',
                label: 'Format Jupyter block (toggle LSP)',
                filePath: state.currentFile ? `${state.currentFile}#${blockId}` : blockId,
            })) {
                block.currentContent = editableElement.value;

                // Trigger input event to update syntax highlighting and markdown source.
                editableElement.dispatchEvent(new Event('input', { bubbles: true }));
            }

            if (formattedFilePath.length > 0) {
                block.lspFilePath = formattedFilePath;
                await NotesLspOpenDocument(formattedFilePath, '', block.currentContent);

                // Enable typos-lsp spellchecking for this block, rendering with
                // the same red wavy underline as aspell. Falls back to aspell.
                try {
                    const typosOk = await NotesTyposOpenDocument(formattedFilePath, '', block.currentContent);
                    const handle = jupyterSpellCheckHandles[blockId];
                    if (typosOk) {
                        block.typosFilePath = formattedFilePath;
                        handle?.setMode('external');
                    } else {
                        block.typosFilePath = '';
                        handle?.setMode('aspell');
                    }
                } catch (err) {
                    block.typosFilePath = '';
                    jupyterSpellCheckHandles[blockId]?.setMode('aspell');
                    console.error('block typos open failed:', err);
                }

                // If this block's textarea is currently focused, activate it as the LSP target.
                if (document.activeElement === editableElement) {
                    state.lspActiveBlockId = blockId;
                    state.lspActiveBlockEditor = editableElement;
                }
            }
        } catch (err) {
            console.error('Error formatting code:', err);
            notifyTerminal(String(err && err.message ? err.message : err), 'error');
            // Disable LSP mode on error
            block.lspMode = false;
            lspBtn.dataset.lspEnabled = 'false';
        }
    } else if (block.lspFilePath) {
        try {
            await NotesLspCloseDocument(block.lspFilePath);
        } catch (err) {
            console.error('Error closing LSP document for code block:', err);
        }
        if (block.typosFilePath) {
            try {
                await NotesTyposCloseDocument(block.typosFilePath);
            } catch (err) {
                console.error('Error closing typos document for code block:', err);
            }
            block.typosFilePath = '';
            jupyterSpellCheckHandles[blockId]?.setMode('aspell');
        }
        block.lspFilePath = '';
        // Clear active LSP target if this block was active.
        if (state.lspActiveBlockId === blockId) {
            state.lspActiveBlockId = null;
            state.lspActiveBlockEditor = null;
            hideLspHoverTooltip();
            hideLspCompletion();
        }
    }

    // Persist LSP mode state to document cache
    await persistJupyterBlockState(blockId);
}

async function persistJupyterBlockState(blockId) {
    try {
        const docCache = await GetDocumentCache(state.currentFile);
        if (!docCache) return;

        if (!docCache.jupyterBlockState) {
            docCache.jupyterBlockState = {};
        }

        const block = state.jupyterCodeBlocks[blockId];
        if (block) {
            docCache.jupyterBlockState[blockId] = {
                lspMode: block.lspMode,
                runtime: block.runtime
            };
        }

        await SetDocumentCache(state.currentFile, docCache);
    } catch (err) {
        console.error('Error persisting code block state:', err);
    }
}

async function restoreJupyterBlockState(blockId) {
    const block = state.jupyterCodeBlocks[blockId];
    if (!block) return;

    const lspBtn = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-lsp-notes`);

    // Restore saved runtime + LSP preference from the document cache. A saved
    // lspMode of `false` represents an explicit user opt-out; the absence of any
    // saved state means we fall back to the LSP-availability default below.
    let savedLspMode = null;
    try {
        const docCache = await GetDocumentCache(state.currentFile);
        const savedState = docCache?.jupyterBlockState?.[blockId];
        if (savedState) {
            if (typeof savedState.lspMode === 'boolean') {
                savedLspMode = savedState.lspMode;
            }
            if (savedState.runtime) {
                block.runtime = savedState.runtime;
                const runtimeLink = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-runtime-dropdown`);
                if (runtimeLink) {
                    runtimeLink.textContent = savedState.runtime;
                }
            }
        }
    } catch (err) {
        console.error('Error restoring code block state:', err);
    }

    // Determine whether an LSP server is available for this runtime. When none
    // is available, hide the LSP button entirely. When one is available, enable
    // LSP mode by default unless the user explicitly disabled it previously.
    let lspAvailable = false;
    try {
        lspAvailable = await NotesLspAvailableForRuntime(block.runtime);
    } catch (err) {
        console.error('LSP availability check failed:', err);
    }

    if (!lspBtn) return;

    if (!lspAvailable) {
        lspBtn.style.display = 'none';
        return;
    }
    lspBtn.style.display = '';

    const shouldEnable = savedLspMode === null ? true : savedLspMode;
    if (shouldEnable && !block.lspMode) {
        await toggleLspModeForBlock(blockId);
    }
}

let fileListRefreshSequence = 0;
let requestedFileContext = null;
let fileListRefreshInFlight = 0;
let fileIndexRefreshPending = false;
let fileIndexRefreshQueued = false;
let fileIndexRoots = new Map();

function normalizeFileIndexRoot(path) {
    return String(path || '').replaceAll('\\', '/').replace(/\/+$/, '');
}

function indexedFileList(context, roots) {
    const files = [];
    for (const [field, prefix, recursive] of [
        ['globalNotes', '$GLOBAL', false], ['userNotes', '$NOTES', false], ['projectRoot', '$PROJECT', true],
    ]) {
        const snapshot = roots.get(normalizeFileIndexRoot(context[field]));
        if (!snapshot) continue;
        for (const path of snapshot.files || []) {
            if (recursive || !/[\\/]/.test(path)) files.push(`${prefix}/${path}`);
        }
    }
    return files;
}

function indexedFileStatus(roots) {
    const snapshots = Array.from(roots.values());
    const busy = snapshots.some((root) => root.state === 'scanning' || root.state === 'refreshing');
    const error = snapshots.map((root) => root.error || '').filter(Boolean).join('\n');
    const truncated = snapshots.some((root) => root.truncated);
    return { state: busy ? 'refreshing' : error ? 'error' : truncated ? 'truncated' : 'ready', error, truncated };
}

function renderFileIndexStatus(snapshot) {
    const status = document.getElementById('notes-file-index-status');
    const spinner = document.getElementById('notes-file-index-spinner');
    const warning = document.getElementById('notes-file-index-warning');
    const busy = snapshot.state === 'scanning' || snapshot.state === 'refreshing';
    const problem = Boolean(snapshot.error || snapshot.truncated);
    spinner.hidden = !busy;
    warning.hidden = busy || !problem;
    status.hidden = !busy && !problem;
    const label = busy ? 'Refreshing files'
        : snapshot.error || (snapshot.truncated ? 'File list truncated at the filesystem entry limit' : '');
    status.title = label;
    status.setAttribute('aria-label', label);
    elements.list.setAttribute('aria-busy', busy ? 'true' : 'false');
}

function queueFileIndexRefresh() {
    fileIndexRefreshPending = true;
    if (fileIndexRefreshQueued || fileListRefreshInFlight > 0) return;
    fileIndexRefreshQueued = true;
    queueMicrotask(() => {
        fileIndexRefreshQueued = false;
        if (!fileIndexRefreshPending || fileListRefreshInFlight > 0 || !requestedFileContext) return;
        fileIndexRefreshPending = false;
        void refreshFiles({ context: requestedFileContext, skipHistoryRestore: true });
    });
}

async function refreshFiles(options = {}) {
    const sequence = ++fileListRefreshSequence;
    fileListRefreshInFlight++;
    try {
        const skipHistoryRestore = Boolean(options?.skipHistoryRestore);
        const previousWorkspaceKey = String(state.currentWorkspaceKey || '');
        const context = options.context || await GetNotesContext();
        if (sequence !== fileListRefreshSequence) return;
        requestedFileContext = context;
        state.currentProjectRoot = context.projectRoot || '';
        const workspaceName = context.groupName || '';
        const workspaceKey = `${String(state.currentProjectRoot || '')}::${String(workspaceName || '')}`;
        const workspaceChanged = previousWorkspaceKey !== '' && previousWorkspaceKey !== workspaceKey;
        const workspaceScopedRefresh = previousWorkspaceKey === '' || workspaceChanged;

        if (workspaceChanged) {
            clearTimeout(state.historyPositionTimer);
            state.historyPositionTimer = null;
            notesGitPanel?.setCommitsActive(false);
            notesTabsWorkspace = null;
            await closeBackgroundNotesTabs();
            if (sequence !== fileListRefreshSequence) return;
        }

        state.currentWorkspaceName = String(workspaceName || '');
        state.currentWorkspaceKey = workspaceKey;
        elements.title.innerText = workspaceName;

        if (workspaceScopedRefresh) {
            // Defer AI log loading until the AI tab is selected so it never
            // blocks workspace/file switching. Loading is triggered lazily by
            // setToolsTab('ai') / the AI tab click.
            markAISessionCachePending(workspaceName);
            void refreshAIModelPicker();
        }

        const snapshot = await GetNotesFilesSnapshot(context, Boolean(options.refresh));
        if (sequence !== fileListRefreshSequence) return;
        renderFileIndexStatus(snapshot);
        fileIndexRoots = new Map((snapshot.roots || []).map((root) => [normalizeFileIndexRoot(root.root), root]));
        state.files = Array.isArray(snapshot.files) ? snapshot.files : [];
        state.filteredFiles = null;
        renderFileList();
        const isCurrent = () => sequence === fileListRefreshSequence;
        // A workspace's remembered tabs take the place of its history
        // position.
        const savedTabs = workspaceScopedRefresh ? await GetNotesOpenTabs() : null;
        if (!isCurrent()) return;
        const hasSavedTabs = savedTabs?.workspace === state.currentWorkspaceName
            && Array.isArray(savedTabs.files) && savedTabs.files.length > 0;
        await loadProjectCache({
            skipHistoryRestore: skipHistoryRestore || !workspaceScopedRefresh || hasSavedTabs,
            isCurrent,
        });
        if (!isCurrent()) return;
        if (workspaceScopedRefresh) {
            if (hasSavedTabs) {
                await restoreNotesOpenTabs(savedTabs, { loadActive: !skipHistoryRestore, isCurrent });
                if (!isCurrent()) return;
            }
            notesTabsWorkspace = state.currentWorkspaceName;
            saveNotesOpenTabs();
        }
        await applyFileFilter();

        if (sequence !== fileListRefreshSequence) return;
        if (workspaceChanged) {
            scrollActiveFileListItemIntoView();
        }
    } catch (err) {
        if (sequence !== fileListRefreshSequence) return;
        if (String(err).includes('context canceled')) {
            renderFileIndexStatus({ state: 'ready' });
            return;
        }
        renderFileIndexStatus({ state: 'error', error: String(err) });
        notifyTerminal('Failed to load file list', 'error');
        console.error(err);
    } finally {
        fileListRefreshInFlight--;
        if (fileIndexRefreshPending && fileListRefreshInFlight === 0) queueFileIndexRefresh();
    }
}

function openWorkspaceWindowMenu(event) {
    const tabs = Array.from(document.querySelectorAll('#terminal-tabs .terminal-tab[data-window-id]'))
        .filter((tab) => tab.dataset.windowId && tab.textContent.trim());
    if (tabs.length === 0) {
        return;
    }

    showLocalMenu({
        title: 'Choose a window',
        options: tabs.map((tab) => tab.textContent.trim()),
        x: event.clientX,
        y: event.clientY,
        showNextToMouseCursor: true,
        onSelect: (index) => tabs[index]?.click(),
    });
}

elements.title.addEventListener('click', openWorkspaceWindowMenu);

async function loadProjectCache(options = {}) {
    try {
        const skipHistoryRestore = Boolean(options?.skipHistoryRestore);
        const cache = await GetProjectCache();
        if (options.isCurrent && !options.isCurrent()) return;
        const collapsed = cache?.FileListCollapsed?.[state.currentProjectRoot] || [];
        state.expandedFolders = {};
        for (const key of collapsed) {
            state.expandedFolders[key] = false;
        }
        if (skipHistoryRestore) {
            return;
        }
        const historyPosition = await NotesHistoryCurrentPosition();
        if (options.isCurrent && !options.isCurrent()) return;
        const historyDoc = historyPosition?.file || '';
        if (historyDoc) {
            await loadFile(historyDoc);
            if (options.isCurrent && !options.isCurrent()) return;
            if (historyPosition.line > 1 && state.viewMode === editViewModeForCurrentFileType()) {
                await ensureMonacoVisibleAndLaidOut();
                jumpEditorToLine(historyPosition.line);
            }
        } else if (state.currentFile) {
            if (options.isCurrent && !options.isCurrent()) return;
            await showNoNotesDocument();
        }
    } catch (err) {
        console.error('Failed to load project cache:', err);
    }
}

// Closes the active document and shows an empty editor.
async function showNoNotesDocument() {
    resetLspViewState();
    closeActiveNotesDocument();
    state.swaggerSpec = null;
    state.swaggerRunAvailable = false;
    emitCurrentFileName();
    setMainEditorValue('');
    clearHexSource();
    clearVisibleLspDiagnostics();
    hideLspHoverTooltip();
    updateTabVisibility('markdown');
    setViewMode('viewer');
    await renderMarkdown();
    await renderJupyterView();
    setDirty(false);
}

// Records what the view shows for the current document, so returning to the
// document restores it. Skipped while a load has not yet put the document's
// content in the editor, as the editor state would belong to another file.
function rememberShownDocumentState() {
    const key = activeNotesDocument?.key;
    if (!state.currentFile || !key || focusedView().shownDocumentKey !== key) {
        return;
    }
    focusedView().rememberDocument(key, {
        viewMode: state.viewMode,
        editorState: focusedView().editor?.saveViewState() ?? null,
    });
}

function applyDocumentCache(documentCache, defaultMode) {
    if (typeof documentCache?.WordWrap === 'boolean') {
        setEditorWrapMode(documentCache.WordWrap === true);
    }

    setViewMode(documentCache?.DocumentTab || defaultMode);
    // The Tools panel open/closed state and selected tab are intentionally not
    // restored from cache. The panel keeps whatever state it already had; if the
    // active tab isn't available for the newly loaded document, select Find
    // without changing the panel's collapsed state.
}

function saveProjectCache() {
    const collapsed = Object.entries(state.expandedFolders)
        .filter(([, expanded]) => expanded === false)
        .map(([key]) => key);
    GetProjectCache().then((cache) => {
        const updated = cache || { FileListCollapsed: {} };
        if (!updated.FileListCollapsed) {
            updated.FileListCollapsed = {};
        }
        updated.FileListCollapsed[state.currentProjectRoot] = collapsed;
        return SetProjectCache(updated);
    }).catch((err) => {
        console.error('Failed to save project cache:', err);
    });
}

function getFilteredFiles() {
    return state.filteredFiles !== null ? state.filteredFiles : state.files;
}

async function applyFileFilter() {
    const sequence = fileListRefreshSequence;
    const query = state.fileFilterQuery.trim();
    if (notesGitPanel?.isCommitsActive()) {
        updateListFilterClearButtonVisibility();
        await notesGitPanel.setCommitFilter(query);
        return;
    }
    if (!query) {
        state.filteredFiles = null;
        renderFileList();
        return;
    }
    try {
        // FilterStrings now returns { List, Error }. A non-null Error means the
        // query was malformed or matched nothing, so show the empty (no-match)
        // state rather than falling back to the unfiltered list.
        const result = await FilterStrings(query, state.files);
        if (sequence !== fileListRefreshSequence || query !== state.fileFilterQuery.trim()) return;
        if (result && result.Error) {
            state.filteredFiles = [];
        } else {
            state.filteredFiles = Array.isArray(result?.List) ? result.List : [];
        }
    } catch {
        if (sequence !== fileListRefreshSequence) return;
        state.filteredFiles = null;
    }
    renderFileList();
}

function isProjectFindListModeActive() {
    return String(state.findFilesQuery || '').trim() !== '';
}

function getFindResultMatchLine(item) {
    const ctx = Array.isArray(item?.context) ? item.context : [];
    if (ctx.length === 0) {
        return '';
    }

    const lineNo = Number.parseInt(String(item?.line), 10) || 1;
    const matchIndex = lineNo === 1 ? 0 : Math.min(1, ctx.length - 1);
    return String(ctx[matchIndex] || '').trim();
}

function getVisibleProjectFindResults() {
    const filteredResults = state.findFilesResults;
    if (!filteredResults.length) {
        state.findFilesVirtualStart = 0;
        return [];
    }

    const maxStart = Math.max(filteredResults.length - FIND_FILES_RENDER_PAGE_SIZE, 0);
    if (state.findFilesVirtualStart > maxStart) {
        state.findFilesVirtualStart = maxStart;
    }

    const start = Math.max(state.findFilesVirtualStart, 0);
    return filteredResults.slice(start, start + FIND_FILES_RENDER_PAGE_SIZE);
}

function clampProjectFindScrollTop() {
    if (!elements.list || !state.findFilesResults.length) {
        return;
    }
    const viewportHeight = Math.max(Number(elements.list.clientHeight) || 0, 0);
    const maxScrollTop = Math.max(state.findFilesResults.length * FIND_FILES_VIRTUAL_ROW_HEIGHT - viewportHeight, 0);
    if (elements.list.scrollTop > maxScrollTop) {
        elements.list.scrollTop = maxScrollTop;
    }
}

function resetProjectFindPaging(options = {}) {
    const resetListScroll = options?.resetListScroll !== false;
    state.findFilesVirtualStart = 0;
    if (resetListScroll && elements.list) {
        elements.list.scrollTop = 0;
    }
}

function getProjectFindVirtualStartFromScroll(totalCount, scrollTop) {
    const maxStart = Math.max(totalCount - FIND_FILES_RENDER_PAGE_SIZE, 0);
    const rawStart = Math.floor(Math.max(scrollTop, 0) / FIND_FILES_VIRTUAL_ROW_HEIGHT);
    return Math.min(Math.max(rawStart, 0), maxStart);
}

function scheduleProjectFindVirtualScrollRender() {
    if (state.findFilesScrollRenderQueued) {
        return;
    }

    state.findFilesScrollRenderQueued = true;
    const schedule = typeof requestAnimationFrame === 'function'
        ? requestAnimationFrame
        : (cb) => setTimeout(cb, 16);

    schedule(() => {
        state.findFilesScrollRenderQueued = false;
        if (!isProjectFindListModeActive()) {
            return;
        }

        const filteredResults = state.findFilesResults;
        const nextStart = getProjectFindVirtualStartFromScroll(filteredResults.length, elements.list?.scrollTop || 0);
        if (nextStart === state.findFilesVirtualStart) {
            return;
        }

        state.findFilesVirtualStart = nextStart;
        renderProjectFindResults();
        renderFileList();
    });
}

function getSelectedProjectFindResultIndex(results) {
    const selectedKey = String(state.findFilesSelectedKey || '');
    if (!selectedKey || !Array.isArray(results) || results.length === 0) {
        return -1;
    }

    return results.findIndex((item) => {
        const lineNo = Number.parseInt(String(item?.line), 10) || 1;
        return `${String(item?.path || '')}:${lineNo}` === selectedKey;
    });
}

function renderProjectFindResultsInFileList() {
    if (!elements.list) {
        return;
    }

    const filteredResults = state.findFilesResults;
    const visibleResults = getVisibleProjectFindResults();

    if (!visibleResults.length) {
        const empty = document.createElement('div');
        empty.id = 'notes-empty';
        empty.textContent = 'No matching grep results.';
        elements.list.appendChild(empty);
        return;
    }

    const list = document.createElement('div');
    list.className = 'notes-find-files-list';

    const totalCount = filteredResults.length;
    const startIndex = Math.max(state.findFilesVirtualStart, 0);
    const endIndex = Math.min(startIndex + visibleResults.length, totalCount);

    if (startIndex > 0) {
        const topSpacer = document.createElement('div');
        topSpacer.className = 'notes-find-files-virtual-spacer';
        topSpacer.style.height = `${startIndex * FIND_FILES_VIRTUAL_ROW_HEIGHT}px`;
        list.appendChild(topSpacer);
    }

    visibleResults.forEach((item) => {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'notes-find-files-item';
        button.dataset.file = String(item?.path || '');

        const lineNo = Number.parseInt(String(item?.line), 10) || 1;
        const itemKey = `${String(item?.path || '')}:${lineNo}`;
        const isSelected = itemKey === state.findFilesSelectedKey;
        button.setAttribute('aria-selected', isSelected ? 'true' : 'false');

        const title = document.createElement('span');
        title.className = 'notes-find-files-item-title';
        title.textContent = String(item?.fileName || '');

        const detail = document.createElement('span');
        detail.className = 'notes-find-files-item-detail';
        const source = String(item?.source || state.findFilesSource || '').trim();
        const displayPath = String(item?.displayPath || item?.path || '');
        detail.textContent = `${displayPath}:${lineNo}`;

        button.appendChild(title);
        button.appendChild(detail);

        const ctx = Array.isArray(item?.context) ? item.context : [];
        if (ctx.length > 0) {
            const matchIndex = lineNo === 1 ? 0 : Math.min(1, ctx.length - 1);
            const pre = document.createElement('pre');
            pre.className = 'notes-find-files-item-context';
            ctx.forEach((ctxLine, i) => {
                const span = document.createElement('span');
                span.className = i === matchIndex
                    ? 'notes-find-files-context-match'
                    : 'notes-find-files-context-other';
                span.textContent = String(ctxLine).trim();
                pre.appendChild(span);
            });
            button.appendChild(pre);
        }

        button.addEventListener('click', () => {
            state.findFilesSelectedKey = itemKey;
            renderFileList();
            openProjectFindResult(item);
        });

        list.appendChild(button);
    });

    if (endIndex < totalCount) {
        const bottomSpacer = document.createElement('div');
        bottomSpacer.className = 'notes-find-files-virtual-spacer';
        bottomSpacer.style.height = `${(totalCount - endIndex) * FIND_FILES_VIRTUAL_ROW_HEIGHT}px`;
        list.appendChild(bottomSpacer);
    }

    elements.list.appendChild(list);
}

function updateListFilterClearButtonVisibility() {
    if (!elements.listFilterClear || !elements.listFilter) {
        return;
    }

    const hasValue = (elements.listFilter.value || '').trim().length > 0;
    elements.listFilterClear.dataset.visible = hasValue ? 'true' : 'false';
    elements.listFilterClear.setAttribute('aria-hidden', hasValue ? 'false' : 'true');
}

function renderFileList() {
    const projectFindMode = isProjectFindListModeActive();
    const previousScrollTop = elements.list ? elements.list.scrollTop : 0;

    closeSharedTooltip();
    updateListFilterClearButtonVisibility();
    elements.list.innerHTML = '';

    if (projectFindMode) {
        clampProjectFindScrollTop();
        const filteredResults = state.findFilesResults;
        state.findFilesVirtualStart = getProjectFindVirtualStartFromScroll(filteredResults.length, previousScrollTop);
        renderProjectFindResultsInFileList();
        if (elements.list && previousScrollTop > 0) {
            elements.list.scrollTop = previousScrollTop;
        }
        return;
    }

    const filteredFiles = getFilteredFiles();
    const hasActiveFilter = state.fileFilterQuery.trim() !== '';

    if (state.files.length === 0) {
        const empty = document.createElement('div');
        empty.id = 'notes-empty';
        empty.textContent = 'No notes found.';
        elements.list.appendChild(empty);
        if (elements.list && previousScrollTop > 0) {
            elements.list.scrollTop = previousScrollTop;
        }
        return;
    }

    if (filteredFiles.length === 0) {
        const empty = document.createElement('div');
        empty.id = 'notes-empty';
        empty.textContent = 'No matching files.';
        elements.list.appendChild(empty);
        if (elements.list && previousScrollTop > 0) {
            elements.list.scrollTop = previousScrollTop;
        }
        return;
    }

    // Group files by category
    const categories = {
        '$GLOBAL': [],
        '$NOTES': [],
        '$PROJECT': [],
        '$HISTORY': []
    };

    filteredFiles.forEach((file) => {
        const { category } = splitCategoryPath(file);

        if (category === '$GLOBAL') {
            categories['$GLOBAL'].push(file);
        } else if (category === '$NOTES') {
            categories['$NOTES'].push(file);
        } else if (category === '$PROJECT') {
            categories['$PROJECT'].push(file);
        } else if (category === '$HISTORY') {
            categories['$HISTORY'].push(file);
        }
    });

    // Render each category
    Object.keys(categories).forEach((category) => {
        const files = categories[category];
        if (files.length === 0) {
            return;
        }

        const categoryTree = buildFileTree(files);

        const categoryExpanded = hasActiveFilter ? true : state.expandedCategories[category];

        // Create category header
        const categoryHeader = document.createElement('div');
        categoryHeader.className = 'notes-category-header';
        categoryHeader.dataset.category = category;
        categoryHeader.dataset.expanded = categoryExpanded ? 'true' : 'false';
        
        const arrow = document.createElement('span');
        arrow.className = 'notes-category-arrow';
        arrow.textContent = categoryExpanded ? '▼' : '▶';
        
        const label = document.createElement('span');
        label.textContent = category;
        
        categoryHeader.appendChild(arrow);
        categoryHeader.appendChild(label);

        elements.list.appendChild(categoryHeader);

        // Create category content container
        const categoryContent = document.createElement('div');
        categoryContent.className = 'notes-category-content';
        categoryContent.dataset.expanded = categoryExpanded ? 'true' : 'false';

        renderTreeNodesList(categoryContent, category, categoryTree);

        elements.list.appendChild(categoryContent);
    });

    if (elements.list && previousScrollTop > 0) {
        elements.list.scrollTop = previousScrollTop;
    }
}

function scrollActiveFileListItemIntoView() {
    if (!state.currentFile || !elements.list) {
        return;
    }

    const items = Array.from(elements.list.querySelectorAll('.notes-file'));
    const activeItem = items.find((item) => item?.dataset?.file === state.currentFile);
    if (!activeItem || typeof activeItem.scrollIntoView !== 'function') {
        return;
    }

    activeItem.scrollIntoView({
        //block: 'nearest',
        block: 'center',
        inline: 'nearest',
    });
}

function toggleCategory(category) {
    state.expandedCategories[category] = !state.expandedCategories[category];
    renderFileList();
}

function toggleFolder(folderKey) {
    state.expandedFolders[folderKey] = !(state.expandedFolders[folderKey] !== false);
    saveProjectCache();
    renderFileList();
}

function collectFolderKeys(category, nodes) {
    const keys = [];

    function walk(entries) {
        entries.forEach((entry) => {
            if (entry.type !== 'folder') {
                return;
            }

            keys.push(`${category}${PRIMARY_PATH_SEPARATOR}${entry.path}`);

            if (Array.isArray(entry.children) && entry.children.length > 0) {
                walk(entry.children);
            }
        });
    }

    walk(Array.isArray(nodes) ? nodes : []);
    return keys;
}

// Rebuilds a category's file tree on demand (e.g. for the delegated context-menu
// handler, which no longer keeps per-node tree references around after render).
function getCategoryTreeNodes(category) {
    const files = getFilteredFiles().filter((file) => splitCategoryPath(file).category === category);
    return buildFileTree(files);
}

function findFolderNodeByPath(nodes, path) {
    for (const node of nodes) {
        if (node.type !== 'folder') {
            continue;
        }
        if (node.path === path) {
            return node;
        }
        const found = findFolderNodeByPath(node.children, path);
        if (found) {
            return found;
        }
    }
    return null;
}

async function openFileParentFolderContextMenu(file, x, y) {
    const { category, relativePath } = splitCategoryPath(file);
    if (!category) {
        return;
    }

    const segments = getPathParts(relativePath);
    const folderPath = segments.slice(0, -1).join(PRIMARY_PATH_SEPARATOR);
    const categoryNodes = getCategoryTreeNodes(category);
    const folderNode = folderPath ? findFolderNodeByPath(categoryNodes, folderPath) : null;
    const folderKey = folderPath ? `${category}${PRIMARY_PATH_SEPARATOR}${folderPath}` : category;
    const folderName = folderNode?.name || `${category} folders`;
    const fileItems = [];
    try {
        const resolvedPath = await ResolveFilePath(file);
        if (resolvedPath) {
            const fileUrl = `file://${String(resolvedPath).replaceAll('\\', '/')}`;
            const label = getPathFileName(file);
            const actions = await GetHyperlinkMenuActions(fileUrl, label);
            for (const action of actions || []) {
                fileItems.push({
                    title: String(action.title || ''),
                    icon: Number(action.icon) || 0,
                    onSelect: () => {
                        RunHyperlinkMenuAction(fileUrl, label, String(action.action || '')).catch((err) => {
                            notifyTerminal(`Failed to execute file action: ${err}`, 'error');
                        });
                    },
                });
            }
        }
    } catch (err) {
        notifyTerminal(`Failed to load file actions: ${err}`, 'error');
    }

    openFolderTreeContextMenu(
        category,
        folderNode ? folderNode.children : categoryNodes,
        x,
        y,
        folderName,
        folderKey,
        fileItems,
    );
}

function findFolderContextRowAbove(clientY) {
    let closest = null;
    let closestBottom = -Infinity;
    const rows = elements.list.querySelectorAll('.notes-tree-folder, .notes-category-header');

    rows.forEach((row) => {
        const rect = row.getBoundingClientRect();
        if (rect.height <= 0 || rect.bottom > clientY || rect.bottom <= closestBottom) {
            return;
        }
        closest = row;
        closestBottom = rect.bottom;
    });

    return closest;
}

function openFolderContextForListRow(row, x, y) {
    if (row.classList.contains('notes-tree-folder')) {
        const folderKey = row.dataset.folderKey || '';
        const separatorIndex = folderKey.indexOf(PRIMARY_PATH_SEPARATOR);
        const category = separatorIndex === -1 ? folderKey : folderKey.slice(0, separatorIndex);
        const folderPath = separatorIndex === -1 ? '' : folderKey.slice(separatorIndex + 1);
        const folderNode = findFolderNodeByPath(getCategoryTreeNodes(category), folderPath);
        const folderName = folderNode ? folderNode.name : folderPath;
        openFolderTreeContextMenu(category, folderNode?.children || [], x, y, folderName, folderKey, [{
            title: 'Rename folder...',
            icon: CONTEXT_ICON_EDIT,
            onSelect: () => void openRenamePrompt(folderKey, true),
        }]);
        return;
    }

    const category = row.dataset.category || '';
    openFolderTreeContextMenu(category, getCategoryTreeNodes(category), x, y, `${category} folders`);
}

function setFolderExpansionState(folderKeys, expanded) {
    folderKeys.forEach((key) => {
        state.expandedFolders[key] = expanded;
    });
    saveProjectCache();
}

function getFolderCreateTarget(folderPath) {
    const normalized = String(folderPath || '').replaceAll('\\', '/');
    const separatorIndex = normalized.indexOf('/');
    if (separatorIndex === -1) {
        return { location: normalized, pathPrefix: '' };
    }

    const location = normalized.slice(0, separatorIndex);
    const relativePath = normalized.slice(separatorIndex + 1).replace(/\/+$/, '');
    return {
        location,
        pathPrefix: relativePath ? `${relativePath}/` : '',
    };
}

async function pasteAsNewFile(folderPath) {
    let payload;
    try {
        payload = decodeClipboardPayload(await GetClipboardData());
    } catch (err) {
        notifyTerminal('Failed to read clipboard.', 'error');
        console.error(err);
        return;
    }

    const isImage = payload.image !== '';
    if (!isImage && payload.text === '') {
        notifyTerminal('Clipboard is empty.', 'info');
        return;
    }

    const { location, pathPrefix } = getFolderCreateTarget(folderPath);
    openNewFilePrompt({
        location,
        name: `${pathPrefix}clipboard.${isImage ? 'png' : 'md'}`,
        pendingFile: isImage
            ? { imageBase64: payload.image }
            : { text: payload.text },
    });
}

function openFolderTreeContextMenu(category, nodes, x, y, title = 'Folder actions', folderPath = category, fileItems = []) {
    let folderKeys = collectFolderKeys(category, nodes);
    if (folderKeys.length === 0) {
        folderKeys = collectFolderKeys(category, getCategoryTreeNodes(category));
    }
    const { location, pathPrefix } = getFolderCreateTarget(folderPath);
    const menuItems = [...fileItems];
    if (menuItems.length > 0) menuItems.push({ title: '-' });

    if (folderKeys.length > 0) {
        menuItems.push(
            {
                title: 'Collapse Folders',
                icon: CONTEXT_ICON_COLLAPSE_ALL,
                onSelect: () => {
                    setFolderExpansionState(folderKeys, false);
                    renderFileList();
                },
            },
            {
                title: 'Expand Folders',
                icon: CONTEXT_ICON_EXPAND_ALL,
                onSelect: () => {
                    setFolderExpansionState(folderKeys, true);
                    renderFileList();
                },
            },
            { title: '-' },
        );
    }

    menuItems.push(
        {
            title: 'Create new file...',
            icon: CONTEXT_ICON_ADD,
            onSelect: () => openNewFilePrompt({ location, name: pathPrefix }),
        },
        {
            title: 'Paste as new file...',
            icon: CONTEXT_ICON_PASTE,
            onSelect: () => void pasteAsNewFile(folderPath),
        },
    );

    showNotesLocalMenu(menuItems, x, y, title);
}

/**
 * Show/hide tabs based on file type
 */
function updateTabVisibility(fileType) {
    updateToolsTabVisibility(fileType);
    updateGitDiffTabForCurrentFile();

    if (fileType === 'error') {
        elements.tabMeta.style.display = '';
        elements.tabViewer.style.display = 'none';
        elements.tabEditor.style.display = 'none';
        elements.tabHex.style.display = 'none';
        elements.tabJupyter.style.display = 'none';
        elements.tabSwaggerView.style.display = 'none';
        elements.tabSwaggerEdit.style.display = 'none';
        elements.tabSwaggerRun.style.display = 'none';
        elements.tabImageView.style.display = 'none';
        elements.tabCsvView.style.display = 'none';
        elements.tabCsvEdit.style.display = 'none';
        elements.tabCsvRun.style.display = 'none';
        return;
    }

    const isJson  = fileType === 'json';
    const isHtml  = fileType === 'html';
    const isCode  = fileType === 'code';
    const isBinary = fileType === 'binary';
    const isImage = fileType === 'image';
    const isCsv   = fileType === 'csv';

    if (isImage) {
        // Image files use a single View tab.
        elements.tabImageView.style.display = '';
        elements.tabHex.style.display = '';
        elements.tabMeta.style.display = '';
        elements.tabViewer.style.display = 'none';
        elements.tabEditor.style.display = 'none';
        elements.tabJupyter.style.display = 'none';
        elements.tabSwaggerView.style.display = 'none';
        elements.tabSwaggerEdit.style.display = 'none';
        elements.tabSwaggerRun.style.display = 'none';
        elements.tabCsvView.style.display = 'none';
        elements.tabCsvEdit.style.display = 'none';
        elements.tabCsvRun.style.display = 'none';
        return;
    }

    if (isCsv) {
        // CSV files use View + Edit + Run tabs.
        elements.tabCsvView.style.display = '';
        elements.tabCsvEdit.style.display = '';
        elements.tabCsvRun.style.display = '';
        elements.tabHex.style.display = '';
        elements.tabMeta.style.display = '';
        elements.tabImageView.style.display = 'none';
        elements.tabViewer.style.display = 'none';
        elements.tabEditor.style.display = 'none';
        elements.tabJupyter.style.display = 'none';
        elements.tabSwaggerView.style.display = 'none';
        elements.tabSwaggerEdit.style.display = 'none';
        elements.tabSwaggerRun.style.display = 'none';
        return;
    }

    // Hide image + csv tabs for all other types
    elements.tabImageView.style.display = 'none';
    elements.tabCsvView.style.display = 'none';
    elements.tabCsvEdit.style.display = 'none';
    elements.tabCsvRun.style.display = 'none';

    if (isCode) {
        // SVG files also retain their image preview alongside the text editor.
        elements.tabEditor.style.display = '';
        elements.tabImageView.style.display = isSvgFile(state.currentFile) ? '' : 'none';
        elements.tabHex.style.display = '';
        elements.tabMeta.style.display = '';
        elements.tabViewer.style.display = 'none';
        elements.tabJupyter.style.display = 'none';
        elements.tabSwaggerView.style.display = 'none';
        elements.tabSwaggerEdit.style.display = 'none';
        elements.tabSwaggerRun.style.display = 'none';
        return;
    }

    if (isHtml) {
        // HTML files use View + Edit tabs plus Hex + Meta.
        elements.tabViewer.style.display = '';
        elements.tabEditor.style.display = '';
        elements.tabHex.style.display = '';
        elements.tabMeta.style.display = '';
        elements.tabJupyter.style.display = 'none';
        elements.tabSwaggerView.style.display = 'none';
        elements.tabSwaggerEdit.style.display = 'none';
        elements.tabSwaggerRun.style.display = 'none';
        return;
    }

    if (isBinary) {
        // Binary files use Hex + Meta tabs.
        elements.tabEditor.style.display = 'none';
        elements.tabHex.style.display = '';
        elements.tabMeta.style.display = '';
        elements.tabViewer.style.display = 'none';
        elements.tabJupyter.style.display = 'none';
        elements.tabSwaggerView.style.display = 'none';
        elements.tabSwaggerEdit.style.display = 'none';
        elements.tabSwaggerRun.style.display = 'none';
        return;
    }

    // Markdown tabs
    elements.tabViewer.style.display = isJson ? 'none' : '';
    elements.tabEditor.style.display = isJson ? 'none' : '';
    elements.tabHex.style.display = '';
    elements.tabJupyter.style.display = isJson ? 'none' : '';
    elements.tabMeta.style.display = '';

    // JSON/YAML tabs
    elements.tabSwaggerView.style.display = isJson ? '' : 'none';
    elements.tabSwaggerEdit.style.display = isJson ? '' : 'none';
    elements.tabSwaggerRun.style.display = isJson && state.swaggerRunAvailable ? '' : 'none';
}

function getStructuredDocByteLength(text) {
    if (!text) {
        return 0;
    }
    return new TextEncoder().encode(String(text)).length;
}

function isStructViewTooLarge(text) {
    if (!(notesStructViewMaxSizeKB > 0)) {
        return false;
    }
    return getStructuredDocByteLength(text) > notesStructViewMaxSizeKB * 1024;
}

function renderStructViewTooLargeMessage() {
    const fileName = getPathFileName(state.currentFile) || 'this file';
    const markdown = [
        '# File too large',
        '',
        `Cannot display a structured view because \`${fileName}\` is greater than ${notesStructViewMaxSizeKB} KB (\`$.Notes.StructViewMaxSizeKB\`)`,
    ].join('\n');

    elements.swaggerView.innerHTML = `<div class="markdown-body notes-struct-too-large">${marked.parse(markdown)}</div>`;
}

function renderSwaggerJsonView() {
    if (!elements.swaggerView) {
        return;
    }

    if (state.swaggerViewTooLarge) {
        renderStructViewTooLargeMessage();
        state.swaggerViewCurrent = true;
        return;
    }

    attachJsonViewerEditHandler(elements.swaggerView, commitStructuredViewerEdit);
    renderJsonViewer(elements.swaggerView, state.swaggerSpec ?? (mainDocument.getValue() || '{}'));
    state.swaggerViewCurrent = true;
}

// Renders the JSON/YAML tree viewer non-blocking: paints the AI-panel lazy
// spinner first, yields to the browser so it actually appears, then runs the
// (still synchronous) render. No-op when the tab DOM is already current.
function renderSwaggerJsonViewLazy() {
    if (state.swaggerViewCurrent) {
        return;
    }

    if (!elements.swaggerView) {
        return;
    }

    if (state.swaggerViewTooLarge) {
        renderStructViewTooLargeMessage();
        state.swaggerViewCurrent = true;
        return;
    }

    const spinner = document.createElement('div');
    spinner.className = 'notes-ai-lazy-spinner notes-ai-lazy-spinner-page';
    elements.swaggerView.replaceChildren(spinner);

    // Two rAFs give the browser a chance to paint the spinner before the
    // synchronous tree render blocks the main thread.
    requestAnimationFrame(() => {
        requestAnimationFrame(() => {
            if (state.currentFileType !== 'json' || state.viewMode !== 'swagger-view') {
                return;
            }
            renderSwaggerJsonView();
        });
    });
}

function isYamlStructuredFile(fileName) {
    return /\.ya?ml$/i.test(fileName || '');
}

function isJsonStructuredFile(fileName) {
    return /\.json$/i.test(fileName || '');
}

function formatStructuredEditorJson(pretty) {
    const source = mainDocument.getValue();

    try {
        const parsed = JSON.parse(source);
        const nextText = pretty
            ? JSON.stringify(parsed, null, 2)
            : JSON.stringify(parsed);

        replaceMainDocumentText(nextText, { source: 'structured-json-format' });
    } catch {
        notifyTerminal('Cannot format invalid JSON content', 'warn');
    }
}

function stringifyStructuredDocument(value) {
    if (isYamlStructuredFile(state.currentFile)) {
        return YAML.stringify(value);
    }

    return JSON.stringify(value, null, 2);
}

function parseStructuredScalar(text) {
    if (text === '') {
        return '';
    }

    try {
        const parsed = YAML.parse(text);
        return parsed === undefined ? text : parsed;
    } catch {
        return text;
    }
}

function parseStructuredDocument(fileName, source) {
    const editor = getStructuredEditor(fileName, source);
    if (typeof editor?.parse === 'function') {
        return editor.parse(String(source ?? ''));
    }
    return parseSwaggerSpec(String(source ?? ''));
}

function getValueAtPath(root, path) {
    return path.reduce((current, segment) => {
        if (current === null || current === undefined) {
            return undefined;
        }

        return current[segment];
    }, root);
}

function setValueAtPath(root, path, value) {
    if (path.length === 0) {
        return value;
    }

    const parentPath = path.slice(0, -1);
    const parent = getValueAtPath(root, parentPath);
    if (parent === null || parent === undefined) {
        throw new Error('Unable to locate parent item for edit.');
    }

    parent[path[path.length - 1]] = value;
    return root;
}

function appendArrayItem(root, path, value) {
    const target = getValueAtPath(root, path);
    if (!Array.isArray(target)) {
        throw new Error('Unable to locate array for insert.');
    }
    target.push(value);
    return root;
}

function deleteAtPath(root, path) {
    if (path.length === 0) {
        return root;
    }

    const parentPath = path.slice(0, -1);
    const parent = getValueAtPath(root, parentPath);
    if (parent === null || parent === undefined) {
        return root;
    }

    const key = path[path.length - 1];
    if (Array.isArray(parent)) {
        const index = Number(key);
        if (Number.isInteger(index) && index >= 0 && index < parent.length) {
            parent.splice(index, 1);
        }
    } else if (typeof parent === 'object') {
        delete parent[key];
    }

    return root;
}

function renameObjectKey(root, path, nextKey) {
    if (path.length === 0) {
        throw new Error('Root key cannot be renamed.');
    }

    const parentPath = path.slice(0, -1);
    const currentKey = path[path.length - 1];
    const parent = getValueAtPath(root, parentPath);
    if (!parent || typeof parent !== 'object' || Array.isArray(parent)) {
        throw new Error('Only object properties can be renamed.');
    }

    if (nextKey === currentKey) {
        return root;
    }

    if (!nextKey) {
        throw new Error('Property name cannot be empty.');
    }

    if (Object.prototype.hasOwnProperty.call(parent, nextKey)) {
        throw new Error(`Property "${nextKey}" already exists.`);
    }

    const renamed = {};
    Object.keys(parent).forEach((key) => {
        if (key === currentKey) {
            renamed[nextKey] = parent[key];
            return;
        }

        renamed[key] = parent[key];
    });

    if (parentPath.length === 0) {
        return renamed;
    }

    setValueAtPath(root, parentPath, renamed);
    return root;
}

// validateKeyRename checks a key rename against the parsed document. It throws
// when the rename is invalid and returns false when it is a no-op (the key is
// unchanged); callers use this before performing a surgical text edit.
function validateKeyRename(root, path, nextKey) {
    if (path.length === 0) {
        throw new Error('Root key cannot be renamed.');
    }

    const parentPath = path.slice(0, -1);
    const currentKey = path[path.length - 1];
    const parent = getValueAtPath(root, parentPath);
    if (!parent || typeof parent !== 'object' || Array.isArray(parent)) {
        throw new Error('Only object properties can be renamed.');
    }

    if (String(nextKey) === String(currentKey)) {
        return false;
    }

    if (!nextKey) {
        throw new Error('Property name cannot be empty.');
    }

    if (Object.prototype.hasOwnProperty.call(parent, nextKey)) {
        throw new Error(`Property "${nextKey}" already exists.`);
    }

    return true;
}

async function commitStructuredViewerEdit({ editType, path, text }) {
    try {
        const source = state.swaggerSpec ?? parseStructuredDocument(state.currentFile, mainDocument.getValue());
        if (!source || !Array.isArray(path)) {
            return;
        }

        // Prefer a format-specific surgical editor so only the edited key/value
        // is rewritten, preserving comments and formatting (and therefore git
        // diffs). Fall back to a whole-document re-serialise for any format
        // without a registered editor.
        const originalText = mainDocument.getValue();
        const editor = getStructuredEditor(state.currentFile, originalText);
        let nextText;

        if (editType === 'key') {
            const nextKey = String(text);
            if (!validateKeyRename(source, path, nextKey)) {
                return;
            }

            nextText = editor
                ? editor.renameKey(originalText, path, nextKey)
                : stringifyStructuredDocument(renameObjectKey(source, path, nextKey));
        } else if (editType === 'value') {
            const currentValue = getValueAtPath(source, path);
            const nextValue = parseStructuredScalar(String(text));

            if (Object.is(currentValue, nextValue)) {
                return;
            }

            nextText = editor
                ? editor.setValue(originalText, path, nextValue)
                : stringifyStructuredDocument(setValueAtPath(source, path, nextValue));
        } else if (editType === 'addKey') {
            const key = String(text);
            nextText = editor
                ? editor.addKey(originalText, path, key, '')
                : stringifyStructuredDocument(setValueAtPath(source, [...path, key], ''));
        } else if (editType === 'addItem') {
            nextText = editor
                ? editor.addItem(originalText, path, '')
                : stringifyStructuredDocument(appendArrayItem(source, path, ''));
        } else if (editType === 'delete') {
            nextText = editor
                ? editor.deleteNode(originalText, path)
                : stringifyStructuredDocument(deleteAtPath(source, path));
        } else {
            return;
        }

        replaceMainDocumentText(nextText, { source: 'structured-edit' });
        state.swaggerSpec = parseStructuredDocument(state.currentFile, mainDocument.getValue());

        state.swaggerRunAvailable = hasSwaggerKey(state.swaggerSpec);
        updateTabVisibility('json');

        if (!state.swaggerRunAvailable && state.viewMode === 'swagger-run') {
            setViewMode('swagger-view');
        }

        renderSwaggerJsonView();

        if (state.swaggerRunAvailable && state.viewMode === 'swagger-run') {
            renderSwaggerUI();
        }

        setDirty(true);
        await saveFile();
    } catch (err) {
        notifyTerminal(err?.message || 'Failed to apply structured document edit', 'error');
        console.error(err);
    }
}

// Replaces only the inner YAML body of the document's frontmatter block,
// preserving the surrounding "---" fences (and any BOM/trailing spaces) exactly.
// Returns null if the document has no frontmatter block.
function replaceFrontmatterBlock(raw, newInnerYaml) {
    const text = String(raw ?? '');
    const match = text.match(FRONTMATTER_RX);
    if (!match) {
        return null;
    }

    const fullBlock = match[0];
    const inner = match[1];
    const openingEnd = fullBlock.indexOf('\n') + 1;
    const opening = fullBlock.slice(0, openingEnd);
    const closing = fullBlock.slice(openingEnd + inner.length);
    // YAML.toString() always emits a trailing newline; the captured inner body
    // never includes it, so strip one to avoid inserting a blank line before
    // the closing fence.
    const normalizedInner = String(newInnerYaml).replace(/\r?\n$/, '');

    return opening + normalizedInner + closing + text.slice(fullBlock.length);
}

// Commit handler for the Frontmatter tab's tree viewer. Frontmatter is always
// YAML, so edits are applied surgically to the frontmatter block via the YAML
// editor (preserving comments/formatting) and spliced back into the markdown
// document without touching the body.
async function commitFrontmatterEdit({ editType, path, text }) {
    try {
        if (!Array.isArray(path) || state.frontmatter == null) {
            return;
        }

        const raw = mainDocument.getValue();
        const match = raw.match(FRONTMATTER_RX);
        if (!match) {
            return;
        }

        const innerYaml = match[1];
        let newInner;

        if (editType === 'key') {
            const nextKey = String(text);
            if (!validateKeyRename(state.frontmatter, path, nextKey)) {
                return;
            }
            newInner = yamlEditor.renameKey(innerYaml, path, nextKey);
        } else if (editType === 'value') {
            const currentValue = getValueAtPath(state.frontmatter, path);
            const nextValue = parseStructuredScalar(String(text));
            if (Object.is(currentValue, nextValue)) {
                return;
            }
            newInner = yamlEditor.setValue(innerYaml, path, nextValue);
        } else if (editType === 'addKey') {
            newInner = yamlEditor.addKey(innerYaml, path, String(text), '');
        } else if (editType === 'addItem') {
            newInner = yamlEditor.addItem(innerYaml, path, '');
        } else if (editType === 'delete') {
            newInner = yamlEditor.deleteNode(innerYaml, path);
        } else {
            return;
        }

        const nextRaw = replaceFrontmatterBlock(raw, newInner);
        if (nextRaw == null) {
            return;
        }

        replaceMainDocumentText(nextRaw, { source: 'frontmatter-edit' });
        // Re-parse the frontmatter and re-render the panel (this also clears the
        // inline editor that the viewer left in place after commit).
        applyDocumentFrontmatter();
        setDirty(true);
        await saveFile();
    } catch (err) {
        notifyTerminal(err?.message || 'Failed to apply frontmatter edit', 'error');
        console.error(err);
    }
}

function safeSwaggerInfoUrl(value) {
    if (typeof value !== 'string') {
        return '';
    }

    const trimmed = value.trim();
    return /^https?:\/\//i.test(trimmed) ? trimmed : '';
}

function renderSwaggerInfoMetaValue(label, value) {
    if (!value) {
        return '';
    }

    return `
        <div class="swagger-info-meta-item">
            <span class="swagger-info-meta-label">${label}</span>
            <span class="swagger-info-meta-value">${value}</span>
        </div>
    `;
}

function renderSwaggerInfoMetadata(info) {
    if (!info || typeof info !== 'object') {
        return '';
    }

    const items = [];

    if (typeof info.summary === 'string' && info.summary.trim()) {
        items.push(renderSwaggerInfoMetaValue('Summary', escapeInfoText(info.summary.trim())));
    }

    if (typeof info.version === 'string' && info.version.trim()) {
        items.push(renderSwaggerInfoMetaValue('Version', escapeInfoText(info.version.trim())));
    }

    const termsUrl = safeSwaggerInfoUrl(info.termsOfService);
    if (termsUrl) {
        items.push(renderSwaggerInfoMetaValue(
            'Terms',
            `<a href="${escapeInfoText(termsUrl)}" target="_blank" rel="noopener noreferrer">${escapeInfoText(termsUrl)}</a>`
        ));
    }

    if (info.contact && typeof info.contact === 'object') {
        const contactParts = [];
        if (typeof info.contact.name === 'string' && info.contact.name.trim()) {
            contactParts.push(escapeInfoText(info.contact.name.trim()));
        }

        const contactUrl = safeSwaggerInfoUrl(info.contact.url);
        if (contactUrl) {
            contactParts.push(`<a href="${escapeInfoText(contactUrl)}" target="_blank" rel="noopener noreferrer">${escapeInfoText(contactUrl)}</a>`);
        }

        if (typeof info.contact.email === 'string' && info.contact.email.trim()) {
            const email = info.contact.email.trim();
            contactParts.push(`<a href="mailto:${encodeURIComponent(email)}">${escapeInfoText(email)}</a>`);
        }

        if (contactParts.length > 0) {
            items.push(renderSwaggerInfoMetaValue('Contact', contactParts.join(' · ')));
        }
    }

    if (info.license && typeof info.license === 'object') {
        const licenseName = typeof info.license.name === 'string' && info.license.name.trim()
            ? info.license.name.trim()
            : '';
        const licenseUrl = safeSwaggerInfoUrl(info.license.url);

        if (licenseName || licenseUrl) {
            const licenseValue = licenseUrl
                ? `<a href="${escapeInfoText(licenseUrl)}" target="_blank" rel="noopener noreferrer">${escapeInfoText(licenseName || licenseUrl)}</a>`
                : escapeInfoText(licenseName);
            items.push(renderSwaggerInfoMetaValue('License', licenseValue));
        }
    }

    if (items.length === 0) {
        return '';
    }

    return `<div class="swagger-info-meta">${items.join('')}</div>`;
}

function updateSwaggerLayoutMode() {
    if (!elements.swaggerRunWrap) {
        return;
    }

    const width = elements.swaggerRunWrap.getBoundingClientRect().width;
    if (width <= 0) {
        return;
    }

    const compact = width <= 900;
    elements.swaggerRunWrap.setAttribute('data-layout', compact ? 'compact' : 'wide');
}

/**
 * Render the Swagger/OpenAPI UI in the Run tab
 */
function renderSwaggerUI() {
    if (!state.swaggerSpec || !elements.swaggerEndpoints || !elements.swaggerRequestBuilder || !elements.swaggerResponse) {
        return;
    }

    const swaggerInfoEl = focusedView().parts.swaggerInfo;
    if (swaggerInfoEl) {
        const info = state.swaggerSpec.info || {};
        const title = typeof info.title === 'string' && info.title.trim() ? info.title.trim() : '';
        const description = typeof info.description === 'string' && info.description.trim() ? info.description.trim() : '';
        const metadata = renderSwaggerInfoMetadata(info);
        if (title || description || metadata) {
            swaggerInfoEl.innerHTML =
                (title ? `<h1 class="swagger-info-title">${escapeInfoText(title)}</h1>` : '') +
                (description ? `<div class="swagger-info-description markdown-body">${marked.parse(description)}</div>` : '') +
                metadata;
            processMarkdownContainer(swaggerInfoEl);
            swaggerInfoEl.style.display = '';
        } else {
            swaggerInfoEl.innerHTML = '';
            swaggerInfoEl.style.display = 'none';
        }
    }

    const currentFilterInput = elements.swaggerEndpoints.querySelector('#notes-swagger-endpoint-filter');
    const restoreFilterFocus = document.activeElement === currentFilterInput;
    const filterSelectionStart = restoreFilterFocus ? currentFilterInput.selectionStart : null;
    const filterSelectionEnd = restoreFilterFocus ? currentFilterInput.selectionEnd : null;
    
    // If no endpoint selected, select the first one
    if (!state.swaggerSelectedEndpoint) {
        const paths = extractPaths(state.swaggerSpec);
        if (paths.length > 0 && paths[0].methods.length > 0) {
            state.swaggerSelectedEndpoint = {
                path: paths[0].path,
                method: paths[0].methods[0].method
            };
        }
    }

    const endpointListHtml = generateEndpointListHTML(
        state.swaggerSpec,
        state.swaggerSelectedEndpoint,
        state.swaggerEndpointFilter
    );

    elements.swaggerEndpoints.innerHTML = `
        <input
            id="notes-swagger-endpoint-filter"
            class="swagger-endpoint-filter"
            type="text"
            placeholder="Filter operations..."
            autocomplete="off"
            autocorrect="off"
            autocapitalize="off"
            spellcheck="false"
            value="${state.swaggerEndpointFilter.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/\"/g, '&quot;')}"
        />
        ${endpointListHtml}
    `;
    
    disableNativeTextAssistanceIn(elements.swaggerEndpoints);

    // Render request builder and response
    elements.swaggerRequestBuilder.innerHTML = generateRequestBuilderHTML(state.swaggerSpec, state.swaggerSelectedEndpoint);
    disableNativeTextAssistanceIn(elements.swaggerRequestBuilder);
    elements.swaggerResponse.innerHTML = generateResponseHTML(state.swaggerSpec, state.swaggerSelectedEndpoint);

    // Render parameter descriptions using the same markdown pipeline as preview/info.
    elements.swaggerRequestBuilder.querySelectorAll('.swagger-param-description[data-markdown]').forEach((descEl) => {
        const markdown = descEl.getAttribute('data-markdown') || '';
        descEl.innerHTML = marked.parse(markdown);
        processMarkdownContainer(descEl);
    });

    setupSwaggerMethodSelector();
    setupSwaggerHeaderDropdowns();
    
    // Add tab switching logic for nested tabs
    setupSwaggerTabSwitching();
    setupSwaggerEndpointSelection();
    setupSwaggerSendButton();

    if (restoreFilterFocus) {
        const nextFilterInput = elements.swaggerEndpoints.querySelector('#notes-swagger-endpoint-filter');
        if (nextFilterInput) {
            nextFilterInput.focus();
            const start = typeof filterSelectionStart === 'number' ? filterSelectionStart : nextFilterInput.value.length;
            const end = typeof filterSelectionEnd === 'number' ? filterSelectionEnd : start;
            nextFilterInput.setSelectionRange(start, end);
        }
    }
}

function getSwaggerMethodsForPath(path) {
    if (!path || !state.swaggerSpec || !state.swaggerSpec.paths || !state.swaggerSpec.paths[path]) {
        return [];
    }

    const pathItem = state.swaggerSpec.paths[path];
    const methodOrder = ['get', 'post', 'put', 'delete', 'patch', 'head', 'options'];
    const methods = [];

    for (const method of methodOrder) {
        if (pathItem && pathItem[method]) {
            methods.push(method.toUpperCase());
        }
    }

    const currentMethod = state.swaggerSelectedEndpoint && state.swaggerSelectedEndpoint.method
        ? state.swaggerSelectedEndpoint.method.toUpperCase()
        : '';
    if (currentMethod && !methods.includes(currentMethod)) {
        methods.unshift(currentMethod);
    }

    return methods;
}

function setupSwaggerMethodSelector() {
    const methodButton = elements.swaggerRequestBuilder.querySelector('.swagger-method-selector');
    if (!methodButton || !state.swaggerSelectedEndpoint || !state.swaggerSelectedEndpoint.path) {
        return;
    }

    methodButton.textContent = state.swaggerSelectedEndpoint.method;
    methodButton.addEventListener('click', () => {
        const methods = getSwaggerMethodsForPath(state.swaggerSelectedEndpoint.path);
        if (methods.length === 0) {
            return;
        }

        const rect = methodButton.getBoundingClientRect();
        showNotesLocalMenu(
            methods.map((method) => ({
                title: method,
                icon: method === String(state.swaggerSelectedEndpoint.method || '').toUpperCase() ? 0xf00c : 0,
                onSelect: () => {
                    state.swaggerSelectedEndpoint = {
                        path: state.swaggerSelectedEndpoint.path,
                        method,
                    };
                    renderSwaggerUI();
                },
            })),
            rect.left,
            rect.bottom,
            'Select method',
        );
    });
}

function setupSwaggerEndpointSelection() {
    const filterInput = elements.swaggerEndpoints.querySelector('#notes-swagger-endpoint-filter');
    if (filterInput) {
        filterInput.addEventListener('input', (event) => {
            state.swaggerEndpointFilter = event.target.value || '';
            renderSwaggerUI();
        });
    }

    const endpointButtons = elements.swaggerEndpoints.querySelectorAll('.swagger-endpoint-item');
    endpointButtons.forEach((button) => {
        button.addEventListener('click', () => {
            const path = button.getAttribute('data-path') || '';
            const method = button.getAttribute('data-method') || '';
            if (!path || !method) {
                return;
            }

            state.swaggerSelectedEndpoint = { path, method };
            renderSwaggerUI();
        });
    });
}

/**
 * Wire up the Send button to execute the current endpoint via the Go backend.
 */
function setupSwaggerSendButton() {
    const sendBtn = elements.swaggerRequestBuilder.querySelector('.swagger-send-btn');
    if (!sendBtn) {
        return;
    }

    sendBtn.addEventListener('click', () => {
        sendSwaggerRequest();
    });
}

async function sendSwaggerRequest() {
    if (!state.swaggerSpec || !state.swaggerSelectedEndpoint) {
        return;
    }

    const sendBtn = elements.swaggerRequestBuilder.querySelector('.swagger-send-btn');
    if (sendBtn) {
        sendBtn.disabled = true;
        sendBtn.dataset.sending = 'true';
        sendBtn.textContent = 'Sending…';
    }

    // Collect headers from the displayed header items
    // Values may be <input>, <button> (interactive) or <span> (static)
    const headers = {};
    elements.swaggerRequestBuilder.querySelectorAll('.swagger-header-item').forEach((item) => {
        const name = item.querySelector('.swagger-header-name')?.textContent?.trim();
        const valueEl = item.querySelector('.swagger-header-input, .swagger-header-value');
        if (!name || !valueEl) return;
        const value = valueEl instanceof HTMLInputElement
            ? valueEl.value.trim()
            : (valueEl.textContent?.trim() || '');
        if (name && value) {
            headers[name] = value;
        }
    });

    // Collect body from the editable textarea
    const bodyTextarea = elements.swaggerRequestBuilder.querySelector('.swagger-body-editor');
    const body = bodyTextarea ? bodyTextarea.value : '';

    // Collect parameter values from the form inputs
    const parameters = {};
    elements.swaggerRequestBuilder.querySelectorAll('.swagger-param-input').forEach((input) => {
        const paramName = input.dataset.paramName;
        const paramIn = input.dataset.paramIn;
        const value = input.value?.trim();
        if (paramName && value) {
            parameters[paramName] = value;
        }
    });

    const url = buildRequestUrl(state.swaggerSpec, state.swaggerSelectedEndpoint, parameters);
    const doc = activeNotesDocument;

    try {
        const response = await SwaggerRequest({
            method: state.swaggerSelectedEndpoint.method,
            url,
            headers,
            body,
        });
        if (doc !== activeNotesDocument) {
            return;
        }

        elements.swaggerResponse.innerHTML = generateLiveResponseHTML(response);
        setupSwaggerResponseTabs();
    } catch (err) {
        if (doc !== activeNotesDocument) {
            return;
        }
        elements.swaggerResponse.innerHTML = generateLiveResponseHTML({
            error: String(err?.message || err),
        });
    } finally {
        if (sendBtn) {
            sendBtn.disabled = false;
            sendBtn.dataset.sending = 'false';
            sendBtn.textContent = 'Send';
        }
    }
}

function setupSwaggerHeaderDropdowns() {
    if (!elements.swaggerRequestBuilder) return;

    elements.swaggerRequestBuilder.querySelectorAll('.swagger-header-dropdown').forEach((btn) => {
        btn.addEventListener('click', () => {
            const headerName = btn.dataset.headerName;
            const options = JSON.parse(btn.dataset.headerOptions || '[]');
            const input = btn.closest('.swagger-header-value-wrap')?.querySelector('.swagger-header-input');
            const currentValue = input?.value?.trim() || '';

            if (!options.length) return;

            const rect = btn.getBoundingClientRect();
            const menuItems = options.map((opt) => ({
                title: opt,
                icon: opt === currentValue ? 0xf00c : 0,
                onSelect: () => {
                    if (input) {
                        input.value = opt;
                    }
                },
            }));

            showNotesLocalMenu(menuItems, rect.left, rect.bottom, `Select ${headerName || 'header'} value`);
        });
    });
}

function setupSwaggerResponseTabs() {
    const responseTabs = elements.swaggerResponse.querySelectorAll('.swagger-response-tab');
    const responsePanels = elements.swaggerResponse.querySelectorAll('.swagger-response-panel');

    responseTabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const panelName = tab.getAttribute('data-tab');
            responsePanels.forEach(panel => panel.classList.remove('swagger-response-panel-active'));
            const selectedPanel = elements.swaggerResponse.querySelector(`.swagger-response-panel[data-panel="${panelName}"]`);
            if (selectedPanel) selectedPanel.classList.add('swagger-response-panel-active');
            responseTabs.forEach(t => t.setAttribute('aria-selected', 'false'));
            tab.setAttribute('aria-selected', 'true');
        });
    });
}

/**
 * Setup event listeners for nested tabs in swagger UI
 */
function setupSwaggerTabSwitching() {
    // Request tabs
    const requestTabs = elements.swaggerRequestBuilder.querySelectorAll('.swagger-request-tab');
    const requestPanels = elements.swaggerRequestBuilder.querySelectorAll('.swagger-request-panel');
    
    requestTabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const panelName = tab.getAttribute('data-tab');
            
            // Hide all panels
            requestPanels.forEach(panel => {
                panel.classList.remove('swagger-request-panel-active');
                panel.setAttribute('data-panel', panel.getAttribute('data-panel'));
            });
            
            // Show selected panel
            const selectedPanel = elements.swaggerRequestBuilder.querySelector(`.swagger-request-panel[data-panel="${panelName}"]`);
            if (selectedPanel) {
                selectedPanel.classList.add('swagger-request-panel-active');
            }
            
            // Update tab selection
            requestTabs.forEach(t => t.setAttribute('aria-selected', 'false'));
            tab.setAttribute('aria-selected', 'true');
        });
    });
    
    // Response tabs
    const responseTabs = elements.swaggerResponse.querySelectorAll('.swagger-response-tab');
    const responsePanels = elements.swaggerResponse.querySelectorAll('.swagger-response-panel');
    
    responseTabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const panelName = tab.getAttribute('data-tab');
            
            // Hide all panels
            responsePanels.forEach(panel => {
                panel.classList.remove('swagger-response-panel-active');
            });
            
            // Show selected panel
            const selectedPanel = elements.swaggerResponse.querySelector(`.swagger-response-panel[data-panel="${panelName}"]`);
            if (selectedPanel) {
                selectedPanel.classList.add('swagger-response-panel-active');
            }
            
            // Update tab selection
            responseTabs.forEach(t => t.setAttribute('aria-selected', 'false'));
            tab.setAttribute('aria-selected', 'true');
        });
    });
}

let notesLoadSeq = 0;

// Puts loaded text in the active document. Unchanged text is left alone, so a
// document reopened from a background tab keeps its undo history.
function loadDocumentText(text) {
    if (mainDocument.getValue() !== normaliseLineEndings(text)) {
        setMainEditorValue(text);
    }
}

async function loadFile(file, options = {}) {
    if (!file) {
        return;
    }

    if (activeGitDiff && !await closeNotesGitDiff()) {
        return;
    }

    // A newer load supersedes this one, and once this load has activated its
    // document it stops if that document is no longer the active one.
    const loadSeq = ++notesLoadSeq;
    let loadedDocument = null;
    const isStale = () => loadSeq !== notesLoadSeq
        || (loadedDocument !== null && loadedDocument !== activeNotesDocument);

    rememberShownDocumentState();

    const previousDocument = activeNotesDocument;
    if (state.currentFile && state.currentFile !== file) {
        saveDocumentCache();
        // Background tabs are kept saved, so save pending edits now. A
        // conflicted or deleted document keeps its edits until the user
        // chooses.
        if (state.dirty && savesNotesDocumentAutomatically(previousDocument)) {
            clearTimeout(state.autosaveTimer);
            state.autosaveTimer = null;
            await saveFile();
            if (isStale()) {
                return;
            }
        }
        flushDocumentChanges(previousDocument);
    }

    const skipDocumentCacheRestore = Boolean(options?.skipDocumentCacheRestore);

    state.suspendDocumentCacheSave = true;

    if (state.currentFile && state.currentFile !== file) {
        // The previous document's LSP and typos registrations are released
        // when it closes, on switching below.
        resetLspViewState();
        lspSpellCheckExclusions.symbols = [];
        lspSpellCheckExclusions.tokens = [];
        lspSpellCheckExclusions.keywords = [];
        notesSpellCheckHandle?.setExclusions([]);
        // Clear rendered diagnostics immediately while the new file loads.
        clearVisibleLspDiagnostics();
    }

    const fileName = file ? getPathFileName(file) : 'json file';
    let stickyId = null;

    try {
        clearHexSource();

        // Capture the current project context to prevent autosave issues if user switches projects
        const project = await GetCurrentProject();
        if (isStale()) {
            return;
        }
        // The view's own memory of this document takes priority over the
        // persisted cache, which records the mode last used in any view.
        const rememberedView = skipDocumentCacheRestore
            ? null
            : focusedView().recallDocument(notesDocumentKey(file, project));
        const documentCachePromise = (skipDocumentCacheRestore
            ? Promise.resolve(null)
            : GetDocumentCache(file).catch((error) => {
                console.error('Failed to load document cache:', error);
                return null;
            })).then((cache) => (rememberedView?.viewMode
            ? { ...cache, DocumentTab: rememberedView.viewMode }
            : cache));

        const loadingJsonByExtension = isStructuredDataFile(file);
        const loadingMarkdown = isMarkdownNotesFile(file);
        const loadingHtml     = isHtmlViewFile(file);
        const loadingSvg      = isSvgFile(file);
        const loadingImage    = isImageFile(file) && !loadingSvg;
        const loadingCsv      = isCsvFile(file);
        stickyId = loadingJsonByExtension ? Date.now() : null;

        if (!loadingMarkdown) {
            // Keep wrap-mode effects from leaking into non-markdown editors.
            setEditorWrapMode(false);
        }

        if (loadingImage) {
            const documentCache = await documentCachePromise;
            if (isStale()) {
                return;
            }
            loadedDocument = activateNotesDocument(file, { project });
            emitCurrentFileName();
            await refreshFileMetaMarkdown(file);
            if (isStale()) {
                return;
            }
            state.currentFileType = 'image';
            setCodeEditorMode(false);
            elements.editorShell.dataset.fileType = 'image';
            state.swaggerSpec = null;
            state.swaggerRunAvailable = false;
            updateTabVisibility('image');
            applyDocumentCache(documentCache, 'image-view');

            await configureImagePreview(file, fileName);
            if (isStale()) {
                return;
            }

            state.suspendDocumentCacheSave = false;
            saveDocumentCache();
            setDirty(false);
            renderFileList();
            showDocumentFindState();
            return;
        }

        /*if (loadingJson) {
            openStickyProgress(stickyId, `Loading ${fileName}… reading file`);
        }*/

        const result = await GetFile(file);
        const documentCache = await documentCachePromise;
        if (isStale()) {
            return;
        }

        const reopenedDocument = notesDocuments.get(file, project);
        // A background tab whose save failed keeps its unsaved edits, as does
        // the active document when reloaded after a rename.
        const keepsUnsavedEdits = Boolean(reopenedDocument?.dirty)
            && (reopenedDocument !== previousDocument || Boolean(options?.keepUnsavedEdits));
        loadedDocument = activateNotesDocument(file, { project });
        const fileUri = await resolveNotesFileUri(file);
        if (isStale()) {
            return;
        }
        state.currentFileUri = fileUri;
        clearCurrentFileLspDiagnosticsCache();
        emitCurrentFileName();
        await refreshFileMetaMarkdown(file);
        if (isStale()) {
            return;
        }

        if (result.error !== '') {
            if (stickyId) {
                closeStickyProgress(stickyId, result.error, 'warn');
            } else {
                notifyTerminal(result.error, 'warn');
            }
            updateTabVisibility('error');
            setViewMode('meta');
            setDirty(false);
            renderFileList();
            return;
        }

        const isBinaryFile = Boolean(result.binary ?? result.text);
        const doc = keepsUnsavedEdits && !isBinaryFile ? mainDocument.getValue() : result.contents;
        // Edits that match the file on disk are no longer unsaved.
        const diskMatchesEdits = keepsUnsavedEdits && !isBinaryFile
            && normaliseLineEndings(result.contents) === doc;
        if (diskMatchesEdits) {
            loadedDocument.conflicted = false;
        }

        if (isBinaryFile) {
            state.currentFileType = 'binary';
            setCodeEditorMode(false);
            elements.editorShell.dataset.fileType = 'binary';
            state.swaggerSpec = null;
            state.swaggerRunAvailable = false;

            updateTabVisibility('binary');
            setHexSource(file, 'base64', doc || '', {
                fontSize: result.fontSize,
                adjustCellHeight: result.adjustCellHeight,
            });
            applyDocumentCache(documentCache, 'hex');

            if (stickyId) {
                closeStickyProgress(stickyId);
            }

            setDirty(false);
            focusedView().shownDocumentKey = activeNotesDocument.key;
            renderFileList();

            showDocumentFindState();
            return;
        }

        const loadingJson = isStructuredDataFile(file, doc);

        // Keep hex source data available, but only render when hex tab is opened.
        setHexSource(file, 'text', doc || '', {
            fontSize: result.fontSize,
            adjustCellHeight: result.adjustCellHeight,
        });
        
        // Detect file type
        if (loadingJson) {
            state.currentFileType = 'json';
            setCodeEditorMode(true);
            elements.editorShell.dataset.fileType = 'json';
            state.swaggerViewTooLarge = isStructViewTooLarge(doc);
            updateStickyProgress(stickyId, `Loading ${fileName}… parsing json`);
            await yieldToUI();
            if (isStale()) {
                closeStickyProgress(stickyId);
                return;
            }
            state.swaggerSpec = parseStructuredDocument(file, doc);
            state.swaggerRunAvailable = hasSwaggerKey(state.swaggerSpec);

            if (!state.swaggerSpec) {
                closeStickyProgress(stickyId, `Failed to parse ${fileName}`, 'warn');
            }

            state.swaggerSelectedEndpoint = null;
            state.swaggerEndpointFilter = '';
            
            // Update UI for JSON / swagger-capable JSON
            updateTabVisibility('json');
            
            // Set editor content (use regular editor with line numbers for JSON/YAML)
            loadDocumentText(doc || '');
            refreshEditorLanguage(file, doc || '');
            applyDocumentCache(documentCache, 'swagger-edit');

            // Render JSON tree view
            updateStickyProgress(stickyId, `Loading ${fileName}… rendering viewer`);
            await yieldToUI();
            if (isStale()) {
                closeStickyProgress(stickyId);
                return;
            }
            // Defer JSON/YAML tree render until the View tab is clicked; loadFile no longer eagerly renders it.
            state.swaggerViewCurrent = false;
            
            // Render swagger UI only for JSON documents with a top-level swagger key
            if (state.swaggerRunAvailable) {
                updateStickyProgress(stickyId, `Loading ${fileName}… rendering run view`);
                await yieldToUI();
                if (isStale()) {
                    closeStickyProgress(stickyId);
                    return;
                }
                renderSwaggerUI();
            } else {
                elements.swaggerResponse.innerHTML = '';
                elements.swaggerRequestBuilder.innerHTML = '';
                elements.swaggerEndpoints.innerHTML = '';
            }
            if (state.viewMode === 'swagger-view') {
                renderSwaggerJsonViewLazy();
            }
            closeStickyProgress(stickyId);
        } else if (loadingMarkdown) {
            state.currentFileType = 'markdown';
            setEditorWrapMode(true);  // Word wrap on by default for markdown; cached per-doc preference overrides later
            //setCodeEditorMode(false);
            setCodeEditorMode(true); // TODO: maybe support toggling?
            elements.editorShell.dataset.fileType = 'markdown';
            state.swaggerSpec = null;
            state.swaggerRunAvailable = false;

            // Update UI for markdown
            updateTabVisibility('markdown');

            // Set editor content
            loadDocumentText(doc || '');
            refreshEditorLanguage(file, doc || '');
            applyDocumentCache(documentCache, 'viewer');

            // Render markdown views
            await renderMarkdown();
            if (isStale()) {
                return;
            }
            await renderJupyterView();
            if (isStale()) {
                return;
            }
        } else if (loadingHtml) {
            state.currentFileType = 'html';
            setCodeEditorMode(true);
            elements.editorShell.dataset.fileType = 'html';
            state.swaggerSpec = null;
            state.swaggerRunAvailable = false;

            // Update UI for HTML
            updateTabVisibility('html');

            // Set editor content
            loadDocumentText(doc || '');
            refreshEditorLanguage(file, doc || '');
            applyDocumentCache(documentCache, 'viewer');

            // Selecting the cached/default view mode renders the sandboxed preview.
        } else if (loadingCsv) {
            state.currentFileType = 'csv';
            setCodeEditorMode(false);
            elements.editorShell.dataset.fileType = 'csv';
            state.swaggerSpec = null;
            state.swaggerRunAvailable = false;

            // Update UI for CSV
            updateTabVisibility('csv');

            // Set editor content (raw text for Edit tab)
            loadDocumentText(doc || '');
            applyDocumentCache(documentCache, 'csv-view');
        } else {
            state.currentFileType = 'code';
            setCodeEditorMode(true);
            elements.editorShell.dataset.fileType = 'code';
            state.swaggerSpec = null;
            state.swaggerRunAvailable = false;
            
            // Update UI for code
            updateTabVisibility('code');
            
            // Set editor content
            loadDocumentText(doc || '');
            refreshEditorLanguage(file, doc || '');
            if (loadingSvg) {
                await configureImagePreview(file, fileName);
                if (isStale()) {
                    return;
                }
            }
            applyDocumentCache(documentCache, loadingSvg ? 'image-view' : 'editor');
        }
        
        setDirty(keepsUnsavedEdits && !diskMatchesEdits);
        focusedView().shownDocumentKey = activeNotesDocument.key;
        focusedView().editor?.restoreViewState(rememberedView?.editorState);
        renderFileList();

        // JSON/YAML tree view render is deferred until the View tab is clicked (see swaggerViewCurrent).

        showDocumentFindState();

        await refreshMainReservedWords(file);
        if (isStale()) {
            return;
        }
        if (isCurrentFileLspEligible()) {
            await openCurrentLspDocument(mainDocument.getValue());
            if (isStale()) {
                return;
            }
        }

        // Spellcheck via typos-lsp for eligible files (falls back to aspell).
        await openCurrentTyposDocument(mainDocument.getValue());
        if (isStale()) {
            return;
        }

        notesSpellCheckHandle?.check();
        state.suspendDocumentCacheSave = false;
        saveDocumentCache();

        saveProjectCache();
    } catch (err) {
        state.suspendDocumentCacheSave = false;
        if (stickyId) {
            closeStickyProgress(stickyId, `Failed to load ${getPathFileName(file)}`, 'error');
        }
        notifyTerminal(`Failed to load ${file}`, 'error');
        console.error(err);
    }
}

async function configureImagePreview(file, fileName, refresh = false) {
    // ResolveFilePath expands $NOTES/$PROJECT/etc variables the same way
    // GetFile does. The asset handler streams files to the webview.
    const doc = activeNotesDocument;
    const resolvedPath = await ResolveFilePath(file);
    if (doc !== activeNotesDocument) {
        return;
    }
    const previewPath = resolvedPath || file;
    const assetURL = notesAssetURL(previewPath);
    state.currentFileUri = filePathToUri(previewPath);
    elements.imageViewImg.onerror = () => {
        notifyTerminal(`Failed to load image: ${fileName}`, 'error');
    };
    elements.imageViewImg.src = refresh
        ? `${assetURL}${assetURL.includes('?') ? '&' : '?'}v=${Date.now()}`
        : assetURL;
    elements.imageViewImg.dataset.originalFilename = fileName;
    enableFullscreenImages(elements.imageViewWrap);
    enableImageContextMenus(elements.imageViewWrap);
}

// Saves a document, by default the active one. The document is captured when
// the save starts, and only updated afterwards if it is still open.
async function saveFile(doc = activeNotesDocument) {
    if (!doc.path) {
        setStatus('Select a note before saving', true);
        return false;
    }
    if (doc.conflicted) {
        setStatus(`${doc.path} changed on disk: keep your edits or load the disk version first`, true);
        return false;
    }

    const file = doc.path;
    try {
        const content = doc.buffer.getValue();

        // Use the saved project context to prevent overwrites if user switched projects
        await SaveFile(file, content, doc.project || '');
        maybeRefreshProjectFindAfterSave();
        if (!isNotesDocumentOpen(doc)) {
            return true;
        }
        if (doc.deleted) {
            doc.deleted = false;
            renderAllNotesTabs();
        }
        if (doc.lspOpenFile === file) {
            await NotesLspSaveDocument(file);
        }
        if (doc !== activeNotesDocument) {
            if (doc.buffer.getValue() === content) {
                doc.dirty = false;
                renderAllNotesTabs();
            }
            return true;
        }
        if (doc.fileType === 'markdown' && state.viewMode === 'editor') {
            applyDocumentFrontmatter();
            refreshToolsToC();
        }
        // Edits made while saving are still unsaved; their autosave is pending.
        if (doc.buffer.getValue() === content) {
            setDirty(false);
        }
        return true;
    } catch (err) {
        notifyTerminal(`Failed to save ${file}`, 'error');
        console.error(err);
        return false;
    }
}

// cmd/ctrl+s: format the active editor (main editor or focused Jupyter code
// block) through the shared format routine, then save the document to disk.
async function saveAndFormat() {
    if (!state.currentFile) {
        setStatus('Select a note before saving', true);
        return;
    }
    try {
        if (state.lspActiveBlockId && state.jupyterCodeBlocks[state.lspActiveBlockId]) {
            await formatJupyterBlock(state.lspActiveBlockId, { notifyOnError: true });
        } else {
            await formatMainEditor({ notifyOnError: true });
        }
    } catch (err) {
        console.error('format on save failed:', err);
    }
    // Saving explicitly keeps the edits over a version changed on disk.
    activeNotesDocument.conflicted = false;
    renderAllNotesTabs();
    await saveFile();
}

function openDeletePrompt(file) {
    state.deletingFile = file;
    state.deleteConfirmAction = null;
    const fileName = getPathFileName(file);
    if (elements.deleteModalTitle) {
        setMarkdownModalHeading(elements.deleteModalTitle, 'Delete note');
    }
    elements.deleteConfirm.textContent = 'Delete';
    elements.deleteModalBody.textContent = `Are you sure you want to delete "${fileName}"?`;
    elements.deleteModal.dataset.open = 'true';
    elements.deleteModal.setAttribute('aria-hidden', 'false');
    setTimeout(() => {
        elements.deleteConfirm.focus();
    }, 0);
}

// Opens the shared delete-confirmation modal for an arbitrary destructive
// action (not just file deletion). `onConfirm` runs when the user confirms.
function openConfirmPrompt({ title = 'Confirm', body = 'Are you sure?', confirmLabel = 'Delete', onConfirm = null }) {
    state.deletingFile = null;
    state.deleteConfirmAction = typeof onConfirm === 'function' ? onConfirm : null;
    if (elements.deleteModalTitle) {
        setMarkdownModalHeading(elements.deleteModalTitle, title);
    }
    elements.deleteConfirm.textContent = confirmLabel;
    elements.deleteModalBody.textContent = body;
    elements.deleteModal.dataset.open = 'true';
    elements.deleteModal.setAttribute('aria-hidden', 'false');
    setTimeout(() => {
        elements.deleteConfirm.focus();
    }, 0);
}

function closeDeletePrompt() {
    elements.deleteModal.dataset.open = 'false';
    elements.deleteModal.setAttribute('aria-hidden', 'true');
    state.deletingFile = null;
    state.deleteConfirmAction = null;
}

async function confirmDelete() {
    // A generic confirm action takes precedence over file deletion (e.g. the
    // JSON/YAML tree's "Delete key/item" reuses this same modal).
    if (state.deleteConfirmAction) {
        const action = state.deleteConfirmAction;
        state.deleteConfirmAction = null;
        closeDeletePrompt();
        try {
            await action();
        } catch (err) {
            notifyTerminal(err?.message || 'Action failed', 'error');
            console.error(err);
        }
        return;
    }

    if (!state.deletingFile) {
        setStatus('Select a note to delete', true);
        return;
    }

    const fileToDelete = state.deletingFile;
    const fileName = getPathFileName(fileToDelete);
    const project = await GetCurrentProject();
    const deletedKey = notesDocumentKey(fileToDelete, project);
    const openDocuments = notesDocuments.list().filter((doc) => doc.key === deletedKey);
    // The active document closes here unless it keeps unsaved edits; other
    // open documents of the file are handled like an external delete.
    const deletingCurrentFile = activeNotesDocument.key === deletedKey && !state.dirty;
    const fileUri = deletingCurrentFile
        ? state.currentFileUri
        : await resolveNotesFileUri(fileToDelete);

    state.mutatingFile = fileToDelete;
    try {
        if (deletingCurrentFile) {
            await closeOpenLspDocument();
            await closeCurrentTyposDocument();
        }
        await DeleteFile(fileToDelete);
        if (fileUri) {
            lspDiagnosticsStore.delete(fileUri);
        }
        for (const doc of openDocuments) {
            if (!deletingCurrentFile || doc !== activeNotesDocument) {
                await handleNotesDocumentDeleted(doc);
            }
        }
        const tabIndex = focusedView().tabs.indexOf(deletedKey);
        if (deletingCurrentFile) {
            for (const view of notesViewList().filter((candidate) => candidate !== focusedView() && candidate.tabs.includes(deletedKey))) {
                await closeNotesTab(view, deletedKey, { discard: true });
            }
            closeActiveNotesDocument();
            state.lspHoverLastKey = '';
            emitCurrentFileName();
            setMainEditorValue('');
            elements.swaggerView.innerHTML = '';
            clearVisibleLspDiagnostics();
            hideLspHoverTooltip();
            await renderMarkdown();
            setDirty(false);
        }
        closeDeletePrompt();
        await refreshFiles({ skipHistoryRestore: true });
        if (deletingCurrentFile) {
            // The next open tab takes over; with none, the surviving history
            // position (ADR 0064).
            const tabs = focusedView().tabs;
            const neighbour = findNotesDocument(tabs[Math.min(tabIndex, tabs.length - 1)]);
            const entry = neighbour ? null : await NotesHistoryCurrentPosition();
            if (neighbour) {
                await loadFile(neighbour.path);
            } else if (entry?.file && entry.file !== fileToDelete) {
                await loadFile(entry.file);
                setViewMode(editViewModeForCurrentFileType());
                await ensureMonacoVisibleAndLaidOut();
                jumpEditorToLine(entry.line);
            }
        }
        setStatus(`Deleted ${fileName}`, false);
    } catch (err) {
        notifyTerminal(`Failed to delete ${fileName}`, 'error');
        console.error(err);
    } finally {
        state.mutatingFile = '';
    }
}

function openFindBar() {
    if (elements.toolsPanel.dataset.collapsed === 'true') {
        setToolsPanelCollapsed(false);
    }

    setToolsTab('find');
    setTimeout(() => {
        if (elements.findInput?.disabled) {
            return;
        }
        elements.findInput.focus();
        elements.findInput.select();
    }, 0);
}

function closeFindBar() {
    clearHighlights();
    state.findMatches = [];
    state.findCurrentIndex = -1;
    state.findQuery = '';
    elements.findInput.value = '';
    elements.findCounter.textContent = '';
    if (elements.replaceInput) {
        elements.replaceInput.value = '';
    }
    updateFindInputClearButtonVisibility();
    updateReplaceInputClearButtonVisibility();

    if (state.findFilesTimer) {
        clearTimeout(state.findFilesTimer);
        state.findFilesTimer = null;
    }
    cleanupProjectFindStreamListeners();
    resetProjectFindPaging({ resetListScroll: false });
    state.findFilesQuery = '';
    state.findFilesMode = 'grep';
    state.findFilesSource = '';
    state.findFilesReferenceSymbol = '';
    state.findFilesResults = [];
    state.findFilesLastExecutedSignature = '';
    state.findFilesBusy = false;
    state.findFilesError = '';
    state.findFilesSelectedKey = '';
    if (elements.findFilesInput) {
        elements.findFilesInput.value = '';
    }
    updateFindFilesClearButtonVisibility();
    renderProjectFindResults();
    renderFileList();

}

function clearProjectFindResults({ keepInputFocus = true } = {}) {
    if (state.findFilesTimer) {
        clearTimeout(state.findFilesTimer);
        state.findFilesTimer = null;
    }

    cleanupProjectFindStreamListeners();
    resetProjectFindPaging({ resetListScroll: false });

    state.findFilesQuery = '';
    state.findFilesMode = 'grep';
    state.findFilesSource = '';
    state.findFilesReferenceSymbol = '';
    state.findFilesResults = [];
    state.findFilesLastExecutedSignature = '';
    state.findFilesBusy = false;
    state.findFilesError = '';
    state.findFilesSelectedKey = '';

    if (elements.findFilesInput) {
        elements.findFilesInput.value = '';
        if (keepInputFocus) {
            elements.findFilesInput.focus();
        }
    }

    updateFindFilesClearButtonVisibility();
    renderProjectFindResults();
    renderFileList();
}

function updateFindFilesClearButtonVisibility() {
    if (!elements.findFilesClear || !elements.findFilesInput) {
        return;
    }

    const hasValue = (elements.findFilesInput.value || '').trim().length > 0
        || state.findFilesMode === 'references'
        || state.findFilesResults.length > 0;
    elements.findFilesClear.dataset.visible = hasValue ? 'true' : 'false';
    elements.findFilesClear.setAttribute('aria-hidden', hasValue ? 'false' : 'true');
    if (elements.findReferences) {
        elements.findReferences.dataset.active = state.findFilesMode === 'references' ? 'true' : 'false';
    }
}

const findFieldHistory = new Map();

function setFindFieldHistory(inputEl, values) {
    if (!inputEl || !inputEl.id) {
        return;
    }

    const history = Array.isArray(values) ? values.filter((value) => String(value || '').trim() !== '') : [];
    findFieldHistory.set(inputEl.id, history);
}

function getFindFieldHistory(inputEl) {
    if (!inputEl || !inputEl.id) {
        return [];
    }

    const history = findFieldHistory.get(inputEl.id);
    return Array.isArray(history) ? history : [];
}

function applyFindHistoryValue(inputEl, value) {
    if (!inputEl) {
        return;
    }

    inputEl.value = String(value || '');
    inputEl.focus();
    inputEl.setSelectionRange(inputEl.value.length, inputEl.value.length);

    if (inputEl === elements.findInput) {
        updateFindInputClearButtonVisibility();
        performFind();
        return;
    }

    if (inputEl === elements.findFilesInput) {
        updateFindFilesClearButtonVisibility();
        scheduleProjectFindSearch();
        return;
    }

    if (inputEl === elements.replaceInput) {
        updateReplaceInputClearButtonVisibility();
    }
}

function openFindHistoryMenu(inputEl, x, y) {
    if (!inputEl) {
        return false;
    }

    const history = getFindFieldHistory(inputEl);
    if (history.length === 0) {
        return false;
    }

    showLocalMenu({
        title: 'Past items',
        options: history,
        x,
        y,
        showNextToMouseCursor: true,
        onSelect: (index) => {
            const selected = history[index];
            if (selected === undefined) {
                return;
            }
            applyFindHistoryValue(inputEl, selected);
        },
    });

    return true;
}

function tryOpenFindHistoryMenuForInput(inputEl) {
    if (!inputEl) {
        return false;
    }

    const rect = inputEl.getBoundingClientRect();
    return openFindHistoryMenu(inputEl, rect.left, rect.bottom);
}

async function hydrateFindFieldHistory(inputEl) {
    if (!inputEl || !inputEl.id) {
        return;
    }

    try {
        const values = await GetNotesFindFieldValues(inputEl.id);
        setFindFieldHistory(inputEl, values);
    } catch (err) {
        console.error('Failed to read notes find field history:', err);
    }
}

function persistFindFieldHistory(inputEl) {
    if (!inputEl || !inputEl.id) {
        return;
    }

    const value = String(inputEl.value || '').trim();
    if (!value) {
        return;
    }

    AddNotesFindFieldValue(inputEl.id, value).then((values) => {
        setFindFieldHistory(inputEl, values);
    }).catch((err) => {
        console.error('Failed to persist notes find field history:', err);
    });
}

function updateFindInputClearButtonVisibility() {
    if (!elements.findInputClear || !elements.findInput) {
        return;
    }

    const hasValue = (elements.findInput.value || '').trim().length > 0;
    elements.findInputClear.dataset.visible = hasValue ? 'true' : 'false';
    elements.findInputClear.setAttribute('aria-hidden', hasValue ? 'false' : 'true');
}

function updateReplaceInputClearButtonVisibility() {
    if (!elements.replaceInputClear || !elements.replaceInput) {
        return;
    }

    const hasValue = (elements.replaceInput.value || '').trim().length > 0;
    elements.replaceInputClear.dataset.visible = hasValue ? 'true' : 'false';
    elements.replaceInputClear.setAttribute('aria-hidden', hasValue ? 'false' : 'true');
}

function editorOffsetForLine(lineNumber) {
    const targetLine = Math.max(1, Number.parseInt(String(lineNumber), 10) || 1);
    const text = getMainEditorValue();
    let currentLine = 1;
    let offset = 0;

    while (currentLine < targetLine && offset < text.length) {
        const nextBreak = text.indexOf('\n', offset);
        if (nextBreak === -1) {
            offset = text.length;
            break;
        }
        offset = nextBreak + 1;
        currentLine += 1;
    }

    return offset;
}

function jumpEditorToLine(lineNumber) {
    const start = editorOffsetForLine(lineNumber);
    const text = getMainEditorValue();
    const nextBreak = text.indexOf('\n', start);
    // Grep navigation highlights the whole matched line.
    jumpEditorToOffset(start, nextBreak === -1 ? text.length : nextBreak);
}

// Reveals an offset in Monaco, waiting for it if the editor is still being
// created. Defaults to a collapsed caret; pass `end` to select a range.
function jumpEditorToOffset(start, end = start) {
    setMainEditorSelectionRange(start, end);

    if (isMonacoActive()) {
        focusedView().editor.revealOffset(start);
        focusedView().editor.focus();
        return;
    }

    const schedule = typeof requestAnimationFrame === 'function'
        ? requestAnimationFrame
        : (cb) => setTimeout(cb, 16);

    const tryRevealWhenReady = (remainingFrames) => {
        if (isMonacoActive()) {
            focusedView().editor.setSelectionOffsets(start, end);
            focusedView().editor.revealOffset(start);
            focusedView().editor.focus();
            return;
        }

        if (remainingFrames > 0) {
            schedule(() => tryRevealWhenReady(remainingFrames - 1));
        }
    };

    // Monaco can be created/layouted asynchronously when editor view opens.
    // Retry for a bit longer so grep-navigation still lands on the requested line.
    tryRevealWhenReady(20);
}

function maybeRefreshProjectFindAfterSave() {
    if (!isProjectFindListModeActive()) {
        return;
    }

    const query = String(state.findFilesQuery || '').trim();
    if (!query) {
        return;
    }

    // Force a fresh grep run after save/autosave so file-list matches stay current.
    state.findFilesLastExecutedSignature = '';
    const scrollTop = Number(elements.list?.scrollTop) || 0;
    runProjectFindSearch(query, {
        preserveScrollTop: scrollTop,
        preserveVirtualStart: true,
    });
}

function renderProjectFindResults() {
    if (!elements.findFilesResults) {
        return;
    }

    if (elements.findFilesHeading) {
        const source = String(state.findFilesSource || '').trim();
        elements.findFilesHeading.textContent = source
            ? `References (${source})`
            : 'For files containing';
    }

    clampProjectFindScrollTop();

    const query = String(state.findFilesQuery || '').trim();
    elements.findFilesResults.innerHTML = '';

    const placeholder = document.createElement('div');
    placeholder.className = 'notes-find-files-empty';

    if (!query) {
        placeholder.textContent = 'Enter text to search project files.';
    } else if (state.findFilesBusy) {
        placeholder.textContent = 'Searching...';
    } else if (state.findFilesError) {
        placeholder.textContent = state.findFilesError;
    } else {
        const filteredResults = state.findFilesResults;
        const totalCount = filteredResults.length;
        const selectedIndex = getSelectedProjectFindResultIndex(filteredResults);
        const maxStart = Math.max(totalCount - FIND_FILES_RENDER_PAGE_SIZE, 0);
        const clampedStart = Math.min(Math.max(state.findFilesVirtualStart, 0), maxStart);
        const start = totalCount === 0 ? 0 : clampedStart + 1;
        const end = Math.min(clampedStart + FIND_FILES_RENDER_PAGE_SIZE, totalCount);
        if (selectedIndex >= 0) {
            placeholder.textContent = `${selectedIndex + 1} of ${totalCount} (showing ${start}-${end})`;
        } else {
            placeholder.textContent = `${totalCount} result${totalCount === 1 ? '' : 's'} (showing ${start}-${end})`;
        }
    }

    elements.findFilesResults.appendChild(placeholder);
}

async function runProjectFindSearch(query, options = {}) {
    const trimmed = String(query || '').trim();
    const preserveVirtualStart = options?.preserveVirtualStart === true;
    const preserveScrollTop = Math.max(0, Number(options?.preserveScrollTop) || 0);
    state.findFilesQuery = trimmed;
    if (state.findFilesMode !== 'references') {
        state.findFilesSource = '';
        state.findFilesReferenceSymbol = '';
    }
    if (trimmed) {
        persistFindFieldHistory(elements.findFilesInput);
    }
    const fileFilter = String(state.fileFilterQuery || '').trim().toLowerCase();
    const searchSignature = `${trimmed}|${state.findOptions.caseSensitive ? 1 : 0}|${state.findOptions.regex ? 1 : 0}|${state.findOptions.wholeWord ? 1 : 0}|${fileFilter}`;

    if (trimmed && searchSignature === state.findFilesLastExecutedSignature) {
        return;
    }

    if (!trimmed) {
        cleanupProjectFindStreamListeners();
        resetProjectFindPaging();
        state.findFilesBusy = false;
        state.findFilesError = '';
        state.findFilesResults = [];
        state.findFilesLastExecutedSignature = '';
        renderProjectFindResults();
        renderFileList();
        return;
    }

    const searchSeq = state.findFilesSeq + 1;
    state.findFilesSeq = searchSeq;
    state.findFilesBusy = true;
    state.findFilesError = '';
    state.findFilesResults = [];
    if (!preserveVirtualStart) {
        resetProjectFindPaging();
    }
    cleanupProjectFindStreamListeners();
    renderProjectFindResults();
    renderFileList();

    const applyPreservedScrollPosition = () => {
        if (!preserveVirtualStart) {
            return;
        }

        state.findFilesVirtualStart = getProjectFindVirtualStartFromScroll(
            state.findFilesResults.length,
            preserveScrollTop,
        );

        if (elements.list) {
            elements.list.scrollTop = preserveScrollTop;
        }
    };

    // Set up event listeners for streaming results
    const onBatch = (batch) => {
        if (state.findFilesSeq !== searchSeq) {
            return; // Ignore results from old searches
        }
        if (!Array.isArray(batch)) {
            return;
        }
        const mappedBatch = batch.map((item) => ({
            fileName: String(item?.fileName || ''),
            path: String(item?.path || ''),
            line: Number.parseInt(String(item?.line), 10) || 1,
            source: String(item?.source || state.findFilesSource || ''),
            context: Array.isArray(item?.context) ? item.context.map((l) => String(l)) : [],
        }));
        state.findFilesResults = state.findFilesResults.concat(mappedBatch);
        applyPreservedScrollPosition();
        scheduleFindFilesRender();
    };

    const onError = (error) => {
        if (state.findFilesSeq !== searchSeq) {
            return; // Ignore errors from old searches
        }
        state.findFilesError = String(error || '');
        state.findFilesBusy = false;
        cleanupProjectFindStreamListeners();
        scheduleFindFilesRender();
    };

    const onDone = () => {
        if (state.findFilesSeq !== searchSeq) {
            return; // Ignore completion from old searches
        }
        state.findFilesLastExecutedSignature = searchSignature;
        state.findFilesBusy = false;
        applyPreservedScrollPosition();
        cleanupProjectFindStreamListeners();
        scheduleFindFilesRender();
    };

    try {
        state.findFilesStreamHandlers = { onBatch, onError, onDone };

        EventsOn('notesGrepBatch', onBatch);
        EventsOn('notesGrepError', onError);
        EventsOn('notesGrepDone', onDone);

        NotesGrepStream(trimmed, {
            caseSensitive: state.findOptions.caseSensitive,
            regex: state.findOptions.regex,
            wholeWord: state.findOptions.wholeWord,
            fileFilter,
        });
    } catch (err) {
        cleanupProjectFindStreamListeners();
        if (state.findFilesSeq === searchSeq) {
            state.findFilesError = String(err && err.message ? err.message : err);
            state.findFilesBusy = false;
            renderProjectFindResults();
            renderFileList();
        }
    }
}

function cleanupProjectFindStreamListeners() {
    if (!state.findFilesStreamHandlers) {
        return;
    }

    EventsOff('notesGrepBatch', 'notesGrepError', 'notesGrepDone');
    state.findFilesStreamHandlers = null;
}

function scheduleFindFilesRender() {
    if (state.findFilesRenderQueued) {
        return;
    }

    state.findFilesRenderQueued = true;
    const schedule = typeof requestAnimationFrame === 'function'
        ? requestAnimationFrame
        : (cb) => setTimeout(cb, 16);

    schedule(() => {
        state.findFilesRenderQueued = false;
        renderProjectFindResults();
        renderFileList();
    });
}

function updateFindOptionButtons() {
    if (elements.findOptionCase) {
        elements.findOptionCase.setAttribute('data-active', state.findOptions.caseSensitive ? 'true' : 'false');
    }
    if (elements.findOptionRegex) {
        elements.findOptionRegex.setAttribute('data-active', state.findOptions.regex ? 'true' : 'false');
    }
    if (elements.findOptionWord) {
        elements.findOptionWord.setAttribute('data-active', state.findOptions.wholeWord ? 'true' : 'false');
    }
}

function scheduleProjectFindSearch() {
    const query = String(elements.findFilesInput?.value || '');
    state.findFilesQuery = query;

    if (state.findFilesTimer) {
        clearTimeout(state.findFilesTimer);
        state.findFilesTimer = null;
    }

    if (!query.trim()) {
        cleanupProjectFindStreamListeners();
        resetProjectFindPaging();
        state.findFilesError = '';
        state.findFilesResults = [];
        state.findFilesBusy = false;
        state.findFilesLastExecutedSignature = '';
        renderProjectFindResults();
        renderFileList();
        return;
    }

    state.findFilesTimer = setTimeout(() => {
        state.findFilesTimer = null;
        runProjectFindSearch(query);
    }, FIND_FILES_SEARCH_DEBOUNCE_MS);
}

function triggerImmediateProjectFindSearch() {
    const query = String(elements.findFilesInput?.value || '');

    if (state.findFilesTimer) {
        clearTimeout(state.findFilesTimer);
        state.findFilesTimer = null;
    }

    runProjectFindSearch(query);
}

function editViewModeForCurrentFileType() {
    if (state.currentFileType === 'json') {
        return 'swagger-edit';
    }
    if (state.currentFileType === 'csv') {
        return 'csv-edit';
    }
    return 'editor';
}

async function openProjectFindResult(item) {
    const file = String(item?.path || '');
    if (!file) {
        return;
    }

    try {
        state.findFilesSelectedKey = `${file}:${Number.parseInt(String(item?.line), 10) || 1}`;
        renderProjectFindResults();
        renderFileList();
        await NotesHistoryAddPosition(file, Number.parseInt(String(item?.line), 10) || 1);
        await loadFile(file, {
            skipDocumentCacheRestore: true,
        });
        setViewMode(editViewModeForCurrentFileType());
        setToolsPanelCollapsed(false);
        setToolsTab('find');
        renderProjectFindResults();
        renderFileList();
        jumpEditorToLine(item?.line);
    } catch (err) {
        notifyTerminal(String(err && err.message ? err.message : err), 'error');
        console.error(err);
    }
}

function isFindAvailableInCurrentMode() {
    return state.viewMode !== 'swagger-run'
        && state.viewMode !== 'image-view'
    && state.viewMode !== 'html-view'
        && state.viewMode !== 'hex'
        && state.viewMode !== 'meta';
}

function updateFindAvailability() {
    const available = isFindAvailableInCurrentMode();
    // Do not set disabled — that swallows click events and prevents the
    // notification from firing. Use aria-disabled for accessibility only.
    document.getElementById('notes-find')?.setAttribute('aria-disabled', available ? 'false' : 'true');
    if (elements.toolsTabFind) {
        elements.toolsTabFind.setAttribute('aria-disabled', available ? 'false' : 'true');
    }

    if (elements.findControls) {
        elements.findControls.dataset.disabled = available ? 'false' : 'true';
    }

    if (elements.findInput) {
        elements.findInput.disabled = !available;
    }

    if (elements.findPrev) {
        elements.findPrev.disabled = !available;
    }

    if (elements.findNext) {
        elements.findNext.disabled = !available;
    }

    updateReplaceAvailability();
}

function isReplaceAvailableInCurrentMode() {
    return isMainEditorFindActive();
}

function updateReplaceAvailability() {
    const enabled = isReplaceAvailableInCurrentMode();

    if (elements.replaceControls) {
        elements.replaceControls.dataset.disabled = enabled ? 'false' : 'true';
    }

    if (elements.replaceInput) {
        elements.replaceInput.disabled = !enabled;
    }

    if (elements.replaceOne) {
        elements.replaceOne.disabled = !enabled;
    }

    if (elements.replaceAll) {
        elements.replaceAll.disabled = !enabled;
    }
}

function escapeRegExp(value) {
    return String(value || '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function buildFindPattern() {
    const query = String(state.findQuery || '');
    if (!query) {
        return null;
    }

    let source = state.findDocOptions.regex ? query : escapeRegExp(query);
    if (state.findDocOptions.wholeWord) {
        source = `\\b(?:${source})\\b`;
    }

    const flags = `g${state.findDocOptions.caseSensitive ? '' : 'i'}`;
    try {
        return new RegExp(source, flags);
    } catch (_err) {
        return null;
    }
}

function isMonacoFindProxyActive() {
    return isMonacoActive() && isMainEditorFindActive();
}

function buildMonacoFindRequest() {
    const query = String(state.findQuery || '');
    if (!query) {
        return null;
    }

    let search = query;
    let isRegex = state.findDocOptions.regex;

    if (state.findDocOptions.wholeWord) {
        const core = state.findDocOptions.regex ? query : escapeRegExp(query);
        search = `\\b(?:${core})\\b`;
        isRegex = true;
    }

    return {
        query: search,
        isRegex,
        matchCase: state.findDocOptions.caseSensitive,
    };
}

function getFindDocSearchSignature(query) {
    const q = String(query || '');
    return `${q}|${state.findDocOptions.caseSensitive ? 1 : 0}|${state.findDocOptions.regex ? 1 : 0}|${state.findDocOptions.wholeWord ? 1 : 0}`;
}

function replaceCurrentMatch() {
    if (!isMainEditorFindActive()) {
        return;
    }

    const query = String(elements.findInput?.value || '');
    if (!query) {
        return;
    }
    const signature = getFindDocSearchSignature(query);

    const replacement = String(elements.replaceInput?.value || '');
    persistFindFieldHistory(elements.findInput);
    if (replacement) {
        persistFindFieldHistory(elements.replaceInput);
    }

    if (!state.findQuery || state.findQuery !== query || state.findDocLastExecutedSignature !== signature || state.findMatches.length === 0) {
        performFind();
    }

    if (state.findMatches.length === 0) {
        return;
    }

    const currentIndex = state.findCurrentIndex >= 0 ? state.findCurrentIndex : 0;
    const match = state.findMatches[currentIndex];
    if (!match || typeof match.start !== 'number' || typeof match.end !== 'number') {
        return;
    }

    replaceMainEditorRange(match.start, match.end, replacement);

    performFind();
}

function replaceAllMatches() {
    if (!isMainEditorFindActive()) {
        return;
    }

    const query = String(elements.findInput?.value || '');
    if (!query) {
        return;
    }

    const source = getMainEditorValue();
    if (!source) {
        return;
    }

    const replacement = String(elements.replaceInput?.value || '');
    persistFindFieldHistory(elements.findInput);
    if (replacement) {
        persistFindFieldHistory(elements.replaceInput);
    }
    const pattern = buildFindPattern();
    if (!pattern) {
        return;
    }
    if (!pattern.test(source)) {
        return;
    }
    pattern.lastIndex = 0;

    const next = source.replace(pattern, replacement);
    if (next === source) {
        return;
    }

    replaceMainDocumentText(next, { source: 'notes-find-replace-all' });
    performFind();
}

function getActiveFindContainer() {
    if (state.viewMode === 'jupyter') {
        return elements.jupyter;
    }

    if (state.viewMode === 'csv-view' || state.viewMode === 'csv-run') {
        return elements.csvView;
    }

    if (state.viewMode === 'swagger-view') {
        return elements.swaggerView;
    }

    return elements.preview;
}

function isMainEditorFindActive() {
    if (state.viewMode === 'editor') {
        return true;
    }

    if (state.viewMode === 'swagger-edit') {
        return state.currentFileType === 'json';
    }

    return false;
}

// Shows the active document's Find query after a load (ADR 0100 question 10).
// Matches are found again for the loaded content: here for the editor, and by
// the rendered panes once they render. Project-search results are not tied to
// a document, so they are left alone.
function showDocumentFindState() {
    clearHighlights();
    state.findMatches = [];
    state.findCurrentIndex = -1;
    state.findDocLastExecutedSignature = '';
    elements.findInput.value = state.findQuery;
    updateFindInputClearButtonVisibility();
    if (state.findQuery
        && elements.toolsTabFind?.getAttribute('aria-selected') === 'true'
        && isFindAvailableInCurrentMode()
        && isMainEditorFindActive()) {
        performFind();
        return;
    }
    updateFindCounter();
}

function clearHighlights() {
    // Clear highlights in all rendered panes that support find.
    [elements.preview, elements.jupyter, elements.swaggerView, elements.csvView].forEach((container) => {
        if (!container) {
            return;
        }

        const highlights = container.querySelectorAll('.find-highlight');
        highlights.forEach((el) => {
            const parent = el.parentNode;
            parent.replaceChild(document.createTextNode(el.textContent), el);
            parent.normalize();
        });
    });
}

function performFind() {
    if (!isFindAvailableInCurrentMode()) {
        closeFindBar();
        return;
    }

    const query = elements.findInput.value;
    if (!query) {
        clearHighlights();
        state.findMatches = [];
        state.findCurrentIndex = -1;
        state.findQuery = '';
        state.findDocLastExecutedSignature = '';
        updateFindCounter();
        return;
    }

    state.findQuery = query;
    state.findDocLastExecutedSignature = getFindDocSearchSignature(query);
    clearHighlights();
    state.findMatches = [];
    state.findCurrentIndex = -1;

    const findPattern = buildFindPattern();
    if (!findPattern) {
        updateFindCounter();
        return;
    }

    if (isMainEditorFindActive()) {
        findInEditor(findPattern);
    } else {
        findInRenderedPane(findPattern);
    }

    if (state.findMatches.length > 0) {
        state.findCurrentIndex = 0;
        highlightCurrentMatch({ focusEditor: false });
    }

    updateFindCounter();
}

function findInEditor(findPattern) {
    if (isMonacoFindProxyActive()) {
        const request = buildMonacoFindRequest();
        if (!request) {
            return;
        }

        const matches = focusedView().editor.findMatches(request.query, {
            isRegex: request.isRegex,
            matchCase: request.matchCase,
            limit: 10000,
        });

        for (const match of matches) {
            state.findMatches.push({ start: match.start, end: match.end });
        }
        return;
    }

    if (!findPattern) {
        return;
    }

    const text = getMainEditorValue();
    let match;
    while ((match = findPattern.exec(text)) !== null) {
        if (!match[0]) {
            findPattern.lastIndex++;
            continue;
        }
        state.findMatches.push({
            start: match.index,
            end: match.index + match[0].length
        });
    }
}

function findInRenderedPane(findPattern) {
    const container = getActiveFindContainer();
    if (!container || !findPattern) {
        return;
    }

    const walker = document.createTreeWalker(
        container,
        NodeFilter.SHOW_TEXT,
        null,
        false
    );

    const nodesToProcess = [];
    let node;
    while ((node = walker.nextNode())) {
        const nodePattern = new RegExp(findPattern.source, findPattern.flags);
        if (nodePattern.test(node.textContent)) {
            nodesToProcess.push(node);
        }
    }

    nodesToProcess.forEach((textNode) => {
        const text = String(textNode.textContent || '');
        const parts = [];
        let lastIndex = 0;
        const nodePattern = new RegExp(findPattern.source, findPattern.flags);
        let match;

        while ((match = nodePattern.exec(text)) !== null) {
            if (!match[0]) {
                nodePattern.lastIndex++;
                continue;
            }

            const index = match.index;
            const matchText = match[0];
            if (index > lastIndex) {
                parts.push(document.createTextNode(text.substring(lastIndex, index)));
            }

            const highlight = document.createElement('span');
            highlight.className = 'find-highlight';
            highlight.textContent = matchText;
            parts.push(highlight);
            state.findMatches.push(highlight);

            lastIndex = index + matchText.length;
        }

        if (lastIndex < text.length) {
            parts.push(document.createTextNode(text.substring(lastIndex)));
        }

        const parent = textNode.parentNode;
        parts.forEach((part) => {
            parent.insertBefore(part, textNode);
        });
        parent.removeChild(textNode);
    });
}

function highlightCurrentMatch({ focusEditor = true } = {}) {
    if (state.findMatches.length === 0 || state.findCurrentIndex === -1) {
        return;
    }

    if (isMonacoFindProxyActive()) {
        const match = state.findMatches[state.findCurrentIndex];
        if (!match) {
            return;
        }

        if (focusEditor) {
            focusedView().editor.focus();
        }
        focusedView().editor.setSelectionOffsets(match.start, match.end);
        focusedView().editor.revealOffset(match.start);
        return;
    }

    if (isMainEditorFindActive()) {
        const match = state.findMatches[state.findCurrentIndex];
        setMainEditorSelectionRange(match.start, match.end);
    } else {
        const activeContainer = getActiveFindContainer();
        if (!activeContainer) {
            return;
        }

        // Clear previous active highlight
        const prevActive = activeContainer.querySelector('.find-highlight-active');
        if (prevActive) {
            prevActive.classList.remove('find-highlight-active');
        }

        // Highlight current match
        const currentMatch = state.findMatches[state.findCurrentIndex];
        currentMatch.classList.add('find-highlight-active');
        currentMatch.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
}

function nextMatch() {
    if (state.findMatches.length === 0) {
        return;
    }

    state.findCurrentIndex = (state.findCurrentIndex + 1) % state.findMatches.length;
    highlightCurrentMatch();
    updateFindCounter();
}

function prevMatch() {
    if (state.findMatches.length === 0) {
        return;
    }

    state.findCurrentIndex = (state.findCurrentIndex - 1 + state.findMatches.length) % state.findMatches.length;
    highlightCurrentMatch();
    updateFindCounter();
}

function updateFindCounter() {
    if (state.findMatches.length === 0) {
        elements.findCounter.textContent = 'No matches';
    } else {
        elements.findCounter.textContent = `${state.findCurrentIndex + 1} of ${state.findMatches.length}`;
    }
}

function updateFindDocOptionButtons() {
    if (elements.findDocOptionCase) {
        elements.findDocOptionCase.setAttribute('data-active', state.findDocOptions.caseSensitive ? 'true' : 'false');
    }
    if (elements.findDocOptionRegex) {
        elements.findDocOptionRegex.setAttribute('data-active', state.findDocOptions.regex ? 'true' : 'false');
    }
    if (elements.findDocOptionWord) {
        elements.findDocOptionWord.setAttribute('data-active', state.findDocOptions.wholeWord ? 'true' : 'false');
    }
}

function openNewFilePrompt({ location = '$NOTES', name = '', pendingFile = null } = {}) {
    state.renamingFile = null;
    state.pendingNewFile = pendingFile;
    elements.modal.dataset.open = 'true';
    elements.modal.setAttribute('aria-hidden', 'false');
    elements.modalInput.value = name;
    elements.modalLocation.textContent = location;
    elements.modalLocation.style.display = '';
    setNotesModalTitle('New note name', true);
    elements.modalCreate.textContent = 'Create';
    setTimeout(() => {
        elements.modalInput.focus();
    }, 0);
}

async function openCreateGitWorktreePrompt(projectRoot) {
    let branches;
    let defaultPath;
    try {
        [branches, defaultPath] = await Promise.all([
            ListGitBranches(projectRoot),
            GetDefaultGitWorktreePath(projectRoot),
        ]);
    } catch (error) {
        notifyTerminal(String(error?.message || error), 'error');
        return;
    }

    state.renamingFile = null;
    state.pendingNewFile = null;
    state.textPromptHandler = null;
    const prompt = { branch: '', createBranch: false };
    state.gitWorktreePrompt = prompt;
    elements.modal.dataset.open = 'true';
    elements.modal.setAttribute('aria-hidden', 'false');
    elements.modal.querySelector('#notes-modal-card').classList.add('notes-worktree-dialog');
    elements.modalLocationRow.style.display = 'none';
    elements.modalBody.style.display = 'flex';
    elements.modalBody.replaceChildren();
    elements.modalInput.value = '';
    setNotesModalTitle('Create new worktree', true);
    elements.modalCreate.textContent = 'Create';

    const form = document.createElement('div');
    form.className = 'notes-worktree-form';
    elements.modalBody.appendChild(form);

    const branchField = notesGraphField('Branch');
    const branchButton = document.createElement('button');
    branchButton.type = 'button';
    branchButton.className = 'notes-graph-dropdown';
    prompt.branchButton = branchButton;
    branchButton.textContent = 'Select branch';
    branchField.field.appendChild(branchButton);
    form.appendChild(branchField.field);

    const pathField = notesGraphField('Path');
    const pathInput = document.createElement('textarea');
    pathInput.className = 'notes-worktree-path-input';
    disableNativeTextAssistance(pathInput);
    pathInput.rows = 2;
    pathInput.wrap = 'soft';
    pathInput.disabled = true;
    pathInput.addEventListener('input', () => {
        pathInput.value = pathInput.value.replace(/[\r\n]+/g, ' ');
    });
    pathInput.addEventListener('keydown', (event) => {
        if (event.key === 'Enter') {
            event.preventDefault();
            void prompt.submit?.();
        }
    });
    pathField.field.appendChild(pathInput);
    form.appendChild(pathField.field);

    const updateBranch = (branch, createBranch) => {
        prompt.branch = branch;
        prompt.createBranch = createBranch;
        branchButton.textContent = `${branch || 'Select branch'}`;
        const pathSeparator = defaultPath.includes('\\') ? '\\' : '/';
        pathInput.value = branch
            ? `${defaultPath.replace(/[\\/]+$/, '')}${pathSeparator}${branch.split('/').join(pathSeparator)}`
            : '';
        pathInput.disabled = !branch;
        elements.modalCreate.disabled = !branch;
    };

    branchButton.addEventListener('click', () => {
        const menuItems = [{
            title: 'Create new branch',
            icon: CONTEXT_ICON_ADD,
            onSelect: async () => {
                const branch = await openGitBranchPrompt();
                if (!branch) {
                    updateBranch('', false);
                    return;
                }
                updateBranch(branch, true);
            },
        }, { title: '-', icon: 0 }];
        menuItems.push(...[...branches].sort((left, right) => left.localeCompare(right)).map((branch) => ({
            title: branch,
            icon: prompt.branch === branch && !prompt.createBranch ? CONTEXT_ICON_TICK : 0,
            onSelect: () => updateBranch(branch, false),
        })));
        const rect = branchButton.getBoundingClientRect();
        showNotesLocalMenu(menuItems, rect.left, rect.bottom, 'Branch');
    });

    prompt.submit = async () => {
        if (!prompt.branch) {
            notifyTerminal('Select a branch for the new worktree.', 'error');
            return;
        }
        const worktreePath = pathInput.value.trim();
        if (!worktreePath) {
            notifyTerminal('Enter a path for the new worktree.', 'error');
            return;
        }
        const branch = prompt.branch;
        const createBranch = prompt.createBranch;
        finishGitWorktreePrompt();
        void createNotesGitWorktree(projectRoot, branch, worktreePath, createBranch);
    };

    elements.modalCreate.disabled = true;
    setTimeout(() => branchButton.focus(), 0);
}

async function createNotesGitWorktree(projectRoot, branch, worktreePath, createBranch) {
    const stickyId = `git-worktree-${Date.now()}-${Math.random().toString(36).slice(2)}`;
    await openStickyProgress(stickyId, 'Creating worktree');
    try {
        const worktree = await CreateGitWorktree(projectRoot, branch, worktreePath, createBranch);
        await switchNotesGitWorktree(projectRoot, worktree.path || worktreePath);
        closeStickyProgress(stickyId);
    } catch (error) {
        closeStickyProgress(stickyId, String(error?.message || error), 'error');
    }
}

async function switchNotesGitWorktree(projectRoot, worktreePath) {
    window.dispatchEvent(new CustomEvent('ttyphoon-focus-terminal'));
    await SwitchGitWorktree(projectRoot, worktreePath);
}

function finishGitWorktreePrompt() {
    if (!state.gitWorktreePrompt) {
        return;
    }
    state.gitWorktreePrompt = null;
    elements.modal.querySelector('#notes-modal-card').classList.remove('notes-worktree-dialog');
    elements.modalBody.replaceChildren();
    elements.modalBody.style.display = 'none';
    elements.modalLocationRow.style.display = '';
    elements.modalLocation.style.display = '';
    elements.modalInput.placeholder = 'example-note';
    elements.modal.dataset.open = 'false';
    elements.modal.setAttribute('aria-hidden', 'true');
    elements.modalCreate.disabled = false;
    elements.modalCreate.textContent = 'Create';
    setNotesModalTitle('New note name');
}

function openGitBranchPrompt() {
    return new Promise((resolve) => {
        state.gitBranchPromptHandler = resolve;
        elements.modalBody.style.display = 'none';
        elements.modalLocationRow.style.display = '';
        elements.modalLocation.style.display = 'none';
        elements.modalInput.value = '';
        elements.modalInput.placeholder = 'branch-name';
        setNotesModalTitle('Create branch', true);
        elements.modalCreate.textContent = 'Create';
        elements.modalCreate.disabled = false;
        setTimeout(() => elements.modalInput.focus(), 0);
    });
}

function finishGitBranchPrompt(value) {
    if (!state.gitBranchPromptHandler) {
        return;
    }
    const handler = state.gitBranchPromptHandler;
    state.gitBranchPromptHandler = null;
    elements.modalBody.style.display = 'flex';
    elements.modalLocationRow.style.display = 'none';
    elements.modalLocation.style.display = 'none';
    elements.modalInput.value = '';
    elements.modalInput.placeholder = 'example-note';
    setNotesModalTitle('Create new worktree', true);
    elements.modalCreate.textContent = 'Create';
    handler(value);
    setTimeout(() => state.gitWorktreePrompt?.branchButton?.focus(), 0);
}

async function openRenamePrompt(file, folder = false) {
    const source = String(file || '').trim();
    state.renamingFile = file;
    state.renamingFolder = folder;
    state.pendingNewFile = null;
    elements.modal.dataset.open = 'true';
    elements.modal.setAttribute('aria-hidden', 'false');
    let location = '$NOTES';
    let name = source;

    try {
        const resolved = await ResolveNoteLocation(source);
        const parsed = splitResolvedNoteLocation(resolved);
        location = parsed.location;
        name = parsed.name;
    } catch {
        location = '$NOTES';
        name = source;
    }

    elements.modalLocation.textContent = location;
    elements.modalLocation.style.display = '';
    elements.modalInput.value = name;
    setNotesModalTitle(folder ? 'Rename folder' : 'Rename note', true);
    elements.modalCreate.textContent = 'Rename';
    setTimeout(() => {
        elements.modalInput.focus();
        elements.modalInput.select();
    }, 0);
}

function closeNewFilePrompt() {
    if (state.graphPromptHandler) {
        finishNotesTableGraphPrompt();
        return;
    }
    if (state.textPromptHandler) {
        finishTextPrompt(null);
        return;
    }
    if (state.gitBranchPromptHandler) {
        finishGitBranchPrompt(null);
        return;
    }
    if (state.gitWorktreePrompt) {
        finishGitWorktreePrompt();
        return;
    }
    state.pendingNewFile = null;
    elements.modal.dataset.open = 'false';
    elements.modal.setAttribute('aria-hidden', 'true');
}

// Reuses the note create/rename modal as a generic single-line text prompt, so
// features needing one-line input (e.g. renaming an AI session) don't fall back
// to window.prompt. Resolves the trimmed input, or null when cancelled.
function openTextPrompt({ title = 'Enter value', value = '', confirmLabel = 'OK', placeholder = '' } = {}) {
    return new Promise((resolve) => {
        state.renamingFile = null;
        state.textPromptHandler = resolve;
        elements.modal.dataset.open = 'true';
        elements.modal.setAttribute('aria-hidden', 'false');
        elements.modalLocation.style.display = 'none';
        elements.modalInput.value = value;
        elements.modalInput.placeholder = placeholder;
        setNotesModalTitle(title);
        elements.modalCreate.textContent = confirmLabel;
        setTimeout(() => {
            elements.modalInput.focus();
            elements.modalInput.select();
        }, 0);
    });
}

function finishTextPrompt(value) {
    if (!state.textPromptHandler) {
        return;
    }
    const handler = state.textPromptHandler;
    state.textPromptHandler = null;
    elements.modalLocation.style.display = '';
    elements.modalInput.placeholder = 'example-note';
    elements.modal.dataset.open = 'false';
    elements.modal.setAttribute('aria-hidden', 'true');
    handler(value);
}

function normalizeNoteName(rawName) {
    const trimmed = (rawName || '').trim();
    if (trimmed === '') {
        return '';
    }

    const leaf = trimmed.split('/').filter(Boolean).pop() || '';
    if (/\.[^./]+$/.test(leaf)) {
        return trimmed;
    }

    return `${trimmed}.md`;
}

function normalizeNotePath(rawName) {
    const fileName = normalizeNoteName(rawName);
    if (fileName === '') {
        return '';
    }

    if (fileName.startsWith('$') || fileName.startsWith('/')) {
        return fileName;
    }

    return `$NOTES/${fileName}`;
}

function splitResolvedNoteLocation(rawPath) {
    const source = String(rawPath || '').trim();
    for (const location of NOTE_LOCATIONS) {
        if (source === location) {
            return { location, name: '' };
        }

        if (source.startsWith(`${location}/`) || source.startsWith(`${location}\\`)) {
            return {
                location,
                name: source.slice(location.length + 1),
            };
        }
    }

    return {
        location: '$NOTES',
        name: source,
    };
}

function deriveImageExtension(mimeType) {
    if (!mimeType) {
        return 'png';
    }

    const subtype = mimeType.split('/')[1] || '';
    const normalized = subtype.toLowerCase().split('+')[0];
    if (normalized === 'jpeg') {
        return 'jpg';
    }

    if (/^[a-z0-9]+$/.test(normalized)) {
        return normalized;
    }

    return 'png';
}

function buildImagePaths(notePath, epoch, extension) {
    const slash = notePath.lastIndexOf('/');
    const dir = slash === -1 ? '' : notePath.slice(0, slash + 1);
    const file = slash === -1 ? notePath : notePath.slice(slash + 1);
    const imageDirName = `${file}.d`;
    const imageFileName = `${epoch}.${extension}`;
    const markdownImagePath = `${imageDirName}/${imageFileName}`;
    return {
        imagePath: `${dir}${markdownImagePath}`,
        imageFileName: markdownImagePath,
    };
}

function getMarkdownImageAtCursor(markdown, cursor) {
    if (!markdown || !Number.isFinite(cursor)) {
        return null;
    }

    const imageRegex = /!\[[^\]]*\]\(([^)]+)\)/g;
    let match;

    while ((match = imageRegex.exec(markdown)) !== null) {
        const start = match.index;
        const end = start + match[0].length;
        if (cursor < start || cursor > end) {
            continue;
        }

        const rawTarget = (match[1] || '').trim();
        if (rawTarget === '') {
            return null;
        }

        let imagePath = rawTarget;
        if (rawTarget.startsWith('<') && rawTarget.endsWith('>')) {
            imagePath = rawTarget.slice(1, -1).trim();
        } else {
            const splitAt = rawTarget.search(/\s/);
            if (splitAt !== -1) {
                imagePath = rawTarget.slice(0, splitAt).trim();
            }
        }

        return {
            markdown: match[0],
            markdownStart: start,
            markdownEnd: end,
            imagePath,
        };
    }

    return null;
}

function isRelativeMarkdownImagePath(imagePath) {
    if (!imagePath) {
        return false;
    }

    if (imagePath.startsWith('/') || imagePath.startsWith('$') || imagePath.startsWith('//')) {
        return false;
    }

    // Exclude schemes like http:, https:, data:, file:, etc.
    if (/^[a-z][a-z0-9+.-]*:/i.test(imagePath)) {
        return false;
    }

    return true;
}

function resolveRelativeAssetPath(notePath, relativePath) {
    const slash = notePath.lastIndexOf('/');
    const dir = slash === -1 ? '' : notePath.slice(0, slash + 1);
    let decodedPath = relativePath;
    try {
        decodedPath = decodeURIComponent(relativePath);
    } catch {
        // Keep malformed escapes literal so existing paths remain deletable.
    }
    return `${dir}${decodedPath}`;
}

function enableImageContextMenus(container) {
    const images = container.querySelectorAll('img');
    images.forEach((img) => {
        // The AI panel re-runs this over surviving nodes, so don't stack listeners.
        if (img.dataset.imageMenuBound === 'true') {
            return;
        }
        img.dataset.imageMenuBound = 'true';

        img.addEventListener('contextmenu', async (e) => {
            if (hasRenderedSelection(container)) return;
            e.preventDefault();
            e.stopPropagation();
            
            const src = img.src;
            if (!src) return;
            const sectionHeading = getRenderedSectionHeading(container, img, e.clientY);
            
            // Use the original filename from the data attribute if available
            let filename = img.dataset.originalFilename || img.alt || 'Image';
            
            // Images are served by the Go asset handler, so the bytes aren't in
            // hand. Fetch them only once an action needs them, otherwise every
            // right-click would pull the whole file for a menu that may be dismissed.
            const resolveDataURL = async () => {
                if (src.startsWith('data:')) {
                    return src;
                }
                const response = await fetch(src);
                const blob = await response.blob();
                return await new Promise((resolve, reject) => {
                    const reader = new FileReader();
                    reader.onload = () => resolve(reader.result);
                    reader.onerror = () => reject(reader.error);
                    reader.readAsDataURL(blob);
                });
            };

            showLocalMenu({
                title: filename,
                options: ['Copy image to clipboard', 'Save image...', 'Ask AI (image)...', ...(sectionHeading ? ['Collapse section'] : [])],
                x: e.clientX,
                y: e.clientY,
                showNextToMouseCursor: true,
                icons: [0xf0c5, 0xf0c7, CONTEXT_ICON_ASK_AI, ...(sectionHeading ? [CONTEXT_ICON_COLLAPSE_ALL] : [])],
                onSelect: async (index) => {
                    if (index === 3 && sectionHeading) {
                        collapseCollapsibleHeading(sectionHeading);
                        return;
                    }
                    let dataURLToCopy;
                    try {
                        dataURLToCopy = await resolveDataURL();
                    } catch (err) {
                        console.error('Failed to load image:', err);
                        notifyTerminal(`Failed to load image: ${filename}`, 'error');
                        return;
                    }

                    if (index === 0) {
                        TerminalCopyImageDataURL(dataURLToCopy).catch(() => {
                            notifyTerminal('Failed to copy image to clipboard', 'error');
                        });
                    } else if (index === 1) {
                        saveImageToFile(filename, dataURLToCopy);
                    } else if (index === 2) {
                        askAIAboutImage(filename, dataURLToCopy);
                    }
                },
            });
        });
    });
}

function copyTextToClipboard(text) {
    if (!text) {
        return;
    }

    ClipboardSetText(text).catch(() => {});
}

function getLinkTextFromAnchor(anchor) {
    if (!(anchor instanceof Element)) {
        return '';
    }

    const text = (anchor.textContent || '').trim();
    if (text.length > 0) {
        return text;
    }

    return String(anchor.getAttribute('href') || '').trim();
}

async function openHyperlinkContextMenu(anchor) {
    if (!(anchor instanceof HTMLAnchorElement)) {
        return;
    }

    const href = String(anchor.getAttribute('href') || '').trim();
    if (!href) {
        return;
    }

    const absoluteUrl = String(anchor.href || href);
    const label = getLinkTextFromAnchor(anchor);

    try {
        await DisplayHyperlinkMenu(absoluteUrl, label);
    } catch {
        setStatus('Failed to open hyperlink actions', true);
    }
}

function getJsonEditableCopyText(editable) {
    if (!(editable instanceof Element)) {
        return '';
    }

    const editType = editable.getAttribute('data-json-edit');
    if (editType === 'key') {
        const pathAttr = editable.getAttribute('data-json-path') || '[]';
        try {
            const path = JSON.parse(pathAttr);
            return String(path[path.length - 1] ?? '');
        } catch {
            return (editable.textContent || '').replace(/^"|"$/g, '');
        }
    }

    if (editType === 'value') {
        const rawValueAttr = editable.getAttribute('data-json-value');
        if (rawValueAttr) {
            try {
                const parsedValue = JSON.parse(rawValueAttr);
                return parsedValue === null ? 'null' : String(parsedValue);
            } catch {
                // Fall through to text content if the attribute cannot be parsed.
            }
        }
        return (editable.textContent || '').replace(/^"|"$/g, '');
    }

    return editable.textContent || '';
}

function getEditorSelectionText() {
    const { start, end } = mainDocument.getSelection();
    return mainDocument.getValue().slice(start, end);
}

function hasRenderedSelection(container) {
    const selection = window.getSelection();
    if (!selection || selection.rangeCount === 0 || selection.isCollapsed) {
        return false;
    }

    const anchorNode = selection.anchorNode;
    const focusNode = selection.focusNode;
    const selectionInContainer =
        (anchorNode && container.contains(anchorNode)) &&
        (focusNode && container.contains(focusNode));

    return Boolean(selectionInContainer);
}

function getRenderedCodeBlockText(container, eventTarget) {
    if (!(eventTarget instanceof Element) || !container || !container.contains(eventTarget)) {
        return '';
    }

    const codeEl = eventTarget.closest('pre code, pre, code');
    if (!codeEl || !container.contains(codeEl)) {
        return '';
    }

    const pre = codeEl.closest('pre');
    if (!pre || !container.contains(pre)) {
        return '';
    }

    const preCode = pre.querySelector('code');
    return String((preCode ? preCode.textContent : pre.textContent) || '');
}

function getRenderedCodeBlockLanguage(container, eventTarget) {
    if (!(eventTarget instanceof Element) || !container || !container.contains(eventTarget)) {
        return '';
    }

    const codeEl = eventTarget.closest('pre code');
    if (!codeEl || !container.contains(codeEl)) {
        return '';
    }

    const languageClass = Array.from(codeEl.classList).find((name) => name.startsWith('language-'));
    return languageClass ? languageClass.slice('language-'.length).toLowerCase() : '';
}

function codeBlockFilename(language) {
    const extensions = {
        'c++': 'cpp', csharp: 'cs', 'f#': 'fs', golang: 'go', javascript: 'js',
        markdown: 'md', plaintext: 'txt', powershell: 'ps1', shell: 'sh', text: 'txt',
        typescript: 'ts', yaml: 'yaml', yml: 'yaml',
    };
    const normalized = String(language || '').trim().toLowerCase();
    const extension = extensions[normalized] || (normalized && /^[a-z0-9]+$/.test(normalized) ? normalized : 'txt');
    return `code-block.${extension}`;
}

function createSaveCodeBlockMenuItem(code, language) {
    return {
        title: 'Save code...',
        icon: 0xf0c7,
        onSelect: async () => {
            try {
                const path = await SaveCodeDialog(codeBlockFilename(language));
                if (!path) {
                    return;
                }
                await SaveFile(path, code, '');
                notifyTerminal(`Saved code block ${path}`, 'info');
            } catch (err) {
                console.error('Failed to save code block:', err);
                notifyTerminal('Failed to save code block', 'error');
            }
        },
    };
}

function createCopyMenuItem(getText, title = 'Copy') {
    return {
        title,
        icon: CONTEXT_ICON_COPY,
        onSelect: () => {
            copyTextToClipboard(getText());
        },
    };
}

const renderedSelectionCopier = createRenderedSelectionCopier({
    write: CopyFormattedSelection,
    notify: notifyTerminal,
});
document.addEventListener('copy', renderedSelectionCopier.handleCopy, true);

function createRenderedCopyMenuItem(container, title = 'Copy selection') {
    return { ...renderedSelectionCopier.menuItem(container), title };
}

async function copyAIMessage(kind, root) {
    try {
        let snapshot;
        if (kind === 'prompt') {
            const source = await aiPipelineFormatter.promptMarkdown(root || undefined);
            const host = document.createElement('div');
            const content = document.createElement('div');
            host.append(content);
            await renderPrintMarkdown(content, source);
            snapshot = captureRenderedElement(content);
        } else {
            snapshot = captureRenderedElements(await aiPipelineFormatter.outputElements(root));
        }
        if (!snapshot) {
            notifyTerminal(`No AI ${kind} to copy`, 'info');
            return;
        }
        await renderedSelectionCopier.copy(snapshot);
    } catch (error) {
        console.error(`Failed to copy AI ${kind}`, error);
        notifyTerminal(`Failed to copy AI ${kind}: ${error instanceof Error ? error.message : error}`, 'error');
    }
}

function getRenderedCopyBlockText(block) {
    if (!(block instanceof Element)) {
        return '';
    }

    const copy = block.cloneNode(true);
    copy.querySelectorAll('.notes-copy-block-actions').forEach((actions) => actions.remove());
    const code = copy.matches('pre') ? copy.querySelector('code') : null;
    return String((code || copy).textContent || '');
}

function ensureRenderedBlockCopyButton(block) {
    if (!(block instanceof HTMLElement) || block.dataset.copyButtonBound === 'true') {
        return;
    }

    block.dataset.copyButtonBound = 'true';
    block.classList.add('notes-copy-block');

    const actions = document.createElement('div');
    actions.className = 'notes-copy-block-actions';

    // Only code blocks (not blockquotes) can be meaningfully sent to a shell.
    if (block.matches('pre')) {
        const terminalButton = document.createElement('button');
        terminalButton.type = 'button';
        terminalButton.className = 'notes-send-terminal-block-button';
        terminalButton.title = 'Send block to terminal';
        terminalButton.setAttribute('aria-label', 'Send block to terminal');
        terminalButton.textContent = String.fromCodePoint(CONTEXT_ICON_SEND_TO_TERMINAL);
        terminalButton.addEventListener('click', async (event) => {
            event.preventDefault();
            event.stopPropagation();
            try {
                const content = getRenderedCopyBlockText(block);
                const code = block.querySelector('code');
                const languageClass = Array.from(code?.classList || []).find((name) => name.startsWith('language-'));
                const language = languageClass ? languageClass.slice('language-'.length) : hljs.highlightAuto(content)?.language || '';
                const descriptions = language ? await GetLanguageDescriptions(language) : [];
                const runtime = descriptions?.[0] || language;
                if (!runtime) {
                    throw new Error('Cannot determine the runtime; select one in Run mode');
                }
                await SendToTerminal(content, runtime);
                window.dispatchEvent(new CustomEvent('ttyphoon-focus-terminal'));
                terminalButton.dataset.sent = 'true';
                setTimeout(() => {
                    terminalButton.dataset.sent = 'false';
                }, 700);
            } catch (err) {
                console.error('Error sending to terminal:', err);
                notifyTerminal(`Failed to send to terminal: ${err}`, 'error');
            }
        });
        actions.appendChild(terminalButton);
    }

    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'notes-copy-block-button';
    button.title = 'Copy block to clipboard';
    button.setAttribute('aria-label', 'Copy block to clipboard');
    button.textContent = String.fromCodePoint(CONTEXT_ICON_COPY);
    button.addEventListener('click', (event) => {
        event.preventDefault();
        event.stopPropagation();
        copyTextToClipboard(getRenderedCopyBlockText(block));
        button.dataset.copied = 'true';
        setTimeout(() => {
            button.dataset.copied = 'false';
        }, 700);
    });
    actions.appendChild(button);

    // Compositor-only transform instead of position: sticky, which was
    // observed to flicker the panel's translucent background on WKWebView
    // whenever the buttons faded in/out (e.g. hovering the block).
    block.insertBefore(actions, block.firstChild);
    const syncActionsOffset = () => {
        actions.style.transform = `translateY(${block.scrollTop}px)`;
    };
    syncActionsOffset();
    block.addEventListener('scroll', syncActionsOffset, { passive: true });
}

function initRenderedBlockCopyButtons(container) {
    if (!container || container.dataset.copyBlockButtonsBound === 'true') {
        return;
    }

    container.dataset.copyBlockButtonsBound = 'true';
    container.addEventListener('mouseover', (event) => {
        const target = event.target instanceof Element ? event.target : null;
        const block = target?.closest('pre, blockquote');
        if (block && container.contains(block)) {
            ensureRenderedBlockCopyButton(block);
        }
    });
}

function createFindMenuItem(title = 'Find text...') {
    return {
        title,
        icon: CONTEXT_ICON_FIND,
        onSelect: () => {
            openFindBar();
        },
    };
}

function reportPrintError(error) {
    console.error('Failed to print document', error);
    notifyTerminal('Failed to print document', 'error');
}

async function renderPrintMarkdown(container, source) {
    container.innerHTML = marked.parse(source);
    await processMarkdownContainer(container);
}

function createPrintMenuItem(title = 'Print...') {
    return {
        title,
        icon: CONTEXT_ICON_PRINT,
        onSelect: () => {
            void printCurrentDocument().catch(reportPrintError);
        },
    };
}

async function printCurrentDocument() {
    const viewMode = state.viewMode;
    const printMarkdown = viewMode === 'viewer' || viewMode === 'jupyter';
    if (!printMarkdown && viewMode !== 'editor') {
        discardPrintDocument();
        closeSharedTooltip();
        await WindowPrint();
        return;
    }
    const file = splitCategoryPath(state.currentFile);
    const filename = file.category === '$PROJECT' ? file.relativePath : getPathFileName(state.currentFile);
    await printDocument({
        source: printMarkdown ? parseFrontmatter(getMainEditorValue()).body : getMainEditorValue(),
        filename: viewMode === 'editor' ? filename : '',
        renderMarkdown: printMarkdown ? renderPrintMarkdown : null,
        dismissMenu: closeSharedTooltip,
        print: WindowPrint,
    });
}

function getCurrentDocumentContentsForAI() {
    const fileType = String(state.currentFileType || '').toLowerCase();

    let contents = '';
    if (fileType === 'image') {
        contents = String(elements.imageViewImg?.src || '').trim();
    } else if (fileType === 'binary') {
        contents = String(state.hexSourceValue || mainDocument.getValue());
    } else {
        contents = mainDocument.getValue();
    }

    if (!contents && fileType === 'meta') {
        contents = String(state.fileMetaMarkdown || elements.meta?.textContent || '');
    }

    const maxChars = 120000;
    if (contents.length > maxChars) {
        contents = `${contents.slice(0, maxChars)}\n\n[document truncated for AI input]`;
    }

    return contents;
}

async function askAIAboutCurrentDocument() {
    if (!state.currentFile) {
        setStatus('Open a document first', true);
        return;
    }

    const fileName = state.currentFile;
    const fileType = state.currentFileType || 'unknown';

    // Route image documents through the dedicated image path instead of
    // embedding the base64 data URL as document text: that text is persisted
    // verbatim to the session history and previously froze the AI Settings
    // modal when it tried to render it back out.
    if (String(fileType).toLowerCase() === 'image') {
        await askAIAboutImage(getPathFileName(fileName) || fileName, String(elements.imageViewImg?.src || '').trim());
        return;
    }

    const contents = getCurrentDocumentContentsForAI();
    if (!contents) {
        notifyTerminal('This document has no content to send to AI', 'warn');
        return;
    }

    const aiContext = [
        `Document: ${fileName}`,
        `Type: ${fileType}`,
        '',
        contents,
    ].join('\n');

    setToolsPanelCollapsed(false);
    setAIPanelLive(true);

    try {
        await AskAI('notesDocument', fileName, aiContext);
    } catch (err) {
        notifyTerminal('Failed to ask AI about this document', 'error');
        console.error(err);
    }
}

async function askAIAboutImage(filename, dataURL) {
    const imageData = String(dataURL || '').trim();
    if (!/^data:image\/[a-zA-Z0-9.+-]+;base64,.+$/i.test(imageData)) {
        notifyTerminal('Unable to prepare this image for AI', 'warn');
        return;
    }

    const imageName = String(filename || 'Image').trim() || 'Image';
    setToolsPanelCollapsed(false);
    setAIPanelLive(true);

    try {
        await AskAIImage(imageName, imageData);
    } catch (err) {
        notifyTerminal('Failed to ask AI about this image', 'error');
        console.error(err);
    }
}

function openAISessionManagementModal() {
    if (!elements.aiSettingsModal) {
        return;
    }

    renderAISessionManagement();
    elements.aiSettingsModal.dataset.open = 'true';
    elements.aiSettingsModal.setAttribute('aria-hidden', 'false');
}

function closeAISessionManagementModal() {
    if (!elements.aiSettingsModal) {
        return;
    }

    closeAIToolMetadataModal();
    elements.aiSettingsModal.dataset.open = 'false';
    elements.aiSettingsModal.setAttribute('aria-hidden', 'true');
}

function normalizeAISessionManagement(stateValue) {
    const next = stateValue && typeof stateValue === 'object' ? stateValue : {};
    return {
        activeSessionId: Number(next.activeSessionId) || 0,
        sessions: Array.isArray(next.sessions) ? next.sessions : [],
        history: Array.isArray(next.history) ? next.history : [],
    };
}

function applyAISessionManagement(nextState) {
    state.aiSessionManagement = normalizeAISessionManagement(nextState);
}

async function loadAISessionManagement() {
    try {
        const [sessionState, toolsList, mcpServers] = await Promise.all([
            GetAISessionManagement(),
            GetAIToolsList(),
            GetAIMcpServers(),
        ]);
        applyAISessionManagement(sessionState);
        state.aiToolsList = Array.isArray(toolsList) ? toolsList : [];
        state.aiMcpServersList = Array.isArray(mcpServers) ? mcpServers : [];
    } catch (err) {
        console.error('Failed to load AI session management:', err);
        applyAISessionManagement(null);
        state.aiToolsList = [];
        state.aiMcpServersList = [];
    }
}

function renderAISessionManagement() {
    renderAISessionList();
    renderAISessionHistory();
    renderAIToolsList();
    renderAIMcpServersList();
}

function summarizePromptText(value, fallback) {
    const normalized = String(value || '').replace(/\s+/g, ' ').trim();
    if (!normalized) {
        return fallback;
    }
    if (normalized.length <= 90) {
        return normalized;
    }
    return `${normalized.slice(0, 87).trimEnd()}...`;
}

// Backend-provided prompt log metadata, refreshed on aiJobFinish / panel init.
// Each entry is { sessionId, promptId, heading, summary } from Go via
// ListAIPromptLogs. The heading is stored in SQLite when the prompt starts.
async function refreshAIPromptJumpFromBackend() {
    if (!elements.toolsAIPromptJump) {
        return [];
    }

    let metas = [];
    try {
        const raw = await ListAIPromptLogs();
        metas = Array.isArray(raw) ? raw : [];
    } catch (err) {
        console.error('Failed to load AI prompt logs:', err);
        metas = [];
    }

    const targets = metas.map((meta, index) => ({
        source: 'backend',
        sessionId: Number(meta?.sessionId) || 0,
        promptId: Number(meta?.promptId) || 0,
        status: String(meta?.status || 'success'),
        started: String(meta?.started || ''),
        summary: summarizePromptText(String(meta?.heading || ''), `Prompt ${index + 1}`),
    })).filter((entry) => entry.sessionId > 0 && entry.promptId > 0);

    state.aiPromptJumpTargets = targets;
    updateAIPromptJumpAvailability();
    return targets;
}

function updateAIPromptJumpAvailability() {
    if (!elements.toolsAIPromptJump) {
        return;
    }
    elements.toolsAIPromptJump.disabled = !state.aiRunActive
        && (!Array.isArray(state.aiPromptJumpTargets) || state.aiPromptJumpTargets.length === 0);
}

function renderAIPromptJumpDropdown() {
    if (!elements.toolsAIPromptJump) {
        return;
    }
    updateAIPromptJumpAvailability();
    void refreshAIPromptJumpFromBackend();
}

function scheduleAIPromptJumpRefresh() {
    const now = Date.now();
    if (!aiPromptJumpRefreshDeadline) {
        aiPromptJumpRefreshDeadline = now + AI_PROMPT_JUMP_REFRESH_MAX_DELAY_MS;
    }

    if (aiPromptJumpRefreshTimer) {
        clearTimeout(aiPromptJumpRefreshTimer);
    }

    // Streaming calls this once per chunk; without the deadline cap the debounce
    // would reset forever and the dropdown would never list finished prompts.
    const delay = Math.max(0, Math.min(40, aiPromptJumpRefreshDeadline - now));
    aiPromptJumpRefreshTimer = setTimeout(() => {
        aiPromptJumpRefreshTimer = null;
        aiPromptJumpRefreshDeadline = 0;
        renderAIPromptJumpDropdown();
    }, delay);
}

async function resumeLiveAIOutput() {
    if (!state.aiRunActive) {
        return;
    }
    state.aiSelectedPrompt = { source: 'live' };
    setAIPanelLive(true);
    await loadAISessionCache('');
}

async function renderAIStoredPrompt(blocks, target) {
    const promptText = await GetAISessionPrompt(target.sessionId, target.promptId);
    aiPipelineFormatter.setBlocks(blocks.map((block) => ({ ...block, userPrompt: promptText })), (block) =>
        GetAIStreamBlockContent(target.sessionId, target.promptId, String(block?.blockId || '')), {
        collapseIntermediate: true,
        prompt: promptText || target.summary,
        timestamp: target.started,
    });
}

async function jumpToAIPromptTarget(target) {
    if (!target) {
        return;
    }

    if (target.source === 'live') {
        if (!state.aiRunActive) {
            return;
        }
        await resumeLiveAIOutput();
        return;
    }

    if (target.source === 'backend' && target.sessionId > 0 && target.promptId > 0) {
        state.aiSelectedPrompt = target;
        updateAIPromptJumpAvailability();
        try {
            // Showing history: suspend live emits so a running agent can't append here.
            setAIPanelLive(false);
            const blocks = typeof ListAIStreamBlocks === 'function'
                ? await ListAIStreamBlocks(target.sessionId, target.promptId)
                : [];
            if (Array.isArray(blocks) && blocks.length > 0 && typeof GetAIStreamBlockContent === 'function') {
                await renderAIStoredPrompt(blocks, target);
            } else {
                aiPipelineFormatter.clear();
            }
        } catch (err) {
            console.error('Failed to load AI prompt log:', err);
        }
        return;
    }
}

function openAIPromptJumpMenu() {
    if (!elements.toolsAIPromptJump) {
        return;
    }

    const targets = Array.isArray(state.aiPromptJumpTargets) ? state.aiPromptJumpTargets : [];

    const menuTargets = [
        ...(state.aiRunActive ? [{ source: 'live', summary: 'Live output' }] : []),
        ...[...targets].reverse(),
    ];
    const prefix = (target) => {
        if (target.source === 'live' || target.status === 'running') return '💭';
        if (target.status === 'failed') return '‼️';
        if (target.status === 'cancelled') return '🚫';
        return '✅';
    };
    const selected = (target) => {
        const current = state.aiSelectedPrompt;
        return current?.source === target.source
            && (target.source === 'live'
                || (Number(current?.sessionId) === Number(target.sessionId)
                    && Number(current?.promptId) === Number(target.promptId)));
    };
    const options = menuTargets.map((target) => `${prefix(target)} ${target.summary}`);
    const icons = menuTargets.map((target) => selected(target) ? CONTEXT_ICON_TICK : 0x20);
    const rect = elements.toolsAIPromptJump.getBoundingClientRect();
    showLocalMenu({
        title: 'Prompts',
        options,
        icons,
        x: rect.left,
        y: rect.bottom,
        showNextToMouseCursor: true,
        onSelect: (index) => {
            if (typeof index !== 'number' || index < 0 || index >= menuTargets.length) {
                return;
            }
            void jumpToAIPromptTarget(menuTargets[index]);
        },
    });
}

function activateAISession(sessionId) {
    const id = Number(sessionId) || 0;
    if (id <= 0) {
        return;
    }

    SetActiveAISession(id).then((sessionState) => {
        applyAISessionManagement(sessionState);
        renderAISessionManagement();
        void loadAISessionCache('');
    }).catch((err) => {
        notifyTerminal('Failed to switch AI session', 'error');
        console.error(err);
    });
}

async function createNewAISession() {
    const summary = await openTextPrompt({
        title: 'New AI session',
        confirmLabel: 'Create',
        placeholder: 'Optional session name',
    });
    if (summary === null) {
        return;
    }

    CreateAISession(summary).then((sessionState) => {
        applyAISessionManagement(sessionState);
        renderAISessionManagement();
        void loadAISessionCache('');
        notifyTerminal('AI session created', 'info');
    }).catch((err) => {
        notifyTerminal('Failed to create AI session', 'error');
        console.error(err);
    });
}

async function openAISessionMenu() {
    if (!elements.toolsAISession) {
        return;
    }

    try {
        applyAISessionManagement(await GetAISessionManagement());
    } catch (err) {
        notifyTerminal('Failed to load AI sessions', 'error');
        console.error(err);
        return;
    }

    const sessions = Array.isArray(state.aiSessionManagement?.sessions)
        ? state.aiSessionManagement.sessions
        : [];
    const items = [{
        title: 'New session',
        icon: CONTEXT_ICON_ADD,
        onSelect: createNewAISession,
    }];

    if (sessions.length > 0) {
        items.push({ title: '-', icon: 0 });
    }

    items.push(...sessions.map((session) => {
        const sessionId = Number(session?.tableId) || 0;
        const isActive = session?.active === true || sessionId === state.aiSessionManagement.activeSessionId;
        return {
            title: String(session?.summary || `Session ${sessionId}`),
            icon: isActive ? CONTEXT_ICON_TICK : 0,
            onSelect: () => activateAISession(sessionId),
        };
    }));

    const rect = elements.toolsAISession.getBoundingClientRect();
    showLocalMenu({
        title: 'Sessions',
        options: items.map((item) => item.title),
        icons: items.map((item) => item.icon),
        x: rect.left,
        y: rect.bottom,
        showNextToMouseCursor: true,
        onSelect: (index) => items[index]?.onSelect?.(),
    });
}

function initAIPromptJumpObserver() {
    if (!elements.aiOutput || aiPromptJumpObserver) {
        return;
    }

    aiPromptJumpObserver = new MutationObserver(() => {
        scheduleAIPromptJumpRefresh();
    });

    aiPromptJumpObserver.observe(elements.aiOutput, {
        // No characterData: streaming rewrites text nodes constantly and each
        // change would queue a MutationRecord. Structural changes are enough to
        // spot new prompts, and the stream path also refreshes explicitly.
        childList: true,
        subtree: true,
    });

    scheduleAIPromptJumpRefresh();
}

// Renders an ISO timestamp as 'yyyy-mm-dd hh:mm' (or 'yyyy-mm-dd' without time),
// in local time. Returns '' for anything unparseable.
function formatSessionTimestamp(iso, includeTime) {
    if (!iso) {
        return '';
    }
    const date = new Date(iso);
    if (Number.isNaN(date.getTime())) {
        return '';
    }
    const pad = (n) => String(n).padStart(2, '0');
    const datePart = `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
    if (!includeTime) {
        return datePart;
    }
    return `${datePart} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function renderAISessionList() {
    if (!elements.aiSettingsSessionsList) {
        return;
    }

    const sessions = Array.isArray(state.aiSessionManagement?.sessions) ? state.aiSessionManagement.sessions : [];
    if (sessions.length === 0) {
        elements.aiSettingsSessionsList.textContent = 'No sessions yet for this workspace.';
        elements.aiSettingsSessionsList.dataset.empty = 'true';
        return;
    }

    elements.aiSettingsSessionsList.dataset.empty = 'false';
    const items = sessions.map((session) => {
        const tableId = Number(session.tableId) || 0;
        const safeSummary = escapeHtml(session.summary || `Session ${tableId}`);
        const count = Number(session.entryCount) || 0;
        const active = session.active === true;
        const latestDateTime = formatSessionTimestamp(session.updated, true);
        const createdDate = formatSessionTimestamp(session.created, false);
        const latestHtml = latestDateTime
            ? `<div class="notes-ai-settings-session-meta">Latest: ${escapeHtml(latestDateTime)}${createdDate ? ` (${escapeHtml(createdDate)})` : ''}</div>`
            : '';
        return `
            <div class="notes-ai-settings-session-item" data-session-id="${tableId}" data-active="${active ? 'true' : 'false'}">
                <div class="notes-ai-settings-session-main">
                    <div class="notes-ai-settings-session-heading">
                        <span class="notes-ai-settings-session-count">${count}</span>
                        <span class="notes-ai-settings-session-name">${safeSummary}</span>
                    </div>
                    ${latestHtml}
                </div>
                <div class="notes-ai-settings-session-actions">
                    <button type="button" class="notes-ai-settings-session-rename" data-action="rename" data-session-id="${tableId}" aria-label="Rename session"></button>
                    <button type="button" class="notes-ai-settings-session-delete" data-action="delete" data-session-id="${tableId}" aria-label="Delete session"></button>
                </div>
            </div>
        `;
    }).join('');

    elements.aiSettingsSessionsList.innerHTML = items;
}

function renderAISessionHistory() {
    if (!elements.aiSettingsHistoryList) {
        return;
    }

    const sessions = Array.isArray(state.aiSessionManagement?.history) ? state.aiSessionManagement.history : [];
    if (sessions.length === 0) {
        elements.aiSettingsHistoryList.textContent = 'No history yet for the active session.';
        elements.aiSettingsHistoryList.dataset.empty = 'true';
        return;
    }

    elements.aiSettingsHistoryList.dataset.empty = 'false';
    const items = sessions.slice(0, 12).map((entry) => {
        const entryId = Number(entry.id) || 0;
        const safeTitle = escapeHtml(entry.prompt || 'AI prompt');
        const safeCmdLine = escapeHtml(entry.commandLine || '');
        const safeExcerpt = escapeHtml(entry.excerpt || '');
        // Output can contain real newlines; flatten to a single flowing line
        // like the excerpt already is, since both are clamped to 2 lines.
        const safeOutput = escapeHtml((entry.outputBlock || '').replace(/\s*\n+\s*/g, ' ').trim());
        const cmdLineHtml = safeCmdLine ? `<div class="notes-ai-settings-history-meta">${safeCmdLine}</div>` : '';
        const outputHtml = safeOutput ? `<div class="notes-ai-settings-history-output">${safeOutput}</div>` : '';
        const excerptHtml = safeExcerpt ? `<div class="notes-ai-settings-history-excerpt">${safeExcerpt}</div>` : '';
        return `
            <div class="notes-ai-settings-history-item" data-entry-id="${entryId}">
                <div class="notes-ai-settings-history-main">
                    <div class="notes-ai-settings-history-heading">${safeTitle}</div>
                    ${cmdLineHtml}
                    ${outputHtml}
                    ${excerptHtml}
                </div>
                <button type="button" class="notes-ai-settings-history-erase" data-action="erase" data-entry-id="${entryId}" aria-label="Erase this prompt" title="Erase this prompt"></button>
            </div>
        `;
    }).join('');

    elements.aiSettingsHistoryList.innerHTML = items;
}

function renderAIToolsList() {
    if (!elements.aiSettingsToolsList) {
        return;
    }

    const allTools = Array.isArray(state.aiToolsList) ? state.aiToolsList : [];
    if (allTools.length === 0) {
        elements.aiSettingsToolsList.textContent = 'No tools available.';
        elements.aiSettingsToolsList.dataset.empty = 'true';
        return;
    }

    const query = String(elements.aiSettingsToolsFilter?.value || '').trim().toLowerCase();
    const tools = allTools.map((tool, index) => ({ tool, index }))
        .filter(({ tool }) => `${tool.name || ''}\n${tool.description || ''}`.toLowerCase().includes(query));
    if (tools.length === 0) {
        elements.aiSettingsToolsList.textContent = 'No tools match this filter.';
        elements.aiSettingsToolsList.dataset.empty = 'true';
        return;
    }
    elements.aiSettingsToolsList.dataset.empty = 'false';
    const items = tools.map(({ tool, index }) => {
        const safeName = escapeHtml(tool.name || 'Unknown tool');
        const toolState = String(tool.state || (tool.enabled === true ? 'always' : 'disabled'));
        const state = toolStateDisplay(toolState);
        const subagent = subagentAccessDisplay(tool.allowInSubagent === true);
        return `
            <div class="notes-ai-settings-tool-item" data-tool-index="${index}">
                <button type="button" class="notes-ai-settings-tool-state" data-tool-name="${safeName}" data-tool-state="${state.value}" title="${state.label}" aria-label="${state.label}">${state.letter}</button>
            <button type="button" class="notes-ai-settings-tool-subagent" data-tool-name="${safeName}" data-subagent-access="${subagent.value}" title="${subagent.label}" aria-label="${subagent.label}">${subagent.letter}</button>
                <button type="button" class="notes-ai-settings-tool-name" aria-label="Show details for ${safeName}">${safeName}</button>
            </div>
        `;
    }).join('');

    elements.aiSettingsToolsList.innerHTML = items;
    elements.aiSettingsToolsList.querySelectorAll('.notes-ai-settings-tool-name').forEach((button) => {
        button.addEventListener('click', (event) => {
            event.stopPropagation();
            const item = button.closest('.notes-ai-settings-tool-item');
            const index = Number(item?.dataset.toolIndex);
            if (Number.isInteger(index) && index >= 0) {
                void openAIToolMetadataModal(index);
            }
        });
    });
}

function toolStateDisplay(state) {
    return ({
        always: { value: 'always', letter: 'A', label: 'Enabled: Always allow' },
        session: { value: 'session', letter: 'S', label: 'Enabled: Current session' },
        approval: { value: 'approval', letter: 'P', label: 'Enabled: Ask for permission' },
        disabled: { value: 'disabled', letter: 'D', label: 'Disabled' },
    })[state] || { value: 'disabled', letter: 'D', label: 'Disabled' };
}

function subagentAccessDisplay(allowed) {
    return allowed
        ? { value: 'allow', letter: 'Y', label: 'Allow in subagents: Yes' }
        : { value: 'deny', letter: 'N', label: 'Allow in subagents: No' };
}

function getMcpServerDisplayName(server) {
    const name = String(server?.name || '').trim() || 'Unknown MCP server';
    const source = String(server?.source || '').trim();
    const sourceLabel = source ? source.split('/').pop() : 'unknown source';
    return `${name} (${sourceLabel})`;
}

function renderAIMcpServersList() {
    if (!elements.aiSettingsMcpList) {
        return;
    }

    const allServers = Array.isArray(state.aiMcpServersList) ? state.aiMcpServersList : [];
    if (allServers.length === 0) {
        elements.aiSettingsMcpList.textContent = 'No MCP servers found in config.';
        elements.aiSettingsMcpList.dataset.empty = 'true';
        return;
    }

    const query = String(elements.aiSettingsMcpFilter?.value || '').trim().toLowerCase();
    const servers = allServers.map((server, index) => ({ server, index }))
        .filter(({ server }) => `${getMcpServerDisplayName(server)}\n${server.source || ''}`.toLowerCase().includes(query));
    if (servers.length === 0) {
        elements.aiSettingsMcpList.textContent = 'No MCP servers match this filter.';
        elements.aiSettingsMcpList.dataset.empty = 'true';
        return;
    }
    elements.aiSettingsMcpList.dataset.empty = 'false';
    const items = servers.map(({ server, index }) => {
        const safeName = escapeHtml(getMcpServerDisplayName(server));
        const safeKey = escapeHtml(String(server.key || ''));
        const checked = server.loaded === true;
        const loadable = server.loadable !== false;
        const disabledAttr = loadable ? '' : 'disabled';
        const itemClass = loadable ? 'notes-ai-settings-tool-item' : 'notes-ai-settings-tool-item notes-ai-settings-tool-item-disabled';

        return `
            <div class="${itemClass}" data-mcp-index="${index}">
                <input type="checkbox" class="notes-ai-settings-tool-checkbox" data-mcp-key="${safeKey}" ${checked ? 'checked' : ''} ${disabledAttr}>
                <span class="notes-ai-settings-tool-name">${safeName}</span>
            </div>
        `;
    }).join('');

    elements.aiSettingsMcpList.innerHTML = items;
}

function formatAIToolMetadataMarkdown(tool) {
    const name = String(tool?.name || '').trim() || 'Unknown tool';
    const description = String(tool?.description || '').trim() || 'No description available.';
    const allowInSubagent = tool?.allowInSubagent === true;

    let schema = String(tool?.schema || '').trim();
    if (!schema) {
        schema = '{}';
    }

    return `## ${name}\n\n- [${allowInSubagent ? 'x' : ' '}] Allow in sub-agent\n\n${description}\n\n---\n\n### JSON Schema\n\n\`\`\`json\n${schema}\n\`\`\``;
}

function formatAIMcpServerMetadataMarkdown(server) {
    const name = getMcpServerDisplayName(server);
    const loaded = server?.loaded === true;
    const loadable = server?.loadable !== false;
    const source = String(server?.source || '').trim() || 'Unknown source';
    const serverType = String(server?.serverType || server?.type || 'command').trim();
    const loadedFrom = String(server?.loadedFrom || '').trim();
    const config = String(server?.config || '{}').trim() || '{}';

    const conflict = (!loadable && loadedFrom)
        ? `\n\n> This server name is already loaded from:\n> ${loadedFrom}`
        : '';

    return `## ${name}\n\n- [${loaded ? 'x' : ' '}] Loaded\n- Source: ${source}\n- Type: ${serverType}${conflict}\n\n### Config\n\n\`\`\`json\n${config}\n\`\`\``;
}

async function openAIToolMetadataModal(index) {
    if (!elements.aiToolMetaModal || !elements.aiToolMetaCard || !elements.aiToolMetaContent) {
        return;
    }

    const tools = Array.isArray(state.aiToolsList) ? state.aiToolsList : [];
    const tool = tools[index];
    if (!tool) {
        return;
    }

    const markdown = formatAIToolMetadataMarkdown(tool);
    elements.aiToolMetaContent.innerHTML = marked.parse(markdown);
    await processMarkdownContainer(elements.aiToolMetaContent);

    const stateButton = document.createElement('button');
    stateButton.type = 'button';
    stateButton.className = 'notes-ai-settings-tool-state';
    stateButton.dataset.toolName = String(tool.name || '');
    stateButton.dataset.toolIndex = String(index);
    const toolState = String(tool.state || (tool.enabled === true ? 'always' : 'disabled'));
    const displayState = toolStateDisplay(toolState);
    stateButton.dataset.toolState = displayState.value;
    stateButton.textContent = displayState.letter;
    stateButton.title = displayState.label;
    stateButton.setAttribute('aria-label', displayState.label);
    elements.aiToolMetaContent.insertBefore(stateButton, elements.aiToolMetaContent.firstChild?.nextSibling || null);

    const allowCheckbox = elements.aiToolMetaContent.querySelector('input[type="checkbox"]');
    if (allowCheckbox instanceof HTMLInputElement) {
        allowCheckbox.disabled = false;
        allowCheckbox.checked = tool.allowInSubagent === true;
        allowCheckbox.dataset.configType = 'subagent-tool';
        allowCheckbox.dataset.toolIndex = String(index);
        allowCheckbox.dataset.toolName = String(tool.name || '');
        allowCheckbox.setAttribute('aria-label', 'Allow in sub-agent');
    }

    elements.aiToolMetaModal.dataset.open = 'true';
    elements.aiToolMetaModal.setAttribute('aria-hidden', 'false');
    elements.aiToolMetaCard.focus();
}

async function openAIMcpServerMetadataModal(index) {
    if (!elements.aiToolMetaModal || !elements.aiToolMetaCard || !elements.aiToolMetaContent) {
        return;
    }

    const servers = Array.isArray(state.aiMcpServersList) ? state.aiMcpServersList : [];
    const server = servers[index];
    if (!server) {
        return;
    }

    const markdown = formatAIMcpServerMetadataMarkdown(server);
    elements.aiToolMetaContent.innerHTML = marked.parse(markdown);
    await processMarkdownContainer(elements.aiToolMetaContent);

    const enabledCheckbox = elements.aiToolMetaContent.querySelector('input[type="checkbox"]');
    if (enabledCheckbox instanceof HTMLInputElement) {
        enabledCheckbox.disabled = server.loadable === false;
        enabledCheckbox.checked = server.loaded === true;
        enabledCheckbox.dataset.configType = 'mcp';
        enabledCheckbox.dataset.mcpIndex = String(index);
        enabledCheckbox.dataset.mcpKey = String(server.key || '');
        enabledCheckbox.setAttribute('aria-label', 'Loaded');
    }

    elements.aiToolMetaModal.dataset.open = 'true';
    elements.aiToolMetaModal.setAttribute('aria-hidden', 'false');
    elements.aiToolMetaCard.focus();
}

function closeAIToolMetadataModal() {
    if (!elements.aiToolMetaModal || !elements.aiToolMetaContent) {
        return;
    }

    elements.aiToolMetaModal.dataset.open = 'false';
    elements.aiToolMetaModal.setAttribute('aria-hidden', 'true');
    elements.aiToolMetaContent.innerHTML = '';

    void loadAISessionManagement().then(() => {
        renderAISessionManagement();
    });
}

async function refreshAIModelPicker() {
    if (!elements.aiSettingsModelPicker) {
        return;
    }

    try {
        const [options, current, limits] = await Promise.all([
            ListAIModelSelections(),
            GetCurrentAIModelSelection(),
            GetAIExecutionLimits(),
        ]);

        state.aiModelSelections = Array.isArray(options) ? options : [];
        state.aiCurrentModelSelection = String(current || '').trim();

        const fallback = state.aiModelSelections[0] || 'Model';
        elements.aiSettingsModelPicker.textContent = state.aiCurrentModelSelection || fallback;
        elements.aiSettingsModelPicker.disabled = state.aiModelSelections.length === 0;
        renderAIExecutionLimits(limits);
    } catch (err) {
        console.error('Failed to refresh AI model picker:', err);
        elements.aiSettingsModelPicker.textContent = 'Model';
        elements.aiSettingsModelPicker.disabled = true;
        renderAIExecutionLimits({});
    }
}

function renderAIExecutionLimits(limits) {
    if (!elements.aiSettingsExecutionLimits) {
        return;
    }

    const value = (key) => limits && limits[key] !== null && limits[key] !== undefined && limits[key] !== ''
        ? String(limits[key])
        : 'Unknown';
    elements.aiSettingsExecutionLimits.innerHTML = [
        ['Agent step budget', value('agentSteps')],
        ['Provider max iterations', value('providerMaxIterations')],
        ['Model context window', value('modelContextWindow')],
        ['Model max output tokens', value('modelMaxOutputTokens')],
        ['Request timeout', value('requestTimeout')],
    ].map(([label, text]) => `<dt>${label}</dt><dd>${text}</dd>`).join('');
}

function openAIModelPickerMenu() {
    if (!elements.aiSettingsModelPicker) {
        return;
    }

    const options = Array.isArray(state.aiModelSelections) ? state.aiModelSelections : [];
    if (options.length === 0) {
        return;
    }

    const current = String(state.aiCurrentModelSelection || '').trim();
    const icons = options.map((option) => String(option || '').trim() === current ? CONTEXT_ICON_TICK : 0x20);

    const rect = elements.aiSettingsModelPicker.getBoundingClientRect();
    showLocalMenu({
        title: 'Model',
        options,
        icons,
        x: rect.left,
        y: rect.bottom,
        showNextToMouseCursor: true,
        onSelect: (index) => {
            const selected = options[index];
            if (typeof selected !== 'string' || selected.trim() === '') {
                return;
            }

            SetCurrentAIModelSelection(selected).then(() => {
                state.aiCurrentModelSelection = selected;
                elements.aiSettingsModelPicker.textContent = selected;
            }).catch((err) => {
                notifyTerminal('Failed to update AI model', 'error');
                console.error(err);
                void refreshAIModelPicker();
            });
        },
    });
}

async function askAIFromToolbar() {
    setToolsPanelCollapsed(false);
    setToolsTab('ai');
    setAIPanelLive(true);

    try {
        await AskAI('notesPromptToolbar', '', '');
    } catch (err) {
        notifyTerminal('Failed to ask AI', 'error');
        console.error(err);
    }
}

async function askAISkillsFromToolbar() {
    setToolsPanelCollapsed(false);
    setToolsTab('ai');

    try {
        const rect = elements.toolsAISkills.getBoundingClientRect();
        await ShowAISkillsMenu(rect.left, rect.bottom + 4);
    } catch (err) {
        notifyTerminal('Failed to show AI skills', 'error');
        console.error(err);
    }
}

function createAskAIDocumentMenuItem() {
    return {
        title: 'Ask AI...',
        icon: CONTEXT_ICON_ASK_AI,
        onSelect: () => {
            void askAIAboutCurrentDocument();
        },
    };
}

function showNotesLocalMenu(menuItems, x, y, title = 'Select an action', onHighlight = null, onCancel = null) {
    showLocalMenu({
        title,
        options: menuItems.map((item) => item.title),
        icons: menuItems.map((item) => item.icon),
        x,
        y,
        showNextToMouseCursor: true,
        onSelect: (index) => {
            const item = menuItems[index];
            if (item && typeof item.onSelect === 'function') {
                item.onSelect();
            }
        },
        onHighlight: onHighlight || null,
        onCancel: onCancel || null,
    });
}

function extractTableData(table, options = {}) {
    const getCellContent = (cell) => {
        const source = cell.querySelector('.notes-table-cell-wrap > span:first-child') || cell;
        const clone = source.cloneNode(true);
        clone.querySelectorAll('.notes-sort-icon, .notes-table-col-resize-handle, .notes-cellref').forEach((node) => node.remove());

        const breaks = [];
        clone.querySelectorAll('br').forEach((lineBreak, index) => {
            const marker = `\uE000NOTES_BR_${index}\uE001`;
            breaks.push(marker);
            lineBreak.replaceWith(document.createTextNode(marker));
        });

        let text = String(clone.textContent || '').trim();
        breaks.forEach((marker) => {
            text = text.replaceAll(marker, options.preserveBreakTags ? '<br/>' : '\n');
        });
        return text;
    };

    const rows = [];
    const headerRow = table.querySelector('thead tr');
    if (headerRow) {
        rows.push(Array.from(headerRow.querySelectorAll('th, td')).map(cell => getCellContent(cell)));
    }
    Array.from(table.querySelectorAll('tbody tr:not(.notes-table-row-filtered)')).forEach(tr => {
        rows.push(Array.from(tr.querySelectorAll('td, th')).map(cell => getCellContent(cell)));
    });
    return rows;
}

function tableDataToCsv(rows) {
    return (rows || []).map(row => (row || []).map(field => escapeCsvField(field)).join(',')).join('\n');
}

function tableDataToMarkdown(rows) {
    if (!rows || rows.length === 0) return '';
    const header = rows[0];
    const body = rows.slice(1);
    const headerLine = `| ${header.join(' | ')} |`;
    const separatorLine = `| ${header.map(() => '---').join(' | ')} |`;
    const bodyLines = body.map(row => `| ${row.join(' | ')} |`);
    return [headerLine, separatorLine, ...bodyLines].join('\n');
}

function createTableCopyMenuItems(table) {
    return [
        {
            title: 'Copy table (CSV)',
            icon: CONTEXT_ICON_COPY,
            onSelect: () => {
                const data = extractTableData(table);
                copyTextToClipboard(tableDataToCsv(data));
            },
        },
        {
            title: 'Copy table (Markdown)',
            icon: CONTEXT_ICON_COPY,
            onSelect: () => {
                const data = extractTableData(table, { preserveBreakTags: true });
                copyTextToClipboard(tableDataToMarkdown(data));
            },
        },
        {
            title: 'Copy table (Formatted)',
            icon: CONTEXT_ICON_COPY,
            onSelect: () => {
                void renderedSelectionCopier.copy(captureRenderedElement(table), CopyFormattedTable);
            },
        },
    ];
}

function getCellPosition(target, table) {
    const cell = target.closest('td, th');
    if (!cell || !table.contains(cell)) return null;
    const tr = cell.parentElement;
    const colIndex = Array.from(tr.children).indexOf(cell);
    const isHeader = tr.parentElement && tr.parentElement.tagName === 'THEAD';
    if (isHeader) return { row: 0, col: colIndex };
    const bodyRows = Array.from(table.querySelectorAll('tbody tr'));
    const rowOffset = bodyRows.indexOf(tr);
    return rowOffset >= 0 ? { row: rowOffset + 1, col: colIndex } : null;
}

function createTableInsertMenuItems(table, target, tableIndex) {
    const pos = target instanceof Element ? getCellPosition(target, table) : null;
    if (!pos) return [];

    const isCsv = state.currentFileType === 'csv';

    return [
        {
            title: 'Insert row (after)',
            icon: 0xf0ab,
            onSelect: () => {
                if (isCsv) {
                    insertCsvRowAfter(pos.row);
                } else {
                    const blocks = findMarkdownTableBlocks(getMainEditorValue());
                    const block = blocks[tableIndex];
                    if (block) insertMarkdownRowAfter(block, pos.row);
                }
            },
        },
        {
            title: 'Insert column (after)',
            icon: 0xf0a9,
            onSelect: () => {
                if (isCsv) {
                    insertCsvColumnAfter(pos.col);
                } else {
                    const blocks = findMarkdownTableBlocks(getMainEditorValue());
                    const block = blocks[tableIndex];
                    if (block) insertMarkdownColumnAfter(block, pos.col);
                }
            },
        },
        { title: '-' },
        {
            title: 'Delete row',
            icon: 0xf057,
            onSelect: () => {
                if (isCsv) {
                    deleteCsvRow(pos.row);
                } else {
                    const blocks = findMarkdownTableBlocks(getMainEditorValue());
                    const block = blocks[tableIndex];
                    if (block) deleteMarkdownRow(block, pos.row);
                }
            },
        },
        {
            title: 'Delete column',
            icon: 0xf057,
            onSelect: () => {
                if (isCsv) {
                    deleteCsvColumn(pos.col);
                } else {
                    const blocks = findMarkdownTableBlocks(getMainEditorValue());
                    const block = blocks[tableIndex];
                    if (block) deleteMarkdownColumn(block, pos.col);
                }
            },
        },
    ];
}

function initAIOutputContextMenu(container) {
    if (container) container.dataset.copyRendered = 'true';
    if (!container) {
        return;
    }

    if (container.dataset.notesLinkHoverBound !== 'true') {
        container.dataset.notesLinkHoverBound = 'true';
        container.addEventListener('mouseover', (e) => {
            const anchor = e.target instanceof Element ? e.target.closest('a[href]') : null;
            if (anchor && container.contains(anchor)) {
                showHyperlinkHoverTooltip(anchor, e.clientX, e.clientY);
            }
        });
        container.addEventListener('mousemove', (e) => {
            const anchor = e.target instanceof Element ? e.target.closest('a[href]') : null;
            if (anchor && container.contains(anchor)) {
                const href = String(anchor.href || anchor.getAttribute('href') || '').trim();
                const displayHref = formatHyperlinkHoverHref(href);
                if (displayHref) {
                    showHyperlinkHoverTooltip(anchor, e.clientX, e.clientY);
                } else {
                    positionHyperlinkHoverTooltip(e.clientX, e.clientY);
                }
                return;
            }
            hideHyperlinkHoverTooltip();
        });
        container.addEventListener('mouseout', (e) => {
            const leavingAnchor = e.target instanceof Element ? e.target.closest('a[href]') : null;
            if (!leavingAnchor || !container.contains(leavingAnchor)) {
                return;
            }
            const related = e.relatedTarget;
            if (related instanceof Element) {
                const nextAnchor = related.closest('a[href]');
                if (nextAnchor && container.contains(nextAnchor)) {
                    return;
                }
            }
            hideHyperlinkHoverTooltip();
        });
        container.addEventListener('mousedown', () => {
            hideHyperlinkHoverTooltip();
        });
    }

    initRenderedBlockCopyButtons(container);

    container.addEventListener('contextmenu', (e) => {
        const sectionHeading = getRenderedSectionHeading(container, e.target, e.clientY);
        const anchor = e.target instanceof Element ? e.target.closest('a[href]') : null;
        if (anchor && container.contains(anchor) && !sectionHeading && !hasRenderedSelection(container)) {
            e.preventDefault();
            e.stopPropagation();
            openHyperlinkContextMenu(anchor);
            return;
        }

        // Images carry their own menu, bound per element.
        if (e.target instanceof Element && e.target.closest('img') && !hasRenderedSelection(container)) {
            return;
        }

        e.preventDefault();

        const table = e.target instanceof Element ? e.target.closest('table') : null;
        const codeBlockText = getRenderedCodeBlockText(container, e.target);
        const codeBlockLanguage = getRenderedCodeBlockLanguage(container, e.target);
        const aiBlock = e.target instanceof Element ? e.target.closest('[data-block-kind]') : null;
        const isFinalAIResponse = aiBlock?.dataset.blockKind === 'text'
            && aiBlock?.dataset.blockStatus === 'closed';
        const copyRoot = e.target instanceof Element ? e.target.closest('.notes-ai-job') : null;
        const tableItems = table && container.contains(table)
            ? [
                ...createTableCopyMenuItems(table),
                { title: '-' },
                createClearTableSortMenuItem(table),
                createTableFilterMenuItem(table),
                createTableGraphMenuItem(table),
                { title: '-' },
            ]
            : [];

        const wordWrapItems = (table && container.contains(table))
            ? [{
                title: 'Word wrap table contents',
                icon: state.markdownTableWordWrapMode ? 0xf00c : 0x20,
                onSelect: () => {
                    state.markdownTableWordWrapMode = !state.markdownTableWordWrapMode;
                    applyNotesTableWordWrapMode(elements.preview);
                    applyNotesTableWordWrapMode(elements.jupyter);
                    applyNotesTableWordWrapMode(elements.aiOutput);
                },
            }, { title: '-' }]
            : [];

        const copyItems = codeBlockText
            ? [
                createRenderedCopyMenuItem(container),
                createCopyMenuItem(() => codeBlockText, 'Copy code'),
                { title: '-' },
            ]
            : [
                createRenderedCopyMenuItem(container),
                { title: '-' },
            ];
        copyItems.splice(copyItems.length - 1, 0,
            {
                title: 'Copy prompt',
                icon: CONTEXT_ICON_COPY,
                onSelect: () => copyAIMessage('prompt', copyRoot),
            },
            {
                title: 'Copy response',
                icon: CONTEXT_ICON_COPY,
                onSelect: () => {
                    const root = copyRoot || Array.from(container.querySelectorAll('.notes-ai-job')).at(-1);
                    return copyAIMessage('response', root);
                },
            },
        );
        const codeSaveItems = codeBlockText && isFinalAIResponse
            ? [createSaveCodeBlockMenuItem(codeBlockText, codeBlockLanguage), { title: '-' }]
            : [];

        const menuItems = [
            ...copyItems,
            ...codeSaveItems,
            ...(sectionHeading ? [createCollapseSectionMenuItem(sectionHeading), { title: '-' }] : []),
            ...(anchor ? [{ title: 'Link actions', onSelect: () => openHyperlinkContextMenu(anchor) }, { title: '-' }] : []),
            ...tableItems,
            ...wordWrapItems,
            {
                title: 'Print...',
                icon: CONTEXT_ICON_PRINT,
                onSelect: () => {
                    void (async () => {
                        const source = await aiPipelineFormatter.outputMarkdown();
                        if (!source.trim()) {
                            notifyTerminal('No AI output to print', 'info');
                            return;
                        }
                        await printDocument({
                            source,
                            renderMarkdown: renderPrintMarkdown,
                            dismissMenu: closeSharedTooltip,
                            print: WindowPrint,
                        });
                    })().catch(reportPrintError);
                },
            },
        ];

        showNotesLocalMenu(menuItems, e.clientX, e.clientY);
    });
}

function pinAIScrollableBlocksToBottom(container, options = {}) {
    // Tool output and reasoning stream in, so keep the clamped fenced and quote
    // blocks showing their latest lines.
    const blocks = container.querySelectorAll('pre, blockquote');
    if (options.lastOnly) {
        // Only the final block is still growing; the rest were pinned when their
        // batch was committed, and each read/write here forces a layout.
        const last = blocks[blocks.length - 1];
        if (last) {
            last.scrollTop = last.scrollHeight;
        }
        return;
    }
    for (const block of blocks) {
        block.scrollTop = block.scrollHeight;
    }
}

async function processAIMarkdownContainer(container, options = {}) {
    if (options.streaming) {
        // This subtree is re-rendered every frame while streaming, so only run
        // the cheap DOM passes. Mermaid rendering, image loading (a Go IPC round
        // trip per image), table wiring and auto-hyperlinking would all be
        // repeated and discarded; finishJob applies them once at the end.
        processLinks(container, { enableBookmarks: true });
        pinAIScrollableBlocksToBottom(container, { lastOnly: true });
        return;
    }

    void setupTableColumnResizing(container, state.markdownTableWordWrapMode, '');
    // AI panel code blocks stay unhighlighted; they're mostly tool output, not source.
    await processMarkdownContainer(container, { syntaxHighlighting: false });
    // Finalized AI Markdown is split into streaming batches. Join their DOM
    // siblings before grouping sections, without changing the stored Markdown.
    for (const wrapper of container.querySelectorAll('.notes-ai-stable, .notes-ai-batch, .notes-ai-tail')) {
        wrapper.replaceWith(...wrapper.childNodes);
    }
    setupCollapsibleHeadings(container);
    enableImageContextMenus(container);
    wrapTablesForHorizontalScroll(container);
    setupNotesTableQueries(container, 'ai', '');
    applyNotesTableWordWrapMode(container);
    pinAIScrollableBlocksToBottom(container);
}

function initRenderedNotesContextMenu(container, viewMode) {
    if (container) container.dataset.copyRendered = 'true';
    if (container && container.dataset.notesLinkHoverBound !== 'true') {
        container.dataset.notesLinkHoverBound = 'true';

        container.addEventListener('mouseover', (e) => {
            const anchor = e.target instanceof Element ? e.target.closest('a[href]') : null;
            if (anchor && container.contains(anchor)) {
                showHyperlinkHoverTooltip(anchor, e.clientX, e.clientY);
            }
        });

        container.addEventListener('mousemove', (e) => {
            const anchor = e.target instanceof Element ? e.target.closest('a[href]') : null;
            if (anchor && container.contains(anchor)) {
                const href = String(anchor.href || anchor.getAttribute('href') || '').trim();
                const displayHref = formatHyperlinkHoverHref(href);
                if (displayHref) {
                    showHyperlinkHoverTooltip(anchor, e.clientX, e.clientY);
                } else {
                    positionHyperlinkHoverTooltip(e.clientX, e.clientY);
                }
                return;
            }

            hideHyperlinkHoverTooltip();
        });

        container.addEventListener('mouseout', (e) => {
            const leavingAnchor = e.target instanceof Element ? e.target.closest('a[href]') : null;
            if (!leavingAnchor || !container.contains(leavingAnchor)) {
                return;
            }

            const related = e.relatedTarget;
            if (related instanceof Element) {
                const nextAnchor = related.closest('a[href]');
                if (nextAnchor && container.contains(nextAnchor)) {
                    return;
                }
            }

            hideHyperlinkHoverTooltip();
        });

        container.addEventListener('mousedown', () => {
            hideHyperlinkHoverTooltip();
        });
    }

    initRenderedBlockCopyButtons(container);

    container.addEventListener('dblclick', (e) => {
        if (viewMode !== 'jupyter' || state.viewMode !== 'jupyter' || state.currentFileType !== 'markdown') {
            return;
        }
        const block = e.target instanceof Element
            ? e.target.closest('[data-run-markdown-source-start]')
            : null;
        if (!block || !container.contains(block)) {
            return;
        }
        e.preventDefault();
        e.stopPropagation();
        beginMarkdownRunBlockEdit(container, block);
    });

    container.addEventListener('contextmenu', (e) => {
        const sectionHeading = getRenderedSectionHeading(container, e.target, e.clientY);
        const runMarkdownBlock = viewMode === 'jupyter'
            && state.currentFileType === 'markdown'
            && container === elements.jupyter
            && e.target instanceof Element
            ? e.target.closest('[data-run-markdown-source-start]')
            : null;
        const anchor = e.target instanceof Element ? e.target.closest('a[href]') : null;
        if (anchor && container.contains(anchor) && !runMarkdownBlock && !sectionHeading && !hasRenderedSelection(container)) {
            e.preventDefault();
            e.stopPropagation();
            openHyperlinkContextMenu(anchor);
            return;
        }

        if (e.target instanceof Element && e.target.closest('img') && !hasRenderedSelection(container)) {
            return;
        }

        e.preventDefault();

        const table = e.target instanceof Element ? e.target.closest('table') : null;
        const codeBlockText = getRenderedCodeBlockText(container, e.target);
        const codeBlockLanguage = getRenderedCodeBlockLanguage(container, e.target);
        const isRunMode = state.viewMode === 'jupyter';
        const tableIndex = table ? Array.from(container.querySelectorAll('table')).indexOf(table) : -1;
        const tableItems = table && container.contains(table)
            ? [
                ...createTableCopyMenuItems(table),
                { title: '-' },
                createClearTableSortMenuItem(table),
                createTableFilterMenuItem(table),
                createTableGraphMenuItem(table),
                { title: '-' },
            ]
            : [];
        const insertItems = (table && isRunMode && container.contains(table))
            ? [...createTableInsertMenuItems(table, e.target, tableIndex), { title: '-' }]
            : [];
        const wordWrapItems = (table && container.contains(table) && (viewMode === 'viewer' || viewMode === 'jupyter'))
            ? [{
                title: 'Word wrap table contents',
                icon: state.markdownTableWordWrapMode ? 0xf00c : 0x20,
                onSelect: () => {
                    state.markdownTableWordWrapMode = !state.markdownTableWordWrapMode;
                    applyNotesTableWordWrapMode(elements.preview);
                    applyNotesTableWordWrapMode(elements.jupyter);
                },
            }, { title: '-' }]
            : [];

        const copyItems = codeBlockText
            ? [
                createRenderedCopyMenuItem(container),
                createCopyMenuItem(() => codeBlockText, 'Copy code'),
                { title: '-' },
            ]
            : [
                createRenderedCopyMenuItem(container, 'Copy'),
                { title: '-' },
            ];
        const editMarkdownItems = runMarkdownBlock
            ? [{
                title: 'Edit',
                icon: CONTEXT_ICON_EDIT,
                onSelect: () => beginMarkdownRunBlockEdit(container, runMarkdownBlock),
            }, { title: '-' }]
            : [];
        const codeSaveItems = codeBlockText && viewMode === 'viewer'
            ? [createSaveCodeBlockMenuItem(codeBlockText, codeBlockLanguage), { title: '-' }]
            : [];

        const allMenuItems = [
            ...copyItems,
            ...codeSaveItems,
            ...editMarkdownItems,
            ...(sectionHeading ? [createCollapseSectionMenuItem(sectionHeading), { title: '-' }] : []),
            ...(anchor ? [{ title: 'Link actions', onSelect: () => openHyperlinkContextMenu(anchor) }, { title: '-' }] : []),
            ...tableItems,
            ...wordWrapItems,
            ...insertItems,
            createFindMenuItem('Find'),
            createAskAIDocumentMenuItem(),
            createPrintMenuItem('Print'),
        ];

        showNotesLocalMenu(allMenuItems, e.clientX, e.clientY);
    });
}

function parseJsonNodePath(attr) {
    if (!attr) {
        return null;
    }
    try {
        const parsed = JSON.parse(attr);
        return Array.isArray(parsed) ? parsed : null;
    } catch {
        return null;
    }
}

function makeUniqueChildKey(parentObject) {
    const base = 'newKey';
    if (!parentObject || typeof parentObject !== 'object' || Array.isArray(parentObject)) {
        return base;
    }
    if (!Object.prototype.hasOwnProperty.call(parentObject, base)) {
        return base;
    }
    let suffix = 2;
    while (Object.prototype.hasOwnProperty.call(parentObject, `${base}${suffix}`)) {
        suffix += 1;
    }
    return `${base}${suffix}`;
}

async function addStructuredTreeKey(container, getRoot, parentPath) {
    const root = typeof getRoot === 'function' ? getRoot() : null;
    const parentObject = parentPath.length === 0 ? root : getValueAtPath(root, parentPath);
    const key = makeUniqueChildKey(parentObject);

    if (typeof container.__jsonViewerOnEditCommit === 'function') {
        await container.__jsonViewerOnEditCommit({ editType: 'addKey', path: parentPath, text: key });
    }

    // The viewer re-renders during commit; immediately start editing the new
    // key so the user can type its name.
    startJsonViewerKeyEdit(container, [...parentPath, key]);
}

function initStructuredDataTreeContextMenu(container, options = {}) {
    if (!container || container.dataset.jsonTreeContextMenuBound === 'true') {
        return;
    }

    container.dataset.jsonTreeContextMenuBound = 'true';

    const isActive = typeof options.isActive === 'function'
        ? options.isActive
        : () => state.viewMode === 'swagger-view';
    const getRoot = typeof options.getRoot === 'function' ? options.getRoot : () => null;
    const menuTitle = options.menuTitle || 'Document';

    container.addEventListener('contextmenu', (e) => {
        if (!isActive()) {
            return;
        }

        const targetEl = e.target instanceof Element ? e.target : null;
        if (!targetEl || !container.contains(targetEl)) {
            return;
        }

        const editable = targetEl.closest('.json-editable');
        const node = targetEl.closest('.json-node');
        if (!editable && !node) {
            return;
        }

        e.preventDefault();
        e.stopPropagation();

        const menuItems = [];

        if (editable) {
            menuItems.push(
                {
                    title: 'Copy',
                    icon: CONTEXT_ICON_COPY,
                    onSelect: () => {
                        copyTextToClipboard(getJsonEditableCopyText(editable));
                    },
                },
                {
                    title: 'Edit',
                    icon: CONTEXT_ICON_EDIT,
                    onSelect: () => {
                        editable.dispatchEvent(new MouseEvent('dblclick', {
                            bubbles: true,
                            cancelable: true,
                            view: window,
                        }));
                    },
                },
            );
        }

        const nodeType = node ? node.getAttribute('data-node-type') : null;
        const containerNode = nodeType === 'object' || nodeType === 'array' ? node : null;
        const nodePath = node ? parseJsonNodePath(node.getAttribute('data-json-path')) : null;

        // Add key/item into the targeted container.
        if (containerNode) {
            const containerPath = parseJsonNodePath(containerNode.getAttribute('data-json-path')) || [];

            if (menuItems.length > 0) {
                menuItems.push({ title: '-' });
            }

            if (nodeType === 'object') {
                menuItems.push({
                    title: 'Add key',
                    icon: CONTEXT_ICON_ADD,
                    onSelect: async () => {
                        await addStructuredTreeKey(container, getRoot, containerPath);
                    },
                });
            } else {
                menuItems.push({
                    title: 'Add item',
                    icon: CONTEXT_ICON_ADD,
                    onSelect: async () => {
                        if (typeof container.__jsonViewerOnEditCommit === 'function') {
                            await container.__jsonViewerOnEditCommit({ editType: 'addItem', path: containerPath });
                        }
                    },
                });
            }

            menuItems.push(
                {
                    title: 'Expand all',
                    icon: CONTEXT_ICON_EXPAND_ALL,
                    onSelect: () => {
                        expandJsonViewerSubtree(container, containerNode);
                    },
                },
                {
                    title: 'Collapse all',
                    icon: CONTEXT_ICON_COLLAPSE_ALL,
                    onSelect: () => {
                        collapseJsonViewerSubtree(containerNode);
                    },
                },
            );
        }

        // Delete the targeted node (anything but the document root).
        if (node && nodePath && nodePath.length > 0) {
            const lastSegment = nodePath[nodePath.length - 1];
            const deleteLabel = typeof lastSegment === 'number'
                ? `item ${lastSegment}`
                : `"${lastSegment}"`;
            menuItems.push(
                { title: '-' },
                {
                    title: 'Delete',
                    icon: CONTEXT_ICON_DELETE,
                    onSelect: () => {
                        openConfirmPrompt({
                            title: 'Delete field',
                            body: `Are you sure you want to delete ${deleteLabel}?`,
                            confirmLabel: 'Delete',
                            onConfirm: async () => {
                                if (typeof container.__jsonViewerOnEditCommit === 'function') {
                                    await container.__jsonViewerOnEditCommit({ editType: 'delete', path: nodePath });
                                }
                            },
                        });
                    },
                },
            );
        }

        menuItems.push({ title: '-' }, createAskAIDocumentMenuItem());

        showNotesLocalMenu(menuItems, e.clientX, e.clientY, menuTitle);
    });
}

async function createNewFile() {
    if (state.graphPromptHandler) {
        state.graphPromptSubmit?.();
        return;
    }
    if (state.textPromptHandler) {
        finishTextPrompt((elements.modalInput.value || '').trim());
        return;
    }
    if (state.gitBranchPromptHandler) {
        finishGitBranchPrompt((elements.modalInput.value || '').trim());
        return;
    }
    if (state.gitWorktreePrompt) {
        void state.gitWorktreePrompt.submit?.();
        return;
    }

    // Handle rename operation
    if (state.renamingFile) {
        const name = (elements.modalInput.value || '').trim();
        if (name === '') {
            notifyTerminal('File name cannot be empty', 'warn');
            return;
        }

        const useLocationSelector = elements.modalLocation.style.display !== 'none';
        const fileName = useLocationSelector
            ? await ComposeNoteLocationPath((elements.modalLocation.textContent || '$NOTES').trim(), name)
            : name;

        const oldPath = state.renamingFile;
        const oldPrefix = `${oldPath}${PRIMARY_PATH_SEPARATOR}`;
        const renamingFolder = Boolean(state.renamingFolder);
        const project = await GetCurrentProject();
        const oldKey = notesDocumentKey(oldPath, project);
        // Open documents of the renamed file, or of files in the renamed
        // folder, follow it to the new path (ADR 0100 question 12).
        const renamedDocuments = notesDocuments.list().filter((doc) => doc.path.startsWith(oldPath)
            && (doc.key === oldKey || (renamingFolder && doc.key.startsWith(`${oldKey}${PRIMARY_PATH_SEPARATOR}`))));
        const renamingCurrent = renamedDocuments.includes(activeNotesDocument);
        state.mutatingFile = renamingCurrent ? state.currentFile : oldPath;
        state.mutatingFolder = renamingFolder ? oldPath : '';
        try {
            // Unsaved edits are saved so they move with the file.
            for (const doc of renamedDocuments) {
                if (doc.dirty && savesNotesDocumentAutomatically(doc)) {
                    clearTimeout(doc.autosaveTimer);
                    doc.autosaveTimer = null;
                    await saveFile(doc);
                }
            }
            await RenameFile(oldPath, fileName);
            if (renamingFolder) {
                for (const key of Object.keys(state.expandedFolders)) {
                    if (key === oldPath || key.startsWith(oldPrefix)) {
                        state.expandedFolders[fileName + key.slice(oldPath.length)] = state.expandedFolders[key];
                        delete state.expandedFolders[key];
                    }
                }
                saveProjectCache();
            }
            for (const doc of renamedDocuments) {
                await settleLspDocumentCloses(releaseLspDocument(doc));
                const previousKey = doc.key;
                doc.uri = '';
                await notesDocuments.move(doc, fileName + doc.path.slice(oldPath.length));
                notesViewList().forEach((view) => view.renameDocument(previousKey, doc.key));
            }
            if (renamedDocuments.length > 0) {
                syncWatchedNotesFiles();
                renderAllNotesTabs();
            }
            if (renamingCurrent) {
                await loadFile(activeNotesDocument.path, { keepUnsavedEdits: true });
            }
            await refreshFiles({ skipHistoryRestore: true });
            closeNewFilePrompt();
            setStatus(`Renamed to ${fileName}`, false);
        } catch (err) {
            notifyTerminal(`Failed to rename ${state.renamingFolder ? 'folder' : 'file'}: ${err}`, 'error');
            console.error(err);
        } finally {
            state.mutatingFile = '';
            state.mutatingFolder = '';
        }
        return;
    }

    const name = normalizeNoteName(elements.modalInput.value);
    if (name === '') {
        notifyTerminal('File name cannot be empty', 'warn');
        return;
    }

    const fileName = await ComposeNoteLocationPath(
        (elements.modalLocation.textContent || '$NOTES').trim(),
        name,
    );

    const exists = state.files.some((file) => file === fileName);
    if (exists) {
        closeNewFilePrompt();
        await loadFile(fileName);
        addHistoryPosition(fileName);
        notifyTerminal(`${fileName} already exists`, 'warn');
        return;
    }

    try {
        const pendingFile = state.pendingNewFile;
        if (pendingFile?.imageBase64) {
            await SaveBinaryFile(fileName, pendingFile.imageBase64);
        } else {
            await SaveFile(fileName, pendingFile?.text || '', '');
        }
        await refreshFiles();
        await loadFile(fileName);
        addHistoryPosition(fileName);
        setViewMode('editor');
        closeNewFilePrompt();
        setStatus(`Created ${fileName}`, false);
    } catch (err) {
        notifyTerminal(`Failed to create ${fileName}: ${err}`, 'error');
        //console.error(err);
    }
}

async function createAndOpenFile(filename, contents) {
    const fileName = normalizeNotePath(filename);
    if (fileName === '') {
        setStatus('File name cannot be empty.', true);
        return;
    }

    try {
        await SaveFile(fileName, contents || '', '');
        await refreshFiles();
        await loadFile(fileName);
        addHistoryPosition(fileName);
        //setViewMode('editor');
        setViewMode('viewer');
        setStatus(`Created ${fileName}`, false);
    } catch (err) {
        notifyTerminal(`Failed to create ${fileName}: ${err}`, 'error');
        //console.error(err);
    }
}

async function saveImageToFile(filename, dataURL) {
    try {
        // Open save dialog via Wails runtime API (through Go binding)
        const savedPath = await SaveImageDialog(filename);
        
        if (!savedPath) {
            return; // User cancelled
        }
        
        // Extract base64 data from dataURL
        const base64Data = dataURL.split(',')[1];
        if (!base64Data) {
            setStatus('Failed to extract image data.', true);
            return;
        }
        
        // Save the file
        await SaveBinaryFile(savedPath, base64Data);
        setStatus(`Image saved to ${savedPath}`, false);
    } catch (err) {
        notifyTerminal(`Failed to save image: ${err.message || err}`, 'error');
        console.error('Error saving image:', err);
    }
}

EventsOn("notesCreateAndOpen", params => {
    createAndOpenFile(params.filename, params.contents);
});

EventsOn("notesContextChanged", (context) => {
    if (!context || typeof context !== 'object') return;
    if (activeGitDiff && context.projectRoot !== activeGitDiff.root) {
        closeNotesGitDiff();
    }
    if (requestedFileContext && ['projectRoot', 'globalNotes', 'userNotes', 'groupName']
        .every((field) => context[field] === requestedFileContext[field])) return;
    void refreshFiles({ context });
});

EventsOn('notesFileIndexChanged', (event) => {
    if (!event?.root || !requestedFileContext) return;
    const root = normalizeFileIndexRoot(event.root);
    if (!['projectRoot', 'globalNotes', 'userNotes'].some((field) =>
        normalizeFileIndexRoot(requestedFileContext[field]) === root)) return;
    const snapshot = fileIndexRoots.get(root);
    if (snapshot && event.revision <= snapshot.revision) return;
    if (fileListRefreshInFlight > 0 || !snapshot || !event.state || event.baseVersion !== snapshot.version
        || event.version < snapshot.version || event.version > snapshot.version + 1) {
        queueFileIndexRefresh();
        return;
    }
    const files = new Set(snapshot.files || []);
    for (const path of event.removed || []) files.delete(path);
    for (const path of event.added || []) files.add(path);
    fileIndexRoots.set(root, {
        root: event.root, files: Array.from(files).sort(), version: event.version,
        revision: event.revision, state: event.state, stale: event.stale,
        truncated: event.truncated, error: event.error,
    });
    renderFileIndexStatus(indexedFileStatus(fileIndexRoots));
    if (event.version !== snapshot.version) {
        fileListRefreshSequence++;
        state.files = indexedFileList(requestedFileContext, fileIndexRoots);
        void applyFileFilter();
    }
});

// Reloads the active document from disk, keeping the editor caret roughly in
// place.
async function reloadActiveNotesDocument() {
    const shouldRestoreCaret = state.viewMode === 'editor' || state.viewMode === 'swagger-edit';
    const previousSelection = shouldRestoreCaret
        ? {
            ...mainDocument.getSelection(),
            length: mainDocument.getValue().length,
            hadFocus: Boolean(focusedView().parts.monacoEditor?.contains(document.activeElement)),
        }
        : null;

    try {
        await loadFile(state.currentFile);

        if (previousSelection && (state.viewMode === 'editor' || state.viewMode === 'swagger-edit')) {
            const nextLen = mainDocument.getValue().length;
            const ratio = previousSelection.length > 0 ? nextLen / previousSelection.length : 1;
            const nextStart = Math.min(Math.round(previousSelection.start * ratio), nextLen);
            const nextEnd = Math.min(Math.round(previousSelection.end * ratio), nextLen);

            mainDocument.setSelection(nextStart, nextEnd);
            if (previousSelection.hadFocus && isMonacoActive()) {
                focusedView().editor.focus();
            }
        }
    } catch (err) {
        setStatus(`Failed to reload ${state.currentFile}`, true);
        console.error(err);
    }
}

// The event names the open documents of the changed or deleted file. Without
// names, the change is to the active document.
EventsOn("notesFileChanged", async (event) => {
    const keys = Array.isArray(event?.keys) ? event.keys : null;
    const docs = keys ? keys.map(findNotesDocument).filter(Boolean) : [activeNotesDocument];
    for (const doc of docs) {
        if (event?.deleted && doc.path && !isNotesPathMutating(doc.path)) {
            await handleNotesDocumentDeleted(doc);
        } else {
            await handleNotesDocumentChangedOnDisk(doc);
        }
    }
});

EventsOn("notesShowLspOptions", async () => {
    if (!isCurrentFileLspEligible()) {
        notifyTerminal("No Language Server Protocol (LSP) is not supported for this file type", "warn");
        return;
    }

    const languageID = await ResolveNotesLspLanguage(state.currentFile);
    if (!languageID) {
        notifyTerminal("No Language Server Protocol (LSP) has been defined for this file type", "warn");
        return;
    }

    const rect = focusedView().parts.monacoEditor.getBoundingClientRect();
    const x = Math.max(rect.left + 24, Math.min(rect.right - 24, rect.left + rect.width / 2));
    const y = Math.max(rect.top + 24, Math.min(rect.bottom - 24, rect.top + rect.height / 2));
    await showEditorLspOptionsMenu(x, y);
});

EventsOn("notesRunLspFormatDocument", async () => {
    if (!isCurrentFileLspEligible()) {
        notifyTerminal("No Language Server Protocol (LSP) is not supported for this file type", "warn");
        return;
    }

    const languageID = await ResolveNotesLspLanguage(state.currentFile);
    if (!languageID) {
        notifyTerminal("No Language Server Protocol (LSP) has been defined for this file type", "warn");
        return;
    }

    await formatCurrentLspDocument();
});

EventsOn("notesRunLspGoToSymbol", async () => {
    if (!isCurrentFileLspEligible()) {
        notifyTerminal("No Language Server Protocol (LSP) is not supported for this file type", "warn");
        return;
    }

    const languageID = await ResolveNotesLspLanguage(state.currentFile);
    if (!languageID) {
        notifyTerminal("No Language Server Protocol (LSP) has been defined for this file type", "warn");
        return;
    }

    await goToCurrentLspSymbol();
});

EventsOn("notesRunLspGoToWorkspaceSymbol", async () => {
    if (!isCurrentFileLspEligible()) {
        notifyTerminal("Language Server Protocol (LSP) is not supported for this file type", "warn");
        return;
    }

    const languageID = await ResolveNotesLspLanguage(state.currentFile);
    if (!languageID) {
        notifyTerminal("No Language Server Protocol (LSP) has been defined for this file type", "warn");
        return;
    }

    await goToWorkspaceLspSymbol();
});

// Output is recorded for the document that started the run, and shown if that
// document is active.
EventsOn("noteRun", (data) => {
    const { runId, output, isError } = data;
    const run = jupyterRuns.get(String(runId ?? ''));
    if (!run) return;
    const { doc, blockId } = run;
    const text = String(output ?? '');
    const isErr = String(isError) === 'true';
    if (!doc.jupyterOutputs.has(blockId)) {
        doc.jupyterOutputs.set(blockId, []);
    }
    doc.jupyterOutputs.get(blockId).push({ text, isError: isErr });
    if (doc !== activeNotesDocument) return;

    const outputBlock = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-output`);
    if (!outputBlock) return;
    appendJupyterOutput(outputBlock, text, isErr);
    scrollJupyterOutputToBottom(outputBlock);
});

EventsOn("noteComplete", (data) => {
    const run = finishJupyterRun(String(data?.runId ?? ''));
    if (!run || run.doc !== activeNotesDocument) return;
    const { blockId } = run;
    // A newer run of the same block is still in progress.
    if (run.doc.runningJupyterBlocks.has(blockId)) return;

    // Toggle buttons back to Run
    const runBtn = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-run-notes`);
    const stopBtn = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-stop-notes`);
    if (runBtn) runBtn.style.display = 'inline-block';
    if (stopBtn) stopBtn.style.display = 'none';
});

// Tools Panel Event Handlers
function setToolsPanelCollapsed(collapsed) {
    const isCollapsed = collapsed === true;
    elements.toolsPanel.dataset.collapsed = isCollapsed ? 'true' : 'false';
    notesGitPanel?.setWatchActive(!isCollapsed && elements.toolsTabGit?.getAttribute('aria-selected') === 'true');
    window.dispatchEvent(new CustomEvent('ttyphoon-notes-layout'));
    if (elements.toolsRestore) {
        elements.toolsRestore.style.display = isCollapsed ? 'inline-flex' : 'none';
    }
    // If AI output was rendered while hidden, force a layout-time bottom snap
    // when the panel becomes visible again.
    if (!isCollapsed && state.aiStickToBottom && isAIToolsTabActive()) {
        requestAnimationFrame(() => {
            requestAnimationFrame(() => {
                scrollAIOutputToBottom();
            });
        });
    }
}

function toggleToolsPanel() {
    const isCollapsed = elements.toolsPanel.dataset.collapsed === 'true';
    setToolsPanelCollapsed(!isCollapsed);
}
function isAIToolsTabActive() {
    if (!elements.toolsPanel) {
        return false;
    }

    const activePane = elements.toolsPanel.querySelector('.notes-tools-pane[data-active="true"]');
    return activePane?.dataset?.tab === 'ai';
}

function setToolsTab(tabName) {
    const nextTab = String(tabName || '').trim().toLowerCase();
    if (!nextTab) {
        return;
    }

    const tabButtons = elements.toolsTabs?.querySelectorAll('[role="tab"]') || [];
    for (const button of tabButtons) {
        const isActive = button.dataset.tab === nextTab;
        button.setAttribute('aria-selected', isActive ? 'true' : 'false');
    }

    const panes = elements.toolsPanel?.querySelectorAll('.notes-tools-pane') || [];
    for (const pane of panes) {
        const isActive = pane.dataset.tab === nextTab;
        pane.dataset.active = isActive ? 'true' : 'false';
    }

    notesGitPanel?.setWatchActive(nextTab === 'git' && elements.toolsPanel?.dataset.collapsed !== 'true');

    if (nextTab === 'ai' && state.aiStickToBottom) {
        // Switching to AI can reveal previously hidden content; scroll after layout.
        requestAnimationFrame(() => {
            requestAnimationFrame(() => {
                scrollAIOutputToBottom();
            });
        });
    }

    if (nextTab === 'ai') {
        maybeLoadPendingAISessionCache();
    }
}

function saveDocumentCache() {
    if (!state.currentFile || state.suspendDocumentCacheSave) {
        return;
    }

    // Only the document view mode is persisted. The Tools panel open/closed
    // state and selected tab are deliberately not cached.
    SetDocumentCache(state.currentFile, {
        DocumentTab: state.viewMode || '',
        ToolsOpen: false,
        ToolsTab: '',
        WordWrap: state.markdownWrapMode === true,
    }).catch((err) => {
        console.error('Failed to save document cache:', err);
    });
}

function clearAIOutput() {
    const shouldStick = state.aiStickToBottom || isAIOutputNearBottom();
    aiPipelineFormatter.clear();
    scheduleAIPromptJumpRefresh();
    ClearAILog().catch((err) => {
        console.error('Failed to clear AI log:', err);
    });
    if (shouldStick) {
        requestAnimationFrame(() => {
            scrollAIOutputToBottom();
        });
    }
}

function startAIJob(title) {
    // A fresh job replaces the panel contents, so drop any deferred log load
    // that would otherwise overwrite it when the AI tab is (auto-)selected.
    state.aiSessionCachePending = false;
    state.aiSessionCachePendingWorkspace = '';

    // Clear the panel so only the new prompt renders.
    aiPipelineFormatter.clear();
    aiPipelineFormatter.startJob(String(title || ''));
    scheduleAIPromptJumpRefresh();
    requestAnimationFrame(() => {
        scrollAIOutputToBottom();
    });
}

let aiActiveRunId = null;
let aiRunGeneration = 0;
let aiCompletionRevealVersion = 0;
let aiCompletionRevealPending = false;

// Tells Go whether the panel is following live output; guarded so stale
// generated bindings can't throw.
function setAIPanelLive(live) {
    state.aiPanelLive = Boolean(live);
    try {
        if (typeof SetAIPanelLive === 'function') {
            void Promise.resolve(SetAIPanelLive(String(state.currentWorkspaceName || ''), Boolean(live))).catch(() => {});
        }
    } catch (err) {
        console.error('Failed to update AI panel live state:', err);
    }
    updateAIPromptJumpAvailability();
}

function startAIStreamJob(payload) {
    aiCompletionRevealVersion++;
    aiCompletionRevealPending = false;
    const runId = Number(payload?.runId);
    aiActiveRunId = Number.isSafeInteger(runId) && runId > 0 ? runId : null;
    aiRunGeneration++;
    state.aiRunActive = true;
    state.aiSelectedPrompt = { source: 'live' };
    // A new job always follows live output, whichever surface triggered it.
    setAIPanelLive(true);
    startAIJob(payload?.title ?? payload);
    void refreshAIPromptJumpFromBackend();
}

async function finishAIJob(payload = {}) {
    const generation = aiRunGeneration;
    const runId = aiActiveRunId;
    const wasViewingLive = state.aiPanelLive && state.aiSelectedPrompt?.source === 'live';
    const following = wasViewingLive && state.aiStickToBottom;
    const version = ++aiCompletionRevealVersion;
    aiCompletionRevealPending = following;
    if (following) {
        state.aiStickToBottom = false;
        clearAIBottomScrollRetries();
    }
    try {
        if (wasViewingLive) {
            await finalizeAITableIdentities(payload);
            if (generation !== aiRunGeneration || runId !== aiActiveRunId) return;
            await aiPipelineFormatter.finishJob({ scrollToBottom: false });
            if (version === aiCompletionRevealVersion && runId === aiActiveRunId && following && state.aiPanelLive) {
                const response = aiPipelineFormatter.finalResponseElement();
                if (response) {
                    const output = elements.aiOutput;
                    output.scrollTop = Math.max(0, output.scrollTop + response.getBoundingClientRect().top
                        - output.getBoundingClientRect().top - output.clientTop);
                } else {
                    state.aiStickToBottom = true;
                }
            }
        }
        if (generation !== aiRunGeneration || runId !== aiActiveRunId) return;
        state.aiRunActive = false;
        aiActiveRunId = null;
        state.aiSelectedPrompt = null;
        setAIPanelLive(false);
        const targets = await refreshAIPromptJumpFromBackend();
        if (generation !== aiRunGeneration) return;
        const promptId = Number(payload?.promptId) || 0;
        const completedTarget = (promptId > 0
            ? targets.find((target) => target.promptId === promptId)
            : null) || (promptId > 0 ? null : targets.at(-1));
        if (completedTarget) {
            state.aiSelectedPrompt = completedTarget;
            updateAIPromptJumpAvailability();
            if (!wasViewingLive) {
                await jumpToAIPromptTarget(completedTarget);
            }
        }
    } catch (err) {
        console.error('Failed to finalize AI output:', err);
    } finally {
        if (version === aiCompletionRevealVersion) aiCompletionRevealPending = false;
        updateAIScrollBottomButton();
        if (generation === aiRunGeneration) void refreshAIPromptJumpFromBackend();
    }
}

function isAIOutputNearBottom() {
    if (!elements.aiOutput) {
        return true;
    }

    const remaining = elements.aiOutput.scrollHeight - (elements.aiOutput.scrollTop + elements.aiOutput.clientHeight);
    return remaining <= AI_BOTTOM_THRESHOLD_PX;
}

function updateAIScrollBottomButton() {
    if (!elements.aiScrollBottom) {
        return;
    }

    const visible = !isAIOutputNearBottom();
    elements.aiScrollBottom.dataset.visible = visible ? 'true' : 'false';
}

function pauseAIAutoScroll() {
    if (!state.aiStickToBottom) {
        return;
    }
    state.aiStickToBottom = false;
    clearAIBottomScrollRetries();
    updateAIScrollBottomButton();
}

function clearAIBottomScrollRetries() {
    for (const timerId of aiBottomScrollRetryTimers) {
        clearTimeout(timerId);
    }
    aiBottomScrollRetryTimers = [];

    if (aiBottomChaseHandle) {
        cancelAnimationFrame(aiBottomChaseHandle);
        aiBottomChaseHandle = 0;
    }
    if (aiStreamScrollHandle) {
        cancelAnimationFrame(aiStreamScrollHandle);
        aiStreamScrollHandle = 0;
    }
    aiBottomChaseUntil = 0;
}

function scheduleAIStreamScrollToBottom() {
    if (!elements.aiOutput || !state.aiStickToBottom || aiStreamScrollHandle) {
        return;
    }
    aiStreamScrollHandle = requestAnimationFrame(() => {
        aiStreamScrollHandle = 0;
        if (!elements.aiOutput || !state.aiStickToBottom) {
            return;
        }
        elements.aiOutput.scrollTop = elements.aiOutput.scrollHeight;
    });
}

function requestAIScrollToBottom() {
    aiCompletionRevealVersion++;
    aiCompletionRevealPending = false;
    if (!elements.aiOutput) {
        return;
    }

    state.aiStickToBottom = true;
    clearAIBottomScrollRetries();

    // Run immediately and then retry after known async render phases:
    // requestAnimationFrame, markup parse completion, and lazy chunk expansion.
    scrollAIOutputToBottom();
    const retryDelays = [0, 40, 120, 260, 520];
    for (const delay of retryDelays) {
        const timerId = setTimeout(() => {
            if (!elements.aiOutput || !state.aiStickToBottom) {
                return;
            }
            scrollAIOutputToBottom();
        }, delay);
        aiBottomScrollRetryTimers.push(timerId);
    }
}

function scrollAIOutputToBottom() {
    if (!elements.aiOutput || !state.aiStickToBottom) {
        return;
    }

    const output = elements.aiOutput;
    // Streaming calls this once per chunk. Extend the deadline of the chase
    // that's already in flight instead of starting another: overlapping rAF
    // loops would each force a layout every frame and stall the main thread.
    aiBottomChaseUntil = nowMs() + AI_BOTTOM_CHASE_MS;
    if (aiBottomChaseHandle) {
        return;
    }

    let lastHeight = -1;
    let stableFrames = 0;

    const chaseBottom = () => {
        aiBottomChaseHandle = 0;

        if (!elements.aiOutput || !state.aiStickToBottom) {
            return;
        }

        output.scrollTop = output.scrollHeight;

        const currentHeight = Number(output.scrollHeight) || 0;
        stableFrames = Math.abs(currentHeight - lastHeight) < 1 ? stableFrames + 1 : 0;
        lastHeight = currentHeight;

        if (stableFrames >= 2 && nowMs() >= aiBottomChaseUntil) {
            updateAIScrollBottomButton();
            return;
        }

        aiBottomChaseHandle = requestAnimationFrame(chaseBottom);
    };

    output.scrollTop = output.scrollHeight;
    aiBottomChaseHandle = requestAnimationFrame(chaseBottom);
}

// Blocks are stored without framing, so the live view must be rebuilt from the
// block index; concatenating their content would lose every fence and quote.
async function restoreAILiveBlocks(workspaceName) {
    if (typeof ListAILiveStreamBlocks !== 'function' || typeof GetAIStreamBlockContent !== 'function') {
        return false;
    }

    let blocks = [];
    try {
        blocks = await ListAILiveStreamBlocks(String(workspaceName || ''));
    } catch (err) {
        console.error('Failed to load live AI stream blocks:', err);
        return false;
    }

    if (!Array.isArray(blocks) || blocks.length === 0) {
        return false;
    }

    const active = blocks.some((block) => block?.active === true
        || (block?.promptId !== undefined && Number(block.promptId) === 0));
    state.aiRunActive = active;
    const lastBlock = blocks.at(-1);
    const storedTarget = !active
        ? state.aiPromptJumpTargets.find((target) =>
            target.sessionId === Number(lastBlock?.sessionId)
            && target.promptId === Number(lastBlock?.promptId))
        : null;
    if (active) {
        aiPipelineFormatter.setBlocks(blocks, (block) => GetAIStreamBlockContent(
            Number(block?.sessionId) || 0,
            Number(block?.promptId) || 0,
            String(block?.blockId || ''),
        ));
    } else {
        await renderAIStoredPrompt(blocks, storedTarget || {
            sessionId: Number(lastBlock?.sessionId) || 0,
            promptId: Number(lastBlock?.promptId) || 0,
        });
    }
    const runId = Number(lastBlock?.runId) || 0;
    aiActiveRunId = active && runId > 0 ? runId : null;
    state.aiSelectedPrompt = active
        ? { source: 'live' }
        : storedTarget || {
            source: 'backend',
            sessionId: Number(lastBlock?.sessionId) || 0,
            promptId: Number(lastBlock?.promptId) || 0,
        };
    setAIPanelLive(active);
    requestAIScrollToBottom();
    return true;
}

async function loadAISessionCache(workspaceName) {
    try {
        const targets = await refreshAIPromptJumpFromBackend();
        const restored = await restoreAILiveBlocks(workspaceName);

        if (!restored) {
            if (state.aiRunActive) {
                setAIPanelLive(true);
            } else {
                // Fully reset the panel before rendering so content from a previously
                // active workspace cannot persist across a workspace (tmux tab) switch.
                aiPipelineFormatter.clear();
                aiActiveRunId = null;
                state.aiSelectedPrompt = null;
                setAIPanelLive(false);
            }
        }

        if (!state.aiRunActive && !state.aiSelectedPrompt) {
            state.aiSelectedPrompt = targets.at(-1) || null;
        }
        if (elements.aiSettingsModal?.dataset?.open === 'true') {
            void loadAISessionManagement().then(() => {
                renderAISessionManagement();
            });
        }
    } catch (err) {
        console.error('Failed to load AI session cache:', err);
    }
}

// markAISessionCachePending clears the panel immediately (cheap) and records
// that the AI log for the given workspace still needs loading. The actual fetch
// + render is deferred until the AI tab is opened so it never blocks the rest
// of the UI on workspace/file switching.
function markAISessionCachePending(workspaceName) {
    state.aiSessionCachePending = true;
    state.aiSessionCachePendingWorkspace = String(workspaceName || '');
    aiPipelineFormatter.clear();
    // The run being left belongs to the old workspace; restoring the new one
    // re-adopts whichever run is live there.
    aiActiveRunId = null;
    aiRunGeneration++;
    state.aiRunActive = false;
    state.aiPromptJumpTargets = [];
    state.aiSelectedPrompt = null;
    // A workspace switch resets the panel, so stop showing a stale historical
    // prompt from the workspace being left.
    setAIPanelLive(true);

    if (isAIToolsTabActive()) {
        maybeLoadPendingAISessionCache();
    }
}

// maybeLoadPendingAISessionCache fires a non-blocking load when a deferred AI
// log is outstanding. Safe to call repeatedly; it no-ops once consumed.
function maybeLoadPendingAISessionCache() {
    if (!state.aiSessionCachePending) {
        return;
    }
    const workspaceName = state.aiSessionCachePendingWorkspace;
    state.aiSessionCachePending = false;
    state.aiSessionCachePendingWorkspace = '';
    void loadAISessionCache(workspaceName);
}

const aiPipelineFormatter = createAIPipelineFormatter(elements.aiOutput, {
    marked,
    processMarkdownContainer: processAIMarkdownContainer,
    processCodeContainer: processAIMarkdownContainer,
    codeMaxLines: Number.parseInt(
        getComputedStyle(document.documentElement)
            .getPropertyValue('--notes-ai-code-max-lines')
            .trim(),
        10,
    ) || 10,
});

// Event emitted by Go when an AI job begins (before first chunk)
EventsOn("aiJobStart", (title) => {
    startAIStreamJob(title);
});

// Event emitted by Go when an AI job finishes
EventsOn("aiJobFinish", (payload) => {
    const runId = Number(payload?.runId);
    if (Number.isSafeInteger(runId) && runId > 0 && runId !== aiActiveRunId) return;
    void finishAIJob(payload);
});

EventsOn("aiStreamBlock", (payload) => {
    const runId = Number(payload?.runId);
    if (runId !== aiActiveRunId || !payload?.blockId) {
        return;
    }
    // Backend gating is authoritative; this also covers blocks emitted in the
    // window between a workspace switch and the SetAIPanelLive round-trip.
    const blockWorkspace = String(payload?.workspace ?? '').trim();
    const panelWorkspace = String(state.currentWorkspaceName || '').trim();
    if (blockWorkspace && panelWorkspace && blockWorkspace !== panelWorkspace) {
        return;
    }
    aiPipelineFormatter.appendBlock(payload);
    if (state.aiStickToBottom) {
        scheduleAIStreamScrollToBottom();
    } else {
        scheduleAIScrollButtonUpdate();
    }
});

EventsOn("aiToolStateChanged", (payload) => {
    const name = String(payload?.name || '');
    const stateValue = String(payload?.state || '');
    const tool = state.aiToolsList.find(item => item?.name === name);
    if (tool) {
        if (stateValue) {
            tool.state = stateValue;
            tool.enabled = stateValue !== 'disabled';
        }
        if (typeof payload?.allowInSubagent === 'boolean') {
            tool.allowInSubagent = payload.allowInSubagent;
        }
    }
    renderAIToolsList();
});

// Intercept ttyphoon://ai/... links rendered anywhere in the notes UI
document.addEventListener('ttyphoon-ai-prompt', async (e) => {
    const prompt = e.detail?.prompt;
    if (!prompt) {
        return;
    }
    const tools = String(e.detail?.tools ?? '');
    setToolsPanelCollapsed(false);
    setToolsTab('ai');
    setAIPanelLive(true);
    try {
        await AskAI('notesPromptUri', prompt, tools);
    } catch (err) {
        notifyTerminal('Failed to ask AI', 'error');
        console.error(err);
    }
});

function markAIToolPermissionResolved(requestId, decision, anchor) {
    if (!elements.aiOutput) {
        return;
    }
    for (const link of elements.aiOutput.querySelectorAll('a[href^="ttyphoon://ai-tool-permission"]')) {
        const href = link.getAttribute('href') || '';
        const query = href.includes('?') ? href.slice(href.indexOf('?') + 1) : '';
        const params = new URLSearchParams(query);
        if (params.get('request') !== requestId) {
            continue;
        }
        link.classList.add('ai-tool-permission-resolved');
        link.removeAttribute('href');
        if (link === anchor || params.get('decision') === decision) {
            link.classList.add('ai-tool-permission-chosen');
        }
    }
}

EventsOn('aiToolPermissionResolved', (payload) => {
    const requestId = String(payload?.requestId || '');
    const decision = String(payload?.decision || '');
    if (requestId && decision) {
        markAIToolPermissionResolved(requestId, decision, null);
    }
});

EventsOn('aiToolPermissionResolutionError', (payload) => {
    const message = String(payload?.error || 'Could not resolve tool access request');
    notifyTerminal(message, 'error');
});

// Access-request options rendered in the AI panel output (see
// agent.RequestToolPermission) resolve here rather than via a native/context menu.
document.addEventListener('ttyphoon-ai-tool-permission', async (e) => {
    const requestId = String(e.detail?.requestId || '');
    const decision = String(e.detail?.decision || '');
    const anchor = e.detail?.anchor;
    if (!requestId || !decision) {
        return;
    }

    if (decision.startsWith('deny-')) {
        try {
            await PromptAIToolPermissionReason(requestId, decision);
        } catch (err) {
            notifyTerminal('Failed to open tool denial input', 'error');
            console.error(err);
        }
        return;
    }

    try {
        await ResolveAIToolPermission(requestId, decision);
        markAIToolPermissionResolved(requestId, decision, anchor);
    } catch (err) {
        notifyTerminal('Failed to resolve tool access request', 'error');
        console.error(err);
    }
});

document.addEventListener('ttyphoon-ai-user-question', async (e) => {
    const requestId = String(e.detail?.requestId || '');
    const answer = String(e.detail?.answer || '');
    const anchor = e.detail?.anchor;
    if (!requestId || !answer) {
        return;
    }

    if (elements.aiOutput) {
        Array.from(elements.aiOutput.querySelectorAll('a')).forEach((link) => {
            const href = link.getAttribute('href') || '';
            if (href.startsWith('ttyphoon://ai-user-question') && href.includes(`request=${requestId}`)) {
                link.classList.add('ai-user-question-resolved');
                link.removeAttribute('href');
            }
        });
    }
    if (anchor instanceof HTMLElement) {
        anchor.classList.add('ai-user-question-chosen');
    }

    const resolveUserQuestion = typeof ResolveAIUserQuestion === 'function' ? ResolveAIUserQuestion : null;

    try {
        await resolveUserQuestion(requestId, answer);
    } catch (err) {
        notifyTerminal('Failed to resolve user question', 'error');
        console.error(err);
    }
});

// Event emitted by Go after the user selects a file from the ViewFileInNotes menu.
// Opens the file and selects the first (default) tab for the file type.
EventsOn('viewFileInNotesOpen', async (payload) => {
    const file = typeof payload === 'string' ? payload : String(payload?.file || payload?.filename || '');
    const line = typeof payload === 'object' ? Number.parseInt(String(payload?.line), 10) || 1 : 1;
    if (!file) return;

    try {
        addHistoryPosition(file, line);
        await loadFile(file);
    } catch (err) {
        notifyTerminal(`Failed to load file: ${file}`, 'warn');
        console.error(err);
    }
});

// Event emitted by Go to open a file directly in the Edit tab.
EventsOn('viewFileInNotesEdit', async (payload) => {
    const file = typeof payload === 'string' ? payload : String(payload?.file || payload?.filename || '');
    const line = typeof payload === 'object' ? Number.parseInt(String(payload?.line), 10) || 1 : 1;
    if (!file) return;

    try {
        addHistoryPosition(file, line);
        await loadFile(file);
        setViewMode('editor');
    } catch (err) {
        notifyTerminal(`Failed to load file: ${file}`, 'warn');
        console.error(err);
    }
});

// Event listener for generic file action dialog (rename or delete any file link)
EventsOn('fileActionDialog', (payload) => {
    const action = typeof payload === 'object' ? payload.action : '';
    const filePath = typeof payload === 'object' ? payload.filePath : '';
    
    if (!filePath) return;
    
    switch (action) {
        case 'rename':
            void openRenamePrompt(filePath);
            break;
        case 'delete':
            openDeletePrompt(filePath);
            break;
    }
});

// Setup Tools panel listeners
if (elements.toolsMinimize) {
    elements.toolsMinimize.addEventListener('click', toggleToolsPanel);
}
if (elements.toolsAIAsk) {
    elements.toolsAIAsk.addEventListener('click', () => {
        void askAIFromToolbar();
    });
}
if (elements.toolsAISkills) {
    elements.toolsAISkills.addEventListener('click', () => {
        void askAISkillsFromToolbar();
    });
}
if (elements.toolsAIPromptJump) {
    elements.toolsAIPromptJump.addEventListener('click', () => {
        openAIPromptJumpMenu();
    });
}
if (elements.toolsAISession) {
    elements.toolsAISession.addEventListener('click', () => {
        void openAISessionMenu();
    });
}
if (elements.aiSettingsModelPicker) {
    elements.aiSettingsModelPicker.addEventListener('click', () => {
        openAIModelPickerMenu();
    });
}
if (elements.aiOutput) {
    elements.aiOutput.addEventListener('wheel', (event) => {
        if (event.deltaY !== 0) {
            aiCompletionRevealVersion++;
            aiCompletionRevealPending = false;
            pauseAIAutoScroll();
        }
    }, { passive: true });
    elements.aiOutput.addEventListener('touchmove', () => {
        aiCompletionRevealVersion++;
        aiCompletionRevealPending = false;
        pauseAIAutoScroll();
    }, { passive: true });
    elements.aiOutput.addEventListener('pointerdown', () => {
        if (aiCompletionRevealPending) {
            aiCompletionRevealVersion++;
            aiCompletionRevealPending = false;
        }
    });
    elements.aiOutput.addEventListener('scroll', () => {
        // Programmatic bottom writes stay in follow mode; moving away from the
        // bottom is user intent and requires Latest to resume following.
        if (aiScrollButtonHandle) {
            return;
        }
        aiScrollButtonHandle = requestAnimationFrame(() => {
            aiScrollButtonHandle = 0;
            if (!isAIOutputNearBottom()) {
                pauseAIAutoScroll();
            }
            updateAIScrollBottomButton();
        });
    }, { passive: true });
}
if (elements.aiScrollBottom) {
    elements.aiScrollBottom.addEventListener('click', () => {
        requestAIScrollToBottom();
    });
}
if (elements.toolsAISettings) {
    elements.toolsAISettings.addEventListener('click', () => {
        void loadAISessionManagement().then(() => {
            openAISessionManagementModal();
        });
    });
}
if (elements.aiSettingsSessionsList) {
    elements.aiSettingsSessionsList.addEventListener('click', async (event) => {
        const target = event.target instanceof HTMLElement ? event.target : null;
        if (!target) {
            return;
        }

        // Handle rename button click
        const renameButton = target.closest('button[data-action="rename"]');
        if (renameButton) {
            const sessionId = Number(renameButton.dataset.sessionId) || 0;
            if (sessionId <= 0) {
                return;
            }

            const sessions = Array.isArray(state.aiSessionManagement?.sessions) ? state.aiSessionManagement.sessions : [];
            const session = sessions.find((item) => Number(item.tableId) === sessionId);
            const nextSummary = await openTextPrompt({
                title: 'Rename session',
                value: String(session?.summary || ''),
                confirmLabel: 'Rename',
            });
            if (nextSummary === null) {
                return;
            }

            const trimmedSummary = nextSummary.trim();
            if (!trimmedSummary) {
                notifyTerminal('Rename cancelled: name is empty', 'warn');
                return;
            }

            RenameAISession(sessionId, trimmedSummary).then((sessionState) => {
                applyAISessionManagement(sessionState);
                renderAISessionManagement();
                notifyTerminal('AI session renamed', 'info');
            }).catch((err) => {
                notifyTerminal('Failed to rename AI session', 'error');
                console.error(err);
            });
            return;
        }

        // Handle delete button click
        const deleteButton = target.closest('button[data-action="delete"]');
        if (deleteButton) {
            const sessionId = Number(deleteButton.dataset.sessionId) || 0;
            if (sessionId <= 0) {
                return;
            }

            DeleteAISession(sessionId).then((sessionState) => {
                applyAISessionManagement(sessionState);
                renderAISessionManagement();
                void loadAISessionCache('');
                notifyTerminal('AI session deleted', 'info');
            }).catch((err) => {
                notifyTerminal('Failed to delete AI session', 'error');
                console.error(err);
            });
            return;
        }

        // Handle session item click for activation
        const sessionItem = target.closest('.notes-ai-settings-session-item');
        if (sessionItem) {
            const sessionId = Number(sessionItem.dataset.sessionId) || 0;
            if (sessionId <= 0 || target.closest('button')) {
                return; // Don't activate if clicking a button inside the item
            }

            activateAISession(sessionId);
            return;
        }
    });
}
if (elements.aiSettingsHistoryList) {
    elements.aiSettingsHistoryList.addEventListener('click', (event) => {
        const eraseButton = event.target instanceof HTMLElement
            ? event.target.closest('button[data-action="erase"]')
            : null;
        if (!eraseButton) {
            return;
        }

        const entryId = Number(eraseButton.dataset.entryId) || 0;
        if (entryId <= 0) {
            return;
        }

        DeleteAIHistoryEntry(entryId).then((sessionState) => {
            applyAISessionManagement(sessionState);
            renderAISessionManagement();
            void refreshAIPromptJumpFromBackend();
        }).catch((err) => {
            notifyTerminal('Failed to erase this prompt', 'error');
            console.error(err);
        });
    });
}
if (elements.aiSettingsSessionNew) {
    elements.aiSettingsSessionNew.addEventListener('click', () => {
        createNewAISession();
    });
}
if (elements.aiSettingsModal) {
    elements.aiSettingsModal.addEventListener('click', (event) => {
        if (event.target === elements.aiSettingsModal) {
            closeAISessionManagementModal();
        }
    });
}
elements.aiSettingsToolsFilter?.addEventListener('input', renderAIToolsList);
elements.aiSettingsMcpFilter?.addEventListener('input', renderAIMcpServersList);
if (elements.aiSettingsToolsList) {
    elements.aiSettingsToolsList.addEventListener('click', (event) => {
        const stateButton = event.target instanceof HTMLButtonElement
            ? event.target.closest('.notes-ai-settings-tool-state')
            : null;
        if (!stateButton) {
            return;
        }

        const toolName = String(stateButton.dataset.toolName || '').trim();
        if (!toolName) {
            return;
        }

        const rect = stateButton.getBoundingClientRect();
        ShowAIToolStateMenu(toolName, rect.left, rect.bottom + 4);
    });

    elements.aiSettingsToolsList.addEventListener('click', (event) => {
        const subagentButton = event.target instanceof HTMLButtonElement
            ? event.target.closest('.notes-ai-settings-tool-subagent')
            : null;
        if (!subagentButton) {
            return;
        }

        const toolName = String(subagentButton.dataset.toolName || '').trim();
        if (!toolName) {
            return;
        }

        const rect = subagentButton.getBoundingClientRect();
        ShowAIToolSubagentMenu(toolName, rect.left, rect.bottom + 4);
    });
}
if (elements.aiSettingsMcpList) {
    elements.aiSettingsMcpList.addEventListener('change', (event) => {
        const checkbox = event.target instanceof HTMLInputElement ? event.target : null;
        if (!checkbox || checkbox.type !== 'checkbox' || checkbox.disabled) {
            return;
        }

        const serverKey = String(checkbox.dataset.mcpKey || '').trim();
        if (!serverKey) {
            return;
        }

        const enabled = checkbox.checked;
        checkbox.disabled = true;
        SetAIMcpServerEnabled(serverKey, enabled).then(() => {
            void loadAISessionManagement().then(() => {
                renderAISessionManagement();
            });
        }).catch((err) => {
            notifyTerminal(`Failed to ${enabled ? 'load' : 'unload'} MCP server: ${err?.message || err}`, 'error');
            console.error(err);
            checkbox.checked = !enabled;
            void loadAISessionManagement().then(renderAISessionManagement);
        }).finally(() => { checkbox.disabled = false; });
    });

    elements.aiSettingsMcpList.addEventListener('click', (event) => {
        if (event.target instanceof HTMLInputElement && event.target.type === 'checkbox') {
            return;
        }

        const item = event.target instanceof Element ? event.target.closest('.notes-ai-settings-tool-item') : null;
        if (!item || !elements.aiSettingsMcpList.contains(item)) {
            return;
        }

        const index = Number(item.dataset.mcpIndex);
        if (!Number.isInteger(index) || index < 0) {
            return;
        }

        void openAIMcpServerMetadataModal(index);
    });
}
if (elements.aiToolMetaModal && elements.aiToolMetaCard) {
    elements.aiToolMetaModal.addEventListener('click', (event) => {
        if (event.target === elements.aiToolMetaModal) {
            closeAIToolMetadataModal();
        }
    });

    elements.aiToolMetaCard.addEventListener('click', (event) => {
        event.stopPropagation();
    });
}
if (elements.aiToolMetaContent) {
    elements.aiToolMetaContent.addEventListener('click', (event) => {
        const stateButton = event.target instanceof HTMLButtonElement
            ? event.target.closest('.notes-ai-settings-tool-state')
            : null;
        if (!stateButton) {
            return;
        }

        const toolName = String(stateButton.dataset.toolName || '').trim();
        if (!toolName) {
            return;
        }

        const rect = stateButton.getBoundingClientRect();
        ShowAIToolStateMenu(toolName, rect.left, rect.bottom + 4);
    });

    elements.aiToolMetaContent.addEventListener('change', (event) => {
        const checkbox = event.target instanceof HTMLInputElement ? event.target : null;
        if (!checkbox || checkbox.type !== 'checkbox') {
            return;
        }

        const configType = String(checkbox.dataset.configType || '').trim();
        if (configType === 'mcp') {
            if (checkbox.disabled) return;
            const serverKey = String(checkbox.dataset.mcpKey || '').trim();
            if (!serverKey) {
                return;
            }

            const enabled = checkbox.checked;
            checkbox.disabled = true;
            SetAIMcpServerEnabled(serverKey, enabled).then(() => {
                void loadAISessionManagement().then(() => {
                    renderAISessionManagement();
                });
            }).catch((err) => {
                notifyTerminal(`Failed to ${enabled ? 'load' : 'unload'} MCP server: ${err?.message || err}`, 'error');
                console.error(err);
                checkbox.checked = !enabled;
                void loadAISessionManagement().then(renderAISessionManagement);
            }).finally(() => { checkbox.disabled = false; });
            return;
        }

        if (configType !== 'subagent-tool') {
            return;
        }

        const toolName = String(checkbox.dataset.toolName || '').trim();
        if (!toolName) {
            return;
        }

        const allowed = checkbox.checked;
        SetAIToolSubagentAllowed(toolName, allowed).then(() => {
            const index = Number.parseInt(String(checkbox.dataset.toolIndex || ''), 10);
            if (Number.isInteger(index) && index >= 0 && index < state.aiToolsList.length) {
                state.aiToolsList[index] = {
                    ...state.aiToolsList[index],
                    allowInSubagent: allowed,
                };
            }
        }).catch((err) => {
            notifyTerminal(`Failed to update sub-agent access for ${toolName}`, 'error');
            console.error(err);
            checkbox.checked = !allowed;
        });
    });
}
if (elements.aiSettingsHistoryClear) {
    elements.aiSettingsHistoryClear.addEventListener('click', () => {
        ClearAISessionHistory().then((sessionState) => {
            applyAISessionManagement(sessionState);
            renderAISessionManagement();
            void loadAISessionCache('');
            notifyTerminal('AI session history cleared', 'info');
        }).catch((err) => {
            notifyTerminal('Failed to clear AI session history', 'error');
            console.error(err);
        });
    });
}
if (elements.toolsRestore) {
    elements.toolsRestore.addEventListener('click', () => setToolsPanelCollapsed(false));
}
if (elements.toolsTabAI) {
    elements.toolsTabAI.addEventListener('click', () => {
        setToolsTab('ai');
        void refreshAIModelPicker();
    });
}
if (elements.toolsTabToC) {
    elements.toolsTabToC.addEventListener('click', () => setToolsTab('toc'));
}
if (elements.toolsTabFrontmatter) {
    elements.toolsTabFrontmatter.addEventListener('click', () => setToolsTab('frontmatter'));
}
if (elements.toolsTabFind) {
    elements.toolsTabFind.addEventListener('click', () => {
        openFindBar();
    });
}

if (elements.toolsTabLog) {
    elements.toolsTabLog.addEventListener('click', () => setToolsTab('log'));
}

const notesLogPanel = initNotesLogPanel(elements, EventsOn);
GetNotesMaxLogLines().then((maxLogLines) => {
    notesLogPanel?.setMaxLogLines?.(maxLogLines);
}).catch((err) => {
    console.error('Failed to load notes log line limit:', err);
});

GetNotesStructViewMaxSizeKB().then((maxSizeKb) => {
    if (typeof maxSizeKb === 'number' && Number.isFinite(maxSizeKb) && maxSizeKb >= 0) {
        notesStructViewMaxSizeKB = maxSizeKb;
    }
}).catch((err) => {
    console.error('Failed to load notes structured view size limit:', err);
});

let notesGitPanel = null;
let notesGitDiffAdapter = null;
let activeGitDiff = null;

async function ensureNotesGitDiffAdapter() {
    if (notesGitDiffAdapter) {
        return notesGitDiffAdapter;
    }
    notesGitDiffAdapter = await createMonacoDiffAdapter(elements.gitDiffEditor, {
        ...getMonacoTypographyOptions(),
        themeColors: latestWindowStyle?.colors || null,
        onChange: (value) => {
            if (activeGitDiff?.diff?.readOnly) {
                return;
            }
            replaceMainDocumentText(String(value || ''), { source: 'notes-git-diff-edit' });
        },
        onSelectionChange: (start, end, activeOffset) => {
            mainDocument.setSelection(start, end);
            scheduleHistoryPosition(state.currentFile, activeOffset);
        },
        onPaste: (event) => {
            handleEditorImagePaste(event);
        },
        onPrimaryPaste: () => {
            if (state.viewMode !== 'editor' || state.currentFileType !== 'markdown') {
                return false;
            }
            void pasteFromGoClipboard(null, true);
            return true;
        },
        onContextMenu: (event) => {
            openMainEditorContextMenu(event);
        },
        onCompletionRequest: (payload) => {
            return handleMonacoCompletionRequest(payload);
        },
    });
    return notesGitDiffAdapter;
}

async function openNotesGitDiff({ root, file, diff }) {
    if (activeGitDiff && !await closeNotesGitDiff()) {
        return;
    }
    if (state.dirty && !await saveFile()) {
        return;
    }

    const relativePath = String(file.path || '').replaceAll('\\', '/');
    const notesFilePath = `$PROJECT/${relativePath}`;
    if (diff.deleted) {
        await closeOpenLspDocument();
        await closeCurrentTyposDocument();
        clearVisibleLspDiagnostics();
        activateNotesDocument(notesFilePath, { project: root });
        state.currentFileUri = '';
        state.currentFileType = 'code';
        elements.editorShell.dataset.fileType = 'code';
        state.swaggerSpec = null;
        state.swaggerRunAvailable = false;
        setCodeEditorMode(true);
        setMainEditorValue('');
        refreshEditorLanguage(notesFilePath, '');
        updateTabVisibility('code');
        setViewMode('editor');
        emitCurrentFileName();
        setDirty(false);
    } else {
        await loadFile(notesFilePath);
        if (state.currentFile !== notesFilePath) {
            return;
        }
    }
    await NotesHistoryAddPosition(notesFilePath, 1);

    activeGitDiff = { root, file, diff, notesFilePath };
    setNotesGitDiffTabActive(true);
    elements.gitDiffMessage.hidden = !diff.binary && !diff.readOnly;
    elements.gitDiffEditor.hidden = diff.binary;

    if (diff.binary) {
        elements.gitDiffMessage.textContent = 'Binary file; text diff is unavailable.';
    } else if (diff.readOnly) {
        elements.gitDiffMessage.textContent = 'Symbolic link; diff is read-only.';
        const editor = await ensureNotesGitDiffAdapter();
        editor.setDiff(diff);
        notesGitDiffAdapter = editor;
    } else {
        const editor = await ensureNotesGitDiffAdapter();
        editor.setDiff(diff);
        editor.setModifiedValue(String(diff.working || ''));
        editor.setWordWrap(Boolean(state.markdownWrapMode));
        editor.focus();
    }

    requestAnimationFrame(() => notesGitDiffAdapter?.layout());
}

async function saveActiveGitDiff() {
    if (!activeGitDiff || activeGitDiff.diff.readOnly) {
        return false;
    }
    return saveFile();
}

async function closeNotesGitDiff() {
    if (!activeGitDiff) {
        return true;
    }
    if (state.dirty && !await saveFile()) {
        return false;
    }
    const wasSelected = elements.tabDiff.getAttribute('aria-selected') === 'true';
    activeGitDiff = null;
    setNotesGitDiffTabActive(false);
    updateGitDiffTabForCurrentFile();
    if (wasSelected) {
        setViewMode(state.viewMode);
    }
    return true;
}

function setNotesGitDiffTabActive(active) {
    const isActive = active === true;
    focusedView().root.dataset.gitDiff = isActive ? 'true' : 'false';
    elements.gitDiffWrap.dataset.active = isActive ? 'true' : 'false';
    elements.tabDiff.style.display = activeGitDiff ? '' : 'none';
    const tabs = elements.tabs.querySelectorAll('[role="tab"]');
    for (const tab of tabs) {
        tab.setAttribute('aria-selected', tab === elements.tabDiff && isActive ? 'true' : 'false');
    }
}

async function checkoutNotesGitBranch(root, branch, create = false) {
    try {
        await CheckoutGitBranch(root, branch, create);
        notifyTerminal(`Switched to ${branch}`, 'info');
    } catch (error) {
        notifyTerminal(String(error?.message || error), 'error');
    }
}

function notifyGitTransfer(output, successMessage) {
    if (output.error) {
        notifyTerminal(output.stderr || output.stdout || output.error, 'error');
        return;
    }
    notifyTerminal(successMessage, 'info');
}

async function createNotesGitTagWorktree(root, tag, worktreePath) {
    const stickyId = `git-tag-worktree-${Date.now()}-${Math.random().toString(36).slice(2)}`;
    await openStickyProgress(stickyId, 'Creating worktree');
    try {
        const worktree = await CreateGitTagWorktree(root, tag, worktreePath);
        await switchNotesGitWorktree(root, worktree.path || worktreePath);
        closeStickyProgress(stickyId);
    } catch (error) {
        closeStickyProgress(stickyId, String(error?.message || error), 'error');
    }
}

elements.gitTags.addEventListener('click', async () => {
    const root = notesGitPanel?.getCurrentRoot();
    if (!root) return;
    try {
        const tags = await ListGitTags(root);
        const rect = elements.gitTags.getBoundingClientRect();
        const items = [
            {
                title: 'Create new tag', icon: CONTEXT_ICON_ADD,
                onSelect: async () => {
                    const tag = await openTextPrompt({ title: 'Create new tag', confirmLabel: 'Create', placeholder: 'v1.2.3' });
                    if (!tag?.trim()) return;
                    try {
                        await CreateGitTag(root, tag.trim());
                        notifyTerminal(`Created tag ${tag.trim()}`, 'info');
                    } catch (error) {
                        notifyTerminal(String(error?.message || error), 'error');
                    }
                },
            },
            { title: '-', icon: 0 },
            {
                title: 'Pull tags', icon: 0xf019,
                onSelect: async () => {
                    try {
                        notifyGitTransfer(await RunGitAction(root, 'pull-tags', {}), 'Tags pulled');
                    } catch (error) {
                        notifyTerminal(String(error?.message || error), 'error');
                    }
                },
            },
            { title: '-', icon: 0 },
            ...tags.map((tag) => ({
                title: tag,
                icon: 0,
                onSelect: () => showNotesLocalMenu([
                    {
                        title: 'Info', icon: 0xf129,
                        onSelect: async () => {
                            try {
                                await ShowGitRefInfo(root, tag, true);
                            } catch (error) {
                                notifyTerminal(String(error?.message || error), 'error');
                            }
                        },
                    },
                    { title: 'Copy tag name', icon: CONTEXT_ICON_COPY, onSelect: () => { void ClipboardSetText(tag).catch((error) => notifyTerminal(String(error), 'error')); } },
                    {
                        title: 'Push tag', icon: 0xf093,
                        onSelect: async () => {
                            try {
                                notifyGitTransfer(await RunGitAction(root, 'push-tag', { tag }), `Pushed tag ${tag}`);
                            } catch (error) {
                                notifyTerminal(String(error?.message || error), 'error');
                            }
                        },
                    },
                    { title: '-', icon: 0 },
                    {
                        title: 'Checkout', icon: 0xf126,
                        onSelect: async () => {
                            try {
                                await CheckoutGitTag(root, tag);
                                notifyTerminal(`Checked out tag ${tag}`, 'info');
                            } catch (error) {
                                notifyTerminal(String(error?.message || error), 'error');
                            }
                        },
                    },
                    {
                        title: 'Create new worktree', icon: CONTEXT_ICON_ADD,
                        onSelect: async () => {
                            try {
                                const base = await GetDefaultGitWorktreePath(root);
                                const path = `${String(base).replace(/[\\/]+$/, '')}/${tag}`;
                                await createNotesGitTagWorktree(root, tag, path);
                            } catch (error) {
                                notifyTerminal(String(error?.message || error), 'error');
                            }
                        },
                    },
                    { title: '-', icon: 0 },
                    {
                        title: 'Delete tag', icon: 0xf2ed,
                        onSelect: async () => {
                            try {
                                await DeleteGitTag(root, tag);
                                notifyTerminal(`Deleted tag ${tag}`, 'info');
                            } catch (error) {
                                notifyTerminal(String(error?.message || error), 'error');
                            }
                        },
                    },
                ], rect.left, rect.bottom, tag),
            })),
        ];
        showNotesLocalMenu(items, rect.left, rect.bottom, 'Tags');
    } catch (error) {
        notifyTerminal(String(error?.message || error), 'error');
    }
});

elements.gitBranches.addEventListener('click', async () => {
    const root = notesGitPanel?.getCurrentRoot();
    if (!root) return;
    try {
        const branches = await ListGitBranches(root);
        const rect = elements.gitBranches.getBoundingClientRect();
        const items = [
            {
                title: 'Create new branch', icon: CONTEXT_ICON_ADD,
                onSelect: async () => {
                    const branch = await openTextPrompt({ title: 'Create new branch', confirmLabel: 'Create', placeholder: 'branch-name' });
                    if (branch?.trim()) await checkoutNotesGitBranch(root, branch.trim(), true);
                },
            },
            { title: '-', icon: 0 },
            ...[...branches].sort((left, right) => left.localeCompare(right)).map((branch) => ({
                title: branch,
                icon: branch === notesGitPanel?.getCurrentStatus()?.branch ? CONTEXT_ICON_TICK : 0,
                onSelect: () => showNotesLocalMenu([
                    {
                        title: 'Info', icon: 0xf129,
                        onSelect: async () => {
                            try {
                                await ShowGitRefInfo(root, branch, false);
                            } catch (error) {
                                notifyTerminal(String(error?.message || error), 'error');
                            }
                        },
                    },
                    { title: 'Copy branch name', icon: CONTEXT_ICON_COPY, onSelect: () => { void ClipboardSetText(branch).catch((error) => notifyTerminal(String(error), 'error')); } },
                    { title: '-', icon: 0 },
                    { title: 'Checkout', icon: 0xf126, onSelect: () => checkoutNotesGitBranch(root, branch) },
                    {
                        title: 'Fetch latest', icon: 0xf019,
                        onSelect: async () => {
                            try {
                                const output = await RunGitAction(root, 'fetch-branch', { branch });
                                notifyGitTransfer(output, `Fetch completed for ${branch}`);
                            } catch (error) {
                                notifyTerminal(String(error?.message || error), 'error');
                            }
                        },
                    },
                    {
                        title: 'Create new worktree', icon: CONTEXT_ICON_ADD,
                        onSelect: async () => {
                            try {
                                const base = await GetDefaultGitWorktreePath(root);
                                await createNotesGitWorktree(root, branch, `${String(base).replace(/[\\/]+$/, '')}/${branch}`, false);
                            } catch (error) {
                                notifyTerminal(String(error?.message || error), 'error');
                            }
                        },
                    },
                    { title: '-', icon: 0 },
                    {
                        title: 'Delete branch', icon: 0xf2ed,
                        onSelect: async () => {
                            try {
                                await DeleteGitBranch(root, branch);
                                notifyTerminal(`Deleted branch ${branch}`, 'info');
                            } catch (error) {
                                notifyTerminal(String(error?.message || error), 'error');
                            }
                        },
                    },
                ], rect.left, rect.bottom, branch),
            })),
        ];
        showNotesLocalMenu(items, rect.left, rect.bottom, 'Branches');
    } catch (error) {
        notifyTerminal(String(error?.message || error), 'error');
    }
});

elements.toolsTabGit.addEventListener('click', () => {
    setToolsPanelCollapsed(false);
    setToolsTab('git');
});

disableNativeTextAssistanceIn(document);
attachSpellCheck(elements.gitCommitMessage);

notesGitPanel = initNotesGitPanel(elements, {
    getStatus: GetGitStatus,
    getBrowserLink: GetGitBrowserLink,
    getBranchDiffFiles: GetGitBranchDiffFiles,
    listCommits: ListGitCommits,
    listCommitsWithFiles: ListGitCommitsWithFiles,
    filterStrings: FilterStrings,
    openRemote: BrowserOpenURL,
    refreshStatus: RefreshGitStatus,
    setStatusWatch: SetGitStatusWatch,
    getDiff: GetGitDiff,
    getDiffAgainstBranch: GetGitDiffAgainstBranch,
    stageFile: StageGitFile,
    stageAll: StageAllGitFiles,
    unstageFile: UnstageGitFile,
    runGitAction: RunGitAction,
    confirmGitPush: ConfirmGitPush,
    getCurrentProject: GetCurrentProject,
    eventsOn: EventsOn,
}, {
    showTooltip: (text) => { void showSharedTooltip(text); },
    hideTooltip: closeSharedTooltip,
    notify: notifyTerminal,
    notifyTransfer: notifyGitTransfer,
    onOpenDiff: openNotesGitDiff,
    showCommitMenu: ({ root, commit, x, y, compare }) => {
        const fullHash = commit.fullHash || commit.hash;
        const items = [
            {
                title: 'Info', icon: 0xf129,
                onSelect: async () => {
                    try {
                        await ShowGitRefInfo(root, fullHash, false);
                    } catch (error) {
                        notifyTerminal(String(error?.message || error), 'error');
                    }
                },
            },
            { title: 'Copy commit hash', icon: CONTEXT_ICON_COPY, onSelect: () => ClipboardSetText(fullHash) },
            { title: '-', icon: 0 },
            { title: 'Compare commit', icon: 0xe13a, onSelect: compare },
            {
                title: 'Create worktree', icon: CONTEXT_ICON_ADD,
                onSelect: async () => {
                    try {
                        const defaultPath = await GetDefaultGitWorktreePath(root);
                        const worktreePath = `${String(defaultPath).replace(/[\\/]+$/, '')}/${commit.hash}`;
                        await createNotesGitWorktree(root, fullHash, worktreePath, false);
                    } catch (error) {
                        notifyTerminal(String(error?.message || error), 'error');
                    }
                },
            },
        ];
        showNotesLocalMenu(items, x, y, commit.hash);
    },
    getCommitFilter: () => state.fileFilterQuery,
    onCommitsChange: (active) => {
        const wasActive = !elements.gitCommitList.hidden;
        elements.list.hidden = active;
        elements.gitCommitList.hidden = !active;
        elements.listFilter.placeholder = active
            ? elements.gitFilenameSearch.checked ? 'Filter commits by filename...' : 'Filter commits...'
            : 'Filter files...';
        if (!active && wasActive) {
            if (isProjectFindListModeActive()) scheduleProjectFindSearch();
            else void applyFileFilter();
        }
    },
    beforeFileAction: async ({ root, path }) => {
        if (!activeGitDiff || activeGitDiff.root !== root || activeGitDiff.file.path !== path || !state.dirty) {
            return true;
        }
        return saveActiveGitDiff();
    },
    onStatusChange: (status, root) => {
        updateGitDiffTabForCurrentFile(status, root);
    },
    showWorkspaceMenu: async (root, button) => {
        let worktrees;
        try {
            worktrees = await ListGitWorktrees(root);
        } catch (error) {
            notifyTerminal(String(error?.message || error), 'error');
            return;
        }
        const rect = button.getBoundingClientRect();
        const activeRoot = String(notesGitPanel?.getCurrentRoot() || root).replace(/\\/g, '/').replace(/\/$/, '');
        const items = [{ title: 'New worktree', icon: CONTEXT_ICON_ADD, onSelect: () => void openCreateGitWorktreePrompt(root) }, { title: '-', icon: 0 }];
        items.push(...[...worktrees]
            .sort((left, right) => left.branch.localeCompare(right.branch))
            .map((worktree) => ({
                title: worktree.branch || '(detached HEAD)',
                icon: String(worktree.path || '').replace(/\\/g, '/').replace(/\/$/, '') === activeRoot ? CONTEXT_ICON_TICK : 0,
                onSelect: () => {
                    const actionItems = [
                        {
                            title: 'Info',
                            icon: 0xf129,
                            onSelect: async () => {
                                try {
                                    await ShowGitWorktreeInfo(root, worktree.path);
                                } catch (error) {
                                    notifyTerminal(String(error?.message || error), 'error');
                                }
                            },
                        },
                        {
                            title: 'Switch',
                            icon: 0xf204,
                            onSelect: async () => {
                                try {
                                    await switchNotesGitWorktree(root, worktree.path);
                                } catch (error) {
                                    notifyTerminal(String(error?.message || error), 'error');
                                }
                            },
                        },
                        { title: '-', icon: 0 },
                        {
                            title: 'Remove',
                            icon: CONTEXT_ICON_DELETE,
                            onSelect: async () => {
                                try {
                                    await RemoveGitWorktree(root, worktree.path);
                                    notifyTerminal(`Worktree removed: ${worktree.branch || worktree.path}`, 'info');
                                } catch (error) {
                                    notifyTerminal(String(error?.message || error), 'error');
                                }
                            },
                        },
                    ];
                    showLocalMenu({
                        title: worktree.branch || '(detached HEAD)',
                        options: actionItems.map((item) => item.title),
                        icons: actionItems.map((item) => item.icon),
                        x: rect.left,
                        y: rect.bottom,
                        showNextToMouseCursor: true,
                        onSelect: (index) => actionItems[index]?.onSelect?.(),
                    });
                },
            })));
        showLocalMenu({
            title: 'Worktrees',
            options: items.map((item) => item.title),
            icons: items.map((item) => item.icon),
            x: rect.left,
            y: rect.bottom,
            showNextToMouseCursor: true,
            onSelect: (index) => items[index]?.onSelect?.(),
        });
    },
    showCompareMenu: async (root, currentBranch, selectedBranch, button, selectBranch) => {
        let branches;
        try {
            branches = await ListGitBranches(root);
        } catch (error) {
            notifyTerminal(String(error?.message || error), 'error');
            return;
        }
        const rect = button.getBoundingClientRect();
        const items = [
            {
                title: `HEAD (${currentBranch || 'Detached HEAD'})`,
                icon: selectedBranch === '' ? CONTEXT_ICON_TICK : 0,
                onSelect: () => void selectBranch(''),
            },
            { title: '-', icon: 0 },
            ...branches.filter((branch) => branch !== currentBranch).sort((left, right) => left.localeCompare(right)).map((branch) => ({
                title: branch,
                icon: branch === selectedBranch ? CONTEXT_ICON_TICK : 0,
                onSelect: () => void selectBranch(branch),
            })),
        ];
        showLocalMenu({
            title: 'Compare against',
            options: items.map((item) => item.title),
            icons: items.map((item) => item.icon),
            x: rect.left,
            y: rect.bottom,
            showNextToMouseCursor: true,
            onSelect: (index) => items[index]?.onSelect?.(),
        });
    },
});

initNotesAIPanel(elements);
initAIPromptJumpObserver();
void refreshAIPromptJumpFromBackend();
if (elements.toolsToC) {
    elements.toolsToC.addEventListener('click', (event) => {
        const tocButton = event.target.closest('.notes-tools-toc-item');
        if (!tocButton) {
            return;
        }

        if (state.currentFileType === 'markdown' && state.viewMode === 'editor') {
            let line = Number.parseInt(String(tocButton.dataset.sourceLine || ''), 10);
            if (!Number.isFinite(line) || line < 1) {
                const level = Number.parseInt(String(tocButton.dataset.level || '1'), 10) || 1;
                const text = String(tocButton.dataset.text || tocButton.textContent || '').trim().toLowerCase();
                const occurrence = Number.parseInt(String(tocButton.dataset.occurrence || '0'), 10) || 0;
                const matches = getMarkdownHeadingSourceDescriptors(mainDocument.getValue())
                    .filter((heading) => heading.level === level && heading.text.toLowerCase() === text);
                line = matches[occurrence]?.sourceLine;
            }
            if (Number.isFinite(line) && line > 0) {
                jumpEditorToLine(line);
            }
            return;
        }

        if (state.viewMode !== 'viewer' && state.viewMode !== 'jupyter') {
            return;
        }

        scrollToToolsToCHeading(tocButton);
    });
}

setToolsTab('find');

// Always start minimized on application launch.
setToolsPanelCollapsed(true);

function applyWindowStyle(result) {
    latestWindowStyle = result;

    document.body.style.color = `rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue})`;
    document.body.style.backgroundColor = `rgb(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue})`;

    const notesStatusFontSize = result.fontSize - 2;
    const notesTitleFontSize = result.fontSize + 4;

    let style = document.getElementById('notes-theme-style');
    if (!style) {
        style = document.createElement('style');
        style.id = 'notes-theme-style';
        document.head.appendChild(style);
    }

    style.textContent = `
        :root {
            --bg: rgb(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue});
            --bg-rgb: ${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue};
            --fg: rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue});
            --fg-rgb: ${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue};
            --accent: rgb(${result.colors.accent.Red}, ${result.colors.accent.Green}, ${result.colors.accent.Blue});
            --accent-rgb: ${result.colors.accent.Red}, ${result.colors.accent.Green}, ${result.colors.accent.Blue};
            --link: rgb(${result.colors.link.Red}, ${result.colors.link.Green}, ${result.colors.link.Blue});
            --link-rgb: ${result.colors.link.Red}, ${result.colors.link.Green}, ${result.colors.link.Blue};
            --red: rgb(${result.colors.red.Red}, ${result.colors.red.Green}, ${result.colors.red.Blue});
            --red-rgb: ${result.colors.red.Red}, ${result.colors.red.Green}, ${result.colors.red.Blue};
            --green: rgb(${result.colors.green.Red}, ${result.colors.green.Green}, ${result.colors.green.Blue});
            --green-rgb: ${result.colors.green.Red}, ${result.colors.green.Green}, ${result.colors.green.Blue};
            --yellow: rgb(${result.colors.yellow.Red}, ${result.colors.yellow.Green}, ${result.colors.yellow.Blue});
            --yellow-rgb: ${result.colors.yellow.Red}, ${result.colors.yellow.Green}, ${result.colors.yellow.Blue};
            --blue: rgb(${result.colors.blue.Red}, ${result.colors.blue.Green}, ${result.colors.blue.Blue});
            --blue-rgb: ${result.colors.blue.Red}, ${result.colors.blue.Green}, ${result.colors.blue.Blue};
            --magenta: rgb(${result.colors.magenta.Red}, ${result.colors.magenta.Green}, ${result.colors.magenta.Blue});
            --magenta-rgb: ${result.colors.magenta.Red}, ${result.colors.magenta.Green}, ${result.colors.magenta.Blue};
            --cyan: rgb(${result.colors.cyan.Red}, ${result.colors.cyan.Green}, ${result.colors.cyan.Blue});
            --cyan-rgb: ${result.colors.cyan.Red}, ${result.colors.cyan.Green}, ${result.colors.cyan.Blue};
            --red-bright: rgb(${result.colors.redBright.Red}, ${result.colors.redBright.Green}, ${result.colors.redBright.Blue});
            --red-bright-rgb: ${result.colors.redBright.Red}, ${result.colors.redBright.Green}, ${result.colors.redBright.Blue};
            --green-bright: rgb(${result.colors.greenBright.Red}, ${result.colors.greenBright.Green}, ${result.colors.greenBright.Blue});
            --green-bright-rgb: ${result.colors.greenBright.Red}, ${result.colors.greenBright.Green}, ${result.colors.greenBright.Blue};
            --yellow-bright: rgb(${result.colors.yellowBright.Red}, ${result.colors.yellowBright.Green}, ${result.colors.yellowBright.Blue});
            --yellow-bright-rgb: ${result.colors.yellowBright.Red}, ${result.colors.yellowBright.Green}, ${result.colors.yellowBright.Blue};
            --blue-bright: rgb(${result.colors.blueBright.Red}, ${result.colors.blueBright.Green}, ${result.colors.blueBright.Blue});
            --blue-bright-rgb: ${result.colors.blueBright.Red}, ${result.colors.blueBright.Green}, ${result.colors.blueBright.Blue};
            --magenta-bright: rgb(${result.colors.magentaBright.Red}, ${result.colors.magentaBright.Green}, ${result.colors.magentaBright.Blue});
            --magenta-bright-rgb: ${result.colors.magentaBright.Red}, ${result.colors.magentaBright.Green}, ${result.colors.magentaBright.Blue};
            --cyan-bright: rgb(${result.colors.cyanBright.Red}, ${result.colors.cyanBright.Green}, ${result.colors.cyanBright.Blue});
            --cyan-bright-rgb: ${result.colors.cyanBright.Red}, ${result.colors.cyanBright.Green}, ${result.colors.cyanBright.Blue};
            --selection: rgb(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue});
            --selection-rgb: ${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue};
            --error: rgb(${result.colors.error.Red}, ${result.colors.error.Green}, ${result.colors.error.Blue});
            --error-rgb: ${result.colors.error.Red}, ${result.colors.error.Green}, ${result.colors.error.Blue};
            --font-family: ${result.fontFamily};
            --notes-font-size: ${result.fontSize}px;
            --notes-font-size-minus-1: ${result.fontSize - 1}px;
            --notes-font-size-minus-2: ${result.fontSize - 2}px;
            --notes-font-size-plus-4: ${result.fontSize + 4}px;
            --notes-font-size-plus-7: ${result.fontSize + 7}px;
            --notes-status-font-size: ${notesStatusFontSize}px;
            --notes-title-font-size: ${notesTitleFontSize}px;
            --darken-background-overlay: ${DARKEN_BACKGROUND_OVERLAY};
        }

        ${getScrollbarStyles(result.colors)}
        ${getMarkdownBaseTextSizeStyles('.notes-view .notes-preview', result.fontSize)}
        ${getMarkdownBaseTextSizeStyles('.notes-view .notes-jupyter', result.fontSize)}
        ${getMarkdownBaseTextSizeStyles('.notes-view .notes-meta', result.fontSize)}
        ${getMarkdownBaseTextSizeStyles('.notes-view .notes-csv-view', result.fontSize)}
        ${getMarkdownBaseTextSizeStyles('.notes-view .notes-swagger-info', result.fontSize)}
        ${getMarkdownBaseTextSizeStyles('.notes-view .notes-swagger-run-wrap', result.fontSize)}
        ${getMarkdownBaseTextSizeStyles('.notes-view .notes-swagger-request-builder .swagger-param-description', result.fontSize)}
        ${getMarkdownContentStyles(result.colors, result.fontSize, 'markdown-body')}
        ${getMarkdownContentStyles(result.colors, result.fontSize, 'swagger-ui')}
        ${getCheckboxStyles(result.colors, result.fontSize, 'markdown-body')}
        ${getHighlightJsTheme(result.colors, true)}
        ${getHexDumpStyles(result.fontSize, result.adjustCellHeight)}
        ${getSwaggerUIStyles(result.colors, result.fontSize)}
    `;

    for (const view of notesViewList()) {
        view.editor?.setTypography(getMonacoTypographyOptions());
        view.editor?.applyTheme(result.colors);
    }
}

GetWindowStyle().then((result) => {
    applyWindowStyle(result);
});

EventsOn('terminalStyleUpdate', payload => {
    const result = Array.isArray(payload?.[0]) ? payload[0] : payload;
    if (result && result.colors) {
        applyWindowStyle(result);
    }
});

bindSharedTooltipMouseTracking();

// ----------------------------------------------------------------------------
// Hyperlink tooltips
// ----------------------------------------------------------------------------

const notesHyperlinkTooltip = createHyperlinkTooltip({
    getTooltip: GetHyperlinkTooltip,
    showTooltip: showSharedTooltip,
    closeTooltip: closeSharedTooltip,
    reportError: (error) => console.error('Unable to render hyperlink tooltip:', error),
});

function hideHyperlinkHoverTooltip() {
    notesHyperlinkTooltip.close();
}

function positionHyperlinkHoverTooltip(x, y) {
    updateSharedTooltipPointer(Number(x) || 0, Number(y) || 0);
}

function formatHyperlinkHoverHref(href) {
    const value = String(href || '').trim();
    if (!value) {
        return '';
    }

    if (/^ttyphoon:\/\/ai(?:[/?#]|$)/i.test(value)) {
        try {
            const parsed = new URL(value);
            const lines = ['New AI query'];

            for (const [key, queryValue] of parsed.searchParams.entries()) {
                const safeKey = String(key || '').trim();
                if (!safeKey) {
                    continue;
                }

                lines.push(`- ${safeKey}: ${String(queryValue || '')}`);
            }

            return lines.join('\n');
        } catch {
            return 'New AI query';
        }
    }

    return value.startsWith('wails://wails/') ? value.slice('wails://wails/'.length) : value;
}

function showHyperlinkHoverTooltip(anchor, x, y) {
    if (!(anchor instanceof HTMLAnchorElement)) {
        return;
    }

    const href = String(anchor.href || anchor.getAttribute('href') || '').trim();
    const displayHref = formatHyperlinkHoverHref(href);
    if (!displayHref) {
        hideHyperlinkHoverTooltip();
        return;
    }

    positionHyperlinkHoverTooltip(x, y);
    void notesHyperlinkTooltip.show(anchor, href, displayHref);
}

function filePathToUri(filePath) {
    const source = String(filePath || '').trim();
    if (!source) {
        return '';
    }

    const normalized = source.replace(/\\/g, '/');
    const prefixed = normalized.startsWith('/') ? normalized : `/${normalized}`;

    try {
        return new URL(`file://${encodeURI(prefixed)}`).toString();
    } catch {
        return `file://${prefixed}`;
    }
}

async function resolveNotesFileUri(filePath) {
    if (!filePath) {
        return '';
    }

    try {
        const resolved = await ResolveFilePath(filePath);
        return filePathToUri(resolved || filePath);
    } catch {
        return filePathToUri(filePath);
    }
}

document.addEventListener('scroll', () => {
    hideHyperlinkHoverTooltip();
}, true);

function insertEditorText(text, target) {
    if (!text) {
        return;
    }

    target.focus();
    document.execCommand('insertText', false, text);
}

async function savePastedImageDataUrl(dataUrl, mimeType) {
    if (!state.currentFile) {
        notifyTerminal('Select a note before pasting an image', 'info');
        return;
    }

    const comma = dataUrl.indexOf(',');
    if (comma <= 0 || comma >= dataUrl.length - 1) {
        notifyTerminal('Clipboard image format is invalid', 'error');
        return;
    }

    const base64Payload = dataUrl.slice(comma + 1);
    const epoch = Math.floor(Date.now() / 1000);
    const ext = deriveImageExtension(mimeType || 'image/png');
    const paths = buildImagePaths(state.currentFile, epoch, ext);

    try {
        await SaveBinaryFile(paths.imagePath, base64Payload);

        const alt = String(epoch);
        const markdownPath = paths.imageFileName.split('/').map(encodeURIComponent).join('/');
        const markdownImage = `![${alt}](${markdownPath})`;
        insertTextInMainEditor(markdownImage);
        notifyTerminal(`Saved image ${paths.imageFileName}`, 'info');
    } catch (err) {
        notifyTerminal('Failed to save pasted image', 'error');
        console.error(err);
    }
}

function handleEditorImagePaste(event) {
    if (state.viewMode !== 'editor') {
        return;
    }

    const items = event.clipboardData && event.clipboardData.items;
    if (!items) {
        return;
    }

    for (const item of items) {
        if (!item.type.startsWith('image/')) {
            continue;
        }

        event.preventDefault();

        const file = item.getAsFile();
        if (!file) {
            return;
        }

        const reader = new FileReader();
        reader.onload = async (e) => {
            const dataUrl = String(e.target.result || '');
            await savePastedImageDataUrl(dataUrl, file.type);
        };
        reader.readAsDataURL(file);

        // Only handle the first image item
        return;
    }
}

function decodeClipboardPayload(payload) {
    if (!payload || typeof payload !== 'object') {
        return { text: '', image: '' };
    }

    return {
        text: String(payload.text || ''),
        image: String(payload.image || ''),
    };
}

// targetEditor is a Jupyter/Run textarea; null pastes into the main document.
async function pasteFromGoClipboard(targetEditor = null, allowImagePaste = true) {
    try {
        const payload = await GetClipboardData();
        const { text, image } = decodeClipboardPayload(payload);

        if (allowImagePaste && image !== '') {
            const dataUrl = `data:image/png;base64,${image}`;
            await savePastedImageDataUrl(dataUrl, 'image/png');
            return;
        }

        if (text !== '') {
            if (!targetEditor) {
                insertTextInMainEditor(text);
            } else {
                insertEditorText(text, targetEditor);
            }
        }
    } catch (err) {
        notifyTerminal('Failed to paste from clipboard.', 'error');
        console.error(err);
    }
}

let notesSpellCheckHandle = null;

// Per-block spellcheck overlay handles for Jupyter code blocks, keyed by blockId.
const jupyterSpellCheckHandles = {};

initNotesLsp({
    CONTEXT_ICON_CODE,
    CONTEXT_ICON_FIND,
    elements,
    jupyterSpellCheckHandles,
    mainDocument,
    notesMutationAdapter,
    state,
    addHistoryPosition,
    applyMonacoLineIndentationForSelection,
    applyMonacoTabIndent,
    currentEditorToken,
    escapeHtml,
    fileUriToPath,
    getMainEditorSelectionRange,
    getMainEditorValue,
    isMonacoActive,
    isSvgFile,
    jumpEditorToOffset,
    loadFile,
    normalizePathForMatch,
    notifyTerminal,
    openTextPrompt,
    replaceMainDocumentText,
    resolveNotesFileFromAbsolutePath,
    runProjectFindSearch,
    setMainEditorSelectionRange,
    setMainEditorValue,
    setProjectReferencesResults,
    setToolsPanelCollapsed,
    setToolsTab,
    shouldMonacoTabIndent,
    showNotesLocalMenu,
    updateFindOptionButtons,
    getMonacoMainEditor: () => focusedView().editor,
    getFocusedEditorHost: () => focusedView().parts.monacoEditor,
    getNotesSpellCheckHandle: () => notesSpellCheckHandle,
    getActiveNotesDocument: () => activeNotesDocument,
    isNotesDocumentOpen,
});

async function refreshMainReservedWords(file) {
    let words = [];
    try {
        const language = await ResolveNotesLspLanguage(file);
        words = language ? (await GetNotesLanguageReservedWords(language)) || [] : [];
    } catch (error) {
        console.error('Loading document reserved words failed:', error);
    }
    if (state.currentFile !== file) {
        return;
    }
    lspSpellCheckExclusions.keywords = Array.isArray(words) ? words : [];
    lspSpellCheckExclusions.tokens = [];
    applyLspSpellCheckExclusions();
}

async function refreshJupyterReservedWords(blockId) {
    const block = state.jupyterCodeBlocks[blockId];
    const handle = jupyterSpellCheckHandles[blockId];
    if (!block || !handle) {
        return;
    }
    const sequence = (block.reservedWordsSequence || 0) + 1;
    block.reservedWordsSequence = sequence;
    const language = block.runtime || block.language;
    let words = [];
    try {
        words = language ? (await GetNotesLanguageReservedWords(language)) || [] : [];
    } catch (error) {
        console.error('Loading code-block reserved words failed:', error);
    }
    if (state.jupyterCodeBlocks[blockId] !== block || jupyterSpellCheckHandles[blockId] !== handle || block.reservedWordsSequence !== sequence) {
        return;
    }
    handle.setExclusions(Array.isArray(words) ? words : []);
}

mainDocument.onChange(() => {
    if (isGitDiffActive() && notesGitDiffAdapter.getModifiedValue() !== mainDocument.getValue()) {
        notesGitDiffAdapter.setModifiedValue(mainDocument.getValue());
    }

    state.lastEditorInputAt = Date.now();
    setDirty(true);
    state.lspHoverLastKey = '';
    hideLspHoverTooltip();
    clearCurrentFileLspDiagnosticsCache();
    clearVisibleLspDiagnostics({ preserveCompletion: true });

    // Trigger characters open the menu via Monaco's onDidType (see
    // handleMonacoCompletionRequest); the caret is not final yet at this point.
    if (state.lspCompletionVisible) {
        // Re-filter the open menu against the text now under the caret.
        void requestLspCompletionAfterSync(mainDocument.getValue(), '', 1);
    }

    if (isMonacoActive()) {
        // In Monaco (Edit) mode, defer view regeneration until the View tab is
        // clicked.  This avoids expensive re-renders on every keystroke /
        // autosave cycle, which is especially noticeable for JSON/YAML files.
        state.viewTainted = true;
    } else if (usesCodeEditorDecorations()) {
        refreshEditorLanguage(state.currentFile, mainDocument.getValue());
        if (state.currentFileType === 'markdown' || state.currentFileType === 'html') {
            scheduleRender();
        }
    } else if (state.currentFileType === 'csv') {
        renderCsvView(mainDocument.getValue(), { interactive: state.viewMode === 'csv-run' });
    } else {
        scheduleRender();
    }

    if (state.currentFileType === 'json') {
        // Revalidate JSON/YAML and only expose Run for docs with a swagger key.
        // The tree view render itself is deferred to the View tab click.
        state.swaggerSpec = parseStructuredDocument(state.currentFile, mainDocument.getValue());
        state.swaggerRunAvailable = hasSwaggerKey(state.swaggerSpec);
        state.swaggerViewCurrent = false;
        updateTabVisibility('json');
        if (!state.swaggerRunAvailable && state.viewMode === 'swagger-run') {
            setViewMode('swagger-view');
        } else if (state.swaggerRunAvailable && state.viewMode === 'swagger-run' && !isMonacoActive()) {
            renderSwaggerUI();
        }
        if (state.viewMode === 'swagger-view' && !isMonacoActive()) {
            renderSwaggerJsonViewLazy();
        }
    }
    scheduleLspDidChange();
    scheduleTyposDidChange();
    scheduleAutoSave();
    notesSpellCheckHandle.scheduleCheck();
    scheduleCurrentHistoryPosition();
});

notesSpellCheckHandle = createSpellCheckEngine({
    getText: () => mainDocument.getValue(),
    onMisspellingsChange: (misspellings) => {
        if (!isMonacoActive()) {
            return;
        }
        focusedView().editor.setTyposMisspellings(Array.isArray(misspellings) ? misspellings : []);
    },
});
notesSpellCheckHandle.scheduleCheck();

function openMainEditorContextMenu(e) {
    e.preventDefault();

    const menuItems = [
        createCopyMenuItem(() => getMainEditorSelectionText(), 'Copy'),
        {
            title: 'Paste',
            icon: CONTEXT_ICON_PASTE,
            onSelect: async () => {
                await pasteFromGoClipboard(null, !isStructuredDataFile(state.currentFile));
            },
        },
    ];

    const isCodeLike = state.currentFileType === 'code' || state.currentFileType === 'markdown' || state.currentFileType === 'json' || state.currentFileType === 'html';
    const isStructuredEdit = state.currentFileType === 'json' && state.viewMode === 'swagger-edit';
    

    if (isJsonStructuredFile(state.currentFile)) {
        menuItems.push(
        { title: '-' },
        {
            title: 'Format: Minify',
            icon: 0,
            onSelect: () => {
                formatStructuredEditorJson(false);
            },
        },
        {
            title: 'Format: Expand All',
            icon: 0,
            onSelect: () => {
                formatStructuredEditorJson(true);
            },
        });
    }

    if (isCodeLike && (state.viewMode === 'editor' || isStructuredEdit)) {
        menuItems.push(
            { title: '-' },
            {
                title: 'Word wrap',
                icon: state.markdownWrapMode ? 0xf00c : 0x20,
                onSelect: () => {
                    toggleMarkdownWrapMode();
                },
            },
        );

    }

    if (state.currentFileType === 'markdown') {
        menuItems.push(
            { title: '-' },
            {
                title: 'Insert checkbox',
                icon: CONTEXT_ICON_CHECKBOX,
                onSelect: () => {
                    insertTextAtMainEditorLineStart('- [ ] ');
                },
            },
            {
                title: 'Insert code block',
                icon: CONTEXT_ICON_CODE,
                onSelect: () => {
                    const range = getMainEditorSelectionRange();
                    const selected = getMainEditorSelectionText();
                    replaceMainEditorRange(range.start, range.end, '```\n' + selected + '\n```');
                    setMainEditorSelectionRange(range.start + 3, range.start + 3);
                },
            },
            {
                title: 'Insert table 3x1',
                icon: CONTEXT_ICON_TABLE,
                onSelect: () => {
                    insertTextInMainEditor('| A | B | C |\n| --- | --- | --- |\n| cell | cell | cell |\n');
                },
            },
            {
                title: 'Update Table of Contents',
                icon: 0xf0ae,
                onSelect: () => {
                    updateMarkdownTableOfContents();
                },
            },
        );

        const cursor = getMainEditorSelectionRange().start;
        const imageAtCursor = getMarkdownImageAtCursor(getMainEditorValue(), cursor);
        if (state.currentFile && imageAtCursor && isRelativeMarkdownImagePath(imageAtCursor.imagePath)) {
            menuItems.push(
            { title: '-' },
            {
                title: 'Delete image from disk',
                icon: CONTEXT_ICON_DELETE,
                onSelect: async () => {
                    const imageDiskPath = resolveRelativeAssetPath(state.currentFile, imageAtCursor.imagePath);

                    try {
                        await DeleteFile(imageDiskPath);

                        replaceMainEditorRange(imageAtCursor.markdownStart, imageAtCursor.markdownEnd, '');
                        notifyTerminal(`Deleted image ${imageAtCursor.imagePath}.`, 'info');
                    } catch (err) {
                        notifyTerminal(`Failed to delete image ${imageAtCursor.imagePath}.`, 'error');
                        console.error(err);
                    }
                },
            });
        }
    }

    if (isCurrentFileLspEligible()) {
        menuItems.push(
            { title: '-' },
            {
                title: 'LSP options...',
                icon: CONTEXT_ICON_CODE,
                onSelect: async () => {
                    await showEditorLspOptionsMenu(e.clientX, e.clientY);
                },
            },
        );
    }

    menuItems.push(
        { title: '-' },
        {
            title: 'Find references',
            icon: CONTEXT_ICON_FIND,
            onSelect: () => {
                void findReferencesFromEditor();
            },
        },
        createFindMenuItem('Find text...'),
        createAskAIDocumentMenuItem(),
        createPrintMenuItem('Print...'),
    );

    if (isMonacoActive()) {
        menuItems.push(
            { title: '-' },
            {
                title: 'Editor options',
                icon: CONTEXT_ICON_CODE,
                onSelect: () => {
                    setTimeout(() => {
                        if (isGitDiffActive()) {
                            notesGitDiffAdapter?.openCommandPalette?.();
                        } else {
                            focusedView().editor?.openCommandPalette?.();
                        }
                    }, 0);
                },
        });
    }

    showNotesLocalMenu(menuItems, e.clientX, e.clientY);
}

initRenderedNotesContextMenu(lspHoverTooltipEl, 'viewer');
initRenderedNotesContextMenu(lspTooltipEl, 'viewer');
initAIOutputContextMenu(elements.aiOutput);

// The caller's modal ID keys saved table state, as a file path does for documents.
initMarkdownModal({
    copyMenuItem: (container) => createRenderedCopyMenuItem(container),
    onPrint: (markdown) => {
        void printDocument({
            source: markdown,
            renderMarkdown: renderPrintMarkdown,
            dismissMenu: closeSharedTooltip,
            print: WindowPrint,
        }).catch(reportPrintError);
    },
    onTablesRendered: (container, { id, wrapped }) => {
        void setupTableColumnResizing(container, wrapped, id, 'markdown-modal');
        setupNotesTableQueries(container, 'markdown-modal', id);
    },
    tableMenuItems: (table) => [
        createClearTableSortMenuItem(table),
        createTableFilterMenuItem(table),
        createTableGraphMenuItem(table),
    ],
});

initStructuredDataTreeContextMenu(elements.toolsFrontmatter, {
    isActive: () => state.currentFileType === 'markdown' && state.frontmatter != null,
    getRoot: () => state.frontmatter,
    menuTitle: 'Frontmatter field',
});

// Binds a view's own controls. Events only reach the focused view's content
// (see bindNotesViewFocus), so handlers act on the active document and the
// focused view's parts.
function bindNotesView(view) {
    bindNotesViewTabs(view);
    bindNotesViewFocus(view);
    observeNotesViewEditorSize(view);
    if (view.parts.previewWrap) {
        view.parts.previewWrap.addEventListener('scroll', () => {
            syncToolsToCHighlightForMode();
        });
    }
    if (view.parts.jupyterWrap) {
        view.parts.jupyterWrap.addEventListener('scroll', () => {
            syncToolsToCHighlightForMode();
        });
    }
    initRenderedNotesContextMenu(view.parts.preview, 'viewer');
    initRenderedNotesContextMenu(view.parts.jupyter, 'jupyter');
    initRenderedNotesContextMenu(view.parts.swaggerRunWrap, 'swagger-run');
    view.parts.csvView.addEventListener('contextmenu', (e) => {
        e.preventDefault();
        const table = e.target instanceof Element ? e.target.closest('table') : null;
        if (!table || !view.parts.csvView.contains(table)) return;
        const menuItems = [
            ...createTableCopyMenuItems(table),
            { title: '-' },
            createClearTableSortMenuItem(table),
            createTableFilterMenuItem(table),
            createTableGraphMenuItem(table),
        ];
        const isRunMode = state.viewMode === 'csv-run';
        if (isRunMode) {
            const insertItems = createTableInsertMenuItems(table, e.target, 0);
            if (insertItems.length > 0) {
                menuItems.push({ title: '-' }, ...insertItems);
            }
        }

        menuItems.push({ title: '-' }, createAskAIDocumentMenuItem());

        showNotesLocalMenu(menuItems, e.clientX, e.clientY);
    });
    initStructuredDataTreeContextMenu(view.parts.swaggerView, {
        isActive: () => state.viewMode === 'swagger-view',
        getRoot: () => state.swaggerSpec ?? parseStructuredDocument(state.currentFile, mainDocument.getValue()),
        menuTitle: 'Document',
    });
    view.parts.tabEditor.addEventListener('click', () => {
        setViewMode('editor');
    });

    view.parts.tabHex.addEventListener('click', () => {
        setViewMode('hex');
    });

    view.parts.tabViewer.addEventListener('click', () => {
        setViewMode('viewer');
        if (state.viewTainted && state.currentFileType === 'markdown') {
            state.viewTainted = false;
            scheduleRender();
        }
    });

    view.parts.tabJupyter.addEventListener('click', () => {
        setViewMode('jupyter');
        renderJupyterView();
    });

    view.parts.tabSwaggerView.addEventListener('click', () => {
        setViewMode('swagger-view');
        state.viewTainted = false;
        renderSwaggerJsonViewLazy();
    });

    view.parts.tabSwaggerEdit.addEventListener('click', () => {
        setViewMode('swagger-edit');
    });

    view.parts.tabSwaggerRun.addEventListener('click', () => {
        setViewMode('swagger-run');
        updateSwaggerLayoutMode();
        renderSwaggerUI();
    });

    view.parts.tabImageView.addEventListener('click', async () => {
        if (state.currentFileType === 'code' && isSvgFile(state.currentFile)) {
            if (state.dirty && !await saveFile()) {
                return;
            }
            try {
                await configureImagePreview(state.currentFile, getPathFileName(state.currentFile), true);
            } catch (err) {
                notifyTerminal(`Failed to load image: ${getPathFileName(state.currentFile)}`, 'error');
                console.error(err);
                return;
            }
        }
        setViewMode('image-view');
    });

    view.parts.tabCsvView.addEventListener('click', () => {
        setViewMode('csv-view');
    });

    view.parts.tabCsvEdit.addEventListener('click', () => {
        setViewMode('csv-edit');
    });

    view.parts.tabCsvRun.addEventListener('click', () => {
        setViewMode('csv-run');
    });

    view.parts.tabMeta.addEventListener('click', () => {
        setViewMode('meta');
    });

    view.parts.tabDiff.addEventListener('click', () => {
        if (activeGitDiff) {
            setNotesGitDiffTabActive(true);
            notesGitDiffAdapter?.focus();
            return;
        }
        const filePath = currentProjectRelativeGitPath();
        const projectRoot = state.currentFileProject || notesGitPanel?.getCurrentRoot();
        if (filePath && projectRoot) {
            void notesGitPanel?.openFileDiff(filePath, projectRoot);
        }
    });
    if (typeof ResizeObserver !== 'undefined') {
        const swaggerPaneResizeObserver = new ResizeObserver(() => {
            if (view === focusedView()) {
                updateSwaggerLayoutMode();
            }
        });
        swaggerPaneResizeObserver.observe(view.parts.swaggerRunWrap);
        view.disposers.push(() => swaggerPaneResizeObserver.disconnect());
    }
}

bindNotesView(firstNotesView);

async function openRecentFilesMenu(x, y) {
    try {
        const recentFiles = await NotesRecentFiles();
        const files = Array.isArray(recentFiles) ? recentFiles : [];
        if (files.length === 0) {
            setStatus('No recent files', false);
            return;
        }

        const menuItems = files.map((file) => ({
            title: file,
            icon: file === state.currentFile ? CONTEXT_ICON_TICK : 0,
            onSelect: () => {
                addHistoryPosition(file, currentEditorLineForFile(file));
                loadFile(file).catch((err) => {
                    notifyTerminal(`Failed to load file: ${file}`, 'warn');
                    console.error(err);
                });
            },
        }));

        showNotesLocalMenu(menuItems, x, y, 'Recent files');
    } catch {
        setStatus('Failed to open recent files menu', true);
    }
}

elements.currentFileTitle.addEventListener('click', (event) => {
    event.preventDefault();
    if (event.detail === 0) {
        const rect = elements.currentFileTitle.getBoundingClientRect();
        void openRecentFilesMenu(rect.left, rect.bottom);
        return;
    }
    void openRecentFilesMenu(event.clientX, event.clientY);
});

if (elements.status) {
    elements.status.title = 'Recent files';
    elements.status.tabIndex = 0;
    elements.status.setAttribute('role', 'button');

    elements.status.addEventListener('click', (event) => {
        event.preventDefault();
        void openRecentFilesMenu(event.clientX, event.clientY);
    });

    elements.status.addEventListener('keydown', (event) => {
        if (event.key !== 'Enter' && event.key !== ' ') {
            return;
        }

        event.preventDefault();
        const rect = elements.status.getBoundingClientRect();
        void openRecentFilesMenu(rect.left, rect.bottom);
    });
}

function getVisibleNotesTabs() {
    if (state.currentFileType === 'json') {
        const tabs = [elements.tabSwaggerView, elements.tabSwaggerEdit];
        if (state.swaggerRunAvailable && elements.tabSwaggerRun?.style.display !== 'none') {
            tabs.push(elements.tabSwaggerRun);
        }
        tabs.push(elements.tabHex);
        tabs.push(elements.tabMeta);
        return tabs.filter(Boolean);
    }

    if (state.currentFileType === 'code') {
        const tabs = [elements.tabEditor];
        if (isSvgFile(state.currentFile)) tabs.push(elements.tabImageView);
        tabs.push(elements.tabHex, elements.tabMeta);
        return tabs.filter(Boolean);
    }

    if (state.currentFileType === 'binary') {
        return [elements.tabHex, elements.tabMeta].filter(Boolean);
    }

    if (state.currentFileType === 'image') {
        return [elements.tabImageView, elements.tabHex, elements.tabMeta].filter(Boolean);
    }

    if (state.currentFileType === 'csv') {
        return [elements.tabCsvView, elements.tabCsvEdit, elements.tabCsvRun, elements.tabHex, elements.tabMeta].filter(Boolean);
    }

    return [elements.tabViewer, elements.tabEditor, elements.tabJupyter, elements.tabHex, elements.tabMeta].filter((tab) => tab && tab.style.display !== 'none');
}

function cycleNotesTabs(direction = 1) {
    const visibleTabs = getVisibleNotesTabs();
    if (visibleTabs.length <= 1) {
        return;
    }

    const currentIndex = visibleTabs.findIndex((tab) => tab.getAttribute('aria-selected') === 'true');
    const baseIndex = currentIndex === -1 ? 0 : currentIndex;
    const step = direction < 0 ? -1 : 1;
    const nextIndex = (baseIndex + step + visibleTabs.length) % visibleTabs.length;
    visibleTabs[nextIndex].click();
}

elements.historyPrev.addEventListener('click', async () => {
    try {
        const entry = await NotesHistoryPreviousPosition();
        await loadFile(entry.file);
        setViewMode(editViewModeForCurrentFileType());
        await ensureMonacoVisibleAndLaidOut();
        jumpEditorToLine(entry.line);
    } catch (err) {
        notifyTerminal(String(err && err.message ? err.message : err), 'info');
        focusActiveEditorForViewMode();
    }
});

elements.historyNext.addEventListener('click', async () => {
    try {
        const entry = await NotesHistoryNextPosition();
        await loadFile(entry.file);
        setViewMode(editViewModeForCurrentFileType());
        await ensureMonacoVisibleAndLaidOut();
        jumpEditorToLine(entry.line);
    } catch (err) {
        notifyTerminal(String(err && err.message ? err.message : err), 'info');
        focusActiveEditorForViewMode();
    }
});

elements.newFile.addEventListener('click', () => {
    openNewFilePrompt();
});

elements.modalCancel.addEventListener('click', () => {
    closeNewFilePrompt();
});

elements.modal.addEventListener('click', (event) => {
    if (event.target === elements.modal) {
        closeNewFilePrompt();
    }
});

elements.modalCreate.addEventListener('click', () => {
    createNewFile();
});

if (elements.modalLocation) {
    elements.modalLocation.addEventListener('click', (event) => {
        const rect = event.currentTarget.getBoundingClientRect();
        showLocalMenu({
            title: 'Location',
            options: NOTE_LOCATIONS,
            x: rect.left,
            y: rect.bottom,
            showNextToMouseCursor: true,
            onSelect: (index) => {
                const option = NOTE_LOCATIONS[index];
                if (option) {
                    elements.modalLocation.textContent = option;
                    elements.modalInput.focus();
                }
            },
        });
    });
}

document.getElementById('notes-fullsize-btn')?.addEventListener('click', () => {
    window.dispatchEvent(new CustomEvent('ttyphoon-notes-fullsize-toggle'));
});

elements.deleteCancel.addEventListener('click', () => {
    closeDeletePrompt();
});

elements.deleteConfirm.addEventListener('click', () => {
    confirmDelete();
});

elements.findInput.addEventListener('input', () => {
    updateFindInputClearButtonVisibility();
    performFind();
});

if (elements.findInputClear) {
    elements.findInputClear.addEventListener('click', () => {
        elements.findInput.value = '';
        updateFindInputClearButtonVisibility();
        performFind();
        elements.findInput.focus();
    });
}

if (elements.findDocOptionCase) {
    elements.findDocOptionCase.addEventListener('click', () => {
        state.findDocOptions.caseSensitive = !state.findDocOptions.caseSensitive;
        updateFindDocOptionButtons();
        performFind();
    });
}

if (elements.findDocOptionRegex) {
    elements.findDocOptionRegex.addEventListener('click', () => {
        state.findDocOptions.regex = !state.findDocOptions.regex;
        updateFindDocOptionButtons();
        performFind();
    });
}

if (elements.findDocOptionWord) {
    elements.findDocOptionWord.addEventListener('click', () => {
        state.findDocOptions.wholeWord = !state.findDocOptions.wholeWord;
        updateFindDocOptionButtons();
        performFind();
    });
}

if (elements.findFilesInput) {
    elements.findFilesInput.addEventListener('input', () => {
        state.findFilesMode = 'grep';
        state.findFilesSource = '';
        state.findFilesReferenceSymbol = '';
        updateFindFilesClearButtonVisibility();
        scheduleProjectFindSearch();
    });

if (elements.findReferences) {
    elements.findReferences.addEventListener('click', () => {
        void findReferencesFromEditor();
    });
}
    elements.findFilesInput.addEventListener('keydown', (event) => {
        if (event.key === 'ArrowDown' && !event.metaKey && !event.ctrlKey && !event.altKey) {
            if (tryOpenFindHistoryMenuForInput(elements.findFilesInput)) {
                event.preventDefault();
                return;
            }
        }

        if (event.key !== 'Enter') {
            return;
        }

        event.preventDefault();
        const query = String(elements.findFilesInput.value || '').trim();
        if (!query) {
            return;
        }

        runProjectFindSearch(query);
    });

    if (elements.findFilesClear) {
        elements.findFilesClear.addEventListener('click', () => {
            clearProjectFindResults({ keepInputFocus: true });
        });
    }

    // Setup grep option buttons
    if (elements.findOptionCase) {
        elements.findOptionCase.addEventListener('click', () => {
            state.findOptions.caseSensitive = !state.findOptions.caseSensitive;
            updateFindOptionButtons();
            triggerImmediateProjectFindSearch();
        });
    }

    if (elements.findOptionRegex) {
        elements.findOptionRegex.addEventListener('click', () => {
            state.findOptions.regex = !state.findOptions.regex;
            updateFindOptionButtons();
            triggerImmediateProjectFindSearch();
        });
    }

    if (elements.findOptionWord) {
        elements.findOptionWord.addEventListener('click', () => {
            state.findOptions.wholeWord = !state.findOptions.wholeWord;
            updateFindOptionButtons();
            triggerImmediateProjectFindSearch();
        });
    }
}

if (elements.replaceOne) {
    elements.replaceOne.addEventListener('mousedown', (event) => {
        event.preventDefault();
        replaceCurrentMatch();
    });
}

if (elements.replaceInput) {
    elements.replaceInput.addEventListener('input', () => {
        updateReplaceInputClearButtonVisibility();
    });
}

if (elements.replaceInputClear) {
    elements.replaceInputClear.addEventListener('click', () => {
        if (!elements.replaceInput) {
            return;
        }
        elements.replaceInput.value = '';
        updateReplaceInputClearButtonVisibility();
        elements.replaceInput.focus();
    });
}

if (elements.replaceAll) {
    elements.replaceAll.addEventListener('mousedown', (event) => {
        event.preventDefault();
        replaceAllMatches();
    });
}

if (elements.listFilter) {
    elements.listFilter.addEventListener('input', (event) => {
        state.fileFilterQuery = event.target.value || '';
        if (notesGitPanel?.isCommitsActive()) {
            void applyFileFilter();
            return;
        }
        if (isProjectFindListModeActive()) {
            scheduleProjectFindSearch();
            return;
        }
        applyFileFilter();
    });

    elements.listFilter.addEventListener('keydown', (event) => {
        if (event.key === 'Escape' && elements.listFilter.value) {
            event.preventDefault();
            elements.listFilter.value = '';
            state.fileFilterQuery = '';
            if (notesGitPanel?.isCommitsActive()) {
                void applyFileFilter();
                return;
            }
            if (isProjectFindListModeActive()) {
                scheduleProjectFindSearch();
                return;
            }
            applyFileFilter();
            return;
        }

        if (event.key === 'ArrowDown') {
            event.preventDefault();
            const items = notesGitPanel?.isCommitsActive()
                ? Array.from(elements.gitCommitList.querySelectorAll('.notes-git-commit'))
                : Array.from(elements.list.querySelectorAll('.notes-file'));
            if (items.length > 0) {
                items[0].focus();
            }
        }
    });

    elements.list.addEventListener('keydown', (event) => {
        if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp' && event.key !== 'Enter') {
            return;
        }

        const focused = document.activeElement;
        if (!focused || !focused.classList.contains('notes-file')) {
            return;
        }

        const items = Array.from(elements.list.querySelectorAll('.notes-file'));
        const currentIndex = items.indexOf(focused);
        if (currentIndex === -1) {
            return;
        }

        if (event.key === 'Enter') {
            event.preventDefault();
            focused.click();
        } else if (event.key === 'ArrowDown') {
            event.preventDefault();
            const next = items[currentIndex + 1];
            if (next) {
                next.focus();
            }
        } else if (event.key === 'ArrowUp') {
            event.preventDefault();
            if (currentIndex === 0) {
                elements.listFilter.focus();
            } else {
                items[currentIndex - 1].focus();
            }
        }
    });
}

if (elements.list) {
    const showFileTooltip = (event) => {
        const row = event.target.closest('[data-tooltip]');
        if (!row || row.contains(event.relatedTarget)) {
            return;
        }
        if (event.type === 'mouseover') {
            updateSharedTooltipPointer(event.clientX, event.clientY);
        }
        void showSharedTooltip(row.dataset.tooltip);
    };
    const hideFileTooltip = (event) => {
        const row = event.target.closest('[data-tooltip]');
        if (row && !row.contains(event.relatedTarget)) {
            closeSharedTooltip();
        }
    };
    elements.list.addEventListener('mouseover', showFileTooltip);
    elements.list.addEventListener('mouseout', hideFileTooltip);
    elements.list.addEventListener('focusin', showFileTooltip);
    elements.list.addEventListener('focusout', hideFileTooltip);
    elements.list.addEventListener('scroll', () => {
        closeSharedTooltip();
        if (!isProjectFindListModeActive()) {
            return;
        }

        scheduleProjectFindVirtualScrollRender();
    });
}
    // Delegated handlers for file/folder/category rows: renderFileList() rebuilds
    // the whole tree on every change, so per-node listeners would mean re-creating
    // one-to-several closures per file/folder on every render. A single delegated
    // listener per event type avoids that cost regardless of tree size.
    elements.list.addEventListener('click', (event) => {
        closeSharedTooltip();
        const fileItem = event.target.closest('.notes-file');
        if (fileItem) {
            addHistoryPosition(fileItem.dataset.file, currentEditorLineForFile(fileItem.dataset.file));
            loadFile(fileItem.dataset.file);
            return;
        }

        const folderButton = event.target.closest('.notes-tree-folder');
        if (folderButton) {
            toggleFolder(folderButton.dataset.folderKey);
            return;
        }

        const categoryHeader = event.target.closest('.notes-category-header');
        if (categoryHeader && state.fileFilterQuery.trim() === '') {
            toggleCategory(categoryHeader.dataset.category);
        }
    });

    elements.list.addEventListener('dblclick', (event) => {
        const fileItem = event.target.closest('.notes-file');
        if (!fileItem) {
            return;
        }
        event.preventDefault();
        void openRenamePrompt(fileItem.dataset.file);
    });

    elements.list.addEventListener('contextmenu', (event) => {
        closeSharedTooltip();
        const fileItem = event.target.closest('.notes-file');
        if (fileItem) {
            event.preventDefault();
            event.stopPropagation();
            void openFileParentFolderContextMenu(fileItem.dataset.file, event.clientX, event.clientY);
            return;
        }

        const folderButton = event.target.closest('.notes-tree-folder');
        if (folderButton) {
            event.preventDefault();
            event.stopPropagation();
            openFolderContextForListRow(folderButton, event.clientX, event.clientY);
            return;
        }

        const categoryHeader = event.target.closest('.notes-category-header');
        if (categoryHeader) {
            event.preventDefault();
            event.stopPropagation();
            openFolderContextForListRow(categoryHeader, event.clientX, event.clientY);
            return;
        }

        const folderContextRow = findFolderContextRowAbove(event.clientY);
        if (folderContextRow) {
            event.preventDefault();
            event.stopPropagation();
            openFolderContextForListRow(folderContextRow, event.clientX, event.clientY);
        }
    });

if (elements.listFilterClear && elements.listFilter) {
    elements.listFilterClear.addEventListener('click', () => {
        elements.listFilter.value = '';
        state.fileFilterQuery = '';
        if (notesGitPanel?.isCommitsActive()) {
            void applyFileFilter();
            elements.listFilter.focus();
            return;
        }
        if (isProjectFindListModeActive()) {
            scheduleProjectFindSearch();
            return;
        }
        applyFileFilter();
        elements.listFilter.focus();
    });
}

elements.findNext.addEventListener('mousedown', (event) => {
    event.preventDefault();
    persistFindFieldHistory(elements.findInput);
    nextMatch();
});

elements.findPrev.addEventListener('mousedown', (event) => {
    event.preventDefault();
    persistFindFieldHistory(elements.findInput);
    prevMatch();
});

// Initialize splitter for resizable panels
(function initSplitter() {
    const splitter = document.getElementById('notes-splitter');
    const app = document.getElementById('notes-app');
    const splitterWidth = 2;
    const minPaneWidth = 200;
    let isResizing = false;
    let hasManualSplit = false;
    let manualSplitRatio = 1 / 3;
    let resizeChanged = false;
    splitter.setAttribute('role', 'separator');
    splitter.setAttribute('aria-orientation', 'vertical');
    splitter.setAttribute('aria-label', 'Resize file list');
    splitter.tabIndex = 0;

    function canResize() {
        return app.isConnected && !app.closest('[data-ai-maximized="true"], [data-log-maximized="true"]');
    }

    function clampLeftWidth(totalWidth, leftWidth) {
        const available = totalWidth - splitterWidth;
        const minimum = Math.min(minPaneWidth, available / 2);
        return Math.min(Math.max(leftWidth, minimum), available - minimum);
    }

    function applyManualSplitToCurrentWidth() {
        if (!hasManualSplit || !canResize()) {
            return;
        }

        const appRect = app.getBoundingClientRect();
        if (appRect.width <= splitterWidth) {
            return;
        }

        const availableWidth = appRect.width - splitterWidth;
        const desiredLeftWidth = availableWidth * manualSplitRatio;
        const leftWidth = clampLeftWidth(appRect.width, desiredLeftWidth);
        const rightWidth = appRect.width - leftWidth - splitterWidth;

        app.style.gridTemplateColumns = `${leftWidth}px ${splitterWidth}px ${rightWidth}px`;
        splitter.setAttribute('aria-valuenow', String(Math.round(manualSplitRatio * 100)));
    }

    splitter.addEventListener('mousedown', (e) => {
        if (e.button !== 0 || !canResize() || app.getBoundingClientRect().width <= minPaneWidth * 2 + splitterWidth) return;
        e.preventDefault();
        isResizing = true;
        resizeChanged = false;
        document.body.style.cursor = 'col-resize';
        document.body.style.userSelect = 'none';
    });

    document.addEventListener('mousemove', (e) => {
        if (!isResizing || !canResize()) return;

        const appRect = app.getBoundingClientRect();
        const newLeftWidth = e.clientX - appRect.left;
        const minWidth = minPaneWidth;
        const maxWidth = appRect.width - minPaneWidth - splitterWidth;

        if (newLeftWidth > minWidth && newLeftWidth < maxWidth) {
            const rightWidth = appRect.width - newLeftWidth - splitterWidth;
            app.style.gridTemplateColumns = `${newLeftWidth}px ${splitterWidth}px ${rightWidth}px`;
            hasManualSplit = true;
            manualSplitRatio = newLeftWidth / (appRect.width - splitterWidth);
            resizeChanged = true;
            splitter.setAttribute('aria-valuenow', String(Math.round(manualSplitRatio * 100)));
        }
    });

    document.addEventListener('mouseup', () => {
        if (isResizing) {
            if (resizeChanged && canResize()) void saveLayoutRatio('fileListRatio', manualSplitRatio);
            isResizing = false;
            document.body.style.cursor = '';
            document.body.style.userSelect = '';
        }
    });

    window.addEventListener('resize', () => {
        if (isResizing) {
            return;
        }

        applyManualSplitToCurrentWidth();
    });

    splitter.addEventListener('keydown', (event) => {
        if (!canResize() || !['ArrowLeft', 'ArrowRight'].includes(event.key)) return;
        event.preventDefault();
        const width = app.getBoundingClientRect().width - splitterWidth;
        if (width <= minPaneWidth * 2) return;
        manualSplitRatio = clampLeftWidth(width + splitterWidth, width * manualSplitRatio + (event.key === 'ArrowRight' ? 16 : -16)) / width;
        hasManualSplit = true;
        applyManualSplitToCurrentWidth();
        void saveLayoutRatio('fileListRatio', manualSplitRatio);
    });
    splitter.addEventListener('dblclick', () => {
        if (!canResize()) return;
        manualSplitRatio = 1 / 3;
        hasManualSplit = true;
        applyManualSplitToCurrentWidth();
        void saveLayoutRatio('fileListRatio', 0);
    });
    if (typeof ResizeObserver !== 'undefined') {
        const observer = new ResizeObserver(applyManualSplitToCurrentWidth);
        observer.observe(app);
    }
    window.addEventListener('ttyphoon-notes-layout', applyManualSplitToCurrentWidth);
    void restoreLayoutRatio('fileListRatio', 1 / 3).then((ratio) => {
        if (hasManualSplit) return;
        manualSplitRatio = ratio;
        hasManualSplit = true;
        applyManualSplitToCurrentWidth();
    });
})();

(function initToolsResizer() {
    const handle = document.getElementById('notes-tools-resizer');
    const panel = document.getElementById('notes-panel');
    const tools = elements.toolsPanel;
    let ratio = 0.35;
    let edited = false;
    let dragging = false;
    let changed = false;

    function enabled() {
        return panel.isConnected && tools.dataset.collapsed === 'false'
            && !tools.closest('[data-ai-maximized="true"], [data-log-maximized="true"]');
    }
    // The Tools size is a share of the space below the view tab rows, which
    // sit inside #notes-panel above the panes.
    function availableHeight() {
        return panel.getBoundingClientRect().height - 4 - focusedView().parts.tabs.getBoundingClientRect().height;
    }
    function apply() {
        handle.dataset.enabled = enabled() ? 'true' : 'false';
        if (!enabled()) return;
        const available = availableHeight();
        if (available <= 0) return;
        const minimum = Math.min(120, available / 2);
        const maximum = available - Math.min(140, available / 2);
        const height = Math.min(Math.max(available * ratio, minimum), maximum);
        tools.style.setProperty('--notes-tools-size', `${height}px`);
        handle.setAttribute('aria-valuenow', String(Math.round(ratio * 100)));
    }
    function resize(height) {
        const available = availableHeight();
        if (available < 260) return false;
        ratio = Math.min(Math.max(height, 120), available - 140) / available;
        edited = true;
        apply();
        return true;
    }
    handle.addEventListener('mousedown', (event) => {
        if (event.button !== 0 || !enabled()) return;
        event.preventDefault();
        dragging = true;
        changed = false;
        handle.dataset.dragging = 'true';
        document.body.style.cursor = 'row-resize';
        document.body.style.userSelect = 'none';
    });
    document.addEventListener('mousemove', (event) => {
        if (!dragging || !enabled()) return;
        changed = resize(panel.getBoundingClientRect().bottom - event.clientY) || changed;
    });
    document.addEventListener('mouseup', () => {
        if (!dragging) return;
        dragging = false;
        handle.dataset.dragging = 'false';
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
        if (changed && enabled()) void saveLayoutRatio('toolsRatio', ratio);
    });
    handle.addEventListener('keydown', (event) => {
        if (!enabled() || !['ArrowUp', 'ArrowDown'].includes(event.key)) return;
        event.preventDefault();
        const height = tools.getBoundingClientRect().height + (event.key === 'ArrowUp' ? 16 : -16);
        if (resize(height)) void saveLayoutRatio('toolsRatio', ratio);
    });
    handle.addEventListener('dblclick', () => {
        if (!enabled()) return;
        ratio = 0.35;
        edited = true;
        apply();
        void saveLayoutRatio('toolsRatio', 0);
    });
    window.addEventListener('resize', apply);
    window.addEventListener('ttyphoon-notes-layout', apply);
    if (typeof ResizeObserver !== 'undefined') {
        const observer = new ResizeObserver(apply);
        observer.observe(panel);
    }
    const maximizeRoot = tools.closest('#notes-pane') || document.getElementById('notes-app');
    const observer = new MutationObserver(() => { apply(); window.dispatchEvent(new CustomEvent('ttyphoon-notes-layout')); });
    observer.observe(maximizeRoot, { attributes: true, attributeFilter: ['data-ai-maximized', 'data-log-maximized'] });
    void restoreLayoutRatio('toolsRatio', 0.35).then((restored) => {
        if (!edited) ratio = restored;
        apply();
    });
})();

document.addEventListener('keydown', (event) => {
    if (!(event.metaKey || event.ctrlKey) || event.altKey || event.key.toLowerCase() !== 'f') {
        return;
    }
    if (document.getElementById('fullscreen-image-overlay')
        || window.ttyphoonInputboxOpen === true
        || window.terminalFocusedState === true) {
        return;
    }

    event.preventDefault();
    event.stopPropagation();
    const findTabActive = elements.toolsTabFind?.getAttribute('aria-selected') === 'true';
    const panelVisible = elements.toolsPanel?.dataset.collapsed !== 'true';
    if (findTabActive && panelVisible) {
        setToolsPanelCollapsed(true);
    } else {
        openFindBar();
    }
}, true);

document.addEventListener('keydown', (event) => {
    const command = notesDocumentTabCommandForKey(event);
    if (!command
        || !app.isConnected
        || document.getElementById('fullscreen-image-overlay')
        || window.ttyphoonInputboxOpen === true
        || window.terminalFocusedState === true
        || elements.modal.dataset.open === 'true'
        || elements.deleteModal.dataset.open === 'true') {
        return;
    }
    event.preventDefault();
    event.stopPropagation();
    void NOTES_DOCUMENT_TAB_COMMANDS[command]();
}, true);

EventsOn('notesDocumentTabCommand', (command) => {
    void NOTES_DOCUMENT_TAB_COMMANDS[command]?.();
});

document.addEventListener('keydown', (event) => {
    // Block keyboard shortcuts if fullscreen image overlay is open
    if (document.getElementById('fullscreen-image-overlay')) {
        return;
    }

    if (window.ttyphoonInputboxOpen === true) {
        return;
    }

    if (window.terminalFocusedState === true) {
        return;
    }

    if (event.ctrlKey && !event.metaKey && !event.altKey && event.key === 'Tab') {
        event.preventDefault();
        cycleNotesTabs(event.shiftKey ? -1 : 1);
        return;
    }

    if (event.metaKey && !event.ctrlKey && !event.altKey && event.key.toLowerCase() === 'p') {
        event.preventDefault();
        ShowCommandPalette().catch(() => {});
        return;
    }

    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 's') {
        event.preventDefault();
        if (activeGitDiff?.diff.readOnly) {
            return;
        }
            saveAndFormat();
        return;
    }

    if (event.key === 'Escape' && elements.modal.dataset.open === 'true') {
        event.preventDefault();
        closeNewFilePrompt();
    } else if (event.key === 'Escape' && elements.deleteModal.dataset.open === 'true') {
        event.preventDefault();
        closeDeletePrompt();
    } else if (event.key === 'Escape' && elements.aiToolMetaModal?.dataset?.open === 'true') {
        event.preventDefault();
        closeAIToolMetadataModal();
    } else if (event.key === 'Escape' && elements.aiSettingsModal?.dataset?.open === 'true') {
        event.preventDefault();
        closeAISessionManagementModal();
    }
});

function isFunctionKey(key) {
    return /^F([1-9]|1[0-9]|2[0-4])$/.test(String(key || ''));
}

let notesHotkeyPrefixActive = false;

document.addEventListener('keydown', (event) => {
    // Block keyboard shortcuts if fullscreen image overlay is open
    if (document.getElementById('fullscreen-image-overlay')) {
        return;
    }

    if (window.ttyphoonInputboxOpen === true) {
        return;
    }

    if (window.terminalFocusedState === true) {
        return;
    }

    const shouldRouteToGo = notesHotkeyPrefixActive || isFunctionKey(event.key);
    if (!shouldRouteToGo) {
        return;
    }

    // During a prefix sequence, always consume plain keys before the browser/editor sees them.
    event.preventDefault();
    event.stopPropagation();

    NotesKeyPress(
        event.key,
        event.ctrlKey,
        event.altKey,
        event.shiftKey,
        event.metaKey,
    ).then((result) => {
        notesHotkeyPrefixActive = Boolean(result?.prefixActive);
    }).catch(() => {
        notesHotkeyPrefixActive = false;
    });
}, true);

window.addEventListener('beforeunload', () => {
    closeOpenLspDocument();
    closeCurrentTyposDocument();
    NotesLspStopAll();
    try {
        if (typeof NotesTableDisposeAll === 'function') {
            void Promise.resolve(NotesTableDisposeAll()).catch(() => {});
        }
    } catch {
        // Nothing useful to do while the window is going away.
    }
});

elements.modalInput.addEventListener('keydown', (event) => {
    if (event.key === 'Enter') {
        event.preventDefault();
        createNewFile();
    }
});

elements.findInput.addEventListener('keydown', (event) => {
    if (event.key === 'ArrowDown' && !event.metaKey && !event.ctrlKey && !event.altKey) {
        if (tryOpenFindHistoryMenuForInput(elements.findInput)) {
            event.preventDefault();
            return;
        }
    }

    if (event.key === 'Enter') {
        event.preventDefault();
        persistFindFieldHistory(elements.findInput);
        if (event.shiftKey) {
            prevMatch();
        } else {
            nextMatch();
        }
    }
});

if (elements.replaceInput) {
    elements.replaceInput.addEventListener('keydown', (event) => {
        if (event.key === 'ArrowDown' && !event.metaKey && !event.ctrlKey && !event.altKey) {
            if (tryOpenFindHistoryMenuForInput(elements.replaceInput)) {
                event.preventDefault();
                return;
            }
        }

        if (event.key !== 'Enter') {
            return;
        }

        event.preventDefault();
        persistFindFieldHistory(elements.findInput);
        persistFindFieldHistory(elements.replaceInput);
        if (event.shiftKey) {
            replaceAllMatches();
        } else {
            replaceCurrentMatch();
        }
    });
}

setViewMode('editor');
renderProjectFindResults();
updateFindFilesClearButtonVisibility();
updateFindInputClearButtonVisibility();
updateReplaceInputClearButtonVisibility();
hydrateFindFieldHistory(elements.findInput);
hydrateFindFieldHistory(elements.replaceInput);
hydrateFindFieldHistory(elements.findFilesInput);

// Initial scan is followed by context-change events, not terminal focus scans.
refreshFiles();

if (typeof ResizeObserver === 'undefined') {
    window.addEventListener('resize', () => {
        updateSwaggerLayoutMode();
    });
}
