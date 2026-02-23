import { GetWindowStyle, GetParameters } from '../wailsjs/go/main/WApp';
import { VisualInputBox } from '../wailsjs/go/main/WApp';

import { WindowSetSize, Quit } from '../wailsjs/runtime/runtime';


function autoGrow() {
    let input = document.getElementById('input')
    if (input.scrollHeight > input.clientHeight)
        input.style.height = input.scrollHeight + 'px';

    if (window.scrollHeight > window.clientHeight)
        WindowSetSize(window.style.width, window.scrollHeight);
}

window.autoGrow = autoGrow;

document.querySelector('#app').innerHTML = `
    <div class="title" id="title">{{Title}}</div>
      <div class="input-box">
        <div class="input" id="input" contenteditable="plaintext-only" onkeydown="autoGrow();" onkeyup="autoGrow();"></div>
        <div class="btn" onclick="send()">Send</div>
      </div>
    </div>
`;

let inputElement = document.getElementById("input");
setTimeout(function() { inputElement.focus() }, 0);

let titleElement = document.getElementById("title");

inputElement.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
        Quit();
    } else if (e.key === "Enter" && e.ctrlKey) {
        e.preventDefault();
        window.send();
    }
});

GetWindowStyle().then((result) => {
    document.body.style.color           = `rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue})`;
    document.body.style.backgroundColor = `rgb(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue})`;
    inputElement.style.color            = `rgb(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue})`;
    inputElement.style.backgroundColor  = `rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue})`;
    titleElement.style.fontSize = result.fontSize * 2;
    
    const style = document.createElement('style');
    style.textContent = `::selection {
        background-color: rgb(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue});
    }`;
    document.head.appendChild(style);

    document.querySelectorAll('div').forEach(div => {
        div.style.fontFamily = result.fontFamily;
        div.style.fontSize   = result.fontSize;
    });
})

GetParameters().then((result) => {
    titleElement.innerText = result['title']
    //titleElement.innerText = result
})

window.send = function () {
    try {
        //VisualInputBox(inputElement.value)
        VisualInputBox(inputElement.innerText)
            .then((result) => {
                titleElement.innerText = result;
            })
            .catch((err) => {
                console.error(err);
            });
    } catch (err) {
        console.error(err);
    }
};
