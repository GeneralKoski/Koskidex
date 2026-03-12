import { useState, useEffect } from 'react';
import { Zap, Globe } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import Hero from './components/Hero';
import DemoSetup from './components/DemoSetup';
import SearchUI from './components/SearchUI';
import IntegrationSnippets from './components/IntegrationSnippets';

function App() {
  const { t, i18n } = useTranslation();
  const [activeIndex, setActiveIndex] = useState<string>('');

  const handleIndexReady = (indexName: string) => {
    setActiveIndex(indexName);
  };

  const handleClear = () => {
    setActiveIndex('');
  };

  const changeLanguage = (lng: string) => {
    i18n.changeLanguage(lng);
  };

  useEffect(() => {
    document.documentElement.lang = i18n.language;
  }, [i18n.language]);

  return (
    <>
      <div className="blob blob-1"></div>
      <div className="blob blob-2"></div>

      <nav className="fixed top-0 left-0 right-0 z-50 glass-effect border-x-0 border-t-0 rounded-none bg-slate-900/60 transition-all duration-300">
        <div className="container mx-auto px-6 h-20 flex justify-between items-center">
          <div className="text-2xl font-bold flex items-center gap-2 tracking-tight">
            <Zap className="w-6 h-6 text-blue-500 fill-blue-500/20" />
            Koskidex
          </div>
          <div className="hidden md:flex items-center gap-8 font-medium">
            <a href="#demo" className="text-slate-300 hover:text-white transition-colors" aria-label={t('nav.demo')}>
              {t('nav.demo')}
            </a>
            <a href="#integration" className="text-slate-300 hover:text-white transition-colors" aria-label={t('nav.integration')}>
              {t('nav.integration')}
            </a>
            
            <div className="flex items-center gap-2 bg-slate-800/50 px-3 py-1.5 rounded-lg border border-slate-700/50">
              <Globe className="w-4 h-4 text-slate-400" />
              <select 
                value={i18n.language.split('-')[0]} 
                onChange={(e: React.ChangeEvent<HTMLSelectElement>) => changeLanguage(e.target.value)}
                className="bg-transparent text-sm text-slate-300 focus:outline-none cursor-pointer"
                aria-label="Select Language"
              >
                <option value="en">EN</option>
                <option value="it">IT</option>
              </select>
            </div>

            <a href="https://github.com/general-koski/koskidex" target="_blank" rel="noreferrer" className="btn btn-outline border-slate-700 hover:bg-slate-800" aria-label="View on GitHub">
              {t('nav.github')}
            </a>
          </div>
        </div>
      </nav>

      <main className="min-h-screen">
        <Hero />
        
        <section id="demo" className="py-20 bg-slate-900/40 relative border-y border-slate-800/50">
          <div className="container mx-auto px-4 relative z-10">
            <h2 className="text-4xl font-bold text-center mb-4">{t('demo.title')}</h2>
            <p className="text-slate-400 text-center mb-12 text-lg">{t('demo.subtitle')}</p>
            
            <DemoSetup 
              activeIndex={activeIndex} 
              onIndexReady={handleIndexReady} 
              onClear={handleClear} 
            />
            
            <SearchUI activeIndex={activeIndex} />
          </div>
          
          <div className="absolute inset-0 bg-[url('https://grainy-gradients.vercel.app/noise.svg')] opacity-20 mix-blend-overlay pointer-events-none"></div>
        </section>

        <IntegrationSnippets />
      </main>

      <footer className="py-12 border-t border-slate-800/50 text-center text-slate-500 bg-[#0B1120]">
        <div className="container mx-auto px-4">
          <div className="flex items-center justify-center gap-2 mb-4">
            <Zap className="w-5 h-5 text-blue-500/50" />
            <span className="font-semibold text-slate-400">Koskidex</span>
          </div>
          <p>{t('footer.made_with')}</p>
        </div>
      </footer>
    </>
  );
}

export default App;
