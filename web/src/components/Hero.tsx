import { Box, Clock, Layers, Play, Server, FileText } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

export default function Hero() {
  const { t } = useTranslation();

  return (
    <section className="pt-32 md:pt-48 pb-16 md:pb-24 text-center container mx-auto px-4 relative">
      <h1 className="text-4xl sm:text-6xl md:text-8xl font-black leading-[1.1] mb-6 md:mb-8 tracking-tighter text-white">
        {t("hero.the")}{" "}
        <span className="text-gradient drop-shadow-sm">
          {t("hero.title_lightning")}
        </span>
        <br />
        <span className="opacity-90">{t("hero.title_rest")}</span>
      </h1>
      <p className="text-base sm:text-xl md:text-2xl text-slate-400/80 max-w-3xl mx-auto mb-10 md:mb-14 font-light leading-relaxed">
        {t("hero.subtitle")}
      </p>

      <div className="flex flex-col sm:flex-row flex-wrap justify-center gap-3 sm:gap-6 mb-12 md:mb-16">
        <a
          href="#demo"
          className="group relative inline-flex items-center justify-center px-7 py-4 sm:px-10 sm:py-5 font-bold text-white transition-all duration-300 bg-blue-600 rounded-2xl hover:bg-blue-500 hover:scale-105 active:scale-95 shadow-xl shadow-blue-500/20"
        >
          <div className="absolute -inset-0.5 bg-gradient-to-r from-blue-500 to-purple-500 rounded-2xl blur opacity-30 group-hover:opacity-50 transition duration-300"></div>
          <span className="relative flex items-center gap-2 text-base sm:text-xl">
            <Play className="w-5 h-5 sm:w-6 sm:h-6 fill-current animate-pulse" />
            {t("hero.cta_demo")}
          </span>
        </a>

        <Link
          to="/docs"
          className="group inline-flex items-center justify-center px-7 py-4 sm:px-10 sm:py-5 font-bold text-slate-300 transition-all duration-300 bg-white/5 border border-white/10 rounded-2xl hover:bg-white/10 hover:text-white hover:scale-105 active:scale-95"
        >
          <FileText className="w-5 h-5 mr-2 text-slate-500 group-hover:text-blue-400 transition-colors" />
          <span className="text-base sm:text-xl">{t("hero.cta_docs")}</span>
        </Link>
      </div>

      <div className="glass-effect rounded-2xl md:rounded-3xl p-6 md:p-10 max-w-5xl mx-auto grid grid-cols-2 md:grid-cols-4 gap-6 md:gap-4 relative overflow-hidden">
        <div className="flex flex-col items-center text-center px-2 md:px-4">
          <div className="w-10 h-10 md:w-12 md:h-12 rounded-xl md:rounded-2xl bg-blue-500/10 flex items-center justify-center mb-3 md:mb-4">
            <Server className="w-5 h-5 md:w-6 md:h-6 text-blue-400" />
          </div>
          <span className="text-2xl md:text-3xl font-bold mb-1">~15MB</span>
          <span className="text-[11px] md:text-sm text-slate-400 uppercase tracking-widest font-semibold">
            {t("hero.stats.binary_size")}
          </span>
        </div>

        <div className="flex flex-col items-center text-center px-2 md:px-4 md:border-l md:border-white/5">
          <div className="w-10 h-10 md:w-12 md:h-12 rounded-xl md:rounded-2xl bg-purple-500/10 flex items-center justify-center mb-3 md:mb-4">
            <Layers className="w-5 h-5 md:w-6 md:h-6 text-purple-400" />
          </div>
          <span className="text-2xl md:text-3xl font-bold mb-1">&lt;20MB</span>
          <span className="text-[11px] md:text-sm text-slate-400 uppercase tracking-widest font-semibold">
            {t("hero.stats.ram_usage")}
          </span>
        </div>

        <div className="flex flex-col items-center text-center px-2 md:px-4 md:border-l md:border-white/5">
          <div className="w-10 h-10 md:w-12 md:h-12 rounded-xl md:rounded-2xl bg-green-500/10 flex items-center justify-center mb-3 md:mb-4">
            <Clock className="w-5 h-5 md:w-6 md:h-6 text-green-400" />
          </div>
          <span className="text-2xl md:text-3xl font-bold mb-1">&lt;5ms</span>
          <span className="text-[11px] md:text-sm text-slate-400 uppercase tracking-widest font-semibold">
            {t("hero.stats.response_time")}
          </span>
        </div>

        <div className="flex flex-col items-center text-center px-2 md:px-4 md:border-l md:border-white/5">
          <div className="w-10 h-10 md:w-12 md:h-12 rounded-xl md:rounded-2xl bg-orange-500/10 flex items-center justify-center mb-3 md:mb-4">
            <Box className="w-5 h-5 md:w-6 md:h-6 text-orange-400" />
          </div>
          <span className="text-2xl md:text-3xl font-bold mb-1">
            {t("hero.stats.zero")}
          </span>
          <span className="text-[11px] md:text-sm text-slate-400 uppercase tracking-widest font-semibold">
            {t("hero.stats.dependencies")}
          </span>
        </div>
      </div>
    </section>
  );
}
