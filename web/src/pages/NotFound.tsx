import { ArrowLeft } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link, useLocation } from "react-router-dom";
import { useDocumentMeta } from "../hooks/useDocumentMeta";

export default function NotFound() {
  const { t } = useTranslation();
  const { pathname } = useLocation();

  useDocumentMeta({
    title: t("meta.not_found_title"),
    description: t("meta.not_found_desc"),
  });

  return (
    <div className="min-h-screen max-w-6xl mx-auto px-4 md:px-6 pt-32 pb-20 flex items-center">
      <div className="max-w-xl">
        <p className="font-mono text-sm text-muted">GET {pathname} 404</p>
        <h1 className="mt-4 font-wide font-black text-7xl md:text-9xl tracking-[-0.04em]">404</h1>
        <p className="mt-4 text-lg text-muted leading-relaxed">{t("errors.page_not_found")}</p>
        <Link
          to="/"
          className="mt-8 inline-flex items-center gap-2 px-5 py-3 rounded-md bg-ink text-paper font-semibold hover:bg-link transition-colors"
        >
          <ArrowLeft className="w-4 h-4" />
          {t("docs.back_home")}
        </Link>
      </div>
    </div>
  );
}
