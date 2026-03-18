import {
  ArrowLeft,
  Book,
  Box,
  Code,
  Container,
  Globe,
  Key,
  Server,
  Terminal,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

function CodeBlock({ children, label }: { children: string; label?: string }) {
  return (
    <div className="relative">
      {label && (
        <div className="absolute top-0 right-0 px-2.5 py-1 bg-slate-800/80 text-[10px] text-slate-500 font-mono rounded-bl-lg rounded-tr-xl border-b border-l border-white/5">
          {label}
        </div>
      )}
      <pre className="bg-[#0B1120] rounded-xl p-4 md:p-5 font-mono text-[11px] sm:text-[13px] text-slate-300 overflow-x-auto border border-white/5 leading-relaxed">
        <code className="whitespace-pre">{children}</code>
      </pre>
    </div>
  );
}

function SectionCard({ children }: { children: React.ReactNode }) {
  return (
    <div className="glass-effect p-5 md:p-8 rounded-2xl md:rounded-[2rem] border-white/5 bg-slate-900/40">
      {children}
    </div>
  );
}

const NAV_ITEMS = [
  { id: "quickstart", icon: Terminal, color: "text-emerald-500" },
  { id: "docker-compose", icon: Container, color: "text-blue-500" },
  { id: "php", icon: Server, color: "text-orange-400" },
  { id: "js", icon: Globe, color: "text-yellow-400" },
  { id: "python", icon: Code, color: "text-green-400" },
  { id: "api", icon: Box, color: "text-purple-400" },
  { id: "architecture", icon: Book, color: "text-blue-500" },
] as const;

const SECTION_TITLE_KEYS: Record<string, string> = {
  quickstart: "docs.sections.quickstart.title",
  "docker-compose": "docs.sections.docker_compose.title",
  php: "docs.sections.php.title",
  js: "docs.sections.js.title",
  python: "docs.sections.python.title",
  api: "docs.sections.api.title",
  architecture: "docs.sections.architecture.title",
};

export default function Documentation() {
  const { t } = useTranslation();
  const [activeSection, setActiveSection] = useState("quickstart");
  const sectionRefs = useRef<Record<string, HTMLElement | null>>({});
  const mobileNavRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    window.scrollTo(0, 0);
  }, []);

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            setActiveSection(entry.target.id);
          }
        }
      },
      { rootMargin: "-20% 0px -60% 0px", threshold: 0 }
    );

    for (const item of NAV_ITEMS) {
      const el = sectionRefs.current[item.id];
      if (el) observer.observe(el);
    }

    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    if (!mobileNavRef.current) return;
    const activeBtn = mobileNavRef.current.querySelector(`[data-section="${activeSection}"]`);
    if (activeBtn) {
      activeBtn.scrollIntoView({ behavior: "smooth", block: "nearest", inline: "center" });
    }
  }, [activeSection]);

  const scrollToSection = (id: string) => {
    const el = sectionRefs.current[id];
    if (el) {
      el.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  };

  const setSectionRef = (id: string) => (el: HTMLElement | null) => {
    sectionRefs.current[id] = el;
  };

  return (
    <div className="pt-24 md:pt-32 pb-16 md:pb-24 min-h-screen relative overflow-x-clip">
      <div className="absolute top-0 right-0 w-1/2 h-1/2 bg-blue-500/5 blur-[120px] rounded-full pointer-events-none"></div>
      <div className="absolute bottom-0 left-0 w-1/2 h-1/2 bg-purple-500/5 blur-[120px] rounded-full pointer-events-none"></div>

      <div className="container mx-auto px-4 relative z-10">
        <div className="max-w-5xl mx-auto">
          <Link
            to="/"
            className="inline-flex items-center gap-2 text-slate-400 hover:text-white mb-8 md:mb-12 transition-colors group"
          >
            <ArrowLeft className="w-4 h-4 group-hover:-translate-x-1 transition-transform" />
            {t("docs.back_home")}
          </Link>

          <div className="mb-12 md:mb-20">
            <div className="flex items-center gap-3 md:gap-4 mb-4 md:mb-6">
              <div className="w-10 h-10 md:w-12 md:h-12 bg-blue-500/10 rounded-xl md:rounded-2xl flex items-center justify-center shrink-0">
                <Book className="w-5 h-5 md:w-6 md:h-6 text-blue-500" />
              </div>
              <h1 className="text-3xl md:text-5xl font-black tracking-tight text-white">
                {t("docs.title")}
              </h1>
            </div>
            <p className="text-base md:text-xl text-slate-400 font-light max-w-3xl leading-relaxed">
              {t("docs.subtitle")}
            </p>
          </div>
        </div>

        {/* Mobile horizontal nav */}
        <div
          ref={mobileNavRef}
          className="lg:hidden flex gap-2 overflow-x-auto pb-4 mb-8 scrollbar-hide -mx-4 px-4 sticky top-[72px] z-30 bg-gradient-to-b from-[#0f172a] via-[#0f172a]/95 to-transparent pt-2"
        >
          {NAV_ITEMS.map((item) => {
            const Icon = item.icon;
            const isActive = activeSection === item.id;
            return (
              <button
                key={item.id}
                data-section={item.id}
                onClick={() => scrollToSection(item.id)}
                className={`flex items-center gap-1.5 px-3 py-2 rounded-lg text-xs font-semibold whitespace-nowrap transition-all shrink-0 ${
                  isActive
                    ? "bg-white/10 text-white border border-white/10"
                    : "bg-white/5 text-slate-500 border border-transparent hover:text-slate-300 hover:bg-white/[0.07]"
                }`}
              >
                <Icon className={`w-3.5 h-3.5 ${isActive ? item.color : ""}`} />
                {t(SECTION_TITLE_KEYS[item.id])}
              </button>
            );
          })}
        </div>

        {/* Desktop layout: sidebar + content */}
        <div className="flex gap-10 max-w-[1280px] mx-auto">
          {/* Sticky sidebar — desktop only */}
          <aside className="hidden lg:block w-56 shrink-0">
            <nav className="sticky top-32 flex flex-col gap-1">
              {NAV_ITEMS.map((item) => {
                const Icon = item.icon;
                const isActive = activeSection === item.id;
                return (
                  <button
                    key={item.id}
                    onClick={() => scrollToSection(item.id)}
                    className={`flex items-center gap-2.5 px-3 py-2.5 rounded-xl text-left text-sm font-medium transition-all ${
                      isActive
                        ? "bg-white/10 text-white border border-white/10 shadow-lg shadow-white/[0.02]"
                        : "text-slate-500 hover:text-slate-300 hover:bg-white/5 border border-transparent"
                    }`}
                  >
                    <Icon className={`w-4 h-4 shrink-0 ${isActive ? item.color : ""}`} />
                    <span className="truncate">{t(SECTION_TITLE_KEYS[item.id])}</span>
                  </button>
                );
              })}
            </nav>
          </aside>

          {/* Main content */}
          <div className="flex-1 min-w-0 max-w-5xl grid gap-16 md:gap-20">

            {/* ──────────── Quick Start ──────────── */}
            <section id="quickstart" ref={setSectionRef("quickstart")} className="scroll-mt-32">
              <div className="flex items-center gap-3 mb-6 md:mb-8">
                <Terminal className="w-7 h-7 md:w-8 md:h-8 text-emerald-500" />
                <h2 className="text-xl md:text-3xl font-black text-white uppercase tracking-wider">
                  {t("docs.sections.quickstart.title")}
                </h2>
              </div>
              <SectionCard>
                <p className="text-slate-300 mb-8 leading-relaxed">
                  {t("docs.sections.quickstart.desc")}
                </p>

                <div className="grid gap-6">
                  <div>
                    <h3 className="text-white font-semibold text-sm mb-3 flex items-center gap-2">
                      <span className="w-6 h-6 rounded-full bg-emerald-500/15 text-emerald-400 text-xs font-bold flex items-center justify-center">1</span>
                      {t("docs.sections.quickstart.step1")}
                    </h3>
                    <CodeBlock label="terminal">{`git clone https://github.com/GeneralKoski/Koskidex.git\ncd Koskidex\ndocker compose up -d`}</CodeBlock>
                    <p className="text-emerald-400/80 text-xs mt-2 font-medium">{t("docs.sections.quickstart.running_at")}</p>
                  </div>

                  <div>
                    <h3 className="text-white font-semibold text-sm mb-3 flex items-center gap-2">
                      <span className="w-6 h-6 rounded-full bg-emerald-500/15 text-emerald-400 text-xs font-bold flex items-center justify-center">2</span>
                      {t("docs.sections.quickstart.step2")}
                    </h3>
                    <CodeBlock label="terminal">{`curl http://localhost:7700/health\n# {"status":"ok","uptime":"5s"}`}</CodeBlock>
                  </div>

                  <div>
                    <h3 className="text-white font-semibold text-sm mb-3 flex items-center gap-2">
                      <span className="w-6 h-6 rounded-full bg-emerald-500/15 text-emerald-400 text-xs font-bold flex items-center justify-center">3</span>
                      {t("docs.sections.quickstart.step3")}
                    </h3>
                    <CodeBlock label="terminal">{`curl -X POST http://localhost:7700/indexes \\\n  -H 'Content-Type: application/json' \\\n  -d '{"name": "movies"}'\n\ncurl -X POST http://localhost:7700/indexes/movies/documents \\\n  -H 'Content-Type: application/json' \\\n  -d '[
  {"id": "1", "title": "The Matrix", "genre": "Sci-Fi", "year": 1999},
  {"id": "2", "title": "Inception", "genre": "Action", "year": 2010},
  {"id": "3", "title": "Interstellar", "genre": "Sci-Fi", "year": 2014}
]'`}</CodeBlock>
                  </div>

                  <div>
                    <h3 className="text-white font-semibold text-sm mb-3 flex items-center gap-2">
                      <span className="w-6 h-6 rounded-full bg-emerald-500/15 text-emerald-400 text-xs font-bold flex items-center justify-center">4</span>
                      {t("docs.sections.quickstart.step4")}
                    </h3>
                    <CodeBlock label="terminal">{`curl "http://localhost:7700/indexes/movies/search?q=matrx"\ncurl "http://localhost:7700/indexes/movies/search?q=interstllar"\ncurl "http://localhost:7700/indexes/movies/search?q=Sci-Fi&filter=year>2000"`}</CodeBlock>
                  </div>
                </div>
              </SectionCard>
            </section>

            {/* ──────────── Docker Compose Integration ──────────── */}
            <section id="docker-compose" ref={setSectionRef("docker-compose")} className="scroll-mt-32">
              <div className="flex items-center gap-3 mb-6 md:mb-8">
                <Container className="w-7 h-7 md:w-8 md:h-8 text-blue-500" />
                <h2 className="text-xl md:text-3xl font-black text-white uppercase tracking-wider">
                  {t("docs.sections.docker_compose.title")}
                </h2>
              </div>
              <SectionCard>
                <p className="text-slate-300 mb-8 leading-relaxed">
                  {t("docs.sections.docker_compose.desc")}
                </p>

                <div className="grid gap-8">
                  <div>
                    <h3 className="text-white font-semibold text-sm mb-3">{t("docs.sections.docker_compose.add_service")}</h3>
                    <CodeBlock label="docker-compose.yml">{`services:
  # ... your existing services (app, db, redis, etc.)

  koskidex:
    build: ./koskidex  # path to the cloned Koskidex repo
    ports:
      - "\${KOSKIDEX_PORT:-7700}:7700"
    volumes:
      - koskidex_data:/data
    restart: unless-stopped
    command: ["/app/koskidex", "--port", "7700", "--data-dir", "/data", "--api-key", "\${KOSKIDEX_API_KEY}"]
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:7700/health"]
      interval: 30s
      timeout: 5s
      retries: 3

volumes:
  koskidex_data:`}</CodeBlock>
                    <p className="text-slate-500 text-xs mt-2">Clone Koskidex inside your project: <span className="text-slate-300 font-mono">git clone https://github.com/GeneralKoski/Koskidex.git koskidex</span></p>
                  </div>

                  <div>
                    <h3 className="text-white font-semibold text-sm mb-3">{t("docs.sections.docker_compose.env_vars")}</h3>
                    <CodeBlock label=".env">{`# Koskidex connection
KOSKIDEX_HOST=http://koskidex:7700
KOSKIDEX_API_KEY=your-secret-key

# Port mapping (optional, default 7700)
KOSKIDEX_PORT=7700`}</CodeBlock>
                    <p className="text-slate-500 text-xs mt-2">Leave <span className="text-slate-300 font-mono">KOSKIDEX_API_KEY</span> empty to disable authentication.</p>
                  </div>

                  <div className="p-5 bg-blue-500/5 rounded-xl border border-blue-500/10">
                    <h4 className="text-blue-400 font-bold text-sm mb-3 flex items-center gap-2">
                      <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 6V2m0 0L8 6m4-4 4 4M6 12H2m0 0 4 4m-4-4 4-4m16 4h-4m4 0-4 4m4-4-4-4M12 22v-4m0 4-4-4m4 4 4-4"/></svg>
                      {t("docs.sections.docker_compose.sail_title")}
                    </h4>
                    <p className="text-slate-400 text-sm mb-4">{t("docs.sections.docker_compose.sail_desc")}</p>
                    <CodeBlock label="docker-compose.yml">{`services:
  laravel.test:
    # ... your Sail config
    depends_on:
      - mysql
      - koskidex
    environment:
      KOSKIDEX_HOST: 'http://koskidex:7700'
      KOSKIDEX_API_KEY: '\${KOSKIDEX_API_KEY}'

  koskidex:
    build: ./koskidex
    command: ["/app/koskidex", "--port", "7700", "--data-dir", "/data", "--api-key", "\${KOSKIDEX_API_KEY}"]
    ports:
      - "\${KOSKIDEX_PORT:-7700}:7700"
    volumes:
      - sail-koskidex:/data
    networks:
      - sail

volumes:
  sail-koskidex:
    driver: local`}</CodeBlock>
                    <div className="mt-3">
                      <CodeBlock label=".env">{`KOSKIDEX_HOST=http://koskidex:7700\nKOSKIDEX_API_KEY=your-secret-key`}</CodeBlock>
                    </div>
                  </div>
                </div>
              </SectionCard>
            </section>

            {/* ──────────── PHP / Laravel ──────────── */}
            <section id="php" ref={setSectionRef("php")} className="scroll-mt-32">
              <div className="flex items-center gap-3 mb-6 md:mb-8">
                <Server className="w-7 h-7 md:w-8 md:h-8 text-orange-400" />
                <h2 className="text-xl md:text-3xl font-black text-white uppercase tracking-wider">
                  {t("docs.sections.php.title")}
                </h2>
              </div>
              <SectionCard>
                <p className="text-slate-300 mb-8 leading-relaxed">
                  {t("docs.sections.php.desc")}
                </p>

                <div className="grid gap-6">
                  <div>
                    <h3 className="text-white font-semibold text-sm mb-3">{t("docs.sections.php.step1_title")}</h3>
                    <p className="text-slate-400 text-sm mb-3">Copy the integration files from the Koskidex repo into your Laravel project:</p>
                    <CodeBlock label="terminal">{`# From your Laravel project root
cp koskidex/examples/laravel/app/Services/KoskidexClient.php \\
   app/Services/KoskidexClient.php

cp koskidex/examples/laravel/app/Traits/Searchable.php \\
   app/Traits/Searchable.php

cp koskidex/examples/laravel/config/koskidex.php \\
   config/koskidex.php`}</CodeBlock>
                  </div>

                  <div>
                    <h3 className="text-white font-semibold text-sm mb-3">{t("docs.sections.php.step3_title")}</h3>
                    <CodeBlock label=".env">{`KOSKIDEX_HOST=http://koskidex:7700\nKOSKIDEX_API_KEY=your-secret-key`}</CodeBlock>
                    <p className="text-slate-500 text-xs mt-2">The <span className="text-slate-300 font-mono">KoskidexClient</span> reads these values and automatically sends the Bearer token on every request.</p>
                  </div>

                  <div>
                    <h3 className="text-white font-semibold text-sm mb-3">{t("docs.sections.php.step4_title")}</h3>
                    <CodeBlock label="config/koskidex.php">{`return [
    'host' => env('KOSKIDEX_HOST', 'http://localhost:7700'),
    'api_key' => env('KOSKIDEX_API_KEY', ''),

    'indices' => [
        App\\Models\\Movie::class => [
            'index_name' => 'movies',
            'searchable_fields' => ['title', 'director', 'genre'],
            'hit_threshold' => 70,
        ],
    ],
];`}</CodeBlock>
                  </div>

                  <div>
                    <h3 className="text-white font-semibold text-sm mb-3">{t("docs.sections.php.step5_title")}</h3>
                    <CodeBlock label="app/Models/Movie.php">{`<?php

namespace App\\Models;

use Illuminate\\Database\\Eloquent\\Model;
use App\\Traits\\Searchable;

class Movie extends Model
{
    use Searchable;

    protected $fillable = ['title', 'director', 'genre', 'year'];
}`}</CodeBlock>
                  </div>

                  <div>
                    <h3 className="text-white font-semibold text-sm mb-3">{t("docs.sections.php.step6_title")}</h3>
                    <CodeBlock label="php">{`// Auto-synced on create/update/delete
Movie::create([
    'title' => 'The Matrix',
    'director' => 'Wachowskis',
    'genre' => 'Sci-Fi',
    'year' => 1999,
]);

// Search — returns Eloquent Collection in Koskidex ranking order
$results = Movie::koskidexSearch('matrx');`}</CodeBlock>
                  </div>

                  <div className="p-4 bg-emerald-500/5 rounded-xl border border-emerald-500/10 text-sm text-slate-400">
                    <span className="text-emerald-400 font-bold mr-2">Pro Tip:</span>
                    {t("docs.sections.php.sync_note")}
                  </div>
                </div>
              </SectionCard>
            </section>

            {/* ──────────── JavaScript / Node.js ──────────── */}
            <section id="js" ref={setSectionRef("js")} className="scroll-mt-32">
              <div className="flex items-center gap-3 mb-6 md:mb-8">
                <Globe className="w-7 h-7 md:w-8 md:h-8 text-yellow-400" />
                <h2 className="text-xl md:text-3xl font-black text-white uppercase tracking-wider">
                  {t("docs.sections.js.title")}
                </h2>
              </div>
              <SectionCard>
                <p className="text-slate-300 mb-8 leading-relaxed">
                  {t("docs.sections.js.desc")}
                </p>

                <div className="grid gap-6">
                  <div>
                    <h3 className="text-white font-semibold text-sm mb-3">{t("docs.sections.js.step1_title")}</h3>
                    <CodeBlock label="terminal">{`# Install from the cloned Koskidex repo\nnpm install ./koskidex/examples/nodejs`}</CodeBlock>
                  </div>

                  <div>
                    <h3 className="text-white font-semibold text-sm mb-3">{t("docs.sections.js.step2_title")}</h3>
                    <CodeBlock label="js">{`const KoskidexClient = require('koskidex-node');

// Without authentication
const client = new KoskidexClient('http://localhost:7700');

// With API key
const client = new KoskidexClient('http://localhost:7700', 'your-secret-key');`}</CodeBlock>
                  </div>

                  <div>
                    <h3 className="text-white font-semibold text-sm mb-3">{t("docs.sections.js.step3_title")}</h3>
                    <CodeBlock label="js">{`await client.createIndex('movies');

await client.addDocuments('movies', [
  { id: '1', title: 'The Matrix', genre: 'Sci-Fi' },
  { id: '2', title: 'Inception', genre: 'Action' },
]);`}</CodeBlock>
                  </div>

                  <div>
                    <h3 className="text-white font-semibold text-sm mb-3">{t("docs.sections.js.step4_title")}</h3>
                    <CodeBlock label="js">{`const results = await client.search('movies', 'matrx');\nconsole.log(results.hits); // [{ id: '1', document: { title: 'The Matrix', ... } }]`}</CodeBlock>
                  </div>
                </div>
              </SectionCard>
            </section>

            {/* ──────────── Python ──────────── */}
            <section id="python" ref={setSectionRef("python")} className="scroll-mt-32">
              <div className="flex items-center gap-3 mb-6 md:mb-8">
                <Code className="w-7 h-7 md:w-8 md:h-8 text-green-400" />
                <h2 className="text-xl md:text-3xl font-black text-white uppercase tracking-wider">
                  {t("docs.sections.python.title")}
                </h2>
              </div>
              <SectionCard>
                <p className="text-slate-300 mb-8 leading-relaxed">
                  {t("docs.sections.python.desc")}
                </p>

                <div className="grid gap-6">
                  <div>
                    <h3 className="text-white font-semibold text-sm mb-3">{t("docs.sections.python.step1_title")}</h3>
                    <CodeBlock label="terminal">{`# Install from the cloned Koskidex repo\npip install ./koskidex/examples/python`}</CodeBlock>
                  </div>

                  <div>
                    <h3 className="text-white font-semibold text-sm mb-3">{t("docs.sections.python.step2_title")}</h3>
                    <CodeBlock label="python">{`from koskidex_client import KoskidexClient

# Without authentication
client = KoskidexClient('http://localhost:7700')

# With API key
client = KoskidexClient('http://localhost:7700', api_key='your-secret-key')

# Create index
client.create_index('movies')

# Add documents
client.add_documents('movies', [
    {'id': '1', 'title': 'The Matrix', 'genre': 'Sci-Fi'},
    {'id': '2', 'title': 'Inception', 'genre': 'Action'},
])

# Search (typo-tolerant)
results = client.search('movies', 'matrx')
print(results['hits'])`}</CodeBlock>
                  </div>
                </div>
              </SectionCard>
            </section>

            {/* ──────────── API Reference ──────────── */}
            <section id="api" ref={setSectionRef("api")} className="scroll-mt-32">
              <div className="flex items-center gap-3 mb-6 md:mb-8">
                <Box className="w-7 h-7 md:w-8 md:h-8 text-purple-400" />
                <h2 className="text-xl md:text-3xl font-black text-white uppercase tracking-wider">
                  {t("docs.sections.api.title")}
                </h2>
              </div>
              <SectionCard>
                <p className="text-slate-300 mb-8 leading-relaxed">
                  {t("docs.sections.api.desc")}
                </p>

                <div className="grid gap-8">
                  <div>
                    <h3 className="text-white font-semibold text-sm mb-4 flex items-center gap-2">
                      <Terminal className="w-4 h-4 text-slate-500" />
                      {t("docs.sections.api.endpoints_title")}
                    </h3>
                    <div className="overflow-x-auto">
                      <table className="w-full text-sm text-left">
                        <thead>
                          <tr className="border-b border-white/5 text-slate-500 text-xs uppercase tracking-wider">
                            <th className="py-3 pr-4 font-semibold">Method</th>
                            <th className="py-3 pr-4 font-semibold">Endpoint</th>
                            <th className="py-3 font-semibold">Description</th>
                          </tr>
                        </thead>
                        <tbody className="text-slate-300 font-mono text-xs">
                          {[
                            ["POST", "/indexes", "Create an index"],
                            ["GET", "/indexes", "List all indexes"],
                            ["GET", "/indexes/{name}", "Get index info"],
                            ["DELETE", "/indexes/{name}", "Delete index"],
                            ["POST", "/indexes/{name}/documents", "Add documents (single, array, or file upload)"],
                            ["GET", "/indexes/{name}/documents/{id}", "Get document by ID"],
                            ["DELETE", "/indexes/{name}/documents/{id}", "Delete document"],
                            ["GET", "/indexes/{name}/search?q=", "Full-text search (GET)"],
                            ["POST", "/indexes/{name}/search", "Full-text search (POST body)"],
                            ["GET", "/indexes/{name}/settings", "Get index settings"],
                            ["PUT", "/indexes/{name}/settings", "Update settings"],
                            ["GET", "/health", "Health check (no auth required)"],
                          ].map(([method, endpoint, desc]) => (
                            <tr key={`${method}-${endpoint}`} className="border-b border-white/5 hover:bg-white/[0.02]">
                              <td className="py-2.5 pr-4">
                                <span className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                                  method === "GET" ? "bg-emerald-500/10 text-emerald-400" :
                                  method === "POST" ? "bg-blue-500/10 text-blue-400" :
                                  method === "PUT" ? "bg-yellow-500/10 text-yellow-400" :
                                  "bg-red-500/10 text-red-400"
                                }`}>{method}</span>
                              </td>
                              <td className="py-2.5 pr-4 text-slate-200">{endpoint}</td>
                              <td className="py-2.5 text-slate-500 font-sans">{desc}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </div>

                  <div>
                    <h3 className="text-white font-semibold text-sm mb-3 flex items-center gap-2">
                      <Key className="w-4 h-4 text-slate-500" />
                      {t("docs.sections.api.auth_title")}
                    </h3>
                    <p className="text-slate-400 text-sm mb-3">{t("docs.sections.api.auth_desc")}</p>
                    <CodeBlock>{`# Start Koskidex with an API key (any string you choose):\n./koskidex --api-key your-secret-key\n\n# Or via Docker Compose (reads from .env):\n# command: ["/app/koskidex", "--api-key", "\${KOSKIDEX_API_KEY}"]\n\n# Then pass it on every request:\ncurl -H "Authorization: Bearer your-secret-key" \\\n  http://localhost:7700/indexes\n\n# /health is always public — no token needed:\ncurl http://localhost:7700/health`}</CodeBlock>
                  </div>

                  <div>
                    <h3 className="text-white font-semibold text-sm mb-3">{t("docs.sections.api.search_params")}</h3>
                    <CodeBlock>{`# Pagination (default: limit=20, max: 1000)
curl "http://localhost:7700/indexes/movies/search?q=matrix&limit=10&offset=0"

# Field filters (=, !=, >, <, >=, <=)
curl "http://localhost:7700/indexes/movies/search?q=matrix&filter=genre=Sci-Fi"
curl "http://localhost:7700/indexes/movies/search?q=movie&filter=year>2000,genre=Action"

# Dynamic fuzziness (0 = exact, 1 = one typo, 2 = two typos, AUTO = adaptive)
curl "http://localhost:7700/indexes/movies/search?q=matrx&fuzziness=AUTO"

# Explicit sorting (comma-separated, :asc or :desc)
curl "http://localhost:7700/indexes/movies/search?q=nolan&sort=year:desc"
curl "http://localhost:7700/indexes/products/search?q=apple&sort=price:asc,rating:desc"

# Faceted search (returns counts per distinct value)
curl "http://localhost:7700/indexes/movies/search?q=the&facets=genre,director"

# Geospatial filtering (Haversine distance in meters)
curl "http://localhost:7700/indexes/places/search?q=pizza&filter=distance(_geo,45.46,9.19)<5000"

# OR / NOT operators
curl "http://localhost:7700/indexes/movies/search?q=matrix OR inception"
curl "http://localhost:7700/indexes/movies/search?q=movie -horror"

# POST search (for complex queries or vector search)
curl -X POST http://localhost:7700/indexes/movies/search \\
  -H 'Content-Type: application/json' \\
  -d '{"q":"matrix","fuzziness":"AUTO","sort":"year:desc","facets":"genre"}'`}</CodeBlock>
                  </div>
                </div>
              </SectionCard>
            </section>

            {/* ──────────── Architecture ──────────── */}
            <section id="architecture" ref={setSectionRef("architecture")} className="scroll-mt-32">
              <div className="flex items-center gap-3 mb-6 md:mb-8">
                <Book className="w-7 h-7 md:w-8 md:h-8 text-blue-500" />
                <h2 className="text-xl md:text-3xl font-black text-white uppercase tracking-wider">
                  {t("docs.sections.architecture.title")}
                </h2>
              </div>
              <SectionCard>
                <p className="text-slate-300 mb-8 leading-relaxed">
                  {t("docs.sections.architecture.desc")}
                </p>

                <div className="grid md:grid-cols-3 gap-6">
                  <div className="p-5 bg-white/5 rounded-xl border border-white/5">
                    <h4 className="text-white font-bold mb-2 flex items-center gap-2 text-sm">
                      <Server className="w-4 h-4 text-blue-400" />
                      {t("docs.sections.architecture.concepts.indices")}
                    </h4>
                    <p className="text-slate-400 text-sm leading-relaxed">
                      {t("docs.sections.architecture.concepts.indices_desc")}
                    </p>
                  </div>
                  <div className="p-5 bg-white/5 rounded-xl border border-white/5">
                    <h4 className="text-white font-bold mb-2 flex items-center gap-2 text-sm">
                      <Code className="w-4 h-4 text-emerald-400" />
                      {t("docs.sections.architecture.concepts.documents")}
                    </h4>
                    <p className="text-slate-400 text-sm leading-relaxed">
                      {t("docs.sections.architecture.concepts.documents_desc")}
                    </p>
                  </div>
                  <div className="p-5 bg-white/5 rounded-xl border border-white/5">
                    <h4 className="text-white font-bold mb-2 flex items-center gap-2 text-sm">
                      <Globe className="w-4 h-4 text-purple-400" />
                      {t("docs.sections.architecture.concepts.search")}
                    </h4>
                    <p className="text-slate-400 text-sm leading-relaxed">
                      {t("docs.sections.architecture.concepts.search_desc")}
                    </p>
                  </div>
                  <div className="p-5 bg-white/5 rounded-xl border border-white/5">
                    <h4 className="text-white font-bold mb-2 flex items-center gap-2 text-sm">
                      <Box className="w-4 h-4 text-yellow-400" />
                      {t("docs.sections.architecture.concepts.storage")}
                    </h4>
                    <p className="text-slate-400 text-sm leading-relaxed">
                      {t("docs.sections.architecture.concepts.storage_desc")}
                    </p>
                  </div>
                </div>
              </SectionCard>
            </section>

          </div>
        </div>
      </div>
    </div>
  );
}
