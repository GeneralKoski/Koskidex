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
      // Create Index (we don't care if it errors because it already exists)
      await fetch(`${API_URL}/indexes`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: indexName }),
      });

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
        title: "The Matrix",
        genre: "Sci-Fi",
        year: 1999,
        director: "Wachowskis",
      },
      {
        id: "m2",
        title: "The Godfather",
        genre: "Crime",
        year: 1972,
        director: "Francis Ford Coppola",
      },
      {
        id: "m3",
        title: "Goodfellas",
        genre: "Biography",
        year: 1990,
        director: "Martin Scorsese",
      },
      {
        id: "m4",
        title: "Pulp Fiction",
        genre: "Crime",
        year: 1994,
        director: "Quentin Tarantino",
      },
      {
        id: "m5",
        title: "Interstellar",
        genre: "Sci-Fi",
        year: 2014,
        director: "Christopher Nolan",
      },
      {
        id: "m6",
        title: "Inception",
        genre: "Action",
        year: 2010,
        director: "Christopher Nolan",
      },
    ];
    setupIndex("movies", movies);
  };

  const handleLoadProducts = () => {
    const products: Document[] = [
      {
        id: "p1",
        name: "Apple MacBook Pro 14",
        category: "Laptops",
        price: "$1999",
      },
      {
        id: "p2",
        name: "Apple iPhone 15 Pro",
        category: "Smartphones",
        price: "$999",
      },
      {
        id: "p3",
        name: "Samsung Galaxy S24 Ultra",
        category: "Smartphones",
        price: "$1199",
      },
      {
        id: "p4",
        name: "Sony WH-1000XM5 Headphones",
        category: "Audio",
        price: "$398",
      },
      {
        id: "p5",
        name: "Dell XPS 13 Plus",
        category: "Laptops",
        price: "$1299",
      },
    ];
    setupIndex("products", products);
  };

  const handleClear = async () => {
    if (!activeIndex) return;
    setStatus({ type: "loading", msg: t("demo.setup.status.clearing") });
    try {
      await fetch(`${API_URL}/indexes/${activeIndex}`, { method: "DELETE" });
      setStatus({ type: "idle", msg: t("demo.setup.status.cleared") });
      onClear();
    } catch (e: unknown) {
      console.error(e);
      setStatus({ type: "error", msg: t("demo.setup.status.error_clear") });
    }
  };

  return (
    <div className="glass-effect rounded-2xl p-6 md:p-8 mb-8 text-center max-w-4xl mx-auto shadow-2xl shadow-blue-900/10">
      <h3 className="text-xl font-bold mb-2">{t("demo.setup.title")}</h3>
      <p className="text-slate-400 mb-6 text-sm">{t("demo.setup.subtitle")}</p>

      <div className="flex flex-wrap justify-center gap-4 mb-6">
        <button
          onClick={handleLoadMovies}
          disabled={status.type === "loading"}
          aria-label={t("demo.setup.load_movies")}
          className={`btn ${activeIndex === "movies" ? "btn-primary" : "btn-secondary"}`}
        >
          <Film className="w-4 h-4" /> {t("demo.setup.load_movies")}
        </button>
        <button
          onClick={handleLoadProducts}
          disabled={status.type === "loading"}
          aria-label={t("demo.setup.load_products")}
          className={`btn ${activeIndex === "products" ? "btn-primary" : "btn-secondary"}`}
        >
          <Package className="w-4 h-4" /> {t("demo.setup.load_products")}
        </button>

        <button
          onClick={handleClear}
          disabled={!activeIndex || status.type === "loading"}
          aria-label={t("demo.setup.clear")}
          className="btn btn-danger ml-0 md:ml-4"
        >
          <Trash2 className="w-4 h-4" /> {t("demo.setup.clear")}
        </button>
      </div>

      <div className="min-h-[24px] text-sm flex items-center justify-center">
        {status.type === "loading" && (
          <span className="text-blue-400 animate-pulse">{status.msg}</span>
        )}
        {status.type === "success" && (
          <span className="text-emerald-400 flex items-center gap-2">
            <CheckCircle2 className="w-4 h-4" /> {status.msg}
          </span>
        )}
        {status.type === "error" && (
          <span className="text-red-400 flex items-center gap-2">
            <AlertCircle className="w-4 h-4" /> {status.msg}
          </span>
        )}
        {status.type === "idle" && status.msg && (
          <span className="text-slate-400">{status.msg}</span>
        )}
      </div>
    </div>
  );
}
