# API de lectura de FaroContenido

Faro entrega los **datos** de tus artículos y tu sitio pone el diseño. Es contenido público (solo artículos publicados), con CORS abierto y caché corta.

`{sitio}` es el UUID del sitio o su dominio (con o sin `www`).

| Ruta | Qué devuelve |
| --- | --- |
| `GET /api/v1/{sitio}/posts?limit=20&page=1` | `site` (id, nombre, dominio, canonical, prefijo), `total` y `posts[]` con `slug`, `title`, `summary`, `meta_description`, `published_at`, `path` e `image` |
| `GET /api/v1/{sitio}/posts/{slug}` | lo anterior más `canonical`, `seo` (title, description), `content` (estructura: intro, secciones, FAQ), `html` listo para insertar y `jsonld` (BlogPosting y FAQPage) |
| `GET /api/v1/{sitio}/posts/{slug}/image` | portada en JPEG (1376×768) |

`image` es `{url, alt, width, height}` o `null` si el artículo no tiene portada.

## Conexión

Para que el panel marque tu sitio como **conectado**, la página de tu blog (`canonical + prefijo`, por ejemplo `/blog`) debe incluir `<meta name="faro-site" content="{site.id}">`. Faro la revisa sola cada pocos minutos.

## Con Next.js

El panel (*Conectar mi sitio → Con tu propio diseño*) genera `lib/faro.js` y la página del artículo con tu dominio ya puesto. Usa `fetch(..., { next: { revalidate: 300 } })` para que un artículo nuevo aparezca en unos minutos sin volver a desplegar.
