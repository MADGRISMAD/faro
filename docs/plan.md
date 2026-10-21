> Actualización del 20 de octubre de 2026: Manuel cambió el backend de la Fase 1 a Go. El código y el README reflejan esa decisión; las fases futuras de este plan son históricas.

# FaroContenido (nombre provisional) — plan de Codex, 2026-10-20

Plataforma tipo Soro/Holo: blog con SEO generado + redes sociales desde la misma app.
Stack: SvelteKit (SSR, adapter-node) + Rust (axum, sqlx, Postgres), worker con cola en Postgres. Docker + Caddy.

## Decisiones y supuestos
- Redacción: Gemini 3.8 Flash (0.75/3.75 USD por M tokens); masivo: 3.5 Flash-Lite. Sonnet 5.5 SOLO para artículo -> posts de redes.
- Llaves Gemini: `hostess/hosstess/backend/.env` (GEMINI_API_KEY, GEMINI_MODEL). No copiar a git. Mejor una llave nueva para este proyecto.
- Widget embebible: `/api/embed/<site_uuid>.js`; el SEO real lo da el HTML renderizado en servidor (/blog por proxy o subdominio), JSON-LD y sitemap.
- Integración con la base del cliente: webhook firmado (HMAC) + API REST; NO conexión directa a bases ajenas.
- Voz: Voice Studio (github.com/debpalash/VoiceStudio, local, AGPL-3.0, beta, pesada: no cabe en el VPS) como motor opcional; Gemini TTS de respaldo.
- Video: narrado con FFmpeg (gráficos + subtítulos + voz) por defecto.
- Laya (open-source de Convai, Apache 2.0, modelo en inglés 421M): capa de decisión (puntuar temas, intención de búsqueda, publicar/descartar). No investiga ni redacta. Medir con datos propios antes de depender de él. Datos de palabras clave: fuente externa (DataForSEO/Ahrefs).
- Publicar en IG/FB/TikTok por API oficial exige revisión de las apps (semanas). MVP: exportar/programar a mano.

## Fases
1. MVP: generar -> revisar -> publicar -> mostrar en sitio de ejemplo (SSR, widget, JSON-LD, sitemap).
2. Calidad editorial, investigación, calendario + capa de decisión (Laya).
3. Conector CMS (webhook firmado).
4. Artículo -> posts IG/FB/TikTok con Sonnet.
5. Video narrado (voz + FFmpeg).
6. Publicación oficial en redes.
7. Cobro y operación.

Costo estimado: ~0.09 USD por artículo investigado; ~0.13 USD con paquete social y video narrado.

## Prompt de la fase 1 (para Codex)
Implementa únicamente la fase 1 del plan FaroContenido en `~/Desktop/FaroContenido`, como proyecto independiente. Usa SvelteKit con SSR, Rust/axum/sqlx y PostgreSQL. No modifiques MiColmena ni otros proyectos, no accedas al VPS y no hagas push.
Construye autenticación de propietario, aislamiento por cliente, configuración de sitio, generación real de artículo con `gemini-3.8-flash`, job persistente, editor, aprobación y publicación. Incluye reserva atómica de presupuesto, registro de uso, máximo dos intentos y protección contra duplicados. Para la prueba real se autorizan hasta US$0.50 de consumo; sin grounding.
Publica HTML completo con canonical, metadatos, JSON-LD BlogPosting, sitemap y 404 correctos. Implementa el widget `/api/embed/<site_uuid>.js` y un sitio de ejemplo independiente con `/blog/*` por proxy local. Demuestra que el artículo se lee sin JavaScript; el widget lista y enlaza esas páginas.
Secretos solo del servidor; localiza la configuración Gemini existente sin imprimir ni copiar valores. Si no está disponible, usa proveedor simulado y reporta que falta la comprobación real.
Entrega Docker Compose local, instrucciones y pruebas (generación/publicación, HTML inicial, XSS, aislamiento, presupuesto, idempotencia). No añadas redes, Sonnet, video, pagos ni CMS todavía. Reporta qué funciona, evidencia, costo real y bloqueos.
