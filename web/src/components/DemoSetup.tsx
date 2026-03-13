import { AlertCircle, CheckCircle2, Film, Package, Trash2 } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { Document } from "../types";

const API_URL = import.meta.env.VITE_API_URL || "http://localhost:7700";

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
  const [previewTags, setPreviewTags] = useState<string[]>([]);
  const [status, setStatus] = useState<{
    type: "idle" | "loading" | "success" | "error";
    msg: string;
  }>({
    type: "idle",
    msg: "",
  });

  const setupIndex = async (indexName: string, data: Document[], tags: string[]) => {
    setStatus({
      type: "loading",
      msg: t("demo.setup.status.creating", { name: indexName }),
    });

    try {
      // Create Index (handle already exists 409 gracefully)
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

      // Add Documents
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
      setPreviewTags(tags);
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
      {
        id: "m1",
        title: t("datasets.movies.titles.matrix"),
        genre: t("datasets.movies.genres.sci_fi"),
        year: 1999,
        director: "Wachowskis",
      },
      {
        id: "m2",
        title: t("datasets.movies.titles.godfather"),
        genre: t("datasets.movies.genres.crime"),
        year: 1972,
        director: "Francis Ford Coppola",
      },
      {
        id: "m3",
        title: t("datasets.movies.titles.goodfellas"),
        genre: t("datasets.movies.genres.biography"),
        year: 1990,
        director: "Martin Scorsese",
      },
      {
        id: "m4",
        title: t("datasets.movies.titles.pulp_fiction"),
        genre: t("datasets.movies.genres.crime"),
        year: 1994,
        director: "Quentin Tarantino",
      },
      {
        id: "m5",
        title: t("datasets.movies.titles.interstellar"),
        genre: t("datasets.movies.genres.sci_fi"),
        year: 2014,
        director: "Christopher Nolan",
      },
      {
        id: "m6",
        title: t("datasets.movies.titles.inception"),
        genre: t("datasets.movies.genres.action"),
        year: 2010,
        director: "Christopher Nolan",
      },
    ];
    setupIndex("movies", movies, [t("datasets.movies.titles.matrix"), t("datasets.movies.titles.inception"), "Tarantino", t("datasets.movies.genres.sci_fi"), t("datasets.movies.titles.godfather")]);
  };

  const handleLoadProducts = () => {
    const products: Document[] = [
      {
        id: "p1",
        name: t("datasets.products.names.macbook"),
        category: t("datasets.products.categories.laptops"),
        price: "$1999",
      },
      {
        id: "p2",
        name: t("datasets.products.names.iphone"),
        category: t("datasets.products.categories.smartphones"),
        price: "$999",
      },
      {
        id: "p3",
        name: t("datasets.products.names.samsung"),
        category: t("datasets.products.categories.smartphones"),
        price: "$1199",
      },
      {
        id: "p4",
        name: t("datasets.products.names.sony"),
        category: t("datasets.products.categories.audio"),
        price: "$398",
      },
      {
        id: "p5",
        name: t("datasets.products.names.dell"),
        category: t("datasets.products.categories.laptops"),
        price: "$1299",
      },
    ];
    setupIndex("products", products, ["MacBook", "iPhone", t("datasets.products.categories.smartphones"), t("datasets.products.categories.audio"), t("datasets.products.categories.laptops")]);
  };

  const handleClear = async () => {
    if (!activeIndex) return;
    setStatus({ type: "loading", msg: t("demo.setup.status.clearing") });
    try {
      await fetch(`${API_URL}/indexes/${activeIndex}`, { method: "DELETE" });
      setStatus({ type: "idle", msg: t("demo.setup.status.cleared") });
      setPreviewTags([]);
      onClear();
    } catch (e: unknown) {
      console.error(e);
      setStatus({ type: "error", msg: t("demo.setup.status.error_clear") });
    }
  };

  return (
    <div className="glass-effect rounded-[2.5rem] p-10 md:p-16 mb-16 text-center max-w-5xl mx-auto shadow-2xl shadow-blue-500/5 border-white/5 relative overflow-hidden">
      <div className="flex items-center justify-center gap-4 mb-8">
        <div className="h-px w-16 bg-gradient-to-r from-transparent via-blue-500/20 to-transparent"></div>
        <h3 className="text-3xl font-black tracking-tight uppercase text-slate-200">{t("demo.setup.title")}</h3>
        <div className="h-px w-16 bg-gradient-to-l from-transparent via-blue-500/20 to-transparent"></div>
      </div>
      <p className="text-slate-400 mb-12 text-xl max-w-2xl mx-auto leading-relaxed font-light">{t("demo.setup.subtitle")}</p>

      <div className="flex flex-wrap justify-center items-center gap-6 mb-12">
        <button
          onClick={handleLoadMovies}
          disabled={status.type === "loading"}
          aria-label={t("demo.setup.load_movies")}
          className={`relative px-8 py-4 rounded-2xl font-black transition-all flex items-center gap-3 transform hover:scale-105 active:scale-95 group ${
            activeIndex === "movies" 
              ? "bg-blue-600 text-white shadow-xl shadow-blue-500/40" 
              : "bg-white/5 hover:bg-white/10 text-slate-300 border border-white/10"
          }`}
        >
          {activeIndex === "movies" && (
            <div className="absolute -inset-0.5 bg-blue-500 rounded-2xl blur opacity-30 animate-pulse"></div>
          )}
          <Film className={`relative w-6 h-6 ${activeIndex === "movies" ? "animate-pulse" : ""}`} /> 
          <span className="relative">{t("demo.setup.load_movies")}</span>
        </button>

        <button
          onClick={handleLoadProducts}
          disabled={status.type === "loading"}
          aria-label={t("demo.setup.load_products")}
          className={`relative px-8 py-4 rounded-2xl font-black transition-all flex items-center gap-3 transform hover:scale-105 active:scale-95 group ${
            activeIndex === "products" 
              ? "bg-purple-600 text-white shadow-xl shadow-purple-500/40" 
              : "bg-white/5 hover:bg-white/10 text-slate-300 border border-white/10"
          }`}
        >
          {activeIndex === "products" && (
            <div className="absolute -inset-0.5 bg-purple-500 rounded-2xl blur opacity-30 animate-pulse"></div>
          )}
          <Package className={`relative w-6 h-6 ${activeIndex === "products" ? "animate-pulse" : ""}`} /> 
          <span className="relative">{t("demo.setup.load_products")}</span>
        </button>

        <div className="h-12 w-px bg-white/10 mx-4 hidden md:block"></div>

        <button
          onClick={handleClear}
          disabled={!activeIndex || status.type === "loading"}
          aria-label={t("demo.setup.clear")}
          className="relative px-6 py-4 rounded-2xl bg-red-500/5 hover:bg-red-500/10 text-red-500/70 hover:text-red-500 border border-red-500/10 transition-all flex items-center gap-3 group disabled:opacity-30 disabled:hover:scale-100 transform hover:scale-105 active:scale-95"
        >
          <Trash2 className="w-6 h-6 group-hover:rotate-12 transition-transform" /> 
          <span className="font-bold">{t("demo.setup.clear")}</span>
        </button>
      </div>

      <div className="min-h-[32px] mb-8 flex items-center justify-center">
        {status.type === "loading" && (
          <span className="text-blue-400 animate-pulse font-medium">{status.msg}</span>
        )}
        {status.type === "success" && (
          <span className="text-emerald-400 flex items-center gap-2 font-medium">
            <CheckCircle2 className="w-4 h-4" /> {status.msg}
          </span>
        )}
        {status.type === "error" && (
          <span className="text-red-400 flex items-center gap-2 font-medium">
            <AlertCircle className="w-4 h-4" /> {status.msg}
          </span>
        )}
        {status.type === "idle" && status.msg && (
          <span className="text-slate-400 font-medium">{status.msg}</span>
        )}
      </div>

      {previewTags.length > 0 && (
        <div className="pt-8 border-t border-white/5 animate-in fade-in slide-in-from-bottom-4 duration-500">
          <h4 className="text-sm font-bold text-slate-500 uppercase tracking-[0.2em] mb-4">{t("demo.setup.preview_title")}</h4>
          <p className="text-slate-400 mb-6 font-light">{t("demo.setup.preview_subtitle")}</p>
          <div className="flex flex-wrap justify-center gap-3">
            {previewTags.map((tag) => (
              <span 
                key={tag}
                className="px-4 py-2 rounded-xl bg-white/5 border border-white/10 text-slate-300 text-sm font-medium hover:bg-white/10 hover:border-blue-500/30 transition-all cursor-default"
              >
                {tag}
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
