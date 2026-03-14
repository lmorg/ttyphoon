import './style.css';
import './app.css';

// Remove any body margin/padding immediately so there is no layout flash.
document.body.style.margin = '0';
document.body.style.padding = '0';
document.body.style.overflow = 'hidden';

const app = document.getElementById('app') || document.body;

// The split layout: notes on the left half, terminal on the right half.
// Both panes are created synchronously here.  notes.js and terminal.js are
// loaded as dynamic imports below, so their module bodies run *after* this
// synchronous code — they will find #notes-pane and #terminal-pane in the DOM.
app.style.cssText = [
    'display:flex',
    'width:100vw',
    'height:100vh',
    'margin:0',
    'padding:0',
    'overflow:hidden',
    'box-sizing:border-box',
].join(';');

const notesPane = document.createElement('div');
notesPane.id = 'notes-pane';
notesPane.style.cssText = [
    'width:50%',
    'height:100%',
    'overflow:hidden',
    'position:relative',
    // Subtle separator between the two halves.  The notes module injects CSS
    // variables for the theme colours; until those land we use a semi-transparent
    // white border that is unobtrusive on any background.
    'border-right:1px solid rgba(255,255,255,0.12)',
    'box-sizing:border-box',
    'flex-shrink:0',
].join(';');

const terminalPane = document.createElement('div');
terminalPane.id = 'terminal-pane';
terminalPane.style.cssText = [
    'flex:1',
    'height:100%',
    'overflow:hidden',
    'position:relative',
    'min-width:0',
].join(';');

app.appendChild(notesPane);
app.appendChild(terminalPane);

// Dynamic imports — the promises resolve asynchronously, but the resolution
// microtask queue starts only after this synchronous module body finishes.
// By then #notes-pane and #terminal-pane exist, so each module finds its root.
import('./notes.js');
import('./terminal.js');
