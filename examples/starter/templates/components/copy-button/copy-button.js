// Loaded as <script type="module"> via {% asset type="module" %}.
// The bare specifier below is resolved by the page's importmap, which Grove
// builds from the asset manifest at server start — so this import survives
// content hashing without a bundler.
import { copyText } from "components/copy-button/clipboard";

function wire(root) {
  const btn = root.querySelector('.copy-button__btn');
  const status = root.querySelector('.copy-button__status');
  if (!btn) return;
  btn.addEventListener('click', () => {
    const value = root.dataset.copyValue || '';
    copyText(value).then(() => {
      root.classList.add('copy-button--copied');
      if (status) status.textContent = 'Copied';
      setTimeout(() => {
        root.classList.remove('copy-button--copied');
        if (status) status.textContent = '';
      }, 1800);
    });
  });
}

document.querySelectorAll('.copy-button').forEach(wire);
