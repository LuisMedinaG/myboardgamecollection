# Pico CSS Onboarding Prompt

Use this prompt to bring a new agent instance up to speed on the CSS setup for this project.

---

## Prompt

We use **Pico CSS** as our CSS framework in this project. Before you touch any UI or template work, read and internalize the following rules:

### 1. Reference file — read this first
There is a file at the project root called `pico-reference.html`. It contains canonical HTML patterns for every Pico component we use (navbar, grid, card, form, button, modal, table, tooltip, busy states, etc.). **Read this file before building or modifying any UI component.** Do not guess at structure.

### 2. Never read `pico.min.css`
The file `static/styles/pico.min.css` is minified and contains thousands of lines. Reading it wastes context and provides no structural value. It is off-limits.

### 3. How Pico works
Pico is a **classless** framework — it auto-styles standard HTML5 semantic elements. Avoid inventing custom classes for basic layout or typography. Use the right element and Pico handles the rest.

Key patterns to remember:
- **Page wrapper:** `<main class="container">` (centered) or `<main class="container-fluid">` (full-width)
- **Responsive grid:** add `class="grid"` to a parent `<div>` — each direct child becomes an equal-width column
- **Cards:** use `<article>`, optionally with `<header>` and `<footer>` inside
- **Buttons:** `<button>` is primary by default; use `class="secondary"`, `class="contrast"`, `class="outline"` for variants
- **Loading states:** `aria-busy="true"` on any element shows a spinner
- **Tooltips:** `data-tooltip="..."` on any element
- **Title + subtitle pairs:** wrap in `<hgroup>`
- **Modals:** use `<dialog>` — toggle the `open` attribute to show/hide

### 4. Custom CSS
Custom styles live in `static/styles/`. The barrel file `static/style.css` imports them in order. When writing custom CSS:
- Put overrides in the relevant module file (e.g. `game-detail.css`, `vibes.css`)
- New design tokens (colors, spacing, etc.) go in `variables.css`
- Use native CSS nesting (`&`) — no preprocessor

### 5. Templates
Templates are in `templates/`. They use Go's `html/template` package with HTMX for dynamic behavior. When editing templates, use semantic HTML so Pico can do its job — avoid adding wrapper divs just for styling.
