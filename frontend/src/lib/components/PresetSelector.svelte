<script>
  import { onMount } from 'svelte';
  import { presetsStore } from '../stores/presets.js';
  import { configStore, crawlerStore } from '../stores/crawler.js';

  let presets = [];
  let loading = false;
  let error = null;
  let selectedPreset = '';
  let showSaveModal = false;
  let showDeleteConfirm = false;
  let newPresetName = '';
  let saveError = '';

  let status;
  crawlerStore.subscribe(value => status = value.status);

  $: disabled = status !== 'stopped';

  presetsStore.subscribe(value => {
    presets = value.presets;
    loading = value.loading;
    error = value.error;
  });

  onMount(() => {
    presetsStore.loadPresets();
  });

  async function handleLoadPreset() {
    if (!selectedPreset) return;

    const result = await presetsStore.loadPreset(selectedPreset);
    if (result.success) {
      configStore.applyPreset(result.config);
    } else {
      presetsStore.clearError();
      error = result.error;
    }
  }

  function openSaveModal() {
    newPresetName = '';
    saveError = '';
    showSaveModal = true;
  }

  function closeSaveModal() {
    showSaveModal = false;
    newPresetName = '';
    saveError = '';
  }

  async function handleSavePreset() {
    if (!newPresetName.trim()) {
      saveError = 'Please enter a preset name';
      return;
    }

    // Validate name format
    const validName = /^[a-zA-Z0-9][a-zA-Z0-9_-]*$/.test(newPresetName);
    if (!validName) {
      saveError = 'Name can only contain letters, numbers, dashes, and underscores';
      return;
    }

    if (newPresetName.length > 50) {
      saveError = 'Name is too long (max 50 characters)';
      return;
    }

    const config = configStore.getPresetConfig();
    const result = await presetsStore.savePreset(newPresetName, config);

    if (result.success) {
      selectedPreset = newPresetName;
      closeSaveModal();
    } else {
      saveError = result.error;
    }
  }

  function openDeleteConfirm() {
    if (!selectedPreset) return;
    showDeleteConfirm = true;
  }

  function closeDeleteConfirm() {
    showDeleteConfirm = false;
  }

  async function handleDeletePreset() {
    if (!selectedPreset) return;

    const result = await presetsStore.deletePreset(selectedPreset);
    if (result.success) {
      selectedPreset = '';
    }
    closeDeleteConfirm();
  }

  function handleKeydown(e) {
    if (e.key === 'Enter' && showSaveModal) {
      handleSavePreset();
    }
    if (e.key === 'Escape') {
      if (showSaveModal) closeSaveModal();
      if (showDeleteConfirm) closeDeleteConfirm();
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="preset-selector">
  <div class="preset-row">
    <select
      bind:value={selectedPreset}
      {disabled}
      class="preset-dropdown"
    >
      <option value="">Select a preset...</option>
      {#each presets as preset}
        <option value={preset.name}>{preset.name}</option>
      {/each}
    </select>

    <button
      class="btn-load"
      on:click={handleLoadPreset}
      disabled={disabled || !selectedPreset}
      title="Load selected preset"
    >
      Load
    </button>

    <button
      class="btn-save"
      on:click={openSaveModal}
      {disabled}
      title="Save current settings as preset"
    >
      Save
    </button>

    <button
      class="btn-delete"
      on:click={openDeleteConfirm}
      disabled={disabled || !selectedPreset}
      title="Delete selected preset"
    >
      Delete
    </button>
  </div>

  {#if error}
    <div class="error-message">{error}</div>
  {/if}
</div>

<!-- Save Modal -->
{#if showSaveModal}
  <!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
  <div class="modal-overlay" on:click={closeSaveModal}>
    <!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
    <div class="modal" on:click|stopPropagation>
      <h3>Save Preset</h3>
      <div class="modal-body">
        <label for="presetName">Preset Name</label>
        <!-- svelte-ignore a11y-autofocus -->
        <input
          type="text"
          id="presetName"
          bind:value={newPresetName}
          placeholder="e.g., my-site-settings"
          autofocus
        />
        {#if saveError}
          <div class="save-error">{saveError}</div>
        {/if}
      </div>
      <div class="modal-actions">
        <button class="btn-cancel" on:click={closeSaveModal}>Cancel</button>
        <button class="btn-confirm" on:click={handleSavePreset}>Save</button>
      </div>
    </div>
  </div>
{/if}

<!-- Delete Confirmation Modal -->
{#if showDeleteConfirm}
  <!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
  <div class="modal-overlay" on:click={closeDeleteConfirm}>
    <!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
    <div class="modal" on:click|stopPropagation>
      <h3>Delete Preset</h3>
      <div class="modal-body">
        <p>Are you sure you want to delete "{selectedPreset}"?</p>
        <p class="warning">This action cannot be undone.</p>
      </div>
      <div class="modal-actions">
        <button class="btn-cancel" on:click={closeDeleteConfirm}>Cancel</button>
        <button class="btn-delete-confirm" on:click={handleDeletePreset}>Delete</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .preset-selector {
    margin-bottom: 16px;
  }

  .preset-row {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .preset-dropdown {
    flex: 1;
    min-width: 0;
    padding: 8px 12px;
    border: 1px solid #2a3f5f;
    border-radius: 4px;
    background: #0f0f23;
    color: #fff;
    font-size: 0.9rem;
    cursor: pointer;
  }

  .preset-dropdown:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .preset-dropdown:focus {
    outline: none;
    border-color: #4a9eff;
  }

  button {
    padding: 8px 12px;
    border: none;
    border-radius: 4px;
    font-size: 0.85rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-load {
    background: #4a9eff;
    color: #fff;
  }

  .btn-load:hover:not(:disabled) {
    background: #3a8eef;
  }

  .btn-save {
    background: #22c55e;
    color: #fff;
  }

  .btn-save:hover:not(:disabled) {
    background: #16a34a;
  }

  .btn-delete {
    background: #ef4444;
    color: #fff;
  }

  .btn-delete:hover:not(:disabled) {
    background: #dc2626;
  }

  .error-message {
    margin-top: 8px;
    padding: 8px 12px;
    background: #450a0a;
    border: 1px solid #ef4444;
    border-radius: 4px;
    color: #fca5a5;
    font-size: 0.85rem;
  }

  /* Modal styles */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal {
    background: #16213e;
    border-radius: 8px;
    padding: 20px;
    min-width: 320px;
    max-width: 90vw;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.5);
  }

  .modal h3 {
    margin: 0 0 16px 0;
    color: #fff;
    font-size: 1.1rem;
  }

  .modal-body {
    margin-bottom: 20px;
  }

  .modal-body label {
    display: block;
    font-size: 0.85rem;
    margin-bottom: 6px;
    color: #aaa;
  }

  .modal-body input {
    width: 100%;
    padding: 10px 12px;
    border: 1px solid #2a3f5f;
    border-radius: 4px;
    background: #0f0f23;
    color: #fff;
    font-size: 0.9rem;
  }

  .modal-body input:focus {
    outline: none;
    border-color: #4a9eff;
  }

  .modal-body p {
    margin: 0 0 8px 0;
    color: #ccc;
    font-size: 0.9rem;
  }

  .modal-body .warning {
    color: #fca5a5;
    font-size: 0.85rem;
  }

  .save-error {
    margin-top: 8px;
    color: #fca5a5;
    font-size: 0.85rem;
  }

  .modal-actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
  }

  .btn-cancel {
    background: #2a3f5f;
    color: #ccc;
  }

  .btn-cancel:hover {
    background: #3a5f8f;
  }

  .btn-confirm {
    background: #22c55e;
    color: #fff;
  }

  .btn-confirm:hover {
    background: #16a34a;
  }

  .btn-delete-confirm {
    background: #ef4444;
    color: #fff;
  }

  .btn-delete-confirm:hover {
    background: #dc2626;
  }
</style>
