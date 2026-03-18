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

type RawMovie = { id: string; title: string; genreKey: string; year: number; director: string; rating: number };
type RawProduct = { id: string; name: string; categoryKey: string; brand: string; price: number; rating: number };

const MOVIES_RAW: RawMovie[] = [
  { id: "m1", title: "The Matrix", genreKey: "sci_fi", year: 1999, director: "Lana & Lilly Wachowski", rating: 8.7 },
  { id: "m2", title: "The Godfather", genreKey: "crime", year: 1972, director: "Francis Ford Coppola", rating: 9.2 },
  { id: "m3", title: "Goodfellas", genreKey: "crime", year: 1990, director: "Martin Scorsese", rating: 8.7 },
  { id: "m4", title: "Pulp Fiction", genreKey: "crime", year: 1994, director: "Quentin Tarantino", rating: 8.9 },
  { id: "m5", title: "Interstellar", genreKey: "sci_fi", year: 2014, director: "Christopher Nolan", rating: 8.7 },
  { id: "m6", title: "Inception", genreKey: "action", year: 2010, director: "Christopher Nolan", rating: 8.8 },
  { id: "m7", title: "The Shawshank Redemption", genreKey: "drama", year: 1994, director: "Frank Darabont", rating: 9.3 },
  { id: "m8", title: "The Dark Knight", genreKey: "action", year: 2008, director: "Christopher Nolan", rating: 9.0 },
  { id: "m9", title: "Fight Club", genreKey: "drama", year: 1999, director: "David Fincher", rating: 8.8 },
  { id: "m10", title: "Forrest Gump", genreKey: "drama", year: 1994, director: "Robert Zemeckis", rating: 8.8 },
  { id: "m11", title: "The Lord of the Rings: The Fellowship of the Ring", genreKey: "fantasy", year: 2001, director: "Peter Jackson", rating: 8.8 },
  { id: "m12", title: "Star Wars: A New Hope", genreKey: "sci_fi", year: 1977, director: "George Lucas", rating: 8.6 },
  { id: "m13", title: "The Silence of the Lambs", genreKey: "thriller", year: 1991, director: "Jonathan Demme", rating: 8.6 },
  { id: "m14", title: "Schindler's List", genreKey: "drama", year: 1993, director: "Steven Spielberg", rating: 9.0 },
  { id: "m15", title: "Gladiator", genreKey: "action", year: 2000, director: "Ridley Scott", rating: 8.5 },
  { id: "m16", title: "The Departed", genreKey: "crime", year: 2006, director: "Martin Scorsese", rating: 8.5 },
  { id: "m17", title: "Saving Private Ryan", genreKey: "war", year: 1998, director: "Steven Spielberg", rating: 8.6 },
  { id: "m18", title: "Jurassic Park", genreKey: "adventure", year: 1993, director: "Steven Spielberg", rating: 8.2 },
  { id: "m19", title: "Alien", genreKey: "horror", year: 1979, director: "Ridley Scott", rating: 8.5 },
  { id: "m20", title: "Blade Runner", genreKey: "sci_fi", year: 1982, director: "Ridley Scott", rating: 8.1 },
  { id: "m21", title: "The Truman Show", genreKey: "comedy", year: 1998, director: "Peter Weir", rating: 8.2 },
  { id: "m22", title: "Toy Story", genreKey: "animation", year: 1995, director: "John Lasseter", rating: 8.3 },
  { id: "m23", title: "WALL-E", genreKey: "animation", year: 2008, director: "Andrew Stanton", rating: 8.4 },
  { id: "m24", title: "Spirited Away", genreKey: "animation", year: 2001, director: "Hayao Miyazaki", rating: 8.6 },
  { id: "m25", title: "Parasite", genreKey: "thriller", year: 2019, director: "Bong Joon-ho", rating: 8.5 },
  { id: "m26", title: "The Grand Budapest Hotel", genreKey: "comedy", year: 2014, director: "Wes Anderson", rating: 8.1 },
  { id: "m27", title: "Mad Max: Fury Road", genreKey: "action", year: 2015, director: "George Miller", rating: 8.1 },
  { id: "m28", title: "Whiplash", genreKey: "drama", year: 2014, director: "Damien Chazelle", rating: 8.5 },
  { id: "m29", title: "Get Out", genreKey: "horror", year: 2017, director: "Jordan Peele", rating: 7.7 },
  { id: "m30", title: "The Social Network", genreKey: "drama", year: 2010, director: "David Fincher", rating: 7.8 },
  { id: "m31", title: "Django Unchained", genreKey: "western", year: 2012, director: "Quentin Tarantino", rating: 8.4 },
  { id: "m32", title: "No Country for Old Men", genreKey: "thriller", year: 2007, director: "Joel & Ethan Coen", rating: 8.2 },
  { id: "m33", title: "The Prestige", genreKey: "mystery", year: 2006, director: "Christopher Nolan", rating: 8.5 },
  { id: "m34", title: "Eternal Sunshine of the Spotless Mind", genreKey: "romance", year: 2004, director: "Michel Gondry", rating: 8.3 },
  { id: "m35", title: "La La Land", genreKey: "musical", year: 2016, director: "Damien Chazelle", rating: 8.0 },
  { id: "m36", title: "2001: A Space Odyssey", genreKey: "sci_fi", year: 1968, director: "Stanley Kubrick", rating: 8.3 },
  { id: "m37", title: "Titanic", genreKey: "romance", year: 1997, director: "James Cameron", rating: 7.9 },
  { id: "m38", title: "The Shining", genreKey: "horror", year: 1980, director: "Stanley Kubrick", rating: 8.4 },
  { id: "m39", title: "Back to the Future", genreKey: "adventure", year: 1985, director: "Robert Zemeckis", rating: 8.5 },
  { id: "m40", title: "Terminator 2: Judgment Day", genreKey: "action", year: 1991, director: "James Cameron", rating: 8.6 },
  { id: "m41", title: "The Lion King", genreKey: "animation", year: 1994, director: "Roger Allers & Rob Minkoff", rating: 8.5 },
  { id: "m42", title: "Braveheart", genreKey: "war", year: 1995, director: "Mel Gibson", rating: 8.4 },
  { id: "m43", title: "The Usual Suspects", genreKey: "mystery", year: 1995, director: "Bryan Singer", rating: 8.5 },
  { id: "m44", title: "Reservoir Dogs", genreKey: "crime", year: 1992, director: "Quentin Tarantino", rating: 8.3 },
  { id: "m45", title: "Jaws", genreKey: "thriller", year: 1975, director: "Steven Spielberg", rating: 8.0 },
  { id: "m46", title: "Oppenheimer", genreKey: "drama", year: 2023, director: "Christopher Nolan", rating: 8.3 },
  { id: "m47", title: "Everything Everywhere All at Once", genreKey: "comedy", year: 2022, director: "Daniel Kwan & Daniel Scheinert", rating: 7.8 },
  { id: "m48", title: "Dune: Part Two", genreKey: "sci_fi", year: 2024, director: "Denis Villeneuve", rating: 8.5 },
  { id: "m49", title: "The Batman", genreKey: "action", year: 2022, director: "Matt Reeves", rating: 7.8 },
  { id: "m50", title: "Spider-Man: Across the Spider-Verse", genreKey: "animation", year: 2023, director: "Joaquim Dos Santos", rating: 8.6 },
];

