import { Github, Globe, Zap } from "lucide-react";
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

  return (
    <nav
      className={`fixed top-0 left-0 right-0 z-50 transition-all duration-500 transform ${
        showNav ? "translate-y-0 opacity-100" : "-translate-y-full opacity-0"
      } ${
        lastScrollY > 50
          ? "glass-effect bg-slate-900/80 border-b border-white/5 py-4 shadow-2xl shadow-blue-500/10 rounded-b-3xl rounded-t-none"
          : "bg-transparent py-7 border-transparent"
      }`}
    >
      <div className="container mx-auto px-6 h-20 flex justify-between items-center">
        <Link
          to="/"
          onClick={(e) => {
            if (isHome) {
              e.preventDefault();
              window.scrollTo({ top: 0, behavior: "smooth" });
            }
          }}
          className="text-3xl font-black flex items-center gap-2 tracking-tighter text-white"
        >
          <Zap className="w-8 h-8 text-blue-500 fill-blue-500" />
          Koskidex
        </Link>
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
            className="text-slate-300 hover:text-white transition-colors italic"
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
