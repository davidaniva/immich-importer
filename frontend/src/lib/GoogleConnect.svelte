<script>
  import { createEventDispatcher, onMount } from 'svelte';

  const dispatch = createEventDispatcher();

  let loading = false;
  let error = null;
  let authURL = null;
  let waitingForCallback = false;

  async function startAuth() {
    loading = true;
    error = null;

    try {
      authURL = await window.go.main.App.GetGoogleAuthURL();

      // Open auth URL in browser
      window.runtime.BrowserOpenURL(authURL);

      waitingForCallback = true;

      // Wait for the callback (OAuth code will be received)
      // The Go backend handles this via the local HTTP server
      // We'll poll for completion
      pollForCompletion();
    } catch (e) {
      error = e.message || 'Failed to start authentication';
      loading = false;
    }
  }

  async function pollForCompletion() {
    // Check every second if auth is complete
    const interval = setInterval(async () => {
      try {
        const state = await window.go.main.App.GetInitialState();
        if (state.hasGoogleAuth) {
          clearInterval(interval);
          dispatch('complete');
        }
      } catch (e) {
        // Ignore polling errors
      }
    }, 1000);

    // Timeout after 5 minutes
    setTimeout(() => {
      clearInterval(interval);
      if (waitingForCallback) {
        error = 'Authentication timed out. Please try again.';
        loading = false;
        waitingForCallback = false;
      }
    }, 5 * 60 * 1000);
  }
</script>

<div class="connect">
  <div class="icon">
    <svg width="64" height="64" viewBox="0 0 24 24" fill="none">
      <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4"/>
      <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
      <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
      <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
    </svg>
  </div>

  <h2>Connect to Google</h2>
  <p class="info">
    Sign in with your Google account to access your Google Takeout files.
  </p>

  {#if error}
    <p class="error">{error}</p>
  {/if}

  {#if waitingForCallback}
    <div class="waiting">
      <div class="spinner"></div>
      <p>Waiting for Google sign-in...</p>
      <p class="hint">Complete the sign-in in your browser</p>
    </div>
  {:else}
    <button on:click={startAuth} disabled={loading}>
      {loading ? 'Connecting...' : 'Sign in with Google'}
    </button>
  {/if}
</div>

<style>
  .connect {
    text-align: center;
    width: 100%;
    max-width: 400px;
  }

  .icon {
    margin-bottom: 1.5rem;
  }

  h2 {
    color: #333;
    margin-bottom: 0.5rem;
  }

  .info {
    color: #666;
    margin-bottom: 1.5rem;
  }

  button {
    padding: 0.75rem 2rem;
    border: none;
    border-radius: 6px;
    font-size: 1rem;
    cursor: pointer;
    background: #4285f4;
    color: white;
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
  }

  button:hover {
    background: #357ae8;
  }

  button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .error {
    color: #c00;
    margin-bottom: 1rem;
  }

  .waiting {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
  }

  .waiting p {
    color: #666;
  }

  .waiting .hint {
    font-size: 0.9rem;
    color: #999;
  }

  .spinner {
    width: 40px;
    height: 40px;
    border: 3px solid #eee;
    border-top-color: #4285f4;
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>
