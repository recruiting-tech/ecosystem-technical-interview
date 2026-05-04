// Thin client over the indexer's /search endpoint. We don't generate from
// OpenAPI here because the indexer doesn't ship one — it's a search
// service, not a CRUD API. If the surface grows, switch to openapi-typegen.

export type CampDoc = {
  id: string;
  name: string;
  sport: string;
  location: string;
  capacity: number;
  start_date: string;
  end_date: string;
  // Row card already renders this when present; the *filter* is the
  // Ch3 stretch goal — wire it up to the form control if you have time.
  skill_level?: string | null;
};

export type SearchHit = {
  id: string;
  doc: CampDoc;
};

export type SearchResponse = {
  hits: SearchHit[];
};

export async function searchCamps(params: {
  q?: string;
  sport?: string;
  // skill_level?: string;  // TODO(ch3-stretch)
  limit?: number;
}): Promise<SearchResponse> {
  const qs = new URLSearchParams();
  if (params.q) qs.set("q", params.q);
  if (params.sport) qs.set("sport", params.sport);
  // if (params.skill_level) qs.set("skill_level", params.skill_level);  // TODO(ch3-stretch)
  if (params.limit) qs.set("limit", String(params.limit));

  const res = await fetch(`/api/search?${qs.toString()}`);
  if (!res.ok) {
    throw new Error(`search failed: ${res.status}`);
  }
  return res.json();
}
