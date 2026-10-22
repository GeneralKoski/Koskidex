import { Github, Menu, X } from "lucide-react";
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

const LANGUAGES = ["en", "it"] as const;

function Logo() {
  return (
    <span className="flex items-center gap-2.5">
      <span
        aria-hidden="true"
        className="grid place-items-center w-7 h-7 rounded-[5px] bg-ink text-paper font-wide font-black text-sm"
      >
        k
      </span>
      <span className="font-wide font-extrabold text-lg tracking-tight">Koskidex</span>
    </span>
  );
}

function LanguageSwitch({ changeLanguage }: { changeLanguage: (lng: string) => void }) {
  const { t, i18n } = useTranslation();
  const current = i18n.language.split("-")[0];
  return (
    <div role="group" aria-label={t("nav.select_language")} className="flex rounded-md border border-rule p-0.5 text-xs font-semibold">
      {LANGUAGES.map((lng) => (
        <button
          key={lng}
          type="button"
          onClick={() => changeLanguage(lng)}
          aria-pressed={current === lng}
          className={`px-2 py-1 rounded-[4px] uppercase transition-colors ${
            current === lng ? "bg-ink text-paper" : "text-muted hover:text-ink"
          }`}
        >
          {lng}
        </button>
      ))}
    </div>
  );
}

function Navbar({
  changeLanguage,
  lastScrollY,
  showNav,
}: {
  changeLanguage: (lng: string) => void;
  lastScrollY: number;
  showNav: boolean;
}) {
  const { t } = useTranslation();
  const location = useLocation();
  const isHome = location.pathname === "/";
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    setMobileOpen(false);
  }, [location.pathname]);

  const linkClass = "text-muted hover:text-ink transition-colors";

  return (
    <nav
      className={`fixed top-0 left-0 right-0 z-50 transition-transform duration-300 ${
        showNav || mobileOpen ? "translate-y-0" : "-translate-y-full"
      } ${
        lastScrollY > 20 || mobileOpen
          ? "bg-paper/90 backdrop-blur-md border-b border-rule"
          : "bg-transparent border-b border-transparent"
      }`}
    >
      <div className="max-w-6xl mx-auto px-4 md:px-6 h-16 flex justify-between items-center">
        <Link
          to="/"
          aria-label="Koskidex"
          onClick={(e) => {
            if (isHome) {
              e.preventDefault();
              window.scrollTo({ top: 0, behavior: "smooth" });
            }
          }}
        >
          <Logo />
        </Link>

        <div className="flex md:hidden items-center gap-2">
          <LanguageSwitch changeLanguage={changeLanguage} />
          <button
            type="button"
            onClick={() => setMobileOpen(!mobileOpen)}
            className="p-2 -mr-2 text-ink"
            aria-label={t("nav.toggle_menu")}
            aria-expanded={mobileOpen}
          >
            {mobileOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
          </button>
        </div>

        <div className="hidden md:flex items-center gap-7 text-sm font-medium">
          {isHome ? (
            <>
              <a href="#demo" className={linkClass}>{t("nav.demo")}</a>
              <a href="#integration" className={linkClass}>{t("nav.integration")}</a>
            </>
          ) : (
            <Link to="/" className={linkClass}>{t("docs.back_home")}</Link>
          )}
          <Link to="/docs" className={linkClass}>{t("nav.features")}</Link>
          <LanguageSwitch changeLanguage={changeLanguage} />
          <a
            href="https://github.com/GeneralKoski/Koskidex"
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-2 px-3 py-1.5 rounded-md border border-ink/80 text-ink hover:bg-ink hover:text-paper transition-colors"
          >
            <Github className="w-4 h-4" />
            {t("nav.github")}
          </a>
        </div>
      </div>

      {mobileOpen && (
        <div className="md:hidden border-t border-rule bg-paper px-4 py-3 flex flex-col text-base font-medium">
          {isHome ? (
            <>
              <a href="#demo" onClick={() => setMobileOpen(false)} className="py-3 border-b border-rule">{t("nav.demo")}</a>
              <a href="#integration" onClick={() => setMobileOpen(false)} className="py-3 border-b border-rule">{t("nav.integration")}</a>
            </>
          ) : (
            <Link to="/" className="py-3 border-b border-rule">{t("docs.back_home")}</Link>
          )}
          <Link to="/docs" className="py-3 border-b border-rule">{t("nav.features")}</Link>
          <a
            href="https://github.com/GeneralKoski/Koskidex"
            target="_blank"
            rel="noreferrer"
            className="py-3 flex items-center gap-2"
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
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-[60] focus:px-3 focus:py-2 focus:bg-ink focus:text-paper focus:rounded-md"
      >
        {t("nav.skip")}
      </a>
      <Navbar
        showNav={showNav}
        lastScrollY={lastScrollY}
        changeLanguage={changeLanguage}
      />
      <main id="main" className="min-h-screen">
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

      <footer className="border-t border-rule">
        <div className="max-w-6xl mx-auto px-4 md:px-6 py-10 flex flex-col sm:flex-row gap-6 sm:items-center justify-between text-sm text-muted">
          <div className="flex flex-col gap-1">
            <span className="text-ink"><Logo /></span>
            <span className="mt-2">{t("footer.license")} {t("footer.made_by")}</span>
            <span className="max-w-xl text-xs">{t("footer.privacy")}</span>
          </div>
          <div className="flex gap-6">
            <Link to="/docs" className="hover:text-ink transition-colors">{t("nav.features")}</Link>
            <a href="https://github.com/GeneralKoski/Koskidex" target="_blank" rel="noreferrer" className="hover:text-ink transition-colors">
              {t("nav.github")}
            </a>
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
        <AppContent />
      </BrowserRouter>
    </ErrorBoundary>
  );
}

export default App;
