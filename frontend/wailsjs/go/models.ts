export namespace dispatcher {
	
	export class WindowStyleT {
	    fg: types.Colour;
	    bg: types.Colour;
	    pos: types.XY;
	    size: types.XY;
	    alwaysOnTop: boolean;
	    frameLess: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WindowStyleT(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fg = this.convertValues(source["fg"], types.Colour);
	        this.bg = this.convertValues(source["bg"], types.Colour);
	        this.pos = this.convertValues(source["pos"], types.XY);
	        this.size = this.convertValues(source["size"], types.XY);
	        this.alwaysOnTop = source["alwaysOnTop"];
	        this.frameLess = source["frameLess"];
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

