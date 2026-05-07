(() => {
  const status = document.getElementById("status-text");
  const card = document.getElementById("status-card");

  const setStatus = (msg) => {
    if (status) {
      status.textContent = msg;
    }
  };

  if (!window.Go) {
    setStatus("wasm_exec.js failed to load.");
    return;
  }

  // Create a Promise that the Go runtime will resolve when initialization is complete.
  // This implements event-driven initialization without blocking boot.js.
  let runtimeReadyResolve;
  let runtimeReadyReject;
  const runtimeReadyPromise = new Promise((resolve, reject) => {
    runtimeReadyResolve = resolve;
    runtimeReadyReject = reject;
  });

  // Set up the callback namespace that Go will invoke
  if (!window.murmur) {
    window.murmur = {};
  }
  window.murmur.onRuntimeReady = (errMsg) => {
    if (errMsg) {
      // Error case: Go passed an error message
      runtimeReadyReject(new Error(errMsg));
    } else {
      // Success case: runtime initialized successfully
      runtimeReadyResolve();
    }
  };

  const go = new Go();

  const run = async () => {
    try {
      setStatus("Fetching murmur.wasm...");
      const result = await WebAssembly.instantiateStreaming(fetch("./murmur.wasm"), go.importObject);
      
      setStatus("Starting runtime...");
      // Call go.run() without awaiting — it will invoke our callback when ready
      go.run(result.instance);
      
      // Wait for the Go runtime to signal completion of initialization
      await runtimeReadyPromise;
      
      setStatus("WASM build: work in progress (see docs/IMPLEMENTATION_STATUS.md)");
      if (card) {
        card.style.opacity = "0.8";
      }
    } catch (err) {
      console.error("Failed to start MURMUR wasm runtime", err);
      setStatus("Failed to start runtime. See console logs.");
    }
  };

  run();
})();
