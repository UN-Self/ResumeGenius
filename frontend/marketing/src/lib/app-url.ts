/**
 * Returns the base URL for the workbench app.
 * In production (served through the gateway on port 80/443), uses relative paths.
 * In development (direct access to marketing port), the inline rewrite script
 * (APP_URL_REWRITE_SCRIPT) patches data-app-link anchors at runtime.
 */
export function appUrl(path: string): string {
  return `/app${path}`
}

/** Inline JS that rewrites app links at runtime for dev environments. */
export const APP_URL_REWRITE_SCRIPT = `
<script>
(function(){
  var port = window.location.port;
  if (port && port !== '80' && port !== '443') {
    var appBase = 'http://localhost:3000';
    var apiBase = 'http://localhost:8080';
    // Rewrite app links
    document.querySelectorAll('a[data-app-link]').forEach(function(a){
      a.href = appBase + a.getAttribute('data-app-link');
    });
    // Rewrite API fetch base for auth check in Nav
    window.__rg_api_base = apiBase;
  }
})();
</script>
`
