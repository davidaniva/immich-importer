<script>
  async function openImmich() {
    try {
      const state = await window.go.main.App.GetInitialState();
      if (state?.serverURL) {
        window.runtime.BrowserOpenURL(state.serverURL);
      }
    } catch (e) {
      // Ignore
    }
  }

  async function resetAndStartOver() {
    try {
      await window.go.main.App.ResetState();
      window.location.reload();
    } catch (e) {
      // Ignore
    }
  }
</script>

<div class="complete">
  <div class="icon">
    <svg width="80" height="80" viewBox="0 0 24 24" fill="none" stroke="#4caf50" stroke-width="2">
      <circle cx="12" cy="12" r="10"/>
      <path d="M8 12l2 2 6-6"/>
    </svg>
  </div>

  <h2>Import Complete!</h2>

  <p class="info">
    Your Google Photos have been successfully imported to Immich.
  </p>

  <div class="actions">
    <button class="primary" on:click={openImmich}>
      Open Immich
    </button>

    <button class="secondary" on:click={resetAndStartOver}>
      Import More Photos
    </button>
  </div>

  <div class="cleanup">
    <h3>Cleanup</h3>
    <p>
      You may want to:
    </p>
    <ul>
      <li>Delete the Takeout files from your Google Drive</li>
      <li>Revoke the API key in Immich Settings &rarr; API Keys</li>
    </ul>
  </div>
</div>

<style>
  .complete {
    text-align: center;
    width: 100%;
    max-width: 500px;
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
    margin-bottom: 2rem;
  }

  .actions {
    display: flex;
    gap: 1rem;
    justify-content: center;
    margin-bottom: 2rem;
  }

  button {
    padding: 0.75rem 1.5rem;
    border: none;
    border-radius: 6px;
    font-size: 1rem;
    cursor: pointer;
  }

  button.primary {
    background: #4caf50;
    color: white;
  }

  button.primary:hover {
    background: #43a047;
  }

  button.secondary {
    background: #f0f0f0;
    color: #333;
    border: 1px solid #ddd;
  }

  button.secondary:hover {
    background: #e5e5e5;
  }

  .cleanup {
    text-align: left;
    background: #f9f9f9;
    border-radius: 8px;
    padding: 1.5rem;
  }

  .cleanup h3 {
    color: #555;
    font-size: 1rem;
    margin-bottom: 0.5rem;
  }

  .cleanup p {
    color: #666;
    font-size: 0.9rem;
    margin-bottom: 0.5rem;
  }

  .cleanup ul {
    color: #666;
    font-size: 0.9rem;
    padding-left: 1.5rem;
  }

  .cleanup li {
    margin-bottom: 0.25rem;
  }
</style>
