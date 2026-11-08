let val = false;

export function toggle() {
  val = !val;
  document.body.classList.toggle('showExtras', val);
}

export default function value() {
  return val;
}

const params = new URLSearchParams(location.search);
if (params.has('extras')) {
  toggle();
}
