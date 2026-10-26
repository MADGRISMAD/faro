# Redes sociales (fase 1: Facebook)

Cada artículo que pasa a **publicado** (autopiloto o botón Publicar) se comparte una vez en la página de Facebook del sitio, con el título, el resumen y el enlace. Aprobar o dejar en borrador no comparte nada.

- Conexión: `POST /sites/{id}/facebook/start` → Facebook Login → `GET /facebook/callback` → si hay varias páginas, `POST /sites/{id}/facebook/page`. Se guarda solo el token de la página, cifrado con `TOKEN_ENCRYPTION_KEY`.
- Cola: la tabla `social_posts`. La llena el disparador `articles_social` de Postgres (cubre todas las rutas de publicación) y la vacía `socialTick` en el worker cada 30 s.
- Estados: `pending` → `sending` → `sent`. Token revocado: la conexión queda `broken` y el panel pide reconectar. Falla temporal: un reintento a los 10 min. Sin respuesta de Facebook: `unknown` (no se reenvía para no duplicar).
- Desconectar es local: borra el token del sitio y cancela lo pendiente.

## Configurar en Meta
1. App en developers.facebook.com ligada al portafolio de Hyperion; producto Facebook Login.
2. URI de redirección OAuth válida: `https://faro.hyperionmkt.com/api/facebook/callback`.
3. Política de privacidad: `https://www.hyperionmkt.com/faro/privacidad`; instrucciones de eliminación: `https://www.hyperionmkt.com/faro/privacidad#eliminar-datos`.
4. En el servidor: `FACEBOOK_APP_ID` y `FACEBOOK_APP_SECRET` en `~/farocontenido/.env`, y reiniciar backend y worker.
5. En modo desarrollo solo funciona con administradores o testers de la app. Para clientes externos: verificación del negocio y revisión de `pages_show_list`, `pages_read_engagement` y `pages_manage_posts` con un video del flujo.
