import { GetWindowStyle, GetParameters } from '../wailsjs/go/main/WApp';
import { VisualInputBox } from '../wailsjs/go/main/WApp';

function autoGrow() {
    if (document.getElementById('input').scrollHeight > document.getElementById('input').clientHeight)
        document.getElementById('input').style.height = document.getElementById('input').scrollHeight + 'px';
}

window.autoGrow = autoGrow;

document.querySelector('#app').innerHTML = `
    <div class="title" id="title">{{Title}}</div>
      <div class="input-box">
        <div class="input" id="input" contenteditable="plaintext-only" onkeydown="autoGrow();" onkeyup="autoGrow();"></div>
        <!--<textarea class="input" id="input" onkeydown="autoGrow();" onkeydown="autoGrow();"></textarea>-->
        <div class="btn" onclick="send()">Send</div>
      </div>
    </div>
`;

let inputElement = document.getElementById("input");
setTimeout(function() { inputElement.focus() }, 0);

let titleElement = document.getElementById("title");

GetWindowStyle().then((result) => {
    document.body.style.color           = `rgb(${result.fg.Red}, ${result.fg.Green}, ${result.fg.Blue})`;
    document.body.style.backgroundColor = `rgb(${result.bg.Red}, ${result.bg.Green}, ${result.bg.Blue})`;
    inputElement.style.color            = `rgb(${result.bg.Red}, ${result.bg.Green}, ${result.bg.Blue})`;
    inputElement.style.backgroundColor  = `rgb(${result.fg.Red}, ${result.fg.Green}, ${result.fg.Blue})`;
    let titleFontSize = result.fontSize * 2;
    titleElement.style.fontSize = titleFontSize;

    document.querySelectorAll('div').forEach(div => {
        div.style.fontFamily = result.fontFamily;
        div.style.fontSize   = result.fontSize;
    });
})

GetParameters().then((result) => {
    titleElement.innerText = result['title']
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
