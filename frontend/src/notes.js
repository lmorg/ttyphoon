import {
    GetWindowStyle, GetFile, GetImage,
    ListFiles, SaveFile, SaveBinaryFile, DeleteFile, RenameFile,
    RunNote, RunFunction, StopNote, SendIpc, SendToTerminal,
    GetLanguageDescriptions, GetAllLanguageDescriptions, TerminalCopyImageDataURL,
    ResolveFilePath, GetHyperlinkMenuActions, RunHyperlinkMenuAction,
    DisplayHyperlinkMenu,
    SaveImageDialog, WindowPrint, GetClipboardData, SwaggerRequest, NotesKeyPress,
    ShowCommandPalette, GetCurrentProject, GetFileMetaMarkdown,
} from '../wailsjs/go/main/WApp';
import { EventsOn, ClipboardSetText } from '../wailsjs/runtime/runtime';

import { showLocalMenu } from './popup_menu';

import { marked } from "marked";
import hljs from "highlight.js/lib/common";
import YAML from 'yaml';

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

import { configureMarked, processMarkdownContainer, enableFullscreenImages } from './markdown-utils.js';
import { getScrollbarStyles, getMarkdownContentStyles, getHighlightJsTheme, getCheckboxStyles, getMarkdownBaseTextSizeStyles, getSwaggerUIStyles, DARKEN_BACKGROUND_OVERLAY } from './style-utils.js';
import { 
    isStructuredDataFile, hasSwaggerKey, parseSwaggerSpec, generateRequestBuilderHTML, generateResponseHTML,
    extractPaths, generateEndpointListHTML, buildRequestUrl, generateLiveResponseHTML, escapeInfoText
} from './swagger-utils.js';
import { attachJsonViewerEditHandler, renderJsonViewer } from './json-viewer.js';
import { getHexDumpStyles, renderHexDump } from './hex-viewer.js';
import {
    evaluateTableFormula,
    isTableFormula,
    getCellReference,
    parseTableFunctionCall,
    resolveTableFunctionArg,
    resolveTableFunctionArgs,
    resolveTableFunctionArgsAsync,
} from './table-expressions.js';

const CONTEXT_ICON_COPY = 0xf0c5;
const CONTEXT_ICON_PASTE = 0xf0ea;
const CONTEXT_ICON_FIND = 0xf002;
const CONTEXT_ICON_PRINT = 0xf02f;
const CONTEXT_ICON_CHECKBOX = 0xf14a;
const CONTEXT_ICON_CODE = 0xf121;
const CONTEXT_ICON_TABLE = 0xf0ce;
const CONTEXT_ICON_EDIT = 0xf044;
const CONTEXT_ICON_DELETE = 0xf2ed;

// Inject cell reference CSS if not present
function ensureCellRefStyle() {
    if (document.getElementById('notes-cellref-style')) return;
    const style = document.createElement('style');
    style.id = 'notes-cellref-style';
    style.textContent = `
    .notes-cellref {
        position: absolute;
        right: 0;
        bottom: 0;
        opacity: 0.2;
        font-size: calc(var(--notes-table-font-size, 1em) - 2px);
        color: currentColor;
        pointer-events: none;
        z-index: 1;
        line-height: 1;
        user-select: none;
        text-align: right;
    }
    .notes-table-cell-wrap {
        position: relative;
        display: block;
        width: 100%;
        height: 100%;
    }
    .notes-table-cell-wrap > span:first-child {
        font-family: "Lato", var(--font-family), sans-serif !important;
        font-size: inherit !important;
        font-weight: inherit !important;
        letter-spacing: 1px !important;
    }
    td .notes-cellref, th .notes-cellref {
        /* ensure always visible */
    }
    .notes-sort-icon {
        display: inline-block;
        font-family: "Font Awesome Solid";
        font-weight: 900;
        margin-right: 6px;
        color: var(--red);
        vertical-align: middle;
        font-style: normal;
        pointer-events: none;
        user-select: none;
    }
    thead th {
        cursor: pointer;
        user-select: none;
    }
    `;
    document.head.appendChild(style);
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

app.innerHTML = `
    <div id="notes-app">
        <aside id="notes-sidebar">
            <div id="notes-sidebar-header">
                <div id="notes-title">Notes</div>
                <div id="notes-list-filter-wrap">
                    <input id="notes-list-filter" type="text" placeholder="Filter files..." autocomplete="off" autocorrect="off" autocapitalize="off" />
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
                    <button id="notes-rename" type="button" class="notes-toolbar-btn" title="Rename" aria-label="Rename current note">&#xf044;</button>
                    <button id="notes-delete" type="button" class="notes-toolbar-btn" title="Delete" aria-label="Delete current note">&#xf2ed;</button>
                    <button id="notes-find" type="button" class="notes-toolbar-btn" title="Find" aria-label="Find">&#xf002;</button>
                </div>
            </div>
            <div id="notes-panel">
                <div id="notes-editor-wrap" role="tabpanel">
                    <div id="notes-editor-shell" data-code-view="false">
                        <div id="notes-editor-gutter-wrap" aria-hidden="true">
                            <div id="notes-editor-gutter"></div>
                        </div>
                        <div id="notes-editor-scroll">
                            <pre id="notes-editor-highlight" aria-hidden="true"><code id="notes-editor-highlight-code" class="hljs"></code></pre>
                            <textarea id="notes-editor" autocorrect="off" autocapitalize="off" autocomplete="off" spellcheck="false" data-gramm="false" data-gramm_editor="false" data-enable-grammarly="false"></textarea>
                        </div>
                    </div>
                </div>
                <div id="notes-hex-wrap" role="tabpanel">
                    <div id="notes-hex"></div>
                </div>
                <div id="notes-preview-wrap" class="markdown-body" role="tabpanel">
                    <div id="notes-preview"></div>
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
                <div id="notes-ai-panel" class="notes-ai-panel" data-collapsed="true">
                    <div class="notes-ai-header">
                        <button id="notes-ai-toggle" type="button" class="notes-ai-toggle" title="Toggle AI panel">AI ▾</button>
                        <button id="notes-ai-clear" type="button" class="notes-ai-clear" title="Clear AI output">Clear</button>
                    </div>
                    <div id="notes-ai-output" class="notes-ai-output"></div>
                </div>
                <button id="notes-ai-restore" type="button" class="notes-ai-restore" title="Show AI panel">AI</button>
            </div>
        </main>
    </div>
    <div id="notes-modal" data-open="false" aria-hidden="true">
        <div id="notes-modal-card" role="dialog" aria-modal="true" aria-labelledby="notes-modal-title">
            <div id="notes-modal-title">New note name</div>
            <input id="notes-modal-input" type="text" placeholder="example-note" autocomplete="off" />
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
    <div id="notes-find-bar" data-open="false" aria-hidden="true">
        <input id="notes-find-input" type="text" placeholder="Find..." autocomplete="off" />
        <span id="notes-find-counter"></span>
        <button id="notes-find-prev" type="button" title="Previous match" tabindex="-1">↑</button>
        <button id="notes-find-next" type="button" title="Next match" tabindex="-1">↓</button>
        <button id="notes-find-close" type="button" title="Close find" tabindex="-1">✕</button>
    </div>
`;

const elements = {
    title: document.getElementById('notes-title'),
    list: document.getElementById('notes-list'),
    listFilter: document.getElementById('notes-list-filter'),
    listFilterClear: document.getElementById('notes-list-filter-clear'),
    editor: document.getElementById('notes-editor'),
    editorShell: document.getElementById('notes-editor-shell'),
    editorGutter: document.getElementById('notes-editor-gutter'),
    editorHighlight: document.getElementById('notes-editor-highlight'),
    editorHighlightCode: document.getElementById('notes-editor-highlight-code'),
    preview: document.getElementById('notes-preview'),
    jupyter: document.getElementById('notes-jupyter'),
    status: document.getElementById('notes-status'),
    newFile: document.getElementById('notes-new'),
    rename: document.getElementById('notes-rename'),
    delete: document.getElementById('notes-delete'),
    find: document.getElementById('notes-find'),
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
    modalInput: document.getElementById('notes-modal-input'),
    modalCancel: document.getElementById('notes-modal-cancel'),
    modalCreate: document.getElementById('notes-modal-create'),
    deleteModal: document.getElementById('notes-delete-modal'),
    deleteModalBody: document.getElementById('notes-delete-modal-body'),
    deleteCancel: document.getElementById('notes-delete-cancel'),
    deleteConfirm: document.getElementById('notes-delete-confirm'),
    findBar: document.getElementById('notes-find-bar'),
    findInput: document.getElementById('notes-find-input'),
    findCounter: document.getElementById('notes-find-counter'),
    findPrev: document.getElementById('notes-find-prev'),
    findNext: document.getElementById('notes-find-next'),
    findClose: document.getElementById('notes-find-close'),
    aiPanel: document.getElementById('notes-ai-panel'),
    aiToggle: document.getElementById('notes-ai-toggle'),
    aiClear: document.getElementById('notes-ai-clear'),
    aiOutput: document.getElementById('notes-ai-output'),
    aiRestore: document.getElementById('notes-ai-restore')
};

const state = {
    files: [],
    currentFile: '',
    currentFileProject: '',  // The project path when file was opened, prevents overwrites on project switch
    currentFileType: 'markdown',  // 'markdown' | 'json' | 'code' | 'image' | 'csv' | 'binary'
    dirty: false,
    renderTimer: null,
    autosaveTimer: null,
    viewMode: 'viewer',
    renamingFile: null,
    deletingFile: null,
    findMatches: [],
    findCurrentIndex: -1,
    findQuery: '',
    fileFilterQuery: '',
    expandedCategories: {
        '$GLOBAL': true,
        '$NOTES': true,
        '$PROJECT': true,
        '$HISTORY': false,
    },
    expandedFolders: {},
    jupyterCodeBlocks: {},
    jupyterBlockCounter: 0,
    swaggerSpec: null,
    swaggerRunAvailable: false,
    swaggerSelectedEndpoint: null,
    swaggerEndpointFilter: '',
    editorLanguage: '',
    fileMetaMarkdown: '',
    hexSourceType: '',
    hexSourceValue: '',
    hexSourceFile: '',
    hexSourceOptions: null,
    hexRenderedFile: '',
    hexLoadingPromise: null,
};

let lastAutoCopiedViewerSelection = '';

function activeViewerWrap() {
    return elements.previewWrap || null;
}

function getViewerSelectionText() {
    const viewer = activeViewerWrap();
    if (!viewer || typeof window.getSelection !== 'function') {
        return '';
    }

    const selection = window.getSelection();
    if (!selection || selection.rangeCount === 0) {
        return '';
    }

    const text = String(selection.toString() || '').trim();
    if (!text) {
        return '';
    }

    const anchorNode = selection.anchorNode;
    const focusNode = selection.focusNode;
    const anchorInViewer = anchorNode ? viewer.contains(anchorNode) : false;
    const focusInViewer = focusNode ? viewer.contains(focusNode) : false;

    return anchorInViewer || focusInViewer ? text : '';
}

function handleViewerSelectionAutoCopy() {
    const text = getViewerSelectionText();
    if (!text) {
        lastAutoCopiedViewerSelection = '';
        return;
    }

    if (text === lastAutoCopiedViewerSelection) {
        return;
    }

    lastAutoCopiedViewerSelection = text;
    ClipboardSetText(text).then(() => {
        notifyTerminal('Selection copied to clipboard', 'info');
    }).catch(() => {});
}

configureMarked();

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
        java: 'java',
        kt: 'kotlin',
        swift: 'swift',
        php: 'php',
        rb: 'ruby',
        sh: 'bash',
        bash: 'bash',
        zsh: 'bash',
        fish: 'bash',
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
        md: 'markdown',
        markdown: 'markdown',
        html: 'xml',
        xml: 'xml',
        plist: 'xml',
        manifest: 'xml',
        css: 'css',
        scss: 'scss',
        dockerfile: 'dockerfile',
        makefile: 'makefile',
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
    if (!elements.editor || !elements.editorGutter || !elements.editorHighlight) {
        return;
    }

    const scrollTop = elements.editor.scrollTop;
    const scrollLeft = elements.editor.scrollLeft;
    elements.editorGutter.style.transform = `translateY(${-scrollTop}px)`;

    // For wrapped markdown, keep vertical transform sync so overlay tracks touchpad/scrollbar movement.
    const isMarkdownWrapped = elements.editorShell?.dataset?.fileType === 'markdown';
    if (isMarkdownWrapped) {
        elements.editorHighlight.style.transform = `translate(0px, ${-scrollTop}px)`;
    } else {
        elements.editorHighlight.style.transform = `translate(${-scrollLeft}px, ${-scrollTop}px)`;
    }
}

