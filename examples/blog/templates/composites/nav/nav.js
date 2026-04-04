document.querySelectorAll('[data-nav-toggle]').forEach(function (toggle) {
  var nav = toggle.closest('[data-nav]');
  if (!nav) return;
  var links = nav.querySelector('[data-nav-links]');
  if (!links) return;

  toggle.addEventListener('click', function () {
    var expanded = toggle.getAttribute('aria-expanded') === 'true';
    toggle.setAttribute('aria-expanded', String(!expanded));
    links.classList.toggle('nav-links-open');
  });
});
