import MiniSearch from "minisearch";

// Pattern matching all job number formats (legacy, turbo, proposals, subjobs)
// Matches: YY-NNN, YY-NNNN, PYY-NNN, PYY-NNNN, with optional -SS (subjob) and -N (nested subjob)
const JOB_NUMBER_PATTERN = /P?[0-9]{2}-[0-9]{3,4}(?:-[0-9]{1,2})?(?:-[0-9])?/gi;

// Get MiniSearch's default tokenizer
const defaultTokenize = MiniSearch.getDefault("tokenize");

/**
 * Custom tokenizer that treats job numbers as single tokens.
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
