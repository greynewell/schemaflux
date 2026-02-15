// --- Copy button handler ---
document.querySelectorAll('[data-copy]').forEach(function(btn) {
  btn.addEventListener('click', function() {
    var text = btn.dataset.copy;
    navigator.clipboard.writeText(text).then(function() {
      var span = btn.querySelector('span');
      if (span) {
        var original = span.textContent;
        span.textContent = 'Copied!';
        setTimeout(function() { span.textContent = original; }, 2000);
      }
    });
  });
});

// --- Smooth scroll for anchor links ---
document.querySelectorAll('a[href^="#"]').forEach(function(link) {
  link.addEventListener('click', function(e) {
    var target = document.querySelector(link.getAttribute('href'));
    if (target) {
      e.preventDefault();
      target.scrollIntoView({ behavior: 'smooth', block: 'start' });
      history.pushState(null, '', link.getAttribute('href'));
    }
  });
});

// --- Active nav highlighting ---
(function() {
  var path = window.location.pathname;
  var links = document.querySelectorAll('.site-header nav a');
  for (var i = 0; i < links.length; i++) {
    var href = links[i].getAttribute('href');
    if (href && !href.startsWith('http') && path.indexOf(href) === 0 && href !== '/') {
      links[i].style.color = 'var(--color-primary)';
    }
  }
})();
