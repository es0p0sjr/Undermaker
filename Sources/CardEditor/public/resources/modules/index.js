import serviceWorker from './sw.register.js';
import newGroup from './group.js';
import { ready as effects } from './effects.js';

const preloads = [
  serviceWorker(),
  effects,
];

function ready() {
  document.querySelectorAll('[legacy], #loading').forEach((el) => el.remove());
  document.querySelector('#draggable-live-region').remove();

  newGroup();
}

Promise.all(preloads)
  .then(ready)
  // eslint-disable-next-line no-console
  .catch(console.error);
