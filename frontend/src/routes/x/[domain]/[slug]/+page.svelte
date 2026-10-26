<script lang="ts">
	import Article from '../../../blog/[slug]/+page.svelte';
	export let data;
</script>

<Article {data} />
