/**
 * Shared class strings for form controls and labels. Use the components in
 * this folder where possible; these are for the few places that need the raw
 * classes (e.g. a third-party input).
 */

/** Text inputs, selects and textareas. 16px text so iOS does not zoom on focus. */
export const controlClass =
  'w-full bg-surface-container-high text-on-surface text-base rounded-xl px-3 py-2 outline-none ' +
  'placeholder:text-on-surface-faint focus:ring-1 focus:ring-primary/40 disabled:opacity-50 transition-fluid'

/** Small uppercase label: form fields, table headers, metadata. */
export const labelClass = 'text-xs text-on-surface-faint uppercase tracking-widest font-medium'
