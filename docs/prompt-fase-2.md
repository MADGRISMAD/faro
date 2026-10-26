# Prompt para Codex: completar FaroContenido (todo lo que falta + quitar lo que no se pidió)

Trabaja en ~/Desktop/FaroContenido (ya existe la fase 1: backend en Go, frontend SvelteKit, Postgres, widget, blog SSR). Español de México en todo. Autor de commits: madgrismad <madgrismad@gmail.com>, sin Co-Authored-By ni atribución a IA. No hagas push ni accedas por SSH al VPS (yo despliego). No toques otros proyectos de ~/Desktop.

## Decisión ya tomada
El backend se queda en **Go** (Manuel lo prefiere; ignora lo de Rust). Frontend SvelteKit. No reescribas lo que ya funciona sin razón.

## El producto que Manuel pidió (lista de verificación: lo que no esté aquí, no debe existir)
Una plataforma tipo **Soro (trysoro.com) y Holo**, pero con redes sociales dentro:
1. **Blog con SEO automático:** investiga qué escribir, redacta, programa y publica artículos en el sitio del cliente.
2. **Widget embebible** como el de Soro: `<div id="faro-blog"></div><script src="https://HOST/api/embed/<uuid>" defer></script>` que inserta las publicaciones en el blog del sitio. Mantén el equivalente actual y amplíalo (ver B).
3. **Integración con una base de datos / CMS del cliente** y opción de **anexar widgets** (ver B).
4. **Redes sociales desde la app:** a partir de cada artículo del blog se crean publicaciones para **Instagram, Facebook y TikTok**, con **Claude Sonnet en el medio** para esa transformación (ver C).
5. **Videos** de esas publicaciones, con **voz** (ver D) y un **agente de decisiones** para elegir qué hacer (ver E).
6. **Gemini** con las credenciales ya existentes y un modelo mejor que el que usan los otros proyectos, pero barato (ya está: gemini-3.8-flash; sigue verificándolo con models.list antes de usarlo y usa el Flash-Lite para tareas masivas).

## 0. Auditoría primero (obligatoria)
Antes de programar, compara lo construido contra la lista de arriba y entrega un cuadro corto: "pedido / existe / falta / sobra". Elimina o esconde lo que sobra (todo lo que no sirva a los puntos 1 a 6). Candidatos que Manuel siente que no pidió: formulario técnico de sitio (dominio, URL canónica, prefijo) como primer paso, el aviso de "presupuesto US$0.50" en pantalla, la portada genérica con el texto "Ideas que se convierten en artículos", la numeración "01 · 02 · 03" y la página única de panel. Sustitúyelos por el flujo de abajo. Lista en tu reporte qué quitaste y por qué.

## A. Flujo y panel (rehacer la experiencia)
- **Onboarding tipo Soro:** el usuario pega la URL de su sitio -> el servidor la lee (con protección SSRF: solo http/https, bloquea IPs privadas/loopback/metadata, límite de tamaño y de tiempo, sin redirecciones a redes privadas) -> Gemini Flash-Lite resume negocio, público, tono y 10 temas candidatos -> el usuario confirma. Nada de formularios largos al inicio.
- **App con barra lateral:** Inicio (resumen), Plan de contenido (calendario), Artículos, Redes, Videos, Integraciones, Uso y límites, Ajustes. Responsive en teléfono.
- **Modo autopiloto** por sitio (apagado por defecto): genera y programa N artículos por semana, que pasan por tu aprobación salvo que actives "publicar solo". Siempre con tope de costo.
- **Plan de contenido:** calendario mensual con arrastrar para reprogramar, estados (idea, borrador, aprobado, programado, publicado), y publicación programada por el worker (una sola vez aunque se reinicie).
- **Editor** mejorado: vista previa del artículo y de cómo se verá en el widget, puntaje SEO simple explicado (título, meta, encabezados, longitud, enlaces internos, palabra clave), historial de revisiones.
- Diseño: **sin degradados y sin fondos de cuadrícula tenues** (parece IA). Conserva la identidad actual (serif + terracota) pero con aspecto de producto real: estados de carga, error y vacío, atajos claros.

