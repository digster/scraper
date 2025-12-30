<script>
  import { crawlerStore } from '../stores/crawler.js';

  let state;
  crawlerStore.subscribe(value => state = value);

  $: progress = state.progress;
  $: status = state.status;

  function formatBytes(bytes) {
    if (!bytes) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB'];
    let unitIndex = 0;
    let value = bytes;
    while (value >= 1024 && unitIndex < units.length - 1) {
      value /= 1024;
      unitIndex++;
    }
    return `${value.toFixed(1)} ${units[unitIndex]}`;
  }
</script>

<div class="dashboard">
  <h2>Progress</h2>

  <div class="status-row">
    <span class="label">Status:</span>
    <span class="status status-{status}">{status}</span>
  </div>

  {#if progress}
    <div class="elapsed">
      <span class="label">Elapsed:</span>
      <span class="value">{progress.elapsedTime}</span>
    </div>

    <div class="progress-bar-container">
      <div class="progress-bar" style="width: {progress.percentage}%"></div>
      <span class="progress-text">{progress.percentage?.toFixed(1) || 0}%</span>
    </div>

    <div class="metrics-grid">
      <div class="metric">
        <span class="metric-label">Processed</span>
        <span class="metric-value">{progress.urlsProcessed || 0}</span>
      </div>
      <div class="metric">
        <span class="metric-label">Saved</span>
        <span class="metric-value success">{progress.urlsSaved || 0}</span>
      </div>
      <div class="metric">
        <span class="metric-label">Errors</span>
        <span class="metric-value error">{progress.urlsErrored || 0}</span>
      </div>
      <div class="metric">
        <span class="metric-label">Queue</span>
        <span class="metric-value">{progress.queueSize || 0}</span>
      </div>
      <div class="metric">
        <span class="metric-label">Speed</span>
        <span class="metric-value">{(progress.pagesPerSecond || 0).toFixed(2)} p/s</span>
      </div>
      <div class="metric">
        <span class="metric-label">Downloaded</span>
        <span class="metric-value">{formatBytes(progress.bytesDownloaded)}</span>
      </div>
    </div>

    {#if progress.currentUrl}
      <div class="current-url">
        <span class="label">Current:</span>
        <span class="url" title={progress.currentUrl}>{progress.currentUrl}</span>
      </div>
    {/if}
  {:else if status === 'stopped'}
    <div class="no-data">
      Configure and start a crawl to see progress
    </div>
  {/if}
</div>

<style>
  .dashboard {
    background: #16213e;
    border-radius: 8px;
    padding: 16px;
  }

  h2 {
    font-size: 1.1rem;
    margin-bottom: 16px;
    color: #fff;
  }

  .status-row, .elapsed {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 12px;
  }

  .label {
    color: #aaa;
    font-size: 0.9rem;
  }

  .status {
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 0.85rem;
    text-transform: uppercase;
  }

  .status-stopped { background: #374151; color: #9ca3af; }
  .status-running { background: #065f46; color: #34d399; }
  .status-paused { background: #78350f; color: #fbbf24; }

  .value {
    color: #fff;
    font-family: monospace;
  }

  .progress-bar-container {
    position: relative;
    height: 24px;
    background: #0f0f23;
    border-radius: 4px;
    overflow: hidden;
    margin-bottom: 16px;
  }

  .progress-bar {
    height: 100%;
    background: linear-gradient(90deg, #3b82f6, #8b5cf6);
    transition: width 0.3s ease;
  }

  .progress-text {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    font-size: 0.85rem;
    font-weight: 600;
    color: #fff;
    text-shadow: 0 1px 2px rgba(0,0,0,0.5);
  }

  .metrics-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 12px;
    margin-bottom: 16px;
  }

  .metric {
    background: #0f0f23;
    border-radius: 6px;
    padding: 12px;
    text-align: center;
  }

  .metric-label {
    display: block;
    font-size: 0.75rem;
    color: #888;
    margin-bottom: 4px;
    text-transform: uppercase;
  }

  .metric-value {
    font-size: 1.2rem;
    font-weight: 600;
    color: #fff;
    font-family: monospace;
  }

  .metric-value.success { color: #22c55e; }
  .metric-value.error { color: #ef4444; }

  .current-url {
    display: flex;
    gap: 8px;
    align-items: flex-start;
    padding: 8px;
    background: #0f0f23;
    border-radius: 4px;
  }

  .url {
    color: #60a5fa;
    font-size: 0.85rem;
    word-break: break-all;
    flex: 1;
    font-family: monospace;
  }

  .no-data {
    text-align: center;
    color: #666;
    padding: 32px;
  }
</style>
