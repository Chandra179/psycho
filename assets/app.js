document.body.addEventListener('htmx:responseError', function (e) {
  const alert = document.createElement('div');
  alert.setAttribute('role', 'alert');
  alert.className = 'rounded-2xl border border-rose-200 bg-rose-50 p-5 text-sm leading-relaxed text-rose-800';

  const title = document.createElement('p');
  title.className = 'font-semibold';
  title.textContent = 'We could not analyze this writing.';

  const message = document.createElement('p');
  message.className = 'mt-1';
  message.textContent = e.detail.xhr.responseText.trim() || 'Please try again.';

  alert.append(title, message);
  const wrapper = document.createElement('div');
  wrapper.className = 'w-full max-w-[60rem] mx-auto px-4 sm:px-6 lg:px-8';
  wrapper.append(alert);
  document.getElementById('result').replaceChildren(wrapper);
});
document.body.addEventListener('htmx:afterSwap', function (e) {
  if (e.detail.target.id === 'result') {
    e.detail.target.scrollIntoView({ behavior: 'smooth', block: 'start' });
  }
});
