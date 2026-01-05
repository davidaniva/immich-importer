<script>
  import { createEventDispatcher, onMount } from 'svelte';

  const dispatch = createEventDispatcher();

  let files = [];
  let selectedFiles = new Set();
  let loading = true;
  let error = null;
  let starting = false;

  onMount(async () => {
    await loadFiles();
  });

  async function loadFiles() {
    loading = true;
    error = null;

    try {
      files = await window.go.main.App.ListTakeoutFiles();

      // Auto-select all files
      files.forEach(f => selectedFiles.add(f.id));
      selectedFiles = selectedFiles; // Trigger reactivity
    } catch (e) {
      error = e.message || 'Failed to load files';
    } finally {
      loading = false;
    }
  }

  function toggleFile(id) {
    if (selectedFiles.has(id)) {
      selectedFiles.delete(id);
    } else {
      selectedFiles.add(id);
    }
    selectedFiles = selectedFiles;
  }

  function toggleAll() {
    if (selectedFiles.size === files.length) {
      selectedFiles.clear();
    } else {
      files.forEach(f => selectedFiles.add(f.id));
    }
    selectedFiles = selectedFiles;
  }

  function formatSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
    return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
  }

  function getTotalSize() {
    let total = 0;
    files.forEach(f => {
      if (selectedFiles.has(f.id)) {
        total += f.size;
      }
    });
    return formatSize(total);
  }

  async function startImport() {
    if (selectedFiles.size === 0) return;

    starting = true;
    error = null;

    try {
      await window.go.main.App.StartImport(Array.from(selectedFiles));
      dispatch('start');
    } catch (e) {
      error = e.message || 'Failed to start import';
      starting = false;
    }
  }
</script>

<div class="selector">
  <h2>Select Takeout Files</h2>
  <p class="info">
    Choose the Google Takeout files to import to Immich.
  </p>

  {#if error}
    <p class="error">{error}</p>
  {/if}

  {#if loading}
    <div class="loading">
      <div class="spinner"></div>
      <p>Searching for Takeout files...</p>
    </div>
  {:else if files.length === 0}
    <div class="empty">
      <p>No Takeout files found in your Google Drive.</p>
      <p class="hint">
        Make sure you've created a Google Takeout export and it's available in your Drive.
        Note that Takeout exports can take hours or even days to generate.
      </p>
      <button on:click={loadFiles}>Refresh</button>
    </div>
  {:else}
    <div class="file-list">
      <div class="header">
        <label>
          <input
            type="checkbox"
            checked={selectedFiles.size === files.length}
            on:change={toggleAll}
          />
          Select All ({files.length} files)
        </label>
      </div>

      {#each files as file}
        <div class="file" class:selected={selectedFiles.has(file.id)}>
          <label>
            <input
              type="checkbox"
              checked={selectedFiles.has(file.id)}
              on:change={() => toggleFile(file.id)}
            />
            <span class="name">{file.name}</span>
            <span class="size">{formatSize(file.size)}</span>
          </label>
        </div>
      {/each}
    </div>

    <div class="actions">
      <div class="summary">
        {selectedFiles.size} files selected ({getTotalSize()})
      </div>
      <button
        class="primary"
        on:click={startImport}
        disabled={starting || selectedFiles.size === 0}
      >
        {starting ? 'Starting...' : 'Start Import'}
      </button>
    </div>
  {/if}
</div>

<style>
  .selector {
    width: 100%;
    max-width: 600px;
  }

  h2 {
    color: #333;
    margin-bottom: 0.5rem;
    text-align: center;
  }

  .info {
    color: #666;
    text-align: center;
    margin-bottom: 1.5rem;
  }

  .error {
    color: #c00;
    text-align: center;
    margin-bottom: 1rem;
  }

  .loading {
    text-align: center;
    padding: 2rem;
  }

  .loading p {
    color: #666;
    margin-top: 1rem;
  }

  .spinner {
    width: 40px;
    height: 40px;
    margin: 0 auto;
    border: 3px solid #eee;
    border-top-color: #4a90d9;
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  .empty {
    text-align: center;
    padding: 2rem;
    background: #fff;
    border-radius: 8px;
  }

  .empty .hint {
    color: #999;
    font-size: 0.9rem;
    margin-top: 0.5rem;
  }

  .empty button {
    margin-top: 1rem;
    padding: 0.5rem 1rem;
    background: #4a5568;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
  }

  .file-list {
    background: #fff;
    border-radius: 8px;
    overflow: hidden;
    max-height: 400px;
    overflow-y: auto;
  }

  .header {
    padding: 0.75rem 1rem;
    background: #f0f0f0;
    border-bottom: 1px solid #ddd;
  }

  .header label {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-weight: 500;
    cursor: pointer;
  }

  .file {
    padding: 0.75rem 1rem;
    border-bottom: 1px solid #eee;
  }

  .file:last-child {
    border-bottom: none;
  }

  .file.selected {
    background: #f0f7ff;
  }

  .file label {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    cursor: pointer;
  }

  .file .name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .file .size {
    color: #666;
    font-size: 0.9rem;
  }

  .actions {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 1rem;
    padding-top: 1rem;
    border-top: 1px solid #ddd;
  }

  .summary {
    color: #666;
    font-size: 0.9rem;
  }

  button.primary {
    padding: 0.75rem 1.5rem;
    background: #4a90d9;
    color: white;
    border: none;
    border-radius: 6px;
    font-size: 1rem;
    cursor: pointer;
  }

  button.primary:hover {
    background: #3d7dc4;
  }

  button.primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
