import type { PageLoad } from './$types';
import { pb } from '$lib/pocketbase';

export const load: PageLoad = async ({ params }) => {
  const record = await pb.collection('clients').getOne(params.cid);
  
  return {
    record,
    params,
  };
}; 