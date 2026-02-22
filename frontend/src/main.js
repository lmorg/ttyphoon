import './style.css';
import './app.css';

import { GetWindowType } from '../wailsjs/go/main/WApp';

GetWindowType().then((result) => {
    switch(result) {
    case "inputBox":
        import('./inputbox.js');
        break;
    case "markdown":
        import('./markdown.js');
        break;
    default:
        // code block
    };
});
