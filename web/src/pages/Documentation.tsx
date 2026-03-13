import { ArrowLeft, Book, Code, Cpu, Globe, Server, Terminal } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { useEffect } from "react";

export default function Documentation() {
  const { t } = useTranslation();

  useEffect(() => {
    window.scrollTo(0, 0);
  }, []);

  return (
    <div className="pt-32 pb-24 min-h-screen relative overflow-hidden">
      {/* Background Decor */}
      <div className="absolute top-0 right-0 w-1/2 h-1/2 bg-blue-500/5 blur-[120px] rounded-full pointer-events-none"></div>
      <div className="absolute bottom-0 left-0 w-1/2 h-1/2 bg-purple-500/5 blur-[120px] rounded-full pointer-events-none"></div>

      <div className="container mx-auto px-4 max-w-5xl relative z-10">
        <Link 
          to="/" 
          className="inline-flex items-center gap-2 text-slate-400 hover:text-white mb-12 transition-colors group"
        >
          <ArrowLeft className="w-4 h-4 group-hover:-translate-x-1 transition-transform" />
          {t("docs.back_home")}
        </Link>

        <div className="mb-20">
          <div className="flex items-center gap-4 mb-6">
            <div className="w-12 h-12 bg-blue-500/10 rounded-2xl flex items-center justify-center">
              <Book className="w-6 h-6 text-blue-500" />
            </div>
            <h1 className="text-5xl font-black tracking-tight text-white italic">
              {t("docs.title")}
            </h1>
          </div>
          <p className="text-xl text-slate-400 font-light max-w-3xl leading-relaxed">
            {t("docs.subtitle")}
          </p>
        </div>

        <div className="grid gap-20">
          {/* Architecture Section */}
          <section id="architecture" className="scroll-mt-32">
            <div className="flex items-center gap-3 mb-8">
              <Book className="w-8 h-8 text-blue-500" />
              <h2 className="text-3xl font-black text-white uppercase tracking-wider">{t("docs.sections.architecture.title")}</h2>
            </div>
            <div className="glass-effect p-8 rounded-[2rem] border-white/5 bg-slate-900/40">
              <p className="text-slate-300 mb-10 leading-relaxed text-lg">
                {t("docs.sections.architecture.desc")}
              </p>
              
              <div className="grid md:grid-cols-3 gap-8 mb-12">
                <div className="p-6 bg-white/5 rounded-2xl border border-white/5">
                  <h4 className="text-white font-bold mb-3 flex items-center gap-2">
                    <Server className="w-4 h-4 text-blue-400" /> {t("docs.sections.architecture.concepts.indices")}
                  </h4>
                  <p className="text-slate-400 text-sm leading-relaxed">
                    {t("docs.sections.architecture.concepts.indices_desc")}
                  </p>
                </div>
                <div className="p-6 bg-white/5 rounded-2xl border border-white/5">
                  <h4 className="text-white font-bold mb-3 flex items-center gap-2">
                    <Code className="w-4 h-4 text-emerald-400" /> {t("docs.sections.architecture.concepts.documents")}
                  </h4>
                  <p className="text-slate-400 text-sm leading-relaxed">
                    {t("docs.sections.architecture.concepts.documents_desc")}
                  </p>
                </div>
                <div className="p-6 bg-white/5 rounded-2xl border border-white/5">
                  <h4 className="text-white font-bold mb-3 flex items-center gap-2">
                    <Globe className="w-4 h-4 text-purple-400" /> {t("docs.sections.architecture.concepts.search")}
                  </h4>
                  <p className="text-slate-400 text-sm leading-relaxed">
                    {t("docs.sections.architecture.concepts.search_desc")}
                  </p>
                </div>
              </div>

              {/* Customization Details */}
              <div className="border-t border-white/5 pt-10">
                <div className="flex items-center gap-3 mb-6">
                  <Globe className="w-6 h-6 text-yellow-500" />
                  <h3 className="text-xl font-bold text-white">{t("docs.sections.architecture.customization.title")}</h3>
                </div>
                <p className="text-slate-400 mb-8 leading-relaxed">
                  {t("docs.sections.architecture.customization.desc")}
                </p>
                <div className="bg-slate-950/40 rounded-2xl p-6 border border-white/5">
                  <h4 className="text-white font-bold mb-4 flex items-center gap-2">
                    <Terminal className="w-4 h-4 text-slate-500" /> {t("docs.sections.architecture.customization.classes_title")}
                  </h4>
                  <p className="text-slate-400 text-sm mb-6 italic">{t("docs.sections.architecture.customization.classes_desc")}</p>
                  <div className="grid md:grid-cols-2 gap-4">
                    <div className="p-4 bg-black/20 rounded-xl border border-white/5 font-mono text-[11px]">
                      <span className="text-purple-400">{`.koskidex-result-item`}</span> {`{`}
                      <div className="pl-4 text-slate-300">
                        {`border: 1px solid rgba(255,255,255,0.1);`}<br/>
                        {`border-radius: 1rem;`}<br/>
                        {`transition: all 0.2s ease;`}
                      </div>
                      {`}`}
                    </div>
                    <div className="p-4 bg-black/20 rounded-xl border border-white/5 font-mono text-[11px]">
                      <span className="text-purple-400">{`.koskidex-highlight`}</span> {`{`}
                      <div className="pl-4 text-slate-300">
                        {`background: #3b82f6;`}<br/>
                        {`color: white;`}<br/>
                        {`padding: 0 2px;`}
                      </div>
                      {`}`}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </section>

          {/* Go Section */}
          <section id="go" className="scroll-mt-32">
            <div className="flex items-center gap-3 mb-8">
              <Cpu className="w-8 h-8 text-emerald-500" />
              <h2 className="text-3xl font-black text-white uppercase tracking-wider">{t("docs.sections.go.title")}</h2>
            </div>
            <div className="glass-effect p-8 rounded-[2rem] border-white/5 bg-slate-900/40">
              <p className="text-slate-300 mb-6 leading-relaxed">
                {t("docs.sections.go.desc")}
              </p>
              <div className="bg-black/40 rounded-2xl p-6 font-mono text-sm border border-white/5 mb-6 overflow-x-auto">
                <div className="text-emerald-500 mb-2"># {t("docs.sections.go.clone_build")}</div>
                <div className="text-slate-200">git clone https://github.com/GeneralKoski/Koskidex.git</div>
                <div className="text-slate-200">cd Koskidex</div>
                <div className="text-slate-200">go build -o koskidex main.go</div>
                <div className="text-slate-400 py-2"># {t("docs.sections.go.run_engine")}</div>
                <div className="text-slate-200">./koskidex --port 7700 --data-dir ./data</div>
              </div>
              <p className="text-slate-400 text-sm italic">
                {t("docs.sections.go.port_desc", { port: "7700" })}
              </p>
            </div>
          </section>

          {/* PHP Section */}
          <section id="php" className="scroll-mt-32">
            <div className="flex items-center gap-3 mb-8">
              <Server className="w-8 h-8 text-blue-400" />
              <h2 className="text-3xl font-black text-white uppercase tracking-wider">{t("docs.sections.php.title")}</h2>
            </div>
            <div className="glass-effect p-8 rounded-[2rem] border-white/5 bg-slate-900/40">
              <p className="text-slate-300 mb-8 leading-relaxed text-lg">
                {t("docs.sections.php.desc")}
              </p>
              
              <div className="grid lg:grid-cols-2 gap-8 mb-10">
                <div className="flex flex-col">
                  <h3 className="text-white font-bold mb-4 flex items-center gap-2">
                    <Terminal className="w-4 h-4 text-slate-500" /> {t("docs.sections.php.command")}
                  </h3>
                  <div className="bg-black/40 rounded-2xl p-5 font-mono text-xs border border-white/5">
                    <span className="text-blue-400">composer</span> require <span className="text-emerald-400">generalkoski/koskidex-php</span>
                  </div>
                  
                  <h3 className="text-white font-bold mt-8 mb-4 flex items-center gap-2">
                     <Code className="w-4 h-4 text-slate-500" /> {t("docs.sections.php.laravel_trait")}
                  </h3>
                  <div className="p-5 bg-white/5 rounded-2xl border border-white/5">
                    <p className="text-slate-400 text-xs italic mb-4">{t("docs.sections.php.laravel_trait_desc")}</p>
                    <pre className="text-xs text-blue-300 leading-relaxed overflow-x-auto">
                      {`use Koski\\Integrations\\Searchable;\n\nclass Movie extends Model {\n  use Searchable;\n}`}
                    </pre>
                  </div>
                </div>

                <div className="flex flex-col">
                  <h3 className="text-white font-bold mb-4 flex items-center gap-2">
                    <Cpu className="w-4 h-4 text-slate-500" /> {t("docs.sections.php.config_title")}
                  </h3>
                  <div className="p-5 bg-white/5 rounded-2xl border border-white/5 h-full">
                    <p className="text-slate-400 text-xs mb-4 leading-relaxed">{t("docs.sections.php.config_desc")}</p>
                    <div className="text-xs font-bold text-slate-500 uppercase tracking-widest mb-2">{t("docs.sections.php.mapping_example")}</div>
                    <pre className="text-[10px] leading-relaxed text-emerald-400/80 overflow-x-auto bg-black/20 p-3 rounded-lg">
{`'indices' => [\n    App\\Models\\Movie::class => [\n        'index_name' => 'movies',\n        'searchable_fields' => ['title', 'director'],\n        'hit_threshold' => 70,\n    ],\n],`}
                    </pre>
                  </div>
                </div>
              </div>
              
              <div className="p-6 bg-blue-500/5 rounded-2xl border border-blue-500/10 text-slate-400 text-sm leading-relaxed italic">
                 <span className="text-blue-400 font-bold not-italic mr-2">Pro Tip:</span>
                 {t("docs.sections.php.sync_desc")}
              </div>
            </div>
          </section>

          {/* JS Section */}
          <section id="js" className="scroll-mt-32">
            <div className="flex items-center gap-3 mb-8">
              <Globe className="w-8 h-8 text-yellow-400" />
              <h2 className="text-3xl font-black text-white uppercase tracking-wider">{t("docs.sections.js.title")}</h2>
            </div>
            <div className="glass-effect p-8 rounded-[2rem] border-white/5 bg-slate-900/40">
              <p className="text-slate-300 mb-8 leading-relaxed text-lg">
                {t("docs.sections.js.desc")}
              </p>
              
              <div className="grid lg:grid-cols-2 gap-8">
                <div>
                  <h3 className="text-white font-bold mb-4">{t("docs.sections.js.command")}</h3>
                  <div className="bg-black/40 rounded-2xl p-6 font-mono text-sm border border-white/5 mb-8">
                    <span className="text-blue-400">npm</span> install <span className="text-emerald-400">koskidex-js</span>
                  </div>
                </div>

                <div className="p-6 bg-slate-950/40 rounded-2xl border border-white/5">
                  <div className="flex items-center justify-between mb-4">
                    <span className="text-xs font-bold text-slate-500 uppercase tracking-widest">{t("docs.sections.js.client_usage")}</span>
                    <Code className="w-4 h-4 text-blue-500" />
                  </div>
                  <pre className="text-xs leading-relaxed text-slate-300 overflow-x-auto">
{`import { Koskidex } from 'koskidex-js';\n\nconst koski = new Koskidex('http://localhost:7700');\n\n// ${t("docs.sections.js.search_desc")}\nconst results = await koski.search('index_name', 'query');`}
                  </pre>
                </div>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}
