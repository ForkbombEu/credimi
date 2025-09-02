<!-- EditorWithRunner.svelte -->
<script lang="ts">
	import { FileText, Monitor, Play, SplitSquareHorizontal } from 'lucide-svelte';
	import { onMount } from 'svelte';

	import { Button } from '@/components/ui/button';

	import CodeEditor from './codeEditor.svelte';

	interface Props {
		value?: string;
		running?: boolean;
		output?: string;
		error?: string;
		onRun?: ((code: string) => void) | undefined;
	}

	let {
		value = $bindable(),
		running = false,
		output = '',
		error = '',
		onRun = undefined
	}: Props = $props();

	let editorValue = $state(value || '');
	let activeTab = $state<'editor' | 'output' | 'split'>('editor');

	$effect(() => {
		const currentValue = value || '';
		if (currentValue !== editorValue) {
			editorValue = currentValue;
		}
	});

	function updateValue(newValue: string) {
		editorValue = newValue;
		value = newValue;
	}

	const status = $derived(
		running ? 'Running...' : error ? 'Error' : output ? 'Complete' : 'Ready'
	);

	function run() {
		if (onRun) {
			activeTab = 'split';
			onRun(editorValue);
		}
	}

	function handleKeydown(event: KeyboardEvent) {
		if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') {
			event.preventDefault();
			run();
		}
	}

	onMount(() => {
		if (output || error) {
			activeTab = 'output';
		}
	});
</script>

