# MC-SoFX Controller Development Guide

## Tailwind CSS Integration

### Overview
This project now uses Tailwind CSS v4.1.11 standalone binary for all styling needs, with minimal custom CSS for dynamic layouts and webaudio-controls theming.

### Architecture
- **Tailwind Utilities**: All static styling (colors, spacing, typography, layout)
- **CSS Custom Properties**: Dynamic grid layouts that change based on AudioUnit layout data
- **Custom CSS**: Only for webaudio-controls component theming

### Development Workflow

#### Building CSS
```bash
# Build CSS once
make css

# Watch for changes and rebuild automatically
make css-watch

# Build minified CSS for production
make css-prod
```

#### VS Code Tasks
Use Cmd+Shift+P → "Tasks: Run Task" and select:
- "Build Tailwind CSS" - One-time build
- "Watch Tailwind CSS" - Auto-rebuild on changes (runs in background)
- "Build CSS for Production" - Optimized build

#### Development Server
```bash
# Start development environment (CSS watcher + Go server)
./dev.sh
```

### Styling Approach

#### ✅ Use Tailwind Utilities For:
- Colors: `bg-gray-800`, `text-blue-400`, `border-gray-700`
- Spacing: `p-4`, `mb-2`, `gap-2`, `mt-1`
- Layout: `flex`, `grid`, `items-center`, `justify-center`
- Typography: `text-sm`, `text-xs`, `font-bold`
- Borders & Radius: `border`, `rounded-lg`, `rounded`

#### ✅ Use CSS Custom Properties For:
- Dynamic grid columns/rows based on layout data
- Theming variables that need to be consistent across components

#### ❌ Avoid:
- Custom CSS classes for layout (use Tailwind utilities)
- Inline styles for static values (use Tailwind utilities)
- Hardcoded colors (use Tailwind color palette)

### File Structure
```
frontend/
├── src/
│   └── input.css          # Tailwind imports + custom CSS
├── static/
│   └── style.css          # Generated CSS output
└── app.html               # Vue template with Tailwind classes
```

### CSS Custom Properties
```css
:root {
  --audio-primary: #3b82f6;     /* Blue for primary controls */
  --audio-secondary: #e5e7eb;   /* Light gray for secondary */
  --audio-accent: #60a5fa;      /* Lighter blue for accents */
  --audio-dark: #1e293b;        /* Dark blue for backgrounds */
  --audio-surface: #334155;     /* Medium blue-gray for surfaces */
}
```

### Dynamic Grid System
The layout grid uses CSS custom properties to allow dynamic sizing:
```css
.grid-audio-dynamic {
  display: grid;
  grid-template-columns: var(--grid-cols, repeat(4, 1fr));
  grid-template-rows: var(--grid-rows, repeat(2, minmax(300px, auto)));
  gap: var(--grid-gap, 1rem);
}
```

Vue.js sets these properties dynamically:
```javascript
dynamicGridStyle() {
  return {
    '--grid-cols': `repeat(${this.currentLayout.grid.columns}, 1fr)`,
    '--grid-rows': `repeat(${this.currentLayout.grid.rows}, minmax(300px, auto))`,
    '--grid-gap': `${Math.max(this.currentLayout.grid.gutter, 16)}px`
  };
}
```

### Benefits
1. **Consistency**: All spacing, colors, and sizing use Tailwind's design system
2. **Efficiency**: JIT compilation only includes classes actually used
3. **Maintainability**: Easy to modify design system globally
4. **Performance**: Optimized CSS output with automatic purging
5. **Developer Experience**: Great autocomplete and IntelliSense support
6. **Flexibility**: Can handle both static utility classes and dynamic layouts

### Future Enhancements
As the UI becomes more complex, you can easily:
- Add Tailwind plugins for specialized functionality
- Extend the color palette for new themes
- Create component variants using Tailwind's modular approach
- Add responsive design with Tailwind's breakpoint system
- Implement dark/light mode switching with Tailwind's dark mode utilities