function renderEditorDecorations() {
    if (!elements.editor || !elements.editorGutter || !elements.editorHighlightCode) {
        return;
    }

    const content = elements.editor.value || '';
    const lineCount = Math.max(1, content.split('\n').length);
    elements.editorGutter.textContent = Array.from({ length: lineCount }, (_, index) => String(index + 1)).join('\n');

    // Match highlight layer height to the full scrollable editor content.
    if (elements.editorHighlight) {
        const contentHeight = Math.max(elements.editor.scrollHeight, elements.editor.clientHeight);
        const contentWidth = Math.max(elements.editor.scrollWidth, elements.editor.clientWidth);
        elements.editorHighlight.style.minHeight = `${contentHeight}px`;
        elements.editorHighlight.style.minWidth = `${contentWidth}px`;
    }

    const language = state.editorLanguage || 'plaintext';
    try {
        if (hljs.getLanguage(language)) {
            elements.editorHighlightCode.innerHTML = hljs.highlight(content, { language, ignoreIllegals: true }).value;
        } else if (language === 'markdown') {
            // Avoid auto-detect for markdown; mis-detection often paints headings as comments.
            elements.editorHighlightCode.innerHTML = escapeEditorHtml(content);
        } else {
            elements.editorHighlightCode.innerHTML = hljs.highlightAuto(content).value;
        }
    } catch {
        elements.editorHighlightCode.innerHTML = escapeEditorHtml(content);
    }

    syncEditorScrollDecorations();
}

function refreshEditorLanguage(file, content) {
    state.editorLanguage = inferEditorLanguage(file, content);
    if (elements.editorHighlightCode) {
        elements.editorHighlightCode.className = `hljs language-${state.editorLanguage || 'plaintext'}`;
    }
    renderEditorDecorations();
}

function usesCodeEditorDecorations() {
    return state.currentFileType === 'code' || state.currentFileType === 'json' || state.currentFileType === 'markdown';
}

function isMarkdownNotesFile(fileName) {
    return /\.(md|markdown)$/i.test(String(fileName || ''));
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

    elements.editor.value = serializeCsvRows(rows);
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
    elements.editor.value = serializeCsvRows(rows);
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
    elements.editor.value = serializeCsvRows(rows);
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
    elements.editor.value = serializeCsvRows(rows);
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

    elements.editor.value = serializeCsvRows(rows);
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
                    selection.removeAllRanges();
                    selection.addRange(range);
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
}