<div class="border-border rounded-lg border">
	<!-- Header -->
	<div class="bg-muted/30 border-border flex items-center justify-between border-b px-4 py-3">
		<!-- Tab Navigation -->
		<div class="flex items-center gap-1">
			<button
				onclick={(e) => {
					e.preventDefault();
					e.stopPropagation();
					activeTab = 'editor';
				}}
				class="flex items-center gap-2 rounded-t-md px-3 py-2 text-sm font-medium transition-colors {activeTab ===
				'editor'
					? 'bg-background text-foreground border-primary border-b-2'
					: 'text-muted-foreground hover:text-foreground hover:bg-muted/50'}"
			>
				<FileText class="h-4 w-4" />
				<span class="hidden sm:inline">Editor</span>
			</button>
			<button
				onclick={(e) => {
					e.preventDefault();
					e.stopPropagation();
					activeTab = 'output';
				}}
				class="flex items-center gap-2 rounded-t-md px-3 py-2 text-sm font-medium transition-colors {activeTab ===
				'output'
					? 'bg-background text-foreground border-primary border-b-2'
					: 'text-muted-foreground hover:text-foreground hover:bg-muted/50'}"
			>
				<Monitor class="h-4 w-4" />
				<span class="hidden sm:inline">Output</span>
			</button>
			<button
				onclick={(e) => {
					e.preventDefault();
					e.stopPropagation();
					activeTab = 'split';
				}}
				class="flex items-center gap-2 rounded-t-md px-3 py-2 text-sm font-medium transition-colors {activeTab ===
				'split'
					? 'bg-background text-foreground border-primary border-b-2'
					: 'text-muted-foreground hover:text-foreground hover:bg-muted/50'}"
			>
				<SplitSquareHorizontal class="h-4 w-4" />
				<span class="hidden sm:inline">Split</span>
			</button>
		</div>

		<!-- Actions -->
		<div class="flex items-center gap-2">
			<!-- Status -->
			<div class="flex items-center gap-2">
				<div
					class="h-2 w-2 rounded-full {running
						? 'bg-yellow-500'
						: error
							? 'bg-red-500'
							: output
								? 'bg-green-500'
								: 'bg-gray-400'}"
				></div>
				<span class="text-muted-foreground text-sm">{status}</span>
			</div>

			{#if onRun}
				<Button
					onclick={(e) => {
						e.preventDefault();
						e.stopPropagation();
						run();
					}}
					size="sm"
					disabled={running}
					class="h-8 px-3 text-xs"
				>
					<Play class="mr-1 h-3 w-3" />
					{running ? 'Running...' : 'Run'}
				</Button>
			{/if}
		</div>
	</div>

	<!-- Content Area -->
	<div class="relative">
		<!-- Editor Tab -->
		{#if activeTab === 'editor'}
			<div
				class="min-h-[400px] overflow-hidden"
				onkeydown={handleKeydown}
				role="textbox"
				tabindex="0"
			>
				<CodeEditor
					bind:value={editorValue}
					lang="yaml"
					minHeight={400}
					maxHeight={600}
					onChange={updateValue}
					class="rounded-none border-0"
				/>
			</div>
		{/if}

		<!-- Output Tab -->
		{#if activeTab === 'output'}
			<div class="max-h-[600px] min-h-[400px] overflow-hidden">
				{#if running}
					<div class="flex h-[400px] items-center justify-center">
						<div class="text-center">
							<svg
								class="text-primary mx-auto mb-4 h-8 w-8 animate-spin"
								viewBox="0 0 24 24"
								fill="none"
							>
								<circle
									class="opacity-25"
									cx="12"
									cy="12"
									r="10"
									stroke="currentColor"
									stroke-width="4"
								></circle>
								<path
									class="opacity-75"
									fill="currentColor"
									d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
								></path>
							</svg>
							<p class="text-muted-foreground text-sm">Running workflow...</p>
						</div>
					</div>
				{:else if error && error.trim()}
					<div class="h-[400px] overflow-auto">
						<div class="h-full border-l-4 border-red-500 bg-red-50 p-4">
							<div class="mb-2 flex items-center">
								<svg
									class="mr-2 h-5 w-5 text-red-500"
									viewBox="0 0 24 24"
									fill="currentColor"
								>
									<path
										d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"
									/>
								</svg>
								<h3 class="font-medium text-red-800">Execution Error</h3>
							</div>
							<pre
								class="overflow-auto whitespace-pre-wrap rounded bg-red-100/50 p-3 font-mono text-sm leading-relaxed text-red-700">{error}</pre>
						</div>
					</div>
				{:else if output && output.trim()}
					<div class="h-[400px] overflow-auto">
						<pre
							class="text-foreground bg-muted/20 whitespace-pre-wrap p-4 font-mono text-sm leading-relaxed">{output}</pre>
					</div>
				{:else}
					<div class="flex h-[400px] items-center justify-center">
						<div class="max-w-sm text-center">
							<div
								class="bg-muted mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full"
							>
								<svg
									class="text-muted-foreground h-6 w-6"
									viewBox="0 0 24 24"
									fill="currentColor"
								>
									<path d="M8 5v14l11-7z" />
								</svg>
							</div>
							<h3 class="text-foreground mb-2 font-medium">Ready to run</h3>
							<p class="text-muted-foreground text-sm">
								Click the Run button or press ⌘+Enter to execute your YAML
								configuration and see the results here.
							</p>
						</div>
					</div>
				{/if}
			</div>
		{/if}

		<!-- Split View -->
		{#if activeTab === 'split'}
			<div class="flex h-[400px]">
				<!-- Editor Side -->
				<div
					class="border-border flex-1 overflow-hidden border-r"
					onkeydown={handleKeydown}
					role="textbox"
					tabindex="0"
				>
					<CodeEditor
						bind:value={editorValue}
						lang="yaml"
						minHeight={400}
						maxHeight={400}
						onChange={updateValue}
						class="rounded-none border-0"
					/>
				</div>

				<!-- Output Side -->
				<div class="h-[400px] flex-1 overflow-hidden">
					{#if running}
						<div class="flex h-full items-center justify-center">
							<div class="text-center">
								<svg
									class="text-primary mx-auto mb-4 h-8 w-8 animate-spin"
									viewBox="0 0 24 24"
									fill="none"
								>
									<circle
										class="opacity-25"
										cx="12"
										cy="12"
										r="10"
										stroke="currentColor"
										stroke-width="4"
									></circle>
									<path
										class="opacity-75"
										fill="currentColor"
										d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
									></path>
								</svg>
								<p class="text-muted-foreground text-sm">Running workflow...</p>
							</div>
						</div>
					{:else if error && error.trim()}
						<div class="h-full overflow-auto">
							<div class="h-full border-l-4 border-red-500 bg-red-50 p-4">
								<div class="mb-2 flex items-center">
									<svg
										class="mr-2 h-5 w-5 text-red-500"
										viewBox="0 0 24 24"
										fill="currentColor"
									>
										<path
											d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"
										/>
									</svg>
									<h3 class="font-medium text-red-800">Execution Error</h3>
								</div>
								<pre
									class="max-h-80 overflow-auto whitespace-pre-wrap rounded bg-red-100/50 p-3 font-mono text-sm leading-relaxed text-red-700">{error}</pre>
							</div>
						</div>
					{:else if output && output.trim()}
						<div class="h-full overflow-auto">
							<pre
								class="text-foreground bg-muted/20 h-full whitespace-pre-wrap p-4 font-mono text-sm leading-relaxed">{output}</pre>
						</div>
					{:else}
						<div class="flex h-full items-center justify-center">
							<div class="max-w-sm px-4 text-center">
								<div
									class="bg-muted mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full"
								>
									<svg
										class="text-muted-foreground h-6 w-6"
										viewBox="0 0 24 24"
										fill="currentColor"
									>
										<path d="M8 5v14l11-7z" />
									</svg>
								</div>
								<h3 class="text-foreground mb-2 font-medium">Ready to run</h3>
								<p class="text-muted-foreground text-sm">
									Execute your YAML configuration to see results here.
								</p>
							</div>
						</div>
					{/if}
				</div>
			</div>
		{/if}
	</div>
</div>

<style>
	pre {
		tab-size: 2;
		font-variant-ligatures: none;
	}
</style>
