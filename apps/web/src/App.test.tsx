import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, it, expect, vi } from "vitest";
import { App } from "./App";

const tennisDoc = {
  id: "doc-1",
  name: "Spring Tennis Academy",
  sport: "tennis",
  location: "Bradenton, FL",
  capacity: 30,
  start_date: "2026-04-10",
  end_date: "2026-04-14",
};
const footballDoc = { ...tennisDoc, id: "doc-2", name: "Summer Football Camp", sport: "football" };

describe("App", () => {
  beforeEach(() => {
    globalThis.fetch = vi.fn(async (input: RequestInfo | URL) => {
      const url = typeof input === "string" ? input : input.toString();
      const params = new URL(url, "http://localhost").searchParams;
      const sport = params.get("sport") ?? "";
      const all = [
        { id: tennisDoc.id, doc: tennisDoc },
        { id: footballDoc.id, doc: footballDoc },
      ];
      const filtered = sport ? all.filter((h) => h.doc.sport === sport) : all;
      return new Response(JSON.stringify({ hits: filtered }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }) as unknown as typeof fetch;
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders both camps initially, then filters down to football when the sport changes", async () => {
    const user = userEvent.setup();
    render(<App />);

    await waitFor(() => {
      expect(screen.getByText("Spring Tennis Academy")).toBeInTheDocument();
      expect(screen.getByText("Summer Football Camp")).toBeInTheDocument();
    });

    await user.selectOptions(screen.getByLabelText("filter by sport"), "football");

    await waitFor(() => {
      expect(screen.queryByText("Spring Tennis Academy")).not.toBeInTheDocument();
    });
    expect(screen.getByText("Summer Football Camp")).toBeInTheDocument();
  });
});