## B. SEO, widgets e integración con base de datos / CMS
- **Investigación de temas:** agrupa temas por intención (informativa, comparación, compra, local), propón títulos y palabra clave. NO inventes volúmenes de búsqueda: si no hay fuente real, no muestres números. Deja una interfaz (trait/provider) para conectar DataForSEO u otra API de palabras clave más adelante.
- **Enlaces internos** automáticos entre artículos del mismo sitio; **JSON-LD** (BlogPosting, FAQPage cuando haya FAQ, BreadcrumbList), sitemap, RSS/Atom por sitio, imágenes con alt y Open Graph.
- **Widgets embebibles** (todos con shadow DOM, sin cookies, CORS correcto, caché y aislamiento de estilos): lista de artículos, últimos N, artículo individual, y tarjeta de CTA. Selector de tema (claro/oscuro/colores de marca) y generador de código para copiar en el panel.
- **Integraciones con el CMS/base del cliente (en este orden):**
  1. **Webhook firmado** (HMAC-SHA256, timestamp, id de evento, reintentos con backoff, bloqueo SSRF en destinos) que avisa de artículo publicado/actualizado/eliminado + **API REST** con token por sitio (solo lectura) para que el cliente lea el contenido y lo guarde en SU base de datos.
  2. **WordPress** (REST API con Application Password) para publicar directo, y **exportación** a Markdown/JSON.
  - NO hagas conexión directa a bases de datos ajenas.
- Blog servido por proxy `/blog` o subdominio, como ya existe; documenta cómo configurarlo con Nginx, Caddy y Cloudflare.

## C. Redes sociales con Claude Sonnet en el medio
- Pantalla **Redes:** conectar cuentas (Instagram Business/Creator, Facebook Page, TikTok) con OAuth oficial. Tokens **cifrados en reposo** (AES-GCM con llave en variable de entorno), renovación, revocación y manejo de expiración. Sin credenciales de app configuradas, la pantalla muestra "Conectar" desactivado con instrucciones claras y todo el flujo funciona en **modo manual**.
- **Transformación artículo -> publicaciones:** cada artículo publicado genera, con **Claude Sonnet (`claude-sonnet-5-5`, API de Anthropic)**, un paquete JSON con: post de Instagram (caption, hashtags, idea visual, texto para carrusel), post de Facebook, y guion corto de TikTok/Reels (gancho, escenas, voz en off, CTA). Prompt corto, salida JSON con esquema, caché por (revisión + versión de prompt + idioma), tope de tokens, lote (batch) para lo no urgente, y reserva/registro de costo como el resto. JSON inválido jamás se publica.
- **Editor de publicaciones** por red con vista previa estilo cada plataforma, edición, aprobación, programación y estado.
- **Pieza visual:** plantilla de imagen (tarjeta con título y marca del cliente, generada en el servidor con fuentes y colores de la marca; sin degradados) lista para IG/FB.
- **Publicación:**
  - Instagram y Facebook por Graph API oficial (Content Publishing; Page token) y TikTok por Content Posting API. Respeta límites, consulta cuotas (`content_publishing_limit`) y no fijes números diarios no verificados.
  - Implementa el flujo real con las credenciales por variable de entorno, **pero** debe existir siempre el **modo manual**: descargar paquete (imagen/video + texto), copiar texto y botón "Marcar como publicado". Nada debe depender de que Meta o TikTok aprueben la app.
  - TikTok sin auditoría solo publica como `SELF_ONLY`; muéstralo en la interfaz y exige selección de privacidad y consentimiento como pide su documentación.
  - Idempotencia: un timeout no duplica publicaciones; concilia el estado remoto.
- Pruebas con un proveedor simulado de cada red. No hagas llamadas reales de publicación.

## D. Video y voz
- **Voz:** interfaz `Voice` con dos motores:
  1. **Voice Studio** (github.com/debpalash/VoiceStudio, local, AGPL-3.0, beta; se usa por HTTP/CLI/MCP como servicio EXTERNO, sin copiar su código a este repo: la licencia es AGPL). Configura la URL por variable de entorno. Lee su documentación en GitHub para usar su API real. No cabe en el VPS (3.8 GB compartidos): debe poder correr en otra máquina o en la Mac del dueño.
  2. **Gemini TTS** como respaldo (`es-MX`), con muestra de prueba.
- **Video narrado de 30–45 s** vertical (1080x1920, 30 fps, MP4 H.264/AAC): guion del paso C -> voz -> escenas con texto animado y recursos de la marca -> subtítulos quemados -> FFmpeg. Un render a la vez, límites de duración/tamaño/RAM, almacenamiento con caducidad, `ffprobe` para validar.
- **Avatar parlante:** fuera de alcance por ahora (no lo pidió Manuel en esta forma). No lo agregues.
- Reproductor y descarga en la pantalla Videos; subida a Instagram/TikTok por el mismo flujo del punto C.

