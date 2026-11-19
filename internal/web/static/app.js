/* Aprimoramento progressivo do Cultivar.
 *
 * Nenhuma jornada depende deste arquivo: ele registra o service worker,
 * melhora o fechamento do menu "Mais" e avisa quando a rede cai.
 */
(() => {
  "use strict";

  if ("serviceWorker" in navigator) {
    window.addEventListener("load", () => {
      navigator.serviceWorker.register("/sw.js", { scope: "/" }).catch(() => {
        // Sem service worker a aplicação continua funcionando normalmente.
      });
    });
  }

  const moreMenu = document.querySelector("[data-more-menu]");
  if (moreMenu) {
    document.addEventListener("click", (event) => {
      if (moreMenu.open && !moreMenu.contains(event.target)) {
        moreMenu.open = false;
      }
    });

    document.addEventListener("keydown", (event) => {
      if (event.key !== "Escape" || !moreMenu.open) {
        return;
      }
      moreMenu.open = false;
      const summary = moreMenu.querySelector("summary");
      if (summary) {
        summary.focus();
      }
    });

    moreMenu.addEventListener("click", (event) => {
      if (event.target instanceof Element && event.target.closest("a")) {
        moreMenu.open = false;
      }
    });
  }

  const offlineBanner = document.querySelector("[data-offline-banner]");
  if (offlineBanner) {
    const updateNetworkState = () => {
      offlineBanner.hidden = navigator.onLine;
    };
    updateNetworkState();
    window.addEventListener("online", updateNetworkState);
    window.addEventListener("offline", updateNetworkState);
  }

  if (window.matchMedia("(display-mode: standalone)").matches) {
    document.documentElement.classList.add("is-standalone");
  }
})();
