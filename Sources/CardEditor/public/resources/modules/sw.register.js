export default function register() {
  return new Promise((res) => {
    window.addEventListener('load', () => {
      if ('serviceWorker' in navigator) {
        res(navigator.serviceWorker.register('/service-worker.js'));
      } else {
        res();
      }
    });
  });
}
