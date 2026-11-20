// Escena 1: la app en modo demo, de punta a punta. Cuerpo de `async (page) => { ... }`;
// armar.mjs le antepone comun.js. Parámetros (hash de about:blank): video, destino, app.
const cfg = params(page);
const dir = cfg.video || '.playwright-mcp/video';
const destino = cfg.destino || 'C:\\demo\\video';
const app = cfg.app || 'http://localhost:34115';

const {ctx, p} = await recordingPage(page, dir);
const t0 = Date.now();
const marks = {};
const mark = (k) => { marks[k] = +((Date.now() - t0) / 1000).toFixed(2); };

// Portada
await p.setContent(`<html><body style="margin:0;height:720px;background:#0D0F22;color:#E2E8F0;font-family:Inter,'Segoe UI',sans-serif;
  display:flex;flex-direction:column;justify-content:center;padding:0 96px;box-sizing:border-box">
  <div style="color:#38BDF8;font-weight:700;letter-spacing:.08em;font-size:16px">DEMO · DATOS FICTICIOS</div>
  <div style="font-size:46px;font-weight:700;margin:14px 0 10px;font-family:Consolas,monospace">outlook_descargas_adjuntos</div>
  <div style="font-size:22px;color:#94A3B8">Descarga en bloque los adjuntos de una carpeta de Outlook Classic</div>
  <div style="font-size:18px;color:#7DD3FC;margin-top:26px">Go · Wails v2 · React + TypeScript · COM desde Go</div></body></html>`);
mark('portada');
await wait(p, 3200);

// Arranque
await p.goto(app);
const tree = p.getByRole('treeitem', {name: /Facturas proveedores/});
await tree.waitFor();
mark('app');
await wait(p, 600);
await caption(p, 'Modo demo: una fuente de correos simulada, sin Outlook ni datos reales');
await wait(p, 2200);
await clickOn(p, p.getByRole('button', {name: 'Cambiar a tema oscuro'}));
await caption(p, 'Tema claro y oscuro, guardado entre sesiones');
await wait(p, 1800);

// 1. Explorar
mark('explorar');
await caption(p, '1 · Explora cuentas y carpetas: las subcarpetas cargan bajo demanda');
await clickOn(p, p.getByRole('treeitem', {name: /Proyectos/}).getByRole('button', {name: /^(Expandir|Contraer)$/}));
await wait(p, 1400);
// Las raíces (cuentas y PST) son grupos con su botón, no treeitems.
await clickOn(p, p.getByRole('group').filter({hasText: 'Archivo local (PST)'}).last().getByRole('button', {name: 'Expandir'}).first());
await wait(p, 1200);
await clickOn(p, tree);
await wait(p, 1200);

// 2. Configurar
mark('configurar');
await caption(p, '2 · Rango de fechas, filtro por extensiones y organización por mes');
await clickOn(p, p.getByRole('button', {name: 'Últimos 30 días'}));
await wait(p, 500);
const ext = p.getByPlaceholder('+ agregar');
await clickOn(p, ext);
await p.keyboard.type('pdf', {delay: 90});
await p.keyboard.press('Enter');
await p.keyboard.type('xlsx', {delay: 90});
await p.keyboard.press('Enter');
await wait(p, 500);
await clickOn(p, p.getByRole('radio', {name: 'Plana'}));
await wait(p, 900);
await clickOn(p, p.getByRole('radio', {name: 'Por mes'}));
await wait(p, 700);
const dest = p.locator('input.input.mono').first();
await clickOn(p, dest);
await p.keyboard.press('Control+A');
await p.keyboard.type(destino, {delay: 45});
await wait(p, 900);

// 3. Descargar, cancelar y relanzar
mark('descargar');
await caption(p, '3 · El job corre en segundo plano: progreso por fase y actividad en vivo');
await clickOn(p, p.getByRole('button', {name: 'Iniciar descarga'}));
const pct = async () => {
  const t = (await p.locator('.titlebar-status').textContent({timeout: 300}).catch(() => '')) || '';
  const m = t.match(/(\d+)\s*%/);
  return m ? +m[1] : -1;
};
for (let i = 0; i < 300 && (await pct()) < 12; i++) await wait(p, 100);
await wait(p, 1200);
mark('cancelar');
await caption(p, 'Cancelar es inmediato: la UI vuelve al instante y se escribe el reporte parcial');
await clickOn(p, p.getByRole('button', {name: 'Cancelar descarga'}));
await p.getByText('Descarga cancelada').waitFor();
await wait(p, 2600);
await clickOn(p, p.getByRole('button', {name: 'Nueva descarga'}));
await wait(p, 900);
mark('relanzar');
await caption(p, 'Se relanza con la misma configuración; los nombres repetidos llevan sufijo _N');
await clickOn(p, p.getByRole('button', {name: 'Iniciar descarga'}));
await wait(p, 2500);
await caption(p, 'Lo que no se descarga queda registrado con su motivo: nada se omite en silencio');
await p.getByText('Descarga completada').waitFor({timeout: 120000});
mark('resumen');

// 4. Resumen
await caption(p, '4 · Resumen y reporte Excel. Al terminar, la app avisa con un sonido y parpadea');
await wait(p, 2800);
await clickOn(p, p.getByRole('button', {name: 'Abrir reporte'}), {click: false});
await wait(p, 1200);
await clickOn(p, p.getByRole('button', {name: 'Abrir carpeta'}), {click: false});
await wait(p, 1800);
mark('fin');

const video = p.video();
await ctx.close();
return {video: await video.path(), marks};
