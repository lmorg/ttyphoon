import {
    GetWindowStyle, GetNotesMaxLogLines, GetNotesColumnWidths, SetNotesColumnWidths, GetFile, GetImage,
    GetNotesStructViewMaxSizeKB,
    ListFiles, SaveFile, SaveBinaryFile, DeleteFile, RenameFile,
    CancelNotesListFiles,
    RunNote, RunFunction, StopNote, SendIpc, SendToTerminal,
    GetLanguageDescriptions, GetAllLanguageDescriptions, TerminalCopyImageDataURL,
    ResolveFilePath, GetHyperlinkMenuActions, RunHyperlinkMenuAction,
    DisplayHyperlinkMenu,
    SaveImageDialog, WindowPrint, GetClipboardData, SwaggerRequest, NotesKeyPress,
    ShowCommandPalette, GetCurrentProject, GetCurrentGroupName, GetFileMetaMarkdown, AskAI,
    ShowAISkillsMenu,
    GetAISessionCache,
    GetAISessionManagement, CreateAISession, SetActiveAISession, DeleteAISession,
    ListAIModelSelections, GetCurrentAIModelSelection, SetCurrentAIModelSelection,
    ListAIPromptLogs, GetAIPromptLog,
    GetAIToolsList, SetAIToolSubagentAllowed, SetAIToolState, ShowAIToolStateMenu, ShowAIToolSubagentMenu, ResolveAIToolPermission, GetAIMcpServers, SetAIMcpServerEnabled, ClearAISessionHistory, ClearAILog,
    ResolveNotesLspLanguage, NotesLspAvailableForRuntime, NotesRecentFiles, ResolveNoteLocation, ComposeNoteLocationPath,
    NotesHistoryPrevious, NotesHistoryNext, NotesHistoryAdd, NotesHistoryCurrent, NotesGrepStream,
    GetProjectCache, SetProjectCache,
    GetNotesFindFieldValues, AddNotesFindFieldValue,
    GetDocumentCache, SetDocumentCache, FormatCodeBlock, FormatNotesContent, CompleteSyntax,
    NotesLspOpenDocument, NotesLspChangeDocument, NotesLspSaveDocument,
    NotesLspCloseDocument, NotesLspStopAll, NotesLspHover, NotesLspCompletion,
    NotesTyposOpenDocument, NotesTyposChangeDocument, NotesTyposCloseDocument,
    NotesLspCodeLens, NotesLspExecuteCodeLens,
    NotesLspInlayHints,
    NotesLspDefinition, NotesLspDocumentSymbols, NotesLspWorkspaceSymbols, NotesLspFormat, NotesLspFormatRange, NotesLspCodeActions, NotesLspApplyCodeAction,
    GetNotesLanguageTabIndent, GetNotesLanguageReservedWords,
    NotesLspSignatureHelp,
    NotesLspPrepareRename, NotesLspRename,
    FilterStrings,
} from '../wailsjs/go/main/WApp';
import { EventsOn, EventsOff, ClipboardSetText } from '../wailsjs/runtime/runtime';

import {
    showLocalMenu,
} from './popup_menu';
import { initNotesLogPanel } from './notes-log-panel';
import { initNotesAIPanel } from './notes-ai-panel';
import { bindSharedTooltipMouseTracking, closeSharedTooltip, showSharedTooltip, updateSharedTooltipPointer } from './shared_tooltip';
import { attachVimMode } from './vim-mode';
import { attachSpellCheck } from './spellcheck';
import { createEditorUndoManager } from './editor_undo_manager';
import { createEditorMutationAdapter } from './editor_mutation_adapter';
import { createMonacoAdapter } from './monaco_adapter';
import './notes.css';

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

import { configureMarked, processMarkdownContainer, enableFullscreenImages, processLinks } from './markdown-utils.js';
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

app.innerHTML = `
    <div id="notes-app">
        <aside id="notes-sidebar">
            <div id="notes-sidebar-header">
                <div id="notes-title">Notes</div>
                <div id="notes-list-filter-wrap">
                    <input id="notes-list-filter" type="text" placeholder="Filter files..." autocomplete="off" autocorrect="off" autocapitalize="off" spellcheck="false" />
                    <button id="notes-list-filter-clear" type="button" title="Clear filter" aria-label="Clear filter">&#xf410;</button>
                </div>
            </div>
            <div id="notes-list" role="list"></div>
        </aside>
        <div id="notes-splitter"></div>
        <main id="notes-main">
            <div id="notes-tabs" role="tablist">
                <button id="notes-tab-viewer" type="button" class="tab" role="tab" aria-selected="true">View</button>
                <button id="notes-tab-editor" type="button" class="tab" role="tab" aria-selected="false">Edit</button>
                <button id="notes-tab-jupyter" type="button" class="tab" role="tab" aria-selected="false">Run</button>
                <button id="notes-tab-swagger-view" type="button" class="tab" role="tab" aria-selected="false" style="display: none;" data-swagger="true">View</button>
                <button id="notes-tab-swagger-edit" type="button" class="tab" role="tab" aria-selected="false" style="display: none;" data-swagger="true">Edit</button>
                <button id="notes-tab-swagger-run" type="button" class="tab" role="tab" aria-selected="false" style="display: none;" data-swagger="true">Run</button>
                <button id="notes-tab-csv-view"   type="button" class="tab" role="tab" aria-selected="false" style="display: none;">View</button>
                <button id="notes-tab-csv-edit"   type="button" class="tab" role="tab" aria-selected="false" style="display: none;">Edit</button>
                <button id="notes-tab-csv-run"    type="button" class="tab" role="tab" aria-selected="false" style="display: none;">Run</button>
                <button id="notes-tab-image-view" type="button" class="tab" role="tab" aria-selected="false" style="display: none;">View</button>
                <button id="notes-tab-hex" type="button" class="tab" role="tab" aria-selected="false">Hex</button>
                <button id="notes-tab-meta" type="button" class="tab" role="tab" aria-selected="false">Meta</button>
                <div id="notes-toolbar" class="notes-toolbar">
                    <button id="notes-new" type="button" class="notes-toolbar-btn" title="New" aria-label="New note">&#xe494;</button>
                    <button id="notes-history-prev" type="button" class="notes-toolbar-btn" title="Previous document" aria-label="Previous document">&#xf359;</button>
                    <button id="notes-history-next" type="button" class="notes-toolbar-btn" title="Next document" aria-label="Next document">&#xf35a;</button>
                    <button id="notes-fullsize-btn" type="button" class="notes-toolbar-btn" title="Full size" aria-label="Full size">&#xf065;</button>
                </div>
            </div>
            <div id="notes-panel">
                <div id="notes-editor-wrap" role="tabpanel">
                    <div id="notes-editor-shell" data-code-view="false">
                        <div id="notes-editor-scroll">
                            <textarea id="notes-editor" autocorrect="off" autocapitalize="off" autocomplete="off" spellcheck="false" data-gramm="false" data-gramm_editor="false" data-enable-grammarly="false"></textarea>
                            <div id="notes-monaco-editor" aria-hidden="true"></div>
                        </div>
                    </div>
                </div>
                <div id="notes-hex-wrap" role="tabpanel">
                    <div id="notes-hex"></div>
                </div>
                <div id="notes-preview-wrap" class="markdown-body" role="tabpanel">
                    <div id="notes-preview"></div>
                </div>
                <div id="notes-html-view-wrap" role="tabpanel">
                    <iframe id="notes-html-view-frame" title="HTML preview" sandbox=""></iframe>
                </div>
                <div id="notes-jupyter-wrap" class="markdown-body" role="tabpanel">
                    <div id="notes-jupyter"></div>
                </div>
                <div id="notes-csv-view-wrap" role="tabpanel">
                    <div id="notes-csv-view" class="markdown-body"></div>
                </div>
                <div id="notes-image-view-wrap" role="tabpanel">
                    <img id="notes-image-view-img" alt="" />
                </div>
                <div id="notes-meta-wrap" class="markdown-body" role="tabpanel">
                    <div id="notes-meta"></div>
                </div>
                <div id="notes-swagger-view-wrap" role="tabpanel" style="display: none;">
                    <div id="notes-swagger-view" class="json-viewer"></div>
                </div>
                <div id="notes-swagger-run-wrap" class="swagger-ui" role="tabpanel" style="display: none;">
                    <div id="notes-swagger-layout" class="swagger-layout">
                        <div id="notes-swagger-info" class="swagger-info markdown-body"></div>
                        <aside id="notes-swagger-endpoints" class="swagger-endpoints-pane"></aside>
                        <section id="notes-swagger-main" class="swagger-main-pane">
                            <div id="notes-swagger-request-builder"></div>
                            <div id="notes-swagger-response"></div>
                        </section>
                    </div>
                </div>
                <div id="notes-tools-panel" class="notes-tools-panel" data-collapsed="true">
                    <div class="notes-tools-header">
                        <div id="notes-tools-tabs" role="tablist" class="tools-tabs-container">
                            <button id="notes-tools-tab-toc" type="button" class="tools-tab" role="tab" aria-selected="false" aria-controls="notes-tools-toc-pane" data-tab="toc" style="display: none;">ToC</button>
                            <button id="notes-tools-tab-frontmatter" type="button" class="tools-tab" role="tab" aria-selected="false" aria-controls="notes-tools-frontmatter-pane" data-tab="frontmatter" style="display: none;">Frontmatter</button>
                            <button id="notes-tools-tab-find" type="button" class="tools-tab" role="tab" aria-selected="true" aria-controls="notes-tools-find-pane" data-tab="find">Find</button>
                            <button id="notes-tools-tab-ai"  type="button" class="tools-tab" role="tab" aria-selected="false" aria-controls="notes-tools-ai-pane" data-tab="ai">AI</button>
                            <button id="notes-tools-tab-log" type="button" class="tools-tab" role="tab" aria-selected="false" aria-controls="notes-tools-log-pane" data-tab="log">Log</button>
                        </div>
                        <button id="notes-tools-minimize" type="button" class="notes-tools-minimize" title="Minimize Tools panel"></button>
                    </div>
                    <div id="notes-tools-content" class="notes-tools-content">
                        <div id="notes-tools-ai-pane" class="notes-tools-pane" data-tab="ai" data-active="false">
                            <div class="notes-tools-pane-header">
                                <button id="notes-tools-ai-prompt-jump" type="button" class="notes-ai-model-picker" title="Prompt">Prompt</button>
                                <button id="notes-tools-ai-ask" type="button" class="notes-tools-clear" title="Ask AI">Ask…</button>
                                <button id="notes-tools-ai-skills" type="button" class="notes-tools-clear" title="Ask AI with a skill">Skills</button>
                                <button id="notes-tools-ai-maximize" type="button" class="notes-tools-clear" title="Maximize AI view">Maximize</button>
                                <button id="notes-tools-clear" type="button" class="notes-tools-clear" title="Clear AI output">Clear</button>
                                <button id="notes-tools-ai-settings" type="button" class="notes-tools-clear" title="AI session management">Settings</button>
                            </div>
                            <div id="notes-ai-output" class="notes-ai-output"></div>
                            <button id="notes-ai-scroll-bottom" type="button" class="notes-ai-scroll-bottom" data-visible="false" title="Scroll to latest AI output">Latest</button>
                        </div>
                        <div id="notes-tools-toc-pane" class="notes-tools-pane" data-tab="toc" data-active="false">
                            <div id="notes-tools-toc" class="notes-tools-toc"></div>
                        </div>
                        <div id="notes-tools-frontmatter-pane" class="notes-tools-pane" data-tab="frontmatter" data-active="false">
                            <div id="notes-tools-frontmatter" class="notes-tools-frontmatter"></div>
                        </div>
                        <div id="notes-tools-find-pane" class="notes-tools-pane" data-tab="find" data-active="true">
                            <div class="notes-tools-find-wrap markdown-body">
                                <h1 class="notes-find-heading">In open file</h1>
                                <div id="notes-find-controls" data-disabled="false">
                                    <div id="notes-find-row">
                                        <div id="notes-find-input-wrap">
                                            <input id="notes-find-input" type="text" placeholder="Find..." autocomplete="off" autocorrect="off" autocapitalize="off" spellcheck="false" />
                                            <button id="notes-find-input-clear" type="button" title="Clear find" aria-label="Clear find">&#xf410;</button>
                                        </div>
                                        <div id="notes-find-doc-options" class="notes-find-options">
                                            <button id="notes-find-doc-option-case" type="button" class="notes-find-option-btn" title="Case sensitive" data-active="false">Aa</button>
                                            <button id="notes-find-doc-option-regex" type="button" class="notes-find-option-btn" title="Regex" data-active="false">.*</button>
                                            <button id="notes-find-doc-option-word" type="button" class="notes-find-option-btn" title="Whole word" data-active="false">␣W</button>
                                        </div>
                                    </div>
                                    <div id="notes-find-actions">
                                        <span id="notes-find-counter"></span>
                                        <button id="notes-find-prev" type="button" title="Previous match">↑</button>
                                        <button id="notes-find-next" type="button" title="Next match">↓</button>
                                    </div>
                                </div>
                                <div id="notes-replace-controls" data-disabled="true">
                                    <div id="notes-replace-input-wrap">
                                        <input id="notes-replace-input" type="text" placeholder="Replace..." autocomplete="off" autocorrect="off" autocapitalize="off" spellcheck="false" />
                                        <button id="notes-replace-input-clear" type="button" title="Clear replace" aria-label="Clear replace">&#xf410;</button>
                                    </div>
                                    <div id="notes-replace-actions">
                                        <button id="notes-replace-one" type="button" title="Replace current match">Replace</button>
                                        <button id="notes-replace-all" type="button" title="Replace all matches">Replace all</button>
                                    </div>
                                </div>
                                <h1 class="notes-find-heading">For files containing</h1>
                                <div id="notes-find-files-row">
                                    <div id="notes-find-files-input-wrap">
                                        <input id="notes-find-files-input" type="text" placeholder="Search project files..." autocomplete="off" autocorrect="off" autocapitalize="off" spellcheck="false" />
                                        <button id="notes-find-files-clear" type="button" title="Close results" aria-label="Close results">&#xf410;</button>
                                    </div>
                                    <div id="notes-find-options" class="notes-find-options">
                                        <button id="notes-find-option-case" type="button" class="notes-find-option-btn" title="Case sensitive" data-active="false">Aa</button>
                                        <button id="notes-find-option-regex" type="button" class="notes-find-option-btn" title="Regex" data-active="false">.*</button>
                                        <button id="notes-find-option-word" type="button" class="notes-find-option-btn" title="Whole word" data-active="false">␣W</button>
                                    </div>
                                </div>
                                <div id="notes-find-files-results"></div>
                            </div>
                        </div>
                        <div id="notes-tools-log-pane" class="notes-tools-pane" data-tab="log" data-active="false">
                            <div class="notes-tools-pane-header">
                                <button id="notes-tools-log-copy" type="button" class="notes-tools-clear" title="Copy log to clipboard">Copy</button>
                                <button id="notes-tools-log-deselect" type="button" class="notes-tools-clear" title="Deselect log lines">Deselect</button>
                                <button id="notes-tools-log-trace" type="button" class="notes-tools-clear" title="Show trace-level log lines">Trace</button>
                                <button id="notes-tools-log-timestamp" type="button" class="notes-tools-clear" title="Toggle timestamps">Timestamp</button>
                                <button id="notes-tools-log-wordwrap" type="button" class="notes-tools-clear" title="Toggle word wrap">Wrap</button>
                                <button id="notes-tools-log-maximize" type="button" class="notes-tools-clear" title="Maximize log view">Maximize</button>
                                <button id="notes-tools-log-clear" type="button" class="notes-tools-clear" title="Clear log">Clear</button>
                            </div>
                            <div id="notes-log-output" class="notes-log-output" style="white-space: pre; overflow-wrap: normal;"></div>
                            <button id="notes-log-scroll-bottom" type="button" class="notes-ai-scroll-bottom" data-visible="false" title="Scroll to latest log output">Latest</button>
                        </div>
                    </div>
                </div>
                <button id="notes-tools-restore" type="button" class="notes-tools-restore" title="Show Tools panel">Tools</button>
            </div>
        </main>
    </div>
    <div id="notes-modal" data-open="false" aria-hidden="true">
        <div id="notes-modal-card" role="dialog" aria-modal="true" aria-labelledby="notes-modal-title">
            <div id="notes-modal-title">New note name</div>
            <div id="notes-modal-location-row">
                <button id="notes-modal-location" type="button" aria-label="Location" title="Select location">$NOTES</button>
                <input id="notes-modal-input" type="text" placeholder="example-note" autocomplete="off" autocorrect="off" autocapitalize="off" spellcheck="false" />
            </div>
            <div id="notes-modal-actions">
                <button id="notes-modal-cancel" type="button">Cancel</button>
                <button id="notes-modal-create" type="button">Create</button>
            </div>
        </div>
    </div>
    <div id="notes-delete-modal" data-open="false" aria-hidden="true">
        <div id="notes-delete-modal-card" role="dialog" aria-modal="true" aria-labelledby="notes-delete-modal-title">
            <div id="notes-delete-modal-title">Delete note</div>
            <div id="notes-delete-modal-body"></div>
            <div id="notes-delete-modal-actions">
                <button id="notes-delete-cancel" type="button">Cancel</button>
                <button id="notes-delete-confirm" type="button">Delete</button>
            </div>
        </div>
    </div>
    <div id="notes-ai-settings-modal" data-open="false" aria-hidden="true">
        <div id="notes-ai-settings-card" class="markdown-body" role="dialog" aria-modal="true" aria-labelledby="notes-ai-settings-title">
            <h1 id="notes-ai-settings-title">AI Session Management</h1>

            <h2>Default Model</h2>
            <p>Choose the default model for new AI requests.</p>
            <div class="notes-ai-settings-controls">
                <button id="notes-ai-settings-model-picker" type="button" class="notes-ai-model-picker" title="Select AI model">Model</button>
            </div>

            <h2>Tools</h2>
            <p>Enable or disable tools for AI requests.</p>
            <div id="notes-ai-settings-tools-list" class="notes-ai-settings-tools-list"></div>

            <h2>MCP Servers</h2>
            <p>Load or unload MCP servers for AI requests.</p>
            <div id="notes-ai-settings-mcp-list" class="notes-ai-settings-tools-list"></div>

            <h2>Sessions</h2>
            <p>Create, switch, clear, or delete sessions for this workspace.</p>
            <div id="notes-ai-settings-sessions-list" class="notes-ai-settings-sessions-list"></div>
            <div class="notes-ai-settings-controls">
                <button id="notes-ai-settings-session-new" type="button">New session</button>
                <button id="notes-ai-settings-history-clear" type="button">Clear active history</button>
            </div>

            <h2>Active Session Transcript</h2>
            <p>Recent prompts and responses from the active session.</p>
            <div id="notes-ai-settings-history-list" class="notes-ai-settings-history-list"></div>

            <div id="notes-ai-tool-meta-modal" data-open="false" aria-hidden="true">
                <div id="notes-ai-tool-meta-card" class="markdown-body" role="dialog" aria-modal="true" aria-label="Tool metadata" tabindex="-1">
                    <div id="notes-ai-tool-meta-content"></div>
                </div>
            </div>
        </div>
    </div>
`;

const elements = {
    appRoot: app,
    title: document.getElementById('notes-title'),
    list: document.getElementById('notes-list'),
    listFilter: document.getElementById('notes-list-filter'),
    listFilterClear: document.getElementById('notes-list-filter-clear'),
    editor: document.getElementById('notes-editor'),
    editorShell: document.getElementById('notes-editor-shell'),
    monacoEditor: document.getElementById('notes-monaco-editor'),
    preview: document.getElementById('notes-preview'),
    htmlViewWrap: document.getElementById('notes-html-view-wrap'),
    htmlViewFrame: document.getElementById('notes-html-view-frame'),
    jupyter: document.getElementById('notes-jupyter'),
    status: document.getElementById('notes-status'),
    newFile: document.getElementById('notes-new'),
    historyPrev: document.getElementById('notes-history-prev'),
    historyNext: document.getElementById('notes-history-next'),
    tabEditor: document.getElementById('notes-tab-editor'),
    tabHex: document.getElementById('notes-tab-hex'),
    tabViewer: document.getElementById('notes-tab-viewer'),
    tabJupyter: document.getElementById('notes-tab-jupyter'),
    tabSwaggerView: document.getElementById('notes-tab-swagger-view'),
    tabSwaggerEdit: document.getElementById('notes-tab-swagger-edit'),
    tabSwaggerRun: document.getElementById('notes-tab-swagger-run'),
    tabImageView: document.getElementById('notes-tab-image-view'),
    tabMeta: document.getElementById('notes-tab-meta'),
    tabCsvView: document.getElementById('notes-tab-csv-view'),
    tabCsvEdit: document.getElementById('notes-tab-csv-edit'),
    tabCsvRun: document.getElementById('notes-tab-csv-run'),
    editorWrap: document.getElementById('notes-editor-wrap'),
    hexWrap: document.getElementById('notes-hex-wrap'),
    hex: document.getElementById('notes-hex'),
    previewWrap: document.getElementById('notes-preview-wrap'),
    jupyterWrap: document.getElementById('notes-jupyter-wrap'),
    imageViewWrap: document.getElementById('notes-image-view-wrap'),
    imageViewImg: document.getElementById('notes-image-view-img'),
    metaWrap: document.getElementById('notes-meta-wrap'),
    meta: document.getElementById('notes-meta'),
    csvViewWrap: document.getElementById('notes-csv-view-wrap'),
    csvView: document.getElementById('notes-csv-view'),
    swaggerViewWrap: document.getElementById('notes-swagger-view-wrap'),
    swaggerRunWrap: document.getElementById('notes-swagger-run-wrap'),
    swaggerView: document.getElementById('notes-swagger-view'),
    swaggerEndpoints: document.getElementById('notes-swagger-endpoints'),
    swaggerRequestBuilder: document.getElementById('notes-swagger-request-builder'),
    swaggerResponse: document.getElementById('notes-swagger-response'),
    modal: document.getElementById('notes-modal'),
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
    findFilesClear: document.getElementById('notes-find-files-clear'),
    findFilesResults: document.getElementById('notes-find-files-results'),
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
    toolsAIMaximize: document.getElementById('notes-tools-ai-maximize'),
    toolsAIAsk: document.getElementById('notes-tools-ai-ask'),
    toolsAISkills: document.getElementById('notes-tools-ai-skills'),
    toolsAISettings: document.getElementById('notes-tools-ai-settings'),
    toolsClear: document.getElementById('notes-tools-clear'),
    aiOutput: document.getElementById('notes-ai-output'),
    aiScrollBottom: document.getElementById('notes-ai-scroll-bottom'),
    aiSettingsModal: document.getElementById('notes-ai-settings-modal'),
    aiSettingsSessionsList: document.getElementById('notes-ai-settings-sessions-list'),
    aiSettingsSessionNew: document.getElementById('notes-ai-settings-session-new'),
    aiSettingsHistoryList: document.getElementById('notes-ai-settings-history-list'),
    aiSettingsToolsList: document.getElementById('notes-ai-settings-tools-list'),
    aiSettingsMcpList: document.getElementById('notes-ai-settings-mcp-list'),
    aiToolMetaModal: document.getElementById('notes-ai-tool-meta-modal'),
    aiToolMetaCard: document.getElementById('notes-ai-tool-meta-card'),
    aiToolMetaContent: document.getElementById('notes-ai-tool-meta-content'),
    aiSettingsHistoryClear: document.getElementById('notes-ai-settings-history-clear'),
    logOutput: document.getElementById('notes-log-output'),
    toolsLogMaximize: document.getElementById('notes-tools-log-maximize'),
    toolsLogTimestamp: document.getElementById('notes-tools-log-timestamp'),
    toolsLogTrace: document.getElementById('notes-tools-log-trace'),
    toolsLogWordwrap: document.getElementById('notes-tools-log-wordwrap'),
    toolsLogCopy: document.getElementById('notes-tools-log-copy'),
    toolsLogDeselect: document.getElementById('notes-tools-log-deselect'),
    toolsLogClear: document.getElementById('notes-tools-log-clear'),
    logScrollBottom: document.getElementById('notes-log-scroll-bottom'),
    toolsRestore: document.getElementById('notes-tools-restore')
};

