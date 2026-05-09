/**
 * Returns the base URL for the workbench app.
 * In production (served through the gateway on port 80/443), uses relative paths.
 * In development (direct access to marketing port), points to the workbench dev server.
 */
export function appUrl(path: string): string {
  // Runtime detection via inline script — see BaseLayout.astro
  return `/app${path}`
}

/** Inline JS that rewrites app links at runtime for dev environments. */
export const APP_URL_REWRITE_SCRIPT = `
<script>
(function(){
  var port = window.location.port;
  // Only rewrite on non-standard ports (dev mode)
  if (port && port !== '80' && port !== '443') {
    var appPort = '3000';
    document.querySelectorAll('a[data-app-link]').forEach(function(a){
      var path = a.getAttribute('data-app-link');
      a.href = 'http://localhost:' + appPort + path;
    });
  }
})();
</script>
`
