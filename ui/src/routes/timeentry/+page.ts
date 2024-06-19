import type { TimeTypesRecord, DivisionsRecord, JobsRecord } from "$lib/pocketbase-types"
import { pb } from '$lib/pocketbase'
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
  let jobs: JobsRecord[]
  let timetypes: TimeTypesRecord[]
  let divisions: DivisionsRecord[]

  try {
    // load required data
    console.log('loading data')
    jobs = await pb.collection('jobs').getFullList()
    timetypes = await pb.collection('time_types').getFullList()
    divisions = await pb.collection('divisions').getFullList()
    return {
      jobs,
      timetypes,
      divisions,
    };
  } catch (error) {
    console.error(`loading data: ${error}`)
  }
};