const state = {
    files: [],
    currentFile: '',
    currentFileUri: '',
    currentFileProject: '',  // The project path when file was opened, prevents overwrites on project switch
    currentFileType: 'markdown',  // 'markdown' | 'json' | 'html' | 'code' | 'image' | 'csv' | 'binary'
    suspendDocumentCacheSave: false,
    dirty: false,
    viewTainted: false,
    useMonacoEditor: false,
    renderTimer: null,
    autosaveTimer: null,
    viewMode: 'viewer',
    renamingFile: null,
    deletingFile: null,
    deleteConfirmAction: null,
    findMatches: [],
    findCurrentIndex: -1,
    findQuery: '',
    findDocLastExecutedSignature: '',
    findDocOptions: {
        caseSensitive: false,
        regex: false,
        wholeWord: false,
    },
    findFilesQuery: '',
    findFilesResults: [],
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
    lastRestoredDocument: null,  // sentinel: LastDocument value at the time it was last restored; null means never restored
    jupyterCodeBlocks: {},
    jupyterBlockCounter: 0,
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
    aiSessionCache: '',
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
    lspChangeTimer: null,
    lspOpenFile: '',
    typosOpenFile: '',
    typosChangeTimer: null,
    lspHoverTimer: null,
    lspHoverLastKey: '',
    lspHoverMouseX: 0,
    lspHoverMouseY: 0,
    lspInlayHints: [],
    lspInlayRequestId: 0,
    lspCompletionItems: [],
    lspCompletionIndex: 0,
    lspCompletionVisible: false,
    lastEditorInputAt: 0,
};

const LSP_CHANGE_DEBOUNCE_MS = 200;
const LSP_DIAGNOSTIC_RENDER_IDLE_MS = 220;
const LSP_HOVER_DEBOUNCE_MS = 250;
const AI_BOTTOM_THRESHOLD_PX = 24;
// How long the bottom-chase keeps running after the last request for it.
const AI_BOTTOM_CHASE_MS = 400;
const NOTE_LOCATIONS = ['$GLOBAL', '$NOTES', '$PROJECT'];

let monacoMainEditor = null;
let suppressMonacoChange = false;
let latestWindowStyle = null;
let aiPromptJumpRefreshTimer = null;
let aiPromptJumpObserver = null;
let aiBottomScrollRetryTimers = [];
let aiBottomChaseHandle = 0;
let aiBottomChaseUntil = 0;
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

function isMonacoPhase0Enabled() {
    try {
        const userAgent = String(globalThis?.navigator?.userAgent || '');
        if (/jsdom/i.test(userAgent)) {
            return false;
        }

        const value = window.localStorage.getItem('notes-editor-monaco');
        return value !== '0';
    } catch {
        return true;
    }
}

function isMonacoActive() {
    return state.useMonacoEditor && monacoMainEditor !== null;
}

function getMainEditorValue() {
    if (isMonacoActive()) {
        return monacoMainEditor.getValue();
    }
    return String(elements.editor?.value || '');
}

function setMainEditorValue(value) {
    const text = String(value || '');
    if (elements.editor) {
        elements.editor.value = text;
    }

    if (isMonacoActive()) {
        suppressMonacoChange = true;
        monacoMainEditor.setValue(text);
        suppressMonacoChange = false;
    }
}

function getMainEditorSelectionRange() {
    if (isMonacoActive()) {
        return monacoMainEditor.getSelectionOffsets();
    }

    const start = Number(elements.editor?.selectionStart) || 0;
    const end = Number(elements.editor?.selectionEnd) || start;
    return { start, end };
}

function setMainEditorSelectionRange(start, end) {
    if (isMonacoActive()) {
        monacoMainEditor.setSelectionOffsets(start, end);
        return;
    }

    elements.editor.setSelectionRange(start, end);
}

function getMainEditorSelectionText() {
    if (isMonacoActive()) {
        return monacoMainEditor.getSelectionText();
    }

    const range = getMainEditorSelectionRange();
    return getMainEditorValue().slice(range.start, range.end);
}

