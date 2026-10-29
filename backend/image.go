package main

// Imagen de portada por artículo con Nano Banana 2 Lite (el modelo de imagen más barato de Gemini).
// Una sola imagen por artículo, ilustración SIN texto (los modelos deforman las letras) y guardada como JPEG comprimido.

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const imageReserve = 150000 // US$0.15 por imagen: una de 1K con Nano Banana 2 son ~1,100 tokens de imagen más ~400 de razonamiento (US$0.09 a tarifa de imagen)

type ImagePrompt struct {
	Title, Business, Audience, Tone, Style, Notes string
	Palette                                       []string // colores de la marca (hex); vacío = paleta sobria libre
}

// Estilos que el cliente puede elegir en Brand DNA. Cada uno describe técnica, luz y textura, no solo un adjetivo:
// así dos portadas del mismo sitio se ven de la misma serie. Las muestras del selector (static/estilos/) salieron de aquí.
var imageStyles = map[string]string{
	"memphis":    "ilustración vectorial plana estilo Memphis corporativo: el sujeto en formas limpias con contornos finos y uniformes, rodeado de pocas figuras geométricas decorativas (círculos, medios círculos, triángulos, zigzags, líneas onduladas y retículas de puntos) bien ordenadas en los bordes; colores sólidos sin degradados, sin texturas ni sombras realistas, mucho espacio libre y un acabado pulido de marca tecnológica, nunca infantil ni recargado",
	"clean":      "ilustración editorial vectorial plana, formas geométricas limpias, sombras planas suaves y un grano fino sutil, como la portada de una revista de negocios moderna",
	"photo":      "fotografía editorial realista, luz natural lateral suave, poca profundidad de campo (lente de 35 mm), colores naturales; si aparecen personas, de espaldas, de perfil o solo sus manos, nunca rostros de frente",
	"watercolor": "pintura en acuarela sobre papel de grano grueso, manchas que se difuminan en los bordes, contornos finos de tinta y mucho blanco del papel",
	"3d":         "render 3D estilizado tipo arcilla o juguete, formas redondeadas, materiales mate, luz de estudio suave con sombras de contacto y fondo de un solo color",
	"vintage":    "ilustración retro de mediados del siglo XX, impresión serigráfica con trama de puntos, paleta cálida limitada y textura de papel envejecido",
	"abstract":   "cartel moderno con formas geométricas y orgánicas que sugieren el tema, colores intensos y contrastados, composición dinámica",
}

type ImageResult struct {
	Data                      []byte // PNG o JPEG tal como lo devolvió el modelo
	InputTokens, OutputTokens int
	Unknown                   bool
	Err                       error
}

func imageText(p ImagePrompt) string {
	style, ok := imageStyles[p.Style]
	if !ok {
		style = imageStyles["memphis"]
	}
	colors := "\nColores: una paleta sobria y armónica de 3 o 4 tonos, más blanco."
	if len(p.Palette) > 0 {
		colors = "\nColores (obligatorio): usa únicamente los colores de la marca " + strings.Join(p.Palette, ", ") + ", sus tonos más claros, blanco y un gris muy oscuro para contornos. Ningún otro color: nada de azules, verdes, amarillos ni otros tonos que no estén en esa lista."
	}
	notes := ""
	if n := strings.TrimSpace(p.Notes); n != "" {
		notes = "\nIndicaciones de la marca (tienen prioridad sobre el estilo): " + clean(n, 300) + "."
	}
	return fmt.Sprintf(`Portada para un artículo de blog, formato horizontal 16:9.
Tema del artículo (solo para saber qué mostrar; NO lo escribas en la imagen): %s
Negocio: %s. Lectores: %s.
Qué mostrar: una escena concreta que los lectores reconozcan de su día a día con este tema: el objeto, el lugar o el momento real del que trata (por ejemplo el mostrador de una tienda, unas manos cobrando con una terminal, la cocina de un restaurante), no ideas abstractas. Un solo sujeto principal que se entienda a primera vista, con detalles que lo hagan propio de México cuando aplique. Si el tema compara dos opciones, muéstralas lado a lado solo con objetos.
Composición: sujeto un poco fuera del centro, fondo sencillo y aire alrededor para que se pueda recortar a cuadrado sin perder lo importante; profundidad entre primer plano y fondo.
Estilo: %s.%s
Contraste: buena separación entre el sujeto y el fondo.
Evita: personas sonriendo a la cámara, apretones de manos, gente señalando pantallas, hologramas, cerebros o robots genéricos de «inteligencia artificial», engranes, focos, flechas de crecimiento, gráficas de adorno, monedas flotando y oficinas genéricas.%s
Prohibido absolutamente: texto, letras, palabras, números, logotipos, marcas comerciales reales, marcas de agua, letreros y pantallas con texto legible. Los productos, empaques, latas y botellas van lisos o con formas simples, sin nombres ni etiquetas que se lean.`, p.Title, p.Business, p.Audience, style, colors, notes)
}

