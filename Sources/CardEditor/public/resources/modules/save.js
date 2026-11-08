/* global domtoimage */
export default function save(element, as) {
  element.classList.toggle('saving', true);
  image(element).then((url) => {
    const link = document.createElement('a');
    link.download = `${as || element.querySelector('.name input').value || 'undercard'}.png`;
    link.href = url;
    link.click();
  }).catch((error) => console.error('Ooops.', error))
    .then(() => {
      element.classList.toggle('saving', false);
    });
}

export function image(element) {
  return domtoimage.toPng(element, { filter });
}

function filter(node) {
  if (node.classList && node.classList.contains('no-save')) {
    return false;
  }
  return (node.nodeName !== 'IMG' || node.src);
}
