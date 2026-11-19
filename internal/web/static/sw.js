// Service worker do Cultivar.
//
// A versão é substituída pelo servidor ao entregar /sw.js. Ela muda sempre que
// o conteúdo dos assets embutidos muda, o que invalida o cache antigo.
const ASSET_VERSION = "__ASSET_VERSION__";
const CACHE_NAME = "cultivar-" + ASSET_VERSION;

const PRECACHE = [
  "/offline",
  "/manifest.webmanifest",
  "/static/app.css?v=" + ASSET_VERSION,
  "/static/app.js?v=" + ASSET_VERSION,
  "/static/icons/icon-192.png",
  "/static/icons/icon-512.png",
  "/static/icons/icon-maskable-512.png",
  "/static/icons/icon.svg",
];

self.addEventListener("install", (event) => {
  event.waitUntil(
    (async () => {
      const cache = await caches.open(CACHE_NAME);
      await Promise.all(
        PRECACHE.map(async (url) => {
          try {
            await cache.add(new Request(url, { cache: "reload" }));
          } catch {
            // Um recurso indisponível não deve impedir a instalação inteira.
          }
        }),
      );
      await self.skipWaiting();
    })(),
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    (async () => {
      const keys = await caches.keys();
      await Promise.all(
        keys
          .filter((key) => key.startsWith("cultivar-") && key !== CACHE_NAME)
          .map((key) => caches.delete(key)),
      );
      await self.clients.claim();
    })(),
  );
});

self.addEventListener("fetch", (event) => {
  const request = event.request;
  if (request.method !== "GET") {
    return;
  }

  const url = new URL(request.url);
  if (url.origin !== self.location.origin) {
    return;
  }

  // Navegações usam a rede primeiro e nunca entram no cache.
  if (request.mode === "navigate") {
    event.respondWith(
      fetch(request).catch(async () => {
        const offline = await caches.match("/offline");
        return (
          offline ||
          new Response("Sem conexão. Tente novamente quando a rede voltar.", {
            status: 503,
            headers: { "Content-Type": "text/plain; charset=utf-8" },
          })
        );
      }),
    );
    return;
  }

  // Apenas assets públicos e estáveis são servidos do cache.
  const isStatic = url.pathname.startsWith("/static/");
  const isManifest = url.pathname === "/manifest.webmanifest";
  if (!isStatic && !isManifest) {
    return;
  }

  event.respondWith(
    (async () => {
      const cached = await caches.match(request);
      if (cached) {
        return cached;
      }

      const response = await fetch(request);
      if (response.ok) {
        const cache = await caches.open(CACHE_NAME);
        cache.put(request, response.clone()).catch(() => {
          // Falha de cota ou cache indisponível não impede a resposta.
        });
      }
      return response;
    })(),
  );
});
