import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import wails from "@wailsio/runtime/plugins/vite";
import { fileURLToPath, URL } from "node:url";
import { resolve, join, extname } from "node:path";
import {
  createReadStream,
  existsSync,
  statSync,
  readdirSync,
  mkdirSync,
  copyFileSync,
} from "node:fs";

const vditorDist = resolve(
  fileURLToPath(new URL(".", import.meta.url)),
  "node_modules/vditor/dist"
);

const MIME: Record<string, string> = {
  ".css": "text/css",
  ".js": "application/javascript",
  ".mjs": "application/javascript",
  ".json": "application/json",
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".gif": "image/gif",
  ".svg": "image/svg+xml",
  ".woff": "font/woff",
  ".woff2": "font/woff2",
  ".ttf": "font/ttf",
  ".eot": "application/vnd.ms-fontobject",
  ".map": "application/json",
};

/**
 * Vditor 在运行时通过 `cdn` 选项动态加载 i18n / 主题 / katex / mermaid 等
 * 子资源。这个插件在 dev 模式以中间件拦截 `/vditor/dist/**` 请求并直接从
 * `node_modules/vditor/dist` 读取返回；在 build 模式于产物写入后把整个
 * `dist` 目录拷贝到 `dist/vditor/dist`，使嵌入资源路径与 Vditor 的拼接
 * `${cdn}/dist/...` 完全匹配。
 */
function vditorAssets() {
  return {
    name: "vditor-assets",
    configureServer(server: any) {
      server.middlewares.use((req: any, res: any, next: any) => {
        const url: string = req.url || "";
        if (!url.startsWith("/vditor/dist/")) return next();
        const relPath = decodeURIComponent(url.replace("/vditor/dist/", "").split("?")[0]);
        const filePath = join(vditorDist, relPath);
        if (!existsSync(filePath) || statSync(filePath).isDirectory()) return next();
        res.setHeader("Content-Type", MIME[extname(filePath)] || "application/octet-stream");
        createReadStream(filePath).pipe(res);
      });
    },
    writeBundle() {
      const dest = resolve(
        fileURLToPath(new URL(".", import.meta.url)),
        "dist/vditor/dist"
      );
      function copyDir(src: string, d: string) {
        if (!existsSync(d)) mkdirSync(d, { recursive: true });
        for (const entry of readdirSync(src, { withFileTypes: true })) {
          const s = join(src, entry.name);
          const t = join(d, entry.name);
          if (entry.isDirectory()) copyDir(s, t);
          else copyFileSync(s, t);
        }
      }
      copyDir(vditorDist, dest);
    },
  };
}

// https://vitejs.dev/config/
export default defineConfig({
  // 相对路径：Wails 内嵌 webview 友好（参考 SuperSender）
  base: "./",
  server: {
    host: "::",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
    sourcemap: false,
    target: "es2020",
    chunkSizeWarningLimit: 2048,
    // esbuild CSS 压缩默认合并 vendor 前缀：当 backdrop-filter 与 -webkit-backdrop-filter
    // 同时存在时，会移除标准 backdrop-filter 只保留 -webkit- 前缀。但 WebView2 的 GPU
    // 合成管线只识别标准 backdrop-filter，导致 build 后毛玻璃失效（dev 不压缩故正常）。
    // 禁用 CSS 压缩以保留标准属性。桌面应用对 CSS 体积不敏感。
    cssMinify: false,
  },
  plugins: [vue(), wails("./bindings"), vditorAssets()],
  resolve: {
    alias: {
      "@app-config": fileURLToPath(new URL("../app.config.json", import.meta.url)),
    },
  },
  optimizeDeps: {
    include: ["vditor"],
  },
});
