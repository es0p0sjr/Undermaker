/* global gtag */
import card from './card.js';
import draggable from './draggable.js';
import tip from './tippy.js';
import save from './save.js';

let id = 0;

export default function newGroup() {
  const container = document.createElement('div');
  container.innerHTML = document.querySelector('#group').innerHTML;
  setupName(container);
  setupButtons(container);
  container.id = `group${id++}`;
  container.classList.add('group');
  if (this) {
    this.after(container);
  } else {
    document.body.append(container);
  }

  draggable(container);

  return container;
}

/** @param {HTMLElement} container */
function generate(monster, container) {
  gtag('event', `create_${monster ? 'monster' : 'spell'}`);
  const wrapper = card(monster);
  container.querySelector('.cards').append(wrapper);

  tip(wrapper); // must be done after adding to document
}

/** @param {HTMLElement} container */
function setupName(container) {
  const name = container.querySelector('.group-name');
  const input = container.querySelector('.group-name + input');
  name.onclick = () => {
    name.style.display = 'none';
    input.value = name.textContent;
    input.focus();
  };
  input.onblur = () => {
    name.style.display = '';
    const text = input.value.trim();
    if (text) {
      name.textContent = text;
    }
  };
  if (id) {
    name.textContent = `Group ${id}`;
  }
  input.placeholder = name.textContent;
}

/** @param {HTMLElement} container */
function setupButtons(container) {
  container.querySelector('.save').parentElement.onclick = () => save(container, container.querySelector('.group-name').textContent);
  container.querySelector('.book').parentElement.onclick = newGroup.bind(container);
  container.querySelector('.monster').parentElement.onclick = () => generate(true, container);
  container.querySelector('.spell').parentElement.onclick = () => generate(false, container);
  container.querySelector('.text').parentElement.remove();
}