func (*Mock) Image(_ context.Context, _ ImagePrompt, _ Config) ImageResult {
	img := image.NewRGBA(image.Rect(0, 0, 64, 36))
	for y := 0; y < 36; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, color.RGBA{214, 128, 93, 255}) // color liso: es solo el relleno de las pruebas
		}
	}
	var b bytes.Buffer
	_ = jpeg.Encode(&b, img, nil)
	return ImageResult{Data: b.Bytes(), InputTokens: 100, OutputTokens: 1120}
}

func (g *Gemini) Image(ctx context.Context, p ImagePrompt, c Config) ImageResult {
	body, _ := json.Marshal(map[string]any{
		"contents":         []any{map[string]any{"parts": []any{map[string]string{"text": imageText(p)}}}},
		"generationConfig": map[string]any{"responseModalities": []string{"IMAGE"}, "imageConfig": map[string]any{"aspectRatio": "16:9"}},
	})
	req, e := http.NewRequestWithContext(ctx, "POST", "https://generativelanguage.googleapis.com/v1beta/models/"+c.ImageModel+":generateContent", bytes.NewReader(body))
	if e != nil {
		return ImageResult{Err: e}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.Key)
	res, e := g.Client.Do(req)
	if e != nil {
		return ImageResult{Unknown: true, Err: errors.New("No se pudo contactar a Gemini")}
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return ImageResult{Err: fmt.Errorf("Gemini respondió con estado %d al generar la imagen", res.StatusCode)}
	}
	data, e := io.ReadAll(io.LimitReader(res.Body, 20<<20))
	var wire struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Inline struct {
						Data string `json:"data"`
					} `json:"inlineData"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		U struct {
			P int `json:"promptTokenCount"`
			C int `json:"candidatesTokenCount"`
			T int `json:"thoughtsTokenCount"`
		} `json:"usageMetadata"`
	}
	if e != nil || json.Unmarshal(data, &wire) != nil {
		return ImageResult{Err: errors.New("Respuesta de imagen inválida")}
	}
	r := ImageResult{InputTokens: wire.U.P, OutputTokens: wire.U.C + wire.U.T}
	for _, cand := range wire.Candidates {
		for _, part := range cand.Content.Parts {
			if part.Inline.Data != "" {
				if r.Data, e = base64.StdEncoding.DecodeString(part.Inline.Data); e != nil {
					r.Err = errors.New("Imagen corrupta")
				}
				return r
			}
		}
	}
	r.Err = errors.New("Gemini no devolvió imagen")
	return r
}

// imageCost: la salida de imagen se cobra a su propia tarifa; la entrada (texto corto) a la del modelo de texto.
func imageCost(c Config, in, out int) int64 {
	return (int64(in)*c.InputRate + int64(out)*c.ImageRate + 999) / 1000
}

// toJPEG valida que sea una imagen real y la guarda como JPEG (≈150 KB en lugar de ~2 MB de PNG).
func toJPEG(raw []byte) ([]byte, error) {
	cfg, _, e := image.DecodeConfig(bytes.NewReader(raw))
	if e != nil || cfg.Width < 64 || cfg.Height < 36 || cfg.Width*cfg.Height > 25_000_000 {
		return nil, errors.New("la imagen no es válida")
	}
	img, _, e := image.Decode(bytes.NewReader(raw))
	if e != nil {
		return nil, errors.New("la imagen no es válida")
	}
	var b bytes.Buffer
	if e = jpeg.Encode(&b, img, &jpeg.Options{Quality: 82}); e != nil {
		return nil, e
	}
	return b.Bytes(), nil
}

func (a *App) runImage(ctx context.Context, j *WorkItem) {
	callCtx, cancel := context.WithTimeout(ctx, 100*time.Second)
	defer cancel()
	ip := ImagePrompt{Title: j.Prompt.Topic, Business: j.Prompt.Business, Audience: j.Prompt.Audience, Tone: j.Prompt.Tone}
	_ = a.DB.QueryRowContext(ctx, `SELECT image_style,image_notes FROM sites WHERE id=$1`, j.Site).Scan(&ip.Style, &ip.Notes)
	ip.Palette = a.sitePalette(callCtx, j.Site)
	r := a.Provider.Image(callCtx, ip, a.Config)
	var jpg []byte
	if r.Err == nil {
		if jpg, r.Err = toJPEG(r.Data); r.Err != nil {
			r.Err = errors.New("Gemini devolvió una imagen inválida")
		}
	}
	if e := a.finishImage(ctx, j, r, jpg); e != nil {
		slog.Error("No se pudo completar la imagen", "error", e)
	}
}

func (a *App) finishImage(ctx context.Context, j *WorkItem, r ImageResult, jpg []byte) error {
	tx, e := a.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var status string
	var attempt int
	if e = tx.QueryRowContext(ctx, `SELECT status,attempts FROM jobs WHERE id=$1 FOR UPDATE`, j.ID).Scan(&status, &attempt); e != nil {
		return e
	}
	if status != "running" || attempt != j.Attempt {
		return errors.New("job cambió de estado durante la llamada")
	}
	cost, usage := imageCost(a.Config, r.InputTokens, r.OutputTokens), "measured"
	if r.Err != nil && r.Unknown && r.InputTokens == 0 && r.OutputTokens == 0 {
		cost, usage = j.Reserve, "unknown"
	}
	if cost > j.Reserve {
		return errors.New("uso mayor que reserva: requiere conciliación")
	}
	if _, e = tx.ExecContext(ctx, `UPDATE tenants SET reserved_microusd=reserved_microusd-$1,spent_microusd=spent_microusd+$2 WHERE id=$3`, j.Reserve, cost, j.Tenant); e != nil {
		return e
	}
	if cost > 0 {
		if _, e = tx.ExecContext(ctx, `INSERT INTO usage_events(tenant_id,job_id,attempt,model,input_tokens,output_tokens,cost_microusd,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, j.Tenant, j.ID, j.Attempt, a.Config.ImageModel, r.InputTokens, r.OutputTokens, cost, usage); e != nil {
			return e
		}
	}
	if r.Err != nil {
		next := "queued"
		if j.Attempt >= 2 {
			next = "failed" // sin imagen el artículo se publica igual
		}
		if _, e = tx.ExecContext(ctx, `UPDATE jobs SET status=$1,reserved_microusd=0,error_message=$2,updated_at=now() WHERE id=$3`, next, r.Err.Error(), j.ID); e != nil {
			return e
		}
		return tx.Commit()
	}
	alt := "Ilustración de portada: " + clean(j.Prompt.Topic, 160)
	if _, e = tx.ExecContext(ctx, `INSERT INTO article_images(article_id,tenant_id,mime,data,alt) VALUES($1,$2,'image/jpeg',$3,$4) ON CONFLICT(article_id) DO UPDATE SET data=EXCLUDED.data,alt=EXCLUDED.alt`, j.Article, j.Tenant, jpg, alt); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, `UPDATE articles SET has_image=true WHERE id=$1`, j.Article); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, `UPDATE jobs SET status='done',reserved_microusd=0,error_message=NULL,updated_at=now() WHERE id=$1`, j.ID); e != nil {
		return e
	}
	return tx.Commit()
}

