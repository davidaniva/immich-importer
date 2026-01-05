<script>
  import { onMount } from 'svelte';
  import Setup from './lib/Setup.svelte';
  import GoogleConnect from './lib/GoogleConnect.svelte';
  import FileSelector from './lib/FileSelector.svelte';
  import Progress from './lib/Progress.svelte';
  import Complete from './lib/Complete.svelte';

  let step = 'loading';
  let initialState = null;
  let error = null;

  onMount(async () => {
    try {
      // Get initial state from Go backend
      initialState = await window.go.main.App.GetInitialState();

      // Determine starting step based on state
      if (initialState.needsSetup) {
        step = 'setup';
      } else if (!initialState.hasGoogleAuth) {
        step = 'connect';
      } else if (initialState.status === 'downloading' || initialState.status === 'uploading') {
        step = 'importing';
      } else if (initialState.status === 'complete') {
        step = 'complete';
      } else {
        step = 'select';
      }

      // Listen for events from Go backend
      window.runtime.EventsOn('progress', (data) => {
        // Progress updates are handled in Progress component
      });

      window.runtime.EventsOn('error', (msg) => {
        error = msg;
      });

      window.runtime.EventsOn('complete', () => {
        step = 'complete';
      });
    } catch (e) {
      error = e.message || 'Failed to initialize';
      step = 'setup';
    }
  });

  function goToStep(newStep) {
    step = newStep;
    error = null;
  }
</script>

<main>
  <header>
    <h1>Immich Google Photos Importer</h1>
    {#if initialState?.serverURL}
      <p class="server-url">Connected to: {initialState.serverURL}</p>
    {/if}
  </header>

  {#if error}
    <div class="error">
      <p>{error}</p>
      <button on:click={() => error = null}>Dismiss</button>
    </div>
  {/if}

  <div class="content">
    {#if step === 'loading'}
      <div class="loading">
        <p>Loading...</p>
      </div>
    {:else if step === 'setup'}
      <Setup
        {initialState}
        on:complete={() => goToStep('connect')}
      />
    {:else if step === 'connect'}
      <GoogleConnect
        on:complete={() => goToStep('select')}
      />
    {:else if step === 'select'}
      <FileSelector
        on:start={() => goToStep('importing')}
      />
    {:else if step === 'importing'}
      <Progress
        on:complete={() => goToStep('complete')}
      />
    {:else if step === 'complete'}
      <Complete />
    {/if}
  </div>
</main>

<style>
  main {
    max-width: 800px;
    margin: 0 auto;
    padding: 2rem;
    height: 100%;
    display: flex;
    flex-direction: column;
  }

  header {
    text-align: center;
    margin-bottom: 2rem;
  }

  h1 {
    color: #333;
    font-size: 1.8rem;
    margin-bottom: 0.5rem;
  }

  .server-url {
    color: #666;
    font-size: 0.9rem;
  }

  .content {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
  }

  .loading {
    color: #666;
  }

  .error {
    background: #fee;
    border: 1px solid #fcc;
    border-radius: 8px;
    padding: 1rem;
    margin-bottom: 1rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .error p {
    color: #c00;
    margin: 0;
  }

  .error button {
    background: #c00;
    color: white;
    border: none;
    padding: 0.5rem 1rem;
    border-radius: 4px;
    cursor: pointer;
  }
</style>
