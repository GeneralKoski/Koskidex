import { useTranslation } from "react-i18next";
import DemoSetup from "../components/DemoSetup";
import Hero from "../components/Hero";
import IntegrationSnippets from "../components/IntegrationSnippets";
import SearchUI from "../components/SearchUI";
import { useDocumentMeta } from "../hooks/useDocumentMeta";

interface HomeProps {
  activeIndex: string;
  onIndexReady: (name: string) => void;
  onClear: () => void;
}

export default function Home({
  activeIndex,
  onIndexReady,
  onClear,
}: HomeProps) {
  const { t } = useTranslation();

  useDocumentMeta({
    title: t("meta.home_title"),
    description: t("meta.home_desc"),
    path: "/",
  });

  return (
    <>
      <Hero />

      <section id="demo" className="scroll-mt-16 bg-surface border-y border-rule">
        <div className="max-w-6xl mx-auto px-4 md:px-6 py-16 md:py-24 grid lg:grid-cols-[18rem_1fr] gap-10 lg:gap-14">
          <div className="min-w-0">
            <h2 className="font-wide font-black text-3xl md:text-4xl tracking-[-0.02em]">
              {t("demo.title")}
            </h2>
            <p className="mt-4 text-muted leading-relaxed">{t("demo.subtitle")}</p>
            <div className="mt-8">
              <DemoSetup
                activeIndex={activeIndex}
                onIndexReady={onIndexReady}
                onClear={onClear}
              />
            </div>
          </div>
          <div className="min-w-0">
            <SearchUI activeIndex={activeIndex} />
          </div>
        </div>
      </section>

      <IntegrationSnippets />
    </>
  );
}
