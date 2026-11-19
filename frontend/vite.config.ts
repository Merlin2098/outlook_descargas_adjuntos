/// <reference types="vitest" />
import {defineConfig} from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  // css: los tests leen tokens.css?raw para comprobar el contraste del tema.
  test: {environment: 'node', css: true},
})
