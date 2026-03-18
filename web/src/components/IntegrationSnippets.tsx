import { ArrowRight, Rocket } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

function CodeBlock({ children }: { children: string }) {
  return (
    <pre className="bg-[#0B1120] rounded-xl p-4 font-mono text-[12px] sm:text-[13px] text-slate-300 overflow-x-auto border border-white/5 leading-relaxed">
      <code className="whitespace-pre">{children}</code>
    </pre>
  );
}

function Step({ n, title, children }: { n: number; title: string; children: React.ReactNode }) {
  return (
    <div className="flex gap-4">
      <div className="flex flex-col items-center shrink-0">
        <div className="w-8 h-8 rounded-full bg-blue-500/15 border border-blue-500/30 text-blue-400 text-sm font-bold flex items-center justify-center">
          {n}
        </div>
        <div className="w-px flex-1 bg-gradient-to-b from-blue-500/30 to-transparent mt-2" />
      </div>
      <div className="pb-8 flex-1 min-w-0">
        <h3 className="text-white font-semibold text-sm mb-3">{title}</h3>
        {children}
      </div>
    </div>
  );
}

export default function IntegrationSnippets() {
  const { t } = useTranslation();

  return (
    <section
      id="integration"
      className="py-16 md:py-24 border-t border-slate-800/50 relative overflow-hidden"
    >
      <div className="container mx-auto px-4 max-w-3xl">
        <div className="text-center mb-12 md:mb-16">
          <div className="w-14 h-14 bg-blue-500/10 rounded-2xl flex items-center justify-center mx-auto mb-5">
            <Rocket className="w-7 h-7 text-blue-500" />
          </div>
          <h2 className="text-2xl md:text-4xl font-black mb-3 tracking-tight">{t("integration.title")}</h2>
          <p className="text-slate-400 text-sm md:text-lg font-light">{t("integration.subtitle")}</p>
        </div>

        <div className="glass-effect rounded-2xl p-5 md:p-8 shadow-xl border-white/5 bg-slate-900/40">
          <Step n={1} title={t("integration.step1_title")}>
            <CodeBlock>{`git clone https://github.com/GeneralKoski/Koskidex.git\ncd Koskidex`}</CodeBlock>
          </Step>

          <Step n={2} title={t("integration.step2_title")}>
            <CodeBlock>{`docker compose up -d`}</CodeBlock>
            <p className="text-slate-500 text-xs mt-2">API → <span className="text-slate-300">localhost:7700</span> &nbsp;·&nbsp; Frontend → <span className="text-slate-300">localhost:8080</span></p>
          </Step>

          <Step n={3} title={t("integration.step3_title")}>
            <CodeBlock>{`# Create an index\ncurl -X POST http://localhost:7700/indexes \\\n  -d '{"name": "movies"}'\n\n# Add documents\ncurl -X POST http://localhost:7700/indexes/movies/documents \\\n  -H 'Content-Type: application/json' \\\n  -d '[{"id": "1", "title": "The Matrix", "genre": "Sci-Fi"}]'\n\n# Search (typo-tolerant!)\ncurl "http://localhost:7700/indexes/movies/search?q=matrx"`}</CodeBlock>
          </Step>

          <div className="pt-2 text-center">
            <Link
              to="/docs"
              className="inline-flex items-center gap-2 text-blue-400 hover:text-blue-300 font-bold text-sm transition-colors group"
            >
              {t("integration.view_full")}
              <ArrowRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
            </Link>
          </div>
        </div>
      </div>

      <div className="absolute -left-24 bottom-0 w-96 h-96 bg-blue-500/5 blur-[120px] rounded-full pointer-events-none"></div>
      <div className="absolute -right-24 top-0 w-96 h-96 bg-purple-500/5 blur-[120px] rounded-full pointer-events-none"></div>
    </section>
  );
}
