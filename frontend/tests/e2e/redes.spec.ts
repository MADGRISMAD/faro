import { test, expect } from '@playwright/test';
const panel = 'http://localhost:18080';

test('conectar Facebook (simulado), elegir la página y desconectar', async ({ page }) => {
	const email = `fb-${Date.now()}@faro.test`, domain = `fb-${Date.now()}.faro.test`;
	await page.goto(`${panel}/dashboard`);
	await page.getByLabel('Correo').fill(email);
	await page.getByLabel('Contraseña').fill('ClaveLocalSegura2026!');
	await page.getByRole('button', { name: 'Crear cuenta', exact: true }).last().click();
	await expect(page.getByRole('heading', { name: 'Pega la dirección de tu sitio' })).toBeVisible();
	const made = await page.evaluate(async (domain) => (await fetch('/api/sites', { method: 'POST', headers: { 'content-type': 'application/json', 'x-csrf-token': localStorage.getItem('faro_csrf') || '' }, body: JSON.stringify({ name: domain, domain, canonical_url: `https://${domain}`, blog_prefix: '/blog', brand_description: 'x', tone: 'x' }) })).status, domain);
	expect(made).toBe(200);
	await page.goto(`${panel}/dashboard/redes`);
	await page.getByRole('button', { name: 'Conectar Facebook' }).click();
	await expect(page.getByText('Elige en qué página')).toBeVisible();
	await page.getByLabel('Otra página').check();
	await page.getByRole('button', { name: 'Usar esta página' }).click();
	await expect(page.getByText('Conectado a')).toContainText('Otra página');
	await page.screenshot({ path: '../screenshots/panel-redes.png', fullPage: true });
	await page.getByRole('button', { name: 'Desconectar' }).click();
	await page.getByRole('alertdialog').getByRole('button', { name: 'Desconectar' }).click();
	await expect(page.getByRole('button', { name: 'Conectar Facebook' })).toBeVisible();
});
