<script>
  import { configStore, crawlerStore } from '../stores/crawler.js';

  let config;
  configStore.subscribe(value => config = value);

  let state;
  crawlerStore.subscribe(value => state = value);

  $: isRunning = state.status === 'running';
  $: isPaused = state.status === 'paused';
  $: isStopped = state.status === 'stopped';

  async function startCrawl() {
    if (window.go && window.go.app && window.go.app.App) {
      try {
        crawlerStore.clearLogs();
        await window.go.app.App.StartCrawl(config);
      } catch (e) {
        crawlerStore.setError(e.toString());
      }
    }
  }

  async function pauseCrawl() {
    if (window.go && window.go.app && window.go.app.App) {
      try {
        await window.go.app.App.PauseCrawl();
      } catch (e) {
        crawlerStore.setError(e.toString());
      }
    }
  }

  async function resumeCrawl() {
    if (window.go && window.go.app && window.go.app.App) {
      try {
        await window.go.app.App.ResumeCrawl();
      } catch (e) {
        crawlerStore.setError(e.toString());
      }
    }
  }

  async function stopCrawl() {
    if (window.go && window.go.app && window.go.app.App) {
      try {
        await window.go.app.App.StopCrawl();
      } catch (e) {
        crawlerStore.setError(e.toString());
      }
    }
  }
</script>

<div class="control-buttons">
  {#if isStopped}
    <button class="btn-start" on:click={startCrawl} disabled={!config.url}>
      Start
    </button>
  {:else}
    {#if isPaused}
      <button class="btn-resume" on:click={resumeCrawl}>
        Resume
      </button>
    {:else}
      <button class="btn-pause" on:click={pauseCrawl}>
        Pause
      </button>
    {/if}
    <button class="btn-stop" on:click={stopCrawl}>
      Stop
    </button>
  {/if}

  {#if state.error}
    <div class="error-message">
      {state.error}
    </div>
  {/if}
</div>

<style>
  .control-buttons {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
  }

  button {
    flex: 1;
    min-width: 100px;
    padding: 12px 24px;
    border: none;
    border-radius: 6px;
    font-size: 1rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;
  }

  button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-start {
    background: #22c55e;
    color: #fff;
  }

  .btn-start:hover:not(:disabled) {
    background: #16a34a;
  }

  .btn-pause {
    background: #f59e0b;
    color: #fff;
  }

  .btn-pause:hover {
    background: #d97706;
  }

  .btn-resume {
    background: #22c55e;
    color: #fff;
  }

  .btn-resume:hover {
    background: #16a34a;
  }

  .btn-stop {
    background: #ef4444;
    color: #fff;
  }

  .btn-stop:hover {
    background: #dc2626;
  }

  .error-message {
    width: 100%;
    padding: 12px;
    background: #450a0a;
    border: 1px solid #ef4444;
    border-radius: 6px;
    color: #fca5a5;
    font-size: 0.9rem;
  }
</style>