func serveImage(w http.ResponseWriter, r *http.Request, db *sql.DB, query string, private bool, args ...any) {
	var mime, alt string
	var data []byte
	e := db.QueryRowContext(r.Context(), query, args...).Scan(&mime, &data, &alt)
	if errors.Is(e, sql.ErrNoRows) {
		fail(w, 404, "Sin imagen")
		return
	}
	if e != nil {
		dbError(w, e)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("X-Image-Alt", strings.Map(func(r rune) rune {
		if r < 32 || r > 126 {
			return -1
		}
		return r
	}, alt))
	if private {
		w.Header().Set("Cache-Control", "private, max-age=300")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}
	w.Write(data)
}

func (a *App) articleImage(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, false)
	if !ok {
		return
	}
	id, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	serveImage(w, r, a.DB, `SELECT mime,data,alt FROM article_images WHERE article_id=$1 AND tenant_id=$2`, true, id, t)
}

func (a *App) publicArticleImage(w http.ResponseWriter, r *http.Request) {
	site, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	serveImage(w, r, a.DB, `SELECT i.mime,i.data,i.alt FROM article_images i JOIN articles a ON a.id=i.article_id WHERE a.site_id=$1 AND (a.slug=$2 OR a.id IN (SELECT article_id FROM article_slug_aliases WHERE site_id=a.site_id AND old_slug=$2)) AND a.status='published'`, false, site, r.PathValue("slug"))
}
