const jobs = {
  jobA: {
    id: "jobA",
    number: "26-1001.01",
    description: "Project A",
    parent_id: "jobB",
    parent_number: "26-1001",
  },
  jobB: { id: "jobB", number: "26-1001", description: "Project B" },
};
const dataFor = (id: keyof typeof jobs) => ({
  job: { ...jobs[id], status: "Active", project_value: 1000, allocations: [], children: [] },
  notes: [],
});
export const props = $state({ data: dataFor("jobA") });
// Keep the real page mounted when route data changes, as SvelteKit does.
export function navigate(id: keyof typeof jobs) {
  props.data = dataFor(id);
}
