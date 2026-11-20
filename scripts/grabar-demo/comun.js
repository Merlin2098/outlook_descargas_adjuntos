// Fragmento compartido por las escenas (se concatena delante de cada una en grabar.md; Playwright MCP
// ejecuta un único archivo, sin módulos). Define el cursor visible, los rótulos y el arranque del contexto.

const CURSOR_SCRIPT = `
  addEventListener('DOMContentLoaded', () => {
    const c = document.createElement('div');
    c.id = 'demo-cursor';
    c.innerHTML = '<svg width="22" height="22" viewBox="0 0 24 24"><path d="M4 2l16 9.5-7 1.6-3.6 6.6z" fill="#fff" stroke="#0D0F22" stroke-width="1.6" stroke-linejoin="round"/></svg>';
    Object.assign(c.style, {position: 'fixed', left: '-40px', top: '-40px', zIndex: 2147483647, pointerEvents: 'none',
      transition: 'transform .08s', filter: 'drop-shadow(0 1px 2px rgba(0,0,0,.5))'});
    document.documentElement.appendChild(c);
    addEventListener('mousemove', (e) => { c.style.left = e.clientX - 3 + 'px'; c.style.top = e.clientY - 2 + 'px'; }, true);
    addEventListener('mousedown', (e) => {
      c.style.transform = 'scale(.82)';
      const r = document.createElement('div');
      Object.assign(r.style, {position: 'fixed', left: e.clientX - 18 + 'px', top: e.clientY - 18 + 'px', width: '36px', height: '36px',
        borderRadius: '50%', border: '3px solid #38BDF8', zIndex: 2147483646, pointerEvents: 'none', opacity: '1',
        transition: 'transform .45s ease-out, opacity .45s ease-out'});
      document.documentElement.appendChild(r);
      requestAnimationFrame(() => { r.style.transform = 'scale(1.8)'; r.style.opacity = '0'; });
      setTimeout(() => r.remove(), 500);
    }, true);
    addEventListener('mouseup', () => { c.style.transform = ''; }, true);
  });
`;

/** Parámetros de la escena, tomados del hash de la página actual del MCP (about:blank#clave=valor&...). */
function params(page) {
  const hash = page.url().split('#')[1] || '';
  return Object.fromEntries(new URLSearchParams(hash));
}

/** Abre un contexto que graba video a 1280×720 con el cursor visible. */
async function recordingPage(page, dir) {
  const ctx = await page.context().browser().newContext({
    viewport: {width: 1280, height: 720}, deviceScaleFactor: 1,
    recordVideo: {dir, size: {width: 1280, height: 720}},
  });
  await ctx.addInitScript(CURSOR_SCRIPT);
  const p = await ctx.newPage();
  p.setDefaultTimeout(30000);
  return {ctx, p};
}

const wait = (p, ms) => p.waitForTimeout(ms);

/** Mueve el cursor hasta el centro del elemento con una trayectoria suave y hace clic. */
async function clickOn(p, locator, {pause = 250, click = true} = {}) {
  const box = await locator.boundingBox();
  if (!box) throw new Error('elemento sin caja: ' + locator);
  await p.mouse.move(box.x + box.width / 2, box.y + box.height / 2, {steps: 22});
  await wait(p, pause);
  if (click) await p.mouse.click(box.x + box.width / 2, box.y + box.height / 2);
}

/** Rótulo explicativo sobre el video (no forma parte de la app). */
async function caption(p, text) {
  await p.evaluate((t) => {
    let el = document.getElementById('demo-caption');
    if (!el) {
      el = document.createElement('div');
      el.id = 'demo-caption';
      Object.assign(el.style, {position: 'fixed', left: '50%', bottom: '86px', transform: 'translateX(-50%)', zIndex: 2147483645,
        background: 'rgba(13,15,34,.92)', color: '#E2E8F0', border: '1px solid #38BDF8', borderRadius: '999px',
        padding: '9px 20px', font: '600 15px Inter, system-ui, sans-serif', pointerEvents: 'none',
        boxShadow: '0 6px 24px rgba(0,0,0,.35)', transition: 'opacity .25s', whiteSpace: 'nowrap'});
      document.body.appendChild(el);
    }
    el.style.opacity = '0';
    setTimeout(() => { el.textContent = t; el.style.opacity = t ? '1' : '0'; }, 180);
  }, text);
  await wait(p, 250);
}
