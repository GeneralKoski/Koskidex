import { Code, Terminal, HardDrive, Globe, ArrowRight } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

export default function IntegrationSnippets() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState("curl");
  const [method, setMethod] = useState<"package" | "repo">("package");

  const tabs = [
    { id: "curl", label: "cURL" },
    { id: "php", label: "PHP (Laravel)" },
    { id: "node", label: "Node.js" },
    { id: "python", label: "Python" },
  ];

  type SnippetMap = Record<string, Record<"package" | "repo", string>>;

  const codeSnippets: SnippetMap = {
    curl: {
      package: `# 1. ${t("docs.sections.go.clone_build")}\ncurl -X POST http://localhost:7700/indexes -d '{"name": "movies"}'\n\n# 2. ${t("demo.setup.status.loading_plain")}\ncurl -X POST http://localhost:7700/indexes/movies/documents \\
  -H 'Content-Type: application/json' -d '[
  {"id": "1", "title": "${t("datasets.movies.titles.matrix")}", "genre": "${t("datasets.movies.genres.sci_fi")}"}
]'\n\n# 3. ${t("demo.search.search_plain")}\ncurl "http://localhost:7700/indexes/movies/search?q=matrx"`,
      repo: `# ${t("docs.sections.js.client_usage")}\n# ${t("docs.sections.php.sync_desc").split(".")[0]}\n\ncurl -X POST http://localhost:7700/indexes -d '{"name": "movies"}'\n\ncurl "http://localhost:7700/indexes/movies/search?q=matrx"`,
    },
    php: {
      package: `# 1. ${t("docs.sections.php.command")}\n# composer require GeneralKoski/Koskidex-laravel\n\nuse App\\Traits\\Searchable;\n\nclass Movie extends Model {\n    use Searchable;\n}\n\n// ${t("docs.sections.php.sync_desc")}\nMovie::create(['title' => '${t("datasets.movies.titles.matrix")}']);\n\n// ${t("demo.search.search_plain")}\n$results = Movie::koskidexSearch('matrx');`,
      repo: `# 1. ${t("docs.sections.php.laravel_trait")}\n# "repositories": [{ "type": "path", "url": "./path/to/koskidex/examples/laravel" }]\n\n# 2. ${t("docs.sections.php.command")}\n# composer require GeneralKoski/Koskidex-laravel\n\nuse App\\Traits\\Searchable;\n\nclass Movie extends Model {\n    use Searchable;\n}\n\n$results = Movie::koskidexSearch('matrx');`,
    },
    node: {
      package: `# 1. ${t("docs.sections.js.command")}\n# npm install koskidex-node\n\nconst KoskidexClient = require('koskidex-node');\nconst client = new KoskidexClient('http://localhost:7700');\n\n// ${t("demo.setup.status.loading_plain")}\nawait client.addDocuments('movies', moviesArray);\n\n// ${t("demo.search.search_plain")}\nconst results = await client.search('movies', 'matrihx');`,
      repo: `# 1. ${t("docs.sections.js.client_usage")}\n# or link it locally:\n# npm install ../path-to-koskidex/examples/nodejs\n\nconst KoskidexClient = require('koskidex-node');\nconst client = new KoskidexClient('http://localhost:7700');\n\nconst results = await client.search('movies', 'matrihx');`,
    },
    python: {
      package: `# 1. ${t("docs.sections.js.command")}\n# pip install koskidex\n\nfrom koskidex_client import KoskidexClient\nclient = KoskidexClient('http://localhost:7700')\n\n# ${t("demo.search.search_plain")}\nres = client.search('movies', 'mtrx')`,
      repo: `# 1. ${t("docs.sections.js.client_usage")}\n# to your project directory.\n\nfrom koskidex_client import KoskidexClient\nclient = KoskidexClient('http://localhost:7700')\n\nres = client.search('movies', 'mtrx')`,
    },
  };

  return (
    <section
      id="integration"
      className="py-24 border-t border-slate-800/50 relative overflow-hidden"
    >
      <div className="container mx-auto px-4 max-w-5xl">
        <div className="text-center mb-16">
          <div className="w-16 h-16 bg-blue-500/10 rounded-3xl flex items-center justify-center mx-auto mb-6">
            <Code className="w-8 h-8 text-blue-500" />
          </div>
          <h2 className="text-4xl font-black mb-4 tracking-tight">{t("integration.title")}</h2>
          <p className="text-slate-400 text-xl font-light">{t("integration.subtitle")}</p>
        </div>

        <div className="glass-effect rounded-[2.5rem] overflow-hidden shadow-2xl border-white/5 bg-slate-900/40">
          <div className="flex border-b border-white/5 bg-slate-950/40 overflow-x-auto hide-scrollbar">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                aria-label={tab.label}
                aria-selected={activeTab === tab.id}
                className={`px-8 py-5 text-sm font-black transition-all whitespace-nowrap border-b-2 flex items-center gap-2 ${
                  activeTab === tab.id
                    ? "border-blue-500 text-blue-400 bg-blue-500/10"
                    : "border-transparent text-slate-500 hover:text-slate-300 hover:bg-white/5"
                }`}
              >
                {tab.id === "curl" && (
                  <Terminal className="w-4 h-4 opacity-70" />
                )}
                {tab.label}
              </button>
            ))}
          </div>

          <div className="p-8 border-b border-white/5 bg-slate-950/20">
            <div className="flex flex-col md:flex-row items-center justify-between gap-6">
               <div className="flex items-center gap-3">
                  <div className="w-8 h-8 rounded-lg bg-white/5 flex items-center justify-center">
                    <HardDrive className="w-4 h-4 text-slate-400" />
                  </div>
                  <span className="text-sm font-bold text-slate-300">{t("integration.install_method")}</span>
               </div>
               
               <div className="flex bg-slate-900 p-1 rounded-xl border border-white/5 self-stretch md:self-auto">
                  <button
                    onClick={() => setMethod("package")}
                    className={`flex-1 md:flex-none px-6 py-2 rounded-lg text-xs font-black tracking-wider uppercase transition-all flex items-center justify-center gap-2 ${
                      method === "package" 
                        ? "bg-blue-600 text-white shadow-lg shadow-blue-500/20" 
                        : "text-slate-500 hover:text-slate-300"
                    }`}
                  >
                    <Globe className="w-3.5 h-3.5" />
                    {t("integration.method_package")}
                  </button>
                  <button
                    onClick={() => setMethod("repo")}
                    className={`flex-1 md:flex-none px-6 py-2 rounded-lg text-xs font-black tracking-wider uppercase transition-all flex items-center justify-center gap-2 ${
                      method === "repo" 
                        ? "bg-slate-700 text-white" 
                        : "text-slate-500 hover:text-slate-300"
                    }`}
                  >
                    <HardDrive className="w-3.5 h-3.5" />
                    {t("integration.method_repo")}
                  </button>
               </div>
            </div>
          </div>

          <div className="p-8 bg-[#0B1120]/80 relative group h-[400px] border-b border-white/5">
            <div className="absolute right-6 top-6 flex gap-3 opacity-0 group-hover:opacity-100 transition-opacity z-10">
              <button
                onClick={() => {
                  navigator.clipboard.writeText(codeSnippets[activeTab][method]);
                }}
                className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-white text-xs font-black rounded-xl border border-white/5 transition-all active:scale-95"
              >
                {t("common.copy")}
              </button>
              <div className="px-3 py-2 bg-slate-900/80 text-[10px] text-slate-500 font-mono rounded-xl border border-white/5 flex items-center uppercase tracking-widest leading-none">
                {activeTab}
              </div>
            </div>
            
            <pre className="font-mono text-sm text-slate-300 overflow-x-auto h-full leading-relaxed custom-scrollbar text-left">
              <code className="block py-4 whitespace-pre-wrap">{codeSnippets[activeTab][method]}</code>
            </pre>
          </div>

          <div className="p-6 bg-slate-950/40 text-center">
            <Link 
              to="/docs" 
              className="inline-flex items-center gap-2 text-blue-400 hover:text-blue-300 font-black text-sm uppercase tracking-widest transition-colors group"
            >
              {t("integration.view_full")}
              <ArrowRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
            </Link>
          </div>
        </div>
      </div>
      
      {/* Decorative blobs */}
      <div className="absolute -left-24 bottom-0 w-96 h-96 bg-blue-500/5 blur-[120px] rounded-full pointer-events-none"></div>
      <div className="absolute -right-24 top-0 w-96 h-96 bg-purple-500/5 blur-[120px] rounded-full pointer-events-none"></div>
    </section>
  );
}
