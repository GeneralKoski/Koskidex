import React, { useState, useCallback, useEffect, useRef } from 'react';
import { Search as SearchIcon, Ghost, Video, Calendar, Tag, DollarSign, Ticket, AlertCircle, Star, ShoppingBag, ArrowUpDown, X } from 'lucide-react';
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

  const controlLabel = "text-xs text-muted";

  return (
    <div className="flex flex-col gap-5">
      <div className="relative">
        <SearchIcon className="absolute left-4 top-1/2 -translate-y-1/2 text-muted w-5 h-5 pointer-events-none" />
        <input
          ref={inputRef}
          type="search"
          value={query}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => setQuery(e.target.value)}
          placeholder={activeIndex ? t('demo.search.placeholder') : t('demo.search.load_dataset')}
          disabled={!activeIndex}
          aria-label={t('demo.search.placeholder')}
          className="w-full bg-paper border-2 border-ink rounded-lg py-3.5 pl-12 pr-28 text-base placeholder:text-muted focus:shadow-[4px_4px_0_var(--color-mark)] transition-shadow disabled:border-rule disabled:cursor-not-allowed"
        />
        <div className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-2 pointer-events-none">
          <kbd className="hidden md:inline-flex items-center px-1.5 py-0.5 border border-rule rounded text-[11px] text-muted font-mono">
            ⌘K
          </kbd>
          {displayTime !== null && (
            <span className="font-mono text-xs tabular-nums text-muted flex items-center gap-1.5">
              {isSearching && (
                <span className="animate-spin w-2.5 h-2.5 border-[1.5px] border-current border-t-transparent rounded-full" />
              )}
              {displayTime} {t('demo.search.ms')}
            </span>
          )}
        </div>
      </div>

      {activeIndex && (
        <div className="flex flex-wrap items-end gap-x-6 gap-y-3">
          <label className="flex flex-col gap-1.5">
            <span className={controlLabel}>{t('demo.search.sort')}</span>
            <span className="relative flex items-center">
              <ArrowUpDown className="absolute left-2.5 w-3.5 h-3.5 text-muted pointer-events-none" />
              <select
                value={sortValue}
                onChange={(e) => setSortValue(e.target.value)}
                className="appearance-none bg-paper border border-rule hover:border-ink rounded-md pl-8 pr-3 py-1.5 text-sm"
              >
                {sortOptions.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {t(`demo.search.${opt.label}`)}
                  </option>
                ))}
              </select>
            </span>
          </label>

          <div className="flex flex-col gap-1.5">
            <span id="fuzziness-label" className={controlLabel}>{t('demo.search.fuzziness')}</span>
            <div role="group" aria-labelledby="fuzziness-label" className="flex rounded-md border border-rule p-0.5">
              {FUZZINESS_OPTIONS.map((opt) => (
                <button
                  key={opt}
                  type="button"
                  onClick={() => setFuzziness(opt)}
                  aria-pressed={fuzziness === opt}
                  className={`px-2.5 py-1 rounded-[4px] font-mono text-xs transition-colors ${
                    fuzziness === opt ? "bg-ink text-paper" : "text-muted hover:text-ink"
                  }`}
                >
                  {opt}
                </button>
              ))}
            </div>
          </div>

          {results && !error && (
            <span className="ml-auto font-mono text-xs tabular-nums text-muted">
              {t('demo.search.hits', { count: results.total_hits })}
            </span>
          )}
        </div>
      )}

      {activeIndex && facetEntries.length > 0 && (
        <div className="flex flex-wrap items-center gap-1.5" aria-label={t('demo.search.facets')} role="group">
          {activeFacet ? (
            <button
              type="button"
              onClick={() => setActiveFacet(null)}
              aria-label={t('demo.search.remove_filter', { value: activeFacet })}
              className="flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-mark text-mark-ink"
            >
              {activeFacet}
              <X className="w-3 h-3" />
            </button>
          ) : (
            facetEntries.map(([value, count]) => (
              <button
                key={value}
                type="button"
                onClick={() => setActiveFacet(value)}
                className="flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs border border-rule bg-paper hover:border-ink transition-colors"
              >
                {value}
                <span className="font-mono tabular-nums text-muted">{count}</span>
              </button>
            ))
          )}
        </div>
      )}

      <div className="min-h-[260px]">
        {!activeIndex ? (
          <div className="h-[260px] grid place-items-center text-center border border-dashed border-rule rounded-lg px-6">
            <div>
              <Ghost className="w-8 h-8 mx-auto mb-3 text-muted" />
              <p className="text-sm text-muted">{t('demo.search.load_dataset')}</p>
            </div>
          </div>
        ) : isSearching && !results ? (
          <div className="flex flex-col">
            {[1, 2, 3].map(i => (
              <div key={i} className="py-4 border-b border-rule animate-pulse">
                <div className="h-4 bg-rule rounded w-2/3 mb-2.5"></div>
                <div className="h-3 bg-rule/70 rounded w-1/3"></div>
              </div>
            ))}
          </div>
        ) : error === "connection_error" ? (
          <div className="h-[260px] grid place-items-center text-center border border-dashed border-bad/50 rounded-lg px-6">
            <div>
              <AlertCircle className="w-8 h-8 mx-auto mb-3 text-bad" />
              <p className="text-sm font-semibold text-bad">{t('search_status.offline_title')}</p>
              <p className="text-muted max-w-xs mt-1.5 text-sm">{t('search_status.offline_desc', { url: API_URL })}</p>
            </div>
          </div>
        ) : results?.hits?.length === 0 ? (
          <div className="h-[260px] grid place-items-center text-center border border-dashed border-rule rounded-lg px-6">
            <div>
              <SearchIcon className="w-8 h-8 mx-auto mb-3 text-muted" />
              <p className="text-sm text-muted break-words max-w-full">
                {t("demo.search.no_results", { query })}
              </p>
            </div>
          </div>
        ) : (
          <ul className="max-h-[420px] overflow-y-auto border-t border-rule" aria-label={t('demo.search.results')}>
            {results?.hits?.map((hit: Hit) => {
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
                <li key={hit.id} className="hl py-3.5 border-b border-rule flex flex-col gap-1.5">
                  <div className="font-semibold leading-snug break-words">
                    {renderHighlightLine(titleLine)}
                  </div>
                  <div className="flex flex-wrap gap-x-4 gap-y-1 text-[13px] text-muted">
                    {metaItems.map((item, idx) => (
                      <span key={idx} className="flex items-center gap-1.5">
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
