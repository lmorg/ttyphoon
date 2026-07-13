export namespace grep {
	
	export class Options {
	    CaseSensitive: boolean;
	    Regex: boolean;
	    WholeWord: boolean;
	    FileFilter: string;
	
	    static createFrom(source: any = {}) {
	        return new Options(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CaseSensitive = source["CaseSensitive"];
	        this.Regex = source["Regex"];
	        this.WholeWord = source["WholeWord"];
	        this.FileFilter = source["FileFilter"];
	    }
	}

}

export namespace jupyter {
	
	export class FormatCodeReturnT {
	    Code: string;
	    FilePath: string;
	    Err: string;
	    HasFormatter: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FormatCodeReturnT(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Code = source["Code"];
	        this.FilePath = source["FilePath"];
	        this.Err = source["Err"];
	        this.HasFormatter = source["HasFormatter"];
	    }
	}

}

export namespace lsp {
	
	export class ApplyCodeActionResult {
	    content: string;
	    changed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ApplyCodeActionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content = source["content"];
	        this.changed = source["changed"];
	    }
	}
	export class CodeActionItem {
	    title: string;
	    kind?: string;
	
	    static createFrom(source: any = {}) {
	        return new CodeActionItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.kind = source["kind"];
	    }
	}
	export class CodeLensItem {
	    index: number;
	    title: string;
	    line: number;
	    character: number;
	
	    static createFrom(source: any = {}) {
	        return new CodeLensItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.title = source["title"];
	        this.line = source["line"];
	        this.character = source["character"];
	    }
	}
	export class CompletionItem {
	    label: string;
	    detail?: string;
	    insertText?: string;
	    kind?: number;
	    deprecated?: boolean;
	    tags?: number[];
	
	    static createFrom(source: any = {}) {
	        return new CompletionItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.detail = source["detail"];
	        this.insertText = source["insertText"];
	        this.kind = source["kind"];
	        this.deprecated = source["deprecated"];
	        this.tags = source["tags"];
	    }
	}
	export class DefinitionLocation {
	    uri: string;
	    filePath?: string;
	    line: number;
	    character: number;
	
	    static createFrom(source: any = {}) {
	        return new DefinitionLocation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uri = source["uri"];
	        this.filePath = source["filePath"];
	        this.line = source["line"];
	        this.character = source["character"];
	    }
	}
	export class Position {
	    line: number;
	    character: number;
	
	    static createFrom(source: any = {}) {
	        return new Position(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.line = source["line"];
	        this.character = source["character"];
	    }
	}
	export class Range {
	    start: Position;
	    end: Position;
	
	    static createFrom(source: any = {}) {
	        return new Range(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start = this.convertValues(source["start"], Position);
	        this.end = this.convertValues(source["end"], Position);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Diagnostic {
	    range: Range;
	    severity: number;
	    code?: any;
	    source?: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new Diagnostic(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.range = this.convertValues(source["range"], Range);
	        this.severity = source["severity"];
	        this.code = source["code"];
	        this.source = source["source"];
	        this.message = source["message"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DocumentSymbolItem {
	    name: string;
	    detail?: string;
	    kind: number;
	    line: number;
	    character: number;
	    containerName?: string;
	
	    static createFrom(source: any = {}) {
	        return new DocumentSymbolItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.detail = source["detail"];
	        this.kind = source["kind"];
	        this.line = source["line"];
	        this.character = source["character"];
	        this.containerName = source["containerName"];
	    }
	}
	export class FormatResult {
	    content: string;
	    changed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FormatResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content = source["content"];
	        this.changed = source["changed"];
	    }
	}
	export class InlayHintItem {
	    label: string;
	    tooltip?: string;
	    kind?: number;
	    line: number;
	    character: number;
	    paddingLeft?: boolean;
	    paddingRight?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new InlayHintItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.tooltip = source["tooltip"];
	        this.kind = source["kind"];
	        this.line = source["line"];
	        this.character = source["character"];
	        this.paddingLeft = source["paddingLeft"];
	        this.paddingRight = source["paddingRight"];
	    }
	}
	
	export class PrepareRenameResult {
	    canRename: boolean;
	    placeholder?: string;
	
	    static createFrom(source: any = {}) {
	        return new PrepareRenameResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.canRename = source["canRename"];
	        this.placeholder = source["placeholder"];
	    }
	}
	
	export class RenameResult {
	    content: string;
	    changed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RenameResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content = source["content"];
	        this.changed = source["changed"];
	    }
	}
	export class SemanticTokenItem {
	    line: number;
	    character: number;
	    length: number;
	    tokenType: number;
	    tokenModifiers: number;
	
	    static createFrom(source: any = {}) {
	        return new SemanticTokenItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.line = source["line"];
	        this.character = source["character"];
	        this.length = source["length"];
	        this.tokenType = source["tokenType"];
	        this.tokenModifiers = source["tokenModifiers"];
	    }
	}
	export class WorkspaceSymbolItem {
	    name: string;
	    detail?: string;
	    kind: number;
	    line: number;
	    character: number;
	    containerName?: string;
	    uri?: string;
	    filePath?: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceSymbolItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.detail = source["detail"];
	        this.kind = source["kind"];
	        this.line = source["line"];
	        this.character = source["character"];
	        this.containerName = source["containerName"];
	        this.uri = source["uri"];
	        this.filePath = source["filePath"];
	    }
	}

}

export namespace main {
	
	export class ClipboardData {
	    text: string;
	    image: string;
	
	    static createFrom(source: any = {}) {
	        return new ClipboardData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.image = source["image"];
	    }
	}
	export class ColoursT {
	    fg: types.Colour;
	    bg: types.Colour;
	    black: types.Colour;
	    red: types.Colour;
	    green: types.Colour;
	    yellow: types.Colour;
	    blue: types.Colour;
	    magenta: types.Colour;
	    cyan: types.Colour;
	    white: types.Colour;
	    blackBright: types.Colour;
	    redBright: types.Colour;
	    greenBright: types.Colour;
	    yellowBright: types.Colour;
	    blueBright: types.Colour;
	    magentaBright: types.Colour;
	    cyanBright: types.Colour;
	    whiteBright: types.Colour;
	    accent: types.Colour;
	    selection: types.Colour;
	    link: types.Colour;
	    error: types.Colour;
	
	    static createFrom(source: any = {}) {
	        return new ColoursT(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fg = this.convertValues(source["fg"], types.Colour);
	        this.bg = this.convertValues(source["bg"], types.Colour);
	        this.black = this.convertValues(source["black"], types.Colour);
	        this.red = this.convertValues(source["red"], types.Colour);
	        this.green = this.convertValues(source["green"], types.Colour);
	        this.yellow = this.convertValues(source["yellow"], types.Colour);
	        this.blue = this.convertValues(source["blue"], types.Colour);
	        this.magenta = this.convertValues(source["magenta"], types.Colour);
	        this.cyan = this.convertValues(source["cyan"], types.Colour);
	        this.white = this.convertValues(source["white"], types.Colour);
	        this.blackBright = this.convertValues(source["blackBright"], types.Colour);
	        this.redBright = this.convertValues(source["redBright"], types.Colour);
	        this.greenBright = this.convertValues(source["greenBright"], types.Colour);
	        this.yellowBright = this.convertValues(source["yellowBright"], types.Colour);
	        this.blueBright = this.convertValues(source["blueBright"], types.Colour);
	        this.magentaBright = this.convertValues(source["magentaBright"], types.Colour);
	        this.cyanBright = this.convertValues(source["cyanBright"], types.Colour);
	        this.whiteBright = this.convertValues(source["whiteBright"], types.Colour);
	        this.accent = this.convertValues(source["accent"], types.Colour);
	        this.selection = this.convertValues(source["selection"], types.Colour);
	        this.link = this.convertValues(source["link"], types.Colour);
	        this.error = this.convertValues(source["error"], types.Colour);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CompleteSyntaxReturnT {
	    applied: boolean;
	    start: number;
	    end: number;
	    text: string;
	    cursor: number;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new CompleteSyntaxReturnT(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.applied = source["applied"];
	        this.start = source["start"];
	        this.end = source["end"];
	        this.text = source["text"];
	        this.cursor = source["cursor"];
	        this.error = source["error"];
	    }
	}
	export class FilterResultsT {
	    List: string[];
	    Error: any;
	
	    static createFrom(source: any = {}) {
	        return new FilterResultsT(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.List = source["List"];
	        this.Error = source["Error"];
	    }
	}
	export class GetFileReturnT {
	    contents: string;
	    binary: boolean;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new GetFileReturnT(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.contents = source["contents"];
	        this.binary = source["binary"];
	        this.error = source["error"];
	    }
	}
	export class RunFunctionReturnT {
	    Output: string;
	    IsError: boolean;
	    CellId: string;
	
	    static createFrom(source: any = {}) {
	        return new RunFunctionReturnT(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Output = source["Output"];
	        this.IsError = source["IsError"];
	        this.CellId = source["CellId"];
	    }
	}
	export class SpellCheckSuggestionT {
	    misspeltWord: string;
	    wordStart: number;
	    wordLength: number;
	    suggestions: string[];
	
	    static createFrom(source: any = {}) {
	        return new SpellCheckSuggestionT(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.misspeltWord = source["misspeltWord"];
	        this.wordStart = source["wordStart"];
	        this.wordLength = source["wordLength"];
	        this.suggestions = source["suggestions"];
	    }
	}
	export class WindowStyleT {
	    colors?: ColoursT;
	    statusBar: boolean;
	    fontFamily: string;
	    fontSize: number;
	    adjustCellWidth: number;
	    adjustCellHeight: number;
	
	    static createFrom(source: any = {}) {
	        return new WindowStyleT(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.colors = this.convertValues(source["colors"], ColoursT);
	        this.statusBar = source["statusBar"];
	        this.fontFamily = source["fontFamily"];
	        this.fontSize = source["fontSize"];
	        this.adjustCellWidth = source["adjustCellWidth"];
	        this.adjustCellHeight = source["adjustCellHeight"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace notes {
	
	export class DocumentCacheT {
	    DocumentTab: string;
	    ToolsOpen: boolean;
	    ToolsTab: string;
	    WordWrap: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DocumentCacheT(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.DocumentTab = source["DocumentTab"];
	        this.ToolsOpen = source["ToolsOpen"];
	        this.ToolsTab = source["ToolsTab"];
	        this.WordWrap = source["WordWrap"];
	    }
	}
	export class ProjectCacheT {
	    FileListCollapsed: Record<string, Array<string>>;
	
	    static createFrom(source: any = {}) {
	        return new ProjectCacheT(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.FileListCollapsed = source["FileListCollapsed"];
	    }
	}

}

export namespace sessiondb {
	
	export class FrontendHistoryItemT {
	    id: number;
	    prompt: string;
	    commandLine: string;
	    outputBlock: string;
	    response: string;
	    excerpt: string;
	
	    static createFrom(source: any = {}) {
	        return new FrontendHistoryItemT(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.prompt = source["prompt"];
	        this.commandLine = source["commandLine"];
	        this.outputBlock = source["outputBlock"];
	        this.response = source["response"];
	        this.excerpt = source["excerpt"];
	    }
	}
	export class FrontendSessionMetaT {
	    tableId: number;
	    summary: string;
	    created: string;
	    updated: string;
	    active: boolean;
	    entryCount: number;
	
	    static createFrom(source: any = {}) {
	        return new FrontendSessionMetaT(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tableId = source["tableId"];
	        this.summary = source["summary"];
	        this.created = source["created"];
	        this.updated = source["updated"];
	        this.active = source["active"];
	        this.entryCount = source["entryCount"];
	    }
	}
	export class FrontendStateT {
	    activeSessionId: number;
	    sessions: FrontendSessionMetaT[];
	    history: FrontendHistoryItemT[];
	
	    static createFrom(source: any = {}) {
	        return new FrontendStateT(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.activeSessionId = source["activeSessionId"];
	        this.sessions = this.convertValues(source["sessions"], FrontendSessionMetaT);
	        this.history = this.convertValues(source["history"], FrontendHistoryItemT);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace swagger {
	
	export class RequestT {
	    method: string;
	    url: string;
	    headers: Record<string, string>;
	    body: string;
	
	    static createFrom(source: any = {}) {
	        return new RequestT(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.method = source["method"];
	        this.url = source["url"];
	        this.headers = source["headers"];
	        this.body = source["body"];
	    }
	}
	export class ResponseT {
	    statusCode: number;
	    status: string;
	    headers: Record<string, string>;
	    body: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ResponseT(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.statusCode = source["statusCode"];
	        this.status = source["status"];
	        this.headers = source["headers"];
	        this.body = source["body"];
	        this.error = source["error"];
	    }
	}

}

export namespace types {
	
	export class Colour {
	    Red: number;
	    Green: number;
	    Blue: number;
	    Alpha: number;
	
	    static createFrom(source: any = {}) {
	        return new Colour(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Red = source["Red"];
	        this.Green = source["Green"];
	        this.Blue = source["Blue"];
	        this.Alpha = source["Alpha"];
	    }
	}
	export class InputBoxCallbackResultT {
	    value: string;
	    variables: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new InputBoxCallbackResultT(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.variables = source["variables"];
	    }
	}
	export class XY {
	    X: number;
	    Y: number;
	
	    static createFrom(source: any = {}) {
	        return new XY(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.X = source["X"];
	        this.Y = source["Y"];
	    }
	}

}

