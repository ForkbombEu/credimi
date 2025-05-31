<!-- src/routes/tests/[type]/+page.svelte -->
<script lang="ts">
	import Parent from '../+page.svelte';
	import type { WalletTestParams } from '../_partials';

	export let data: WalletTestParams;
</script>

<Parent {data} />