function replaceMainEditorRange(start, end, text) {
    if (isMonacoActive()) {
        monacoMainEditor.replaceRange(start, end, text, 'notes-main-editor-replace');
        const nextCursor = Number(start) + String(text || '').length;
        monacoMainEditor.setSelectionOffsets(nextCursor, nextCursor);
        return;
    }

    elements.editor.focus();
    elements.editor.setSelectionRange(start, end);
    document.execCommand('insertText', false, String(text || ''));
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
    const computed = elements.editor ? getComputedStyle(elements.editor) : null;
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

async function ensureMonacoMainEditor() {
    if (!state.useMonacoEditor || !elements.monacoEditor || monacoMainEditor) {
        return;
    }

    const typography = getMonacoTypographyOptions();

    monacoMainEditor = await createMonacoAdapter(elements.monacoEditor, {
        value: String(elements.editor?.value || ''),
        language: state.editorLanguage || 'plaintext',
        highlightJs: hljs,
        fontSize: typography.fontSize,
        fontFamily: typography.fontFamily,
        lineHeight: typography.lineHeight,
        themeColors: latestWindowStyle?.colors || null,
        onChange: (nextValue) => {
            if (suppressMonacoChange) {
                return;
            }

            if (elements.editor) {
                elements.editor.value = String(nextValue || '');
                elements.editor.dispatchEvent(new Event('input', { bubbles: true }));
            }
        },
        onSelectionChange: (start, end) => {
            if (!elements.editor) {
                return;
            }
            elements.editor.selectionStart = Number(start) || 0;
            elements.editor.selectionEnd = Number(end) || Number(start) || 0;
            if (isCurrentFileLspEligible() && state.lspOpenFile === state.currentFile) {
                state.lspHoverLastKey = '';
                scheduleLspHover();
            }
        },
        onPointerMove: (x, y) => {
            state.lspHoverMouseX = Number(x) || 0;
            state.lspHoverMouseY = Number(y) || 0;
        },
        onBlur: () => {
            hideLspHoverTooltip();
        },
        onPaste: (event) => {
            handleEditorImagePaste(event);
        },
        onContextMenu: (event) => {
            openMainEditorContextMenu(event);
        },
        onCompletionRequest: (payload) => {
            return handleMonacoCompletionRequest(payload);
        },
    });

    monacoMainEditor.setWordWrap(Boolean(state.markdownWrapMode));
    if (latestWindowStyle?.colors) {
        monacoMainEditor.applyTheme(latestWindowStyle.colors);
    }

    // Monaco boot can race against file loading. Always re-sync from the hidden
    // textarea source-of-truth immediately after adapter creation.
    const latestText = String(elements.editor?.value || '');
    if (monacoMainEditor.getValue() !== latestText) {
        suppressMonacoChange = true;
        monacoMainEditor.setValue(latestText);
        suppressMonacoChange = false;
    }

    monacoMainEditor.setLanguage(state.editorLanguage || 'plaintext');
    monacoMainEditor.setTypography(getMonacoTypographyOptions());
    const spellcheckMisspellings = notesSpellCheckHandle?.getMisspellings?.();
    if (Array.isArray(spellcheckMisspellings)) {
        monacoMainEditor.setTyposMisspellings(spellcheckMisspellings);
    }
    monacoMainEditor.layout();

    requestAnimationFrame(() => {
        if (!isMonacoActive()) {
            return;
        }
        monacoMainEditor.layout();
    });
}

// Force Monaco to re-measure against its real, visible box across the next two
// animation frames. A display:none -> visible transition (tab switch, pane
// maximize/restore) doesn't give Monaco a correct size in the same frame, so a
// single synchronous layout() can latch a stale/zero viewport (blank editor or
// content offset with a top shadow). Two deferred measured layouts fix that.
function scheduleMonacoRelayout() {
    if (!isMonacoActive()) {
        return;
    }
    requestAnimationFrame(() => {
        if (!isMonacoActive()) {
            return;
        }
        monacoMainEditor.layout();
        requestAnimationFrame(() => {
            if (isMonacoActive()) {
                monacoMainEditor.layout();
            }
        });
    });
}

// Lazily create Monaco (only ever while its container is visible/sized) and push
// the current document, language, wrap mode and theme before laying it out. This
// is the single entry point used whenever the editor tab becomes visible.
async function ensureMonacoVisibleAndLaidOut() {
    if (!state.useMonacoEditor) {
        return;
    }

    await ensureMonacoMainEditor();
    if (!isMonacoActive()) {
        return;
    }

    const latestText = String(elements.editor?.value || '');
    if (monacoMainEditor.getValue() !== latestText) {
        suppressMonacoChange = true;
        monacoMainEditor.setValue(latestText);
        suppressMonacoChange = false;
    }

    monacoMainEditor.setLanguage(state.editorLanguage || 'plaintext');
    monacoMainEditor.setWordWrap(Boolean(state.markdownWrapMode));
    monacoMainEditor.setTypography(getMonacoTypographyOptions());
    const spellcheckMisspellings = notesSpellCheckHandle?.getMisspellings?.();
    if (Array.isArray(spellcheckMisspellings)) {
        monacoMainEditor.setTyposMisspellings(spellcheckMisspellings);
    }
    if (latestWindowStyle?.colors) {
        monacoMainEditor.applyTheme(latestWindowStyle.colors);
    }

    monacoMainEditor.layout();
    scheduleMonacoRelayout();
}

function syncEditorEngineMode() {
    if (!elements.editorShell) {
        return;
    }

    elements.editorShell.dataset.monacoEnabled = state.useMonacoEditor ? 'true' : 'false';
}

configureMarked();
state.useMonacoEditor = isMonacoPhase0Enabled();
syncEditorEngineMode();

// Belt-and-suspenders relayout: any size change to the editor box (pane
// maximize/restore, splitter drag, window resize) triggers a measured Monaco
// relayout. This complements Monaco's own automaticLayout, which can latch a
// stale viewport across display:none -> visible transitions.
if (typeof ResizeObserver !== 'undefined' && elements.monacoEditor?.parentElement) {
    const monacoResizeObserver = new ResizeObserver(() => {
        scheduleMonacoRelayout();
    });
    monacoResizeObserver.observe(elements.monacoEditor.parentElement);
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

function syncEditorScrollDecorations() {
    if (isMonacoActive()) {
        monacoMainEditor.layout();
    }
}

function renderEditorDecorations() {
    if (!isMonacoActive()) {
        return;
    }
    renderLspEditorDecorations();
    monacoMainEditor.layout();
}

function refreshEditorLanguage(file, content) {
    state.editorLanguage = inferEditorLanguage(file, content);
    if (isMonacoActive()) {
        monacoMainEditor.setLanguage(state.editorLanguage || 'plaintext');
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

    // Monaco mode owns completion for the main Edit surface.
    if (isMonacoActive() && textarea === elements.editor) {
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
    const rows = parseCsv(elements.editor?.value || '');
    while (rows.length <= rowIndex) {
        rows.push([]);
    }

    const row = rows[rowIndex];
    while (row.length <= columnIndex) {
        row.push('');
    }

    row[columnIndex] = String(value ?? '').trim();

    notesMutationAdapter.replaceDocumentText(elements.editor, {
        text: serializeCsvRows(rows),
        selectionStart: Number(elements.editor.selectionStart) || 0,
        selectionEnd: Number(elements.editor.selectionEnd) || 0,
        source: 'csv-table-edit',
        label: 'Update CSV cell',
        emit: true,
    });
    setDirty(true);
    renderCsvView(elements.editor.value, { interactive: state.viewMode === 'csv-run' });
    scheduleAutoSave();
    saveFile();

    return true;
}

function insertCsvRowAfter(rowIndex) {
    const rows = parseCsv(elements.editor?.value || '');
    const colCount = rows.length > 0 ? Math.max(...rows.map(r => r.length)) : 1;
    const newRow = Array(colCount).fill('');
    rows.splice(rowIndex + 1, 0, newRow);
    notesMutationAdapter.replaceDocumentText(elements.editor, {
        text: serializeCsvRows(rows),
        selectionStart: Number(elements.editor.selectionStart) || 0,
        selectionEnd: Number(elements.editor.selectionEnd) || 0,
        source: 'csv-table-edit',
        label: 'Insert CSV row',
        emit: true,
    });
    setDirty(true);
    renderCsvView(elements.editor.value, { interactive: state.viewMode === 'csv-run' });
    scheduleAutoSave();
    saveFile();
}

function insertCsvColumnAfter(columnIndex) {
    const rows = parseCsv(elements.editor?.value || '');
    rows.forEach(row => {
        while (row.length <= columnIndex) row.push('');
        row.splice(columnIndex + 1, 0, '');
    });
    notesMutationAdapter.replaceDocumentText(elements.editor, {
        text: serializeCsvRows(rows),
        selectionStart: Number(elements.editor.selectionStart) || 0,
        selectionEnd: Number(elements.editor.selectionEnd) || 0,
        source: 'csv-table-edit',
        label: 'Insert CSV column',
        emit: true,
    });
    setDirty(true);
    renderCsvView(elements.editor.value, { interactive: state.viewMode === 'csv-run' });
    scheduleAutoSave();
    saveFile();
}

function deleteCsvRow(rowIndex) {
    const rows = parseCsv(elements.editor?.value || '');
    if (rowIndex < 0 || rowIndex >= rows.length) {
        return;
    }

    // Keep at least one row so table structure remains valid.
    if (rows.length <= 1) {
        return;
    }

    rows.splice(rowIndex, 1);
    notesMutationAdapter.replaceDocumentText(elements.editor, {
        text: serializeCsvRows(rows),
        selectionStart: Number(elements.editor.selectionStart) || 0,
        selectionEnd: Number(elements.editor.selectionEnd) || 0,
        source: 'csv-table-edit',
        label: 'Delete CSV row',
        emit: true,
    });
    setDirty(true);
    renderCsvView(elements.editor.value, { interactive: state.viewMode === 'csv-run' });
    scheduleAutoSave();
    saveFile();
}

function deleteCsvColumn(columnIndex) {
    const rows = parseCsv(elements.editor?.value || '');
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

    notesMutationAdapter.replaceDocumentText(elements.editor, {
        text: serializeCsvRows(rows),
        selectionStart: Number(elements.editor.selectionStart) || 0,
        selectionEnd: Number(elements.editor.selectionEnd) || 0,
        source: 'csv-table-edit',
        label: 'Delete CSV column',
        emit: true,
    });
    setDirty(true);
    renderCsvView(elements.editor.value, { interactive: state.viewMode === 'csv-run' });
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
                cell.setAttribute('spellcheck', 'false');
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
    setupTableSorting(elements.csvView);
    void setupTableColumnResizing(elements.csvView, false, state.currentFile);
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
    SendIpc('terminal-sticky-create', {
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
    const raw = elements.editor?.value || '';
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

    // If the Frontmatter tab was the active tab but is no longer available,
    // close the Tools panel rather than silently switching tabs.
    if (elements.toolsTabFrontmatter?.getAttribute('aria-selected') === 'true') {
        setToolsPanelCollapsed(true);
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

async function renderMarkdown() {
    const markdown = applyDocumentFrontmatter();
    elements.preview.innerHTML = marked.parse(markdown);

    // Apply common markdown processing
    await processMarkdownContainer(elements.preview);

    // Enable context menus on images
    enableImageContextMenus(elements.preview);

    // Keep checkboxes readonly in viewer mode
    setupInteractiveCheckboxes(elements.preview, false);

    // Enable collapsible H1-H6 sections
    setupCollapsibleHeadings(elements.preview);

    // Keep table horizontal scrolling local to each table rather than the whole section.
    wrapTablesForHorizontalScroll(elements.preview);

    // Enable column sorting on all tables
    setupTableSorting(elements.preview);

    // Apply the word-wrap CSS class and enable resizable table columns with persisted widths.
    elements.preview.classList.toggle('notes-table-wordwrap-on', state.markdownTableWordWrapMode);
    await setupTableColumnResizing(elements.preview, state.markdownTableWordWrapMode, state.currentFile);

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
    elements.htmlViewFrame.srcdoc = String(elements.editor?.value || '');
}

function getToCHeadingKey(level, text) {
    return `${String(level)}::${String(text || '').trim().toLowerCase()}`;
}

function getHeadingDescriptors(root) {
    const headings = Array.from(root?.querySelectorAll?.('h1, h2, h3, h4, h5, h6') || []);
    const seen = new Map();
    const descriptors = [];

    headings.forEach((heading) => {
        const level = Number.parseInt(String(heading.tagName).slice(1), 10) || 1;
        const text = String(heading.textContent || '').trim();
        if (!text) {
            return;
        }

        const key = getToCHeadingKey(level, text);
        const occurrence = seen.get(key) || 0;
        seen.set(key, occurrence + 1);

        descriptors.push({
            heading,
            level,
            text,
            occurrence,
            anchor: String(heading.id || ''),
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

    const descriptors = getHeadingDescriptors(elements.preview);
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

    // If the previously-selected tab is no longer available for this document
    // type, close the Tools panel rather than silently switching tabs. The ToC
    // and Frontmatter tabs are the only document-type-specific tabs.
    const tocActive = elements.toolsTabToC?.getAttribute('aria-selected') === 'true';
    const frontmatterActive = elements.toolsTabFrontmatter?.getAttribute('aria-selected') === 'true';
    if (!showToC && (tocActive || frontmatterActive)) {
        setToolsPanelCollapsed(true);
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

function setupCollapsibleHeadings(container, exclusive = false) {
    const headings = container.querySelectorAll('h1, h2, h3, h4, h5, h6');
    const headingList = Array.from(headings);

    headingList.forEach((heading) => {
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

        heading.style.cursor = 'pointer';

        heading.addEventListener('mouseenter', () => {
            heading.style.textDecoration = 'underline';
        });
        heading.addEventListener('mouseleave', () => {
            heading.style.textDecoration = '';
        });

        heading.addEventListener('dblclick', () => {
            if (exclusive) {
                headingList.forEach((otherHeading) => {
                    setCollapsibleHeadingState(otherHeading, otherHeading !== heading);
                });
                return;
            }

            const isCollapsed = heading.dataset.collapsed === 'true';

            if (isCollapsed) {
                // Expand: animate from 0 → scrollHeight, then release max-height cap.
                expandCollapsibleHeading(heading);
            } else {
                // Collapse: pin to exact current height (no jump), then animate to 0.
                collapseCollapsibleHeading(heading);
            }
        });
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

function updateMarkdownTableCell(block, sourceRowIndex, columnIndex, value) {
    if (!block || !Array.isArray(block.rowLineIndexes)) {
        return false;
    }

    const lineIndex = block.rowLineIndexes[sourceRowIndex];
    if (!Number.isInteger(lineIndex)) {
        return false;
    }

    const lines = String(elements.editor.value || '').split('\n');
    if (lineIndex < 0 || lineIndex >= lines.length) {
        return false;
    }

    const parsed = parseMarkdownTableRow(lines[lineIndex]);
    while (parsed.cells.length <= columnIndex) {
        parsed.cells.push('');
    }

    parsed.cells[columnIndex] = String(value ?? '').trim();
    lines[lineIndex] = serializeMarkdownTableRow(parsed.cells, parsed.hasLeadingPipe, parsed.hasTrailingPipe);
    elements.editor.value = lines.join('\n');

    setDirty(true);
    scheduleRender();
    scheduleAutoSave();
    saveFile();

    return true;
}

function insertMarkdownRowAfter(block, sourceRowIndex) {
    if (!block || !Array.isArray(block.rowLineIndexes)) return;

    const lines = String(elements.editor.value || '').split('\n');
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
    elements.editor.value = lines.join('\n');

    setDirty(true);
    scheduleRender();
    scheduleAutoSave();
    saveFile();
    renderJupyterView();
}

function insertMarkdownColumnAfter(block, columnIndex) {
    if (!block || !Array.isArray(block.rowLineIndexes)) return;

    const lines = String(elements.editor.value || '').split('\n');

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

    elements.editor.value = lines.join('\n');

    setDirty(true);
    scheduleRender();
    scheduleAutoSave();
    saveFile();
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

    const lines = String(elements.editor.value || '').split('\n');
    const lineIndex = block.rowLineIndexes[sourceRowIndex];
    if (!Number.isInteger(lineIndex) || lineIndex < 0 || lineIndex >= lines.length) {
        return;
    }

    lines.splice(lineIndex, 1);
    elements.editor.value = lines.join('\n');

    setDirty(true);
    scheduleRender();
    scheduleAutoSave();
    saveFile();
    renderJupyterView();
}

function deleteMarkdownColumn(block, columnIndex) {
    if (!block || !Array.isArray(block.rowLineIndexes)) {
        return;
    }

    const lines = String(elements.editor.value || '').split('\n');
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

    elements.editor.value = lines.join('\n');

    setDirty(true);
    scheduleRender();
    scheduleAutoSave();
    saveFile();
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

async function runMarkdownTableFunction(container, fnName, fnArgs, row, col, rows, cellId) {
    const block = findMarkdownFunctionCodeBlock(container, fnName);
    if (!block) {
        return '#ERR';
    }

    const currentCellId = String(cellId || getCellReference(row, col));

    // Create a function executor for nested function calls
    const functionExecutor = async (nestedFnName, nestedFnArgs, nestedRow, nestedCol) => {
        return runMarkdownTableFunction(container, nestedFnName, nestedFnArgs, nestedRow, nestedCol, rows, currentCellId);
    };

    // Resolve arguments with support for nested function calls
    const resolvedArgArrays = await Promise.all(
        fnArgs.map((arg) => resolveTableFunctionArgsAsync(functionExecutor, arg, row, col, rows))
    );
    const resolvedArgs = resolvedArgArrays.flatMap((arr) => arr);

    const runtime = await resolveRuntimeForFunctionLanguage(block.language);

    try {
        const result = await RunFunction(state.currentFile, fnName, currentCellId, block.code, resolvedArgs, runtime);

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

async function evaluateTableFormulasInPlace(container) {
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
                cell.innerHTML = `<span class="notes-table-cell-wrap"><span>${escapeHtml(val)}</span><span class="notes-cellref">${ref}</span></span>`;
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
                    ? await runMarkdownTableFunction(container, fnCall.fnName, fnCall.args, row, col, rows, ref)
                    : evaluateTableFormula(val, row, col, rows);

                rows[row][col] = String(content ?? '');
                cell.dataset.formula = val;
                cell.innerHTML = `<span class="notes-table-cell-wrap"><span>${escapeHtml(content)}</span><span class="notes-cellref">${ref}</span></span>`;
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
                ? await runMarkdownTableFunction(container, fnCall.fnName, fnCall.args, row, col, rows, ref)
                : evaluateTableFormula(val, row, col, rows);
            rows[row][col] = String(content ?? '');
            cell.dataset.formula = val;
            cell.innerHTML = `<span class="notes-table-cell-wrap"><span>${escapeHtml(content)}</span><span class="notes-cellref">${ref}</span></span>`;

            if (!isJsdom) {
                await flushUiPaint();
            }
        }
    }
}

function setupInteractiveMarkdownTables(container, isEditable) {
    const blocks = findMarkdownTableBlocks(elements.editor?.value || '');

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

async function setupTableColumnResizing(container, wrapped, filename = state.currentFile) {
    if (!container || filename === null) {
        return;
    }

    const viewName = container === elements.preview
        ? 'view'
        : container === elements.jupyter
            ? 'run'
            : container === elements.csvView
                ? 'csv'
                : container === elements.aiOutput
                    ? 'ai-panel'
                    : String(state.viewMode || 'view');

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

        const storedWidths = await GetNotesColumnWidths(filename, viewName, headings, wrapped).catch(() => []);
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

                SetNotesColumnWidths(filename, viewName, headings, wrapped, widths).catch((err) => {
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

function setupTableSorting(container) {
    if (!container) return;

    const getCellText = (cell) => {
        return getTableCellTextContent(cell);
    };

    Array.from(container.querySelectorAll('table')).forEach((table) => {
        const tbody = table.querySelector('tbody');
        if (!tbody) return;
        const headerRow = table.querySelector('thead tr');
        if (!headerRow) return;
        const headerCells = Array.from(headerRow.querySelectorAll('th'));

        // Stamp original order so we can restore it on clear
        Array.from(tbody.querySelectorAll('tr')).forEach((row, i) => {
            row.dataset.originalSortOrder = String(i);
        });

        const clearSortIcons = () => {
            headerCells.forEach((th) => {
                const icon = th.querySelector('.notes-sort-icon');
                if (icon) icon.remove();
                delete th.dataset.sortType;
            });
        };

        const clearSort = () => {
            clearSortIcons();
            const rows = Array.from(tbody.querySelectorAll('tr'));
            rows.sort((a, b) => Number(a.dataset.originalSortOrder) - Number(b.dataset.originalSortOrder));
            rows.forEach(row => tbody.appendChild(row));
        };

        const applySort = (colIndex, sortType) => {
            clearSortIcons();
            const rows = Array.from(tbody.querySelectorAll('tr'));
            rows.sort((a, b) => {
                const aText = getCellText(a.querySelectorAll('td, th')[colIndex] || a);
                const bText = getCellText(b.querySelectorAll('td, th')[colIndex] || b);
                if (sortType === 'num-asc')  return (parseFloat(aText) || 0) - (parseFloat(bText) || 0);
                if (sortType === 'num-desc') return (parseFloat(bText) || 0) - (parseFloat(aText) || 0);
                if (sortType === 'char-asc')  return aText.localeCompare(bText);
                if (sortType === 'char-desc') return bText.localeCompare(aText);
                return 0;
            });
            rows.forEach(row => tbody.appendChild(row));

            // Stamp sort icon onto the header cell
            const th = headerCells[colIndex];
            if (th) {
                th.dataset.sortType = sortType;
                const iconCodePoint = { 'num-asc': 0xf162, 'num-desc': 0xf886, 'char-asc': 0xf15d, 'char-desc': 0xf881 }[sortType];
                const iconSpan = document.createElement('span');
                iconSpan.className = 'notes-sort-icon';
                iconSpan.textContent = String.fromCodePoint(iconCodePoint);
                th.prepend(iconSpan);
            }
        };

        headerCells.forEach((th, colIndex) => {
            th.addEventListener('click', (e) => {
                if (table.dataset.resizeDragActive === 'true') {
                    e.preventDefault();
                    e.stopPropagation();
                    return;
                }

                e.preventDefault();
                e.stopPropagation();

                const headerText = getCellText(th) || `Column ${colIndex + 1}`;
                const menuItems = [
                    { title: 'Sort by number (low to high)',     icon: 0xf162, onSelect: () => { applySort(colIndex, 'num-asc'); clearTableHighlight(table); } },
                    { title: 'Sort by number (high to low)',     icon: 0xf886, onSelect: () => { applySort(colIndex, 'num-desc'); clearTableHighlight(table); } },
                    { title: 'Sort by characters (low to high)', icon: 0xf15d, onSelect: () => { applySort(colIndex, 'char-asc'); clearTableHighlight(table); } },
                    { title: 'Sort by characters (high to low)', icon: 0xf881, onSelect: () => { applySort(colIndex, 'char-desc'); clearTableHighlight(table); } },
                    { title: '-' },
                    { title: 'Clear sorting', icon: 0, onSelect: () => { clearSort(); clearTableHighlight(table); } },
                ];

                const highlightCallback = (itemIndex) => {
                    const item = menuItems[itemIndex];
                    if (!item) return;
                    clearTableHighlight(table);
                    if (item.title === 'Clear sorting') {
                        highlightEntireTable(table, true);
                    } else if (item.title.startsWith('Sort')) {
                        highlightTableColumn(table, colIndex, true);
                    }
                };

                showNotesLocalMenu(
                    menuItems,
                    e.clientX,
                    e.clientY,
                    `Sort: ${headerText}`,
                    highlightCallback,
                    () => clearTableHighlight(table),
                );
            });
        });
    });
}

function toggleCheckboxInMarkdown(checkboxIndex, isChecked) {
    const lines = elements.editor.value.split('\n');
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
        notesMutationAdapter.replaceDocumentText(elements.editor, {
            text: lines.join('\n'),
            selectionStart: Number(elements.editor.selectionStart) || 0,
            selectionEnd: Number(elements.editor.selectionEnd) || 0,
            source: 'markdown-checkbox',
            label: 'Toggle markdown checkbox',
            emit: true,
        });
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

    const source = String(elements.editor.value || '');
    const result = updateMarkdownTableOfContentsText(source);
    if (!result.updated) {
        notifyTerminal('No headings found to generate a table of contents.', 'info');
        return;
    }

    notesMutationAdapter.replaceDocumentText(elements.editor, {
        text: result.text,
        selectionStart: 0,
        selectionEnd: 0,
        source: 'markdown-toc',
        label: 'Update markdown table of contents',
        emit: true,
    });
    if (usesCodeEditorDecorations()) {
        refreshEditorLanguage(state.currentFile, elements.editor.value);
    }
    hideLspHoverTooltip();
    hideLspCompletion();
    elements.editor.focus();
    elements.editor.setSelectionRange(0, 0);
    elements.editor.scrollTop = 0;
    elements.editor.scrollLeft = 0;
    if (usesCodeEditorDecorations()) {
        syncEditorScrollDecorations();
    }
    setDirty(true);
    scheduleRender();
    scheduleAutoSave();
    saveFile();
    notifyTerminal('Table of contents updated.', 'info');
}

function updateMarkdownCodeBlock(blockIndex, newContent) {
    const markdown = elements.editor.value;
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
    notesMutationAdapter.replaceDocumentText(elements.editor, {
        text: result,
        selectionStart: Number(elements.editor.selectionStart) || 0,
        selectionEnd: Number(elements.editor.selectionEnd) || 0,
        source: 'markdown-code-block',
        label: 'Update markdown code block',
        emit: true,
    });
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
    if (state.autosaveTimer) {
        clearTimeout(state.autosaveTimer);
    }
    state.autosaveTimer = setTimeout(() => {
        state.autosaveTimer = null;
        saveFile();
    }, 1000);
}

function isCurrentFileLspEligible() {
    return state.currentFileType === 'code' || state.currentFileType === 'json' || state.currentFileType === 'markdown' || state.currentFileType === 'html';
}

// Returns the active Jupyter code block LSP context, or null when the main editor is active.
function getActiveLspTarget() {
    if (!state.lspActiveBlockId || !state.lspActiveBlockEditor) {
        return null;
    }
    const block = state.jupyterCodeBlocks[state.lspActiveBlockId];
    if (!block || !block.lspMode || !block.lspFilePath) {
        return null;
    }
    return { filePath: block.lspFilePath, editor: state.lspActiveBlockEditor };
}

function clearLspChangeTimer() {
    if (state.lspChangeTimer) {
        clearTimeout(state.lspChangeTimer);
        state.lspChangeTimer = null;
    }
}

function clearLspHoverTimer() {
    if (state.lspHoverTimer) {
        clearTimeout(state.lspHoverTimer);
        state.lspHoverTimer = null;
    }
}

function hideLspHoverTooltip() {
    if (lspHoverTooltipEl) {
        lspHoverTooltipEl.style.display = 'none';
        lspHoverTooltipEl.innerHTML = '';
    }
}

function hideLspCompletion() {
    state.lspCompletionVisible = false;
    state.lspCompletionItems = [];
    state.lspCompletionIndex = 0;
    if (lspCompletionEl) {
        lspCompletionEl.style.display = 'none';
        lspCompletionEl.innerHTML = '';
    }
}

function closeOpenLspTooltips() {
    let closedAny = false;

    if (lspTooltipEl && lspTooltipEl.style.display !== 'none' && lspTooltipEl.style.display !== '') {
        lspTooltipEl.style.display = 'none';
        closedAny = true;
    }

    if (lspHoverTooltipEl && lspHoverTooltipEl.style.display !== 'none' && lspHoverTooltipEl.style.display !== '') {
        hideLspHoverTooltip();
        closedAny = true;
    }

    if (lspCompletionEl && lspCompletionEl.style.display !== 'none' && lspCompletionEl.style.display !== '') {
        hideLspCompletion();
        closedAny = true;
    }

    return closedAny;
}

function lspCompletionKindMeta(kind) {
    switch (Number(kind || 0)) {
    case 2:
        return { iconClass: 'codicon codicon-symbol-method', badge: 'method', title: 'Method' };
    case 3:
        return { iconClass: 'codicon codicon-symbol-function', badge: 'function', title: 'Function' };
    case 4:
        return { iconClass: 'codicon codicon-symbol-constructor', badge: 'constructor', title: 'Constructor' };
    case 5:
        return { iconClass: 'codicon codicon-symbol-field', badge: 'field', title: 'Field' };
    case 6:
        return { iconClass: 'codicon codicon-symbol-variable', badge: 'variable', title: 'Variable' };
    case 7:
        return { iconClass: 'codicon codicon-symbol-class', badge: 'class', title: 'Class' };
    case 8:
        return { iconClass: 'codicon codicon-symbol-interface', badge: 'interface', title: 'Interface' };
    case 9:
        return { iconClass: 'codicon codicon-symbol-module', badge: 'module', title: 'Module' };
    case 10:
        return { iconClass: 'codicon codicon-symbol-property', badge: 'property', title: 'Property' };
    case 11:
        return { iconClass: 'codicon codicon-symbol-unit', badge: 'unit', title: 'Unit' };
    case 12:
        return { iconClass: 'codicon codicon-symbol-value', badge: 'value', title: 'Value' };
    case 13:
        return { iconClass: 'codicon codicon-symbol-enum', badge: 'enum', title: 'Enum' };
    case 14:
        return { iconClass: 'codicon codicon-symbol-keyword', badge: 'keyword', title: 'Keyword' };
    case 15:
        return { iconClass: 'codicon codicon-symbol-snippet', badge: 'snippet', title: 'Snippet' };
    case 17:
        return { iconClass: 'codicon codicon-symbol-file', badge: 'file', title: 'File' };
    case 18:
        return { iconClass: 'codicon codicon-symbol-reference', badge: 'reference', title: 'Reference' };
    case 19:
        return { iconClass: 'codicon codicon-folder', badge: 'folder', title: 'Folder' };
    case 21:
        return { iconClass: 'codicon codicon-symbol-constant', badge: 'constant', title: 'Constant' };
    case 22:
        return { iconClass: 'codicon codicon-symbol-struct', badge: 'struct', title: 'Struct' };
    case 24:
        return { iconClass: 'codicon codicon-symbol-event', badge: 'event', title: 'Event' };
    case 25:
        return { iconClass: 'codicon codicon-symbol-operator', badge: 'operator', title: 'Operator' };
    case 26:
        return { iconClass: 'codicon codicon-symbol-type-parameter', badge: 'type', title: 'Type parameter' };
    default:
        return { iconClass: 'codicon codicon-symbol-misc', badge: '', title: 'Symbol' };
    }
}

function replaceCurrentIdentifierWithCompletion(text) {
    const target = getActiveLspTarget();

    if (!target && isMonacoActive()) {
        const source = monacoMainEditor.getValue() || '';
        const selection = monacoMainEditor.getSelectionOffsets();
        const cursor = selection.start || 0;
        const left = source.slice(0, cursor);
        const match = left.match(/[A-Za-z0-9_-]+$/);
        const start = match ? cursor - match[0].length : cursor;
        const insertText = String(text || '');

        hideLspCompletion();
        monacoMainEditor.replaceRange(start, cursor, insertText, 'lsp-completion');
        const next = start + insertText.length;
        monacoMainEditor.setSelectionOffsets(next, next);
        monacoMainEditor.focus();
        return;
    }

    const editor = target ? target.editor : elements.editor;
    if (!editor) {
        return;
    }

    const source = editor.value || '';
    const cursor = editor.selectionStart || 0;
    const left = source.slice(0, cursor);
    const match = left.match(/[A-Za-z0-9_-]+$/);
    const start = match ? cursor - match[0].length : cursor;
    // Hide completion BEFORE dispatching input event so the input handler doesn't try to filter.
    hideLspCompletion();
    notesMutationAdapter.replaceRange(editor, {
        start,
        end: cursor,
        text,
        source: 'lsp-completion',
        label: 'Apply completion item',
        emit: true,
    });
}

function commitActiveLspCompletion() {
    if (!state.lspCompletionVisible || state.lspCompletionItems.length === 0) {
        return false;
    }

    const index = Math.max(0, Math.min(state.lspCompletionIndex, state.lspCompletionItems.length - 1));
    const item = state.lspCompletionItems[index];
    if (!item) {
        return false;
    }

    replaceCurrentIdentifierWithCompletion(item.insertText || item.label);
    hideLspCompletion();
    return true;
}

function renderLspCompletionPopup() {
    if (!lspCompletionEl) {
        return;
    }

    if (!state.lspCompletionVisible || state.lspCompletionItems.length === 0) {
        hideLspCompletion();
        return;
    }

    const activeIndex = Math.max(0, Math.min(state.lspCompletionIndex, state.lspCompletionItems.length - 1));
    state.lspCompletionIndex = activeIndex;

    const fragment = document.createDocumentFragment();
    state.lspCompletionItems.forEach((item, idx) => {
        const row = document.createElement('div');
        row.className = 'tty-menu-row notes-lsp-completion-item';
        row.dataset.active = idx === activeIndex ? 'true' : 'false';
        row.dataset.deprecated = item.deprecated === true ? 'true' : 'false';
        row.classList.toggle('is-active', idx === activeIndex);
        row.classList.toggle('is-deprecated', item.deprecated === true);

        const kind = lspCompletionKindMeta(item.kind);

        const icon = document.createElement('span');
        icon.className = `tty-menu-row-icon notes-lsp-completion-icon ${kind.iconClass}`;
        icon.title = kind.title;
        icon.setAttribute('aria-label', kind.title);
        row.appendChild(icon);

        const label = document.createElement('span');
        label.className = 'tty-menu-row-label notes-lsp-completion-label';
        label.textContent = String(item.label || '');
        row.appendChild(label);

        if (item.detail) {
            const detail = document.createElement('span');
            detail.className = 'notes-lsp-completion-detail';
            detail.textContent = String(item.detail);
            row.appendChild(detail);
        }

        if (kind.badge) {
            const badge = document.createElement('span');
            badge.className = 'notes-lsp-completion-kind';
            badge.textContent = kind.badge;
            badge.title = kind.title;
            row.appendChild(badge);
        }

        row.addEventListener('mousedown', (event) => {
            event.preventDefault();
            state.lspCompletionIndex = idx;
            commitActiveLspCompletion();
        });

        fragment.appendChild(row);
    });

    lspCompletionEl.innerHTML = '';
    lspCompletionEl.appendChild(fragment);
    lspCompletionEl.style.display = 'block';

    const anchor = (isMonacoActive() && !state.lspActiveBlockId)
        ? getMonacoCompletionMenuAnchor()
        : getLspAnchorViewportPoint();
    const rawX = anchor.x;
    const rawY = anchor.y;
    const x = Math.min(rawX + 14, window.innerWidth - lspCompletionEl.offsetWidth - 8);
    const y = Math.min(rawY + 18, window.innerHeight - lspCompletionEl.offsetHeight - 8);
    lspCompletionEl.style.left = `${Math.max(8, x)}px`;
    lspCompletionEl.style.top = `${Math.max(8, y)}px`;
}

function moveLspCompletionSelection(delta) {
    if (!state.lspCompletionVisible || state.lspCompletionItems.length === 0) {
        return false;
    }

    const count = state.lspCompletionItems.length;
    const current = Math.max(0, Math.min(state.lspCompletionIndex, count - 1));
    const next = (current + delta + count) % count;
    state.lspCompletionIndex = next;
    renderLspCompletionPopup();

    if (lspCompletionEl) {
        const activeRow = lspCompletionEl.querySelector('.notes-lsp-completion-item.is-active');
        if (activeRow && typeof activeRow.scrollIntoView === 'function') {
            activeRow.scrollIntoView({ block: 'nearest' });
        }
    }

    return true;
}

async function requestLspCompletion(triggerKind = 1, triggerChar = '') {
    const blockTarget = getActiveLspTarget();
    if (!blockTarget && (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible())) {
        hideLspCompletion();
        return;
    }

    const completionFile = blockTarget ? blockTarget.filePath : state.currentFile;
    const completionSource = blockTarget ? (blockTarget.editor.value || '') : getMainEditorValue();
    const completionSelection = blockTarget
        ? { start: blockTarget.editor.selectionStart || 0 }
        : getMainEditorSelectionRange();
    const pos = offsetToLspPosition(completionSource, completionSelection.start || 0);
    try {
        const items = await NotesLspCompletion(completionFile, pos.line, pos.character, triggerKind, triggerChar);
        const list = Array.isArray(items) ? items : [];
        if (list.length === 0) {
            hideLspCompletion();
            return;
        }

        state.lspCompletionItems = list;
        state.lspCompletionIndex = 0;
        state.lspCompletionVisible = true;
        renderLspCompletionPopup();
    } catch {
        hideLspCompletion();
    }
}

function getMonacoCompletionMenuAnchor() {
    const caretPoint = monacoMainEditor?.getCursorViewportPoint?.();
    if (caretPoint && Number.isFinite(caretPoint.x) && Number.isFinite(caretPoint.y)) {
        return caretPoint;
    }

    const rect = elements.monacoEditor?.getBoundingClientRect?.();
    if (rect) {
        return {
            x: rect.left + 24,
            y: rect.top + 24,
        };
    }

    return getLspAnchorViewportPoint();
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
    const selection = monacoMainEditor?.getSelectionOffsets?.();
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

    const selection = monacoMainEditor.getSelectionOffsets();
    const start = Number(selection?.start) || 0;
    const end = Number(selection?.end) || start;
    const indentation = await getIndentationString();

    hideLspCompletion();
    monacoMainEditor.replaceRange(start, end, indentation, 'notes-monaco-tab-indent');
    const next = start + indentation.length;
    monacoMainEditor.setSelectionOffsets(next, next);
    monacoMainEditor.focus();
}

async function applyMonacoLineIndentationForSelection(outdent = false) {
    if (!isMonacoActive()) {
        return false;
    }

    const selection = monacoMainEditor.getSelectionOffsets();
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
    monacoMainEditor.replaceRange(edit.start, edit.end, edit.text, outdent ? 'notes-monaco-shift-tab-outdent' : 'notes-monaco-tab-indent-lines');
    monacoMainEditor.setSelectionOffsets(edit.selectionStart, edit.selectionEnd);
    monacoMainEditor.focus();
    return true;
}

function handleMonacoCompletionRequest(payload = {}) {
    if (!isMonacoActive() || state.lspActiveBlockId) {
        return false;
    }

    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
        return false;
    }

    const source = String(payload.source || '');
    const key = String(payload.key || '');
    const ctrlKey = payload.ctrlKey === true;
    const metaKey = payload.metaKey === true;
    const altKey = payload.altKey === true;
    const shiftKey = payload.shiftKey === true;

    if (source === 'keydown' && !ctrlKey && !metaKey && !altKey && key === 'Tab') {
        const selection = monacoMainEditor?.getSelectionOffsets?.();
        const start = Number(selection?.start) || 0;
        const end = Number(selection?.end) || start;
        if (end > start) {
            void applyMonacoLineIndentationForSelection(shiftKey);
            return true;
        }
    }

    if (source === 'keydown' && state.lspCompletionVisible) {
        if (key === 'Escape') {
            hideLspCompletion();
            return true;
        }

        if (!ctrlKey && !metaKey && !altKey && key === 'ArrowDown') {
            return moveLspCompletionSelection(1);
        }

        if (!ctrlKey && !metaKey && !altKey && key === 'ArrowUp') {
            return moveLspCompletionSelection(-1);
        }

        if (!ctrlKey && !metaKey && !altKey && (key === 'Enter' || key === 'Tab')) {
            return commitActiveLspCompletion();
        }

        if (!ctrlKey && !metaKey && !altKey && (key === 'Backspace' || key === 'Delete')) {
            requestAnimationFrame(() => {
                void requestLspCompletionAfterSync(getMainEditorValue(), '', 1);
            });
        }
    }

    if (source === 'type' && (key === '.' || key === ':' || key === '>')) {
        void requestLspCompletionAfterSync(getMainEditorValue(), key, 2);
        return true;
    }

    if (source === 'type' && state.lspCompletionVisible) {
        if (/^[-_A-Za-z0-9]$/.test(key)) {
            return false;
        }

        if (!/^\s$/.test(key)) {
            hideLspCompletion();
        }
    }

    if (source === 'keydown') {
        const isTab = key === 'Tab' && !ctrlKey && !metaKey && !altKey;
        const isCtrlSpace = (key === ' ' || key === 'Space' || key === 'Spacebar')
            && (ctrlKey || metaKey)
            && !altKey;

        if (isTab && shouldMonacoTabIndent()) {
            void applyMonacoTabIndent();
            return true;
        }

        if (isTab || isCtrlSpace) {
            void requestLspCompletionAfterSync(getMainEditorValue(), '', 1);
            return true;
        }
    }

    return false;
}

async function requestLspCompletionAfterSync(content, triggerChar = '', triggerKind = 2) {
    const blockTarget = getActiveLspTarget();
    if (!blockTarget && (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible())) {
        return;
    }

    const syncFile = blockTarget ? blockTarget.filePath : state.currentFile;
    try {
        await NotesLspChangeDocument(syncFile, String(content || ''));
    } catch {
        // Fall through to completion request even when fast sync fails.
    }

    await requestLspCompletion(triggerKind, triggerChar);
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

function lspPositionToEditorOffset(content, line, character) {
    const source = String(content || '');
    const lines = source.split('\n');
    const safeLine = Math.max(0, Math.min(Number(line) || 0, Math.max(lines.length - 1, 0)));
    const lineContent = lines[safeLine] || '';
    const safeCharacter = Math.max(0, Math.min(Number(character) || 0, lineContent.length));

    let offset = 0;
    for (let i = 0; i < safeLine; i += 1) {
        offset += (lines[i] || '').length + 1;
    }

    return offset + safeCharacter;
}

// typos-lsp encodes the misspelt word and its correction(s) in the diagnostic
// message using backticks, e.g. "`teh` should be `the`" or, for multiple
// corrections, "`recieve` should be `receive`, `relieve`". The first quoted
// token is the misspelt word; the remainder are suggestions.
function parseTyposDiagnosticMessage(message) {
    const matches = String(message || '').match(/`([^`]+)`/g);
    if (!matches || matches.length === 0) {
        return { word: '', suggestions: [] };
    }
    const unquote = (s) => s.slice(1, -1);
    return {
        word: unquote(matches[0]),
        suggestions: matches.slice(1).map(unquote),
    };
}

// Convert typos-lsp diagnostics into the misspelling shape consumed by the
// spellcheck canvas overlay (absolute character offsets into `text`).
function typosDiagnosticsToMisspellings(diagnostics, text) {
    const source = String(text || '');
    const list = [];
    for (const diag of Array.isArray(diagnostics) ? diagnostics : []) {
        if (!diag || !diag.range || !diag.range.start || !diag.range.end) {
            continue;
        }
        const start = lspPositionToEditorOffset(source, diag.range.start.line, diag.range.start.character);
        const end = lspPositionToEditorOffset(source, diag.range.end.line, diag.range.end.character);
        if (end <= start) {
            continue;
        }
        const parsed = parseTyposDiagnosticMessage(diag.message);
        list.push({
            misspeltWord: parsed.word || source.slice(start, end),
            wordStart: start,
            wordLength: end - start,
            suggestions: parsed.suggestions,
        });
    }
    return list;
}

function getLspSpellCheckExclusionSet() {
    return new Set([
        ...lspSpellCheckExclusions.symbols,
        ...lspSpellCheckExclusions.tokens,
        ...lspSpellCheckExclusions.keywords,
    ].map((entry) => String(entry || '').toLowerCase()));
}

// Route an incoming typos-lsp diagnostics payload to the spellcheck overlay of
// whichever editor owns the document (the main editor or a Jupyter code block).
function routeTyposDiagnostics(uri, diagnostics) {
    const targetPath = normalizePathForMatch(fileUriToPath(uri));
    if (!targetPath) {
        return;
    }

    // Main editor.
    if (state.typosOpenFile && notesSpellCheckHandle
        && normalizePathForMatch(fileUriToPath(state.currentFileUri)) === targetPath) {
        notesSpellCheckHandle.setMisspellings(
            typosDiagnosticsToMisspellings(diagnostics, getMainEditorValue()));
        return;
    }

    // Jupyter code blocks.
    for (const blockId of Object.keys(state.jupyterCodeBlocks || {})) {
        const block = state.jupyterCodeBlocks[blockId];
        if (!block || !block.typosFilePath) {
            continue;
        }
        if (normalizePathForMatch(block.typosFilePath) !== targetPath) {
            continue;
        }
        const handle = jupyterSpellCheckHandles[blockId];
        if (!handle) {
            return;
        }
        const editable = elements.jupyter?.querySelector(`[data-block-id="${blockId}"] .jupyter-code-editable`);
        const text = editable ? (editable.value || '') : (block.currentContent || '');
        handle.setMisspellings(typosDiagnosticsToMisspellings(diagnostics, text));
        return;
    }
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

async function requestLspInlayHints() {
    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
        state.lspInlayRequestId += 1;
        state.lspInlayHints = [];
        renderLspEditorDecorations();
        return;
    }

    const requestId = state.lspInlayRequestId + 1;
    state.lspInlayRequestId = requestId;

    try {
        const hints = await NotesLspInlayHints(state.currentFile);
        if (requestId !== state.lspInlayRequestId || state.lspOpenFile !== state.currentFile) {
            return;
        }

        state.lspInlayHints = Array.isArray(hints)
            ? hints.filter((item) => item && String(item.label || '') !== '')
            : [];
        renderLspEditorDecorations();
    } catch {
        if (requestId !== state.lspInlayRequestId) {
            return;
        }

        state.lspInlayHints = [];
        renderLspEditorDecorations();
    }
}

async function requestLspDefinition() {
    if (!state.currentFile || !isCurrentFileLspEligible()) {
        return;
    }

    const selection = getMainEditorSelectionRange();
    const pos = offsetToLspPosition(getMainEditorValue(), selection.start || 0);

    try {
        const locations = await NotesLspDefinition(state.currentFile, pos.line, pos.character);
        const target = Array.isArray(locations) && locations.length > 0 ? locations[0] : null;
        if (!target) {
            notifyTerminal('Definition not found', 'info');
            return;
        }

        const targetAbsPath = normalizePathForMatch(target.filePath || fileUriToPath(target.uri));
        if (!targetAbsPath) {
            notifyTerminal('Definition target is not a local file', 'warn');
            return;
        }

        const targetNotesFile = await resolveNotesFileFromAbsolutePath(targetAbsPath);
        if (!targetNotesFile) {
            notifyTerminal('Definition target is outside indexed Notes files', 'warn');
            return;
        }

        await loadFile(targetNotesFile);

        const targetLine = Math.max(0, Number(target.line) || 0);
        const targetChar = Math.max(0, Number(target.character) || 0);
        const offset = lspPositionToEditorOffset(getMainEditorValue(), targetLine, targetChar);
        setMainEditorSelectionRange(offset, offset);
        if (isMonacoActive()) {
            monacoMainEditor.focus();
            monacoMainEditor.revealOffset(offset);
        } else {
            elements.editor.focus();
            const lineHeight = parseFloat(getComputedStyle(elements.editor).lineHeight) || 18;
            elements.editor.scrollTop = Math.max(0, (targetLine - 2) * lineHeight);
            syncEditorScrollDecorations();
        }

        state.lspHoverLastKey = '';
        scheduleLspHover();
    } catch {
        notifyTerminal('Failed to resolve definition', 'error');
    }
}

async function formatCurrentLspDocument(options = {}) {
    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
        return;
    }

    const preferSelection = options.preferSelection !== false;
    const notifyOnError = options.notifyOnError !== false;

    try {
        const selection = getMainEditorSelectionRange();
        const selectionStart = Number(selection.start) || 0;
        const selectionEnd = Number(selection.end) || selectionStart;
        const currentText = getMainEditorValue();

        let result;
        if (preferSelection && selectionEnd > selectionStart) {
            const startPos = offsetToLspPosition(currentText, selectionStart);
            const endPos = offsetToLspPosition(currentText, selectionEnd);

            result = await NotesLspFormatRange(
                state.currentFile,
                startPos.line,
                startPos.character,
                endPos.line,
                endPos.character,
            );
        } else {
            result = await NotesLspFormat(state.currentFile);
        }

        if (!result || !result.changed) {
            return;
        }

        const nextContent = String(result.content || '');
        if (nextContent === String(currentText || '')) {
            return;
        }

        const prevContent = String(currentText || '');
        const prevLen = prevContent.length;
        const nextLen = nextContent.length;
        const ratio = prevLen > 0 ? nextLen / prevLen : 1;
        const nextSelectionStart = Math.min(Math.round(selectionStart * ratio), nextLen);
        const nextSelectionEnd = Math.min(Math.round(selectionEnd * ratio), nextLen);

        if (isMonacoActive()) {
            setMainEditorValue(nextContent);
            setMainEditorSelectionRange(nextSelectionStart, nextSelectionEnd);
        } else {
            notesMutationAdapter.replaceDocumentText(elements.editor, {
                text: nextContent,
                selectionStart: nextSelectionStart,
                selectionEnd: nextSelectionEnd,
                source: preferSelection && selectionEnd > selectionStart ? 'lsp-format-range' : 'lsp-format',
                label: preferSelection && selectionEnd > selectionStart ? 'Format selected range' : 'Format document',
                emit: true,
            });
        }
    } catch {
        if (notifyOnError) {
            notifyTerminal('Failed to format document', 'error');
        }
    }
}

function symbolDisplayLabel(item) {
    const name = String(item?.name || '').trim();
    const detail = String(item?.detail || '').trim();
    const container = String(item?.containerName || '').trim();

    let label = name;
    if (detail) {
        label += ` - ${detail}`;
    }
    if (container) {
        label += ` (${container})`;
    }

    return label;
}

function workspaceSymbolDisplayLabel(item) {
    const base = symbolDisplayLabel(item);
    const filePath = normalizePathForMatch(item?.filePath || fileUriToPath(item?.uri || ''));
    if (!filePath) {
        return base;
    }

    const shortPath = filePath.split('/').slice(-2).join('/');
    return `${base} - ${shortPath}`;
}

function codeLensDisplayLabel(item) {
    const title = String(item?.title || '').trim() || 'Code lens';
    const line = Math.max(0, Number(item?.line) || 0);
    return `${title} (line ${line + 1})`;
}

async function goToCurrentLspSymbol() {
    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
        return;
    }

    let symbols = [];
    try {
        symbols = await NotesLspDocumentSymbols(state.currentFile);
    } catch {
        notifyTerminal('Failed to fetch document symbols', 'error');
        return;
    }

    const entries = Array.isArray(symbols)
        ? symbols.filter((item) => item && String(item.name || '').trim() !== '')
        : [];

    if (entries.length === 0) {
        notifyTerminal('No symbols found', 'info');
        return;
    }

    const options = entries.map((item) => symbolDisplayLabel(item));
    const icons = options.map(() => CONTEXT_ICON_CODE);
    showLocalMenu({
        title: 'Go to symbol',
        options,
        icons,
        x: window.innerWidth / 2,
        y: window.innerHeight / 2,
        showNextToMouseCursor: true,
        onSelect: (index) => {
            const picked = entries[index];
            if (!picked) {
                return;
            }

            const line = Math.max(0, Number(picked.line) || 0);
            const character = Math.max(0, Number(picked.character) || 0);
            const offset = lspPositionToEditorOffset(elements.editor.value || '', line, character);

            elements.editor.focus();
            elements.editor.setSelectionRange(offset, offset);

            const lineHeight = parseFloat(getComputedStyle(elements.editor).lineHeight) || 18;
            elements.editor.scrollTop = Math.max(0, (line - 2) * lineHeight);
            syncEditorScrollDecorations();

            state.lspHoverLastKey = '';
            scheduleLspHover();
        },
    });
}

async function goToWorkspaceLspSymbol() {
    if (!state.currentFile || !isCurrentFileLspEligible()) {
        return;
    }

    if (state.lspOpenFile !== state.currentFile) {
        await openCurrentLspDocument(elements.editor.value || '');
    }

    if (state.lspOpenFile !== state.currentFile) {
        notifyTerminal('Language server is not active for the current file', 'warn');
        return;
    }

    let symbols = [];
    try {
        symbols = await NotesLspWorkspaceSymbols(state.currentFile, '');
    } catch {
        notifyTerminal('Failed to fetch workspace symbols', 'error');
        return;
    }

    const entries = Array.isArray(symbols)
        ? symbols.filter((item) => item && String(item.name || '').trim() !== '')
        : [];
    if (entries.length === 0) {
        notifyTerminal('No workspace symbols found', 'info');
        return;
    }

    const options = entries.map((item) => workspaceSymbolDisplayLabel(item));
    const icons = options.map(() => CONTEXT_ICON_CODE);
    showLocalMenu({
        title: 'Go to workspace symbol',
        options,
        icons,
        x: window.innerWidth / 2,
        y: window.innerHeight / 2,
        showSearch: true,
        hideItemsUntilQuery: true,
        showNextToMouseCursor: true,
        onSelect: async (index) => {
            try {
                const picked = entries[index];
                if (!picked) {
                    return;
                }

                const targetAbsPath = normalizePathForMatch(picked.filePath || fileUriToPath(picked.uri));
                if (!targetAbsPath) {
                    notifyTerminal('Workspace symbol target is not a local file', 'warn');
                    return;
                }

                const targetNotesFile = await resolveNotesFileFromAbsolutePath(targetAbsPath);
                if (!targetNotesFile) {
                    notifyTerminal('Workspace symbol target is outside indexed Notes files', 'warn');
                    return;
                }

                await loadFile(targetNotesFile);

                const line = Math.max(0, Number(picked.line) || 0);
                const character = Math.max(0, Number(picked.character) || 0);
                const offset = lspPositionToEditorOffset(elements.editor.value || '', line, character);
                elements.editor.focus();
                elements.editor.setSelectionRange(offset, offset);

                const lineHeight = parseFloat(getComputedStyle(elements.editor).lineHeight) || 18;
                elements.editor.scrollTop = Math.max(0, (line - 2) * lineHeight);
                syncEditorScrollDecorations();

                state.lspHoverLastKey = '';
                scheduleLspHover();
            } catch (err) {
                console.error('Workspace symbol navigation failed:', err);
                notifyTerminal('Failed to navigate to workspace symbol', 'error');
            }
        },
    });
}

async function runCurrentLspCodeLens() {
    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
        return;
    }

    let lenses = [];
    try {
        lenses = await NotesLspCodeLens(state.currentFile);
    } catch {
        notifyTerminal('Failed to fetch code lens actions', 'error');
        return;
    }

    const entries = Array.isArray(lenses)
        ? lenses.filter((item) => item && Number.isFinite(Number(item.index)))
        : [];
    if (entries.length === 0) {
        notifyTerminal('No code lens actions found', 'info');
        return;
    }

    const options = entries.map((item) => codeLensDisplayLabel(item));
    const icons = options.map(() => CONTEXT_ICON_CODE);
    showLocalMenu({
        title: 'Code lens',
        options,
        icons,
        x: window.innerWidth / 2,
        y: window.innerHeight / 2,
        showNextToMouseCursor: true,
        onSelect: async (index) => {
            const picked = entries[index];
            if (!picked) {
                return;
            }

            try {
                const applied = await NotesLspExecuteCodeLens(state.currentFile, Number(picked.index) || 0);
                if (!applied) {
                    notifyTerminal('Code lens had no executable command', 'info');
                }
            } catch {
                notifyTerminal('Failed to run code lens', 'error');
            }
        },
    });
}

function isPositionWithinRange(line, character, range) {
    if (!range || !range.start || !range.end) {
        return false;
    }

    const startLine = Number(range.start.line) || 0;
    const startChar = Number(range.start.character) || 0;
    const endLine = Number(range.end.line) || startLine;
    const endChar = Number(range.end.character) || startChar;

    if (line < startLine || line > endLine) {
        return false;
    }
    if (line === startLine && character < startChar) {
        return false;
    }
    if (line === endLine && character > endChar) {
        return false;
    }

    return true;
}

function lspDiagnosticsAtPosition(line, character) {
    const uri = state.currentFileUri;
    if (!uri) {
        return [];
    }

    const diagnostics = lspDiagnosticsStore.get(uri) || [];
    return diagnostics.filter((diag) => isPositionWithinRange(line, character, diag.range));
}

function formatLspDiagnosticsForHover(diagnostics) {
    const list = Array.isArray(diagnostics) ? diagnostics : [];
    if (list.length === 0) {
        return '';
    }

    const severityLabel = (value) => {
        switch (Number(value) || 0) {
        case 1: return 'Error';
        case 2: return 'Warning';
        case 3: return 'Info';
        case 4: return 'Hint';
        default: return 'Info';
        }
    };

    const lines = ['Diagnostics:'];
    for (const diag of list) {
        const message = String(diag?.message || '').trim();
        if (!message) {
            continue;
        }
        const severity = severityLabel(diag?.severity);
        const source = String(diag?.source || '').trim();
        const prefix = source ? `${severity} (${source})` : severity;
        lines.push(`- ${prefix}: ${message}`);
    }

    return lines.length > 1 ? lines.join('\n') : '';
}

async function getLspCodeActionsForCursor() {
    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
        return { line: 0, character: 0, diagnostics: [], actions: [] };
    }

    const selection = getMainEditorSelectionRange();
    const pos = offsetToLspPosition(getMainEditorValue(), selection.start || 0);
    const diagnostics = lspDiagnosticsAtPosition(pos.line, pos.character);

    try {
        const actions = await NotesLspCodeActions(state.currentFile, pos.line, pos.character, diagnostics);
        return {
            line: pos.line,
            character: pos.character,
            diagnostics,
            actions: Array.isArray(actions) ? actions : [],
        };
    } catch {
        return {
            line: pos.line,
            character: pos.character,
            diagnostics,
            actions: [],
        };
    }
}

function lspCodeActionMenuSection(kind) {
    const value = String(kind || '').trim().toLowerCase();
    if (value === 'quickfix' || value.startsWith('quickfix.')) {
        return { key: 'quickfix', label: 'Quick fix', rank: 0 };
    }
    if (value === 'refactor' || value.startsWith('refactor.')) {
        return { key: 'refactor', label: 'Refactor', rank: 1 };
    }
    if (value === 'source' || value.startsWith('source.')) {
        return { key: 'source', label: 'Source action', rank: 2 };
    }

    return { key: 'other', label: 'Code action', rank: 3 };
}

function buildLspCodeActionMenuItems(actions, line, character, diagnostics) {
    const grouped = new Map();

    (Array.isArray(actions) ? actions : []).forEach((action, index) => {
        const title = String(action?.title || '').trim();
        if (!title) {
            return;
        }

        const section = lspCodeActionMenuSection(action?.kind);
        const entry = {
            title: `${section.label}: ${title}`,
            icon: CONTEXT_ICON_CODE,
            rank: section.rank,
            onSelect: () => {
                void applyLspCodeActionFromCursor(index, line, character, diagnostics);
            },
        };

        if (!grouped.has(section.key)) {
            grouped.set(section.key, []);
        }
        grouped.get(section.key).push(entry);
    });

    const sections = Array.from(grouped.values())
        .filter((items) => items.length > 0)
        .sort((left, right) => (left[0]?.rank || 0) - (right[0]?.rank || 0));

    const menuItems = [];
    sections.forEach((items, index) => {
        if (index > 0) {
            menuItems.push({ title: '-' });
        }
        menuItems.push(...items);
    });

    return menuItems;
}

async function showEditorLspOptionsMenu(x, y) {
    if (!isCurrentFileLspEligible()) {
        return;
    }

    const menuItems = [
        {
            title: 'Format document',
            icon: CONTEXT_ICON_CODE,
            onSelect: () => {
                void formatCurrentLspDocument();
            },
        },
        {
            title: 'Go to symbol...',
            icon: CONTEXT_ICON_CODE,
            onSelect: () => {
                void goToCurrentLspSymbol();
            },
        },
        {
            title: 'Go to workspace symbol...',
            icon: CONTEXT_ICON_CODE,
            onSelect: () => {
                void goToWorkspaceLspSymbol();
            },
        },
        {
            title: 'Signature help',
            icon: CONTEXT_ICON_CODE,
            onSelect: () => {
                void requestLspSignatureHelpFromCursor(1, '');
            },
        },
        {
            title: 'Code lens...',
            icon: CONTEXT_ICON_CODE,
            onSelect: () => {
                void runCurrentLspCodeLens();
            },
        },
        {
            title: 'Rename symbol...',
            icon: CONTEXT_ICON_CODE,
            onSelect: () => {
                void renameCurrentLspSymbol();
            },
        },
    ];

    const codeActionData = await getLspCodeActionsForCursor();
    if (codeActionData.actions.length > 0) {
        menuItems.push({ title: '-' });
        menuItems.push(
            ...buildLspCodeActionMenuItems(
                codeActionData.actions,
                codeActionData.line,
                codeActionData.character,
                codeActionData.diagnostics,
            ),
        );
    }

    showNotesLocalMenu(menuItems, x, y, 'LSP options');
}

async function applyLspCodeActionFromCursor(index, line, character, diagnostics) {
    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
        return;
    }

    try {
        const result = await NotesLspApplyCodeAction(state.currentFile, line, character, index, diagnostics || []);
        if (!result || !result.changed) {
            return;
        }

        const nextContent = String(result.content || '');
        const currentText = getMainEditorValue();
        if (nextContent === String(currentText || '')) {
            return;
        }

        const selection = getMainEditorSelectionRange();
        const selectionStart = Number(selection.start) || 0;
        const selectionEnd = Number(selection.end) || selectionStart;
        const nextLen = nextContent.length;
        if (isMonacoActive()) {
            setMainEditorValue(nextContent);
            setMainEditorSelectionRange(Math.min(selectionStart, nextLen), Math.min(selectionEnd, nextLen));
        } else {
            notesMutationAdapter.replaceDocumentText(elements.editor, {
                text: nextContent,
                selectionStart: Math.min(selectionStart, nextLen),
                selectionEnd: Math.min(selectionEnd, nextLen),
                source: 'lsp-code-action',
                label: 'Apply code action',
                emit: true,
            });
        }
    } catch {
        notifyTerminal('Failed to apply code action', 'error');
    }
}

async function renameCurrentLspSymbol() {
    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
        return;
    }

    const selection = getMainEditorSelectionRange();
    const selectionStart = Number(selection.start) || 0;
    const selectionEnd = Number(selection.end) || selectionStart;
    const currentSelection = selectionEnd > selectionStart
        ? String(getMainEditorValue() || '').slice(selectionStart, selectionEnd)
        : '';

    const pos = offsetToLspPosition(getMainEditorValue(), selectionStart);

    let prepare;
    try {
        prepare = await NotesLspPrepareRename(state.currentFile, pos.line, pos.character);
    } catch {
        notifyTerminal('Failed to prepare symbol rename', 'error');
        return;
    }

    if (!prepare || !prepare.canRename) {
        notifyTerminal('Rename not available at this cursor position', 'info');
        return;
    }

    const suggested = String(prepare.placeholder || currentSelection || '').trim();
    const nextName = window.prompt('Rename symbol to:', suggested);
    if (nextName === null) {
        return;
    }

    const trimmedName = String(nextName).trim();
    if (!trimmedName) {
        notifyTerminal('Rename cancelled: new name is empty', 'warn');
        return;
    }

    try {
        const result = await NotesLspRename(state.currentFile, pos.line, pos.character, trimmedName);
        if (!result || !result.changed) {
            notifyTerminal('No rename edits applied', 'info');
            return;
        }

        const nextContent = String(result.content || '');
        if (nextContent === String(getMainEditorValue() || '')) {
            return;
        }

        const nextLen = nextContent.length;
        if (isMonacoActive()) {
            setMainEditorValue(nextContent);
            setMainEditorSelectionRange(Math.min(selectionStart, nextLen), Math.min(selectionEnd, nextLen));
        } else {
            notesMutationAdapter.replaceDocumentText(elements.editor, {
                text: nextContent,
                selectionStart: Math.min(selectionStart, nextLen),
                selectionEnd: Math.min(selectionEnd, nextLen),
                source: 'lsp-rename',
                label: 'Rename symbol',
                emit: true,
            });
        }
    } catch {
        notifyTerminal('Failed to rename symbol', 'error');
    }
}

function offsetToLspPosition(content, offset) {
    const source = String(content || '');
    const clampedOffset = Math.max(0, Math.min(Number(offset) || 0, source.length));
    const before = source.slice(0, clampedOffset);
    const lines = before.split('\n');
    return {
        line: Math.max(0, lines.length - 1),
        character: lines[lines.length - 1]?.length || 0,
    };
}

function getEditorCaretViewportPoint() {
    const editor = elements.editor;
    if (!editor || typeof editor.selectionStart !== 'number') {
        return null;
    }

    const editorRect = editor.getBoundingClientRect();
    if (!editorRect || editorRect.width <= 0 || editorRect.height <= 0) {
        return null;
    }

    const computed = window.getComputedStyle(editor);
    const mirror = document.createElement('div');
    const marker = document.createElement('span');
    const source = String(editor.value || '');
    const caretOffset = Math.max(0, Math.min(Number(editor.selectionStart) || 0, source.length));
    const beforeCaret = source.slice(0, caretOffset);
    const afterCaret = source.slice(caretOffset);
    const isWrapModeEnabled = elements.editorShell?.dataset?.wrapMode === 'true';

    mirror.style.position = 'absolute';
    mirror.style.left = '-10000px';
    mirror.style.top = '-10000px';
    mirror.style.visibility = 'hidden';
    mirror.style.pointerEvents = 'none';
    mirror.style.whiteSpace = isWrapModeEnabled ? 'pre-wrap' : 'pre';
    mirror.style.wordBreak = isWrapModeEnabled ? 'break-word' : 'normal';
    mirror.style.overflowWrap = isWrapModeEnabled ? 'break-word' : 'normal';
    mirror.style.wordWrap = isWrapModeEnabled ? 'break-word' : 'normal';
    mirror.style.tabSize = computed.tabSize;
    mirror.style.fontFamily = computed.fontFamily;
    mirror.style.fontSize = computed.fontSize;
    mirror.style.fontWeight = computed.fontWeight;
    mirror.style.fontStyle = computed.fontStyle;
    mirror.style.fontVariant = computed.fontVariant;
    mirror.style.letterSpacing = computed.letterSpacing;
    mirror.style.lineHeight = computed.lineHeight;
    mirror.style.textTransform = computed.textTransform;
    mirror.style.textIndent = computed.textIndent;
    mirror.style.textRendering = computed.textRendering;
    mirror.style.paddingTop = computed.paddingTop;
    mirror.style.paddingRight = computed.paddingRight;
    mirror.style.paddingBottom = computed.paddingBottom;
    mirror.style.paddingLeft = computed.paddingLeft;
    mirror.style.borderTopWidth = computed.borderTopWidth;
    mirror.style.borderRightWidth = computed.borderRightWidth;
    mirror.style.borderBottomWidth = computed.borderBottomWidth;
    mirror.style.borderLeftWidth = computed.borderLeftWidth;
    mirror.style.borderStyle = computed.borderStyle;
    mirror.style.boxSizing = computed.boxSizing;
    mirror.style.width = `${editor.clientWidth}px`;

    mirror.textContent = beforeCaret;
    marker.textContent = afterCaret.length > 0 ? afterCaret[0] : ' ';
    mirror.appendChild(marker);
    document.body.appendChild(mirror);

    const markerOffsetLeft = marker.offsetLeft;
    const markerOffsetTop = marker.offsetTop;
    mirror.remove();

    if (!Number.isFinite(markerOffsetLeft) || !Number.isFinite(markerOffsetTop)) {
        return null;
    }

    return {
        x: editorRect.left + markerOffsetLeft - editor.scrollLeft,
        y: editorRect.top + markerOffsetTop - editor.scrollTop,
    };
}

function getLspAnchorViewportPoint() {
    const caretPoint = getEditorCaretViewportPoint();
    if (caretPoint) {
        return caretPoint;
    }

    if (Number.isFinite(state.lspHoverMouseX) && Number.isFinite(state.lspHoverMouseY)
        && (state.lspHoverMouseX !== 0 || state.lspHoverMouseY !== 0)) {
        return {
            x: state.lspHoverMouseX,
            y: state.lspHoverMouseY,
        };
    }

    const fallbackRect = elements.editor?.getBoundingClientRect();
    if (fallbackRect) {
        return {
            x: fallbackRect.left + 20,
            y: fallbackRect.top + 20,
        };
    }

    return { x: 20, y: 20 };
}

function scheduleLspHover() {
    if (state.lspCompletionVisible) {
        clearLspHoverTimer();
        hideLspHoverTooltip();
        return;
    }

    const blockTarget = getActiveLspTarget();
    if (!blockTarget && (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible())) {
        hideLspHoverTooltip();
        return;
    }

    const hoverFilePath = blockTarget ? blockTarget.filePath : state.currentFile;
    const hoverEditor = blockTarget ? blockTarget.editor : elements.editor;

    clearLspHoverTimer();
    state.lspHoverTimer = setTimeout(async () => {
        state.lspHoverTimer = null;

        const pos = offsetToLspPosition(hoverEditor.value || '', hoverEditor.selectionStart || 0);
        const key = `${hoverFilePath}:${pos.line}:${pos.character}`;
        if (key === state.lspHoverLastKey) {
            return;
        }
        state.lspHoverLastKey = key;

        try {
            const hoverText = await NotesLspHover(hoverFilePath, pos.line, pos.character);
            const diagnosticsText = (!blockTarget && hoverFilePath === state.currentFile)
                ? formatLspDiagnosticsForHover(lspDiagnosticsAtPosition(pos.line, pos.character))
                : '';

            const sections = [];
            if (hoverText) {
                sections.push(String(hoverText));
            }
            if (diagnosticsText) {
                sections.push(diagnosticsText);
            }

            if (sections.length === 0) {
                hideLspHoverTooltip();
                return;
            }

            await renderLspHoverTooltipText(sections.join('\n\n'));
            lspHoverTooltipEl.style.display = 'block';

            const anchor = getLspAnchorViewportPoint();
            const rawX = anchor.x;
            const rawY = anchor.y;
            const x = Math.min(rawX + 14, window.innerWidth - lspHoverTooltipEl.offsetWidth - 8);
            const y = Math.min(rawY + 14, window.innerHeight - lspHoverTooltipEl.offsetHeight - 8);
            lspHoverTooltipEl.style.left = `${Math.max(8, x)}px`;
            lspHoverTooltipEl.style.top = `${Math.max(8, y)}px`;
        } catch {
            hideLspHoverTooltip();
        }
    }, LSP_HOVER_DEBOUNCE_MS);
}

async function renderLspHoverTooltipText(text) {
    lspHoverTooltipEl.innerHTML = marked.parse(escapeHtml(text));
    await processMarkdownContainer(lspHoverTooltipEl);
}

async function requestLspSignatureHelpFromCursor(triggerKind = 1, triggerChar = '') {
    if (state.lspCompletionVisible) {
        hideLspHoverTooltip();
        return;
    }

    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
        return;
    }

    const pos = offsetToLspPosition(elements.editor.value || '', elements.editor.selectionStart || 0);

    try {
        const text = await NotesLspSignatureHelp(state.currentFile, pos.line, pos.character, triggerKind, triggerChar);
        if (!text) {
            notifyTerminal('No signature help available', 'info');
            return;
        }

        await renderLspHoverTooltipText(text);
        lspHoverTooltipEl.style.display = 'block';

        const anchor = getLspAnchorViewportPoint();
        const rawX = anchor.x;
        const rawY = anchor.y;
        const x = Math.min(rawX + 14, window.innerWidth - lspHoverTooltipEl.offsetWidth - 8);
        const y = Math.min(rawY + 14, window.innerHeight - lspHoverTooltipEl.offsetHeight - 8);
        lspHoverTooltipEl.style.left = `${Math.max(8, x)}px`;
        lspHoverTooltipEl.style.top = `${Math.max(8, y)}px`;
    } catch {
        notifyTerminal('Failed to fetch signature help', 'error');
    }
}

async function closeOpenLspDocument() {
    clearLspChangeTimer();
    clearLspHoverTimer();
    hideLspHoverTooltip();
    hideLspCompletion();
    state.lspInlayRequestId += 1;
    state.lspInlayHints = [];

    const openFile = state.lspOpenFile;
    if (!openFile) {
        return;
    }

    try {
        await NotesLspCloseDocument(openFile);
    } catch (err) {
        console.error('notes lsp close failed:', err);
    } finally {
        state.lspOpenFile = '';
        state.lspHoverLastKey = '';
    }
}

async function openCurrentLspDocument(content) {
    if (!state.currentFile || !isCurrentFileLspEligible()) {
        return;
    }

    try {
        const languageID = await ResolveNotesLspLanguage(state.currentFile);
        if (!languageID) {
            return;
        }

        await NotesLspOpenDocument(state.currentFile, languageID, String(content || ''));
        state.lspOpenFile = state.currentFile;
        lspSpellCheckExclusions.keywords = (await GetNotesLanguageReservedWords(languageID)) || [];
        lspSpellCheckExclusions.tokens = [];
        applyLspSpellCheckExclusions();

        if (isMonacoActive()) {
            monacoMainEditor.configureLsp({
                completion: async ({ line, character }) => {
                    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
                        return [];
                    }
                    try {
                        await NotesLspChangeDocument(state.currentFile, getMainEditorValue());
                    } catch {
                        // Completion can still work with last synced state.
                    }
                    const items = await NotesLspCompletion(state.currentFile, line, character, 1, '');
                    return Array.isArray(items) ? items : [];
                },
                signature: async ({ line, character }) => {
                    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
                        return '';
                    }
                    return await NotesLspSignatureHelp(state.currentFile, line, character, 1, '');
                },
                formatDocument: async () => {
                    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
                        return null;
                    }
                    return await NotesLspFormat(state.currentFile);
                },
                formatRange: async ({ start, end }) => {
                    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
                        return null;
                    }
                    return await NotesLspFormatRange(
                        state.currentFile,
                        Number(start?.line) || 0,
                        Number(start?.character) || 0,
                        Number(end?.line) || 0,
                        Number(end?.character) || 0,
                    );
                },
                definition: async ({ line, character }) => {
                    if (!state.currentFile || !isCurrentFileLspEligible()) {
                        return [];
                    }

                    const locations = await NotesLspDefinition(state.currentFile, line, character);
                    return Array.isArray(locations) ? locations : [];
                },
                prepareRename: async ({ line, character }) => {
                    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
                        return { canRename: false };
                    }
                    return await NotesLspPrepareRename(state.currentFile, line, character);
                },
                rename: async ({ line, character, newName }) => {
                    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
                        return null;
                    }
                    return await NotesLspRename(state.currentFile, line, character, String(newName || ''));
                },
                codeActions: async ({ line, character }) => {
                    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
                        return [];
                    }
                    const diagnostics = lspDiagnosticsAtPosition(line, character);
                    const actions = await NotesLspCodeActions(state.currentFile, line, character, diagnostics);
                    return Array.isArray(actions) ? actions : [];
                },
                applyCodeAction: async (action) => {
                    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
                        return null;
                    }

                    const selection = getMainEditorSelectionRange();
                    const pos = offsetToLspPosition(getMainEditorValue(), selection.start || 0);
                    const diagnostics = lspDiagnosticsAtPosition(pos.line, pos.character);
                    const actionIndex = Number(action?.index);
                    if (!Number.isFinite(actionIndex)) {
                        return null;
                    }
                    return await NotesLspApplyCodeAction(
                        state.currentFile,
                        pos.line,
                        pos.character,
                        actionIndex,
                        diagnostics,
                    );
                },
            });
        }

        await requestLspInlayHints();
        void updateSpellCheckExclusionsFromDocSymbols();
    } catch (err) {
        console.error('notes lsp open failed:', err);
    }
}

/**
 * Fetch document symbols from the LSP server and use their names as
 * spell-check exclusions. Document symbols are available immediately after
 * didOpen, making them
 * a reliable exclusion source on file open.
 */
async function updateSpellCheckExclusionsFromDocSymbols() {
    const fileAtCall = state.currentFile;
    if (!notesSpellCheckHandle || !fileAtCall || state.lspOpenFile !== fileAtCall || !isCurrentFileLspEligible()) {
        return;
    }

    function extractNames(symbols) {
        const names = [];
        function walk(syms) {
            if (!Array.isArray(syms)) return;
            for (const s of syms) {
                if (s && s.name) names.push(String(s.name));
                if (s && s.children) walk(s.children);
            }
        }
        walk(symbols);
        return names;
    }

    // First attempt — may return empty if gopls hasn't analysed the file yet.
    try {
        const symbols = await NotesLspDocumentSymbols(fileAtCall);
        if (state.currentFile !== fileAtCall) return;
        const names = extractNames(symbols);
        if (names.length > 0) {
            lspSpellCheckExclusions.symbols = names;
            applyLspSpellCheckExclusions();
            return;
        }
    } catch {
        // fall through to retry
    }

    // Retry after a short delay to give gopls time to finish analysing.
    await new Promise(resolve => setTimeout(resolve, 2000));
    if (state.currentFile !== fileAtCall || state.lspOpenFile !== fileAtCall) return;

    try {
        const symbols = await NotesLspDocumentSymbols(fileAtCall);
        if (state.currentFile !== fileAtCall) return;
        const names = extractNames(symbols);
        if (names.length > 0) {
            lspSpellCheckExclusions.symbols = names;
            applyLspSpellCheckExclusions();
        }
    } catch {
        // LSP not available or request failed — degrade silently.
    }
}

function scheduleLspDidChange() {
    if (!state.currentFile || state.lspOpenFile !== state.currentFile || !isCurrentFileLspEligible()) {
        return;
    }

    clearLspChangeTimer();
    state.lspChangeTimer = setTimeout(async () => {
        state.lspChangeTimer = null;
        try {
            await NotesLspChangeDocument(state.currentFile, elements.editor.value || '');
            await requestLspInlayHints();
        } catch (err) {
            console.error('notes lsp change failed:', err);
        }
    }, LSP_CHANGE_DEBOUNCE_MS);
}

// ── typos-lsp spellchecking (main editor) ──────────────────────────────────
//
// When typos-lsp is available, eligible project files are spellchecked by it
// instead of aspell, rendering with the same red wavy-underline chrome. aspell
// remains the fallback (and is always used for form fields like the input box).

async function openCurrentTyposDocument(content) {
    if (!notesSpellCheckHandle) {
        return;
    }
    if (!state.currentFile || !isCurrentFileLspEligible()) {
        // Non-eligible files keep aspell.
        notesSpellCheckHandle.setMode('aspell');
        if (isMonacoActive()) {
            monacoMainEditor.setTyposMisspellings([]);
        }
        return;
    }

    try {
        const languageID = await ResolveNotesLspLanguage(state.currentFile);
        const ok = await NotesTyposOpenDocument(state.currentFile, languageID || '', String(content || ''));
        if (ok) {
            state.typosOpenFile = state.currentFile;
            notesSpellCheckHandle.setMode('external');
            const diagnostics = state.currentFileUri ? (typosDiagnosticsStore.get(state.currentFileUri) || []) : [];
            notesSpellCheckHandle.setMisspellings(typosDiagnosticsToMisspellings(diagnostics, getMainEditorValue()));
        } else {
            state.typosOpenFile = '';
            notesSpellCheckHandle.setMode('aspell');
            if (isMonacoActive()) {
                monacoMainEditor.setTyposMisspellings([]);
            }
        }
    } catch (err) {
        state.typosOpenFile = '';
        notesSpellCheckHandle.setMode('aspell');
        if (isMonacoActive()) {
            monacoMainEditor.setTyposMisspellings([]);
        }
        console.error('notes typos open failed:', err);
    }
}

async function closeCurrentTyposDocument() {
    if (state.typosChangeTimer) {
        clearTimeout(state.typosChangeTimer);
        state.typosChangeTimer = null;
    }
    const openFile = state.typosOpenFile;
    state.typosOpenFile = '';
    notesSpellCheckHandle?.setMode('aspell');
    if (isMonacoActive()) {
        monacoMainEditor.setTyposMisspellings([]);
    }
    if (!openFile) {
        return;
    }
    try {
        await NotesTyposCloseDocument(openFile);
    } catch (err) {
        console.error('notes typos close failed:', err);
    }
}

function scheduleTyposDidChange() {
    if (!state.typosOpenFile || state.typosOpenFile !== state.currentFile) {
        return;
    }
    if (state.typosChangeTimer) {
        clearTimeout(state.typosChangeTimer);
    }
    state.typosChangeTimer = setTimeout(async () => {
        state.typosChangeTimer = null;
        try {
            await NotesTyposChangeDocument(state.currentFile, elements.editor.value || '');
        } catch (err) {
            console.error('notes typos change failed:', err);
        }
    }, LSP_CHANGE_DEBOUNCE_MS);
}

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

    const content = elements.editor.value || '';

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
            if (applyFormattedText(elements.editor, formatted, {
                source: 'format-command',
                label: 'Format main editor (command)',
            })) {
                elements.editor.dispatchEvent(new Event('input', { bubbles: true }));
            }
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
    const label = state.currentFile ? state.currentFile : 'No file selected';
    elements.status.textContent = isDirty ? `${label} (unsaved)` : label;
}

function focusActiveEditorForViewMode() {
    if (!elements.editor && !isMonacoActive()) {
        return;
    }

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

        if (isMonacoActive()) {
            monacoMainEditor.focus();
            return;
        }

        if (!elements.editor) {
            return;
        }

        elements.editor.focus({ preventScroll: true });
    }, 0);
}

function emitCurrentFileName() {
    const fileName = state.currentFile ? getPathFileName(state.currentFile) : '';
    app.dataset.currentFileName = fileName;
    window.dispatchEvent(new CustomEvent('notes-current-file', {
        detail: { fileName }
    }));
}

function setViewMode(mode) {
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
        state.viewMode = 'editor';
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
    
    const isStructuredEdit = state.currentFileType === 'json' && state.viewMode === 'swagger-edit';
    elements.editorWrap.dataset.active = (isEditor || isStructuredEdit) ? 'true' : 'false';
    elements.hexWrap.dataset.active = isHex ? 'true' : 'false';
    elements.previewWrap.dataset.active = isViewer ? 'true' : 'false';
    elements.htmlViewWrap.dataset.active = isHtmlView ? 'true' : 'false';
    elements.jupyterWrap.dataset.active = isJupyter ? 'true' : 'false';
    elements.metaWrap.dataset.active = isMeta ? 'true' : 'false';
    
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
            renderCsvView(elements.editor.value, { interactive: isCsvRun });
        }
    }

    // The main editor wrap is shared by code/markdown/json edit and csv edit.
    // Whenever it becomes visible, (lazily) create Monaco and lay it out against
    // its now-visible box. Fall back to legacy decorations only when Monaco is
    // disabled.
    const editorWrapActive = elements.editorWrap.dataset.active === 'true';
    if (state.useMonacoEditor && editorWrapActive) {
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
        monacoMainEditor.setWordWrap(wrapEnabled);
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

async function renderJupyterView() {
    // Reset jupyter state for the new render
    state.jupyterCodeBlocks = {};
    state.jupyterBlockCounter = 0;
    
    const markdown = applyDocumentFrontmatter();
    elements.jupyter.innerHTML = marked.parse(markdown);
    
    // Apply common markdown processing
    await processMarkdownContainer(elements.jupyter);

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
            setupInteractiveMarkdownTables(elements.jupyter, true);

            // Keep table horizontal scrolling local to each table rather than the whole section.
            wrapTablesForHorizontalScroll(elements.jupyter);

            // Enable column sorting on all tables
            setupTableSorting(elements.jupyter);

            elements.jupyter.classList.toggle('notes-table-wordwrap-on', state.markdownTableWordWrapMode);
            void setupTableColumnResizing(elements.jupyter, state.markdownTableWordWrapMode, state.currentFile);

            // Re-apply find highlights when Find tab is active in jupyter mode.
            if (elements.toolsTabFind?.getAttribute('aria-selected') === 'true' && state.findQuery && state.viewMode === 'jupyter') {
                setTimeout(() => {
                    performFind();
                }, 0);
            }
        });
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
    
    try {
        await RunNote(state.currentFile, blockId, block.currentContent, block.runtime);
    } catch (err) {
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
    try {
        await StopNote(blockId);
    } catch (err) {
        console.error('Error stopping code:', err);
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
            await SendToTerminal(block.currentContent);
            window.dispatchEvent(new CustomEvent('ttyphoon-focus-terminal'));
        } catch (err) {
            console.error('Error sending to terminal:', err);
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

async function refreshFiles(options = {}) {
    try {
        const skipHistoryRestore = Boolean(options?.skipHistoryRestore);
        const previousWorkspaceKey = String(state.currentWorkspaceKey || '');
        const files = await ListFiles();
        state.files = Array.isArray(files) ? files : [];
        state.currentProjectRoot = await GetCurrentProject();
        const workspaceName = await GetCurrentGroupName();
        const workspaceKey = `${String(state.currentProjectRoot || '')}::${String(workspaceName || '')}`;
        const workspaceChanged = previousWorkspaceKey !== '' && previousWorkspaceKey !== workspaceKey;
        const workspaceScopedRefresh = previousWorkspaceKey === '' || workspaceChanged;

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

        await loadProjectCache({
            skipHistoryRestore: skipHistoryRestore || !workspaceScopedRefresh,
        });
        await applyFileFilter();

        if (workspaceChanged) {
            scrollActiveFileListItemIntoView();
        }
    } catch (err) {
        notifyTerminal('Failed to load file list', 'error');
        console.error(err);
    }
}

async function loadProjectCache(options = {}) {
    try {
        const skipHistoryRestore = Boolean(options?.skipHistoryRestore);
        const cache = await GetProjectCache();
        const collapsed = cache?.FileListCollapsed?.[state.currentProjectRoot] || [];
        state.expandedFolders = {};
        for (const key of collapsed) {
            state.expandedFolders[key] = false;
        }
        if (skipHistoryRestore) {
            return;
        }
        const historyDoc = await NotesHistoryCurrent();
        const hasHistoryDoc = historyDoc && state.files.includes(historyDoc);
        if (hasHistoryDoc && historyDoc !== state.lastRestoredDocument) {
            state.lastRestoredDocument = historyDoc;
            await loadFile(historyDoc);
        } else if (!hasHistoryDoc) {
            state.lastRestoredDocument = '';
        }
    } catch (err) {
        console.error('Failed to load project cache:', err);
    }
}

async function restoreDocumentCache(file) {
    const documentCache = await GetDocumentCache(file);
    if (!documentCache) {
        return;
    }

    if (typeof documentCache.WordWrap === 'boolean') {
        setEditorWrapMode(documentCache.WordWrap === true);
    }

    if (documentCache.DocumentTab) {
        setViewMode(documentCache.DocumentTab);

        if (documentCache.DocumentTab === 'jupyter') {
            await renderJupyterView();
        } else if (documentCache.DocumentTab === 'swagger-view') {
            renderSwaggerJsonViewLazy();
        } else if (documentCache.DocumentTab === 'swagger-run') {
            updateSwaggerLayoutMode();
            renderSwaggerUI();
        }
    }

    // The Tools panel open/closed state and selected tab are intentionally not
    // restored from cache. The panel keeps whatever state it already had; if the
    // active tab isn't available for the newly loaded document, the panel is
    // closed (handled by updateToolsTabVisibility).
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
    const query = state.fileFilterQuery.trim();
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
        if (result && result.Error) {
            state.filteredFiles = [];
        } else {
            state.filteredFiles = Array.isArray(result?.List) ? result.List : [];
        }
    } catch {
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
        detail.textContent = `${String(item?.path || '')}:${lineNo}`;

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
                span.textContent = String(ctxLine);
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

function setFolderExpansionState(folderKeys, expanded) {
    folderKeys.forEach((key) => {
        state.expandedFolders[key] = expanded;
    });
    saveProjectCache();
}

function openFolderTreeContextMenu(category, nodes, x, y, title = 'Folder actions') {
    const folderKeys = collectFolderKeys(category, nodes);
    if (folderKeys.length === 0) {
        return;
    }

    showNotesLocalMenu([
        {
            title: 'Collapse Folders',
            icon: 0xf146,
            onSelect: () => {
                setFolderExpansionState(folderKeys, false);
                renderFileList();
            },
        },
        {
            title: 'Expand Folders',
            icon: 0xf0fe,
            onSelect: () => {
                setFolderExpansionState(folderKeys, true);
                renderFileList();
            },
        },
    ], x, y, title);
}

/**
 * Show/hide tabs based on file type
 */
function updateTabVisibility(fileType) {
    updateToolsTabVisibility(fileType);

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
        // Code files use a single Edit tab.
        elements.tabEditor.style.display = '';
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
    if (!elements.swaggerView || !elements.editor) {
        return;
    }

    if (state.swaggerViewTooLarge) {
        renderStructViewTooLargeMessage();
        state.swaggerViewCurrent = true;
        return;
    }

    attachJsonViewerEditHandler(elements.swaggerView, commitStructuredViewerEdit);
    renderJsonViewer(elements.swaggerView, state.swaggerSpec ?? (elements.editor.value || '{}'));
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
    const source = String(elements.editor?.value || '');

    try {
        const parsed = JSON.parse(source);
        const nextText = pretty
            ? JSON.stringify(parsed, null, 2)
            : JSON.stringify(parsed);

        notesMutationAdapter.replaceDocumentText(elements.editor, {
            text: nextText,
            selectionStart: Number(elements.editor.selectionStart) || 0,
            selectionEnd: Number(elements.editor.selectionEnd) || 0,
            source: 'structured-json-format',
            label: pretty ? 'Format JSON (expand)' : 'Format JSON (minify)',
            emit: true,
        });
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
        const source = state.swaggerSpec ?? parseSwaggerSpec(elements.editor.value);
        if (!source || !Array.isArray(path)) {
            return;
        }

        // Prefer a format-specific surgical editor so only the edited key/value
        // is rewritten, preserving comments and formatting (and therefore git
        // diffs). Fall back to a whole-document re-serialise for any format
        // without a registered editor.
        const editor = getStructuredEditor(state.currentFile);
        const originalText = elements.editor.value;
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

        notesMutationAdapter.replaceDocumentText(elements.editor, {
            text: nextText,
            selectionStart: Number(elements.editor.selectionStart) || 0,
            selectionEnd: Number(elements.editor.selectionEnd) || 0,
            source: 'structured-edit',
            label: 'Apply structured edit',
            emit: true,
        });
        state.swaggerSpec = parseSwaggerSpec(elements.editor.value);

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

        const raw = elements.editor?.value || '';
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

        notesMutationAdapter.replaceDocumentText(elements.editor, {
            text: nextRaw,
            selectionStart: Number(elements.editor.selectionStart) || 0,
            selectionEnd: Number(elements.editor.selectionEnd) || 0,
            source: 'frontmatter-edit',
            label: 'Apply frontmatter edit',
            emit: true,
        });
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

    const swaggerInfoEl = document.getElementById('notes-swagger-info');
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
    
    // Render request builder and response
    elements.swaggerRequestBuilder.innerHTML = generateRequestBuilderHTML(state.swaggerSpec, state.swaggerSelectedEndpoint);
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

    try {
        const response = await SwaggerRequest({
            method: state.swaggerSelectedEndpoint.method,
            url,
            headers,
            body,
        });

        elements.swaggerResponse.innerHTML = generateLiveResponseHTML(response);
        setupSwaggerResponseTabs();
    } catch (err) {
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

async function loadFile(file, options = {}) {
    if (!file) {
        return;
    }

    if (state.currentFile && state.currentFile !== file) {
        saveDocumentCache();
    }

    const skipDocumentCacheRestore = Boolean(options?.skipDocumentCacheRestore);
    const keepFindTabOpen = Boolean(options?.keepFindTabOpen);
    const switchingDocument = Boolean(state.currentFile && state.currentFile !== file);

    state.suspendDocumentCacheSave = true;

    if (state.currentFile && state.currentFile !== file) {
        await closeOpenLspDocument();
        await closeCurrentTyposDocument();
        lspSpellCheckExclusions.symbols = [];
        lspSpellCheckExclusions.tokens = [];
        lspSpellCheckExclusions.keywords = [];
        notesSpellCheckHandle?.setExclusions([]);
        // Clear rendered diagnostics immediately while the new file loads.
        state.currentFileUri = '';
        clearVisibleLspDiagnostics();
    }

    if (switchingDocument && !keepFindTabOpen) {
        setToolsTab('find');
    }

    const fileName = file ? getPathFileName(file) : 'json file';
    let stickyId = null;

    try {
        clearHexSource();

        // Capture the current project context to prevent autosave issues if user switches projects
        state.currentFileProject = await GetCurrentProject();

        const loadingJson     = isStructuredDataFile(file);
        const loadingMarkdown = isMarkdownNotesFile(file);
        const loadingHtml     = isHtmlViewFile(file);
        const loadingImage    = isImageFile(file);
        const loadingCsv      = isCsvFile(file);
        stickyId = loadingJson ? Date.now() : null;

        if (!loadingMarkdown) {
            // Keep wrap-mode effects from leaking into non-markdown editors.
            setEditorWrapMode(false);
        }

        if (loadingImage) {
            state.currentFile = file;
            emitCurrentFileName();
            await refreshFileMetaMarkdown(file);
            state.currentFileType = 'image';
            setCodeEditorMode(false);
            elements.editorShell.dataset.fileType = 'image';
            state.swaggerSpec = null;
            state.swaggerRunAvailable = false;
            updateTabVisibility('image');

            // ResolveFilePath expands $NOTES/$PROJECT/etc variables the same way
            // GetFile does. GetImage expects a path without a leading separator
            // (it prepends one itself), so strip it after resolution.
            const resolvedPath = await ResolveFilePath(file);
            state.currentFileUri = filePathToUri(resolvedPath || file);
            const imageData = await GetImage(resolvedPath.replace(/^[/\\]+/, ''));
            if (imageData.startsWith('error:')) {
                notifyTerminal(`Failed to load image: ${imageData}`, 'error');
                return;
            }
            elements.imageViewImg.src = imageData;
            elements.imageViewImg.dataset.originalFilename = fileName;
            enableFullscreenImages(elements.imageViewWrap);
            enableImageContextMenus(elements.imageViewWrap);

            setViewMode('image-view');
            if (!skipDocumentCacheRestore) {
                await restoreDocumentCache(file);
            }
            state.suspendDocumentCacheSave = false;
            saveDocumentCache();
            setDirty(false);
            renderFileList();
            if (!keepFindTabOpen && elements.toolsTabFind?.getAttribute('aria-selected') === 'true') {
                closeFindBar();
            }
            return;
        }

        /*if (loadingJson) {
            openStickyProgress(stickyId, `Loading ${fileName}… reading file`);
        }*/

        const result = await GetFile(file);

        state.currentFile = file;
        state.currentFileUri = await resolveNotesFileUri(file);
        clearCurrentFileLspDiagnosticsCache();
        emitCurrentFileName();
        await refreshFileMetaMarkdown(file);

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

        const doc = result.contents;
        const isBinaryFile = Boolean(result.binary ?? result.text);

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
            setViewMode('hex');

            if (stickyId) {
                closeStickyProgress(stickyId);
            }

            setDirty(false);
            renderFileList();

            if (!keepFindTabOpen && elements.toolsTabFind?.getAttribute('aria-selected') === 'true') {
                closeFindBar();
            }
            return;
        }

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
            state.swaggerSpec = parseSwaggerSpec(doc);
            state.swaggerRunAvailable = hasSwaggerKey(state.swaggerSpec);

            if (!state.swaggerSpec) {
                closeStickyProgress(stickyId, `Failed to parse ${fileName}`, 'warn');
            }

            state.swaggerSelectedEndpoint = null;
            state.swaggerEndpointFilter = '';
            
            // Update UI for JSON / swagger-capable JSON
            updateTabVisibility('json');
            
            // Set editor content (use regular editor with line numbers for JSON/YAML)
            setMainEditorValue(doc || '');
            refreshEditorLanguage(file, doc || '');

            // Render JSON tree view
            updateStickyProgress(stickyId, `Loading ${fileName}… rendering viewer`);
            await yieldToUI();
            // Defer JSON/YAML tree render until the View tab is clicked; loadFile no longer eagerly renders it.
            state.swaggerViewCurrent = false;
            
            // Render swagger UI only for JSON documents with a top-level swagger key
            if (state.swaggerRunAvailable) {
                updateStickyProgress(stickyId, `Loading ${fileName}… rendering run view`);
                await yieldToUI();
                renderSwaggerUI();
            } else {
                elements.swaggerResponse.innerHTML = '';
                elements.swaggerRequestBuilder.innerHTML = '';
                elements.swaggerEndpoints.innerHTML = '';
            }
            
            // Set default view mode to editor for JSON/YAML files
            setViewMode('swagger-edit');
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
            setMainEditorValue(doc || '');
            refreshEditorLanguage(file, doc || '');

            // Render markdown views
            await renderMarkdown();
            await renderJupyterView();

            // Set default view mode to viewer
            setViewMode('viewer');
        } else if (loadingHtml) {
            state.currentFileType = 'html';
            setCodeEditorMode(true);
            elements.editorShell.dataset.fileType = 'html';
            state.swaggerSpec = null;
            state.swaggerRunAvailable = false;

            // Update UI for HTML
            updateTabVisibility('html');

            // Set editor content
            setMainEditorValue(doc || '');
            refreshEditorLanguage(file, doc || '');

            // Render sandboxed preview and default to View first.
            renderHtmlView();
            setViewMode('viewer');
        } else if (loadingCsv) {
            state.currentFileType = 'csv';
            setCodeEditorMode(false);
            elements.editorShell.dataset.fileType = 'csv';
            state.swaggerSpec = null;
            state.swaggerRunAvailable = false;

            // Update UI for CSV
            updateTabVisibility('csv');

            // Set editor content (raw text for Edit tab)
            setMainEditorValue(doc || '');

            // Render table view
            renderCsvView(doc || '');

            // Default to the table view
            setViewMode('csv-view');
        } else {
            state.currentFileType = 'code';
            setCodeEditorMode(true);
            elements.editorShell.dataset.fileType = 'code';
            state.swaggerSpec = null;
            state.swaggerRunAvailable = false;
            
            // Update UI for code
            updateTabVisibility('code');
            
            // Set editor content
            setMainEditorValue(doc || '');
            refreshEditorLanguage(file, doc || '');
            
            // Set default view mode to editor
            setViewMode('editor');
        }
        
        setDirty(false);
        renderFileList();

        // JSON/YAML tree view render is deferred until the View tab is clicked (see swaggerViewCurrent).

        // Clear active Find state when loading a new file.
        if (!keepFindTabOpen && elements.toolsTabFind?.getAttribute('aria-selected') === 'true') {
            closeFindBar();
        }

        if (isCurrentFileLspEligible()) {
            await openCurrentLspDocument(elements.editor.value || '');
        }

        // Spellcheck via typos-lsp for eligible files (falls back to aspell).
        await openCurrentTyposDocument(elements.editor.value || '');

        if (!skipDocumentCacheRestore) {
            await restoreDocumentCache(file);
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

async function saveFile() {
    if (!state.currentFile) {
        setStatus('Select a note before saving', true);
        return;
    }

    try {
        const content = getMainEditorValue();
        
        // Use the saved project context to prevent overwrites if user switched projects
        await SaveFile(state.currentFile, content, state.currentFileProject || '');
        if (state.lspOpenFile === state.currentFile && isCurrentFileLspEligible()) {
            await NotesLspSaveDocument(state.currentFile);
        }
        setDirty(false);
        maybeRefreshProjectFindAfterSave();
    } catch (err) {
        notifyTerminal(`Failed to save ${state.currentFile}`, 'error');
        console.error(err);
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
    await saveFile();
}

function openDeletePrompt(file) {
    state.deletingFile = file;
    state.deleteConfirmAction = null;
    const fileName = getPathFileName(file);
    if (elements.deleteModalTitle) {
        elements.deleteModalTitle.textContent = 'Delete note';
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
        elements.deleteModalTitle.textContent = title;
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
    const fileUri = state.currentFile === fileToDelete
        ? state.currentFileUri
        : await resolveNotesFileUri(fileToDelete);

    try {
        if (state.currentFile === fileToDelete) {
            await closeOpenLspDocument();
            await closeCurrentTyposDocument();
        }
        await DeleteFile(fileToDelete);
        if (fileUri) {
            lspDiagnosticsStore.delete(fileUri);
        }
        if (state.currentFile === fileToDelete) {
            state.currentFile = '';
            state.currentFileUri = '';
            state.currentFileProject = '';
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
        setStatus(`Deleted ${fileName}`, false);
    } catch (err) {
        notifyTerminal(`Failed to delete ${fileName}`, 'error');
        console.error(err);
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

function scrollEditorToSelection(editor, selectionStart) {
    // Calculate which line the selection starts on and scroll to it
    const text = editor.value;
    const beforeSelection = text.substring(0, selectionStart);
    const linesBefore = beforeSelection.split('\n').length - 1;
    
    // Get the line height from computed styles
    const styles = window.getComputedStyle(editor);
    const lineHeight = parseFloat(styles.lineHeight) || parseFloat(styles.fontSize) * 1.4;
    
    // Calculate the scroll position (subtract half viewport height to center the line)
    const viewportHeight = editor.clientHeight;
    const targetScrollTop = Math.max(0, (linesBefore * lineHeight) - (viewportHeight / 2));
    
    editor.scrollTop = targetScrollTop;
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

    const hasValue = (elements.findFilesInput.value || '').trim().length > 0;
    elements.findFilesClear.dataset.visible = hasValue ? 'true' : 'false';
    elements.findFilesClear.setAttribute('aria-hidden', hasValue ? 'false' : 'true');
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
    const editor = elements.editor;
    if (!editor) {
        return 0;
    }

    const targetLine = Math.max(1, Number.parseInt(String(lineNumber), 10) || 1);
    const text = String(editor.value || '');
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
    if (!elements.editor) {
        return;
    }

    const start = editorOffsetForLine(lineNumber);
    const text = getMainEditorValue();
    const nextBreak = text.indexOf('\n', start);
    const end = nextBreak === -1 ? text.length : nextBreak;

    setMainEditorSelectionRange(start, end);

    if (isMonacoActive()) {
        monacoMainEditor.revealOffset(start);
        monacoMainEditor.focus();
        return;
    }

    if (state.useMonacoEditor) {
        const schedule = typeof requestAnimationFrame === 'function'
            ? requestAnimationFrame
            : (cb) => setTimeout(cb, 16);

        const tryRevealWhenReady = (remainingFrames) => {
            if (isMonacoActive()) {
                monacoMainEditor.setSelectionOffsets(start, end);
                monacoMainEditor.revealOffset(start);
                monacoMainEditor.focus();
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

    const editor = elements.editor;
    editor.focus();
    scrollEditorToSelection(editor, start);
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
        await NotesHistoryAdd(file);
        await loadFile(file, {
            skipDocumentCacheRestore: true,
            keepFindTabOpen: true,
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
    return Boolean(getActiveFindEditor());
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
    return isMonacoActive() && state.viewMode === 'editor';
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
    const editorEl = getActiveFindEditor();
    if (!editorEl && !isMonacoFindProxyActive()) {
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

    if (isMonacoFindProxyActive()) {
        replaceMainEditorRange(match.start, match.end, replacement);
    } else {
        editorEl.focus();
        editorEl.setSelectionRange(match.start, match.end);
        editorEl.setRangeText(replacement, match.start, match.end, 'end');
        editorEl.dispatchEvent(new Event('input', { bubbles: true }));
    }

    performFind();
}

function replaceAllMatches() {
    const editorEl = getActiveFindEditor();
    if (!editorEl && !isMonacoFindProxyActive()) {
        return;
    }

    const query = String(elements.findInput?.value || '');
    if (!query) {
        return;
    }

    const source = isMonacoFindProxyActive()
        ? getMainEditorValue()
        : String(editorEl.value || '');
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

    if (isMonacoFindProxyActive()) {
        setMainEditorValue(next);
        elements.editor.dispatchEvent(new Event('input', { bubbles: true }));
    } else {
        editorEl.value = next;
        editorEl.dispatchEvent(new Event('input', { bubbles: true }));
    }
    performFind();
}

function getActiveFindContainer() {
    if (state.viewMode === 'jupyter') {
        return elements.jupyter;
    }

    if (state.viewMode === 'swagger-view') {
        return elements.swaggerView;
    }

    return elements.preview;
}

function getActiveFindEditor() {
    if (state.viewMode === 'editor') {
        return elements.editor;
    }

    if (state.viewMode === 'swagger-edit') {
        return state.currentFileType === 'json' ? elements.editor : null;
    }

    return null;
}

function clearHighlights() {
    // Clear highlights in all rendered panes that support find.
    [elements.preview, elements.jupyter, elements.swaggerView].forEach((container) => {
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

    const activeEditor = getActiveFindEditor();
    if (activeEditor) {
        activeEditor.setSelectionRange(0, 0);
    }
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

    if (getActiveFindEditor()) {
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

        const matches = monacoMainEditor.findMatches(request.query, {
            isRegex: request.isRegex,
            matchCase: request.matchCase,
            limit: 10000,
        });

        for (const match of matches) {
            state.findMatches.push({ start: match.start, end: match.end });
        }
        return;
    }

    const editorEl = getActiveFindEditor();
    if (!editorEl || !findPattern) {
        return;
    }

    const text = String(editorEl.value || '');
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

    const editorEl = getActiveFindEditor();
    if (isMonacoFindProxyActive()) {
        const match = state.findMatches[state.findCurrentIndex];
        if (!match) {
            return;
        }

        if (focusEditor) {
            monacoMainEditor.focus();
        }
        monacoMainEditor.setSelectionOffsets(match.start, match.end);
        monacoMainEditor.revealOffset(match.start);
        return;
    }

    if (editorEl) {
        const match = state.findMatches[state.findCurrentIndex];

        if (focusEditor) {
            editorEl.focus();
            editorEl.setSelectionRange(match.start, match.end);
            // Ensure the editor scrolls to show the selection
            scrollEditorToSelection(editorEl, match.start);
        } else {
            // Scroll to the match without permanently stealing focus.
            // Set selection and scroll without permanently focusing the editor.
            editorEl.setSelectionRange(match.start, match.end);
            scrollEditorToSelection(editorEl, match.start);
        }
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

function openNewFilePrompt() {
    state.renamingFile = null;
    elements.modal.dataset.open = 'true';
    elements.modal.setAttribute('aria-hidden', 'false');
    elements.modalInput.value = '';
    elements.modalLocation.textContent = '$NOTES';
    elements.modalLocation.style.display = '';
    elements.modal.querySelector('#notes-modal-title').textContent = 'New note name';
    elements.modalCreate.textContent = 'Create';
    setTimeout(() => {
        elements.modalInput.focus();
    }, 0);
}

async function openRenamePrompt(file) {
    const source = String(file || '').trim();
    state.renamingFile = file;
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
    elements.modal.querySelector('#notes-modal-title').textContent = 'Rename note';
    elements.modalCreate.textContent = 'Rename';
    setTimeout(() => {
        elements.modalInput.focus();
        elements.modalInput.select();
    }, 0);
}

function closeNewFilePrompt() {
    elements.modal.dataset.open = 'false';
    elements.modal.setAttribute('aria-hidden', 'true');
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
    return `${dir}${relativePath}`;
}

function enableImageContextMenus(container) {
    const images = container.querySelectorAll('img');
    images.forEach((img) => {
        img.addEventListener('contextmenu', async (e) => {
            e.preventDefault();
            
            const src = img.src;
            if (!src) return;
            
            // Use the original filename from the data attribute if available
            let filename = img.dataset.originalFilename || 'Image';
            
            // For relative image paths (from note markdown images), convert to dataURL
            let dataURLToCopy = src;
            if (src.startsWith('file://') || (!src.startsWith('data:') && !src.startsWith('http'))) {
                // It's a file path, we need to fetch and convert to dataURL
                try {
                    const response = await fetch(src);
                    const blob = await response.blob();
                    dataURLToCopy = await new Promise((resolve) => {
                        const reader = new FileReader();
                        reader.onload = () => resolve(reader.result);
                        reader.readAsDataURL(blob);
                    });
                } catch (err) {
                    console.error('Failed to load image for clipboard:', err);
                    return;
                }
            }
            
            showLocalMenu({
                title: filename,
                options: ['Copy image to clipboard', 'Save image...', 'Ask AI...'],
                x: e.clientX,
                y: e.clientY,
                showNextToMouseCursor: true,
                icons: [0xf0c5, 0xf0c7, CONTEXT_ICON_ASK_AI],
                onSelect: (index) => {
                    if (index === 0) {
                        TerminalCopyImageDataURL(dataURLToCopy).catch(() => {
                            notifyTerminal('Failed to copy image to clipboard', 'error');
                        });
                    } else if (index === 1) {
                        saveImageToFile(filename, dataURLToCopy);
                    } else if (index === 2) {
                        askAIAboutCurrentDocument();
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

async function openFileListContextMenu(file, x, y) {
    const menuItems = [];

    let fileUrl = '';
    const fileLabel = getPathFileName(file);
    try {
        const resolvedPath = await ResolveFilePath(file);
        const normalized = String(resolvedPath || '').replaceAll('\\', '/');
        if (normalized) {
            if (/^[a-zA-Z]:\//.test(normalized)) {
                fileUrl = `file:///${normalized}`;
            } else {
                fileUrl = `file://${normalized.startsWith('/') ? normalized : `/${normalized}`}`;
            }
        }
    } catch {
        // Keep local actions available even if path resolution fails.
    }

    let goMenuItems = [];
    if (fileUrl) {
        try {
            const resolvedMenuItems = await GetHyperlinkMenuActions(fileUrl, fileLabel || fileUrl);
            goMenuItems = Array.isArray(resolvedMenuItems) ? resolvedMenuItems : [];
        } catch {
            setStatus('Failed to load file actions', true);
        }
    }

    if (goMenuItems.length > 0) {
        //menuItems.push({ title: '-', icon: 0 });

        goMenuItems.forEach((item) => {
            menuItems.push({
                title: String(item?.title || ''),
                icon: Number(item?.icon) || 0,
                onSelect: () => {
                    RunHyperlinkMenuAction(fileUrl, fileLabel || fileUrl, String(item?.action || ''))
                        .catch(() => {
                            setStatus('Failed to execute file action', true);
                        });
                },
            });
        });
    }

    showNotesLocalMenu(menuItems, x, y, getPathFileName(file) || 'File actions');
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
    const start = elements.editor.selectionStart;
    const end = elements.editor.selectionEnd;
    return elements.editor.value.slice(start, end);
}

function getRenderedSelectionText(container) {
    const selection = window.getSelection();
    if (!selection || selection.rangeCount === 0 || selection.isCollapsed) {
        return '';
    }

    const anchorNode = selection.anchorNode;
    const focusNode = selection.focusNode;
    const selectionInContainer =
        (anchorNode && container.contains(anchorNode)) ||
        (focusNode && container.contains(focusNode));

    if (!selectionInContainer) {
        return '';
    }

    return selection.toString();
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

function createCopyMenuItem(getText, title = 'Copy') {
    return {
        title,
        icon: CONTEXT_ICON_COPY,
        onSelect: () => {
            copyTextToClipboard(getText());
        },
    };
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

function createPrintMenuItem(title = 'Print...') {
    return {
        title,
        icon: CONTEXT_ICON_PRINT,
        onSelect: () => {
            WindowPrint();
        },
    };
}

function getCurrentDocumentContentsForAI() {
    const fileType = String(state.currentFileType || '').toLowerCase();

    let contents = '';
    if (fileType === 'image') {
        contents = String(elements.imageViewImg?.src || '').trim();
    } else if (fileType === 'binary') {
        contents = String(state.hexSourceValue || elements.editor.value || '');
    } else {
        contents = String(elements.editor.value || '');
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

    try {
        await AskAI('notesDocument', fileName, aiContext);
    } catch (err) {
        notifyTerminal('Failed to ask AI about this document', 'error');
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

function collectAIPromptTargets() {
    // Deprecated: legacy DOM-based prompt collector kept as a fallback for the
    // initial refresh before the async backend list resolves.
    if (!elements.aiOutput) {
        return [];
    }

    const targets = [];

    const prefixPromptHeadings = elements.aiOutput.querySelectorAll('.notes-ai-prefix h2, .notes-ai-prefix h3');
    for (const heading of prefixPromptHeadings) {
        const label = String(heading.textContent || '').trim().toLowerCase();
        if (label !== 'prompt') {
            continue;
        }

        let summaryText = '';
        let sibling = heading.nextElementSibling;
        while (sibling) {
            summaryText = summarizePromptText(sibling.textContent, '');
            if (summaryText) {
                break;
            }
            sibling = sibling.nextElementSibling;
        }

        targets.push({
            source: 'dom',
            element: heading,
            summary: summarizePromptText(summaryText, `Prompt ${targets.length + 1}`),
        });
    }

    return targets;
}

// Backend-provided prompt log metadata, refreshed on aiJobFinish / panel init.
// Each entry is { sessionId, promptId, heading, summary } from Go via
// ListAIPromptLogs. The heading is the "<!-- request heading: … -->" comment
// captured when the prompt started; summary is the rendered dropdown label.
async function refreshAIPromptJumpFromBackend() {
    if (!elements.toolsAIPromptJump) {
        return;
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
        summary: summarizePromptText(String(meta?.heading || ''), `Prompt ${index + 1}`),
    })).filter((entry) => entry.sessionId > 0 && entry.promptId > 0);

    state.aiPromptJumpTargets = targets;
    elements.toolsAIPromptJump.disabled = targets.length === 0;
}

function renderAIPromptJumpDropdown() {
    // Initial synchronous render uses any DOM-collected prompts, then the
    // backend list overrides asynchronously.
    if (!elements.toolsAIPromptJump) {
        return;
    }
    const domTargets = collectAIPromptTargets();
    if (!Array.isArray(state.aiPromptJumpTargets) || state.aiPromptJumpTargets.length === 0) {
        state.aiPromptJumpTargets = domTargets;
    }
    elements.toolsAIPromptJump.disabled = state.aiPromptJumpTargets.length === 0;
    void refreshAIPromptJumpFromBackend();
}

function scheduleAIPromptJumpRefresh() {
    if (aiPromptJumpRefreshTimer) {
        clearTimeout(aiPromptJumpRefreshTimer);
    }

    aiPromptJumpRefreshTimer = setTimeout(() => {
        aiPromptJumpRefreshTimer = null;
        renderAIPromptJumpDropdown();
    }, 40);
}

async function jumpToAIPrompt(index) {
    const target = state.aiPromptJumpTargets[index];
    if (!target) {
        return;
    }

    if (target.source === 'backend' && target.sessionId > 0 && target.promptId > 0) {
        try {
            const markdown = await GetAIPromptLog(target.sessionId, target.promptId);
            aiPipelineFormatter.clear();
            if (markdown) {
                setAIFinalOutput(String(markdown), { forceBottom: true });
            }
            state.aiSessionCache = String(markdown || '');
        } catch (err) {
            console.error('Failed to load AI prompt log:', err);
        }
        return;
    }

    if (!elements.aiOutput || !(target.element instanceof Element)) {
        return;
    }

    const nextTop = Math.max(0, Number(target.element.offsetTop) - 8);
    elements.aiOutput.scrollTo({ top: nextTop, behavior: 'smooth' });
    state.aiStickToBottom = false;
    updateAIScrollBottomButton();
}

function openAIPromptJumpMenu() {
    if (!elements.toolsAIPromptJump) {
        return;
    }

    const targets = Array.isArray(state.aiPromptJumpTargets) ? state.aiPromptJumpTargets : [];
    if (targets.length === 0) {
        return;
    }

    const reversedTargets = [...targets].reverse();
    const options = reversedTargets.map((target) => target.summary);
    const rect = elements.toolsAIPromptJump.getBoundingClientRect();
    showLocalMenu({
        title: 'Prompts',
        options,
        x: rect.left,
        y: rect.bottom,
        showNextToMouseCursor: true,
        onSelect: (index) => {
            if (typeof index !== 'number' || index < 0 || index >= reversedTargets.length) {
                return;
            }
            const originalIndex = targets.indexOf(reversedTargets[index]);
            void jumpToAIPrompt(originalIndex);
        },
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
        const safeUpdated = escapeHtml(session.updated || '');
        const safeCreated = escapeHtml(session.created || '');
        const count = Number(session.entryCount) || 0;
        const active = session.active === true;
        const createdHtml = safeCreated ? `<div class="notes-ai-settings-session-meta">Created ${safeCreated}</div>` : '';
        const updatedHtml = safeUpdated ? `<div class="notes-ai-settings-session-meta">Updated ${safeUpdated}</div>` : '';
        return `
            <div class="notes-ai-settings-session-item" data-session-id="${tableId}" data-active="${active ? 'true' : 'false'}">
                <div class="notes-ai-settings-session-main">
                    <div class="notes-ai-settings-session-heading">${safeSummary}</div>
                    <div class="notes-ai-settings-session-meta">${count} entr${count === 1 ? 'y' : 'ies'}</div>
                    ${createdHtml}
                    ${updatedHtml}
                </div>
                <button type="button" class="notes-ai-settings-session-delete" data-action="delete" data-session-id="${tableId}" aria-label="Delete session"></button>
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
        const safeTitle = escapeHtml(entry.prompt || 'AI prompt');
        const safeCmdLine = escapeHtml(entry.commandLine || '');
        const safeExcerpt = escapeHtml(entry.excerpt || '');
        const safeOutput = escapeHtml(entry.outputBlock || '');
        const cmdLineHtml = safeCmdLine ? `<div class="notes-ai-settings-history-meta">${safeCmdLine}</div>` : '';
        const outputHtml = safeOutput ? `<div class="notes-ai-settings-history-output">${safeOutput}</div>` : '';
        const excerptHtml = safeExcerpt ? `<div class="notes-ai-settings-history-excerpt">${safeExcerpt}</div>` : '';
        return `
            <div class="notes-ai-settings-history-item">
                <div class="notes-ai-settings-history-heading">${safeTitle}</div>
                ${cmdLineHtml}
                ${outputHtml}
                ${excerptHtml}
            </div>
        `;
    }).join('');

    elements.aiSettingsHistoryList.innerHTML = items;
}

function renderAIToolsList() {
    if (!elements.aiSettingsToolsList) {
        return;
    }

    const tools = Array.isArray(state.aiToolsList) ? state.aiToolsList : [];
    if (tools.length === 0) {
        elements.aiSettingsToolsList.textContent = 'No tools available.';
        elements.aiSettingsToolsList.dataset.empty = 'true';
        return;
    }

    elements.aiSettingsToolsList.dataset.empty = 'false';
    const items = tools.map((tool, index) => {
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

    const servers = Array.isArray(state.aiMcpServersList) ? state.aiMcpServersList : [];
    if (servers.length === 0) {
        elements.aiSettingsMcpList.textContent = 'No MCP servers found in config.';
        elements.aiSettingsMcpList.dataset.empty = 'true';
        return;
    }

    elements.aiSettingsMcpList.dataset.empty = 'false';
    const items = servers.map((server, index) => {
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

    return `## ${name}\n\n- [${allowInSubagent ? 'x' : ' '}] Allow in sub-agent\n\n${description}\n\n### JSON Schema\n\n\`\`\`json\n${schema}\n\`\`\``;
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

    return `## ${name}\n\n- [${loaded ? 'x' : ' '}] Loaded\n- [${loadable ? 'x' : ' '}] Loadable\n- Source: ${source}\n- Type: ${serverType}${conflict}\n\n### Config\n\n\`\`\`json\n${config}\n\`\`\``;
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
        const [options, current] = await Promise.all([
            ListAIModelSelections(),
            GetCurrentAIModelSelection(),
        ]);

        state.aiModelSelections = Array.isArray(options) ? options : [];
        state.aiCurrentModelSelection = String(current || '').trim();

        const fallback = state.aiModelSelections[0] || 'Model';
        elements.aiSettingsModelPicker.textContent = state.aiCurrentModelSelection || fallback;
        elements.aiSettingsModelPicker.disabled = state.aiModelSelections.length === 0;
    } catch (err) {
        console.error('Failed to refresh AI model picker:', err);
        elements.aiSettingsModelPicker.textContent = 'Model';
        elements.aiSettingsModelPicker.disabled = true;
    }
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

function extractTableData(table) {
    const getCellContent = (cell) => {
        // Try to get content from the wrapped span (excludes cell refs)
        const wrap = cell.querySelector('.notes-table-cell-wrap > span:first-child');
        if (wrap) return String(wrap.textContent || '').trim();
        // Otherwise exclude sort icons and cell refs from childNodes
        return Array.from(cell.childNodes)
            .filter(n => !n.classList?.contains('notes-sort-icon') && !n.classList?.contains('notes-cellref'))
            .map(n => n.textContent)
            .join('')
            .trim();
    };

    const rows = [];
    const headerRow = table.querySelector('thead tr');
    if (headerRow) {
        rows.push(Array.from(headerRow.querySelectorAll('th, td')).map(cell => getCellContent(cell)));
    }
    Array.from(table.querySelectorAll('tbody tr')).forEach(tr => {
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
                const data = extractTableData(table);
                copyTextToClipboard(tableDataToMarkdown(data));
            },
        },
    ];
}

function highlightTableRow(table, rowIndex, isHighlighted) {
    const rows = table.querySelectorAll('tr');
    if (rowIndex < 0 || rowIndex >= rows.length) return;
    const row = rows[rowIndex];
    if (isHighlighted) {
        row.style.backgroundColor = 'var(--accent)';
        row.style.color = 'var(--bg)';
    } else {
        row.style.backgroundColor = '';
        row.style.color = '';
    }
}

function highlightTableColumn(table, colIndex, isHighlighted) {
    table.querySelectorAll('tr').forEach(row => {
        const cell = row.children[colIndex];
        if (cell) {
            if (isHighlighted) {
                cell.style.backgroundColor = 'var(--accent)';
                cell.style.color = 'var(--bg)';
            } else {
                cell.style.backgroundColor = '';
                cell.style.color = '';
            }
        }
    });
}

function clearTableHighlight(table) {
    table.querySelectorAll('tr').forEach(r => {
        r.style.backgroundColor = '';
        r.style.color = '';
        r.querySelectorAll('td, th').forEach(c => {
            c.style.backgroundColor = '';
            c.style.color = '';
        });
    });
}

function highlightEntireTable(table, isHighlighted) {
    table.querySelectorAll('tr').forEach(row => {
        if (isHighlighted) {
            row.style.backgroundColor = 'var(--accent)';
            row.style.color = 'var(--bg)';
        } else {
            row.style.backgroundColor = '';
            row.style.color = '';
        }
        row.querySelectorAll('td, th').forEach(cell => {
            if (isHighlighted) {
                cell.style.backgroundColor = 'var(--accent)';
                cell.style.color = 'var(--bg)';
            } else {
                cell.style.backgroundColor = '';
                cell.style.color = '';
            }
        });
    });
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
                    const blocks = findMarkdownTableBlocks(elements.editor?.value || '');
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
                    const blocks = findMarkdownTableBlocks(elements.editor?.value || '');
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
                    const blocks = findMarkdownTableBlocks(elements.editor?.value || '');
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
                    const blocks = findMarkdownTableBlocks(elements.editor?.value || '');
                    const block = blocks[tableIndex];
                    if (block) deleteMarkdownColumn(block, pos.col);
                }
            },
        },
    ];
}

function initAIOutputContextMenu(container) {
    if (!container) {
        return;
    }

    container.addEventListener('contextmenu', (e) => {
        const anchor = e.target instanceof Element ? e.target.closest('a[href]') : null;
        if (anchor && container.contains(anchor)) {
            e.preventDefault();
            e.stopPropagation();
            openHyperlinkContextMenu(anchor);
            return;
        }

        e.preventDefault();

        const table = e.target instanceof Element ? e.target.closest('table') : null;
        const codeBlockText = getRenderedCodeBlockText(container, e.target);
        const tableItems = table && container.contains(table)
            ? [...createTableCopyMenuItems(table), { title: '-' }]
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
                createCopyMenuItem(() => getRenderedSelectionText(container), 'Copy selection'),
                createCopyMenuItem(() => codeBlockText, 'Copy code'),
                { title: '-' },
            ]
            : [
                createCopyMenuItem(() => getRenderedSelectionText(container), 'Copy'),
                { title: '-' },
            ];

        const menuItems = [
            ...copyItems,
            ...tableItems,
            ...wordWrapItems,
        ];

        let highlightCallback = null;
        let cancelCallback = null;
        if (table && container.contains(table)) {
            highlightCallback = (itemIndex) => {
                const item = menuItems[itemIndex];
                if (!item) {
                    return;
                }
                clearTableHighlight(table);
                if (item.title.toLowerCase().includes('copy table') || item.title.toLowerCase().includes('word wrap table')) {
                    highlightEntireTable(table, true);
                }
            };
            cancelCallback = () => clearTableHighlight(table);
        }

        showNotesLocalMenu(menuItems, e.clientX, e.clientY, 'Select an action', highlightCallback, cancelCallback);
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
    wrapTablesForHorizontalScroll(container);
    setupTableSorting(container);
    applyNotesTableWordWrapMode(container);
    pinAIScrollableBlocksToBottom(container);
}

function initRenderedNotesContextMenu(container, viewMode) {
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
                if (displayHref && hyperlinkHoverTooltipEl.dataset.href !== displayHref) {
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

    container.addEventListener('contextmenu', (e) => {
        const anchor = e.target instanceof Element ? e.target.closest('a[href]') : null;
        if (anchor && container.contains(anchor)) {
            e.preventDefault();
            e.stopPropagation();
            openHyperlinkContextMenu(anchor);
            return;
        }

        if (e.target instanceof Element && e.target.closest('img')) {
            return;
        }

        e.preventDefault();

        const table = e.target instanceof Element ? e.target.closest('table') : null;
        const codeBlockText = getRenderedCodeBlockText(container, e.target);
        const isRunMode = state.viewMode === 'jupyter';
        const tableIndex = table ? Array.from(container.querySelectorAll('table')).indexOf(table) : -1;
        const tableItems = table && container.contains(table)
            ? [...createTableCopyMenuItems(table), { title: '-' }]
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
                createCopyMenuItem(() => getRenderedSelectionText(container), 'Copy selection'),
                createCopyMenuItem(() => codeBlockText, 'Copy code'),
                { title: '-' },
            ]
            : [
                createCopyMenuItem(() => getRenderedSelectionText(container), 'Copy'),
                { title: '-' },
            ];

        const allMenuItems = [
            ...copyItems,
            ...tableItems,
            ...wordWrapItems,
            ...insertItems,
            createFindMenuItem('Find'),
            createAskAIDocumentMenuItem(),
            createPrintMenuItem('Print'),
        ];

        // Set up highlight callback for table row/column items if table exists
        let highlightCallback = null;
        let cancelCallback = null;
        if (table) {
            if (isRunMode) {
                const pos = getCellPosition(e.target, table);
                if (pos) {
                    highlightCallback = (itemIndex) => {
                        const item = allMenuItems[itemIndex];
                        if (!item) return;
                        // Unhighlight all first
                        clearTableHighlight(table);
                        // Highlight based on item title
                        if (item.title.toLowerCase().includes('copy table')) {
                            highlightEntireTable(table, true);
                        } else if (item.title.includes('row') && !item.title.includes('column')) {
                            highlightTableRow(table, pos.row, true);
                        } else if (item.title.includes('column')) {
                            highlightTableColumn(table, pos.col, true);
                        }
                    };
                    cancelCallback = () => clearTableHighlight(table);
                }
            } else {
                // Enable highlight for copy table items even when not in Run mode
                highlightCallback = (itemIndex) => {
                    const item = allMenuItems[itemIndex];
                    if (!item) return;
                    clearTableHighlight(table);
                    if (item.title.toLowerCase().includes('copy table')) {
                        highlightEntireTable(table, true);
                    }
                };
                cancelCallback = () => clearTableHighlight(table);
            }
        }

        showNotesLocalMenu(allMenuItems, e.clientX, e.clientY, 'Select an action', highlightCallback, cancelCallback);
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
    const menuTitle = options.menuTitle || 'JSON/YAML field';

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

        try {
            await RenameFile(state.renamingFile, fileName);
            await refreshFiles({ skipHistoryRestore: true });
            if (state.currentFile === state.renamingFile) {
                await loadFile(fileName);
            }
            closeNewFilePrompt();
            setStatus(`Renamed to ${fileName}`, false);
        } catch (err) {
            notifyTerminal(`Failed to rename file: ${err}`, 'error');
            console.error(err);
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
        notifyTerminal(`${fileName} already exists`, 'warn');
        return;
    }

    try {
        await SaveFile(fileName, '', '');
        await refreshFiles();
        await loadFile(fileName);
        setViewMode('editor');
        closeNewFilePrompt();
        setStatus(`Created ${fileName}`, false);
    } catch (err) {
        notifyTerminal(`Failed to create ${fileName}`, 'error');
        console.error(err);
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
        //setViewMode('editor');
        setViewMode('viewer');
        setStatus(`Created ${fileName}`, false);
    } catch (err) {
        notifyTerminal(`Failed to create ${fileName}`, 'error');
        console.error(err);
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

EventsOn("notesUpdate", () => {
    CancelNotesListFiles().catch(() => {}).finally(() => {
        refreshFiles();
    });
});

EventsOn("notesFileChanged", async () => {
    if (!state.currentFile) {
        return;
    }

    if (state.dirty) {
        setStatus('File changed on disk but has unsaved edits in Notes', true);
        return;
    }

    const shouldRestoreCaret = (state.viewMode === 'editor' || state.viewMode === 'swagger-edit') && !!elements.editor;
    const previousSelection = shouldRestoreCaret
        ? {
            start: Number(elements.editor.selectionStart) || 0,
            end: Number(elements.editor.selectionEnd) || 0,
            length: String(elements.editor.value || '').length,
            hadFocus: document.activeElement === elements.editor,
        }
        : null;

    try {
        await loadFile(state.currentFile);

        if (previousSelection && (state.viewMode === 'editor' || state.viewMode === 'swagger-edit')) {
            const nextLen = String(elements.editor.value || '').length;
            const ratio = previousSelection.length > 0 ? nextLen / previousSelection.length : 1;
            const nextStart = Math.min(Math.round(previousSelection.start * ratio), nextLen);
            const nextEnd = Math.min(Math.round(previousSelection.end * ratio), nextLen);

            if (previousSelection.hadFocus) {
                elements.editor.focus();
            }
            elements.editor.setSelectionRange(nextStart, nextEnd);
        }
    } catch (err) {
        setStatus(`Failed to reload ${state.currentFile}`, true);
        console.error(err);
    }
});

EventsOn("notesShowLspOptions", async () => {
    if (!isCurrentFileLspEligible() || !elements.editor) {
        notifyTerminal("No Language Server Protocol (LSP) is not supported for this file type", "warn");
        return;
    }

    const languageID = await ResolveNotesLspLanguage(state.currentFile);
    if (!languageID) {
        notifyTerminal("No Language Server Protocol (LSP) has been defined for this file type", "warn");
        return;
    }

    const rect = elements.editor.getBoundingClientRect();
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

EventsOn("noteRun", (data) => {
    const { blockId, output, isError } = data;

    const outputBlock = elements.jupyter.querySelector(`[data-block-id="${blockId}"] .jupyter-output`);
    if (!outputBlock) return;

    const text = String(output ?? '');
    const isErr = String(isError) === 'true';

    if (outputBlock.childNodes.length > 0 && text.length > 0 && text[0] !== '\n' && text[0] !== '\r') {
        outputBlock.appendChild(document.createTextNode('\n'));
    }

    const span = document.createElement('span');
    span.className = isErr ? 'jupyter-output-line-error' : 'jupyter-output-line';
    span.textContent = text;
    outputBlock.appendChild(span);
    scrollJupyterOutputToBottom(outputBlock);
});

EventsOn("noteComplete", (data) => {
    const { blockId } = data;

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

    // Per-prompt log files: clear the panel so only the new prompt renders.
    aiPipelineFormatter.clear();
    aiPipelineFormatter.startJob(String(title || ''));
    scheduleAIPromptJumpRefresh();
    requestAnimationFrame(() => {
        scrollAIOutputToBottom();
    });
}

const aiStreamOrder = {
    runId: null,
    nextSequence: 0,
    pending: new Map(),
    finalSequence: null,
};

function startOrderedAIJob(payload) {
    const runId = Number(payload?.runId);
    if (!Number.isSafeInteger(runId) || runId < 1) {
        startAIJob(payload);
        return;
    }
    aiStreamOrder.runId = runId;
    aiStreamOrder.nextSequence = 0;
    aiStreamOrder.pending.clear();
    aiStreamOrder.finalSequence = null;
    startAIJob(payload.title);
}

function flushOrderedAIStream() {
    while (aiStreamOrder.pending.has(aiStreamOrder.nextSequence)) {
        appendAIText(aiStreamOrder.pending.get(aiStreamOrder.nextSequence));
        aiStreamOrder.pending.delete(aiStreamOrder.nextSequence);
        aiStreamOrder.nextSequence++;
    }
    if (aiStreamOrder.finalSequence !== null && aiStreamOrder.nextSequence > aiStreamOrder.finalSequence) {
        aiStreamOrder.finalSequence = null;
        finishAIJob();
    }
}

function appendOrderedAIStream(payload) {
    if (!payload || typeof payload !== 'object') {
        const text = String(payload ?? '');
        if (text) appendAIText(text);
        return;
    }
    const runId = Number(payload.runId);
    const sequence = Number(payload.sequence);
    const text = String(payload.text ?? '');
    if (runId !== aiStreamOrder.runId || !Number.isSafeInteger(sequence) || sequence < aiStreamOrder.nextSequence || !text) {
        return;
    }
    aiStreamOrder.pending.set(sequence, text);
    flushOrderedAIStream();
}

function finishOrderedAIJob(payload) {
    const runId = Number(payload?.runId);
    const finalSequence = Number(payload?.finalSequence);
    if (runId !== aiStreamOrder.runId || !Number.isInteger(finalSequence)) {
        finishAIJob();
        return;
    }
    aiStreamOrder.finalSequence = finalSequence;
    flushOrderedAIStream();
}

function finishAIJob() {
    aiPipelineFormatter.finishJob();
    // Newly-finalized prompt log is now on disk; refresh the dropdown.
    void refreshAIPromptJumpFromBackend();
}

function appendAIText(text) {
    aiPipelineFormatter.appendChunk(text);
    scheduleAIPromptJumpRefresh();
    // The scroll listener keeps aiStickToBottom current, so don't re-measure the
    // container here — this runs once per streamed chunk and each measurement
    // forces a synchronous layout.
    if (state.aiStickToBottom) {
        scrollAIOutputToBottom();
    } else {
        scheduleAIScrollButtonUpdate();
    }
}

function setAIFinalOutput(text, options = {}) {
    const forceBottom = options.forceBottom === true;
    const shouldStick = forceBottom || state.aiStickToBottom || isAIOutputNearBottom();
    aiPipelineFormatter.setText(String(text || ''));
    scheduleAIPromptJumpRefresh();
    if (shouldStick) {
        requestAIScrollToBottom();
    } else {
        requestAnimationFrame(() => {
            updateAIScrollBottomButton();
        });
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

function clearAIBottomScrollRetries() {
    for (const timerId of aiBottomScrollRetryTimers) {
        clearTimeout(timerId);
    }
    aiBottomScrollRetryTimers = [];

    if (aiBottomChaseHandle) {
        cancelAnimationFrame(aiBottomChaseHandle);
        aiBottomChaseHandle = 0;
    }
    aiBottomChaseUntil = 0;
}

function requestAIScrollToBottom() {
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
    if (!elements.aiOutput) {
        return;
    }

    const output = elements.aiOutput;
    state.aiStickToBottom = true;

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

async function loadAISessionCache(workspaceName) {
    try {
        const cache = await GetAISessionCache(String(workspaceName || ''));
        state.aiSessionCache = String(cache || '');
        // The session log file on disk is the source of truth for the panel.
        // Fully reset the panel before rendering so content from a previously
        // active workspace cannot persist across a workspace (tmux tab) switch.
        aiPipelineFormatter.clear();
        if (state.aiSessionCache) {
            setAIFinalOutput(state.aiSessionCache, { forceBottom: true });
        }
        void refreshAIPromptJumpFromBackend();
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
    state.aiSessionCache = '';
    aiPipelineFormatter.clear();

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
    startOrderedAIJob(title);
    setToolsTab('ai');
    if (elements.toolsPanel.dataset.collapsed === 'true') {
        toggleToolsPanel();
    }
});

// Event emitted by Go when an AI job finishes
EventsOn("aiJobFinish", (payload) => {
    finishOrderedAIJob(payload);
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
    try {
        await AskAI('notesPromptUri', prompt, tools);
    } catch (err) {
        notifyTerminal('Failed to ask AI', 'error');
        console.error(err);
    }
});

// Access-request options rendered in the AI panel output (see
// agent.RequestWritePermission) resolve here rather than via a native/context menu.
document.addEventListener('ttyphoon-ai-tool-permission', async (e) => {
    const requestId = String(e.detail?.requestId || '');
    const decision = String(e.detail?.decision || '');
    const anchor = e.detail?.anchor;
    if (!requestId || !decision) {
        return;
    }

    if (elements.aiOutput) {
        Array.from(elements.aiOutput.querySelectorAll('a')).forEach((link) => {
            const href = link.getAttribute('href') || '';
            if (href.startsWith('ttyphoon://ai-tool-permission') && href.includes(`request=${requestId}`)) {
                link.classList.add('ai-tool-permission-resolved');
                link.removeAttribute('href');
            }
        });
    }
    if (anchor instanceof HTMLElement) {
        anchor.classList.add('ai-tool-permission-chosen');
    }

    try {
        await ResolveAIToolPermission(requestId, decision);
    } catch (err) {
        notifyTerminal('Failed to resolve tool access request', 'error');
        console.error(err);
    }
});

// Event listener for streaming AI responses
EventsOn("aiResponseStream", (chunk) => {
    appendOrderedAIStream(chunk);
});

// Event emitted by Go after the user selects a file from the ViewFileInNotes menu.
// Opens the file and selects the first (default) tab for the file type.
EventsOn('viewFileInNotesOpen', async (payload) => {
    const file = typeof payload === 'string' ? payload : String(payload ?? '');
    if (!file) return;

    try {
        NotesHistoryAdd(file).catch(() => {});
        await loadFile(file);
    } catch (err) {
        notifyTerminal(`Failed to load file: ${file}`, 'warn');
        console.error(err);
    }
});

// Event emitted by Go to open a file directly in the Edit tab.
EventsOn('viewFileInNotesEdit', async (payload) => {
    const file = typeof payload === 'string' ? payload : String(payload ?? '');
    if (!file) return;

    try {
        NotesHistoryAdd(file).catch(() => {});
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
if (elements.toolsClear) {
    elements.toolsClear.addEventListener('click', clearAIOutput);
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
if (elements.aiSettingsModelPicker) {
    elements.aiSettingsModelPicker.addEventListener('click', () => {
        openAIModelPickerMenu();
    });
}
if (elements.aiOutput) {
    elements.aiOutput.addEventListener('scroll', () => {
        // The bottom-chase writes scrollTop every frame, so this fires
        // constantly while streaming; coalesce the measurements it triggers.
        if (aiScrollButtonHandle) {
            return;
        }
        aiScrollButtonHandle = requestAnimationFrame(() => {
            aiScrollButtonHandle = 0;
            state.aiStickToBottom = isAIOutputNearBottom();
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
    elements.aiSettingsSessionsList.addEventListener('click', (event) => {
        const target = event.target instanceof HTMLElement ? event.target : null;
        if (!target) {
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

            SetActiveAISession(sessionId).then((sessionState) => {
                applyAISessionManagement(sessionState);
                renderAISessionManagement();
                void loadAISessionCache('');
            }).catch((err) => {
                notifyTerminal('Failed to switch AI session', 'error');
                console.error(err);
            });
            return;
        }
    });
}
if (elements.aiSettingsSessionNew) {
    elements.aiSettingsSessionNew.addEventListener('click', () => {
        CreateAISession().then((sessionState) => {
            applyAISessionManagement(sessionState);
            renderAISessionManagement();
            void loadAISessionCache('');
            notifyTerminal('AI session created', 'info');
        }).catch((err) => {
            notifyTerminal('Failed to create AI session', 'error');
            console.error(err);
        });
    });
}
if (elements.aiSettingsModal) {
    elements.aiSettingsModal.addEventListener('click', (event) => {
        if (event.target === elements.aiSettingsModal) {
            closeAISessionManagementModal();
        }
    });
}
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
        if (!checkbox || checkbox.type !== 'checkbox') {
            return;
        }

        const serverKey = String(checkbox.dataset.mcpKey || '').trim();
        if (!serverKey) {
            return;
        }

        const enabled = checkbox.checked;
        SetAIMcpServerEnabled(serverKey, enabled).then(() => {
            void loadAISessionManagement().then(() => {
                renderAISessionManagement();
            });
        }).catch((err) => {
            notifyTerminal(`Failed to ${enabled ? 'load' : 'unload'} MCP server`, 'error');
            console.error(err);
            checkbox.checked = !enabled;
        });
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
            const serverKey = String(checkbox.dataset.mcpKey || '').trim();
            if (!serverKey) {
                return;
            }

            const enabled = checkbox.checked;
            SetAIMcpServerEnabled(serverKey, enabled).then(() => {
                void loadAISessionManagement().then(() => {
                    renderAISessionManagement();
                });
            }).catch((err) => {
                notifyTerminal(`Failed to ${enabled ? 'load' : 'unload'} MCP server`, 'error');
                console.error(err);
                checkbox.checked = !enabled;
            });
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

initNotesAIPanel(elements);
initAIPromptJumpObserver();
void refreshAIPromptJumpFromBackend();
if (elements.toolsToC) {
    elements.toolsToC.addEventListener('click', (event) => {
        const tocButton = event.target.closest('.notes-tools-toc-item');
        if (!tocButton) {
            return;
        }

        if (state.viewMode !== 'viewer' && state.viewMode !== 'jupyter') {
            return;
        }

        scrollToToolsToCHeading(tocButton);
    });
}
if (elements.previewWrap) {
    elements.previewWrap.addEventListener('scroll', () => {
        syncToolsToCHighlightForMode();
    });
}
if (elements.jupyterWrap) {
    elements.jupyterWrap.addEventListener('scroll', () => {
        syncToolsToCHighlightForMode();
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
        ${getMarkdownBaseTextSizeStyles('#notes-preview', result.fontSize)}
        ${getMarkdownBaseTextSizeStyles('#notes-jupyter', result.fontSize)}
        ${getMarkdownBaseTextSizeStyles('#notes-meta', result.fontSize)}
        ${getMarkdownBaseTextSizeStyles('#notes-csv-view', result.fontSize)}
        ${getMarkdownBaseTextSizeStyles('#notes-swagger-info', result.fontSize)}
        ${getMarkdownBaseTextSizeStyles('#notes-swagger-run-wrap', result.fontSize)}
        ${getMarkdownBaseTextSizeStyles('#notes-swagger-request-builder .swagger-param-description', result.fontSize)}
        ${getMarkdownContentStyles(result.colors, result.fontSize, 'markdown-body')}
        ${getMarkdownContentStyles(result.colors, result.fontSize, 'swagger-ui')}
        ${getCheckboxStyles(result.colors, result.fontSize, 'markdown-body')}
        ${getHighlightJsTheme(result.colors, true)}
        ${getHexDumpStyles(result.fontSize, result.adjustCellHeight)}
        ${getSwaggerUIStyles(result.colors, result.fontSize)}
    `;

    if (isMonacoActive()) {
        monacoMainEditor.setTypography(getMonacoTypographyOptions());
        monacoMainEditor.applyTheme(result.colors);
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
// LSP Diagnostics
// ----------------------------------------------------------------------------

// Keyed by file URI → array of Diagnostic objects from the server.
const lspDiagnosticsStore = new Map();
const typosDiagnosticsStore = new Map();

// Tooltip element — created once and re-positioned on hover.
const lspTooltipEl = (() => {
    const el = document.createElement('div');
    el.id = 'notes-lsp-tooltip';
    document.body.appendChild(el);
    return el;
})();

const lspHoverTooltipEl = (() => {
    const el = document.createElement('div');
    el.id = 'notes-lsp-hover-tooltip';
    el.className = 'markdown-body';
    el.addEventListener('mousedown', (e) => {
        if (e.button === 0) e.preventDefault(); // left click: keep editor focus
    });
    el.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') hideLspHoverTooltip();
    });
    document.body.appendChild(el);
    return el;
})();

const lspCompletionEl = (() => {
    const el = document.createElement('div');
    el.id = 'notes-lsp-completion';
    document.body.appendChild(el);
    return el;
})();

function hideHyperlinkHoverTooltip() {
    closeSharedTooltip();
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
    showSharedTooltip(displayHref);
}

const LSP_SEVERITY_CLASS = ['', 'error', 'warning', 'info', 'hint'];

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

function clearVisibleLspDiagnostics(options = {}) {
    const preserveCompletion = options && options.preserveCompletion === true;
    if (isMonacoActive()) {
        monacoMainEditor.setDiagnostics([]);
    }
    if (lspTooltipEl) {
        lspTooltipEl.style.display = 'none';
    }
    hideLspHoverTooltip();
    if (!preserveCompletion) {
        hideLspCompletion();
    }
}

function clearCurrentFileLspDiagnosticsCache() {
    if (!state.currentFileUri) {
        return;
    }
    lspDiagnosticsStore.delete(state.currentFileUri);
}

function renderLspDiagnostics() {
    if (!isMonacoActive()) {
        return;
    }

    // Collect diagnostics for the currently open file.
    const currentUri = state.currentFileUri || null;

    const diags = currentUri ? (lspDiagnosticsStore.get(currentUri) || []) : [];
    monacoMainEditor.setDiagnostics(diags);
}

function renderLspEditorDecorations() {
    renderLspDiagnostics();
    renderLspInlayHints();
}

function renderLspInlayHints() {
    // Inlay hints are rendered by Monaco providers in Monaco-only mode.
}

document.addEventListener('scroll', () => {
    hideHyperlinkHoverTooltip();
}, true);

EventsOn('notesLspDiagnostics', payload => {
    const data = Array.isArray(payload?.[0]) ? payload[0] : payload;
    if (!data || typeof data.uri !== 'string') return;

    lspDiagnosticsStore.set(data.uri, Array.isArray(data.diagnostics) ? data.diagnostics : []);
    if (data.uri === state.currentFileUri) {
        // While the user is actively typing, avoid repainting diagnostics from an
        // older editor snapshot; wait for a short idle window.
        if (state.lspChangeTimer || (Date.now() - state.lastEditorInputAt) < LSP_DIAGNOSTIC_RENDER_IDLE_MS) {
            return;
        }
        renderLspEditorDecorations();
    }
});

EventsOn('notesTyposDiagnostics', payload => {
    const data = Array.isArray(payload?.[0]) ? payload[0] : payload;
    if (!data || typeof data.uri !== 'string') return;
    typosDiagnosticsStore.set(data.uri, Array.isArray(data.diagnostics) ? data.diagnostics : []);
    routeTyposDiagnostics(data.uri, Array.isArray(data.diagnostics) ? data.diagnostics : []);
});

function insertEditorText(text, target = elements.editor) {
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
        const markdownImage = `![${alt}](${paths.imageFileName})`;
        const start = elements.editor.selectionStart;
        const end = elements.editor.selectionEnd;
        const value = elements.editor.value;

        elements.editor.value = value.slice(0, start) + markdownImage + value.slice(end);
        elements.editor.selectionStart = start + markdownImage.length;
        elements.editor.selectionEnd = start + markdownImage.length;

        setDirty(true);
        scheduleRender();
        scheduleAutoSave();
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

async function pasteFromGoClipboard(targetEditor = elements.editor, allowImagePaste = true) {
    try {
        const payload = await GetClipboardData();
        const { text, image } = decodeClipboardPayload(payload);

        if (allowImagePaste && image !== '') {
            const dataUrl = `data:image/png;base64,${image}`;
            await savePastedImageDataUrl(dataUrl, 'image/png');
            return;
        }

        if (text !== '') {
            if (isMonacoActive() && targetEditor === elements.editor) {
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

// Three independent exclusion sources — always merged before being applied.
// Separating them prevents one source from clobbering another when they
// arrive at different times.
const lspSpellCheckExclusions = { symbols: [], tokens: [], keywords: [] };

function applyLspSpellCheckExclusions() {
    const merged = Array.from(getLspSpellCheckExclusionSet());
    notesSpellCheckHandle?.setExclusions(merged);
}

if (elements.editor) {
    let editorInputSequence = 0;

    function resolveUndoRedoDirection(event) {
        const key = String(event?.key || '').toLowerCase();
        const hasPrimaryModifier = Boolean(event?.metaKey || event?.ctrlKey);
        if (!hasPrimaryModifier || event?.altKey) {
            return '';
        }

        if (key === 'z') {
            return event.shiftKey ? 'redo' : 'undo';
        }

        if (key === 'y') {
            return 'redo';
        }

        return '';
    }

    function maybeHandleManagedUndoRedoShortcut(event) {
        if (!event || document.activeElement !== elements.editor) {
            return false;
        }

        const direction = resolveUndoRedoDirection(event);
        if (!direction) {
            return false;
        }

        const canApply = direction === 'undo'
            ? notesUndoManager.canUndo()
            : notesUndoManager.canRedo();
        if (!canApply) {
            return false;
        }

        event.preventDefault();
        event.stopPropagation();

        const applySnapshot = (tx, txDirection) => {
            notesMutationAdapter.applySnapshot(elements.editor, tx, txDirection, true);
        };

        if (direction === 'undo') {
            notesUndoManager.undo(applySnapshot);
            return true;
        }

        notesUndoManager.redo(applySnapshot);
        return true;
    }

    function maybeDispatchInputFallbackAfterShortcut(event) {
        const isModifierShortcut = (event.ctrlKey || event.metaKey) && !event.altKey;
        if (!isModifierShortcut) {
            return;
        }

        const key = String(event.key || '').toLowerCase();
        if (key !== 'z' && key !== 'v') {
            return;
        }

        if (document.activeElement !== elements.editor) {
            return;
        }

        const valueBefore = elements.editor.value;
        const seqBefore = editorInputSequence;

        setTimeout(() => {
            // Some WebKit edit paths can mutate textarea value without emitting `input`.
            // Re-emit `input` only when content changed and no native input was observed.
            if (editorInputSequence !== seqBefore) {
                return;
            }
            if (elements.editor.value === valueBefore) {
                return;
            }
            elements.editor.dispatchEvent(new Event('input', { bubbles: true }));
        }, 0);
    }

    elements.editor.addEventListener('keydown', async (event) => {
        if (maybeHandleManagedUndoRedoShortcut(event)) {
            return;
        }

        maybeDispatchInputFallbackAfterShortcut(event);

        if (event.key === 'Escape' && closeOpenLspTooltips()) {
            event.preventDefault();
            event.stopPropagation();
            return;
        }

        if (!event.ctrlKey && !event.metaKey && !event.altKey) {
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

        if (maybeHandleSyntaxCompletionKey(event, elements.editor, {
            docPath: state.currentFile || '',
            languageHint: state.editorLanguage || '',
        })) {
            return;
        }

        if (event.key !== 'Tab' || event.ctrlKey || event.metaKey || event.altKey) {
            return;
        }

        if (state.lspCompletionVisible) {
            // When completion is already open, Tab dismisses it and inserts indentation.
            hideLspCompletion();
            event.preventDefault();
            event.stopPropagation();

            const start = elements.editor.selectionStart;
            const end = elements.editor.selectionEnd;
            const indentation = await getIndentationString();
            notesMutationAdapter.replaceRange(elements.editor, {
                start,
                end,
                text: indentation,
                source: 'editor-tab-indent',
                label: 'Tab indentation (completion open)',
                emit: true,
            });
            return;
        }

        const source = elements.editor.value || '';
        const cursor = elements.editor.selectionStart || 0;
        const lineStart = source.lastIndexOf('\n', Math.max(0, cursor - 1)) + 1;
        const leftOfCaret = source.slice(lineStart, cursor);
        const isWhitespaceOnlyLeft = /^\s*$/.test(leftOfCaret);
        const shouldTryLspCompletion = isCurrentFileLspEligible()
            && state.currentFile
            && state.lspOpenFile === state.currentFile
            && !isWhitespaceOnlyLeft;

        if (shouldTryLspCompletion) {
            // With non-whitespace prefix on this line, use Tab to open completion rather than insert indentation.
            event.preventDefault();
            event.stopPropagation();
            hideLspHoverTooltip();
            hideLspCompletion();
            void requestLspCompletion();
            return;
        }

        // Keep Tab inside the editor so it doesn't trigger app-level focus hotkeys.
        event.preventDefault();
        event.stopPropagation();

        const start = elements.editor.selectionStart;
        const end = elements.editor.selectionEnd;
        const indentation = await getIndentationString();
        notesMutationAdapter.replaceRange(elements.editor, {
            start,
            end,
            text: indentation,
            source: 'editor-tab-indent',
            label: 'Tab indentation',
            emit: true,
        });
    });

    elements.editor.addEventListener('input', () => {
        if (isMonacoActive() && !suppressMonacoChange) {
            const textareaValue = String(elements.editor.value || '');
            if (monacoMainEditor.getValue() !== textareaValue) {
                suppressMonacoChange = true;
                monacoMainEditor.setValue(textareaValue);
                suppressMonacoChange = false;
            }
        }

        editorInputSequence += 1;
        state.lastEditorInputAt = Date.now();
        setDirty(true);
        state.lspHoverLastKey = '';
        hideLspHoverTooltip();
        state.lspInlayRequestId += 1;
        state.lspInlayHints = [];
        clearCurrentFileLspDiagnosticsCache();
        clearVisibleLspDiagnostics({ preserveCompletion: true });

        if (!isMonacoActive()) {
            const cursor = elements.editor.selectionStart || 0;
            const value = elements.editor.value || '';
            const prevChar = cursor > 0 ? value[cursor - 1] : '';

            // If LSP completion menu is visible, keep it open and re-filter based on current position.
            // Otherwise, check for trigger characters to open it.
            if (state.lspCompletionVisible) {
                // Menu is open - sync first, then re-request completion to filter by current text.
                void requestLspCompletionAfterSync(value, '', 1);
            } else {
                // Menu is closed - check for trigger characters
                if (prevChar === '.' || prevChar === ':' || prevChar === '>') {
                    void requestLspCompletionAfterSync(value, prevChar);
                } else {
                    // Don't close the menu if user types an identifier character (keeps menu open while typing to filter)
                    const isIdentifierChar = /[A-Za-z0-9_-]/.test(prevChar);
                    if (!isIdentifierChar) {
                        hideLspCompletion();
                    }
                }
            }
        } else {
            const cursor = elements.editor.selectionStart || 0;
            const value = elements.editor.value || '';
            const prevChar = cursor > 0 ? value[cursor - 1] : '';

            if (state.lspCompletionVisible) {
                void requestLspCompletionAfterSync(value, '', 1);
            } else if (prevChar === '.' || prevChar === ':' || prevChar === '>') {
                void requestLspCompletionAfterSync(value, prevChar);
            } else {
                const isIdentifierChar = /[A-Za-z0-9_-]/.test(prevChar);
                if (!isIdentifierChar) {
                    hideLspCompletion();
                }
            }
        }

        if (isMonacoActive()) {
            // In Monaco (Edit) mode, defer view regeneration until the View tab is
            // clicked.  This avoids expensive re-renders on every keystroke /
            // autosave cycle, which is especially noticeable for JSON/YAML files.
            state.viewTainted = true;
        } else if (usesCodeEditorDecorations()) {
            refreshEditorLanguage(state.currentFile, elements.editor.value);
            if (state.currentFileType === 'markdown' || state.currentFileType === 'html') {
                scheduleRender();
            }
        } else if (state.currentFileType === 'csv') {
            renderCsvView(elements.editor.value, { interactive: state.viewMode === 'csv-run' });
        } else {
            scheduleRender();
        }

        if (state.currentFileType === 'json') {
            // Revalidate JSON/YAML and only expose Run for docs with a swagger key.
            // The tree view render itself is deferred to the View tab click.
            state.swaggerSpec = parseSwaggerSpec(elements.editor.value);
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
    });

    elements.editor.addEventListener('scroll', () => {
        hideLspHoverTooltip();
        hideLspCompletion();
        if (usesCodeEditorDecorations()) {
            syncEditorScrollDecorations();
        }
    });

    elements.editor.addEventListener('mousemove', (event) => {
        state.lspHoverMouseX = event.clientX;
        state.lspHoverMouseY = event.clientY;
    });

    elements.editor.addEventListener('mouseup', () => {
        state.lspHoverLastKey = '';
        scheduleLspHover();
    });

    elements.editor.addEventListener('keyup', () => {
        scheduleLspHover();
    });

    elements.editor.addEventListener('blur', () => {
        hideLspHoverTooltip();
        hideLspCompletion();
    });

    elements.editor.addEventListener('paste', (event) => {
        handleEditorImagePaste(event);
    });

    attachVimMode(elements.editor, {
        mutationAdapter: notesMutationAdapter,
        undoManager: notesUndoManager,
        filePathResolver: () => state.currentFile || '',
    });
    notesSpellCheckHandle = attachSpellCheck(elements.editor, {
        onMisspellingsChange: (misspellings) => {
            if (!isMonacoActive()) {
                return;
            }
            monacoMainEditor.setTyposMisspellings(Array.isArray(misspellings) ? misspellings : []);
        },
    });
}

let _editorSelectionBeforeContextMenu = null;

elements.editor.addEventListener('mousedown', (e) => {
    if (e.button === 2) {
        _editorSelectionBeforeContextMenu = {
            start: elements.editor.selectionStart,
            end: elements.editor.selectionEnd,
        };
    }
});

function openMainEditorContextMenu(e) {
    // Restore selection that WebKit changed on right-click
    if (_editorSelectionBeforeContextMenu !== null) {
        elements.editor.selectionStart = _editorSelectionBeforeContextMenu.start;
        elements.editor.selectionEnd = _editorSelectionBeforeContextMenu.end;
        _editorSelectionBeforeContextMenu = null;
    }
    e.preventDefault();

    const menuItems = [
        createCopyMenuItem(() => getMainEditorSelectionText(), 'Copy'),
        {
            title: 'Paste',
            icon: CONTEXT_ICON_PASTE,
            onSelect: async () => {
                await pasteFromGoClipboard(elements.editor, !isStructuredDataFile(state.currentFile));
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
                        monacoMainEditor?.openCommandPalette?.();
                    }, 0);
                },
        });
    }

    showNotesLocalMenu(menuItems, e.clientX, e.clientY);
}

elements.editor.addEventListener('contextmenu', (e) => {
    openMainEditorContextMenu(e);
});

initRenderedNotesContextMenu(elements.preview, 'viewer');
initRenderedNotesContextMenu(elements.jupyter, 'jupyter');
initRenderedNotesContextMenu(elements.swaggerRunWrap, 'swagger-run');
initRenderedNotesContextMenu(lspHoverTooltipEl, 'viewer');
initRenderedNotesContextMenu(lspTooltipEl, 'viewer');
initAIOutputContextMenu(elements.aiOutput);

elements.csvView.addEventListener('contextmenu', (e) => {
    e.preventDefault();
    const table = e.target instanceof Element ? e.target.closest('table') : null;
    if (!table || !elements.csvView.contains(table)) return;
    const menuItems = [...createTableCopyMenuItems(table)];
    const isRunMode = state.viewMode === 'csv-run';
    if (isRunMode) {
        const insertItems = createTableInsertMenuItems(table, e.target, 0);
        if (insertItems.length > 0) {
            menuItems.push({ title: '-' }, ...insertItems);
        }
    }

    menuItems.push({ title: '-' }, createAskAIDocumentMenuItem());

    let highlightCallback = null;
    let cancelCallback = null;
    if (isRunMode) {
        const pos = getCellPosition(e.target, table);
        if (pos) {
            highlightCallback = (itemIndex) => {
                const item = menuItems[itemIndex];
                if (!item) return;
                // Unhighlight all first
                clearTableHighlight(table);
                // Highlight based on item title
                if (item.title.toLowerCase().includes('copy table')) {
                    highlightEntireTable(table, true);
                } else if (item.title.includes('row') && !item.title.includes('column')) {
                    highlightTableRow(table, pos.row, true);
                } else if (item.title.includes('column')) {
                    highlightTableColumn(table, pos.col, true);
                }
            };
            cancelCallback = () => clearTableHighlight(table);
        }
    } else {
        // Enable highlight for copy table items even when not in Run mode
        highlightCallback = (itemIndex) => {
            const item = menuItems[itemIndex];
            if (!item) return;
            clearTableHighlight(table);
            if (item.title.toLowerCase().includes('copy table')) {
                highlightEntireTable(table, true);
            }
        };
        cancelCallback = () => clearTableHighlight(table);
    }

    showNotesLocalMenu(menuItems, e.clientX, e.clientY, 'Select an action', highlightCallback, cancelCallback);
});
initStructuredDataTreeContextMenu(elements.swaggerView, {
    isActive: () => state.viewMode === 'swagger-view',
    getRoot: () => state.swaggerSpec ?? parseSwaggerSpec(elements.editor.value),
    menuTitle: 'JSON/YAML field',
});
initStructuredDataTreeContextMenu(elements.toolsFrontmatter, {
    isActive: () => state.currentFileType === 'markdown' && state.frontmatter != null,
    getRoot: () => state.frontmatter,
    menuTitle: 'Frontmatter field',
});

elements.tabEditor.addEventListener('click', () => {
    setViewMode('editor');
});

elements.tabHex.addEventListener('click', () => {
    setViewMode('hex');
});

elements.tabViewer.addEventListener('click', () => {
    setViewMode('viewer');
    if (state.viewTainted && state.currentFileType === 'markdown') {
        state.viewTainted = false;
        scheduleRender();
    }
});

elements.tabJupyter.addEventListener('click', () => {
    setViewMode('jupyter');
    renderJupyterView();
});

elements.tabSwaggerView.addEventListener('click', () => {
    setViewMode('swagger-view');
    state.viewTainted = false;
    renderSwaggerJsonViewLazy();
});

elements.tabSwaggerEdit.addEventListener('click', () => {
    setViewMode('swagger-edit');
});

elements.tabSwaggerRun.addEventListener('click', () => {
    setViewMode('swagger-run');
    updateSwaggerLayoutMode();
    renderSwaggerUI();
});

elements.tabImageView.addEventListener('click', () => {
    setViewMode('image-view');
});

elements.tabCsvView.addEventListener('click', () => {
    setViewMode('csv-view');
});

elements.tabCsvEdit.addEventListener('click', () => {
    setViewMode('csv-edit');
});

elements.tabCsvRun.addEventListener('click', () => {
    setViewMode('csv-run');
});

elements.tabMeta.addEventListener('click', () => {
    setViewMode('meta');
});

if (elements.status) {
    elements.status.title = 'Recent files';
    elements.status.tabIndex = 0;
    elements.status.setAttribute('role', 'button');

    const openRecentFilesMenu = async (x, y) => {
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
                    NotesHistoryAdd(file).catch(() => {});
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
    };

    elements.status.addEventListener('click', (event) => {
        event.preventDefault();
        openRecentFilesMenu(event.clientX, event.clientY);
    });

    elements.status.addEventListener('keydown', (event) => {
        if (event.key !== 'Enter' && event.key !== ' ') {
            return;
        }

        event.preventDefault();
        const rect = elements.status.getBoundingClientRect();
        openRecentFilesMenu(rect.left, rect.bottom);
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
        return [elements.tabEditor, elements.tabHex, elements.tabMeta].filter(Boolean);
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
        const file = await NotesHistoryPrevious();
        await loadFile(file);
    } catch (err) {
        notifyTerminal(String(err && err.message ? err.message : err), 'info');
        focusActiveEditorForViewMode();
    }
});

elements.historyNext.addEventListener('click', async () => {
    try {
        const file = await NotesHistoryNext();
        await loadFile(file);
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
        updateFindFilesClearButtonVisibility();
        scheduleProjectFindSearch();
    });

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
            if (isProjectFindListModeActive()) {
                scheduleProjectFindSearch();
                return;
            }
            applyFileFilter();
            return;
        }

        if (event.key === 'ArrowDown') {
            event.preventDefault();
            const items = Array.from(elements.list.querySelectorAll('.notes-file'));
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
    elements.list.addEventListener('scroll', () => {
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
        const fileItem = event.target.closest('.notes-file');
        if (fileItem) {
            NotesHistoryAdd(fileItem.dataset.file).catch(() => {});
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

    elements.list.addEventListener('contextmenu', async (event) => {
        const fileItem = event.target.closest('.notes-file');
        if (fileItem) {
            event.preventDefault();
            event.stopPropagation();
            await openFileListContextMenu(fileItem.dataset.file, event.clientX, event.clientY);
            return;
        }

        const folderButton = event.target.closest('.notes-tree-folder');
        if (folderButton) {
            event.preventDefault();
            event.stopPropagation();
            const folderKey = folderButton.dataset.folderKey || '';
            const separatorIndex = folderKey.indexOf(PRIMARY_PATH_SEPARATOR);
            const category = separatorIndex === -1 ? folderKey : folderKey.slice(0, separatorIndex);
            const folderPath = separatorIndex === -1 ? '' : folderKey.slice(separatorIndex + 1);
            const folderNode = findFolderNodeByPath(getCategoryTreeNodes(category), folderPath);
            const folderName = folderNode ? folderNode.name : folderPath;
            openFolderTreeContextMenu(category, folderNode?.children || [], event.clientX, event.clientY, folderName);
            return;
        }

        const categoryHeader = event.target.closest('.notes-category-header');
        if (categoryHeader) {
            event.preventDefault();
            event.stopPropagation();
            const category = categoryHeader.dataset.category;
            openFolderTreeContextMenu(category, getCategoryTreeNodes(category), event.clientX, event.clientY, `${category} folders`);
        }
    });

if (elements.listFilterClear && elements.listFilter) {
    elements.listFilterClear.addEventListener('click', () => {
        elements.listFilter.value = '';
        state.fileFilterQuery = '';
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
    let manualSplitRatio = 0.33;

    function clampLeftWidth(totalWidth, leftWidth) {
        const maxWidth = totalWidth - minPaneWidth - splitterWidth;
        return Math.min(Math.max(leftWidth, minPaneWidth), maxWidth);
    }

    function applyManualSplitToCurrentWidth() {
        if (!hasManualSplit) {
            return;
        }

        const appRect = app.getBoundingClientRect();
        if (appRect.width <= splitterWidth + (minPaneWidth * 2)) {
            return;
        }

        const availableWidth = appRect.width - splitterWidth;
        const desiredLeftWidth = availableWidth * manualSplitRatio;
        const leftWidth = clampLeftWidth(appRect.width, desiredLeftWidth);
        const rightWidth = appRect.width - leftWidth - splitterWidth;

        app.style.gridTemplateColumns = `${leftWidth}px ${splitterWidth}px ${rightWidth}px`;
        manualSplitRatio = leftWidth / availableWidth;
    }

    splitter.addEventListener('mousedown', (e) => {
        e.preventDefault();
        isResizing = true;
        document.body.style.cursor = 'col-resize';
        document.body.style.userSelect = 'none';
    });

    document.addEventListener('mousemove', (e) => {
        if (!isResizing) return;

        const appRect = app.getBoundingClientRect();
        const newLeftWidth = e.clientX - appRect.left;
        const minWidth = minPaneWidth;
        const maxWidth = appRect.width - minPaneWidth - splitterWidth;

        if (newLeftWidth > minWidth && newLeftWidth < maxWidth) {
            const rightWidth = appRect.width - newLeftWidth - splitterWidth;
            app.style.gridTemplateColumns = `${newLeftWidth}px ${splitterWidth}px ${rightWidth}px`;
            hasManualSplit = true;
            manualSplitRatio = newLeftWidth / (appRect.width - splitterWidth);
        }
    });

    document.addEventListener('mouseup', () => {
        if (isResizing) {
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
})();

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
        saveAndFormat();
        return;
    }

    if ((event.metaKey || event.ctrlKey) && !event.altKey && event.key.toLowerCase() === 'f') {
        event.preventDefault();
        const findTabActive = elements.toolsTabFind?.getAttribute('aria-selected') === 'true';
        const panelVisible = elements.toolsPanel?.dataset.collapsed !== 'true';
        if (findTabActive && panelVisible) {
            setToolsPanelCollapsed(true);
        } else {
            openFindBar();
        }
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

// Load initial notes list on startup; later notesUpdate events will refresh it.
refreshFiles();

if (typeof ResizeObserver !== 'undefined' && elements.swaggerRunWrap) {
    const swaggerPaneResizeObserver = new ResizeObserver(() => {
        updateSwaggerLayoutMode();
    });
    swaggerPaneResizeObserver.observe(elements.swaggerRunWrap);
} else {
    window.addEventListener('resize', () => {
        updateSwaggerLayoutMode();
    });
}