function setCodeEditorMode(enabled) {
    if (!elements.editorShell) {
        return;
    }

    elements.editorShell.dataset.codeView = enabled ? 'true' : 'false';

    if (!enabled) {
        delete elements.editorShell.dataset.fileType;
        if (elements.editorGutter) {
            elements.editorGutter.textContent = '';
            elements.editorGutter.style.transform = '';
        }
        if (elements.editorHighlight) {
            elements.editorHighlight.style.transform = '';
            elements.editorHighlight.style.minHeight = '';
            elements.editorHighlight.style.minWidth = '';
        }
        if (elements.editorHighlightCode) {
            elements.editorHighlightCode.innerHTML = '';
        }
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
        const hasActiveFilter = state.fileFilterQuery.trim() !== '';
        const expanded = hasActiveFilter || state.expandedFolders[folderKey] !== false;
        folder.dataset.folderKey = folderKey;
        folder.dataset.expanded = expanded ? 'true' : 'false';
        folder.setAttribute('aria-expanded', expanded ? 'true' : 'false');

        folder.addEventListener('click', () => {
            toggleFolder(folderKey);
        });

        folder.addEventListener('contextmenu', (event) => {
            event.preventDefault();
            event.stopPropagation();
            openFolderTreeContextMenu(category, node.children || [], event.clientX, event.clientY, node.name);
        });
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

        item.addEventListener('click', () => {
            loadFile(node.file);
        });

        item.addEventListener('dblclick', (e) => {
            e.preventDefault();
            openRenamePrompt(node.file);
        });

	    item.addEventListener('contextmenu', async (e) => {
	        e.preventDefault();
	        e.stopPropagation();
	        await openFileListContextMenu(node.file, e.clientX, e.clientY);
	    });

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

function renderMarkdown() {
    const markdown = elements.editor.value || '';
    elements.preview.innerHTML = marked.parse(markdown);

    // Apply common markdown processing
    processMarkdownContainer(elements.preview);

    // Enable context menus on images
    enableImageContextMenus(elements.preview);

    // Keep checkboxes readonly in viewer mode
    setupInteractiveCheckboxes(elements.preview, false);

    // Enable collapsible H1-H6 sections
    setupCollapsibleHeadings(elements.preview);

    // Enable column sorting on all tables
    setupTableSorting(elements.preview);

    // Re-apply find highlights if find bar is open and in viewer mode
    if (elements.findBar.dataset.open === 'true' && state.findQuery && state.viewMode === 'viewer') {
        setTimeout(() => {
            performFind();
        }, 0);
    }
}

function renderMetaView() {
    const markdown = state.fileMetaMarkdown || '# Unknown file';
    elements.meta.innerHTML = marked.parse(markdown);
    processMarkdownContainer(elements.meta);
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

function setupCollapsibleHeadings(container) {
    const headings = container.querySelectorAll('h1, h2, h3, h4, h5, h6');

    headings.forEach((heading) => {
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
        wrapper.style.overflowX = 'auto';
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

        heading.addEventListener('click', () => {
            const isCollapsed = heading.dataset.collapsed === 'true';

            if (isCollapsed) {
                // Expand: animate from 0 → scrollHeight, then release max-height cap.
                wrapper.style.maxHeight = wrapper.scrollHeight + 'px';
                wrapper.style.opacity = '1';
                wrapper.addEventListener('transitionend', () => {
                    if (heading.dataset.collapsed !== 'true') {
                        wrapper.style.maxHeight = '100000px';
                    }
                }, { once: true });
                heading.dataset.collapsed = 'false';
                heading.style.fontStyle = '';
            } else {
                // Collapse: pin to exact current height (no jump), then animate to 0.
                const height = wrapper.scrollHeight;
                wrapper.style.transition = 'none';
                wrapper.style.maxHeight = height + 'px';
                wrapper.offsetHeight; // force reflow so the pin takes effect
                wrapper.style.transition = 'max-height 0.3s ease, opacity 0.3s ease';
                requestAnimationFrame(() => {
                    wrapper.style.maxHeight = '0px';
                    wrapper.style.opacity = '0';
                });
                heading.dataset.collapsed = 'true';
                heading.style.fontStyle = 'italic';
            }
        });
    });
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
        const result = await RunFunction(currentCellId, block.code, resolvedArgs, runtime);

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

function setupTableSorting(container) {
    if (!container) return;

    const getCellText = (cell) => {
        const wrap = cell.querySelector('.notes-table-cell-wrap > span:first-child');
        if (wrap) return String(wrap.textContent || '').trim();
        // Exclude sort icon from raw text comparison
        return Array.from(cell.childNodes)
            .filter(n => !n.classList?.contains('notes-sort-icon'))
            .map(n => n.textContent)
            .join('')
            .trim();
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
        elements.editor.value = lines.join('\n');
        saveFile();
        // Keep viewer in sync when changes are made from jupyter mode
        if (state.viewMode === 'jupyter') {
            renderMarkdown();
        }
        // Don't re-render jupyter here to avoid resetting checkbox focus
    }
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
    elements.editor.value = result;
    return true;
}

function scheduleRender() {
    if (state.renderTimer) {
        clearTimeout(state.renderTimer);
    }
    state.renderTimer = setTimeout(() => {
        state.renderTimer = null;
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

function setDirty(isDirty) {
    state.dirty = isDirty;
    const label = state.currentFile ? state.currentFile : 'No file selected';
    elements.status.textContent = isDirty ? `${label} (unsaved)` : label;
}

function focusActiveEditorForViewMode() {
    if (!elements.editor) {
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

        if (!stillShouldFocus || !elements.editor) {
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
    const isMeta = state.viewMode === 'meta';
    
    elements.tabEditor.setAttribute('aria-selected', isEditor ? 'true' : 'false');
    elements.tabHex.setAttribute('aria-selected', isHex ? 'true' : 'false');
    elements.tabViewer.setAttribute('aria-selected', isViewer ? 'true' : 'false');
    elements.tabJupyter.setAttribute('aria-selected', isJupyter ? 'true' : 'false');
    elements.tabMeta.setAttribute('aria-selected', isMeta ? 'true' : 'false');
    
    const isStructuredEdit = state.currentFileType === 'json' && state.viewMode === 'swagger-edit';
    elements.editorWrap.dataset.active = (isEditor || isStructuredEdit) ? 'true' : 'false';
    elements.hexWrap.dataset.active = isHex ? 'true' : 'false';
    elements.previewWrap.dataset.active = isViewer ? 'true' : 'false';
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

    if ((isEditor && usesCodeEditorDecorations()) || isStructuredEdit) {
        renderEditorDecorations();
    }

    if (isMeta) {
        renderMetaView();
    }

    updateFindAvailability();

    if (isHex) {
        void ensureHexDumpForCurrentFile();
    }
    
    // Re-perform find if find bar is open
    if (elements.findBar.dataset.open === 'true' && state.findQuery) {
        performFind();
    }

    focusActiveEditorForViewMode();
}

function renderJupyterView() {
    // Reset jupyter state for the new render
    state.jupyterCodeBlocks = {};
    state.jupyterBlockCounter = 0;
    
    const markdown = elements.editor.value || '';
    elements.jupyter.innerHTML = marked.parse(markdown);
    
    // Apply common markdown processing
    processMarkdownContainer(elements.jupyter);

    // Enable context menus on images
    enableImageContextMenus(elements.jupyter);
    
    // Enable checkbox editing and save behavior in jupyter mode
    setupInteractiveCheckboxes(elements.jupyter, true);

    // Enable collapsible H1-H6 sections
    setupCollapsibleHeadings(elements.jupyter);

    // Render code blocks immediately so they are not blocked by table evaluation.
    convertToJupyterCodeBlocks();

    evaluateTableFormulasInPlace(elements.jupyter)
        .catch((err) => {
            console.warn('Table formula evaluation failed:', err);
        })
        .finally(() => {
            setupInteractiveMarkdownTables(elements.jupyter, true);

            // Enable column sorting on all tables
            setupTableSorting(elements.jupyter);

            // Re-apply find highlights if find bar is open and in jupyter mode
            if (elements.findBar.dataset.open === 'true' && state.findQuery && state.viewMode === 'jupyter') {
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
            currentContent: content
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
        
        const runTerminalBtn = document.createElement('button');
        runTerminalBtn.type = 'button';
        runTerminalBtn.className = 'jupyter-btn jupyter-run-terminal';
        runTerminalBtn.textContent = 'Send to terminal';
        runTerminalBtn.addEventListener('click', () => runCodeBlockInTerminal(blockId));
        
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
        toolbar.appendChild(runTerminalBtn);
        toolbar.appendChild(runtimeLink);
        
        const editableCode = document.createElement('textarea');
        editableCode.className = 'jupyter-code-editable';
        editableCode.dataset.language = language;
        editableCode.value = content;
        editableCode.setAttribute('autocorrect', 'off');
        editableCode.setAttribute('autocapitalize', 'off');
        editableCode.setAttribute('autocomplete', 'off');
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
        editableCode.addEventListener('keydown', (event) => {
            if (event.key !== 'Tab' || event.ctrlKey || event.metaKey || event.altKey) {
                return;
            }

            // Insert tab character and keep focus in the code editor
            event.preventDefault();
            event.stopPropagation();

            const start = editableCode.selectionStart;
            const end = editableCode.selectionEnd;
            editableCode.setRangeText('\t', start, end, 'end');
            editableCode.dispatchEvent(new Event('input'));
        });
        editableCode.addEventListener('scroll', () => {
            syncHighlightViewport();
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
        await RunNote(blockId, block.currentContent, block.runtime);
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
        } catch (err) {
            console.error('Error sending to terminal:', err);
        }
}

async function refreshFiles() {
    try {
        const files = await ListFiles();
        state.files = Array.isArray(files) ? files : [];
        renderFileList();
    } catch (err) {
        setStatus('Failed to load file list.', true);
        console.error(err);
    }
}

function getFilteredFiles() {
    const query = state.fileFilterQuery.trim().toLowerCase();
    if (!query) {
        return state.files;
    }

    return state.files.filter((file) => {
        const normalizedFile = String(file || '').toLowerCase();
        const fileName = getPathFileName(file).toLowerCase();
        return normalizedFile.includes(query) || fileName.includes(query);
    });
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
    updateListFilterClearButtonVisibility();
    elements.list.innerHTML = '';

    const filteredFiles = getFilteredFiles();
    const hasActiveFilter = state.fileFilterQuery.trim() !== '';

    if (state.files.length === 0) {
        const empty = document.createElement('div');
        empty.id = 'notes-empty';
        empty.textContent = 'No notes found.';
        elements.list.appendChild(empty);
        return;
    }

    if (filteredFiles.length === 0) {
        const empty = document.createElement('div');
        empty.id = 'notes-empty';
        empty.textContent = 'No matching files.';
        elements.list.appendChild(empty);
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

        if (!hasActiveFilter) {
            categoryHeader.addEventListener('click', () => {
                toggleCategory(category);
            });
        }

        categoryHeader.addEventListener('contextmenu', (event) => {
            event.preventDefault();
            event.stopPropagation();
            openFolderTreeContextMenu(category, categoryTree, event.clientX, event.clientY, `${category} folders`);
        });
        
        elements.list.appendChild(categoryHeader);

        // Create category content container
        const categoryContent = document.createElement('div');
        categoryContent.className = 'notes-category-content';
        categoryContent.dataset.expanded = categoryExpanded ? 'true' : 'false';

        renderTreeNodesList(categoryContent, category, categoryTree);

        elements.list.appendChild(categoryContent);
    });
}

function toggleCategory(category) {
    state.expandedCategories[category] = !state.expandedCategories[category];
    renderFileList();
}

function toggleFolder(folderKey) {
    state.expandedFolders[folderKey] = !(state.expandedFolders[folderKey] !== false);
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

function setFolderExpansionState(folderKeys, expanded) {
    folderKeys.forEach((key) => {
        state.expandedFolders[key] = expanded;
    });
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

function renderSwaggerJsonView() {
    if (!elements.swaggerView || !elements.editor) {
        return;
    }

    attachJsonViewerEditHandler(elements.swaggerView, commitStructuredViewerEdit);
    renderJsonViewer(elements.swaggerView, state.swaggerSpec ?? (elements.editor.value || '{}'));
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
        elements.editor.value = pretty
            ? JSON.stringify(parsed, null, 2)
            : JSON.stringify(parsed);

        elements.editor.dispatchEvent(new Event('input'));
    } catch {
        setStatus('Cannot format invalid JSON content.', true);
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

async function commitStructuredViewerEdit({ editType, path, text }) {
    try {
        const source = state.swaggerSpec ?? parseSwaggerSpec(elements.editor.value);
        if (!source || !Array.isArray(path)) {
            return;
        }

        let nextDocument = source;

        if (editType === 'key') {
            nextDocument = renameObjectKey(nextDocument, path, String(text));
        } else if (editType === 'value') {
            const currentValue = getValueAtPath(nextDocument, path);
            const nextValue = parseStructuredScalar(String(text));

            if (Object.is(currentValue, nextValue)) {
                return;
            }

            nextDocument = setValueAtPath(nextDocument, path, nextValue);
        } else {
            return;
        }

        elements.editor.value = stringifyStructuredDocument(nextDocument);
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
        setStatus(err?.message || 'Failed to apply structured document edit.', true);
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

async function loadFile(file) {
    if (!file) {
        return;
    }

    const fileName = file ? getPathFileName(file) : 'json file';
    let stickyId = null;

    try {
        clearHexSource();

        // Capture the current project context to prevent autosave issues if user switches projects
        state.currentFileProject = await GetCurrentProject();

        const loadingJson     = isStructuredDataFile(file);
        const loadingMarkdown = isMarkdownNotesFile(file);
        const loadingImage    = isImageFile(file);
        const loadingCsv      = isCsvFile(file);
        stickyId = loadingJson ? Date.now() : null;

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
            const imageData = await GetImage(resolvedPath.replace(/^[/\\]+/, ''));
            if (imageData.startsWith('error:')) {
                setStatus(`Failed to load image: ${imageData}`, true);
                return;
            }
            elements.imageViewImg.src = imageData;
            elements.imageViewImg.dataset.originalFilename = fileName;
            enableFullscreenImages(elements.imageViewWrap);
            enableImageContextMenus(elements.imageViewWrap);

            setViewMode('image-view');
            setDirty(false);
            renderFileList();
            if (elements.findBar.dataset.open === 'true') {
                closeFindBar();
            }
            return;
        }

        if (loadingJson) {
            openStickyProgress(stickyId, `Loading ${fileName}… reading file`);
        }

        const result = await GetFile(file);

        state.currentFile = file;
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

            if (elements.findBar.dataset.open === 'true') {
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
            elements.editor.value = doc || '';
            refreshEditorLanguage(file, doc || '');

            // Render JSON tree view
            updateStickyProgress(stickyId, `Loading ${fileName}… rendering viewer`);
            await yieldToUI();
            renderSwaggerJsonView();
            
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
            setCodeEditorMode(true);
            elements.editorShell.dataset.fileType = 'markdown';
            state.swaggerSpec = null;
            state.swaggerRunAvailable = false;

            // Update UI for markdown
            updateTabVisibility('markdown');

            // Set editor content
            elements.editor.value = doc || '';
            refreshEditorLanguage(file, doc || '');

            // Render markdown views
            renderMarkdown();
            renderJupyterView();

            // Set default view mode to viewer
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
            elements.editor.value = doc || '';

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
            elements.editor.value = doc || '';
            refreshEditorLanguage(file, doc || '');
            
            // Set default view mode to editor
            setViewMode('editor');
        }
        
        setDirty(false);
        renderFileList();
        
        // Refresh the JSON viewer when switching to JSON files
        if (state.currentFileType === 'json') {
            renderSwaggerJsonView();
        }
        
        // Close find bar when loading a new file
        if (elements.findBar.dataset.open === 'true') {
            closeFindBar();
        }
    } catch (err) {
        if (stickyId) {
            closeStickyProgress(stickyId, `Failed to load ${getPathFileName(file)}`, 'error');
        }
        setStatus(`Failed to load ${file}.`, true);
        console.error(err);
    }
}

async function saveFile() {
    if (!state.currentFile) {
        setStatus('Select a note before saving.', true);
        return;
    }

    try {
        const content = state.currentFileType === 'json' 
            ? elements.editor.value 
            : elements.editor.value;
        
        // Use the saved project context to prevent overwrites if user switched projects
        await SaveFile(state.currentFile, content, state.currentFileProject || '');
        setDirty(false);
    } catch (err) {
        setStatus(`Failed to save ${state.currentFile}.`, true);
        console.error(err);
    }
}

function openDeletePrompt(file) {
    state.deletingFile = file;
    const fileName = getPathFileName(file);
    elements.deleteModalBody.textContent = `Are you sure you want to delete "${fileName}"?`;
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
}

async function confirmDelete() {
    if (!state.deletingFile) {
        setStatus('Select a note to delete.', true);
        return;
    }

    const fileToDelete = state.deletingFile;
    const fileName = getPathFileName(fileToDelete);

    try {
        await DeleteFile(fileToDelete);
        if (state.currentFile === fileToDelete) {
            state.currentFile = '';
            state.currentFileProject = '';
            emitCurrentFileName();
            elements.editor.value = '';
            elements.swaggerView.innerHTML = '';
            renderMarkdown();
            setDirty(false);
        }
        closeDeletePrompt();
        await refreshFiles();
        setStatus(`Deleted ${fileName}.`, false);
    } catch (err) {
        setStatus(`Failed to delete ${fileName}.`, true);
        console.error(err);
    }
}

function openFindBar() {
    if (!isFindAvailableInCurrentMode()) {
        notifyTerminal('Find not supported in this view', 'info');
        return;
    }

    elements.findBar.dataset.open = 'true';
    elements.findBar.setAttribute('aria-hidden', 'false');
    setTimeout(() => {
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
    elements.findBar.dataset.open = 'false';
    elements.findBar.setAttribute('aria-hidden', 'true');
    clearHighlights();
    state.findMatches = [];
    state.findCurrentIndex = -1;
    state.findQuery = '';
    elements.findCounter.textContent = '';
}

function isFindAvailableInCurrentMode() {
    return state.viewMode !== 'swagger-run' && state.viewMode !== 'image-view' && state.viewMode !== 'hex';
}

function updateFindAvailability() {
    const available = isFindAvailableInCurrentMode();
    // Do not set disabled — that swallows click events and prevents the
    // notification from firing. Use aria-disabled for accessibility only.
    elements.find.setAttribute('aria-disabled', available ? 'false' : 'true');

    if (!available && elements.findBar.dataset.open === 'true') {
        closeFindBar();
    }
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
        closeFindBar();
        return;
    }

    state.findQuery = query;
    clearHighlights();
    state.findMatches = [];
    state.findCurrentIndex = -1;

    if (getActiveFindEditor()) {
        findInEditor();
    } else {
        findInRenderedPane();
    }

    if (state.findMatches.length > 0) {
        state.findCurrentIndex = 0;
        highlightCurrentMatch({ focusEditor: false });
    }

    updateFindCounter();
}

function findInEditor() {
    const editorEl = getActiveFindEditor();
    if (!editorEl) {
        return;
    }

    const text = editorEl.value.toLowerCase();
    const query = state.findQuery.toLowerCase();
    let index = 0;

    while ((index = text.indexOf(query, index)) !== -1) {
        state.findMatches.push({
            start: index,
            end: index + query.length
        });
        index += query.length;
    }
}

function findInRenderedPane() {
    const query = state.findQuery;
    const container = getActiveFindContainer();
    if (!container) {
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
        if (node.textContent.toLowerCase().includes(query.toLowerCase())) {
            nodesToProcess.push(node);
        }
    }

    nodesToProcess.forEach((textNode) => {
        const text = textNode.textContent;
        const lowerText = text.toLowerCase();
        const lowerQuery = query.toLowerCase();
        const parts = [];
        let lastIndex = 0;
        let index;

        while ((index = lowerText.indexOf(lowerQuery, lastIndex)) !== -1) {
            if (index > lastIndex) {
                parts.push(document.createTextNode(text.substring(lastIndex, index)));
            }

            const highlight = document.createElement('span');
            highlight.className = 'find-highlight';
            highlight.textContent = text.substring(index, index + query.length);
            parts.push(highlight);
            state.findMatches.push(highlight);

            lastIndex = index + query.length;
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

function openNewFilePrompt() {
    state.renamingFile = null;
    elements.modal.dataset.open = 'true';
    elements.modal.setAttribute('aria-hidden', 'false');
    elements.modalInput.value = '';
    elements.modal.querySelector('#notes-modal-title').textContent = 'New note name';
    elements.modalCreate.textContent = 'Create';
    setTimeout(() => {
        elements.modalInput.focus();
    }, 0);
}

function openRenamePrompt(file) {
    state.renamingFile = file;
    elements.modal.dataset.open = 'true';
    elements.modal.setAttribute('aria-hidden', 'false');
    elements.modalInput.value = file;
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

    if (trimmed.toLowerCase().endsWith('.md')) {
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
                options: ['Copy image to clipboard', 'Save image...'],
                x: e.clientX,
                y: e.clientY,
                showNextToMouseCursor: true,
                icons: [0xf0c5, 0xf0c7],
                onSelect: (index) => {
                    if (index === 0) {
                        TerminalCopyImageDataURL(dataURLToCopy).catch(() => {
                            setStatus('Failed to copy image to clipboard.', true);
                        });
                    } else if (index === 1) {
                        saveImageToFile(filename, dataURLToCopy);
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
            setStatus('Failed to load file actions.', true);
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
                            setStatus('Failed to execute file action.', true);
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
        setStatus('Failed to open hyperlink actions.', true);
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

function initRenderedNotesContextMenu(container, viewMode) {
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
        const isRunMode = state.viewMode === 'jupyter';
        const tableIndex = table ? Array.from(container.querySelectorAll('table')).indexOf(table) : -1;
        const tableItems = table && container.contains(table)
            ? [...createTableCopyMenuItems(table), { title: '-' }]
            : [];
        const insertItems = (table && isRunMode && container.contains(table))
            ? [...createTableInsertMenuItems(table, e.target, tableIndex), { title: '-' }]
            : [];

        const allMenuItems = [
            createCopyMenuItem(() => getRenderedSelectionText(container), 'Copy'),
            { title: '-' },
            ...tableItems,
            ...insertItems,
            createFindMenuItem('Find'),
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

function initStructuredDataTreeContextMenu(container) {
    if (!container || container.dataset.jsonTreeContextMenuBound === 'true') {
        return;
    }

    container.dataset.jsonTreeContextMenuBound = 'true';

    container.addEventListener('contextmenu', (e) => {
        if (state.viewMode !== 'swagger-view') {
            return;
        }

        const target = e.target instanceof Element ? e.target.closest('.json-editable') : null;
        if (!target || !container.contains(target)) {
            return;
        }

        e.preventDefault();
        e.stopPropagation();

        showNotesLocalMenu([
            {
                title: 'Copy',
                icon: CONTEXT_ICON_COPY,
                onSelect: () => {
                    copyTextToClipboard(getJsonEditableCopyText(target));
                },
            },
            {
                title: 'Edit',
                icon: CONTEXT_ICON_EDIT,
                onSelect: () => {
                    target.dispatchEvent(new MouseEvent('dblclick', {
                        bubbles: true,
                        cancelable: true,
                        view: window,
                    }));
                },
            },
        ], e.clientX, e.clientY, 'JSON/YAML field');
    });
}

async function createNewFile() {
    // Handle rename operation
    if (state.renamingFile) {
        const fileName = (elements.modalInput.value || '').trim();
        if (fileName === '') {
            setStatus('File name cannot be empty.', true);
            return;
        }

        try {
            await RenameFile(state.renamingFile, fileName);
            await refreshFiles();
            if (state.currentFile === state.renamingFile) {
                await loadFile(fileName);
            }
            closeNewFilePrompt();
            setStatus(`Renamed to ${fileName}.`, false);
        } catch (err) {
            setStatus(`Failed to rename file.`, true);
            console.error(err);
        }
        return;
    }

    let fileName = normalizeNoteName(elements.modalInput.value);
    if (fileName === '') {
        setStatus('File name cannot be empty.', true);
        return;
    }

    // Handle new file creation

    const exists = state.files.some((file) => file === fileName);
    if (exists) {
        closeNewFilePrompt();
        await loadFile(fileName);
        setStatus(`${fileName} already exists.`, false);
        return;
    }

    try {
        await SaveFile(fileName, '', '');
        await refreshFiles();
        await loadFile(fileName);
        setViewMode('editor');
        closeNewFilePrompt();
        setStatus(`Created ${fileName}.`, false);
    } catch (err) {
        setStatus(`Failed to create ${fileName}.`, true);
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
        setStatus(`Created ${fileName}.`, false);
    } catch (err) {
        setStatus(`Failed to create ${fileName}.`, true);
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
        setStatus(`Image saved to ${savedPath}.`, false);
    } catch (err) {
        setStatus(`Failed to save image: ${err.message || err}`, true);
        console.error('Error saving image:', err);
    }
}

EventsOn("notesCreateAndOpen", params => {
    createAndOpenFile(params.filename, params.contents);
});

EventsOn("notesUpdate", group => {
    elements.title.innerText = group;
    refreshFiles();
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

// AI Panel Event Handlers
function setAIPanelCollapsed(collapsed) {
    const isCollapsed = collapsed === true;
    elements.aiPanel.dataset.collapsed = isCollapsed ? 'true' : 'false';
    elements.aiToggle.textContent = isCollapsed ? 'AI ▲' : 'AI ▼';
    if (elements.aiRestore) {
        elements.aiRestore.style.display = isCollapsed ? 'inline-flex' : 'none';
    }
    localStorage.setItem('notes-ai-panel-collapsed', String(isCollapsed));
}

function toggleAIPanel() {
    const isCollapsed = elements.aiPanel.dataset.collapsed === 'true';
    setAIPanelCollapsed(!isCollapsed);
}

function clearAIOutput() {
    elements.aiOutput.textContent = '';
}

function appendAIText(text) {
    if (elements.aiOutput.textContent === 'No AI response yet') {
        elements.aiOutput.textContent = '';
    }
    elements.aiOutput.appendChild(document.createTextNode(text));
    elements.aiOutput.scrollTop = elements.aiOutput.scrollHeight;
}

// Event listener for streaming AI responses
EventsOn("aiResponseStream", (chunk) => {
    const text = String(chunk ?? '');
    if (text) {
        appendAIText(text);
        // Auto-expand AI panel when response starts
        if (elements.aiPanel.dataset.collapsed === 'true') {
            toggleAIPanel();
        }
    }
});

// Event emitted by Go after the user selects a file from the ViewFileInNotes menu.
EventsOn('viewFileInNotesOpen', async (payload) => {
    const file = typeof payload === 'string' ? payload : String(payload ?? '');
    if (!file) return;

    try {
        await loadFile(file);
    } catch (err) {
        setStatus(`Failed to load file: ${file}`, true);
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
            openRenamePrompt(filePath);
            break;
        case 'delete':
            openDeletePrompt(filePath);
            break;
    }
});

// Setup AI panel listeners
if (elements.aiToggle) {
    elements.aiToggle.addEventListener('click', toggleAIPanel);
}
if (elements.aiClear) {
    elements.aiClear.addEventListener('click', clearAIOutput);
}
if (elements.aiRestore) {
    elements.aiRestore.addEventListener('click', () => setAIPanelCollapsed(false));
}

// Always start minimized on application launch.
setAIPanelCollapsed(true);

function applyWindowStyle(result) {
    document.body.style.color = `rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue})`;
    document.body.style.backgroundColor = `rgb(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue})`;

    //const notesFileSize = result.fontSize * 2;
    const notesStatusFontSize = result.fontSize - 2;
    const notesTitleFontSize = result.fontSize + 4;

    const style = document.createElement('style');
    style.textContent = `
        :root {
            --bg: rgb(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue});
            --fg: rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue});
            --accent: rgb(${result.colors.yellow.Red}, ${result.colors.yellow.Green}, ${result.colors.yellow.Blue});
            --link: rgb(${result.colors.link.Red}, ${result.colors.link.Green}, ${result.colors.link.Blue});
            --red: rgb(${result.colors.red.Red}, ${result.colors.red.Green}, ${result.colors.red.Blue});
            --green: rgb(${result.colors.green.Red}, ${result.colors.green.Green}, ${result.colors.green.Blue});
            --yellow: rgb(${result.colors.yellow.Red}, ${result.colors.yellow.Green}, ${result.colors.yellow.Blue});
            --blue: rgb(${result.colors.blue.Red}, ${result.colors.blue.Green}, ${result.colors.blue.Blue});
            --magenta: rgb(${result.colors.magenta.Red}, ${result.colors.magenta.Green}, ${result.colors.magenta.Blue});
            --cyan: rgb(${result.colors.cyan.Red}, ${result.colors.cyan.Green}, ${result.colors.cyan.Blue});
            --red-bright: rgb(${result.colors.redBright.Red}, ${result.colors.redBright.Green}, ${result.colors.redBright.Blue});
            --green-bright: rgb(${result.colors.greenBright.Red}, ${result.colors.greenBright.Green}, ${result.colors.greenBright.Blue});
            --yellow-bright: rgb(${result.colors.yellowBright.Red}, ${result.colors.yellowBright.Green}, ${result.colors.yellowBright.Blue});
            --blue-bright: rgb(${result.colors.blueBright.Red}, ${result.colors.blueBright.Green}, ${result.colors.blueBright.Blue});
            --magenta-bright: rgb(${result.colors.magentaBright.Red}, ${result.colors.magentaBright.Green}, ${result.colors.magentaBright.Blue});
            --cyan-bright: rgb(${result.colors.cyanBright.Red}, ${result.colors.cyanBright.Green}, ${result.colors.cyanBright.Blue});
            --selection: rgb(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue});
            --error: rgb(${result.colors.error.Red}, ${result.colors.error.Green}, ${result.colors.error.Blue});
            --font-family: ${result.fontFamily};
        }

        * {
            box-sizing: border-box;
            font-family: var(--font-family);
        }

        body {
            margin: 0 !important;
            padding: 0 !important;
        }

        ::selection {
            background-color: var(--selection);
        }

        ${getScrollbarStyles(result.colors)}

        #notes-app {
            display: grid;
            grid-template-columns: 1fr 2px 2fr;
            height: 100%;
            overflow: hidden;
            color: var(--fg);
            background: var(--bg);
        }

        #notes-sidebar {
            display: flex;
            flex-direction: column;
            padding: 0;
            gap: 12px;
            min-height: 0;
            overflow: hidden;
            background-color: ${DARKEN_BACKGROUND_OVERLAY};
        }

        #notes-sidebar-header {
            display: flex;
            flex-direction: column;
            gap: 12px;
        }

        #notes-title {
            font-size: ${notesTitleFontSize}px;
            color: var(--accent);
            padding: 10px 10px 0 10px;
        }

        #notes-list-filter-wrap {
            position: relative;
            padding: 0 10px;
        }

        #notes-list-filter {
            width: 100%;
            border-radius: 5px;
            border: 1px solid rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.45);
            background: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.08);
            color: var(--fg);
            padding: 6px 28px 6px 8px;
            font-size: ${result.fontSize - 1}px;
            outline: none;
        }

        #notes-list-filter-clear {
            position: absolute;
            top: 50%;
            right: 16px;
            transform: translateY(-50%);
            border: 0;
            background: transparent;
            color: rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 1);
            cursor: pointer;
            padding: 0;
            line-height: 1;
            display: none;
            font-family: "Font Awesome Solid", "Font Awesome", sans-serif;
            font-weight: 900;
            font-size: ${result.fontSize + 7}px;
        }

        #notes-list-filter-clear[data-visible="true"] {
            display: block;
        }

        #notes-list-filter-clear:hover {
            color: var(--accent);
        }

        #notes-list-filter::placeholder {
            color: rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.55);
        }

        #notes-list-filter:focus {
            border-color: var(--accent);
            background: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.16);
        }

        #notes-actions {
            display: flex;
            gap: 10px;
        }

        #notes-actions button {
            border-radius: 0;
            border: 2px solid var(--fg);
            background: transparent;
            color: var(--fg);
            padding: 6px 10px;
            cursor: pointer;
        }

        #notes-actions button:hover {
            border-color: var(--selection);
            color: var(--selection);
        }

        #notes-modal {
            position: fixed;
            inset: 0;
            display: none;
            align-items: center;
            justify-content: center;
            background: rgba(0, 0, 0, 0.45);
            z-index: 999;
        }

        #notes-modal[data-open="true"] {
            display: flex;
        }

        #notes-delete-modal {
            position: fixed;
            inset: 0;
            display: none;
            align-items: center;
            justify-content: center;
            background: rgba(0, 0, 0, 0.45);
            z-index: 999;
        }

        #notes-delete-modal[data-open="true"] {
            display: flex;
        }

        #notes-modal-card {
            min-width: 360px;
            max-width: 80vw;
            border: 2px solid var(--fg);
            background: var(--bg);
            color: var(--fg);
            padding: 14px;
            display: flex;
            flex-direction: column;
            gap: 10px;
            border-radius: 5px;
        }

        #notes-delete-modal-card {
            min-width: 360px;
            max-width: 80vw;
            border: 2px solid var(--fg);
            background: var(--bg);
            color: var(--fg);
            padding: 14px;
            display: flex;
            flex-direction: column;
            gap: 10px;
            border-radius: 5px;
        }

        #notes-modal-title {
            color: var(--accent);
            font-size: ${result.fontSize}px;
        }

        #notes-delete-modal-title {
            color: var(--accent);
            font-size: ${result.fontSize}px;
        }

        #notes-delete-modal-body {
            opacity: 0.9;
        }

        #notes-modal-input {
            border-radius: 0;
            border: 1px solid var(--fg);
            background: transparent;
            color: var(--fg);
            padding: 8px;
            font-size: ${result.fontSize}px;
            outline: none;
        }

        #notes-modal-input:focus {
            border-color: var(--accent);
        }

        .notes-toolbar {
            display: flex;
            gap: 4px;
            margin-left: auto;
            align-items: center;
            height: 20px;
        }

        .notes-toolbar-btn {
            border: none;
            background: transparent;
            color: var(--fg);
            font-size: 16px;
            cursor: pointer;
            padding: 6px 8px;
            display: flex;
            align-items: center;
            justify-content: center;
            border-radius: 4px;
            font-family: "Font Awesome Solid", "Font Awesome", sans-serif;
            font-weight: 900;
            transition: color 0.2s, background-color 0.2s;
            border-width: 1px !important;
        }

        /*.notes-toolbar-btn:hover {
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.3);
            color: var(--fg);
        }*/

        #notes-new:hover {
            color: var(--green) !important;
        }

        #notes-rename:hover, #notes-find:hover {
            color: var(--yellow) !important;
            border-radius: 5px;
            border-color: var(--yellow) !important;
            background-color: rgba(${result.colors.yellow.Red}, ${result.colors.yellow.Green}, ${result.colors.yellow.Blue}, 0.3);
        }

        #notes-delete:hover {
            color: var(--red) !important;
        }

        #notes-modal-create:hover {
            border-color: var(--green) !important;
            color: var(--green) !important;
        }

        #notes-modal-actions {
            display: flex;
            gap: 10px;
            justify-content: flex-end;
        }

        #notes-delete-modal-actions {
            display: flex;
            gap: 10px;
            justify-content: flex-end;
        }

        #notes-modal-actions button {
            border-radius: 5px;
            border: 2px solid var(--fg);
            background: transparent;
            color: var(--fg);
            padding: 6px 10px;
            cursor: pointer;
        }

        #notes-delete-modal-actions button {
            border-radius: 5px;
            border: 2px solid var(--fg);
            background: transparent;
            color: var(--fg);
            padding: 6px 10px;
            cursor: pointer;
        }

        #notes-modal-actions button:hover {
            border-color: var(--selection);
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.2);
            transition: all 0.2s ease;
        }

        #notes-delete-modal-actions button:hover {
            border-color: var(--selection);
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.2);
            transition: all 0.2s ease;
        }

        #notes-delete-confirm {
            border-color: var(--error);
            color: var(--error);
        }

        #notes-delete-confirm:hover {
            border-color: var(--error) !important;
            color: var(--error) !important;
            background-color: rgba(${result.colors.error.Red}, ${result.colors.error.Green}, ${result.colors.error.Blue}, 0.2);
            transition: all 0.2s ease;
        }

        #notes-list {
            display: flex;
            flex-direction: column;
            gap: 3px;
            overflow-y: auto;
            overflow-x: hidden;
            flex: 1;
            font-family: var(--font-family);
            font-size: ${result.fontSize}px;
            line-height: 1.25;
            padding-right: 5px;
        }

        .notes-category-header {
            display: flex;
            align-items: center;
            gap: 6px;
            padding: 3px 6px;
            cursor: pointer;
            color: var(--accent);
            /*font-weight: bold;*/
            border: 2px solid transparent;
            user-select: none;
            border-radius: 5px;
        }

        .notes-category-header:hover {
            /*border-color: var(--selection);*/
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.25);
        }

        .notes-category-arrow {
            font-size: ${result.fontSize - 2}px;
            width: 12px;
            display: inline-block;
        }

        .notes-category-content {
            display: flex;
            flex-direction: column;
            gap: 0;
            padding-left: 6px;
        }

        .notes-category-content[data-expanded="false"] {
            display: none;
        }

        .notes-file {
            min-height: 0;
            text-align: left;
            border-radius: 5px;
            border: none;
            background: transparent;
            color: var(--fg);
            padding: 1px 6px;
            cursor: pointer;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
            font-family: var(--font-family);
            font-size: ${result.fontSize}px;
            line-height: 1.25;
            -webkit-user-select: none;
            user-select: none;
        }

        .notes-tree-folder,
        .notes-tree-file {
            display: flex;
            align-items: center;
            gap: 2px;
            width: 100%;
            min-width: 0;
        }

        .notes-tree-folder {
            min-height: 0;
            text-align: left;
            border-radius: 5px;
            border: none;
            background: transparent;
            color: var(--yellow);
            padding: 1px 6px;
            cursor: pointer;
            font-family: var(--font-family);
            font-size: ${result.fontSize}px;
            line-height: 1.25;
            -webkit-user-select: none;
            user-select: none;
        }

        .notes-tree-folder:hover {
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.25);
        }

        .notes-tree-folder[data-expanded="false"] .notes-tree-label {
            font-style: italic;
        }

        .notes-tree-indent {
            flex: 0 0 auto;
        }

        .notes-tree-indent {
            display: inline-flex;
            align-self: stretch;
        }

        .notes-tree-branch {
            position: relative;
            display: block;
            align-self: stretch;
            width: 2ch;
            height: auto;
            color: rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.65);
        }

        .notes-tree-branch-continue::before,
        .notes-tree-branch-elbow::before {
            content: '';
            position: absolute;
            left: 0.8ch;
            top: -1px;
            bottom: -1px;
            border-left: 1px solid currentColor;
        }

        .notes-tree-branch-end::before {
            content: '';
            position: absolute;
            left: 0.8ch;
            top: -1px;
            bottom: 50%;
            border-left: 1px solid currentColor;
        }

        .notes-tree-branch-elbow::after,
        .notes-tree-branch-end::after {
            content: '';
            position: absolute;
            left: 0.8ch;
            top: calc(50% - 0.5px);
            width: 1.1ch;
            border-top: 1px solid currentColor;
        }

        .notes-tree-branch-end::after {
            top: 50%;
        }

        .notes-tree-label {
            min-width: 0;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
            font-family: var(--font-family);
            font-size: ${result.fontSize}px;
            line-height: 1.25;
        }

        .notes-file[data-active="true"] {
            background-color: var(--accent);
            color: var(--bg);
        }

        .notes-file[data-active="true"]:hover {
            color: var(--accent);
        }

        .notes-file:hover {
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.25);
        }

        #notes-empty {
            opacity: 0.7;
        }

        #notes-status {
            font-size: ${notesStatusFontSize}px;
            opacity: 0.8;
            color: var(--fg);
        }

        #notes-status[data-state="error"] {
            color: var(--error);
        }

        #notes-splitter {
            padding: 0;
            margin: 0;
            position: relative;
            width: 2px;
            cursor: col-resize;
            user-select: none;
            touch-action: none;
            flex-shrink: 0;
            background: ${DARKEN_BACKGROUND_OVERLAY};
        }

        #notes-splitter::after {
            content: '';
            position: absolute;
            /*left: 50%;*/
            top: 0;
            /*transform: translateX(-50%);*/
            width: 2px;
            height: 100%;
            background: color-mix(in srgb, var(--fg) 20%, transparent);
        }

        #notes-splitter:hover::after {
            background: var(--accent);
        }

        #notes-main {
            display: flex;
            flex-direction: column;
            padding: 0;
            height: 100%;
            min-height: 0;
            min-width: 0;
            background-color: ${DARKEN_BACKGROUND_OVERLAY};
        }

        #notes-panel {
            background-color: var(--bg);
        }

        #notes-tabs {
            display: flex;
            gap: 8px;
            padding: 6px 8px 0px 8px;
            border-bottom: 1px solid rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.2);
            /*align-items: center;*/

            box-sizing: border-box;
        }

        #notes-tabs button {
            border-radius: 0;
            border: 1px solid transparent;
            background: transparent;
            color: var(--fg);
            padding: 6px 12px;
            cursor: pointer;
        }

        #notes-tabs button[aria-selected="true"] {
            border-color: rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.2);
            border-bottom: 5px;
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.2);
            border-color: rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.2) !important;
        }

        .tab {
            border-top-left-radius: 5px !important;
            border-top-right-radius: 5px !important;
            border: 1px solid !important;
            border-bottom: 0 !important;
            border-color: rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.2) !important;
        }

        .tab:hover {
            border: 1px solid !important;
            border-bottom: 0 !important;
            border-color: rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.2) !important;
        }

        #notes-tabs button:hover {
            border-color: var(--selection);
        }

        #notes-new:hover {
            border-color: var(--green) !important;
            color: var(--green) !important;
            background-color: rgba(${result.colors.green.Red}, ${result.colors.green.Green}, ${result.colors.green.Blue}, 0.2);
            border-radius: 5px;
        }

        #notes-delete {
            color: var(--error);
        }

        #notes-delete:hover {
            border-color: var(--error) !important;
            color: var(--error);
            background-color: rgba(${result.colors.error.Red}, ${result.colors.error.Green}, ${result.colors.error.Blue}, 0.2);
            border-radius: 5px;
        }

        #notes-panel {
            position: relative;
            flex: 1;
            min-height: 0;
            min-width: 0;
            display: flex;
            flex-direction: column;
        }

        #notes-editor-wrap,
        #notes-preview-wrap,
        #notes-jupyter-wrap,
        #notes-meta-wrap {
            flex: 1;
            display: none;
            min-height: 0;
            border-bottom: 1px solid rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.2);
        }

        #notes-preview-wrap,
        #notes-jupyter-wrap,
        #notes-meta-wrap {
            padding-right: 10px;
            padding-bottom: 10px;
            overflow: auto;
        }

        #notes-swagger-view-wrap,
        #notes-swagger-run-wrap,
        #notes-hex-wrap {
            flex: 1;
            display: none;
            min-width: 0;
            min-height: 0;
            overflow: hidden;
        }

        #notes-pane[data-terminal-focused="true"] #notes-preview-wrap,
        #notes-pane[data-terminal-focused="true"] #notes-jupyter-wrap,
        #notes-pane[data-terminal-focused="true"] #notes-meta-wrap,
        #notes-pane[data-terminal-focused="true"] #notes-csv-view-wrap,
        #notes-pane[data-terminal-focused="true"] #notes-swagger-view-wrap,
        #notes-pane[data-terminal-focused="true"] #notes-swagger-run-wrap,
        #notes-pane[data-terminal-focused="true"] #notes-hex-wrap {
            background-color: ${DARKEN_BACKGROUND_OVERLAY};
        }

        #notes-editor-wrap[data-active="true"],
        #notes-hex-wrap[data-active="true"],
        #notes-preview-wrap[data-active="true"],
        #notes-jupyter-wrap[data-active="true"],
        #notes-meta-wrap[data-active="true"] {
            display: block;
        }

        #notes-swagger-view-wrap[data-active="true"],
        #notes-swagger-run-wrap[data-active="true"] {
            display: flex !important;
        }

        #notes-image-view-wrap {
            flex: 1;
            display: none;
            min-height: 0;
            overflow: auto;
            align-items: center;
            justify-content: center;
            padding: 16px;
            box-sizing: border-box;
        }

        #notes-image-view-wrap[data-active="true"] {
            display: flex;
        }

        #notes-image-view-img {
            max-width: 100%;
            max-height: 100%;
            object-fit: contain;
            cursor: zoom-in;
            user-select: none;
            border-radius: 4px;
        }

        #notes-csv-view-wrap {
            flex: 1;
            display: none;
            min-height: 0;
            min-width: 0;
            overflow-x: auto;
            overflow-y: auto;
        }

        #notes-csv-view-wrap[data-active="true"] {
            display: block;
        }

        #notes-csv-view {
            padding: 8px 16px;
            min-width: 0;
        }

        #notes-csv-view table {
            width: max-content;
            min-width: 100%;
            border-collapse: collapse;
        }

        #notes-csv-view th,
        #notes-csv-view td {
            border-bottom: 1px solid rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.18);
            padding: 4px 8px;
            text-align: left;
            white-space: nowrap;
        }

        #notes-csv-view thead th {
            border-bottom: 1px solid rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.28);
        }

        .notes-csv-empty {
            opacity: 0.5;
            font-style: italic;
            padding: 8px;
        }

        .notes-ai-panel {
            display: flex;
            flex-direction: column;
            border-top: 1px solid rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.5);
            transition: all 0.3s ease;
            overflow: hidden;
        }

        .notes-ai-panel[data-collapsed="false"] {
            flex: 0 1 35%;
            overflow-y: auto;
        }

        .notes-ai-panel[data-collapsed="true"] {
            flex: 0 0 0;
            min-height: 0;
            border-top: 0;
            opacity: 0;
            pointer-events: none;
        }

        .notes-ai-restore {
            display: none;
            position: absolute;
            right: 12px;
            bottom: 12px;
            z-index: 2;
            border-radius: 999px;
            border: 1px solid rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.4);
            background: rgba(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue}, 0.9);
            color: var(--fg);
            padding: 6px 12px;
            cursor: pointer;
            font-size: ${result.fontSize - 2}px;
            align-items: center;
            justify-content: center;
        }

        .notes-ai-restore:hover {
            border-color: var(--fg);
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 1);
        }

        .notes-ai-header {
            display: flex;
            gap: 8px;
            align-items: center;
            padding: 8px 12px;
            background: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.1);
            border-bottom: 1px solid rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.2);
            flex-shrink: 0;
        }

        .notes-ai-header button {
            border-radius: 3px;
            border: 1px solid rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.3);
            background: transparent;
            color: var(--fg);
            padding: 4px 10px;
            cursor: pointer;
            font-size: ${result.fontSize - 2}px;
            transition: all 0.2s ease;
        }

        .notes-ai-header button:hover {
            border-color: var(--fg);
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.2);
        }

        #notes-ai-clear:hover {
            color: var(--error);
            border-color: var(--error);
        }

        #notes-ai-output {
            flex: 1;
            padding: 12px;
            font-size: ${result.fontSize}px;
            line-height: 1.5;
            overflow-x: hidden;
            overflow-y: auto;
            white-space: pre-wrap;
            word-wrap: break-word;
            overflow-wrap: anywhere;
            word-break: break-word;
            font-family: var(--font-family);
            color: var(--fg);
            background-color: ${DARKEN_BACKGROUND_OVERLAY};
        }

        #notes-ai-output:empty::before {
            content: "No AI response yet";
            opacity: 0.5;
            font-style: italic;
        }

        #notes-editor {
            position: absolute;
            inset: 0;
            width: 100%;
            height: 100%;
            resize: none;
            border-radius: 0;
            border: 0;
            background: transparent;
            color: var(--fg);
            caret-color: var(--fg);
            padding: 10px 14px;
            font-size: ${result.fontSize}px;
            line-height: 1.4;
            white-space: pre-wrap;
            overflow-wrap: break-word;
            word-break: break-word;
            overflow-y: auto;
            overflow-x: hidden;
            font-family: var(--font-family);
            -webkit-user-modify: read-write-plaintext-only;
        }

        #notes-editor:focus {
            outline: none;
            box-shadow: none;
            border: 0;
        }

        #notes-editor:not(:focus) {
            background-color: transparent;
        }

        #notes-editor,
        .jupyter-code-editable,
        .jupyter-highlight,
        .jupyter-highlight code,
        .jupyter-highlight .hljs,
        .swagger-body-editor,
        #notes-editor-highlight,
        #notes-editor-highlight code,
        #notes-editor-highlight .hljs {
            tab-size: 4;
            -moz-tab-size: 4;
            letter-spacing: normal;
            font-variant-ligatures: none;
            font-feature-settings: "liga" 0, "calt" 0;
        }

        #notes-editor-shell {
            position: relative;
            display: grid;
            grid-template-columns: 1fr;
            height: 100%;
            width: 100%;
            border: 1px solid rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0);
            background-color: rgba(0, 0, 0, 0.22);
            transition: border-color 120ms ease;
        }

        #notes-editor-shell:focus-within {
            border-color: var(--accent);
            background-color: transparent;
        }

        #notes-editor-shell[data-code-view="true"] {
            grid-template-columns: max-content 1fr;
        }

        #notes-editor-shell[data-code-view="true"][data-file-type="markdown"] {
            grid-template-columns: 1fr;
        }

        #notes-editor-shell[data-code-view="true"][data-file-type="markdown"] #notes-editor-gutter-wrap {
            display: none;
        }

        #notes-editor-shell[data-code-view="true"] #notes-editor {
            color: transparent;
            white-space: pre;
            overflow-wrap: normal;
            word-break: normal;
            overflow: auto;
        }

        #notes-editor-shell[data-code-view="true"][data-file-type="markdown"] #notes-editor {
            white-space: pre-wrap;
            overflow-wrap: break-word;
            word-break: break-word;
            overflow-y: auto;
            overflow-x: hidden;
        }

        #notes-editor-shell[data-code-view="true"][data-file-type="markdown"] #notes-editor-highlight {
            white-space: pre-wrap;
            overflow-wrap: break-word;
            word-break: break-word;
            overflow: hidden;
        }

        #notes-editor-shell[data-code-view="true"][data-file-type="markdown"] #notes-editor-highlight code {
            white-space: pre-wrap;
            overflow-wrap: break-word;
            word-break: break-word;
        }

        #notes-editor-shell[data-code-view="false"] #notes-editor-highlight,
        #notes-editor-shell[data-code-view="false"] #notes-editor-gutter-wrap {
            display: none;
        }

        #notes-editor-gutter-wrap {
            position: relative;
            overflow: hidden;
            border-right: 1px solid rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.2);
            background: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.08);
            min-width: 48px;
        }

        #notes-editor-gutter {
            padding: 10px 10px 10px 12px;
            font-size: ${result.fontSize}px;
            line-height: 1.4;
            white-space: pre;
            text-align: right;
            color: rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.6);
            user-select: none;
            font-family: var(--font-family);
        }

        #notes-editor-scroll {
            position: relative;
            min-width: 0;
            height: 100%;
            overflow: hidden;
        }

        #notes-editor-shell[data-file-type="markdown"] #notes-editor-scroll {
            overflow: hidden;
        }

        #notes-editor-highlight {
            position: absolute;
            inset: 0;
            margin: 0;
            padding: 10px 14px;
            overflow: hidden;
            pointer-events: none;
            white-space: pre;
            line-height: 1.4;
            font-size: ${result.fontSize}px;
            font-family: var(--font-family);
        }

        #notes-editor-highlight code {
            display: block;
            padding: 0;
            background: transparent;
            white-space: pre;
        }

        #notes-editor-highlight .hljs {
            overflow: visible !important;
            padding: 0 !important;
            background: transparent !important;
        }

        #notes-preview-wrap,
        #notes-jupyter-wrap,
        #notes-meta-wrap {
            overflow-y: auto;
            padding-left: 16px;
        }

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

        #notes-preview,
        #notes-jupyter,
        #notes-meta,
        #notes-csv-view {
            min-width: 0;
        }

        #notes-preview img,
        #notes-jupyter img {
            max-width: 100%;
            height: auto;
        }

        #notes-find-bar {
            border-radius: 5px;
            position: absolute;
            top: 16px;
            right: 16px;
            display: none;
            align-items: center;
            gap: 8px;
            padding: 8px 12px;
            background: var(--bg);
            border: 2px solid var(--fg);
            z-index: 100;
        }

        #notes-find-bar[data-open="true"] {
            display: flex;
        }

        #notes-find-input {
            border-radius: 0;
            border: 1px solid var(--fg);
            background: transparent;
            color: var(--fg);
            padding: 4px 8px;
            font-size: ${result.fontSize}px;
            outline: none;
            min-width: 200px;
        }

        #notes-find-input:focus {
            border-color: var(--accent);
        }

        #notes-find-counter {
            font-size: ${result.fontSize - 2}px;
            opacity: 0.8;
            white-space: nowrap;
        }

        #notes-find-bar button {
            border-radius: 5px;
            border: 2px solid var(--fg);
            background: transparent;
            color: var(--fg);
            padding: 4px 8px;
            cursor: pointer;
            font-size: ${result.fontSize}px;
        }

        #notes-find-bar button:hover {
            border-color: var(--accent);
            color: var(--accent);
            transition: all 0.2s ease;
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.3);
        }

        .find-highlight {
            background-color: var(--accent);
            color: var(--bg);
        }

        .find-highlight-active {
            background-color: var(--blue);
            color: var(--bg);
        }

        /* Jupyter UI Styles */

        #notes-jupyter-wrap pre:not(.jupyter-highlight) {
            border-left: 0;
            padding-left: 10px;
            /*white-space: pre-wrap;
            word-wrap: break-word;*/
        }

        .jupyter-code-block {
            margin: 16px 0;
            border: 2px solid var(--fg);
            border-radius: 5px;
        }

        .jupyter-code-block:focus-within {
            border-color: var(--accent);
        }

        .jupyter-toolbar {
            display: flex;
            gap: 8px;
            padding: 0px;
            padding-left: 8px;
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.2);
            border-bottom: 2px solid var(--fg);
            align-items: center;
        }

        .jupyter-btn {
            padding: 5px 12px;
            margin-top: 8px;
            margin-bottom: 8px;
            background-color: transparent;
            border: 1px solid var(--fg);
            color: var(--fg);
            cursor: pointer;
            font-size: ${result.fontSize - 2}px;
            border-radius: 5px;
            transition: all 0.2s ease;
            align-items: center;
            vertical-align: middle;
        }
     
        .jupyter-btn:hover {
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.3);
            border-color: var(--accent);
            color: var(--accent);
        }

        .jupyter-btn:active {
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.5);
        }

        .jupyter-stop-notes {
            border-color: var(--red);
            color: var(--red);
        }

        .jupyter-stop-notes:hover {
            background-color: rgba(${result.colors.red.Red}, ${result.colors.red.Green}, ${result.colors.red.Blue}, 0.3);
            border-color: var(--red);
            color: var(--red);
        }

        .jupyter-stop-notes:active {
            background-color: rgba(${result.colors.red.Red}, ${result.colors.red.Green}, ${result.colors.red.Blue}, 0.5);
        }

        .jupyter-runtime-dropdown {
            margin: 8px;
            padding: 5px 24px 5px 12px;
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0);
            border: none;
            color: var(--accent);
            font-size: ${result.fontSize - 2}px;
            opacity: 0.8;
            cursor: pointer;
            outline: none;
            text-align: right;
            align-items: right;
            vertical-align: middle;
            background: none;
            font-family: var(--font-family);
        }

        .jupyter-runtime-dropdown:hover {
            opacity: 1;
            color: var(--fg);
        }

        .jupyter-runtime-dropdown:focus {
            opacity: 1;
            color: var(--fg);
        }

        .jupyter-code-editor {
            display: flex;
            align-items: stretch;
            background-color: var(--bg);
            max-height: calc((25 * 1.5em) + 24px);
            overflow: hidden;
        }

        .jupyter-code-area {
            position: relative;
            flex: 1;
            min-width: 0;
            overflow: hidden;
            max-height: calc((25 * 1.5em) + 24px);
        }

        #notes-jupyter .jupyter-highlight {
            position: absolute;
            inset: 0;
            margin: 0 !important;
            padding: 12px !important;
            border: 0 !important;
            border-left: 0 !important;
            pointer-events: none;
            overflow: hidden !important;
            white-space: pre !important;
            word-wrap: normal !important;
            overflow-wrap: normal !important;
            font-family: var(--font-family);
            font-size: ${result.fontSize}px;
            line-height: 1.5;
            background: transparent;
        }

        .jupyter-highlight code {
            display: block;
            padding: 0;
            background: transparent;
            white-space: pre;
        }

        .jupyter-highlight .hljs {
            overflow: visible !important;
            padding: 0 !important;
            background: transparent !important;
        }

        .jupyter-line-numbers {
            min-width: 42px;
            margin: 0;
            padding: 0;
            border-right: 1px solid rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.2);
            color: var(--fg);
            opacity: 0.45;
            font-family: var(--font-family);
            font-size: ${result.fontSize}px;
            line-height: 1.5;
            text-align: right;
            white-space: pre;
            user-select: none;
            pointer-events: none;
            overflow: hidden;
            max-height: calc((25 * 1.5em) + 24px);
        }

        .jupyter-line-numbers-inner {
            padding: 12px 8px 12px 10px;
            white-space: pre;
            line-height: 1.5;
            font-family: var(--font-family);
            font-size: ${result.fontSize}px;
            text-align: right;
            transform: translateY(0);
            will-change: transform;
        }

        .jupyter-code-editable {
            position: relative;
            z-index: 1;
            width: 100%;
            margin: 0;
            padding: 12px;
            background-color: transparent;
            border: 1px solid transparent;
            color: transparent;
            caret-color: var(--fg);
            font-family: var(--font-family);
            font-size: ${result.fontSize}px;
            line-height: 1.5;
            overflow-x: auto;
            overflow-y: auto;
            max-height: calc((25 * 1.5em) + 24px);
            outline: none;
            resize: none;
            box-sizing: border-box;
            white-space: pre;
        }

        .jupyter-code-editable:focus {
            outline: none;
            border-color: transparent;
        }

        .jupyter-code-block:not(:focus-within) .jupyter-code-area {
            background-color: ${DARKEN_BACKGROUND_OVERLAY};
        }

        .jupyter-output-wrapper {
            border-top: 2px solid var(--fg);
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.1);
        }

        .jupyter-output-toggle {
            width: 100%;
            padding: 8px 12px;
            background-color: transparent;
            border: none;
            border-bottom: 1px solid var(--fg);
            color: var(--fg);
            cursor: pointer;
            font-size: ${result.fontSize - 2}px;
            text-align: left;
            transition: all 0.2s ease;
        }

        .jupyter-output-toggle:hover {
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.2);
            color: var(--accent);
        }

        .jupyter-output {
            margin: 0;
            padding: 12px;
            background-color: var(--bg);
            color: var(--fg);
            font-family: var(--font-family);
            font-size: ${result.fontSize - 2}px;
            line-height: 1.4;
            overflow-x: auto;
            overflow-y: auto;
            max-height: calc((25 * 1.4em) + 24px);
            white-space: pre-wrap;
            word-wrap: break-word;
            border: none;
        }

        .jupyter-output-line {
            color: var(--green);
        }

        .jupyter-output-line-error {
            color: var(--error);
        }

        #notes-swagger-view-wrap {
            display: flex;
            flex-direction: column;
            padding: 0px;
        }

        #notes-swagger-view {
            overflow-y: auto;
            overflow-x: hidden;
            width: 100%;
            height: 100%;
            padding-right: 8px;
            font-family: var(--font-family);
            font-size: ${result.fontSize}px;
            line-height: 1.45;
        }

        .json-viewer-error {
            color: var(--error);
            border: 1px solid rgba(${result.colors.error.Red}, ${result.colors.error.Green}, ${result.colors.error.Blue}, 0.4);
            background-color: rgba(${result.colors.error.Red}, ${result.colors.error.Green}, ${result.colors.error.Blue}, 0.12);
            border-radius: 4px;
            padding: 10px;
            white-space: pre-wrap;
        }

        .json-node {
            color: var(--fg);
        }

        .json-node[data-expanded="false"] > .json-children {
            display: none;
        }

        .json-row {
            display: flex;
            align-items: baseline;
            flex-wrap: wrap;
            gap: 6px;
            min-height: 22px;
        }

        .json-toggle,
        .json-toggle-placeholder {
            width: 16px;
            min-width: 16px;
            height: 16px;
            display: inline-flex;
            align-items: center;
            justify-content: center;
        }

        .json-toggle {
            border: none;
            background: transparent;
            color: var(--green);
            padding: 0;
            margin: 0;
            cursor: pointer;
        }

        .json-node[data-expanded="false"] > .json-row > .json-toggle {
            color: var(--red);
        }

        .json-toggle:hover {
            filter: brightness(1.15);
        }

        .json-toggle:hover::before {
            opacity: 1;
        }

        .json-toggle::before {
            /*content: "\\f146";*/
            /*font-family: "Font Awesome Solid", "Font Awesome", sans-serif;*/
            content: "▼";
            font-weight: 900;
            font-size: 12px;
            line-height: 1;
            opacity: 0.3;
        }

        .json-node[data-expanded="false"] > .json-row > .json-toggle::before {
            /*content: "\\f0fe";*/
            content: "▶";
        }

        .json-key {
            color: var(--accent);
            word-break: break-all;
            overflow-wrap: anywhere;
        }

        .json-editable {
            border-radius: 3px;
            cursor: text;
        }

        .json-editable:hover {
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.18);
        }

        .json-editing,
        .json-editing:hover {
            background-color: transparent;
        }

        .json-inline-editor {
            border: 1px solid rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.55);
            background-color: rgba(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue}, 0.98);
            color: var(--fg);
            border-radius: 3px;
            padding: 1px 6px;
            font: inherit;
            line-height: inherit;
            min-width: 72px;
            box-sizing: border-box;
            outline: none;
            box-shadow: 0 0 0 1px rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.18);
        }

        .json-inline-editor:focus {
            border-color: var(--accent);
        }

        .json-colon,
        .json-brace {
            color: rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.85);
        }

        .json-meta {
            color: rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.55);
            margin-left: 6px;
            font-style: italic;
            font-size: ${Math.max(result.fontSize - 2, 10)}px;
        }

        .json-value-string {
            color: var(--green);
            word-break: break-all;
            overflow-wrap: anywhere;
        }

        .json-value-number {
            color: var(--cyan);
        }

        .json-value-boolean {
            color: var(--yellow);
        }

        .json-value-null {
            color: var(--magenta);
        }

        #notes-swagger-run-wrap {
            display: flex;
            flex-direction: column;
            height: 100%;
            overflow: hidden;
            padding: 0;
        }

        ${getSwaggerUIStyles(result.colors, result.fontSize)}

    `;

    document.head.appendChild(style);
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

refreshFiles();
window.refreshFiles = refreshFiles;

function insertEditorText(text, target = elements.editor) {
    if (!text) {
        return;
    }

    target.focus();
    document.execCommand('insertText', false, text);
}

async function savePastedImageDataUrl(dataUrl, mimeType) {
    if (!state.currentFile) {
        setStatus('Select a note before pasting an image.', true);
        return;
    }

    const comma = dataUrl.indexOf(',');
    if (comma <= 0 || comma >= dataUrl.length - 1) {
        setStatus('Clipboard image format is invalid.', true);
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
        setStatus(`Saved image ${paths.imageFileName}.`, false);
    } catch (err) {
        setStatus('Failed to save pasted image.', true);
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
            insertEditorText(text, targetEditor);
        }
    } catch (err) {
        setStatus('Failed to paste from clipboard.', true);
        console.error(err);
    }
}

if (elements.editor) {
    elements.editor.addEventListener('keydown', (event) => {
        if (event.key !== 'Tab' || event.ctrlKey || event.metaKey || event.altKey) {
            return;
        }

        // Keep Tab inside the editor so it doesn't trigger app-level focus hotkeys.
        event.preventDefault();
        event.stopPropagation();

        const start = elements.editor.selectionStart;
        const end = elements.editor.selectionEnd;
        elements.editor.setRangeText('\t', start, end, 'end');
        elements.editor.dispatchEvent(new Event('input'));
    });

    elements.editor.addEventListener('input', () => {
        setDirty(true);
        if (usesCodeEditorDecorations()) {
            refreshEditorLanguage(state.currentFile, elements.editor.value);
        } else if (state.currentFileType === 'csv') {
            renderCsvView(elements.editor.value, { interactive: state.viewMode === 'csv-run' });
        } else {
            scheduleRender();
        }

        if (state.currentFileType === 'json') {
            // Revalidate JSON/YAML, refresh the viewer, and only expose Run for docs with a swagger key.
            state.swaggerSpec = parseSwaggerSpec(elements.editor.value);
            state.swaggerRunAvailable = hasSwaggerKey(state.swaggerSpec);
            updateTabVisibility('json');
            renderSwaggerJsonView();

            if (!state.swaggerRunAvailable && state.viewMode === 'swagger-run') {
                setViewMode('swagger-view');
            } else if (state.swaggerRunAvailable && state.viewMode === 'swagger-run') {
                renderSwaggerUI();
            }
        }
        scheduleAutoSave();
    });

    elements.editor.addEventListener('scroll', () => {
        if (usesCodeEditorDecorations()) {
            syncEditorScrollDecorations();
        }
    });

    elements.editor.addEventListener('paste', (event) => {
        handleEditorImagePaste(event);
    });

    elements.editor.addEventListener('mouseup', async (event) => {
        // Middle-click should paste via the same text/image-aware clipboard logic.
        if (event.button !== 1 || state.viewMode !== 'editor') {
            return;
        }

        event.preventDefault();
        await pasteFromGoClipboard(elements.editor, true);
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

elements.editor.addEventListener('contextmenu', (e) => {
    // Restore selection that WebKit changed on right-click
    if (_editorSelectionBeforeContextMenu !== null) {
        elements.editor.selectionStart = _editorSelectionBeforeContextMenu.start;
        elements.editor.selectionEnd = _editorSelectionBeforeContextMenu.end;
        _editorSelectionBeforeContextMenu = null;
    }
    e.preventDefault();

    const menuItems = [
        createCopyMenuItem(() => getEditorSelectionText(), 'Copy'),
        {
            title: 'Paste',
            icon: CONTEXT_ICON_PASTE,
            onSelect: async () => {
                await pasteFromGoClipboard(elements.editor, !isStructuredDataFile(state.currentFile));
            },
        },
    ];

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

    menuItems.push(
        { title: '-' },
        createFindMenuItem('Find text...'),
        createPrintMenuItem('Print...'),
    );

    if (state.currentFileType === 'markdown') {
        menuItems.push(
            { title: '-' },
            {
                title: 'Insert checkbox',
                icon: CONTEXT_ICON_CHECKBOX,
                onSelect: () => {
                    const lineStart = elements.editor.value.lastIndexOf('\n', elements.editor.selectionStart - 1) + 1;
                    elements.editor.focus();
                    elements.editor.selectionStart = lineStart;
                    elements.editor.selectionEnd = lineStart;
                    document.execCommand('insertText', false, '- [ ] ');
                },
            },
            {
                title: 'Insert code block',
                icon: CONTEXT_ICON_CODE,
                onSelect: () => {
                    const selStart = elements.editor.selectionStart;
                    const selected = elements.editor.value.slice(selStart, elements.editor.selectionEnd);
                    elements.editor.focus();
                    document.execCommand('insertText', false, '```\n' + selected + '\n```');
                    elements.editor.selectionStart = selStart + 3;
                    elements.editor.selectionEnd = selStart + 3;
                },
            },
            {
                title: 'Insert table 3x1',
                icon: CONTEXT_ICON_TABLE,
                onSelect: () => {
                    elements.editor.focus();
                    document.execCommand('insertText', false, '| A | B | C |\n| --- | --- | --- |\n| cell | cell | cell |\n');
                },
            },
        );

        const imageAtCursor = getMarkdownImageAtCursor(elements.editor.value, elements.editor.selectionStart);
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

                        elements.editor.focus();
                        elements.editor.selectionStart = imageAtCursor.markdownStart;
                        elements.editor.selectionEnd = imageAtCursor.markdownEnd;
                        document.execCommand('insertText', false, '');
                        notifyTerminal(`Deleted image ${imageAtCursor.imagePath}.`, 'info');
                    } catch (err) {
                        notifyTerminal(`Failed to delete image ${imageAtCursor.imagePath}.`, 'error');
                        console.error(err);
                    }
                },
            });
        }
    }

    showNotesLocalMenu(menuItems, e.clientX, e.clientY);
});

initRenderedNotesContextMenu(elements.preview, 'viewer');
initRenderedNotesContextMenu(elements.jupyter, 'jupyter');
initRenderedNotesContextMenu(elements.swaggerRunWrap, 'swagger-run');

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
initStructuredDataTreeContextMenu(elements.swaggerView);

elements.tabEditor.addEventListener('click', () => {
    setViewMode('editor');
});

elements.tabHex.addEventListener('click', () => {
    setViewMode('hex');
});

elements.tabViewer.addEventListener('click', () => {
    setViewMode('viewer');
});

elements.tabJupyter.addEventListener('click', () => {
    setViewMode('jupyter');
    renderJupyterView();
});

elements.tabSwaggerView.addEventListener('click', () => {
    setViewMode('swagger-view');
    renderSwaggerJsonView();
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

elements.newFile.addEventListener('click', () => {
    openNewFilePrompt();
});

elements.rename.addEventListener('click', () => {
    if (!state.currentFile) {
        notifyTerminal('Select a note to rename.', 'warn');
        return;
    }
    openRenamePrompt(state.currentFile);
});

elements.modalCancel.addEventListener('click', () => {
    closeNewFilePrompt();
});

elements.modalCreate.addEventListener('click', () => {
    createNewFile();
});

elements.delete.addEventListener('click', () => {
    if (!state.currentFile) {
        notifyTerminal('Select a note to delete.', 'warn');
        return;
    }
    openDeletePrompt(state.currentFile);
});

elements.find.addEventListener('click', () => {
    openFindBar();
});

elements.deleteCancel.addEventListener('click', () => {
    closeDeletePrompt();
});

elements.deleteConfirm.addEventListener('click', () => {
    confirmDelete();
});

elements.findInput.addEventListener('input', () => {
    performFind();
});

if (elements.listFilter) {
    elements.listFilter.addEventListener('input', (event) => {
        state.fileFilterQuery = event.target.value || '';
        renderFileList();
    });

    elements.listFilter.addEventListener('keydown', (event) => {
        if (event.key === 'Escape' && elements.listFilter.value) {
            event.preventDefault();
            elements.listFilter.value = '';
            state.fileFilterQuery = '';
            renderFileList();
        }
    });
}

if (elements.listFilterClear && elements.listFilter) {
    elements.listFilterClear.addEventListener('click', () => {
        elements.listFilter.value = '';
        state.fileFilterQuery = '';
        renderFileList();
        elements.listFilter.focus();
    });
}

elements.findNext.addEventListener('mousedown', (event) => {
    event.preventDefault();
    nextMatch();
});

elements.findPrev.addEventListener('mousedown', (event) => {
    event.preventDefault();
    prevMatch();
});

elements.findClose.addEventListener('mousedown', (event) => {
    event.preventDefault();
    closeFindBar();
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
        saveFile();
    }

    if (event.key === 'Escape' && elements.findBar.dataset.open === 'true') {
        event.preventDefault();
        closeFindBar();
    } else if (event.key === 'Escape' && elements.modal.dataset.open === 'true') {
        event.preventDefault();
        closeNewFilePrompt();
    } else if (event.key === 'Escape' && elements.deleteModal.dataset.open === 'true') {
        event.preventDefault();
        closeDeletePrompt();
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

document.addEventListener('mouseup', (event) => {
    if (document.getElementById('fullscreen-image-overlay')) {
        return;
    }

    if (event.button !== 0) {
        return;
    }

    handleViewerSelectionAutoCopy();
});

elements.modalInput.addEventListener('keydown', (event) => {
    if (event.key === 'Enter') {
        event.preventDefault();
        createNewFile();
    }
});

elements.findInput.addEventListener('keydown', (event) => {
    if (event.key === 'Enter') {
        event.preventDefault();
        if (event.shiftKey) {
            prevMatch();
        } else {
            nextMatch();
        }
    }
});

setViewMode('editor');

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
