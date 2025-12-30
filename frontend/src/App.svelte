<script>
  import { onMount } from 'svelte';
  import { crawlerStore, configStore } from './lib/stores/crawler.js';
  import ConfigForm from './lib/components/ConfigForm.svelte';
  import ProgressDashboard from './lib/components/ProgressDashboard.svelte';
  import LogViewer from './lib/components/LogViewer.svelte';
  import ControlButtons from './lib/components/ControlButtons.svelte';

  let showAdvanced = false;

  onMount(() => {
    // Set up event listeners for Wails events
    if (window.runtime) {
      window.runtime.EventsOn('progress', (event) => {
        crawlerStore.setProgress(event.data);
      });

      window.runtime.EventsOn('log', (event) => {
        crawlerStore.addLog({
          timestamp: new Date(event.timestamp).toLocaleTimeString(),
          level: event.data.level,
          message: event.data.message,
        });
      });

      window.runtime.EventsOn('crawl_started', () => {
        crawlerStore.setStatus('running');
        crawlerStore.setError(null);
      });

      window.runtime.EventsOn('crawl_paused', () => {
        crawlerStore.setStatus('paused');
      });

      window.runtime.EventsOn('crawl_resumed', () => {
        crawlerStore.setStatus('running');
      });

      window.runtime.EventsOn('crawl_stopped', () => {
        crawlerStore.setStatus('stopped');
      });

      window.runtime.EventsOn('crawl_completed', () => {
        crawlerStore.setStatus('stopped');
      });

      window.runtime.EventsOn('error', (event) => {
        crawlerStore.setError(event.data.message);
      });
    }
  });
</script>

<main>
  <header>
    <h1>Web Scraper</h1>
  </header>

  <div class="container">
    <div class="left-panel">
      <ConfigForm bind:showAdvanced />
      <ControlButtons />
    </div>

    <div class="right-panel">
      <ProgressDashboard />
      <LogViewer />
    </div>
  </div>
</main>

<style>
  main {
    height: 100%;
    display: flex;
    flex-direction: column;
    padding: 16px;
    gap: 16px;
  }

  header {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  header h1 {
    font-size: 1.5rem;
    font-weight: 600;
    color: #fff;
  }

  .container {
    flex: 1;
    display: grid;
    grid-template-columns: 350px 1fr;
    gap: 16px;
    min-height: 0;
  }

  .left-panel {
    display: flex;
    flex-direction: column;
    gap: 16px;
    overflow-y: auto;
    min-height: 0;
    padding-bottom: 8px;
  }

  .right-panel {
    display: flex;
    flex-direction: column;
    gap: 16px;
    min-height: 0;
  }

  @media (max-width: 900px) {
    .container {
      grid-template-columns: 1fr;
    }
  }
</style>
