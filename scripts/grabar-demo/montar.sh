#!/usr/bin/env bash
# Monta el video demo a partir de las dos escenas grabadas por Playwright:
#   scripts/grabar-demo/montar.sh <app.webm> <tarjetas.webm> <salida_dir>
# Genera <salida_dir>/demo.mp4 (H.264, 1280×720) y <salida_dir>/demo.gif (vista previa ≤ 10 MB).
set -euo pipefail
app="$1"; tarjetas="$2"; out="$3"
mkdir -p "$out"
fade=0.6
dur_app=$(ffprobe -v error -show_entries format=duration -of csv=p=0 "$app")
offset=$(awk -v d="$dur_app" -v f="$fade" 'BEGIN{printf "%.2f", d-f}')

# MP4: escena de la app + fundido + tarjetas. 30 fps constantes para que xfade no desincronice.
ffmpeg -y -v error -i "$app" -i "$tarjetas" -filter_complex \
  "[0:v]fps=30,format=yuv420p,setpts=PTS-STARTPTS[a];[1:v]fps=30,format=yuv420p,setpts=PTS-STARTPTS[b];[a][b]xfade=transition=fade:duration=${fade}:offset=${offset}[v]" \
  -map "[v]" -c:v libx264 -preset slow -crf 24 -movflags +faststart -an "$out/demo.mp4"

# GIF: el mismo video a 1,5× de velocidad, 10 fps y 880 px, con paleta propia.
ffmpeg -y -v error -i "$out/demo.mp4" -filter_complex \
  "setpts=PTS/1.5,fps=10,scale=880:-1:flags=lanczos,split[x][y];[x]palettegen=max_colors=128:stats_mode=diff[p];[y][p]paletteuse=dither=bayer:bayer_scale=4:diff_mode=rectangle" \
  "$out/demo.gif"

for f in demo.mp4 demo.gif; do
  printf '%s  %s  %ss\n' "$f" "$(du -h "$out/$f" | cut -f1)" "$(ffprobe -v error -show_entries format=duration -of csv=p=0 "$out/$f")"
done
