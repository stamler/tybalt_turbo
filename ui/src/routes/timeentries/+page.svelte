<script lang="ts">
	import type { PageData } from './$types';
	import type { TimeEntriesRecord } from '$lib/pocketbase-types';

	let { data }: { data: PageData } = $props();

	function hoursString(item: TimeEntriesRecord) {
		const hoursArray = [];
		if (item.hours) hoursArray.push(item.hours + ' hrs');
		if (item.meals_hours) hoursArray.push(item.meals_hours + ' hrs meals');
		return hoursArray.join(' + ');
	}
</script>

<!-- Show the list of items here -->
<ul class="flex flex-col">
	<!-- iterate over each key in the object -->
	{#each (data.items as TimeEntriesRecord[]) as item}
		<li class="flex even:bg-neutral-200 odd:bg-neutral-100">
			<div class="w-32">{item.date}</div>
			<div class="flex flex-col w-full">
				<div class="headline_wrapper">
					<div class="headline">
						{#if item.expand?.time_type.code === 'R'}
							<span>{item.expand.division.name}</span>
						{:else}
							<span>{item.expand?.time_type.name}</span>
						{/if}
					</div>
					<div class="byline">
						{#if item.expand?.time_type.code === 'OTO'}
							<span>${item.payout_request_amount}</span>
						{/if}
					</div>
				</div>
				<div class="firstline">
					{#if ['R', 'RT'].includes(item.expand?.time_type.code) && item.job !== ''}
						<span>{item.expand?.job.number}</span>
						{#if item.expand?.job.category}
							<span class="label">{item.expand.job.category}</span>
						{/if}
					{/if}
				</div>
				<div class="secondline">{hoursString(item)}</div>
				{#if item.description}
					<div class="thirdline">
						{#if item.work_record !== ''}
							<span>Work Record: {item.work_record} / </span>
						{/if}
						<span>{item.description}</span>
					</div>
				{/if}
			</div>
			<div class="rowactionsbox">
				<a href="/placeholder.html">a link</a>
			</div>
		</li>
	{/each}
</ul>
