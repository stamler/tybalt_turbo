/**
 * NOTE: DEAD CODE (eventually deletable).
 *
 * This file used to be referenced by:
 * - `ui/src/lib/stores/jobs.ts` via `tokenize`/`searchOptions.tokenize`
 * - `ui/src/lib/components/NoteForm.svelte` via `tokenize`/`searchOptions.tokenize`
 *
 * Both indexes now use MiniSearch's default tokenizer instead of this custom job-aware tokenizer.
 * Search behavior also moved to AND semantics (globally in `createCollectionStore` and explicitly in `NoteForm`)
 * with `number` field boosting (currently `boost: { number: 3 }`), so job number matches are prioritized
 * while still matching across other indexed fields.
 *
 * Implication: job-number parsing is now standard tokenization rather than custom full/partial job-number token
 * extraction. This specifically allows searching by the second segment of a job number on its own (for example,
 * searching `731` can now match `24-731`), because tokens are no longer forced into a single monolithic `YY-NNN`
 * style token. With the old custom tokenizer + prefix search, matching often depended on the year-prefixed token
 * and did not reliably support the second segment independently. Ranking bias toward job numbers is now handled by
 * field boost, not tokenizer-specific token shaping.
 */
import MiniSearch from "minisearch";

// Strict pattern for indexing - matches complete job number formats only
// Matches: YY-NNN, YY-NNNN, PYY-NNN, PYY-NNNN, with optional -SS (subjob) and -N (nested subjob)
const JOB_NUMBER_PATTERN = /P?[0-9]{2}-[0-9]{3,4}(?:-[0-9]{1,2})?(?:-[0-9])?/gi;

// Permissive pattern for search queries - matches partial job numbers
// Matches anything from YY- onwards (e.g., "25-", "24-7", "24-73", "24-731", "24-731-01")
const JOB_NUMBER_SEARCH_PATTERN = /P?[0-9]{2}-[0-9]*(?:-[0-9]*)?(?:-[0-9])?/gi;

// Get MiniSearch's default tokenizer
const defaultTokenize = MiniSearch.getDefault("tokenize");

/**
 * Custom tokenizer for indexing - treats complete job numbers as single tokens.
 * Job numbers are extracted first, then the remaining text is tokenized normally.
 */
export function jobAwareTokenize(text: string, fieldName?: string): string[] {
  if (!text) return [];

  // Find all job numbers in the text
  const jobNumbers = text.match(JOB_NUMBER_PATTERN) || [];

  // Remove job numbers from text to tokenize the rest
  const remainingText = text.replace(JOB_NUMBER_PATTERN, " ");

  // Tokenize remaining text with default tokenizer
  const otherTokens = defaultTokenize(remainingText, fieldName);

  // Return job numbers (lowercased for consistency) plus other tokens
  return [...jobNumbers.map((j) => j.toLowerCase()), ...otherTokens];
}

/**
 * Custom tokenizer for search queries - treats partial job numbers as single tokens.
 * More permissive than the index tokenizer to allow prefix matching (e.g., "24-73" matches "24-731").
 */
export function jobAwareTokenizeSearch(text: string, fieldName?: string): string[] {
  if (!text) return [];

  // Find all job number patterns (including partials) in the text
  const jobNumbers = text.match(JOB_NUMBER_SEARCH_PATTERN) || [];

  // Remove job numbers from text to tokenize the rest
  const remainingText = text.replace(JOB_NUMBER_SEARCH_PATTERN, " ");

  // Tokenize remaining text with default tokenizer
  const otherTokens = defaultTokenize(remainingText, fieldName);

  // Return job numbers (lowercased for consistency) plus other tokens
  return [...jobNumbers.map((j) => j.toLowerCase()), ...otherTokens];
}
