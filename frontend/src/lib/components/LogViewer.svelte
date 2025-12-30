<script>
  import { crawlerStore } from '../stores/crawler.js';
  import { afterUpdate } from 'svelte';

  let state;
  crawlerStore.subscribe(value => state = value);

  let logContainer;
  let autoScroll = true;

  $: logs = state.logs;

  afterUpdate(() => {
    if (autoScroll && logContainer) {
      logContainer.scrollTop = logContainer.scrollHeight;
    }
  });

  function handleScroll() {
    if (logContainer) {
      const { scrollTop, scrollHeight, clientHeight } = logContainer;
      autoScroll = scrollHeight - scrollTop - clientHeight < 50;
    }
  }

  function clearLogs() {
    crawlerStore.clearLogs();
  }

  function getLevelClass(level) {
    switch (level?.toLowerCase()) {
      case 'error': return 'level-error';
      case 'warn': return 'level-warn';
      case 'debug': return 'level-debug';
      default: return 'level-info';
    }
  }
</script>

<div class="log-viewer">
  <div class="header">
    <h2>Logs</h2>
    <div class="controls">
      <label>
        <input type="checkbox" bind:checked={autoScroll} />
        Auto-scroll
      </label>
      <button on:click={clearLogs}>Clear</button>
    </div>
  </div>

  <div class="log-container" bind:this={logContainer} on:scroll={handleScroll}>
    {#if logs.length === 0}
      <div class="no-logs">No logs yet</div>
    {:else}
      {#each logs as log}
        <div class="log-entry {getLevelClass(log.level)}">
          <span class="timestamp">{log.timestamp}</span>
          <span class="level">[{log.level?.toUpperCase() || 'INFO'}]</span>
          <span class="message">{log.message}</span>
        </div>
      {/each}
    {/if}
  </div>
</div>

<style>
  .log-viewer {
    flex: 1;
    display: flex;
    flex-direction: column;
    background: #16213e;
    border-radius: 8px;
    min-height: 200px;
    overflow: hidden;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;
    border-bottom: 1px solid #2a3f5f;
  }

  h2 {
    font-size: 1.1rem;
    color: #fff;
    margin: 0;
  }

  .controls {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .controls label {
    display: flex;
    align-items: center;
    gap: 6px;
    color: #aaa;
    font-size: 0.85rem;
    cursor: pointer;
  }

  .controls button {
    padding: 4px 12px;
    background: #2a3f5f;
    border: none;
    border-radius: 4px;
    color: #fff;
    font-size: 0.85rem;
    cursor: pointer;
  }

  .controls button:hover {
    background: #3a5f8f;
  }

  .log-container {
    flex: 1;
    overflow-y: auto;
    padding: 8px;
    font-family: 'Monaco', 'Consolas', monospace;
    font-size: 0.8rem;
    background: #0f0f23;
  }

  .no-logs {
    text-align: center;
    color: #666;
    padding: 24px;
  }

  .log-entry {
    display: flex;
    gap: 8px;
    padding: 4px 8px;
    border-radius: 2px;
    margin-bottom: 2px;
  }

  .log-entry:hover {
    background: rgba(255, 255, 255, 0.05);
  }

  .timestamp {
    color: #666;
    flex-shrink: 0;
  }

  .level {
    flex-shrink: 0;
    font-weight: 600;
    width: 60px;
  }

  .message {
    color: #ccc;
    word-break: break-all;
  }

  .level-info .level { color: #60a5fa; }
  .level-warn .level { color: #fbbf24; }
  .level-error .level { color: #f87171; }
  .level-debug .level { color: #a78bfa; }

  .level-error .message { color: #fca5a5; }
  .level-warn .message { color: #fde68a; }
</style>
