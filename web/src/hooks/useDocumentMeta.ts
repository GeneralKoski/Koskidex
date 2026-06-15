import { useEffect } from "react";

const SITE_URL = "https://koskidex.martin-trajkovski.it";

function setMeta(selector: string, attr: string, value: string) {
  let el = document.head.querySelector<HTMLMetaElement>(selector);
  if (!el) {
    el = document.createElement("meta");
    const [key, val] = selector
      .replace(/^meta\[/, "")
      .replace(/\]$/, "")
      .split("=");
    el.setAttribute(key, val.replace(/['"]/g, ""));
    document.head.appendChild(el);
  }
  el.setAttribute(attr, value);
}

function setCanonical(href: string) {
  let link = document.head.querySelector<HTMLLinkElement>('link[rel="canonical"]');
  if (!link) {
    link = document.createElement("link");
    link.setAttribute("rel", "canonical");
    document.head.appendChild(link);
  }
  link.setAttribute("href", href);
}

interface DocumentMeta {
  title: string;
  description: string;
  path?: string;
}

// useDocumentMeta sets the document title, description and Open Graph / Twitter
// tags for the current route. Lightweight alternative to react-helmet for an SPA.
export function useDocumentMeta({ title, description, path = "" }: DocumentMeta) {
  useEffect(() => {
    const url = SITE_URL + path;

    document.title = title;
    setMeta('meta[name="description"]', "content", description);
    setMeta('meta[property="og:title"]', "content", title);
    setMeta('meta[property="og:description"]', "content", description);
    setMeta('meta[property="og:url"]', "content", url);
    setMeta('meta[property="twitter:title"]', "content", title);
    setMeta('meta[property="twitter:description"]', "content", description);
    setMeta('meta[property="twitter:url"]', "content", url);
    setCanonical(url);
  }, [title, description, path]);
}
