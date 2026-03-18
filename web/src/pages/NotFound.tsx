import { Home, SearchX } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

export default function NotFound() {
  const { t } = useTranslation();

  return (
    <div className="min-h-screen flex items-center justify-center px-6">
      <div className="text-center">
        <SearchX className="w-14 h-14 md:w-20 md:h-20 text-blue-500/30 mx-auto mb-4 md:mb-6" />
        <h1 className="text-6xl md:text-8xl font-black text-white mb-3 md:mb-4">404</h1>
        <p className="text-lg md:text-xl text-slate-400 mb-6 md:mb-8">
          {t("errors.page_not_found", "Page not found")}
        </p>
        <Link
          to="/"
          className="inline-flex items-center gap-2 px-6 py-3 bg-blue-600 hover:bg-blue-500 text-white font-semibold rounded-xl transition-colors"
        >
          <Home className="w-5 h-5" />
          {t("docs.back_home", "Back to Home")}
        </Link>
      </div>
    </div>
  );
}
