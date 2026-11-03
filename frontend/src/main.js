import { Events } from '@wailsio/runtime';
import './style.css';
import './app.css';
import './ttyphoon.js';

void Events.Emit('ttyphoon:frontend-ready');
