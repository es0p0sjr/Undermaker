export const effects = [];
export const specials = ['ATK', 'DMG', 'HP', 'KR', 'cost', 'G', 'TOKEN', 'BASE', 'COMMON', 'RARE', 'EPIC', 'LEGENDARY', 'DT'];

const tribeBlacklist = ['Tem', 'Frog']

const div = document.querySelector('#descriptionTip div');
function addType(type) {
  const el = document.createElement('span');
  el.innerText = type.replace('s?', '');
  div.append(el, ' ');
}

function load(resource) {
  return fetch(`/resources/data/${resource}.json`)
    .then((res) => res.json());
}

function addTribes() {
  const container = document.createElement('div');
  container.innerHTML = document.querySelector('#selectTribe').innerHTML;
  const last = container.querySelector('img:last-child');
  effects.forEach((effect) => {
    if (!effect.endsWith('?')) return;
    const name = effect.substring(0, effect.length - 2);
    if (tribeBlacklist.includes(name)) return;
    last.before(getTribe(name));
  });
  document.querySelector('#selectTribe').innerHTML = container.innerHTML;
}

function getTribe(name) {
  const container = document.createElement('div');
  container.innerHTML = document.querySelector('#selectTribeImg').innerHTML;
  const img = container.querySelector('img');
  img.src = `/resources/tribes/${name.toUpperCase().replace(' ', '_')}.png`;
  img.title = name;
  return img;
}

const resources = ['effects', 'extra'];

export const ready = Promise.all(resources.map(load))
  .then((res) => {
    // Populate effects array
    res.forEach((data) => effects.push(...data));

    effects.forEach(addType);
    specials.forEach(addType);
  })
  .then(addTribes);
