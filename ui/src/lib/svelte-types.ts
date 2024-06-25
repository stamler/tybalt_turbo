import type { TimeEntriesRecord } from '$lib/pocketbase-types';

export interface TimeEntriesPageData {
  item: TimeEntriesRecord;
  editing: boolean;
  id: string | null;
}