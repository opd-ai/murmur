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

  const go = new Go();

  const run = async () => {
    try {
      setStatus("Fetching murmur.wasm...");
      const result = await WebAssembly.instantiateStreaming(fetch("./murmur.wasm"), go.importObject);
      setStatus("Starting runtime...");
      await go.run(result.instance);
      setStatus("Runtime started.");
      if (card) {
        card.style.opacity = "0.6";
      }
    } catch (err) {
      console.error("Failed to start MURMUR wasm runtime", err);
      setStatus("Failed to start runtime. See console logs.");
    }
  };

  run();
})();
