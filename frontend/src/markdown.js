import { VisualInputBox } from '../wailsjs/go/main/WApp';

function autoGrow() {
    if (document.getElementById('input').scrollHeight > document.getElementById('input').clientHeight) document.getElementById('input').style.height = document.getElementById('input').scrollHeight + 'px';
}

window.autoGrow = autoGrow;

document.querySelector('#app').innerHTML = `
    <div class="title" id="title">{{Title}}</div>
      <div class="input-box">
        <!--<input class="input" id="input" type="text" autocomplete="off" />-->
        <!--<div class="input" id="input" contenteditable="plaintext-only" onkeydown="autoGrow();" onkeyup="autoGrow();"></div>-->
        <textarea class="input" id="input" onkeydown="autoGrow();" onkeydown="autoGrow();"></textarea>
        <a class="btn" onclick="send()">Send</a>
      </div>
    </div>
`;

let inputElement = document.getElementById("input");
setTimeout(function() { inputElement.focus() }, 0);


let titleElement = document.getElementById("title");

GetWindowStyle().then((result) => {
    document.body.style.color           = `rgb(${result.fg.Red}, ${result.fg.Green}, ${result.fg.Blue})`;
    document.body.style.backgroundColor = `rgb(${result.bg.Red}, ${result.bg.Green}, ${result.bg.Blue})`;
    inputElement.style.color           = `rgb(${result.bg.Red}, ${result.bg.Green}, ${result.bg.Blue})`;
    inputElement.style.backgroundColor = `rgb(${result.fg.Red}, ${result.fg.Green}, ${result.fg.Blue})`;
})

GetParameters().then((result) => {
    titleElement.innerText = result['title']
})

window.send = function () {
    try {
        VisualInputBox(inputElement.value)
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
