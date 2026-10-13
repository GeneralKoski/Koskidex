import { ArrowRight } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import CodeBlock from "./CodeBlock";

function Step({ n, title, children }: { n: number; title: string; children: React.ReactNode }) {
  return (
    <li className="grid grid-cols-[2.25rem_1fr] gap-x-4">
      <span className="font-wide font-black text-2xl leading-none pt-0.5 tabular-nums" aria-hidden="true">
        {n}
      </span>
      <div className="min-w-0 pb-10">
        <h3 className="font-semibold text-lg mb-3">{title}</h3>
        {children}
      </div>
    </li>
  );
}

export default function IntegrationSnippets() {
  const { t } = useTranslation();

  return (
    <section id="integration" className="scroll-mt-16">
      <div className="max-w-6xl mx-auto px-4 md:px-6 py-16 md:py-24 grid lg:grid-cols-[18rem_1fr] gap-10 lg:gap-14">
        <div>
          <h2 className="font-wide font-black text-3xl md:text-4xl tracking-[-0.02em]">{t("integration.title")}</h2>
          <p className="mt-4 text-muted leading-relaxed">{t("integration.subtitle")}</p>
          <Link
            to="/docs"
            className="mt-6 inline-flex items-center gap-2 font-semibold text-link underline-offset-4 hover:underline"
          >
            {t("integration.view_full")}
            <ArrowRight className="w-4 h-4" />
          </Link>
        </div>

        <ol className="min-w-0">
          <Step n={1} title={t("integration.step1_title")}>
            <CodeBlock>{`git clone https://github.com/GeneralKoski/Koskidex.git\ncd Koskidex`}</CodeBlock>
          </Step>

          <Step n={2} title={t("integration.step2_title")}>
            <CodeBlock>{`docker compose up -d`}</CodeBlock>
            <p className="mt-3 text-sm text-muted leading-relaxed">{t("integration.step2_note")}</p>
          </Step>

          <Step n={3} title={t("integration.step3_title")}>
            <CodeBlock>{`# Create an index\ncurl -X POST http://localhost:7700/indexes \\\n  -H 'Content-Type: application/json' \\\n  -d '{"name": "movies"}'\n\n# Add documents\ncurl -X POST http://localhost:7700/indexes/movies/documents \\\n  -H 'Content-Type: application/json' \\\n  -d '[{"id": "1", "title": "The Matrix", "genre": "Sci-Fi"}]'\n\n# Search (typo-tolerant)\ncurl "http://localhost:7700/indexes/movies/search?q=matrx"`}</CodeBlock>
          </Step>
        </ol>
      </div>
    </section>
  );
}
