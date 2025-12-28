// Client-side livereload script that we attach to the end of the body
(function (liveRoute) {
  const es = new EventSource(liveRoute);
  let needsReload = false;

  // Handle the eventsource connection
  es.addEventListener("open", function (e) {
    console.debug("livereload: connected to", liveRoute);
    if (needsReload) {
      console.debug("livereload: reloading page");
      document.dispatchEvent(
        new CustomEvent("reload", {
          bubbles: true,
          detail: {},
        }),
      );
    }
    needsReload = false;
  });

  // Handle errors
  es.addEventListener("error", function (e) {
    if (es.readyState === EventSource.CONNECTING) {
      console.debug("livereload: connection lost, reconnecting...");
      needsReload = true;
    } else if (es.readyState === EventSource.CLOSED) {
      console.debug("livereload: connection closed");
    } else {
      console.debug("livereload: error", e);
    }
  });

  // Handle reload events
  es.addEventListener("reload", function (e) {
    console.debug("livereload: eventsource got 'reload' event");
    const evs = e.data.split(";");
    const events = [];
    for (let i = 0; i < evs.length; i++) {
      const event = evs[i].split(":");
      if (event.length !== 2) continue;
      const op = event[0];
      const path = event[1];
      events.push({ op, path });
    }
    document.dispatchEvent(
      new CustomEvent("reload", {
        bubbles: true,
        detail: { events },
      }),
    );
  });

  // Close the event source when the browser window is closed
  window.addEventListener("beforeunload", function () {
    console.debug("livereload: closing event source");
    es.close();
  });

  // Default reload implementation. This can be overriden with
  // e.stopImmediatePropagation() to prevent the default behavior.
  window.addEventListener("reload", function (e) {
    console.debug("livereload: window got 'reload' event, reloading...");
    document.location.reload();
  });
})("__LIVE_ROUTE__");
