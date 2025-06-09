// lib/stores/collectionEvents.ts
import { writable } from 'svelte/store';

type CollectionEvent = {
  collection: string;
  action: 'create' | 'update' | 'delete';
  recordId: string;
}

export const collectionEvents = writable<CollectionEvent | null>(null);

export function emitCollectionEvent(collection: string, action: 'create' | 'update' | 'delete', recordId: string) {
  collectionEvents.set({ collection, action, recordId });
}