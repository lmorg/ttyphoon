export namespace dispatcher {
	
	export enum WindowTypeT {
	    sdl = "sdl",
	    inputBox = "inputBox",
	    markdown = "markdown",
	}
	export class ColoursT {
	    fg: types.Colour;
	    bg: types.Colour;
	    green: types.Colour;
	    yellow: types.Colour;
	    magenta: types.Colour;
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
	        this.green = this.convertValues(source["green"], types.Colour);
	        this.yellow = this.convertValues(source["yellow"], types.Colour);
	        this.magenta = this.convertValues(source["magenta"], types.Colour);
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
	export class WindowStyleT {
	    pos: types.XY;
	    size: types.XY;
	    alwaysOnTop: boolean;
	    frameLess: boolean;
	    fontFamily: string;
	    fontSize: number;
	    title: string;
	    colors?: ColoursT;
	
	    static createFrom(source: any = {}) {
	        return new WindowStyleT(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pos = this.convertValues(source["pos"], types.XY);
	        this.size = this.convertValues(source["size"], types.XY);
	        this.alwaysOnTop = source["alwaysOnTop"];
	        this.frameLess = source["frameLess"];
	        this.fontFamily = source["fontFamily"];
	        this.fontSize = source["fontSize"];
	        this.title = source["title"];
	        this.colors = this.convertValues(source["colors"], ColoursT);
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

