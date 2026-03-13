import React, { useState, useCallback, useEffect } from 'react';
import { Search as SearchIcon, Ghost, Video, Calendar, Tag, DollarSign, Ticket, AlertCircle } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { SearchResponse, Hit, MovieDocument, ProductDocument } from '../types';

const API_URL = import.meta.env.VITE_API_URL || "http://localhost:7700";

interface SearchUIProps {
  activeIndex: string;
}

export default function SearchUI({ activeIndex }: SearchUIProps) {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResponse | null>(null);
  const [isSearching, setIsSearching] = useState(false);
  const [error, setError] = useState<"connection_error" | null>(null);
  const [lastProcessingTime, setLastProcessingTime] = useState<number | null>(null);

  const performSearch = useCallback(
    async (q: string) => {
      if (!activeIndex) return;
      setIsSearching(true);

      try {
        const res = await fetch(
          `${API_URL}/indexes/${activeIndex}/search?q=${encodeURIComponent(q)}`,
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
    [activeIndex],
  );

  // Debounce hook
  useEffect(() => {
    const timer = setTimeout(() => {
      performSearch(query);
    }, 150);

    return () => clearTimeout(timer);
  }, [query, performSearch]);

  // When active index changes, clear or pre-search empty
  useEffect(() => {
    if (!activeIndex) {
      setQuery("");
      setResults(null);
      setLastProcessingTime(null);
    } else {
      performSearch(query); // Refetch current query on new index
    }
  }, [activeIndex, performSearch, query]);

  // Custom renderer for Highlights from Golang backend which injects <em>
  const renderHighlightLine = (html: string) => (
    <span dangerouslySetInnerHTML={{ __html: html }} />
  );

  const displayTime = lastProcessingTime !== null 
    ? (lastProcessingTime === 0 ? t('search_status.instant') : lastProcessingTime) 
    : null;

  return (
    <div className="glass-effect rounded-2xl p-6 md:p-8 w-full shadow-xl shadow-blue-500/10 mb-8 relative">
      <div className="relative mb-6 group">
        <SearchIcon className="absolute left-6 top-1/2 -translate-y-1/2 text-slate-400 group-focus-within:text-blue-400 transition-colors w-6 h-6" />
        <input
          type="text"
          value={query}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => setQuery(e.target.value)}
          placeholder={t('demo.search.placeholder')}
          disabled={!activeIndex}
          aria-label={t('demo.search.placeholder')}
          className="w-full bg-slate-900/60 border border-slate-700/50 rounded-xl py-4 pl-14 pr-24 text-base text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400/50 transition-all disabled:opacity-50 disabled:cursor-not-allowed font-medium shadow-inner"
        />

        {displayTime !== null && (
          <div className="absolute right-4 top-1/2 -translate-y-1/2 bg-blue-500/10 text-blue-400 px-3 py-1.5 rounded-md text-xs font-bold tracking-wider flex items-center gap-2 animate-in fade-in duration-300">
            {isSearching && (
              <span className="animate-spin w-3 h-3 border-2 border-blue-400 border-t-transparent rounded-full" />
            )}
            {displayTime}{t('demo.search.ms')}
          </div>
        )}
      </div>

      <div className="min-h-[200px]">
        {!activeIndex ? (
          <div className="flex flex-col items-center justify-center h-[200px] text-slate-500 text-center animate-in fade-in duration-700">
            <Ghost className="w-16 h-16 mb-4 opacity-20" />
            <p className="text-lg">{t('demo.search.load_dataset')}</p>
          </div>
        ) : isSearching && !results ? (
          <div className="flex flex-col gap-3">
            {[1, 2, 3].map(i => (
              <div key={i} className="bg-slate-900/40 border border-slate-800/60 p-4 rounded-xl animate-pulse">
                <div className="h-6 bg-slate-800 rounded w-3/4 mb-3"></div>
                <div className="flex gap-4">
                  <div className="h-4 bg-slate-800 rounded w-1/4"></div>
                  <div className="h-4 bg-slate-800 rounded w-1/4"></div>
                </div>
              </div>
            ))}
          </div>
        ) : error === "connection_error" ? (
          <div className="flex flex-col items-center justify-center h-[200px] text-red-400 text-center animate-in zoom-in-95 duration-500">
            <AlertCircle className="w-16 h-16 mb-4 opacity-50" />
            <p className="text-lg font-bold">{t('search_status.offline_title')}</p>
            <p className="text-slate-500 max-w-xs mt-2">{t('search_status.offline_desc', { url: API_URL })}</p>
          </div>
        ) : results?.hits?.length === 0 ? (
          <div className="flex flex-col items-center justify-center min-h-[200px] text-slate-500 text-center animate-in zoom-in-95 duration-500 p-4">
            <SearchIcon className="w-16 h-16 mb-4 opacity-20" />
            <p className="text-lg break-words max-w-full">
              {t("demo.search.no_results", { query })}
            </p>
          </div>
        ) : (
          <ul className="flex flex-col gap-2.5" aria-label="Search results">
            {results?.hits?.map((hit: Hit, index: number) => {
              let titleLine = '';
              let metaItems: { icon: React.ReactNode; text: string }[] = [];

              if (activeIndex === "movies") {
                const doc = hit.document as MovieDocument;
                titleLine = hit.highlights.title || doc.title;
                metaItems = [
                  {
                    icon: <Video className="w-4 h-4 shrink-0" />,
                    text: hit.highlights.director || doc.director,
                  },
                  {
                    icon: <Calendar className="w-4 h-4 shrink-0" />,
                    text: hit.highlights.year || String(doc.year),
                  },
                  {
                    icon: <Ticket className="w-4 h-4 shrink-0" />,
                    text: hit.highlights.genre || doc.genre,
                  },
                ];
              } else if (activeIndex === "products") {
                const doc = hit.document as ProductDocument;
                titleLine = hit.highlights.name || doc.name;
                metaItems = [
                  {
                    icon: <Tag className="w-4 h-4 shrink-0" />,
                    text: hit.highlights.category || doc.category,
                  },
                  {
                    icon: <DollarSign className="w-4 h-4 shrink-0" />,
                    text: doc.price,
                  },
                ];
              }

              return (
                <li
                  key={hit.id}
                  style={{ animationDelay: `${index * 50}ms` }}
                  className="bg-slate-900/40 border border-slate-800/60 p-4 rounded-xl hover:bg-slate-800/50 hover:border-blue-500/30 hover:-translate-y-1 transition-all cursor-default flex flex-col gap-1.5 relative overflow-hidden group animate-in slide-in-from-bottom-4 fade-in duration-500 fill-mode-both"
                >
                  <div className="absolute left-0 top-0 bottom-0 w-1 bg-gradient-to-b from-blue-500 to-indigo-500 opacity-0 group-hover:opacity-100 transition-opacity"></div>

                  <div className="text-lg font-semibold text-slate-100 group-hover:text-blue-400 transition-colors break-words">
                    {renderHighlightLine(titleLine)}
                  </div>

                  <div className="flex flex-wrap gap-3 text-xs text-slate-400">
                    {metaItems.map((item, idx) => (
                      <span
                        key={idx}
                        className="flex items-center gap-1.5 bg-slate-900/50 px-2 py-0.5 rounded-md border border-slate-800/50"
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
