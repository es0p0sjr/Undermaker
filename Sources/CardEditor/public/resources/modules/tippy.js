import tippy from './3rdparty/tippy.js';
import { isDragging } from './draggable.js';
import extras from './extras.js';

let active;

// Important! Wrapper must exist on document before registering
export default function registerTips(wrapper) {
  const nameCell = wrapper.querySelector('.name');
  const monster = wrapper.querySelector('table.monster') !== null;
  tippy(nameCell, {
    theme: 'black',
    target: 'input',
    content: document.getElementById('selectSoul').innerHTML,
    placement: 'right-start',
    distance: -5,
    onMount(e) {
      e.popper.querySelectorAll('span.selectable').forEach((span) => {
        span.onclick = modifySoul.bind(span, nameCell, e.popper);
      });
    },
    onShow(e) {
      return !isDragging() && (!monster || extras());
    },
  });
  const image = wrapper.querySelector('.image img');
  tippy(wrapper.querySelector('.image'), {
    content: 'Click to Select Image',
    placement: 'top',
    arrow: true,
    trigger: 'mouseenter',
    size: 'small',
    interactive: false,
    onShown() {},
    onShow() {
      return !isDragging() && !image.src;
    },
  });
  // TODO: Set description keywords to allow insertion
  tippy(wrapper.querySelector('.description textarea'), {
    content: document.getElementById('descriptionTip').innerHTML,
    arrow: true,
    hideOnClick: true,
  });
  tippy(wrapper.querySelector('.rarity'), {
    theme: 'black',
    trigger: 'mouseenter',
    hideOnClick: true,
    content: document.getElementById('selectRarity').innerHTML,
    placement: 'top',
    size: 'small',
    distance: -1,
    shouldPopperHideOnBlur() { return true; },
    onMount(e) {
      const tip = e.popper._tippy;
      e.popper.querySelectorAll('img.selectable').forEach((item) => {
        item.onclick = () => {
          editEvent('rarity');
          e.popper.querySelector('img.active').classList.remove('active');
          item.classList.add('active');
          tip.reference.querySelector('img').src = item.src;
          tip.hide();
        };
      });
    },
  });
  const tribe = wrapper.querySelector('.tribe');
  tippy(tribe, {
    theme: 'black',
    trigger: 'mouseenter focus',
    content: document.getElementById('selectTribe').innerHTML,
    placement: 'top-end',
    onMount(e) {
      e.popper.querySelectorAll('img.selectable').forEach((item) => {
        item.onclick = () => {
          editEvent('tribe');
          e.popper.querySelector('img.active').classList.remove('active');
          item.classList.add('active');
          const isMonster = item.src.includes('MONSTER');
          tribe.classList.toggle('none', isMonster);
          tribe.classList.toggle('no-save', isMonster);
          tribe.classList.toggle('smallerIcon', item.src.includes('SPIDER'));
          tribe.src = item.src;
          e.popper._tippy.hide();
        };
      });
    },
  });
}

/*
export function registerGroup(group) {
  tippy(group.querySelector('.new'), {
    content: document.getElementById('groupButtons').innerHTML,
    placement: 'bottom-end',
    trigger: 'click',
    theme: 'button',
    distance: 0,
    hideOnClick: true,
    onMount(e) {
      e.popper.querySelectorAll('button').forEach((button) => {
        const monster = button.textContent === 'Monster';
        if (monster || button.textContent === 'Spell') {
          button.onclick = () => generate(monster, group);
        }
      });
    },
  });
}
// */

function modifySoul(nameCell, popper) {
  const activeSoul = popper.querySelector('span.selectable.active');
  const input = nameCell.querySelector('input');
  if (this === activeSoul) {
    input.focus();
    return;
  }
  if (activeSoul) {
    activeSoul.classList.remove('active');
    nameCell.classList.remove(activeSoul.textContent);
  }
  editEvent('soul');
  this.classList.add('active');
  // Modify the cell
  nameCell.classList.add(this.textContent);
  input.focus();
}

tippy.setDefaults({
  onShown,
  onShow,
  onHide,
  placement: 'right-end',
  a11y: false,
  arrow: false,
  trigger: 'manual',
  duration: 0,
  hideOnClick: false,
  // performance: true,
  shouldPopperHideOnBlur() {
    return false;
  },
  interactive: true,
  ignoreAttributes: true,
  animateFill: false,
});

function onShown(instance) {
  active = instance;
  closeAll(instance);
}

function onHide(instance) {
  if (active === instance) {
    active = null;
  }
}

function onShow() {
  return !isDragging() && !active;
}

function closeAll(except) {
  document.querySelectorAll('.tippy-popper').forEach((p) => {
    if (!except || p._tippy !== except) p._tippy.hide();
  });
}
