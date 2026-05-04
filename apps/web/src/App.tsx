import { useState, useEffect } from "react";
import { searchCamps, type SearchHit } from "./api";

const SPORTS = ["", "football", "basketball", "soccer", "tennis", "volleyball"] as const;

export function App() {
  const [q, setQ] = useState("");
  const [sport, setSport] = useState<string>("");
  // const [skillLevel, setSkillLevel] = useState<string>("");  // TODO(ch3-stretch)
  const [hits, setHits] = useState<SearchHit[]>([]);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setErr(null);
    searchCamps({ q, sport })
      .then((res) => {
        if (!cancelled) setHits(res.hits ?? []);
      })
      .catch((e: Error) => {
        if (!cancelled) setErr(e.message);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [q, sport]);

  return (
    <main className="mx-auto max-w-4xl px-6 py-10">
      <header className="mb-8">
        <h1 className="text-3xl font-semibold tracking-tight">Camp search</h1>
        <p className="mt-1 text-sm text-zinc-600">
          Search across all camps. Results update as you type.
        </p>
      </header>

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3 mb-6">
        <label className="sm:col-span-2">
          <span className="block text-sm font-medium text-zinc-700">Query</span>
          <input
            type="search"
            placeholder="e.g. Bradenton, football, summer"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            className="mt-1 block w-full rounded-md border border-zinc-300 px-3 py-2 shadow-sm focus:border-zinc-500 focus:outline-none focus:ring-1 focus:ring-zinc-500"
            aria-label="search query"
          />
        </label>
        <label>
          <span className="block text-sm font-medium text-zinc-700">Sport</span>
          <select
            value={sport}
            onChange={(e) => setSport(e.target.value)}
            className="mt-1 block w-full rounded-md border border-zinc-300 px-3 py-2 shadow-sm focus:border-zinc-500 focus:outline-none focus:ring-1 focus:ring-zinc-500"
            aria-label="filter by sport"
          >
            {SPORTS.map((s) => (
              <option key={s} value={s}>
                {s === "" ? "Any sport" : s}
              </option>
            ))}
          </select>
        </label>

        {/*
          Ch3 stretch goal — wire up the skill_level filter end to end.

          <label>
            <span className="block text-sm font-medium text-zinc-700">Skill level</span>
            <select
              value={skillLevel}
              onChange={(e) => setSkillLevel(e.target.value)}
              className="..."
            >
              <option value="">Any</option>
              <option value="beginner">Beginner</option>
              <option value="intermediate">Intermediate</option>
              <option value="advanced">Advanced</option>
            </select>
          </label>
        */}
      </div>

      {err && (
        <div role="alert" className="mb-4 rounded-md bg-red-50 px-3 py-2 text-sm text-red-800">
          Search failed: {err}
        </div>
      )}

      {loading && hits.length === 0 ? (
        <p className="text-sm text-zinc-500">Searching…</p>
      ) : hits.length === 0 ? (
        <p className="text-sm text-zinc-500">No camps match that query.</p>
      ) : (
        <ul className="divide-y divide-zinc-200 rounded-md border border-zinc-200 bg-white">
          {hits.map((h) => (
            <li key={h.id} className="px-4 py-3">
              <div className="flex items-baseline justify-between gap-3">
                <div className="font-medium">{h.doc.name}</div>
                <div className="flex gap-2 text-xs uppercase tracking-wide">
                  {h.doc.skill_level && (
                    <span className="rounded bg-zinc-100 px-2 py-0.5 text-zinc-700">
                      {h.doc.skill_level}
                    </span>
                  )}
                  <span className="text-zinc-500">{h.doc.sport}</span>
                </div>
              </div>
              <div className="mt-1 text-sm text-zinc-600">
                {h.doc.location} · {h.doc.start_date} → {h.doc.end_date} · capacity{" "}
                {h.doc.capacity}
              </div>
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}
