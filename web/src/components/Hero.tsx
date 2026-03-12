import { Box, Clock, Github, Layers, Server } from "lucide-react";
import { useTranslation } from "react-i18next";

export default function Hero() {
  const { t } = useTranslation();

  return (
    <section className="pt-40 pb-20 text-center container mx-auto px-4 relative">
      <h1 className="text-5xl md:text-7xl font-bold leading-tight mb-6 tracking-tight">
        The <span className="text-gradient">{t('hero.title_lightning')}</span>
        <br />
        {t('hero.title_rest')}
      </h1>
      <p className="text-lg md:text-xl text-slate-400 max-w-2xl mx-auto mb-10">
        {t('hero.subtitle')}
      </p>

      <div className="flex justify-center gap-4 mb-16">
        <a href="#demo" className="btn btn-primary">
          {t('hero.cta_demo')}
        </a>
        <a href="#integration" className="btn btn-outline">
          <Github className="w-5 h-5 mr-2" />
          {t('nav.github')}
        </a>
      </div>

      <div className="glass-effect rounded-2xl p-8 max-w-4xl mx-auto flex flex-col md:flex-row justify-between gap-8 md:gap-4">
        <div className="flex flex-col gap-2 items-center md:items-start text-center md:text-left">
          <Server className="w-6 h-6 text-blue-400 mb-1" />
          <span className="text-3xl font-bold">~15MB</span>
          <span className="text-sm text-slate-400">{t('hero.stats.binary_size')}</span>
        </div>

        <div className="flex flex-col gap-2 items-center md:items-start text-center md:text-left">
          <Layers className="w-6 h-6 text-purple-400 mb-1" />
          <span className="text-3xl font-bold">&lt;20MB</span>
          <span className="text-sm text-slate-400">{t('hero.stats.ram_usage')}</span>
        </div>

        <div className="flex flex-col gap-2 items-center md:items-start text-center md:text-left">
          <Clock className="w-6 h-6 text-green-400 mb-1" />
          <span className="text-3xl font-bold">&lt;5ms</span>
          <span className="text-sm text-slate-400">{t('hero.stats.response_time')}</span>
        </div>

        <div className="flex flex-col gap-2 items-center md:items-start text-center md:text-left">
          <Box className="w-6 h-6 text-orange-400 mb-1" />
          <span className="text-3xl font-bold">{t('hero.stats.zero')}</span>
          <span className="text-sm text-slate-400">{t('hero.stats.dependencies')}</span>
        </div>
      </div>
    </section>
  );
}
