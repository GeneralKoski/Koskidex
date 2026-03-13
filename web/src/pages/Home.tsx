import { useTranslation } from "react-i18next";
import DemoSetup from "../components/DemoSetup";
import Hero from "../components/Hero";
import IntegrationSnippets from "../components/IntegrationSnippets";
import SearchUI from "../components/SearchUI";

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

  return (
    <>
      <Hero />

      <section
        id="demo"
        className="py-20 bg-slate-900/40 relative border-y border-slate-800/50"
      >
        <div className="container mx-auto px-4 relative z-10">
          <h2 className="text-4xl font-bold text-center mb-4">
            {t("demo.title")}
          </h2>
          <p className="text-slate-400 text-center mb-12 text-lg">
            {t("demo.subtitle")}
          </p>

          <div className="grid lg:grid-cols-2 gap-8 items-start max-w-7xl mx-auto">
            <DemoSetup
              activeIndex={activeIndex}
              onIndexReady={onIndexReady}
              onClear={onClear}
            />
            <SearchUI activeIndex={activeIndex} />
          </div>
        </div>

        <div className="absolute inset-0 bg-[url('https://grainy-gradients.vercel.app/noise.svg')] opacity-20 mix-blend-overlay pointer-events-none"></div>
      </section>

      <IntegrationSnippets />
    </>
  );
}
