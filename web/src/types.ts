export type MovieDocument = {
  id: string;
  title: string;
  genre: string;
  year: number;
  director: string;
  rating: number;
};

export type ProductDocument = {
  id: string;
  name: string;
  category: string;
  brand: string;
  price: number;
  rating: number;
};

export type Document = MovieDocument | ProductDocument;

export type Hit = {
  id: string;
  document: Document;
  highlights: Record<string, string>;
};

export type SearchResponse = {
  query: string;
  processing_time_ms: number;
  total_hits: number;
  hits: Hit[];
};
