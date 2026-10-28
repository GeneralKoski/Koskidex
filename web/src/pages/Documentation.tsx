import { ArrowLeft, TriangleAlert } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import CodeBlock from "../components/CodeBlock";
import { useDocumentMeta } from "../hooks/useDocumentMeta";

const NAV_ITEMS = [
  { id: "quickstart", key: "quickstart" },
  { id: "docker-compose", key: "docker_compose" },
  { id: "php", key: "php" },
  { id: "js", key: "js" },
  { id: "python", key: "python" },
  { id: "api", key: "api" },
  { id: "settings", key: "settings" },
  { id: "config", key: "config" },
  { id: "architecture", key: "architecture" },
] as const;

const ENDPOINTS: [string, string, string][] = [
  ["POST", "/indexes", "create_index"],
  ["GET", "/indexes", "list_indexes"],
  ["GET", "/indexes/{name}", "get_index"],
  ["DELETE", "/indexes/{name}", "delete_index"],
  ["POST", "/indexes/{name}/documents", "add_documents"],
  ["GET", "/indexes/{name}/documents", "list_documents"],
  ["GET", "/indexes/{name}/documents/{id}", "get_document"],
  ["DELETE", "/indexes/{name}/documents/{id}", "delete_document"],
  ["POST", "/indexes/{name}/documents/delete-batch", "delete_batch"],
  ["GET", "/indexes/{name}/search?q=", "search_get"],
  ["POST", "/indexes/{name}/search", "search_post"],
  ["GET", "/indexes/{name}/settings", "get_settings"],
  ["PUT", "/indexes/{name}/settings", "put_settings"],
  ["GET", "/indexes/{name}/sitemap.xml", "sitemap"],
  ["GET", "/robots.txt", "robots"],
  ["GET", "/health", "health"],
];

const FLAGS: [string, string, string, string][] = [
  ["--port", "KOSKIDEX_PORT", "7700", "port"],
  ["--data-dir", "KOSKIDEX_DATA_DIR", "./data", "data_dir"],
  ["--api-key", "KOSKIDEX_API_KEY", "", "api_key"],
  ["--log-level", "KOSKIDEX_LOG_LEVEL", "info", "log_level"],
  ["--rate-limit", "KOSKIDEX_RATE_LIMIT", "0", "rate_limit"],
  ["--cors-origin", "KOSKIDEX_CORS_ORIGIN", "*", "cors_origin"],
  ["--tls-cert, --tls-key", "KOSKIDEX_TLS_CERT, KOSKIDEX_TLS_KEY", "", "tls"],
  ["--protected-indexes", "KOSKIDEX_PROTECTED_INDEXES", "", "protected"],
  ["--version", "", "", "version"],
];

const SETTINGS_FIELDS = [
  ["searchable_fields", "searchable_fields"],
  ["displayed_fields", "displayed_fields"],
  ["field_weights", "field_weights"],
  ["synonym_groups", "synonym_groups"],
  ["synonyms", "synonyms"],
  ["synonym_fields", "synonym_fields"],
  ["stop_words", "stop_words"],
  ["typo_tolerance", "typo_tolerance"],
  ["scoring_mode", "scoring_mode"],
  ["retrieval_mode", "retrieval_mode"],
  ["stemmer", "stemmer"],
  ["tokenizer", "tokenizer"],
  ["normalize_dates, normalize_amounts", "normalize"],
  ["embedder", "embedder"],
  ["hybrid_mode, fusion_mode", "hybrid"],
  ["sitemap", "sitemap"],
] as const;

const METHOD_STYLE: Record<string, string> = {
  GET: "text-ok",
  POST: "text-link",
  PUT: "text-ink",
  DELETE: "text-bad",
};

function Section({
  id,
  title,
  desc,
  setRef,
  children,
}: {
  id: string;
  title: string;
  desc: string;
  setRef: (el: HTMLElement | null) => void;
  children: React.ReactNode;
}) {
  return (
    <section id={id} ref={setRef} className="scroll-mt-28 border-t border-ink/80 pt-6">
      <h2 className="font-wide font-black text-2xl md:text-3xl tracking-[-0.02em]">{title}</h2>
      <p className="mt-3 text-muted leading-relaxed max-w-[62ch]">{desc}</p>
      <div className="mt-8 grid gap-8">{children}</div>
    </section>
  );
}

