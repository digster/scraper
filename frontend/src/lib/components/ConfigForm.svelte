<script>
  import { configStore, crawlerStore } from '../stores/crawler.js';

  export let showAdvanced = false;

  let config;
  configStore.subscribe(value => config = value);

  let status;
  crawlerStore.subscribe(value => status = value.status);

  // Update store directly from input event (fixes timing issue with bind:value)
  function handleInput(field) {
    return (e) => {
      const value = e.target.type === 'number' ? parseInt(e.target.value) || 0 : e.target.value;
      configStore.update(c => ({ ...c, [field]: value }));
    };
  }

  // Tooltip descriptions for options
  const tooltips = {
    maxDepth: "Maximum number of link hops from the starting URL. Depth is measured by discovery steps, not URL path depth.",
    delay: "Time to wait between fetches (e.g., 1s, 500ms). Helps avoid overwhelming servers and getting blocked.",
    minContent: "Minimum text content length (characters) required for a page to be saved. Filters out empty or minimal pages.",
    fetchMode: "HTTP Client is fast but may be blocked by anti-bot protection. Browser mode uses real Chrome to bypass such measures.",
    concurrent: "Process multiple URLs in parallel (up to 10 simultaneous requests). Faster but more resource intensive.",
    ignoreRobots: "Bypass robots.txt rules that restrict crawling. Use responsibly and only when permitted.",
    headless: "Run browser without visible window. Disable for debugging or manual CAPTCHA solving.",
    waitForLogin: "Pause before crawling to allow manual login. Browser will open to the URL, letting you log in before the crawl begins.",
    prefixFilter: "Only crawl URLs that start with this prefix. Leave empty to crawl any discovered URL.",
    excludeExtensions: "Skip downloading files with these extensions (comma-separated). Useful for excluding assets like images or scripts.",
    linkSelectors: "CSS selectors to filter which links to follow. Default follows all links with href attribute.",
    userAgent: "HTTP User-Agent header sent with requests. Some sites block non-browser user agents.",
    stateFile: "JSON file storing crawl progress. Allows resuming interrupted crawls from where they left off."
  };

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
      on:input={handleInput('url')}
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
      <label for="depth">
        Max Depth
        <span class="info-icon" title={tooltips.maxDepth}>i</span>
      </label>
      <input
        type="number"
        id="depth"
        bind:value={config.maxDepth}
        on:input={handleInput('maxDepth')}
        min="1"
        disabled={status !== 'stopped'}
      />
    </div>

    <div class="form-group">
      <label for="delay">
        Delay
        <span class="info-icon" title={tooltips.delay}>i</span>
      </label>
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
      <label for="minContent">
        Min Content
        <span class="info-icon" title={tooltips.minContent}>i</span>
      </label>
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
    <label for="fetchMode">
      Fetch Mode
      <span class="info-icon" title={tooltips.fetchMode}>i</span>
    </label>
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
          <span class="info-icon" title={tooltips.headless}>i</span>
        </label>
        {#if !config.headless}
          <label class="wait-login-toggle">
            <input
              type="checkbox"
              bind:checked={config.waitForLogin}
              disabled={status !== 'stopped'}
            />
            Wait for Login
            <span class="info-icon" title={tooltips.waitForLogin}>i</span>
          </label>
        {/if}
      {/if}
    </div>
  </div>

  <div class="checkbox-group">
    <label>
      <input type="checkbox" bind:checked={config.concurrent} disabled={status !== 'stopped'} />
      Concurrent Mode
      <span class="info-icon" title={tooltips.concurrent}>i</span>
    </label>
    <label>
      <input type="checkbox" bind:checked={config.ignoreRobots} disabled={status !== 'stopped'} />
      Ignore robots.txt
      <span class="info-icon" title={tooltips.ignoreRobots}>i</span>
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
        <label for="prefixFilter">
          Prefix Filter
          <span class="info-icon" title={tooltips.prefixFilter}>i</span>
        </label>
        <input
          type="text"
          id="prefixFilter"
          bind:value={config.prefixFilter}
          placeholder="https://example.com/docs"
          disabled={status !== 'stopped'}
        />
      </div>

      <div class="form-group">
        <label for="excludeExtensions">
          Exclude Extensions
          <span class="info-icon" title={tooltips.excludeExtensions}>i</span>
        </label>
        <input
          type="text"
          id="excludeExtensions"
          bind:value={config.excludeExtensions}
          placeholder="e.g., js,css,png,jpg"
          disabled={status !== 'stopped'}
        />
      </div>

      <div class="form-group">
        <label for="linkSelectors">
          Link Selectors
          <span class="info-icon" title={tooltips.linkSelectors}>i</span>
        </label>
        <input
          type="text"
          id="linkSelectors"
          bind:value={config.linkSelectors}
          placeholder="e.g., a[href], .nav-link"
          disabled={status !== 'stopped'}
        />
      </div>

      <div class="form-group">
        <label for="userAgent">
          User Agent
          <span class="info-icon" title={tooltips.userAgent}>i</span>
        </label>
        <input
          type="text"
          id="userAgent"
          list="userAgentOptions"
          bind:value={config.userAgent}
          placeholder="Default: WebScraper/1.0"
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
        <label for="stateFile">
          State File
          <span class="info-icon" title={tooltips.stateFile}>i</span>
        </label>
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
    display: inline-flex;
    align-items: center;
    gap: 4px;
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
    gap: 12px;
    flex-wrap: wrap;
    row-gap: 8px;
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
    display: inline-flex;
    align-items: center;
    gap: 4px;
    color: #ccc;
    cursor: pointer;
    font-size: 0.85rem;
  }

  .headless-toggle input[type="checkbox"] {
    width: 16px;
    height: 16px;
    cursor: pointer;
  }

  .info-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 12px;
    height: 12px;
    min-width: 12px;
    margin-left: 2px;
    font-size: 8px;
    font-weight: bold;
    font-style: italic;
    color: #666;
    background: #2a3f5f;
    border-radius: 50%;
    cursor: help;
    position: relative;
    vertical-align: baseline;
    transform: translateY(-1px);
  }

  .info-icon:hover {
    background: #4a9eff;
    color: #fff;
  }

  /* Checkbox group labels need special handling */
  .checkbox-group .info-icon {
    margin-left: 2px;
    vertical-align: middle;
    transform: none;
  }

  .headless-toggle .info-icon {
    margin-left: 2px;
    vertical-align: middle;
    transform: none;
  }

  .wait-login-toggle {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    color: #ccc;
    cursor: pointer;
    font-size: 0.85rem;
  }

  .wait-login-toggle input[type="checkbox"] {
    width: 16px;
    height: 16px;
    cursor: pointer;
  }

  .wait-login-toggle .info-icon {
    margin-left: 2px;
    vertical-align: middle;
    transform: none;
  }
</style>