## E. Agente de decisiones (Laya)
**Laya** (laya-ai.com, de Convai; modelo abierto Apache 2.0, ~421M parámetros, responde decisiones TIPADAS —elegir, puntuar, clasificar— en ms; modelo publicado en inglés; NO investiga ni redacta). Verifica en su repositorio oficial y en Hugging Face cómo se ejecuta y su licencia antes de usarlo. Úsalo como capa de decisión detrás de una interfaz `Decider`:
- Decisiones: puntuar y ordenar temas candidatos, clasificar intención de búsqueda, decidir qué artículos convertir en video, qué red priorizar y qué horario, y si un contenido se publica o se descarta.
- Implementa `Decider` con (1) Laya como servicio local opcional (perfil de Docker Compose `laya`, no obligatorio, no en el VPS) y (2) **respaldo con Gemini Flash-Lite** con salida tipada, que es el predeterminado si Laya no está disponible.
- **Evalúa con honestidad:** crea un pequeño conjunto de prueba (30–50 ejemplos en español de México, con etiquetas que tú defines) y compara Laya contra el respaldo en exactitud y latencia. Si Laya en español rinde mal, déjalo desactivado y repórtalo. No inventes resultados.
- Muestra en la interfaz el "porqué" de cada decisión (puntaje y motivo) y permite al usuario corregirla.

## F. Costos, límites y seguridad (transversal)
- Reserva atómica y registro de costo por operación: Gemini (redacción y Flash-Lite), Sonnet, voz, render. Tarifas en configuración. Pantalla **Uso y límites** clara para el cliente (créditos, no dólares crudos) y tope por plan; el sistema nunca debe gastar más de lo reservado. Máximo 2 intentos, una generación simultánea por cliente.
- Variables nuevas (todas opcionales, con `.env.example`): `ANTHROPIC_API_KEY`, `META_APP_ID/SECRET`, `TIKTOK_CLIENT_KEY/SECRET`, `TOKEN_ENCRYPTION_KEY`, `VOICESTUDIO_URL`, `LAYA_URL`. **Nunca imprimas ni registres valores.** Si falta `ANTHROPIC_API_KEY`, el paso de Sonnet usa el proveedor simulado y la interfaz lo dice.
- Seguridad: aislamiento entre clientes con prueba, XSS en todo contenido generado (artículos, captions, previews), SSRF en lectura de URL y webhooks, CSRF, rate limit, cabeceras, logs sin contenido sensible, tokens cifrados.
- **Copias de seguridad:** script `scripts/backup.sh` (pg_dump comprimido, rotación de 14 días) y su restauración probada.
- Dejo hueco para la etiqueta de analítica Umami (variable `UMAMI_SCRIPT` opcional en el layout público y el panel).

## G. Pruebas y evidencia (ejecútalas de verdad)
- Go: `go vet`, `go test -race` con Postgres real: idempotencia, presupuesto bajo concurrencia, aislamiento entre clientes, SSRF (IPs privadas, redirecciones, DNS que resuelve a IP privada), firma HMAC y replay del webhook, cifrado de tokens, transformación con JSON inválido, programación que no duplica tras reinicio.
- Frontend: check, lint, unitarias. E2E con Playwright: onboarding por URL -> plan -> artículo -> aprobar -> publicar -> paquete social -> video -> modo manual de publicación. Repite la lectura pública con JavaScript desactivado.
- Una corrida REAL con Gemini (tope US$0.50 en total) y, si existe `ANTHROPIC_API_KEY`, una con Sonnet (tope US$0.10). Sin llaves, simulado y dilo claro. Costo real medido en el reporte.
- Capturas de cada pantalla nueva (escritorio y teléfono) y revisa que no haya degradados ni cuadrículas.

## Entrega
Commits pequeños en español. README actualizado (un comando para levantar todo con Docker Compose, variables, cómo probar, cómo conectar Meta/TikTok/Voice Studio/Laya). `compose.prod.yaml` actualizado para el VPS (sin publicar el puerto de Postgres; Laya, Voice Studio y render pesado como perfiles opcionales fuera del VPS). Reporta en español: cuadro de auditoría, qué funciona con evidencia, qué NO pudiste comprobar (publicación real en redes, Laya en español, Voice Studio real), costo real, modelo usado y decisiones tomadas por tu cuenta.
