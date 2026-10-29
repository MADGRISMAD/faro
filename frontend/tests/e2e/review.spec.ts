import { test, expect } from '@playwright/test';
const panel='http://localhost:18080';const client='http://localhost:18082';
// lo que no se puede respaldar (cifras, promesas) se marca al guardar y el panel lo muestra antes de publicar
test('al guardar un artículo con una cifra sin fuente o una garantía, el panel pide revisarlo',async({page})=>{
 const email=`rev-${Date.now()}@faro.test`;const domain=`rev-${Date.now()}.faro.test`;await page.goto(`${panel}/dashboard`);
 await page.getByLabel('Correo').fill(email);await page.getByLabel('Contraseña').fill('ClaveLocalSegura2026!');await page.getByRole('button',{name:'Crear cuenta',exact:true}).last().click();
 await expect(page.getByRole('heading',{name:'Pega la dirección de tu sitio'})).toBeVisible();
 const made=await page.evaluate(async({domain,client})=>{const r=await fetch('/api/sites',{method:'POST',headers:{'content-type':'application/json','x-csrf-token':localStorage.getItem('faro_csrf')||''},body:JSON.stringify({name:'Taller Norte',domain,canonical_url:client,blog_prefix:'/blog',brand_description:'Herramientas y consejos útiles para dueños de talleres.',tone:'Cercano'})});return r.status},{domain,client});expect(made).toBe(200);
 await page.goto(`${panel}/dashboard/articulos`);await page.getByRole('button',{name:'+ Nuevo artículo'}).click();await page.getByPlaceholder('¿De qué quieres escribir?').fill('Afinación del motor');await page.getByRole('button',{name:'Escribir con IA'}).click();
 await expect(page.getByText('Afinación del motor').first()).toBeVisible({timeout:45000});
 await page.locator('button.post').first().click();await expect(page.getByLabel('Título')).toBeVisible();
 await expect(page.getByText('Revisa esto antes de publicar')).toHaveCount(0);
 await page.getByLabel('Introducción').fill('Una afinación reduce hasta un 70% las fallas y te garantizamos el resultado.');
 await page.getByRole('button',{name:'Guardar cambios'}).click();await expect(page.getByText('Cambios guardados.')).toBeVisible();
 await page.getByRole('button',{name:'Cerrar',exact:true}).click();await page.locator('button.post').first().click();
 const box=page.getByRole('note');await expect(box).toContainText('Revisa esto antes de publicar');await expect(box).toContainText('«70%»');await expect(box).toContainText('«garantizamos»');
 await expect(page.getByText('Por revisar').first()).toBeVisible();
 await page.screenshot({path:'../screenshots/review.png',fullPage:false});
});
