import { ArrowDown, BookOpen } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

function TraceRow({
  label,
  delay,
  children,
}: {
  label: string;
  delay: number;
  children: React.ReactNode;
}) {
  return (
    <div
      className="trace-line grid grid-cols-[6.5rem_1fr] sm:grid-cols-[8rem_1fr] gap-3 py-2.5 border-t border-white/10"
      style={{ animationDelay: `${delay}ms` }}
    >
      <span className="text-code-muted">{label}</span>
      <span className="min-w-0 break-words">{children}</span>
    </div>
  );
}

function QueryTrace() {
  const { t } = useTranslation();
  return (
    <figure className="bg-code text-code-ink rounded-xl font-mono text-[12.5px] sm:text-[13px] leading-relaxed shadow-[0_1px_0_rgba(0,0,0,0.04),0_24px_48px_-24px_rgba(21,23,28,0.45)] overflow-hidden">
      <div className="px-4 sm:px-5 py-3 text-code-muted border-b border-white/10 overflow-x-auto whitespace-nowrap">
        <span className="text-[#9fe0b8]">GET</span> /indexes/movies/search?q=matrx
      </div>
      <div className="px-4 sm:px-5 pb-2">
        <div className="grid grid-cols-[6.5rem_1fr] sm:grid-cols-[8rem_1fr] gap-3 py-2.5">
          <span className="text-code-muted">{t("hero.trace.query")}</span>
          <span>
            matrx<span className="caret ml-px inline-block w-[7px] h-[1.05em] align-[-2px] bg-code-ink" aria-hidden="true" />
          </span>
        </div>
        <TraceRow label={t("hero.trace.tokens")} delay={300}>
          ["matrx"]
        </TraceRow>
        <TraceRow label={t("hero.trace.match")} delay={650}>
          matrx <span className="text-code-muted">→</span> matrix{" "}
          <span className="ml-1 rounded border border-white/20 px-1.5 text-[11px] text-code-muted">{t("hero.trace.edit")}</span>
        </TraceRow>
        <TraceRow label={t("hero.trace.postings")} delay={1000}>
          matrix <span className="text-code-muted">→</span> [m1]
        </TraceRow>
        <TraceRow label={t("hero.trace.result")} delay={1350}>
          <span className="font-sans text-[15px] font-semibold">
            The{" "}
            <span className="marker rounded-[2px] px-0.5" style={{ animationDelay: "1600ms" }}>
              Matrix
            </span>
          </span>
          <span className="block text-code-muted">1999, Lana &amp; Lilly Wachowski</span>
        </TraceRow>
      </div>
      <figcaption className="px-4 sm:px-5 py-3 border-t border-white/10 font-sans text-xs text-code-muted">
        {t("hero.trace.caption")}
      </figcaption>
    </figure>
  );
}

const SPECS = [
  { key: "language" },
  { key: "runtime", note: true },
  { key: "api" },
  { key: "storage" },
  { key: "latency", note: true },
  { key: "ranking" },
] as const;

export default function Hero() {
  const { t } = useTranslation();

  const scrollToDemo = (e: React.MouseEvent<HTMLAnchorElement>) => {
    e.preventDefault();
    document.getElementById("demo")?.scrollIntoView({ behavior: "smooth" });
  };

  return (
    <section className="max-w-6xl mx-auto px-4 md:px-6 pt-28 md:pt-36 pb-16 md:pb-24">
      <div className="grid lg:grid-cols-[1.05fr_1fr] gap-12 lg:gap-14 items-center">
        <div className="min-w-0">
          <h1 className="font-wide font-black text-[2.6rem] leading-[0.98] sm:text-6xl lg:text-[3.6rem] tracking-[-0.03em]">
            {t("hero.title")}
          </h1>
          <p className="mt-6 md:mt-8 text-lg md:text-xl text-muted leading-relaxed max-w-[34rem]">
            {t("hero.subtitle")}
          </p>
          <div className="mt-8 md:mt-10 flex flex-col sm:flex-row gap-3">
            <a
              href="#demo"
              onClick={scrollToDemo}
              className="inline-flex items-center justify-center gap-2 px-5 py-3 rounded-md bg-ink text-paper font-semibold hover:bg-link transition-colors"
            >
              <ArrowDown className="w-4 h-4" />
              {t("hero.cta_demo")}
            </a>
            <Link
              to="/docs"
              className="inline-flex items-center justify-center gap-2 px-5 py-3 rounded-md border border-rule bg-surface font-semibold hover:border-ink transition-colors"
            >
              <BookOpen className="w-4 h-4" />
              {t("hero.cta_docs")}
            </Link>
          </div>
        </div>

        <div className="min-w-0">
          <QueryTrace />
        </div>
      </div>

      <div className="mt-16 md:mt-24">
        <h2 className="sr-only">{t("hero.specs.title")}</h2>
        <dl className="grid sm:grid-cols-2 lg:grid-cols-3 gap-x-10">
          {SPECS.map((s) => (
            <div key={s.key} className="border-t border-ink/80 pt-3 pb-7">
              <dt className="text-sm text-muted">{t(`hero.specs.${s.key}`)}</dt>
              <dd className="mt-1">
                <span className="font-wide font-bold text-lg leading-snug">{t(`hero.specs.${s.key}_value`)}</span>
                {"note" in s && (
                  <span className="block mt-1 text-sm text-muted">
                    {t(`hero.specs.${s.key}_note`)}
                  </span>
                )}
              </dd>
            </div>
          ))}
        </dl>
      </div>
    </section>
  );
}
