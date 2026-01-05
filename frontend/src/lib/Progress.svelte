<script>
  import { createEventDispatcher, onMount, onDestroy } from 'svelte';

  const dispatch = createEventDispatcher();

  let phase = 'downloading';
  let current = 0;
  let total = 0;
  let currentFile = '';
  let error = null;
  let jobState = null;

  let progressListener = null;
  let errorListener = null;
  let completeListener = null;

  onMount(async () => {
    // Get initial state
    try {
      jobState = await window.go.main.App.GetJobState();
      if (jobState) {
        phase = jobState.status;
        if (jobState.uploadState) {
          current = jobState.uploadState.uploadedPhotos;
          total = jobState.uploadState.totalPhotos;
        }
      }
    } catch (e) {
      // Ignore
    }

    // Listen for progress events
    progressListener = window.runtime.EventsOn('progress', (data) => {
      phase = data.phase;
      current = data.current;
      total = data.total;
      currentFile = data.currentFile;
    });

    errorListener = window.runtime.EventsOn('error', (msg) => {
      error = msg;
    });

    completeListener = window.runtime.EventsOn('complete', () => {
      dispatch('complete');
    });
  });

  onDestroy(() => {
    if (progressListener) {
      window.runtime.EventsOff('progress');
    }
    if (errorListener) {
      window.runtime.EventsOff('error');
    }
    if (completeListener) {
      window.runtime.EventsOff('complete');
    }
  });

  function getProgressPercent() {
    if (total === 0) return 0;
    return Math.round((current / total) * 100);
  }

  function getPhaseLabel() {
    switch (phase) {
      case 'downloading':
        return 'Downloading from Google Drive';
      case 'uploading':
        return 'Uploading to Immich';
      case 'complete':
        return 'Import Complete';
      default:
        return 'Processing';
    }
  }

  async function cancelImport() {
    try {
      await window.go.main.App.CancelImport();
    } catch (e) {
      // Ignore
    }
  }
</script>

<div class="progress">
  <h2>{getPhaseLabel()}</h2>

  {#if error}
    <div class="error">
      <p>{error}</p>
      <p class="hint">
        The import will resume from where it left off when you restart the app.
      </p>
    </div>
  {:else}
    <div class="progress-container">
      <div class="progress-bar">
        <div class="progress-fill" style="width: {getProgressPercent()}%"></div>
      </div>
      <div class="progress-text">
        {current} / {total} ({getProgressPercent()}%)
      </div>
    </div>

    {#if currentFile}
      <p class="current-file" title={currentFile}>
        {currentFile}
      </p>
    {/if}

    <div class="info">
      <p>
        {#if phase === 'downloading'}
          Downloading Takeout files from Google Drive...
        {:else if phase === 'uploading'}
          Uploading photos to your Immich server...
        {/if}
      </p>
      <p class="hint">
        You can close this app and it will resume where it left off.
      </p>
    </div>
  {/if}

  <button class="cancel" on:click={cancelImport}>
    Cancel Import
  </button>
</div>

<style>
  .progress {
    width: 100%;
    max-width: 500px;
    text-align: center;
  }

  h2 {
    color: #333;
    margin-bottom: 2rem;
  }

  .progress-container {
    margin-bottom: 1rem;
  }

  .progress-bar {
    height: 24px;
    background: #e0e0e0;
    border-radius: 12px;
    overflow: hidden;
    margin-bottom: 0.5rem;
  }

  .progress-fill {
    height: 100%;
    background: linear-gradient(90deg, #4a90d9, #6bb3f5);
    border-radius: 12px;
    transition: width 0.3s ease;
  }

  .progress-text {
    font-size: 1.1rem;
    color: #555;
    font-weight: 500;
  }

  .current-file {
    color: #666;
    font-size: 0.9rem;
    margin: 1rem 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .info {
    margin: 2rem 0;
  }

  .info p {
    color: #666;
    margin-bottom: 0.5rem;
  }

  .info .hint {
    color: #999;
    font-size: 0.9rem;
  }

  .error {
    background: #fee;
    border: 1px solid #fcc;
    border-radius: 8px;
    padding: 1.5rem;
    margin: 1rem 0;
  }

  .error p {
    color: #c00;
    margin-bottom: 0.5rem;
  }

  .error .hint {
    color: #666;
    font-size: 0.9rem;
  }

  button.cancel {
    margin-top: 1rem;
    padding: 0.5rem 1rem;
    background: #f0f0f0;
    color: #666;
    border: 1px solid #ddd;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.9rem;
  }

  button.cancel:hover {
    background: #e5e5e5;
  }
</style>
