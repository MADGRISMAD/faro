# FaroContenido — Fase 1

Panel editorial para generar artículos en español de México, revisarlos y publicarlos en un blog servido por SSR. El backend y el worker están hechos en **Go** por decisión de Manuel; comparten binario y usan PostgreSQL como cola persistente. El frontend y las páginas públicas usan SvelteKit con `adapter-node`.

## Arranque local

Con Docker activo, desde la raíz del proyecto:

```sh
docker-compose up -d --build
```

Ese comando levanta PostgreSQL, API, worker, frontend y sitio de ejemplo. Sin `.env`, el worker usa el proveedor simulado. Para usar Gemini, crea un `.env` privado a partir de `.env.example` y configura `GEMINI_API_KEY` y `GEMINI_MODEL=gemini-3.8-flash`. Las tarifas configuradas en Compose son `0.75` USD por millón de tokens de entrada y `3.75` de salida; pueden sustituirse en `.env`. La llave nunca se envía al frontend. Este checkout ya tiene un `.env` local ignorado por Git.

| Servicio | URL local |
| --- | --- |
| Panel | http://localhost:18080/dashboard |
| API | http://localhost:18081/health |
| Sitio de ejemplo | http://localhost:18082/ |
| Blog del ejemplo | http://localhost:18082/blog |
| Página con widget | http://localhost:18082/widget |
| PostgreSQL | localhost:15432 |

Para usar el ejemplo, crea una cuenta y un sitio con dominio `localhost:18082`, URL canónica `http://localhost:18082` y prefijo `/blog`. El dominio se reserva de forma única. El panel muestra el UUID público y el código del widget. Los artículos requieren aprobación antes de publicarse.

## Qué funciona

- Registro/login con Argon2id, sesiones en cookie HttpOnly y SameSite=Lax, CSRF por sesión, validación del origen y límite de intentos. Las consultas privadas filtran por cliente.
- Alta y edición de sitios; generación asíncrona con job en Postgres, `FOR UPDATE SKIP LOCKED`, máximo dos intentos, una generación simultánea por cliente e idempotencia por clave.
- Reserva atómica del presupuesto antes de llamar al proveedor, liquidación en unidades enteras de microdólar, registro de modelo, tokens y costo. Si se interrumpe el worker y no se conoce el uso, se conserva el importe reservado como costo **incierto** y se registra así.
- Proveedor Gemini y proveedor simulado detrás de una interfaz. Se solicita JSON con esquema y `maxOutputTokens` desde la primera llamada, sin grounding.
- Editor de título, slug, metadatos, secciones y FAQ; estados borrador → aprobado → publicado. El contenido se convierte a texto estructurado sin HTML crudo.
- Blog SSR con canonical, Open Graph, JSON-LD `BlogPosting`, sitemap, robots y 404 reales. Los borradores no se exponen.
- Widget JS público `/api/embed/<uuid>.js`, con CORS `*`, sin cookies, DOM aislado mediante Shadow DOM, texto insertado con `textContent` y caché de 60 segundos. El sitio de ejemplo hace proxy de `/blog/*` y de los recursos del frontend.

## Verificación ejecutada

El 20 de octubre de 2026 se comprobó en esta máquina:

| Prueba | Resultado |
| --- | --- |
| `TEST_DATABASE_URL=postgres://faro:faro@localhost:15432/faro_test go test -v ./...` | 3 pruebas pasaron con Postgres real y proveedor simulado: reserva concurrente, aislamiento, idempotencia, dos intentos, borrador privado, flujo editorial y saneamiento XSS. |
| `go vet ./...` y `gofmt` | Sin errores. |
| `npm run check`, `npm run lint`, `npm test`, `npm run build` | Pasaron; 1 prueba unitaria del canonical. |
| `npm audit` | 0 vulnerabilidades después de actualizar SvelteKit y Vitest. |
| `npx playwright test` | 1 recorrido completo pasó: registro → sitio → generar → editar → aprobar → publicar → proxy SSR → widget; también 404 y lectura sin JavaScript. |
| `curl` del artículo publicado | HTTP 200, texto presente en el HTML inicial, canonical y JSON-LD presentes; sitemap y robots HTTP 200; artículo inexistente HTTP 404. |
| Docker Compose | Los cinco servicios arrancaron y se comprobaron por HTTP. |
| Vista móvil | A 390 px, panel y artículo conservan el ancho de pantalla sin desplazamiento horizontal. |

Capturas: [panel](docs/screenshots/panel.png), [editor](docs/screenshots/editor.png), [blog](docs/screenshots/blog.png), [widget](docs/screenshots/widget.png).

Para repetir las pruebas Go sin tocar los datos del panel, crea una base de pruebas una vez:

```sh
docker-compose exec -T postgres createdb -U faro faro_test
cd backend && TEST_DATABASE_URL=postgres://faro:faro@localhost:15432/faro_test go test -v ./...
```

El E2E requiere los servicios levantados y un **worker simulado** para no consumir Gemini. Su archivo está en `frontend/tests/e2e/flow.spec.ts`; usa un dominio de prueba único en cada ejecución.

## Generación real y costo

La llave privada existente respondió a `models.list`: `gemini-3.8-flash` estaba disponible. Se usó ese ID explícito, sin fallback. Una generación real terminó en un intento y registró **120 tokens de entrada, 1,082 de salida y US$0.004148** con la tarifa configurada, muy por debajo del máximo autorizado de US$0.50. Google confirma el [modelo](https://ai.google.dev/gemini-api/docs/models/gemini-3.8-flash) y la [tarifa vigente hasta el 31 de diciembre de 2026](https://ai.google.dev/gemini-api/docs/pricing).

## Decisiones y límites de esta fase

El sitio de ejemplo demuestra el proxy y el widget; cada sitio real debe apuntar su ruta del blog al frontend y cargar el script desde el host de FaroContenido. Antes de abrir registros a clientes externos hace falta verificar la propiedad de los dominios. El límite de intentos vive en memoria del proceso API, adecuado para esta instancia local. No se añadieron redes sociales, CMS, pagos, voz, video ni despliegue.


## Pegar una URL y que la IA haga el resto (octubre 2026)

- **Investigación automática:** en el panel solo se pega la dirección del sitio (con o sin `https`/`www`). Un job lee el sitio con protección SSRF, Gemini deduce negocio, público, tono, zona y temas, y arranca el autopiloto.
- **Autopiloto prudente:** 1 artículo al inicio, 1 por semana las dos primeras semanas y máximo 3 por semana después. En sectores de salud, legal o finanzas, o si el artículo sale corto (<500 palabras), queda en borrador.
- **Imagen de portada:** una por artículo con `gemini-3.1-flash-lite-image` (Nano Banana 2 Lite), ilustración sin texto, guardada como JPEG. Se puede apagar y elegir estilo en *Brand DNA*. Costo medido: ~US$0.047 por imagen. `GEMINI_IMAGE_MODEL=off` la desactiva globalmente.
- **Panel:** Inicio, Brand DNA, Calendario, Artículos, Palabras clave y Ajustes.
- **E2E sin gastar:** `docker-compose -f compose.yaml -f compose.mock.yaml up -d --build worker` y luego `npx playwright test` en `frontend/`.
