# Koskidex Web UI 🎨

A modern, glassmorphic exploration interface for Koskidex, built with **React**, **TypeScript**, and **Tailwind CSS**.

## ✨ Features

- **Real-time Search**: Instant results as you type with sub-10ms latency (via Koskidex backend).
- **Multi-language**: Seamless switching between English and Italian.
- **Interactive Demos**: Populate indexes with movies or products using one click.
- **Integration Snippets**: Ready-to-use code examples for various languages.
- **Responsive Design**: Mobile-friendly glassmorphic interface.

## 🛠️ Development

### Prerequisites
- Node.js 18+
- Koskidex backend running (default: `http://localhost:7700`)

### Setup
```bash
npm install
```

### Run Locally
```bash
npm run dev
```

### Environment Variables
- `VITE_API_URL`: The URL of your Koskidex backend. Defaults to `http://localhost:7700`.

## 📦 Build
```bash
npm run build
```
The static files will be generated in the `dist/` directory, ready to be served by Nginx or any static host.
