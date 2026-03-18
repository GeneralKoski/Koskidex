import React, { useState, useCallback, useEffect, useRef } from 'react';
import { Search as SearchIcon, Ghost, Video, Calendar, Tag, DollarSign, Ticket, AlertCircle, Star, ShoppingBag, ArrowUpDown, Zap, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { SearchResponse, Hit, MovieDocument, ProductDocument } from '../types';

const API_URL = import.meta.env.VITE_API_URL || "/api";

interface SearchUIProps {
  activeIndex: string;
}

const SORT_OPTIONS_MOVIES = [
  { value: "", label: "sort_relevance" },
  { value: "year:desc", label: "sort_year_desc" },
  { value: "year:asc", label: "sort_year_asc" },
  { value: "rating:desc", label: "sort_rating_desc" },
];

const SORT_OPTIONS_PRODUCTS = [
  { value: "", label: "sort_relevance" },
  { value: "price:asc", label: "sort_price_asc" },
  { value: "price:desc", label: "sort_price_desc" },
  { value: "rating:desc", label: "sort_rating_desc" },
];

const FUZZINESS_OPTIONS = ["AUTO", "0", "1", "2"];

function getFacetField(activeIndex: string): string {
  return activeIndex === "movies" ? "genre" : "category";
}

export default function SearchUI({ activeIndex }: SearchUIProps) {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResponse | null>(null);
  const [isSearching, setIsSearching] = useState(false);
  const [error, setError] = useState<"connection_error" | null>(null);
  const [lastProcessingTime, setLastProcessingTime] = useState<number | null>(null);
  const [sortValue, setSortValue] = useState("");
  const [fuzziness, setFuzziness] = useState("AUTO");
  const [activeFacet, setActiveFacet] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        inputRef.current?.focus();
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, []);

  useEffect(() => {
    setSortValue("");
    setFuzziness("AUTO");
    setActiveFacet(null);
  }, [activeIndex]);

  const performSearch = useCallback(
    async (q: string) => {
      if (!activeIndex) return;
      setIsSearching(true);

      try {
        const params = new URLSearchParams();
        if (q) params.set("q", q);
        params.set("fuzziness", fuzziness);
        params.set("facets", getFacetField(activeIndex));
        if (sortValue) params.set("sort", sortValue);
        if (activeFacet) params.set("filter", `${getFacetField(activeIndex)}=${activeFacet}`);

        const res = await fetch(
          `${API_URL}/indexes/${activeIndex}/search?${params.toString()}`,
        );
        if (!res.ok) {
          throw new Error(`Search failed: ${res.status}`);
        }
        const data: SearchResponse = await res.json();
        setResults(data);
        setLastProcessingTime(data.processing_time_ms);
        setError(null);
      } catch (error) {
        console.error("Search failed", error);
        setError("connection_error");
        setResults(null);
      } finally {
        setIsSearching(false);
      }
    },
    [activeIndex, fuzziness, sortValue, activeFacet],
  );

  useEffect(() => {
    const timer = setTimeout(() => {
      performSearch(query);
    }, 150);
    return () => clearTimeout(timer);
  }, [query, performSearch]);

  useEffect(() => {
    if (!activeIndex) {
      setQuery("");
      setResults(null);
      setLastProcessingTime(null);
    } else {
      performSearch(query);
    }
  }, [activeIndex, performSearch, query]);

  const renderHighlightLine = (html: string) => (
    <span dangerouslySetInnerHTML={{ __html: html }} />
  );

  const displayTime = lastProcessingTime !== null
    ? (lastProcessingTime === 0 ? t('search_status.instant') : lastProcessingTime)
    : null;

  const sortOptions = activeIndex === "movies" ? SORT_OPTIONS_MOVIES : SORT_OPTIONS_PRODUCTS;

  const facetData = results?.facets?.[getFacetField(activeIndex)] || {};
  const facetEntries = Object.entries(facetData).sort((a, b) => b[1] - a[1]);

  return (
    <div className="flex flex-col gap-4">
      {/* Search input */}
      <div className="relative group">
        <SearchIcon className="absolute left-5 top-1/2 -translate-y-1/2 text-slate-500 group-focus-within:text-blue-400 transition-colors w-5 h-5" />
        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => setQuery(e.target.value)}
          placeholder={activeIndex ? t('demo.search.placeholder') : t('demo.search.load_dataset')}
          disabled={!activeIndex}
          aria-label={t('demo.search.placeholder')}
          className="w-full bg-slate-950/50 border border-slate-700/40 rounded-xl py-3.5 pl-12 pr-28 text-[15px] text-white placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500/40 transition-all disabled:opacity-40 disabled:cursor-not-allowed font-medium"
        />
        <div className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-2">
          <kbd className="hidden md:inline-flex items-center gap-0.5 px-1.5 py-0.5 bg-slate-800/50 border border-slate-700/40 rounded text-[10px] text-slate-500 font-mono">
            <span className="text-[11px]">⌘</span>K
          </kbd>
          {displayTime !== null && (
            <div className="bg-blue-500/10 text-blue-400 px-2.5 py-1 rounded-md text-[11px] font-bold tracking-wider flex items-center gap-1.5">
              {isSearching && (
                <span className="animate-spin w-2.5 h-2.5 border-[1.5px] border-blue-400 border-t-transparent rounded-full" />
              )}
              {displayTime}{t('demo.search.ms')}
            </div>
          )}
        </div>
      </div>

      {/* Controls: Sort + Fuzziness */}
      {activeIndex && (
        <div className="flex flex-wrap items-center gap-2">
          <div className="flex items-center gap-1.5">
            <ArrowUpDown className="w-3.5 h-3.5 text-slate-500" />
            <select
              value={sortValue}
              onChange={(e) => setSortValue(e.target.value)}
              className="bg-slate-900/60 border border-slate-700/40 rounded-lg px-2.5 py-1.5 text-xs text-slate-300 focus:outline-none focus:border-blue-500/40 cursor-pointer"
            >
              {sortOptions.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {t(`demo.search.${opt.label}`)}
                </option>
              ))}
            </select>
          </div>

          <div className="flex items-center gap-1.5">
            <Zap className="w-3.5 h-3.5 text-slate-500" />
            <div className="flex rounded-lg border border-slate-700/40 overflow-hidden">
              {FUZZINESS_OPTIONS.map((opt) => (
                <button
                  key={opt}
                  onClick={() => setFuzziness(opt)}
                  className={`px-2.5 py-1.5 text-xs font-medium transition-all ${
                    fuzziness === opt
                      ? "bg-blue-500/15 text-blue-400"
                      : "bg-slate-900/60 text-slate-500 hover:text-slate-300"
                  }`}
                >
                  {opt}
                </button>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Facets */}
      {activeIndex && facetEntries.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {activeFacet && (
            <button
              onClick={() => setActiveFacet(null)}
              className="flex items-center gap-1 px-2.5 py-1 rounded-lg text-[11px] font-semibold bg-red-500/10 text-red-400 border border-red-500/20 hover:bg-red-500/20 transition-all"
            >
              <X className="w-3 h-3" />
              {activeFacet}
            </button>
          )}
          {!activeFacet && facetEntries.map(([value, count]) => (
            <button
              key={value}
              onClick={() => setActiveFacet(value)}
              className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-[11px] font-medium bg-slate-900/40 text-slate-400 border border-slate-800/40 hover:bg-slate-800/40 hover:text-slate-200 hover:border-blue-500/20 transition-all"
            >
              {value}
              <span className="text-[10px] text-slate-600 bg-slate-800/60 px-1.5 py-0.5 rounded-full">{count}</span>
            </button>
          ))}
        </div>
      )}

      {/* Results area */}
      <div className="min-h-[240px]">
        {!activeIndex ? (
          <div className="flex flex-col items-center justify-center h-[240px] text-slate-600 text-center">
            <Ghost className="w-12 h-12 mb-3 opacity-30" />
            <p className="text-sm">{t('demo.search.load_dataset')}</p>
          </div>
        ) : isSearching && !results ? (
          <div className="flex flex-col gap-2.5">
            {[1, 2, 3].map(i => (
              <div key={i} className="bg-slate-900/30 border border-slate-800/40 p-4 rounded-xl animate-pulse">
                <div className="h-5 bg-slate-800/60 rounded w-3/4 mb-2.5"></div>
                <div className="flex gap-3">
                  <div className="h-3.5 bg-slate-800/40 rounded w-1/4"></div>
                  <div className="h-3.5 bg-slate-800/40 rounded w-1/4"></div>
                </div>
              </div>
            ))}
          </div>
        ) : error === "connection_error" ? (
          <div className="flex flex-col items-center justify-center h-[240px] text-red-400 text-center">
            <AlertCircle className="w-12 h-12 mb-3 opacity-40" />
            <p className="text-sm font-semibold">{t('search_status.offline_title')}</p>
            <p className="text-slate-500 max-w-xs mt-1.5 text-xs">{t('search_status.offline_desc', { url: API_URL })}</p>
          </div>
        ) : results?.hits?.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-[240px] text-slate-500 text-center">
            <SearchIcon className="w-12 h-12 mb-3 opacity-20" />
            <p className="text-sm break-words max-w-full">
              {t("demo.search.no_results", { query })}
            </p>
          </div>
        ) : (
          <ul className="flex flex-col gap-2 max-h-[260px] overflow-y-auto pr-2 custom-scrollbar" aria-label="Search results">
            {results?.hits?.map((hit: Hit, index: number) => {
              let titleLine = '';
              let metaItems: { icon: React.ReactNode; text: string }[] = [];

              if (activeIndex === "movies") {
                const doc = hit.document as MovieDocument;
                titleLine = hit.highlights.title || doc.title;
                metaItems = [
                  { icon: <Video className="w-3.5 h-3.5 shrink-0" />, text: hit.highlights.director || doc.director },
                  { icon: <Calendar className="w-3.5 h-3.5 shrink-0" />, text: hit.highlights.year || String(doc.year) },
                  { icon: <Ticket className="w-3.5 h-3.5 shrink-0" />, text: hit.highlights.genre || doc.genre },
                  { icon: <Star className="w-3.5 h-3.5 shrink-0" />, text: String(doc.rating) },
                ];
              } else if (activeIndex === "products") {
                const doc = hit.document as ProductDocument;
                titleLine = hit.highlights.name || doc.name;
                metaItems = [
                  { icon: <ShoppingBag className="w-3.5 h-3.5 shrink-0" />, text: doc.brand },
                  { icon: <Tag className="w-3.5 h-3.5 shrink-0" />, text: hit.highlights.category || doc.category },
                  { icon: <DollarSign className="w-3.5 h-3.5 shrink-0" />, text: `$${doc.price}` },
                  { icon: <Star className="w-3.5 h-3.5 shrink-0" />, text: String(doc.rating) },
                ];
              }

              return (
                <li
                  key={hit.id}
                  style={{ animationDelay: `${index * 40}ms` }}
                  className="shrink-0 bg-slate-900/30 border border-slate-800/40 px-4 py-3 rounded-xl hover:bg-slate-800/40 hover:border-blue-500/20 transition-all cursor-default flex flex-col gap-1 relative overflow-hidden group animate-in slide-in-from-bottom-2 fade-in duration-300 fill-mode-both"
                >
                  <div className="absolute left-0 top-0 bottom-0 w-0.5 bg-gradient-to-b from-blue-500 to-indigo-500 opacity-0 group-hover:opacity-100 transition-opacity"></div>

                  <div className="text-[15px] font-semibold text-slate-100 group-hover:text-blue-400 transition-colors break-words">
                    {renderHighlightLine(titleLine)}
                  </div>

                  <div className="flex flex-wrap gap-2 text-[11px] text-slate-400">
                    {metaItems.map((item, idx) => (
                      <span
                        key={idx}
                        className="flex items-center gap-1 bg-slate-900/40 px-2 py-0.5 rounded border border-slate-800/40"
                      >
                        {item.icon}
                        {renderHighlightLine(item.text)}
                      </span>
                    ))}
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </div>
  );
}
