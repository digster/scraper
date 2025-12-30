<script>
  import { configStore, crawlerStore } from '../stores/crawler.js';

  export let showAdvanced = false;

  let config;
  configStore.subscribe(value => config = value);

  let status;
  crawlerStore.subscribe(value => status = value.status);

  async function browseDirectory() {
    if (window.go && window.go.app && window.go.app.App) {
      try {
        const dir = await window.go.app.App.BrowseDirectory();
        if (dir) {
          configStore.update(c => ({ ...c, outputDir: dir }));
        }
      } catch (e) {
        console.error('Failed to browse directory:', e);
      }
    }
  }

  async function browseStateFile() {
    if (window.go && window.go.app && window.go.app.App) {
      try {
        const file = await window.go.app.App.BrowseFile();
        if (file) {
          configStore.update(c => ({ ...c, stateFile: file }));
        }
      } catch (e) {
        console.error('Failed to browse file:', e);
      }
    }
  }
</script>

<div class="config-form">
  <h2>Configuration</h2>

  <div class="form-group">
    <label for="url">URL *</label>
    <input
      type="url"
      id="url"
      bind:value={config.url}
      placeholder="https://example.com"
      disabled={status !== 'stopped'}
    />
  </div>

  <div class="form-group">
    <label for="outputDir">Output Directory</label>
    <div class="input-with-button">
      <input
        type="text"
        id="outputDir"
        bind:value={config.outputDir}
        placeholder="Auto-generated from URL"
        disabled={status !== 'stopped'}
      />
      <button on:click={browseDirectory} disabled={status !== 'stopped'}>...</button>
    </div>
  </div>

  <div class="form-row">
    <div class="form-group">
      <label for="depth">Max Depth</label>
      <input
        type="number"
        id="depth"
        bind:value={config.maxDepth}
        min="1"
        disabled={status !== 'stopped'}
      />
    </div>

    <div class="form-group">
      <label for="delay">Delay</label>
      <input
        type="text"
        id="delay"
        bind:value={config.delay}
        placeholder="1s"
        disabled={status !== 'stopped'}
      />
    </div>
  </div>

  <div class="form-row">
    <div class="form-group">
      <label for="minContent">Min Content</label>
      <input
        type="number"
        id="minContent"
        bind:value={config.minContent}
        min="0"
        disabled={status !== 'stopped'}
      />
    </div>
  </div>

  <div class="form-group fetch-mode-group">
    <label for="fetchMode">Fetch Mode</label>
    <div class="fetch-mode-row">
      <select
        id="fetchMode"
        bind:value={config.fetchMode}
        disabled={status !== 'stopped'}
      >
        <option value="http">HTTP Client</option>
        <option value="browser">Browser (Chrome)</option>
      </select>
      {#if config.fetchMode === 'browser'}
        <label class="headless-toggle">
          <input
            type="checkbox"
            bind:checked={config.headless}
            disabled={status !== 'stopped'}
          />
          Headless
        </label>
      {/if}
    </div>
  </div>

  <div class="checkbox-group">
    <label>
      <input type="checkbox" bind:checked={config.concurrent} disabled={status !== 'stopped'} />
      Concurrent Mode
    </label>
    <label>
      <input type="checkbox" bind:checked={config.ignoreRobots} disabled={status !== 'stopped'} />
      Ignore robots.txt
    </label>
    <label>
      <input type="checkbox" bind:checked={config.verbose} disabled={status !== 'stopped'} />
      Verbose
    </label>
  </div>

  <button class="toggle-advanced" on:click={() => showAdvanced = !showAdvanced}>
    {showAdvanced ? '▼' : '▶'} Advanced Settings
  </button>

  {#if showAdvanced}
    <div class="advanced-settings">
      <div class="form-group">
        <label for="prefixFilter">Prefix Filter</label>
        <input
          type="text"
          id="prefixFilter"
          bind:value={config.prefixFilter}
          placeholder="https://example.com/docs"
          disabled={status !== 'stopped'}
        />
      </div>

      <div class="form-group">
        <label for="excludeExtensions">Exclude Extensions</label>
        <input
          type="text"
          id="excludeExtensions"
          bind:value={config.excludeExtensions}
          placeholder="js,css,png,jpg"
          disabled={status !== 'stopped'}
        />
      </div>

      <div class="form-group">
        <label for="linkSelectors">Link Selectors</label>
        <input
          type="text"
          id="linkSelectors"
          bind:value={config.linkSelectors}
          placeholder="a[href], .nav-link"
          disabled={status !== 'stopped'}
        />
      </div>

      <div class="form-group">
        <label for="userAgent">User Agent</label>
        <input
          type="text"
          id="userAgent"
          list="userAgentOptions"
          bind:value={config.userAgent}
          placeholder="Default WebScraper/1.0"
          disabled={status !== 'stopped'}
        />
        <datalist id="userAgentOptions">
          <option value="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36">Chrome (Windows)</option>
          <option value="Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36">Chrome (Mac)</option>
          <option value="Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0">Firefox (Windows)</option>
          <option value="Mozilla/5.0 (Macintosh; Intel Mac OS X 14_7_2) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.2 Safari/605.1.15">Safari (Mac)</option>
          <option value="Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)">Googlebot</option>
          <option value="WebScraper/1.0">WebScraper/1.0</option>
        </datalist>
      </div>

      <div class="form-group">
        <label for="stateFile">State File (for resume)</label>
        <div class="input-with-button">
          <input
            type="text"
            id="stateFile"
            bind:value={config.stateFile}
            placeholder="Auto-generated"
            disabled={status !== 'stopped'}
          />
          <button on:click={browseStateFile} disabled={status !== 'stopped'}>...</button>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .config-form {
    background: #16213e;
    border-radius: 8px;
    padding: 16px;
  }

  h2 {
    font-size: 1.1rem;
    margin-bottom: 16px;
    color: #fff;
  }

  .form-group {
    margin-bottom: 12px;
  }

  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }

  label {
    display: block;
    font-size: 0.85rem;
    margin-bottom: 4px;
    color: #aaa;
  }

  input[type="text"],
  input[type="url"],
  input[type="number"] {
    width: 100%;
    padding: 8px 12px;
    border: 1px solid #2a3f5f;
    border-radius: 4px;
    background: #0f0f23;
    color: #fff;
    font-size: 0.9rem;
  }

  input:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  input:focus {
    outline: none;
    border-color: #4a9eff;
  }

  .input-with-button {
    display: flex;
    gap: 8px;
  }

  .input-with-button input {
    flex: 1;
  }

  .input-with-button button {
    padding: 8px 12px;
    background: #2a3f5f;
    border: none;
    border-radius: 4px;
    color: #fff;
    cursor: pointer;
  }

  .input-with-button button:hover:not(:disabled) {
    background: #3a5f8f;
  }

  .input-with-button button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .checkbox-group {
    display: flex;
    flex-wrap: wrap;
    gap: 16px;
    margin: 16px 0;
  }

  .checkbox-group label {
    display: flex;
    align-items: center;
    gap: 6px;
    cursor: pointer;
    color: #ccc;
  }

  .checkbox-group input[type="checkbox"] {
    width: 16px;
    height: 16px;
    cursor: pointer;
  }

  .toggle-advanced {
    width: 100%;
    padding: 8px;
    background: transparent;
    border: 1px solid #2a3f5f;
    border-radius: 4px;
    color: #aaa;
    cursor: pointer;
    text-align: left;
    font-size: 0.9rem;
  }

  .toggle-advanced:hover {
    background: #1a2847;
  }

  .advanced-settings {
    margin-top: 16px;
    padding-top: 16px;
    border-top: 1px solid #2a3f5f;
  }

  .fetch-mode-group {
    margin-bottom: 16px;
  }

  .fetch-mode-row {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  select {
    padding: 8px 12px;
    border: 1px solid #2a3f5f;
    border-radius: 4px;
    background: #0f0f23;
    color: #fff;
    font-size: 0.9rem;
    cursor: pointer;
  }

  select:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  select:focus {
    outline: none;
    border-color: #4a9eff;
  }

  .headless-toggle {
    display: flex;
    align-items: center;
    gap: 6px;
    color: #ccc;
    cursor: pointer;
    font-size: 0.85rem;
  }

  .headless-toggle input[type="checkbox"] {
    width: 16px;
    height: 16px;
    cursor: pointer;
  }
</style>
