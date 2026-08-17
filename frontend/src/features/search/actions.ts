"use server";

import { suggest } from "@/features/search/api";

/**
 * The bridge that lets the client-side search box ask for suggestions: the
 * bearer-token client is server-only, so autocomplete goes through a Server
 * Action. Failures degrade to "no suggestions" — an autocomplete that surfaces
 * an error toast is worse than one that stays quiet.
 */
export async function getSuggestions(q: string): Promise<string[]> {
  const query = q.trim();
  if (query.length < 2) return [];

  try {
    return await suggest(query);
  } catch {
    return [];
  }
}
