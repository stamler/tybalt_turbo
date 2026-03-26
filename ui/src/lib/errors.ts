export function getApiErrorMessage(error: any, fallback: string): string {
  return error?.response?.error || error?.response?.message || fallback;
}
