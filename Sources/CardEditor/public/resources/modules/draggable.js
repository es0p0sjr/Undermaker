import 'https://cdnjs.cloudflare.com/ajax/libs/draggable/1.0.0-beta.8/draggable.bundle.js';
import { editing } from './card.js';

const {
  Draggable,
  // Sensors,
  Sortable,
} = window.Draggable;
const instance = new Sortable([], {
  draggable: '.cardWrapper',
  classes: {
    'source:dragging': 'dragging',
  },
  // handle: '.name',
}).removePlugin(Draggable.Plugins.Focusable, Draggable.Plugins.Announcement)
  // .removeSensor(Sensors.TouchSensor)
  .on('drag:start', (e) => {
    if (editing || !e.sourceContainer.parentElement.classList.contains('sortmode')) e.cancel();
  });

export default function setup(group) {
  instance.addContainer(group.querySelector('.cards'));
  group.querySelector('.sidebar .sort').parentElement.onclick = () => {
    group.classList.toggle('sortmode');
  };
}

export function isDragging() {
  return instance && instance.isDragging();
}
