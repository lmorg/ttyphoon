import { SendIpc } from '../wailsjs/go/main/WApp';

import { EventsOn, WindowShow, WindowHide, WindowSetPosition } from '../wailsjs/runtime/runtime';

document.querySelector('#app').innerHTML = `
<div id="ttyphoon-bob"></div>
<img id="ttyphoon-preview">
`;

const preview = document.getElementById('ttyphoon-preview')
const bob = document.getElementById('ttyphoon-bob')

var visible = false

EventsOn("previewOpen", params => {
    WindowSetPosition(params.x, params.y);
    if (visible === false) {
        visible = true;
        preview.src = params.url;
        bob.innerText = params.url;
        WindowShow();
        SendIpc("focus", {});
    }
});

EventsOn("previewHide", params => {
    //WindowHide();
    //visible = false;
});
