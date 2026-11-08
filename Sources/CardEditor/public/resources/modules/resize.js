export default function resize(container, {
  size = 12,
  minsize = 7,
  steps = 0.5,
  height = true,
} = {}) {
  const type = height ? 'Height' : 'Width';
  let newSize = size;
  container.style.fontSize = '';
  while (container[`scroll${type}`] > container[`client${type}`] && newSize - steps >= minsize) {
    newSize -= steps;
    container.style.fontSize = `${newSize}px`;
  }
}
