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
