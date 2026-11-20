// Escena 2: tarjetas finales generadas por `go run ./scripts/grabar-demo/tarjetas` con los datos reales
// de la descarga grabada en la escena 1. Parámetros (hash): video, tarjetas (URL file:/// de la carpeta).
const cfg = params(page);
const dir = cfg.video || '.playwright-mcp/video';
if (!cfg.tarjetas) throw new Error('falta el parámetro tarjetas');

const {ctx, p} = await recordingPage(page, dir);
const show = async (name, ms) => {
  await p.goto(`${cfg.tarjetas}/${name}`);
  await p.evaluate(() => document.fonts.ready);
  await wait(p, ms);
};
await show('archivos.html', 5200);
await show('reporte.html', 6500);
await show('terminal.html', 5500);

const video = p.video();
await ctx.close();
return {video: await video.path()};
