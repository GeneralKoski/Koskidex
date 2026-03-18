import { Github, Globe, Menu, X, Zap } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  BrowserRouter,
  Link,
  Route,
  Routes,
  useLocation,
} from "react-router-dom";
import ErrorBoundary from "./components/ErrorBoundary";
import Documentation from "./pages/Documentation";
import Home from "./pages/Home";
import NotFound from "./pages/NotFound";

function Navbar({
  changeLanguage,
  lastScrollY,
  showNav,
}: {
  changeLanguage: (lng: string) => void;
  lastScrollY: number;
  showNav: boolean;
}) {
  const { t, i18n } = useTranslation();
  const location = useLocation();
  const isHome = location.pathname === "/";
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    setMobileOpen(false);
  }, [location.pathname]);

  return (
    <nav
      className={`fixed top-0 left-0 right-0 z-50 transition-all duration-500 transform ${
        showNav || mobileOpen ? "translate-y-0 opacity-100" : "-translate-y-full opacity-0"
      } ${
        lastScrollY > 50 || mobileOpen
          ? "glass-effect bg-slate-900/80 border-b border-white/5 py-3 md:py-4 shadow-2xl shadow-blue-500/10 rounded-b-3xl rounded-t-none"
          : "bg-transparent py-5 md:py-7 border-transparent"
      }`}
    >
      <div className="container mx-auto px-4 md:px-6 h-14 md:h-20 flex justify-between items-center">
        <Link
          to="/"
          onClick={(e) => {
            if (isHome) {
              e.preventDefault();
              window.scrollTo({ top: 0, behavior: "smooth" });
            }
          }}
          className="text-2xl md:text-3xl font-black flex items-center gap-2 tracking-tighter text-white"
        >
          <Zap className="w-6 h-6 md:w-8 md:h-8 text-blue-500 fill-blue-500" />
          Koskidex
        </Link>

        {/* Mobile controls */}
        <div className="flex md:hidden items-center gap-3">
          <label className="flex items-center gap-1.5 bg-slate-800/40 px-2 py-1 rounded-lg border border-white/5 cursor-pointer">
            <Globe className="w-3.5 h-3.5 text-slate-400" />
            <select
              value={i18n.language.split("-")[0]}
              onChange={(e) => changeLanguage(e.target.value)}
              className="bg-transparent text-xs text-slate-300 focus:outline-none cursor-pointer font-medium appearance-none"
              aria-label={t("nav.select_language")}
            >
              <option value="en" className="bg-slate-900">EN</option>
              <option value="it" className="bg-slate-900">IT</option>
            </select>
          </label>
          <button
            onClick={() => setMobileOpen(!mobileOpen)}
            className="p-2 text-slate-300 hover:text-white transition-colors"
            aria-label="Toggle menu"
          >
            {mobileOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
          </button>
        </div>

        {/* Desktop nav */}
        <div className="hidden md:flex items-center gap-8 font-medium">
          {isHome ? (
            <>
              <a
                href="#demo"
                className="text-slate-300 hover:text-white transition-colors"
              >
                {t("nav.demo")}
              </a>
              <a
                href="#integration"
                className="text-slate-300 hover:text-white transition-colors"
              >
                {t("nav.integration")}
              </a>
            </>
          ) : (
            <Link
              to="/"
              className="text-slate-300 hover:text-white transition-colors"
            >
              {t("docs.back_home")}
            </Link>
          )}

          <Link
            to="/docs"
            className="text-slate-300 hover:text-white transition-colors"
          >
            {t("nav.features")}
          </Link>

          <label
            className="flex items-center gap-2 bg-slate-800/40 hover:bg-slate-800/60 px-3 py-1.5 rounded-lg border border-white/5 hover:border-white/10 transition-all duration-300 cursor-pointer group"
            onClick={(e) => {
              const select = e.currentTarget.querySelector("select");
              if (select) {
                try {
                  select.showPicker();
                } catch {
                  select.focus();
                }
              }
            }}
          >
            <Globe className="w-4 h-4 text-slate-400 group-hover:text-blue-400 transition-colors" />
            <select
              value={i18n.language.split("-")[0]}
              onChange={(e) => changeLanguage(e.target.value)}
              onClick={(e) => e.stopPropagation()}
              className="bg-transparent text-sm text-slate-300 group-hover:text-white focus:outline-none cursor-pointer font-medium appearance-none"
              aria-label={t("nav.select_language")}
            >
              <option value="en" className="bg-slate-900">
                EN
              </option>
              <option value="it" className="bg-slate-900">
                IT
              </option>
            </select>
          </label>

          <a
            href="https://github.com/GeneralKoski/Koskidex"
            target="_blank"
            rel="noreferrer"
            className="group relative inline-flex items-center gap-2 px-3 py-1.5 bg-slate-800/40 hover:bg-slate-800/60 text-slate-300 hover:text-white rounded-lg border border-white/5 hover:border-blue-500/30 transition-all duration-300 shadow-lg hover:shadow-blue-500/10"
            aria-label={t("nav.github")}
          >
            <Github className="w-4 h-4 group-hover:scale-110 transition-transform duration-300" />
            <span className="text-sm font-medium">{t("nav.github")}</span>
          </a>
        </div>
      </div>

      {/* Mobile dropdown */}
      {mobileOpen && (
        <div className="md:hidden border-t border-white/5 px-4 py-4 flex flex-col gap-3 bg-slate-900/95 backdrop-blur-xl">
          {isHome ? (
            <>
              <a href="#demo" onClick={() => setMobileOpen(false)} className="text-slate-300 hover:text-white transition-colors py-2 text-sm font-medium">{t("nav.demo")}</a>
              <a href="#integration" onClick={() => setMobileOpen(false)} className="text-slate-300 hover:text-white transition-colors py-2 text-sm font-medium">{t("nav.integration")}</a>
            </>
          ) : (
            <Link to="/" className="text-slate-300 hover:text-white transition-colors py-2 text-sm font-medium">{t("docs.back_home")}</Link>
          )}
          <Link to="/docs" className="text-slate-300 hover:text-white transition-colors py-2 text-sm font-medium">{t("nav.features")}</Link>
          <a
            href="https://github.com/GeneralKoski/Koskidex"
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-2 text-slate-300 hover:text-white transition-colors py-2 text-sm font-medium"
          >
            <Github className="w-4 h-4" />
            {t("nav.github")}
          </a>
        </div>
      )}
    </nav>
  );
}