function Block({ title, children }: { title?: string; children: React.ReactNode }) {
  return (
    <div className="min-w-0">
      {title && <h3 className="font-semibold mb-3">{title}</h3>}
      {children}
    </div>
  );
}

function Note({ children }: { children: React.ReactNode }) {
  return <p className="mt-3 text-sm text-muted leading-relaxed max-w-[70ch]">{children}</p>;
}

export default function Documentation() {
  const { t } = useTranslation();
  const [activeSection, setActiveSection] = useState("quickstart");
  const sectionRefs = useRef<Record<string, HTMLElement | null>>({});
  const mobileNavRef = useRef<HTMLDivElement>(null);

  useDocumentMeta({
    title: t("meta.docs_title"),
    description: t("meta.docs_desc"),
    path: "/docs",
  });

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

  const s = (key: string) => t(`docs.sections.${key}`);

  return (
    <div className="max-w-6xl mx-auto px-4 md:px-6 pt-24 md:pt-32 pb-20 md:pb-28">
      <Link
        to="/"
        className="inline-flex items-center gap-2 text-sm text-muted hover:text-ink transition-colors"
      >
        <ArrowLeft className="w-4 h-4" />
        {t("docs.back_home")}
      </Link>

      <header className="mt-6 mb-10 md:mb-16">
        <h1 className="font-wide font-black text-4xl md:text-6xl tracking-[-0.03em]">{t("docs.title")}</h1>
        <p className="mt-4 text-lg md:text-xl text-muted leading-relaxed max-w-[44rem]">{t("docs.subtitle")}</p>
      </header>

      <div
        ref={mobileNavRef}
        className="lg:hidden flex gap-1.5 overflow-x-auto scrollbar-hide -mx-4 px-4 py-2 mb-8 sticky top-16 z-30 bg-paper border-b border-rule"
      >
        {NAV_ITEMS.map((item) => {
          const isActive = activeSection === item.id;
          return (
            <button
              key={item.id}
              type="button"
              data-section={item.id}
              onClick={() => scrollToSection(item.id)}
              aria-current={isActive ? "true" : undefined}
              className={`px-3 py-1.5 rounded-full text-sm whitespace-nowrap shrink-0 transition-colors ${
                isActive ? "bg-ink text-paper" : "text-muted hover:text-ink"
              }`}
            >
              {s(`${item.key}.title`)}
            </button>
          );
        })}
      </div>

      <div className="flex gap-12">
        <aside className="hidden lg:block w-56 shrink-0">
          <nav className="sticky top-28" aria-label={t("docs.on_this_page")}>
            <p className="text-sm font-semibold mb-3">{t("docs.on_this_page")}</p>
            <ul className="flex flex-col border-l border-rule">
              {NAV_ITEMS.map((item) => {
                const isActive = activeSection === item.id;
                return (
                  <li key={item.id}>
                    <button
                      type="button"
                      onClick={() => scrollToSection(item.id)}
                      aria-current={isActive ? "true" : undefined}
                      className={`-ml-px w-full text-left pl-4 py-1.5 text-sm border-l-2 transition-colors ${
                        isActive
                          ? "border-ink text-ink font-semibold"
                          : "border-transparent text-muted hover:text-ink"
                      }`}
                    >
                      {s(`${item.key}.title`)}
                    </button>
                  </li>
                );
              })}
            </ul>
          </nav>
        </aside>

        <div className="flex-1 min-w-0 grid gap-16 md:gap-20">
          <Section id="quickstart" title={s("quickstart.title")} desc={s("quickstart.desc")} setRef={setSectionRef("quickstart")}>
            <Block title={`1. ${s("quickstart.step1")}`}>
              <CodeBlock>{`git clone https://github.com/GeneralKoski/Koskidex.git\ncd Koskidex\ndocker compose up -d`}</CodeBlock>
              <Note>{s("quickstart.running_at")}</Note>
            </Block>
            <Block title={`2. ${s("quickstart.step2")}`}>
              <CodeBlock>{`curl http://localhost:7700/health\n# {"documents":100000,"indexes":1,"status":"ok","uptime":"2m5s","version":"dev"}`}</CodeBlock>
            </Block>
            <Block title={`3. ${s("quickstart.step3")}`}>
              <CodeBlock>{`curl -X POST http://localhost:7700/indexes \\\n  -H 'Content-Type: application/json' \\\n  -d '{"name": "movies"}'\n\ncurl -X POST http://localhost:7700/indexes/movies/documents \\\n  -H 'Content-Type: application/json' \\\n  -d '[
  {"id": "1", "title": "The Matrix", "genre": "Sci-Fi", "year": 1999},
  {"id": "2", "title": "Inception", "genre": "Action", "year": 2010},
  {"id": "3", "title": "Interstellar", "genre": "Sci-Fi", "year": 2014}
]'`}</CodeBlock>
            </Block>
            <Block title={`4. ${s("quickstart.step4")}`}>
              <CodeBlock>{`curl "http://localhost:7700/indexes/movies/search?q=matrx"\ncurl "http://localhost:7700/indexes/movies/search?q=interstllar"\ncurl "http://localhost:7700/indexes/movies/search?q=Sci-Fi&filter=year>2000"`}</CodeBlock>
            </Block>
          </Section>

          <Section id="docker-compose" title={s("docker_compose.title")} desc={s("docker_compose.desc")} setRef={setSectionRef("docker-compose")}>
            <Block title={s("docker_compose.add_service")}>
              <p className="text-sm text-muted mb-3">{s("docker_compose.clone_hint")}</p>
              <div className="mb-3">
                <CodeBlock>{`git clone https://github.com/GeneralKoski/Koskidex.git koskidex`}</CodeBlock>
              </div>
              <CodeBlock label="docker-compose.yml">{`services:
  # ... your existing services (app, db, redis, etc.)

  koskidex:
    build: ./koskidex  # path to the cloned Koskidex repo
    ports:
      - "\${KOSKIDEX_PORT:-7700}:7700"
    volumes:
      - koskidex_data:/data
    environment:
      KOSKIDEX_API_KEY: "\${KOSKIDEX_API_KEY}"
    restart: unless-stopped

volumes:
  koskidex_data:`}</CodeBlock>
            </Block>
            <Block title={s("docker_compose.env_vars")}>
              <CodeBlock label=".env">{`# Koskidex connection
KOSKIDEX_HOST=http://koskidex:7700
KOSKIDEX_API_KEY=your-secret-key

# Port mapping (optional, default 7700)
KOSKIDEX_PORT=7700`}</CodeBlock>
              <Note>{s("docker_compose.env_note")}</Note>
            </Block>
            <Block title={s("docker_compose.sail_title")}>
              <p className="text-sm text-muted mb-3">{s("docker_compose.sail_desc")}</p>
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
    environment:
      KOSKIDEX_API_KEY: '\${KOSKIDEX_API_KEY}'
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
            </Block>
          </Section>

          <Section id="php" title={s("php.title")} desc={s("php.desc")} setRef={setSectionRef("php")}>
            <Block title={s("php.step1_title")}>
              <p className="text-sm text-muted mb-3">{s("php.step1_desc")}</p>
              <CodeBlock>{`# From your Laravel project root
cp koskidex/examples/laravel/app/Services/KoskidexClient.php \\
   app/Services/KoskidexClient.php

cp koskidex/examples/laravel/app/Traits/Searchable.php \\
   app/Traits/Searchable.php

cp koskidex/examples/laravel/config/koskidex.php \\
   config/koskidex.php`}</CodeBlock>
            </Block>
            <Block title={s("php.step2_title")}>
              <CodeBlock label=".env">{`KOSKIDEX_HOST=http://koskidex:7700\nKOSKIDEX_API_KEY=your-secret-key`}</CodeBlock>
              <Note>{s("php.step2_desc")}</Note>
            </Block>
            <Block title={s("php.step3_title")}>
              <CodeBlock label="config/koskidex.php">{`return [
    'host' => env('KOSKIDEX_HOST', 'http://localhost:7700'),
    'api_key' => env('KOSKIDEX_API_KEY', ''),

    'indices' => [
        App\\Models\\Movie::class => [
            'index_name' => 'movies',
            'searchable_fields' => ['title', 'director', 'genre'],
        ],
    ],
];`}</CodeBlock>
            </Block>
            <Block title={s("php.step4_title")}>
              <p className="text-sm text-muted mb-3">{s("php.step4_desc")}</p>
              <CodeBlock label="php artisan tinker">{`app(\\App\\Services\\KoskidexClient::class)->createIndex('movies');`}</CodeBlock>
            </Block>
            <Block title={s("php.step5_title")}>
              <CodeBlock label="app/Models/Movie.php">{`<?php

namespace App\\Models;

use Illuminate\\Database\\Eloquent\\Model;
use App\\Traits\\Searchable;

class Movie extends Model
{
    use Searchable;

    protected $fillable = ['title', 'director', 'genre', 'year'];
}`}</CodeBlock>
            </Block>
            <Block title={s("php.step6_title")}>
              <CodeBlock label="php">{`// Sent to Koskidex on save, removed on delete
Movie::create([
    'title' => 'The Matrix',
    'director' => 'Wachowskis',
    'genre' => 'Sci-Fi',
    'year' => 1999,
]);

// Returns an Eloquent Collection in Koskidex ranking order
$results = Movie::koskidexSearch('matrx');`}</CodeBlock>
            </Block>
            <p className="text-sm leading-relaxed border-l-2 border-mark pl-4">
              <strong className="font-semibold">{s("php.tip")}:</strong> {s("php.sync_note")}
            </p>
          </Section>

          <Section id="js" title={s("js.title")} desc={s("js.desc")} setRef={setSectionRef("js")}>
            <Block title={s("js.step1_title")}>
              <CodeBlock>{`# From your project root, with Koskidex cloned in ./koskidex
cp koskidex/examples/nodejs/koskidex_client.js ./koskidex_client.js
npm install axios`}</CodeBlock>
            </Block>
            <Block title={s("js.step2_title")}>
              <CodeBlock label="js">{`const KoskidexClient = require('./koskidex_client');

// Without authentication
const client = new KoskidexClient('http://localhost:7700');

// With an API key:
// const client = new KoskidexClient('http://localhost:7700', 'your-secret-key');`}</CodeBlock>
            </Block>
            <Block title={s("js.step3_title")}>
              <CodeBlock label="js">{`await client.createIndex('movies');

await client.addDocuments('movies', [
  { id: '1', title: 'The Matrix', genre: 'Sci-Fi' },
  { id: '2', title: 'Inception', genre: 'Action' },
]);`}</CodeBlock>
            </Block>
            <Block title={s("js.step4_title")}>
              <CodeBlock label="js">{`const results = await client.search('movies', 'matrx');
console.log(results.hits);
// [{ id: '1', document: { title: 'The Matrix', ... },
//    highlights: { title: 'The <em>Matrix</em>' }, score: ... }]`}</CodeBlock>
            </Block>
          </Section>

          <Section id="python" title={s("python.title")} desc={s("python.desc")} setRef={setSectionRef("python")}>
            <Block title={s("python.step1_title")}>
              <CodeBlock>{`# From your project root, with Koskidex cloned in ./koskidex
cp koskidex/examples/python/koskidex_client.py .
pip install requests`}</CodeBlock>
            </Block>
            <Block title={s("python.step2_title")}>
              <CodeBlock label="python">{`from koskidex_client import KoskidexClient

# Without authentication
client = KoskidexClient('http://localhost:7700')

# With an API key:
# client = KoskidexClient('http://localhost:7700', api_key='your-secret-key')

client.create_index('movies')

client.add_documents('movies', [
    {'id': '1', 'title': 'The Matrix', 'genre': 'Sci-Fi'},
    {'id': '2', 'title': 'Inception', 'genre': 'Action'},
])

# Typo-tolerant search
results = client.search('movies', 'matrx')
print(results['hits'])`}</CodeBlock>
            </Block>
          </Section>

          <Section id="api" title={s("api.title")} desc={s("api.desc")} setRef={setSectionRef("api")}>
            <Block title={s("api.endpoints_title")}>
              <div className="overflow-x-auto">
                <table className="w-full text-sm text-left">
                  <thead>
                    <tr className="border-b border-ink/80">
                      <th scope="col" className="py-2.5 pr-4 font-semibold">{s("api.method")}</th>
                      <th scope="col" className="py-2.5 pr-4 font-semibold">{s("api.endpoint")}</th>
                      <th scope="col" className="py-2.5 font-semibold">{s("api.description")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {ENDPOINTS.map(([method, endpoint, key]) => (
                      <tr key={`${method}-${endpoint}`} className="border-b border-rule align-top">
                        <td className={`py-2.5 pr-4 font-mono text-xs font-semibold ${METHOD_STYLE[method]}`}>{method}</td>
                        <td className="py-2.5 pr-4 font-mono text-xs whitespace-nowrap">{endpoint}</td>
                        <td className="py-2.5 text-muted min-w-[14rem]">{t(`docs.endpoints.${key}`)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </Block>

            <Block title={s("api.auth_title")}>
              <p className="text-sm text-muted mb-3">{s("api.auth_desc")}</p>
              <CodeBlock>{`# Start Koskidex with an API key (any string you choose)
./koskidex --api-key your-secret-key
# or: KOSKIDEX_API_KEY=your-secret-key ./koskidex

# Then pass it on every request
curl -H "Authorization: Bearer your-secret-key" \\
  http://localhost:7700/indexes

# /health is always public
curl http://localhost:7700/health`}</CodeBlock>
            </Block>

            <Block title={s("api.search_params")}>
              <p className="text-sm text-muted mb-3">{s("api.search_params_desc")}</p>
              <CodeBlock>{`# Pagination
curl "http://localhost:7700/indexes/movies/search?q=matrix&limit=10&offset=0"

# Only ids and scores, no documents or highlights
curl "http://localhost:7700/indexes/movies/search?q=matrix&limit=10000&ids_only=true"

# Field filters (=, :, !=, >, <, >=, <=), comma means AND
curl "http://localhost:7700/indexes/movies/search?q=matrix&filter=genre=Sci-Fi"
curl "http://localhost:7700/indexes/movies/search?q=movie&filter=year>2000,genre:Action"

# Typos allowed per word: 0, 1, 2 or AUTO
curl "http://localhost:7700/indexes/movies/search?q=matrx&fuzziness=AUTO"

# Sorting (comma-separated, :asc or :desc)
curl "http://localhost:7700/indexes/movies/search?q=nolan&sort=year:desc"
curl "http://localhost:7700/indexes/products/search?q=apple&sort=price:asc,rating:desc"

# Facets (counts per distinct value)
curl "http://localhost:7700/indexes/movies/search?q=the&facets=genre,director"

# Geo filter (Haversine distance in meters)
curl "http://localhost:7700/indexes/places/search?q=pizza&filter=distance(_geo,45.46,9.19)<5000"

# OR and NOT operators (spaces URL-encoded)
curl "http://localhost:7700/indexes/movies/search?q=matrix%20OR%20inception"
curl "http://localhost:7700/indexes/movies/search?q=movie%20-horror"

# POST body: same parameters, plus a vector
curl -X POST http://localhost:7700/indexes/movies/search \\
  -H 'Content-Type: application/json' \\
  -d '{"q":"matrix","fuzziness":"AUTO","sort":"year:desc","facets":"genre","vector":[0.12,-0.03,0.88]}'

# Hybrid search: the index embedder computes the query vector
curl -X POST http://localhost:7700/indexes/movies/search \\
  -H 'Content-Type: application/json' \\
  -d '{"q":"space travel","hybrid":true}'`}</CodeBlock>
            </Block>

            <Block title={s("api.response_title")}>
              <p className="text-sm text-muted mb-3">{s("api.response_desc")}</p>
              <CodeBlock label="json">{`{
  "query": "matrx",
  "hits": [
    {
      "id": "1",
      "document": { "id": "1", "title": "The Matrix", "genre": "Sci-Fi", "year": 1999 },
      "highlights": { "title": "The <em>Matrix</em>" },
      "score": <float>
    }
  ],
  "total_hits": 1,
  "facets": {},
  "limit": 20,
  "offset": 0,
  "processing_time_ms": <int>
}`}</CodeBlock>
            </Block>
          </Section>

          <Section id="settings" title={s("settings.title")} desc={s("settings.desc")} setRef={setSectionRef("settings")}>
            <p className="flex gap-3 text-sm leading-relaxed border-l-2 border-bad pl-4">
              <TriangleAlert className="w-4 h-4 mt-0.5 shrink-0 text-bad" aria-hidden="true" />
              <span>{s("settings.warning")}</span>
            </p>
            <Block title={s("settings.example")}>
              <CodeBlock>{`curl -X PUT http://localhost:7700/indexes/movies/settings \\
  -H 'Content-Type: application/json' \\
  -d '{
  "searchable_fields": ["title", "director", "genre"],
  "field_weights": {"title": 3, "director": 1.5},
  "synonym_groups": [["film", "movie"]],
  "stop_words": {"the": true, "a": true},
  "typo_tolerance": {
    "enabled": true,
    "min_word_length_one_typo": 4,
    "min_word_length_two_typos": 8,
    "disable_on_numbers": true
  },
  "scoring_mode": "bm25",
  "bm25_k1": 1.2,
  "bm25_b": 0.75
}'`}</CodeBlock>
            </Block>
            <Block title={s("settings.fields_title")}>
              <dl className="border-t border-rule">
                {SETTINGS_FIELDS.map(([name, key]) => (
                  <div key={key} className="grid sm:grid-cols-[14rem_1fr] gap-x-6 gap-y-1 py-3 border-b border-rule">
                    <dt className="font-mono text-xs font-semibold pt-0.5 break-words">{name}</dt>
                    <dd className="text-sm text-muted leading-relaxed">{s(`settings.fields.${key}`)}</dd>
                  </div>
                ))}
              </dl>
            </Block>
          </Section>

          <Section id="config" title={s("config.title")} desc={s("config.desc")} setRef={setSectionRef("config")}>
            <div className="overflow-x-auto">
              <table className="w-full text-sm text-left">
                <thead>
                  <tr className="border-b border-ink/80">
                    <th scope="col" className="py-2.5 pr-4 font-semibold">{s("config.flag")}</th>
                    <th scope="col" className="py-2.5 pr-4 font-semibold">{s("config.env")}</th>
                    <th scope="col" className="py-2.5 pr-4 font-semibold">{s("config.default")}</th>
                    <th scope="col" className="py-2.5 font-semibold">{s("api.description")}</th>
                  </tr>
                </thead>
                <tbody>
                  {FLAGS.map(([flag, env, def, key]) => (
                    <tr key={flag} className="border-b border-rule align-top">
                      <td className="py-2.5 pr-4 font-mono text-xs whitespace-nowrap">{flag}</td>
                      <td className="py-2.5 pr-4 font-mono text-xs text-muted">{env}</td>
                      <td className="py-2.5 pr-4 font-mono text-xs">{def}</td>
                      <td className="py-2.5 text-muted min-w-[14rem]">{s(`config.${key}`)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Section>

          <Section id="architecture" title={s("architecture.title")} desc={s("architecture.desc")} setRef={setSectionRef("architecture")}>
            <dl className="grid md:grid-cols-2 gap-x-10">
              {(["indices", "documents", "search", "storage"] as const).map((key) => (
                <div key={key} className="border-t border-rule pt-4 pb-6">
                  <dt className="font-wide font-bold text-lg">{s(`architecture.concepts.${key}`)}</dt>
                  <dd className="mt-2 text-sm text-muted leading-relaxed">{s(`architecture.concepts.${key}_desc`)}</dd>
                </div>
              ))}
            </dl>
          </Section>
        </div>
      </div>
    </div>
  );
}
