// Escena 2: los archivos descargados y el cierre. Usa las páginas que genera
// `go run ./scripts/grabar-demo/tarjetas` con los datos reales de la descarga grabada en la escena 1:
// la carpeta de destino, la factura PDF (el archivo real, en el visor del navegador), su XML, un estado
// de cuenta XLSX, el reporte y `go test`. Parámetros (hash): video, tarjetas (URL file:/// de la carpeta).
const cfg = params(page);
const dir = cfg.video || '.playwright-mcp/video';
if (!cfg.tarjetas) throw new Error('falta el parámetro tarjetas');

const {ctx, p} = await recordingPage(page, dir);
const goto = async (name) => {
  await p.goto(`${cfg.tarjetas}/${name}`);
  await p.evaluate(() => document.fonts.ready);
};
const row = (sel) => p.locator(`.r${sel}`).first();
const open = async (locator) => {
  await clickOn(p, locator);
  await wait(p, 450);
  const box = await locator.boundingBox();
  await p.mouse.dblclick(box.x + box.width / 2, box.y + box.height / 2);
  await p.waitForLoadState('load');
};
const back = async () => {
  await p.goBack();
  await p.evaluate(() => document.fonts.ready);
  await wait(p, 500);
};

// Carpeta de destino
await goto('explorador.html');
await caption(p, '5 · Los adjuntos quedan en disco, organizados por mes');
await wait(p, 2000);
await open(p.locator('.r:has(.ext:text-is("DIR"))').last()); // la última carpeta (el mes más reciente)
await wait(p, 1200);
const lista = p.locator('.list');
const caja = await lista.boundingBox();
await p.mouse.move(caja.x + caja.width / 2, caja.y + caja.height / 2, {steps: 15});
await caption(p, 'Facturas en PDF y XML, estados de cuenta, consultas… con su nombre original');
await wait(p, 2200);

// Baja con la rueda del ratón hasta que la fila quede visible dentro de la lista.
const scrollTo = async (locator) => {
  for (let i = 0; i < 60; i++) {
    const r = await locator.boundingBox();
    if (r && r.y > caja.y + 40 && r.y + r.height < caja.y + caja.height - 60) return;
    await p.mouse.wheel(0, 90);
    await wait(p, 70);
  }
};

// Factura PDF: el archivo real en el visor de PDF del navegador
await scrollTo(row('.hot:has(.ext.pdf)'));
await wait(p, 600);
await open(row('.hot:has(.ext.pdf)'));
await wait(p, 5200);
await back();

// La misma factura en XML, si el filtro de la escena 1 la incluyó
if (await p.locator('.r.hot:has(.ext.xml)').count()) {
  await open(row('.hot:has(.ext.xml)'));
  await wait(p, 4200);
  await back();
}

// Estado de cuenta en Excel
await open(row('.hot:has(.ext.xlsx)'));
await wait(p, 4800);

// Cierre
await goto('reporte.html');
await wait(p, 6000);
await goto('terminal.html');
await wait(p, 5000);

const video = p.video();
await ctx.close();
return {video: await video.path()};