function AppContent() {
  const { t, i18n } = useTranslation();
  const [activeIndex, setActiveIndex] = useState<string>("");
  const [showNav, setShowNav] = useState(true);
  const [lastScrollY, setLastScrollY] = useState(0);

  useEffect(() => {
    const handleScroll = () => {
      if (window.scrollY > lastScrollY && window.scrollY > 100)
        setShowNav(false);
      else setShowNav(true);
      setLastScrollY(window.scrollY);
    };
    window.addEventListener("scroll", handleScroll);
    return () => window.removeEventListener("scroll", handleScroll);
  }, [lastScrollY]);

  const changeLanguage = (lng: string) => i18n.changeLanguage(lng);

  useEffect(() => {
    document.documentElement.lang = i18n.language;
  }, [i18n.language]);

  return (
    <>
      <Navbar
        showNav={showNav}
        lastScrollY={lastScrollY}
        changeLanguage={changeLanguage}
      />
      <main className="min-h-screen">
        <Routes>
          <Route
            path="/"
            element={
              <Home
                activeIndex={activeIndex}
                onIndexReady={(name) => setActiveIndex(name)}
                onClear={() => setActiveIndex("")}
              />
            }
          />
          <Route path="/docs" element={<Documentation />} />
          <Route path="*" element={<NotFound />} />
        </Routes>
      </main>

      <footer className="py-12 border-t border-slate-800/50 text-center text-slate-500 bg-[#0B1120]">
        <div className="container mx-auto px-4">
          <div className="flex items-center justify-center gap-2 mb-4">
            <Zap className="w-5 h-5 text-blue-500/50" />
            <span className="font-semibold text-slate-400">
              Koskidex {t("common.search_engine")}
            </span>
          </div>
        </div>
      </footer>
    </>
  );
}

function App() {
  return (
    <ErrorBoundary>
      <BrowserRouter>
        <div className="blob blob-1"></div>
        <div className="blob blob-2"></div>
        <AppContent />
      </BrowserRouter>
    </ErrorBoundary>
  );
}

export default App;
