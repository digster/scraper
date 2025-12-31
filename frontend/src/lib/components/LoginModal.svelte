<script>
  import { createEventDispatcher } from 'svelte';

  export let visible = false;
  export let url = '';

  const dispatch = createEventDispatcher();

  function confirmLogin() {
    dispatch('confirm');
  }
</script>

{#if visible}
  <div class="modal-overlay">
    <div class="modal">
      <h2>Login Required</h2>
      <p>The browser has opened to:</p>
      <code>{url}</code>
      <p>Please complete the login process in the browser window, then click the button below to continue crawling.</p>
      <button class="btn-confirm" on:click={confirmLogin}>
        Login Complete - Start Crawling
      </button>
    </div>
  </div>
{/if}

<style>
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
    border-radius: 12px;
    padding: 24px;
    max-width: 500px;
    width: 90%;
    text-align: center;
    border: 1px solid #2a3f5f;
  }

  h2 {
    color: #fff;
    margin: 0 0 16px 0;
    font-size: 1.25rem;
  }

  p {
    color: #aaa;
    margin: 0 0 12px 0;
    line-height: 1.5;
  }

  code {
    display: block;
    background: #0f0f23;
    padding: 8px 12px;
    border-radius: 4px;
    color: #4a9eff;
    margin-bottom: 16px;
    word-break: break-all;
    font-size: 0.9rem;
  }

  .btn-confirm {
    background: #22c55e;
    color: #fff;
    border: none;
    padding: 12px 24px;
    border-radius: 6px;
    font-size: 1rem;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.2s;
  }

  .btn-confirm:hover {
    background: #16a34a;
  }

  .btn-confirm:active {
    background: #15803d;
  }
</style>
