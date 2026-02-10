// --- Share & Copy-link handlers ---
document.querySelectorAll('[data-share]').forEach(function(btn){
  if(!navigator.share){btn.style.display='none';return}
  btn.addEventListener('click',function(){
    navigator.share({title:btn.dataset.title,text:btn.dataset.text,url:btn.dataset.url}).catch(function(){})
  })
});
document.querySelectorAll('[data-copy-link]').forEach(function(btn){
  btn.addEventListener('click',function(){
    navigator.clipboard.writeText(btn.dataset.url).then(function(){
      var s=btn.querySelector('span');var o=s.textContent;s.textContent='Copied!';setTimeout(function(){s.textContent=o},2000)
    })
  })
});

// --- Servings slider ---
(function(){
  var article = document.querySelector('article[data-base-servings]');
  if (!article) return;
  var baseServings = parseInt(article.dataset.baseServings, 10);
  var currentServings = baseServings;
  var slider = document.querySelector('.servings-range');
  var numInput = document.querySelector('.servings-input');
  if (!slider || !numInput) return;
  var ingredients = article.querySelectorAll('.ingredient[data-base-qty]');
  var FRAC = {
    '0.125':'\u215B','0.2':'\u2155','0.25':'\u00BC',
    '0.333':'\u2153','0.5':'\u00BD','0.667':'\u2154','0.75':'\u00BE'
  };
  function snapFraction(f) {
    if (f < 0.0625) return 0;
    var targets = [0.125,0.2,0.25,0.333,0.5,0.667,0.75,1];
    var best = 0, bestDist = Math.abs(f);
    for (var i = 0; i < targets.length; i++) {
      var d = Math.abs(f - targets[i]);
      if (d < bestDist) { bestDist = d; best = targets[i]; }
    }
    return bestDist < 0.05 ? best : f;
  }
  function formatQty(n) {
    if (n === 0) return '0';
    var whole = Math.floor(n);
    var frac = snapFraction(n - whole);
    var key = frac.toFixed(3).replace(/0+$/,'').replace(/\.$/,'');
    var sym = FRAC[key] || '';
    if (whole === 0 && sym) return sym;
    if (whole > 0 && sym) return whole + sym;
    if (frac === 0) return '' + whole;
    var r = Math.round(n * 10) / 10;
    return r === Math.floor(r) ? '' + r : r.toFixed(1);
  }
  function setServings(n) {
    n = Math.max(1, Math.min(99, Math.round(n)));
    currentServings = n;
    slider.value = Math.min(n, parseInt(slider.max, 10));
    numInput.value = n;
    var pct = ((n - 1) / (parseInt(slider.max, 10) - 1)) * 100;
    slider.style.background = 'linear-gradient(to right, var(--color-primary) ' + pct + '%, var(--color-border) ' + pct + '%)';
    var ratio = n / baseServings;
    for (var i = 0; i < ingredients.length; i++) {
      var el = ingredients[i];
      var qtyEl = el.querySelector('.ingredient-qty');
      if (qtyEl) {
        qtyEl.textContent = formatQty(parseFloat(el.dataset.baseQty) * ratio);
      }
    }
  }
  slider.addEventListener('input', function() { setServings(parseInt(slider.value, 10)); });
  numInput.addEventListener('input', function() {
    var v = parseInt(numInput.value, 10);
    if (!isNaN(v) && v >= 1) setServings(v);
  });
  numInput.addEventListener('blur', function() { numInput.value = currentServings; });
  document.querySelectorAll('.servings-btn').forEach(function(btn) {
    btn.addEventListener('click', function() {
      if (btn.dataset.dir === '+') setServings(currentServings + 1);
      else setServings(currentServings - 1);
    });
  });
  setServings(baseServings);
})();

// --- Copy-list & Buy-all handlers ---
document.querySelectorAll('[data-copy-list]').forEach(function(btn) {
  btn.addEventListener('click', function() {
    var card = btn.closest('.card');
    if (!card) return;
    var names = Array.from(card.querySelectorAll('.ingredient-name, .gear-name'))
      .map(function(el) { return el.textContent.trim(); });
    if (names.length === 0) return;
    navigator.clipboard.writeText(names.join('\n')).then(function() {
      var s = btn.querySelector('span');
      var o = s.textContent;
      s.textContent = 'Copied!';
      setTimeout(function() { s.textContent = o; }, 2000);
    });
  });
});

// --- Favorites handler (localStorage) ---
(function() {
  var STORAGE_KEY = 'claudechef_favorites';
  function getFavorites() {
    try {
      var data = localStorage.getItem(STORAGE_KEY);
      return data ? JSON.parse(data) : [];
    } catch(e) { return []; }
  }
  function saveFavorites(arr) {
    try { localStorage.setItem(STORAGE_KEY, JSON.stringify(arr)); } catch(e) {}
  }
  function isSaved(slug) {
    return getFavorites().indexOf(slug) !== -1;
  }
  function toggleFavorite(slug) {
    var favs = getFavorites();
    var idx = favs.indexOf(slug);
    if (idx === -1) { favs.push(slug); }
    else { favs.splice(idx, 1); }
    saveFavorites(favs);
    return idx === -1;
  }
  document.querySelectorAll('[data-favorite]').forEach(function(btn) {
    var slug = btn.dataset.slug;
    if (!slug) return;
    var span = btn.querySelector('span');
    if (isSaved(slug)) {
      btn.classList.add('saved');
      if (span) span.textContent = 'Saved';
    }
    btn.addEventListener('click', function() {
      var nowSaved = toggleFavorite(slug);
      if (nowSaved) {
        btn.classList.add('saved');
        if (span) span.textContent = 'Saved';
      } else {
        btn.classList.remove('saved');
        if (span) span.textContent = 'Save';
      }
    });
  });
})();
