import './style.css';
import './app.css';

import { GetWindowType } from '../wailsjs/go/main/WApp';

GetWindowType().then((result) => {
    switch(result) {
    case "sdl":
    case "terminal":
        import('./terminal.js');
        break;
    case "inputBox":
        import('./inputbox.js');
        break;
    case "markdown":
    case "history":
        import('./markdown.js');
        break;
    case "preview":
        import('./preview.js');
        break;
    case "notes":
        import('./notes.js');
        break;
    default:
        // code block
    };
});
