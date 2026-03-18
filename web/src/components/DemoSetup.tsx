import { AlertCircle, CheckCircle2, Film, Package, Trash2 } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { Document } from "../types";

const API_URL = import.meta.env.VITE_API_URL || "/api";

interface DemoSetupProps {
  onIndexReady: (indexName: string) => void;
  onClear: () => void;
  activeIndex: string;
}

export default function DemoSetup({
  onIndexReady,
  onClear,
  activeIndex,
}: DemoSetupProps) {
  const { t } = useTranslation();
  const [status, setStatus] = useState<{
    type: "idle" | "loading" | "success" | "error";
    msg: string;
  }>({
    type: "idle",
    msg: "",
  });

  const setupIndex = async (indexName: string, data: Document[]) => {
    setStatus({
      type: "loading",
      msg: t("demo.setup.status.creating", { name: indexName }),
    });

    try {
      const createRes = await fetch(`${API_URL}/indexes`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: indexName }),
      });

      if (!createRes.ok && createRes.status !== 409) {
        throw new Error(`Failed to ensure index exists: ${createRes.status}`);
      }

      setStatus({
        type: "loading",
        msg: t("demo.setup.status.loading", { count: data.length }),
      });

      const res = await fetch(`${API_URL}/indexes/${indexName}/documents`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      });

      if (!res.ok) throw new Error("Failed to add documents");

      setStatus({
        type: "success",
        msg: t("demo.setup.status.success", { name: indexName }),
      });
      onIndexReady(indexName);
    } catch (e: unknown) {
      console.error(e);
      setStatus({
        type: "error",
        msg: t("demo.setup.status.error_connection", { url: API_URL }),
      });
    }
  };

  const handleLoadMovies = () => {
    const movies: Document[] = [
      { id: "m1", title: t("datasets.movies.titles.matrix"), genre: t("datasets.movies.genres.sci_fi"), year: 1999, director: "Wachowskis" },
      { id: "m2", title: t("datasets.movies.titles.godfather"), genre: t("datasets.movies.genres.crime"), year: 1972, director: "Francis Ford Coppola" },
      { id: "m3", title: t("datasets.movies.titles.goodfellas"), genre: t("datasets.movies.genres.biography"), year: 1990, director: "Martin Scorsese" },
      { id: "m4", title: t("datasets.movies.titles.pulp_fiction"), genre: t("datasets.movies.genres.crime"), year: 1994, director: "Quentin Tarantino" },
      { id: "m5", title: t("datasets.movies.titles.interstellar"), genre: t("datasets.movies.genres.sci_fi"), year: 2014, director: "Christopher Nolan" },
      { id: "m6", title: t("datasets.movies.titles.inception"), genre: t("datasets.movies.genres.action"), year: 2010, director: "Christopher Nolan" },
    ];
    setupIndex("movies", movies);
  };

  const handleLoadProducts = () => {
    const products: Document[] = [
      { id: "p1", name: t("datasets.products.names.macbook"), category: t("datasets.products.categories.laptops"), price: "$1999" },
      { id: "p2", name: t("datasets.products.names.iphone"), category: t("datasets.products.categories.smartphones"), price: "$999" },
      { id: "p3", name: t("datasets.products.names.samsung"), category: t("datasets.products.categories.smartphones"), price: "$1199" },
      { id: "p4", name: t("datasets.products.names.sony"), category: t("datasets.products.categories.audio"), price: "$398" },
      { id: "p5", name: t("datasets.products.names.dell"), category: t("datasets.products.categories.laptops"), price: "$1299" },
    ];
    setupIndex("products", products);
  };

  const handleClear = async () => {
    if (!activeIndex) return;
    setStatus({ type: "loading", msg: t("demo.setup.status.clearing") });
    try {
      await fetch(`${API_URL}/indexes/${activeIndex}`, { method: "DELETE" });
      setStatus({ type: "idle", msg: "" });
      onClear();
    } catch (e: unknown) {
      console.error(e);
      setStatus({ type: "error", msg: t("demo.setup.status.error_clear") });
    }
  };

  const isLoading = status.type === "loading";

  return (
    <div className="flex flex-col gap-3">
      {/* Dataset selector tabs */}
      <div className="flex flex-wrap items-center gap-2">
        <button
          onClick={handleLoadMovies}
          disabled={isLoading}
          aria-label={t("demo.setup.load_movies")}
          className={`flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-semibold transition-all ${
            activeIndex === "movies"
              ? "bg-blue-500/15 text-blue-400 border border-blue-500/30 shadow-sm shadow-blue-500/10"
              : "bg-white/5 text-slate-400 border border-white/5 hover:bg-white/10 hover:text-slate-200"
          } disabled:opacity-50`}
        >
          <Film className="w-4 h-4" />
          {t("demo.setup.load_movies")}
        </button>

        <button
          onClick={handleLoadProducts}
          disabled={isLoading}
          aria-label={t("demo.setup.load_products")}
          className={`flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-semibold transition-all ${
            activeIndex === "products"
              ? "bg-purple-500/15 text-purple-400 border border-purple-500/30 shadow-sm shadow-purple-500/10"
              : "bg-white/5 text-slate-400 border border-white/5 hover:bg-white/10 hover:text-slate-200"
          } disabled:opacity-50`}
        >
          <Package className="w-4 h-4" />
          {t("demo.setup.load_products")}
        </button>

        <div className="flex-1" />

        {activeIndex && (
          <button
            onClick={handleClear}
            disabled={isLoading}
            aria-label={t("demo.setup.clear")}
            className="flex items-center gap-1.5 px-3 py-2 rounded-lg text-xs font-medium text-red-400/70 hover:text-red-400 hover:bg-red-500/10 transition-all disabled:opacity-30"
          >
            <Trash2 className="w-3.5 h-3.5" />
            {t("demo.setup.clear")}
          </button>
        )}
      </div>

      {/* Status bar */}
      {status.msg && (
        <div className={`flex items-center gap-2 text-xs font-medium px-1 ${
          status.type === "loading" ? "text-blue-400" :
          status.type === "success" ? "text-emerald-400" :
          status.type === "error" ? "text-red-400" :
          "text-slate-500"
        }`}>
          {status.type === "loading" && (
            <span className="animate-spin w-3 h-3 border-2 border-current border-t-transparent rounded-full shrink-0" />
          )}
          {status.type === "success" && <CheckCircle2 className="w-3.5 h-3.5 shrink-0" />}
          {status.type === "error" && <AlertCircle className="w-3.5 h-3.5 shrink-0" />}
          {status.msg}
        </div>
      )}
    </div>
  );
}
