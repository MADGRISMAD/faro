export function canonical(site:{canonical_url:string,blog_prefix:string},slug?:string){return `${site.canonical_url.replace(/\/$/,'')}${site.blog_prefix}${slug?'/'+encodeURIComponent(slug):''}`}