const PRODUCTS_RAW: RawProduct[] = [
  { id: "p1", name: "MacBook Pro 16 M3 Max", categoryKey: "laptops", brand: "Apple", price: 2499, rating: 4.7 },
  { id: "p2", name: "MacBook Air 15 M3", categoryKey: "laptops", brand: "Apple", price: 1299, rating: 4.8 },
  { id: "p3", name: "Dell XPS 15", categoryKey: "laptops", brand: "Dell", price: 1799, rating: 4.5 },
  { id: "p4", name: "ThinkPad X1 Carbon Gen 11", categoryKey: "laptops", brand: "Lenovo", price: 1649, rating: 4.6 },
  { id: "p5", name: "ROG Strix G16 Gaming Laptop", categoryKey: "gaming", brand: "ASUS", price: 1599, rating: 4.4 },
  { id: "p6", name: "iPhone 15 Pro Max", categoryKey: "smartphones", brand: "Apple", price: 1199, rating: 4.7 },
  { id: "p7", name: "Samsung Galaxy S24 Ultra", categoryKey: "smartphones", brand: "Samsung", price: 1299, rating: 4.6 },
  { id: "p8", name: "Google Pixel 8 Pro", categoryKey: "smartphones", brand: "Google", price: 999, rating: 4.5 },
  { id: "p9", name: "OnePlus 12", categoryKey: "smartphones", brand: "OnePlus", price: 799, rating: 4.3 },
  { id: "p10", name: "Sony WH-1000XM5 Headphones", categoryKey: "audio", brand: "Sony", price: 348, rating: 4.7 },
  { id: "p11", name: "AirPods Pro 2nd Generation", categoryKey: "audio", brand: "Apple", price: 249, rating: 4.8 },
  { id: "p12", name: "Bose QuietComfort Ultra", categoryKey: "audio", brand: "Bose", price: 429, rating: 4.6 },
  { id: "p13", name: "Sennheiser Momentum 4 Wireless", categoryKey: "audio", brand: "Sennheiser", price: 349, rating: 4.5 },
  { id: "p14", name: "iPad Pro 12.9 M2", categoryKey: "tablets", brand: "Apple", price: 1099, rating: 4.7 },
  { id: "p15", name: "Samsung Galaxy Tab S9 Ultra", categoryKey: "tablets", brand: "Samsung", price: 1199, rating: 4.5 },
  { id: "p16", name: "Apple Watch Ultra 2", categoryKey: "wearables", brand: "Apple", price: 799, rating: 4.6 },
  { id: "p17", name: "Samsung Galaxy Watch 6 Classic", categoryKey: "wearables", brand: "Samsung", price: 329, rating: 4.3 },
  { id: "p18", name: "Sony Alpha A7 IV Mirrorless Camera", categoryKey: "cameras", brand: "Sony", price: 2498, rating: 4.8 },
  { id: "p19", name: "Canon EOS R6 Mark II", categoryKey: "cameras", brand: "Canon", price: 2499, rating: 4.7 },
  { id: "p20", name: "GoPro Hero 12 Black", categoryKey: "cameras", brand: "GoPro", price: 399, rating: 4.4 },
  { id: "p21", name: "Nintendo Switch OLED", categoryKey: "gaming", brand: "Nintendo", price: 349, rating: 4.7 },
  { id: "p22", name: "PlayStation 5 Console", categoryKey: "gaming", brand: "Sony", price: 499, rating: 4.8 },
  { id: "p23", name: "Xbox Series X", categoryKey: "gaming", brand: "Microsoft", price: 499, rating: 4.6 },
  { id: "p24", name: "LG UltraGear 27GP950 4K Monitor", categoryKey: "monitors", brand: "LG", price: 799, rating: 4.5 },
  { id: "p25", name: "Samsung Odyssey G9 49 Ultrawide", categoryKey: "monitors", brand: "Samsung", price: 1299, rating: 4.4 },
  { id: "p26", name: "Apple Studio Display", categoryKey: "monitors", brand: "Apple", price: 1599, rating: 4.3 },
  { id: "p27", name: "Razer BlackWidow V4 Pro Keyboard", categoryKey: "accessories", brand: "Razer", price: 229, rating: 4.5 },
  { id: "p28", name: "Logitech MX Master 3S Mouse", categoryKey: "accessories", brand: "Logitech", price: 99, rating: 4.8 },
  { id: "p29", name: "Samsung T7 Shield 2TB SSD", categoryKey: "storage", brand: "Samsung", price: 159, rating: 4.6 },
  { id: "p30", name: "WD Black SN850X 2TB NVMe", categoryKey: "storage", brand: "Western Digital", price: 179, rating: 4.7 },
];

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
    const movies: Document[] = MOVIES_RAW.map((m) => ({
      id: m.id,
      title: m.title,
      genre: t(`datasets.movies.genres.${m.genreKey}`),
      year: m.year,
      director: m.director,
      rating: m.rating,
    }));
    setupIndex("movies", movies);
  };

  const handleLoadProducts = () => {
    const products: Document[] = PRODUCTS_RAW.map((p) => ({
      id: p.id,
      name: p.name,
      category: t(`datasets.products.categories.${p.categoryKey}`),
      brand: p.brand,
      price: p.price,
      rating: p.rating,
    }));
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
