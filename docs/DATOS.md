# Datos reales: Analytics y palabras clave

Faro no inventa números. Hay dos fuentes externas y cada una necesita credenciales en el `.env` del servidor. Sin ellas, el panel lo dice y deja los campos vacíos.

## Analytics (Google Search Console) — gratis

Muestra impresiones, clics, CTR, posición media, las búsquedas con las que te encuentran y tus páginas más vistas en Google, comparado con el periodo anterior.

1. En [Google Cloud Console](https://console.cloud.google.com/) crea un proyecto y activa la **Google Search Console API**.
2. *APIs y servicios → Pantalla de consentimiento de OAuth*: tipo Externo, agrega el scope `https://www.googleapis.com/auth/webmasters.readonly` y, mientras la app esté en pruebas, tu correo como usuario de prueba.
3. *Credenciales → Crear credenciales → ID de cliente de OAuth → Aplicación web*. En **URI de redireccionamiento autorizado** pon: `https://TU-DOMINIO-DE-FARO/api/google/callback`.
4. En el `.env` del servidor:
   ```
   GOOGLE_CLIENT_ID=...
   GOOGLE_CLIENT_SECRET=...
   TOKEN_ENCRYPTION_KEY=<64 caracteres hex: openssl rand -hex 32>
   ```
   `TOKEN_ENCRYPTION_KEY` cifra (AES-GCM) el token de Google en la base. Si la cambias, hay que volver a conectar los sitios.
5. El cliente debe tener su sitio **verificado en Search Console** con la misma cuenta con la que autoriza. Google reporta con 2-3 días de retraso y un sitio nuevo tarda semanas en aparecer.

## Palabras clave (volumen, dificultad, costo por clic) — de pago

Usa [DataForSEO](https://dataforseo.com/): *Google Ads Search Volume* (volumen mensual, CPC y competencia) y *Bulk Keyword Difficulty* (dificultad 0-100). El precio es por llamada y no por palabra: Faro pide los temas de un sitio en un solo lote y guarda el resultado 30 días, así que cuesta centavos por sitio. Consulta las tarifas vigentes en su sitio.

```
DATAFORSEO_LOGIN=...
DATAFORSEO_PASSWORD=...
# opcionales (México y español por defecto):
DATAFORSEO_LOCATION_CODE=2484
DATAFORSEO_LANGUAGE_CODE=es
```

Con datos, los temas se ordenan por oportunidad (`volumen / (dificultad + 10)`) y el autopiloto escribe primero los de mayor oportunidad.

## Qué está probado

Todo el flujo está probado contra servicios simulados que reproducen la forma de las respuestas documentadas por Google y DataForSEO. **No se ha probado contra los servicios reales** hasta que se carguen credenciales verdaderas.
