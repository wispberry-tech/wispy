document.querySelectorAll('[data-button]').forEach(function (btn) {
  btn.addEventListener('click', function () {
    if (btn.classList.contains('btn-loading')) return;
    btn.classList.add('btn-loading');
    btn.setAttribute('aria-busy', 'true');
  });
});
