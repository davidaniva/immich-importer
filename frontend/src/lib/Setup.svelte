<script>
  import { createEventDispatcher } from 'svelte';

  export let initialState = {};

  const dispatch = createEventDispatcher();

  let serverURL = initialState?.initialServerURL || '';
  let token = '';
  let loading = false;
  let error = null;

  // If we have initial values from bootstrap, use them
  $: if (initialState?.hasSetupToken && initialState?.initialServerURL) {
    serverURL = initialState.initialServerURL;
  }

  async function handleSetup() {
    if (!serverURL || !token) {
      error = 'Server URL and token are required';
      return;
    }

    loading = true;
    error = null;

    try {
      await window.go.main.App.Setup(serverURL, token);
      dispatch('complete');
    } catch (e) {
      error = e.message || 'Setup failed';
    } finally {
      loading = false;
    }
  }

  async function handleAutoSetup() {
    if (!initialState?.hasSetupToken) return;

    loading = true;
    error = null;

    try {
      await window.go.main.App.Setup('', '');
      dispatch('complete');
    } catch (e) {
      error = e.message || 'Setup failed';
    } finally {
      loading = false;
    }
  }
</script>

<div class="setup">
  <div class="icon">
    <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M12 2L2 7l10 5 10-5-10-5z"/>
      <path d="M2 17l10 5 10-5"/>
      <path d="M2 12l10 5 10-5"/>
    </svg>
  </div>

  <h2>Connect to Immich</h2>

  {#if initialState?.hasSetupToken}
    <p class="info">
      This app was configured to connect to your Immich server.
    </p>

    <button class="primary" on:click={handleAutoSetup} disabled={loading}>
      {loading ? 'Connecting...' : 'Connect to Server'}
    </button>

    <div class="divider">
      <span>or enter manually</span>
    </div>
  {/if}

  <form on:submit|preventDefault={handleSetup}>
    <div class="field">
      <label for="serverURL">Server URL</label>
      <input
        type="url"
        id="serverURL"
        bind:value={serverURL}
        placeholder="https://photos.example.com"
        disabled={loading}
      />
    </div>

    <div class="field">
      <label for="token">Setup Token</label>
      <input
        type="text"
        id="token"
        bind:value={token}
        placeholder="Token from Immich"
        disabled={loading}
      />
    </div>

    {#if error}
      <p class="error">{error}</p>
    {/if}

    <button type="submit" disabled={loading || !serverURL || !token}>
      {loading ? 'Connecting...' : 'Connect'}
    </button>
  </form>
</div>

<style>
  .setup {
    text-align: center;
    width: 100%;
    max-width: 400px;
  }

  .icon {
    color: #4a5568;
    margin-bottom: 1.5rem;
  }

  h2 {
    color: #333;
    margin-bottom: 1rem;
  }

  .info {
    color: #666;
    margin-bottom: 1.5rem;
  }

  form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .field {
    text-align: left;
  }

  label {
    display: block;
    margin-bottom: 0.25rem;
    color: #555;
    font-size: 0.9rem;
  }

  input {
    width: 100%;
    padding: 0.75rem;
    border: 1px solid #ddd;
    border-radius: 6px;
    font-size: 1rem;
  }

  input:focus {
    outline: none;
    border-color: #4a90d9;
  }

  button {
    padding: 0.75rem 1.5rem;
    border: none;
    border-radius: 6px;
    font-size: 1rem;
    cursor: pointer;
    background: #4a5568;
    color: white;
  }

  button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  button.primary {
    background: #4a90d9;
  }

  .divider {
    display: flex;
    align-items: center;
    margin: 1.5rem 0;
  }

  .divider::before,
  .divider::after {
    content: '';
    flex: 1;
    height: 1px;
    background: #ddd;
  }

  .divider span {
    padding: 0 1rem;
    color: #999;
    font-size: 0.9rem;
  }

  .error {
    color: #c00;
    font-size: 0.9rem;
  }
</style>
