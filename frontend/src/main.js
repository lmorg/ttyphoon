import './style.css';
import './app.css';

import { GetWindowType } from '../wailsjs/go/main/WApp';

GetWindowType().then((result) => {
    switch(result) {
    case "inputBox":
        import('./inputbox.js');
        break;
    case "markdown":
    case "history":
        import('./markdown.js');
        break;
    case "preview":
        //import('./preview.js');
        break;
    default:
        // code block
    };
});
