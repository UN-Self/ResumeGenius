/**
 * Shared ProseMirror attribute helpers for custom extensions.
 *
 * `class` and `style` attributes on extensions share the same null-suppression
 * pattern: when the attribute isn't present, return {} to prevent an empty
 * `class=""` or `style=""` from appearing in the serialized HTML.
 */

/**
 * Create a parseHTML/renderHTML attribute pair for an optional string attribute.
 * renderHTML returns {} when the attribute value is falsy, suppressing the
 * attribute from the DOM output.
 */
export function nullSafeAttr(name: string) {
  return {
    default: null,
    parseHTML: (element: HTMLElement) => element.getAttribute(name),
    renderHTML: (attributes: Record<string, string>) => {
      if (!attributes[name]) return {}
      return { [name]: attributes[name] }
    },
  }
}
